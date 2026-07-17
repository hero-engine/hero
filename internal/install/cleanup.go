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
// those dead bytes unconditionally — under render-direct, every
// known legacy location has no current consumer, so preserving
// "user edits" there only manifests as recurring warnings.

// removeLegacyDir walks legacyDir and removes everything inside,
// then removes the dir itself if it's empty.
//
//   - Symlinks are removed only if they point into a Hero-managed
//     tree (defensive: don't follow a symlink out of the project).
//   - Regular files and directories are removed unconditionally.
//
// Honors opts.DryRun.
//
// Kept as a thin wrapper (nil predicate = allow-all = wholesale) for
// dead legacy locations that have no current consumer — .codex/commands,
// .github/copilot/{agents,commands,skills}, and the P2-era
// .hero/{agents,commands,skills} mirror. Directories that are ALSO live
// loader dirs (they legitimately hold user content) must use
// removeLegacyDirMatching with a predicate instead.
func removeLegacyDir(opts Options, legacyDir string) error {
	return removeLegacyDirMatching(opts, legacyDir, nil)
}

// removeLegacyDirMatching removes entries in legacyDir for which
// shouldRemove returns true; a nil predicate matches every entry
// (wholesale — the removeLegacyDir behavior). Use a predicate when
// legacyDir is ALSO a live loader directory that legitimately holds user
// content — e.g. .codex/agents, where only pre-.toml *.md dead-bytes are
// Hero's to remove and any other file (a user .toml agent, notes, a
// subdir) must survive. The empty-dir cleanup only fires when nothing is
// left after the scoped removal, so a live dir with surviving user files
// or current .toml is preserved rather than rmdir'd.
func removeLegacyDirMatching(opts Options, legacyDir string, shouldRemove func(name string) bool) error {
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
		if shouldRemove != nil && !shouldRemove(e.Name()) {
			continue // not ours to remove — leave user files / current .toml in place
		}
		full := filepath.Join(legacyDir, e.Name())
		if err := removeLegacyEntry(opts, full, e); err != nil {
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
			fmt.Fprintf(os.Stderr, "  cleanup %s (empty dir would be removed)\n", legacyDir)
			return nil
		}
		if err := os.Remove(legacyDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Fprintf(os.Stderr, "  cleanup %s (empty after removing dead bytes)\n", legacyDir)
	}
	return nil
}

func removeLegacyEntry(opts Options, full string, e fs.DirEntry) error {
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
			fmt.Fprintf(os.Stderr, "  warning: %s symlinks to %q (not a Hero-managed target) — left in place\n", full, target)
			return nil
		}
		if opts.DryRun {
			fmt.Fprintf(os.Stderr, "  cleanup %s (would remove symlink → %s)\n", full, target)
			return nil
		}
		if err := os.Remove(full); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "  cleanup %s (removed Hero symlink → %s)\n", full, target)
		return nil
	}
	// Directory or regular file → remove unconditionally.
	if opts.DryRun {
		fmt.Fprintf(os.Stderr, "  cleanup %s (would remove)\n", full)
		return nil
	}
	if err := os.RemoveAll(full); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "  cleanup %s (removed dead bytes)\n", full)
	return nil
}

// cleanupLegacyCanonicalSymlinks removes the architecture-flip leftovers
// from the single-source-install P2 layout:
//
//  1. Harness-dir symlinks pointing into `.hero/`
//     (.claude/agents -> ../.hero/agents, etc.).
//  2. The `.hero/{agents,commands,skills}/` canonical mirror dirs.
//
// Idempotent: a project already migrated to the render-direct layout
// has none of these legacy artifacts, so the helper is a no-op.
// legacyHarnessKindPaths returns the absolute paths under projectDir that
// the P2→render-direct migration removes if present as symlinks pointing
// at .hero/. Single source of truth for "where the legacy layout left
// artifacts"; consumed by cleanupLegacyCanonicalSymlinks (mutating) and
// by the enumeration test that guards against new targets being added
// to install without being added here.
//
// Every harness Hero has ever materialized agents/commands/skills under
// must be listed exhaustively — a missing entry means dangling symlinks
// survive both `hero install` and `hero install --migrate`. See bug
// `upgrade-strands-install-layout`.
func legacyHarnessKindPaths(projectDir string) []string {
	return []string{
		filepath.Join(projectDir, ".claude", "agents"),
		filepath.Join(projectDir, ".claude", "commands"),
		filepath.Join(projectDir, ".claude", "skills"),
		filepath.Join(projectDir, ".opencode", "agents"),
		filepath.Join(projectDir, ".opencode", "commands"),
		filepath.Join(projectDir, ".opencode", "skills"),
		filepath.Join(projectDir, ".cursor", "rules", "agents"),
		filepath.Join(projectDir, ".cursor", "rules", "commands"),
		filepath.Join(projectDir, ".cursor", "rules", "skills"),
		filepath.Join(projectDir, ".codex", "agents"),
		filepath.Join(projectDir, ".codex", "commands"),
		filepath.Join(projectDir, ".codex", "skills"),
		filepath.Join(projectDir, ".ai", "agents"),
		filepath.Join(projectDir, ".ai", "commands"),
		filepath.Join(projectDir, ".ai", "skills"),
		filepath.Join(projectDir, ".github", "copilot", "skills"),
		filepath.Join(projectDir, ".github", "skills"),
		filepath.Join(projectDir, ".agents", "skills"),
	}
}

func cleanupLegacyCanonicalSymlinks(opts Options, projectDir string) error {
	// 1) Remove harness-dir symlinks that point into .hero/.
	for _, hp := range legacyHarnessKindPaths(projectDir) {
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

	// 2) Remove the canonical mirror dirs at .hero/{agents,commands,skills}/.
	for _, kind := range []string{"agents", "commands", "skills"} {
		dir := filepath.Join(projectDir, ".hero", kind)
		if err := removeLegacyDir(opts, dir); err != nil {
			fmt.Printf("  warning: cleanup %s: %v\n", dir, err)
		}
	}

	return nil
}

// LegacyDriftFinding describes one stale install artifact on disk
// that the next `hero install` (or `hero upgrade`) would clean up.
type LegacyDriftFinding struct {
	// Path is the absolute filesystem path of the artifact.
	Path string
	// Kind is "broken_symlink" (dangling harness-dir symlink pointing
	// at .hero/) or "legacy_canonical_dir" (leftover .hero/agents/,
	// .hero/commands/, or .hero/skills/ mirror directory).
	Kind string
	// Target is the symlink target (empty for legacy_canonical_dir).
	Target string
}

// DetectLegacyDrift returns the set of dangling harness-dir symlinks
// and stranded `.hero/{agents,commands,skills}/` mirror directories
// at projectDir. Read-only — does not mutate the filesystem.
//
// Shape parallels what cleanupLegacyCanonicalSymlinks would act on,
// but without performing the cleanup. Used by `hero check` and the
// root command preamble to surface drift loudly before the user
// hits the silent failure mode (Claude Code session with no Hero
// skills/agents loaded).
//
// Returns an empty slice on a clean install. Safe to call on any
// project, including ones that never had Hero installed.
func DetectLegacyDrift(projectDir string) []LegacyDriftFinding {
	var out []LegacyDriftFinding

	for _, hp := range legacyHarnessKindPaths(projectDir) {
		info, err := os.Lstat(hp)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, _ := os.Readlink(hp)
		// Only flag symlinks plausibly written by Hero (target points
		// into .hero/). User-authored symlinks at managed paths are
		// still removed by the mutating cleanup per the unconditional-
		// removal policy, but for *detection* we want to avoid false
		// positives that would scream "Hero is broken" at users who
		// hand-crafted their own layout.
		if !looksLikeHeroSymlinkTarget(target) {
			continue
		}
		// Resolve and check existence — symlinks that point at real
		// .hero/<kind>/ dirs are not drift, just legacy-but-functional.
		resolved := target
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(filepath.Dir(hp), target)
		}
		if _, err := os.Stat(resolved); err == nil {
			// Symlink resolves cleanly to an existing dir. Still legacy
			// — flag as legacy_canonical_dir below — but not a
			// broken_symlink that breaks harness loading right now.
			continue
		}
		out = append(out, LegacyDriftFinding{
			Path:   hp,
			Kind:   "broken_symlink",
			Target: target,
		})
	}

	for _, kind := range []string{"agents", "commands", "skills"} {
		dir := filepath.Join(projectDir, ".hero", kind)
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		out = append(out, LegacyDriftFinding{
			Path: dir,
			Kind: "legacy_canonical_dir",
		})
	}

	return out
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
