package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/projection"
	"github.com/hero-engine/hero/internal/snapshot"
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

<user>.local.md is fully rebuilt every checkpoint — do not hand-edit
it. Anything you write outside the marker block is wiped on the next
run (a one-time backup is written alongside the file the first time
non-empty hand-content is detected). For preserved per-machine notes,
use a separately-named file like .hero/notes/<user>.md that no
automated tool touches.`,
	RunE: runNextCheckpoint,
}

func init() {
	nextCheckpointCmd.Flags().BoolVarP(&checkpointQuiet, "quiet", "q", false, "suppress success output")
}

func runNextCheckpoint(cmd *cobra.Command, args []string) error {
	// Auto-emit the user's last ask from the Stop-hook transcript
	// payload BEFORE projecting, so the just-recorded UserAsk renders
	// into .hero/next/<user>.md in this same checkpoint pass. Reads
	// stdin exactly once (via resolveSessionContext); writeCheckpoint
	// never touches stdin, so there is no double-consume. Entirely
	// best-effort — it never fails or hangs the Stop hook.
	autoEmitUserAsk(cmd.InOrStdin())

	path, err := writeCheckpoint()
	if err != nil {
		return err
	}
	if !checkpointQuiet {
		fmt.Printf("checkpoint written → %s\n", path)
	}
	return nil
}

// autoEmitUserAsk records the user's last transcript message as a
// UserAsk graph node so the projected handoff stops depending on the
// agent remembering to run `hero next ask`. It mirrors the manual
// command's (user, repo, domain) derivation exactly so the recorded
// node lands on the same singleton the projection reads.
//
// Contract: this is the end-of-turn Stop hook, which must NEVER fail or
// hang. Every error path is a no-op that logs to stderr and returns —
// same best-effort contract as writeUserHandoffFile / projectSnapshot.
//
//   - No stdin payload (other harnesses, the git post-commit fallback):
//     resolveSessionContext returns an empty TranscriptPath → return.
//   - Transcript missing / unreadable / malformed: lastUserAskFromTranscript
//     returns "" → return, leaving any existing UserAsk untouched.
//   - The transcript read is bounded (64 KiB / 1000 lines) so an
//     oversized or blocking transcript can't hang the hook.
func autoEmitUserAsk(stdin io.Reader) {
	ctx := resolveSessionContext(stdin, "")
	if ctx.TranscriptPath == "" {
		return
	}
	last := lastUserAskFromTranscript(ctx.TranscriptPath)
	if last == "" {
		return
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: auto-emit user ask: loading config: %v\n", err)
		return
	}
	heroDir := cfg.HeroDir(projectRoot)

	store, err := graph.Open(heroDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: auto-emit user ask: open graph: %v\n", err)
		return
	}
	defer store.Close()

	user := nextUserSlug(cfg)
	if user == "" {
		return
	}
	repoKey := gitutil.RepoKey(projectRoot)
	domain := graph.DomainFor(cfg, graph.IntrinsicActive)

	if err := handoff.RecordAsk(store, repoKey, handoff.UserAsk{
		User:      user,
		Domain:    domain,
		Text:      last,
		SessionID: ctx.SessionID,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: auto-emit user ask: record: %v\n", err)
	}
}

// writeCheckpoint writes two files:
//
//   - NEXT.md (or .hero/next/<user>.md in team mode): agent-authored
//     content only. Any embedded machine block is stripped on first
//     run after migration. This file no longer churns per-turn.
//
//   - .hero/next/<user>.local.md: gitignored. Contains only the
//     marker-bounded machine block (branch, recent commits, working-
//     tree, hot files, activity-since-last-checkpoint). Total
//     rewrite — any content outside the markers is discarded on each
//     run, with a one-time backup written when non-empty hand-content
//     is detected.
//
// Returns the NEXT.md path so the user-facing success message points
// at the file they think of as "their handoff."
func writeCheckpoint() (string, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}

	// A-1 stabilizer (cross-machine-handoff-slug-mismatch): pin the
	// current machine's derived identity into the COMMITTED hero.json so
	// every clone derives the same handoff slug regardless of local git
	// config. Best-effort and one-time — only fires when defaultAgent is
	// unset (so the slug is currently coming from volatile git/$USER).
	if persisted := persistDefaultAgentIfUnset(projectRoot, cfg); persisted != nil {
		cfg = *persisted
	}

	heroDir := cfg.HeroDir(projectRoot)
	nextPath := resolveNextPath(heroDir, cfg)
	localPath := resolveLocalStatePath(heroDir, cfg)

	// sharedNextPath is the project-shape NEXT.md the snapshot pointer
	// and the project projection target. In solo mode it equals
	// nextPath (resolveNextPath returns .hero/NEXT.md). In team mode
	// resolveNextPath returns the per-user .hero/next/<user>.md, which
	// is owned by writeUserHandoffFile (personal-briefing render) — so
	// the project-shape NEXT.md write must target the shared file
	// instead, otherwise both writers race for the per-user path and
	// the per-user file flips between project-shape and personal-
	// briefing content within a single checkpoint.
	sharedNextPath := filepath.Join(heroDir, nextFileName)

	// Pre-flight migration gate (revises AC-14 of next-as-projection):
	// when the repo hasn't been migrated to projection mode AND the
	// existing file carries legacy hand-authored content, auto-migrate
	// silently rather than refusing. The migration captures the legacy
	// body as a durable Note, ingests structured fields into the graph,
	// and flips next.projected — there is no human judgment required, so
	// Hero does the transition itself instead of punting a CLI
	// incantation back to the user.
	//
	// On migration FAILURE we keep the exact no-clobber safety the
	// original gate protected: NEXT.md is left byte-for-byte untouched
	// (never the nextPlaceholder overwrite), next.projected stays false,
	// and the user gets an actionable, human message.
	if !cfg.NextProjected() {
		if reason := detectUnmigratedNextMD(nextPath); reason != "" {
			if err := migrateToProjection(projectRoot, cfg, io.Discard); err != nil {
				return "", fmt.Errorf(
					"automatic NEXT.md migration failed (%s): %w — your NEXT.md was left untouched; "+
						"run `hero next migrate-to-projection` to retry or inspect the error",
					reason, err,
				)
			}
			// Reload config so the rest of writeCheckpoint sees
			// next.projected == true and takes the projection path
			// (which regenerates NEXT.md from the graph, now including
			// the just-ingested content).
			if reloaded, lerr := config.Load(projectRoot); lerr == nil {
				cfg = reloaded
			}
		}
	}

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
		if err := writeProjectedNextMD(sharedNextPath, projectRoot, heroDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: NEXT.md projection failed: %v\n", err)
			// Fall through to the legacy path so we still produce
			// a current file rather than nothing.
		} else {
			goto local
		}
	}
	{
		nextExisting, _ := os.ReadFile(sharedNextPath)
		nextBody := stripMachineBlock(string(nextExisting))
		if strings.TrimSpace(nextBody) == "" {
			nextBody = nextPlaceholder(projectRoot)
		}
		nextBody = strings.TrimRight(nextBody, "\n") + "\n"

		if _, err := writeFileIfChanged(sharedNextPath, []byte(nextBody), 0o644); err != nil {
			return "", fmt.Errorf("writing %s: %w", sharedNextPath, err)
		}
	}

local:

	// Local state file: marker-bounded machine block only. Total
	// rewrite each turn. Before discarding pre-existing hand-content,
	// back it up once so an accidental loss is recoverable.
	machineBlock := buildMachineBlock(projectRoot, heroDir, priorCheckpoint)
	localExisting, _ := os.ReadFile(localPath)
	if backupPath, ok, err := backupHandContentIfNeeded(localPath, string(localExisting)); err != nil {
		fmt.Fprintf(os.Stderr, "warning: backing up prior .local.md hand-content: %v\n", err)
	} else if ok {
		fmt.Fprintf(os.Stderr, "notice: pre-existing hand-content in %s backed up to %s before discard\n", localPath, backupPath)
	}
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

	// Project the project-shape snapshot (SNAPSHOT.md) and refresh
	// the SNAPSHOT pointer in NEXT.md / AGENTS.md. Non-fatal: the
	// snapshot projector logs and continues on every error so the
	// checkpoint never fails because of snapshot-side issues.
	projectSnapshot(projectRoot, heroDir, cfg, sharedNextPath)

	// Keep the project graph's Commit nodes current so the next
	// `hero resume`'s "Just changed" reflects commits made this session
	// — including ones made outside the git post-commit hook, since this
	// runs on every Stop/PreCompact checkpoint too. `Commit` nodes are
	// otherwise only written by `hero scan`/`graph reingest`, neither of
	// which runs on commit or session start (resume-brief-missing-
	// project-context). Bounded limit keeps `git log` cheap on the hot
	// path; WriteGitLogGraph is idempotent so repeated checkpoints never
	// duplicate nodes. Best-effort by the same contract as the rest of
	// the checkpoint: a graph/ingest error warns to stderr and is
	// swallowed — it must never fail the Stop hook.
	ingestRecentCommits(projectRoot, heroDir)

	return nextPath, nil
}

// ingestRecentCommits upserts the most recent commits into the graph as
// Commit nodes, keyed by gitutil.RepoKey(projectRoot) so the writer and
// the digest's justChangedSection reader agree on the repo partition.
// Bounded, idempotent, and entirely best-effort: every error path warns
// to stderr and returns, so the Stop-hook checkpoint never fails.
func ingestRecentCommits(projectRoot, heroDir string) {
	store, err := graph.Open(heroDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: commit graph ingest: open graph: %v\n", err)
		return
	}
	defer store.Close()
	repoKey := gitutil.RepoKey(projectRoot)
	if _, err := gitutil.WriteGitLogGraph(projectRoot, repoKey, 50, store); err != nil {
		fmt.Fprintf(os.Stderr, "warning: commit graph ingest failed: %v\n", err)
	}
}

// projectSnapshot refreshes .hero/SNAPSHOT.md and the pointer line
// inside NEXT.md / AGENTS.md. Best-effort: errors are logged to
// stderr and do not propagate. The archive evaluator runs inside
// the same call.
func projectSnapshot(projectRoot, heroDir string, cfg config.Config, nextPath string) {
	missionPath := filepath.Join(heroDir, "mission.md")
	mission := readMissionOneLiner(missionPath)
	projectName := filepath.Base(projectRoot)

	agentsMD := filepath.Join(projectRoot, "AGENTS.md")
	// Only update AGENTS.md when it already exists — never create one
	// just to drop the pointer. The discovery story is "if AGENTS.md
	// is here, we add a single-line pointer to it"; not "we manage
	// AGENTS.md".
	if _, err := os.Stat(agentsMD); err != nil {
		agentsMD = ""
	}

	archiveCfg := cfg.SnapshotArchive()
	_, err := snapshot.Project(snapshot.ProjectOptions{
		ProjectRoot:  projectRoot,
		HeroDir:      heroDir,
		ProjectName:  projectName,
		Mission:      mission,
		NextMDPath:   nextPath,
		AgentsMDPath: agentsMD,
		ArchiveConfig: snapshot.ArchiveConfig{
			StalenessCutoff:   archiveCfg.StalenessCutoff,
			MilestonesEnabled: cfg.SnapshotMilestonesEnabled(),
			ReleaseTagPattern: archiveCfg.ReleaseTagPattern,
			Retention:         archiveCfg.Retention,
			RetentionCount:    archiveCfg.RetentionCount,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: snapshot projection failed: %v\n", err)
	}
}

// readMissionOneLiner extracts the first non-empty, non-heading line
// of .hero/mission.md body (after frontmatter) to use as the
// snapshot's mission strap-line. Returns empty when the file is
// missing.
func readMissionOneLiner(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	src := string(data)
	// Strip a leading frontmatter block --- ... --- so the strap-line
	// we surface is body content, not a `title:` field.
	if strings.HasPrefix(src, "---") {
		rest := strings.TrimPrefix(src, "---")
		if i := strings.Index(rest, "\n---"); i >= 0 {
			src = rest[i+len("\n---"):]
		}
	}
	inMission := false
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## Mission") {
			inMission = true
			continue
		}
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "---") || strings.HasPrefix(t, ">") {
			continue
		}
		if inMission || strings.HasPrefix(t, "**") {
			// Strip leading bold markers.
			t = strings.Trim(t, "*_ ")
			if t != "" {
				return t
			}
		}
	}
	return ""
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
	cfg, _ := config.Load(projectRoot)
	content, err := projection.NextMD(store, projection.NextMDOptions{
		RepoKey:                 repoKey,
		Branch:                  currentBranch(projectRoot),
		Vocab:                   activeVocab(&cfg),
		Methodology:             activeMethodology(&cfg),
		HeroDir:                 heroDir,
		ProjectRoot:             projectRoot,
		RoadmapRecencyDays:      cfg.Roadmap.AmbientRecencyDaysOrDefault(),
		RoadmapStopNaggingHours: cfg.Roadmap.StopNaggingHoursOrDefault(),
	})
	if err != nil {
		return fmt.Errorf("project NEXT.md: %w", err)
	}
	_, err = writeProjectedFileIfSemanticChanged(nextPath, []byte(content), 0o644)
	return err
}

// writeUserHandoffFile renders .hero/next/<user>.md from the graph in
// BOTH solo and team mode. In team mode this file is the primary
// handoff (resolveNextPath returns it); in solo mode it is the
// per-user companion to the shared NEXT.md. The render is total-
// rewrite from the user-graph nodes (UserAsk / NextSuggestion /
// SessionReflection) — projections always win, and the semantic-
// change guard below suppresses updated-only churn.
func writeUserHandoffFile(projectRoot, heroDir string, cfg config.Config) error {
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

// persistDefaultAgentIfUnset is the A-1 stabilizer for
// cross-machine-handoff-slug-mismatch. When the merged config has no
// tracking.defaultAgent, the handoff slug is being derived from
// volatile local git/$USER config and will diverge across machines.
// This pins the current machine's derived slug into the COMMITTED
// hero.json (never hero.local.json) so every clone derives the same
// identity and the handoff keys line up.
//
// It is deliberately surgical: it reads the committed hero.json on its
// own (NOT the local-merged cfg), so a defaultAgent already set in
// hero.local.json does not suppress the committed write, and local-only
// secrets are never round-tripped into the committed file. Best-effort:
// any error is logged and skipped — a checkpoint must never fail on it.
//
// Returns the updated config when it wrote (so the caller can keep
// using the freshly-pinned slug this turn), or nil when it did nothing.
func persistDefaultAgentIfUnset(projectRoot string, merged config.Config) *config.Config {
	// Only act when the EFFECTIVE identity is unpinned. If the merged
	// config already has a defaultAgent (from committed or local config),
	// identity is already stable enough — don't churn the committed file.
	if merged.Tracking != nil && merged.Tracking.DefaultAgent != "" {
		return nil
	}

	slug := gitUserName()
	if slug == "" || slug == "unknown" {
		// Nothing stable to pin — leave the file alone rather than
		// committing a useless "unknown" identity.
		return nil
	}

	heroDir := filepath.Join(projectRoot, merged.Folder)
	configPath := filepath.Join(heroDir, config.ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		// No committed hero.json (uninitialized workspace) → nothing to
		// pin into. Skip silently; A-1 only stabilizes existing repos.
		return nil
	}

	// Read the COMMITTED file on its own — no local merge, no default
	// fill — so we write back exactly what was there plus the new field.
	var committed config.Config
	if err := json.Unmarshal(data, &committed); err != nil {
		fmt.Fprintf(os.Stderr, "warning: A-1 defaultAgent persist: parse hero.json: %v\n", err)
		return nil
	}
	if committed.Tracking != nil && committed.Tracking.DefaultAgent != "" {
		// Committed file already pins it (merged check missed it only if
		// local cleared it, which it can't). Nothing to do.
		return nil
	}
	if committed.Folder == "" {
		committed.Folder = merged.Folder
	}
	if committed.Tracking == nil {
		committed.Tracking = &config.TrackingConfig{}
	}
	committed.Tracking.DefaultAgent = slug

	if err := committed.Save(projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "warning: A-1 defaultAgent persist: write hero.json: %v\n", err)
		return nil
	}

	// Reflect the pin in the in-memory config the rest of this checkpoint
	// uses, so the slug is consistent within the turn.
	if merged.Tracking == nil {
		merged.Tracking = &config.TrackingConfig{}
	}
	merged.Tracking.DefaultAgent = slug
	return &merged
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

// legacyNextHeaders are the section headers used by hand-written
// pre-projection NEXT.md drafts. Their presence — combined with
// `next.projected == false` in hero.json — signals that the repo
// has unmigrated content that the projection path would silently
// wipe. The pre-flight gate at the top of writeCheckpoint refuses
// to write in that state and directs the user to run
// `hero next migrate-to-projection` first.
var legacyNextHeaders = []string{
	"## Just finished",
	"## Next",
	"## Tried and failed",
	"## Context to carry forward",
}

// detectUnmigratedNextMD returns a short human-readable reason
// string if NEXT.md at nextPath looks like unmigrated legacy
// content with substantive hand-written prose under legacy
// headers. Returns "" when the file is missing, empty, or only
// carries the agent-fills-this-in placeholder body that
// nextPlaceholder() emits — that placeholder is the legacy path's
// own output, not "real" content the user could lose. Callers
// should only consult this when `next.projected == false`; once
// migrated, the projection path owns the file and legacy fragments
// can only appear as transient merge debris (which the merge driver
// / Stop-hook self-heal already handle).
func detectUnmigratedNextMD(nextPath string) string {
	body, err := os.ReadFile(nextPath)
	if err != nil || len(bytes.TrimSpace(body)) == 0 {
		return ""
	}
	src := string(body)
	if strings.Contains(src, machineBlockStart) {
		return "legacy <!-- BEGIN HERO MACHINE STATE --> markers present"
	}
	for _, h := range legacyNextHeaders {
		if sectionHasRealContent(src, h) {
			return "legacy section header `" + h + "` contains hand-written content"
		}
	}
	return ""
}

// sectionHasRealContent returns true if the section under header
// has any non-blank, non-italic-placeholder body line. The italic
// markers (`_..._` or `*..*`) are the convention nextPlaceholder()
// uses to mark "the agent fills this in" — those are not real
// content and must not trigger the migration gate.
func sectionHasRealContent(src, header string) bool {
	lines := strings.Split(src, "\n")
	inSection := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if !inSection {
			if trimmed == header {
				inSection = true
			}
			continue
		}
		// Next H2 header ends the section.
		if strings.HasPrefix(trimmed, "## ") {
			return false
		}
		body := strings.TrimSpace(trimmed)
		if body == "" {
			continue
		}
		// Italic placeholder lines (`_..._` / `*..*`) — and the
		// "(omit if nothing meaningful)" comment baked into the
		// header itself — count as empty.
		if isItalicPlaceholder(body) {
			continue
		}
		return true
	}
	return false
}

// isItalicPlaceholder returns true if line is wrapped in single
// underscores or asterisks — the markdown convention
// nextPlaceholder() uses for "fill this in" hints.
func isItalicPlaceholder(line string) bool {
	if len(line) < 2 {
		return false
	}
	first, last := line[0], line[len(line)-1]
	return (first == '_' && last == '_') || (first == '*' && last == '*')
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
// file is machine-state-only — total rewrite every turn. Any
// content the caller supplies in `existing` is discarded; callers
// should back it up first via backupHandContentIfNeeded.
func rebuildLocalState(existing, machineBlock string) string {
	_ = existing
	return machineBlock + "\n"
}

// backupHandContentIfNeeded inspects existing .local.md content. If
// there is non-trivial content outside the marker block, it writes
// that content to <localPath>.bak.<UTC-RFC3339-timestamp> and returns
// (path, true, nil). When the hand-content matches the most-recent
// existing backup byte-for-byte, the write is skipped (idempotent
// across reruns with the same drift). Returns ("", false, nil) when
// there's nothing to back up.
func backupHandContentIfNeeded(localPath, existing string) (string, bool, error) {
	if strings.TrimSpace(existing) == "" {
		return "", false, nil
	}
	hand := strings.TrimSpace(stripMachineBlock(existing))
	if hand == "" {
		return "", false, nil
	}
	handBytes := []byte(hand + "\n")
	handSum := sha256.Sum256(handBytes)
	handHex := hex.EncodeToString(handSum[:])

	// Idempotent: if the most-recent existing backup in the same
	// dir matches this hand-content, skip the write.
	if prior, ok := mostRecentBackup(localPath); ok {
		if data, err := os.ReadFile(prior); err == nil {
			priorSum := sha256.Sum256(data)
			if hex.EncodeToString(priorSum[:]) == handHex {
				return "", false, nil
			}
		}
	}

	stamp := time.Now().UTC().Format(time.RFC3339)
	// Colons in RFC3339 are file-name safe on POSIX but awkward;
	// keep them — the spec explicitly says RFC3339 timestamps.
	backupPath := localPath + ".bak." + stamp
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(backupPath, handBytes, 0o644); err != nil {
		return "", false, err
	}
	return backupPath, true, nil
}

// mostRecentBackup returns the path to the most recently-written
// .bak.* file alongside localPath, if any.
func mostRecentBackup(localPath string) (string, bool) {
	dir := filepath.Dir(localPath)
	base := filepath.Base(localPath) + ".bak."
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	var matches []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasPrefix(e.Name(), base) {
			continue
		}
		matches = append(matches, e)
	}
	if len(matches) == 0 {
		return "", false
	}
	sort.Slice(matches, func(i, j int) bool {
		ii, _ := matches[i].Info()
		jj, _ := matches[j].Info()
		return ii.ModTime().After(jj.ModTime())
	})
	return filepath.Join(dir, matches[0].Name()), true
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
