package serve

import (
	"os"
	"time"
)

// parentWatchdogInterval is how often the portable poll checks ppid.
// 30s is a deliberate tradeoff: an orphan lives at most ~30s past its
// parent's death, which is invisible operationally, while the poll cost
// is one getppid() syscall per tick — negligible.
//
// It is a var (not const) so tests can shrink it to exercise the poll
// loop quickly; production code never reassigns it.
var parentWatchdogInterval = 30 * time.Second

// Seam vars default to the real runtime functions; tests override them to
// exercise the reparent-decision branch in-process without faking an OS
// reparent. Production code never reassigns these.
var (
	watchdogExit    = os.Exit
	watchdogGetppid = os.Getppid
)

// startParentWatchdog launches the parent-liveness backstop. It first
// arms the OS-native fast path (Linux: PR_SET_PDEATHSIG; darwin: no-op),
// then runs a portable ppid poll that covers platforms without a native
// signal (notably macOS). The poll exits the process only when we've been
// reparented to launchd/init (ppid==1) OR the original parent is confirmed
// dead. A bare ppid change (an intermediate wrapper exited while the
// session is still live) is a false positive and is ignored.
//
// It fires ONLY when the parent is already dead, so it can never disrupt
// a live session.
// The returned channel is closed when the watchdog goroutine has fully
// exited (after `done` is closed). It exists as a happens-before join for
// tests that overwrite the package-level seam vars in cleanup — restoring
// them while the goroutine is still mid-tick reading them is a data race
// (-race catches it in CI). Production callers discard it: the goroutine
// simply dies with the process.
func startParentWatchdog(done <-chan struct{}) <-chan struct{} {
	armParentDeathSignal() // platform-specific; no-op on darwin

	startPpid := watchdogGetppid()
	interval := parentWatchdogInterval
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				now := watchdogGetppid()
				if now != startPpid && (now == 1 || !singletonIsAlive(startPpid)) {
					// Exit only when reparented to launchd/init (ppid==1)
					// or the original parent is confirmed dead. A mere
					// ppid change — an intermediate wrapper process exited
					// while the session is still live — is a false positive
					// that must NOT tear down a live session.
					watchdogExit(0)
				}
			}
		}
	}()
	return stopped
}
