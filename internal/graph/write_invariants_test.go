package graph

import (
	"errors"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

// TestUpsertNodeRequiresDomainForNonGlobalType verifies AC #2:
// non-global node types must carry a Domain on upsert.
func TestUpsertNodeRequiresDomainForNonGlobalType(t *testing.T) {
	s := openTestStore(t)
	_, err := s.UpsertNode(&Node{Type: "Feature", Key: "x", ContentHash: "h"})
	if !errors.Is(err, ErrDomainRequired) {
		t.Errorf("got %v, want ErrDomainRequired", err)
	}
}

// TestUpsertNodeAcceptsEmptyDomainForGlobalType verifies AC #12:
// Mission (and other global allow-list types) may have Domain == "".
func TestUpsertNodeAcceptsEmptyDomainForGlobalType(t *testing.T) {
	s := openTestStore(t)
	for _, typ := range []string{"Mission", "Person", "Org", "Repo", "Unit"} {
		_, err := s.UpsertNode(&Node{Type: typ, Key: "k-" + typ, ContentHash: "h"})
		if err != nil {
			t.Errorf("%s with empty Domain: %v", typ, err)
		}
	}
}

// TestUpsertNodeRejectsDomainMutation verifies first-write-wins on the
// domain partition. Relocating a node across domains is a v2 retag
// concern.
func TestUpsertNodeRejectsDomainMutation(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.UpsertNode(&Node{Type: "Feature", Key: "x", Domain: "engineering", ContentHash: "h1"}); err != nil {
		t.Fatalf("initial: %v", err)
	}
	_, err := s.UpsertNode(&Node{Type: "Feature", Key: "x", Domain: "pm", ContentHash: "h2"})
	if !errors.Is(err, ErrDomainMutation) {
		t.Errorf("got %v, want ErrDomainMutation", err)
	}
}

// TestUpsertNodeSelfHealsGlobalDomain verifies that a global node type
// (Person, Repo) originally created with a non-empty domain can be
// corrected to domain="" on subsequent upsert. This handles nodes that
// were written before their type was added to the global allow-list.
func TestUpsertNodeSelfHealsGlobalDomain(t *testing.T) {
	s := openTestStore(t)

	// Simulate a Repo node created before it was a global type.
	_, err := s.UpsertNode(&Node{Type: "Repo", Key: "acme/app", Domain: "engineering", ContentHash: "h1"})
	if err != nil {
		t.Fatalf("initial upsert with domain: %v", err)
	}

	// Now upsert with the correct empty domain — should self-heal.
	id2, err := s.UpsertNode(&Node{Type: "Repo", Key: "acme/app", Domain: "", ContentHash: "h2"})
	if err != nil {
		t.Fatalf("self-healing upsert: %v", err)
	}
	if id2 == 0 {
		t.Fatal("expected non-zero ID after self-healing upsert")
	}

	// Verify the current node has empty domain.
	node, err := s.GetNode("Repo", "acme/app")
	if err != nil {
		t.Fatalf("GetCurrentNode: %v", err)
	}
	if node.Domain != "" {
		t.Errorf("domain after self-heal = %q, want empty", node.Domain)
	}
}

// TestUpsertNodeSelfHealOnlyForGlobalTypes verifies that the self-heal
// path only applies to global node types. A non-global type changing
// domain still errors.
func TestUpsertNodeSelfHealOnlyForGlobalTypes(t *testing.T) {
	s := openTestStore(t)

	// Create a Feature with domain "engineering".
	_, err := s.UpsertNode(&Node{Type: "Feature", Key: "x", Domain: "engineering", ContentHash: "h1"})
	if err != nil {
		t.Fatalf("initial: %v", err)
	}

	// Attempt to change domain to "pm" — should still be rejected.
	_, err = s.UpsertNode(&Node{Type: "Feature", Key: "x", Domain: "pm", ContentHash: "h2"})
	if !errors.Is(err, ErrDomainMutation) {
		t.Errorf("got %v, want ErrDomainMutation for non-global type domain change", err)
	}
}

// TestEdgeInheritsFromNodeDomain verifies AC #3: when the caller does
// not set Edge.Domain, the edge inherits from the from-node's domain.
func TestEdgeInheritsFromNodeDomain(t *testing.T) {
	s := openTestStore(t)
	fromID, _ := s.UpsertNode(&Node{Type: "Feature", Key: "a", Domain: "pm", ContentHash: "h-a"})
	toID, _ := s.UpsertNode(&Node{Type: "Feature", Key: "b", Domain: "pm", ContentHash: "h-b"})

	edgeID, err := s.UpsertEdge(&Edge{FromID: fromID, ToID: toID, Type: "depends_on"})
	if err != nil {
		t.Fatalf("UpsertEdge: %v", err)
	}

	edges, err := s.EdgesFrom(fromID, "depends_on")
	if err != nil {
		t.Fatalf("EdgesFrom: %v", err)
	}
	if len(edges) != 1 || edges[0].ID != edgeID {
		t.Fatalf("expected one edge, got %+v", edges)
	}
	if edges[0].Domain != "pm" {
		t.Errorf("edge.Domain = %q, want %q (inherited from from-node)", edges[0].Domain, "pm")
	}
}

// TestEdgeRejectsGlobalFromNodeWithoutExplicitDomain verifies the
// "Mission has an edge into PM but the edge is now silently global"
// trap. The from-node is global, so the edge must specify Domain or
// fail with ErrEdgeDomainRequired.
func TestEdgeRejectsGlobalFromNodeWithoutExplicitDomain(t *testing.T) {
	s := openTestStore(t)
	missionID, _ := s.UpsertNode(&Node{Type: "Mission", Key: "ours", ContentHash: "h-m"})
	targetID, _ := s.UpsertNode(&Node{Type: "Feature", Key: "x", Domain: "engineering", ContentHash: "h-x"})

	_, err := s.UpsertEdge(&Edge{FromID: missionID, ToID: targetID, Type: "mentions"})
	if !errors.Is(err, ErrEdgeDomainRequired) {
		t.Errorf("got %v, want ErrEdgeDomainRequired", err)
	}

	// With explicit Domain set, the write succeeds.
	if _, err := s.UpsertEdge(&Edge{
		FromID: missionID, ToID: targetID, Type: "mentions", Domain: "engineering",
	}); err != nil {
		t.Errorf("explicit Domain: %v", err)
	}
}

// TestDomainForActiveFallback verifies DomainFor's "engineering"
// fallback when cfg.Domain is unset (pre-migration workspaces).
func TestDomainForActiveFallback(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.Config
		hint NodeHint
		want string
	}{
		{"active with pm", config.Config{Domain: "pm"}, IntrinsicActive, "pm"},
		{"active empty fallbacks engineering", config.Config{Domain: ""}, IntrinsicActive, "engineering"},
		{"code always engineering", config.Config{Domain: "pm"}, IntrinsicCode, "engineering"},
		{"global always empty", config.Config{Domain: "pm"}, IntrinsicGlobal, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DomainFor(tc.cfg, tc.hint)
			if got != tc.want {
				t.Errorf("DomainFor(%+v, %v) = %q, want %q", tc.cfg, tc.hint, got, tc.want)
			}
		})
	}
}

// TestIsCrossDomainAllowedKind verifies the v1 allow-list. New kinds
// added to the set must update this list — keeps the v1 contract
// stable.
func TestIsCrossDomainAllowedKind(t *testing.T) {
	for _, k := range []string{"handoff", "derived_from", "realizes"} {
		if !IsCrossDomainAllowedKind(k) {
			t.Errorf("%q should be cross-domain allowed in v1", k)
		}
	}
	for _, k := range []string{"depends_on", "imports", "mentions", "blocks"} {
		if IsCrossDomainAllowedKind(k) {
			t.Errorf("%q should NOT be in cross-domain allow-list", k)
		}
	}
}
