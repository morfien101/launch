package processmanager

import (
	"os"
	"os/exec"
	"sync"

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
	return p.exiting
}
