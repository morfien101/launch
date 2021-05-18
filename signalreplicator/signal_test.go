package signalreplicator

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestReplicator(t *testing.T) {
	chan1 := make(chan os.Signal, 1)
	chan2 := make(chan os.Signal, 1)

	Register(chan1)
	Register(chan2)

	Send(syscall.SIGHUP)

	output1 := <-chan1
	output2 := <-chan2

	Remove(chan2)

	Send(syscall.SIGTERM)

	// Sleeping to allow any race conditions to flush.
	// Such as the signal just not making it to the removed channel
	// by the time we look for it below.
	time.Sleep(time.Millisecond * 10)
	select {
	case <-chan2:
		t.Logf("chan2 got a signal after being removed")
		t.Fail()
	default:
	}
	output3 := <-chan1

	t.Logf("Got signals: %v, %v, %v", output1, output2, output3)
	if output1 != syscall.SIGHUP {
		t.Logf("output 1 should have been a SIGHUP signal. Got: %s", output1)
		t.Fail()
	}
	if output3 != syscall.SIGTERM {
		t.Logf("output 1 should have been a SIGTERM signal. Got: %s", output3)
		t.Fail()
	}
}
