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

// TestUpdateGitAttributes_BindsAllFourPathsToUnion asserts the managed
// .gitattributes block binds every projected handoff file to git's
// built-in merge=union strategy and names no custom driver. union is a
// git built-in, so it travels via the tracked .gitattributes alone —
// fresh clones / CI resolve these merges marker-free without any
// per-clone .git/config registration.
func TestUpdateGitAttributes_BindsAllFourPathsToUnion(t *testing.T) {
	dir := t.TempDir()
	if err := updateGitAttributes(dir); err != nil {
		t.Fatalf("updateGitAttributes: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if err != nil {
		t.Fatalf("read .gitattributes: %v", err)
	}
	ga := string(data)

	for _, path := range []string{
		".hero/next/*.md",
		".hero/NEXT.md",
		".hero/QUEUE.md",
		".hero/SNAPSHOT.md",
		".hero/events.log",
	} {
		want := path + " merge=union"
		if !strings.Contains(ga, want) {
			t.Errorf("missing union directive %q in:\n%s", want, ga)
		}
	}
	if strings.Contains(ga, "merge=hero-next") {
		t.Errorf(".gitattributes still names the deleted custom driver:\n%s", ga)
	}
	if !strings.Contains(ga, gaMarkerStart) || !strings.Contains(ga, gaMarkerEnd) {
		t.Errorf(".gitattributes managed block missing markers:\n%s", ga)
	}
}

// TestUpdateGitAttributes_Idempotent re-runs the writer and asserts the
// managed block doesn't grow or duplicate — re-run safety the install /
// upgrade paths depend on.
func TestUpdateGitAttributes_Idempotent(t *testing.T) {
	dir := t.TempDir()
	if err := updateGitAttributes(dir); err != nil {
		t.Fatalf("first updateGitAttributes: %v", err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if err := updateGitAttributes(dir); err != nil {
		t.Fatalf("second updateGitAttributes: %v", err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if string(first) != string(second) {
		t.Errorf("not idempotent:\nfirst  = %q\nsecond = %q", first, second)
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
	// Staging is a per-path loop so a missing path (dropped QUEUE.md,
	// empty next/*.md glob) doesn't abort staging of the others.
	if !contains(body, "for p in .hero/NEXT.md .hero/next/*.md .hero/SNAPSHOT.md .hero/QUEUE.md") {
		t.Errorf("pre-commit must loop over the projected handoff paths; got:\n%s", body)
	}
	if !contains(body, `git add -- "$p"`) {
		t.Errorf("pre-commit must git-add each path individually; got:\n%s", body)
	}
	for _, p := range []string{".hero/NEXT.md", ".hero/next/*.md", ".hero/SNAPSHOT.md", ".hero/QUEUE.md"} {
		if !contains(body, p) {
			t.Errorf("pre-commit staging must include %q; got:\n%s", p, body)
		}
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
		t.Errorf("pre-commit must reference .hero/QUEUE.md in the staging loop; got:\n%s", body)
	}
	if !contains(body, "for p in .hero/NEXT.md .hero/next/*.md .hero/SNAPSHOT.md .hero/QUEUE.md") {
		t.Errorf("pre-commit must stage NEXT.md, next/*.md, SNAPSHOT.md, and QUEUE.md in the loop; got:\n%s", body)
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

func TestHookScript_IndexFreshnessPipelineOrder(t *testing.T) {
	tests := []struct {
		kind string
		want []string
	}{
		{
			kind: "pre-commit",
			want: []string{
				"hero index --if-stale -q >/dev/null 2>&1 || true",
				"hero scan --code --incremental --deadline 10s -q >/dev/null 2>&1 || true",
				"hero next checkpoint -q >/dev/null 2>&1 || true",
				"hero queue write -q >/dev/null 2>&1 || true",
				"git add -- \"$p\" 2>/dev/null || true",
			},
		},
		{
			kind: "post-merge",
			want: []string{
				"hero index --if-stale -q >/dev/null 2>&1 || true",
				"hero scan --code --incremental --deadline 10s -q >/dev/null 2>&1 || true",
				"hero next checkpoint -q >/dev/null 2>&1 || true",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			body := hookScript(tt.kind)
			last := -1
			for _, command := range tt.want {
				at := strings.Index(body, command)
				if at < 0 {
					t.Fatalf("%s hook missing %q:\n%s", tt.kind, command, body)
				}
				if at <= last {
					t.Fatalf("%s hook command %q is out of order:\n%s", tt.kind, command, body)
				}
				last = at
			}
			if tt.kind == "post-merge" {
				if strings.Contains(body, "hero queue write") || strings.Contains(body, "git add") {
					t.Fatalf("post-merge must not write queue or stage files:\n%s", body)
				}
			}
		})
	}
}

func TestManagedPreCommitSuppressesFailingHeroOutputAndAllowsCommit(t *testing.T) {
	repo := initTestGitRepo(t)
	_ = exec.Command("git", "-C", repo, "config", "user.name", "Hook Test").Run()
	_ = exec.Command("git", "-C", repo, "config", "user.email", "hook@example.com").Run()
	if err := installNextHooksQuiet(repo); err != nil {
		t.Fatalf("install hooks: %v", err)
	}

	binDir := t.TempDir()
	stub := filepath.Join(binDir, "hero")
	if err := os.WriteFile(stub, []byte(`#!/usr/bin/env sh
echo "stub hero stdout"
echo "stub hero stderr" >&2
exit 41
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("commit me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repo, "add", "tracked.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	commit := exec.Command("git", "-C", repo, "commit", "-m", "silent hook")
	commit.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := commit.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit must survive failing Hero commands: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "stub hero stdout") ||
		strings.Contains(string(out), "stub hero stderr") {
		t.Fatalf("managed hook leaked failing command output:\n%s", out)
	}
}

func TestInstallHooksHelpUsesFullScanForManualRecovery(t *testing.T) {
	if !strings.Contains(nextInstallHooksCmd.Long, "hero scan --code") {
		t.Fatalf("install-hooks help missing full-scan recovery command:\n%s", nextInstallHooksCmd.Long)
	}
	if strings.Contains(nextInstallHooksCmd.Long, "hero scan --code --incremental") {
		t.Fatalf("incremental scan cannot recover a missing cache:\n%s", nextInstallHooksCmd.Long)
	}
}

// TestHandoffFileList_SingleSourceOfTruth is the single-list invariant
// (Secondary Defect 2): the staging `git add` pathspec and the
// .gitattributes merge=union block must cover the exact same set of
// handoff paths, because both are derived from handoffFilePaths. A
// future edit that adds a path to staging but forgets .gitattributes
// (or vice versa) turns this red.
func TestHandoffFileList_SingleSourceOfTruth(t *testing.T) {
	// Every path in the constant must appear in BOTH the staging line
	// and the .gitattributes block.
	stagingBody := hookScript("pre-commit")

	dir := t.TempDir()
	if err := updateGitAttributes(dir); err != nil {
		t.Fatalf("updateGitAttributes: %v", err)
	}
	gaData, err := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if err != nil {
		t.Fatalf("read .gitattributes: %v", err)
	}
	ga := string(gaData)

	for _, p := range handoffFilePaths {
		if !contains(stagingBody, p) {
			t.Errorf("staging list missing %q (derived from handoffFilePaths); got:\n%s", p, stagingBody)
		}
		if !contains(ga, p+" merge=union") {
			t.Errorf(".gitattributes missing %q merge=union (derived from handoffFilePaths); got:\n%s", p, ga)
		}
	}

	// SNAPSHOT.md specifically must be in both — it was the historical
	// gap that motivated the single-source-of-truth refactor.
	if !contains(stagingBody, ".hero/SNAPSHOT.md") {
		t.Errorf("staging must include .hero/SNAPSHOT.md; got:\n%s", stagingBody)
	}
	if !contains(ga, ".hero/SNAPSHOT.md merge=union") {
		t.Errorf(".gitattributes must include .hero/SNAPSHOT.md merge=union; got:\n%s", ga)
	}

	// The constant must NOT include gitignored local-only paths.
	for _, p := range handoffFilePaths {
		if strings.Contains(p, ".local.md") || strings.Contains(p, "graph.db") {
			t.Errorf("handoffFilePaths must never include gitignored path %q", p)
		}
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
	postMerge, err := os.ReadFile(filepath.Join(repo, ".git", "hooks", "post-merge"))
	if err != nil {
		t.Fatalf("read regenerated post-merge hook: %v", err)
	}
	for _, generated := range []string{string(body), string(postMerge)} {
		if !strings.Contains(generated, "hero scan --code --incremental --deadline 10s -q >/dev/null 2>&1 || true") {
			t.Errorf("regenerated hook missing index freshness command:\n%s", generated)
		}
	}
	if stale, err := preCommitHookStale(repo); err != nil || stale {
		t.Fatalf("regenerated pre-commit current = %v, err=%v", !stale, err)
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

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
