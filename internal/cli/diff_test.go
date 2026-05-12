package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiff_RequiresArg(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("diff")
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestDiff_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("diff", "some/spec.md")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestDiff_InvalidSpecPath(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("diff", "/nonexistent/spec.md")
	if err == nil {
		t.Fatal("expected error for invalid spec path")
	}
	if !strings.Contains(err.Error(), "parsing spec") {
		t.Errorf("error = %q, want 'parsing spec'", err.Error())
	}
}

func TestDiff_NoChangesSection(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export

## Goal

Export data.
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	output, err := runCmd("diff", specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "No files listed") {
		t.Errorf("expected 'No files listed' message, got: %s", output)
	}
}

func TestDiff_WithChangesSection(t *testing.T) {
	env := newTestEnv(t)

	// Initialize a git repo in the test dir so git diff works
	initGitRepo(t, env.dir)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export

## Goal

Export data.

## Changes

- internal/export/csv.go
- internal/export/json.go
`)

	// Create one of the files so it shows up in git as untracked
	csvDir := filepath.Join(env.dir, "internal", "export")
	os.MkdirAll(csvDir, 0o755)
	os.WriteFile(filepath.Join(csvDir, "csv.go"), []byte("package export\n"), 0o644)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	output, err := runCmd("diff", specPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "csv-export") {
		t.Errorf("expected spec slug in output, got: %s", output)
	}
}

// initGitRepo creates a minimal git repo in dir with an initial commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "add", "."},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}

	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git command %v failed: %v\n%s", args, err, out)
		}
	}
}
