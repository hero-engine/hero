package tracker

import (
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

func openTestStore(t *testing.T) *graph.Store {
	t.Helper()
	s, err := graph.Open(filepath.Join(t.TempDir(), "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func sampleSprintItems() ([]SprintItem, *SprintInfo) {
	info := &SprintInfo{
		ID:    "10042",
		Name:  "Sprint 42",
		Goal:  "Ship the new auth flow",
		Start: "2026-04-14",
		End:   "2026-04-28",
		State: "active",
	}
	items := []SprintItem{
		{
			ID:          "PROJ-101",
			Title:       "OAuth login endpoint",
			Type:        "story",
			Status:      "In Progress",
			Priority:    "high",
			Assignee:    "alice@example.com",
			SprintName:  "Sprint 42",
			StoryPoints: 5,
			Labels:      []string{"auth", "backend"},
			LinkedIDs: []LinkedItem{
				{ID: "PROJ-102", LinkType: "blocks"},
			},
		},
		{
			ID:         "PROJ-102",
			Title:      "Frontend OAuth integration",
			Type:       "story",
			Status:     "Open",
			Assignee:   "bob@example.com",
			SprintName: "Sprint 42",
			EpicID:     "PROJ-50",
			EpicTitle:  "Auth Modernization",
		},
		{
			ID:         "PROJ-103",
			Title:      "Refresh-token bug",
			Type:       "bug",
			Status:     "Done",
			Priority:   "highest",
			SprintName: "Sprint 42",
		},
	}
	return items, info
}

func TestWriteSprintGraph_HappyPath(t *testing.T) {
	store := openTestStore(t)
	items, info := sampleSprintItems()

	summary, err := WriteSprintGraph(items, info, "test-repo", store)
	if err != nil {
		t.Fatalf("WriteSprintGraph: %v", err)
	}
	if summary.Sprints != 1 {
		t.Errorf("Sprints = %d, want 1", summary.Sprints)
	}
	// 3 in-batch + 1 epic stub
	if summary.Issues != 4 {
		t.Errorf("Issues = %d, want 4 (3 + 1 epic stub)", summary.Issues)
	}
	if summary.Persons != 2 {
		t.Errorf("Persons = %d, want 2", summary.Persons)
	}

	stats, _ := store.Stats()
	if stats.NodesByType["Sprint"] != 1 {
		t.Errorf("Sprint nodes = %d, want 1", stats.NodesByType["Sprint"])
	}
	if stats.NodesByType["Issue"] != 4 {
		t.Errorf("Issue nodes = %d, want 4", stats.NodesByType["Issue"])
	}
	if stats.NodesByType["Person"] != 2 {
		t.Errorf("Person nodes = %d, want 2", stats.NodesByType["Person"])
	}

	// Edges: 3 issue→sprint + 1 subtask→epic = 4 belongs_to
	if stats.EdgesByType["belongs_to"] != 4 {
		t.Errorf("belongs_to edges = %d, want 4", stats.EdgesByType["belongs_to"])
	}
	// 2 assignees
	if stats.EdgesByType["assigned_to"] != 2 {
		t.Errorf("assigned_to = %d, want 2", stats.EdgesByType["assigned_to"])
	}
	// PROJ-101 blocks PROJ-102
	if stats.EdgesByType["blocks"] != 1 {
		t.Errorf("blocks = %d, want 1", stats.EdgesByType["blocks"])
	}
}

func TestWriteSprintGraph_Idempotent(t *testing.T) {
	store := openTestStore(t)
	items, info := sampleSprintItems()
	if _, err := WriteSprintGraph(items, info, "test-repo", store); err != nil {
		t.Fatalf("first: %v", err)
	}
	before, _ := store.Stats()
	if _, err := WriteSprintGraph(items, info, "test-repo", store); err != nil {
		t.Fatalf("second: %v", err)
	}
	after, _ := store.Stats()
	if before.HistoryRows.Nodes != after.HistoryRows.Nodes ||
		before.HistoryRows.Edges != after.HistoryRows.Edges {
		t.Errorf("history grew: nodes %d→%d, edges %d→%d",
			before.HistoryRows.Nodes, after.HistoryRows.Nodes,
			before.HistoryRows.Edges, after.HistoryRows.Edges)
	}
}

func TestWriteSprintGraph_MergesWithCommitStubIssue(t *testing.T) {
	store := openTestStore(t)
	// Commit-ref parser creates a thin stub first.
	if _, err := store.UpsertNode(&graph.Node{
		Type:        "Issue",
		Domain:      "engineering",
		Key:         "PROJ-101",
		Props:       map[string]any{"key": "PROJ-101"},
		ContentHash: "stub",
	}); err != nil {
		t.Fatal(err)
	}

	items, info := sampleSprintItems()
	if _, err := WriteSprintGraph(items, info, "test-repo", store); err != nil {
		t.Fatalf("WriteSprintGraph: %v", err)
	}
	stats, _ := store.Stats()
	// PROJ-101 should be ONE current row even though it was upserted twice
	// (stub then enriched).
	if stats.NodesByType["Issue"] != 4 {
		t.Errorf("Issue current = %d, want 4 (3 in batch + 1 epic stub)", stats.NodesByType["Issue"])
	}
	// Get the current PROJ-101 — should now have title from the sprint ingest.
	n, err := store.GetNode("Issue", "PROJ-101", "")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if n.Props["title"] != "OAuth login endpoint" {
		t.Errorf("expected enriched title, got %v", n.Props["title"])
	}
}

func TestWriteSprintGraph_GlobalPersonIdentity(t *testing.T) {
	store := openTestStore(t)
	// Pre-create a person from git-log path (lowercase email).
	if _, err := store.UpsertNode(&graph.Node{
		Type:        "Person",
		Key:         "alice@example.com",
		Props:       map[string]any{"email": "alice@example.com", "name": "Alice"},
		ContentHash: "git-stub",
	}); err != nil {
		t.Fatal(err)
	}
	items, info := sampleSprintItems()
	if _, err := WriteSprintGraph(items, info, "test-repo", store); err != nil {
		t.Fatalf("WriteSprintGraph: %v", err)
	}
	stats, _ := store.Stats()
	if stats.NodesByType["Person"] != 2 {
		t.Errorf("Person nodes = %d, want 2 (alice merged, bob new)", stats.NodesByType["Person"])
	}
}

func TestWriteIssuesGraph_UpsertsIssuesAndPersons(t *testing.T) {
	store := openTestStore(t)
	issues := []Issue{
		{ID: "PROJ-201", Title: "Bug A", Status: "Open", Assignee: "alice@example.com", IssueType: "Bug"},
		{ID: "PROJ-202", Title: "Story B", Status: "In Progress", Priority: "high", Labels: []string{"frontend"}},
		{ID: "", Title: "missing id — should be skipped"},
	}
	summary, err := WriteIssuesGraph(issues, "repo-x", store)
	if err != nil {
		t.Fatalf("WriteIssuesGraph: %v", err)
	}
	if summary.Issues != 2 {
		t.Errorf("Issues = %d, want 2", summary.Issues)
	}
	if summary.Persons != 1 {
		t.Errorf("Persons = %d, want 1", summary.Persons)
	}
	stats, _ := store.Stats()
	if stats.NodesByType["Issue"] != 2 {
		t.Errorf("Issue nodes = %d, want 2", stats.NodesByType["Issue"])
	}
	if stats.NodesByType["Person"] != 1 {
		t.Errorf("Person nodes = %d, want 1", stats.NodesByType["Person"])
	}
}

func TestWriteIssuesGraph_Idempotent(t *testing.T) {
	store := openTestStore(t)
	issues := []Issue{{ID: "PROJ-300", Title: "Stable", Status: "Open"}}
	if _, err := WriteIssuesGraph(issues, "repo-x", store); err != nil {
		t.Fatalf("first: %v", err)
	}
	first, _ := store.GetNodeID("Issue", "PROJ-300", "")
	if _, err := WriteIssuesGraph(issues, "repo-x", store); err != nil {
		t.Fatalf("second: %v", err)
	}
	second, _ := store.GetNodeID("Issue", "PROJ-300", "")
	if first != second {
		t.Errorf("expected idempotent upsert, got node id %d → %d", first, second)
	}
}
