package filelogger

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/processlogger"
)

func TestDeleteOldFiles(t *testing.T) {
	testLogContent := `
logline1
logline2
logline3
`
	fileslimit := 10
	filenames := make([]string, fileslimit)

	dir, err := ioutil.TempDir("./", "")
	if err != nil {
		t.Fatalf("Could not create temp dir for test files.")
	}

	defer os.RemoveAll(dir)

	for i := 0; i < fileslimit; i++ {
		fName := fmt.Sprintf("./%s/file%02d.log", dir, i+1)
		err := ioutil.WriteFile(fName, []byte(testLogContent), 0600)
		if err != nil {
			t.Fatalf("Could not create test file. %s", err)
		}
		filenames[i] = fName
	}
	var oneHundredMegs datasize.ByteSize
	err = oneHundredMegs.UnmarshalText([]byte("100 mb"))
	if err != nil {
		t.Fatal(err)
	}
	config := configfile.FileLogger{
		Filename:        "/tmp/file01.log",
		SizeLimit:       oneHundredMegs.Bytes(),
		HistoricalFiles: 2,
	}
	rw, err := newRW(config)
	if err != nil {
		t.Fatal(err)
	}
	rw.historicalFilePaths = filenames
	rw.deleteOldFiles()

	t.Log(rw.historicalFilePaths)

	errCheck := func(e error) {
		t.Log(e)
		t.Fail()
	}

	for i := 0; i < fileslimit; i++ {
		_, err := os.Stat(filenames[i])
		if i < 2 {
			if err != nil {
				errCheck(err)
			}
			continue
		}

		if err == nil {
			errCheck(err)
		}
	}
}

// TestRotateWriterClose guards against regression of the infinite recursion bug
// where Close() called itself instead of w.fp.Close().
func TestRotateWriterClose(t *testing.T) {
	dir, err := ioutil.TempDir("", "rw_close_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	config := configfile.FileLogger{
		Filename:        filepath.Join(dir, "test.log"),
		SizeLimit:       1024 * 1024,
		HistoricalFiles: 1,
	}
	rw, err := newRW(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil {
		t.Fatalf("Close() returned an error: %v", err)
	}
}

// TestFileLogManagerRegisterAndSubmit guards against regression of the nil map
// panic where FileLogManager was constructed without initialising filetracker.
func TestFileLogManagerRegisterAndSubmit(t *testing.T) {
	dir, err := ioutil.TempDir("", "flm_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logConf := configfile.LoggingConfig{
		Logfile: configfile.FileLogger{
			Filename:        filepath.Join(dir, "test.log"),
			SizeLimit:       1024 * 1024,
			HistoricalFiles: 1,
		},
	}

	flm := &FileLogManager{filetracker: make(map[string]*rotateWriter)}
	if err := flm.RegisterConfig(logConf, configfile.DefaultLoggerDetails{}); err != nil {
		t.Fatalf("RegisterConfig() returned an error: %v", err)
	}

	flm.Submit(processlogger.LogMessage{
		Config:  logConf,
		Message: "regression test message\n",
	})

	if err := <-flm.Shutdown(); err != nil {
		t.Fatalf("Shutdown() returned an error: %v", err)
	}
}
