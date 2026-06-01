package snapshot

import (
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

// TestRollupChildSizes_AllDeclared sums per-tier midpoints and
// re-buckets to confirm the points-sum approach beats max-tier
// arithmetic. Three smalls (3+3+3=9) should land at medium, not stay
// at small.
func TestRollupChildSizes_AllDeclared(t *testing.T) {
	children := []*spec.Spec{
		{Slug: "a", Size: "small"},
		{Slug: "b", Size: "small"},
		{Slug: "c", Size: "small"},
	}
	r := RollupChildSizes(children, nil)
	if r.Indeterminate {
		t.Fatalf("expected determinate rollup, got indeterminate")
	}
	if r.ChildCount != 3 {
		t.Errorf("ChildCount = %d, want 3", r.ChildCount)
	}
	if r.Tier != "medium" {
		t.Errorf("Tier = %q, want medium (3 smalls @ 3 pts = 9, medium band)", r.Tier)
	}
}

// TestRollupChildSizes_MixedDeclaredAndComputed covers the fallback
// path: a child with no declared size uses the SizeProvider's
// computed bucket.
func TestRollupChildSizes_MixedDeclaredAndComputed(t *testing.T) {
	children := []*spec.Spec{
		{Slug: "a", Size: "medium"}, // 7
		{Slug: "b"},                  // no declared → computed "large" → 14
	}
	sizeFn := func(s *spec.Spec) string {
		if s.Slug == "b" {
			return "large"
		}
		return ""
	}
	r := RollupChildSizes(children, sizeFn)
	if r.Indeterminate {
		t.Fatalf("expected determinate rollup, got indeterminate")
	}
	if r.Points != 21 {
		t.Errorf("Points = %v, want 21", r.Points)
	}
	if r.Tier != "x-large" {
		t.Errorf("Tier = %q, want x-large (21 pts lands in 20-39 band)", r.Tier)
	}
}

// TestRollupChildSizes_UnknownChildIsIndeterminate covers the
// "missing both declared and computed" case. Per spec Risks, this
// must NOT silently understate — flag indeterminate.
func TestRollupChildSizes_UnknownChildIsIndeterminate(t *testing.T) {
	children := []*spec.Spec{
		{Slug: "a", Size: "medium"},
		{Slug: "b"}, // declared empty, sizeFn returns ""
	}
	sizeFn := func(s *spec.Spec) string { return "" }
	r := RollupChildSizes(children, sizeFn)
	if !r.Indeterminate {
		t.Fatal("expected indeterminate rollup when a child has no declared or computed size")
	}
	if r.Tier != "" {
		t.Errorf("Tier = %q, want empty when indeterminate", r.Tier)
	}
}

// TestRollupChildSizes_EmptyChildrenIsZero covers the no-children
// edge case — empty rollup must not flag anything.
func TestRollupChildSizes_EmptyChildrenIsZero(t *testing.T) {
	r := RollupChildSizes(nil, nil)
	if r.Indeterminate {
		t.Error("expected determinate (zero) rollup for empty children")
	}
	if r.ChildCount != 0 {
		t.Errorf("ChildCount = %d, want 0", r.ChildCount)
	}
	if r.Tier != "" {
		t.Errorf("Tier = %q, want empty for empty rollup", r.Tier)
	}
}

// TestContainerDrift_DeclaredBelowRollup is the canonical drift
// case: declared `medium` vs rollup `large` triggers drift.
func TestContainerDrift_DeclaredBelowRollup(t *testing.T) {
	parent := &spec.Spec{Slug: "init-one", Type: spec.TypeInitiative, Size: "medium"}
	// 2 larges = 28 pts → x-large
	children := []*spec.Spec{
		{Slug: "a", Size: "large"},
		{Slug: "b", Size: "large"},
	}
	d := ContainerDrift(parent, children, nil)
	if d == nil {
		t.Fatal("expected drift report, got nil")
	}
	if d.Declared != "medium" || d.Rollup != "x-large" {
		t.Errorf("got declared=%q rollup=%q, want medium / x-large", d.Declared, d.Rollup)
	}
}

// TestContainerDrift_DeclaredAtOrAboveRollupNoDrift covers both
// equal and strictly-greater declared cases — neither flags.
func TestContainerDrift_DeclaredAtOrAboveRollupNoDrift(t *testing.T) {
	cases := []struct {
		name      string
		declared  string
		childSize string
	}{
		{"declared-equals-rollup", "small", "small"},
		{"declared-above-rollup", "large", "small"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parent := &spec.Spec{Slug: "init-x", Type: spec.TypeInitiative, Size: c.declared}
			children := []*spec.Spec{{Slug: "k", Size: c.childSize}}
			if d := ContainerDrift(parent, children, nil); d != nil {
				t.Errorf("expected no drift, got %+v", d)
			}
		})
	}
}

// TestContainerDrift_DeclaredUnsetWithRollup flags drift when the
// initiative carries no declared size but has determinable scope —
// the user should be prompted to stamp it.
func TestContainerDrift_DeclaredUnsetWithRollup(t *testing.T) {
	parent := &spec.Spec{Slug: "init-z", Type: spec.TypeInitiative}
	children := []*spec.Spec{{Slug: "c1", Size: "medium"}}
	d := ContainerDrift(parent, children, nil)
	if d == nil {
		t.Fatal("expected drift report for unset declared with non-empty rollup")
	}
	if d.Declared != "" {
		t.Errorf("Declared = %q, want empty", d.Declared)
	}
	if d.Rollup != "medium" {
		t.Errorf("Rollup = %q, want medium", d.Rollup)
	}
}

// TestContainerDrift_IndeterminateSkipsWhenDeclared covers the
// "indeterminate rollup + declared set" branch — when the user has
// declared a size, we trust it and skip rather than emit a noisy
// indeterminate flag.
func TestContainerDrift_IndeterminateSkipsWhenDeclared(t *testing.T) {
	parent := &spec.Spec{Slug: "init-y", Type: spec.TypeInitiative, Size: "large"}
	children := []*spec.Spec{
		{Slug: "a", Size: "medium"},
		{Slug: "b"}, // no declared, no computed
	}
	d := ContainerDrift(parent, children, func(*spec.Spec) string { return "" })
	if d != nil {
		t.Fatalf("expected nil (indeterminate, declared trusted), got %+v", d)
	}
}

// TestContainerDrift_IndeterminateSurfacedWhenUnset covers the
// "indeterminate rollup + no declared" branch — surface so the user
// knows the rollup couldn't be computed.
func TestContainerDrift_IndeterminateSurfacedWhenUnset(t *testing.T) {
	parent := &spec.Spec{Slug: "init-y", Type: spec.TypeInitiative}
	children := []*spec.Spec{
		{Slug: "a", Size: "medium"},
		{Slug: "b"}, // unknown
	}
	d := ContainerDrift(parent, children, func(*spec.Spec) string { return "" })
	if d == nil {
		t.Fatal("expected indeterminate report when declared is unset")
	}
	if !d.Indeterminate {
		t.Error("expected Indeterminate=true")
	}
}

// TestContainerDrift_NoChildrenIsNil covers the empty-initiative
// case. The Constraints section is explicit: no children → no drift,
// rollup is the empty sum.
func TestContainerDrift_NoChildrenIsNil(t *testing.T) {
	parent := &spec.Spec{Slug: "init-empty", Type: spec.TypeInitiative, Size: "small"}
	if d := ContainerDrift(parent, nil, nil); d != nil {
		t.Errorf("expected nil for empty initiative, got %+v", d)
	}
}

// TestBuildParentMap exercises the parent/child-of relation
// extraction shared with rollupInitiatives.
func TestBuildParentMap(t *testing.T) {
	specs := []*spec.Spec{
		{Slug: "init-1", Type: spec.TypeInitiative},
		{Slug: "child-a", Relations: []spec.Relation{{Kind: "parent", Target: "init-1"}}},
		{Slug: "child-b", Relations: []spec.Relation{{Kind: "child-of", Target: "init-1"}}},
		{Slug: "unrelated", Relations: []spec.Relation{{Kind: "relates-to", Target: "init-1"}}},
	}
	m := BuildParentMap(specs)
	got := m["init-1"]
	if len(got) != 2 {
		t.Errorf("expected 2 children for init-1, got %d", len(got))
	}
}
