package processlogger

import "github.com/morfien101/launch/configfile"

// Logger is a logger interface that can start stop and send logs.
// This is the shipper interface.
type Logger interface {
	RegisterConfig(configfile.LoggingConfig, configfile.DefaultLoggerDetails) error
	Start() error
	Shutdown() chan error
	Submit(LogMessage)
}

// LogsManager is something that can start, stop and submit logs.
type LogsManager interface {
	StartLoggers(configfile.Processes, configfile.LoggingConfig) error
	Submit(log LogMessage)
	Shutdown() []error
}
