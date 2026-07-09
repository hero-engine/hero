package cli

import (
	"strings"
	"testing"
)

// hero blocked must derive blockers from frontmatter (the durable
// source), matching hero queue, rather than only reading graph.db —
// which is gitignored and empty on a fresh clone until reingest.
func TestBlocked_DerivesFromFrontmatterAtColdStart(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/blocker-spec/spec.md", `---
title: Blocker Spec
type: feature
status: planning
slug: blocker-spec
---
# Blocker Spec
`)
	env.addSpec("planning/features/dependent-spec/spec.md", `---
title: Dependent Spec
type: feature
status: planning
slug: dependent-spec
depends-on: blocker-spec
---
# Dependent Spec
`)
	env.indexAll()
	// No manual graph seeding — the reconcile in runBlocked must build
	// the edges from frontmatter.

	out, err := runCmd("blocked", "--all-domains")
	if err != nil {
		t.Fatalf("blocked errored: %v\n%s", err, out)
	}
	if !strings.Contains(out, "dependent-spec") || !strings.Contains(out, "blocker-spec") {
		t.Errorf("blocked should derive the dependency from frontmatter at cold start, got:\n%s", out)
	}
}
