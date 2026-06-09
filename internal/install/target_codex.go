package install

import (
	"fmt"
	"os"
	"path/filepath"
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
//   - Symlinks (or renders fallback copy) canonical skills into BOTH
//     .agents/skills/ (preferred) AND .codex/skills/ (back-compat).
//   - Does NOT install commands at any scope (no loader exists).
//   - Cleans up dead bytes from prior installs at .codex/agents/*.md
//     and .codex/commands/*.

func runCodex(opts Options) (*Result, error) {
	destBase, err := resolveCodexPaths(opts)
	if err != nil {
		return nil, err
	}

	result := &Result{}

	// Cleanup of dead bytes from prior install layouts.
	// .codex/agents/*.md (Codex requires .toml — markdown is dead;
	// the dir is repopulated by renderToFile below with .toml).
	// .codex/commands/* (no loader at any scope; nothing repopulates).
	if err := removeLegacyDir(opts, filepath.Join(destBase, "agents")); err != nil {
		return nil, fmt.Errorf("cleanup .codex/agents: %w", err)
	}
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

	// AGENTS.md (project root) or ~/.codex/AGENTS.md (global) via the
	// shared managed-region writer. Already correct.
	if agentsMdPath := resolveAgentsMdPath(opts); agentsMdPath != "" {
		if err := installAgentsMd(opts, result, agentsMdPath); err != nil {
			return nil, fmt.Errorf("installing AGENTS.md: %w", err)
		}
	}

	// Hooks at .codex/hooks.json (project) or ~/.codex/hooks.json (global).
	if err := wireCodexHooks(opts, result); err != nil {
		fmt.Printf("  warning: could not wire codex hooks: %v\n", err)
	}

	return result, nil
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
