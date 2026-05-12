package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/watch"
	"github.com/spf13/cobra"
)

var (
	watchMode     string
	watchInterval int
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor workspace for changes and auto-reindex",
	Long: `Watches the hero workspace for file changes and automatically reindexes
specs, runs health checks, and surfaces nudges.

Two modes are available:

  local  — Continuous polling mode for single-user development.
           Watches for file changes, auto-reindexes, and prints change summaries.
           Runs until interrupted (Ctrl+C).

  ci     — One-shot mode for CI/CD pipelines (e.g. GitHub Actions).
           Scans the workspace once, validates all specs, runs health checks,
           and exits with a non-zero code if issues are found.

Examples:
  hero watch                   # local mode, default 2s interval
  hero watch --interval 5      # local mode, 5s interval
  hero watch --mode ci         # one-shot CI mode`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().StringVar(&watchMode, "mode", "local", "watch mode: local or ci")
	watchCmd.Flags().IntVar(&watchInterval, "interval", 2, "polling interval in seconds (local mode only)")
}

func runWatch(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	switch watchMode {
	case "local":
		return runWatchLocal(heroDir, cfg, projectRoot)
	case "ci":
		return runWatchCI(heroDir, cfg, projectRoot)
	default:
		return fmt.Errorf("unknown mode %q (use 'local' or 'ci')", watchMode)
	}
}

func runWatchLocal(heroDir string, cfg config.Config, projectRoot string) error {
	interval := time.Duration(watchInterval) * time.Second
	if interval < time.Second {
		interval = time.Second
	}

	fmt.Printf("Watching %s (every %s) — press Ctrl+C to stop\n", heroDir, interval)

	handler := func(events []watch.Event) {
		fmt.Printf("\n%s\n", watch.Summary(events))

		// Show each change
		for _, e := range events {
			fmt.Printf("  %s\n", e)
		}

		// Reindex changed specs
		specEvents := watch.SpecEvents(events)
		if len(specEvents) > 0 {
			reindexSpecs(heroDir, specEvents)
		}
	}

	w := watch.New(heroDir, interval, handler)
	return w.Run()
}

func runWatchCI(heroDir string, cfg config.Config, projectRoot string) error {
	fmt.Println("Hero CI check")
	fmt.Println("=============")

	// Rebuild the full index
	stats, err := index.Rebuild(heroDir)
	if err != nil {
		return fmt.Errorf("rebuilding index: %w", err)
	}

	fmt.Printf("\nIndexed: %d specs\n", stats.TotalSpecs)
	fmt.Printf("  %d features, %d bugs, %d conventions, %d decisions\n",
		stats.Features, stats.Bugs, stats.Conventions, stats.Decisions)

	issues := 0

	// Check for stale specs
	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	staleDays := 14
	if cfg.Team != nil && cfg.Team.StaleDays > 0 {
		staleDays = cfg.Team.StaleDays
	}

	stale, err := idx.CheckStale(staleDays)
	if err == nil && len(stale) > 0 {
		issues += len(stale)
		fmt.Printf("\nStale specs (>%d days):\n", staleDays)
		for _, s := range stale {
			fmt.Printf("  %-30s  %-10s  %s\n", s.Slug, s.Status, s.Title)
		}
	}

	// Check for unclaimed specs
	unclaimed, err := idx.CheckUnclaimed()
	if err == nil && len(unclaimed) > 0 {
		issues += len(unclaimed)
		fmt.Printf("\nUnclaimed specs:\n")
		for _, s := range unclaimed {
			fmt.Printf("  %-30s  %-10s  %s\n", s.Slug, s.Status, s.Title)
		}
	}

	// Validate all specs (check for parse errors)
	parseErrors := validateAllSpecs(heroDir)
	issues += parseErrors

	fmt.Println()
	if issues == 0 {
		fmt.Println("No issues found.")
	} else {
		fmt.Printf("%d issue(s) found.\n", issues)
		return fmt.Errorf("CI check found %d issue(s)", issues)
	}

	return nil
}

// reindexSpecs reindexes only the changed spec files.
func reindexSpecs(heroDir string, events []watch.Event) {
	idx, err := index.Open(heroDir)
	if err != nil {
		fmt.Printf("  warning: could not open index: %v\n", err)
		return
	}
	defer idx.Close()

	for _, e := range events {
		switch e.Kind {
		case watch.EventCreated, watch.EventModified:
			s, err := spec.ParseFile(e.Path)
			if err != nil {
				fmt.Printf("  warning: parse %s: %v\n", e.Path, err)
				continue
			}
			content, err := os.ReadFile(e.Path)
			if err != nil {
				continue
			}
			if err := idx.IndexSpec(s, string(content)); err != nil {
				fmt.Printf("  warning: index %s: %v\n", s.Slug, err)
				continue
			}
			fmt.Printf("  reindexed: %s\n", s.Slug)

		case watch.EventDeleted:
			// Try to extract slug from path for removal
			s, err := slugFromPath(e.Path)
			if err == nil {
				if err := idx.RemoveSpec(s); err != nil {
					fmt.Printf("  warning: remove %s: %v\n", s, err)
				} else {
					fmt.Printf("  removed from index: %s\n", s)
				}
			}
		}
	}
}

// slugFromPath extracts a slug from a spec.md path.
// e.g., ".hero/specs/my-feature/spec.md" -> "my-feature"
func slugFromPath(path string) (string, error) {
	s, err := spec.ParseFile(path)
	if err != nil {
		// File might be deleted — try to extract from directory name
		dir := filepath.Dir(path)
		slug := filepath.Base(dir)
		if slug == "" || slug == "." || slug == "/" {
			return "", fmt.Errorf("could not determine slug from path %q", path)
		}
		return slug, nil
	}
	return s.Slug, nil
}

// validateAllSpecs walks the hero directory and tries to parse all spec files.
// Returns the number of parse errors found.
func validateAllSpecs(heroDir string) int {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		fmt.Printf("\nValidation error: %v\n", err)
		return 1
	}

	errors := 0
	for _, s := range specs {
		if s.Title == "" {
			fmt.Printf("\nValidation: %s has no title\n", s.Path)
			errors++
		}
	}

	return errors
}
