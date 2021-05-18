package internallogger

import (
	"fmt"

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/processlogger"
	"github.com/morfien101/launch/processlogger/console"
)

// NewFakeLogger will return a logger that can be used for testing in other packages.
// It will just print to the console.
func NewFakeLogger() *InternalLogger {
	cfg := configfile.Config{
		ProcessManager: configfile.ProcessManager{
			LoggerConfig: configfile.LoggingConfig{
				Engine:      "console",
				ProcessName: "FakeInternalLogger",
			},
		},
		DefaultLoggerConfig: configfile.DefaultLoggerDetails{
			Config: configfile.LoggingConfig{
				Engine: "console",
			},
		},
	}

	processlogger.RegisterLogger(console.LoggerTag, func() processlogger.Logger {
		return &console.Console{}
	})

	lm := processlogger.New(2, cfg.DefaultLoggerConfig)
	err := lm.StartLoggers(configfile.Processes{}, cfg.ProcessManager.LoggerConfig)
	if err != nil {
		fmt.Printf("Error starting fake logger. Error: %s\n", err)
	}
	return New(cfg.ProcessManager.LoggerConfig, lm)
}
