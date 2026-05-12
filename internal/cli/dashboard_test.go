package cli

import (
	"strings"
	"testing"
)

func TestDashboard_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("dashboard")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestDashboard_Empty(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	output, err := runCmd("dashboard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Hero Dashboard") {
		t.Error("expected dashboard header")
	}
	if !strings.Contains(output, "Total: 0 specs") {
		t.Errorf("expected 0 specs, got: %s", output)
	}
	if !strings.Contains(output, "Tracker: not configured") {
		t.Error("expected tracker not configured message")
	}
}

func TestDashboard_WithSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
claimed_by: alice
---
# CSV Export
`)

	env.addSpec("planning/bugs/null-pointer/spec.md", `---
title: Null Pointer
type: bug
status: delivering
tracker_id: "42"
---
# Null Pointer
`)

	env.addSpec("specs/auth/spec.md", `---
title: Auth
type: feature
status: completed
---
# Auth
`)

	env.addSpec("knowledge/conventions/error-handling/spec.md", `---
title: Error Handling
type: convention
status: active
---
# Error Handling
`)

	output, err := runCmd("dashboard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Hero Dashboard") {
		t.Error("expected dashboard header")
	}
	if !strings.Contains(output, "Planning") {
		t.Error("expected Planning section")
	}
	if !strings.Contains(output, "Delivering") {
		t.Error("expected Delivering section")
	}
	if !strings.Contains(output, "Completed") {
		t.Error("expected Completed section")
	}
	if !strings.Contains(output, "Total: 4 specs (2 in-flight)") {
		t.Errorf("expected correct totals, got: %s", output)
	}
	if !strings.Contains(output, "alice") {
		t.Error("expected claimed assignment to show")
	}
}

func TestDashboard_WithTracker(t *testing.T) {
	env := newTestEnv(t)

	writeTrackerConfig(env, "github", "acme/widgets")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
tracker_id: "42"
---
# CSV Export
`)

	output, err := runCmd("dashboard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Tracker: github (acme/widgets)") {
		t.Errorf("expected tracker info, got: %s", output)
	}
	if !strings.Contains(output, "Linked: 1/1") {
		t.Errorf("expected linked count, got: %s", output)
	}
}
