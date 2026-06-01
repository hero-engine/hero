package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSize_Get_Unset(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/no-size/spec.md", `---
title: No Size
type: feature
status: planning
---

## Goal
Has no declared size.
`)

	out, err := runCmd("size", "no-size")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "(unset)") {
		t.Errorf("expected (unset) for spec without size, got: %q", out)
	}
}

func TestSize_Get_Declared(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/big-thing/spec.md", `---
title: Big Thing
type: feature
status: planning
size: large
---

## Goal
Declared large.
`)

	out, err := runCmd("size", "big-thing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "large" {
		t.Errorf("expected 'large', got: %q", strings.TrimSpace(out))
	}
}

func TestSize_Set_NewField(t *testing.T) {
	env := newTestEnv(t)

	specPath := "planning/features/scope-up/spec.md"
	env.addSpec(specPath, `---
title: Scope Up
type: feature
status: planning
---

## Goal
Bump size after scope creep.
`)

	_, err := runCmd("size", "scope-up", "x-large")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the frontmatter was written.
	data, err := os.ReadFile(filepath.Join(env.heroDir, specPath))
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	if !strings.Contains(string(data), "size: x-large") {
		t.Errorf("expected 'size: x-large' in frontmatter, got:\n%s", string(data))
	}

	// Get should now return x-large.
	out, err := runCmd("size", "scope-up")
	if err != nil {
		t.Fatalf("get after set failed: %v", err)
	}
	if strings.TrimSpace(out) != "x-large" {
		t.Errorf("expected 'x-large' after set, got: %q", strings.TrimSpace(out))
	}
}

func TestSize_Set_UpdateExisting(t *testing.T) {
	env := newTestEnv(t)

	specPath := "planning/features/grow/spec.md"
	env.addSpec(specPath, `---
title: Grow
type: feature
status: planning
size: medium
priority: P1
---

## Goal
Update size in place.
`)

	_, err := runCmd("size", "grow", "large")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(env.heroDir, specPath))
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "size: large") {
		t.Errorf("expected 'size: large' in frontmatter, got:\n%s", content)
	}
	if strings.Contains(content, "size: medium") {
		t.Errorf("old 'size: medium' should have been replaced, got:\n%s", content)
	}
	if !strings.Contains(content, "priority: P1") {
		t.Errorf("other frontmatter (priority) should be preserved, got:\n%s", content)
	}
}

func TestSize_Set_RejectsInvalidTier(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/whatever/spec.md", `---
title: Whatever
type: feature
status: planning
---

## Goal
Reject invalid sizes.
`)

	_, err := runCmd("size", "whatever", "huge")
	if err == nil {
		t.Fatal("expected error for invalid tier 'huge'")
	}
	if !strings.Contains(err.Error(), "invalid size") {
		t.Errorf("expected 'invalid size' error, got: %v", err)
	}
}

func TestSize_Set_SpecNotFound(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("size", "no-such-spec", "medium")
	if err == nil {
		t.Fatal("expected error for missing spec")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestSize_Check_NoDrift(t *testing.T) {
	env := newTestEnv(t)

	// Spec with no declared size — must not surface.
	env.addSpec("planning/features/no-decl/spec.md", `---
title: No Declaration
type: feature
status: planning
---

## Goal
Unset size.

## Changes
- internal/foo.go
`)

	out, err := runCmd("size", "--check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No size drift") {
		t.Errorf("expected 'No size drift' message, got: %q", out)
	}
}

func TestSize_Check_FindsDrift(t *testing.T) {
	env := newTestEnv(t)

	// Heavy spec declared trivial — must drift. The spec has
	// many files + sections + words to push the computed bucket
	// well above trivial.
	bigBody := strings.Repeat("This spec has a lot of content. ", 100)
	env.addSpec("planning/features/drifted/spec.md", `---
title: Drifted
type: feature
status: planning
size: trivial
---

## Goal
Declared trivial but reality is much larger.

## Design
`+bigBody+`

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

	out, err := runCmd("size", "--check")
	if err == nil {
		t.Fatal("expected non-zero exit when drift is found")
	}
	if !strings.Contains(out, "drifted") {
		t.Errorf("expected drifted slug in output, got: %q", out)
	}
	if !strings.Contains(out, "trivial") {
		t.Errorf("expected declared 'trivial' in output, got: %q", out)
	}
}

func TestSize_Check_SkipsContainerSpecs(t *testing.T) {
	env := newTestEnv(t)

	// Initiative with declared size — container drift ships in
	// slice 3, so this slice must not flag it.
	env.addSpec("planning/initiatives/big-init/spec.md", `---
title: Big Init
type: initiative
status: planning
size: small
---

## Goal
Container drift is out of scope for slice 2.

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

	out, err := runCmd("size", "--check")
	if err != nil {
		t.Fatalf("expected no drift error for initiative, got: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No size drift") {
		t.Errorf("expected 'No size drift' when only initiative present, got: %q", out)
	}
}

func TestSize_NoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("size", "anything")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}
