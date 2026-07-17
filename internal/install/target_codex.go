package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Codex CLI loader paths (source-verified against
// github.com/openai/codex):
//
//	Project agents:  .codex/agents/<name>.toml         (TOML, requires `developer_instructions`)
//	Global agents:   ~/.codex/agents/<name>.toml
//	  Source: codex-rs/core/src/config/agent_roles.rs:73-96, 217-225, 518-550
//
//	Project skills:  .codex/skills/<name>/SKILL.md      (config-layer walk)
//	                 .agents/skills/<name>/SKILL.md     (preferred, repo walk)
//	Global skills:   ~/.agents/skills/<name>/SKILL.md   (current)
//	                 ~/.codex/skills/<name>/SKILL.md    (deprecated)
//	  Source: codex-rs/core-skills/src/loader.rs:106-110, 270-374
//
//	Commands: NO LOADER at any scope. SlashCommand is a built-in
//	enum (codex-rs/tui/src/slash_command.rs:8-76). Codex's own
//	migration tooling converts .claude/commands/* into skills under
//	.agents/skills/.
//
//	AGENTS.md:       project root + ~/.codex/AGENTS.md (concatenated)
//	  Source: docs/agents_md.md, https://developers.openai.com/codex/guides/agents-md
//
//	Hooks:           .codex/hooks.json or .codex/config.toml [[hooks.X]]
//	  Source: https://developers.openai.com/codex/hooks
//
// Hero implementation:
//   - Renders canonical agents (markdown) into TOML via
//     renderCodexAgentToml. Cannot symlink — format differs.
//   - Writes canonical skills (a rendered copy, not a symlink) into
//     .agents/skills/ — the preferred repo-walk path the Codex loader
//     reads. Does not write the deprecated .codex/skills/ config-layer
//     path.
//   - Installs commands as skills under .agents/skills/command-<name>/
//     (Codex has no command loader — SlashCommand is a built-in enum).
//   - Cleans up dead bytes from prior installs: ONLY the pre-.toml *.md
//     dead-bytes at .codex/agents/ (that dir is the LIVE loader dir and
//     holds user .toml agents, so the cleanup is .md-scoped by
//     construction — dropped Hero .toml agents are pruned provenance-safely
//     by pruneStaleFiles), and the whole of .codex/commands/* (no loader
//     at any scope; nothing repopulates it).
//   - Prunes skill dirs at the .agents/skills dest whose canonical source
//     is gone, so a renamed command or skill doesn't keep loading forever
//     (see prune.go for the provenance rules).

func runCodex(opts Options) (*Result, error) {
	destBase, err := resolveCodexPaths(opts)
	if err != nil {
		return nil, err
	}

	result := &Result{}

	// Cleanup of dead bytes from prior install layouts.
	// .codex/agents is the LIVE dir Codex loads <name>.toml agents from, so
	// it legitimately holds user files. Remove ONLY pre-.toml *.md dead-bytes
	// here; dropped Hero .toml agents are pruned provenance-safely by
	// pruneStaleFiles (see prune.go / manifest-driven-prune), and user files
	// (a hand-authored .toml agent, a subdir) are left untouched.
	if err := removeLegacyDirMatching(opts, filepath.Join(destBase, "agents"), isLegacyCodexAgentMarkdown); err != nil {
		return nil, fmt.Errorf("cleanup .codex/agents: %w", err)
	}
	// .codex/commands has no loader at any scope; nothing repopulates it, so
	// wholesale removal is correct.
	if err := removeLegacyDir(opts, filepath.Join(destBase, "commands")); err != nil {
		return nil, fmt.Errorf("cleanup .codex/commands: %w", err)
	}

	// Render canonical agents into TOML at <destBase>/agents/<name>.toml.
	agentsDest := filepath.Join(destBase, "agents")
	if err := renderToFile(opts, result, "agents", agentsDest, renderCodexAgentToml); err != nil {
		return nil, fmt.Errorf("render codex agents to toml: %w", err)
	}

	// Skills: render to .agents/skills/ (cross-tool standard; also read by
	// OpenCode as a fallback). Project mode writes under <projectRoot>/.agents/skills;
	// global mode under ~/.agents/skills.
	skillsDest := codexSkillsDest(opts)
	if err := installSkillsNested(opts, result, skillsDest); err != nil {
		return nil, fmt.Errorf("installing skills to %s: %w", skillsDest, err)
	}

	// Commands as Codex skills — Codex's SlashCommand is a built-in enum
	// and cannot load external command definitions. Emit each command as a
	// skill at command-<name>/SKILL.md so Codex agents can read and execute
	// Hero workflows step-by-step.
	if err := renderToFile(opts, result, "commands", skillsDest, renderCommandAsCodexSkill); err != nil {
		return nil, fmt.Errorf("render commands as codex skills: %w", err)
	}

	// Converge: both writers above render into skillsDest, so the prune runs
	// once, after both, over their combined output.
	written, err := codexSkillDirNames(opts)
	if err != nil {
		return nil, fmt.Errorf("enumerating codex skill dirs: %w", err)
	}
	if err := pruneStaleSkillDirs(opts, staleSkillPrune{
		dest:    skillsDest,
		written: written,
		// `source-command-` is a dead Hero namespace: a superseded layout
		// rendered commands under that prefix. Nothing writes it now, so
		// every such dir on disk is an orphan.
		ownedPrefixes: []string{codexCommandSkillPrefix, "source-command-"},
	}); err != nil {
		return nil, fmt.Errorf("prune stale codex skills: %w", err)
	}
	result.skillDirs = written

	// AGENTS.md (project root) or ~/.codex/AGENTS.md (global) via the
	// harness-native mapping — Codex's native root file is AGENTS.md.
	if err := installNativeInstructionFile(opts, result); err != nil {
		return nil, fmt.Errorf("installing AGENTS.md: %w", err)
	}

	// Hooks at .codex/hooks.json (project) or ~/.codex/hooks.json (global).
	if err := wireCodexHooks(opts, result); err != nil {
		fmt.Printf("  warning: could not wire codex hooks: %v\n", err)
	}

	return result, nil
}

// isLegacyCodexAgentMarkdown reports whether name is a pre-.toml Hero
// agent dead-byte in .codex/agents. Codex's loader ignores .md there, so
// any .md is non-functional legacy and Hero's to remove; .toml (current
// render + manifest-pruned) and every other user file are preserved. A
// user file that is itself a .md is inert in this dir and indistinguishable
// from the dead-bytes, so it is removed too — the load-bearing data-loss
// vector (a user's live .toml agent, or any subdir/other file) stays
// protected.
func isLegacyCodexAgentMarkdown(name string) bool {
	return strings.HasSuffix(name, ".md")
}

// codexSkillDirNames returns every skill dir name a Codex install
// materializes at the skills dest: the canonical skills, plus one
// command-<name> dir per command (renderCommandAsCodexSkill).
func codexSkillDirNames(opts Options) ([]string, error) {
	names, err := canonicalSkillDirNames(opts)
	if err != nil {
		return nil, err
	}
	srcFS := opts.sourceFS()
	domain := opts.Domain
	if domain == "" {
		domain = "engineering"
	}
	commands, err := selectFlatContent(srcFS, "commands", domain)
	if err != nil {
		return nil, err
	}
	for _, name := range trimMDNames(commands) {
		names = append(names, codexCommandSkillDir(name))
	}
	return names, nil
}

// codexSkillsDest returns the .agents/skills/ destination for the
// current install mode. Project mode writes under the project root;
// global mode writes under the user's home dir.
func codexSkillsDest(opts Options) string {
	if opts.Mode == ModeProject {
		return filepath.Join(opts.TargetDir, ".agents", "skills")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: still write under HOME-equivalent if Options carries one
		// in TargetDir during a global-mode test. resolveCodexPaths errors
		// well before this point if home is unavailable.
		return filepath.Join(opts.TargetDir, ".agents", "skills")
	}
	return filepath.Join(home, ".agents", "skills")
}

func resolveCodexPaths(opts Options) (destBase string, err error) {
	switch opts.Mode {
	case ModeProject:
		if opts.TargetDir == "" {
			return "", fmt.Errorf("project mode requires a target directory")
		}
		info, err := os.Stat(opts.TargetDir)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
		}
		return filepath.Join(opts.TargetDir, ".codex"), nil
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(home, ".codex"), nil
	default:
		return "", fmt.Errorf("unknown mode: %s", opts.Mode)
	}
}
