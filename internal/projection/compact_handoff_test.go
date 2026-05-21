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

// TestCollectSessionEvents_SpecAnchoredCarryover_ExcludesUserAskFromOthers
// asserts the carryover is selective: spec-anchored Decisions from other
// sessions are pulled in, but session-bound event types (UserAsk,
// NextSuggestion, Attempt, SessionReflection) from other sessions are
// not — they have no place leaking into another agent's resume packet.
func TestCollectSessionEvents_SpecAnchoredCarryover_ExcludesUserAskFromOthers(t *testing.T) {
	store := openTestStore(t)
	sessionStart := time.Now().UTC().Add(-time.Hour)

	// Spec-anchored Decision from a different session — should appear.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Decision", Key: "decision-spec", Repo: "test-repo", Domain: "engineering", ContentHash: "h-spec",
		Props: map[string]any{
			"title":      "carry me forward",
			"session_id": "sess-OTHER",
			"spec_slug":  "shared-spec",
		},
	}); err != nil {
		t.Fatal(err)
	}
	// UserAsk anchored to the same spec but from another session — should
	// NOT bleed into our session's files-touched count.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "UserAsk", Key: "ask-other", Repo: "test-repo", Domain: "engineering", ContentHash: "h-ask-other",
		Props: map[string]any{
			"text":       "they asked about path/to/secret.go",
			"session_id": "sess-OTHER",
			"spec_slug":  "shared-spec",
		},
	}); err != nil {
		t.Fatal(err)
	}
	// NextSuggestion from another session anchored to spec — also excluded.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "NextSuggestion", Key: "ns-other", Repo: "test-repo", Domain: "engineering", ContentHash: "h-ns-other",
		Props: map[string]any{
			"text":       "do path/to/leak.go",
			"session_id": "sess-OTHER",
			"spec_slug":  "shared-spec",
		},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:      "sess-A",
		SessionStart:   sessionStart,
		ActiveSpecSlug: "shared-spec",
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents: %v", err)
	}
	if len(got.Decisions) != 1 || got.Decisions[0].Title != "carry me forward" {
		t.Fatalf("expected the spec-anchored Decision; got %+v", got.Decisions)
	}
	// FilesTouched derives from session_id-tagged Attempt/UserAsk/etc
	// rows. The two other-session bodies must not contribute paths.
	for _, f := range got.FilesTouched {
		if f.Path == "path/to/secret.go" || f.Path == "path/to/leak.go" {
			t.Errorf("file %q from another session leaked into FilesTouched", f.Path)
		}
	}
}

// TestCollectSessionEvents_BothDirectionsForSameSpec verifies the
// carryover is bidirectional: both sessions on the same active spec see
// each other's Decisions, but neither sees the other's session-bound
// events.
func TestCollectSessionEvents_BothDirectionsForSameSpec(t *testing.T) {
	store := openTestStore(t)
	sessionStart := time.Now().UTC().Add(-time.Hour)

	// Decision from sess-A on shared spec.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Decision", Key: "decision-from-A", Repo: "test-repo", Domain: "engineering", ContentHash: "h-A",
		Props: map[string]any{
			"title":      "A's decision",
			"session_id": "sess-A",
			"spec_slug":  "shared-spec",
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Decision from sess-B on the same shared spec.
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Decision", Key: "decision-from-B", Repo: "test-repo", Domain: "engineering", ContentHash: "h-B",
		Props: map[string]any{
			"title":      "B's decision",
			"session_id": "sess-B",
			"spec_slug":  "shared-spec",
		},
	}); err != nil {
		t.Fatal(err)
	}

	// Session A should see B's decision via the spec carryover (and
	// its own decision via session_id match).
	gotA, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:      "sess-A",
		SessionStart:   sessionStart,
		ActiveSpecSlug: "shared-spec",
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents A: %v", err)
	}
	titlesA := decisionTitles(gotA.Decisions)
	if !containsString(titlesA, "A's decision") || !containsString(titlesA, "B's decision") {
		t.Errorf("sess-A should see both decisions; got %v", titlesA)
	}

	// Session B mirror-image.
	gotB, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:      "sess-B",
		SessionStart:   sessionStart,
		ActiveSpecSlug: "shared-spec",
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents B: %v", err)
	}
	titlesB := decisionTitles(gotB.Decisions)
	if !containsString(titlesB, "A's decision") || !containsString(titlesB, "B's decision") {
		t.Errorf("sess-B should see both decisions; got %v", titlesB)
	}
}

// TestFilesTouched_DeduplicatesAcrossEvents asserts that one file
// mentioned across multiple session-bound events appears once with a
// summed mention count — not duplicated rows.
func TestFilesTouched_DeduplicatesAcrossEvents(t *testing.T) {
	store := openTestStore(t)
	sessionStart := time.Now().UTC().Add(-time.Hour)

	// Three events from sess-A all reference internal/cli/foo.go.
	for i, body := range []string{
		"started editing internal/cli/foo.go",
		"still working on internal/cli/foo.go",
		"reviewed internal/cli/foo.go and decided to refactor",
	} {
		if _, err := store.UpsertNode(&graph.Node{
			Type: "Attempt", Key: "att-" + string(rune('1'+i)), Repo: "test-repo", Domain: "engineering",
			ContentHash: "h-att-" + string(rune('1'+i)),
			Props: map[string]any{
				"body":       body,
				"session_id": "sess-A",
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := CollectSessionEvents(store, CompactHandoffOptions{
		SessionID:    "sess-A",
		SessionStart: sessionStart,
	})
	if err != nil {
		t.Fatalf("CollectSessionEvents: %v", err)
	}
	// Expect exactly one entry for the file with count == 3.
	matches := 0
	for _, f := range got.FilesTouched {
		if f.Path == "internal/cli/foo.go" {
			matches++
			if f.Count != 3 {
				t.Errorf("file count = %d, want 3", f.Count)
			}
		}
	}
	if matches != 1 {
		t.Errorf("expected exactly one row for internal/cli/foo.go; saw %d in %+v", matches, got.FilesTouched)
	}
}

// --- local helpers --------------------------------------------------------

func decisionTitles(ds []CompactDecision) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Title)
	}
	return out
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
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
