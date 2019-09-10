package configfile

// ProcessManager hold configuration for the Process Manger itself
type ProcessManager struct {
	LoggerConfig LoggingConfig  `yaml:"logging_config"`
	DebugLogging bool           `yaml:"debug_logging,omitempty"`
	DebugOptions PMDebugOptions `yaml:"debug_options,omitempty"`
}

// PMDebugOptions holds configuration for debugging
type PMDebugOptions struct {
	PrintGeneratedConfig bool `yaml:"show_generated_config"`
}
