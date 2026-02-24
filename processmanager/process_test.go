package processmanager

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"syscall"
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

// makeTestProcess builds a Process wired for testing. closePipesChan is buffered
// so the send inside runProcess never blocks even when no pipe drainer is running.
func makeTestProcess(t *testing.T, cmd string, args ...string) *Process {
	t.Helper()
	return &Process{
		config: &configfile.Process{
			Name: "test",
			CMD:  cmd,
			Args: args,
		},
		pmlogger:       internallogger.NewFakeLogger(),
		sigChan:        make(chan os.Signal, 1),
		closePipesChan: make(chan bool, 1),
		proc:           exec.Command(cmd, args...),
	}
}

// TestRunning guards that running() returns !p.exited, not p.exited.
func TestRunning(t *testing.T) {
	p := &Process{config: &configfile.Process{}}
	if !p.running() {
		t.Error("running() should be true before process exits")
	}
	p.exited = true
	if p.running() {
		t.Error("running() should be false after exited is set")
	}
}

// TestReadExitError verifies the exit-code extraction logic for several cases:
// nil, non-exit errors, specific codes, and a signal-killed process.
func TestReadExitError(t *testing.T) {
	tests := []struct {
		name     string
		errFn    func() error
		wantCode int
	}{
		{
			name:     "nil error returns 0",
			errFn:    func() error { return nil },
			wantCode: 0,
		},
		{
			name:     "plain error returns 0",
			errFn:    func() error { return errors.New("some error") },
			wantCode: 0,
		},
		{
			name: "exit code 42",
			errFn: func() error {
				return exec.Command("sh", "-c", "exit 42").Run()
			},
			wantCode: 42,
		},
		{
			name: "exit code 1",
			errFn: func() error {
				return exec.Command("sh", "-c", "exit 1").Run()
			},
			wantCode: 1,
		},
		{
			name: "signal-killed process is clamped to 1",
			errFn: func() error {
				cmd := exec.Command("sleep", "30")
				if err := cmd.Start(); err != nil {
					t.Fatal("failed to start process:", err)
				}
				_ = cmd.Process.Kill()
				return cmd.Wait()
			},
			wantCode: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := readExitError(tc.errFn()); got != tc.wantCode {
				t.Errorf("readExitError() = %d, want %d", got, tc.wantCode)
			}
		})
	}
}

// TestRunProcessCleanExit verifies that runProcess returns exit code 0 and no
// error for a process that exits successfully, and marks the process as stopped.
func TestRunProcessCleanExit(t *testing.T) {
	p := makeTestProcess(t, "true")
	state := p.runProcess("test")

	if state.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", state.ExitCode)
	}
	if state.Error != nil {
		t.Errorf("expected no error, got %v", state.Error)
	}
	if p.running() {
		t.Error("running() should be false after runProcess returns")
	}
}

// TestRunProcessNonZeroExit verifies that runProcess captures a non-zero exit
// code and returns a non-nil error.
func TestRunProcessNonZeroExit(t *testing.T) {
	p := makeTestProcess(t, "sh", "-c", "exit 3")
	state := p.runProcess("test")

	if state.ExitCode != 3 {
		t.Errorf("expected exit code 3, got %d", state.ExitCode)
	}
	if state.Error == nil {
		t.Error("expected a non-nil error for non-zero exit")
	}
}

// TestRunProcessSigterm verifies that forwarding SIGTERM via sigChan causes the
// child process to exit and runProcess to return promptly.
func TestRunProcessSigterm(t *testing.T) {
	p := makeTestProcess(t, "sleep", "30")
	// Use a short TermTimeout so the internal timer goroutine cleans up quickly
	// after the process exits rather than leaking for the default 30 seconds.
	p.config.TermTimeout = 2

	go func() {
		time.Sleep(50 * time.Millisecond)
		p.sigChan <- syscall.SIGTERM
	}()

	start := time.Now()
	state := p.runProcess("test")
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		t.Errorf("runProcess took too long after SIGTERM: %v", elapsed)
	}
	if state.ExitCode == 0 {
		t.Errorf("expected non-zero exit code after SIGTERM, got 0")
	}
}

// TestRunProcessKillOnTimeout verifies that a process which ignores SIGTERM is
// forcibly killed once TermTimeout elapses.
func TestRunProcessKillOnTimeout(t *testing.T) {
	p := makeTestProcess(t, "sh", "-c", "trap '' TERM; sleep 30")
	p.config.TermTimeout = 1

	go func() {
		time.Sleep(50 * time.Millisecond)
		p.sigChan <- syscall.SIGTERM
	}()

	start := time.Now()
	state := p.runProcess("test")
	elapsed := time.Since(start)

	if elapsed < time.Second {
		t.Errorf("expected runProcess to take at least 1s for the kill timeout, took %v", elapsed)
	}
	if elapsed > 10*time.Second {
		t.Errorf("runProcess took too long waiting for forced kill: %v", elapsed)
	}
	if state.ExitCode == 0 {
		t.Errorf("expected non-zero exit code after being killed, got 0")
	}
}

// TestRunSecretProcessSuccess verifies that RunSecretProcess captures stdout
// from a process that exits cleanly.
func TestRunSecretProcessSuccess(t *testing.T) {
	conf := configfile.SecretProcess{
		Name:        "echo",
		CMD:         "echo",
		Args:        []string{"hello"},
		TermTimeout: 5,
	}
	stdout, stderr, err := RunSecretProcess(conf, internallogger.NewFakeLogger())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stdout != "hello\n" {
		t.Errorf("expected stdout %q, got %q", "hello\n", stdout)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
}

// TestRunSecretProcessNonZeroExit verifies that RunSecretProcess returns an
// error on non-zero exit and surfaces stderr as the second return value.
func TestRunSecretProcessNonZeroExit(t *testing.T) {
	conf := configfile.SecretProcess{
		Name:        "failing",
		CMD:         "sh",
		Args:        []string{"-c", "echo 'oops' >&2; exit 1"},
		TermTimeout: 5,
	}
	stdout, stderr, err := RunSecretProcess(conf, internallogger.NewFakeLogger())

	if err == nil {
		t.Fatal("expected a non-nil error for non-zero exit")
	}
	if stdout != "" {
		t.Errorf("expected empty stdout on failure, got %q", stdout)
	}
	if stderr != "oops\n" {
		t.Errorf("expected stderr %q, got %q", "oops\n", stderr)
	}
}

// TestRunSecretProcessTimeout verifies that RunSecretProcess returns an error
// when the child process exceeds TermTimeout.
func TestRunSecretProcessTimeout(t *testing.T) {
	conf := configfile.SecretProcess{
		Name:        "slow",
		CMD:         "sleep",
		Args:        []string{"30"},
		TermTimeout: 1,
	}

	start := time.Now()
	_, _, err := RunSecretProcess(conf, internallogger.NewFakeLogger())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a non-nil error due to context timeout")
	}
	if elapsed > 5*time.Second {
		t.Errorf("timeout test ran too long: %v", elapsed)
	}
}

// TestExitStatusFormatter verifies that exitStatusFormatter produces valid JSON
// containing the expected name and exit_code fields.
func TestExitStatusFormatter(t *testing.T) {
	pm := &ProcessManger{
		EndList: []*processEnd{
			{Name: "init-step", ProcessType: "init", ExitCode: 0},
			{Name: "main-app", ProcessType: "main", ExitCode: 1},
		},
		pmlogger: internallogger.NewFakeLogger(),
	}

	result := pm.exitStatusFormatter()

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("exitStatusFormatter returned invalid JSON: %v\ngot: %s", err, result)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 entries in JSON output, got %d", len(parsed))
	}
	if parsed[0]["name"] != "init-step" {
		t.Errorf("expected name 'init-step', got %v", parsed[0]["name"])
	}
	if parsed[1]["exit_code"] != float64(1) {
		t.Errorf("expected exit_code 1 for second entry, got %v", parsed[1]["exit_code"])
	}
}
