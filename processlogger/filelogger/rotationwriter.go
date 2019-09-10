package filelogger

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/morfien101/launch/configfile"
)

// rotateWriter can write and rotate a log file
type rotateWriter struct {
	lock                sync.Mutex
	fp                  *os.File
	config              configfile.FileLogger
	watchDogSignals     chan bool
	historicalFilePaths []string
	currentFileSize     uint64
	running             bool
}

// New makes a new rotateWriter. Return nil if error occurs during setup.
func newRW(conf configfile.FileLogger) (*rotateWriter, error) {
	w := &rotateWriter{
		config:              conf,
		watchDogSignals:     make(chan bool, 100),
		historicalFilePaths: make([]string, 0),
		running:             true,
	}
	// rotate gives us the first file
	err := w.rotate()
	if err != nil {
		return nil, err
	}
	return w, nil
}

// watchDog watches the file for size changes and rotates when required.
// WatchDog will also trigger clean up tasks to remove old files.
// watchDog is run as a go routine.
func (w *rotateWriter) watchDog() {
	for {
		select {
		case _, ok := <-w.watchDogSignals:
			// If the channel is closed we can just exit out.
			if !ok {
				return
			}
			// Check to see if the file is too large
			ok = w.tooLarge()
			// Check to see if the rotation has created a new file.
			// This means we could need to delete the files.
			if !ok {
				err := w.rotate()
				if err != nil {
					w.panic(err)
				}
				w.deleteOldFiles()
			}
		}
	}
}

// Loggers should not terminate service.
// Should we need to panic we should handle the situation as best we can
// Trying to keep service running.
func (w *rotateWriter) panic(err error) {
	// we have an err that we need to recover from
	// The best we can do here is print it to the console
	fmt.Println(err)
}

// tooLarge will tell us if the number of bytes we have written is more than the
// file size we want to handle.
// This is infered to avoid millions of os.stat calls
func (w *rotateWriter) tooLarge() bool {
	if w.currentFileSize > w.config.SizeLimit {
		return true
	}
	return false
}

func (w *rotateWriter) deleteOldFiles() {
	keep := make([]string, w.config.HistoricalFiles)
	for index, filename := range w.historicalFilePaths {
		if index < w.config.HistoricalFiles {
			keep[index] = filename
			continue
		}
		err := os.Remove(filename)
		if err != nil {
			w.panic(err)
		}
	}
	w.historicalFilePaths = keep
}

// Write satisfies the io.Writer interface.
func (w *rotateWriter) Write(output []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	n, err := w.fp.Write(output)
	w.currentFileSize = w.currentFileSize + uint64(n)
	return n, err
}

// Close closes out the current file
func (w *rotateWriter) Close() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	close(w.watchDogSignals)
	w.running = false
	return w.Close()
}

// Rotate Perform the actual act of rotating and reopening file.
func (w *rotateWriter) rotate() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	// Close existing file if open
	if w.fp != nil {
		err := w.fp.Close()
		w.fp = nil
		if err != nil {
			return err
		}
	}
	// Rename dest file if it already exists
	_, err := os.Stat(w.config.Filename)
	if err == nil {
		newFileName := w.config.Filename + "." + time.Now().Format(time.RFC3339)
		err = os.Rename(w.config.Filename, newFileName)
		if err != nil {
			return err
		}
		w.updateHistoricalFileNames(newFileName)
	}

	// Create a file.
	w.fp, err = os.Create(w.config.Filename)
	w.currentFileSize = 0
	return err
}

func (w *rotateWriter) updateHistoricalFileNames(newFileName string) {
	w.historicalFilePaths = append([]string{newFileName}, w.historicalFilePaths...)
}
