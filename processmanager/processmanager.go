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
)

// ProcessManger holds the config and state of the running processes.
type ProcessManger struct {
	config        configfile.Processes
	logger        *processlogger.LogManager
	pmlogger      internallogger.IntLogger
	Signals       chan os.Signal
	mainProcesses []*Process
	wg            sync.WaitGroup
	tumble        chan bool
	shuttingDown  bool

	mu         sync.Mutex
	ExitStates []*processExitState
}

// New will create a ProcessManager with the supplied config and return it
func New(
	config configfile.Processes,
	logManager *processlogger.LogManager,
	pmlogger internallogger.IntLogger,
	signalChan chan os.Signal,
) *ProcessManger {
	pm := &ProcessManger{
		config:   config,
		logger:   logManager,
		pmlogger: pmlogger,
		Signals:  signalChan,
		tumble:   make(chan bool, 1),
	}

	// Once we get a single process fail we shutdown everything.
	// A SIGTERM is sent to the master signal channel
	pmlogger.Debugln("Starting terminator go func for when signals arrive")
	go pm.terminator()

	return pm
}

func (pm *ProcessManger) addExitState(pe *processExitState) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.ExitStates = append(pm.ExitStates, pe)
}

func (pm *ProcessManger) signalRecplicator() {
	for {
		select {
		case signalIn := <-pm.Signals:
			f := func(procs []*Process) {
				for _, proc := range procs {
					if proc.running() {
						proc.sigChan <- signalIn
					}
				}
			}
			f(pm.mainProcesses)

		}
	}
}

func (pm *ProcessManger) terminator() {
	for {
		select {
		case _ = <-pm.tumble:
			// If the process manager is already shutting down we will get a few
			// signals from the processes as they turn off. We can safely discard
			// them since we expect them.
			if pm.shuttingDown {
				continue
			}
			// If we are not in shutdown mode trigger it. Them mark it as shutting down.
			pm.shuttingDown = true
			// We only need to send signals to propagate if we have started some mains.
			if pm.mainProcesses != nil {
				pm.Signals <- syscall.SIGTERM
			}
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
		// Create a process object
		proc := newProcess(procConfig, pm.pmlogger)
		pm.pmlogger.Debugf("Attempting to run %s.\n", proc.config.CMD)
		// setup logging hooks
		// If this fails we can't carry on.
		err := pm.setupProcess(proc)
		if err != nil {
			return "", err
		}
		signalReplicatorFuncChan := make(chan struct{}, 1)
		destroySignalReplicator := func() {
			signalReplicatorFuncChan <- struct{}{}
		}
		go func() {
			for {
				select {
				case sig := <-pm.Signals:
					proc.sigChan <- sig
				case <-signalReplicatorFuncChan:
					return
				}
			}
		}()

		// Run the process
		endstate, ok := proc.runProcess(initProcess)
		pm.pmlogger.Debugf("Finished running %s.\n", proc.config.CMD)
		pm.addExitState(&endstate)
		if !ok {
			pm.pmlogger.Debugln("The last init command failed. Stack will now tumble.")
			pm.tumble <- true

			err := fmt.Errorf("Process %s failed. Error reported: %s", procConfig.Name, endstate.Error)
			output := make(chan string, 1)
			pm.exitStatusPrinter(output)

			destroySignalReplicator()

			return <-output, err
		}
		destroySignalReplicator()
	}
	return "", nil
}

// RunMainProcesses will start the processes listed in sequential order.
// Processes are expected to start and stay running. A failure of one
// will cause all the ProcessManger to send termination signals to all
// remaining and subsequently kill the processes manager.
func (pm *ProcessManger) RunMainProcesses() (chan string, error) {
	// We need to wait for signals and repeat them into the processes
	pm.pmlogger.Debugln("Starting signal catcher go func")
	go pm.signalRecplicator()

	pm.pmlogger.Println("Starting Main Processes")
	// Start the main processes
	for _, procConfig := range pm.config.MainProcesses {
		// Create a process object
		pm.wg.Add(1)
		pm.pmlogger.Debugf("Adding %s to the list of main processes.\n", procConfig.CMD)
		proc := newProcess(procConfig, pm.pmlogger)
		pm.mainProcesses = append(pm.mainProcesses, proc)

		// setup logging hooks
		// If this fails we can't carry on.
		err := pm.setupProcess(proc)
		if err != nil {
			return nil, err
		}
		// Run the process
		go func() {
			// Start Delay is only processed at the beinging of Launches life cycle.
			proc.processStartDelay()

			runProc := func() bool {
				pm.pmlogger.Debugf("Starting %s.\n", proc.config.CMD)
				endstate, ok := proc.runProcess(mainProcess)
				pm.addExitState(&endstate)
				pm.pmlogger.Debugf("%s has terminated.\n", proc.config.CMD)
				return ok
			}

			firstRun := true
			for {
				if firstRun {
					firstRun = false
					runProc()
					continue
				}
				if proc.restartAllowed() {
					ok := runProc()
					// setup logging hooks
					// If this fails we can't carry on.
					// We can't throw the error here, but we can log it.
					err := pm.setupProcess(proc)
					if err != nil {
						pm.pmlogger.Errorf("Failed to link process pipes for %s. Error: %s", proc.config.Name, err)
						break
					}
					if !ok {
						proc.addRestart()
					}
					continue
				}
				break
			}

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
	pm.wg.Wait()
	pm.pmlogger.Debugln("passed waitgroup for main processes.")
	pm.exitStatusPrinter(output)
}

func (pm *ProcessManger) exitStatusPrinter(output chan string) {
	b, err := json.Marshal(pm.ExitStates)
	if err != nil {
		pm.pmlogger.Debugf("Error generating end state. Error: %s\n", err)
	}
	output <- string(b)
}

// Setup Process will link create the process object and also link the stdout and stderr.
// An error is returned if anything fails.
func (pm *ProcessManger) setupProcess(proc *Process) error {
	proc.reset()
	proc.proc = exec.Command(proc.config.CMD, proc.config.Args...)

	procStdOut, err := proc.proc.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to connect to stdout pipe. Error: %s", err)
	}

	procStdErr, err := proc.proc.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to connect to stderr pipe. Error: %s", err)
	}

	proc.closePipesChan = pm.redirectOutput(procStdOut, procStdErr, proc.config.LoggerConfig)

	return nil
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
