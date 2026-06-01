package cli

import (
	"strings"
	"testing"
)

// TestCheckSizeDriftSummary_RateLimited covers the spec's
// Implementation Notes constraint: `hero check` MUST collapse leaf
// and container drift into TWO summary lines with counts and a hint
// pointing at `hero size --check`. It MUST NOT dump per-spec rows.
func TestCheckSizeDriftSummary_RateLimited(t *testing.T) {
	env := newTestEnv(t)

	// Two leaf drifts: declared trivial with many files.
	for _, slug := range []string{"leaf-a", "leaf-b"} {
		env.addSpec("planning/features/"+slug+"/spec.md", `---
title: `+slug+`
type: feature
status: planning
size: trivial
---

## Goal
Big body.

## Changes
- internal/a.go
- internal/b.go
- internal/c.go
- internal/d.go
- internal/e.go
- internal/f.go
- internal/g.go
- internal/h.go
`)
	}

	// One container drift: declared small, two large children.
	env.addSpec("planning/initiatives/init-x/spec.md", `---
title: Init X
type: initiative
status: planning
size: small
---

## Goal
Parent.
`)
	for _, slug := range []string{"child-1", "child-2"} {
		env.addSpec("planning/features/"+slug+"/spec.md", `---
title: `+slug+`
type: feature
status: planning
size: large
relations:
  - target: init-x
    kind: parent
---

## Goal
Child.
`)
	}

	env.indexAll()

	out, err := runCmd("check")
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}

	if !strings.Contains(out, "Size drift (leaf):") {
		t.Errorf("expected leaf summary line, got:\n%s", out)
	}
	if !strings.Contains(out, "Size drift (container):") {
		t.Errorf("expected container summary line, got:\n%s", out)
	}
	if !strings.Contains(out, "hero size --check") {
		t.Errorf("expected hint pointing at 'hero size --check', got:\n%s", out)
	}

	// Rate-limit guard: the size drift section MUST be exactly two
	// lines (one leaf, one container) — never per-spec rows. Other
	// `hero check` sections may name slugs for their own reasons
	// (unclaimed, missing kickoff, …) so we can't blanket-forbid the
	// slugs from the full output; instead, isolate the size-drift
	// lines and assert there are exactly the two expected.
	sizeLines := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Size drift (") {
			sizeLines++
		}
	}
	if sizeLines != 2 {
		t.Errorf("expected exactly 2 size-drift summary lines, got %d:\n%s",
			sizeLines, out)
	}
}

// TestSizeCheckCmd_LeafAndContainerDrift covers the `hero size
// --check` extension: both flavors surface, prefixed, with non-zero
// exit on drift.
func TestSizeCheckCmd_LeafAndContainerDrift(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/leaf-x/spec.md", `---
title: Leaf X
type: feature
status: planning
size: trivial
---

## Goal
Body.

## Changes
- internal/a.go
- internal/b.go
- internal/c.go
- internal/d.go
- internal/e.go
- internal/f.go
- internal/g.go
- internal/h.go
`)

	env.addSpec("planning/initiatives/init-x/spec.md", `---
title: Init X
type: initiative
status: planning
size: small
---

## Goal
Parent.
`)
	env.addSpec("planning/features/c1/spec.md", `---
title: C1
type: feature
status: planning
size: large
relations:
  - target: init-x
    kind: parent
---

## Goal
Child.
`)
	env.addSpec("planning/features/c2/spec.md", `---
title: C2
type: feature
status: planning
size: large
relations:
  - target: init-x
    kind: parent
---

## Goal
Child.
`)

	env.indexAll()

	out, err := runCmd("size", "--check")
	if err == nil {
		t.Errorf("expected non-zero exit on drift, got nil error\nOutput:\n%s", out)
	}
	if !strings.Contains(out, "[leaf]") {
		t.Errorf("expected [leaf] prefix in output:\n%s", out)
	}
	if !strings.Contains(out, "[container]") {
		t.Errorf("expected [container] prefix in output:\n%s", out)
	}
	if !strings.Contains(out, "init-x") {
		t.Errorf("expected init-x in container drift output:\n%s", out)
	}
}
