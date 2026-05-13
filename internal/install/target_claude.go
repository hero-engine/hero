package install

import (
	"fmt"
	"path/filepath"
)

// Claude Code reads (source: https://code.claude.com/docs/en/sub-agents,
// https://code.claude.com/docs/en/skills, https://code.claude.com/docs/en/settings):
//
//	.claude/agents/<name>.md          — agent definitions (require `name` + `description` frontmatter)
//	.claude/commands/<name>.md        — slash commands
//	.claude/skills/<name>/SKILL.md    — Anthropic SKILL.md directory layout
//	CLAUDE.md                          — project-level instructions (managed-block)
//	~/.claude/CLAUDE.md                — user-global instructions
//	.claude/settings.json              — hooks + permissions
//
// Hero renders directly from the embedded source into each harness path.
// Project mode writes under <project>/.claude/; global mode writes under
// ~/.claude/. No symlinks; no `.hero/{agents,commands,skills}/` canonical
// mirror.
func runClaude(opts Options) (*Result, error) {
	destBase, claudeMdPath, err := resolveClaudePaths(opts)
	if err != nil {
		return nil, err
	}

	result := &Result{}

	if err := installFlat(opts, result, "agents", filepath.Join(destBase, "agents")); err != nil {
		return nil, fmt.Errorf("installing agents: %w", err)
	}
	if err := installFlat(opts, result, "commands", filepath.Join(destBase, "commands")); err != nil {
		return nil, fmt.Errorf("installing commands: %w", err)
	}
	if err := installSkillsNested(opts, result, filepath.Join(destBase, "skills")); err != nil {
		return nil, fmt.Errorf("installing skills: %w", err)
	}

	if agentsMdPath := resolveAgentsMdPath(opts); agentsMdPath != "" {
		if err := installAgentsMd(opts, result, agentsMdPath); err != nil {
			return nil, fmt.Errorf("installing AGENTS.md: %w", err)
		}
	}

	if err := installClaudeMd(opts, result, claudeMdPath); err != nil {
		return nil, fmt.Errorf("installing CLAUDE.md: %w", err)
	}

	if err := wireClaudeHooks(opts, result); err != nil {
		fmt.Printf("  warning: could not wire claude hooks: %v\n", err)
	}

	if _, err := wireClaudePermissions(opts, result); err != nil {
		fmt.Printf("  warning: could not wire claude permissions: %v\n", err)
	}

	return result, nil
}
