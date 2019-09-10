package configfile

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

// ExampleConfigFile will return a string with an example yaml file.
// All features should be in here to present to the user.
func ExampleConfigFile() (string, error) {
	exampleInitProcesses := []*Process{
		{
			Name:          "Process1",
			CMD:           "/example/bin1",
			Args:          []string{"--arg1", "two"},
			CombindOutput: false,
			LoggerConfig: LoggingConfig{
				Engine: "console",
			},
		},
		{
			Name: "Process2",
			Args: []string{"--print", "extra"},
		},
	}
	exampleMainProcesses := []*Process{
		{
			Name:          "Process1",
			CMD:           "/example/bin1",
			Args:          []string{"--arg1", "--arg2", "--arg3", "extra"},
			CombindOutput: false,
		}, {
			Name: "Process2",
			CMD:  "/example/bin2",
			Args: []string{"--print", "extra"},
		},
	}

	exampleLoggerConfig := DefaultLoggerDetails{
		Config: LoggingConfig{
			Engine: "syslog",
			Syslog: Syslog{
				ProgramName: "example_service",
				Address:     "logs.papertrail.com:16900",
			},
		},
	}

	exampleProcessManagerConfig := ProcessManager{
		LoggerConfig: LoggingConfig{
			Engine: "syslog",
		},
	}
	exampleConfig := &Config{
		ProcessManager: exampleProcessManagerConfig,
		Processes: Processes{
			InitProcesses: exampleInitProcesses,
			MainProcesses: exampleMainProcesses,
		},
		DefaultLoggerConfig: exampleLoggerConfig,
	}

	out, err := yaml.Marshal(exampleConfig)
	if err != nil {
		return "", fmt.Errorf("Creating example failed. Error: %s", err)
	}

	return string(out), nil
}
