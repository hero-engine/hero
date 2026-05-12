package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show a rich terminal summary of the workspace",
	Long: `Displays a compact overview of the hero workspace including:
  - Spec counts by status
  - Stale specs warning
  - Claimed work assignments
  - Tracker integration status`,
	RunE: runDashboard,
}

func runDashboard(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	staleDays := 14
	if cfg.Team != nil && cfg.Team.StaleDays > 0 {
		staleDays = cfg.Team.StaleDays
	}

	// Categorize specs
	counts := map[string]int{}
	var stale []*spec.Spec
	var claimed []*spec.Spec
	var linked int

	for _, s := range specs {
		key := string(s.Status)
		if s.IsKnowledge() {
			key = string(s.Type)
		}
		counts[key]++

		if s.IsWorkSpec() && s.IsInFlight() {
			age := time.Since(s.ModifiedAt)
			if age > time.Duration(staleDays)*24*time.Hour {
				stale = append(stale, s)
			}
		}

		if s.ClaimedBy != "" {
			claimed = append(claimed, s)
		}

		if s.TrackerID != "" {
			linked++
		}
	}

	// Header
	fmt.Println("Hero Dashboard")
	fmt.Println(strings.Repeat("─", 40))

	// Status breakdown
	fmt.Println()
	fmt.Println("Specs by Status:")

	statusOrder := []struct {
		label string
		key   string
	}{
		{"  Planning", "planning"},
		{"  In Review", "in-review"},
		{"  Delivering", "delivering"},
		{"  Completed", "completed"},
		{"  Conventions", "convention"},
		{"  Decisions", "decision"},
		{"  Rules", "rule"},
		{"  External", "external"},
		{"  Context", "context"},
		{"  Notes", "note"},
	}

	for _, s := range statusOrder {
		c := counts[s.key]
		if c > 0 {
			fmt.Printf("  %-14s %d\n", s.label, c)
		}
	}

	total := len(specs)
	inFlight := counts["planning"] + counts["in-review"] + counts["delivering"]
	fmt.Printf("\n  Total: %d specs (%d in-flight)\n", total, inFlight)

	// Stale warnings
	if len(stale) > 0 {
		fmt.Printf("\nStale (%d+ days without update):\n", staleDays)
		for _, s := range stale {
			age := time.Since(s.ModifiedAt)
			fmt.Printf("  %-30s %s  (%s)\n", s.Slug, string(s.Status), formatAge(age.Truncate(time.Hour)))
		}
	}

	// Claimed work
	if len(claimed) > 0 {
		fmt.Println("\nAssignments:")
		for _, s := range claimed {
			fmt.Printf("  %-30s → %s\n", s.Slug, s.ClaimedBy)
		}
	}

	// Tracker status
	fmt.Println()
	if cfg.Tracker != nil && cfg.Tracker.Type != "none" && cfg.Tracker.Type != "" {
		fmt.Printf("Tracker: %s (%s)\n", cfg.Tracker.Type, cfg.Tracker.Project)
		fmt.Printf("  Linked: %d/%d specs\n", linked, total)
	} else {
		fmt.Println("Tracker: not configured")
	}

	return nil
}
