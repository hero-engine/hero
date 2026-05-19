package handoff

import (
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func TestRecordAsk_UpsertsAndSupersedes(t *testing.T) {
	store := openTestStore(t)

	if err := RecordAsk(store, "repo-x", UserAsk{
		User:      "alice",
		Text:      "where did we leave off on the auth bug?",
		SessionID: "sess-1",
	}); err != nil {
		t.Fatalf("RecordAsk: %v", err)
	}

	got, err := LatestAsk(store, "alice", "repo-x", "engineering")
	if err != nil {
		t.Fatalf("LatestAsk: %v", err)
	}
	if got == nil {
		t.Fatal("LatestAsk = nil, want a row")
	}
	if got.Text != "where did we leave off on the auth bug?" {
		t.Errorf("Text = %q", got.Text)
	}
	if got.SessionID != "sess-1" {
		t.Errorf("SessionID = %q", got.SessionID)
	}

	// Second ask supersedes the first — same key, new content, prior
	// row gets valid_to via UpsertNode bitemporal semantics.
	if err := RecordAsk(store, "repo-x", UserAsk{
		User:      "alice",
		Text:      "actually let's talk about the cart bug",
		SessionID: "sess-1",
	}); err != nil {
		t.Fatalf("RecordAsk #2: %v", err)
	}
	got, _ = LatestAsk(store, "alice", "repo-x", "engineering")
	if got == nil || got.Text != "actually let's talk about the cart bug" {
		t.Errorf("Latest after supersede = %+v", got)
	}
}

func TestRecordAsk_ClearWithEmptyText(t *testing.T) {
	store := openTestStore(t)
	_ = RecordAsk(store, "repo", UserAsk{User: "bob", Text: "hello"})

	if err := RecordAsk(store, "repo", UserAsk{User: "bob", Text: ""}); err != nil {
		t.Fatalf("RecordAsk(empty): %v", err)
	}
	got, _ := LatestAsk(store, "bob", "repo", "engineering")
	if got != nil {
		t.Errorf("after clear, LatestAsk = %+v, want nil", got)
	}
}

func TestRecordAsk_PerUserIsolation(t *testing.T) {
	store := openTestStore(t)
	_ = RecordAsk(store, "repo", UserAsk{User: "alice", Text: "alice prompt"})
	_ = RecordAsk(store, "repo", UserAsk{User: "bob", Text: "bob prompt"})

	a, _ := LatestAsk(store, "alice", "repo", "engineering")
	b, _ := LatestAsk(store, "bob", "repo", "engineering")
	if a == nil || a.Text != "alice prompt" {
		t.Errorf("alice = %+v", a)
	}
	if b == nil || b.Text != "bob prompt" {
		t.Errorf("bob = %+v", b)
	}
}

// TestRecordAsk_PerRepoIsolation pins the cross-repo bleed regression:
// an ask recorded in repo A must NOT surface in a LatestAsk read
// against repo B. UpsertNode's partition-key semantics still
// invalidate the prior repo's row when a new ask is recorded in a
// different repo (so the global singleton story is last-write-wins),
// but the repo-scoped read protects readers in repo B from ever
// seeing repo A's content while it's the current row.
func TestRecordAsk_PerRepoIsolation(t *testing.T) {
	store := openTestStore(t)
	if err := RecordAsk(store, "repo-a", UserAsk{User: "alice", Text: "A-context"}); err != nil {
		t.Fatalf("RecordAsk repo-a: %v", err)
	}

	gotB, err := LatestAsk(store, "alice", "repo-b", "engineering")
	if err != nil {
		t.Fatalf("LatestAsk repo-b: %v", err)
	}
	if gotB != nil {
		t.Errorf("LatestAsk(repo-b) = %+v, want nil — repo-a's ask leaked", gotB)
	}

	gotA, _ := LatestAsk(store, "alice", "repo-a", "engineering")
	if gotA == nil || gotA.Text != "A-context" {
		t.Errorf("LatestAsk(repo-a) = %+v, want A-context", gotA)
	}
}

func TestRecordSuggestion_RoundTrip(t *testing.T) {
	store := openTestStore(t)
	if err := RecordSuggestion(store, "repo", NextSuggestion{
		User:      "alice",
		Text:      "let's finish phase 4 of next-as-projection",
		Rationale: "phases 1-3 are merged; this unblocks the projection split",
		SessionID: "sess-2",
	}); err != nil {
		t.Fatalf("RecordSuggestion: %v", err)
	}
	got, _ := LatestSuggestion(store, "alice", "repo", "engineering")
	if got == nil {
		t.Fatal("LatestSuggestion = nil")
	}
	if got.Text == "" || got.Rationale == "" {
		t.Errorf("missing fields: %+v", got)
	}
}

// TestRecordSuggestion_PerRepoIsolation: a suggestion recorded in
// repo A is invisible to a repo-B read.
func TestRecordSuggestion_PerRepoIsolation(t *testing.T) {
	store := openTestStore(t)
	if err := RecordSuggestion(store, "repo-a", NextSuggestion{User: "alice", Text: "phase A"}); err != nil {
		t.Fatal(err)
	}
	gotB, _ := LatestSuggestion(store, "alice", "repo-b", "engineering")
	if gotB != nil {
		t.Errorf("LatestSuggestion(repo-b) = %+v, want nil — repo-a leaked", gotB)
	}
	gotA, _ := LatestSuggestion(store, "alice", "repo-a", "engineering")
	if gotA == nil || gotA.Text != "phase A" {
		t.Errorf("LatestSuggestion(repo-a) = %+v, want phase A", gotA)
	}
}

func TestRecordReflection_AccumulatesNewestFirst(t *testing.T) {
	store := openTestStore(t)

	if err := RecordReflection(store, "repo", SessionReflection{
		User: "alice", Text: "first lesson", Tags: []string{"perf"}, SessionID: "s",
	}); err != nil {
		t.Fatalf("RecordReflection #1: %v", err)
	}
	// Force ordering by sleeping enough that the timestamp suffix differs.
	time.Sleep(1100 * time.Millisecond)
	if err := RecordReflection(store, "repo", SessionReflection{
		User: "alice", Text: "second lesson", SessionID: "s",
	}); err != nil {
		t.Fatalf("RecordReflection #2: %v", err)
	}

	got, err := RecentReflections(store, "alice", "repo", "engineering", 5)
	if err != nil {
		t.Fatalf("RecentReflections: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Text != "second lesson" {
		t.Errorf("got[0] = %q, want newest first", got[0].Text)
	}
}

func TestRecordReflection_LimitTrims(t *testing.T) {
	store := openTestStore(t)
	for _, txt := range []string{"a", "b", "c"} {
		if err := RecordReflection(store, "repo", SessionReflection{
			User: "alice", Text: txt, SessionID: "s",
		}); err != nil {
			t.Fatalf("RecordReflection: %v", err)
		}
		time.Sleep(1100 * time.Millisecond)
	}
	got, _ := RecentReflections(store, "alice", "repo", "engineering", 2)
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// TestRecentReflections_PerRepoIsolation: reflections recorded in
// repo A do not surface in a repo-B query, and vice versa.
func TestRecentReflections_PerRepoIsolation(t *testing.T) {
	store := openTestStore(t)
	if err := RecordReflection(store, "repo-a", SessionReflection{
		User: "alice", Text: "A-lesson", SessionID: "s",
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if err := RecordReflection(store, "repo-b", SessionReflection{
		User: "alice", Text: "B-lesson", SessionID: "s",
	}); err != nil {
		t.Fatal(err)
	}

	gotA, _ := RecentReflections(store, "alice", "repo-a", "engineering", 10)
	if len(gotA) != 1 || gotA[0].Text != "A-lesson" {
		t.Errorf("repo-a reflections = %+v, want only [A-lesson]", gotA)
	}
	gotB, _ := RecentReflections(store, "alice", "repo-b", "engineering", 10)
	if len(gotB) != 1 || gotB[0].Text != "B-lesson" {
		t.Errorf("repo-b reflections = %+v, want only [B-lesson]", gotB)
	}
}

func TestLatestAsk_MissingReturnsNil(t *testing.T) {
	store := openTestStore(t)
	got, err := LatestAsk(store, "nobody", "repo", "engineering")
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("got = %+v, want nil", got)
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
