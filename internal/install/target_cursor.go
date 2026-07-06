package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// Cursor reads .cursor/rules/*.md (or *.mdc) (source:
// https://cursor.com/docs/context/rules). Hero installs agents/,
// commands/, and skills/ as subdirectories under .cursor/rules — Cursor
// has no agent/command/skill primitives, so the kind subdirs are
// human-readable organization. Skills go flat (.cursor/rules/skills/<n>.md)
// because Cursor doesn't follow Anthropic's nested SKILL.md format.
//
// Hero renders content directly from the embedded source to the harness
// path — no symlinks, no canonical mirror.

func runCursor(opts Options) (*Result, error) {
	destBase, err := resolveCursorPaths(opts)
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
	if err := installSkillsFlat(opts, result, filepath.Join(destBase, "skills")); err != nil {
		return nil, fmt.Errorf("installing skills: %w", err)
	}

	return result, nil
}

func resolveCursorPaths(opts Options) (destBase string, err error) {
	switch opts.Mode {
	case ModeProject:
		if opts.TargetDir == "" {
			return "", fmt.Errorf("project mode requires a target directory")
		}
		info, err := os.Stat(opts.TargetDir)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
		}
		destBase = filepath.Join(opts.TargetDir, ".cursor", "rules")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		destBase = filepath.Join(home, ".cursor", "rules")
	default:
		return "", fmt.Errorf("unknown mode: %s", opts.Mode)
	}
	return destBase, nil
}
