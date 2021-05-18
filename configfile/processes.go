package configfile

// Processes holds all the processes that need to be executed.
type Processes struct {
	SecretProcess []*SecretProcess `yaml:"secret_processes,omitempty"`
	InitProcesses []*Process       `yaml:"init_processes,omitempty"`
	MainProcesses []*Process       `yaml:"main_processes"`
}

// Process is a struct that consumes a yaml configration and holds config for a
// process that needs to be run.
type Process struct {
	Name          string        `yaml:"name"`
	CMD           string        `yaml:"command"`
	Args          []string      `yaml:"arguments"`
	LoggerConfig  LoggingConfig `yaml:"logging_config"`
	CombindOutput bool          `yaml:"combine_output,omitempty"`
	TermTimeout   int           `yaml:"termination_timeout_seconds,omitempty"`
	StartDelay    int           `yaml:"start_delay_seconds,omitempty"`
}

// SecretProcess is a struct that consumes a yaml configration and holds config for a
// secret collection process that needs to be run.
type SecretProcess struct {
	Name        string   `yaml:"name"`
	CMD         string   `yaml:"command"`
	Args        []string `yaml:"arguments"`
	TermTimeout int      `yaml:"termination_timeout_seconds,omitempty"`
	Skip        bool     `yaml:"skip"`
}
