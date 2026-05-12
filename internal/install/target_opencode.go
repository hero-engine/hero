package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// opencode reads from:
//
//   .opencode/agents/   — agent definitions
//   .opencode/commands/ — slash commands
//   .opencode/skills/   — Anthropic SKILL.md directory layout
//   AGENTS.md (root)    — primary instructions
//   opencode.json       — config
//
// Under P2, the content dirs become symlinks to the canonical .hero/
// tree. AGENTS.md continues to use the managed-block pattern from P1.
// `opencode.json` continues to merge with Hero's MCP and instruction
// settings.

func runOpenCode(opts Options) (*Result, error) {
	destBase, configDest, err := resolveOpenCodePaths(opts)
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

	if err := installConfig(opts, result, configDest); err != nil {
		return nil, fmt.Errorf("installing config: %w", err)
	}

	if agentsMdPath := resolveAgentsMdPath(opts); agentsMdPath != "" {
		if err := installAgentsMd(opts, result, agentsMdPath); err != nil {
			return nil, fmt.Errorf("installing AGENTS.md: %w", err)
		}
	}

	return result, nil
}

func resolveOpenCodePaths(opts Options) (destBase, configDest string, err error) {
	switch opts.Mode {
	case ModeProject:
		if opts.TargetDir == "" {
			return "", "", fmt.Errorf("project mode requires a target directory")
		}
		info, err := os.Stat(opts.TargetDir)
		if err != nil || !info.IsDir() {
			return "", "", fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
		}
		destBase = filepath.Join(opts.TargetDir, ".opencode")
		configDest = filepath.Join(opts.TargetDir, "opencode.json")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		destBase = filepath.Join(home, ".config", "opencode")
		configDest = filepath.Join(home, ".config", "opencode", "config.json")
	default:
		return "", "", fmt.Errorf("unknown mode: %s", opts.Mode)
	}
	return destBase, configDest, nil
}

// installConfig copies (or merges) opencode.json from the source FS into
// the project's opencode.json.
func installConfig(opts Options, result *Result, configDest string) error {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		return nil
	}

	if _, err := fs.Stat(srcFS, "opencode.json"); err != nil {
		return nil
	}

	if opts.DryRun {
		if _, err := os.Stat(configDest); err == nil {
			progressf(opts, "  opencode.json -> %s (merge)\n", configDest)
			result.Merged = append(result.Merged, configDest)
		} else {
			progressf(opts, "  opencode.json -> %s\n", configDest)
			result.Copied = append(result.Copied, CopyAction{Source: "opencode.json", Dest: configDest})
		}
		return nil
	}

	if _, err := os.Stat(configDest); err == nil {
		srcData, err := fs.ReadFile(srcFS, "opencode.json")
		if err != nil {
			return err
		}
		if err := mergeJSONFromData(srcData, configDest, opts.Force); err != nil {
			return err
		}
		result.Merged = append(result.Merged, configDest)
		return nil
	}

	return copyFileFromFS(opts, result, srcFS, "opencode.json", configDest)
}
