package graph

import (
	"testing"
	"time"
)

// TestFindGraphConflicts_MultipleClientsDetected verifies that two nodes
// with the same (type, key) but different client_ids are surfaced as
// a divergence — SC-2 of graph-conflict-detection.
func TestFindGraphConflicts_MultipleClientsDetected(t *testing.T) {
	s := openTestStore(t)

	now := time.Now().UTC()

	// First push: client "alice" sets status=planning.
	_, err := s.db.Exec(`
		INSERT INTO nodes (type, key, props, client_id, valid_from, valid_to, scope, domain, source, ingested_at)
		VALUES ('Feature', 'my-feature', '{"status":"planning"}', 'alice', ?, NULL, 'team', 'engineering', '{}', ?)`,
		now.Add(-10*time.Minute), now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("insert alice row: %v", err)
	}

	// Second push: client "bob" closes alice's row and sets status=delivering.
	_, err = s.db.Exec(`
		UPDATE nodes SET valid_to = ? WHERE type='Feature' AND key='my-feature' AND client_id='alice'`,
		now.Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("close alice row: %v", err)
	}
	_, err = s.db.Exec(`
		INSERT INTO nodes (type, key, props, client_id, valid_from, valid_to, scope, domain, source, ingested_at)
		VALUES ('Feature', 'my-feature', '{"status":"delivering"}', 'bob', ?, NULL, 'team', 'engineering', '{}', ?)`,
		now.Add(-5*time.Minute), now.Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("insert bob row: %v", err)
	}

	results, err := s.FindGraphConflicts("my-feature")
	if err != nil {
		t.Fatalf("FindGraphConflicts: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected a conflict, got none")
	}
	c := results[0]
	if c.NodeType != "Feature" || c.NodeKey != "my-feature" {
		t.Errorf("unexpected result: type=%s key=%s", c.NodeType, c.NodeKey)
	}
	if len(c.Versions) < 2 {
		t.Errorf("expected ≥2 versions, got %d", len(c.Versions))
	}
}

// TestFindGraphConflicts_SingleClientNoConflict verifies SC-4:
// same-client pushes are NOT flagged as conflicts.
func TestFindGraphConflicts_SingleClientNoConflict(t *testing.T) {
	s := openTestStore(t)

	now := time.Now().UTC()

	// Two rows, same client_id "alice" — represents a re-push/update.
	_, err := s.db.Exec(`
		INSERT INTO nodes (type, key, props, client_id, valid_from, valid_to, scope, domain, source, ingested_at)
		VALUES ('Feature', 'solo-feature', '{"status":"planning"}', 'alice', ?, ?, 'team', 'engineering', '{}', ?)`,
		now.Add(-10*time.Minute), now.Add(-5*time.Minute), now.Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("insert row 1: %v", err)
	}
	_, err = s.db.Exec(`
		INSERT INTO nodes (type, key, props, client_id, valid_from, valid_to, scope, domain, source, ingested_at)
		VALUES ('Feature', 'solo-feature', '{"status":"delivering"}', 'alice', ?, NULL, 'team', 'engineering', '{}', ?)`,
		now.Add(-5*time.Minute), now.Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("insert row 2: %v", err)
	}

	results, err := s.FindGraphConflicts("solo-feature")
	if err != nil {
		t.Fatalf("FindGraphConflicts: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no conflicts for single-client, got %d", len(results))
	}
}
