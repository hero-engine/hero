package synthesize

import (
	"testing"
)

func TestDetectExplicitInitiativeCluster(t *testing.T) {
	hero := t.TempDir()
	// Initiative with two children, both completed → explicit candidate.
	writeSpec(t, hero, "planning/initiatives", "big", "---\ntitle: Big Feature\ntype: initiative\nslug: big\nstatus: planning\nchild:\n  - kid-a\n  - kid-b\n---\n# Big\n")
	writeSpec(t, hero, "specs", "kid-a", "---\ntitle: Kid A\ntype: feature\nslug: kid-a\nstatus: completed\nparent: big\n---\n# A\n")
	writeSpec(t, hero, "specs", "kid-b", "---\ntitle: Kid B\ntype: feature\nslug: kid-b\nstatus: completed\nparent: big\n---\n# B\n")

	cands, err := Detect(hero)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 {
		t.Fatalf("got %d candidates, want 1: %+v", len(cands), cands)
	}
	c := cands[0]
	if c.OutSlug != "big" {
		t.Errorf("OutSlug = %q, want big", c.OutSlug)
	}
	if c.Confidence < 0.9 {
		t.Errorf("explicit cluster confidence = %.2f, want >= 0.9", c.Confidence)
	}
	if len(c.Slugs) != 2 {
		t.Errorf("Slugs = %v, want 2", c.Slugs)
	}
}

func TestDetectCompletenessGate(t *testing.T) {
	hero := t.TempDir()
	// One child still open → initiative not shippable, no candidate.
	writeSpec(t, hero, "planning/initiatives", "big", "---\ntitle: Big\ntype: initiative\nslug: big\nchild:\n  - kid-a\n  - kid-b\n---\n# Big\n")
	writeSpec(t, hero, "specs", "kid-a", "---\ntitle: A\ntype: feature\nslug: kid-a\nstatus: completed\nparent: big\n---\n# A\n")
	writeSpec(t, hero, "planning/features", "kid-b", "---\ntitle: B\ntype: feature\nslug: kid-b\nstatus: delivering\nparent: big\n---\n# B\n")

	cands, err := Detect(hero)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cands {
		for _, s := range c.Slugs {
			if s == "kid-b" {
				t.Errorf("open spec kid-b must not appear in any candidate: %+v", c)
			}
		}
	}
}

func TestDetectDedupAgainstExistingExplainer(t *testing.T) {
	hero := t.TempDir()
	writeSpec(t, hero, "planning/initiatives", "big", "---\ntitle: Big\ntype: initiative\nslug: big\nchild:\n  - kid-a\n  - kid-b\n---\n# Big\n")
	writeSpec(t, hero, "specs", "kid-a", "---\ntitle: A\ntype: feature\nslug: kid-a\nstatus: completed\nparent: big\n---\n# A\n")
	writeSpec(t, hero, "specs", "kid-b", "---\ntitle: B\ntype: feature\nslug: kid-b\nstatus: completed\nparent: big\n---\n# B\n")
	// An explainer already covering both → no candidate.
	writeSpec(t, hero, "knowledge/explainers", "big", "---\ntitle: Big\ntype: explainer\nsynthesized_from:\n  - kid-a\n  - kid-b\nlast_synthesized: 2026-06-23\n---\n# Big\n")

	cands, err := Detect(hero)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 0 {
		t.Errorf("covered cluster should yield no candidates, got %+v", cands)
	}
}

func TestDetectInferredByFileOverlap(t *testing.T) {
	hero := t.TempDir()
	// Two completed features, no parent, but overlapping touched files +
	// shared tags → inferred candidate above threshold.
	writeSpec(t, hero, "specs", "feat-x", "---\ntitle: X\ntype: feature\nslug: feat-x\nstatus: completed\ntags: [auth]\n---\n# X\n## Changes\n- internal/auth/login.go\n")
	writeSpec(t, hero, "specs", "feat-y", "---\ntitle: Y\ntype: feature\nslug: feat-y\nstatus: completed\ntags: [auth]\n---\n# Y\n## Changes\n- internal/auth/login.go\n")

	cands, err := Detect(hero)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 {
		t.Fatalf("got %d candidates, want 1 inferred: %+v", len(cands), cands)
	}
	if len(cands[0].Slugs) != 2 {
		t.Errorf("inferred cluster Slugs = %v, want 2", cands[0].Slugs)
	}
}

func TestDetectHubFileDoesNotChainEverything(t *testing.T) {
	hero := t.TempDir()
	// Eight unrelated completed features that all touch one common "hub"
	// file must NOT collapse into a single giant cluster.
	for _, slug := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		writeSpec(t, hero, "specs", "feat-"+slug,
			"---\ntitle: "+slug+"\ntype: feature\nslug: feat-"+slug+"\nstatus: completed\n---\n# "+slug+"\n## Changes\n- internal/cli/root.go\n")
	}
	cands, err := Detect(hero)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cands {
		if len(c.Slugs) > 2 {
			t.Errorf("hub file should not chain specs into a big cluster, got %d: %v", len(c.Slugs), c.Slugs)
		}
	}
}

func TestDetectUnrelatedSpecsDoNotCluster(t *testing.T) {
	hero := t.TempDir()
	// Two completed features, no shared parent/relation/file/tag → no cluster.
	writeSpec(t, hero, "specs", "feat-x", "---\ntitle: X\ntype: feature\nslug: feat-x\nstatus: completed\n---\n# X\n")
	writeSpec(t, hero, "specs", "feat-y", "---\ntitle: Y\ntype: feature\nslug: feat-y\nstatus: completed\n---\n# Y\n")

	cands, err := Detect(hero)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 0 {
		t.Errorf("unrelated specs must not cluster, got %+v", cands)
	}
}
