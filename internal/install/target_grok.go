package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// runGrok installs Hero into Grok Build's native project or global layout.
// Grok loads Markdown agents and nested skills directly. User-invocable Hero
// workflows are rendered as command-* skills; there is no Hero-owned
// .grok/commands tree.
func runGrok(opts Options) (*Result, error) {
	destBase, err := resolveGrokPaths(opts)
	if err != nil {
		return nil, err
	}

	result := &Result{}
	if err := installFlat(opts, result, "agents", filepath.Join(destBase, "agents")); err != nil {
		return nil, fmt.Errorf("installing grok agents: %w", err)
	}

	skillsDest := filepath.Join(destBase, "skills")
	if err := installSkillsNested(opts, result, skillsDest); err != nil {
		return nil, fmt.Errorf("installing grok skills: %w", err)
	}
	if err := renderToFile(opts, result, "commands", skillsDest, commandAsSkillRenderer("Grok Build")); err != nil {
		return nil, fmt.Errorf("render commands as grok skills: %w", err)
	}

	written, err := commandSkillDirNames(opts)
	if err != nil {
		return nil, fmt.Errorf("enumerating grok skill dirs: %w", err)
	}
	if err := pruneStaleSkillDirs(opts, staleSkillPrune{
		dest:          skillsDest,
		written:       written,
		ownedPrefixes: []string{commandSkillPrefix},
	}); err != nil {
		return nil, fmt.Errorf("prune stale grok skills: %w", err)
	}
	result.skillDirs = written

	if err := installNativeInstructionFile(opts, result); err != nil {
		return nil, fmt.Errorf("installing grok AGENTS.md: %w", err)
	}
	return result, nil
}

func resolveGrokPaths(opts Options) (string, error) {
	switch opts.Mode {
	case ModeProject:
		if opts.TargetDir == "" {
			return "", fmt.Errorf("project mode requires a target directory")
		}
		info, err := os.Stat(opts.TargetDir)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
		}
		return filepath.Join(opts.TargetDir, ".grok"), nil
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(home, ".grok"), nil
	default:
		return "", fmt.Errorf("unknown mode: %s", opts.Mode)
	}
}
