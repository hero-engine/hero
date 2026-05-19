package graph

import "testing"

// TestHandoffStream_BoundaryAwareAlways verifies AC #9: the Handoff
// stream view returns cross-domain handoff edges regardless of the
// caller's active domain, with both endpoint domains attached so the
// dashboard widget can render the boundary label.
func TestHandoffStream_BoundaryAwareAlways(t *testing.T) {
	s := openTestStore(t)

	storyID, err := s.UpsertNode(&Node{
		Type: "Feature", Domain: "pm", Key: "checkout-pain",
		Props:       map[string]any{"title": "Checkout pain"},
		ContentHash: "h-story",
	})
	if err != nil {
		t.Fatalf("seed story: %v", err)
	}
	featID, err := s.UpsertNode(&Node{
		Type: "Feature", Domain: "engineering", Key: "checkout-fix",
		Props:       map[string]any{"title": "Checkout fix"},
		ContentHash: "h-feat",
	})
	if err != nil {
		t.Fatalf("seed feat: %v", err)
	}
	if _, err := s.UpsertEdge(&Edge{
		FromID: storyID, ToID: featID, Type: "handoff",
		Domain: "pm",
	}); err != nil {
		t.Fatalf("seed handoff: %v", err)
	}

	edges, err := s.HandoffStream(true)
	if err != nil {
		t.Fatalf("HandoffStream: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 cross-domain handoff, got %d", len(edges))
	}
	e := edges[0]
	if e.FromDomain != "pm" || e.ToDomain != "engineering" {
		t.Errorf("boundary endpoints = %s → %s, want pm → engineering", e.FromDomain, e.ToDomain)
	}
	if e.Kind != "handoff" {
		t.Errorf("kind = %q, want handoff", e.Kind)
	}
	if e.FromTitle != "Checkout pain" || e.ToTitle != "Checkout fix" {
		t.Errorf("titles unexpected: %q → %q", e.FromTitle, e.ToTitle)
	}
}

// TestCrossDomainUnusualKindWarnings verifies the warning surface
// flags cross-domain edges whose kind isn't in the v1 allow-list.
// handoff/derived_from/realizes are explicitly allowed and must NOT
// surface; mentions/depends_on across domains MUST surface.
func TestCrossDomainUnusualKindWarnings(t *testing.T) {
	s := openTestStore(t)

	pmID, _ := s.UpsertNode(&Node{Type: "Feature", Domain: "pm", Key: "story-1", ContentHash: "h1"})
	engID, _ := s.UpsertNode(&Node{Type: "Feature", Domain: "engineering", Key: "feat-1", ContentHash: "h2"})

	// Allowed cross-domain — must NOT surface.
	if _, err := s.UpsertEdge(&Edge{FromID: pmID, ToID: engID, Type: "handoff", Domain: "pm"}); err != nil {
		t.Fatalf("handoff: %v", err)
	}

	// Unusual cross-domain — MUST surface.
	if _, err := s.UpsertEdge(&Edge{FromID: pmID, ToID: engID, Type: "depends_on", Domain: "pm"}); err != nil {
		t.Fatalf("depends_on: %v", err)
	}

	ws, err := s.CrossDomainUnusualKindWarnings()
	if err != nil {
		t.Fatalf("warnings: %v", err)
	}
	if len(ws) != 1 {
		t.Fatalf("expected 1 unusual warning, got %d (%+v)", len(ws), ws)
	}
	if ws[0].Kind != "depends_on" {
		t.Errorf("warning kind = %q, want depends_on", ws[0].Kind)
	}
	if ws[0].FromDomain != "pm" || ws[0].ToDomain != "engineering" {
		t.Errorf("warning endpoints = %s → %s, want pm → engineering", ws[0].FromDomain, ws[0].ToDomain)
	}
}
