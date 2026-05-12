package reconcile

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

// initGitRepo creates a temporary git repo with a .hero workspace and initial commit.
func initGitRepo(t *testing.T) (projectRoot, heroDir string) {
	t.Helper()
	dir := t.TempDir()
	heroDir = filepath.Join(dir, ".hero")

	// Create hero workspace structure
	for _, sub := range []string{
		filepath.Join(heroDir, "planning", "features"),
		filepath.Join(heroDir, "planning", "bugs"),
		filepath.Join(heroDir, "specs"),
		filepath.Join(heroDir, "knowledge", "conventions"),
	} {
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Write hero.json
	if err := os.WriteFile(filepath.Join(dir, "hero.json"), []byte(`{"folder":".hero"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Init git
	gitRun(t, dir, "init", "-b", "main")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "test")

	// Initial commit with project structure
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial commit")

	return dir, heroDir
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func addSpec(t *testing.T, heroDir, relPath, content string) {
	t.Helper()
	path := filepath.Join(heroDir, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReconcile_NoGitRepo(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	findings := Reconcile(heroDir, dir)
	if len(findings) != 0 {
		t.Errorf("expected no findings for non-git dir, got %d", len(findings))
	}
}

func TestReconcile_PlanningWithModifiedFiles(t *testing.T) {
	projectRoot, heroDir := initGitRepo(t)

	// Create a spec in planning status that references src/main.go
	addSpec(t, heroDir, "planning/features/add-export/spec.md", `---
title: Add Export
type: feature
status: planning
---
# Add Export

## Changes

- `+"`src/main.go`"+` — main entry point
`)

	// Create a feature branch and modify src/main.go
	gitRun(t, projectRoot, "checkout", "-b", "feature/add-export")

	srcDir := filepath.Join(projectRoot, "src")
	os.MkdirAll(srcDir, 0o755)
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, projectRoot, "add", ".")
	gitRun(t, projectRoot, "commit", "-m", "implement export")

	findings := Reconcile(heroDir, projectRoot)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.CurrentStatus != spec.StatusPlanning {
		t.Errorf("expected current status planning, got %s", f.CurrentStatus)
	}
	if f.SuggestedStatus != spec.StatusDelivering {
		t.Errorf("expected suggested status delivering, got %s", f.SuggestedStatus)
	}
	if !f.CanAutoFix() {
		t.Error("planning → delivering should be auto-fixable")
	}
}

func TestReconcile_PlanningWithUncommittedFiles(t *testing.T) {
	projectRoot, heroDir := initGitRepo(t)

	// Spec references a file
	addSpec(t, heroDir, "planning/features/fix-bug/spec.md", `---
title: Fix Bug
type: bug
status: planning
---
# Fix Bug

## Changes

- `+"`lib/util.go`"+` — utility fix
`)

	gitRun(t, projectRoot, "add", ".")
	gitRun(t, projectRoot, "commit", "-m", "add spec")

	// Create the file as an uncommitted change (not staged)
	libDir := filepath.Join(projectRoot, "lib")
	os.MkdirAll(libDir, 0o755)
	if err := os.WriteFile(filepath.Join(libDir, "util.go"), []byte("package lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	findings := Reconcile(heroDir, projectRoot)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	if findings[0].SuggestedStatus != spec.StatusDelivering {
		t.Errorf("expected delivering, got %s", findings[0].SuggestedStatus)
	}
}

func TestReconcile_PlanningWithClaim(t *testing.T) {
	projectRoot, heroDir := initGitRepo(t)

	// Spec is claimed but still planning (no files touched)
	addSpec(t, heroDir, "planning/features/claimed-spec/spec.md", `---
title: Claimed Feature
type: feature
status: planning
claimed_by: alice
---
# Claimed Feature

## Changes

- `+"`internal/foo.go`"+` — new module
`)

	gitRun(t, projectRoot, "add", ".")
	gitRun(t, projectRoot, "commit", "-m", "add spec")

	// The file doesn't exist and hasn't been touched — but spec is claimed
	findings := Reconcile(heroDir, projectRoot)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for claimed spec, got %d", len(findings))
	}

	if findings[0].Evidence == "" {
		t.Error("expected evidence about claim")
	}
}

func TestReconcile_DeliveringSkipped(t *testing.T) {
	projectRoot, heroDir := initGitRepo(t)

	// Spec already in delivering — no finding expected
	addSpec(t, heroDir, "planning/features/in-progress/spec.md", `---
title: In Progress
type: feature
status: delivering
---
# In Progress

## Changes

- `+"`internal/bar.go`"+`
`)

	gitRun(t, projectRoot, "add", ".")
	gitRun(t, projectRoot, "commit", "-m", "add spec")

	findings := Reconcile(heroDir, projectRoot)

	if len(findings) != 0 {
		t.Errorf("delivering spec should produce no findings, got %d", len(findings))
	}
}

func TestReconcile_CompletedSkipped(t *testing.T) {
	projectRoot, heroDir := initGitRepo(t)

	addSpec(t, heroDir, "specs/done-feature/spec.md", `---
title: Done Feature
type: feature
status: completed
---
# Done
`)

	gitRun(t, projectRoot, "add", ".")
	gitRun(t, projectRoot, "commit", "-m", "add spec")

	findings := Reconcile(heroDir, projectRoot)

	if len(findings) != 0 {
		t.Errorf("completed spec should produce no findings, got %d", len(findings))
	}
}

func TestReconcile_NoFilesTouched(t *testing.T) {
	projectRoot, heroDir := initGitRepo(t)

	// Spec with no Changes section — should be skipped
	addSpec(t, heroDir, "planning/features/no-files/spec.md", `---
title: No Files
type: feature
status: planning
---
# No Files

Just an idea, no implementation plan yet.
`)

	gitRun(t, projectRoot, "add", ".")
	gitRun(t, projectRoot, "commit", "-m", "add spec")

	findings := Reconcile(heroDir, projectRoot)

	if len(findings) != 0 {
		t.Errorf("spec with no FilesTouched should be skipped, got %d findings", len(findings))
	}
}

func TestReconcile_KnowledgeSkipped(t *testing.T) {
	projectRoot, heroDir := initGitRepo(t)

	// Convention spec — should never be reconciled
	addSpec(t, heroDir, "knowledge/conventions/naming/spec.md", `---
title: Naming Convention
type: convention
status: active
---
# Naming
`)

	gitRun(t, projectRoot, "add", ".")
	gitRun(t, projectRoot, "commit", "-m", "add convention")

	findings := Reconcile(heroDir, projectRoot)

	if len(findings) != 0 {
		t.Errorf("knowledge spec should be skipped, got %d findings", len(findings))
	}
}

func TestFinding_CanAutoFix(t *testing.T) {
	tests := []struct {
		from spec.Status
		to   spec.Status
		want bool
	}{
		{spec.StatusPlanning, spec.StatusDelivering, true},
		{spec.StatusPlanning, spec.StatusCompleted, false},
		{spec.StatusDelivering, spec.StatusCompleted, false},
		{spec.StatusInReview, spec.StatusDelivering, false},
		{spec.StatusCompleted, spec.StatusCompleted, true}, // completed stuck in planning
	}

	for _, tt := range tests {
		f := Finding{CurrentStatus: tt.from, SuggestedStatus: tt.to}
		if got := f.CanAutoFix(); got != tt.want {
			t.Errorf("CanAutoFix(%s→%s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
