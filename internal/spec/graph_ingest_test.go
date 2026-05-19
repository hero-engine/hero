package spec

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func openSpecTestStore(t *testing.T) *graph.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := graph.Open(filepath.Join(dir, "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func sampleSpecs() []*Spec {
	now := time.Now().UTC()
	return []*Spec{
		{
			Slug: "hero-marketing", Title: "Marketing initiative",
			Type: TypeInitiative, Status: StatusActive,
			Path: "/repo/.hero/planning/initiatives/hero-marketing/spec.md",
			ModifiedAt: now,
		},
		{
			Slug: "hero-positioning", Title: "Positioning",
			Type: TypeFeature, Status: StatusPlanning, Priority: "P0",
			Tags:       []string{"marketing", "messaging"},
			Path:       "/repo/.hero/planning/features/hero-positioning/spec.md",
			ModifiedAt: now,
			Relations: []Relation{
				{Target: "hero-marketing", Kind: "parent"},
			},
		},
		{
			Slug: "hero-landing-page", Title: "Landing page",
			Type: TypeFeature, Status: StatusPlanning,
			Path: "/repo/.hero/planning/features/hero-landing-page/spec.md",
			ModifiedAt: now,
			Relations: []Relation{
				{Target: "hero-positioning", Kind: "depends-on"},
				{Target: "hero-marketing", Kind: "parent"},
			},
		},
	}
}

// TestSpecWriteGraph_PerSpecDomainOverridesFallback verifies the DSKG
// Phase 4 contract: a spec whose frontmatter declares `domain: pm`
// lands under the pm partition even when the workspace fallback is
// engineering. Legacy specs without the field inherit the fallback.
func TestSpecWriteGraph_PerSpecDomainOverridesFallback(t *testing.T) {
	store := openSpecTestStore(t)
	now := time.Now().UTC()
	specs := []*Spec{
		{
			Slug: "pm-story", Title: "PM story",
			Type: TypeFeature, Status: StatusPlanning,
			Domain:     "pm", // frontmatter-declared
			Path:       "/repo/.hero/planning/features/pm-story/spec.md",
			ModifiedAt: now,
		},
		{
			Slug: "legacy-eng", Title: "Legacy engineering",
			Type: TypeFeature, Status: StatusPlanning,
			// Domain unset — should fall back to "engineering"
			Path:       "/repo/.hero/planning/features/legacy-eng/spec.md",
			ModifiedAt: now,
		},
	}
	if _, err := WriteGraph(specs, "test-repo", "engineering", store); err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	pm, _ := store.GetNode("Feature", "pm-story")
	if pm == nil || pm.Domain != "pm" {
		t.Errorf("pm-story.Domain = %q, want pm", domainOrEmpty(pm))
	}
	legacy, _ := store.GetNode("Feature", "legacy-eng")
	if legacy == nil || legacy.Domain != "engineering" {
		t.Errorf("legacy-eng.Domain = %q, want engineering (fallback)", domainOrEmpty(legacy))
	}
}

func domainOrEmpty(n *graph.Node) string {
	if n == nil {
		return "<nil>"
	}
	return n.Domain
}

func TestSpecWriteGraphInsertsNodesAndEdges(t *testing.T) {
	store := openSpecTestStore(t)
	summary, err := WriteGraph(sampleSpecs(), "test-repo", "engineering", store)
	if err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	if summary.Specs != 3 {
		t.Errorf("Specs = %d, want 3", summary.Specs)
	}
	// Two parent edges + one depends_on
	if summary.Edges != 3 {
		t.Errorf("Edges = %d, want 3", summary.Edges)
	}

	stats, _ := store.Stats()
	if stats.NodesByType["Initiative"] != 1 {
		t.Errorf("Initiative nodes = %d, want 1", stats.NodesByType["Initiative"])
	}
	if stats.NodesByType["Feature"] != 2 {
		t.Errorf("Feature nodes = %d, want 2", stats.NodesByType["Feature"])
	}
	if stats.EdgesByType["belongs_to"] != 2 {
		t.Errorf("belongs_to edges = %d, want 2", stats.EdgesByType["belongs_to"])
	}
	if stats.EdgesByType["depends_on"] != 1 {
		t.Errorf("depends_on edges = %d, want 1", stats.EdgesByType["depends_on"])
	}
}

func TestSpecWriteGraphIsIdempotent(t *testing.T) {
	store := openSpecTestStore(t)
	if _, err := WriteGraph(sampleSpecs(), "test-repo", "engineering", store); err != nil {
		t.Fatalf("first WriteGraph: %v", err)
	}
	before, _ := store.Stats()
	if _, err := WriteGraph(sampleSpecs(), "test-repo", "engineering", store); err != nil {
		t.Fatalf("second WriteGraph: %v", err)
	}
	after, _ := store.Stats()
	if before.HistoryRows.Nodes != after.HistoryRows.Nodes ||
		before.HistoryRows.Edges != after.HistoryRows.Edges {
		t.Errorf("history grew on idempotent re-ingest: nodes %d→%d, edges %d→%d",
			before.HistoryRows.Nodes, after.HistoryRows.Nodes,
			before.HistoryRows.Edges, after.HistoryRows.Edges)
	}
}

func TestSpecWriteGraphChildRelationNotEmittedAsEdge(t *testing.T) {
	store := openSpecTestStore(t)
	specs := []*Spec{
		{
			Slug: "parent-feat", Type: TypeFeature, Status: StatusPlanning,
			ModifiedAt: time.Now(),
			Relations: []Relation{
				{Target: "child-feat", Kind: "child"}, // should NOT emit edge
			},
		},
		{
			Slug: "child-feat", Type: TypeFeature, Status: StatusPlanning,
			ModifiedAt: time.Now(),
			Relations: []Relation{
				{Target: "parent-feat", Kind: "parent"}, // SHOULD emit
			},
		},
	}
	if _, err := WriteGraph(specs, "test-repo", "engineering", store); err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	stats, _ := store.Stats()
	if stats.EdgesByType["belongs_to"] != 1 {
		t.Errorf("belongs_to = %d, want 1 (child relation should not emit a duplicate)", stats.EdgesByType["belongs_to"])
	}
}

func TestSpecWriteGraphSkipsUnknownRelationTarget(t *testing.T) {
	store := openSpecTestStore(t)
	specs := []*Spec{
		{
			Slug: "a", Type: TypeFeature, Status: StatusPlanning,
			ModifiedAt: time.Now(),
			Relations:  []Relation{{Target: "does-not-exist", Kind: "depends-on"}},
		},
	}
	summary, err := WriteGraph(specs, "test-repo", "engineering", store)
	if err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	if summary.Edges != 0 {
		t.Errorf("expected 0 edges (target missing), got %d", summary.Edges)
	}
}
