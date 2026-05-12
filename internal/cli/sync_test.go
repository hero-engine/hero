package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
	"testing"
)

// --- sync command tests ---

func TestSync_RequiresArg(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("sync", "spec")
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestSync_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("sync", "spec", "some/spec.md")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestSync_NoTrackerConfigured(t *testing.T) {
	env := newTestEnv(t)

	// Create a spec to sync
	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export

## Goal

Export data to CSV.
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("sync", "spec", specPath)
	if err == nil {
		t.Fatal("expected error for no tracker configured")
	}
	if !strings.Contains(err.Error(), "no tracker configured") {
		t.Errorf("error = %q, want 'no tracker configured'", err.Error())
	}
}

func TestSync_InvalidSpecPath(t *testing.T) {
	env := newTestEnv(t)

	// Configure a tracker with token set
	writeTrackerConfig(env, "github", "acme/widgets")
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	_, err := runCmd("sync", "spec", "/nonexistent/spec.md")
	if err == nil {
		t.Fatal("expected error for invalid spec path")
	}
	if !strings.Contains(err.Error(), "parsing spec") {
		t.Errorf("error = %q, want 'parsing spec'", err.Error())
	}
}

func TestSync_MissingToken(t *testing.T) {
	env := newTestEnv(t)

	// Configure tracker but don't set the token env var
	writeTrackerConfig(env, "github", "acme/widgets")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	os.Unsetenv("HERO_TEST_TOKEN")
	_, err := runCmd("sync", "spec", specPath)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "HERO_TEST_TOKEN") {
		t.Errorf("error = %q, want mention of env var", err.Error())
	}
}

// --- link command tests ---

func TestLink_RequiresArgs(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("sync", "link")
	if err == nil {
		t.Fatal("expected error for missing args")
	}

	_, err = runCmd("sync", "link", "spec.md")
	if err == nil {
		t.Fatal("expected error for missing issue-id arg")
	}
}

func TestLink_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("sync", "link", "spec.md", "42")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestLink_NoTrackerConfigured(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("sync", "link", specPath, "42")
	if err == nil {
		t.Fatal("expected error for no tracker configured")
	}
	if !strings.Contains(err.Error(), "no tracker configured") {
		t.Errorf("error = %q, want 'no tracker configured'", err.Error())
	}
}

func TestLink_AlreadyLinked(t *testing.T) {
	env := newTestEnv(t)

	writeTrackerConfig(env, "github", "acme/widgets")
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
tracker_id: 99
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("sync", "link", specPath, "42")
	if err == nil {
		t.Fatal("expected error for already-linked spec")
	}
	if !strings.Contains(err.Error(), "already linked") {
		t.Errorf("error = %q, want 'already linked'", err.Error())
	}
}

// --- injectFrontmatterField tests ---

func TestInjectFrontmatterField_NewField(t *testing.T) {
	input := "---\ntitle: My Spec\ntype: feature\n---\n# My Spec\n"
	result := spec.SetFrontmatterField(input, "tracker_id", "42")

	if !strings.Contains(result, "tracker_id: 42") {
		t.Errorf("expected tracker_id: 42, got:\n%s", result)
	}
	if !strings.Contains(result, "title: My Spec") {
		t.Errorf("original content should be preserved")
	}
}

func TestInjectFrontmatterField_UpdateExisting(t *testing.T) {
	input := "---\ntitle: My Spec\ntracker_id: old-id\n---\n# My Spec\n"
	result := spec.SetFrontmatterField(input, "tracker_id", "new-id")

	if !strings.Contains(result, "tracker_id: new-id") {
		t.Errorf("expected tracker_id: new-id, got:\n%s", result)
	}
	if strings.Contains(result, "old-id") {
		t.Errorf("old value should be replaced")
	}
}

func TestInjectFrontmatterField_NoFrontmatter(t *testing.T) {
	input := "# My Spec\n\nSome content.\n"
	result := spec.SetFrontmatterField(input, "tracker_id", "42")

	if !strings.HasPrefix(result, "---\n") {
		t.Error("expected frontmatter to be created at start")
	}
	if !strings.Contains(result, "tracker_id: 42") {
		t.Errorf("expected tracker_id: 42, got:\n%s", result)
	}
	if !strings.Contains(result, "# My Spec") {
		t.Error("original content should be preserved")
	}
}

func TestInjectFrontmatterField_EmptyFile(t *testing.T) {
	input := ""
	result := spec.SetFrontmatterField(input, "tracker_id", "42")

	if !strings.Contains(result, "tracker_id: 42") {
		t.Errorf("expected tracker_id: 42, got:\n%s", result)
	}
}

// --- writeTrackerID integration test ---

func TestWriteTrackerID(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")

	content := "---\ntitle: Test\ntype: feature\n---\n# Test\n"
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeTrackerID(specPath, "PROJ-42"); err != nil {
		t.Fatalf("writeTrackerID failed: %v", err)
	}

	data, _ := os.ReadFile(specPath)
	if !strings.Contains(string(data), "tracker_id: PROJ-42") {
		t.Errorf("expected tracker_id in file, got:\n%s", string(data))
	}
}

// --- splitLines / joinLines ---

func TestSplitJoinRoundTrip(t *testing.T) {
	input := "line1\nline2\nline3"
	lines := strings.Split(input, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	result := strings.Join(lines, "\n")
	if result != input {
		t.Errorf("round-trip failed: %q != %q", result, input)
	}
}

// --- Helpers ---

// writeTrackerConfig overwrites the hero.json with a tracker config.
func writeTrackerConfig(env *testEnv, trackerType, project string) {
	env.t.Helper()

	configJSON := `{
  "folder": ".hero",
  "tracker": {
    "type": "` + trackerType + `",
    "project": "` + project + `",
    "token_env": "HERO_TEST_TOKEN"
  }
}`
	configPath := filepath.Join(env.heroDir, "hero.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		env.t.Fatalf("WriteFile: %v", err)
	}
}
