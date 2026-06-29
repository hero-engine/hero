package install

import (
	"os"
	"path/filepath"
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

	// Commands installed flat.
	h.mustBeRegularFile(".claude/commands/design.md")
	h.mustBeRegularFile(".claude/commands/deliver.md")

	// Skills installed as <name>/SKILL.md (regression guard from the
	// recent flat→nested fix).
	h.mustBeRegularFile(".claude/skills/spec-format/SKILL.md")
	h.mustBeRegularFile(".claude/skills/test-strategy/SKILL.md")
	h.mustNotExist(".claude/skills/spec-format.md")

	// Per-target HarnessContract enforcement (install-contract-registry-foundation).
	// Each installed file must satisfy the contract declared in
	// internal/install/contracts.go for its (target, kind) cell.
	// Subsumes the earlier mustBeRegisterableSubagent assertions and
	// extends coverage to commands and skills.
	h.mustSatisfyContract(TargetClaude, KindAgents)
	h.mustSatisfyContract(TargetClaude, KindCommands)
	h.mustSatisfyContract(TargetClaude, KindSkills)

	// Under P1: both AGENTS.md and CLAUDE.md are regular files with the
	// same managed-block treatment. Same body content, different roots so
	// every harness sees Hero regardless of which file it reads.
	h.mustBeRegularFile("AGENTS.md")
	h.mustContain("AGENTS.md", "hero:managed-start")
	h.mustBeRegularFile("CLAUDE.md")
	h.mustContain("CLAUDE.md", "hero:managed-start")
	// The closing-gate terminal contract lives in the shared body, so it
	// reaches the always-loaded root file for every target — not just
	// Codex's AGENTS.md. Assert it on the Claude CLAUDE.md path.
	h.mustContain("CLAUDE.md", "Finish the closing gate before yielding")
	h.mustContain("AGENTS.md", "Finish the closing gate before yielding")

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

// TestHarness_SmokeCodex covers the corrected Codex install layout
// from harness-install-paths-match-loaders:
//   - Agents render as TOML at .codex/agents/<n>.toml
//     (markdown is dead bytes — Codex only reads .toml files)
//   - Commands emitted as skills at .agents/skills/command-<name>/SKILL.md
//     (Codex has no command loader — skills are the bridge)
//   - Skills land at .agents/skills/ (cross-tool standard) — no
//     longer at .codex/skills/
//   - AGENTS.md has a Codex-specific workflow execution section
//   - AGENTS.md and .codex/hooks.json land at root
func TestHarness_SmokeCodex(t *testing.T) {
	h := newInstallHarness(t)
	// Pre-init the .hero/ workspace so installInstructionsMd /
	// installAgentsMd run (they no-op without a workspace).
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := h.Run(TargetCodex, nil)

	h.mustBeRegularFile(".codex/agents/engineer.toml")
	h.mustBeRegularFile(".codex/agents/reviewer.toml")
	h.mustNotExist(".codex/agents/engineer.md")
	h.mustNotExist(".codex/commands")

	h.mustBeRegularFile(".agents/skills/spec-format/SKILL.md")
	h.mustBeRegularFile(".agents/skills/test-strategy/SKILL.md")

	// Commands emitted as Codex-loadable skills with execution preamble.
	h.mustBeRegularFile(".agents/skills/command-deliver/SKILL.md")
	h.mustBeRegularFile(".agents/skills/command-design/SKILL.md")
	h.mustContain(".agents/skills/command-deliver/SKILL.md", "This is a Hero workflow for Codex")
	h.mustContain(".agents/skills/command-deliver/SKILL.md", "purpose: command-workflow")

	h.mustBeRegularFile("AGENTS.md")
	h.mustContain("AGENTS.md", "hero:managed-start")
	// Target-aware AGENTS.md section tells Codex how to execute workflows.
	h.mustContain("AGENTS.md", "Running Hero Workflows in Codex")
	h.mustContain("AGENTS.md", "command-deliver/SKILL.md")
	// Terminal-state contract: a delivery is not finished until the closing
	// gate runs — the agent must run it, not yield with it unrun.
	h.mustContain("AGENTS.md", "not finished until its closing gate runs")
	h.mustContain("AGENTS.md", "run it now instead")
	h.mustBeRegularFile(".codex/hooks.json")

	if len(res.Copied) == 0 {
		t.Error("expected Run to record copied files")
	}
}

// TestHarness_SmokeCopilot covers the corrected Copilot install layout:
//   - Agents render as .prompt.md under .github/prompts/agents/
//   - Commands render as .prompt.md under .github/prompts/commands/
//   - Skills land at .github/skills/<n>/SKILL.md (Copilot's recognized
//     skill folder; .github/copilot/skills/ was never read)
//   - .github/copilot/{agents,commands,skills}/ MUST be absent
//   - .github/copilot-instructions.md is the workspace instruction file
func TestHarness_SmokeCopilot(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := h.Run(TargetCopilot, nil)

	// New paths Copilot actually reads.
	h.mustBeRegularFile(".github/prompts/agents/engineer.prompt.md")
	h.mustBeRegularFile(".github/prompts/commands/design.prompt.md")
	h.mustBeRegularFile(".github/skills/spec-format/SKILL.md")
	h.mustBeRegularFile(".github/copilot-instructions.md")

	// Old dead-bytes locations must NOT be installed.
	h.mustNotExist(".github/copilot/agents")
	h.mustNotExist(".github/copilot/commands")
	h.mustNotExist(".github/copilot/skills")

	if len(res.Copied) == 0 {
		t.Error("expected Run to record copied files")
	}
}

// TestHarness_SmokeGeneric covers the catch-all Generic target. .ai/
// is Hero convention with no consuming loader; layout matches the
// canonical kinds.
func TestHarness_SmokeGeneric(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := h.Run(TargetGeneric, nil)

	h.mustBeRegularFile(".ai/agents/engineer.md")
	h.mustBeRegularFile(".ai/commands/design.md")
	h.mustBeRegularFile(".ai/skills/spec-format/SKILL.md")
	h.mustBeRegularFile("AGENTS.md")

	if len(res.Copied) == 0 {
		t.Error("expected Run to record copied files")
	}
}

// TestHarness_RenderDirect_NoCanonicalMirror confirms the
// render-direct-install architecture: agents/commands/skills land
// directly at each harness's documented destination, and there is
// NO `.hero/{agents,commands,skills}/` canonical mirror or any
// symlinks at the harness destinations.
func TestHarness_RenderDirect_NoCanonicalMirror(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	h.Run(TargetClaude, nil)

	// Harness destinations have real files (not symlinks).
	for _, kindDir := range []string{".claude/agents", ".claude/commands", ".claude/skills"} {
		info, err := os.Lstat(filepath.Join(h.TargetDir, kindDir))
		if err != nil {
			t.Errorf("expected %s to exist as a directory: %v", kindDir, err)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s should NOT be a symlink under render-direct architecture", kindDir)
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", kindDir)
		}
	}
	h.mustBeRegularFile(".claude/agents/engineer.md")
	h.mustBeRegularFile(".claude/commands/design.md")
	h.mustBeRegularFile(".claude/skills/spec-format/SKILL.md")

	// .hero/{agents,commands,skills}/ canonical mirror must NOT exist.
	for _, mirror := range []string{".hero/agents", ".hero/commands", ".hero/skills"} {
		if _, err := os.Stat(filepath.Join(h.TargetDir, mirror)); err == nil {
			t.Errorf(".hero/ canonical mirror %s should not be created under render-direct architecture", mirror)
		}
	}
}

// TestHarness_RenderDirect_MultiTargetIndependent confirms that
// installing multiple harness targets in the same project produces
// independent rendered files per harness — no shared canonical, no
// symlinks. The auto-sync feature keeps them at the same binary
// version (covered by separate auto-sync tests).
func TestHarness_RenderDirect_MultiTargetIndependent(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	h.Run(TargetClaude, nil)
	h.Run(TargetOpenCode, nil)
	h.Run(TargetCursor, nil)

	// Each harness has its own rendered file.
	for _, file := range []string{
		".claude/agents/engineer.md",
		".opencode/agents/engineer.md",
		".cursor/rules/agents/engineer.md",
	} {
		h.mustBeRegularFile(file)
		info, _ := os.Lstat(filepath.Join(h.TargetDir, file))
		if info != nil && info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s should be a regular file, got symlink", file)
		}
	}

	// .hero/agents/ canonical mirror must NOT exist.
	if _, err := os.Stat(filepath.Join(h.TargetDir, ".hero", "agents")); err == nil {
		t.Error(".hero/agents/ canonical mirror should not be created under render-direct architecture")
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
