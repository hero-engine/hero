package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
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

	// Each step below is independently idempotent. The previous version
	// gated everything on `status == completed`, which stranded specs in
	// planning/ whenever something else flipped the status first
	// (`/deliver`, an agent edit, `hero check --reconcile`). Now we run
	// what's needed and skip what isn't, only erroring when the spec
	// has fully completed paperwork.
	alreadyCompleted := s.Status == spec.StatusCompleted
	alreadyMoved := isAlreadyInSpecsDir(specPath, heroDir)
	if alreadyCompleted && alreadyMoved {
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

	fmt.Printf("Completed spec: %s\n", s.Slug)
	return nil
}

// updateFrontmatterStatus rewrites the spec file with an updated status field.
func updateFrontmatterStatus(path, newStatus string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	updated := spec.SetFrontmatterField(content, "status", newStatus)
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

	// Determine the slug directory
	// e.g. .hero/planning/features/csv-export/spec.md → slug dir is csv-export
	specDir := filepath.Dir(absPath)
	slugDir := filepath.Base(specDir)

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

	// Try git mv first (preserves history), fall back to os.Rename
	if isGitRepo(absHeroDir) {
		gitCmd := exec.Command("git", "mv", absPath, destPath)
		gitCmd.Dir = filepath.Dir(absHeroDir)
		if err := gitCmd.Run(); err == nil {
			// Also try to remove the now-empty source directory
			removeEmptyParents(specDir, planningDir)
			return destPath, true, nil
		}
		// git mv failed — fall through to os.Rename
	}

	// Plain filesystem move
	if err := os.Rename(absPath, destPath); err != nil {
		return specPath, false, fmt.Errorf("moving file: %w", err)
	}

	// Clean up empty source directory
	removeEmptyParents(specDir, planningDir)

	return destPath, true, nil
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
