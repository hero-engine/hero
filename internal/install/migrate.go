package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// buildCandidate reads a file's metadata and a short content hash for
// drift detection.
func buildCandidate(path string) (MigrationCandidate, error) {
	info, err := os.Stat(path)
	if err != nil {
		return MigrationCandidate{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return MigrationCandidate{}, err
	}
	sum := sha256.Sum256(data)
	return MigrationCandidate{
		Path:     path,
		ModTime:  info.ModTime(),
		Size:     info.Size(),
		ShortSum: hex.EncodeToString(sum[:])[:8],
	}, nil
}

// migrate.go — single-source-install P3 migration command.
//
// `hero install --migrate` converts a legacy multi-harness install
// (legacy-multi-harness-shape: N harness directories each holding their own drifted
// physical copies of the same agent/command/skill content) into the
// P2 canonical-tree layout (one canonical copy in `.hero/` or user's
// configured override, each harness dir is a symlink into it).
//
// Steps:
//
//   1. Detect installed harness targets in the project (presence of
//      .claude/, .opencode/, .codex/, .cursor/, .github/copilot/, .ai/,
//      etc., with at least one of agents/commands/skills present).
//
//   2. For each content kind (agents, commands, skills), collect every
//      copy currently on disk across all detected harness dirs AND the
//      canonical dir if present. Group by base filename.
//
//   3. For each filename group with drift (different content across
//      copies), pick the file with the newest mtime as the winner.
//      Files with identical content are not "drift" — any copy works.
//
//   4. Promote the winner into the canonical location (`.hero/<kind>/`
//      or the user's configured override).
//
//   5. Run a normal install for each detected target with --force so
//      the harness dirs become symlinks pointing at canonical.
//
//   6. Report what was done, what was reconciled, and any unresolved
//      conditions.
//
// Idempotent: re-running migrate against an already-canonical project
// detects no drift, no migration needed; just runs install through.

// MigrationConflict describes a drift the migration resolved.
type MigrationConflict struct {
	// Kind is "agents", "commands", or "skills".
	Kind string `json:"kind"`
	// File is the base filename (e.g. "engineer.md" or "spec-format")
	// — for skills this is the SKILL.md directory name.
	File string `json:"file"`
	// Candidates is every source path considered, with mtime and a
	// short content-hash for diagnostics.
	Candidates []MigrationCandidate `json:"candidates"`
	// Winner is the path picked (the entry from Candidates with the
	// newest mtime among unique-content candidates).
	Winner string `json:"winner"`
}

// MigrationCandidate is one copy of a content file considered during
// migration.
type MigrationCandidate struct {
	Path     string    `json:"path"`
	ModTime  time.Time `json:"mtime"`
	Size     int64     `json:"size"`
	ShortSum string    `json:"short_sum"` // first 8 chars of sha256 for compact reporting
}

// MigrationReport summarizes what `hero install --migrate` did.
type MigrationReport struct {
	DetectedTargets []Target            `json:"detected_targets"`
	PromotedFiles   map[string][]string `json:"promoted_files"` // kind → list of canonical paths written
	Conflicts       []MigrationConflict `json:"conflicts"`
	SkippedTargets  []Target            `json:"skipped_targets"` // detected but install failed (with reason in StringReport)
	TargetResults   map[Target]*Result  `json:"target_results"`  // per-target install Result
	DryRun          bool                `json:"dry_run"`
	Errors          []string            `json:"errors"`
}

// StringReport produces a human-readable summary of the migration.
func (r *MigrationReport) StringReport() string {
	var sb strings.Builder
	if r.DryRun {
		sb.WriteString("DRY RUN — no changes were written.\n\n")
	}
	sb.WriteString(fmt.Sprintf("Detected harness targets: %v\n", targetNames(r.DetectedTargets)))
	if len(r.Conflicts) == 0 {
		sb.WriteString("No drift detected across harness directories.\n")
	} else {
		sb.WriteString(fmt.Sprintf("Reconciled %d drifted file(s) — newest mtime wins:\n", len(r.Conflicts)))
		for _, c := range r.Conflicts {
			sb.WriteString(fmt.Sprintf("  %s/%s ← winner: %s\n", c.Kind, c.File, c.Winner))
			for _, cand := range c.Candidates {
				marker := "  "
				if cand.Path == c.Winner {
					marker = "→ "
				}
				sb.WriteString(fmt.Sprintf("    %s%s  (mtime=%s sum=%s)\n",
					marker, cand.Path, cand.ModTime.Format(time.RFC3339), cand.ShortSum))
			}
		}
	}
	if len(r.PromotedFiles) > 0 {
		sb.WriteString("\nPromoted to canonical:\n")
		for _, kind := range []string{"agents", "commands", "skills"} {
			files := r.PromotedFiles[kind]
			if len(files) == 0 {
				continue
			}
			sb.WriteString(fmt.Sprintf("  %s: %d file(s)\n", kind, len(files)))
		}
	}
	if len(r.Errors) > 0 {
		sb.WriteString("\nErrors:\n")
		for _, e := range r.Errors {
			sb.WriteString("  - " + e + "\n")
		}
	}
	return sb.String()
}

func targetNames(ts []Target) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = string(t)
	}
	return out
}

// RunMigrate is the entry point for `hero install --migrate`. The
// caller (CLI) renders the returned report.
func RunMigrate(opts Options) (*MigrationReport, error) {
	return runMigrate(opts)
}

func runMigrate(opts Options) (*MigrationReport, error) {
	if opts.Mode != ModeProject || opts.TargetDir == "" {
		return nil, fmt.Errorf("--migrate only works in project mode with an explicit target path")
	}
	if _, err := os.Stat(opts.TargetDir); err != nil {
		return nil, fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
	}
	heroDir := filepath.Join(opts.TargetDir, ".hero")
	if _, err := os.Stat(heroDir); err != nil {
		return nil, fmt.Errorf("--migrate requires an existing .hero/ workspace; run `hero init %s` first", opts.TargetDir)
	}

	report := &MigrationReport{
		PromotedFiles: map[string][]string{},
		TargetResults: map[Target]*Result{},
		DryRun:        opts.DryRun,
	}

	// 1. Detect installed harness targets.
	report.DetectedTargets = DetectInstalledTargets(opts.TargetDir)
	if len(report.DetectedTargets) == 0 {
		return report, fmt.Errorf("no installed harness targets detected at %s — nothing to migrate", opts.TargetDir)
	}

	// 2. Resolve canonical paths from hero.json overrides.
	agentsCanonical, commandsCanonical, skillsCanonical, err := ResolveCanonicalDirs(opts.TargetDir)
	if err != nil {
		return report, err
	}

	// 3. For each kind, gather candidates across harness dirs +
	// canonical, reconcile drift, promote winners.
	for _, kind := range []string{"agents", "commands", "skills"} {
		canonicalKind := map[string]string{
			"agents":   agentsCanonical,
			"commands": commandsCanonical,
			"skills":   skillsCanonical,
		}[kind]

		candidates, err := gatherKindCandidates(opts.TargetDir, kind, canonicalKind, report.DetectedTargets)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("gathering %s candidates: %v", kind, err))
			continue
		}

		// candidates is map[filename][]MigrationCandidate. For each, pick winner.
		for fname, cands := range candidates {
			winner, conflict := pickWinner(cands)
			if conflict != nil {
				conflict.Kind = kind
				conflict.File = fname
				report.Conflicts = append(report.Conflicts, *conflict)
			}
			if winner == "" {
				continue
			}
			// Promote winner to canonical.
			dst := canonicalDestPath(canonicalKind, kind, fname)
			if !opts.DryRun {
				if err := copyFile(winner, dst); err != nil {
					report.Errors = append(report.Errors, fmt.Sprintf("promote %s: %v", fname, err))
					continue
				}
			}
			report.PromotedFiles[kind] = append(report.PromotedFiles[kind], dst)
		}
	}

	// 4. Run normal install for each detected target with --force, but
	// skip the canonical-render step — we've already written the
	// disk-detected winner content into canonical above, and the
	// embedded source would clobber it.
	for _, target := range report.DetectedTargets {
		targetOpts := opts
		targetOpts.Target = target
		targetOpts.Force = true
		targetOpts.SkipCanonicalRender = true

		res, err := runTargetForMigration(targetOpts)
		if err != nil {
			report.SkippedTargets = append(report.SkippedTargets, target)
			report.Errors = append(report.Errors, fmt.Sprintf("install %s: %v", target, err))
			continue
		}
		report.TargetResults[target] = res
	}

	return report, nil
}

// gatherKindCandidates returns every existing copy of every file of
// the given kind across the canonical dir + all detected harness dirs.
// Keyed by file basename (for agents/commands) or directory basename
// (for skills, which use <name>/SKILL.md layout).
func gatherKindCandidates(targetDir, kind, canonical string, targets []Target) (map[string][]MigrationCandidate, error) {
	out := map[string][]MigrationCandidate{}

	// Sources to scan: canonical + each target's harness dir for this kind.
	sources := []string{canonical}
	for _, t := range targets {
		layout := LayoutFor(t)
		if layout == nil {
			continue
		}
		sources = append(sources, filepath.Join(targetDir, layout.SubDir, kind))
	}

	for _, src := range sources {
		if err := scanKindSource(src, kind, out); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// scanKindSource walks a single source dir and adds its candidates to
// out (keyed by file/skill name).
func scanKindSource(src, kind string, out map[string][]MigrationCandidate) error {
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return nil
	}
	// Skip symlinks — they're already pointing at canonical somewhere.
	if linfo, lerr := os.Lstat(src); lerr == nil && linfo.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	if kind == "skills" {
		return scanSkillsDir(src, out)
	}
	return scanFlatDir(src, out)
}

// scanFlatDir handles agents/, commands/, and any flat-layout dir.
func scanFlatDir(src string, out map[string][]MigrationCandidate) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		full := filepath.Join(src, e.Name())
		cand, err := buildCandidate(full)
		if err != nil {
			continue
		}
		out[e.Name()] = append(out[e.Name()], cand)
	}
	return nil
}

// scanSkillsDir handles the SKILL.md directory layout. Skill name is
// the directory name; the candidate file is <dir>/SKILL.md inside.
// Also handles legacy flat-layout `<name>.md` for fully-migrating
// projects that still have stragglers.
func scanSkillsDir(src string, out map[string][]MigrationCandidate) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			skillFile := filepath.Join(src, e.Name(), "SKILL.md")
			if _, err := os.Stat(skillFile); err != nil {
				continue
			}
			cand, err := buildCandidate(skillFile)
			if err != nil {
				continue
			}
			out[e.Name()] = append(out[e.Name()], cand)
		} else if strings.HasSuffix(e.Name(), ".md") {
			// Legacy flat skill — record under base name (no .md).
			name := strings.TrimSuffix(e.Name(), ".md")
			full := filepath.Join(src, e.Name())
			cand, err := buildCandidate(full)
			if err != nil {
				continue
			}
			out[name] = append(out[name], cand)
		}
	}
	return nil
}

// canonicalDestPath produces the destination path under canonical
// where a winner of given kind/name should land.
func canonicalDestPath(canonical, kind, name string) string {
	if kind == "skills" {
		// Skill names are directory names; SKILL.md is the content.
		return filepath.Join(canonical, name, "SKILL.md")
	}
	return filepath.Join(canonical, name)
}

// pickWinner returns the path of the newest-mtime candidate, and a
// MigrationConflict description if there was drift (more than one
// unique content hash). Returns ("", nil) if candidates is empty.
func pickWinner(candidates []MigrationCandidate) (string, *MigrationConflict) {
	if len(candidates) == 0 {
		return "", nil
	}
	if len(candidates) == 1 {
		return candidates[0].Path, nil
	}

	// Check for drift: how many unique content hashes?
	uniqueSums := map[string]bool{}
	for _, c := range candidates {
		uniqueSums[c.ShortSum] = true
	}

	// Pick newest mtime regardless.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ModTime.After(candidates[j].ModTime)
	})
	winner := candidates[0].Path

	if len(uniqueSums) <= 1 {
		// All identical — no drift. Pick newest just for consistency
		// (mtime determinism), no conflict reported.
		return winner, nil
	}

	return winner, &MigrationConflict{
		Candidates: candidates,
		Winner:     winner,
	}
}

// runTargetForMigration invokes the regular target installer with the
// migration's options. Separated so the migration can capture per-target
// failures without aborting the whole run.
func runTargetForMigration(opts Options) (*Result, error) {
	return Run(opts)
}

// copyFile copies src to dst (creating parent dirs). If src == dst,
// it's a no-op (the winner is already at the canonical location).
func copyFile(src, dst string) error {
	srcClean := filepath.Clean(src)
	dstClean := filepath.Clean(dst)
	if srcClean == dstClean {
		return nil
	}

	// If dst is a symlink, follow it for comparison and remove before
	// writing.
	if info, err := os.Lstat(dst); err == nil && info.Mode()&os.ModeSymlink != 0 {
		_ = os.Remove(dst)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
