package graph

import (
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "hero"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenCreatesSchema(t *testing.T) {
	s := openTestStore(t)
	st, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if st.SchemaVersion != schemaVersion {
		t.Errorf("schema_version = %q, want %q", st.SchemaVersion, schemaVersion)
	}
	if st.InstallID == "" {
		t.Error("install_id should be seeded")
	}
	if st.TotalNodes != 0 || st.TotalEdges != 0 {
		t.Errorf("expected empty graph, got nodes=%d edges=%d", st.TotalNodes, st.TotalEdges)
	}
}

func TestUpsertNodeInsertAndIdempotency(t *testing.T) {
	s := openTestStore(t)
	n := &Node{
		Type:        "Package",
		Key:         "internal/cli",
		Props:       map[string]any{"language": "go", "files": 85},
		Scope:       ScopeTeam,
		Domain:      "engineering",
		ContentHash: "hash-1",
		Source:      map[string]any{"kind": "codescan"},
	}
	id1, err := s.UpsertNode(n)
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	// Same content_hash → no-op, same id back, history depth = 1
	id2, err := s.UpsertNode(&Node{
		Type: "Package", Key: "internal/cli", Domain: "engineering",
		Props:       map[string]any{"language": "go", "files": 85},
		ContentHash: "hash-1",
		Source:      map[string]any{"kind": "codescan"},
	})
	if err != nil {
		t.Fatalf("UpsertNode (idempotent): %v", err)
	}
	if id1 != id2 {
		t.Errorf("idempotent upsert returned new id: %d → %d", id1, id2)
	}
	st, _ := s.Stats()
	if st.HistoryRows.Nodes != 1 {
		t.Errorf("history depth = %d, want 1", st.HistoryRows.Nodes)
	}
}

func TestUpsertNodeInvalidatesAndAppends(t *testing.T) {
	s := openTestStore(t)
	first, _ := s.UpsertNode(&Node{
		Type: "Package", Key: "internal/cli", Domain: "engineering",
		Props: map[string]any{"files": 85}, ContentHash: "h1",
	})
	second, err := s.UpsertNode(&Node{
		Type: "Package", Key: "internal/cli", Domain: "engineering",
		Props: map[string]any{"files": 90}, ContentHash: "h2",
	})
	if err != nil {
		t.Fatalf("UpsertNode (update): %v", err)
	}
	if first == second {
		t.Errorf("update should produce a new row id")
	}
	// Current row is the second one.
	got, err := s.GetNode("Package", "internal/cli", "")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.ID != second {
		t.Errorf("current id = %d, want %d", got.ID, second)
	}
	if got.ContentHash != "h2" {
		t.Errorf("current content_hash = %q, want h2", got.ContentHash)
	}
	// History depth is 2.
	st, _ := s.Stats()
	if st.HistoryRows.Nodes != 2 {
		t.Errorf("history depth = %d, want 2", st.HistoryRows.Nodes)
	}
	if st.TotalNodes != 1 {
		t.Errorf("current nodes = %d, want 1", st.TotalNodes)
	}
}

func TestUpsertEdgeAndQueries(t *testing.T) {
	s := openTestStore(t)
	cliID, _ := s.UpsertNode(&Node{Type: "Package", Key: "internal/cli", Domain: "engineering", ContentHash: "h"})
	cfgID, _ := s.UpsertNode(&Node{Type: "Package", Key: "internal/config", Domain: "engineering", ContentHash: "h"})
	idxID, _ := s.UpsertNode(&Node{Type: "Package", Key: "internal/index", Domain: "engineering", ContentHash: "h"})

	if _, err := s.UpsertEdge(&Edge{FromID: cliID, ToID: cfgID, Type: "imports"}); err != nil {
		t.Fatalf("UpsertEdge cli→config: %v", err)
	}
	if _, err := s.UpsertEdge(&Edge{FromID: cliID, ToID: idxID, Type: "imports"}); err != nil {
		t.Fatalf("UpsertEdge cli→index: %v", err)
	}
	// Idempotent
	if _, err := s.UpsertEdge(&Edge{FromID: cliID, ToID: cfgID, Type: "imports"}); err != nil {
		t.Fatalf("UpsertEdge (idempotent): %v", err)
	}

	out, err := s.EdgesFrom(cliID, "imports")
	if err != nil {
		t.Fatalf("EdgesFrom: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("got %d edges, want 2", len(out))
	}

	in, err := s.EdgesTo(cfgID, "")
	if err != nil {
		t.Fatalf("EdgesTo: %v", err)
	}
	if len(in) != 1 || in[0].FromID != cliID {
		t.Errorf("EdgesTo unexpected: %+v", in)
	}
}

func TestNodeUpdateInvalidatesEdges(t *testing.T) {
	s := openTestStore(t)
	a, _ := s.UpsertNode(&Node{Type: "Package", Key: "a", Domain: "engineering", ContentHash: "h1"})
	b, _ := s.UpsertNode(&Node{Type: "Package", Key: "b", Domain: "engineering", ContentHash: "h"})
	if _, err := s.UpsertEdge(&Edge{FromID: a, ToID: b, Type: "imports"}); err != nil {
		t.Fatalf("UpsertEdge: %v", err)
	}
	// Update node a → invalidates the prior a's edges
	if _, err := s.UpsertNode(&Node{Type: "Package", Key: "a", Domain: "engineering", ContentHash: "h2"}); err != nil {
		t.Fatalf("UpsertNode (update): %v", err)
	}
	// Current edges from old a id should now be 0
	out, _ := s.EdgesFrom(a, "imports")
	if len(out) != 0 {
		t.Errorf("expected stale edges invalidated, got %d", len(out))
	}
}

func TestGetNodeNotFound(t *testing.T) {
	s := openTestStore(t)
	_, err := s.GetNode("Package", "nonexistent", "")
	if err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestInvalidateNode(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.UpsertNode(&Node{Type: "Feature", Key: "x", Domain: "engineering", ContentHash: "h"}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	if err := s.InvalidateNode("Feature", "x", ""); err != nil {
		t.Fatalf("InvalidateNode: %v", err)
	}
	if _, err := s.GetNode("Feature", "x", ""); err != ErrNotFound {
		t.Errorf("after invalidate, got %v want ErrNotFound", err)
	}
	if err := s.InvalidateNode("Feature", "x", ""); err != ErrNotFound {
		t.Errorf("invalidate twice should ErrNotFound, got %v", err)
	}
}

func TestListNodesByType(t *testing.T) {
	s := openTestStore(t)
	s.UpsertNode(&Node{Type: "Package", Key: "a", Domain: "engineering", ContentHash: "h"})
	s.UpsertNode(&Node{Type: "Package", Key: "b", Domain: "engineering", ContentHash: "h"})
	s.UpsertNode(&Node{Type: "Feature", Key: "f", Domain: "engineering", ContentHash: "h"})

	pkgs, err := s.ListNodesByType("Package")
	if err != nil {
		t.Fatalf("ListNodesByType: %v", err)
	}
	if len(pkgs) != 2 {
		t.Errorf("got %d packages, want 2", len(pkgs))
	}
	all, _ := s.ListNodesByType("")
	if len(all) != 3 {
		t.Errorf("got %d total, want 3", len(all))
	}
}

// TestPartialMigrationRecovery simulates a v3 migration that partially
// applied (domain column on nodes but not edges) and verifies that
// re-opening the database recovers instead of permanently bricking.
func TestPartialMigrationRecovery(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, "hero")

	// Create a fresh v3 database, then simulate a partial v3 state:
	// roll back to v2, manually add the domain column to nodes only
	// (simulating a crash mid-migration).
	s, err := Open(heroDir)
	if err != nil {
		t.Fatalf("initial Open: %v", err)
	}
	if err := s.RollbackV3(); err != nil {
		t.Fatalf("RollbackV3: %v", err)
	}
	// Add domain column to nodes only — edges still missing it.
	if _, err := s.DB().Exec(`ALTER TABLE nodes ADD COLUMN domain TEXT NOT NULL DEFAULT 'engineering'`); err != nil {
		t.Fatalf("partial add column: %v", err)
	}
	s.Close()

	// Re-open: the migration runner sees schema_version=2, retries v3.
	// The ALTER for nodes.domain will hit "duplicate column name" — this
	// must be tolerated so the edges.domain ALTER can proceed.
	s2, err := Open(heroDir)
	if err != nil {
		t.Fatalf("re-open after partial migration should recover: %v", err)
	}
	defer s2.Close()

	st, err := s2.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if st.SchemaVersion != schemaVersion {
		t.Errorf("schema_version = %q, want %q", st.SchemaVersion, schemaVersion)
	}
}
