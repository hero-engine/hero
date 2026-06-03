//go:build linux

package serve

import (
	"golang.org/x/sys/unix"
	"syscall"
)

// armParentDeathSignal requests SIGTERM when our parent dies. This is
// instant and poll-free on Linux. The portable ticker still runs as a
// backstop (and to cover the race where the parent dies between fork and
// this call). Errors are non-fatal — the poll covers us either way.
func armParentDeathSignal() {
	_ = unix.Prctl(unix.PR_SET_PDEATHSIG, uintptr(syscall.SIGTERM), 0, 0, 0)
}
