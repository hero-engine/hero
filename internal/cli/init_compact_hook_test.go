package cli

import (
	"os/exec"
	"testing"

	"github.com/hero-engine/hero/internal/hooks"
)

// TestInit_DefaultInstallsCompactHook — `hero init` with default flags
// in a git repo should install the Claude SessionStart{compact} entry.
// Catches the regression of init silently dropping the auto-install.
func TestInit_DefaultInstallsCompactHook(t *testing.T) {
	env := newTestEnvEmpty(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	ok, err := hooks.ClaudeCompactHandoffStatus(env.dir)
	if err != nil {
		t.Fatalf("ClaudeCompactHandoffStatus: %v", err)
	}
	if !ok {
		t.Error("expected claude SessionStart{compact} hook installed after `hero init`")
	}
}

// TestInit_NoHooksFlagSkipsCompactHook — `hero init --no-hooks` skips
// both the git pre-commit hook and the host-tool compact hook.
func TestInit_NoHooksFlagSkipsCompactHook(t *testing.T) {
	env := newTestEnvEmpty(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	if _, err := runCmd("init", "--no-hooks"); err != nil {
		t.Fatalf("init --no-hooks: %v", err)
	}

	ok, err := hooks.ClaudeCompactHandoffStatus(env.dir)
	if err != nil {
		t.Fatalf("ClaudeCompactHandoffStatus: %v", err)
	}
	if ok {
		t.Error("--no-hooks should suppress the claude SessionStart{compact} install")
	}
}

// TestInit_IsIdempotentForCompactHook — re-running init on an existing
// workspace fails (already exists), but the compact hook entry must
// not be duplicated when init is interleaved with manual installs.
// We verify this via the install path itself: first init, then a
// second InstallClaudeCompactHandoff should report "already installed"
// (false, nil) without adding a second entry.
func TestInit_IsIdempotentForCompactHook(t *testing.T) {
	env := newTestEnvEmpty(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Second-pass install must be a no-op.
	installed, err := hooks.InstallClaudeCompactHandoff(env.dir)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if installed {
		t.Error("expected second InstallClaudeCompactHandoff to report installed=false (idempotent)")
	}
}
