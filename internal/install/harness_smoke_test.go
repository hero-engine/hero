package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// harness_smoke_test.go — smoke coverage for the installHarness helper and
// the new install-state.json scaffolding. These tests are intentionally
// minimal: they catch breakage in the harness itself (so all the
// per-feature tests that build on it have a known-good baseline) plus
// confirm the install-state file lifecycle.

func TestHarness_SmokeClaude(t *testing.T) {
	h := newInstallHarness(t)
	res := h.Run(TargetClaude, nil)

	// Agents installed flat under .claude/agents/.
	h.mustBeRegularFile(".claude/agents/engineer.md")
	h.mustBeRegularFile(".claude/agents/reviewer.md")

	// Subagent registration contract: every installed agent file must
	// carry `name:` and `description:` frontmatter so Claude Code's Task
	// tool actually exposes them as subagent_type values. Regression
	// guard for claude-subagent-frontmatter-registration.
	h.mustBeRegisterableSubagent(".claude/agents/engineer.md", "engineer")
	h.mustBeRegisterableSubagent(".claude/agents/reviewer.md", "reviewer")

	// Commands installed flat.
	h.mustBeRegularFile(".claude/commands/design.md")
	h.mustBeRegularFile(".claude/commands/deliver.md")

	// Skills installed as <name>/SKILL.md (regression guard from the
	// recent flat→nested fix).
	h.mustBeRegularFile(".claude/skills/spec-format/SKILL.md")
	h.mustBeRegularFile(".claude/skills/test-strategy/SKILL.md")
	h.mustNotExist(".claude/skills/spec-format.md")

	// Under P1: both AGENTS.md and CLAUDE.md are regular files with the
	// same managed-block treatment. Same body content, different roots so
	// every harness sees Hero regardless of which file it reads.
	h.mustBeRegularFile("AGENTS.md")
	h.mustContain("AGENTS.md", "hero:managed-start")
	h.mustBeRegularFile("CLAUDE.md")
	h.mustContain("CLAUDE.md", "hero:managed-start")

	// Sanity: result records copies.
	if len(res.Copied) == 0 {
		t.Error("expected Run to record copied files")
	}
}

func TestHarness_SmokeOpenCode(t *testing.T) {
	h := newInstallHarness(t)
	h.Run(TargetOpenCode, nil)

	h.mustBeRegularFile(".opencode/agents/engineer.md")
	h.mustBeRegularFile(".opencode/commands/design.md")
	h.mustBeRegularFile(".opencode/skills/spec-format/SKILL.md")
	h.mustBeRegularFile("opencode.json")
	h.mustBeRegularFile("AGENTS.md")
}

func TestHarness_SmokeCursor(t *testing.T) {
	h := newInstallHarness(t)
	h.Run(TargetCursor, nil)

	// No .hero/ workspace was pre-initialized → fallback to direct
	// rendering. Cursor uses flat skills in that fallback.
	h.mustBeRegularFile(".cursor/rules/agents/engineer.md")
	h.mustBeRegularFile(".cursor/rules/commands/design.md")
	h.mustBeRegularFile(".cursor/rules/skills/spec-format.md")
}

func TestHarness_CanonicalAndSymlinks(t *testing.T) {
	h := newInstallHarness(t)

	// Pre-init the .hero/ workspace so P2 canonical-and-symlink mode kicks
	// in for the harness target.
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	h.Run(TargetClaude, nil)

	// Canonical content materialized.
	h.mustBeRegularFile(".hero/agents/engineer.md")
	h.mustBeRegularFile(".hero/commands/design.md")
	h.mustBeRegularFile(".hero/skills/spec-format/SKILL.md")

	// Harness dirs are symlinks pointing at canonical.
	for _, link := range []string{".claude/agents", ".claude/commands", ".claude/skills"} {
		target := h.mustBeSymlink(link)
		if !strings.Contains(target, ".hero") {
			t.Errorf("%s symlink should point into .hero/, got %q", link, target)
		}
	}

	// Files visible through the symlink — proves the symlink resolves.
	h.mustBeRegularFile(".claude/agents/engineer.md")
	h.mustBeRegularFile(".claude/skills/spec-format/SKILL.md")
}

// TestHarness_MigratesLegacyDirToSymlink verifies the P2 migration path:
// a legacy install left .claude/agents/ as a regular directory full of
// rendered copies; running install --force migrates that to a symlink
// pointing at canonical content (no files lost — canonical has the same
// content materialized from the same source).
func TestHarness_MigratesLegacyDirToSymlink(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Simulate the legacy state: regular directory with a stale copy.
	legacyAgentsDir := filepath.Join(h.TargetDir, ".claude", "agents")
	if err := os.MkdirAll(legacyAgentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyAgentsDir, "stale.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Without --force, install should refuse to clobber the legacy dir.
	if _, err := Run(Options{
		SourceDir: h.SourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
		Force:     false,
	}); err == nil {
		t.Error("expected install without --force to refuse legacy directory migration")
	}

	// With --force, install migrates the dir to a symlink.
	h.Run(TargetClaude, func(o *Options) { o.Force = true })

	target := h.mustBeSymlink(".claude/agents")
	if !strings.Contains(target, ".hero") {
		t.Errorf("expected symlink to canonical, got %q", target)
	}
	// Files visible through the symlink come from canonical, not the
	// removed stale dir.
	h.mustBeRegularFile(".claude/agents/engineer.md")
	h.mustNotExist(".claude/agents/stale.md")
}

// TestHarness_MultipleTargetsShareCanonical asserts that installing
// multiple harness targets in the same project all converge on the same
// canonical content tree — agents/commands/skills are physically materialized
// once in .hero/, and each harness directory is a separate symlink into it.
func TestHarness_MultipleTargetsShareCanonical(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	h.Run(TargetClaude, nil)
	h.Run(TargetOpenCode, func(o *Options) { o.Force = true })
	h.Run(TargetCursor, func(o *Options) { o.Force = true })

	// Single canonical materialization.
	h.mustBeRegularFile(".hero/agents/engineer.md")

	// Each harness has its own symlink, but they all resolve to the same
	// canonical file.
	for _, link := range []string{".claude/agents", ".opencode/agents", ".cursor/rules/agents"} {
		h.mustBeSymlink(link)
		h.mustBeRegularFile(filepath.Join(link, "engineer.md"))
	}
}

func TestInstallState_LifecycleAfterClaudeInstall(t *testing.T) {
	h := newInstallHarness(t)

	// Pre-create .hero/ so install-state has a place to land.
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	h.Run(TargetClaude, func(o *Options) {
		o.Version = "v0.0.0-test"
	})

	statePath := InstallStatePath(h.TargetDir)
	if statePath == "" {
		t.Fatal("expected install-state path to resolve once .hero/ exists")
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("install-state.json should exist after install: %v", err)
	}

	st, err := ReadInstallState(h.TargetDir)
	if err != nil {
		t.Fatalf("ReadInstallState: %v", err)
	}
	if st.SchemaVersion != installStateSchemaVersion {
		t.Errorf("schema version: got %d want %d", st.SchemaVersion, installStateSchemaVersion)
	}
	if st.HeroVersion != "v0.0.0-test" {
		t.Errorf("hero version not recorded: got %q", st.HeroVersion)
	}
	claudeState, ok := st.Targets["claude"]
	if !ok {
		t.Fatalf("install-state missing claude target entry; got %v", st.Targets)
	}
	// Under P2 the recorded mode reflects the resolved install path —
	// "symlink" when the host supports it, "rendered" otherwise. On
	// modern CI hosts (Linux, macOS) it's almost always "symlink".
	if claudeState.Mode != "symlink" && claudeState.Mode != "rendered" {
		t.Errorf("claude install mode: got %q want one of {symlink, rendered}", claudeState.Mode)
	}
	if claudeState.InstalledAt == "" || claudeState.LastUpdatedAt == "" {
		t.Error("install-state timestamps not populated")
	}
}

func TestInstallState_NoHeroDirIsNoop(t *testing.T) {
	// When .hero/ doesn't exist (e.g. installing into a project that
	// hasn't been initialized yet), install-state.json should not be
	// created — InstallStatePath() returns "" and writes are skipped.
	h := newInstallHarness(t)
	h.Run(TargetClaude, nil)

	if InstallStatePath(h.TargetDir) != "" {
		t.Errorf("InstallStatePath should return empty when .hero/ absent")
	}
	if _, err := os.Stat(filepath.Join(h.TargetDir, ".hero", "install-state.json")); err == nil {
		t.Error("install-state.json should not be created without .hero/")
	}
}
