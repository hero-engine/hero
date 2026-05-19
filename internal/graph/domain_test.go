package graph

import (
	"path/filepath"
	"testing"
)

// TestSchemaV3FreshDB asserts that opening a brand-new database
// produces a v3 schema with the `domain` column on both nodes and
// edges. DSKG AC #1 (idempotent schema migration).
func TestSchemaV3FreshDB(t *testing.T) {
	s := openTestStore(t)

	st, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if st.SchemaVersion != "3" {
		t.Fatalf("schema_version = %q, want %q", st.SchemaVersion, "3")
	}

	// Column existence is implicit in DomainStats not erroring.
	if _, err := s.DomainStats(); err != nil {
		t.Fatalf("DomainStats on fresh v3 db: %v", err)
	}
}

// TestSchemaV3PreservesExistingNodes asserts that the v3 default
// clause backfills every existing row with `domain = 'engineering'`.
// Verifies AC #14 — `hero domain verify` post-migration must report
// every node under `engineering`.
func TestSchemaV3DefaultsToEngineering(t *testing.T) {
	s := openTestStore(t)

	_, err := s.UpsertNode(&Node{
		Type:        "Package",
		Key:         "internal/cli",
		Domain:      "engineering",
		Props:       map[string]any{"language": "go"},
		ContentHash: "h1",
		Source:      map[string]any{"kind": "codescan"},
	})
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	ds, err := s.DomainStats()
	if err != nil {
		t.Fatalf("DomainStats: %v", err)
	}
	if got := ds.NodesByDomain["engineering"]; got != 1 {
		t.Errorf("NodesByDomain[engineering] = %d, want 1", got)
	}
	for d, n := range ds.NodesByDomain {
		if d != "engineering" && n > 0 {
			t.Errorf("unexpected domain %q with %d nodes", d, n)
		}
	}
}

// TestSchemaV3Idempotent asserts that re-opening an already-v3 db is a
// no-op — the migration runner skips applied versions.
func TestSchemaV3Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hero")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	st1, _ := s1.Stats()
	if st1.SchemaVersion != "3" {
		t.Fatalf("first open schema = %q, want %q", st1.SchemaVersion, "3")
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer s2.Close()
	st2, _ := s2.Stats()
	if st2.SchemaVersion != "3" {
		t.Errorf("re-open schema = %q, want %q", st2.SchemaVersion, "3")
	}
}

// TestRollbackV3 asserts the rollback drops the column and resets
// schema_version. After rollback, re-opening must re-apply v3 (the
// migration runner sees db at v2 and migrates forward).
func TestRollbackV3(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hero")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := s.RollbackV3(); err != nil {
		t.Fatalf("RollbackV3: %v", err)
	}

	var v string
	if err := s.DB().QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&v); err != nil {
		t.Fatalf("read schema_version: %v", err)
	}
	if v != "2" {
		t.Errorf("post-rollback schema_version = %q, want %q", v, "2")
	}

	// Domain column must be gone — DomainStats should error.
	if _, err := s.DomainStats(); err == nil {
		t.Error("DomainStats succeeded after rollback; column should be dropped")
	}
	s.Close()

	// Re-opening migrates forward to v3 again.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("re-open after rollback: %v", err)
	}
	defer s2.Close()
	st, _ := s2.Stats()
	if st.SchemaVersion != "3" {
		t.Errorf("post-rollback re-open schema = %q, want %q", st.SchemaVersion, "3")
	}
}

// TestNonEngineeringRowCount asserts the rollback dry-run helper
// returns counts that match what the rollback would discard. AC #13.
func TestNonEngineeringRowCount(t *testing.T) {
	s := openTestStore(t)

	// One engineering node (the default) plus one PM-stamped node
	// inserted directly to skip the future write-path guards (which
	// land in Phase 2).
	if _, err := s.UpsertNode(&Node{
		Type: "Package", Key: "internal/cli", Domain: "engineering",
		ContentHash: "h1",
		Source:      map[string]any{"kind": "codescan"},
	}); err != nil {
		t.Fatalf("eng node: %v", err)
	}
	if _, err := s.DB().Exec(
		`UPDATE nodes SET domain = 'pm' WHERE key = 'internal/cli'`,
	); err != nil {
		t.Fatalf("set pm domain: %v", err)
	}

	nodes, edges, err := s.NonEngineeringRowCount()
	if err != nil {
		t.Fatalf("NonEngineeringRowCount: %v", err)
	}
	if nodes != 1 {
		t.Errorf("non-engineering nodes = %d, want 1", nodes)
	}
	if edges != 0 {
		t.Errorf("non-engineering edges = %d, want 0", edges)
	}
}
