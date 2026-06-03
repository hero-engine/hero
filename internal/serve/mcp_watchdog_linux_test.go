//go:build linux

package serve

import "testing"

// TestArmParentDeathSignal verifies the Linux native fast path arms without
// panicking on its target platform. armParentDeathSignal issues a PR_SET_PDEATHSIG
// prctl and swallows any error (the portable poll is the backstop), so the only
// observable contract here is that it returns cleanly.
func TestArmParentDeathSignal(t *testing.T) {
	armParentDeathSignal()
}
