package sizing

import (
	"strings"
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

// TestSuggestedAction_Mapping covers all four DriftKind cases and
// asserts the returned strings are paste-ready: slug is substituted,
// no template placeholders survive, and the alternative carries the
// expected /split or /compose pointer where applicable.
func TestSuggestedAction_Mapping(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		declared      string
		computed      string
		kind          DriftKind
		wantPrimary   string
		wantAlt       string
		altMustContain string // optional: stronger assertion on alt
	}{
		{
			name:        "leaf-up — declared trivial, computed large",
			slug:        "drifted-leaf",
			declared:    "trivial",
			computed:    "large",
			kind:        DriftKindLeafUp,
			wantPrimary: "'hero size drifted-leaf large' to bump declared",
			wantAlt:     "check whether the spec has grown beyond intent",
		},
		{
			name:           "leaf-down — declared giant, computed small",
			slug:           "shrunk-leaf",
			declared:       "giant",
			computed:       "small",
			kind:           DriftKindLeafDown,
			wantPrimary:    "'hero size shrunk-leaf small' to relax declared",
			wantAlt:        "'/split shrunk-leaf' if the spec is doing two things",
			altMustContain: "/split",
		},
		{
			name:           "container-unset — declared empty, rollup giant",
			slug:           "roadmap-shape",
			declared:       "",
			computed:       "giant",
			kind:           DriftKindContainerUnset,
			wantPrimary:    "'hero size roadmap-shape giant' to acknowledge",
			wantAlt:        "'/compose roadmap-shape' to phase",
			altMustContain: "/compose",
		},
		{
			name:           "container-low — declared small, rollup x-large",
			slug:           "init-1",
			declared:       "small",
			computed:       "x-large",
			kind:           DriftKindContainerLow,
			wantPrimary:    "'hero size init-1 x-large' to bump declared",
			wantAlt:        "'/compose init-1' to phase children",
			altMustContain: "/compose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary, alt := SuggestedAction(tt.slug, tt.declared, tt.computed, tt.kind)
			if primary != tt.wantPrimary {
				t.Errorf("primary = %q, want %q", primary, tt.wantPrimary)
			}
			if alt != tt.wantAlt {
				t.Errorf("alternative = %q, want %q", alt, tt.wantAlt)
			}
			// No template placeholders may survive in either clause.
			for _, bad := range []string{"<slug>", "%s", "<tier>"} {
				if contains(primary, bad) {
					t.Errorf("primary still contains placeholder %q: %q", bad, primary)
				}
				if contains(alt, bad) {
					t.Errorf("alternative still contains placeholder %q: %q", bad, alt)
				}
			}
			// Slug substitution sanity check.
			if !contains(primary, tt.slug) {
				t.Errorf("primary missing slug %q: %q", tt.slug, primary)
			}
			if tt.altMustContain != "" && !contains(alt, tt.altMustContain) {
				t.Errorf("alternative missing %q: %q", tt.altMustContain, alt)
			}
		})
	}
}

// TestClassifyLeafDriftKind covers the up/down classification across
// the ladder including the unknown-tier safe-default.
func TestClassifyLeafDriftKind(t *testing.T) {
	tests := []struct {
		declared string
		computed string
		want     DriftKind
	}{
		{"trivial", "large", DriftKindLeafUp},
		{"small", "x-large", DriftKindLeafUp},
		{"giant", "small", DriftKindLeafDown},
		{"large", "medium", DriftKindLeafDown},
		// Equal — not real drift, but safe-default to LeafUp.
		{"medium", "medium", DriftKindLeafUp},
		// Unknown tiers — safe-default to LeafUp.
		{"huge", "large", DriftKindLeafUp},
	}
	for _, tt := range tests {
		got := ClassifyLeafDriftKind(tt.declared, tt.computed)
		if got != tt.want {
			t.Errorf("ClassifyLeafDriftKind(%q, %q) = %d, want %d",
				tt.declared, tt.computed, got, tt.want)
		}
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
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
