package install

import (
	"fmt"
	"path/filepath"
)

// Claude Code reads:
//
//	.claude/agents/   — agent definitions
//	.claude/commands/ — native slash commands
//	.claude/skills/<name>/SKILL.md — Anthropic SKILL.md directory layout
//	CLAUDE.md          — project-level instructions
//	~/.claude/CLAUDE.md — user-global instructions
//
// Under single-source-install P2, the agents/commands/skills directories
// under .claude/ become directory symlinks pointing at the canonical
// .hero/{agents,commands,skills}/ tree. Claude Code follows the symlinks,
// so editing canonical content takes effect immediately. When symlinks
// aren't available, Hero falls back to rendered copies (see
// linkOrRenderDir for the fallback semantics).
//
// AGENTS.md and CLAUDE.md continue to use the managed-block pattern from
// P1 — both are regular files with versioned managed regions.
func runClaude(opts Options) (*Result, error) {
	destBase, claudeMdPath, err := resolveClaudePaths(opts)
	if err != nil {
		return nil, err
	}

	result := &Result{}

	// Project mode: link harness dirs at canonical .hero/{agents,commands,skills}/.
	// Global mode: fall back to direct rendering (no .hero/ workspace).
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

	// Write AGENTS.md (cross-harness) — managed-block pattern.
	if agentsMdPath := resolveAgentsMdPath(opts); agentsMdPath != "" {
		if err := installAgentsMd(opts, result, agentsMdPath); err != nil {
			return nil, fmt.Errorf("installing AGENTS.md: %w", err)
		}
	}

	// Write CLAUDE.md — same managed-block treatment.
	if err := installClaudeMd(opts, result, claudeMdPath); err != nil {
		return nil, fmt.Errorf("installing CLAUDE.md: %w", err)
	}

	// Wire native Stop and PreCompact hooks into .claude/settings.json so
	// NEXT.md auto-refreshes every turn — the cross-session handoff fix.
	if err := wireClaudeHooks(opts, result); err != nil {
		fmt.Printf("  warning: could not wire claude hooks: %v\n", err)
	}

	// Add Bash(hero:*) to permissions.allow so Claude Code stops
	// prompting on every `hero` invocation.
	if _, err := wireClaudePermissions(opts, result); err != nil {
		fmt.Printf("  warning: could not wire claude permissions: %v\n", err)
	}

	return result, nil
}
