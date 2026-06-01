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
	// Inline next-step hint with paste-ready slug substitution.
	if !strings.Contains(out, "→") {
		t.Errorf("expected inline '→' hint under drift row, got:\n%s", out)
	}
	if !strings.Contains(out, "'hero size drifted ") {
		t.Errorf("expected paste-ready 'hero size drifted <tier>' in inline hint, got:\n%s", out)
	}
	// Footer hint to /roadmap-review.
	if !strings.Contains(out, "Run '/roadmap-review' to triage interactively.") {
		t.Errorf("expected '/roadmap-review' footer hint, got:\n%s", out)
	}
	// No template placeholders may survive in output.
	for _, bad := range []string{"<slug>", "<tier>", "%s"} {
		if strings.Contains(out, bad) {
			t.Errorf("output still contains placeholder %q:\n%s", bad, out)
		}
	}
}

// TestSize_Check_NoDrift_QuietFooter confirms the /roadmap-review
// footer line stays quiet when no drift is found — it's a triage
// pointer, not unconditional noise.
func TestSize_Check_NoDrift_QuietFooter(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/no-decl-2/spec.md", `---
title: No Declaration 2
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
	if strings.Contains(out, "/roadmap-review") {
		t.Errorf("expected no /roadmap-review hint when no drift, got:\n%s", out)
	}
}

// TestSize_Check_LeafDownEmitsSplitHint confirms that when declared
// size overstates computed (declared > computed), the alternative
// pointer routes to /split rather than "scope grew."
func TestSize_Check_LeafDownEmitsSplitHint(t *testing.T) {
	env := newTestEnv(t)

	// Single-file, tiny spec declared giant — computed will be trivial.
	env.addSpec("planning/features/oversold/spec.md", `---
title: Oversold
type: feature
status: planning
size: giant
---

## Goal
One file, declared giant.

## Changes
- internal/x.go
`)

	out, err := runCmd("size", "--check")
	if err == nil {
		t.Fatal("expected non-zero exit when drift is found")
	}
	if !strings.Contains(out, "'/split oversold'") {
		t.Errorf("expected '/split oversold' alternative for leaf-down drift, got:\n%s", out)
	}
}

// TestSize_Check_ErrorPrintsOnce regression-tests the duplicate-error
// fix from spec size-drift-actionable-output: with SilenceErrors=true
// on sizeCmd, cobra no longer prints "Error: <msg>" on top of main.go's
// print. The CLI test harness only sees the returned error (main.go's
// print path), so this asserts the error returns non-nil with the
// exact "size drift found in N spec(s)" message — and that the
// captured stdout does NOT contain a cobra-style duplicate.
func TestSize_Check_ErrorPrintsOnce(t *testing.T) {
	env := newTestEnv(t)

	// Drifted leaf so --check fails.
	bigBody := strings.Repeat("This spec has a lot of content. ", 100)
	env.addSpec("planning/features/dup-check/spec.md", `---
title: Dup Check
type: feature
status: planning
size: trivial
---

## Goal
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
	if !strings.Contains(err.Error(), "size drift found in") {
		t.Errorf("expected 'size drift found in N spec(s)' error, got: %v", err)
	}
	// Cobra's auto-print prefixes errors with "Error: " on stderr.
	// runCmd captures stdout only — but if SilenceErrors were false,
	// cobra writes to stderr through cmd.ErrOrStderr() which defaults
	// to os.Stderr. We assert the captured stdout has no "Error: "
	// prefix as a belt-and-suspenders check; the real proof is the
	// SilenceErrors=true flag flip on sizeCmd.
	if strings.Contains(out, "Error: size drift") {
		t.Errorf("expected no duplicate cobra 'Error:' print in stdout, got:\n%s", out)
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

func TestSize_Ack_NewField(t *testing.T) {
	env := newTestEnv(t)

	specPath := "planning/features/big-deliberate/spec.md"
	env.addSpec(specPath, `---
title: Big Deliberate
type: feature
status: planning
size: giant
---

## Goal
Intentional giant — acknowledge it.
`)

	_, err := runCmd("size", "--ack", "giant", "big-deliberate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(env.heroDir, specPath))
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "size_ack: giant") {
		t.Errorf("expected 'size_ack: giant' in frontmatter, got:\n%s", content)
	}
	// The original `size:` must still be present.
	if !strings.Contains(content, "size: giant") {
		t.Errorf("expected original 'size: giant' to be preserved, got:\n%s", content)
	}
}

func TestSize_Ack_PreservesOtherFrontmatter(t *testing.T) {
	env := newTestEnv(t)

	specPath := "planning/features/preserve/spec.md"
	env.addSpec(specPath, `---
title: Preserve
type: feature
status: planning
size: giant
priority: P1
---

## Goal
Other frontmatter must survive the ack.
`)

	if _, err := runCmd("size", "--ack", "giant", "preserve"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(env.heroDir, specPath))
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	content := string(data)
	for _, want := range []string{"title: Preserve", "priority: P1", "size: giant", "size_ack: giant"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q to be present, got:\n%s", want, content)
		}
	}
}

func TestSize_Ack_Idempotent(t *testing.T) {
	env := newTestEnv(t)

	specPath := "planning/features/already-acked/spec.md"
	env.addSpec(specPath, `---
title: Already Acked
type: feature
status: planning
size: giant
size_ack: giant
---

## Goal
Re-acking should be a no-op write of the same value.
`)

	if _, err := runCmd("size", "--ack", "giant", "already-acked"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(env.heroDir, specPath))
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	// Exactly one size_ack line.
	if got := strings.Count(string(data), "size_ack:"); got != 1 {
		t.Errorf("expected exactly one size_ack line after re-ack, got %d:\n%s", got, string(data))
	}
}

func TestSize_Ack_RejectsInvalidTier(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/some-spec/spec.md", `---
title: Some Spec
type: feature
status: planning
---

## Goal
Invalid ack tier should be rejected.
`)

	_, err := runCmd("size", "--ack", "enormous", "some-spec")
	if err == nil {
		t.Fatal("expected error for invalid ack tier")
	}
	if !strings.Contains(err.Error(), "invalid size") {
		t.Errorf("expected 'invalid size' error, got: %v", err)
	}
}

func TestSize_Ack_RequiresSlug(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("size", "--ack", "giant")
	if err == nil {
		t.Fatal("expected error when --ack is given without a slug")
	}
	if !strings.Contains(err.Error(), "--ack") {
		t.Errorf("expected usage error mentioning --ack, got: %v", err)
	}
}

func TestSize_Ack_SpecNotFound(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("size", "--ack", "giant", "no-such-spec")
	if err == nil {
		t.Fatal("expected error for missing spec")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestSize_Ack_ConflictsWithCheck(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("size", "--check", "--ack", "giant", "any")
	if err == nil {
		t.Fatal("expected error when --check and --ack are both set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' error, got: %v", err)
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
