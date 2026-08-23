package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/domains"
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

func TestBlockedUsesEnabledStackAndFocusedDomainRanking(t *testing.T) {
	env := newTestEnv(t)
	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetDomainComposition(domains.DomainEngineering, []domains.DomainID{domains.DomainPM, domains.DomainQA}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{"engineering", "pm", "qa"} {
		env.addSpec("planning/features/"+domain+"-work/spec.md", fmt.Sprintf(`---
title: %s work
type: feature
status: planning
domain: %s
relations:
  - target: %s-blocker
    kind: depends-on
---

# Goal

Blocked work.
`, domain, domain, domain))
		env.addSpec("planning/features/"+domain+"-blocker/spec.md", fmt.Sprintf(`---
title: %s blocker
type: feature
status: planning
domain: %s
---

# Goal

Unfinished dependency.
`, domain, domain))
	}
	out, err := runCmd("blocked", "--focused-domain", "qa")
	if err != nil {
		t.Fatal(err)
	}
	qa := strings.Index(out, "qa work")
	engineering := strings.Index(out, "engineering work")
	pm := strings.Index(out, "pm work")
	if qa < 0 || engineering < 0 || pm < 0 {
		t.Fatalf("enabled-stack results missing:\n%s", out)
	}
	if !(qa < engineering && engineering < pm) {
		t.Fatalf("focused/primary/extension ranking wrong:\n%s", out)
	}
}

func TestBlockedExplicitDomainRetrievesDisabledExtensionHistory(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/qa-history/spec.md", `---
title: QA historical work
type: feature
status: planning
domain: qa
depends-on: qa-historical-blocker
---

# Goal

Historical QA work.
`)
	env.addSpec("planning/features/qa-historical-blocker/spec.md", `---
title: QA historical blocker
type: feature
status: planning
domain: qa
---

# Goal

Historical dependency.
`)
	defaultOutput, err := runCmd("blocked")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(defaultOutput, "QA historical work") {
		t.Fatalf("disabled QA history leaked into default scope:\n%s", defaultOutput)
	}
	explicitOutput, err := runCmd("blocked", "--domain", "qa")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(explicitOutput, "QA historical work") {
		t.Fatalf("explicit QA scope did not retrieve history:\n%s", explicitOutput)
	}
}
