// Package console is a logger that prints to the console where the
// process manager is running. It is intented to be used for development and
// debugging purposes.
package console

import (
	"fmt"
	"os"

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/processlogger"
)

const (
	//LoggerTag will be used to call this package
	LoggerTag = "console"
)

// Console is a logger that will output to the local stdout and stderr
type Console struct{}

// New will return a new pointer to a Console logger
func init() {
	processlogger.RegisterLogger(LoggerTag, func() processlogger.Logger {
		return &Console{}
	})
}

// RegisterConfig does nothing here.
func (c *Console) RegisterConfig(_ configfile.LoggingConfig, _ configfile.DefaultLoggerDetails) error {
	return nil
}

// Start will start the logging engine. There is nothing to do in this package
func (c *Console) Start() error {
	return nil
}

// IsStarted will let the caller know if it needs to start this service
func (c *Console) IsStarted() bool {
	return true
}

// Shutdown does not really need to do anything in this package.
// It returns a chan bool preloaded with a true to signal that
// the connections are closed.
func (c *Console) Shutdown() chan error {
	ch := make(chan error, 1)
	ch <- nil
	close(ch)
	return ch
}

// Submit will consume a processlogger.LogMessage and send it to the right pipe.
func (c *Console) Submit(msg processlogger.LogMessage) {
	m := fmt.Sprintf("%s: %s", msg.Source, msg.Message)
	if msg.Pipe == processlogger.STDERR {
		c.stdErr(m)
	}
	if msg.Pipe == processlogger.STDOUT {
		c.stdOut(m)
	}
}

// StdOut will copy the message to Stdout
func (c *Console) stdOut(msg string) error {
	if _, err := os.Stdout.WriteString(msg); err != nil {
		return err
	}
	return nil
}

// StdErr will copy the message to Stderr
func (c *Console) stdErr(msg string) error {
	if _, err := os.Stderr.WriteString(msg); err != nil {
		return err
	}
	return nil
}
