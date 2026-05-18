package snapshot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// SnapshotFileName is the bare filename of the live snapshot file
// inside the hero directory (heroDir/SNAPSHOT.md).
const SnapshotFileName = "SNAPSHOT.md"

// ProjectOptions tunes a single Project() call.
type ProjectOptions struct {
	ProjectRoot string
	HeroDir     string
	ProjectName string
	Mission     string

	// NextMDPath / AgentsMDPath are the absolute paths to the anchor
	// files the pointer-writer should idempotently update. Empty
	// disables the corresponding pointer.
	NextMDPath   string
	AgentsMDPath string

	// ArchiveConfig governs the archive evaluator. Zero-value uses
	// the documented defaults.
	ArchiveConfig ArchiveConfig

	// ManualArchive triggers a manual archive on this run regardless
	// of cutoff state.
	ManualArchive       bool
	ManualArchiveLabel  string

	// Now is injected for deterministic tests. Defaults to time.Now().
	Now time.Time
}

// ProjectResult reports what the projector wrote on this call.
type ProjectResult struct {
	SnapshotPath  string // absolute path of the written SNAPSHOT.md
	SnapshotBytes int
	Wrote         bool // false when content hash matched and no write occurred
	Archives      []ArchiveRecord
	SkippedArchives []TriggerHit
	DurationMS    int64
}

// Project is the single entry point for the projection pass.
// It (1) gathers state, (2) builds the snapshot, (3) renders to
// markdown, (4) writes SNAPSHOT.md when bytes changed,
// (5) ensures the NEXT.md / AGENTS.md pointer is in place,
// (6) evaluates archive triggers and writes any that fire,
// (7) applies retention.
//
// All steps are best-effort: pointer / archive failures do not block
// the SNAPSHOT.md write or fail the projector. Errors are returned
// as a combined message at the end so the caller can surface them
// without taking down the Stop hook.
func Project(opts ProjectOptions) (*ProjectResult, error) {
	start := time.Now()
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.HeroDir == "" {
		return nil, fmt.Errorf("snapshot: Project requires HeroDir")
	}

	// 1. Load specs + override.
	allSpecs, err := spec.Discover(opts.HeroDir)
	if err != nil {
		// Tolerate "no workspace" gracefully — render an empty snapshot.
		allSpecs = nil
	}
	override, _ := LoadOverride(opts.HeroDir)

	// 2. Discover shipped tags (best-effort).
	shippedTags := scanGitTags(opts.ProjectRoot, opts.ArchiveConfig.ReleaseTagPattern)

	// 3. Build snapshot.
	snap, err := Build(BuildOptions{
		ProjectRoot: opts.ProjectRoot,
		HeroDir:     opts.HeroDir,
		ProjectName: opts.ProjectName,
		Mission:     opts.Mission,
		Now:         opts.Now,
	}, allSpecs, override, shippedTags)
	if err != nil {
		return nil, fmt.Errorf("snapshot: build: %w", err)
	}

	// 4. Render markdown.
	rendered, err := Render(snap, FormatMarkdown)
	if err != nil {
		return nil, fmt.Errorf("snapshot: render: %w", err)
	}

	// 5. Write SNAPSHOT.md (content-hash skip).
	snapshotPath := filepath.Join(opts.HeroDir, SnapshotFileName)
	wrote, err := writeIfChanged(snapshotPath, rendered)
	if err != nil {
		return nil, fmt.Errorf("snapshot: write %s: %w", snapshotPath, err)
	}

	result := &ProjectResult{
		SnapshotPath:  snapshotPath,
		SnapshotBytes: len(rendered),
		Wrote:         wrote,
	}

	// 6. Pointer writes (idempotent; non-fatal).
	if opts.NextMDPath != "" || opts.AgentsMDPath != "" {
		if err := EnsurePointer(opts.NextMDPath, opts.AgentsMDPath); err != nil {
			fmt.Fprintf(os.Stderr, "snapshot: pointer: %v\n", err)
		}
	}

	// 7. Archive evaluation.
	archives, _ := List(opts.HeroDir)
	newlyCompleted := newlyCompletedInitiatives(allSpecs, archives)
	newTags := newReleaseTags(shippedTags, archives)
	hits := EvaluateTriggers(EvaluateTriggersInput{
		Now:                       opts.Now,
		ExistingArchives:          archives,
		NewlyCompletedInitiatives: newlyCompleted,
		NewReleaseTags:            newTags,
		ManualRequested:           opts.ManualArchive,
		ManualLabel:               opts.ManualArchiveLabel,
		Config:                    opts.ArchiveConfig,
	})
	if len(hits) > 0 {
		gitCommit := headCommit(opts.ProjectRoot)
		out, archiveErr := MaybeWrite(MaybeWriteInput{
			Rendered:  rendered,
			HeroDir:   opts.HeroDir,
			Now:       opts.Now,
			GitCommit: gitCommit,
			Triggers:  hits,
		})
		if archiveErr != nil {
			fmt.Fprintf(os.Stderr, "snapshot: archive: %v\n", archiveErr)
		}
		result.Archives = out.Written
		result.SkippedArchives = out.Skipped

		// 8. Apply retention (only when an archive was actually written).
		if len(out.Written) > 0 {
			if _, err := ApplyRetention(opts.HeroDir, opts.ArchiveConfig); err != nil {
				fmt.Fprintf(os.Stderr, "snapshot: retention: %v\n", err)
			}
		}
	}

	result.DurationMS = time.Since(start).Milliseconds()
	return result, nil
}

// writeIfChanged writes data to path when the existing bytes differ.
// Returns (wrote, error).
func writeIfChanged(path string, data []byte) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == string(data) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, data, 0o644)
}

// scanGitTags lists git tags matching pattern. Best-effort; returns
// an empty map when git is unavailable or no tags match.
func scanGitTags(root, pattern string) map[string]bool {
	out := map[string]bool{}
	cmd := exec.Command("git", "-C", root, "tag", "-l")
	data, err := cmd.Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out[line] = true
	}
	return out
}

// newlyCompletedInitiatives returns initiative slugs that completed
// since the most-recent archive. When no archives exist, returns
// nothing (the first-run baseline).
func newlyCompletedInitiatives(specs []*spec.Spec, archives []ArchiveRecord) []string {
	if len(archives) == 0 {
		return nil
	}
	// Find the newest archive date.
	var newest time.Time
	for _, a := range archives {
		t, err := time.Parse("2006-01-02", a.Date)
		if err == nil && t.After(newest) {
			newest = t
		}
	}
	if newest.IsZero() {
		return nil
	}
	// Add 1 day so we look strictly after that date.
	cutoff := newest.Add(24 * time.Hour)

	var out []string
	for _, s := range specs {
		if s == nil {
			continue
		}
		if s.Type != "initiative" {
			continue
		}
		if s.Status != "completed" {
			continue
		}
		if s.ModifiedAt.After(cutoff) {
			out = append(out, s.Slug)
		}
	}
	return out
}

// newReleaseTags returns tag names present in current that weren't
// represented in any prior archive (best-effort: any tag not labeled
// by a prior archive is "new").
//
// First-run safety: when no archives exist, every tag in the repo
// would otherwise look "new" — that's surprising on install
// (snapshot would write one archive per historical tag). Return
// nothing on first run; the staleness safety net and future
// milestones cover the gap.
func newReleaseTags(current map[string]bool, archives []ArchiveRecord) []string {
	if len(archives) == 0 {
		return nil
	}
	known := map[string]bool{}
	for _, a := range archives {
		if a.Label != "" {
			known[a.Label] = true
		}
	}
	var out []string
	for tag := range current {
		if !known[tag] {
			out = append(out, tag)
		}
	}
	return out
}

// headCommit returns the short SHA of the current HEAD, or empty
// when git isn't available.
func headCommit(root string) string {
	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	data, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
