package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var replayCmd = &cobra.Command{
	Use:   "retro <spec-slug>",
	Short: "Sprint retro — compare spec plan vs actual outcome for a delivered spec",
	Long: `Analyzes a completed (or in-progress) spec and compares what was planned
vs what actually happened. Provides metrics on:
  - Files planned vs files actually changed
  - Time from planning to completion
  - Scope drift (unplanned files touched)
  - Coverage (planned files not yet touched)

Use this after delivery to understand how well the spec predicted the work.
Combine with /retro for a full post-delivery retrospective.`,
	Args: cobra.ExactArgs(1),
	RunE: runReplay,
}

var replayBase string

func init() {
	replayCmd.Flags().StringVar(&replayBase, "base", "HEAD", "git base ref to diff against")
}

func runReplay(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	slug := args[0]

	// Find the spec by slug
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == slug {
			target = s
			break
		}
	}

	if target == nil {
		return fmt.Errorf("spec %q not found", slug)
	}

	// Get git changes
	gitFiles, err := gitDiffFiles(projectRoot, replayBase)
	if err != nil {
		// Non-fatal — may not be in a git repo or base ref may not exist
		gitFiles = nil
	}

	// Build report
	fmt.Printf("Replay: %s\n", target.Slug)
	fmt.Printf("Title:  %s\n", target.Title)
	fmt.Printf("Type:   %s\n", target.Type)
	fmt.Printf("Status: %s\n", target.Status)
	fmt.Println(strings.Repeat("─", 50))

	// Timeline
	fmt.Println("\nTimeline:")
	if !target.CreatedAt.IsZero() {
		fmt.Printf("  Created:  %s\n", target.CreatedAt.Format("2006-01-02"))
	}
	if !target.ModifiedAt.IsZero() {
		fmt.Printf("  Modified: %s\n", target.ModifiedAt.Format("2006-01-02"))
	}
	if !target.CreatedAt.IsZero() && !target.ModifiedAt.IsZero() {
		duration := target.ModifiedAt.Sub(target.CreatedAt)
		if duration > 0 {
			fmt.Printf("  Duration: %s\n", formatDuration(duration))
		}
	}
	if target.Status == spec.StatusCompleted && !target.CreatedAt.IsZero() {
		elapsed := time.Since(target.CreatedAt)
		fmt.Printf("  Total elapsed: %s\n", formatDuration(elapsed))
	}

	// Ownership
	if target.ClaimedBy != "" {
		fmt.Printf("\nOwner: %s\n", target.ClaimedBy)
	}
	if target.TrackerID != "" {
		fmt.Printf("Tracker: %s\n", target.TrackerID)
	}

	// File analysis
	plannedFiles := target.FilesTouched
	if len(plannedFiles) == 0 && len(gitFiles) == 0 {
		fmt.Println("\nNo file tracking data available.")
		fmt.Println("Add a ## Changes section to your spec to enable file-level replay.")
		return nil
	}

	specFiles := make(map[string]bool)
	for _, f := range plannedFiles {
		specFiles[f] = true
	}
	gitFileSet := make(map[string]bool)
	for _, f := range gitFiles {
		gitFileSet[f] = true
	}

	var matched, specOnly, gitOnly []string
	for _, f := range plannedFiles {
		if gitFileSet[f] {
			matched = append(matched, f)
		} else {
			specOnly = append(specOnly, f)
		}
	}
	for _, f := range gitFiles {
		if !specFiles[f] {
			gitOnly = append(gitOnly, f)
		}
	}
	sort.Strings(matched)
	sort.Strings(specOnly)
	sort.Strings(gitOnly)

	fmt.Println("\nFile Analysis:")
	fmt.Printf("  Planned:    %d files\n", len(plannedFiles))
	fmt.Printf("  Changed:    %d files (git)\n", len(gitFiles))
	fmt.Printf("  Matched:    %d files\n", len(matched))
	fmt.Printf("  Unplanned:  %d files (scope drift)\n", len(gitOnly))
	fmt.Printf("  Not touched: %d files (incomplete or dropped)\n", len(specOnly))

	// Accuracy metrics
	if len(plannedFiles) > 0 {
		accuracy := (float64(len(matched)) / float64(len(plannedFiles))) * 100
		fmt.Printf("\n  Accuracy: %.0f%% of planned files were touched\n", accuracy)
	}
	if len(gitFiles) > 0 {
		drift := (float64(len(gitOnly)) / float64(len(gitFiles))) * 100
		fmt.Printf("  Drift:    %.0f%% of changes were unplanned\n", drift)
	}

	// Detail lists
	if len(matched) > 0 {
		fmt.Printf("\nMatched (%d):\n", len(matched))
		for _, f := range matched {
			fmt.Printf("  + %s\n", f)
		}
	}
	if len(specOnly) > 0 {
		fmt.Printf("\nPlanned but not touched (%d):\n", len(specOnly))
		for _, f := range specOnly {
			fmt.Printf("  - %s\n", f)
		}
	}
	if len(gitOnly) > 0 {
		fmt.Printf("\nUnplanned changes (%d):\n", len(gitOnly))
		for _, f := range gitOnly {
			fmt.Printf("  ? %s\n", f)
		}
	}

	// Sections check
	fmt.Println("\nSpec Completeness:")
	sections := []struct{ name, key string }{
		{"Goal", "goal"},
		{"Design", "design"},
		{"Changes", "changes"},
		{"Acceptance Criteria", "acceptance criteria"},
		{"Root Cause", "root cause"},
		{"Fix", "fix"},
	}
	for _, sec := range sections {
		content, exists := target.Sections[sec.key]
		if exists && strings.TrimSpace(content) != "" && !strings.Contains(content, "<!--") {
			fmt.Printf("  [x] %s\n", sec.name)
		} else if exists {
			fmt.Printf("  [ ] %s (template only)\n", sec.name)
		}
	}

	return nil
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days == 0 {
		hours := int(d.Hours())
		if hours == 0 {
			return "less than an hour"
		}
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if days == 1 {
		return "1 day"
	}
	if days < 7 {
		return fmt.Sprintf("%d days", days)
	}
	weeks := days / 7
	remaining := days % 7
	if remaining == 0 {
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	}
	if weeks == 1 {
		return fmt.Sprintf("1 week %d days", remaining)
	}
	return fmt.Sprintf("%d weeks %d days", weeks, remaining)
}
