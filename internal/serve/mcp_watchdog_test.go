package serve

import (
	"bytes"
	"encoding/json"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// TestParentWatchdog_NotStartedUnderSetIO verifies the s.input == os.Stdin
// gate keeps the watchdog off when Run() is driven by a bytes.Buffer (the
// pattern every existing mcp test uses via sendRecv/sendMulti). If the gate
// regressed, the watchdog goroutine would start under the test runner and
// could eventually os.Exit it. This test proves Run() returns promptly with
// a non-stdin input.
func TestParentWatchdog_NotStartedUnderSetIO(t *testing.T) {
	srv := NewMCPServer(t.TempDir(), t.TempDir(), "1.0.0-test")

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "ping",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	in := bytes.NewBufferString(string(data) + "\n")
	var out bytes.Buffer
	srv.SetIO(in, &out)

	done := make(chan error, 1)
	go func() { done <- srv.Run() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return promptly with a bytes.Buffer input — the watchdog gate may have regressed")
	}
}

// TestStartParentWatchdog_StopsOnDone verifies the watchdog goroutine exits
// cleanly when the done channel is closed (the defer close(done) path in
// Run). It does NOT exercise the os.Exit branch — ppid is stable within the
// test binary, which is exactly the point: the goroutine is well-behaved
// while the parent is alive, and tears down on done.
func TestStartParentWatchdog_StopsOnDone(t *testing.T) {
	before := runtime.NumGoroutine()

	done := make(chan struct{})
	startParentWatchdog(done)

	close(done)

	// After done is closed, the watchdog goroutine should return and the
	// count should settle back to (at most) the baseline. Goroutine counts
	// are inherently noisy, so we poll until it drops rather than asserting
	// an exact value at a fixed instant.
	if got := waitForGoroutinesAtMost(before, 2*time.Second); got > before {
		t.Fatalf("watchdog goroutine did not exit after done closed: before=%d now=%d", before, got)
	}
}

// TestParentWatchdog_ExitsOnPpidChange exercises the reparent-decision
// branch: when watchdogGetppid() reports a value different from the one
// captured at startup, the watchdog must call watchdogExit(0). We stub the
// seam vars to simulate a reparent without faking a real OS reparent, and
// stub watchdogExit to record the code rather than actually exiting. It must
// NOT run in parallel — it mutates package-level seam vars and the interval.
func TestParentWatchdog_ExitsOnPpidChange(t *testing.T) {
	origExit := watchdogExit
	origGetppid := watchdogGetppid
	origInterval := parentWatchdogInterval
	defer func() {
		watchdogExit = origExit
		watchdogGetppid = origGetppid
		parentWatchdogInterval = origInterval
	}()

	// First call returns a stable ppid (captured as startPpid); every
	// subsequent call returns a different value, simulating reparenting.
	var calls int64
	const startPpid = 1000
	const reparentedPpid = 1
	watchdogGetppid = func() int {
		if atomic.AddInt64(&calls, 1) == 1 {
			return startPpid
		}
		return reparentedPpid
	}

	// Record the exit code and signal, then block the goroutine so it can't
	// loop again after the "exit" (the real os.Exit would not return).
	exited := make(chan int, 1)
	watchdogExit = func(code int) {
		exited <- code
		select {} // mimic os.Exit never returning
	}

	parentWatchdogInterval = time.Millisecond

	done := make(chan struct{})
	defer close(done)
	startParentWatchdog(done)

	select {
	case code := <-exited:
		if code != 0 {
			t.Fatalf("watchdog exited with code %d, want 0", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog did not call exit on ppid change")
	}
}

// waitForGoroutinesAtMost polls runtime.NumGoroutine() until it is at most
// target or the timeout elapses, returning the last observed count.
func waitForGoroutinesAtMost(target int, timeout time.Duration) int {
	deadline := time.Now().Add(timeout)
	last := runtime.NumGoroutine()
	for time.Now().Before(deadline) {
		last = runtime.NumGoroutine()
		if last <= target {
			return last
		}
		time.Sleep(10 * time.Millisecond)
	}
	return last
}
