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
// signal (notably macOS). The poll exits the process when our parent
// changes from its startup value — i.e. we've been reparented to
// launchd/init because the original parent died.
//
// It fires ONLY when the parent is already dead, so it can never disrupt
// a live session.
func startParentWatchdog(done <-chan struct{}) {
	armParentDeathSignal() // platform-specific; no-op on darwin

	startPpid := watchdogGetppid()
	interval := parentWatchdogInterval
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if watchdogGetppid() != startPpid {
					// Parent reparented away (died). Exit cleanly.
					watchdogExit(0)
				}
			}
		}
	}()
}
