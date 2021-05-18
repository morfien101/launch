package processlogger

import (
	"testing"
	"time"

	"github.com/morfien101/launch/configfile"
	"github.com/silverstagtech/gotracer"
)

type trace struct {
	logger *gotracer.Tracer
}

func (tr *trace) RegisterConfig(configfile.LoggingConfig, configfile.DefaultLoggerDetails) error {
	return nil
}
func (tr *trace) Start() error {
	return nil
}
func (tr *trace) Shutdown() chan error {
	return nil
}
func (tr *trace) Submit(msg LogMessage) {
	tr.logger.Send(msg.Message)
}

func (tr *trace) Logs() []string {
	return tr.logger.Show()
}

func TestLogger(t *testing.T) {
	// Registration needs to happen BEFORE the logger is created.
	// This is to mimic init function calls which don't happen in tests
	tracer1 := &trace{
		logger: gotracer.New(),
	}
	tracer2 := &trace{
		logger: gotracer.New(),
	}
	RegisterLogger("tracer1", func() Logger {
		return tracer1
	})
	RegisterLogger("tracer2", func() Logger {
		return tracer2
	})

	proc1 := &configfile.Process{
		Name: "Test1",
		LoggerConfig: configfile.LoggingConfig{
			Engine:      "tracer1",
			ProcessName: "test1",
		},
	}

	proc2 := &configfile.Process{
		Name: "Test2",
		LoggerConfig: configfile.LoggingConfig{
			Engine:      "tracer2",
			ProcessName: "test2",
		},
	}

	conf := configfile.Config{
		ProcessManager: configfile.ProcessManager{
			LoggerConfig: configfile.LoggingConfig{
				Engine:      "tracer1",
				ProcessName: "test1",
			},
		},
		Processes: configfile.Processes{
			InitProcesses: []*configfile.Process{},
			MainProcesses: []*configfile.Process{proc1, proc2},
		},
		DefaultLoggerConfig: configfile.DefaultLoggerDetails{},
	}
	logManager := New(10, conf.DefaultLoggerConfig)
	err := logManager.StartLoggers(conf.Processes, conf.ProcessManager.LoggerConfig)
	if err != nil {
		t.Logf("Got an error starting the logger. Error: %s", err)
	}

	t.Logf("Available Loggers: %v", logManager.availableLoggers)
	t.Logf("Active Loggers: %v", logManager.activeLoggers)

	msg1 := LogMessage{
		Source:  "proc1",
		Config:  proc1.LoggerConfig,
		Message: "Message 1",
	}
	msg2 := LogMessage{
		Source:  "proc2",
		Config:  proc2.LoggerConfig,
		Message: "Message 2",
	}

	logManager.Submit(msg1)
	logManager.Submit(msg2)

	t.Logf("Number of queues: %d", len(logManager.activeLoggerQ))

	if len(logManager.activeLoggerQ) != 2 {
		t.Logf("Failed to create correct number of channels for logs. Want %d, Got: %d", 2, len(logManager.activeLoggerQ))
		t.Fail()
	}
	// Give the logger a bit of time to flush the logs
	// It can't be too much as we need to make sure its still fast enough
	time.Sleep(time.Millisecond * 2)

	// Tracer 1 will get 2 message as we send to more than 1 logging engine on a process
	if tracer1.logger.Len() != 1 {
		t.Logf("Tracer 1 does not have the correct number of messages. Want: %d, Got: %d", 1, tracer1.logger.Len())
		t.Fail()
	}
	if tracer2.logger.Len() != 1 {
		t.Logf("Tracer 2 does not have the correct number of messages. Want: %d, Got: %d", 1, tracer2.logger.Len())
		t.Fail()
	}
}

func TestUndefinedLogger(t *testing.T) {
	proc1 := &configfile.Process{
		Name: "Test1",
		LoggerConfig: configfile.LoggingConfig{
			Engine:      "tracer1",
			ProcessName: "test1",
		},
	}
	conf := configfile.Config{
		ProcessManager: configfile.ProcessManager{
			LoggerConfig: configfile.LoggingConfig{
				Engine:      "Potato",
				ProcessName: "test1",
			},
		},
		Processes: configfile.Processes{
			InitProcesses: []*configfile.Process{},
			MainProcesses: []*configfile.Process{proc1},
		},
		DefaultLoggerConfig: configfile.DefaultLoggerDetails{},
	}
	logManager := New(10, conf.DefaultLoggerConfig)
	err := logManager.StartLoggers(conf.Processes, conf.ProcessManager.LoggerConfig)
	t.Logf("Bad logger error: %s", err)
	if err == nil {
		t.Logf("A process with a bad logger engine did not cause the logger to fail.")
		t.Fail()
	}
}
