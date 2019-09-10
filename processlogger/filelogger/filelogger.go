// Package filelogger is use to push log messages to a file on disk.
// Given that the process manger is designed to work inside containers
// filelogger also manages the rotation of files.
// Configuration passed in dictates how many files to keep and how large
// they should be.
package filelogger

import (
	"fmt"
	"strings"

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/processlogger"
)

const (
	// LoggerTag is used to identify the logger
	LoggerTag = "logfile"
)

// FileLogManager is used to keep track of the current files that are used
// to write logs to.
type FileLogManager struct {
	filetracker map[string]*rotateWriter
}

var fileLogManager *FileLogManager

func init() {
	processlogger.RegisterLogger(LoggerTag, func() processlogger.Logger {
		return &FileLogManager{}
	})
}

// RegisterConfig will create a new file and router for each config passed in.
func (flm *FileLogManager) RegisterConfig(conf configfile.LoggingConfig, defaults configfile.DefaultLoggerDetails) error {
	if _, ok := flm.filetracker[conf.Logfile.Filename]; ok {
		return nil
	}
	wr, err := newRW(conf.Logfile)
	flm.filetracker[conf.Logfile.Filename] = wr
	if err != nil {
		return err
	}

	return nil
}

// Start will create all the internal components that are required to run the logger
func (flm *FileLogManager) Start() error {
	return nil
}

// Shutdown will close all the files and return a chan error to signal completion
// and forward any errors
func (flm *FileLogManager) Shutdown() chan error {
	errChan := make(chan error, 1)

	go func() {
		errors := make([]string, 0)
		addErr := func(err error) {
			errors = append(errors, err.Error())
		}
		for _, tracker := range flm.filetracker {
			err := tracker.Close()
			if err != nil {
				addErr(err)
			}
		}

		if len(errors) > 0 {
			errChan <- fmt.Errorf(strings.Join(errors, " | "))
		} else {
			errChan <- nil
		}
		close(errChan)
	}()

	return errChan
}

// Submit will write a log message to a file that is dictated by the configuration
// sent with the processlogger.LogMessage
func (flm *FileLogManager) Submit(msg processlogger.LogMessage) {
	flm.filetracker[msg.Config.Logfile.Filename].Write([]byte(msg.Message))
}
