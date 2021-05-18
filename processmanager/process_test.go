package processmanager

import (
	"testing"
	"time"

	"github.com/morfien101/launch/configfile"
	"github.com/morfien101/launch/internallogger"
)

func TestProcessStartDelay(t *testing.T) {
	duration := 1
	proc := &Process{
		config: &configfile.Process{
			StartDelay: duration,
		},
		pmlogger: internallogger.NewFakeLogger(),
	}
	proc.pmlogger.DebugOn(true)

	now := time.Now()
	proc.processStartDelay()
	since := time.Since(now)

	if since < time.Second*time.Duration(duration) {
		t.Log("Process didn't wait to start")
		t.Fail()
	}
}

func TestProcessNoStartDelay(t *testing.T) {
	proc := &Process{
		config:   &configfile.Process{},
		pmlogger: internallogger.NewFakeLogger(),
	}
	proc.pmlogger.DebugOn(true)

	now := time.Now()
	proc.processStartDelay()
	since := time.Since(now)

	if since > time.Millisecond*25 {
		t.Log("Process start delay took too long to complete")
		t.Fail()
	}
}
