package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/version"
)

func TestUninstall_RequiresTarget(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("uninstall")
	if err == nil {
		t.Fatal("expected error for missing --target")
	}
	if !strings.Contains(err.Error(), "--target is required") {
		t.Errorf("error = %q, want '--target is required'", err.Error())
	}
}

func TestUninstall_UnknownTarget(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("uninstall", "--target", "vscode")
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
	if !strings.Contains(err.Error(), "unknown target") {
		t.Errorf("error = %q, want 'unknown target'", err.Error())
	}
}

func TestUninstall_OpenCode_WithManifest(t *testing.T) {
	env := newTestEnv(t)

	// Create fake installed files
	agentsDir := filepath.Join(env.dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0o755)
	os.WriteFile(filepath.Join(agentsDir, "designer.md"), []byte("# Designer"), 0o644)
	os.WriteFile(filepath.Join(agentsDir, "developer.md"), []byte("# Developer"), 0o644)

	commandsDir := filepath.Join(env.dir, ".opencode", "commands")
	os.MkdirAll(commandsDir, 0o755)
	os.WriteFile(filepath.Join(commandsDir, "design.md"), []byte("# Design"), 0o644)

	// Also create a user file that should NOT be removed
	os.WriteFile(filepath.Join(agentsDir, "my-custom-agent.md"), []byte("# Custom"), 0o644)

	// Write a manifest tracking only the hero-installed files
	heroDir := env.heroDir
	checksums := map[string]string{}
	for _, relPath := range []string{
		".opencode/agents/designer.md",
		".opencode/agents/developer.md",
		".opencode/commands/design.md",
	} {
		absPath := filepath.Join(env.dir, relPath)
		cs, _ := version.FileChecksum(absPath)
		checksums[relPath] = cs
	}
	version.StampInstall(heroDir, "dev", "opencode", "project", checksums)

	output, err := runCmd("uninstall", "--target", "opencode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Removed 3") {
		t.Errorf("expected 3 hero files removed, got: %s", output)
	}

	// Hero files should be gone
	if _, err := os.Stat(filepath.Join(agentsDir, "designer.md")); !os.IsNotExist(err) {
		t.Error("hero file designer.md should be removed")
	}
	if _, err := os.Stat(filepath.Join(agentsDir, "developer.md")); !os.IsNotExist(err) {
		t.Error("hero file developer.md should be removed")
	}

	// User file should be preserved
	if _, err := os.Stat(filepath.Join(agentsDir, "my-custom-agent.md")); os.IsNotExist(err) {
		t.Error("user file my-custom-agent.md should be preserved")
	}

	// Agents dir should still exist (has user file)
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		t.Error("agents dir should still exist (user file remains)")
	}

	// Commands dir should be cleaned up (empty after removal)
	if _, err := os.Stat(commandsDir); !os.IsNotExist(err) {
		t.Error("commands dir should be removed (empty)")
	}

	if !strings.Contains(output, "Preserved 1") {
		t.Errorf("expected 1 preserved user file, got: %s", output)
	}
}

func TestUninstall_OpenCode_NoManifest_Fallback(t *testing.T) {
	env := newTestEnv(t)

	// Create files — no manifest, so fallback to known-file-name matching
	agentsDir := filepath.Join(env.dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0o755)
	// Known hero file
	os.WriteFile(filepath.Join(agentsDir, "feature-delivery-lead.md"), []byte("# FDL"), 0o644)
	// User file (unknown name)
	os.WriteFile(filepath.Join(agentsDir, "my-agent.md"), []byte("# Mine"), 0o644)

	commandsDir := filepath.Join(env.dir, ".opencode", "commands")
	os.MkdirAll(commandsDir, 0o755)
	os.WriteFile(filepath.Join(commandsDir, "design.md"), []byte("# Design"), 0o644)
	os.WriteFile(filepath.Join(commandsDir, "my-workflow.md"), []byte("# Workflow"), 0o644)

	output, err := runCmd("uninstall", "--target", "opencode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Known hero files should be removed
	if _, err := os.Stat(filepath.Join(agentsDir, "feature-delivery-lead.md")); !os.IsNotExist(err) {
		t.Error("known hero file should be removed")
	}
	if _, err := os.Stat(filepath.Join(commandsDir, "design.md")); !os.IsNotExist(err) {
		t.Error("known hero command should be removed")
	}

	// User files should be preserved
	if _, err := os.Stat(filepath.Join(agentsDir, "my-agent.md")); os.IsNotExist(err) {
		t.Error("user agent file should be preserved")
	}
	if _, err := os.Stat(filepath.Join(commandsDir, "my-workflow.md")); os.IsNotExist(err) {
		t.Error("user command file should be preserved")
	}

	if !strings.Contains(output, "Removed 2") {
		t.Errorf("expected 2 hero files removed, got: %s", output)
	}
	if !strings.Contains(output, "Preserved 2") {
		t.Errorf("expected 2 user files preserved, got: %s", output)
	}
}

func TestUninstall_Cursor(t *testing.T) {
	env := newTestEnv(t)

	rulesBase := filepath.Join(env.dir, ".cursor", "rules")
	agentsDir := filepath.Join(rulesBase, "agents")
	os.MkdirAll(agentsDir, 0o755)
	os.WriteFile(filepath.Join(agentsDir, "designer.md"), []byte("# Designer"), 0o644)

	// Write manifest
	heroDir := env.heroDir
	cs, _ := version.FileChecksum(filepath.Join(agentsDir, "designer.md"))
	version.StampInstall(heroDir, "dev", "cursor", "project", map[string]string{
		".cursor/rules/agents/designer.md": cs,
	})

	output, err := runCmd("uninstall", "--target", "cursor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Removed 1") {
		t.Errorf("expected 1 file removed, got: %s", output)
	}
}

func TestUninstall_Claude(t *testing.T) {
	env := newTestEnv(t)

	claudeBase := filepath.Join(env.dir, ".claude")
	agentsDir := filepath.Join(claudeBase, "agents")
	os.MkdirAll(agentsDir, 0o755)
	os.WriteFile(filepath.Join(agentsDir, "designer.md"), []byte("# Designer"), 0o644)

	// Write manifest
	heroDir := env.heroDir
	cs, _ := version.FileChecksum(filepath.Join(agentsDir, "designer.md"))
	version.StampInstall(heroDir, "dev", "claude", "project", map[string]string{
		".claude/agents/designer.md": cs,
	})

	// Create CLAUDE.md with hero section
	claudeMdPath := filepath.Join(env.dir, "CLAUDE.md")
	claudeContent := `# My Project

Some instructions.

<!-- hero:managed -->
# Hero — Spec-Driven Workflow

Stuff here.
<!-- hero:managed -->

More instructions.
`
	os.WriteFile(claudeMdPath, []byte(claudeContent), 0o644)

	output, err := runCmd("uninstall", "--target", "claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Cleaned hero section") {
		t.Errorf("expected hero section cleanup, got: %s", output)
	}

	// Verify CLAUDE.md still exists but hero section is removed
	data, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatal("CLAUDE.md should still exist")
	}
	content := string(data)
	if strings.Contains(content, "hero:managed") {
		t.Error("hero section should be removed")
	}
	if !strings.Contains(content, "My Project") {
		t.Error("non-hero content should be preserved")
	}
	if !strings.Contains(content, "More instructions") {
		t.Error("content after hero section should be preserved")
	}
}

func TestUninstall_OpenCode_CleansAgentsMd(t *testing.T) {
	env := newTestEnv(t)

	// Create an AGENTS.md with hero-managed section
	agentsMdPath := filepath.Join(env.dir, "AGENTS.md")
	content := `# My Project Rules

Custom instructions here.

<!-- hero:managed -->
# Hero — Spec-Driven AI Engineering

Hero routing table and instructions.
<!-- hero:managed -->

More custom rules.
`
	os.WriteFile(agentsMdPath, []byte(content), 0o644)

	output, err := runCmd("uninstall", "--target", "opencode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Cleaned hero section") {
		t.Errorf("expected hero section cleanup in AGENTS.md, got: %s", output)
	}

	// Verify AGENTS.md still exists but hero section is removed
	data, err := os.ReadFile(agentsMdPath)
	if err != nil {
		t.Fatal("AGENTS.md should still exist")
	}
	result := string(data)
	if strings.Contains(result, "hero:managed") {
		t.Error("hero section should be removed from AGENTS.md")
	}
	if !strings.Contains(result, "My Project Rules") {
		t.Error("content before hero section should be preserved")
	}
	if !strings.Contains(result, "More custom rules") {
		t.Error("content after hero section should be preserved")
	}
}

func TestUninstall_NothingToRemove(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	output, err := runCmd("uninstall", "--target", "opencode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Nothing to remove") {
		t.Errorf("expected 'Nothing to remove', got: %s", output)
	}
}

func TestUninstall_DryRun(t *testing.T) {
	env := newTestEnv(t)

	agentsDir := filepath.Join(env.dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0o755)
	os.WriteFile(filepath.Join(agentsDir, "feature-delivery-lead.md"), []byte("# FDL"), 0o644)
	os.WriteFile(filepath.Join(agentsDir, "my-custom.md"), []byte("# Custom"), 0o644)

	output, err := runCmd("uninstall", "--target", "opencode", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Would remove") {
		t.Errorf("expected 'Would remove' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "Preserving") {
		t.Errorf("expected 'Preserving' for user file in dry-run output, got: %s", output)
	}

	// Both files should still exist
	if _, err := os.Stat(filepath.Join(agentsDir, "feature-delivery-lead.md")); os.IsNotExist(err) {
		t.Error("hero file should not be removed in dry-run mode")
	}
	if _, err := os.Stat(filepath.Join(agentsDir, "my-custom.md")); os.IsNotExist(err) {
		t.Error("user file should not be removed in dry-run mode")
	}
}

func TestRemoveHeroManagedSection(t *testing.T) {
	// Reset global flag in case a previous test set it
	uninstallDryRun = false

	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	content := `# Intro
<!-- hero:managed -->
# Hero Section
<!-- hero:managed -->
# After
`
	os.WriteFile(path, []byte(content), 0o644)

	cleaned, err := removeHeroManagedSection(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cleaned {
		t.Error("expected section to be cleaned")
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if strings.Contains(result, "hero:managed") {
		t.Error("hero markers should be removed")
	}
	if !strings.Contains(result, "Intro") {
		t.Error("content before hero section should be preserved")
	}
	if !strings.Contains(result, "After") {
		t.Error("content after hero section should be preserved")
	}
}

func TestRemoveHeroManagedSection_NoSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")

	content := "# Just a regular file\n"
	os.WriteFile(path, []byte(content), 0o644)

	cleaned, err := removeHeroManagedSection(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleaned {
		t.Error("should not report cleaning when no section exists")
	}
}

func TestUninstall_ManifestUpdated(t *testing.T) {
	env := newTestEnv(t)

	agentsDir := filepath.Join(env.dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0o755)
	os.WriteFile(filepath.Join(agentsDir, "designer.md"), []byte("# Designer"), 0o644)

	heroDir := env.heroDir
	cs, _ := version.FileChecksum(filepath.Join(agentsDir, "designer.md"))
	version.StampInstall(heroDir, "dev", "opencode", "project", map[string]string{
		".opencode/agents/designer.md": cs,
	})

	_, err := runCmd("uninstall", "--target", "opencode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify manifest no longer tracks the removed file
	info, err := version.Read(heroDir)
	if err != nil {
		t.Fatalf("reading version.json: %v", err)
	}
	if info != nil && info.InstalledFiles != nil {
		if _, exists := info.InstalledFiles[".opencode/agents/designer.md"]; exists {
			t.Error("removed file should no longer be in the manifest")
		}
	}
}
