package cli

import (
	"strings"
	"testing"
)

func TestDiagnose_Single(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/bugs/crash-on-login/spec.md", `---
title: App crashes on login
type: bug
status: planning
tracker_id: BUG-101
---

## Description

The app crashes when the user clicks login.

## Steps to Reproduce

1. Open the app
2. Click login
3. App crashes
`)

	out, err := runCmd("spec", "diagnose", "crash-on-login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "App crashes on login") {
		t.Error("expected output to contain spec title")
	}
	if !strings.Contains(out, "BUG-101") {
		t.Error("expected output to contain tracker ID")
	}
	if !strings.Contains(out, "Investigation Instructions") {
		t.Error("expected output to contain investigation instructions")
	}
	if !strings.Contains(out, "Stage: imported") {
		t.Error("expected stage to be 'imported'")
	}
}

func TestDiagnose_Single_JSON(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/bugs/crash-on-login/spec.md", `---
title: App crashes on login
type: bug
status: planning
---

## Description

Crash on login.
`)

	out, err := runCmd("spec", "diagnose", "--json", "crash-on-login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, `"slug": "crash-on-login"`) {
		t.Error("expected JSON output with slug")
	}
	if !strings.Contains(out, `"stage": "imported"`) {
		t.Error("expected JSON output with stage")
	}
}

func TestDiagnose_NotBug(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/add-export/spec.md", `---
title: Add export feature
type: feature
status: planning
---

## Description

Add CSV export.
`)

	_, err := runCmd("spec", "diagnose", "add-export")
	if err == nil {
		t.Fatal("expected error for non-bug spec")
	}
	if !strings.Contains(err.Error(), "not bug") {
		t.Errorf("expected 'not bug' error, got: %v", err)
	}
}

func TestDiagnose_Completed(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/old-bug/spec.md", `---
title: Old bug
type: bug
status: completed
---

## Description

Already fixed.
`)

	_, err := runCmd("spec", "diagnose", "old-bug")
	if err == nil {
		t.Fatal("expected error for completed spec")
	}
	if !strings.Contains(err.Error(), "already completed") {
		t.Errorf("expected 'already completed' error, got: %v", err)
	}
}

func TestDiagnose_NotFound(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("spec", "diagnose", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing spec")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestDiagnose_Batch(t *testing.T) {
	env := newTestEnv(t)

	// Imported bug (should appear)
	env.addSpec("planning/bugs/crash-on-login/spec.md", `---
title: App crashes on login
type: bug
status: planning
tracker_id: BUG-101
---

## Description

Crash on login.
`)

	// Diagnosed bug (should NOT appear)
	env.addSpec("planning/bugs/timeout-issue/spec.md", `---
title: API timeout
type: bug
status: planning
---

## Description

Timeout on API call.

## Investigation

Found that the timeout is set to 1ms.

## Root Cause

Timeout too low.
`)

	// Feature (should NOT appear)
	env.addSpec("planning/features/add-export/spec.md", `---
title: Add export
type: feature
status: planning
---

## Description

Export feature.
`)

	out, err := runCmd("spec", "diagnose", "--batch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Undiagnosed Bugs (1)") {
		t.Errorf("expected 1 undiagnosed bug, got output: %s", out)
	}
	if !strings.Contains(out, "App crashes on login") {
		t.Error("expected crash-on-login in batch output")
	}
	if strings.Contains(out, "API timeout") {
		t.Error("diagnosed bug should not appear in batch list")
	}
	if strings.Contains(out, "Add export") {
		t.Error("feature should not appear in batch list")
	}
}

func TestDiagnose_Batch_Empty(t *testing.T) {
	_ = newTestEnv(t)

	out, err := runCmd("spec", "diagnose", "--batch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "No undiagnosed bugs") {
		t.Error("expected 'No undiagnosed bugs' message")
	}
}

func TestDiagnose_NoArgs(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("spec", "diagnose")
	if err == nil {
		t.Fatal("expected error when no slug and no --batch")
	}
	if !strings.Contains(err.Error(), "specify a spec slug") {
		t.Errorf("expected helpful error, got: %v", err)
	}
}
