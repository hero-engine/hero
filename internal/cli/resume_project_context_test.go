package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/digest"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
)

// resume_project_context_test.go is the guardrail for resume-brief-
// missing-project-context: the resume brief's project-context sections
// (`Just changed`, in-flight) must reflect real local state, not an
// empty map.
//
//   - Same-machine (Change 1): a commit made during a session becomes a
//     graph Commit node at checkpoint time, so the next resume's "Just
//     changed" lists it. Repeated checkpoints stay idempotent.
//   - Cross-machine (Change 2a): a cold clone (empty graph, only committed
//     files + git history) rebuilds project context from local git log +
//     committed specs at SessionStart ingest time.
//   - Nudge (Change 2b): a cold graph where rebuild is skipped still tells
//     the user to run `hero scan`, and never fails resume.

// gitCommitFile writes a file under dir and commits it with the given
// subject, returning the commit's full SHA.
func gitCommitFile(t *testing.T, dir, file, content, subject string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(file)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][]string{
		{"git", "add", file},
		{"git", "commit", "-q", "-m", subject},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v: %s", c, err, out)
		}
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// countCommitNodes returns the number of live Commit nodes for repoKey.
func countCommitNodes(t *testing.T, heroDir, repoKey string) int {
	t.Helper()
	store, err := graph.Open(heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	defer store.Close()
	var n int
	if err := store.DB().QueryRow(
		`SELECT COUNT(*) FROM nodes WHERE type='Commit' AND repo=? AND valid_to IS NULL`,
		repoKey,
	).Scan(&n); err != nil {
		t.Fatalf("count Commit nodes: %v", err)
	}
	return n
}

// Test_writeCheckpoint_IngestsCommitIntoGraph is Change 1 / AC-1: a
// commit made during a session must become a graph Commit node when
// writeCheckpoint runs, so the next resume's "Just changed" lists it.
func Test_writeCheckpoint_IngestsCommitIntoGraph(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := newTestEnv(t)
	initGitRepo(t, env.dir)

	const subject = "auth: add in-memory token-bucket rate limit on /login"
	sha := gitCommitFile(t, env.dir, "auth.go", "package auth\n", subject)

	repoKey := gitutil.RepoKey(env.dir)

	// Precondition: the graph has no Commit node yet — the commit reached
	// git but nothing ingested it (the bug). (initGitRepo's own init
	// commit is in git history but, like ours, not yet a graph node.)
	if got := countCommitNodes(t, env.heroDir, repoKey); got != 0 {
		t.Fatalf("precondition: expected 0 Commit nodes before checkpoint, got %d", got)
	}

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("writeCheckpoint: %v", err)
	}

	// The just-made commit is now a Commit node (along with initGitRepo's
	// init commit — at least the one we care about must be present).
	if got := countCommitNodes(t, env.heroDir, repoKey); got < 1 {
		t.Fatalf("expected the session commit to become a Commit node after checkpoint, got %d nodes", got)
	}

	// And the resume brief's "Just changed" section lists it — the exact
	// surface the user reads at session start.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	brief, err := digest.Generate(store, digest.Options{
		RepoKey: repoKey,
		Branch:  gitutil.CurrentBranch(env.dir),
	})
	store.Close()
	if err != nil {
		t.Fatalf("digest.Generate: %v", err)
	}
	md := brief.Markdown()
	if !strings.Contains(md, subject) {
		t.Errorf("resume brief 'Just changed' missing commit subject %q:\n%s", subject, md)
	}
	if !strings.Contains(md, sha[:7]) {
		t.Errorf("resume brief 'Just changed' missing commit short-sha %q:\n%s", sha[:7], md)
	}
}

// Test_writeCheckpoint_CommitIngestIdempotent is the idempotence AC:
// running the checkpoint twice over the same history must NOT create
// duplicate Commit nodes.
func Test_writeCheckpoint_CommitIngestIdempotent(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := newTestEnv(t)
	initGitRepo(t, env.dir)
	gitCommitFile(t, env.dir, "a.go", "package a\n", "first commit")
	gitCommitFile(t, env.dir, "b.go", "package b\n", "second commit")

	repoKey := gitutil.RepoKey(env.dir)

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("first writeCheckpoint: %v", err)
	}
	first := countCommitNodes(t, env.heroDir, repoKey)
	if first < 2 {
		t.Fatalf("expected at least the 2 session commits as Commit nodes after first checkpoint, got %d", first)
	}

	if _, err := writeCheckpoint(); err != nil {
		t.Fatalf("second writeCheckpoint: %v", err)
	}
	second := countCommitNodes(t, env.heroDir, repoKey)
	if second != first {
		t.Errorf("commit ingest not idempotent: %d Commit nodes after first checkpoint, %d after second (duplicate nodes created)", first, second)
	}
}

// Test_runNextIngest_ColdCloneRebuildsProjectContext is Change 2a /
// AC-2: a fresh clone (empty graph, only committed files + local git
// history + committed specs) rebuilds project context at SessionStart
// ingest time, so the first resume's "Just changed" and in-flight
// sections are non-empty — rebuilt from local sources alone.
func Test_runNextIngest_ColdCloneRebuildsProjectContext(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := newTestEnv(t)
	initGitRepo(t, env.dir)

	// A committed in-flight spec (Feature) that the rebuild must surface
	// in the brief's "In flight" section (which reads Feature nodes).
	const specSlug = "cold-clone-spec"
	const specTitle = "Cold clone in-flight spec must surface"
	env.addSpec(filepath.Join("planning", "features", specSlug, "spec.md"),
		"---\ntitle: \""+specTitle+"\"\nslug: "+specSlug+"\ntype: feature\nstatus: delivering\n---\n\n# "+specTitle+"\n")

	const commitSubject = "fix: land the cold-clone rebuild"
	gitCommitFile(t, env.dir, "fix.go", "package fix\n", commitSubject)

	repoKey := gitutil.RepoKey(env.dir)

	// Cold-graph precondition: no project-context nodes yet (fresh clone).
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	if !projectGraphCold(store, repoKey) {
		store.Close()
		t.Fatalf("precondition: graph should be cold (no project-context nodes) on a fresh clone")
	}
	store.Close()

	// Run the real SessionStart ingest entry point. With no handoff files
	// present there is nothing to ingest — but the cold-clone rebuild
	// must still fire and populate project context. resetFlags clears
	// ingestQuiet etc.
	if _, err := runCmd("next", "ingest", "-q"); err != nil {
		t.Fatalf("runNextIngest: %v", err)
	}

	// Project context is now rebuilt: a Commit node and the in-flight Bug.
	if got := countCommitNodes(t, env.heroDir, repoKey); got < 1 {
		t.Errorf("cold-clone rebuild did not create Commit nodes (got %d)", got)
	}

	store, err = graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("reopen graph: %v", err)
	}
	if projectGraphCold(store, repoKey) {
		store.Close()
		t.Fatalf("graph still cold after rebuild — project context was not rebuilt")
	}
	brief, err := digest.Generate(store, digest.Options{
		RepoKey: repoKey,
		Branch:  gitutil.CurrentBranch(env.dir),
		Domain:  graph.DomainFor(config.DefaultConfig(), graph.IntrinsicActive),
	})
	store.Close()
	if err != nil {
		t.Fatalf("digest.Generate after rebuild: %v", err)
	}
	md := brief.Markdown()
	if !strings.Contains(md, commitSubject) {
		t.Errorf("resume brief missing rebuilt commit %q:\n%s", commitSubject, md)
	}
	if !strings.Contains(md, specTitle) && !strings.Contains(md, specSlug) {
		t.Errorf("resume brief missing rebuilt in-flight spec %q:\n%s", specSlug, md)
	}
}

// Test_runResume_ColdGraphNudges is Change 2b: when the graph is cold
// and the rebuild is skipped/no-op (e.g. a non-git workspace where
// WriteGitLogGraph writes nothing and there are no specs), resume still
// emits the `hero scan` nudge to stderr and never fails.
func Test_runResume_ColdGraphNudges(t *testing.T) {
	env := newTestEnv(t)

	// Non-git workspace, no specs, no commits → the rebuild is a genuine
	// no-op and the graph stays cold. This is the deferred-path edge the
	// nudge guards.
	repoKey := gitutil.RepoKey(env.dir)
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	if !projectGraphCold(store, repoKey) {
		store.Close()
		t.Fatalf("precondition: graph should be cold")
	}
	store.Close()

	// Capture stderr around runResume to assert the nudge.
	origErr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	resetFlags()
	rootCmd.SetArgs([]string{"resume"})
	runErr := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	w.Close()
	os.Stderr = origErr

	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
	stderr := sb.String()

	if runErr != nil {
		t.Fatalf("resume must not fail on a cold graph: %v", runErr)
	}
	if !strings.Contains(stderr, "hero scan") {
		t.Errorf("cold-graph resume did not emit the `hero scan` nudge to stderr:\n%s", stderr)
	}
}
