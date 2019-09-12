package configfile

var (
	defaultLoggingEngine = LoggingConfig{
		Engine: []string{"console"},
	}

	defaultProcessManager = ProcessManager{
		LoggerConfig: LoggingConfig{
			Engine: []string{"console"},
		},
	}

	defaultProcessManagerSyslog = Syslog{
		ProgramName: "Launch",
	}

	defaultProcTimeout = 30
)
