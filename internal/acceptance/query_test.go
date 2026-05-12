package acceptance

import (
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

func TestStatusByFeature_RollsUpCounts(t *testing.T) {
	store := openTestStore(t)

	// Spec A: 2 passing, 1 failing.
	seedCriterionForParent(t, store, "feat-a:AC-1", "x", "passing", "feat-a")
	seedCriterionForParent(t, store, "feat-a:AC-2", "y", "passing", "feat-a")
	seedCriterionForParent(t, store, "feat-a:AC-3", "z", "failing", "feat-a")
	// Spec B: 1 proposed, 1 regressed.
	seedCriterionForParent(t, store, "feat-b:AC-1", "p", "proposed", "feat-b")
	seedCriterionForParent(t, store, "feat-b:AC-2", "q", "regressed", "feat-b")

	rows, err := StatusByFeature(store, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	a := findStatusRow(rows, "feat-a")
	if a == nil {
		t.Fatal("feat-a row missing")
	}
	if a.Total != 3 || a.Passing != 2 || a.Failing != 1 {
		t.Errorf("feat-a counts = %+v", a)
	}
	if rate := a.PassRate(); rate < 0.66 || rate > 0.67 {
		t.Errorf("feat-a pass-rate = %f, want ~0.667", rate)
	}
	b := findStatusRow(rows, "feat-b")
	if b == nil || b.Proposed != 1 || b.Regressed != 1 {
		t.Errorf("feat-b counts = %+v", b)
	}
}

func TestStatusByFeature_FilterNarrowsToOneSpec(t *testing.T) {
	store := openTestStore(t)
	seedCriterionForParent(t, store, "feat-a:AC-1", "x", "passing", "feat-a")
	seedCriterionForParent(t, store, "feat-b:AC-1", "y", "passing", "feat-b")

	rows, err := StatusByFeature(store, "feat-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Parent != "feat-a" {
		t.Errorf("filtered rows = %+v, want only feat-a", rows)
	}
}

func TestHistory_ReturnsAllRowsOldestFirst(t *testing.T) {
	store := openTestStore(t)

	seedCriterionAt(t, store, "feat-a:AC-1", "x", "proposed", "2026-04-28T10:00:00Z")
	// Same key, status flip — bitemporal: invalidate prior + insert new.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Criterion",
		Key:  "feat-a:AC-1",
		Props: map[string]any{
			"ac_id":     "AC-1",
			"statement": "x",
			"status":    "passing",
			"parent":    "feat-a",
		},
		Repo:        "repo-x",
		ContentHash: hashCriterionStatus("feat-a:AC-1", "x", "passing"),
		Source:      map[string]any{"kind": "test"},
		ValidFrom:   "2026-04-28T11:00:00Z",
	}); err != nil {
		t.Fatalf("flip: %v", err)
	}

	entries, err := History(store, "feat-a:AC-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Status != "proposed" || entries[1].Status != "passing" {
		t.Errorf("order wrong: %+v", entries)
	}
	if entries[1].ValidTo != "" {
		t.Errorf("current row should have empty ValidTo, got %q", entries[1].ValidTo)
	}
	if entries[0].ValidTo == "" {
		t.Errorf("prior row should have non-empty ValidTo")
	}
}

func TestHistory_EmptyForUnknownKey(t *testing.T) {
	store := openTestStore(t)
	entries, err := History(store, "nope:AC-99")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, want empty", entries)
	}
}

func findStatusRow(rows []SpecStatus, parent string) *SpecStatus {
	for i := range rows {
		if rows[i].Parent == parent {
			return &rows[i]
		}
	}
	return nil
}

// seedCriterionForParent is a query-test variant of seedCriterion
// that takes the parent slug explicitly, so rollup tests can vary
// it across multiple specs (the shared seedCriterion hard-codes
// "feat-x").
func seedCriterionForParent(t *testing.T, store *graph.Store, key, statement, status, parent string) {
	t.Helper()
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Criterion",
		Key:  key,
		Props: map[string]any{
			"ac_id":     key,
			"statement": statement,
			"status":    status,
			"parent":    parent,
		},
		Repo:        "repo-x",
		ContentHash: hashCriterionStatus(key, statement, status),
	}); err != nil {
		t.Fatalf("seed criterion: %v", err)
	}
}
