package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// content.go — shared primitives for installing agent/command/skill content
// from the source filesystem into a target directory.
//
// installFlat   — writes each .md file directly into destDir (used by harnesses
//                 that read flat instruction files: Cursor, Copilot, Generic).
// installSkillsNested — writes each skill as destDir/<name>/SKILL.md per the
//                 Anthropic Agent Skills format. Used by harnesses whose Skill
//                 loader requires this directory layout (Claude Code, opencode,
//                 Codex). Includes legacy-flat-file cleanup so re-running
//                 install against a buggy prior install self-migrates.

func installFlat(opts Options, result *Result, kind, destDir string) error {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		return fmt.Errorf("no content source available")
	}

	entries, err := fs.ReadDir(srcFS, kind)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		srcPath := kind + "/" + entry.Name()
		dst := filepath.Join(destDir, entry.Name())

		if err := copyFileFromFS(opts, result, srcFS, srcPath, dst); err != nil {
			return err
		}
	}

	return nil
}

func installSkillsNested(opts Options, result *Result, destDir string) error {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		return fmt.Errorf("no content source available")
	}

	entries, err := fs.ReadDir(srcFS, "skills")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Clean up any legacy flat-file skills at destDir/<name>.md from prior
	// installs that wrote skills as flat files. Anthropic's SKILL.md format
	// requires <name>/SKILL.md directory layout — flat files are silently
	// invisible to the Skill loader even when content is otherwise correct.
	if err := cleanupFlatSkills(opts, destDir); err != nil {
		return fmt.Errorf("cleaning legacy flat skills: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()

		// Canonical source layout: `skills/<name>/SKILL.md`. Read directly
		// from there, write to `<destDir>/<name>/SKILL.md`.
		if entry.IsDir() {
			srcSkill := "skills/" + name + "/SKILL.md"
			if _, err := fs.Stat(srcFS, srcSkill); err != nil {
				continue // directory without SKILL.md — not a skill
			}
			dst := filepath.Join(destDir, name, "SKILL.md")
			if err := copyFileFromFS(opts, result, srcFS, srcSkill, dst); err != nil {
				return err
			}
			continue
		}

		// Legacy flat layout: `skills/<name>.md`. Still supported for
		// backward compat; rendered into the nested layout at dest.
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		base := strings.TrimSuffix(name, ".md")
		srcPath := "skills/" + name
		dst := filepath.Join(destDir, base, "SKILL.md")
		if err := copyFileFromFS(opts, result, srcFS, srcPath, dst); err != nil {
			return err
		}
	}

	return nil
}

// cleanupFlatSkills removes flat *.md files at destDir written by prior
// (buggy) installs that used installFlat for skills. The Anthropic SKILL.md
// directory layout supersedes them; leaving the flat copies behind clutters
// the harness directory and risks confusion. Subdirectories (the correct
// layout) are left untouched.
func cleanupFlatSkills(opts Options, destDir string) error {
	if opts.DryRun {
		return nil
	}
	entries, err := os.ReadDir(destDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		full := filepath.Join(destDir, name)
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing legacy flat skill %s: %w", full, err)
		}
	}
	return nil
}
