package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUserFromOutputPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{".hero/next/chet-bellows.md", "chet-bellows"},
		{"/abs/path/.hero/next/alice.md", "alice"},
		{".hero/next/chet-bellows.local.md", ""},          // local files aren't projected
		{".hero/NEXT.md", ""},                          // project file, not user file
		{".hero/some/other/chet-bellows.md", ""},          // wrong parent dir
		{"random.md", ""},                              // no parent dir at all
	}
	for _, tc := range cases {
		got := userFromOutputPath(tc.path)
		if got != tc.want {
			t.Errorf("userFromOutputPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestMergeMarkerBlock_AppendsWhenAbsent(t *testing.T) {
	out := mergeMarkerBlock("existing user content\n", "# >>>", "# <<<", "# >>>\nMANAGED\n# <<<")
	if !contains(out, "existing user content") {
		t.Errorf("user content lost: %q", out)
	}
	if !contains(out, "MANAGED") {
		t.Errorf("managed block missing: %q", out)
	}
}

func TestMergeMarkerBlock_ReplacesWhenPresent(t *testing.T) {
	src := "user-line-1\n# >>>\nOLD\n# <<<\nuser-line-2\n"
	out := mergeMarkerBlock(src, "# >>>", "# <<<", "# >>>\nNEW\n# <<<")
	if contains(out, "OLD") {
		t.Errorf("old block survived: %q", out)
	}
	if !contains(out, "NEW") {
		t.Errorf("new block missing: %q", out)
	}
	if !contains(out, "user-line-1") || !contains(out, "user-line-2") {
		t.Errorf("user-line preservation failed: %q", out)
	}
}

func TestMergeMarkerBlock_IdempotentOnReRun(t *testing.T) {
	once := mergeMarkerBlock("", "# >>>", "# <<<", "# >>>\nM\n# <<<")
	twice := mergeMarkerBlock(once, "# >>>", "# <<<", "# >>>\nM\n# <<<")
	if once != twice {
		t.Errorf("not idempotent:\nonce  = %q\ntwice = %q", once, twice)
	}
}

// pre-commit must regenerate the projected files AND stage them, so
// that agents committing narrowly (`git add <files> && git commit`)
// don't strand the regenerated handoff state as unstaged drift.
// Spec: pre-commit-auto-stage-next.
func TestHookScript_PreCommit_StagesProjectedFiles(t *testing.T) {
	body := hookScript("pre-commit")

	if !contains(body, "hero next checkpoint -q") {
		t.Errorf("pre-commit must invoke hero next checkpoint -q; got:\n%s", body)
	}
	if !contains(body, "git add -- .hero/NEXT.md .hero/next/*.md") {
		t.Errorf("pre-commit must stage projected NEXT files with the exact pathspec; got:\n%s", body)
	}
	if !contains(body, "2>/dev/null || true") {
		t.Errorf("git add must swallow missing-file errors so a fresh repo doesn't fail the hook; got:\n%s", body)
	}
	if !contains(body, hookMarkerStart) || !contains(body, hookMarkerEnd) {
		t.Errorf("hook block must include managed markers; got:\n%s", body)
	}
}

// post-merge regenerates from graph but should NOT auto-stage —
// it runs after the merge commit is already made.
func TestHookScript_PostMerge_DoesNotStage(t *testing.T) {
	body := hookScript("post-merge")

	if !contains(body, "hero next checkpoint -q") {
		t.Errorf("post-merge must invoke hero next checkpoint -q; got:\n%s", body)
	}
	if contains(body, "git add") {
		t.Errorf("post-merge must NOT contain git add (commit already made); got:\n%s", body)
	}
	if contains(body, "hero queue write") {
		t.Errorf("post-merge must NOT regen queue (next commit picks it up); got:\n%s", body)
	}
}

// pre-commit must regenerate the QUEUE.md snapshot AND stage it so
// the cold-start surface travels with every commit.
func TestHookScript_PreCommit_RefreshesQueueSnapshot(t *testing.T) {
	body := hookScript("pre-commit")

	if !contains(body, "hero queue write -q") {
		t.Errorf("pre-commit must invoke hero queue write -q; got:\n%s", body)
	}
	if !contains(body, ".hero/QUEUE.md") {
		t.Errorf("pre-commit must reference .hero/QUEUE.md in git add pathspec; got:\n%s", body)
	}
	if !contains(body, "git add -- .hero/NEXT.md .hero/next/*.md .hero/QUEUE.md") {
		t.Errorf("pre-commit must stage NEXT.md, next/*.md, and QUEUE.md together; got:\n%s", body)
	}
}

// pre-commit must surgically refresh the search/list index so spec
// edits in this commit are reflected in `hero search` / `hero list`
// without requiring a separate `hero index` run.
// Spec: index-staleness-auto-refresh.
func TestHookScript_PreCommit_RefreshesIndex(t *testing.T) {
	body := hookScript("pre-commit")

	if !contains(body, "hero index --if-stale -q") {
		t.Errorf("pre-commit must invoke hero index --if-stale -q; got:\n%s", body)
	}
}

// initTestGitRepo bootstraps a temp directory as a real git repo and
// returns its path. Used by hook-refresh integration tests that need
// resolveGitDir to succeed.
func initTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q", dir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return dir
}

// installStaleManagedBlock writes a pre-commit hook with the marker
// block but stale content (a previous version of the hook script).
// Returns the file path so tests can re-read it after refresh.
func installStaleManagedBlock(t *testing.T, projectRoot string) string {
	t.Helper()
	hookPath := filepath.Join(projectRoot, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	stale := fmt.Sprintf(`#!/usr/bin/env bash
%s
# Stale managed block — predates `+"`hero queue write`"+` line.
if command -v hero >/dev/null 2>&1; then
  hero next checkpoint -q || true
  git add -- .hero/NEXT.md .hero/next/*.md 2>/dev/null || true
fi
%s
`, hookMarkerStart, hookMarkerEnd)
	if err := os.WriteFile(hookPath, []byte(stale), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return hookPath
}

func TestPreCommitHookStale_DetectsContentDrift(t *testing.T) {
	repo := initTestGitRepo(t)
	installStaleManagedBlock(t, repo)

	stale, err := preCommitHookStale(repo)
	if err != nil {
		t.Fatalf("preCommitHookStale: %v", err)
	}
	if !stale {
		t.Error("expected stale=true for an old managed block")
	}
}

func TestPreCommitHookStale_CurrentBlockIsFresh(t *testing.T) {
	repo := initTestGitRepo(t)
	hookPath := filepath.Join(repo, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Write the *current* hook script — should match exactly.
	body := "#!/usr/bin/env bash\n" + hookScript("pre-commit") + "\n"
	if err := os.WriteFile(hookPath, []byte(body), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	stale, err := preCommitHookStale(repo)
	if err != nil {
		t.Fatalf("preCommitHookStale: %v", err)
	}
	if stale {
		t.Errorf("expected stale=false when block matches hookScript() output")
	}
}

func TestRefreshHooksIfPresent_RefreshesStale(t *testing.T) {
	repo := initTestGitRepo(t)
	hookPath := installStaleManagedBlock(t, repo)

	var out bytes.Buffer
	if err := refreshHooksIfPresent(repo, false, &out); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if !strings.Contains(out.String(), "refreshed pre-commit hook") {
		t.Errorf("expected refresh message, got: %q", out.String())
	}

	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(body), "hero queue write -q") {
		t.Errorf("post-refresh hook missing new content:\n%s", body)
	}
}

func TestRefreshHooksIfPresent_DryRun(t *testing.T) {
	repo := initTestGitRepo(t)
	hookPath := installStaleManagedBlock(t, repo)
	originalBody, _ := os.ReadFile(hookPath)

	var out bytes.Buffer
	if err := refreshHooksIfPresent(repo, true, &out); err != nil {
		t.Fatalf("dry-run refresh: %v", err)
	}
	if !strings.Contains(out.String(), "would refresh pre-commit hook") {
		t.Errorf("dry-run should announce intent, got: %q", out.String())
	}

	// File must not have changed.
	postBody, _ := os.ReadFile(hookPath)
	if string(postBody) != string(originalBody) {
		t.Error("dry-run modified hook file")
	}
}

func TestRefreshHooksIfPresent_SkipsWhenAbsent(t *testing.T) {
	repo := initTestGitRepo(t)
	// No managed block installed.

	var out bytes.Buffer
	if err := refreshHooksIfPresent(repo, false, &out); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if out.String() != "" {
		t.Errorf("expected silent skip when hook not installed, got: %q", out.String())
	}
}

func TestRefreshHooksIfPresent_SkipsNonGitRepo(t *testing.T) {
	dir := t.TempDir()

	var out bytes.Buffer
	if err := refreshHooksIfPresent(dir, false, &out); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if out.String() != "" {
		t.Errorf("expected silent skip outside git, got: %q", out.String())
	}
}

func TestRefreshHooksIfPresent_DryRunCurrent(t *testing.T) {
	repo := initTestGitRepo(t)
	hookPath := filepath.Join(repo, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	body := "#!/usr/bin/env bash\n" + hookScript("pre-commit") + "\n"
	if err := os.WriteFile(hookPath, []byte(body), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var out bytes.Buffer
	if err := refreshHooksIfPresent(repo, true, &out); err != nil {
		t.Fatalf("dry-run on current hook: %v", err)
	}
	if !strings.Contains(out.String(), "is current") {
		t.Errorf("expected 'is current' message, got: %q", out.String())
	}
}

func TestIsQueueOutputPath(t *testing.T) {
	cases := map[string]bool{
		"/repo/.hero/QUEUE.md":            true,
		".hero/QUEUE.md":                  true,
		"QUEUE.md":                        true,
		"/repo/.hero/NEXT.md":             false,
		"/repo/.hero/next/chet-bellows.md":   false,
		"/repo/.hero/queue.md":            false, // case-sensitive
		"/repo/.hero/foo/QUEUE.md.backup": false,
	}
	for path, want := range cases {
		got := isQueueOutputPath(path)
		if got != want {
			t.Errorf("isQueueOutputPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
