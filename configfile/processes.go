package configfile

// Processes holds all the processes that need to be executed.
type Processes struct {
	InitProcesses []*Process `yaml:"init_processes,omitempty"`
	MainProcesses []*Process `yaml:"main_processes"`
}

// Process is a struct that consumes a yaml configration and holds config for a
// process that needs to be run.
type Process struct {
	Name string   `yaml:"name"`
	CMD  string   `yaml:"command"`
	Args []string `yaml:"arguments"`
	// loggingEngine can be either Papertrail or ELK <though ELK is not implimented currently>
	LoggerConfig  LoggingConfig `yaml:"logging_config"`
	CombindOutput bool          `yaml:"combine_output,omitempty"`
	TermTimeout   int           `yaml:"termination_timeout_seconds,omitempty"`
}
