package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete <spec-path>",
	Short: "Mark a spec as completed and move it to specs/",
	Long: `Transitions a spec from planning to completed:
  1. Updates the status to 'completed' in the spec frontmatter
  2. Moves the spec from planning/ to specs/ (uses git mv if available)
  3. Re-indexes the corpus
  4. Syncs to the work tracker if configured

Works with feature, bug, and initiative specs.`,
	Args: cobra.ExactArgs(1),
	RunE: runComplete,
}

func runComplete(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specPath := args[0]

	s, err := spec.ParseFile(specPath)
	if err != nil {
		return fmt.Errorf("parsing spec %s: %w", specPath, err)
	}

	// Validate it's a completable spec
	if !s.IsWorkSpec() && s.Type != spec.TypeInitiative {
		return fmt.Errorf("only feature, bug, and initiative specs can be completed (this is a %s spec)", s.Type)
	}

	// Gate: work specs (feature, bug) must go through `hero spec verify`
	// which checks the Completion Ledger + audit report before flipping
	// status. Direct `hero spec complete` bypasses those gates. Redirect
	// the caller to verify instead.
	if s.IsWorkSpec() && s.Status != spec.StatusCompleted {
		fmt.Fprintf(os.Stderr, "Use `hero spec verify %s` instead — verify checks delivery gates\n"+
			"(Completion Ledger, audit report, test coverage) before marking complete.\n"+
			"If you need to force-complete without gates, use `hero spec verify %s --force`.\n",
			s.Slug, s.Slug)
		return fmt.Errorf("work specs must be completed via `hero spec verify`, not `hero spec complete`")
	}

	// Each step below is independently idempotent. The previous version
	// gated everything on `status == completed`, which stranded specs in
	// planning/ whenever something else flipped the status first
	// (`/deliver`, an agent edit, `hero check --reconcile`). Now we run
	// what's needed and skip what isn't, only erroring when the spec
	// has fully completed paperwork.
	alreadyCompleted := s.Status == spec.StatusCompleted
	alreadyMoved := isAlreadyInSpecsDir(specPath, heroDir)
	if alreadyCompleted && alreadyMoved {
		// Still ensure the dashboard sees this completion — emitCompletionEvents
		// is idempotent and will skip if a delivery_complete event already
		// exists for this slug within the last 24h.
		emitCompletionEvents(heroDir, s.Slug, s.Title)
		fmt.Printf("Spec %s is already completed and lives under specs/ — nothing to do.\n", s.Slug)
		return nil
	}

	// Step 1: Update status in frontmatter (skip if already done).
	if !alreadyCompleted {
		if err := updateFrontmatterStatus(specPath, "completed"); err != nil {
			return fmt.Errorf("updating status: %w", err)
		}
		fmt.Printf("Updated status to completed\n")
	} else {
		fmt.Printf("Status was already completed.\n")
	}

	// Step 2: Move from planning/ to specs/. moveToSpecs is idempotent —
	// it returns (path, false, nil) when the spec isn't under planning/,
	// so we always call it and let it self-no-op.
	destPath, moved, err := moveToSpecs(specPath, heroDir)
	if err != nil {
		return fmt.Errorf("moving spec: %w", err)
	}
	if moved {
		fmt.Printf("Moved %s → %s\n", specPath, destPath)
	}

	// Step 3: Re-index — always safe.
	stats, err := index.Rebuild(heroDir)
	if err != nil {
		return fmt.Errorf("rebuilding index: %w", err)
	}
	fmt.Printf("Re-indexed (%d specs)\n", stats.TotalSpecs)

	// Step 4: Sync to tracker if configured and spec has a tracker_id
	if s.TrackerID != "" && cfg.Tracker != nil && cfg.Tracker.Type != "none" && cfg.Tracker.PostOnDeliver {
		t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
		if err == nil {
			if err := t.UpdateStatus(s.TrackerID, spec.StatusCompleted); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not update tracker: %v\n", err)
			} else {
				fmt.Printf("Updated %s issue %s → Completed\n", t.Name(), s.TrackerID)
			}
		}
	}

	// Step 5: Emit lifecycle events so the dashboard's shipped-spec
	// counters (which read .hero/events.log) reflect this completion.
	// Idempotent — a re-run within 24h skips the duplicate.
	emitCompletionEvents(heroDir, s.Slug, s.Title)

	fmt.Printf("Completed spec: %s\n", s.Slug)
	return nil
}

// emitCompletionEvents writes the delivery_complete + spec.status_changed
// events that the dashboard reads to count shipped specs. Idempotent: if
// a delivery_complete event for the same slug already exists in the last
// 24h, this is a no-op. Errors are swallowed (best-effort emission must
// not fail the completion path).
//
// Called from runComplete and autoArchiveIfCompleted — both paths that
// finalize a spec's completed status. Without this, the dashboard's
// "specs shipped this week" tile chronically read near-zero because no
// emitter wrote to the event log on the human-driven completion path.
func emitCompletionEvents(heroDir, slug, title string) {
	if heroDir == "" || slug == "" {
		return
	}
	logPath := filepath.Join(heroDir, "events.log")

	if hasRecentDeliveryComplete(logPath, slug, 24*time.Hour) {
		return
	}

	agent := os.Getenv("HERO_AGENT")
	if agent == "" {
		agent = "human/" + gitUserName()
	}

	message := title
	if message == "" {
		message = fmt.Sprintf("Completed %s", slug)
	}

	completeEvt := feed.FeedEvent{
		Type:    "delivery_complete",
		Agent:   agent,
		Slug:    slug,
		Message: message,
	}
	_ = feed.AppendEvent(logPath, completeEvt)

	statusEvt := feed.FeedEvent{
		Type:    "spec.status_changed",
		Agent:   agent,
		Slug:    slug,
		Message: "status -> completed",
	}
	_ = feed.AppendEvent(logPath, statusEvt)
}

// hasRecentDeliveryComplete reports whether the event log already
// contains a delivery_complete event for slug within the trailing
// window. Used to dedup emission across re-runs of hero spec complete.
func hasRecentDeliveryComplete(logPath, slug string, window time.Duration) bool {
	events, err := feed.ReadEvents(logPath, feed.Filter{
		Since: time.Now().Add(-window),
		Type:  "delivery_complete",
		Slug:  slug,
	})
	if err != nil {
		return false
	}
	return len(events) > 0
}

// autoArchiveIfCompleted is the auto-archive hook used by deliver
// success paths. When the spec at specPath has status: completed,
// it ensures the file lives under specs/<slug>/ and re-indexes.
// Idempotent — safe to call when status isn't completed or when
// the spec is already under specs/. Returns whether a move occurred.
//
// The caller is the model-driven /deliver flow finishing up — we
// don't want a manual `hero spec complete` step. moveToSpecs handles
// the file move; we only invoke it when status is actually completed
// so a not-yet-finished spec isn't archived prematurely.
func autoArchiveIfCompleted(specPath, heroDir string) (bool, error) {
	return autoArchiveIfCompletedOpt(specPath, heroDir, false)
}

func autoArchiveIfCompletedOpt(specPath, heroDir string, skipGateCheck bool) (bool, error) {
	s, err := spec.ParseFile(specPath)
	if err != nil {
		return false, fmt.Errorf("parsing spec: %w", err)
	}
	if s.Status != spec.StatusCompleted {
		return false, nil
	}

	// Gate check: for work specs not yet archived, verify the delivery
	// gates were satisfied before archiving. This catches the case where
	// an agent edits status: completed in the frontmatter directly
	// without running `hero spec verify`.
	if s.IsWorkSpec() && !isAlreadyInSpecsDir(specPath, heroDir) && !skipGateCheck {
		ledger := spec.ParseLedger(s)
		audit := spec.FindAuditReport(s)
		if !ledger.Found || !audit.Found || audit.Verdict != "SHIP" {
			var missing []string
			if !ledger.Found {
				missing = append(missing, "no Completion Ledger")
			}
			if !audit.Found {
				missing = append(missing, "no audit report")
			} else if audit.Verdict != "SHIP" {
				missing = append(missing, "audit verdict is "+audit.Verdict)
			}
			fmt.Fprintf(os.Stderr,
				"Warning: spec %s has status: completed but delivery gates are not satisfied (%s).\n"+
					"Run `hero spec verify %s` to complete properly, or `hero spec verify %s --force` to override.\n"+
					"Refusing to auto-archive without gates.\n",
				s.Slug, strings.Join(missing, ", "), s.Slug, s.Slug)
			return false, nil
		}
	}

	// Safety net for the model-driven /deliver path: the agent rewrites
	// frontmatter directly with its file edit tool, then runs `hero spec
	// verify` — which lands here. If the agent (or a human hand-edit)
	// didn't add `completed_at:`, stamp it now so no completed spec ever
	// ends up under .hero/specs/ without the field. Best-effort: a write
	// failure logs to stderr and continues, matching the rest of this
	// function's error stance.
	stampCompletedAtFile(specPath)
	if isAlreadyInSpecsDir(specPath, heroDir) {
		// Emit anyway when status is completed — covers the case where
		// the spec was already under specs/ but the event was missed on
		// a prior run. Idempotent inside emitCompletionEvents.
		emitCompletionEvents(heroDir, s.Slug, s.Title)
		return false, nil
	}
	_, moved, err := moveToSpecs(specPath, heroDir)
	if err != nil {
		return false, fmt.Errorf("moving spec: %w", err)
	}
	if moved {
		if _, err := index.Rebuild(heroDir); err != nil {
			return true, fmt.Errorf("rebuilding index: %w", err)
		}
	}
	emitCompletionEvents(heroDir, s.Slug, s.Title)
	return moved, nil
}

// stampCompletedAtFile re-reads the spec file at specPath and, when
// `completed_at:` is missing, writes back the same content with the
// canonical stamp added. Best-effort: any error is logged to stderr
// and swallowed, matching autoArchiveIfCompleted's non-fatal stance
// on auxiliary work.
func stampCompletedAtFile(specPath string) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read spec for completed_at stamp: %v\n", err)
		return
	}
	original := string(data)
	stamped := spec.StampCompletedAt(original)
	if stamped == original {
		return
	}
	if err := os.WriteFile(specPath, []byte(stamped), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not stamp completed_at: %v\n", err)
	}
}

// updateFrontmatterStatus rewrites the spec file with an updated status field.
// When the new status is `completed`, the canonical `completed_at:` stamp
// is also written on the same buffer so the field lands in the same write
// as the status change (no second pass needed). The stamp is idempotent —
// rerunning on an already-stamped spec preserves the original value.
func updateFrontmatterStatus(path, newStatus string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	updated := spec.SetFrontmatterField(content, "status", newStatus)
	if newStatus == "completed" {
		updated = spec.StampCompletedAt(updated)
	}
	return os.WriteFile(path, []byte(updated), 0o644)
}

// moveToSpecs moves a spec from planning/ to specs/.
// Returns the new path and whether a move occurred.
func moveToSpecs(specPath, heroDir string) (string, bool, error) {
	absPath, err := filepath.Abs(specPath)
	if err != nil {
		return specPath, false, err
	}
	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return specPath, false, err
	}

	absHeroDir, err := filepath.Abs(heroDir)
	if err != nil {
		return specPath, false, err
	}
	absHeroDir, err = filepath.EvalSymlinks(absHeroDir)
	if err != nil {
		return specPath, false, err
	}

	planningDir := filepath.Join(absHeroDir, "planning")
	if !strings.HasPrefix(absPath, planningDir+string(filepath.Separator)) {
		// Not under planning/ — nothing to move
		return specPath, false, nil
	}

	// A flat `<slug>.md` spec (e.g. an initiative child stored as a sibling
	// of the initiative's spec.md) does NOT own its directory: that directory
	// holds the initiative and the other children. Archive only the single
	// file, deriving the slug from the spec itself rather than the parent dir
	// name, and skip the sibling/parent cleanup that assumes sole ownership.
	flat := filepath.Base(absPath) != "spec.md"

	// Determine the slug directory
	// e.g. .hero/planning/features/csv-export/spec.md → slug dir is csv-export
	specDir := filepath.Dir(absPath)
	slugDir := filepath.Base(specDir)
	if flat {
		s, err := spec.ParseFile(absPath)
		if err != nil {
			return specPath, false, fmt.Errorf("parsing spec: %w", err)
		}
		slugDir = s.Slug
	}

	// Destination: .hero/specs/<slug>/spec.md
	destDir := filepath.Join(absHeroDir, "specs", slugDir)
	destPath := filepath.Join(destDir, "spec.md")

	// Check for collision
	if _, err := os.Stat(destPath); err == nil {
		return specPath, false, fmt.Errorf("destination already exists: %s", destPath)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return specPath, false, fmt.Errorf("creating destination: %w", err)
	}

	// Try git mv first (preserves history), fall back to os.Rename.
	gitRoot := ""
	if isGitRepo(absHeroDir) {
		gitRoot = filepath.Dir(absHeroDir)
	}
	moved := false
	if gitRoot != "" {
		gitCmd := exec.Command("git", "mv", absPath, destPath)
		gitCmd.Dir = gitRoot
		if err := gitCmd.Run(); err == nil {
			moved = true
		}
		// git mv failed — fall through to os.Rename
	}
	if !moved {
		if err := os.Rename(absPath, destPath); err != nil {
			return specPath, false, fmt.Errorf("moving file: %w", err)
		}
	}

	// A flat-file spec shares its directory with siblings (e.g. an
	// initiative and its other children), so neither the sibling sweep nor
	// the empty-parent cleanup applies — moving only the file is the whole
	// job.
	if !flat {
		// Move any sibling artifacts (delivery-audit.md, plan.md, mocks, …)
		// so the completed spec keeps its delivery record instead of
		// orphaning them in planning/.
		moveSiblingArtifacts(specDir, destDir, gitRoot)

		// Clean up empty source directory
		removeEmptyParents(specDir, planningDir)
	}

	return destPath, true, nil
}

// moveSiblingArtifacts relocates everything left in a spec's source
// directory (after spec.md has moved) into the archived destination, so
// delivery records like delivery-audit.md travel with the spec instead
// of being orphaned in planning/. Best-effort: clobbers nothing and
// swallows per-entry errors, matching the archive path's non-fatal stance.
func moveSiblingArtifacts(srcDir, destDir, gitRoot string) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(destDir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue // never clobber an existing destination
		}
		if gitRoot != "" {
			cmd := exec.Command("git", "mv", src, dst)
			cmd.Dir = gitRoot
			if cmd.Run() == nil {
				continue
			}
		}
		_ = os.Rename(src, dst)
	}
}

// isAlreadyInSpecsDir reports whether the given spec path lives under
// `<heroDir>/specs/`, the destination for completed specs. Used by
// runComplete to detect when no move work remains. Symlink-tolerant
// via filepath.EvalSymlinks; falls back to a direct prefix check when
// resolution fails (best-effort answer rather than a hard error).
func isAlreadyInSpecsDir(specPath, heroDir string) bool {
	abs, err := filepath.Abs(specPath)
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	absHero, err := filepath.Abs(heroDir)
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(absHero); err == nil {
		absHero = resolved
	}
	specsDir := filepath.Join(absHero, "specs") + string(filepath.Separator)
	return strings.HasPrefix(abs, specsDir)
}

// isGitRepo checks if the directory is inside a git repository.
func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}

// removeEmptyParents removes empty directories going up from dir to stopAt.
func removeEmptyParents(dir, stopAt string) {
	for dir != stopAt && dir != "." && dir != "/" {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}
