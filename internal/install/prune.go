package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// prune.go — convergence for nested-skill destinations.
//
// Install renders the current canonical set into <dest>/<name>/SKILL.md,
// but rendering alone never removes a directory whose canonical source was
// renamed or deleted. Codex's skill loader walks `.agents/skills/*/SKILL.md`
// (see target_codex.go), so an orphaned dir keeps loading as a live skill
// forever — a stale workflow the user never asked for and `--force` never
// clears.
//
// The dest dirs are shared ground: `.agents/skills` is a cross-tool
// standard location, and `.claude/skills` / `.opencode/skills` hold
// user-authored skills in real projects. So "remove what I did not just
// write" is not a safe rule. A directory is removed only when Hero can
// prove it wrote it, via one of two proofs:
//
//  1. install-state.json records the skill-dir set from this target's last
//     install (TargetState.SkillDirs). Anything in that set but absent from
//     the current set was a Hero skill and is now gone from canonical.
//     Covers renamed or deleted skills from here on.
//
//  2. The dir name falls under a namespace Hero owns at that dest. Codex
//     renders every command as `command-<name>/`, and an older layout
//     rendered them as `source-command-<name>/`, so both prefixes are
//     Hero's by construction under the Codex skills dest — no prior install
//     state needed. This is what clears orphans written before (1) existed.
//     Targets that do not render commands as skills claim no prefixes: a
//     `command-foo/` under `.claude/skills` is the user's, not Hero's.
//
// Everything else is left alone.

// The file prune (pruneStaleFiles, below) extends this same doctrine from
// nested skill DIRS to the flat rendered FILES — agent, command, and
// Cursor flat-skill files — that the directory mechanism above does not
// reach. The two-proof model narrows to ONE proof for files: manifest
// membership only. A flat agent/command file has no Hero-owned namespace
// at a shared dest (there is no `command-` analogue — `.claude/agents/foo.md`
// looks identical whether Hero or the user wrote it), so the recorded
// TargetState.Files manifest is the sole provenance. A file absent from the
// prior manifest was never recorded as Hero-written and is invisible to the
// prune. Nil/empty prior manifest, or an empty current render, → strict
// no-op. The two prunes are disjoint by construction (see renderToFile's
// path-separator rule): nested skills stay with SkillDirs, flat files stay
// with Files.

// staleSkillPrune describes one dest's convergence pass.
type staleSkillPrune struct {
	// dest is the skills destination directory to converge.
	dest string
	// written is the set of skill dir names this install run materializes
	// at dest. Derived from the canonical selectors rather than
	// result.Copied — copyFileFromFS skips the Copied record when the
	// destination bytes already match, so on a no-op re-install Copied is
	// nearly empty and would condemn the whole tree.
	written []string
	// ownedPrefixes are dir-name prefixes Hero owns at this dest. Empty
	// for targets that only write canonical skills.
	ownedPrefixes []string
}

// pruneStaleSkillDirs removes provably-Hero skill directories at p.dest
// that this install run did not write. Honors opts.DryRun. Returns the
// written set so the caller can record it as the next run's manifest.
func pruneStaleSkillDirs(opts Options, p staleSkillPrune) error {
	writtenSet := map[string]bool{}
	for _, name := range p.written {
		writtenSet[name] = true
	}

	entries, err := os.ReadDir(p.dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	prior := priorSkillDirs(opts)

	var stale []string
	for _, e := range entries {
		if !e.IsDir() || writtenSet[e.Name()] {
			continue
		}
		if !heroOwnsSkillDir(e.Name(), prior, p.ownedPrefixes) {
			continue
		}
		stale = append(stale, e.Name())
	}
	sort.Strings(stale)

	for _, name := range stale {
		full := filepath.Join(p.dest, name)
		if opts.DryRun {
			fmt.Fprintf(os.Stderr, "  cleanup %s (would remove stale skill)\n", full)
			continue
		}
		if err := os.RemoveAll(full); err != nil {
			return fmt.Errorf("removing stale skill %s: %w", full, err)
		}
		fmt.Fprintf(os.Stderr, "  cleanup %s (removed stale skill — no canonical source)\n", full)
	}
	return nil
}

// heroOwnsSkillDir reports whether Hero can prove it authored a skill dir
// of this name at the dest — either a prior install recorded it, or its
// name falls under a Hero-owned namespace at that dest.
func heroOwnsSkillDir(name string, prior map[string]bool, ownedPrefixes []string) bool {
	if prior[name] {
		return true
	}
	for _, prefix := range ownedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// priorSkillDirs returns the skill-dir set recorded by this target's last
// install. Empty when there is no install-state to read — global mode has
// no .hero/ workspace, and a first install has no prior run — which leaves
// the owned-prefix proof as the only one in play.
func priorSkillDirs(opts Options) map[string]bool {
	if opts.Mode != ModeProject || opts.TargetDir == "" {
		return nil
	}
	st, err := ReadInstallState(opts.TargetDir)
	if err != nil || st == nil {
		return nil
	}
	prior, ok := st.Targets[string(opts.Target)]
	if !ok {
		return nil
	}
	out := make(map[string]bool, len(prior.SkillDirs))
	for _, name := range prior.SkillDirs {
		out[name] = true
	}
	return out
}

// pruneNestedSkills converges a dest that holds canonical skills and
// nothing else — the Claude and OpenCode shape. Codex writes commands into
// its skills dest too and calls pruneStaleSkillDirs directly with the
// combined set.
func pruneNestedSkills(opts Options, result *Result, dest string) error {
	written, err := canonicalSkillDirNames(opts)
	if err != nil {
		return err
	}
	if err := pruneStaleSkillDirs(opts, staleSkillPrune{dest: dest, written: written}); err != nil {
		return err
	}
	result.skillDirs = written
	return nil
}

// canonicalSkillDirNames returns the skill dir names installSkillsNested
// materializes for the current source — the canonical skills, before any
// target-specific additions.
func canonicalSkillDirNames(opts Options) ([]string, error) {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		return nil, fmt.Errorf("no content source available")
	}
	skills, err := selectSkillContent(srcFS)
	if err != nil {
		return nil, err
	}
	return skillNames(skills), nil
}

// renderedFileManifest returns this run's flat rendered dest paths as
// TargetDir-relative, forward-slash strings, sorted and deduped. It is the
// canonical "what Hero renders now" record RecordTargetInstall persists as
// TargetState.Files and pruneStaleFiles diffs against. Paths outside
// TargetDir (never expected in project mode) are dropped.
func renderedFileManifest(opts Options, result *Result) []string {
	if result == nil || len(result.rendered) == 0 || opts.TargetDir == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, dst := range result.rendered {
		rel, err := filepath.Rel(opts.TargetDir, dst)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == "" || strings.HasPrefix(rel, "../") {
			continue
		}
		if seen[rel] {
			continue
		}
		seen[rel] = true
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

// pruneStaleFiles removes flat agent/command/flat-skill files this target
// recorded writing on its last install but did not write this run — the
// product dropped them. It is the flat-file sibling of pruneStaleSkillDirs
// and shares its provenance doctrine (see the header above): a file is
// removed only when the recorded prior manifest proves Hero wrote it. Honors
// opts.DryRun. Must run BEFORE RecordTargetInstall (which overwrites the
// prior manifest with this run's set).
func pruneStaleFiles(opts Options, result *Result) error {
	if opts.Mode != ModeProject || opts.TargetDir == "" {
		return nil
	}

	// Prior manifest is the sole provenance proof. Nil/empty → no proof of
	// anything Hero wrote → prune nothing. Covers fresh clone with no
	// install-state.json (AC-13), a pre-Files TargetState (AC-6), and a
	// genuinely empty prior manifest (AC-3).
	st, err := ReadInstallState(opts.TargetDir)
	if err != nil || st == nil {
		return nil
	}
	prior, ok := st.Targets[string(opts.Target)]
	if !ok || len(prior.Files) == 0 {
		return nil
	}

	// This run's rendered set. Empty → a broken/empty content source; skip
	// rather than delete everything the prior manifest listed (AC-12).
	current := renderedFileManifest(opts, result)
	if len(current) == 0 {
		return nil
	}
	currentSet := make(map[string]bool, len(current))
	for _, rel := range current {
		currentSet[rel] = true
	}

	var stale []string
	for _, rel := range prior.Files {
		if !currentSet[rel] {
			stale = append(stale, rel)
		}
	}
	sort.Strings(stale)

	for _, rel := range stale {
		full := filepath.Join(opts.TargetDir, filepath.FromSlash(rel))
		if _, err := os.Stat(full); err != nil {
			// Already gone — tolerate the skill-dir prune having removed it,
			// or a user deletion. Nothing to do (AC-10).
			continue
		}
		if opts.DryRun {
			fmt.Fprintf(os.Stderr, "  cleanup %s (would remove — dropped from product)\n", full)
			continue
		}
		if err := os.Remove(full); err != nil {
			return fmt.Errorf("removing stale file %s: %w", full, err)
		}
		fmt.Fprintf(os.Stderr, "  cleanup %s (removed — dropped from product)\n", full)
		// Best-effort tidy of a now-empty parent dir, matching the Copilot
		// cleanup idiom. Ignore errors: a non-empty dir simply stays.
		_ = os.Remove(filepath.Dir(full))
	}
	return nil
}
