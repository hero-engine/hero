package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These integration tests exercise the real pre-commit staging path in
// a temp git repo: they install Hero hooks, make a commit, and assert
// the projected handoff files land in the commit tree (not as unstaged
// drift). They prove the consolidation closed the projecting-but-not-
// staging gap. Spec: next-unconditional-commit-staging.
//
// The staging `git add` lives inside the hook's `if command -v hero`
// guard, so these tests put a no-op `hero` stub on PATH — that lets the
// checkpoint/index/queue calls succeed as no-ops while the real
// `git add -- ...` line runs against whatever handoff files exist on
// disk. We pre-create the handoff files ourselves to stand in for what
// `hero next checkpoint` would have projected.

// stubHeroOnPath creates a no-op `hero` executable in a temp dir and
// returns that dir, suitable for prepending to PATH so the pre-commit
// hook's `command -v hero` guard passes and the staging line runs.
func stubHeroOnPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("pre-commit shell hook staging test is POSIX-only")
	}
	binDir := t.TempDir()
	stub := filepath.Join(binDir, "hero")
	// Exit 0 for every subcommand so the best-effort `|| true` calls
	// are no-ops and the subsequent `git add` line executes.
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write hero stub: %v", err)
	}
	return binDir
}

// gitCommitWithHero runs `git commit` in dir with the hero stub dir
// prepended to PATH so the pre-commit hook can find `hero` and execute
// its staging line. Deterministic identity mirrors gitRun.
func gitCommitWithHero(t *testing.T, dir, stubBinDir string, args ...string) (string, error) {
	t.Helper()
	full := append([]string{"commit"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"PATH="+stubBinDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE=2025-01-15T08:00:00Z",
		"GIT_COMMITTER_DATE=2025-01-15T08:00:00Z",
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// committedTree returns the set of paths in HEAD's tree.
func committedTree(t *testing.T, dir string) map[string]bool {
	t.Helper()
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git ls-tree: %v\n%s", err, out)
	}
	set := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			set[line] = true
		}
	}
	return set
}

// writeHandoffFiles creates the projected handoff files on disk so the
// pre-commit staging line has something to add. Mirrors what
// `hero next checkpoint` would project.
func writeHandoffFiles(t *testing.T, root string, snapshot bool) {
	t.Helper()
	heroDir := filepath.Join(root, ".hero")
	nextDir := filepath.Join(heroDir, "next")
	if err := os.MkdirAll(nextDir, 0o755); err != nil {
		t.Fatalf("mkdir next: %v", err)
	}
	files := map[string]string{
		filepath.Join(heroDir, "NEXT.md"):  "# NEXT\nhandoff state\n",
		filepath.Join(nextDir, "alice.md"): "# alice handoff\n",
		filepath.Join(heroDir, "QUEUE.md"): "# QUEUE\n",
	}
	if snapshot {
		files[filepath.Join(heroDir, "SNAPSHOT.md")] = "# SNAPSHOT\nproject shape\n"
	}
	for p, body := range files {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
}

// TestIntegration_DefaultInstall_StagesHandoffFiles is Test Plan #3 —
// the default install path (hero next install-hooks) stages NEXT.md and
// SNAPSHOT.md so they travel with the commit tree.
func TestIntegration_DefaultInstall_StagesHandoffFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")
	stubBin := stubHeroOnPath(t)

	if err := installNextHooksQuiet(env.dir); err != nil {
		t.Fatalf("installNextHooksQuiet: %v", err)
	}

	// A tracked code change the user explicitly stages.
	codeFile := filepath.Join(env.dir, "main.go")
	if err := os.WriteFile(codeFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write code: %v", err)
	}
	// Projected handoff files exist on disk (what checkpoint would write).
	writeHandoffFiles(t, env.dir, true)

	gitRun(t, env.dir, "add", "main.go")
	if out, err := gitCommitWithHero(t, env.dir, stubBin, "-q", "-m", "feat: change"); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}

	tree := committedTree(t, env.dir)
	for _, want := range []string{".hero/NEXT.md", ".hero/SNAPSHOT.md", ".hero/next/alice.md", ".hero/QUEUE.md"} {
		if !tree[want] {
			t.Errorf("expected %q in commit tree, got: %v", want, keys(tree))
		}
	}
}

// TestIntegration_GenericInstall_StagesHandoffFiles is Test Plan #4 —
// THE core regression test for this bug. Running the GENERIC
// `hero hooks install` surface must produce a pre-commit that stages
// handoff files on commit, proving the consolidation closed the gap
// where the generic installer projected-but-never-staged.
func TestIntegration_GenericInstall_StagesHandoffFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")
	stubBin := stubHeroOnPath(t)

	// Run the GENERIC install surface (not hero next install-hooks).
	if _, err := runCmd("hooks", "install"); err != nil {
		t.Fatalf("hero hooks install: %v", err)
	}

	// Sanity: the generic `# Hero git hook` marker AND the next-hooks
	// staging marker must coexist in the pre-commit hook.
	hookBody, err := os.ReadFile(filepath.Join(env.dir, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("read pre-commit: %v", err)
	}
	if !strings.Contains(string(hookBody), "# Hero git hook") {
		t.Errorf("generic Hero hook marker missing after install:\n%s", hookBody)
	}
	if !strings.Contains(string(hookBody), hookMarkerStart) {
		t.Errorf("next-hooks staging marker missing after generic install:\n%s", hookBody)
	}
	if !strings.Contains(string(hookBody), `git add -- "$p"`) {
		t.Errorf("staging git add missing after generic install:\n%s", hookBody)
	}
	if !strings.Contains(string(hookBody), ".hero/SNAPSHOT.md") {
		t.Errorf("staging loop missing SNAPSHOT.md after generic install:\n%s", hookBody)
	}

	codeFile := filepath.Join(env.dir, "main.go")
	if err := os.WriteFile(codeFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write code: %v", err)
	}
	writeHandoffFiles(t, env.dir, true)

	gitRun(t, env.dir, "add", "main.go")
	if out, err := gitCommitWithHero(t, env.dir, stubBin, "-q", "-m", "feat: change"); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}

	tree := committedTree(t, env.dir)
	for _, want := range []string{".hero/NEXT.md", ".hero/SNAPSHOT.md"} {
		if !tree[want] {
			t.Errorf("GENERIC install failed to stage %q — projecting-but-not-staging gap NOT closed; tree: %v", want, keys(tree))
		}
	}
}

// TestIntegration_GitignoredFilesNeverStaged is Test Plan #5 —
// gitignored handoff files (*.local.md, graph.db) must never be staged
// even when present on disk.
func TestIntegration_GitignoredFilesNeverStaged(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")
	stubBin := stubHeroOnPath(t)

	// Ignore the local-only / db files the way a real Hero workspace does.
	gitignore := ".hero/next/*.local.md\n.hero/graph.db\n.hero/graph.db*\n"
	if err := os.WriteFile(filepath.Join(env.dir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	if err := installNextHooksQuiet(env.dir); err != nil {
		t.Fatalf("installNextHooksQuiet: %v", err)
	}

	writeHandoffFiles(t, env.dir, true)
	// Gitignored local-only files.
	if err := os.WriteFile(filepath.Join(env.dir, ".hero", "next", "foo.local.md"), []byte("local\n"), 0o644); err != nil {
		t.Fatalf("write local.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(env.dir, ".hero", "graph.db"), []byte("db\n"), 0o644); err != nil {
		t.Fatalf("write graph.db: %v", err)
	}

	codeFile := filepath.Join(env.dir, "main.go")
	if err := os.WriteFile(codeFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write code: %v", err)
	}
	gitRun(t, env.dir, "add", "main.go", ".gitignore")
	if out, err := gitCommitWithHero(t, env.dir, stubBin, "-q", "-m", "feat: change"); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}

	tree := committedTree(t, env.dir)
	for _, forbidden := range []string{".hero/next/foo.local.md", ".hero/graph.db"} {
		if tree[forbidden] {
			t.Errorf("gitignored file %q was staged — must never travel; tree: %v", forbidden, keys(tree))
		}
	}
	// Sanity: the legitimate handoff files still made it.
	if !tree[".hero/NEXT.md"] {
		t.Errorf("expected .hero/NEXT.md in tree; got: %v", keys(tree))
	}
}

// TestIntegration_MissingQueueIsNoOp is Test Plan #6 — when
// .hero/QUEUE.md does not exist, the remaining handoff files still
// stage and the commit succeeds without error.
func TestIntegration_MissingQueueIsNoOp(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")
	stubBin := stubHeroOnPath(t)

	if err := installNextHooksQuiet(env.dir); err != nil {
		t.Fatalf("installNextHooksQuiet: %v", err)
	}

	// Project handoff files but NOT QUEUE.md.
	heroDir := filepath.Join(env.dir, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "next"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "NEXT.md"), []byte("# NEXT\n"), 0o644); err != nil {
		t.Fatalf("write NEXT.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "SNAPSHOT.md"), []byte("# SNAPSHOT\n"), 0o644); err != nil {
		t.Fatalf("write SNAPSHOT.md: %v", err)
	}
	// No QUEUE.md on disk.

	codeFile := filepath.Join(env.dir, "main.go")
	if err := os.WriteFile(codeFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write code: %v", err)
	}
	gitRun(t, env.dir, "add", "main.go")
	if out, err := gitCommitWithHero(t, env.dir, stubBin, "-q", "-m", "feat: change"); err != nil {
		t.Fatalf("commit must succeed without QUEUE.md: %v\n%s", err, out)
	}

	tree := committedTree(t, env.dir)
	if !tree[".hero/NEXT.md"] || !tree[".hero/SNAPSHOT.md"] {
		t.Errorf("remaining handoff files must stage when QUEUE.md absent; tree: %v", keys(tree))
	}
	if tree[".hero/QUEUE.md"] {
		t.Errorf("QUEUE.md should not be in tree (never created); tree: %v", keys(tree))
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
