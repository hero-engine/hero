package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// Cursor reads .cursor/rules/*.md (or *.mdc). Hero installs agents/,
// commands/, and skills/ as subdirectories under .cursor/rules. Cursor
// accepts both .md and .mdc, and skills are kept flat (Cursor doesn't
// follow Anthropic's SKILL.md directory format).
//
// Under P2, the subdirectories become symlinks to the canonical .hero/
// tree. The flat-vs-nested distinction is handled via the canonical
// content layout — .hero/skills/ is nested-SKILL.md by default, but
// Cursor's symlinked view still works because Cursor only cares about
// what's in the linked directory.

func runCursor(opts Options) (*Result, error) {
	destBase, err := resolveCursorPaths(opts)
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
		// Cursor reads .md/.mdc recursively, so the symlinked nested view
		// of .hero/skills/ works. The rendered fallback uses flat layout
		// to preserve legacy behavior for projects without a .hero/
		// workspace yet.
		if _, err := linkOrRenderDir(opts, result, "skills", skillsDir, filepath.Join(destBase, "skills"), false, false); err != nil {
			return nil, fmt.Errorf("linking skills: %w", err)
		}
	} else {
		if err := installFlat(opts, result, "agents", filepath.Join(destBase, "agents")); err != nil {
			return nil, fmt.Errorf("installing agents: %w", err)
		}
		if err := installFlat(opts, result, "commands", filepath.Join(destBase, "commands")); err != nil {
			return nil, fmt.Errorf("installing commands: %w", err)
		}
		if err := installFlat(opts, result, "skills", filepath.Join(destBase, "skills")); err != nil {
			return nil, fmt.Errorf("installing skills: %w", err)
		}
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
