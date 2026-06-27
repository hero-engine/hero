package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
)

// captureStderr swaps os.Stderr with a pipe for the duration of the
// returned restore func. Returns a snapshot accessor that reads
// everything written so far. Used by the WIP advisory tests since the
// advisory writes to stderr (warning, not stdout output).
func captureStderr() (read func() string, restore func()) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	return func() string {
			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			os.Stderr = old
			// Re-establish in case caller continues to use stderr —
			// after a single read the file is consumed.
			return buf.String()
		}, func() {
			// Best-effort restore for defer'd callers when read was
			// never invoked.
			w.Close()
			os.Stderr = old
		}
}

func TestDeliverManual(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-manual/spec.md", `---
title: Manual Feature
type: feature
status: approved
---
# Manual Feature

## Goal

Implement something manually.

## Acceptance Criteria

- Must return 200 on success
- Must log all requests
`)

	env.indexAll()

	output, err := runCmd("spec", "deliver", "--manual", "feat-manual")
	if err != nil {
		t.Fatalf("deliver --manual returned error: %v", err)
	}

	if !strings.Contains(output, "Started manual delivery") {
		t.Errorf("unexpected output: %q", output)
	}
	if !strings.Contains(output, "hero verify feat-manual") {
		t.Errorf("should suggest verify command: %q", output)
	}

	// Check frontmatter was updated
	data, err := os.ReadFile(filepath.Join(env.heroDir, "planning", "features", "feat-manual", "spec.md"))
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "status: delivering") {
		t.Errorf("status should be delivering: %s", content)
	}
	if !strings.Contains(content, "delivery_method: manual") {
		t.Errorf("delivery_method should be manual: %s", content)
	}
}

func TestDeliverManualAlreadyDelivering(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-inprog/spec.md", `---
title: In Progress Feature
type: feature
status: delivering
delivery_method: manual
---
# In Progress Feature
`)

	env.indexAll()

	output, err := runCmd("spec", "deliver", "--manual", "feat-inprog")
	if err != nil {
		t.Fatalf("should not error for already-manual spec: %v", err)
	}
	if !strings.Contains(output, "already in manual delivery") {
		t.Errorf("should mention already delivering: %q", output)
	}
}

func TestDeliverManualCompleted(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/feat-done/spec.md", `---
title: Done Feature
type: feature
status: completed
---
# Done Feature
`)

	env.indexAll()

	_, err := runCmd("spec", "deliver", "--manual", "feat-done")
	if err == nil {
		t.Fatal("should error for completed spec")
	}
	if !strings.Contains(err.Error(), "already completed") {
		t.Errorf("error should mention completed: %v", err)
	}
}

func TestDeliverWithoutManualFlag(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/feat-nomanual/spec.md", `---
title: No Manual Feature
type: feature
status: approved
---
# No Manual Feature
`)
	env.indexAll()

	_, err := runCmd("spec", "deliver", "feat-nomanual")
	if err == nil {
		t.Fatal("deliver without --manual should fail")
	}
	if !strings.Contains(err.Error(), "--manual") {
		t.Errorf("error should mention --manual flag: %v", err)
	}
}

func TestVerify(t *testing.T) {
	env := newTestEnv(t)

	// Verify now enforces gates — a spec without a ledger or audit should FAIL.
	env.addSpec("planning/features/feat-verify/spec.md", `---
title: Verifiable Feature
type: feature
status: delivering
delivery_method: manual
---
# Verifiable Feature

## Acceptance Criteria

- API must return JSON with status field

## Changes

- src/api/handler.go
`)

	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "feat-verify")
	if err == nil {
		t.Fatal("verify should fail — no ledger, no audit")
	}
	if !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("error = %q, want 'verification failed'", err.Error())
	}
}

func TestVerifyNoAcceptanceCriteria(t *testing.T) {
	env := newTestEnv(t)

	// A spec with no AC and no ledger should fail gate 1.
	env.addSpec("planning/features/feat-noac/spec.md", `---
title: No AC Feature
type: feature
status: delivering
---
# No AC Feature

Just a description, no acceptance criteria.
`)

	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "feat-noac")
	if err == nil {
		t.Fatal("verify should fail — no ledger, no audit")
	}
	if !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("error = %q, want 'verification failed'", err.Error())
	}
}

func TestVerifyNonexistent(t *testing.T) {
	env := newTestEnv(t)
	env.indexAll()

	_, err := runCmd("spec", "verify", "nonexistent-spec")
	if err == nil {
		t.Fatal("verify nonexistent spec should fail")
	}
}

// TestIsKickoffMissing covers the kickoff gate predicate. The gate
// itself is skipped under `go test` so existing fixtures stay valid;
// the helper carries the lifecycle contract verbatim. Symptom 2 in
// spec-lifecycle-hygiene-breakdown.
func TestIsKickoffMissing(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		missing bool
	}{
		{"empty", "", true},
		{"whitespace only", "  \n\t\n", true},
		{"untouched import stub", "_TODO: write a paste-ready cold-start prompt — see skills/kickoff-prompt/SKILL.md_", true},
		{"real kickoff", "You are picking up the cache invalidation work. Read X then Y.", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isKickoffMissing(c.body); got != c.missing {
				t.Errorf("isKickoffMissing(%q) = %v, want %v", c.body, got, c.missing)
			}
		})
	}
}

// TestPrintWIPAdvisoryAboveThreshold confirms the soft advisory fires
// when in-flight count meets the threshold and stays silent below it.
// Symptom 3 in spec-lifecycle-hygiene-breakdown.
func TestPrintWIPAdvisoryAboveThreshold(t *testing.T) {
	// Build a fake spec slice with 5 delivering specs (plus the target).
	specs := []*spec.Spec{
		{Slug: "target", Status: spec.StatusDelivering},
		{Slug: "a", Status: spec.StatusDelivering},
		{Slug: "b", Status: spec.StatusDelivering},
		{Slug: "c", Status: spec.StatusDelivering},
		{Slug: "d", Status: spec.StatusDelivering},
		{Slug: "e", Status: spec.StatusDelivering},
		{Slug: "done", Status: spec.StatusCompleted},
	}
	cfg := config.Config{}

	stderr, restore := captureStderr()
	defer restore()
	printWIPAdvisory(specs, cfg, "target")
	got := stderr()

	if !strings.Contains(got, "WARNING:") {
		t.Errorf("expected WARNING in stderr, got: %q", got)
	}
	if !strings.Contains(got, "5 specs already in delivery") {
		t.Errorf("expected '5 specs already' count, got: %q", got)
	}
	for _, slug := range []string{"a", "b", "c", "d", "e"} {
		if !strings.Contains(got, "- "+slug) {
			t.Errorf("expected slug %q listed, got: %q", slug, got)
		}
	}
	if strings.Contains(got, "- target") {
		t.Errorf("target slug should be excluded from the in-flight list, got: %q", got)
	}
}

// TestPrintWIPAdvisoryBelowThreshold confirms no warning fires under
// the threshold.
func TestPrintWIPAdvisoryBelowThreshold(t *testing.T) {
	specs := []*spec.Spec{
		{Slug: "target", Status: spec.StatusDelivering},
		{Slug: "a", Status: spec.StatusDelivering},
		{Slug: "b", Status: spec.StatusDelivering},
	}
	cfg := config.Config{}

	stderr, restore := captureStderr()
	defer restore()
	printWIPAdvisory(specs, cfg, "target")
	got := stderr()

	if got != "" {
		t.Errorf("expected no advisory below threshold, got: %q", got)
	}
}

// TestPrintWIPAdvisoryConfigurableThreshold confirms cfg.Delivery
// overrides the default-5.
func TestPrintWIPAdvisoryConfigurableThreshold(t *testing.T) {
	specs := []*spec.Spec{
		{Slug: "a", Status: spec.StatusDelivering},
		{Slug: "b", Status: spec.StatusDelivering},
	}
	cfg := config.Config{Delivery: &config.DeliveryConfig{WIPWarningThreshold: 2}}

	stderr, restore := captureStderr()
	defer restore()
	printWIPAdvisory(specs, cfg, "target")
	got := stderr()

	if !strings.Contains(got, "threshold 2") {
		t.Errorf("expected custom threshold reflected, got: %q", got)
	}
}

// TestImportScaffoldsKickoffPlaceholder confirms generateImportedSpec
// always writes a `## Kickoff` section so freshly-imported specs trip
// the gate with an actionable TODO rather than being silently
// kickoff-less. Symptom 2 in spec-lifecycle-hygiene-breakdown.
func TestImportScaffoldsKickoffPlaceholder(t *testing.T) {
	issue := tracker.Issue{
		ID:    "PROJ-1",
		Title: "Fix login",
	}
	body := generateImportedSpec(issue, "bug", "jira", "proj-1-fix-login")
	if !strings.Contains(body, "## Kickoff") {
		t.Errorf("imported spec missing `## Kickoff` section: %q", body)
	}
	if !strings.Contains(body, "TODO: write a paste-ready cold-start prompt") {
		t.Errorf("imported kickoff missing TODO stub: %q", body)
	}
	// The TODO stub must trip the gate so it's not silently passable.
	// We isolate the kickoff section by splitting on the next header.
	kickoff := body[strings.Index(body, "## Kickoff")+len("## Kickoff"):]
	if idx := strings.Index(kickoff, "\n## "); idx >= 0 {
		kickoff = kickoff[:idx]
	}
	if !isKickoffMissing(kickoff) {
		t.Error("imported kickoff stub should still trip isKickoffMissing")
	}
}

// TestDeliverManualAutoArchivesOnVerify exercises the full sync deliver
// → completed → archive cycle without anyone running `hero spec complete`.
// With gated verify, the agent no longer flips status directly.
// hero verify checks gates and flips status + archives when all pass.
// This test confirms verify with all gates satisfied archives correctly.
func TestDeliverManualAutoArchivesOnVerify(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-auto/spec.md", `---
title: Auto Archive Feature
type: feature
status: delivering
---
# Auto Archive Feature

## Kickoff

Pick up here.

## Acceptance Criteria

- AC-1: Must work

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Must work | DONE | implemented |

### Exercise-the-feature check

- [x] Exercised: ran the feature, confirmed working
`)

	// Write audit report
	writeFile(t, filepath.Join(env.heroDir, "planning/features/feat-auto/delivery-audit.md"), `# Delivery audit — feat-auto

**Verdict:** SHIP
**Surface:** clean
`)
	env.indexAll()

	output, err := runCmd("spec", "verify", "--skip-tests", "feat-auto")
	if err != nil {
		t.Fatalf("verify: %v\noutput: %s", err, output)
	}

	// Spec must now live under specs/feat-auto/spec.md.
	destPath := filepath.Join(env.heroDir, "specs", "feat-auto", "spec.md")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("spec not moved to %s", destPath)
	}
}

// TestVerifyDoesNotArchiveIncomplete confirms that verify refuses to
// archive a spec that fails gates.
func TestVerifyDoesNotArchiveIncomplete(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-inprog/spec.md", `---
title: In-Progress
type: feature
status: delivering
---
# In-Progress

## Acceptance Criteria

- Whatever
`)
	env.indexAll()

	_, err := runCmd("spec", "verify", "--skip-tests", "feat-inprog")
	if err == nil {
		t.Fatal("verify should fail — no ledger, no audit")
	}

	specPath := filepath.Join(env.heroDir, "planning/features/feat-inprog/spec.md")
	if _, err := os.Stat(specPath); err != nil {
		t.Errorf("spec should still be in planning/: %v", err)
	}
	archivedPath := filepath.Join(env.heroDir, "specs", "feat-inprog", "spec.md")
	if _, err := os.Stat(archivedPath); !os.IsNotExist(err) {
		t.Error("spec should NOT be archived when gates fail")
	}
}

func TestExtractCriteriaItems(t *testing.T) {
	section := `
- First item
- Second item
* Third item
1. Fourth item
2) Fifth item
Not a list item
`
	items := extractCriteriaItems(section)
	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d: %v", len(items), items)
	}
	if items[0] != "First item" {
		t.Errorf("first item = %q", items[0])
	}
}

func TestDeliverRejectsInitiativeWithDrivePointer(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/initiatives/big-thing/spec.md", `---
title: Big Thing
type: initiative
status: planning
---
# Big Thing

## Goal

Run all the children.
`)
	_, err := runCmd("spec", "deliver", "--manual", "big-thing")
	if err == nil {
		t.Fatal("delivering an initiative should error, not flip it to delivering")
	}
	if !strings.Contains(err.Error(), "/drive") {
		t.Errorf("error should point the user to /drive, got: %v", err)
	}
}
