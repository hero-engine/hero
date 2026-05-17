package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

// TestListRender_EngineeringIdentical verifies that with no
// vocabulary / methodology set in hero.json, `hero list --format text`
// output continues to render "feature" as the canonical literal — i.e.
// the engineering corpus's appearance is bit-for-bit unchanged from
// before B6.
func TestListRender_EngineeringIdentical(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	env := newTestEnv(t)
	env.addSpec(filepath.Join("planning", "features", "csv-export", "spec.md"), featureWithKickoff)

	out, err := runCmd("list", "--format", "text")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// The text format renders as `(feature/planning)`. Under engineering
	// defaults (no vocab) the type literal must remain "feature".
	if !strings.Contains(out, "(feature/planning)") {
		t.Errorf("engineering list output missing canonical type %q\n--- output ---\n%s", "(feature/planning)", out)
	}
	// And must NOT show a vocabulary-translated term.
	for _, term := range []string{"(Story", "(story/planning)", "Scope/planning"} {
		if strings.Contains(out, term) {
			t.Errorf("engineering list output unexpectedly contains vocabulary term %q\n--- output ---\n%s", term, out)
		}
	}
}

// TestListRender_AgileScrumStory verifies that with vocabulary set to
// agile-scrum, `hero list --format text` renders engineering's
// `type: feature` specs as "Story" — the canonical user-facing change
// promised by the sprint spec acceptance criteria.
func TestListRender_AgileScrumStory(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	env := newTestEnv(t)
	env.addSpec(filepath.Join("planning", "features", "story-render", "spec.md"), featureWithKickoff)

	// Switch the workspace to agile-scrum.
	cfg := config.DefaultConfig()
	cfg.Vocabulary = "agile-scrum"
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	out, err := runCmd("list", "--format", "text")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "(Story/planning)") {
		t.Errorf("agile-scrum list output missing translated type %q\n--- output ---\n%s", "(Story/planning)", out)
	}
	if strings.Contains(out, "(feature/planning)") {
		t.Errorf("agile-scrum list output still shows canonical type %q\n--- output ---\n%s", "(feature/planning)", out)
	}
}

// TestStatusDialect_QuietByDefault asserts that `hero status` does not
// emit the "Vocabulary:" header line when no vocab/methodology is set.
// Preserves bit-for-bit identical output for engineering workspaces.
func TestStatusDialect_QuietByDefault(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	env := newTestEnv(t)
	env.addSpec(filepath.Join("planning", "features", "demo", "spec.md"), featureWithKickoff)

	out, err := runCmd("status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, term := range []string{"Vocabulary:", "Methodology:"} {
		if strings.Contains(out, term) {
			t.Errorf("engineering status output unexpectedly contains %q\n--- output ---\n%s", term, out)
		}
	}
}

// TestStatusDialect_SurfacesActiveLayer asserts that `hero status`
// prints a "Vocabulary:" line when the workspace declares one — the
// Risk-mitigation surface called out in the sprint spec ("hero status
// should display both active layers prominently").
func TestStatusDialect_SurfacesActiveLayer(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	env := newTestEnv(t)

	cfg := config.DefaultConfig()
	cfg.Vocabulary = "agile-scrum"
	cfg.Methodology = "scrum"
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	out, err := runCmd("status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, "Vocabulary: agile-scrum") {
		t.Errorf("status output missing vocab header\n--- output ---\n%s", out)
	}
	if !strings.Contains(out, "Methodology: scrum") {
		t.Errorf("status output missing methodology header\n--- output ---\n%s", out)
	}
}
