package processmanager

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/morfien101/launch/internallogger"

	"github.com/morfien101/launch/configfile"
)

const (
	initProcess = "init"
	mainProcess = "main"
)

type processExitState struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	ProcessType string `json:"type"`
	Error       error  `json:"runtime_error,omitempty"`
	ExitCode    int    `json:"exit_code"`
}

func newProcessExitState(name, command, procType string) processExitState {
	return processExitState{
		Name:        name,
		Command:     command,
		ProcessType: procType,
		ExitCode:    -1,
	}
}

// Process is used to hold config and state of a process
type Process struct {
	pmlogger internallogger.IntLogger
	sync.RWMutex
	config         *configfile.Process
	exited         bool
	exitcode       int
	sigChan        chan os.Signal
	loggerTag      string
	proc           *exec.Cmd
	closePipesChan chan bool
	blockRestarts  bool
	restartCounter int
}

func newProcess(cfg *configfile.Process, pmlogger internallogger.IntLogger) *Process {
	return &Process{
		config:   cfg,
		pmlogger: pmlogger,
		sigChan:  make(chan os.Signal, 1),
	}
}

func (p *Process) reset() {
	p.Lock()
	defer p.Unlock()
	p.exited = false
	p.exitcode = 0
	p.proc = nil
}

func (p *Process) restartAllowed() bool {
	switch {
	case p.blockRestarts:
		return false
	case p.config.RestartCount <= 0:
		return false
	case p.restartCounter <= p.config.RestartCount:
		return true
	default:
		return false
	}
}

func (p *Process) addRestart() {
	p.restartCounter++
}

func (p *Process) resetRestarts() {
	p.restartCounter = 0
}

func (p *Process) running() bool {
	p.RLock()
	defer p.RUnlock()
	return !p.exited
}

func (p *Process) processStartDelay() {
	if p.config.StartDelay != 0 {
		p.pmlogger.Debugf("Process start is configured for delayed start of %d seconds.\n", p.config.StartDelay)
		time.Sleep(time.Second * time.Duration(p.config.StartDelay))
	}
}

// runProcess will spawn a child process and return only once that child
// has exited either good or bad. A processEnd is returned that contains information about the process.
// A bool is returned to indicate if the process exited in a good state.
func (p *Process) runProcess(processType string) (processExitState, bool) {
	finalState := newProcessExitState(p.config.Name, p.config.CMD, processType)

	if err := p.proc.Start(); err != nil {
		p.exitcode = 1
		finalState.Error = err
		finalState.ExitCode = 1
		return finalState, false
	}

	// Wait for the process to finish
	errChan := make(chan error, 1)
	// Wait for command to terminate
	go p.terminatedNotification(errChan)

	// Wait for signals
	stopSingalWatcher := make(chan struct{}, 1)
	go p.waitForSignal(errChan, stopSingalWatcher)

	// Wait here to get an err.
	// It could be nil which would indicate that the process exited without an error.
	// It could be an exec.ExitError which would indicate that the process terminated badly.
	finalState.Error = <-errChan
	p.exited = true
	stopSingalWatcher <- struct{}{}

	// Try to get the error. It's not possible in all OSs.
	// This should work on Linux and Windows. See below for more details:
	// https://stackoverflow.com/questions/10385551/get-exit-code-go
	if exiterr, ok := finalState.Error.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			exitstatus := status.ExitStatus()
			if exitstatus == -1 {
				exitstatus = 1
			}
			p.exitcode = exitstatus
		} else {
			p.pmlogger.Debugf("Could not determine actual exit code for %s. Assuming 1 because it failed.\n", p.config.Name)
			p.exitcode = 1
		}
	}

	finalState.ExitCode = p.exitcode
	ok := true
	if finalState.ExitCode != 0 {
		ok = false
	}

	return finalState, ok
}

func (p *Process) waitForSignal(errChan chan error, stop chan struct{}) {
	p.pmlogger.Debugf("staring signal watch for %s\n", p.config.Name)
	for {
		select {
		case signal := <-p.sigChan:
			p.processExternalSignal(signal, errChan)
		case <-stop:
			break
		}
	}
}

func (p *Process) terminatedNotification(errChan chan error) {
	errChan <- p.proc.Wait()
	// Close the pipes that redirect std out and err
	p.closePipesChan <- true
}

func (p *Process) processExternalSignal(signal os.Signal, errChan chan error) {
	// Collect signals and pass them onto the process if running.
	p.pmlogger.Printf("Got signal %s, forwarding onto %s\n", signal, p.config.Name)
	err := p.proc.Process.Signal(signal)
	if err != nil {
		// Failed to send signal to process
		errChan <- fmt.Errorf("Failed to send signal %s to running instance of %s. Allowing crash when process manager dies", signal, p.config.CMD)
	}

	if signal == syscall.SIGINT || signal == syscall.SIGTERM {
		// Signals SIGTERM and SIGINT will cause the app to stop.
		// Termination signals are respected and applications will not have the option to restart if a signal
		// from outside is caught.
		p.blockRestarts = true

		// We need to timeout after a specified time to stop zombie applications from blocking the termination of the stack.
		if p.running() {
			go func() {
				// This will always fire. It is used to break this for loop in the timeout case
				// below. The value passed down the channel will determine if any action needs
				// to take place.
				p.pmlogger.Debugf("Starting forceful termination timer for %s\n", p.config.Name)
				time.AfterFunc(time.Duration(p.config.TermTimeout)*time.Second, func() {
					// If a process is still running after X seconds then we just terminate it.
					if p.running() {
						p.pmlogger.Printf("Forcefully killing process %s because termination timeout has been reached.\n", p.config.Name)
						err := p.proc.Process.Kill()
						if err != nil {
							errChan <- fmt.Errorf("Failed to terminate %s. ErrorL %s", p.config.CMD, err)
						}
					}
				})
			}()
		}
	}
}
