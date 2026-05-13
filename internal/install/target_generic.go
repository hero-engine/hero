package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// runGeneric installs to a tool-agnostic layout using .ai/ for agents,
// commands, skills and AGENTS.md at the project root. `.ai/` is a Hero
// convention with no consuming loader — it's the catch-all for tools
// without a dedicated installer. Any MCP-capable tool can pick up Hero
// via .mcp.json (registered in Run()).

func runGeneric(opts Options) (*Result, error) {
	if opts.Mode != ModeProject || opts.TargetDir == "" {
		return nil, fmt.Errorf("generic target only supports project mode")
	}

	info, statErr := os.Stat(opts.TargetDir)
	if statErr != nil || !info.IsDir() {
		return nil, fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
	}

	destBase := filepath.Join(opts.TargetDir, ".ai")
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

	agentsMdPath := filepath.Join(opts.TargetDir, "AGENTS.md")
	if err := installInstructionsMd(opts, result, agentsMdPath, "generic"); err != nil {
		return nil, fmt.Errorf("installing AGENTS.md: %w", err)
	}

	return result, nil
}

// installInstructionsMd generates a tool-agnostic instructions file
// (copilot-instructions.md or AGENTS.md) with Hero context. Used by
// Copilot and Generic targets; Claude and Codex use their target-specific
// generators.
func installInstructionsMd(opts Options, result *Result, destPath, targetName string) error {
	heroDir := filepath.Join(opts.TargetDir, ".hero")
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return nil // no hero workspace, skip
	}

	content := legacyMarker + "\n"
	content += fmt.Sprintf("# Hero Workspace (%s)\n\n", targetName)
	content += "This project uses [Hero](https://github.com/hero-engine/hero) for spec-driven development.\n\n"
	content += "## Available commands\n\n"
	content += "Agents, commands, and skills are installed in "
	if targetName == "copilot" {
		content += "`.github/copilot/`"
	} else {
		content += "`.ai/`"
	}
	content += ". Use the MCP server (registered in `.mcp.json`) for programmatic access.\n\n"
	content += "## Key workflow\n\n"
	content += "1. `/design <feature>` — create a spec\n"
	content += "2. `/deliver <spec>` — implement from spec\n"
	content += "3. `/diagnose <bug>` — investigate and produce fix spec\n"

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
		return err
	}

	result.Copied = append(result.Copied, CopyAction{Source: "hero-generated", Dest: destPath})
	return nil
}
