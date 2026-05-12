package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

func makeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-q", "-b", "main"},
		{"git", "config", "user.email", "alice@example.com"},
		{"git", "config", "user.name", "Alice"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v: %s", c, err, out)
		}
	}
	return dir
}

func writeAndCommit(t *testing.T, dir, file, content, msg string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(file)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][]string{
		{"git", "add", file},
		{"git", "commit", "-q", "-m", msg},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v: %s", c, err, out)
		}
	}
}

func openGitTestStore(t *testing.T) *graph.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := graph.Open(filepath.Join(dir, "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestWriteGitLogGraphCreatesCommitAndPersonNodes(t *testing.T) {
	repo := makeRepo(t)
	writeAndCommit(t, repo, "a.txt", "hello", "first commit")
	writeAndCommit(t, repo, "b.txt", "world", "second commit")

	store := openGitTestStore(t)
	summary, err := WriteGitLogGraph(repo, "test-repo", 0, store)
	if err != nil {
		t.Fatalf("WriteGitLogGraph: %v", err)
	}
	if summary.Commits != 2 {
		t.Errorf("Commits = %d, want 2", summary.Commits)
	}
	stats, _ := store.Stats()
	if stats.NodesByType["Commit"] != 2 {
		t.Errorf("Commit nodes = %d, want 2", stats.NodesByType["Commit"])
	}
	if stats.NodesByType["Person"] != 1 {
		t.Errorf("Person nodes = %d, want 1", stats.NodesByType["Person"])
	}
	if stats.EdgesByType["authored_by"] != 2 {
		t.Errorf("authored_by edges = %d, want 2", stats.EdgesByType["authored_by"])
	}
}

func TestWriteGitLogGraphTouchesEdgesWhenFileNodeExists(t *testing.T) {
	repo := makeRepo(t)
	writeAndCommit(t, repo, "a.txt", "hello", "first commit")

	store := openGitTestStore(t)
	// Pre-seed a File node so touches edges have a target.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "File", Key: "test-repo:a.txt", ContentHash: "h",
	}); err != nil {
		t.Fatalf("seed File: %v", err)
	}
	if _, err := WriteGitLogGraph(repo, "test-repo", 0, store); err != nil {
		t.Fatalf("WriteGitLogGraph: %v", err)
	}
	stats, _ := store.Stats()
	if stats.EdgesByType["touches"] != 1 {
		t.Errorf("touches edges = %d, want 1", stats.EdgesByType["touches"])
	}
}

func TestWriteGitLogGraphIdempotent(t *testing.T) {
	repo := makeRepo(t)
	writeAndCommit(t, repo, "a.txt", "x", "first")
	writeAndCommit(t, repo, "b.txt", "y", "second")

	store := openGitTestStore(t)
	if _, err := WriteGitLogGraph(repo, "test-repo", 0, store); err != nil {
		t.Fatalf("first: %v", err)
	}
	before, _ := store.Stats()
	if _, err := WriteGitLogGraph(repo, "test-repo", 0, store); err != nil {
		t.Fatalf("second: %v", err)
	}
	after, _ := store.Stats()
	if before.HistoryRows.Nodes != after.HistoryRows.Nodes ||
		before.HistoryRows.Edges != after.HistoryRows.Edges {
		t.Errorf("history grew: nodes %d→%d edges %d→%d",
			before.HistoryRows.Nodes, after.HistoryRows.Nodes,
			before.HistoryRows.Edges, after.HistoryRows.Edges)
	}
}

func TestWriteGitLogGraphNonRepoIsNoop(t *testing.T) {
	dir := t.TempDir()
	store := openGitTestStore(t)
	summary, err := WriteGitLogGraph(dir, "not-a-repo", 0, store)
	if err != nil {
		t.Fatalf("expected no error on non-repo, got %v", err)
	}
	if summary.Commits != 0 {
		t.Errorf("expected 0 commits in non-repo, got %d", summary.Commits)
	}
}
