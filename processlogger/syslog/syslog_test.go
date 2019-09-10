package syslog

import (
	"testing"

	syslogger "github.com/silverstagtech/srslog"
)

func TestSyslogConnection(t *testing.T) {
	writer, err := syslogger.Dial(
		"udp",
		"127.0.0.2:514",
		syslogger.LOG_INFO|syslogger.LOG_KERN,
		"launch_test",
	)
	if err != nil {
		t.Logf("Failed to connect. Error: %s", err)
		t.Fail()
	}

	err = writer.Info("Hello from Go")
	if err != nil {
		t.Logf("Failed to write message")
	}
}
