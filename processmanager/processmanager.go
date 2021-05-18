// Package processmanager is responsible for running the processes defined in the configuration.
package processmanager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/internallogger"
	"github.com/morfien101/launch/processlogger"
	"github.com/morfien101/launch/signalreplicator"
)

const (
	initProcess   = "init"
	mainProcess   = "main"
	secretProcess = "secret"
)

// ProcessManger holds the config and state of the running processes.
type ProcessManger struct {
	config        configfile.Processes
	logger        *processlogger.LogManager
	pmlogger      internallogger.IntLogger
	mainProcesses []*Process
	EndList       []*processEnd
	wg            sync.WaitGroup
	tumble        chan bool
	shuttingDown  bool
}

type processEnd struct {
	Name        string `json:"name"`
	ProcessType string `json:"type"`
	Error       error  `json:"runtime_error,omitempty"`
	ExitCode    int    `json:"exit_code"`
}

// New will create a ProcessManager with the supplied config and return it
func New(
	config configfile.Processes,
	logManager *processlogger.LogManager,
	pmlogger internallogger.IntLogger,
) *ProcessManger {
	pm := &ProcessManger{
		config:   config,
		logger:   logManager,
		pmlogger: pmlogger,
		tumble:   make(chan bool, 1),
	}

	// Once we get a single process fail we shutdown everything.
	// A SIGTERM is sent to the master signal channel
	go pm.terminator()

	return pm
}

// terminator is used when we need to terminate the processes in the Main processes stack.
// This is normally because either it has errored and failed or it has finished.
func (pm *ProcessManger) terminator() {
	pm.pmlogger.Debugf("Starting termination watcher\n")
	for range pm.tumble {
		pm.pmlogger.Debugf("Starting the tumble of stacks\n")
		// If the process manager is already shutting down we will get a few
		// signals from the processes as the stack starts breaking down.
		// We can safely discard them since we expect them.
		if pm.shuttingDown {
			pm.pmlogger.Debugf("Already in shutdown process skipping signal replication\n")
			continue
		}
		// If we are not in shutdown mode trigger it. Them mark it as shutting down.
		pm.shuttingDown = true
		// We only need to send signals to propagate if we have started some mains.
		if pm.mainProcesses != nil {
			signalreplicator.Send(syscall.SIGTERM)
			pm.pmlogger.Debugf("Sending %s signal to replicator\n", syscall.SIGTERM)
		}
	}
}

// RunInitProcesses will run all of the processes that are under the
// init processes configuration. All init processes will be run sequentially
// in the order supplied and MUST return success before the next is started.
// If a process does not return successfully an error is returned and further
// processing will stop.
func (pm *ProcessManger) RunInitProcesses() (string, error) {
	pm.pmlogger.Println("Starting Init Processes")
	// Start the main processes
	for _, procConfig := range pm.config.InitProcesses {
		if err := pm.runInitProc(procConfig); err != nil {
			return pm.exitStatusFormatter(), err
		}
	}
	return "", nil
}

func (pm *ProcessManger) runInitProc(procConfig *configfile.Process) error {
	// Create a process object
	proc := &Process{
		config:   procConfig,
		pmlogger: pm.pmlogger,
		sigChan:  make(chan os.Signal, 1),
	}
	pm.pmlogger.Debugf("Attempting to run %s.\n", proc.config.CMD)
	// setup logging hooks
	// If this fails we can't carry on.
	err := pm.setupProcess(proc)
	if err != nil {
		return err
	}

	// Run the process
	endstate := proc.runProcess(initProcess)
	signalreplicator.Remove(proc.sigChan)
	pm.pmlogger.Debugf("Finished running %s.\n", proc.config.CMD)
	pm.EndList = append(pm.EndList, endstate)
	if endstate.Error != nil {
		pm.pmlogger.Debugln("The last init command failed. Stack will now tumble.")
		pm.tumble <- true

		return fmt.Errorf("Process %s failed. Error reported: %s", procConfig.Name, endstate.Error)
	}
	return nil
}

// RunMainProcesses will start the processes listed in sequential order.
// Processes are expected to start and stay running. A failure of one
// will cause all the ProcessManger to send termination signals to all
// remaining and subsequently kill the processes manager.
func (pm *ProcessManger) RunMainProcesses() (chan string, error) {
	// We need to wait for signals and repeat them into the processes
	pm.pmlogger.Debugln("Starting signal catcher go func")

	pm.pmlogger.Println("Starting Main Processes")
	// Start the main processes
	for _, procConfig := range pm.config.MainProcesses {
		// Create a process object
		pm.wg.Add(1)
		pm.pmlogger.Debugf("Adding %s to the list of main processes.\n", procConfig.CMD)
		proc := &Process{
			config:   procConfig,
			pmlogger: pm.pmlogger,
			sigChan:  make(chan os.Signal, 1),
			shutdown: make(chan bool, 1),
		}
		pm.mainProcesses = append(pm.mainProcesses, proc)
		// setup logging hooks
		// If this fails we can't carry on.
		err := pm.setupProcess(proc)
		if err != nil {
			return nil, err
		}
		// Run the process
		go func() {
			pm.pmlogger.Debugf("Starting %s.\n", proc.config.CMD)
			endstate := proc.runProcess(mainProcess)
			signalreplicator.Remove(proc.sigChan)
			pm.EndList = append(pm.EndList, endstate)
			pm.pmlogger.Debugf("%s has terminated.\n", proc.config.CMD)
			pm.tumble <- true
			pm.wg.Done()
		}()
	}

	exitStatusTextChan := make(chan string, 1)
	go pm.waitMain(exitStatusTextChan)

	return exitStatusTextChan, nil
}

func (pm *ProcessManger) waitMain(output chan string) {
	pm.pmlogger.Debugln("Starting wait on waitgroup for main processes.")
	// Wait until all the waitgroups have closed off.
	// This means all the processes have stopped.
	pm.wg.Wait()
	pm.pmlogger.Debugln("passed waitgroup for main processes.")
	// Write to the queue and close it to signal this is complete.
	output <- pm.exitStatusFormatter()
	close(output)
}

func (pm *ProcessManger) exitStatusFormatter() string {
	b, err := json.Marshal(pm.EndList)
	if err != nil {
		pm.pmlogger.Debugf("Error generating end state. Error: %s\n", err)
	}
	return string(b)
}

// Setup Process will link create the process object and also link the stdout and stderr.
// An error is returned if anything fails.
func (pm *ProcessManger) setupProcess(proc *Process) error {
	execProc, stdout, stderr, err := createRunableProcess(proc.config.CMD, proc.config.Args, proc.sigChan)
	if err != nil {
		return err
	}
	proc.proc = execProc
	procStdOut := stdout
	procStdErr := stderr
	proc.closePipesChan = pm.redirectOutput(procStdOut, procStdErr, proc.config.LoggerConfig)

	return nil
}

// createRunableProcess will create a process that can be run later. It will also make sure that the
// output pipes have been linked.
// We can use this in processes managed by the process manager or secret processes which are just
// single processes.
func createRunableProcess(
	command string,
	arguments []string,
	signalChan chan os.Signal,
) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	signalreplicator.Register(signalChan)
	execProc := exec.Command(command, arguments...)

	stdout, err := execProc.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to stdout pipe. Error: %s", err)
	}

	stderr, err := execProc.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to stderr pipe. Error: %s", err)
	}

	return execProc, stdout, stderr, nil
}

// redirectOutput will take the pipes of the process and redirect it to the logger for the process
func (pm *ProcessManger) redirectOutput(stdout, stderr io.ReadCloser, config configfile.LoggingConfig) chan bool {
	closePipetrigger := make(chan bool, 1)
	go func() {
		<-closePipetrigger
		defer stdout.Close()
		defer stderr.Close()
	}()

	stdOutScanner := bufio.NewScanner(stdout)
	stdErrScanner := bufio.NewScanner(stderr)

	newLog := func(from processlogger.Pipe, msg string) processlogger.LogMessage {
		return processlogger.LogMessage{
			Source:  config.ProcessName,
			Pipe:    from,
			Config:  config,
			Message: msg,
		}
	}

	go func() {
		for stdOutScanner.Scan() {
			pm.logger.Submit(newLog(processlogger.STDOUT, stdOutScanner.Text()+"\n"))
		}
	}()
	go func() {
		for stdErrScanner.Scan() {
			pm.logger.Submit(newLog(processlogger.STDOUT, stdErrScanner.Text()+"\n"))
		}
	}()

	return closePipetrigger
}
