package projection

import (
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
)

func TestPickUserSuggestion_AgentWinsWhenFresh(t *testing.T) {
	store := openTestStore(t)

	if err := handoff.RecordSuggestion(store, "repo-x", handoff.NextSuggestion{
		User: "alice", Text: "ship phase 8",
	}); err != nil {
		t.Fatal(err)
	}

	text, _, source := PickUserSuggestion(store, "alice", "repo-x")
	if text != "ship phase 8" {
		t.Errorf("text = %q, want agent text", text)
	}
	if source != SuggestionFromAgent {
		t.Errorf("source = %q, want agent", source)
	}
}

func TestPickUserSuggestion_StaleWhenCommitLandsAfter(t *testing.T) {
	store := openTestStore(t)

	// Agent suggests something now.
	if err := handoff.RecordSuggestion(store, "repo-x", handoff.NextSuggestion{
		User: "alice", Text: "ship phase 8",
	}); err != nil {
		t.Fatal(err)
	}

	// A commit lands a moment later — supersedes the agent's prediction.
	commitTime := time.Now().UTC().Add(1 * time.Second).Format(time.RFC3339)
	seedCommit(t, store, "repo-x", "abc123", "feat: phase 8 — emit-as-you-work skill", commitTime)
	// Need an open Feature for the fallback to have something to point at.
	seedFeature(t, store, "repo-x", "next-thing", "Next Thing", "P1", "planning")

	text, _, source := PickUserSuggestion(store, "alice", "repo-x")
	if source == SuggestionFromAgent {
		t.Errorf("source = agent, want auto-derived after commit landed")
	}
	if text == "" {
		t.Error("derived text empty")
	}
	if text == "ship phase 8" {
		t.Errorf("returned stale agent text instead of derived: %q", text)
	}
}

func TestPickUserSuggestion_DerivesFromOpenFeatureWhenNoAgentSuggestion(t *testing.T) {
	store := openTestStore(t)
	seedFeature(t, store, "repo-x", "auth-bug", "Fix the auth bug", "P0", "planning")

	text, _, source := PickUserSuggestion(store, "alice", "repo-x")
	if source != SuggestionFromOpenFeature {
		t.Errorf("source = %q, want %q", source, SuggestionFromOpenFeature)
	}
	if text != "let's tackle Fix the auth bug" {
		t.Errorf("text = %q", text)
	}
}

func TestPickUserSuggestion_FallsBackToInitiativeWhenNoFeatures(t *testing.T) {
	store := openTestStore(t)
	seedInitiative(t, store, "repo-x", "rebuild", "Rebuild Everything", "planning")

	text, _, source := PickUserSuggestion(store, "alice", "repo-x")
	if source != SuggestionFromInitiative {
		t.Errorf("source = %q", source)
	}
	if text != "pick the next phase of Rebuild Everything" {
		t.Errorf("text = %q", text)
	}
}

func TestPickUserSuggestion_EmptyWhenNothingToSuggest(t *testing.T) {
	store := openTestStore(t)
	text, _, source := PickUserSuggestion(store, "alice", "repo-x")
	if text != "" || source != "" {
		t.Errorf("expected empty, got text=%q source=%q", text, source)
	}
}

func TestStalenessNote_FlagsWhenCommitLandsAfter(t *testing.T) {
	store := openTestStore(t)

	// Ask set 2 hours ago.
	twoHoursAgo := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	// Commit landed 30 minutes ago.
	thirtyMinAgo := time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
	seedCommit(t, store, "repo-x", "abc", "feat: something", thirtyMinAgo)

	note := stalenessNote(store, "repo-x", twoHoursAgo)
	if note == "" {
		t.Fatal("expected staleness note, got empty")
	}
	if !contains(note, "stale") || !contains(note, "1 commit") {
		t.Errorf("note missing expected substrings: %q", note)
	}
}

func TestStalenessNote_QuietWhenFresh(t *testing.T) {
	store := openTestStore(t)

	// Commit landed 1 hour ago, ask set 30 minutes ago — ask is fresh.
	oneHourAgo := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	thirtyMinAgo := time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
	seedCommit(t, store, "repo-x", "abc", "feat: prior work", oneHourAgo)

	note := stalenessNote(store, "repo-x", thirtyMinAgo)
	if note != "" {
		t.Errorf("expected empty note for fresh ask, got %q", note)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- helpers (openTestStore lives in projection_test.go) ------------------

func seedCommit(t *testing.T, store *graph.Store, repoKey, sha, subject, date string) {
	t.Helper()
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Commit",
		Key:  sha,
		Repo: repoKey,
		Props: map[string]any{
			"sha":     sha,
			"subject": subject,
			"date":    date,
		},
		Source: map[string]any{"kind": "test"},
	}); err != nil {
		t.Fatalf("seed commit: %v", err)
	}
}

func seedFeature(t *testing.T, store *graph.Store, repoKey, slug, title, priority, status string) {
	t.Helper()
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Feature",
		Key:  slug,
		Repo: repoKey,
		Props: map[string]any{
			"title":    title,
			"priority": priority,
			"status":   status,
		},
		Source: map[string]any{"kind": "test"},
	}); err != nil {
		t.Fatalf("seed feature: %v", err)
	}
}

func seedInitiative(t *testing.T, store *graph.Store, repoKey, slug, title, status string) {
	t.Helper()
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Initiative",
		Key:  slug,
		Repo: repoKey,
		Props: map[string]any{
			"title":  title,
			"status": status,
		},
		Source: map[string]any{"kind": "test"},
	}); err != nil {
		t.Fatalf("seed initiative: %v", err)
	}
}
