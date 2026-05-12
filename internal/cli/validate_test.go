package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestValidate_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("check", "validate")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestValidate_Empty(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "No specs found") {
		t.Errorf("expected 'No specs found', got: %s", output)
	}
}

func TestValidate_AllValid(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
smoke: deferred
---
# CSV Export

## Goal

Export data.
`)

	env.addSpec("conventions/error-handling/spec.md", `---
title: Error Handling
type: convention
status: active
---
# Error Handling

## Rule

Handle all errors.
`)

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "All 2 specs are valid") {
		t.Errorf("expected all valid, got: %s", output)
	}
}

func TestValidate_BrokenRelation(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
depends-on: nonexistent-spec
---
# CSV Export
`)

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "nonexistent-spec") {
		t.Errorf("expected broken relation warning, got: %s", output)
	}
	if !strings.Contains(output, "not found") {
		t.Errorf("expected 'not found' message, got: %s", output)
	}
}

func TestValidate_InvalidStatus(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: bogus
---
# CSV Export
`)

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "invalid status") {
		t.Errorf("expected invalid status warning, got: %s", output)
	}
}

func TestValidate_ConventionWrongStatus(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("conventions/error-handling/spec.md", `---
title: Error Handling
type: convention
status: planning
---
# Error Handling
`)

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "convention should use") {
		t.Errorf("expected convention status mismatch warning, got: %s", output)
	}
}

func TestValidate_MissingFile(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export

## Changes

- internal/export/csv.go
`)

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "file not found") {
		t.Errorf("expected 'file not found' warning, got: %s", output)
	}
}

// --- validateSpec unit tests ---

func TestValidateSpec_MissingFields(t *testing.T) {
	s := &spec.Spec{}
	issues := validateSpec(s, map[string]bool{}, "/tmp")

	found := 0
	for _, issue := range issues {
		if strings.Contains(issue, "missing title") || strings.Contains(issue, "missing type") || strings.Contains(issue, "missing status") {
			found++
		}
	}
	if found != 3 {
		t.Errorf("expected 3 missing field issues, found %d in %v", found, issues)
	}
}

func TestValidateSpec_ValidRelation(t *testing.T) {
	s := &spec.Spec{
		Title:  "Test",
		Type:   spec.TypeFeature,
		Status: spec.StatusPlanning,
		Relations: []spec.Relation{
			{Target: "other-spec", Kind: "depends-on"},
		},
	}

	slugIndex := map[string]bool{"other-spec": true}
	issues := validateSpec(s, slugIndex, "/tmp")

	for _, issue := range issues {
		if strings.Contains(issue, "not found") {
			t.Errorf("should not flag valid relation, got: %s", issue)
		}
	}
}

func TestValidate_MissingSmoke(t *testing.T) {
	env := newTestEnv(t)

	// Feature spec without smoke: field — should be flagged
	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "missing smoke:") {
		t.Errorf("expected missing smoke: warning for feature without smoke field, got: %s", output)
	}
}

func TestValidate_SmokeDeferred(t *testing.T) {
	env := newTestEnv(t)

	// Feature spec with smoke: deferred — should NOT be flagged
	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
smoke: deferred
---
# CSV Export
`)

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(output, "missing smoke:") {
		t.Errorf("smoke: deferred should not be flagged, got: %s", output)
	}
	if !strings.Contains(output, "All 1 specs are valid") {
		t.Errorf("expected all valid for deferred smoke, got: %s", output)
	}
}

func TestValidate_SmokeBlock(t *testing.T) {
	env := newTestEnv(t)

	// Feature spec with full smoke: block — should NOT be flagged
	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
smoke:
  script: scripts/smoke/csv-export.sh
  expects: [csv-export:AC-1]
  runs_on: [nightly]
---
# CSV Export
`)

	output, err := runCmd("check", "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(output, "missing smoke:") {
		t.Errorf("full smoke: block should not be flagged, got: %s", output)
	}
}

func TestValidateSpec_MissingSmoke(t *testing.T) {
	// Work spec without smoke should be flagged
	s := &spec.Spec{
		Title:  "Some Feature",
		Type:   spec.TypeFeature,
		Status: spec.StatusPlanning,
	}
	issues := validateSpec(s, map[string]bool{}, "/tmp")
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "missing smoke:") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing smoke: issue for feature without smoke field, got: %v", issues)
	}
}

func TestValidateSpec_SmokeDeferred_NoFlag(t *testing.T) {
	// Work spec with smoke: deferred should not be flagged
	s := &spec.Spec{
		Title:  "Some Feature",
		Type:   spec.TypeFeature,
		Status: spec.StatusPlanning,
		Smoke:  &spec.SmokeConfig{Deferred: true},
	}
	issues := validateSpec(s, map[string]bool{}, "/tmp")
	for _, issue := range issues {
		if strings.Contains(issue, "missing smoke:") {
			t.Errorf("smoke: deferred should not produce a smoke: issue, got: %s", issue)
		}
	}
}

func TestValidateSpec_ConventionNoSmoke_OK(t *testing.T) {
	// Convention specs (knowledge) are exempt from the smoke: requirement
	s := &spec.Spec{
		Title:  "Error Handling",
		Type:   spec.TypeConvention,
		Status: spec.StatusActive,
	}
	issues := validateSpec(s, map[string]bool{}, "/tmp")
	for _, issue := range issues {
		if strings.Contains(issue, "missing smoke:") {
			t.Errorf("convention spec should not require smoke:, got: %s", issue)
		}
	}
}
