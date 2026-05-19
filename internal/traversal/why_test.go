package traversal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func TestWhy_TwoHopChain(t *testing.T) {
	store := openStore(t)
	initID := seedNode(t, store, "Initiative", "init-x", "Init X", "repo-x")
	featID := seedNode(t, store, "Feature", "feat-x", "Feat X", "repo-x")
	critID := seedNode(t, store, "Criterion", "feat-x:AC-1", "feat-x:AC-1", "repo-x")
	seedEdge(t, store, featID, initID, "belongs_to")
	seedEdge(t, store, critID, featID, "belongs_to")

	trace, err := Why(store, "repo-x", "feat-x:AC-1", DefaultDepth)
	if err != nil {
		t.Fatalf("Why: %v", err)
	}
	if trace.Target.NodeKey != "feat-x:AC-1" {
		t.Errorf("target = %q, want feat-x:AC-1", trace.Target.NodeKey)
	}
	if len(trace.Chains) != 2 {
		t.Fatalf("hops = %d, want 2: %#v", len(trace.Chains), trace.Chains)
	}
	if trace.Chains[0].NodeKey != "feat-x" {
		t.Errorf("hop 1 = %q, want feat-x", trace.Chains[0].NodeKey)
	}
	if trace.Chains[1].NodeKey != "init-x" {
		t.Errorf("hop 2 = %q, want init-x", trace.Chains[1].NodeKey)
	}
}

func TestWhy_DepthBoundsRecursion(t *testing.T) {
	store := openStore(t)
	a := seedNode(t, store, "Feature", "a", "A", "repo-x")
	b := seedNode(t, store, "Feature", "b", "B", "repo-x")
	c := seedNode(t, store, "Feature", "c", "C", "repo-x")
	d := seedNode(t, store, "Feature", "d", "D", "repo-x")
	seedEdge(t, store, a, b, "belongs_to")
	seedEdge(t, store, b, c, "belongs_to")
	seedEdge(t, store, c, d, "belongs_to")

	trace, err := Why(store, "repo-x", "a", 2)
	if err != nil {
		t.Fatalf("Why: %v", err)
	}
	if len(trace.Chains) != 2 {
		t.Errorf("with depth=2, expected 2 hops, got %d", len(trace.Chains))
	}
}

func TestWhy_NoOriginRendersExplicitMessage(t *testing.T) {
	store := openStore(t)
	seedNode(t, store, "Feature", "lonely", "Lonely", "repo-x")

	trace, err := Why(store, "repo-x", "lonely", DefaultDepth)
	if err != nil {
		t.Fatalf("Why: %v", err)
	}
	md := trace.Markdown()
	if !strings.Contains(md, "no upstream origin") {
		t.Errorf("expected 'no upstream origin' message, got:\n%s", md)
	}
}

func TestWhy_NotFound(t *testing.T) {
	store := openStore(t)
	_, err := Why(store, "repo-x", "nope", DefaultDepth)
	if err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestWhy_BreaksCycles(t *testing.T) {
	store := openStore(t)
	a := seedNode(t, store, "Feature", "a", "A", "repo-x")
	b := seedNode(t, store, "Feature", "b", "B", "repo-x")
	seedEdge(t, store, a, b, "belongs_to")
	seedEdge(t, store, b, a, "belongs_to") // cycle

	trace, err := Why(store, "repo-x", "a", 6)
	if err != nil {
		t.Fatalf("Why: %v", err)
	}
	// Cycle traversal still terminates within the depth cap. We don't
	// assert exact hop count — just that the call returns and the
	// chain length stays bounded.
	if len(trace.Chains) > 12 {
		t.Errorf("cycle blew the depth bound: %d hops", len(trace.Chains))
	}
}

// TestWhy_DepthFourUnder200ms locks in traversal-queries AC-8 — the
// recursive CTE plus markdown render must complete under the v2 spec
// success budget. Uses a synthetic 6-deep chain so the depth bound
// is exercised.
func TestWhy_DepthFourUnder200ms(t *testing.T) {
	store := openStore(t)
	prev := seedNode(t, store, "Feature", "f0", "F0", "repo-x")
	for i := 1; i <= 6; i++ {
		next := seedNode(t, store, "Feature", fmt.Sprintf("f%d", i), fmt.Sprintf("F%d", i), "repo-x")
		seedEdge(t, store, prev, next, "belongs_to")
		prev = next
	}

	// Warm: open store, run once to populate any per-statement
	// SQLite caches.
	if _, err := Why(store, "repo-x", "f0", 6); err != nil {
		t.Fatalf("warm: %v", err)
	}

	const budget = 200 * time.Millisecond
	start := time.Now()
	for i := 0; i < 10; i++ {
		if _, err := Why(store, "repo-x", "f0", 6); err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
	}
	avg := time.Since(start) / 10
	if avg > budget {
		t.Errorf("avg Why latency = %v, want < %v", avg, budget)
	}
}

func TestMarkdown_RendersHopsIndented(t *testing.T) {
	tr := &Trace{
		Target: Hop{NodeType: "Feature", NodeKey: "a", NodeTitle: "A"},
		Chains: []Hop{
			{Depth: 1, NodeType: "Initiative", NodeKey: "i1", NodeTitle: "I1", EdgeType: "belongs_to"},
			{Depth: 2, NodeType: "Initiative", NodeKey: "i2", NodeTitle: "I2", EdgeType: "belongs_to"},
		},
	}
	md := tr.Markdown()
	if !strings.Contains(md, "← _belongs_to_ I1") {
		t.Errorf("missing depth-1 hop: %s", md)
	}
	if !strings.Contains(md, "  ← _belongs_to_ I2") {
		t.Errorf("missing indented depth-2 hop: %s", md)
	}
}

// --- helpers ---

func openStore(t *testing.T) *graph.Store {
	t.Helper()
	store, err := graph.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func seedNode(t *testing.T, store *graph.Store, typ, key, title, repo string) int64 {
	t.Helper()
	domain := "engineering"
	if graph.IsGlobalNodeType(typ) {
		domain = ""
	}
	id, err := store.UpsertNode(&graph.Node{
		Type:        typ,
		Domain:      domain,
		Key:         key,
		Props:       map[string]any{"title": title},
		Repo:        repo,
		ContentHash: typ + "|" + key,
	})
	if err != nil {
		t.Fatalf("seed node %s/%s: %v", typ, key, err)
	}
	return id
}

func seedEdge(t *testing.T, store *graph.Store, from, to int64, typ string) {
	t.Helper()
	if _, err := store.UpsertEdge(&graph.Edge{
		FromID: from, ToID: to, Type: typ,
		Repo: "repo-x",
	}); err != nil {
		t.Fatalf("seed edge: %v", err)
	}
}
