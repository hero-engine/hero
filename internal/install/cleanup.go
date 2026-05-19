package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// cleanup.go — removal of dead bytes from prior install layouts.
//
// As Hero corrects per-harness install paths over time (per
// harness-install-paths-match-loaders), users on prior releases end
// up with content at destinations the consuming harness no longer
// reads (or never did). On the next install/upgrade, Hero removes
// those dead bytes — but ONLY when the destination is detectably
// Hero-authored. User-edited files are left in place with a warning
// so the user can review before deleting.

// removeIfHeroAuthored walks legacyDir and removes entries that are
// detectably Hero-installed:
//
//   - Symlinks pointing inside the project's `.hero/` canonical tree
//     (or its global counterpart) are removed.
//   - Regular files whose bytes match the canonical embedded source
//     for the same kind+name are removed.
//   - Anything else is left in place; a warning is appended to
//     result.Skipped describing the surviving file.
//
// After processing, removes legacyDir itself if it's empty.
//
// canonicalKind names the embedded canonical kind ("agents",
// "commands", "skills") used to verify file content equality.
// Pass empty string to skip content-equality removal (only
// symlinks-to-.hero/ get cleaned up).
//
// When force is true, the byte-equality gate on regular files and
// SKILL.md dirs is skipped — everything in legacyDir is removed
// unconditionally (symlinks still require a Hero-managed target).
// Use for paths Hero no longer writes or reads, where preserving
// "user edits" has no current consumer to protect.
//
// Honors opts.DryRun.
func removeIfHeroAuthored(opts Options, result *Result, legacyDir, canonicalKind string, force bool) error {
	info, err := os.Lstat(legacyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		full := filepath.Join(legacyDir, e.Name())
		if err := removeOneIfHeroAuthored(opts, result, full, canonicalKind, e, force); err != nil {
			return err
		}
	}
	// If the dir is now empty, remove it.
	leftover, err := os.ReadDir(legacyDir)
	if err != nil {
		return err
	}
	if len(leftover) == 0 {
		if opts.DryRun {
			fmt.Fprintf(os.Stderr,"  cleanup %s (empty dir would be removed)\n", legacyDir)
			return nil
		}
		if err := os.Remove(legacyDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Fprintf(os.Stderr,"  cleanup %s (empty after removing dead bytes)\n", legacyDir)
	}
	return nil
}

func removeOneIfHeroAuthored(opts Options, result *Result, full, canonicalKind string, e fs.DirEntry, force bool) error {
	info, err := e.Info()
	if err != nil {
		return err
	}
	// Symlink → only remove if it points into a Hero-managed tree.
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(full)
		if err != nil {
			return nil
		}
		if !looksLikeHeroSymlinkTarget(target) {
			result.Skipped = append(result.Skipped, full+" (symlink target not Hero-managed; left in place)")
			fmt.Fprintf(os.Stderr,"  warning: %s symlinks to %q (not a Hero-managed target) — left in place\n", full, target)
			return nil
		}
		if opts.DryRun {
			fmt.Fprintf(os.Stderr,"  cleanup %s (would remove symlink → %s)\n", full, target)
			return nil
		}
		if err := os.Remove(full); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr,"  cleanup %s (removed Hero symlink → %s)\n", full, target)
		return nil
	}
	// Directory → recurse for the nested skills layout.
	if info.IsDir() {
		// Try to clean up the nested SKILL.md if this looks like a skill dir.
		nested := filepath.Join(full, "SKILL.md")
		if _, err := os.Stat(nested); err == nil {
			if force || matchesNestedSkillCanonical(opts, filepath.Base(full), nested) {
				if opts.DryRun {
					fmt.Fprintf(os.Stderr,"  cleanup %s (would remove Hero-authored skill dir)\n", full)
					return nil
				}
				if err := os.RemoveAll(full); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr,"  cleanup %s (removed Hero-authored skill dir)\n", full)
				return nil
			}
			result.Skipped = append(result.Skipped, full+" (skill dir not Hero-authored; left in place)")
			fmt.Fprintf(os.Stderr,"  warning: %s does not match canonical bytes — left in place\n", full)
			return nil
		}
		// Plain dir without SKILL.md — recurse and clean up Hero-authored files inside.
		return removeIfHeroAuthored(opts, result, full, canonicalKind, force)
	}
	// Regular file → remove unconditionally under force, otherwise
	// require byte equality with canonical embedded source.
	if !force && canonicalKind == "" {
		// No canonical kind to verify against → leave in place.
		return nil
	}
	if !strings.HasSuffix(e.Name(), ".md") && !strings.HasSuffix(e.Name(), ".toml") {
		return nil
	}
	if force || canonicalBytesEqual(opts, canonicalKind, e.Name(), full) {
		if opts.DryRun {
			fmt.Fprintf(os.Stderr,"  cleanup %s (would remove Hero-authored file)\n", full)
			return nil
		}
		if err := os.Remove(full); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr,"  cleanup %s (removed Hero-authored file)\n", full)
		return nil
	}
	result.Skipped = append(result.Skipped, full+" (not Hero-authored bytes; left in place)")
	fmt.Fprintf(os.Stderr,"  warning: %s differs from canonical content — left in place\n", full)
	return nil
}

// matchesNestedSkillCanonical checks whether a nested skill
// destination (<dir>/SKILL.md) matches the canonical skill source
// for skillName. Tries the flat layout (skills/<n>.md) first since
// that's what the embedded source uses; falls back to the nested
// layout (skills/<n>/SKILL.md) for forward compatibility.
func matchesNestedSkillCanonical(opts Options, skillName, destPath string) bool {
	if canonicalBytesEqual(opts, "skills", skillName+".md", destPath) {
		return true
	}
	if canonicalBytesEqual(opts, "skills/"+skillName, "SKILL.md", destPath) {
		return true
	}
	return false
}

// cleanupLegacyCanonicalSymlinks removes the architecture-flip leftovers
// from the single-source-install P2 layout:
//
//  1. Harness-dir symlinks pointing into `.hero/`
//     (.claude/agents -> ../.hero/agents, etc.).
//  2. The `.hero/{agents,commands,skills}/` canonical mirror dirs (only
//     when their contents are detectably Hero-authored — user-edited
//     files are left in place with a warning).
//
// Idempotent: a project already migrated to the render-direct layout
// has none of these legacy artifacts, so the helper is a no-op.
func cleanupLegacyCanonicalSymlinks(opts Options, projectDir string) error {
	// 1) Remove harness-dir symlinks that point into .hero/.
	harnessKindPaths := []string{
		filepath.Join(projectDir, ".claude", "agents"),
		filepath.Join(projectDir, ".claude", "commands"),
		filepath.Join(projectDir, ".claude", "skills"),
		filepath.Join(projectDir, ".opencode", "agents"),
		filepath.Join(projectDir, ".opencode", "commands"),
		filepath.Join(projectDir, ".opencode", "skills"),
		filepath.Join(projectDir, ".cursor", "rules", "agents"),
		filepath.Join(projectDir, ".cursor", "rules", "commands"),
		filepath.Join(projectDir, ".cursor", "rules", "skills"),
		filepath.Join(projectDir, ".codex", "skills"),
		filepath.Join(projectDir, ".ai", "agents"),
		filepath.Join(projectDir, ".ai", "commands"),
		filepath.Join(projectDir, ".ai", "skills"),
		filepath.Join(projectDir, ".github", "copilot", "skills"),
		filepath.Join(projectDir, ".github", "skills"),
		filepath.Join(projectDir, ".agents", "skills"),
	}
	for _, hp := range harnessKindPaths {
		info, err := os.Lstat(hp)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		// Under render-direct-install, Hero never creates symlinks at
		// harness destination paths. Any symlink here is a legacy
		// artifact (either pointing at .hero/<kind>/ from the P2 era,
		// or pointing at top-level agents/commands/skills/ from
		// hero-on-hero dev convenience). Remove unconditionally —
		// content is re-rendered fresh from embedded source.
		target, _ := os.Readlink(hp)
		if opts.DryRun {
			fmt.Printf("  cleanup %s (would remove legacy symlink → %s)\n", hp, target)
			continue
		}
		if err := os.Remove(hp); err != nil {
			fmt.Printf("  warning: could not remove legacy symlink %s: %v\n", hp, err)
			continue
		}
		fmt.Printf("  cleanup %s (removed legacy symlink → %s)\n", hp, target)
	}

	// 2) Remove the canonical mirror dirs at .hero/{agents,commands,skills}/
	//    unconditionally. Under render-direct, Hero no longer writes or
	//    reads these paths — they exist purely as orphans from the P2
	//    single-source-install layout. There is no current consumer to
	//    protect, so byte-equality protection only manifests as recurring
	//    "differs from canonical" noise on upgrade.
	result := &Result{}
	for _, kind := range []string{"agents", "commands", "skills"} {
		dir := filepath.Join(projectDir, ".hero", kind)
		if err := removeIfHeroAuthored(opts, result, dir, kind, true); err != nil {
			fmt.Printf("  warning: cleanup %s: %v\n", dir, err)
		}
	}

	return nil
}

// looksLikeHeroSymlinkTarget reports whether a symlink target string
// is one Hero would have created — i.e. points into a `.hero/` tree
// or into the canonical `.hero/{agents,commands,skills}/` layout.
// Conservative: ignores absolute paths outside `.hero/` and anything
// not matching the expected canonical shape.
func looksLikeHeroSymlinkTarget(target string) bool {
	t := filepath.ToSlash(target)
	return strings.Contains(t, "/.hero/") ||
		strings.HasPrefix(t, ".hero/") ||
		strings.Contains(t, "/.hero")
}

