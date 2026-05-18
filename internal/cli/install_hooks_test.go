package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallSelfHealsPreCommitHook covers Symptom 5 of
// spec-lifecycle-hygiene-breakdown: `hero install <target>` should
// install the pre-commit hook when no managed block exists and the
// user hasn't opted out. Mirrors the init-time install so teammates
// who discover `hero install` first still get hook coverage.
func TestInstallSelfHealsPreCommitHook(t *testing.T) {
	_ = newTestEnvEmpty(t)
	targetDir := t.TempDir()
	if err := exec.Command("git", "init", "-q", targetDir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	out, err := runCmd("install", "project", targetDir, "--target", "codex")
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	if !strings.Contains(out, "Installed pre-commit hook") {
		t.Errorf("install output missing hook confirmation: %q", out)
	}

	hookPath := filepath.Join(targetDir, ".git", "hooks", "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-commit hook not created: %v", err)
	}
	if !strings.Contains(string(data), hookMarkerStart) {
		t.Errorf("hook content missing hero managed marker: %q", string(data))
	}
	if !strings.Contains(string(data), "hero next checkpoint -q") {
		t.Errorf("hook content missing hero next checkpoint invocation: %q", string(data))
	}
}

// TestInstallNoHooksFlag asserts the --no-hooks opt-out suppresses
// the self-heal install — mirrors `hero init --no-hooks`.
func TestInstallNoHooksFlag(t *testing.T) {
	_ = newTestEnvEmpty(t)
	targetDir := t.TempDir()
	if err := exec.Command("git", "init", "-q", targetDir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	out, err := runCmd("install", "project", targetDir, "--target", "codex", "--no-hooks")
	if err != nil {
		t.Fatalf("install --no-hooks returned error: %v", err)
	}

	if strings.Contains(out, "Installed pre-commit hook") {
		t.Errorf("--no-hooks should suppress hook install message: %q", out)
	}

	hookPath := filepath.Join(targetDir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook should not exist under --no-hooks (stat err=%v)", err)
	}
}

// TestInstallSkipsHookWhenAlreadyInstalled asserts the self-heal
// path is idempotent — re-running install over a repo that already
// has the managed block neither errors nor reports a fresh install.
// Critically, the marker block survives untouched (no double-write).
func TestInstallSkipsHookWhenAlreadyInstalled(t *testing.T) {
	_ = newTestEnvEmpty(t)
	targetDir := t.TempDir()
	if err := exec.Command("git", "init", "-q", targetDir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Prime the repo with a managed block already in place.
	if err := installNextHooksQuiet(targetDir); err != nil {
		t.Fatalf("seed installNextHooksQuiet: %v", err)
	}
	hookPath := filepath.Join(targetDir, ".git", "hooks", "pre-commit")
	before, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read seeded hook: %v", err)
	}

	out, err := runCmd("install", "project", targetDir, "--target", "codex")
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	if strings.Contains(out, "Installed pre-commit hook") {
		t.Errorf("install should not announce a fresh hook install when the managed block is already present: %q", out)
	}

	after, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read post-install hook: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("hook file changed across idempotent re-install\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestInstallSkipsHookOutsideGitRepo asserts the self-heal install
// is silently skipped when the target directory is not a git repo.
// Non-git targets are a legitimate use case (greenfield install
// before `git init`), so this must not error.
func TestInstallSkipsHookOutsideGitRepo(t *testing.T) {
	_ = newTestEnvEmpty(t)
	targetDir := t.TempDir()
	// Intentionally no `git init` — install should still succeed
	// and just skip the hook install.

	out, err := runCmd("install", "project", targetDir, "--target", "codex")
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	if strings.Contains(out, "Installed pre-commit hook") {
		t.Errorf("install must not claim hook install outside a git repo: %q", out)
	}

	hookPath := filepath.Join(targetDir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("hook should not exist outside a git repo (stat err=%v)", err)
	}
}

// TestInstallRespectsExplicitOptOutSentinel asserts that a user who
// explicitly removed the hook markers and `touch`ed `.hero/.no-hooks`
// is NOT silently reinstalled-over by the next `hero install`. This
// preserves the user-removal opt-out semantics that `refreshHooksIfPresent`
// has from the upgrade path: when the user has signaled they don't
// want the hook, re-running install must not bring it back.
func TestInstallRespectsExplicitOptOutSentinel(t *testing.T) {
	_ = newTestEnvEmpty(t)
	targetDir := t.TempDir()
	if err := exec.Command("git", "init", "-q", targetDir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Simulate the post-removal state: there is no managed block in
	// the pre-commit hook, but the user has `touch`ed the opt-out
	// sentinel inside .hero/ — they're telling us "don't reinstall".
	heroDir := filepath.Join(targetDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll .hero: %v", err)
	}
	sentinel := filepath.Join(heroDir, ".no-hooks")
	if err := os.WriteFile(sentinel, nil, 0o644); err != nil {
		t.Fatalf("WriteFile sentinel: %v", err)
	}

	out, err := runCmd("install", "project", targetDir, "--target", "codex")
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	if strings.Contains(out, "Installed pre-commit hook") {
		t.Errorf("install must not reinstall the hook when the opt-out sentinel is present: %q", out)
	}

	hookPath := filepath.Join(targetDir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("hook should not exist when .hero/.no-hooks is present (stat err=%v)", err)
	}
}

// TestInstallSkipsHookInDryRun asserts that --dry-run suppresses the
// hook self-heal — the install command's dry-run contract is to
// write nothing.
func TestInstallSkipsHookInDryRun(t *testing.T) {
	_ = newTestEnvEmpty(t)
	targetDir := t.TempDir()
	if err := exec.Command("git", "init", "-q", targetDir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	if _, err := runCmd("install", "project", targetDir, "--target", "codex", "--dry-run"); err != nil {
		t.Fatalf("install --dry-run returned error: %v", err)
	}

	hookPath := filepath.Join(targetDir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("hook should not be created under --dry-run (stat err=%v)", err)
	}
}
