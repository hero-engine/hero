package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComplete_RequiresArg(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("spec", "complete")
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestComplete_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("spec", "complete", "some/spec.md")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestComplete_InvalidSpecPath(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("spec", "complete", "/nonexistent/spec.md")
	if err == nil {
		t.Fatal("expected error for invalid spec path")
	}
	if !strings.Contains(err.Error(), "parsing spec") {
		t.Errorf("error = %q, want 'parsing spec'", err.Error())
	}
}

func TestComplete_ConventionSpecRejected(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("knowledge/conventions/error-handling/spec.md", `---
title: Error Handling
type: convention
status: active
---
# Error Handling
`)

	specPath := filepath.Join(env.heroDir, "knowledge/conventions/error-handling/spec.md")
	_, err := runCmd("spec", "complete", specPath)
	if err == nil {
		t.Fatal("expected error for convention spec")
	}
	if !strings.Contains(err.Error(), "convention") {
		t.Errorf("error = %q, want mention of 'convention'", err.Error())
	}
}

func TestComplete_DecisionSpecRejected(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("knowledge/decisions/use-postgres/spec.md", `---
title: Use Postgres
type: decision
status: accepted
---
# Use Postgres
`)

	specPath := filepath.Join(env.heroDir, "knowledge/decisions/use-postgres/spec.md")
	_, err := runCmd("spec", "complete", specPath)
	if err == nil {
		t.Fatal("expected error for decision spec")
	}
	if !strings.Contains(err.Error(), "decision") {
		t.Errorf("error = %q, want mention of 'decision'", err.Error())
	}
}

// A spec that's already completed AND already moved to specs/ —
// nothing to do, exit 0 with a friendly message rather than erroring.
// Spec: hero-spec-complete-idempotent-move (the older "already
// completed" error contract was the bug that stranded specs in
// planning/ when status flipped before the move).
func TestComplete_FullyComplete_NoOp(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/csv-export/spec.md", `---
title: CSV Export
type: feature
status: completed
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "specs/csv-export/spec.md")
	output, err := runCmd("spec", "complete", specPath)
	if err != nil {
		t.Fatalf("expected no error when spec is fully complete, got: %v", err)
	}
	if !strings.Contains(output, "nothing to do") {
		t.Errorf("expected 'nothing to do' message, got: %s", output)
	}
}

// A spec stranded in planning/ with status already flipped to
// completed (e.g. by /deliver or hero check --reconcile) — the
// command must move it instead of refusing. This is the bug the
// idempotent-move spec fixes.
func TestComplete_StatusFlippedNoMove(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: completed
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	output, err := runCmd("spec", "complete", specPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(output, "Status was already completed") {
		t.Errorf("expected acknowledgment of pre-flipped status, got: %s", output)
	}
	if !strings.Contains(output, "Moved") {
		t.Errorf("expected move to run when spec is in planning/, got: %s", output)
	}

	destPath := filepath.Join(env.heroDir, "specs", "csv-export", "spec.md")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("spec should have been moved to %s", destPath)
	}
	if _, err := os.Stat(specPath); !os.IsNotExist(err) {
		t.Error("source spec should be removed after move")
	}
}

func TestComplete_FeatureSpec(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export

## Goal

Export data to CSV format.
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	output, err := runCmd("spec", "complete", specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check output messages
	if !strings.Contains(output, "Updated status to completed") {
		t.Errorf("output should mention status update, got: %s", output)
	}
	if !strings.Contains(output, "Moved") {
		t.Errorf("output should mention move, got: %s", output)
	}
	if !strings.Contains(output, "Re-indexed") {
		t.Errorf("output should mention re-index, got: %s", output)
	}
	if !strings.Contains(output, "Completed spec: csv-export") {
		t.Errorf("output should show completed slug, got: %s", output)
	}

	// Verify spec was moved to specs/
	destPath := filepath.Join(env.heroDir, "specs", "csv-export", "spec.md")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("spec should exist at %s", destPath)
	}

	// Verify source is gone
	if _, err := os.Stat(specPath); !os.IsNotExist(err) {
		t.Error("source spec should be removed after move")
	}

	// Verify status was updated in the file
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading moved spec: %v", err)
	}
	if !strings.Contains(string(data), "status: completed") {
		t.Errorf("spec should have status: completed, got:\n%s", string(data))
	}
}

func TestComplete_BugSpec(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/bugs/null-pointer/spec.md", `---
title: Null Pointer Fix
type: bug
status: delivering
---
# Null Pointer Fix

## Goal

Fix null pointer in login flow.
`)

	specPath := filepath.Join(env.heroDir, "planning/bugs/null-pointer/spec.md")
	output, err := runCmd("spec", "complete", specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Completed spec: null-pointer") {
		t.Errorf("output should show completed slug, got: %s", output)
	}

	// Verify moved to specs/
	destPath := filepath.Join(env.heroDir, "specs", "null-pointer", "spec.md")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("spec should exist at %s", destPath)
	}
}

func TestComplete_InitiativeSpec(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/initiatives/v2-migration/spec.md", `---
title: V2 Migration
type: initiative
status: planning
---
# V2 Migration

## Goal

Migrate to V2 architecture.
`)

	specPath := filepath.Join(env.heroDir, "planning/initiatives/v2-migration/spec.md")
	output, err := runCmd("spec", "complete", specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Completed spec: v2-migration") {
		t.Errorf("output should show completed slug, got: %s", output)
	}
}

func TestComplete_SpecAlreadyInSpecs(t *testing.T) {
	env := newTestEnv(t)

	// A spec that's in specs/ but not yet marked completed (e.g. was moved manually)
	env.addSpec("specs/csv-export/spec.md", `---
title: CSV Export
type: feature
status: delivering
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "specs/csv-export/spec.md")
	output, err := runCmd("spec", "complete", specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should update status but NOT report a move
	if !strings.Contains(output, "Updated status to completed") {
		t.Errorf("output should mention status update, got: %s", output)
	}
	if strings.Contains(output, "Moved") {
		t.Errorf("output should NOT mention move for spec already in specs/, got: %s", output)
	}

	// Verify status was updated
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("reading spec: %v", err)
	}
	if !strings.Contains(string(data), "status: completed") {
		t.Errorf("spec should have status: completed, got:\n%s", string(data))
	}
}

func TestComplete_StatusUpdatePreservesContent(t *testing.T) {
	env := newTestEnv(t)

	originalContent := `---
title: CSV Export
type: feature
status: planning
tags: [data, export]
claimed_by: alice
---
# CSV Export

## Goal

Export data to CSV format.

## Changes

- internal/export/csv.go
`

	env.addSpec("planning/features/csv-export/spec.md", originalContent)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("spec", "complete", specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	destPath := filepath.Join(env.heroDir, "specs", "csv-export", "spec.md")
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading moved spec: %v", err)
	}
	content := string(data)

	// Verify all original content is preserved
	if !strings.Contains(content, "title: CSV Export") {
		t.Error("title should be preserved")
	}
	if !strings.Contains(content, "tags: [data, export]") {
		t.Error("tags should be preserved")
	}
	if !strings.Contains(content, "claimed_by: alice") {
		t.Error("claimed_by should be preserved")
	}
	if !strings.Contains(content, "## Goal") {
		t.Error("Goal section should be preserved")
	}
	if !strings.Contains(content, "## Changes") {
		t.Error("Changes section should be preserved")
	}
	if !strings.Contains(content, "status: completed") {
		t.Error("status should be updated to completed")
	}
	if strings.Contains(content, "status: planning") {
		t.Error("old status should be replaced")
	}
}

func TestComplete_EmptyPlanningDirCleaned(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("spec", "complete", specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The csv-export directory under planning should be removed
	slugDir := filepath.Join(env.heroDir, "planning/features/csv-export")
	if _, err := os.Stat(slugDir); !os.IsNotExist(err) {
		t.Errorf("empty slug directory should be cleaned up: %s", slugDir)
	}
}

// --- updateFrontmatterStatus tests ---

func TestUpdateFrontmatterStatus(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")

	content := "---\ntitle: Test\ntype: feature\nstatus: planning\n---\n# Test\n"
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := updateFrontmatterStatus(specPath, "completed"); err != nil {
		t.Fatalf("updateFrontmatterStatus failed: %v", err)
	}

	data, _ := os.ReadFile(specPath)
	if !strings.Contains(string(data), "status: completed") {
		t.Errorf("expected status: completed, got:\n%s", string(data))
	}
	if strings.Contains(string(data), "status: planning") {
		t.Error("old status should be replaced")
	}
}

// --- moveToSpecs tests ---

func TestMoveToSpecs_NotUnderPlanning(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(filepath.Join(heroDir, "specs", "my-spec"), 0o755)

	specPath := filepath.Join(heroDir, "specs", "my-spec", "spec.md")
	os.WriteFile(specPath, []byte("# Test"), 0o644)

	result, moved, err := moveToSpecs(specPath, heroDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if moved {
		t.Error("should not move spec that's not under planning/")
	}
	if result != specPath {
		t.Errorf("result path should be unchanged, got %s", result)
	}
}

func TestMoveToSpecs_DestinationExists(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(filepath.Join(heroDir, "planning", "features", "csv-export"), 0o755)
	os.MkdirAll(filepath.Join(heroDir, "specs", "csv-export"), 0o755)

	srcPath := filepath.Join(heroDir, "planning", "features", "csv-export", "spec.md")
	destPath := filepath.Join(heroDir, "specs", "csv-export", "spec.md")

	os.WriteFile(srcPath, []byte("# Source"), 0o644)
	os.WriteFile(destPath, []byte("# Destination"), 0o644)

	_, _, err := moveToSpecs(srcPath, heroDir)
	if err == nil {
		t.Fatal("expected error when destination already exists")
	}
	if !strings.Contains(err.Error(), "destination already exists") {
		t.Errorf("error = %q, want 'destination already exists'", err.Error())
	}
}
