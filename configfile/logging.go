package configfile

// LoggingConfig is a struct that will hold the values of the logging
// configuration of the process or process manager
type LoggingConfig struct {
	Engine      []string   `yaml:"engine,omitempty"`
	ProcessName string     `yaml:"process_name,omitempty"`
	Syslog      Syslog     `yaml:"syslog,omitempty"`
	Logfile     FileLogger `yaml:"file_logger,omitempty"`
}

// DefaultLoggerDetails will hold the default logger configuration
type DefaultLoggerDetails struct {
	Config LoggingConfig `yaml:"logging_config,omitempty"`
}

// Syslog is used to send configuration to the syslog logger
type Syslog struct {
	ProgramName                string `yaml:"program_name"`
	Address                    string `yaml:"address"`
	ConnectionType             string `yaml:"protocol,omitempty"`
	CertificateBundlePath      string `yaml:"cert_bundle_path,omitempty"`
	ExtractLogLevel            bool   `yaml:"extract_log_level,omitempty"`
	OverrideHostname           string `yaml:"override_hostname,omitempty"`
	AddContainerNameToTag      bool   `yaml:"append_container_name_to_tag,omitempty"`
	AddContainerNameToHostname bool   `yaml:"append_container_name_to_hostname,omitempty"`
}

// FileLogger is a logger that will write to files
type FileLogger struct {
	Filename        string `yaml:"filepath"`
	SizeLimit       uint64 `yaml:"size_limit"`
	HistoricalFiles int    `yaml:"historical_files_limit"`
}
