package digest

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

func TestSprintSection_RendersWhenActiveSprintExists(t *testing.T) {
	store := openTestStore(t)
	// Active sprint
	sprintID, _ := store.UpsertNode(&graph.Node{
		Type: "Sprint", Key: "10042",
		Props: map[string]any{
			"name": "Sprint 42", "state": "active",
			"start": "2026-04-14", "end": "2026-04-28",
			"goal": "Ship the new auth flow",
		},
		ContentHash: "h-sprint",
	})
	// Two issues in it
	issueA, _ := store.UpsertNode(&graph.Node{
		Type: "Issue", Key: "PROJ-101",
		Props: map[string]any{
			"key": "PROJ-101", "title": "OAuth login endpoint",
			"status": "In Progress", "assignee": "alice@example.com",
		},
		ContentHash: "h-i1",
	})
	issueB, _ := store.UpsertNode(&graph.Node{
		Type: "Issue", Key: "PROJ-102",
		Props: map[string]any{
			"key": "PROJ-102", "title": "Frontend OAuth", "status": "Open",
		},
		ContentHash: "h-i2",
	})
	store.UpsertEdge(&graph.Edge{FromID: issueA, ToID: sprintID, Type: "belongs_to"})
	store.UpsertEdge(&graph.Edge{FromID: issueB, ToID: sprintID, Type: "belongs_to"})

	b, err := Generate(store, Options{
		RepoKey: "test", AuthorEmail: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	md := b.Markdown()

	for _, want := range []string{
		"## Active sprint", "Sprint 42", "Ship the new auth flow",
		"PROJ-101", "PROJ-102", "← yours",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in:\n%s", want, md)
		}
	}
}

func TestSprintSection_OmittedWhenNoActiveSprint(t *testing.T) {
	store := openTestStore(t)
	// Closed sprint — should NOT render.
	store.UpsertNode(&graph.Node{
		Type: "Sprint", Key: "10000",
		Props:       map[string]any{"name": "Sprint 0", "state": "closed"},
		ContentHash: "h",
	})

	b, err := Generate(store, Options{RepoKey: "test"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if strings.Contains(b.Markdown(), "## Active sprint") {
		t.Errorf("Active sprint should not render when no active sprint exists")
	}
}
