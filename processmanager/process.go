package processmanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/morfien101/launch/internallogger"

	"github.com/morfien101/launch/configfile"
)

// Process is used to hold config and state of a process
type Process struct {
	pmlogger internallogger.IntLogger
	sync.RWMutex
	config         *configfile.Process
	exiting        bool
	exited         bool
	exitcode       int
	shutdown       chan bool
	sigChan        chan os.Signal
	loggerTag      string
	proc           *exec.Cmd
	closePipesChan chan bool
}

func (p *Process) running() bool {
	p.RLock()
	defer p.RUnlock()
	if p.exited {
		return p.exited
	}

	return p.exiting
}

func (p *Process) processStartDelay() {
	if p.config.StartDelay != 0 {
		p.pmlogger.Printf("Process start is configured for delayed start of %d seconds.\n", p.config.StartDelay)
		time.Sleep(time.Second * time.Duration(p.config.StartDelay))
	}
}

// runProcess will spawn a child process and return only once that child
// has exited either good or bad.
func (p *Process) runProcess(processType string) *processEnd {
	finalState := &processEnd{
		Name:        p.config.CMD,
		ProcessType: processType,
		ExitCode:    -1,
	}

	p.processStartDelay()

	if err := p.proc.Start(); err != nil {
		p.exitcode = 1
		finalState.Error = err
		finalState.ExitCode = 1
		return finalState
	}

	// Wait for the process to finish
	done := make(chan error, 1)
	go func() {
		done <- p.proc.Wait()
		// Close the pipes that redirect std out and err
		p.closePipesChan <- true
	}()
	var timeoutError error = nil

	// Wait for signals
	go func() {
		exitTimeout := make(chan bool, 1)
		for {
			select {
			case signal := <-p.sigChan:
				p.Lock()
				p.exiting = true
				p.Unlock()
				// Collect signals and pass them onto the main command that we are running.
				err := p.proc.Process.Signal(signal)
				if err != nil {
					// Failed to send signal to process
					// Sent to done because the process will never end
					done <- fmt.Errorf("failed to send signal %s to running instance of %s. Allowing crash when process manager dies", signal, p.config.CMD)
				}
				// Signals SIGTERM, SIGINT and SIGKILL will cause the app to stop.
				// This means we need to kill the app if it fails to stop itself.
				// The app getting the signal should be enough to stop it. There
				// is a race condition here though. If the app is killed and we
				// read its state just before it gets updated. We will get an
				// incorrect state. Very low probability so not fixing unless required.
				switch signal {
				case syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL:
					// We need to timeout after a specified time.
					// If no time is specified we give it a long timer of 30 seconds.
					// This should be long enough for 99% of processes.
					if p.config.TermTimeout == 0 {
						p.config.TermTimeout = 30
					}

					time.AfterFunc(time.Duration(p.config.TermTimeout)*time.Second, func() {
						exitTimeout <- p.running()
					})
				}
			case timeout := <-exitTimeout:
				if timeout {
					err := p.proc.Process.Kill()
					if err != nil {
						timeoutError = fmt.Errorf("failed to terminate %s", p.config.CMD)
					}
				}
				break
			}
		}
	}()

	// Wait here to get an err.
	// It could be nil which would indicate that the process exited without an error.
	// It could be an exec.ExitError which would indicate that the process terminated badly.
	// We will try to get the error but its not possible in all OSs.
	// This should work on Linux and Windows. See below for more details:
	// https://stackoverflow.com/questions/10385551/get-exit-code-go
	finalState.Error = <-done
	// If the process is killed because of a timeout, we need to indicate that.
	// We still get a message on the channel because the process ends.
	if timeoutError != nil {
		finalState.Error = timeoutError
	}

	p.exited = true

	finalState.ExitCode = readExitError(finalState.Error)
	return finalState
}

// ReadExitError attempts to get the correct exit code from the process
func readExitError(e error) int {
	if exiterr, ok := e.(*exec.ExitError); ok {
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
			return exitstatus
		} else {
			return 1
		}
	}
	return 0
}

// RunSecretProcess will execute the secret process and pass back the STDOUT and STDERR.
// Any error will indicate that the process did not complete successfully.
func RunSecretProcess(secretConfig configfile.SecretProcess, logger internallogger.IntLogger) (stdoutOutput, stderrOutout string, err error) {
	logger.Printf("Collecting secrets from process %s\n", secretConfig.Name)
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second*time.Duration(secretConfig.TermTimeout),
	)
	defer cancel()
	cmd := exec.CommandContext(ctx, secretConfig.CMD, secretConfig.Args...)

	stdout, err := cmd.Output()
	if err != nil {
		stdout := []byte{}
		if exitCode := readExitError(err); exitCode != 0 {
			stdout = err.(*exec.ExitError).Stderr
		}
		return "", string(stdout), err
	}

	return string(stdout), "", nil
}
