//go:build unix

package opsrunner

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup places the child in its own process group and rewires
// context cancellation to signal the entire group, so grandchildren are
// killed with the parent instead of leaking.
//
// exec.CommandContext's default cancellation kills only the direct child.
// A hero subprocess is launched through a `#!/bin/sh` wrapper, so the real
// work runs in grandchildren that would otherwise survive an orphaned parent
// — keeping the inherited stdout/stderr pipes open and blocking the runner's
// waiter goroutine on ioWG.Wait() until they exit on their own. On a busy CI
// runner that stall exceeds the test timeout and fails the release build.
func setupProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	// Negative pid targets the process group led by the child (pgid == pid
	// once Setpgid takes effect), so the shell and every descendant die.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			return err
		}
		return nil
	}
}
