package refs

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	store, err := Open(heroDir, "testsession")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestStore_RegisterAndLookup_ShareableKind(t *testing.T) {
	s := newTestStore(t)
	args := map[string]any{"slug": "hero-ask"}
	id, err := s.Register(KindSpec, "hero-ask", "full", args, "spec body", "fp1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	want := "spec:hero-ask:full"
	if id != want {
		t.Fatalf("ref id = %q, want %q (shareable kinds use deterministic IDs)", id, want)
	}

	e, err := s.Lookup(id)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if e == nil {
		t.Fatalf("lookup returned nil for known ref")
	}
	if e.Kind != KindSpec || e.Slug != "hero-ask" || e.Scope != "full" {
		t.Fatalf("entry mismatch: %+v", e)
	}
	if e.Content != "spec body" {
		t.Fatalf("content = %q", e.Content)
	}
	if e.SourceFingerprint != "fp1" {
		t.Fatalf("fingerprint = %q", e.SourceFingerprint)
	}
}

func TestStore_QueryKind_SessionScoped(t *testing.T) {
	s := newTestStore(t)
	id1, err := s.Register(KindSearch, "abc", "results", map[string]any{"q": "auth"}, "results blob", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if id1 == "search:abc:results" {
		t.Fatalf("query kinds must not use deterministic IDs; got %q", id1)
	}

	// Same slug+scope from a different session should produce a different ID.
	other, err := Open(filepath.Join(t.TempDir(), ".hero"), "othersession")
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	id2, _ := other.Register(KindSearch, "abc", "results", map[string]any{"q": "auth"}, "x", "")
	if id1 == id2 {
		t.Fatalf("session-scoped IDs collided across sessions: %q", id1)
	}
}

func TestStore_LookupMiss(t *testing.T) {
	s := newTestStore(t)
	e, err := s.Lookup("spec:nope:full")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if e != nil {
		t.Fatalf("expected nil for unknown ref, got %+v", e)
	}
}

func TestStore_UpdateContent(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Register(KindSpec, "hero-ask", "full", nil, "v1", "fp1")
	if err := s.UpdateContent(id, "v2", "fp2"); err != nil {
		t.Fatalf("update: %v", err)
	}
	e, _ := s.Lookup(id)
	if e.Content != "v2" || e.SourceFingerprint != "fp2" {
		t.Fatalf("update did not stick: %+v", e)
	}
}

func TestStore_UpdateUnknownIsError(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpdateContent("spec:nope:full", "x", "y"); err == nil {
		t.Fatalf("expected error updating unknown ref")
	}
}

func TestStore_Prune(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Register(KindSpec, "hero-ask", "full", nil, "body", "fp")

	// Force expiry by rewriting expires_at into the past.
	if _, err := s.db.Exec(`UPDATE refs SET expires_at = ? WHERE ref_id = ?`, time.Now().Add(-time.Hour).Unix(), id); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	n, err := s.Prune()
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if n != 1 {
		t.Fatalf("pruned %d, want 1", n)
	}

	e, _ := s.Lookup(id)
	if e != nil {
		t.Fatalf("entry survived prune: %+v", e)
	}
}

func TestStore_Metrics(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Register(KindSpec, "hero-ask", "full", nil, "body", "fp")

	if _, err := s.Lookup(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Lookup("spec:nope:full"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateContent(id, "v2", "fp2"); err != nil {
		t.Fatal(err)
	}

	m := s.Metrics()
	if m.Registers != 1 || m.Hits != 1 || m.Misses != 1 || m.Refetch != 1 {
		t.Fatalf("metrics = %+v, want all 1s", m)
	}
}

func TestStore_PersistMetrics(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Register(KindSpec, "hero-ask", "full", nil, "body", "fp")
	_, _ = s.Lookup(id)

	if err := s.PersistMetrics(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	// In-memory counters should reset.
	if m := s.Metrics(); m.Registers != 0 || m.Hits != 0 {
		t.Fatalf("counters not reset after persist: %+v", m)
	}
	persisted, err := s.PersistedMetrics()
	if err != nil {
		t.Fatalf("persisted: %v", err)
	}
	if persisted.Registers != 1 || persisted.Hits != 1 {
		t.Fatalf("persisted metrics = %+v", persisted)
	}

	// Persisting again accumulates.
	_, _ = s.Lookup(id)
	if err := s.PersistMetrics(); err != nil {
		t.Fatal(err)
	}
	persisted, _ = s.PersistedMetrics()
	if persisted.Hits != 2 {
		t.Fatalf("hits did not accumulate: %+v", persisted)
	}
}

func TestParseRefID(t *testing.T) {
	kind, slug, scope, err := ParseRefID("spec:hero-ask:full")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if kind != KindSpec || slug != "hero-ask" || scope != "full" {
		t.Fatalf("got kind=%q slug=%q scope=%q", kind, slug, scope)
	}

	if _, _, _, err := ParseRefID("not-a-ref"); err == nil {
		t.Fatalf("expected error on malformed ref id")
	}
}

func TestKind_IsShareable(t *testing.T) {
	cases := map[Kind]bool{
		KindSpec:       true,
		KindConvention: true,
		KindDecision:   true,
		KindRule:       true,
		KindSearch:     false,
		KindContext:    false,
		KindRecap:      false,
		KindFeed:       false,
	}
	for k, want := range cases {
		if k.IsShareable() != want {
			t.Errorf("Kind(%q).IsShareable() = %v, want %v", k, k.IsShareable(), want)
		}
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	called := false
	reg.Register(KindSpec, func(slug, scope string, args map[string]any) (string, string, error) {
		called = true
		return "fresh", "fp-fresh", nil
	})

	e := &Entry{RefID: "spec:foo:full", Kind: KindSpec, Slug: "foo", Scope: "full"}
	content, fp, err := reg.Resolve(e)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !called {
		t.Fatalf("resolver not invoked")
	}
	if content != "fresh" || fp != "fp-fresh" {
		t.Fatalf("got content=%q fp=%q", content, fp)
	}
}

func TestRegistry_NoResolver(t *testing.T) {
	reg := NewRegistry()
	e := &Entry{RefID: "search:abc:results", Kind: KindSearch}
	if _, _, err := reg.Resolve(e); err == nil {
		t.Fatalf("expected error when no resolver registered for kind")
	}
}

func TestSessionID_Stable(t *testing.T) {
	a := SessionID("/foo", 1)
	b := SessionID("/foo", 1)
	if a != b {
		t.Fatalf("session id should be stable: %q vs %q", a, b)
	}
	c := SessionID("/foo", 2)
	if a == c {
		t.Fatalf("session id should change with pid")
	}
}
