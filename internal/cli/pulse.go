package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/drift"
	"github.com/hero-engine/hero/internal/pulse"
	"github.com/hero-engine/hero/internal/refs"
	"github.com/spf13/cobra"
)

var pulseCmd = &cobra.Command{
	Use:   "status",
	Short: "Sprint status — done, in-flight, at-risk",
	Long:  `Generates a sprint narrative summary showing done, in-flight, at-risk specs and knowledge updates.`,
	RunE:  runPulse,
}

var (
	pulseWeek  bool
	pulseSince string
	pulseSpec  string
	pulseJSON  bool
	pulseMD    bool
	pulseRefs  bool
)

func init() {
	pulseCmd.Flags().BoolVar(&pulseWeek, "week", false, "rolling 7-day window")
	pulseCmd.Flags().StringVar(&pulseSince, "since", "", "summary since this date (YYYY-MM-DD)")
	pulseCmd.Flags().StringVar(&pulseSpec, "spec", "", "focus on a single spec slug")
	pulseCmd.Flags().BoolVar(&pulseJSON, "json", false, "output as JSON")
	pulseCmd.Flags().BoolVar(&pulseMD, "md", false, "output as Markdown")
	pulseCmd.Flags().BoolVar(&pulseRefs, "refs", false, "show two-tier MCP ref-store metrics across sessions")
}

func runPulse(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if pulseRefs {
		return runPulseRefs(heroDir)
	}

	// Get pulse config with defaults
	var staleDeliveringDays, stalePlanningDays, sprintDays int
	if cfg.Pulse != nil {
		staleDeliveringDays = cfg.Pulse.StaleDeliveringDays
		stalePlanningDays = cfg.Pulse.StalePlanningDays
		sprintDays = cfg.Pulse.SprintDays
	}
	// Apply defaults if unset
	if staleDeliveringDays == 0 {
		staleDeliveringDays = 3
	}
	if stalePlanningDays == 0 {
		stalePlanningDays = 7
	}
	if sprintDays == 0 {
		sprintDays = 14
	}

	period := pulse.CalcPeriod(pulseSince, pulseWeek, sprintDays)

	data, err := pulse.BuildPulse(heroDir, period, staleDeliveringDays, stalePlanningDays)
	if err != nil {
		return fmt.Errorf("building pulse: %w", err)
	}

	// Populate drift warnings for in-flight specs
	driftSummaries := drift.DriftSummaries(heroDir, projectRoot)
	var driftEntries []pulse.DriftEntry
	for _, ds := range driftSummaries {
		driftEntries = append(driftEntries, pulse.DriftEntry{
			Slug:         ds.Slug,
			Title:        ds.Title,
			Warnings:     ds.Warnings,
			HasViolation: ds.HasViolation,
		})
	}
	pulse.PopulateDrift(data, driftEntries)

	// Filter by spec slug if requested
	if pulseSpec != "" {
		data = pulse.FilterBySlug(data, pulseSpec)
	}

	// Render
	switch {
	case pulseJSON:
		out, err := pulse.RenderJSON(data)
		if err != nil {
			return fmt.Errorf("rendering JSON: %w", err)
		}
		fmt.Println(out)
	case pulseMD:
		fmt.Print(pulse.RenderMarkdown(data))
	default:
		fmt.Print(pulse.RenderText(data))
	}

	return nil
}

// runPulseRefs renders two-tier MCP response ref-store metrics across
// every session present under .hero/sessions/. One row per session
// plus a totals row. No external state — purely reads refs.db files.
func runPulseRefs(heroDir string) error {
	sessionsDir := filepath.Join(heroDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No two-tier MCP sessions yet.")
			return nil
		}
		return fmt.Errorf("reading sessions dir: %w", err)
	}

	type sessionRow struct {
		ID  string
		M   refs.Metrics
	}
	var rows []sessionRow
	totals := refs.Metrics{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dbPath := filepath.Join(sessionsDir, e.Name(), "refs.db")
		if _, err := os.Stat(dbPath); err != nil {
			continue
		}
		store, err := refs.Open(heroDir, e.Name())
		if err != nil {
			continue
		}
		m, err := store.PersistedMetrics()
		store.Close()
		if err != nil {
			continue
		}
		rows = append(rows, sessionRow{ID: e.Name(), M: m})
		totals.Registers += m.Registers
		totals.Hits += m.Hits
		totals.Misses += m.Misses
		totals.Refetch += m.Refetch
	}

	if len(rows) == 0 {
		fmt.Println("No two-tier MCP sessions with persisted metrics yet.")
		return nil
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })

	fmt.Println("Two-tier MCP ref-store metrics")
	fmt.Println()
	fmt.Printf("%-16s  %10s  %8s  %8s  %8s\n", "session", "registered", "hits", "misses", "refetch")
	fmt.Printf("%-16s  %10s  %8s  %8s  %8s\n", "-------", "----------", "----", "------", "-------")
	for _, r := range rows {
		id := r.ID
		if len(id) > 16 {
			id = id[:14] + ".."
		}
		fmt.Printf("%-16s  %10d  %8d  %8d  %8d\n", id, r.M.Registers, r.M.Hits, r.M.Misses, r.M.Refetch)
	}
	fmt.Printf("%-16s  %10s  %8s  %8s  %8s\n", "-------", "----------", "----", "------", "-------")
	fmt.Printf("%-16s  %10d  %8d  %8d  %8d\n", "TOTAL", totals.Registers, totals.Hits, totals.Misses, totals.Refetch)

	if totals.Registers > 0 {
		expandRate := float64(totals.Hits) / float64(totals.Registers) * 100
		fmt.Println()
		fmt.Printf("Expand rate: %.1f%% of registered refs were resolved\n", expandRate)
		if totals.Hits > 0 {
			refetchRate := float64(totals.Refetch) / float64(totals.Hits) * 100
			fmt.Printf("Refetch rate: %.1f%% of hits triggered re-fetch\n", refetchRate)
		}
	}

	return nil
}
