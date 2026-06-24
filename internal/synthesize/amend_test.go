package synthesize

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestSplitDeveloperNotes(t *testing.T) {
	body, notes := SplitDeveloperNotes("# T\n\n## What it is\nx\n\n## Developer Notes\nhuman stuff\n")
	if !strings.Contains(body, "## What it is") || strings.Contains(body, "human stuff") {
		t.Errorf("body wrong: %q", body)
	}
	if !strings.HasPrefix(strings.TrimSpace(notes), "## Developer Notes") || !strings.Contains(notes, "human stuff") {
		t.Errorf("notes wrong: %q", notes)
	}

	body2, notes2 := SplitDeveloperNotes("# T\n\n## What it is\nonly\n")
	if notes2 != "" || !strings.Contains(body2, "only") {
		t.Errorf("no-notes case wrong: body=%q notes=%q", body2, notes2)
	}
}

func TestStripFrontmatter(t *testing.T) {
	got := StripFrontmatter("---\ntitle: X\ntype: explainer\n---\n# X\nbody\n")
	if strings.Contains(got, "title:") || !strings.HasPrefix(got, "# X") {
		t.Errorf("StripFrontmatter = %q", got)
	}
}

func TestRenderAmendedPreservesDevNotes(t *testing.T) {
	p := &Packet{
		Title: "Feat",
		Specs: []*spec.Spec{{Slug: "feat-a"}, {Slug: "feat-b"}},
	}
	out := p.RenderAmended("## What it is\nnew body\n", "## Developer Notes\nDO NOT TOUCH\n", "2026-06-23")
	if !strings.Contains(out, "DO NOT TOUCH") {
		t.Error("Developer Notes must be preserved verbatim")
	}
	if !strings.Contains(out, "- feat-a") || !strings.Contains(out, "- feat-b") {
		t.Error("frontmatter must list expanded synthesized_from")
	}
	if !strings.Contains(out, "last_synthesized: 2026-06-23") {
		t.Error("last_synthesized must be bumped")
	}
}

func TestStaleExplainersFlagsNewClusterMember(t *testing.T) {
	hero := t.TempDir()
	// Explainer synthesized from feat-a, long ago.
	writeSpec(t, hero, "knowledge/explainers", "feat-a", "---\ntitle: A\ntype: explainer\nsynthesized_from:\n  - feat-a\nlast_synthesized: 2020-01-01\n---\n# A\n## What it is\nx\n## Developer Notes\n")
	writeSpec(t, hero, "specs", "feat-a", "---\ntitle: A\ntype: feature\nslug: feat-a\nstatus: completed\n---\n# A\n")
	// A new completed spec relating to feat-a, completed recently → stale.
	writeSpec(t, hero, "specs", "feat-b", "---\ntitle: B\ntype: feature\nslug: feat-b\nstatus: completed\ncompleted_at: 2026-01-01\nrelations:\n  - target: feat-a\n    kind: related\n---\n# B\n")

	reports, err := StaleExplainers(hero)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1: %+v", len(reports), reports)
	}
	if len(reports[0].NewSlugs) != 1 || reports[0].NewSlugs[0] != "feat-b" {
		t.Errorf("NewSlugs = %v, want [feat-b]", reports[0].NewSlugs)
	}
}

func TestStaleExplainersCurrentWhenNothingNew(t *testing.T) {
	hero := t.TempDir()
	writeSpec(t, hero, "knowledge/explainers", "feat-a", "---\ntitle: A\ntype: explainer\nsynthesized_from:\n  - feat-a\nlast_synthesized: 2026-06-23\n---\n# A\n## Developer Notes\n")
	writeSpec(t, hero, "specs", "feat-a", "---\ntitle: A\ntype: feature\nslug: feat-a\nstatus: completed\n---\n# A\n")

	reports, err := StaleExplainers(hero)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 0 {
		t.Errorf("expected no stale explainers, got %+v", reports)
	}
}
