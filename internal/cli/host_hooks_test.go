package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/hooks"
)

// resetHostHookFlag clears the cobra-persisted --host flag value between
// runCmd calls. Cobra holds the parsed value on the package-level var so
// a previous test's --host=all bleeds into later test cases unless we
// reset it here.
func resetHostHookFlag() {
	hostHookHostFlag = hostFlagNone
}

// TestHostHooksInstall_AllInstallsGitAndClaude — `hero hooks install
// --host=all` should drop the git pre-commit hook AND the Claude
// SessionStart{compact} entry. Both side-effects must land in one call.
func TestHostHooksInstall_AllInstallsGitAndClaude(t *testing.T) {
	env := newTestEnvEmpty(t)
	t.Setenv("HOME", t.TempDir())
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	// hooks install requires a hero workspace too.
	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer resetHostHookFlag()
	// `hero init` already installed the Claude hook; uninstall to set up
	// a clean state for this test.
	if _, err := hooks.UninstallClaudeCompactHandoff(env.dir); err != nil {
		t.Fatalf("uninstall before scenario: %v", err)
	}
	// Same for Codex — clean slate.
	if _, err := hooks.UninstallCodexCompactHandoff(env.dir); err != nil {
		t.Fatalf("uninstall codex before scenario: %v", err)
	}
	// Also remove the git pre-commit hook installed by init so we can
	// assert it gets re-installed below.
	_ = os.Remove(filepath.Join(env.dir, ".git", "hooks", "pre-commit"))

	out, err := runCmd("hooks", "install", "--host=all")
	if err != nil {
		t.Fatalf("hooks install --host=all: %v", err)
	}

	// Git pre-commit hook installed.
	if _, err := os.Stat(filepath.Join(env.dir, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("expected pre-commit hook to exist after --host=all install: %v", err)
	}
	// Claude SessionStart{compact} installed.
	ok, err := hooks.ClaudeCompactHandoffStatus(env.dir)
	if err != nil {
		t.Fatalf("claude status: %v", err)
	}
	if !ok {
		t.Error("expected ClaudeCompactHandoffStatus=true after --host=all install")
	}
	// Codex SessionStart{compact} installed.
	codexOK, err := hooks.CodexCompactHandoffStatus(env.dir)
	if err != nil {
		t.Fatalf("codex status: %v", err)
	}
	if !codexOK {
		t.Error("expected CodexCompactHandoffStatus=true after --host=all install")
	}
	if !strings.Contains(out, "claude SessionStart{compact}") {
		t.Errorf("output should mention claude SessionStart{compact}; got: %q", out)
	}
	if !strings.Contains(out, "codex SessionStart{compact}") {
		t.Errorf("output should mention codex SessionStart{compact}; got: %q", out)
	}
}

// TestHostHooksInstall_CodexInstallsToProjectFile — `hero hooks install
// --host=codex` writes the SessionStart{compact} entry into
// <projectRoot>/.codex/hooks.json and emits the install confirmation
// plus the trust-prompt note. (Was previously a stub-check test —
// retained-and-repurposed for the real installer.)
func TestHostHooksInstall_CodexInstallsToProjectFile(t *testing.T) {
	env := newTestEnvEmpty(t)
	// Isolate HOME so the feature-flag warning behavior is deterministic.
	t.Setenv("HOME", t.TempDir())
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer resetHostHookFlag()

	out, err := runCmd("hooks", "install", "--host=codex")
	if err != nil {
		t.Fatalf("hooks install --host=codex returned error: %v", err)
	}
	if !strings.Contains(out, "codex SessionStart{compact}") {
		t.Errorf("expected codex install confirmation; got: %q", out)
	}
	// File was actually written.
	if _, err := os.Stat(filepath.Join(env.dir, ".codex", "hooks.json")); err != nil {
		t.Errorf("expected .codex/hooks.json to exist after install; stat err=%v", err)
	}
	ok, err := hooks.CodexCompactHandoffStatus(env.dir)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !ok {
		t.Error("expected CodexCompactHandoffStatus=true after install")
	}
	// Trust-prompt note surfaced (informational).
	if !strings.Contains(strings.ToLower(out), "trust") {
		t.Errorf("expected trust-prompt note; got: %q", out)
	}
	// HOME is empty → feature flag absent → warning surfaced.
	if !strings.Contains(strings.ToLower(out), "codex_hooks") {
		t.Errorf("expected codex_hooks feature-flag warning; got: %q", out)
	}
}

// TestHostHooksStatus_ReportsPerHostState — `hero hooks status
// --host=all` should report Claude state (yes/no) and the Codex stub
// state on separate lines.
func TestHostHooksStatus_ReportsPerHostState(t *testing.T) {
	env := newTestEnvEmpty(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer resetHostHookFlag()

	// init already installed the Claude entry; assert status says yes.
	out, err := runCmd("hooks", "status", "--host=all")
	if err != nil {
		t.Fatalf("status --host=all: %v", err)
	}
	if !strings.Contains(out, "claude SessionStart{compact}: yes") {
		t.Errorf("expected claude SessionStart{compact}: yes; got: %q", out)
	}
	if !strings.Contains(out, "codex") {
		t.Errorf("expected a codex line in --host=all status; got: %q", out)
	}
}

// TestHostHooksUninstall_AllRemovesGitAndClaude — `hooks uninstall
// --host=all` should remove the git-hook side (Hero-managed
// `# Hero git hook` blocks installed by `hero hooks install`), the
// `# >>> hero next hooks (managed) >>>` block installed by `hero init` /
// `hero next install-hooks`, AND the Claude SessionStart{compact} entry.
// All three installer paths must be cleared by one uninstall call.
func TestHostHooksUninstall_AllRemovesGitAndClaude(t *testing.T) {
	env := newTestEnvEmpty(t)
	t.Setenv("HOME", t.TempDir())
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runCmd("init", "--no-hooks"); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer resetHostHookFlag()

	// Install both via --host=all.
	if _, err := runCmd("hooks", "install", "--host=all"); err != nil {
		t.Fatalf("install --host=all: %v", err)
	}
	// Also install the hero-next projection-hook block so we can verify
	// the uninstall picks it up alongside the general hooks + claude.
	if err := installNextHooksQuiet(env.dir); err != nil {
		t.Fatalf("installNextHooksQuiet: %v", err)
	}
	// Sanity: claude installed.
	ok, _ := hooks.ClaudeCompactHandoffStatus(env.dir)
	if !ok {
		t.Fatal("precondition: claude hook should be installed after --host=all install")
	}
	// Sanity: codex installed.
	codexOK, _ := hooks.CodexCompactHandoffStatus(env.dir)
	if !codexOK {
		t.Fatal("precondition: codex hook should be installed after --host=all install")
	}
	// Sanity: at least one git hook now carries the Hero marker.
	hookStatuses, err := hooks.Status(filepath.Join(env.dir, ".git"))
	if err != nil {
		t.Fatalf("hooks.Status: %v", err)
	}
	gitInstalled := false
	for _, h := range hookStatuses {
		if h.HasHero {
			gitInstalled = true
			break
		}
	}
	if !gitInstalled {
		t.Fatal("precondition: at least one git hook should have a Hero block")
	}
	// Hero no longer registers a custom merge driver (projected files
	// use built-in merge=union). Seed a legacy merge.hero-next.* stanza
	// so we can verify uninstall idempotently clears orphaned entries
	// left by older installs.
	if err := exec.Command("git", "-C", env.dir, "config",
		"merge.hero-next.driver", "hero next merge-resolve --output %A").Run(); err != nil {
		t.Fatalf("seed legacy driver: %v", err)
	}
	if !nextMergeDriverRegistered(env.dir) {
		t.Fatal("precondition: seeded legacy merge driver should be present")
	}

	if _, err := runCmd("hooks", "uninstall", "--host=all"); err != nil {
		t.Fatalf("uninstall --host=all: %v", err)
	}

	// All Hero-managed git hooks gone.
	hookStatuses, err = hooks.Status(filepath.Join(env.dir, ".git"))
	if err != nil {
		t.Fatalf("hooks.Status post-uninstall: %v", err)
	}
	for _, h := range hookStatuses {
		if h.HasHero {
			t.Errorf("hook %s still carries Hero block after uninstall --host=all", h.Name)
		}
	}
	// Claude entry removed.
	ok, err = hooks.ClaudeCompactHandoffStatus(env.dir)
	if err != nil {
		t.Fatalf("claude status: %v", err)
	}
	if ok {
		t.Error("expected claude status=false after uninstall --host=all")
	}
	// Codex entry removed.
	codexOK, err = hooks.CodexCompactHandoffStatus(env.dir)
	if err != nil {
		t.Fatalf("codex status: %v", err)
	}
	if codexOK {
		t.Error("expected codex status=false after uninstall --host=all")
	}
	// hero-next merge driver unregistered.
	if nextMergeDriverRegistered(env.dir) {
		t.Error("hero-next merge driver should be unregistered after uninstall --host=all")
	}
	// pre-commit no longer carries the hero-next managed block.
	preCommit := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	if data, rerr := os.ReadFile(preCommit); rerr == nil {
		if strings.Contains(string(data), hookMarkerStart) {
			t.Errorf("pre-commit still contains hero-next block:\n%s", data)
		}
	}
}
