package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestPipeline(t *testing.T) {
	env := newTestEnv(t)

	// Imported bug (no investigation)
	env.addSpec("planning/bugs/bug-imported/spec.md", `---
title: Imported Bug
type: bug
status: planning
tracker_id: PROJ-100
---
# Imported Bug

A bug imported from tracker.
`)

	// Diagnosed bug (has investigation)
	env.addSpec("planning/bugs/bug-diagnosed/spec.md", `---
title: Diagnosed Bug
type: bug
status: planning
---
# Diagnosed Bug

## Investigation

Found the root cause in handler.go.

## Root Cause

Null pointer on line 42.

## Suggested Fix Approach

Add nil check before accessing the field.
`)

	// Approved feature (has design)
	env.addSpec("planning/features/feat-approved/spec.md", `---
title: Approved Feature
type: feature
status: planning
---
# Approved Feature

## Design

Use the existing service layer.

## Changes

- src/service.go
`)

	// Delivering feature
	env.addSpec("planning/features/feat-delivering/spec.md", `---
title: Delivering Feature
type: feature
status: delivering
delivery_method: manual
---
# Delivering Feature
`)

	// Completed feature
	env.addSpec("specs/feat-completed/spec.md", `---
title: Completed Feature
type: feature
status: completed
---
# Completed Feature
`)

	env.indexAll()

	output, err := runCmd("pipeline")
	if err != nil {
		t.Fatalf("pipeline returned error: %v", err)
	}

	// Check stage counts
	if !strings.Contains(output, "imported") {
		t.Errorf("should show imported stage: %q", output)
	}
	if !strings.Contains(output, "diagnosed") {
		t.Errorf("should show diagnosed stage: %q", output)
	}
	if !strings.Contains(output, "delivering") {
		t.Errorf("should show delivering stage: %q", output)
	}

	// Check specific specs appear
	if !strings.Contains(output, "Imported Bug") {
		t.Errorf("should show imported bug: %q", output)
	}
	if !strings.Contains(output, "Diagnosed Bug") {
		t.Errorf("should show diagnosed bug: %q", output)
	}
	if !strings.Contains(output, "Approved Feature") {
		t.Errorf("should show approved feature: %q", output)
	}
	if !strings.Contains(output, "manual") {
		t.Errorf("should show manual delivery method: %q", output)
	}
}

func TestPipelineTypeFilter(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/bugs/bug-filter/spec.md", `---
title: Bug Only
type: bug
status: planning
---
# Bug Only
`)
	env.addSpec("planning/features/feat-filter/spec.md", `---
title: Feature Only
type: feature
status: planning
---
# Feature Only
`)

	env.indexAll()

	output, err := runCmd("pipeline", "--type", "bug")
	if err != nil {
		t.Fatalf("pipeline --type bug returned error: %v", err)
	}
	if !strings.Contains(output, "Bug Only") {
		t.Errorf("should show bug: %q", output)
	}
	if strings.Contains(output, "Feature Only") {
		t.Errorf("should not show feature when filtered to bugs: %q", output)
	}
}

func TestPipelineJSON(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/feat-json/spec.md", `---
title: JSON Feature
type: feature
status: planning
---
# JSON Feature

## Design

Something.
`)

	env.indexAll()

	output, err := runCmd("pipeline", "--json")
	if err != nil {
		t.Fatalf("pipeline --json returned error: %v", err)
	}
	if !strings.Contains(output, `"name"`) {
		t.Errorf("should output JSON: %q", output)
	}
	if !strings.Contains(output, `"approved"`) {
		t.Errorf("should have approved stage in JSON: %q", output)
	}
}

func TestPipelineEmpty(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("pipeline")
	if err != nil {
		t.Fatalf("pipeline returned error: %v", err)
	}
	if !strings.Contains(output, "0 specs") {
		t.Errorf("should show 0 specs: %q", output)
	}
}

func TestClassifyReadiness(t *testing.T) {
	tests := []struct {
		name     string
		sections map[string]string
		specType string
		want     string
	}{
		{"bare bug", map[string]string{}, "bug", "imported"},
		{"bug with investigation", map[string]string{"investigation": "found it"}, "bug", "diagnosed"},
		{"bug with fix plan", map[string]string{"suggested fix approach": "add nil check", "changes": "handler.go"}, "bug", "diagnosed"},
		{"feature with design", map[string]string{"design": "use service"}, "feature", "approved"},
		{"feature bare", map[string]string{}, "feature", "imported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &spec.Spec{
				Sections: tt.sections,
				Type:     spec.Type(tt.specType),
			}
			got := classifyReadiness(s)
			if got != tt.want {
				t.Errorf("classifyReadiness = %q, want %q", got, tt.want)
			}
		})
	}
}
