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

// TestSpecWriteGraphIntakeProvenance locks the intake provenance chain:
// an intake gets a graph node (graphTypeFor), and a promoted feature's
// `derived_from` relation materializes as a derived_from edge
// (graphEdgeForRelation) — the edge `hero why` walks back to the intake.
func TestSpecWriteGraphIntakeProvenance(t *testing.T) {
	store := openSpecTestStore(t)
	specs := []*Spec{
		{
			Slug: "csv-export", Title: "Export to CSV",
			Type: TypeIntake, Status: StatusPromoted,
			Path:       "/repo/.hero/planning/intake/csv-export/spec.md",
			ModifiedAt: time.Now(),
			Relations: []Relation{
				{Target: "csv-export", Kind: "promotes_to"},
			},
		},
		{
			Slug: "csv-export", Title: "Export to CSV",
			Type: TypeFeature, Status: StatusPlanning,
			Path:       "/repo/.hero/planning/features/csv-export/spec.md",
			ModifiedAt: time.Now(),
			Relations: []Relation{
				{Target: "csv-export", Kind: "derived_from"},
			},
		},
	}
	if _, err := WriteGraph(specs, "test-repo", "engineering", store); err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	stats, _ := store.Stats()
	if stats.NodesByType["Intake"] != 1 {
		t.Errorf("Intake nodes = %d, want 1 (intake not graphed)", stats.NodesByType["Intake"])
	}
	if stats.EdgesByType["derived_from"] != 1 {
		t.Errorf("derived_from edges = %d, want 1 (provenance edge not materialized)", stats.EdgesByType["derived_from"])
	}
	if stats.EdgesByType["promotes_to"] != 1 {
		t.Errorf("promotes_to edges = %d, want 1 (forward provenance edge not materialized)", stats.EdgesByType["promotes_to"])
	}

	// The edge must point at the Intake node, not self-loop back to the
	// Feature (the intake and feature share the slug "csv-export").
	var toType string
	if err := store.DB().QueryRow(
		`SELECT n.type FROM edges e JOIN nodes n ON n.id = e.to_id
		  WHERE e.type = 'derived_from' AND e.valid_to IS NULL LIMIT 1`,
	).Scan(&toType); err != nil {
		t.Fatalf("querying derived_from target: %v", err)
	}
	if toType != "Intake" {
		t.Errorf("derived_from points at %q, want Intake (self-loop bug)", toType)
	}

	if err := store.DB().QueryRow(
		`SELECT n.type FROM edges e JOIN nodes n ON n.id = e.to_id
		  WHERE e.type = 'promotes_to' AND e.valid_to IS NULL LIMIT 1`,
	).Scan(&toType); err != nil {
		t.Fatalf("querying promotes_to target: %v", err)
	}
	if toType != "Feature" {
		t.Errorf("promotes_to points at %q, want Feature (self-loop bug)", toType)
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

// relates-to/relates_to must map to a related_to edge (they previously
// fell through to no edge, silently dropping the relationship).
func TestGraphEdgeForRelation_RelatesToMapsToEdge(t *testing.T) {
	cases := map[string]string{
		"related":    "related_to",
		"relates-to": "related_to",
		"relates_to": "related_to",
		"sibling":    "related_to",
		"parent":     "belongs_to",
		"depends-on": "depends_on",
		// conflicts-with is a distinct soft-mutex edge — must NOT fall
		// through to related_to (that would hide it from the /drive gate).
		"conflicts-with": "conflicts_with",
		"conflicts_with": "conflicts_with",
		"child":          "", // inverse of parent — emitted from the child side
	}
	for kind, want := range cases {
		if got := graphEdgeForRelation(kind); got != want {
			t.Errorf("graphEdgeForRelation(%q) = %q, want %q", kind, got, want)
		}
	}
}
