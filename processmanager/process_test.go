package processmanager

import (
	"os/exec"
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

func TestRestartAllowed(t *testing.T) {
	proc := &Process{config: &configfile.Process{}}

	tests := []struct {
		name     string
		count    int
		preload  int
		override bool
		want     bool
	}{
		{
			name:  "Zero count",
			count: 0,
			want:  false,
		},
		{
			name:  "Negative Count",
			count: -20,
			want:  false,
		},
		{
			name:  "Positive count",
			count: 2,
			want:  true,
		},
		{
			// The first run isn't counted towards the restarts.
			name:    "Two fails, two configured",
			count:   2,
			preload: 2,
			want:    true,
		},
		{
			name:    "Three fails, two configured",
			count:   2,
			preload: 3,
			want:    false,
		},
		{
			name:     "Positive count with override",
			count:    2,
			override: true,
			want:     false,
		},
	}

	for _, test := range tests {
		proc.config.RestartCount = test.count
		proc.restartCounter = test.preload
		proc.blockRestarts = test.override

		if got := proc.restartAllowed(); got != test.want {
			t.Logf("%s, Got: %v, Want: %v", test.name, got, test.want)
			t.Fail()
		}
	}
}

// TestProcessReset is a simple test to save from a accidental line delete.
// Resetting these values is important but difficult to notice in integration tests.
func TestProcessReset(t *testing.T) {
	proc := Process{
		exited:   true,
		exitcode: 2,
		proc:     exec.Command("cat", []string{"dog", "mouse"}...),
	}

	proc.reset()

	if proc.exited {
		t.Logf("Reset failed to set exited to false")
		t.Fail()
	}
	if proc.exitcode != 0 {
		t.Logf("Reset failed to set exitcode to 0")
		t.Fail()
	}
	if proc.proc != nil {
		t.Logf("Reset failed remove proc")
		t.Fail()
	}
}
