package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStatusEmpty(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	if !strings.Contains(output, "Specs: (none)") {
		t.Errorf("status should show (none) for empty workspace: %q", output)
	}

	if !strings.Contains(output, "0 in-flight, 0 completed, 0 knowledge entries") {
		t.Errorf("status should show zero totals: %q", output)
	}
}

func TestStatusWithSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/add-export/spec.md", `---
title: Add Export
type: feature
status: planning
---
# Add Export
`)

	env.addSpec("planning/bugs/fix-login/spec.md", `---
title: Fix Login
type: bug
status: delivering
claimed_by: alice
---
# Fix Login
`)

	env.addSpec("knowledge/conventions/error-handling/spec.md", `---
title: Error Handling
type: convention
status: active
---
# Error Handling
`)

	env.addSpec("knowledge/decisions/use-postgres/spec.md", `---
title: Use PostgreSQL
type: decision
status: accepted
---
# Use PostgreSQL
`)

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	// Should show delivering section
	if !strings.Contains(output, "Delivering") {
		t.Error("status missing 'Delivering' section")
	}

	// Should show fix-login in delivering
	if !strings.Contains(output, "fix-login") {
		t.Error("status missing fix-login spec")
	}

	// Should show planning section
	if !strings.Contains(output, "Planning") {
		t.Error("status missing 'Planning' section")
	}

	// Should show conventions
	if !strings.Contains(output, "Conventions") {
		t.Error("status missing conventions section")
	}

	// Should show decisions
	if !strings.Contains(output, "Decisions") {
		t.Error("status missing decisions section")
	}

	// Should show summary line with in-flight counts
	if !strings.Contains(output, "in-flight") {
		t.Error("status missing summary line with in-flight counts")
	}
}

// TestStatus_SurfacesSmokeFailures verifies per-feature-smoke-coverage AC-6:
// hero status surfaces failed smokes in its default output.
func TestStatus_SurfacesSmokeFailures(t *testing.T) {
	env := newTestEnv(t)

	// Seed a last-run.json with one failed and one passed smoke.
	smokeDir := filepath.Join(env.heroDir, "smoke")
	if err := os.MkdirAll(smokeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll smoke dir: %v", err)
	}
	records := []SmokeRunRecord{
		{Slug: "failing-feature", Status: "fail", Timestamp: time.Now(), DurationMS: 250, Error: "smoke script exited 1"},
		{Slug: "passing-feature", Status: "pass", Timestamp: time.Now(), DurationMS: 120},
	}
	data, _ := json.Marshal(records)
	if err := os.WriteFile(filepath.Join(smokeDir, "last-run.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	if !strings.Contains(output, "Smoke failures") {
		t.Errorf("expected 'Smoke failures' section in output; got:\n%s", output)
	}
	if !strings.Contains(output, "failing-feature") {
		t.Errorf("expected failing-feature in smoke failures; got:\n%s", output)
	}
	if strings.Contains(output, "passing-feature") {
		t.Errorf("passing-feature should not appear in smoke failures; got:\n%s", output)
	}
}

// TestStatus_NoSmokeFailuresSilent verifies that when all smokes pass,
// no smoke failure section is rendered.
func TestStatus_NoSmokeFailuresSilent(t *testing.T) {
	env := newTestEnv(t)

	smokeDir := filepath.Join(env.heroDir, "smoke")
	if err := os.MkdirAll(smokeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll smoke dir: %v", err)
	}
	records := []SmokeRunRecord{
		{Slug: "all-good", Status: "pass", Timestamp: time.Now(), DurationMS: 100},
	}
	data, _ := json.Marshal(records)
	if err := os.WriteFile(filepath.Join(smokeDir, "last-run.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if strings.Contains(output, "Smoke failures") {
		t.Errorf("no smoke failures expected; got:\n%s", output)
	}
}

func TestStatusNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("status")
	if err == nil {
		t.Fatal("status should error without workspace")
	}

	if !strings.Contains(err.Error(), "no hero workspace found") {
		t.Errorf("error should mention no workspace: %v", err)
	}
}
