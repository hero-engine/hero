package cli

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/projection"
	"github.com/spf13/cobra"
)

const (
	machineBlockStart = "<!-- BEGIN HERO MACHINE STATE -->"
	machineBlockEnd   = "<!-- END HERO MACHINE STATE -->"

	// localStateSuffix is appended to <user> for the gitignored
	// machine-state-plus-scratch file. Layout: .hero/next/<user>.local.md.
	localStateSuffix = ".local.md"
)

var checkpointQuiet bool

var nextCheckpointCmd = &cobra.Command{
	Use:   "checkpoint",
	Short: "Refresh per-machine state (.hero/next/<user>.local.md) and clean NEXT.md",
	Long: `Writes the machine-state block (branch, recent commits, dirty
files, hot files, activity-since-last-checkpoint) to a gitignored
.hero/next/<user>.local.md file. NEXT.md itself is left as agent-
authored content only — the machine block is no longer embedded
there.

Designed to be invoked from a host-tool Stop hook so machine-derived
context stays fresh without polluting NEXT.md with per-turn churn or
creating merge conflicts on a tracked file.

Hand-written content in <user>.local.md outside the marker block is
preserved across regens — drop reminders, scratch notes, anything
ad-hoc into that file and it survives every checkpoint.`,
	RunE: runNextCheckpoint,
}

func init() {
	nextCheckpointCmd.Flags().BoolVarP(&checkpointQuiet, "quiet", "q", false, "suppress success output")
}

func runNextCheckpoint(cmd *cobra.Command, args []string) error {
	path, err := writeCheckpoint()
	if err != nil {
		return err
	}
	if !checkpointQuiet {
		fmt.Printf("checkpoint written → %s\n", path)
	}
	return nil
}

// writeCheckpoint writes two files:
//
//   - NEXT.md (or .hero/next/<user>.md in team mode): agent-authored
//     content only. Any embedded machine block is stripped on first
//     run after migration. This file no longer churns per-turn.
//
//   - .hero/next/<user>.local.md: gitignored. Contains the marker-
//     bounded machine block (branch, recent commits, working-tree,
//     hot files, activity-since-last-checkpoint) at the top, with any
//     hand-written content preserved verbatim outside the markers.
//
// Returns the NEXT.md path so the user-facing success message points
// at the file they think of as "their handoff."
func writeCheckpoint() (string, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	nextPath := resolveNextPath(heroDir, cfg)
	localPath := resolveLocalStatePath(heroDir, cfg)

	// Prior checkpoint timestamp drives the activity-since delta.
	// Prefer the local state file; fall back to NEXT.md for the
	// one-time migration from the old layout.
	priorCheckpoint := readPriorCheckpoint(nextPath, localPath)

	// NEXT.md: two paths depending on whether projection is enabled.
	//
	// Projected (after migration): total-rewrite from graph each
	//   turn — projections always win. Agent hand-edits are wiped.
	// Legacy (default until migration): preserve agent-authored
	//   content; just strip any embedded machine block from prior
	//   layouts so it doesn't drift back in.
	if cfg.NextProjected() {
		if err := writeProjectedNextMD(nextPath, projectRoot, heroDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: NEXT.md projection failed: %v\n", err)
			// Fall through to the legacy path so we still produce
			// a current file rather than nothing.
		} else {
			goto local
		}
	}
	{
		nextExisting, _ := os.ReadFile(nextPath)
		nextBody := stripMachineBlock(string(nextExisting))
		if strings.TrimSpace(nextBody) == "" {
			nextBody = nextPlaceholder(projectRoot)
		}
		nextBody = strings.TrimRight(nextBody, "\n") + "\n"

		if _, err := writeFileIfChanged(nextPath, []byte(nextBody), 0o644); err != nil {
			return "", fmt.Errorf("writing %s: %w", nextPath, err)
		}
	}

local:

	// Local state file: marker-bounded machine block at the top,
	// any pre-existing hand-written content preserved below.
	machineBlock := buildMachineBlock(projectRoot, heroDir, priorCheckpoint)
	localExisting, _ := os.ReadFile(localPath)
	localBody := rebuildLocalState(string(localExisting), machineBlock)
	if _, err := writeFileIfChanged(localPath, []byte(localBody), 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", localPath, err)
	}

	// User durable file (.hero/next/<user>.md): full projection from
	// the user-graph nodes (UserAsk, NextSuggestion, SessionReflection)
	// plus author-attributed activity. Total-rewrite — projections
	// always win.
	if err := writeUserHandoffFile(projectRoot, heroDir, cfg); err != nil {
		// Non-fatal: a missing graph or graph error shouldn't block
		// the rest of the checkpoint. Surface as a comment in the
		// file but keep going.
		fmt.Fprintf(os.Stderr, "warning: user handoff projection failed: %v\n", err)
	}

	return nextPath, nil
}

// writeProjectedNextMD overwrites NEXT.md with a fresh project-state
// projection from the graph. Used when next.projected = true (set by
// hero next migrate-to-projection). Total-rewrite — any hand content
// in NEXT.md is wiped intentionally.
func writeProjectedNextMD(nextPath, projectRoot, heroDir string) error {
	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("open graph: %w", err)
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(projectRoot)
	content, err := projection.NextMD(store, projection.NextMDOptions{
		RepoKey: repoKey,
		Branch:  currentBranch(projectRoot),
	})
	if err != nil {
		return fmt.Errorf("project NEXT.md: %w", err)
	}
	_, err = writeProjectedFileIfSemanticChanged(nextPath, []byte(content), 0o644)
	return err
}

// writeUserHandoffFile renders .hero/next/<user>.md from the graph.
//
// In team mode, .hero/next/<user>.md is the primary handoff file
// (resolveNextPath returns it) and currently holds agent-authored
// content. To avoid clobbering that during the projection rollout,
// this function is a no-op in team mode — the migration command
// (Phase 7) handles the team-mode switchover deliberately.
func writeUserHandoffFile(projectRoot, heroDir string, cfg config.Config) error {
	if cfg.NextMode() == "team" {
		return nil
	}
	user := nextUserSlug(cfg)
	if user == "" {
		return nil
	}
	userPath := filepath.Join(heroDir, nextDirName, user+".md")

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("open graph: %w", err)
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(projectRoot)
	content, err := projection.UserHandoffMD(store, projection.UserHandoffOptions{
		User:    user,
		RepoKey: repoKey,
	})
	if err != nil {
		return fmt.Errorf("project user handoff: %w", err)
	}

	if _, err := writeProjectedFileIfSemanticChanged(userPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", userPath, err)
	}
	return nil
}

func writeFileIfChanged(path string, content []byte, mode fs.FileMode) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, content) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return false, err
	}
	return true, nil
}

func writeProjectedFileIfSemanticChanged(path string, content []byte, mode fs.FileMode) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil && normalizeUpdatedFrontmatter(string(existing)) == normalizeUpdatedFrontmatter(string(content)) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return writeFileIfChanged(path, content, mode)
}

func normalizeUpdatedFrontmatter(content string) string {
	const placeholder = "updated: <preserved>"
	if !strings.HasPrefix(content, "---\n") {
		return content
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return content
	}
	end += 4

	frontmatter := content[:end]
	rest := content[end:]
	lines := strings.Split(frontmatter, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "updated:") {
			lines[i] = placeholder
		}
	}
	return strings.Join(lines, "\n") + rest
}

// resolveLocalStatePath returns the gitignored per-user machine-state
// file path. Always lives under .hero/next/, named <user>.local.md.
// Same shape in solo and team modes — <user> comes from nextUserSlug.
func resolveLocalStatePath(heroDir string, cfg config.Config) string {
	user := nextUserSlug(cfg)
	return filepath.Join(heroDir, nextDirName, user+localStateSuffix)
}

// readPriorCheckpoint reads the "Updated:" timestamp from the local
// state file (preferred), falling back to NEXT.md so the first run
// after migration still computes a meaningful activity-since delta.
func readPriorCheckpoint(nextPath, localPath string) time.Time {
	if data, err := os.ReadFile(localPath); err == nil {
		if t := parsePriorCheckpoint(string(data)); !t.IsZero() {
			return t
		}
	}
	if data, err := os.ReadFile(nextPath); err == nil {
		return parsePriorCheckpoint(string(data))
	}
	return time.Time{}
}

// rebuildLocalState produces fresh local-state file contents. The
// machine block goes at the top; hand-written content from outside
// the prior block is preserved below. Idempotent: if existing is
// empty, returns just the block.
func rebuildLocalState(existing, machineBlock string) string {
	if strings.TrimSpace(existing) == "" {
		return machineBlock + "\n"
	}
	hand := strings.TrimSpace(stripMachineBlock(existing))
	if hand == "" {
		return machineBlock + "\n"
	}
	return machineBlock + "\n\n" + hand + "\n"
}

// priorCheckpointPattern reads "- **Updated:** YYYY-MM-DD HH:MM UTC"
// lines so successive checkpoints can compute "since last run" deltas
// without an extra state file. Empty/missing blocks return the zero
// time (caller treats as "no prior checkpoint, render full state").
var priorCheckpointPattern = regexp.MustCompile(`(?m)^-\s+\*\*Updated:\*\*\s+(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2})\s+UTC\s*$`)

func parsePriorCheckpoint(existing string) time.Time {
	start := strings.Index(existing, machineBlockStart)
	if start < 0 {
		return time.Time{}
	}
	end := strings.Index(existing, machineBlockEnd)
	if end < 0 {
		end = len(existing)
	}
	block := existing[start:end]
	m := priorCheckpointPattern.FindStringSubmatch(block)
	if m == nil {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02 15:04", m[1])
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func stripMachineBlock(s string) string {
	start := strings.Index(s, machineBlockStart)
	if start < 0 {
		return s
	}
	end := strings.Index(s, machineBlockEnd)
	if end < 0 {
		return strings.TrimRight(s[:start], "\n") + "\n"
	}
	end += len(machineBlockEnd)
	rest := strings.TrimLeft(s[end:], "\n")
	prefix := strings.TrimRight(s[:start], "\n")
	if rest == "" {
		return prefix + "\n"
	}
	return prefix + "\n\n" + rest
}

func buildMachineBlock(projectRoot, heroDir string, priorCheckpoint time.Time) string {
	now := time.Now().UTC()
	var b strings.Builder
	b.WriteString(machineBlockStart + "\n")
	b.WriteString("## Machine state\n")
	b.WriteString("_Auto-written by `hero next checkpoint`. Don't hand-edit this block — it gets rewritten every turn._\n\n")
	fmt.Fprintf(&b, "- **Updated:** %s\n", now.Format("2006-01-02 15:04 UTC"))
	fmt.Fprintf(&b, "- **Branch:** %s\n", currentBranch(projectRoot))
	if !priorCheckpoint.IsZero() {
		fmt.Fprintf(&b, "- **Since last checkpoint:** %s ago\n", durationSince(now, priorCheckpoint))
	}

	// Activity since last checkpoint — the mechanical "Just finished"
	// half. Commits + AC flips that landed between the prior block's
	// timestamp and now. Skipped on first-ever checkpoint (no prior).
	if !priorCheckpoint.IsZero() {
		writeActivitySince(&b, projectRoot, heroDir, priorCheckpoint)
	}

	commits := recentCommits(projectRoot, 5)
	if len(commits) > 0 {
		b.WriteString("\n### Recent commits\n")
		for _, c := range commits {
			fmt.Fprintf(&b, "- `%s` %s\n", c.hash, c.message)
		}
	}

	if isWorkingTreeDirty(projectRoot) {
		files := uncommittedFiles(projectRoot)
		b.WriteString("\n### Working tree (dirty)\n")
		for _, f := range files {
			fmt.Fprintf(&b, "- %s\n", f)
		}
	} else {
		b.WriteString("\n### Working tree\nclean\n")
	}

	hot := hotFiles(projectRoot, 5)
	if len(hot) > 0 {
		b.WriteString("\n### Hot files\n")
		for _, f := range hot {
			fmt.Fprintf(&b, "- %s\n", f)
		}
	}

	b.WriteString(machineBlockEnd + "\n")
	return b.String()
}

// writeActivitySince emits two grouped lists: commits and AC status
// flips landed since priorCheckpoint. Both queries are best-effort —
// a git or graph error skips the corresponding sub-list rather than
// failing the whole checkpoint.
func writeActivitySince(b *strings.Builder, projectRoot, heroDir string, since time.Time) {
	commits := commitsSince(projectRoot, since)
	flips := acFlipsSince(projectRoot, heroDir, since)

	if len(commits) == 0 && len(flips) == 0 {
		return
	}

	fmt.Fprintf(b, "\n### Activity since %s UTC\n", since.Format("2006-01-02 15:04"))

	if len(commits) > 0 {
		b.WriteString("\n**Commits landed:**\n")
		for _, c := range commits {
			fmt.Fprintf(b, "- `%s` %s\n", c.hash, c.message)
		}
	}

	if len(flips) > 0 {
		b.WriteString("\n**AC status flips:**\n")
		for _, f := range flips {
			fmt.Fprintf(b, "- %s `%s` → %s\n", f.glyph(), f.key, f.status)
		}
	}
}

// commitsSince returns commits with author-date ≥ since. Uses
// --no-merges to keep noise out and parses oneline format. Empty on
// any git error (best-effort).
func commitsSince(projectRoot string, since time.Time) []commit {
	cmd := exec.Command("git", "-C", projectRoot, "log",
		"--no-merges",
		"--oneline",
		"--since="+since.Format(time.RFC3339))
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var commits []commit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			commits = append(commits, commit{hash: parts[0], message: parts[1]})
		}
	}
	return commits
}

// acFlip is a Criterion node whose current row landed at-or-after
// the prior checkpoint — i.e. its status flipped recently.
type acFlip struct {
	key, status string
}

func (f acFlip) glyph() string {
	switch f.status {
	case "passing":
		return "✅"
	case "failing":
		return "❌"
	case "regressed":
		return "⚠️"
	case "retired":
		return "⊘"
	default:
		return "◯"
	}
}

// acFlipsSince queries Criterion nodes whose valid_from is ≥ since.
// Filters out scan-default flips (proposed/unknown) — those are
// noise from re-ingest, not real status events. Uses the same graph
// the digest reads, so the data stays consistent across surfaces.
func acFlipsSince(projectRoot, heroDir string, since time.Time) []acFlip {
	store, err := graph.Open(heroDir)
	if err != nil {
		return nil
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(projectRoot)
	criteria, err := acceptance.ChangedSince(store, since.Format(time.RFC3339))
	if err != nil {
		return nil
	}
	var out []acFlip
	for _, c := range criteria {
		if c.Parent == "" {
			continue
		}
		// Filter out the scan-default placeholders — those are
		// re-ingest noise, not user-visible status events.
		switch c.Status {
		case "passing", "failing", "regressed", "retired":
			out = append(out, acFlip{key: c.Key, status: c.Status})
		}
	}
	_ = repoKey // ChangedSince doesn't filter by repo today; future
	// upgrade can scope to repoKey when corpora cross repos.
	return out
}

// durationSince renders a coarse "Xh Ym" / "Xd Yh" age suitable for
// the "Since last checkpoint" header. Sub-minute deltas show as "<1m".
func durationSince(now, prior time.Time) string {
	d := now.Sub(prior)
	if d < time.Minute {
		return "<1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) - h*60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh %dm", h, m)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) - days*24
	if hours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, hours)
}

func nextPlaceholder(projectRoot string) string {
	return fmt.Sprintf(`---
updated: %s
branch: %s
---

## Last user ask
_The agent fills this in with a 1-3 line excerpt of what the user just asked, in their own words._

## Just finished
_The agent fills this in after meaningful work._

## Next
_The concrete next step. Include a runnable pointer: `+"`/deliver <slug>`"+` or a spec path._

## Blocked on (omit if clear)

## Tried and failed (omit if N/A)

## Context to carry forward (omit if nothing meaningful)
`,
		time.Now().UTC().Format(time.RFC3339),
		currentBranch(projectRoot),
	)
}

// --- helpers (shared with handoff.go alias) ---

type commit struct {
	hash    string
	message string
}

func recentCommits(projectRoot string, n int) []commit {
	cmd := exec.Command("git", "-C", projectRoot, "log", "--oneline", fmt.Sprintf("-%d", n))
	out, _ := cmd.Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var commits []commit
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			commits = append(commits, commit{hash: parts[0], message: parts[1]})
		}
	}
	return commits
}

func currentBranch(projectRoot string) string {
	cmd := exec.Command("git", "-C", projectRoot, "rev-parse", "--abbrev-ref", "HEAD")
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func isWorkingTreeDirty(projectRoot string) bool {
	cmd := exec.Command("git", "-C", projectRoot, "status", "--porcelain")
	out, _ := cmd.Output()
	return len(strings.TrimSpace(string(out))) > 0
}

func uncommittedFiles(projectRoot string) []string {
	cmd := exec.Command("git", "-C", projectRoot, "status", "--short")
	out, _ := cmd.Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		files = append(files, line)
	}
	return files
}

// hotFiles returns the most-recently-touched files in the last few commits.
// Uses `git log --name-only -<n>` so it works even on shallow history.
func hotFiles(projectRoot string, n int) []string {
	cmd := exec.Command("git", "-C", projectRoot, "log", "--name-only", "--pretty=format:", "-5")
	out, _ := cmd.Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	seen := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		files = append(files, line)
		if len(files) >= n {
			break
		}
	}
	return files
}
