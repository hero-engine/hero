package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	env.addSpec("planning/features/feat-verify/spec.md", `---
title: Verifiable Feature
type: feature
status: delivering
delivery_method: manual
---
# Verifiable Feature

## Goal

Do something verifiable.

## Acceptance Criteria

- API must return JSON with status field
- Response time must be under 200ms
- Error cases must return 4xx status codes

## Changes

- src/api/handler.go
- src/api/handler_test.go

## Test Strategy

- Unit tests for handler logic
- Integration test for full request cycle
`)

	env.indexAll()

	output, err := runCmd("spec", "verify", "feat-verify")
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}

	// Should show acceptance criteria
	if !strings.Contains(output, "API must return JSON") {
		t.Errorf("should show acceptance criteria: %q", output)
	}
	if !strings.Contains(output, "Response time must be under 200ms") {
		t.Errorf("should show all criteria: %q", output)
	}

	// Should show expected files
	if !strings.Contains(output, "src/api/handler.go") {
		t.Errorf("should show expected files: %q", output)
	}

	// Should show test strategy
	if !strings.Contains(output, "Unit tests for handler logic") {
		t.Errorf("should show test strategy: %q", output)
	}

	// Should show verification prompt
	if !strings.Contains(output, "PASS/FAIL") {
		t.Errorf("should include verification instructions: %q", output)
	}
}

func TestVerifyNoAcceptanceCriteria(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-noac/spec.md", `---
title: No AC Feature
type: feature
status: delivering
---
# No AC Feature

Just a description, no acceptance criteria.
`)

	env.indexAll()

	output, err := runCmd("spec", "verify", "feat-noac")
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}

	if !strings.Contains(output, "No acceptance criteria") {
		t.Errorf("should warn about missing AC: %q", output)
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
