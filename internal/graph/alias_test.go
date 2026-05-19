package graph

import "testing"

func TestResolveAliasReturnsSelfWhenNoAlias(t *testing.T) {
	s := openTestStore(t)
	id, err := s.UpsertNode(&Node{Type: "Feature", Key: "x", Domain: "engineering", ContentHash: "h"})
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	got, err := s.ResolveAlias("Feature", "x")
	if err != nil {
		t.Fatalf("ResolveAlias: %v", err)
	}
	if got != id {
		t.Errorf("got %d, want %d", got, id)
	}
}

func TestMakeAliasAndResolve(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.UpsertNode(&Node{Type: "Feature", Key: "old-name", Domain: "engineering", ContentHash: "h1"}); err != nil {
		t.Fatal(err)
	}
	canonical, err := s.UpsertNode(&Node{Type: "Feature", Key: "new-name", Domain: "engineering", ContentHash: "h2"})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.MakeAlias("Feature", "old-name", "Feature", "new-name"); err != nil {
		t.Fatalf("MakeAlias: %v", err)
	}

	got, err := s.ResolveAlias("Feature", "old-name")
	if err != nil {
		t.Fatalf("ResolveAlias: %v", err)
	}
	if got != canonical {
		t.Errorf("alias did not resolve: got %d, want canonical %d", got, canonical)
	}
}

func TestResolveAliasFollowsChain(t *testing.T) {
	s := openTestStore(t)
	a, _ := s.UpsertNode(&Node{Type: "Feature", Key: "a", Domain: "engineering", ContentHash: "h"})
	_ = a
	_, _ = s.UpsertNode(&Node{Type: "Feature", Key: "b", Domain: "engineering", ContentHash: "h"})
	c, _ := s.UpsertNode(&Node{Type: "Feature", Key: "c", Domain: "engineering", ContentHash: "h"})

	if err := s.MakeAlias("Feature", "a", "Feature", "b"); err != nil {
		t.Fatal(err)
	}
	if err := s.MakeAlias("Feature", "b", "Feature", "c"); err != nil {
		t.Fatal(err)
	}

	got, err := s.ResolveAlias("Feature", "a")
	if err != nil {
		t.Fatalf("ResolveAlias: %v", err)
	}
	if got != c {
		t.Errorf("chain a→b→c didn't reach c: got %d, want %d", got, c)
	}
}

func TestResolveAliasHandlesCycleGracefully(t *testing.T) {
	s := openTestStore(t)
	a, _ := s.UpsertNode(&Node{Type: "Feature", Key: "a", Domain: "engineering", ContentHash: "h"})
	b, _ := s.UpsertNode(&Node{Type: "Feature", Key: "b", Domain: "engineering", ContentHash: "h"})
	_ = a
	_ = b
	if err := s.MakeAlias("Feature", "a", "Feature", "b"); err != nil {
		t.Fatal(err)
	}
	if err := s.MakeAlias("Feature", "b", "Feature", "a"); err != nil {
		t.Fatal(err)
	}
	// No infinite loop, no error — returns *some* id from the cycle.
	got, err := s.ResolveAlias("Feature", "a")
	if err != nil {
		t.Fatalf("ResolveAlias on cycle: %v", err)
	}
	if got != a && got != b {
		t.Errorf("cycle resolved to unexpected id %d", got)
	}
}
