package configfile

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/Flaque/filet"
)

func TestExampleConfig(t *testing.T) {
	out, err := ExampleConfigFile()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	t.Log(out)
}

func TestNewConfig(t *testing.T) {
	out, err := ExampleConfigFile()
	if err != nil {
		t.Fatal(err)
	}

	testingConfigFile := filet.TmpFile(t, "", out)
	_, err = New(testingConfigFile.Name())
	if err != nil {
		t.Fatalf("Failed to create a config struct. Error: %s", err)
	}
}

func TestTemplating(t *testing.T) {
	os.Setenv("SYSLOG_SERVER", "syslog.test.local")
	testYaml := `default_logger_config:
  logging_config:
    engine: syslog
    syslog:
      program_name: example_service
      address: {{ env "SYSLOG_SERVER" }}
      protocol: {{ default ( env "SYSLOG_PROTOCOL" ) "udp" }}`

	testingfile := filet.TmpFile(t, "", testYaml)
	conf, err := New(testingfile.Name())
	if err != nil {
		t.Fatalf("Failed to generate templated configuration. Got Error: %s", err)
	}

	tests := []struct {
		want     string
		got      string
		function string
	}{
		{
			want:     "syslog.test.local",
			got:      conf.DefaultLoggerConfig.Config.Syslog.Address,
			function: `{{ env "SYSLOG_SERVER" }}`,
		},
		{
			want:     "udp",
			got:      conf.DefaultLoggerConfig.Config.Syslog.ConnectionType,
			function: `{{ default ( env "SYSLOG_PROTOCOL" ) "udp" }}`,
		},
	}

	for _, test := range tests {
		if test.want != test.got {
			t.Logf("%s failed. Got %s, Want: %s",
				test.function,
				test.got,
				test.want,
			)
			t.Fail()
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cf := Config{
		Processes: Processes{
			InitProcesses: []*Process{
				&Process{
					Name: "TestInit_1",
					CMD:  "/bin/false",
					Args: []string{"arg1", "arg2"},
				},
			},
			MainProcesses: []*Process{
				&Process{
					Name: "TestMain_1",
					CMD:  "/bin/false",
					Args: []string{"arg1", "arg2"},
				},
			},
		},
	}

	// Test Default Loggers
	cf.setDefaultLoggerConfig()
	if cf.DefaultLoggerConfig.Config.Engine == "" {
		t.Logf("Setting the default logger config resulted in a empty engine")
		t.Fail()
	}

	// Test generated logging for processes from default logger
	cf.setDefaultProcessLogger()
	if cf.Processes.InitProcesses[0].LoggerConfig.Engine != cf.DefaultLoggerConfig.Config.Engine {
		t.Logf("Trying to set the default logger engine for a process in the Init branch did not work")
		t.Fail()
	}
	if cf.Processes.InitProcesses[0].LoggerConfig.ProcessName != cf.Processes.InitProcesses[0].Name {
		t.Logf("Trying to set the default logger process_name for a process in the Init branch did not work")
		t.Fail()
	}
	if cf.Processes.MainProcesses[0].LoggerConfig.Engine != cf.DefaultLoggerConfig.Config.Engine {
		t.Logf("Trying to set the default logger for a process in the Main branch did not work")
		t.Fail()
	}
	if cf.Processes.MainProcesses[0].LoggerConfig.ProcessName != cf.Processes.MainProcesses[0].Name {
		t.Logf("Trying to set the default logger process_name for a process in the Main branch did not work")
		t.Fail()
	}

	// Test ProcessManager config generation
	cf.setDefaultProcessManager()
	if cf.ProcessManager.LoggerConfig.Engine != defaultProcessManager.LoggerConfig.Engine {
		t.Logf("Process manager did not get expect default configuration set")
		t.Fail()
	}

	// visual readout
	out, err := yaml.Marshal(cf)
	if err != nil {
		t.Logf("Failed to marshal the config to yaml. Error: %s", err)
		t.Fail()
	}
	t.Log("\n", string(out))
}

func TestDefaultConfigNotRequired(t *testing.T) {
	pmLogger := "proc_logging"
	defaultLogger := "default_logger"
	initLogger := "init_logger"
	initLoggingName := "init_proc"

	cf := Config{
		Processes: Processes{
			InitProcesses: []*Process{
				&Process{
					Name: "TestInit_1",
					CMD:  "/bin/false",
					Args: []string{"arg1", "arg2"},
					LoggerConfig: LoggingConfig{
						Engine:      initLogger,
						ProcessName: initLoggingName,
					},
				},
			},
		},
		DefaultLoggerConfig: DefaultLoggerDetails{
			Config: LoggingConfig{
				Engine: defaultLogger,
			},
		},
		ProcessManager: ProcessManager{
			LoggerConfig: LoggingConfig{
				Engine: pmLogger,
			},
		},
	}

	// Test Default Loggers
	cf.setDefaultLoggerConfig()
	cf.setDefaultProcessManager()
	cf.setDefaultProcessLogger()

	if cf.ProcessManager.LoggerConfig.Engine != pmLogger {
		t.Logf("process_manager logging config was over written. Got: %s, Want: %s", cf.ProcessManager.LoggerConfig.Engine, pmLogger)
		t.Fail()
	}
	if cf.DefaultLoggerConfig.Config.Engine != defaultLogger {
		t.Logf("default logger engine was overwritten. Got: %s, Want: %s", cf.DefaultLoggerConfig.Config.Engine, defaultLogger)
		t.Fail()
	}
	if cf.Processes.InitProcesses[0].LoggerConfig.ProcessName != initLoggingName {
		t.Logf("init process logging name was overwritten. Got: %s, Want: %s", cf.Processes.InitProcesses[0].LoggerConfig.ProcessName, initLoggingName)
		t.Fail()
	}
	if cf.Processes.InitProcesses[0].LoggerConfig.Engine != initLogger {
		t.Logf("init process logging name was overwritten. Got: %s, Want: %s", cf.Processes.InitProcesses[0].LoggerConfig.Engine, initLogger)
		t.Fail()
	}
}

func TestDefaultTimeoutForProcesses(t *testing.T) {
	cf := Config{
		Processes: Processes{
			InitProcesses: []*Process{
				&Process{
					Name: "TestInit_1",
					CMD:  "/bin/false",
					Args: []string{"arg1", "arg2"},
				},
			},
			MainProcesses: []*Process{
				&Process{
					Name: "TestMain_1",
					CMD:  "/bin/false",
					Args: []string{"arg1", "arg2"},
				},
			},
		},
	}

	cf.setDefaultProcessTimeout()

	if cf.Processes.InitProcesses[0].TermTimeout != defaultProcTimeout {
		t.Logf("Default timeout on init process was not set as expected. Got: %d, Want: %d.", cf.Processes.InitProcesses[0].TermTimeout, defaultProcTimeout)
		t.Fail()
	}
	if cf.Processes.MainProcesses[0].TermTimeout != defaultProcTimeout {
		t.Logf("Default timeout on main process was not set as expected. Got: %d, Want: %d.", cf.Processes.MainProcesses[0].TermTimeout, defaultProcTimeout)
		t.Fail()
	}
}

func TestConfigString(t *testing.T) {
	cf := Config{
		Processes: Processes{
			InitProcesses: []*Process{
				&Process{
					Name: "TestInit_1",
					CMD:  "/bin/false",
					Args: []string{"arg1", "arg2"},
					LoggerConfig: LoggingConfig{
						Engine:      "syslog",
						ProcessName: "test_proc_name",
					},
				},
			},
		},
		DefaultLoggerConfig: DefaultLoggerDetails{
			Config: LoggingConfig{
				Engine: "console",
			},
		},
		ProcessManager: ProcessManager{
			LoggerConfig: LoggingConfig{
				Engine: "console",
			},
		},
	}

	if fmt.Sprintf("%s", cf) == "" {
		t.Logf("Failed to change config to a string")
		t.Fail()
	}
	t.Logf("config as string:\n%s", cf)
}
