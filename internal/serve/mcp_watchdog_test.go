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

// TestParentWatchdog_IgnoresReparentWhileParentAlive exercises the
// reparent-hardening: a bare ppid change to a non-init value while the
// ORIGINAL parent is still alive is a false positive (an intermediate
// wrapper exited, the session is still live) and must NOT trigger exit.
// This is the secondary mid-session death vector the fix closes. It must
// NOT run in parallel — it mutates package-level seam vars.
func TestParentWatchdog_IgnoresReparentWhileParentAlive(t *testing.T) {
	origExit := watchdogExit
	origGetppid := watchdogGetppid
	origAlive := singletonIsAlive
	origInterval := parentWatchdogInterval
	defer func() {
		watchdogExit = origExit
		watchdogGetppid = origGetppid
		singletonIsAlive = origAlive
		parentWatchdogInterval = origInterval
	}()

	const startPpid = 1000
	const reparentedPpid = 999 // non-init: an intermediate wrapper exited
	var calls int64
	watchdogGetppid = func() int {
		if atomic.AddInt64(&calls, 1) == 1 {
			return startPpid
		}
		return reparentedPpid
	}
	// The original parent is still alive — the session is live.
	singletonIsAlive = func(pid int) bool { return pid == startPpid }

	exited := make(chan int, 1)
	watchdogExit = func(code int) {
		exited <- code
		select {}
	}
	parentWatchdogInterval = time.Millisecond

	done := make(chan struct{})
	stopped := startParentWatchdog(done)
	// Join the goroutine before the deferred cleanup restores the seam
	// vars. `close(done)` only signals the goroutine to stop; it may still
	// be mid-tick reading watchdogGetppid/singletonIsAlive when the restore
	// writes them — a data race -race flags in CI. Registered AFTER the
	// seam-restore defer, so LIFO runs it first: close → join → then restore.
	defer func() { close(done); <-stopped }()

	// Give the poll many ticks; it must never fire while the parent lives.
	select {
	case code := <-exited:
		t.Fatalf("watchdog exited (code %d) on a bare reparent while the original parent is alive — false positive", code)
	case <-time.After(200 * time.Millisecond):
		// Expected: no exit.
	}
}

// TestParentWatchdog_ExitsWhenOriginalParentDead verifies the other half of
// the hardened condition: a ppid change to a non-init value is a real orphan
// when the ORIGINAL parent is confirmed dead, so the watchdog must exit. It
// must NOT run in parallel — it mutates package-level seam vars.
func TestParentWatchdog_ExitsWhenOriginalParentDead(t *testing.T) {
	origExit := watchdogExit
	origGetppid := watchdogGetppid
	origAlive := singletonIsAlive
	origInterval := parentWatchdogInterval
	defer func() {
		watchdogExit = origExit
		watchdogGetppid = origGetppid
		singletonIsAlive = origAlive
		parentWatchdogInterval = origInterval
	}()

	const startPpid = 1000
	const reparentedPpid = 999 // non-init, but original parent is gone
	var calls int64
	watchdogGetppid = func() int {
		if atomic.AddInt64(&calls, 1) == 1 {
			return startPpid
		}
		return reparentedPpid
	}
	// The original parent is dead → genuine orphan, must reap.
	singletonIsAlive = func(pid int) bool { return false }

	exited := make(chan int, 1)
	watchdogExit = func(code int) {
		exited <- code
		select {}
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
		t.Fatal("watchdog did not exit when the original parent is confirmed dead")
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
