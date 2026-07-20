package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/snapshot"
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

	// Harness-native: --target claude writes CLAUDE.md ONLY. AGENTS.md must
	// NOT be created — Claude Code reads CLAUDE.md, so a Claude-only install
	// leaves no root file no harness reads.
	h.mustBeRegularFile("CLAUDE.md")
	h.mustContain("CLAUDE.md", "hero:managed-start")
	h.mustNotExist("AGENTS.md")
	// The closing-gate terminal contract lives in the shared body, so it
	// reaches the always-loaded root file. Assert it on the Claude CLAUDE.md.
	h.mustContain("CLAUDE.md", "Finish the closing gate before yielding")

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

func TestHarness_DesignClosingUsesProgressiveACDisclosureForAllTargets(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		path   string
	}{
		{"opencode", TargetOpenCode, ".opencode/commands/design.md"},
		{"cursor", TargetCursor, ".cursor/rules/commands/design.md"},
		{"claude", TargetClaude, ".claude/commands/design.md"},
		{"copilot", TargetCopilot, ".github/prompts/commands/design.prompt.md"},
		{"codex", TargetCodex, ".agents/skills/command-design/SKILL.md"},
		{"generic", TargetGeneric, ".ai/commands/design.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := runOverlayInstall(t, tt.target, "engineering")
			content, err := os.ReadFile(filepath.Join(dir, tt.path))
			if err != nil {
				t.Fatalf("read installed design command: %v", err)
			}
			body := string(content)
			if !strings.Contains(body, "Use progressive disclosure for acceptance criteria") {
				t.Errorf("%s missing progressive-disclosure contract", tt.path)
			}
			if strings.Contains(body, "Show the acceptance criteria in your closing message") {
				t.Errorf("%s retains obsolete unconditional-table contract", tt.path)
			}
		})
	}
}

func TestHarness_DiagnosePullsCredentialSafeTrackerDescriptionForAllTargets(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		path   string
	}{
		{"opencode", TargetOpenCode, ".opencode/commands/diagnose.md"},
		{"cursor", TargetCursor, ".cursor/rules/commands/diagnose.md"},
		{"claude", TargetClaude, ".claude/commands/diagnose.md"},
		{"copilot", TargetCopilot, ".github/prompts/commands/diagnose.prompt.md"},
		{"codex", TargetCodex, ".agents/skills/command-diagnose/SKILL.md"},
		{"generic", TargetGeneric, ".ai/commands/diagnose.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := runOverlayInstall(t, tt.target, "engineering")
			content, err := os.ReadFile(filepath.Join(dir, tt.path))
			if err != nil {
				t.Fatalf("read installed diagnose command: %v", err)
			}
			body := string(content)
			if !strings.Contains(body, "hero sync evidence <slug>") {
				t.Errorf("%s missing full-ticket evidence preflight", tt.path)
			}
			if !strings.Contains(body, "Never claim") {
				t.Errorf("%s missing remote-emptiness truth constraint", tt.path)
			}
		})
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

// TestHarness_InstalledContentSurvivesOrdinaryCommands pins the invariant
// that the codex-install-broken loop kept missing: install is NOT a
// terminal state.
//
// Every prior check of the root instruction file — TestHarness_SmokeCodex
// above, and the manual "verified in AGENTS.md after install" in that
// spec's completion ledger — ran immediately after install, which is the
// one moment the file is guaranteed correct because install just wrote
// it. A different writer then destroyed the file on the next ordinary
// command, and the suite stayed green for two months while the repo's
// own AGENTS.md sat at a 7-line stub.
//
// So: install, then run an ordinary command, then assert the file is
// still complete. Any writer that eats installed content fails here,
// regardless of which package it lives in.
func TestHarness_InstalledContentSurvivesOrdinaryCommands(t *testing.T) {
	// Every install target, per the harness-changes-cover-all-targets
	// tripwire. This bug was filed as "Codex install is broken", but the
	// eraser hit the root file of all five AGENTS.md-reading harnesses —
	// only claude escaped, and only because the pointer path hardcoded the
	// literal string "AGENTS.md" rather than resolving
	// nativeInstructionFile(target). Covering one target here is how the
	// blast radius got misjudged the first time.
	cases := []struct {
		name     string
		target   Target
		rootFile string
	}{
		{"claude", TargetClaude, "CLAUDE.md"},
		{"codex", TargetCodex, "AGENTS.md"},
		{"opencode", TargetOpenCode, "AGENTS.md"},
		{"cursor", TargetCursor, "AGENTS.md"},
		{"copilot", TargetCopilot, "AGENTS.md"},
		{"generic", TargetGeneric, "AGENTS.md"},
	}

	// A distinctive line from the shared managed body — it reaches every
	// target's root file, so its absence means the region was clobbered
	// rather than merely reordered.
	const mustKeep = "Finish the closing gate before yielding"

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newInstallHarness(t)
			heroDir := filepath.Join(h.TargetDir, ".hero")
			if err := os.MkdirAll(heroDir, 0o755); err != nil {
				t.Fatal(err)
			}
			h.Run(tc.target, nil)

			path := filepath.Join(h.TargetDir, tc.rootFile)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s missing right after install: %v", tc.rootFile, err)
			}
			if !strings.Contains(string(before), mustKeep) {
				t.Fatalf("%s lacks %q even at install time — test fixture is wrong",
					tc.rootFile, mustKeep)
			}

			// An ordinary command. `hero snapshot --project` and
			// `hero next checkpoint` both drive this projector; it is the
			// writer that ate AGENTS.md on May 31 and Jun 9 2026.
			if _, err := snapshot.Project(snapshot.ProjectOptions{
				ProjectRoot: h.TargetDir,
				HeroDir:     heroDir,
				ProjectName: "smoke",
				NextMDPath:  filepath.Join(heroDir, "NEXT.md"),
			}); err != nil {
				t.Fatalf("snapshot.Project: %v", err)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s gone after an ordinary command: %v", tc.rootFile, err)
			}
			if !strings.Contains(string(after), mustKeep) {
				t.Errorf("%s lost %q after an ordinary hero command.\n"+
					"install content was destroyed by a later writer — this is the "+
					"codex-install-broken regression.\nsize %d -> %d\n--- after ---\n%s",
					tc.rootFile, mustKeep, len(before), len(after), after)
			}
			if string(before) != string(after) {
				t.Errorf("%s was modified by an ordinary hero command (size %d -> %d); "+
					"only install may write this file's managed region",
					tc.rootFile, len(before), len(after))
			}

			// Ties "content survives" to "and we'd notice if it didn't":
			// after the ordinary command, the install-integrity oracle must
			// also report clean. If a future writer eats the region, the
			// byte-equality above catches it in CI and CheckIntegrity is
			// what catches it in a user's repo via `hero check`.
			findings, err := CheckIntegrity(h.TargetDir, Options{SourceDir: h.SourceDir})
			if err != nil {
				t.Fatalf("CheckIntegrity after ordinary command: %v", err)
			}
			if len(findings) != 0 {
				t.Errorf("CheckIntegrity should report clean after an ordinary command, got %+v", findings)
			}
		})
	}
}
