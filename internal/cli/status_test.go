package cli

import (
	"strings"
	"testing"
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
