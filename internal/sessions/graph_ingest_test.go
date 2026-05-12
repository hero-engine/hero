package sessions

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func openSessionTestStore(t *testing.T) *graph.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := graph.Open(filepath.Join(dir, "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSessionWriteGraphInsertsNodes(t *testing.T) {
	store := openSessionTestStore(t)
	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)

	summary, err := WriteGraph([]*Session{
		{ID: "abc", Name: "phase-2", Agent: "claude", Start: start, End: &end, HeroCalls: 12, SpecsDone: 1},
		{ID: "def", Agent: "cursor", Start: start},
	}, "test-repo", store)
	if err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	if summary.Sessions != 2 {
		t.Errorf("Sessions = %d, want 2", summary.Sessions)
	}
	stats, _ := store.Stats()
	if stats.NodesByType["Session"] != 2 {
		t.Errorf("Session nodes = %d, want 2", stats.NodesByType["Session"])
	}
}

func TestSessionWriteGraphLinksToSpecWhenPresent(t *testing.T) {
	store := openSessionTestStore(t)
	// Pre-create a Feature node so the session edge has a target.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Feature", Key: "graph-memory", ContentHash: "h",
	}); err != nil {
		t.Fatalf("seed Feature: %v", err)
	}
	summary, err := WriteGraph([]*Session{
		{ID: "abc", Start: time.Now(), SpecSlug: "graph-memory"},
		{ID: "xyz", Start: time.Now(), SpecSlug: "no-such-spec"},
	}, "test-repo", store)
	if err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	// One edge for the resolved spec, none for the missing one.
	if summary.Edges != 1 {
		t.Errorf("Edges = %d, want 1", summary.Edges)
	}
	stats, _ := store.Stats()
	if stats.EdgesByType["mentions"] != 1 {
		t.Errorf("mentions edges = %d, want 1", stats.EdgesByType["mentions"])
	}
}

func TestSessionWriteGraphIdempotent(t *testing.T) {
	store := openSessionTestStore(t)
	in := []*Session{{ID: "abc", Start: time.Now().UTC(), HeroCalls: 1}}
	if _, err := WriteGraph(in, "test-repo", store); err != nil {
		t.Fatalf("first: %v", err)
	}
	before, _ := store.Stats()
	if _, err := WriteGraph(in, "test-repo", store); err != nil {
		t.Fatalf("second: %v", err)
	}
	after, _ := store.Stats()
	if before.HistoryRows.Nodes != after.HistoryRows.Nodes {
		t.Errorf("history grew: %d → %d", before.HistoryRows.Nodes, after.HistoryRows.Nodes)
	}
}
