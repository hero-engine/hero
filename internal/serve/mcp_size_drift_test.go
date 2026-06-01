package serve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestToolWarnings_SizeDriftLeafAndContainer covers the slice-3
// MCP integration: hero_warnings emits a per-spec entry for each
// leaf drift and each container drift. Distinct from `hero check`,
// which rate-limits to two summary lines.
func TestToolWarnings_SizeDriftLeafAndContainer(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	mustMkdirAll := func(p string) {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite := func(p, content string) {
		mustMkdirAll(filepath.Dir(p))
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Minimal workspace skeleton.
	mustMkdirAll(filepath.Join(heroDir, "planning", "features"))
	mustMkdirAll(filepath.Join(heroDir, "planning", "initiatives"))
	mustWrite(filepath.Join(tmp, "hero.json"), `{"directory": ".hero"}`)

	// Leaf drift: declared tiny, lots of files → computed at least large.
	mustWrite(filepath.Join(heroDir, "planning", "features", "leaf-drifted", "spec.md"),
		`---
title: Leaf Drifted
type: feature
status: planning
size: trivial
---

## Goal
Body body body.

## Changes
- internal/a.go
- internal/b.go
- internal/c.go
- internal/d.go
- internal/e.go
- internal/f.go
- internal/g.go
- internal/h.go
- internal/i.go
- internal/j.go
`)

	// Container drift: declared small, two large children.
	mustWrite(filepath.Join(heroDir, "planning", "initiatives", "init-drifted", "spec.md"),
		`---
title: Init Drifted
type: initiative
status: planning
size: small
---

## Goal
Parent.
`)
	for _, slug := range []string{"child-a", "child-b"} {
		mustWrite(filepath.Join(heroDir, "planning", "features", slug, "spec.md"),
			`---
title: `+slug+`
type: feature
status: planning
size: large
relations:
  - target: init-drifted
    kind: parent
---

## Goal
Child.
`)
	}

	// We don't strictly need the index for toolWarnings size-drift
	// branch — sizing.CollectDrift uses spec.Discover directly — but
	// other branches (stale/unclaimed checks) want it. Open + close to
	// initialize.
	srv := NewMCPServer(heroDir, tmp, "test")

	out, err := srv.toolWarnings(map[string]interface{}{})
	if err != nil {
		t.Fatalf("toolWarnings: %v", err)
	}

	if !strings.Contains(out, "Size drift (leaf)") {
		t.Errorf("expected leaf drift entry, got:\n%s", out)
	}
	if !strings.Contains(out, "Size drift (container)") {
		t.Errorf("expected container drift entry, got:\n%s", out)
	}
	if !strings.Contains(out, "leaf-drifted") {
		t.Errorf("expected leaf-drifted slug in MCP output (per-spec entry, not rate-limited):\n%s",
			out)
	}
	if !strings.Contains(out, "init-drifted") {
		t.Errorf("expected init-drifted slug in MCP output:\n%s", out)
	}
	// Spec size-drift-actionable-output:
	// - Tier substitution: no literal <tier> placeholder may survive.
	// - Each drift entry that has an alternative action carries it as
	//   a second clause routed via " or ".
	if strings.Contains(out, "<tier>") {
		t.Errorf("expected computed/rollup tier substituted, found literal <tier>:\n%s", out)
	}
	// Leaf entry's alternative for an "up" drift is the "grown beyond
	// intent" clause; the primary is the paste-ready hero size command.
	if !strings.Contains(out, "'hero size leaf-drifted ") {
		t.Errorf("expected leaf primary 'hero size leaf-drifted <tier>' with substituted tier:\n%s", out)
	}
	if !strings.Contains(out, "grown beyond intent") {
		t.Errorf("expected leaf-up alternative ('grown beyond intent') in MCP output:\n%s", out)
	}
	// Container entry's alternative routes to /compose.
	if !strings.Contains(out, "'/compose init-drifted'") {
		t.Errorf("expected container alternative '/compose init-drifted' in MCP output:\n%s", out)
	}
}
