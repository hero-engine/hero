package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSmokeCmd_NoArgs shows help when called with no args and no flags.
func TestSmokeCmd_NoArgs(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("smoke")
	if err != nil {
		t.Fatalf("smoke with no args returned error: %v", err)
	}
	if !strings.Contains(output, "hero smoke") {
		t.Errorf("smoke help output missing expected text: %q", output)
	}
}

// TestSmokeCmd_StatusNoRuns shows the "no runs" message when no last-run.json exists.
func TestSmokeCmd_StatusNoRuns(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("smoke", "status")
	if err != nil {
		t.Fatalf("smoke status returned error: %v", err)
	}
	if !strings.Contains(output, "No smoke runs found") {
		t.Errorf("expected 'No smoke runs found', got: %q", output)
	}
}

// TestSmokeCmd_SlugNotFound errors when the slug doesn't match any spec.
func TestSmokeCmd_SlugNotFound(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("smoke", "nonexistent-feature")
	if err == nil {
		t.Fatal("smoke nonexistent-feature should return error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say not found, got: %v", err)
	}
}

// TestSmokeCmd_SlugDeferred skips gracefully when the spec has smoke: deferred.
func TestSmokeCmd_SlugDeferred(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
smoke: deferred
---
# CSV Export
`)

	output, err := runCmd("smoke", "csv-export")
	if err != nil {
		t.Fatalf("smoke csv-export (deferred) returned error: %v", err)
	}
	if !strings.Contains(output, "deferred") {
		t.Errorf("expected deferred status in output, got: %q", output)
	}
}

// TestSmokeCmd_SlugScriptMissing fails with a clear error when the script file doesn't exist.
func TestSmokeCmd_SlugScriptMissing(t *testing.T) {
	env := newTestEnv(t)

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

	_, err := runCmd("smoke", "csv-export")
	if err == nil {
		t.Fatal("smoke with missing script should return error")
	}
	if !strings.Contains(err.Error(), "smoke failed") {
		t.Errorf("error should mention smoke failed, got: %v", err)
	}
}

// TestSmokeCmd_SlugScriptPasses runs a real script that exits 0 and records pass.
func TestSmokeCmd_SlugScriptPasses(t *testing.T) {
	env := newTestEnv(t)

	// Write a passing smoke script
	scriptDir := filepath.Join(env.dir, "scripts", "smoke")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "csv-export.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'smoke: csv-export OK'\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

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

	output, err := runCmd("smoke", "csv-export")
	if err != nil {
		t.Fatalf("smoke csv-export returned error: %v", err)
	}
	if !strings.Contains(output, "pass") {
		t.Errorf("expected pass in output, got: %q", output)
	}
}

// TestSmokeCmd_SlugScriptFails records fail and returns error when script exits non-zero.
func TestSmokeCmd_SlugScriptFails(t *testing.T) {
	env := newTestEnv(t)

	scriptDir := filepath.Join(env.dir, "scripts", "smoke")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "csv-export.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'FAIL'\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

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

	_, err := runCmd("smoke", "csv-export")
	if err == nil {
		t.Fatal("smoke with failing script should return error")
	}
}

// TestSmokeCmd_StatusAfterRun shows results after a smoke run completes.
func TestSmokeCmd_StatusAfterRun(t *testing.T) {
	env := newTestEnv(t)

	scriptDir := filepath.Join(env.dir, "scripts", "smoke")
	os.MkdirAll(scriptDir, 0o755)
	os.WriteFile(filepath.Join(scriptDir, "csv-export.sh"),
		[]byte("#!/bin/bash\nexit 0\n"), 0o755)

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

	// Run the smoke first
	if _, err := runCmd("smoke", "csv-export"); err != nil {
		t.Fatalf("smoke run failed: %v", err)
	}

	// Now check status
	output, err := runCmd("smoke", "status")
	if err != nil {
		t.Fatalf("smoke status returned error: %v", err)
	}
	if !strings.Contains(output, "csv-export") {
		t.Errorf("status should show csv-export: %q", output)
	}
	if !strings.Contains(output, "pass") {
		t.Errorf("status should show pass: %q", output)
	}
}

// TestSmokeCmd_AllNoScripts reports "nothing to run" when all smokes are deferred.
func TestSmokeCmd_AllNoScripts(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
smoke: deferred
---
# CSV Export
`)

	output, err := runCmd("smoke", "--all")
	if err != nil {
		t.Fatalf("smoke --all returned error: %v", err)
	}
	if !strings.Contains(output, "No feature smokes to run") {
		t.Errorf("expected no-scripts message, got: %q", output)
	}
}

// TestSmokeCmd_AllRunsScripts runs all non-deferred smokes.
func TestSmokeCmd_AllRunsScripts(t *testing.T) {
	env := newTestEnv(t)

	scriptDir := filepath.Join(env.dir, "scripts", "smoke")
	os.MkdirAll(scriptDir, 0o755)
	os.WriteFile(filepath.Join(scriptDir, "feature-a.sh"),
		[]byte("#!/bin/bash\nexit 0\n"), 0o755)
	os.WriteFile(filepath.Join(scriptDir, "feature-b.sh"),
		[]byte("#!/bin/bash\nexit 0\n"), 0o755)

	env.addSpec("planning/features/feature-a/spec.md", `---
title: Feature A
type: feature
status: planning
smoke:
  script: scripts/smoke/feature-a.sh
  runs_on: [nightly]
---
`)
	env.addSpec("planning/features/feature-b/spec.md", `---
title: Feature B
type: feature
status: planning
smoke:
  script: scripts/smoke/feature-b.sh
  runs_on: [nightly]
---
`)

	output, err := runCmd("smoke", "--all")
	if err != nil {
		t.Fatalf("smoke --all returned error: %v", err)
	}
	if !strings.Contains(output, "2 smoke(s) ran") {
		t.Errorf("expected 2 smokes ran, got: %q", output)
	}
	if !strings.Contains(output, "2 passed") {
		t.Errorf("expected 2 passed, got: %q", output)
	}
}

// TestSmokeCmd_SinceNoChanges handles the case where --since finds no changed files.
func TestSmokeCmd_SinceNoChanges(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
smoke:
  script: scripts/smoke/csv-export.sh
  runs_on: [commit-touches:internal/export/*.go, nightly]
---
# CSV Export
`)

	// Use a ref that won't exist — git diff will return nothing (empty output from stderr).
	// The command should handle this gracefully.
	output, err := runCmd("smoke", "--since", "HEAD")
	if err != nil {
		t.Fatalf("smoke --since HEAD returned error: %v", err)
	}
	// HEAD..HEAD has no changes
	if !strings.Contains(output, "No files changed") && !strings.Contains(output, "No smoke scripts triggered") {
		t.Errorf("expected no-changes or no-triggered message, got: %q", output)
	}
}

// TestSmokeCmd_NoWorkspace errors without a .hero directory.
func TestSmokeCmd_NoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("smoke", "--all")
	if err == nil {
		t.Fatal("smoke --all should error without workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace found") {
		t.Errorf("error should mention no workspace: %v", err)
	}
}
