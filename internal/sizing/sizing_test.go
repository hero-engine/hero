package sizing

import (
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

// TestBucketFromPoints_TierBoundaries mirrors the cli/cost_test.go
// regression so any threshold change in either copy triggers a test
// failure — the two implementations MUST stay in sync.
func TestBucketFromPoints_TierBoundaries(t *testing.T) {
	tests := []struct {
		points float64
		bucket string
	}{
		{0.5, EffortTrivial},
		{1.9, EffortTrivial},
		{2.0, EffortSmall},
		{4.9, EffortSmall},
		{5.0, EffortMedium},
		{9.9, EffortMedium},
		{10.0, EffortLarge},
		{19.9, EffortLarge},
		{20.0, EffortXLarge},
		{39.9, EffortXLarge},
		{40.0, EffortGiant},
		{100.0, EffortGiant},
	}
	for _, tt := range tests {
		if got := BucketFromPoints(tt.points); got != tt.bucket {
			t.Errorf("BucketFromPoints(%.1f) = %q, want %q", tt.points, got, tt.bucket)
		}
	}
}

// TestCollectDrift_LeafAndContainer wires the full pipeline:
// one leaf with declared drift, one container declared below rollup,
// plus a clean spec that should not appear.
func TestCollectDrift_LeafAndContainer(t *testing.T) {
	specs := []*spec.Spec{
		// Clean leaf — no drift.
		{
			Slug:         "clean",
			Type:         spec.TypeFeature,
			Status:       spec.StatusPlanning,
			Size:         "trivial",
			FilesTouched: []string{},
			Sections:     map[string]string{"goal": "x"},
		},
		// Drifted leaf — declared trivial, lots of files.
		{
			Slug:   "drifted",
			Type:   spec.TypeFeature,
			Status: spec.StatusPlanning,
			Size:   "trivial",
			FilesTouched: []string{
				"a.go", "b.go", "c.go", "d.go", "e.go",
				"f.go", "g.go", "h.go", "i.go", "j.go",
			},
			Sections: map[string]string{"goal": "x", "design": "y"},
		},
		// Initiative with declared-below-rollup drift. Children are
		// given enough file content to land their computed bucket at
		// large, so the leaf-drift check doesn't flag them too —
		// keeping the container-only signal clean in this test.
		{Slug: "init-1", Type: spec.TypeInitiative, Size: "small"},
		largeishLeaf("child-a", "init-1"),
		largeishLeaf("child-b", "init-1"),
	}

	leaf, container := CollectDrift(specs)

	if len(leaf) != 1 {
		t.Errorf("leaf drift count = %d, want 1", len(leaf))
	}
	if len(leaf) > 0 && leaf[0].Slug != "drifted" {
		t.Errorf("leaf drift slug = %q, want drifted", leaf[0].Slug)
	}

	if len(container) != 1 {
		t.Errorf("container drift count = %d, want 1", len(container))
	}
	if len(container) > 0 {
		if container[0].Slug != "init-1" {
			t.Errorf("container drift slug = %q, want init-1", container[0].Slug)
		}
		if container[0].Declared != "small" {
			t.Errorf("declared = %q, want small", container[0].Declared)
		}
		// 2 larges = 28 pts → x-large
		if container[0].Rollup != "x-large" {
			t.Errorf("rollup = %q, want x-large", container[0].Rollup)
		}
	}
}

// largeishLeaf returns a child feature spec with enough file content
// to compute at the "large" tier. Used by container-drift tests so
// the leaf-drift check ignores these children (declared matches
// computed) and only the parent's container drift fires.
func largeishLeaf(slug, parentSlug string) *spec.Spec {
	return &spec.Spec{
		Slug:   slug,
		Type:   spec.TypeFeature,
		Status: spec.StatusPlanning,
		Size:   "large",
		FilesTouched: []string{
			"a.go", "b.go", "c.go", "d.go", "e.go",
			"f.go", "g.go", "h.go", "i.go",
		},
		Sections: map[string]string{
			"goal": "Lots of goal text describing the change",
		},
		Relations: []spec.Relation{{Kind: "parent", Target: parentSlug}},
	}
}

// TestCollectDrift_UnsetDeclaredLeafIgnored confirms that the leaf
// branch only flags specs that carry a declared size — unset is a
// normal state, not a drift signal.
func TestCollectDrift_UnsetDeclaredLeafIgnored(t *testing.T) {
	specs := []*spec.Spec{
		{Slug: "leaf-no-size", Type: spec.TypeFeature,
			FilesTouched: []string{"a.go", "b.go", "c.go"}},
	}
	leaf, _ := CollectDrift(specs)
	if len(leaf) != 0 {
		t.Errorf("expected no leaf drift for unset declared, got %d", len(leaf))
	}
}
