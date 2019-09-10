// Package internallogger is used to log messages out for the process manager itself.
// There is a normal logger which logs to stdout and stderr and a debug logger which
// logs to stdout only if debug logging is turned on.
package internallogger

import (
	"fmt"

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/processlogger"
)

const (
	// processManagerSource is used to tag the messages when they are sent on to the logging engine
	// required.
	processManagerSource = "launch_process_manager"
)

//IntErrLogger is a logger that will log at Error level
type IntErrLogger interface {
	Errorf(format string, args ...interface{})
	Errorln(s interface{})
}

// IntStdLogger is a logger that will minic fmt.Printf or fmt.Println
type IntStdLogger interface {
	Printf(format string, args ...interface{})
	Println(s interface{})
}

// IntDebugLogger is a logger that will log at standard level but only
// if the debug toggle is turned on.
type IntDebugLogger interface {
	Debugf(format string, args ...interface{})
	Debugln(s interface{})
	DebugOn(on bool)
}

// IntLogger is a fully implemented internal logger. It must have Err, Std and Debug logging.
type IntLogger interface {
	IntDebugLogger
	IntStdLogger
	IntErrLogger
}

// InternalLogger is a logger that the process manager will use internally.
// This logger has a debug bool value which will dictate if the debug logging
// will be produced.
type InternalLogger struct {
	debug      bool
	config     configfile.LoggingConfig
	logManager *processlogger.LogManager
}

// New requires a copy of the config for logging and a log manager to forward logs to.
// It will return a *InternalLogger
func New(config configfile.LoggingConfig, logManager *processlogger.LogManager) *InternalLogger {
	return &InternalLogger{
		config:     config,
		logManager: logManager,
	}
}

// Printf mimics the functionality of fmt.Printf and sends the result to STDOUT
func (il *InternalLogger) Printf(format string, args ...interface{}) {
	il.submit(il.newMsg(fmt.Sprintf(format, args...), processlogger.STDOUT))
}

// Println mimics the functionality of fmt.Println and sends the result to STDOUT
func (il *InternalLogger) Println(s interface{}) {
	il.submit(il.newMsg(fmt.Sprintln(s), processlogger.STDOUT))
}

// Errorf mimics the functionality of fmt.Printf and sends the result to STDERR
func (il *InternalLogger) Errorf(format string, args ...interface{}) {
	il.submit(il.newMsg(fmt.Sprintf(format, args...), processlogger.STDERR))
}

// Errorln mimics the functionality of fmt.Println and sends the result to STDERR
func (il *InternalLogger) Errorln(s interface{}) {
	il.submit(il.newMsg(fmt.Sprintln(s), processlogger.STDERR))
}

// Debugf mimics the functionality of fmt.Printf and sends the result to STDOUT
// if the debug toggle is true
func (il *InternalLogger) Debugf(format string, args ...interface{}) {
	if il.debug {
		il.submit(il.newMsg(fmt.Sprintf(format, args...), processlogger.STDERR))
	}
}

// Debugln mimics the functionality of fmt.Printf and sends the result to STDOUT
// if the debug toggle is true
func (il *InternalLogger) Debugln(s interface{}) {
	if il.debug {
		il.submit(il.newMsg(fmt.Sprintln(s), processlogger.STDERR))
	}
}

// DebugOn is used to turn debug logging on and off.
func (il *InternalLogger) DebugOn(on bool) {
	il.debug = on
}

// newMsg creates a new LogMessage with the required resources and returns a pointer to it.
func (il *InternalLogger) newMsg(msg string, pipe processlogger.Pipe) *processlogger.LogMessage {
	return &processlogger.LogMessage{
		Source:  processManagerSource,
		Pipe:    pipe,
		Config:  il.config,
		Message: msg,
	}
}

// submit wraps the Submit function of the logmanager which will consume the message and route it.
func (il *InternalLogger) submit(lMsg *processlogger.LogMessage) {
	il.logManager.Submit(*lMsg)
}
