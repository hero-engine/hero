package clusters

import (
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestDetect_BelowThresholdReturnsNothing(t *testing.T) {
	// Two specs sharing a tag — below MinItems (3) → no cluster.
	in := Input{
		Specs: []*spec.Spec{
			{Slug: "a", Title: "A", Tags: []string{"serve"}},
			{Slug: "b", Title: "B", Tags: []string{"serve"}},
		},
	}
	got := Detector{}.Detect(in)
	if len(got) != 0 {
		t.Fatalf("expected 0 clusters below threshold, got %d: %+v", len(got), got)
	}
}

func TestDetect_TagClusterAtThreshold(t *testing.T) {
	in := Input{
		Specs: []*spec.Spec{
			{Slug: "a", Title: "A", Tags: []string{"serve"}},
			{Slug: "b", Title: "B", Tags: []string{"serve"}},
			{Slug: "c", Title: "C", Tags: []string{"serve"}},
		},
	}
	got := Detector{}.Detect(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(got))
	}
	if got[0].Kind != "tag" || got[0].Label != "serve" {
		t.Errorf("wrong cluster: kind=%q label=%q", got[0].Kind, got[0].Label)
	}
	if got[0].ItemCount != 3 {
		t.Errorf("ItemCount = %d, want 3", got[0].ItemCount)
	}
	if len(got[0].Items) != 3 {
		t.Errorf("Items len = %d, want top-3", len(got[0].Items))
	}
}

func TestDetect_PathPrefixCluster(t *testing.T) {
	in := Input{
		Specs: []*spec.Spec{
			{Slug: "a", FilesTouched: []string{"internal/serve/pages/now/x.go"}},
			{Slug: "b", FilesTouched: []string{"internal/serve/pages/work/y.go"}},
			{Slug: "c", FilesTouched: []string{"internal/serve/server.go"}},
			// outside the prefix — not counted
			{Slug: "d", FilesTouched: []string{"internal/spec/foo.go"}},
		},
	}
	got := Detector{PathDepth: 2}.Detect(in)
	var hit *Cluster
	for i := range got {
		if got[i].Kind == "path-prefix" && got[i].Label == "internal/serve/" {
			hit = &got[i]
			break
		}
	}
	if hit == nil {
		t.Fatalf("expected internal/serve/ cluster, got %+v", got)
	}
	if hit.ItemCount != 3 {
		t.Errorf("ItemCount = %d, want 3", hit.ItemCount)
	}
}

func TestDetect_PathPrefixDedupesBySlug(t *testing.T) {
	// One spec with multiple paths under the same prefix counts once.
	in := Input{
		Specs: []*spec.Spec{
			{Slug: "a", FilesTouched: []string{"internal/serve/x.go", "internal/serve/y.go"}},
			{Slug: "b", FilesTouched: []string{"internal/serve/z.go"}},
		},
	}
	got := Detector{PathDepth: 2}.Detect(in)
	// Only 2 unique slugs → below threshold of 3.
	if len(got) != 0 {
		t.Fatalf("expected 0 clusters (2 unique slugs below threshold), got %d: %+v", len(got), got)
	}
}

func TestDetect_AggregateModePreservesProject(t *testing.T) {
	in1 := Input{
		Project: "alpha",
		Specs: []*spec.Spec{
			{Slug: "a", Tags: []string{"auth"}},
			{Slug: "b", Tags: []string{"auth"}},
		},
	}
	in2 := Input{
		Project: "beta",
		Specs: []*spec.Spec{
			{Slug: "c", Tags: []string{"auth"}},
		},
	}
	got := Detector{Aggregate: true}.Detect(in1, in2)
	if len(got) != 1 {
		t.Fatalf("expected 1 cross-project cluster, got %d", len(got))
	}
	if got[0].ItemCount != 3 {
		t.Errorf("ItemCount = %d, want 3 (cross-project sum)", got[0].ItemCount)
	}
	seenProjects := map[string]bool{}
	for _, it := range got[0].Items {
		seenProjects[it.Project] = true
	}
	if !seenProjects["alpha"] || !seenProjects["beta"] {
		t.Errorf("expected items from both projects, got %v", seenProjects)
	}
}

func TestDetect_RanksByItemCount(t *testing.T) {
	in := Input{
		Specs: []*spec.Spec{
			{Slug: "a", Tags: []string{"high", "low"}},
			{Slug: "b", Tags: []string{"high", "low"}},
			{Slug: "c", Tags: []string{"high"}},
			{Slug: "d", Tags: []string{"high"}},
			{Slug: "e", Tags: []string{"low"}},
		},
	}
	got := Detector{}.Detect(in)
	if len(got) < 2 {
		t.Fatalf("expected at least 2 clusters, got %d: %+v", len(got), got)
	}
	if got[0].Label != "high" {
		t.Errorf("expected 'high' first (more items), got %q", got[0].Label)
	}
}

func TestDetect_RespectsMaxClusters(t *testing.T) {
	specs := make([]*spec.Spec, 0)
	tags := []string{"a", "b", "c", "d", "e", "f", "g"}
	for _, tag := range tags {
		for i := 0; i < 3; i++ {
			specs = append(specs, &spec.Spec{
				Slug: tag + "-" + string(rune('0'+i)),
				Tags: []string{tag},
			})
		}
	}
	got := Detector{MaxClusters: 3}.Detect(Input{Specs: specs})
	if len(got) != 3 {
		t.Fatalf("expected 3 clusters (MaxClusters), got %d", len(got))
	}
}

func TestDetect_EmptyInputReturnsNothing(t *testing.T) {
	got := Detector{}.Detect()
	if got != nil && len(got) != 0 {
		t.Errorf("expected nil/empty, got %+v", got)
	}
	got = Detector{}.Detect(Input{})
	if len(got) != 0 {
		t.Errorf("expected 0 clusters from empty input, got %d", len(got))
	}
}
