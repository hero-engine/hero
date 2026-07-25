package tasks

import (
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func TestWrite_UpsertsTaskNodes(t *testing.T) {
	store := openTestStore(t)
	parentID := seedFeature(t, store, "checkout-flow")

	parsed := []Task{
		{ID: "T-1", Text: "Fix redirect", Status: StatusTodo, Kind: "qa-blocker"},
		{ID: "T-2", Text: "Migrate token", Status: StatusDone, Done: "2026-05-15T14:22:00Z"},
	}
	summary, err := Write("Feature", "checkout-flow", parentID, parsed, "repo-x", store)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if summary.Tasks != 2 {
		t.Errorf("Tasks upserted = %d, want 2", summary.Tasks)
	}
	if summary.BelongsTo != 2 {
		t.Errorf("BelongsTo = %d, want 2", summary.BelongsTo)
	}

	got, err := store.GetNode("Task", "checkout-flow:T-1", "")
	if err != nil || got == nil {
		t.Fatalf("Task node not found: %v", err)
	}
	if s, _ := got.Props["status"].(string); s != StatusTodo {
		t.Errorf("status = %q, want todo", s)
	}
	if k, _ := got.Props["kind"].(string); k != "qa-blocker" {
		t.Errorf("kind = %q, want qa-blocker", k)
	}
}

func TestWrite_DiscoveredAgainstWiresEdge(t *testing.T) {
	store := openTestStore(t)
	parentID := seedFeature(t, store, "qa-suite")
	siblingID := seedFeature(t, store, "checkout-flow")

	parsed := []Task{
		{ID: "T-1", Text: "Fix it", Status: StatusTodo, DiscoveredAgainst: "checkout-flow"},
	}
	summary, err := Write("Feature", "qa-suite", parentID, parsed, "repo-x", store)
	if err != nil {
		t.Fatal(err)
	}
	if summary.DiscoveredAgainst != 1 {
		t.Errorf("DiscoveredAgainst edges = %d, want 1", summary.DiscoveredAgainst)
	}

	taskID, err := store.GetNodeID("Task", "qa-suite:T-1", "")
	if err != nil {
		t.Fatal(err)
	}
	edges, err := store.EdgesFrom(taskID, "discovered_against")
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 || edges[0].ToID != siblingID {
		t.Errorf("discovered_against edge wrong: %+v", edges)
	}
}

func TestWrite_DiscoveredAgainstUnknownTargetSkipped(t *testing.T) {
	store := openTestStore(t)
	parentID := seedFeature(t, store, "qa-suite")

	parsed := []Task{
		{ID: "T-1", Text: "Found in nowhere", Status: StatusTodo, DiscoveredAgainst: "does-not-exist"},
	}
	summary, err := Write("Feature", "qa-suite", parentID, parsed, "repo-x", store)
	if err != nil {
		t.Fatal(err)
	}
	if summary.DiscoveredAgainst != 0 {
		t.Errorf("DiscoveredAgainst edges = %d, want 0", summary.DiscoveredAgainst)
	}
	// Task node still landed.
	if _, err := store.GetNodeID("Task", "qa-suite:T-1", ""); err != nil {
		t.Errorf("Task node missing: %v", err)
	}
}

func TestWrite_AssigneeUpsertsPerson(t *testing.T) {
	store := openTestStore(t)
	parentID := seedFeature(t, store, "feat-a")

	parsed := []Task{
		{ID: "T-1", Text: "x", Status: StatusTodo, Assignee: "bwheeler"},
	}
	summary, err := Write("Feature", "feat-a", parentID, parsed, "repo-x", store)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AssignedTo != 1 {
		t.Errorf("AssignedTo = %d, want 1", summary.AssignedTo)
	}
	if _, err := store.GetNodeID("Person", "bwheeler", ""); err != nil {
		t.Errorf("Person node missing: %v", err)
	}
}

func TestWrite_StatusFlipBitemporal(t *testing.T) {
	store := openTestStore(t)
	parentID := seedFeature(t, store, "feat-a")

	// Seed at past time so the flip creates a meaningful interval.
	past := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Task",
		Domain:      "engineering",
		Key:  "feat-a:T-1",
		Props: map[string]any{
			"task_id": "T-1", "text": "x", "status": StatusTodo, "parent": "feat-a",
		},
		Repo:        "repo-x",
		ContentHash: hashTask("feat-a:T-1", "x", StatusTodo, "", "", "", "", ""),
		ValidFrom:   past,
	}); err != nil {
		t.Fatal(err)
	}

	// Flip to doing via Write.
	parsed := []Task{{ID: "T-1", Text: "x", Status: StatusDoing}}
	summary, err := Write("Feature", "feat-a", parentID, parsed, "repo-x", store)
	if err != nil {
		t.Fatal(err)
	}
	if summary.StatusFlips != 1 {
		t.Errorf("StatusFlips = %d, want 1", summary.StatusFlips)
	}

	// History should reflect both rows.
	entries, err := History(store, "feat-a:T-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Status != StatusTodo || entries[1].Status != StatusDoing {
		t.Errorf("history order wrong: %+v", entries)
	}
	if entries[1].ValidTo != "" {
		t.Errorf("current row ValidTo should be empty: %+v", entries[1])
	}
}

func TestWrite_NoOpWhenUnchanged(t *testing.T) {
	store := openTestStore(t)
	parentID := seedFeature(t, store, "feat-a")

	parsed := []Task{{ID: "T-1", Text: "x", Status: StatusTodo}}
	if _, err := Write("Feature", "feat-a", parentID, parsed, "repo-x", store); err != nil {
		t.Fatal(err)
	}
	summary, err := Write("Feature", "feat-a", parentID, parsed, "repo-x", store)
	if err != nil {
		t.Fatal(err)
	}
	if summary.StatusFlips != 0 {
		t.Errorf("re-run StatusFlips = %d, want 0", summary.StatusFlips)
	}
	if summary.NoOps != 1 {
		t.Errorf("re-run NoOps = %d, want 1", summary.NoOps)
	}
}

func TestListBySpec_FiltersAndSorts(t *testing.T) {
	store := openTestStore(t)
	parentA := seedFeature(t, store, "feat-a")
	parentB := seedFeature(t, store, "feat-b")

	if _, err := Write("Feature", "feat-a", parentA, []Task{
		{ID: "T-2", Text: "second", Status: StatusTodo},
		{ID: "T-10", Text: "tenth", Status: StatusTodo},
		{ID: "T-1", Text: "first", Status: StatusTodo},
	}, "repo-x", store); err != nil {
		t.Fatal(err)
	}
	if _, err := Write("Feature", "feat-b", parentB, []Task{
		{ID: "T-1", Text: "other", Status: StatusTodo},
	}, "repo-x", store); err != nil {
		t.Fatal(err)
	}

	a, err := ListBySpec(store, "feat-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 3 {
		t.Fatalf("feat-a count = %d, want 3", len(a))
	}
	want := []string{"T-1", "T-2", "T-10"}
	for i, r := range a {
		if r.TaskID != want[i] {
			t.Errorf("order at %d: %q, want %q (full: %+v)", i, r.TaskID, want[i], a)
		}
	}
}

func TestStatusByFeature_RollsUpCounts(t *testing.T) {
	store := openTestStore(t)
	parentA := seedFeature(t, store, "feat-a")
	parentB := seedFeature(t, store, "feat-b")

	if _, err := Write("Feature", "feat-a", parentA, []Task{
		{ID: "T-1", Text: "x", Status: StatusDone},
		{ID: "T-2", Text: "y", Status: StatusDone},
		{ID: "T-3", Text: "z", Status: StatusTodo},
	}, "repo-x", store); err != nil {
		t.Fatal(err)
	}
	if _, err := Write("Feature", "feat-b", parentB, []Task{
		{ID: "T-1", Text: "p", Status: StatusDoing},
	}, "repo-x", store); err != nil {
		t.Fatal(err)
	}

	rows, err := StatusByFeature(store, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	var a *SpecStatus
	for i := range rows {
		if rows[i].Parent == "feat-a" {
			a = &rows[i]
		}
	}
	if a == nil || a.Total != 3 || a.Done != 2 || a.Todo != 1 {
		t.Errorf("feat-a rollup wrong: %+v", a)
	}
	if rate := a.CompletionRate(); rate < 0.66 || rate > 0.67 {
		t.Errorf("feat-a completion rate = %f, want ~0.667", rate)
	}
}

func TestOpenAcrossCorpus_OnlyNonDone(t *testing.T) {
	store := openTestStore(t)
	parentA := seedFeature(t, store, "feat-a")
	if _, err := Write("Feature", "feat-a", parentA, []Task{
		{ID: "T-1", Text: "todo", Status: StatusTodo},
		{ID: "T-2", Text: "doing", Status: StatusDoing},
		{ID: "T-3", Text: "done", Status: StatusDone},
	}, "repo-x", store); err != nil {
		t.Fatal(err)
	}
	open, err := OpenAcrossCorpus(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 2 {
		t.Errorf("open = %d, want 2 (done filtered)", len(open))
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

func seedFeature(t *testing.T, store *graph.Store, slug string) int64 {
	t.Helper()
	id, err := store.UpsertNode(&graph.Node{
		Type:        "Feature",
		Domain:      "engineering",
		Key:         slug,
		Props:       map[string]any{"title": slug, "status": "delivering"},
		Repo:        "repo-x",
		ContentHash: "feat-" + slug,
		Source:      map[string]any{"kind": "test"},
	})
	if err != nil {
		t.Fatalf("seed Feature %s: %v", slug, err)
	}
	return id
}
