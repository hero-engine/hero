package acceptance

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func TestRecord_FlipsProposedToPassing(t *testing.T) {
	store := openTestStore(t)
	seedCriterion(t, store, "feat-x:AC-1", "First criterion", "proposed")

	results := []RunResult{
		{AC: "feat-x:AC-1", Status: "pass", Timestamp: nowRFC()},
	}
	summary, err := Record(results, "repo-x", store)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if summary.Criteria != 1 {
		t.Errorf("Criteria flips = %d, want 1", summary.Criteria)
	}
	got := getStatus(t, store, "feat-x:AC-1")
	if got != "passing" {
		t.Errorf("status = %q, want passing", got)
	}
}

func TestRecord_NoOpWhenStatusUnchanged(t *testing.T) {
	store := openTestStore(t)
	seedCriterion(t, store, "feat-x:AC-1", "First criterion", "passing")

	summary, err := Record([]RunResult{
		{AC: "feat-x:AC-1", Status: "pass"},
	}, "repo-x", store)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if summary.Criteria != 0 || summary.NoOps != 1 {
		t.Errorf("summary = %+v, want NoOps=1 Criteria=0", summary)
	}
}

func TestRecord_PassingThenFailingPromotesToRegressed(t *testing.T) {
	store := openTestStore(t)
	seedCriterion(t, store, "feat-x:AC-1", "First criterion", "passing")

	_, err := Record([]RunResult{
		{AC: "feat-x:AC-1", Status: "fail"},
	}, "repo-x", store)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if got := getStatus(t, store, "feat-x:AC-1"); got != "regressed" {
		t.Errorf("status = %q, want regressed", got)
	}
}

func TestRecord_BitemporalHistoryQuery(t *testing.T) {
	store := openTestStore(t)
	// Seed in the past so the eventual flip (at wall-clock now)
	// leaves a meaningful interval [past, now) where the seed row
	// was current.
	past := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	seedCriterionAt(t, store, "feat-x:AC-1", "First criterion", "proposed", past)

	if _, err := Record([]RunResult{
		{AC: "feat-x:AC-1", Status: "pass"},
	}, "repo-x", store); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// At a moment between seed and flip, the seed row is current.
	betweenSeedAndFlip := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	past1, err := store.GetNodeAt("Criterion", "feat-x:AC-1", betweenSeedAndFlip, "")
	if err != nil {
		t.Fatalf("GetNodeAt(between): %v", err)
	}
	if s, _ := past1.Props["status"].(string); s != "proposed" {
		t.Errorf("at %s: status = %q, want proposed", betweenSeedAndFlip, s)
	}

	// Now: the flipped row is current.
	current, err := store.GetNode("Criterion", "feat-x:AC-1", "")
	if err != nil {
		t.Fatalf("GetNode current: %v", err)
	}
	if s, _ := current.Props["status"].(string); s != "passing" {
		t.Errorf("now: status = %q, want passing", s)
	}
}

func TestRecord_SatisfiedByEdgeOnPass(t *testing.T) {
	store := openTestStore(t)
	seedCriterion(t, store, "feat-x:AC-1", "Criterion text", "proposed")

	summary, err := Record([]RunResult{
		{AC: "feat-x:AC-1", Status: "pass", SHA: "abc123"},
	}, "repo-x", store)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if summary.SatisfiedBy != 1 {
		t.Errorf("SatisfiedBy = %d, want 1", summary.SatisfiedBy)
	}
}

func TestRecord_BreaksEdgeOnRegression(t *testing.T) {
	store := openTestStore(t)
	seedCriterion(t, store, "feat-x:AC-1", "Criterion text", "passing")

	summary, err := Record([]RunResult{
		{AC: "feat-x:AC-1", Status: "fail", SHA: "deadbeef"},
	}, "repo-x", store)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if summary.Breaks != 1 {
		t.Errorf("Breaks = %d, want 1", summary.Breaks)
	}
}

func TestRecord_UnknownACSilentSkip(t *testing.T) {
	store := openTestStore(t)
	summary, err := Record([]RunResult{
		{AC: "no-such-spec:AC-1", Status: "pass"},
	}, "repo-x", store)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if summary.Unknown != 1 || summary.Criteria != 0 {
		t.Errorf("summary = %+v, want Unknown=1", summary)
	}
}

func TestLoadRunResults_MissingFileError(t *testing.T) {
	if _, err := LoadRunResults(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func openTestStore(t *testing.T) *graph.Store {
	t.Helper()
	store, err := graph.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func seedCriterion(t *testing.T, store *graph.Store, key, statement, status string) {
	t.Helper()
	seedCriterionAt(t, store, key, statement, status, "")
}

func seedCriterionAt(t *testing.T, store *graph.Store, key, statement, status, validFrom string) {
	t.Helper()
	n := &graph.Node{
		Type:   "Criterion",
		Domain: "engineering",
		Key:    key,
		Props: map[string]any{
			"ac_id":     key,
			"statement": statement,
			"status":    status,
			"parent":    "feat-x",
		},
		Repo:        "repo-x",
		ContentHash: hashCriterionStatus(key, statement, status),
	}
	if validFrom != "" {
		n.ValidFrom = validFrom
		n.IngestedAt = validFrom
	}
	if _, err := store.UpsertNode(n); err != nil {
		t.Fatalf("seed criterion: %v", err)
	}
}

func getStatus(t *testing.T, store *graph.Store, key string) string {
	t.Helper()
	n, err := store.GetNode("Criterion", key, "")
	if err != nil {
		t.Fatalf("GetNode(%s): %v", key, err)
	}
	s, _ := n.Props["status"].(string)
	return s
}

func nowRFC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
