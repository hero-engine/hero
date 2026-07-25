package memory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

func TestWriteGraph_MissingDir(t *testing.T) {
	store := openTestStore(t)
	summary, err := WriteGraph(filepath.Join(t.TempDir(), "does-not-exist"), "repo-x", store)
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if summary.Files != 0 {
		t.Fatalf("expected 0 files, got %d", summary.Files)
	}
}

func TestWriteGraph_EmptyMemoryDirString(t *testing.T) {
	store := openTestStore(t)
	summary, err := WriteGraph("", "repo-x", store)
	if err != nil {
		t.Fatalf("empty dir should not error: %v", err)
	}
	if summary.Files != 0 {
		t.Fatalf("expected 0 files, got %d", summary.Files)
	}
}

func TestWriteGraph_UpsertsMemoryNodes(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "feedback_avoid_x.md"), `---
name: Avoid X
description: Don't suggest X
type: feedback
---
Body of memory.`)
	mustWrite(t, filepath.Join(dir, "user_role.md"), `---
name: User role
description: User is a senior eng
type: user
---
Senior dev preferences.`)

	store := openTestStore(t)
	summary, err := WriteGraph(dir, "repo-x", store)
	if err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	if summary.Files != 2 {
		t.Fatalf("expected 2 files, got %d", summary.Files)
	}

	nodes, err := store.ListNodesByType("Memory")
	if err != nil {
		t.Fatalf("ListNodesByType: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 Memory nodes, got %d", len(nodes))
	}
	for _, n := range nodes {
		if n.Scope != graph.ScopeLocal {
			t.Errorf("Memory %q scope = %q, want local", n.Key, n.Scope)
		}
		if n.Repo != "repo-x" {
			t.Errorf("Memory %q repo = %q, want repo-x", n.Key, n.Repo)
		}
	}
}

func TestWriteGraph_Idempotent(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.md"), `---
name: A
type: feedback
---
content`)

	store := openTestStore(t)
	if _, err := WriteGraph(dir, "repo-x", store); err != nil {
		t.Fatalf("first WriteGraph: %v", err)
	}
	first, _ := store.GetNodeID("Memory", "a", "")

	if _, err := WriteGraph(dir, "repo-x", store); err != nil {
		t.Fatalf("second WriteGraph: %v", err)
	}
	second, _ := store.GetNodeID("Memory", "a", "")

	if first != second {
		t.Fatalf("expected idempotent re-ingest, got node id %d → %d", first, second)
	}
}

func TestDirForProject(t *testing.T) {
	got := DirForProject("/Users/foo/projects/bar")
	if got == "" {
		t.Fatal("empty path")
	}
	if filepath.Base(got) != "memory" {
		t.Errorf("expected memory dir suffix, got %q", got)
	}
	if filepath.Base(filepath.Dir(got)) != "-Users-foo-projects-bar" {
		t.Errorf("expected encoded project key, got %q", filepath.Dir(got))
	}
}

func openTestStore(t *testing.T) *graph.Store {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	store, err := graph.Open(dir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
