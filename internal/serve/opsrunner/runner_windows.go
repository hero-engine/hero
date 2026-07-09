//go:build windows

package opsrunner

import "os/exec"

// setupProcessGroup is a no-op on Windows, which has no POSIX process groups.
// exec.CommandContext's default cancellation (kill the direct child) applies;
// the opsrunner fake-binary tests are skipped on Windows accordingly.
func setupProcessGroup(cmd *exec.Cmd) {}
