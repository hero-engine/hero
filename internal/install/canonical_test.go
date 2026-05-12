package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// canonical_test.go — tests for the configurable-content-paths
// behavior (path overrides in hero.json that point canonical content
// at a project-relative path other than the default .hero/<kind>/).

// writeHeroJSON writes a minimal .hero/hero.json with the given content
// overrides in place.
func writeHeroJSON(t *testing.T, projectRoot, agentsPath, commandsPath, skillsPath string) {
	t.Helper()
	heroDir := filepath.Join(projectRoot, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{
		"folder": ".hero",
	}
	content := map[string]string{}
	if agentsPath != "" {
		content["agents_path"] = agentsPath
	}
	if commandsPath != "" {
		content["commands_path"] = commandsPath
	}
	if skillsPath != "" {
		content["skills_path"] = skillsPath
	}
	if len(content) > 0 {
		cfg["content"] = content
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestConfigurableContentPaths_DefaultBehavior asserts that with no
// content overrides, P2 behavior is unchanged: canonical lives at
// .hero/{agents,commands,skills}/ and harness dirs symlink there.
func TestConfigurableContentPaths_DefaultBehavior(t *testing.T) {
	h := newInstallHarness(t)
	writeHeroJSON(t, h.TargetDir, "", "", "")

	h.Run(TargetClaude, nil)

	h.mustBeRegularFile(".hero/agents/engineer.md")
	h.mustBeSymlink(".claude/agents")
	target, _ := os.Readlink(filepath.Join(h.TargetDir, ".claude", "agents"))
	if !strings.Contains(target, ".hero/agents") {
		t.Errorf("default canonical should target .hero/agents, got %q", target)
	}
}

// TestConfigurableContentPaths_OverridePointsAtExisting asserts that
// when hero.json points canonical at an external (already-populated)
// directory, install symlinks the harness dirs directly at it without
// materializing anything into .hero/.
func TestConfigurableContentPaths_OverridePointsAtExisting(t *testing.T) {
	h := newInstallHarness(t)
	writeHeroJSON(t, h.TargetDir, "agents", "commands", "skills")

	// Pre-populate the external paths with content (mimics hero's own
	// repo structure: agents/, commands/, skills/ at the project root).
	external := map[string]string{
		"agents/engineer.md":            "hero-on-hero engineer",
		"commands/design.md":            "hero-on-hero design",
		"skills/spec-format/SKILL.md":   "hero-on-hero spec-format",
	}
	for rel, body := range external {
		full := filepath.Join(h.TargetDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	h.Run(TargetClaude, nil)

	// .hero/agents/ etc. should NOT have been materialized (path is
	// external, install treats it as pre-populated).
	h.mustNotExist(".hero/agents")
	h.mustNotExist(".hero/commands")
	h.mustNotExist(".hero/skills")

	// Harness symlinks point at the external paths.
	for _, link := range []string{".claude/agents", ".claude/commands", ".claude/skills"} {
		target, err := os.Readlink(filepath.Join(h.TargetDir, link))
		if err != nil {
			t.Fatalf("readlink %s: %v", link, err)
		}
		// Should resolve to one of agents/commands/skills — NOT to
		// anywhere under .hero/.
		if strings.Contains(target, ".hero") {
			t.Errorf("%s should target external path, got %q", link, target)
		}
	}

	// Content visible through the symlinks comes from the external
	// paths, not from the embedded source.
	data, err := os.ReadFile(filepath.Join(h.TargetDir, ".claude/agents/engineer.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hero-on-hero engineer" {
		t.Errorf("expected external content, got %q", string(data))
	}
}

// TestConfigurableContentPaths_OverrideMissingPath_Errors asserts the
// error case: configured path doesn't exist on disk.
func TestConfigurableContentPaths_OverrideMissingPath_Errors(t *testing.T) {
	h := newInstallHarness(t)
	writeHeroJSON(t, h.TargetDir, "agents-that-do-not-exist", "", "")

	_, err := Run(Options{
		SourceDir: h.SourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
		Force:     true,
	})
	if err == nil {
		t.Fatal("expected install to error when configured content path doesn't exist")
	}
	if !strings.Contains(err.Error(), "agents-that-do-not-exist") {
		t.Errorf("error should name the missing path; got %v", err)
	}
}

// TestConfigurableContentPaths_PartialOverride asserts that overriding
// just one kind leaves the others at the default .hero/<kind>/.
func TestConfigurableContentPaths_PartialOverride(t *testing.T) {
	h := newInstallHarness(t)
	writeHeroJSON(t, h.TargetDir, "agents", "", "")

	// Only `agents` is external; commands and skills should still default.
	if err := os.MkdirAll(filepath.Join(h.TargetDir, "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.TargetDir, "agents", "engineer.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	h.Run(TargetClaude, nil)

	// agents: external (not in .hero/).
	agentsTarget, _ := os.Readlink(filepath.Join(h.TargetDir, ".claude", "agents"))
	if strings.Contains(agentsTarget, ".hero") {
		t.Errorf(".claude/agents should target external agents/, got %q", agentsTarget)
	}
	h.mustNotExist(".hero/agents")

	// commands and skills: default .hero/<kind>/.
	cmdTarget, _ := os.Readlink(filepath.Join(h.TargetDir, ".claude", "commands"))
	if !strings.Contains(cmdTarget, ".hero/commands") {
		t.Errorf(".claude/commands should default to .hero/commands, got %q", cmdTarget)
	}
	h.mustBeRegularFile(".hero/commands/design.md")
	h.mustBeRegularFile(".hero/skills/spec-format/SKILL.md")
}
