package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// GitHub Copilot uses .github/copilot-instructions.md for top-level
// instructions and .github/copilot/ for additional context files.
// Copilot has no skill primitive, so skills are stored as flat reference
// files (rendered fallback never produces nested SKILL.md here, but
// symlinks to .hero/skills/ work and Copilot just sees the linked tree).

func runCopilot(opts Options) (*Result, error) {
	if opts.Mode != ModeProject || opts.TargetDir == "" {
		return nil, fmt.Errorf("copilot target only supports project mode")
	}

	info, statErr := os.Stat(opts.TargetDir)
	if statErr != nil || !info.IsDir() {
		return nil, fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
	}

	destBase := filepath.Join(opts.TargetDir, ".github", "copilot")
	result := &Result{}

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

	instructionsPath := filepath.Join(opts.TargetDir, ".github", "copilot-instructions.md")
	if err := installInstructionsMd(opts, result, instructionsPath, "copilot"); err != nil {
		return nil, fmt.Errorf("installing copilot-instructions.md: %w", err)
	}

	return result, nil
}
