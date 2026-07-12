package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunOpenCodeProject(t *testing.T) {
	// Set up source directory with agents, commands, skills
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
		DryRun:    false,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(result.Copied) == 0 {
		t.Error("expected files to be copied")
	}

	// Check agents were copied flat
	agentPath := filepath.Join(targetDir, ".opencode", "agents", "engineer.md")
	if _, err := os.Stat(agentPath); err != nil {
		t.Errorf("agent not copied: %v", err)
	}

	// Check commands were copied flat
	cmdPath := filepath.Join(targetDir, ".opencode", "commands", "design.md")
	if _, err := os.Stat(cmdPath); err != nil {
		t.Errorf("command not copied: %v", err)
	}

	// Check skills were converted to nested SKILL.md
	skillPath := filepath.Join(targetDir, ".opencode", "skills", "spec-format", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("skill not converted to nested format: %v", err)
	}
}

func TestRunOpenCodeGlobal(t *testing.T) {
	sourceDir := t.TempDir()
	createContent(t, sourceDir)

	// Override HOME for the test
	home := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeGlobal,
		Force:     false,
		DryRun:    false,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(result.Copied) == 0 {
		t.Error("expected files to be copied")
	}

	// Check files went to ~/.config/opencode/
	agentPath := filepath.Join(home, ".config", "opencode", "agents", "engineer.md")
	if _, err := os.Stat(agentPath); err != nil {
		t.Errorf("agent not installed globally: %v", err)
	}
}

func TestRunDryRun(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
		DryRun:    true,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should track what would be copied
	if len(result.Copied) == 0 {
		t.Error("dry run should still track planned copies")
	}

	// But no actual files should exist
	agentPath := filepath.Join(targetDir, ".opencode", "agents", "engineer.md")
	if _, err := os.Stat(agentPath); err == nil {
		t.Error("dry run should not create files")
	}
}

func TestRunRefusesOverwriteOnDrift(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	if _, err := Run(opts); err != nil {
		t.Fatalf("first Run failed: %v", err)
	}

	// Introduce drift: the installed file no longer matches what we'd
	// write. Without --force, the second install must refuse rather
	// than silently clobbering the local edit.
	driftPath := filepath.Join(targetDir, ".opencode", "agents", "engineer.md")
	if err := os.WriteFile(driftPath, []byte("# Engineer Agent (locally edited)"), 0o644); err != nil {
		t.Fatalf("seed drift: %v", err)
	}

	_, err := Run(opts)
	if err == nil {
		t.Fatal("Run should refuse to overwrite drifted file without --force")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Errorf("error should mention refusing to overwrite: %v", err)
	}
}

func TestRunIdempotentReinstall(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	if _, err := Run(opts); err != nil {
		t.Fatalf("first Run failed: %v", err)
	}

	// Same binary, same source, nothing edited: second Run should
	// succeed silently — the canonical content already matches what
	// we would write, so the idempotency contract takes over.
	result, err := Run(opts)
	if err != nil {
		t.Fatalf("idempotent re-run should succeed: %v", err)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("idempotent re-run should not record Skipped entries, got %v", result.Skipped)
	}
}

func TestRunIdempotentAcrossTargets(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	// First harness — populates the canonical .hero/ tree.
	if _, err := Run(Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}); err != nil {
		t.Fatalf("first target install failed: %v", err)
	}

	// Second harness against the same project — must succeed without
	// --force, even though canonical content already exists from the
	// first install. This is the user-facing multi-harness scenario.
	if _, err := Run(Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}); err != nil {
		t.Fatalf("second target install must succeed without --force: %v", err)
	}
}

func TestRunForceOverwrite(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	// First install
	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}
	_, err := Run(opts)
	if err != nil {
		t.Fatalf("first Run failed: %v", err)
	}

	// Second install with force should succeed
	opts.Force = true
	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run --force failed: %v", err)
	}

	if len(result.Copied) == 0 {
		t.Error("force install should copy files")
	}
}

func TestRunUnsupportedTarget(t *testing.T) {
	opts := Options{
		SourceDir: t.TempDir(),
		Target:    Target("vscode"),
		Mode:      ModeProject,
		TargetDir: t.TempDir(),
	}

	_, err := Run(opts)
	if err == nil {
		t.Fatal("unsupported target should return error")
	}

	if !strings.Contains(err.Error(), "unknown target") {
		t.Errorf("error should mention 'unknown target': %v", err)
	}
}

func TestRunClaudeTargetWorks(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	_, err := Run(opts)
	if err != nil {
		t.Fatalf("claude target should work: %v", err)
	}
}

func TestRunInvalidMode(t *testing.T) {
	opts := Options{
		SourceDir: t.TempDir(),
		Target:    TargetOpenCode,
		Mode:      Mode("invalid"),
		TargetDir: t.TempDir(),
	}

	_, err := Run(opts)
	if err == nil {
		t.Fatal("invalid mode should return error")
	}
}

func TestRunProjectModeRequiresTarget(t *testing.T) {
	opts := Options{
		SourceDir: t.TempDir(),
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: "", // missing
	}

	_, err := Run(opts)
	if err == nil {
		t.Fatal("project mode without target dir should fail")
	}
}

func TestRunNonexistentTargetDir(t *testing.T) {
	opts := Options{
		SourceDir: t.TempDir(),
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: "/nonexistent/path",
	}

	_, err := Run(opts)
	if err == nil {
		t.Fatal("nonexistent target dir should fail")
	}
}

func TestMergeJSON(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.json")
	dstPath := filepath.Join(tmpDir, "dst.json")

	srcData := map[string]interface{}{
		"key1": "from_source",
		"key2": "source_only",
		"nested": map[string]interface{}{
			"a": "source_a",
			"b": "source_b",
		},
	}

	dstData := map[string]interface{}{
		"key1": "from_dest",
		"key3": "dest_only",
		"nested": map[string]interface{}{
			"a": "dest_a",
			"c": "dest_c",
		},
	}

	writeJSON(t, srcPath, srcData)
	writeJSON(t, dstPath, dstData)

	// Without force: dest values win on conflict
	srcBytes, _ := os.ReadFile(srcPath)
	if err := mergeJSONFromData(srcBytes, dstPath, false); err != nil {
		t.Fatalf("mergeJSONFromData failed: %v", err)
	}

	result := readJSON(t, dstPath)
	if result["key1"] != "from_dest" {
		t.Errorf("key1 should be from dest without force: %v", result["key1"])
	}
	if result["key2"] != "source_only" {
		t.Errorf("key2 should be added from source: %v", result["key2"])
	}
	if result["key3"] != "dest_only" {
		t.Errorf("key3 should remain from dest: %v", result["key3"])
	}

	nested := result["nested"].(map[string]interface{})
	if nested["a"] != "dest_a" {
		t.Errorf("nested.a should be from dest without force: %v", nested["a"])
	}
	if nested["b"] != "source_b" {
		t.Errorf("nested.b should be added from source: %v", nested["b"])
	}
	if nested["c"] != "dest_c" {
		t.Errorf("nested.c should remain from dest: %v", nested["c"])
	}
}

func TestMergeJSONForce(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.json")
	dstPath := filepath.Join(tmpDir, "dst.json")

	writeJSON(t, srcPath, map[string]interface{}{"key1": "source_wins"})
	writeJSON(t, dstPath, map[string]interface{}{"key1": "dest_loses"})

	srcBytes, _ := os.ReadFile(srcPath)
	if err := mergeJSONFromData(srcBytes, dstPath, true); err != nil {
		t.Fatalf("mergeJSONFromData force failed: %v", err)
	}

	result := readJSON(t, dstPath)
	if result["key1"] != "source_wins" {
		t.Errorf("key1 should be from source with force: %v", result["key1"])
	}
}

func TestDeepMerge(t *testing.T) {
	base := map[string]interface{}{
		"a": "base_a",
		"b": "base_b",
		"nested": map[string]interface{}{
			"x": "base_x",
			"y": "base_y",
		},
	}

	override := map[string]interface{}{
		"b": "override_b",
		"c": "override_c",
		"nested": map[string]interface{}{
			"y": "override_y",
			"z": "override_z",
		},
	}

	result := deepMerge(base, override)

	if result["a"] != "base_a" {
		t.Errorf("a should be from base: %v", result["a"])
	}
	if result["b"] != "override_b" {
		t.Errorf("b should be from override: %v", result["b"])
	}
	if result["c"] != "override_c" {
		t.Errorf("c should be from override: %v", result["c"])
	}

	nested := result["nested"].(map[string]interface{})
	if nested["x"] != "base_x" {
		t.Errorf("nested.x should be from base: %v", nested["x"])
	}
	if nested["y"] != "override_y" {
		t.Errorf("nested.y should be from override: %v", nested["y"])
	}
	if nested["z"] != "override_z" {
		t.Errorf("nested.z should be from override: %v", nested["z"])
	}
}

func TestInstallConfig(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	createContent(t, sourceDir)

	// Write opencode.json in source
	writeJSON(t, filepath.Join(sourceDir, "opencode.json"), map[string]interface{}{
		"model": "claude-3.5-sonnet",
	})

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Config should have been copied
	configPath := filepath.Join(targetDir, "opencode.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("opencode.json not installed: %v", err)
	}

	// Should have at least one copy (config)
	totalActions := len(result.Copied) + len(result.Merged)
	if totalActions == 0 {
		t.Error("expected some copy or merge actions")
	}
}

func TestInstallConfigMerge(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	createContent(t, sourceDir)

	// Write source opencode.json
	writeJSON(t, filepath.Join(sourceDir, "opencode.json"), map[string]interface{}{
		"model": "claude-3.5-sonnet",
		"hero":  true,
	})

	// Write existing target opencode.json
	writeJSON(t, filepath.Join(targetDir, "opencode.json"), map[string]interface{}{
		"model":  "gpt-4",
		"custom": "value",
	})

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(result.Merged) == 0 {
		t.Error("config should have been merged")
	}

	// Existing value should win without force
	merged := readJSON(t, filepath.Join(targetDir, "opencode.json"))
	if merged["model"] != "gpt-4" {
		t.Errorf("model should be dest value without force: %v", merged["model"])
	}
	if merged["hero"] != true {
		t.Errorf("hero should be added from source: %v", merged["hero"])
	}
	if merged["custom"] != "value" {
		t.Errorf("custom should remain from dest: %v", merged["custom"])
	}
}

func TestSkillsNestedConversion(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	_, err := Run(opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// skills/spec-format.md -> skills/spec-format/SKILL.md
	skillDir := filepath.Join(targetDir, ".opencode", "skills", "spec-format")
	if info, err := os.Stat(skillDir); err != nil || !info.IsDir() {
		t.Error("skill should be in nested directory")
	}

	skillFile := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("SKILL.md not readable: %v", err)
	}

	if !strings.Contains(string(data), "Spec Format Skill") {
		t.Error("SKILL.md content should match source")
	}

	// skills/test-strategy.md -> skills/test-strategy/SKILL.md
	skill2 := filepath.Join(targetDir, ".opencode", "skills", "test-strategy", "SKILL.md")
	if _, err := os.Stat(skill2); err != nil {
		t.Error("second skill not installed")
	}
}

// --- Cursor installer tests ---

func TestRunCursorProject(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCursor,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run cursor failed: %v", err)
	}

	if len(result.Copied) == 0 {
		t.Error("expected files to be copied")
	}

	// Agents should be in .cursor/rules/agents/
	agentPath := filepath.Join(targetDir, ".cursor", "rules", "agents", "engineer.md")
	if _, err := os.Stat(agentPath); err != nil {
		t.Errorf("agent not installed to cursor rules: %v", err)
	}

	// Commands should be in .cursor/rules/commands/
	cmdPath := filepath.Join(targetDir, ".cursor", "rules", "commands", "design.md")
	if _, err := os.Stat(cmdPath); err != nil {
		t.Errorf("command not installed to cursor rules: %v", err)
	}

	// Skills should be flat (not nested SKILL.md) in .cursor/rules/skills/
	skillPath := filepath.Join(targetDir, ".cursor", "rules", "skills", "spec-format.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("skill not installed flat to cursor rules: %v", err)
	}

	// Should NOT have nested SKILL.md structure
	nestedSkill := filepath.Join(targetDir, ".cursor", "rules", "skills", "spec-format", "SKILL.md")
	if _, err := os.Stat(nestedSkill); err == nil {
		t.Error("cursor skills should be flat, not nested")
	}
}

func TestRunCursorGlobal(t *testing.T) {
	sourceDir := t.TempDir()
	createContent(t, sourceDir)

	home := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCursor,
		Mode:      ModeGlobal,
		Force:     false,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run cursor global failed: %v", err)
	}

	if len(result.Copied) == 0 {
		t.Error("expected files to be copied")
	}

	// Check files went to ~/.cursor/rules/
	agentPath := filepath.Join(home, ".cursor", "rules", "agents", "engineer.md")
	if _, err := os.Stat(agentPath); err != nil {
		t.Errorf("agent not installed globally for cursor: %v", err)
	}
}

func TestRunCursorNoConfig(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	// Add opencode.json — should NOT be installed for cursor
	writeJSON(t, filepath.Join(sourceDir, "opencode.json"), map[string]interface{}{
		"model": "test",
	})

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCursor,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	_, err := Run(opts)
	if err != nil {
		t.Fatalf("Run cursor failed: %v", err)
	}

	// opencode.json should NOT exist in target
	configPath := filepath.Join(targetDir, "opencode.json")
	if _, err := os.Stat(configPath); err == nil {
		t.Error("opencode.json should not be installed for cursor target")
	}
}

// --- Claude Code installer tests ---

func TestRunClaudeProject(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run claude failed: %v", err)
	}

	if len(result.Copied) == 0 {
		t.Error("expected files to be copied")
	}

	// Agents should be in .claude/agents/
	agentPath := filepath.Join(targetDir, ".claude", "agents", "engineer.md")
	if _, err := os.Stat(agentPath); err != nil {
		t.Errorf("agent not installed to .claude: %v", err)
	}

	// Commands should be in .claude/commands/
	cmdPath := filepath.Join(targetDir, ".claude", "commands", "design.md")
	if _, err := os.Stat(cmdPath); err != nil {
		t.Errorf("command not installed to .claude/commands: %v", err)
	}

	// Skills must be installed as <name>/SKILL.md per Anthropic's SKILL.md
	// format — Claude Code's Skill loader does not register flat .md files.
	skillPath := filepath.Join(targetDir, ".claude", "skills", "spec-format", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("skill not installed to .claude/skills/<name>/SKILL.md: %v", err)
	}
	// Regression guard: the legacy flat file must not coexist.
	flatPath := filepath.Join(targetDir, ".claude", "skills", "spec-format.md")
	if _, err := os.Stat(flatPath); err == nil {
		t.Errorf("legacy flat skill file still present at %s — Skill loader would ignore the nested SKILL.md if flat sibling exists", flatPath)
	}

	// Harness-native model: --target claude writes CLAUDE.md ONLY. It must
	// NOT emit AGENTS.md — a Claude-only install leaves no root file that no
	// Claude session reads.
	claudePath := filepath.Join(targetDir, "CLAUDE.md")
	info, err := os.Lstat(claudePath)
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("CLAUDE.md should be a regular file, got symlink")
	}
	data, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "hero:managed-start") {
		t.Errorf("CLAUDE.md missing versioned managed-region marker")
	}
	if !strings.Contains(body, "Spec-Driven AI Engineering") {
		t.Errorf("CLAUDE.md missing Hero section content")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "AGENTS.md")); err == nil {
		t.Errorf("AGENTS.md must NOT be created for a Claude-only install (harness-native model)")
	}
}

// TestRunClaude_CleansLegacyFlatSkills exercises the migration path for
// projects that were installed before the SKILL.md directory layout fix.
// Such projects have .claude/skills/<name>.md flat files which Claude Code
// silently ignores. Re-running install must remove the flat files and
// produce only <name>/SKILL.md directories.
func TestRunClaude_CleansLegacyFlatSkills(t *testing.T) {
	sourceDir := t.TempDir()
	createContent(t, sourceDir)

	targetDir := t.TempDir()

	// Seed the legacy state: a flat skill file from a prior buggy install.
	flatDir := filepath.Join(targetDir, ".claude", "skills")
	if err := os.MkdirAll(flatDir, 0o755); err != nil {
		t.Fatal(err)
	}
	flatFile := filepath.Join(flatDir, "spec-format.md")
	if err := os.WriteFile(flatFile, []byte("legacy flat skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		SourceDir: sourceDir,
		Mode:      ModeProject,
		Target:    TargetClaude,
		TargetDir: targetDir,
		Force:     true,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// The flat file must be gone.
	if _, err := os.Stat(flatFile); err == nil {
		t.Errorf("legacy flat skill file %s was not cleaned up by install", flatFile)
	}
	// The nested SKILL.md must exist in its place.
	nested := filepath.Join(targetDir, ".claude", "skills", "spec-format", "SKILL.md")
	if _, err := os.Stat(nested); err != nil {
		t.Errorf("nested SKILL.md not present at %s after install: %v", nested, err)
	}
}

func TestRunClaudeGlobal(t *testing.T) {
	sourceDir := t.TempDir()
	createContent(t, sourceDir)

	home := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeGlobal,
		Force:     false,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run claude global failed: %v", err)
	}

	if len(result.Copied) == 0 {
		t.Error("expected files to be copied")
	}

	// Check files went to ~/.claude/
	agentPath := filepath.Join(home, ".claude", "agents", "engineer.md")
	if _, err := os.Stat(agentPath); err != nil {
		t.Errorf("agent not installed globally for claude: %v", err)
	}

	// CLAUDE.md should be at ~/.claude/CLAUDE.md for global
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	if _, err := os.Stat(claudeMd); err != nil {
		t.Errorf("CLAUDE.md not installed globally: %v", err)
	}
}

// TestRunClaude_PreservesUserAuthoredClaudeMd asserts the harness-native
// policy for a user-authored CLAUDE.md: every byte of user content is
// preserved and Hero's managed block is inserted (same managed-region
// pattern, no symlink/shim). Under the harness-native model a Claude-only
// install writes CLAUDE.md and NOT AGENTS.md.
func TestRunClaude_PreservesUserAuthoredClaudeMd(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	existingContent := "# My Project\n\nThis is my project's Claude instructions.\nUser-authored, no Hero markers.\n"
	claudeMdPath := filepath.Join(targetDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(existingContent), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     false,
	}

	if _, err := Run(opts); err != nil {
		t.Fatalf("Run claude failed: %v", err)
	}

	data, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)

	// User content preserved verbatim.
	for _, want := range []string{"# My Project", "This is my project's Claude instructions.", "User-authored, no Hero markers."} {
		if !strings.Contains(got, want) {
			t.Errorf("user content %q missing after install\n--- got ---\n%s", want, got)
		}
	}

	// Hero managed block inserted.
	if !strings.Contains(got, "hero:managed-start") {
		t.Error("expected Hero managed-region marker")
	}
	if !strings.Contains(got, "Spec-Driven AI Engineering") {
		t.Error("expected Hero body content")
	}

	// Must NOT be a symlink.
	info, err := os.Lstat(claudeMdPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("CLAUDE.md should be a regular file with a managed block, not a symlink")
	}

	// Harness-native: a Claude-only install must NOT create AGENTS.md.
	if _, err := os.Stat(filepath.Join(targetDir, "AGENTS.md")); err == nil {
		t.Errorf("AGENTS.md must NOT be created for a Claude-only install")
	}

	// Idempotency: second run produces no further change to CLAUDE.md.
	opts.Force = true
	if _, err := Run(opts); err != nil {
		t.Fatalf("second Run failed: %v", err)
	}
	data2, _ := os.ReadFile(claudeMdPath)
	if string(data2) != got {
		t.Errorf("second install changed CLAUDE.md\n--- first ---\n%s\n--- second ---\n%s", got, string(data2))
	}
}

// TestRunClaude_UpgradesLegacyClaudeMdStub asserts that a CLAUDE.md whose
// entire content is a legacy Hero managed region (no user content) gets
// its managed region upgraded in place to the versioned form.
func TestRunClaude_UpgradesLegacyClaudeMdStub(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	legacyStub := legacyMarker + "\n# Hero — Spec-Driven Workflow\n\nlegacy content\n" + legacyMarker + "\n"
	claudeMdPath := filepath.Join(targetDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(legacyStub), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     true,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("Run claude failed: %v", err)
	}

	info, err := os.Lstat(claudeMdPath)
	if err != nil {
		t.Fatalf("CLAUDE.md missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("CLAUDE.md should be a regular file with managed block")
	}
	data, _ := os.ReadFile(claudeMdPath)
	got := string(data)
	if !strings.Contains(got, "hero:managed-start") {
		t.Error("legacy CLAUDE.md should have been upgraded to versioned form")
	}
	if strings.Contains(got, "legacy content") {
		t.Error("old hero region body should have been replaced")
	}
}

// TestRunClaude_CleansUpLegacySymlink ensures a leftover symlink from
// pre-current-policy installs gets replaced with a regular file containing
// the managed block.
func TestRunClaude_CleansUpLegacySymlink(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	claudeMdPath := filepath.Join(targetDir, "CLAUDE.md")
	agentsMdPath := filepath.Join(targetDir, "AGENTS.md")
	if err := os.WriteFile(agentsMdPath, []byte("placeholder\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("AGENTS.md", claudeMdPath); err != nil {
		t.Skip("symlinks unavailable on this host")
	}

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     true,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("Run claude failed: %v", err)
	}

	info, err := os.Lstat(claudeMdPath)
	if err != nil {
		t.Fatalf("CLAUDE.md missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("legacy symlink should have been replaced with a regular file")
	}
	data, _ := os.ReadFile(claudeMdPath)
	if !strings.Contains(string(data), "hero:managed-start") {
		t.Error("CLAUDE.md should contain the managed block after install")
	}
}

// TestRunClaude_MixedClaudeMd_KeepsUserContent_UpgradesRegion asserts that
// a CLAUDE.md with both a legacy Hero region AND user content outside it
// has its user content preserved verbatim while the legacy region is
// upgraded in place to the versioned form (same body as AGENTS.md).
func TestRunClaude_MixedClaudeMd_KeepsUserContent_UpgradesRegion(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	mixed := "# My Project\n\n" + legacyMarker + "\nold hero content\n" + legacyMarker + "\n\n# Other section\nUser content here.\n"
	claudeMdPath := filepath.Join(targetDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(mixed), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     true,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("Run claude failed: %v", err)
	}

	data, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)

	// User-authored content preserved.
	for _, want := range []string{"# My Project", "# Other section", "User content here."} {
		if !strings.Contains(got, want) {
			t.Errorf("user content %q lost\n--- got ---\n%s", want, got)
		}
	}
	// Legacy hero region body gone.
	if strings.Contains(got, "old hero content") {
		t.Error("old hero region body should have been replaced")
	}
	// Versioned managed region with the canonical body content.
	if !strings.Contains(got, "hero:managed-start") {
		t.Error("expected versioned managed region")
	}
	if !strings.Contains(got, "Spec-Driven AI Engineering") {
		t.Error("expected canonical Hero body content in CLAUDE.md")
	}
}

// TestRunClaude_NoTouchClaudeMd_LeavesEverythingAlone asserts the
// --no-touch-claude-md escape hatch: when set, Hero doesn't even add the
// import block. User accepts that Claude Code won't see Hero content via
// CLAUDE.md.
func TestRunClaude_NoTouchClaudeMd_LeavesEverythingAlone(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	userContent := "# My CLAUDE.md\n\nUntouchable user content.\n"
	claudeMdPath := filepath.Join(targetDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMdPath, []byte(userContent), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		SourceDir:       sourceDir,
		Target:          TargetClaude,
		Mode:            ModeProject,
		TargetDir:       targetDir,
		Force:           true,
		NoTouchClaudeMd: true,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("Run claude failed: %v", err)
	}

	data, _ := os.ReadFile(claudeMdPath)
	if string(data) != userContent {
		t.Errorf("--no-touch-claude-md should leave CLAUDE.md byte-identical\nbefore: %q\nafter:  %q", userContent, string(data))
	}

	// Harness-native: --target claude does not write AGENTS.md, so with
	// --no-touch-claude-md no root instruction file is written at all.
	if _, err := os.Stat(filepath.Join(targetDir, "AGENTS.md")); err == nil {
		t.Errorf("AGENTS.md must NOT be created for a Claude-only install")
	}
}

// --- helpers ---

func createContent(t *testing.T, dir string) {
	t.Helper()

	// Create agents/
	agentDir := filepath.Join(dir, "agents")
	os.MkdirAll(agentDir, 0o755)
	os.WriteFile(filepath.Join(agentDir, "engineer.md"), []byte("# Engineer Agent"), 0o644)
	os.WriteFile(filepath.Join(agentDir, "architect.md"), []byte("# Architect Agent"), 0o644)

	// Create commands/
	cmdDir := filepath.Join(dir, "commands")
	os.MkdirAll(cmdDir, 0o755)
	os.WriteFile(filepath.Join(cmdDir, "design.md"), []byte("# Design Command"), 0o644)

	// Create skills/ (flat .md files)
	skillDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "spec-format.md"), []byte("# Spec Format Skill"), 0o644)
	os.WriteFile(filepath.Join(skillDir, "test-strategy.md"), []byte("# Test Strategy Skill"), 0o644)
}

func writeJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func readJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return result
}

// ---------------------------------------------------------------------------
// MCP registration tests
// ---------------------------------------------------------------------------

func TestRegisterMCP_Cursor_Project(t *testing.T) {
	targetDir := t.TempDir()
	opts := Options{
		Target:    TargetCursor,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	if err := RegisterMCP(TargetCursor, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	configPath := filepath.Join(targetDir, ".cursor", "mcp.json")
	config := readJSON(t, configPath)

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers key missing or wrong type")
	}

	hero, ok := servers["hero"].(map[string]interface{})
	if !ok {
		t.Fatal("hero server entry missing")
	}

	if hero["command"] == nil || hero["command"] == "" {
		t.Error("command should not be empty")
	}

	args, ok := hero["args"].([]interface{})
	if !ok || len(args) != 1 {
		t.Fatalf("args should be [mcp], got %v", hero["args"])
	}
	if args[0] != "mcp" {
		t.Errorf("args = %v, want [mcp]", args)
	}
}

func TestRegisterMCP_Claude_Project(t *testing.T) {
	targetDir := t.TempDir()
	opts := Options{
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	if err := RegisterMCP(TargetClaude, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	configPath := filepath.Join(targetDir, ".mcp.json")
	config := readJSON(t, configPath)

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers key missing")
	}

	_, ok = servers["hero"].(map[string]interface{})
	if !ok {
		t.Fatal("hero server entry missing")
	}
}

func TestRegisterMCP_OpenCode_Project(t *testing.T) {
	targetDir := t.TempDir()
	opts := Options{
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	if err := RegisterMCP(TargetOpenCode, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	// OpenCode MCP config is written to opencode.json at project root
	configPath := filepath.Join(targetDir, "opencode.json")
	config := readJSON(t, configPath)

	mcp, ok := config["mcp"].(map[string]interface{})
	if !ok {
		t.Fatal("mcp key missing")
	}

	hero, ok := mcp["hero"].(map[string]interface{})
	if !ok {
		t.Fatal("hero server entry missing")
	}

	if hero["type"] != "local" {
		t.Errorf("expected type=local, got %v", hero["type"])
	}

	cmd, ok := hero["command"].([]interface{})
	if !ok || len(cmd) < 2 {
		t.Fatalf("expected command array with at least 2 elements, got %v", hero["command"])
	}
	if cmd[len(cmd)-1] != "mcp" {
		t.Errorf("expected last command arg to be 'mcp', got %v", cmd[len(cmd)-1])
	}
}

func TestRegisterMCP_OpenCode_PreservesExistingConfig(t *testing.T) {
	targetDir := t.TempDir()

	// Write existing opencode.json with model config
	existing := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
		"model":   "anthropic/claude-sonnet-4-5",
		"mcp": map[string]interface{}{
			"other-server": map[string]interface{}{
				"type":    "remote",
				"url":     "https://example.com/mcp",
				"enabled": true,
			},
		},
	}
	writeJSON(t, filepath.Join(targetDir, "opencode.json"), existing)

	opts := Options{
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	if err := RegisterMCP(TargetOpenCode, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	config := readJSON(t, filepath.Join(targetDir, "opencode.json"))

	// Model config should be preserved
	if config["model"] != "anthropic/claude-sonnet-4-5" {
		t.Error("existing model config should be preserved")
	}

	mcp := config["mcp"].(map[string]interface{})

	// Both servers should exist
	if _, ok := mcp["other-server"]; !ok {
		t.Error("existing MCP server should be preserved")
	}
	if _, ok := mcp["hero"]; !ok {
		t.Error("hero MCP server should be added")
	}
}

func TestRegisterMCP_OpenCode_CleansLegacyFormat(t *testing.T) {
	targetDir := t.TempDir()

	// Write opencode.json with legacy mcpServers key (from older hero install)
	existing := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"hero": map[string]interface{}{
				"command": "hero",
				"args":    []interface{}{"mcp"},
			},
		},
	}
	writeJSON(t, filepath.Join(targetDir, "opencode.json"), existing)

	opts := Options{
		Target:    TargetOpenCode,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	if err := RegisterMCP(TargetOpenCode, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	config := readJSON(t, filepath.Join(targetDir, "opencode.json"))

	// Legacy mcpServers key should be removed
	if _, ok := config["mcpServers"]; ok {
		t.Error("legacy mcpServers key should be removed")
	}

	// New mcp key should exist with correct format
	mcp, ok := config["mcp"].(map[string]interface{})
	if !ok {
		t.Fatal("mcp key missing")
	}

	hero, ok := mcp["hero"].(map[string]interface{})
	if !ok {
		t.Fatal("hero entry missing")
	}
	if hero["type"] != "local" {
		t.Errorf("expected type=local, got %v", hero["type"])
	}
}

func TestRegisterMCP_PreservesExisting(t *testing.T) {
	targetDir := t.TempDir()
	cursorDir := filepath.Join(targetDir, ".cursor")
	os.MkdirAll(cursorDir, 0o755)

	// Write existing MCP config with another server
	existing := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"other-server": map[string]interface{}{
				"command": "/usr/bin/other",
				"args":    []string{"--flag"},
			},
		},
	}
	writeJSON(t, filepath.Join(cursorDir, "mcp.json"), existing)

	opts := Options{
		Target:    TargetCursor,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	if err := RegisterMCP(TargetCursor, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	config := readJSON(t, filepath.Join(cursorDir, "mcp.json"))
	servers := config["mcpServers"].(map[string]interface{})

	// Both servers should exist
	if _, ok := servers["other-server"]; !ok {
		t.Error("existing server should be preserved")
	}
	if _, ok := servers["hero"]; !ok {
		t.Error("hero server should be added")
	}
}

func TestRegisterMCP_DryRun(t *testing.T) {
	targetDir := t.TempDir()
	opts := Options{
		Target:    TargetCursor,
		Mode:      ModeProject,
		TargetDir: targetDir,
		DryRun:    true,
	}

	if err := RegisterMCP(TargetCursor, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	// File should NOT exist in dry run
	configPath := filepath.Join(targetDir, ".cursor", "mcp.json")
	if _, err := os.Stat(configPath); err == nil {
		t.Error("dry run should not create files")
	}
}

func TestRunInstall_IncludesMCP(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCursor,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     true,
	}

	_, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// MCP config should be created
	configPath := filepath.Join(targetDir, ".cursor", "mcp.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("MCP config should be created by install: %v", err)
	}
}

func TestRunCodexProject(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCodex,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run codex failed: %v", err)
	}
	if len(result.Copied) == 0 {
		t.Error("expected files to be copied")
	}

	// Codex agents render as TOML (Codex requires .toml; markdown is
	// silently dropped). Source: codex-rs/core/src/config/agent_roles.rs:518-550
	agentTomlPath := filepath.Join(targetDir, ".codex", "agents", "engineer.toml")
	if _, err := os.Stat(agentTomlPath); err != nil {
		t.Errorf("agent not rendered as TOML at .codex/agents/engineer.toml: %v", err)
	}
	// Markdown form is dead bytes — must NOT be present.
	if _, err := os.Stat(filepath.Join(targetDir, ".codex", "agents", "engineer.md")); err == nil {
		t.Error("dead .codex/agents/engineer.md should not be installed (Codex only reads .toml)")
	}
	// No commands loader exists in Codex — Hero installs Hero commands as
	// skills under .agents/skills/ instead. (See target_codex.go.)
	if _, err := os.Stat(filepath.Join(targetDir, ".codex", "commands")); err == nil {
		t.Error(".codex/commands should not exist (Codex has no command loader)")
	}
	// Skills land at .agents/skills/<name>/SKILL.md (preferred location;
	// also picked up by OpenCode's cross-tool fallback).
	skillPath := filepath.Join(targetDir, ".agents", "skills", "spec-format", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("skill not installed to .agents/skills/<name>/SKILL.md: %v", err)
	}

	// AGENTS.md at project root
	agentsMd := filepath.Join(targetDir, "AGENTS.md")
	data, err := os.ReadFile(agentsMd)
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	if !strings.Contains(string(data), "hero:managed") {
		t.Error("AGENTS.md missing hero marker")
	}

	// .codex/hooks.json should have Stop hook
	hooksPath := filepath.Join(targetDir, ".codex", "hooks.json")
	hooksData, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf(".codex/hooks.json not created: %v", err)
	}
	if !strings.Contains(string(hooksData), "hero next checkpoint") {
		t.Error("hooks.json missing hero checkpoint command")
	}
}

func TestRunCodexGlobal(t *testing.T) {
	sourceDir := t.TempDir()
	createContent(t, sourceDir)

	home := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCodex,
		Mode:      ModeGlobal,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run codex global failed: %v", err)
	}
	if len(result.Copied) == 0 {
		t.Error("expected files to be copied")
	}

	agentPath := filepath.Join(home, ".codex", "agents", "engineer.toml")
	if _, err := os.Stat(agentPath); err != nil {
		t.Errorf("agent not installed globally to ~/.codex as TOML: %v", err)
	}
	// Skills land at ~/.agents/skills/ in global mode.
	skillPath := filepath.Join(home, ".agents", "skills", "spec-format", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("skill not installed globally to ~/.agents/skills/: %v", err)
	}
}

func TestRegisterMCP_Codex_Project(t *testing.T) {
	targetDir := t.TempDir()
	opts := Options{
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	if err := RegisterMCP(TargetCodex, opts); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	configPath := filepath.Join(targetDir, ".codex", "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf(".codex/config.toml not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[mcp_servers.hero]") {
		t.Error("config.toml missing [mcp_servers.hero]")
	}
	if !strings.Contains(content, "hero:managed") {
		t.Error("config.toml missing hero:managed marker")
	}
}

func TestRegisterMCP_Codex_Idempotent(t *testing.T) {
	targetDir := t.TempDir()
	opts := Options{
		Mode:      ModeProject,
		TargetDir: targetDir,
	}

	// Run twice — should not duplicate the block
	for i := 0; i < 2; i++ {
		if err := RegisterMCP(TargetCodex, opts); err != nil {
			t.Fatalf("RegisterMCP pass %d: %v", i, err)
		}
	}

	configPath := filepath.Join(targetDir, ".codex", "config.toml")
	data, _ := os.ReadFile(configPath)
	content := string(data)

	count := strings.Count(content, "[mcp_servers.hero]")
	if count != 1 {
		t.Errorf("expected exactly one [mcp_servers.hero] section, got %d", count)
	}
}
