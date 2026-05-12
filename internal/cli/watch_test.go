package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/watch"
)

func TestWatchCI_emptyWorkspace(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("watch", "--mode", "ci")
	if err != nil {
		t.Fatalf("watch --mode ci returned error on empty workspace: %v", err)
	}

	if !strings.Contains(output, "Hero CI check") {
		t.Errorf("output missing header: %q", output)
	}
	if !strings.Contains(output, "Indexed: 0 specs") {
		t.Errorf("output missing indexed count: %q", output)
	}
	if !strings.Contains(output, "No issues found") {
		t.Errorf("output missing no-issues message: %q", output)
	}
}

func TestWatchCI_withSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/auth-login/spec.md", `---
title: Auth Login
type: feature
status: completed
---
# Auth Login
Feature spec for authentication login.
`)

	output, err := runCmd("watch", "--mode", "ci")
	if err != nil {
		t.Fatalf("watch --mode ci returned error: %v", err)
	}

	if !strings.Contains(output, "Indexed: 1 specs") {
		t.Errorf("output missing indexed count: %q", output)
	}
	if !strings.Contains(output, "1 features") {
		t.Errorf("output missing feature count: %q", output)
	}
}

func TestWatchCI_requiresWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("watch", "--mode", "ci")
	if err == nil {
		t.Fatal("watch --mode ci should fail without hero workspace")
	}

	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error should mention workspace: %v", err)
	}
}

func TestWatchCI_invalidMode(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("watch", "--mode", "invalid")
	if err == nil {
		t.Fatal("watch with invalid mode should fail")
	}

	if !strings.Contains(err.Error(), "unknown mode") {
		t.Errorf("error should mention unknown mode: %v", err)
	}
}

func TestWatchCI_staleSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/old-feature/spec.md", `---
title: Old Feature
type: feature
status: planning
---
# Old Feature
A feature that has been planning for a while.
`)

	// Set the file mtime to 30 days ago to make it stale
	specPath := filepath.Join(env.heroDir, "specs", "old-feature", "spec.md")
	oldTime := time.Now().AddDate(0, 0, -30)
	os.Chtimes(specPath, oldTime, oldTime)

	output, err := runCmd("watch", "--mode", "ci")
	if err == nil {
		t.Fatal("watch --mode ci should fail with stale specs")
	}

	if !strings.Contains(output, "Stale specs") || !strings.Contains(output, "old-feature") {
		t.Errorf("output should mention stale spec: %q", output)
	}
	if !strings.Contains(output, "issue(s) found") {
		t.Errorf("output should mention issues: %q", output)
	}
}

func TestWatchCI_unclaimedSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/unclaimed-work/spec.md", `---
title: Unclaimed Work
type: feature
status: planning
---
# Unclaimed Work
Nobody has claimed this.
`)

	output, err := runCmd("watch", "--mode", "ci")
	if err == nil {
		t.Fatal("watch --mode ci should fail with unclaimed specs")
	}

	if !strings.Contains(output, "Unclaimed specs") || !strings.Contains(output, "unclaimed-work") {
		t.Errorf("output should mention unclaimed spec: %q", output)
	}
}

func TestWatchCI_multipleSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/feature-a/spec.md", `---
title: Feature A
type: feature
status: completed
---
# Feature A
Done.
`)

	env.addSpec("specs/bug-b/spec.md", `---
title: Bug B
type: bug
status: completed
---
# Bug B
Fixed.
`)

	env.addSpec("knowledge/conventions/naming/spec.md", `---
title: Naming Convention
type: convention
status: active
scope: ["*.go"]
---
# Naming Convention
Use camelCase.
`)

	output, err := runCmd("watch", "--mode", "ci")
	if err != nil {
		t.Fatalf("watch --mode ci returned error: %v", err)
	}

	if !strings.Contains(output, "Indexed: 3 specs") {
		t.Errorf("output missing indexed count: %q", output)
	}
	if !strings.Contains(output, "1 features") {
		t.Errorf("output missing feature count: %q", output)
	}
	if !strings.Contains(output, "1 bugs") {
		t.Errorf("output missing bug count: %q", output)
	}
	if !strings.Contains(output, "1 conventions") {
		t.Errorf("output missing convention count: %q", output)
	}
}

func TestWatch_defaultMode(t *testing.T) {
	// Verify default flag values after reset
	resetFlags()
	if watchMode != "local" {
		t.Errorf("default watchMode = %q, want 'local'", watchMode)
	}
	if watchInterval != 2 {
		t.Errorf("default watchInterval = %d, want 2", watchInterval)
	}
}

func TestSlugFromPath_deleted(t *testing.T) {
	// Test slugFromPath with a non-existent file (simulating deletion)
	slug, err := slugFromPath("/some/hero/specs/my-feature/spec.md")
	if err != nil {
		t.Fatalf("slugFromPath returned error: %v", err)
	}
	if slug != "my-feature" {
		t.Errorf("slug = %q, want 'my-feature'", slug)
	}
}

func TestSlugFromPath_rootPath(t *testing.T) {
	// Root path has no parent directory — should return error
	_, err := slugFromPath("/spec.md")
	if err == nil {
		t.Fatal("slugFromPath should error for root-level path")
	}
}

func TestValidateAllSpecs_noSpecs(t *testing.T) {
	env := newTestEnv(t)
	errors := validateAllSpecs(env.heroDir)
	if errors != 0 {
		t.Errorf("expected 0 errors on empty workspace, got %d", errors)
	}
}

func TestValidateAllSpecs_validSpecs(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("specs/valid-feature/spec.md", `---
title: Valid Feature
type: feature
status: planning
---
# Valid Feature
This is a valid spec.
`)

	errors := validateAllSpecs(env.heroDir)
	if errors != 0 {
		t.Errorf("expected 0 errors, got %d", errors)
	}
}

func TestReindexSpecs_createdSpec(t *testing.T) {
	env := newTestEnv(t)

	// Create a spec file
	env.addSpec("specs/new-thing/spec.md", `---
title: New Thing
type: feature
status: planning
---
# New Thing
A new feature.
`)

	specPath := filepath.Join(env.heroDir, "specs", "new-thing", "spec.md")

	// Call reindexSpecs with a created event
	output := captureStdout(func() {
		reindexSpecs(env.heroDir, []watch.Event{
			{Path: specPath, Kind: watch.EventCreated},
		})
	})

	if !strings.Contains(output, "reindexed: new-thing") {
		t.Errorf("output should mention reindexed spec: %q", output)
	}

	// Verify the spec is in the index
	env.indexAll()
	// If we got here without error, the spec was indexed successfully
}
