package configfile

var (
	defaultLoggingEngine = LoggingConfig{
		Engine: "console",
	}

	defaultProcessManager = ProcessManager{
		LoggerConfig: LoggingConfig{
			Engine: "console",
		},
	}

	defaultProcessManagerSyslog = Syslog{
		ProgramName: "Launch",
	}

	defaultProcTimeout = 30
)
