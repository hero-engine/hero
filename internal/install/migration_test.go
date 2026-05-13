package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// migration_test.go — coverage for the upgrade-from-prior-release
// path defined in harness-install-paths-match-loaders. Verifies:
//
//   - Legacy install layouts (e.g. Codex .codex/agents/*.md from
//     before the TOML switch) get cleaned up automatically.
//   - The corrected layout (e.g. .codex/agents/*.toml) lands in its
//     place.
//   - User-edited content in legacy locations is preserved with a
//     warning, NOT silently deleted.

// TestMigration_CodexLegacyLayoutCleanup simulates a project with the
// pre-fix Codex install (markdown agents at .codex/agents/, no
// .codex/commands cleanup) and asserts that re-running install on the
// corrected target produces the new layout AND removes the dead bytes.
func TestMigration_CodexLegacyLayoutCleanup(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed legacy Codex install: markdown agents at .codex/agents/, a
	// stray command file, AND a .codex/skills/ entry. All bytes match
	// canonical so cleanup recognizes them as Hero-authored.
	legacyAgent := filepath.Join(h.TargetDir, ".codex", "agents", "engineer.md")
	mustMirrorCanonical(t, h, "agents/engineer.md", legacyAgent)
	legacyCmd := filepath.Join(h.TargetDir, ".codex", "commands", "design.md")
	mustMirrorCanonical(t, h, "commands/design.md", legacyCmd)
	legacySkillDir := filepath.Join(h.TargetDir, ".codex", "skills", "spec-format")
	if err := os.MkdirAll(legacySkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustMirrorCanonical(t, h, "skills/spec-format.md", filepath.Join(legacySkillDir, "SKILL.md"))

	h.Run(TargetCodex, nil)

	// Old markdown at .codex/agents/engineer.md must be gone.
	if _, err := os.Stat(legacyAgent); err == nil {
		t.Errorf("legacy %s should have been cleaned up after migration", legacyAgent)
	}
	// Old commands dir must be gone (no loader exists for it in Codex).
	if _, err := os.Stat(filepath.Join(h.TargetDir, ".codex", "commands")); err == nil {
		t.Error(".codex/commands/ should have been cleaned up after migration")
	}
	// New TOML form lands at .codex/agents/engineer.toml.
	h.mustBeRegularFile(".codex/agents/engineer.toml")
	// Skills land at the canonical-symlink location .agents/skills/.
	h.mustBeRegularFile(".agents/skills/spec-format/SKILL.md")
}

// TestMigration_CopilotLegacyLayoutCleanup simulates the pre-fix
// Copilot install (.github/copilot/{agents,commands,skills}/) and
// asserts those dead-byte locations get cleaned up; new
// .github/prompts/ + .github/skills/ paths land instead.
func TestMigration_CopilotLegacyLayoutCleanup(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed legacy Copilot install at the no-loader paths.
	legacyAgent := filepath.Join(h.TargetDir, ".github", "copilot", "agents", "engineer.md")
	mustMirrorCanonical(t, h, "agents/engineer.md", legacyAgent)
	legacyCmd := filepath.Join(h.TargetDir, ".github", "copilot", "commands", "design.md")
	mustMirrorCanonical(t, h, "commands/design.md", legacyCmd)
	legacySkillDir := filepath.Join(h.TargetDir, ".github", "copilot", "skills", "spec-format")
	if err := os.MkdirAll(legacySkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustMirrorCanonical(t, h, "skills/spec-format.md", filepath.Join(legacySkillDir, "SKILL.md"))

	h.Run(TargetCopilot, nil)

	// Legacy .github/copilot/* subdirs must be gone.
	if _, err := os.Stat(filepath.Join(h.TargetDir, ".github", "copilot", "agents")); err == nil {
		t.Error(".github/copilot/agents should have been cleaned up after migration")
	}
	if _, err := os.Stat(filepath.Join(h.TargetDir, ".github", "copilot", "commands")); err == nil {
		t.Error(".github/copilot/commands should have been cleaned up after migration")
	}
	if _, err := os.Stat(filepath.Join(h.TargetDir, ".github", "copilot", "skills")); err == nil {
		t.Error(".github/copilot/skills should have been cleaned up after migration")
	}

	// New paths land instead.
	h.mustBeRegularFile(".github/prompts/agents/engineer.prompt.md")
	h.mustBeRegularFile(".github/prompts/commands/design.prompt.md")
	h.mustBeRegularFile(".github/skills/spec-format/SKILL.md")
}

// TestMigration_PreservesUserEditedLegacyContent confirms that
// user-edited files in legacy install locations are NOT auto-deleted
// during cleanup — only Hero-authored byte-equal content is removed.
// The user gets a warning telling them to review and delete manually.
func TestMigration_PreservesUserEditedLegacyContent(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed a legacy Codex agent with USER-EDITED content (does not
	// match canonical bytes).
	userEdited := filepath.Join(h.TargetDir, ".codex", "agents", "my-custom.md")
	if err := os.MkdirAll(filepath.Dir(userEdited), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userEdited, []byte("# My custom Codex agent\nUser-authored content."), 0o644); err != nil {
		t.Fatal(err)
	}

	h.Run(TargetCodex, nil)

	// User content must still exist after migration.
	if _, err := os.Stat(userEdited); err != nil {
		t.Errorf("user-edited %s should NOT have been deleted: %v", userEdited, err)
	}
	// Confirm content unchanged.
	data, _ := os.ReadFile(userEdited)
	if !strings.Contains(string(data), "User-authored content") {
		t.Errorf("user content was modified: %q", string(data))
	}
	// New TOML form should still land alongside the user file.
	h.mustBeRegularFile(".codex/agents/engineer.toml")
}

// mustMirrorCanonical writes srcPath (read from the harness's source
// FS) verbatim to dst. Used to seed legacy install fixtures with
// content that matches canonical bytes — so the cleanup logic
// recognizes them as Hero-authored.
func mustMirrorCanonical(t *testing.T, h *installHarness, srcPath, dst string) {
	t.Helper()
	full := filepath.Join(h.SourceDir, srcPath)
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read source %s: %v", srcPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}
