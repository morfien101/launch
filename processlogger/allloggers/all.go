// Package allloggers is used to bring in all the packages that contain the loggers
// which we support. This allows the loggers to be written as a plugin and used
// when they are added to the list below.
// We need to do it this way because the compiler needs to know which packages to
// bring in.
package allloggers

import (
	// Adding console logger
	_ "github.com/morfien101/launch/processlogger/console"
	// Adding devnull logger
	_ "github.com/morfien101/launch/processlogger/devnull"
	// Adding ELK logger
	_ "github.com/morfien101/launch/processlogger/filelogger"
	// Adding Syslog logger
	_ "github.com/morfien101/launch/processlogger/syslog"
)
