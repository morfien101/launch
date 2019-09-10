// Package devnull is a black hole logger. It implements the interfaces to be a logger
// but will discard the data it gets.
package devnull

import (
	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/processlogger"
)

const (
	// LoggerTag is the tag that should be used in config files to access this logger
	LoggerTag = "devnull"
)

// DevNull is a discard logger. It will not process logs. Usefull for junk logs
type DevNull struct {
	running bool
}

func init() {
	processlogger.RegisterLogger(LoggerTag, func() processlogger.Logger {
		return &DevNull{}
	})
}

// Shutdown satisfies the processlogger.Logger interface but effectively does nothing
func (d *DevNull) Shutdown() chan error {
	shutdownChan := make(chan error, 1)
	shutdownChan <- nil
	close(shutdownChan)
	d.running = false
	return shutdownChan
}

// RegisterConfig does nothing here.
func (d *DevNull) RegisterConfig(_ configfile.LoggingConfig, _ configfile.DefaultLoggerDetails) error {
	return nil
}

// Start satisfies the processlogger.Logger interface but effectively does nothing
func (d *DevNull) Start() error {
	d.running = true
	return nil
}

// Submit satisfies the processlogger.Logger interface but effectively does nothing
func (d *DevNull) Submit(_ processlogger.LogMessage) {}
