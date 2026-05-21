package projection

import (
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func TestCollectSessionEvents_SessionTaggedDecision(t *testing.T) {
	store := openTestStore(t)

	sessionStart := time.Now().UTC().Add(-time.Hour)
	// Decision tagged with our session_id, valid_from after start.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Decision", Key: "decision-1", Repo: "test-repo", Domain: "engineering", ContentHash: "h-d1",
		Props: map[string]any{
			"title":      "use bcrypt",
			"body":       "bcrypt over scrypt for ubiquity",
			"session_id": "sess-A",
		},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:    "sess-A",
		SessionStart: sessionStart,
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents: %v", err)
	}
	if len(got.Decisions) != 1 || got.Decisions[0].Title != "use bcrypt" {
		t.Fatalf("expected one decision titled use bcrypt; got %+v", got.Decisions)
	}
}

func TestCollectSessionEvents_OtherSessionExcluded(t *testing.T) {
	store := openTestStore(t)
	sessionStart := time.Now().UTC().Add(-time.Hour)

	// Decision tagged with a different session_id and no spec anchor — must be excluded.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Decision", Key: "decision-other", Repo: "test-repo", Domain: "engineering", ContentHash: "h-other",
		Props: map[string]any{
			"title":      "other thing",
			"session_id": "sess-OTHER",
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:    "sess-A",
		SessionStart: sessionStart,
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents: %v", err)
	}
	if len(got.Decisions) != 0 {
		t.Fatalf("expected no decisions; got %+v", got.Decisions)
	}
}

func TestCollectSessionEvents_SpecAnchoredCarryover(t *testing.T) {
	store := openTestStore(t)
	sessionStart := time.Now().UTC().Add(-time.Hour)

	// Decision anchored to active spec but from a different session —
	// the spec-anchored carryover clause must include it.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Decision", Key: "decision-spec", Repo: "test-repo", Domain: "engineering", ContentHash: "h-spec",
		Props: map[string]any{
			"title":      "schema migration plan",
			"session_id": "sess-OTHER",
			"spec_slug":  "auth-refactor",
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:      "sess-A",
		SessionStart:   sessionStart,
		ActiveSpecSlug: "auth-refactor",
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents: %v", err)
	}
	if len(got.Decisions) != 1 || got.Decisions[0].Title != "schema migration plan" {
		t.Fatalf("expected spec-anchored decision; got %+v", got.Decisions)
	}
}

func TestCollectSessionEvents_FilesTouchedFromAttempts(t *testing.T) {
	store := openTestStore(t)
	sessionStart := time.Now().UTC().Add(-time.Hour)

	if _, err := store.UpsertNode(&graph.Node{
		Type: "Attempt", Key: "att-1", Repo: "test-repo", Domain: "engineering", ContentHash: "h-att1",
		Props: map[string]any{
			"body":       "tried changing internal/active/active.go but failed",
			"session_id": "sess-A",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertNode(&graph.Node{
		Type: "SessionReflection", Key: "ref-1", Repo: "test-repo", Domain: "engineering", ContentHash: "h-ref1",
		Props: map[string]any{
			"text":       "internal/active/active.go has a sync.Mutex worth respecting",
			"session_id": "sess-A",
		},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:    "sess-A",
		SessionStart: sessionStart,
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents: %v", err)
	}
	if len(got.FilesTouched) == 0 {
		t.Fatalf("expected at least one file touched; got %+v", got.FilesTouched)
	}
	want := "internal/active/active.go"
	found := false
	for _, f := range got.FilesTouched {
		if f.Path == want && f.Count >= 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s with count >= 2; got %+v", want, got.FilesTouched)
	}
}

func TestCollectSessionEvents_EmptySession(t *testing.T) {
	store := openTestStore(t)
	got, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:    "no-such",
		SessionStart: time.Now().UTC().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil SessionEvents")
	}
	if got.TotalEvents != 0 || len(got.Decisions) != 0 {
		t.Errorf("expected empty session; got %+v", got)
	}
}
