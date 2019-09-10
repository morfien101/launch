package processlogger

// RegisterFunc is a function that can register the loggers
type RegisterFunc func() Logger

// registeredLoggers contains a map of all available loggers and a function
// that will initialize them.
var registeredLoggers map[string]RegisterFunc

// RegisterLogger will register the loggers when the packages are
// read at init time.
func RegisterLogger(name string, regfunc RegisterFunc) {
	if registeredLoggers == nil {
		registeredLoggers = make(map[string]RegisterFunc)
	}
	registeredLoggers[name] = regfunc
}
