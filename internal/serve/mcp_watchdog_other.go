//go:build !linux

package serve

// armParentDeathSignal is a no-op on non-Linux platforms (notably
// darwin), which have no equivalent of PR_SET_PDEATHSIG. The portable
// ppid poll in startParentWatchdog is the sole mechanism there.
func armParentDeathSignal() {}
