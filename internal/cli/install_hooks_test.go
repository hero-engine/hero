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

// --- uninstallNextHooks coverage ----------------------------------------

// initGitRepoForUninstall stands up a minimal git repo we can install
// next-hooks into for the uninstall tests below.
func initGitRepoForUninstall(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := exec.Command("git", "init", "-q", dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	_ = exec.Command("git", "-C", dir, "config", "user.email", "t@t.invalid").Run()
	_ = exec.Command("git", "-C", dir, "config", "user.name", "test").Run()
	return dir
}

// TestUninstallNextHooks_RemovesPreCommitBlock — install next-hooks,
// then uninstall, and assert the managed block (and the otherwise-empty
// hook stub) is gone from .git/hooks/pre-commit.
func TestUninstallNextHooks_RemovesPreCommitBlock(t *testing.T) {
	dir := initGitRepoForUninstall(t)
	if err := installNextHooksQuiet(dir); err != nil {
		t.Fatalf("install: %v", err)
	}
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if data, err := os.ReadFile(hookPath); err != nil || !strings.Contains(string(data), hookMarkerStart) {
		t.Fatalf("precondition: pre-commit should have the hero next block; err=%v data=%q", err, data)
	}

	removed, err := uninstallNextHooks(dir)
	if err != nil {
		t.Fatalf("uninstallNextHooks: %v", err)
	}
	if len(removed) == 0 {
		t.Fatal("expected uninstallNextHooks to report removed paths")
	}
	// File should be gone entirely (stub-only after strip).
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook should be removed when no user content remains: stat err=%v", err)
	}
}

// TestUninstallNextHooks_RemovesGitAttributesBlock — installs, then
// uninstalls; the .gitattributes file should be removed since it had
// no content outside the managed block.
func TestUninstallNextHooks_RemovesGitAttributesBlock(t *testing.T) {
	dir := initGitRepoForUninstall(t)
	if err := installNextHooksQuiet(dir); err != nil {
		t.Fatalf("install: %v", err)
	}
	gaPath := filepath.Join(dir, ".gitattributes")
	if data, err := os.ReadFile(gaPath); err != nil || !strings.Contains(string(data), gaMarkerStart) {
		t.Fatalf("precondition: .gitattributes should have the hero block; err=%v", err)
	}

	if _, err := uninstallNextHooks(dir); err != nil {
		t.Fatalf("uninstallNextHooks: %v", err)
	}
	if _, err := os.Stat(gaPath); !os.IsNotExist(err) {
		t.Errorf(".gitattributes should be removed when no user content remains: stat err=%v", err)
	}
}

// TestUninstallNextHooks_ClearsLegacyMergeDriver — Hero no longer
// registers a custom merge driver (projected files use the built-in
// merge=union), but older installs left a merge.hero-next.* stanza in
// .git/config. Uninstall must idempotently clear that orphaned entry so
// upgraded clones don't carry it forever.
func TestUninstallNextHooks_ClearsLegacyMergeDriver(t *testing.T) {
	dir := initGitRepoForUninstall(t)
	// Install no longer registers the driver — simulate a legacy
	// install by writing the orphaned stanza directly.
	if err := exec.Command("git", "-C", dir, "config",
		"merge.hero-next.driver", "hero next merge-resolve --output %A").Run(); err != nil {
		t.Fatalf("seed legacy driver: %v", err)
	}
	if !nextMergeDriverRegistered(dir) {
		t.Fatal("precondition: legacy merge driver should be present after seeding")
	}

	if _, err := uninstallNextHooks(dir); err != nil {
		t.Fatalf("uninstallNextHooks: %v", err)
	}
	if nextMergeDriverRegistered(dir) {
		t.Error("expected legacy merge.hero-next.* stanza to be cleared after uninstall")
	}
}

// TestInstallNextHooks_DoesNotRegisterMergeDriver guards the core
// simplification: install must NOT write a custom merge driver to
// .git/config (the whole point of switching to built-in merge=union is
// that nothing per-clone is required).
func TestInstallNextHooks_DoesNotRegisterMergeDriver(t *testing.T) {
	dir := initGitRepoForUninstall(t)
	if err := installNextHooksQuiet(dir); err != nil {
		t.Fatalf("install: %v", err)
	}
	if nextMergeDriverRegistered(dir) {
		t.Error("install must not register a custom .git/config merge driver")
	}
}

// TestUninstallNextHooks_PreservesUserContentOutsideMarkers — when the
// hook file and .gitattributes carry user content alongside the hero
// block, uninstall must strip only the managed block.
func TestUninstallNextHooks_PreservesUserContentOutsideMarkers(t *testing.T) {
	dir := initGitRepoForUninstall(t)
	// Seed pre-commit with user content first.
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatal(err)
	}
	const userHookBody = "#!/usr/bin/env bash\nset -e\necho running my custom check\nmy-linter || exit 1\n"
	if err := os.WriteFile(hookPath, []byte(userHookBody), 0o755); err != nil {
		t.Fatal(err)
	}
	// Seed .gitattributes with user content.
	gaPath := filepath.Join(dir, ".gitattributes")
	const userGAContent = "*.go text eol=lf\n*.png binary\n"
	if err := os.WriteFile(gaPath, []byte(userGAContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Install — adds hero block alongside user content.
	if err := installNextHooksQuiet(dir); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Now uninstall.
	if _, err := uninstallNextHooks(dir); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	// Hook file: must still exist, must contain the user content, must
	// not contain the hero marker.
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook after uninstall: %v", err)
	}
	gotStr := string(got)
	if strings.Contains(gotStr, hookMarkerStart) {
		t.Error("hero marker survived in pre-commit hook")
	}
	if !strings.Contains(gotStr, "echo running my custom check") {
		t.Errorf("user content stripped from pre-commit hook:\n%s", gotStr)
	}
	if !strings.Contains(gotStr, "my-linter || exit 1") {
		t.Errorf("user content stripped from pre-commit hook:\n%s", gotStr)
	}

	// .gitattributes: must still exist with the user content, no marker.
	gaGot, err := os.ReadFile(gaPath)
	if err != nil {
		t.Fatalf("read .gitattributes after uninstall: %v", err)
	}
	gaStr := string(gaGot)
	if strings.Contains(gaStr, gaMarkerStart) {
		t.Error("hero marker survived in .gitattributes")
	}
	if !strings.Contains(gaStr, "*.go text eol=lf") {
		t.Errorf("user content stripped from .gitattributes:\n%s", gaStr)
	}
}

// TestUninstallNextHooks_IdempotentNoOpWhenNotInstalled — running
// uninstall on a fresh git repo (or after a prior uninstall) must not
// error and must report no removals.
func TestUninstallNextHooks_IdempotentNoOpWhenNotInstalled(t *testing.T) {
	dir := initGitRepoForUninstall(t)

	removed, err := uninstallNextHooks(dir)
	if err != nil {
		t.Fatalf("uninstall on never-installed: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("expected zero removals on never-installed; got %v", removed)
	}

	// Install once, uninstall twice — second uninstall is also a no-op.
	if err := installNextHooksQuiet(dir); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := uninstallNextHooks(dir); err != nil {
		t.Fatalf("first uninstall: %v", err)
	}
	removed2, err := uninstallNextHooks(dir)
	if err != nil {
		t.Fatalf("second uninstall: %v", err)
	}
	if len(removed2) != 0 {
		t.Errorf("expected zero removals on second uninstall; got %v", removed2)
	}
}

// TestRunHooksUninstall_AlsoRemovesNextHooks — wire-up test: install
// the general `# Hero git hook` block AND the hero-next block, run
// `hero hooks uninstall` (no --host flag), assert both are gone.
func TestRunHooksUninstall_AlsoRemovesNextHooks(t *testing.T) {
	env := newTestEnv(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	_ = exec.Command("git", "-C", env.dir, "config", "user.email", "t@t.invalid").Run()
	_ = exec.Command("git", "-C", env.dir, "config", "user.name", "test").Run()
	defer resetHostHookFlag()

	// Install the general hooks (via the `hooks` CLI surface).
	if _, err := runCmd("hooks", "install"); err != nil {
		t.Fatalf("hooks install: %v", err)
	}
	// Install the hero-next hooks too.
	if err := installNextHooksQuiet(env.dir); err != nil {
		t.Fatalf("installNextHooksQuiet: %v", err)
	}

	// Sanity precondition: pre-commit has BOTH blocks.
	hookPath := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	before, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read pre-commit: %v", err)
	}
	if !strings.Contains(string(before), hookMarkerStart) {
		t.Fatalf("precondition: pre-commit should have hero-next block:\n%s", before)
	}

	// Run general uninstall — should also strip next-hooks.
	if _, err := runCmd("hooks", "uninstall"); err != nil {
		t.Fatalf("hooks uninstall: %v", err)
	}

	// hero-next merge driver should be unregistered.
	if nextMergeDriverRegistered(env.dir) {
		t.Error("merge driver should be unregistered after hooks uninstall")
	}
	// pre-commit hook should not contain the hero-next block (and may
	// not exist at all if it was stub-only).
	if data, err := os.ReadFile(hookPath); err == nil {
		if strings.Contains(string(data), hookMarkerStart) {
			t.Errorf("hero-next block survived hooks uninstall:\n%s", data)
		}
	}
}

// TestHostHooksUninstall_AllRemovesNextHooks — install everything via
// `--host=all`, plus the hero-next block separately, then uninstall via
// `--host=all`. All three (general / next / claude) must be gone.
func TestHostHooksUninstall_AllRemovesNextHooks(t *testing.T) {
	env := newTestEnvEmpty(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	_ = exec.Command("git", "-C", env.dir, "config", "user.email", "t@t.invalid").Run()
	_ = exec.Command("git", "-C", env.dir, "config", "user.name", "test").Run()
	if _, err := runCmd("init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer resetHostHookFlag()

	// hero init may have installed the next-hooks already; ensure they
	// are in place explicitly so we have a deterministic precondition.
	if err := installNextHooksQuiet(env.dir); err != nil {
		t.Fatalf("installNextHooksQuiet: %v", err)
	}
	// Install via --host=all (general + claude).
	if _, err := runCmd("hooks", "install", "--host=all"); err != nil {
		t.Fatalf("install --host=all: %v", err)
	}

	// Uninstall via --host=all.
	if _, err := runCmd("hooks", "uninstall", "--host=all"); err != nil {
		t.Fatalf("uninstall --host=all: %v", err)
	}

	// Assert: hero-next merge driver unregistered.
	if nextMergeDriverRegistered(env.dir) {
		t.Error("hero-next merge driver should be unregistered after --host=all uninstall")
	}
	// pre-commit should not contain hero-next block.
	hookPath := filepath.Join(env.dir, ".git", "hooks", "pre-commit")
	if data, err := os.ReadFile(hookPath); err == nil {
		if strings.Contains(string(data), hookMarkerStart) {
			t.Errorf("hero-next block survived --host=all uninstall:\n%s", data)
		}
	}
	// .gitattributes should not contain hero-next block.
	gaPath := filepath.Join(env.dir, ".gitattributes")
	if data, err := os.ReadFile(gaPath); err == nil {
		if strings.Contains(string(data), gaMarkerStart) {
			t.Errorf("hero-next block survived in .gitattributes:\n%s", data)
		}
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
