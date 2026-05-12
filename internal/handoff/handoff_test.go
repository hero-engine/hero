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

	got, err := LatestAsk(store, "alice")
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
	got, _ = LatestAsk(store, "alice")
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
	got, _ := LatestAsk(store, "bob")
	if got != nil {
		t.Errorf("after clear, LatestAsk = %+v, want nil", got)
	}
}

func TestRecordAsk_PerUserIsolation(t *testing.T) {
	store := openTestStore(t)
	_ = RecordAsk(store, "repo", UserAsk{User: "alice", Text: "alice prompt"})
	_ = RecordAsk(store, "repo", UserAsk{User: "bob", Text: "bob prompt"})

	a, _ := LatestAsk(store, "alice")
	b, _ := LatestAsk(store, "bob")
	if a == nil || a.Text != "alice prompt" {
		t.Errorf("alice = %+v", a)
	}
	if b == nil || b.Text != "bob prompt" {
		t.Errorf("bob = %+v", b)
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
	got, _ := LatestSuggestion(store, "alice")
	if got == nil {
		t.Fatal("LatestSuggestion = nil")
	}
	if got.Text == "" || got.Rationale == "" {
		t.Errorf("missing fields: %+v", got)
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

	got, err := RecentReflections(store, "alice", 5)
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
	got, _ := RecentReflections(store, "alice", 2)
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

func TestLatestAsk_MissingReturnsNil(t *testing.T) {
	store := openTestStore(t)
	got, err := LatestAsk(store, "nobody")
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
