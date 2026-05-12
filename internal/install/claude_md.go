package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// claude_md.go — CLAUDE.md handling under single-source-install P1.
//
// CLAUDE.md gets the same Hero-managed-block treatment as AGENTS.md:
// versioned markers, body content rendered inside, user content outside
// is preserved byte-for-byte. No symlinks. No @import shim. Same pattern
// everywhere.
//
// Why both AGENTS.md and CLAUDE.md? Because the harness ecosystem hasn't
// converged on one — Claude Code reads CLAUDE.md, most other harnesses
// read AGENTS.md. Hero writes both with the same managed body so every
// harness sees the same content. Both files are independently editable:
// the user can author Claude-specific notes in CLAUDE.md and cross-harness
// notes in AGENTS.md, and Hero leaves both untouched outside the markers.
//
// Three behaviors (same as AGENTS.md):
//
//  1. CLAUDE.md doesn't exist → create with the default H1 + managed
//     region.
//  2. CLAUDE.md exists, no managed region → insert at the top (after the
//     H1 if any), preserve user content below.
//  3. CLAUDE.md exists with a managed region (versioned or legacy) →
//     replace the region in place, preserve content outside.
//
// Escape hatch:
//   - Options.NoTouchClaudeMd (--no-touch-claude-md): skip CLAUDE.md
//     entirely. Niche; user accepts Claude Code won't see Hero content
//     via CLAUDE.md (other harnesses still get it via AGENTS.md).

// resolveClaudePaths returns the .claude/ base directory and the
// CLAUDE.md path for the install mode.
func resolveClaudePaths(opts Options) (destBase, claudeMdPath string, err error) {
	switch opts.Mode {
	case ModeProject:
		if opts.TargetDir == "" {
			return "", "", fmt.Errorf("project mode requires a target directory")
		}
		info, err := os.Stat(opts.TargetDir)
		if err != nil || !info.IsDir() {
			return "", "", fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
		}
		destBase = filepath.Join(opts.TargetDir, ".claude")
		claudeMdPath = filepath.Join(opts.TargetDir, "CLAUDE.md")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		destBase = filepath.Join(home, ".claude")
		claudeMdPath = filepath.Join(home, ".claude", "CLAUDE.md")
	default:
		return "", "", fmt.Errorf("unknown mode: %s", opts.Mode)
	}
	return destBase, claudeMdPath, nil
}

// installClaudeMd writes Hero's managed block into CLAUDE.md, following
// the same managed-region pattern as AGENTS.md. If a CLAUDE.md is found
// that's a symlink (e.g. a leftover from the previous symlink-shim
// approach), it's removed first so the new managed-block file can be
// written in its place.
func installClaudeMd(opts Options, result *Result, claudeMdPath string) error {
	if opts.NoTouchClaudeMd {
		return nil
	}

	// If a leftover symlink is in the way (from prior shim-based installs),
	// remove it so writeFile can land cleanly. The target of the symlink
	// (AGENTS.md) is the canonical source and remains.
	if info, err := os.Lstat(claudeMdPath); err == nil && info.Mode()&os.ModeSymlink != 0 && !opts.DryRun {
		if err := os.Remove(claudeMdPath); err != nil {
			return fmt.Errorf("removing legacy CLAUDE.md symlink: %w", err)
		}
	}

	return installManagedMarkdown(opts, result, installManagedSpec{
		Path:        claudeMdPath,
		Label:       "CLAUDE.md",
		DefaultH1:   "# CLAUDE.md",
		Body:        generateAgentsMdBody(),
		AllowSkip:   true,
		SkipEnabled: opts.NoTouchClaudeMd,
	})
}
