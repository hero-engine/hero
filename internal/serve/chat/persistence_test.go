package chat

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chat.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestStoreRoundTrip(t *testing.T) {
	s := openTestStore(t)
	convID, err := s.NewConversation("alice", "global")
	if err != nil {
		t.Fatalf("new conversation: %v", err)
	}
	if convID == "" {
		t.Fatal("expected non-empty conversation id")
	}
	if err := s.AppendMessage(convID, "user", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendMessage(convID, "assistant", "hi"); err != nil {
		t.Fatal(err)
	}
	turns, err := s.History(convID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}
	if turns[0].Role != "user" || turns[0].Content != "hello" {
		t.Errorf("turn[0] = %+v", turns[0])
	}
	if turns[1].Role != "assistant" || turns[1].Content != "hi" {
		t.Errorf("turn[1] = %+v", turns[1])
	}

	scopeTurns, err := s.HistoryByScope("alice", "global", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(scopeTurns) != 2 {
		t.Fatalf("expected 2 scope turns, got %d", len(scopeTurns))
	}
}

func TestStoreClear(t *testing.T) {
	s := openTestStore(t)
	id, err := s.NewConversation("alice", "global")
	if err != nil {
		t.Fatal(err)
	}
	_ = s.AppendMessage(id, "user", "x")
	if err := s.Clear(id); err != nil {
		t.Fatal(err)
	}
	turns, err := s.History(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 0 {
		t.Fatalf("expected 0 turns after clear, got %d", len(turns))
	}
}

func TestStoreConcurrentAppend(t *testing.T) {
	s := openTestStore(t)
	id, err := s.NewConversation("alice", "global")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	const n = 20
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = s.AppendMessage(id, "user", "x")
		}(i)
	}
	wg.Wait()
	turns, err := s.History(id, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != n {
		t.Fatalf("expected %d turns, got %d", n, len(turns))
	}
}

func TestStorePreference(t *testing.T) {
	s := openTestStore(t)
	if err := s.SetPreference("alice", "hero-code"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Preference("alice")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hero-code" {
		t.Errorf("got %q, want hero-code", got)
	}
	if err := s.SetPreference("alice", ""); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Preference("alice")
	if got != "" {
		t.Errorf("expected cleared preference, got %q", got)
	}
}

func TestStoreLatestConversation(t *testing.T) {
	s := openTestStore(t)
	id1, _ := s.NewConversation("alice", "global")
	time.Sleep(2 * time.Millisecond)
	id2, _ := s.NewConversation("alice", "global")
	time.Sleep(2 * time.Millisecond)
	_ = s.AppendMessage(id2, "user", "newer")
	latest, err := s.LatestConversation("alice", "global")
	if err != nil {
		t.Fatal(err)
	}
	if latest != id2 {
		t.Errorf("latest = %s, want %s", latest, id2)
	}
	_ = id1
}
