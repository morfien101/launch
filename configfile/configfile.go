package configfile

import (
	"fmt"
	"io/ioutil"

	"github.com/morfien101/launch/configfile/templating"
	"gopkg.in/yaml.v2"
)

// Config is a struct that represents the YAML file that we want to pass in.
type Config struct {
	ProcessManager      ProcessManager       `yaml:"process_manager"`
	Processes           Processes            `yaml:"processes"`
	DefaultLoggerConfig DefaultLoggerDetails `yaml:"default_logger_config"`
}

// New will return a new config file if one can be read from the location
// specified. An error is also returned if something goes wrong.
func New(filePath string) (*Config, error) {
	// Digest the config file
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Could not read config file. Error: %s", err)
	}

	decodedYaml, err := templating.GenerateTemplate(fileBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode template. Error: %s", err)
	}
	newConfig := &Config{}
	if err := yaml.Unmarshal(decodedYaml, newConfig); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal yaml. Error: %s", err)
	}

	newConfig.setDefaultLoggerConfig()
	newConfig.setDefaultProcessLogger()
	newConfig.setDefaultProcessManager()
	newConfig.setDefaultProcessTimeout()

	return newConfig, nil
}

// setDefaultLoggerConfig will setup a default logging config if one doesn't already exist.
func (cf *Config) setDefaultLoggerConfig() {
	if &cf.DefaultLoggerConfig == nil {
		cf.DefaultLoggerConfig = DefaultLoggerDetails{
			Config: defaultLoggingEngine,
		}
	}
	if cf.DefaultLoggerConfig.Config.Engine == "" {
		cf.DefaultLoggerConfig.Config.Engine = defaultLoggingEngine.Engine
	}
}

// setDefaultProcessLogger will go through the processes and set the default logging if there is
// nothing set. The following rules will apply
// The logging engine will be the default engine
// The process logging name should the be name given to the process
//
// NOTE: setDefaultLoggerConfig should be called first
//
func (cf *Config) setDefaultProcessLogger() {
	createLoggingConfig := func(proc *Process) {
		proc.LoggerConfig = LoggingConfig{}
	}
	setName := func(proc *Process) {
		proc.LoggerConfig.ProcessName = proc.Name
	}
	setEngine := func(proc *Process) {
		proc.LoggerConfig.Engine = cf.DefaultLoggerConfig.Config.Engine
	}
	f := func(procList []*Process) {
		for _, proc := range procList {
			if &proc.LoggerConfig == nil {
				// Create a logging config
				createLoggingConfig(proc)
			}
			if proc.LoggerConfig.ProcessName == "" {
				setName(proc)
			}
			if proc.LoggerConfig.Engine == "" {
				setEngine(proc)
			}
		}
	}

	f(cf.Processes.InitProcesses)
	f(cf.Processes.MainProcesses)
}

func (cf *Config) setDefaultProcessManager() {
	if &cf.ProcessManager == nil {
		cf.ProcessManager = defaultProcessManager
	}
	if &cf.ProcessManager.LoggerConfig == nil {
		cf.ProcessManager.LoggerConfig = defaultProcessManager.LoggerConfig
	}
	if cf.ProcessManager.LoggerConfig.Engine == "" {
		cf.ProcessManager.LoggerConfig.Engine = defaultProcessManager.LoggerConfig.Engine
	}
}

func (cf *Config) setDefaultProcessTimeout() {
	f := func(procs []*Process) {
		for _, proc := range procs {
			if proc.TermTimeout <= 0 {
				proc.TermTimeout = defaultProcTimeout
			}
		}
	}

	f(cf.Processes.InitProcesses)
	f(cf.Processes.MainProcesses)
}

func (cf Config) String() string {
	output, _ := yaml.Marshal(cf)
	return string(output)
}
