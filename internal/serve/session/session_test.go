package session

import (
	"net/http"
	"path/filepath"
	"sync"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shell-sessions.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestLastHomeRoundTrip(t *testing.T) {
	s := newStore(t)

	if _, ok := s.LastHome("alice"); ok {
		t.Fatalf("expected no prior record")
	}
	if err := s.SetLastHome("alice", "work"); err != nil {
		t.Fatalf("SetLastHome: %v", err)
	}
	got, ok := s.LastHome("alice")
	if !ok || got != "work" {
		t.Fatalf("LastHome = (%q, %v), want (work, true)", got, ok)
	}

	// Overwrite.
	if err := s.SetLastHome("alice", "agents"); err != nil {
		t.Fatalf("SetLastHome overwrite: %v", err)
	}
	got, ok = s.LastHome("alice")
	if !ok || got != "agents" {
		t.Fatalf("after overwrite LastHome = (%q, %v), want (agents, true)", got, ok)
	}

	// Other user isolated.
	if _, ok := s.LastHome("bob"); ok {
		t.Fatalf("bob should have no record")
	}
}

func TestTabStateRoundTrip(t *testing.T) {
	s := newStore(t)

	if _, ok := s.TabState("alice", "agents"); ok {
		t.Fatalf("expected no prior tab state")
	}
	if err := s.SetTabState("alice", "agents", "proposals"); err != nil {
		t.Fatalf("SetTabState: %v", err)
	}
	got, ok := s.TabState("alice", "agents")
	if !ok || got != "proposals" {
		t.Fatalf("TabState(agents) = (%q, %v), want (proposals, true)", got, ok)
	}

	// Independent per home.
	if err := s.SetTabState("alice", "people", "roi"); err != nil {
		t.Fatalf("SetTabState people: %v", err)
	}
	if got, ok := s.TabState("alice", "people"); !ok || got != "roi" {
		t.Fatalf("TabState(people) = (%q, %v), want (roi, true)", got, ok)
	}
	if got, ok := s.TabState("alice", "agents"); !ok || got != "proposals" {
		t.Fatalf("agents state should survive setting people; got (%q, %v)", got, ok)
	}

	// Empty tab clears the slot.
	if err := s.SetTabState("alice", "agents", ""); err != nil {
		t.Fatalf("SetTabState clear: %v", err)
	}
	if _, ok := s.TabState("alice", "agents"); ok {
		t.Fatalf("expected agents tab cleared")
	}
}

func TestLastHomeAndTabStateCoexist(t *testing.T) {
	s := newStore(t)
	if err := s.SetLastHome("alice", "work"); err != nil {
		t.Fatalf("SetLastHome: %v", err)
	}
	if err := s.SetTabState("alice", "agents", "scheduled"); err != nil {
		t.Fatalf("SetTabState: %v", err)
	}
	if got, ok := s.LastHome("alice"); !ok || got != "work" {
		t.Fatalf("LastHome lost after SetTabState: got (%q, %v)", got, ok)
	}
	if got, ok := s.TabState("alice", "agents"); !ok || got != "scheduled" {
		t.Fatalf("TabState: got (%q, %v)", got, ok)
	}
}

func TestConcurrentWritesSerialize(t *testing.T) {
	s := newStore(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_ = s.SetLastHome("alice", "work")
		}(i)
		go func(n int) {
			defer wg.Done()
			_ = s.SetTabState("alice", "agents", "proposals")
		}(i)
	}
	wg.Wait()

	if got, ok := s.LastHome("alice"); !ok || got != "work" {
		t.Fatalf("after concurrent writes LastHome = (%q, %v)", got, ok)
	}
	if got, ok := s.TabState("alice", "agents"); !ok || got != "proposals" {
		t.Fatalf("after concurrent writes TabState = (%q, %v)", got, ok)
	}
}

func TestUserIDPrecedence(t *testing.T) {
	t.Setenv("USER", "os-user")
	t.Setenv("USERNAME", "")

	// No request → falls back to OS user.
	if got := UserID(nil); got != "os-user" {
		t.Fatalf("UserID(nil) = %q, want os-user", got)
	}

	// Cookie wins.
	r := newReq()
	r.AddCookie(&http.Cookie{Name: "hero_user", Value: "alice"})
	if got := UserID(r); got != "alice" {
		t.Fatalf("UserID with cookie = %q, want alice", got)
	}

	// Header used when no cookie.
	r2 := newReq()
	r2.Header.Set("X-Hero-User", "bob")
	if got := UserID(r2); got != "bob" {
		t.Fatalf("UserID with header = %q, want bob", got)
	}

	// Bare env fallback.
	r3 := newReq()
	if got := UserID(r3); got != "os-user" {
		t.Fatalf("UserID with bare req = %q, want os-user", got)
	}
}

func TestNilStoreSafety(t *testing.T) {
	var s *Store
	if _, ok := s.LastHome("alice"); ok {
		t.Fatalf("nil store should not return ok")
	}
	if err := s.SetLastHome("alice", "now"); err != nil {
		t.Fatalf("nil store SetLastHome should be no-op, got %v", err)
	}
	if _, ok := s.TabState("alice", "agents"); ok {
		t.Fatalf("nil store TabState should not return ok")
	}
	if err := s.SetTabState("alice", "agents", "x"); err != nil {
		t.Fatalf("nil store SetTabState should be no-op, got %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("nil store Close should not error, got %v", err)
	}
}

func newReq() *http.Request {
	r, _ := http.NewRequest("GET", "/", nil)
	return r
}
