package configfile

var (
	defaultLoggingEngine = LoggingConfig{
		Engine: "console",
	}

	defaultProcessManager = ProcessManager{
		LoggerConfig: defaultLoggingEngine,
	}

	defaultProcTimeout = 30
)
