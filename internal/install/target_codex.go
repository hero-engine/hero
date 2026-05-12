package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// Codex CLI reads:
//
//   .codex/agents/   — agent definitions
//   .codex/commands/ — slash commands
//   .codex/skills/<name>/SKILL.md — Anthropic SKILL.md directory layout
//   AGENTS.md         — instructions at the project root (~/.codex/AGENTS.md in global mode)
//   .codex/config.toml — MCP server registration
//   .codex/hooks.json  — end-of-turn Stop hook
//
// Under P2, the content dirs become symlinks to the canonical .hero/
// tree. AGENTS.md continues to use the managed-block pattern from P1.

func runCodex(opts Options) (*Result, error) {
	destBase, err := resolveCodexPaths(opts)
	if err != nil {
		return nil, err
	}

	result := &Result{}

	if opts.Mode == ModeProject {
		agentsDir, commandsDir, skillsDir, err := ResolveCanonicalDirs(opts.TargetDir)
		if err != nil {
			return nil, err
		}
		if _, err := linkOrRenderDir(opts, result, "agents", agentsDir, filepath.Join(destBase, "agents"), false, false); err != nil {
			return nil, fmt.Errorf("linking agents: %w", err)
		}
		if _, err := linkOrRenderDir(opts, result, "commands", commandsDir, filepath.Join(destBase, "commands"), false, false); err != nil {
			return nil, fmt.Errorf("linking commands: %w", err)
		}
		if _, err := linkOrRenderDir(opts, result, "skills", skillsDir, filepath.Join(destBase, "skills"), true, false); err != nil {
			return nil, fmt.Errorf("linking skills: %w", err)
		}
	} else {
		if err := installFlat(opts, result, "agents", filepath.Join(destBase, "agents")); err != nil {
			return nil, fmt.Errorf("installing agents: %w", err)
		}
		if err := installFlat(opts, result, "commands", filepath.Join(destBase, "commands")); err != nil {
			return nil, fmt.Errorf("installing commands: %w", err)
		}
		if err := installSkillsNested(opts, result, filepath.Join(destBase, "skills")); err != nil {
			return nil, fmt.Errorf("installing skills: %w", err)
		}
	}

	if agentsMdPath := resolveAgentsMdPath(opts); agentsMdPath != "" {
		if err := installAgentsMd(opts, result, agentsMdPath); err != nil {
			return nil, fmt.Errorf("installing AGENTS.md: %w", err)
		}
	}

	if err := wireCodexHooks(opts, result); err != nil {
		fmt.Printf("  warning: could not wire codex hooks: %v\n", err)
	}

	return result, nil
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
