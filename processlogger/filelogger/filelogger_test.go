package filelogger

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/morfien101/launch/configfile"
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
