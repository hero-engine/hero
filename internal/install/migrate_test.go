package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// migrate_test.go — coverage for `hero install --migrate`.

// seedLegacyHarnessCopies creates a legacy-multi-harness-shape directory: agents/
// rendered into both .claude/ and .opencode/ as physical copies. The
// content is identical unless `drift` is non-nil, in which case it
// specifies a per-target override for one filename (simulating drift).
//
// Returns the absolute paths of the seeded files for later assertions.
func seedLegacyHarnessCopies(t *testing.T, projectRoot string, drift map[Target]map[string]string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(projectRoot, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Build per-target file contents: shared baseline + optional overrides.
	baseline := map[string]string{
		"agents/engineer.md": "v1 engineer",
		"agents/reviewer.md": "v1 reviewer",
		"commands/design.md": "v1 design",
	}

	targets := []struct {
		target Target
		subdir string
	}{
		{TargetClaude, ".claude"},
		{TargetOpenCode, ".opencode"},
	}

	for _, tgt := range targets {
		for relPath, content := range baseline {
			if override, ok := drift[tgt.target][relPath]; ok {
				content = override
			}
			full := filepath.Join(projectRoot, tgt.subdir, relPath)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		// Seed a skill in legacy SKILL.md layout (claude+codex+opencode
		// all use that layout).
		skillDir := filepath.Join(projectRoot, tgt.subdir, "skills", "spec-format")
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		skillContent := "v1 spec-format"
		if override, ok := drift[tgt.target]["skills/spec-format"]; ok {
			skillContent = override
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestMigrate_NoDrift_OneCanonical asserts that a legacy-multi-harness-shape
// install with identical content across harness dirs migrates cleanly:
// canonical gets one copy of each file, harness dirs become symlinks.
func TestMigrate_NoDrift_OneCanonical(t *testing.T) {
	h := newInstallHarness(t)
	seedLegacyHarnessCopies(t, h.TargetDir, map[Target]map[string]string{})

	report, err := RunMigrate(Options{
		SourceDir: h.SourceDir,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
		Force:     true,
	})
	if err != nil {
		t.Fatalf("RunMigrate: %v\n%s", err, report.StringReport())
	}

	// No conflicts when content matches.
	if len(report.Conflicts) != 0 {
		t.Errorf("expected no conflicts on identical content, got %d:\n%s", len(report.Conflicts), report.StringReport())
	}
	// Both detected.
	gotTargets := map[Target]bool{}
	for _, t := range report.DetectedTargets {
		gotTargets[t] = true
	}
	if !gotTargets[TargetClaude] || !gotTargets[TargetOpenCode] {
		t.Errorf("expected both claude and opencode detected, got %v", report.DetectedTargets)
	}
	// Canonical now has the promoted content.
	h.mustBeRegularFile(".hero/agents/engineer.md")
	h.mustBeRegularFile(".hero/commands/design.md")
	h.mustBeRegularFile(".hero/skills/spec-format/SKILL.md")
	// Both harness dirs are now symlinks pointing at canonical.
	for _, link := range []string{".claude/agents", ".claude/commands", ".claude/skills", ".opencode/agents", ".opencode/commands", ".opencode/skills"} {
		h.mustBeSymlink(link)
	}
}

// TestMigrate_DriftedCopies_NewestWins asserts the drift-resolution
// behavior: same filename, different content across harness dirs.
// Newest mtime wins; report records the conflict and the winner.
func TestMigrate_DriftedCopies_NewestWins(t *testing.T) {
	h := newInstallHarness(t)
	seedLegacyHarnessCopies(t, h.TargetDir, map[Target]map[string]string{
		TargetOpenCode: {
			"agents/engineer.md": "v2 engineer (opencode, newer)",
		},
	})

	// Make the opencode copy newer by 10s.
	openCodeFile := filepath.Join(h.TargetDir, ".opencode/agents/engineer.md")
	newer := time.Now().Add(10 * time.Second)
	if err := os.Chtimes(openCodeFile, newer, newer); err != nil {
		t.Fatal(err)
	}

	report, err := RunMigrate(Options{
		SourceDir: h.SourceDir,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
		Force:     true,
	})
	if err != nil {
		t.Fatalf("RunMigrate: %v\n%s", err, report.StringReport())
	}

	// Exactly one conflict reported, for agents/engineer.md.
	if len(report.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d:\n%s", len(report.Conflicts), report.StringReport())
	}
	c := report.Conflicts[0]
	if c.Kind != "agents" || c.File != "engineer.md" {
		t.Errorf("conflict identity: got kind=%q file=%q", c.Kind, c.File)
	}
	if !strings.Contains(c.Winner, ".opencode") {
		t.Errorf("expected opencode (newer) to win, got %q", c.Winner)
	}

	// Canonical contains the opencode (winner) content.
	data, _ := os.ReadFile(filepath.Join(h.TargetDir, ".hero/agents/engineer.md"))
	if string(data) != "v2 engineer (opencode, newer)" {
		t.Errorf("canonical should hold the winner's content, got %q", string(data))
	}
}

// TestMigrate_DryRun_MakesNoChanges asserts --dry-run reports drift
// without writing to the filesystem.
func TestMigrate_DryRun_MakesNoChanges(t *testing.T) {
	h := newInstallHarness(t)
	seedLegacyHarnessCopies(t, h.TargetDir, map[Target]map[string]string{
		TargetClaude: {"agents/engineer.md": "drifted"},
	})

	report, err := RunMigrate(Options{
		SourceDir: h.SourceDir,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("RunMigrate dry run: %v", err)
	}

	if !report.DryRun {
		t.Error("report should mark DryRun=true")
	}
	// Canonical should NOT exist (no write happened).
	h.mustNotExist(".hero/agents/engineer.md")
	// Symlinks should not have been created either (the per-target
	// install runs do their own dry-run handling — they don't write).
	if info, err := os.Lstat(filepath.Join(h.TargetDir, ".claude/agents")); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			t.Error("dry run should not have created a symlink")
		}
	}
}

// TestMigrate_NoHarnessesDetected_Errors asserts the no-op-case error
// message when called against a directory with no installed harnesses.
func TestMigrate_NoHarnessesDetected_Errors(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := RunMigrate(Options{
		SourceDir: h.SourceDir,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
	})
	if err == nil {
		t.Fatal("expected error when no harness targets detected")
	}
	if !strings.Contains(err.Error(), "nothing to migrate") {
		t.Errorf("expected helpful error message; got: %v", err)
	}
}

// TestMigrate_Idempotent_AfterFirstRun asserts running migrate twice in
// a row produces zero new changes the second time.
func TestMigrate_Idempotent_AfterFirstRun(t *testing.T) {
	h := newInstallHarness(t)
	seedLegacyHarnessCopies(t, h.TargetDir, map[Target]map[string]string{})

	if _, err := RunMigrate(Options{
		SourceDir: h.SourceDir,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
		Force:     true,
	}); err != nil {
		t.Fatalf("first RunMigrate: %v", err)
	}

	report, err := RunMigrate(Options{
		SourceDir: h.SourceDir,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
		Force:     true,
	})
	if err != nil {
		t.Fatalf("second RunMigrate: %v", err)
	}
	if len(report.Conflicts) != 0 {
		t.Errorf("second run should detect no drift (everything's canonical now), got %d conflicts", len(report.Conflicts))
	}
}
