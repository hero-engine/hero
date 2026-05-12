package cli

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/cost"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "estimate [spec-slug]",
	Short: "Estimate effort for a spec based on complexity signals",
	Long: `Analyzes a spec's structural complexity to produce a relative effort estimate.

Signals used:
  - Files listed in Changes section
  - Number of spec sections filled in
  - Dependency count (relations)
  - Body size (word count)
  - Spec type (bug vs feature vs initiative)

If a slug is provided, estimates that single spec. Without a slug, estimates
all in-flight specs and ranks them by effort.

Estimates are calibrated against completed specs in the same project when
historical data is available.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCost,
}

var (
	costAll     bool
	costHistory bool
)

var costCalibrateCmd = &cobra.Command{
	Use:   "calibrate",
	Short: "Rebuild calibration data from completed specs + git history",
	RunE:  runCostCalibrate,
}

func init() {
	costCmd.Flags().BoolVar(&costAll, "all", false, "estimate all specs, not just in-flight")
	costCmd.Flags().BoolVar(&costHistory, "history", false, "show calibration summary")
	costCmd.AddCommand(costCalibrateCmd)
}

// effort buckets
const (
	effortTrivial = "trivial" // < 2 points
	effortSmall   = "small"   // 2-4 points
	effortMedium  = "medium"  // 5-9 points
	effortLarge   = "large"   // 10-19 points
	effortXLarge  = "x-large" // 20+ points
)

// costEstimate holds the result for a single spec
type costEstimate struct {
	Slug         string
	Title        string
	Type         spec.Type
	Status       spec.Status
	Points       float64
	Bucket       string
	FileCount    int
	SectionCount int
	DependsCount int
	WordCount    int
}

func runCostCalibrate(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)

	estimator := func(s *spec.Spec) float64 {
		allSpecs, _ := spec.Discover(heroDir)
		cal := calibrate(allSpecs)
		est := estimateSpec(s, cal)
		return est.Points
	}

	cal, err := cost.BuildCalibration(heroDir, projectRoot, estimator)
	if err != nil {
		return err
	}

	knowledgeDir := cfg.KnowledgeDir(projectRoot)
	if err := cost.SaveCalibration(knowledgeDir, cal); err != nil {
		return err
	}

	fmt.Printf("Calibrated from %d completed specs.\n", cal.SpecCount)
	fmt.Print(cost.FormatHistory(cal))
	return nil
}

func runCost(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Handle --history flag
	if costHistory {
		knowledgeDir := cfg.KnowledgeDir(projectRoot)
		cal, err := cost.LoadCalibration(knowledgeDir)
		if err != nil {
			fmt.Println("No calibration data. Run 'hero cost calibrate' first.")
			return nil
		}
		fmt.Print(cost.FormatHistory(cal))
		return nil
	}

	allSpecs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	// Compute calibration from completed specs
	calibration := calibrate(allSpecs)

	if len(args) == 1 {
		// Single spec mode
		slug := args[0]
		var target *spec.Spec
		for _, s := range allSpecs {
			if s.Slug == slug {
				target = s
				break
			}
		}
		if target == nil {
			return fmt.Errorf("spec %q not found", slug)
		}
		est := estimateSpec(target, calibration)
		printSingleEstimate(est, calibration)
		return nil
	}

	// Multi-spec mode
	var targets []*spec.Spec
	for _, s := range allSpecs {
		if costAll || s.IsInFlight() {
			targets = append(targets, s)
		}
	}

	if len(targets) == 0 {
		if costAll {
			fmt.Println("No specs found.")
		} else {
			fmt.Println("No in-flight specs. Use --all to estimate all specs.")
		}
		return nil
	}

	var estimates []costEstimate
	for _, s := range targets {
		estimates = append(estimates, estimateSpec(s, calibration))
	}

	// Sort by points descending
	sort.Slice(estimates, func(i, j int) bool {
		return estimates[i].Points > estimates[j].Points
	})

	printEstimateTable(estimates, calibration)
	return nil
}

// calibrationData holds historical context from completed specs
type calibrationData struct {
	CompletedCount int
	AvgFiles       float64
	AvgWords       float64
	AvgSections    float64
	HasHistory     bool
}

func calibrate(specs []*spec.Spec) calibrationData {
	var cal calibrationData
	var totalFiles, totalWords, totalSections int

	for _, s := range specs {
		if s.Status != spec.StatusCompleted || !s.IsWorkSpec() {
			continue
		}
		cal.CompletedCount++
		totalFiles += len(s.FilesTouched)
		totalSections += len(s.Sections)

		// Count words in all sections
		wc := 0
		for _, content := range s.Sections {
			wc += countWords(content)
		}
		totalWords += wc
	}

	if cal.CompletedCount > 0 {
		cal.HasHistory = true
		cal.AvgFiles = float64(totalFiles) / float64(cal.CompletedCount)
		cal.AvgWords = float64(totalWords) / float64(cal.CompletedCount)
		cal.AvgSections = float64(totalSections) / float64(cal.CompletedCount)
	}

	return cal
}

func estimateSpec(s *spec.Spec, cal calibrationData) costEstimate {
	est := costEstimate{
		Slug:         s.Slug,
		Title:        s.Title,
		Type:         s.Type,
		Status:       s.Status,
		FileCount:    len(s.FilesTouched),
		SectionCount: len(s.Sections),
		DependsCount: countDependencies(s),
	}

	// Count total words across sections
	for _, content := range s.Sections {
		est.WordCount += countWords(content)
	}

	// Base points from files (strongest signal)
	filePoints := float64(est.FileCount) * 1.5

	// Section complexity: more filled sections = more thorough spec = bigger scope
	sectionPoints := float64(est.SectionCount) * 0.5

	// Dependencies add coordination overhead
	depPoints := float64(est.DependsCount) * 2.0

	// Word count as a proxy for scope complexity
	wordPoints := 0.0
	if est.WordCount > 500 {
		wordPoints = 2.0
	} else if est.WordCount > 200 {
		wordPoints = 1.0
	} else if est.WordCount > 50 {
		wordPoints = 0.5
	}

	// Type multiplier
	typeMultiplier := 1.0
	switch s.Type {
	case spec.TypeBug:
		typeMultiplier = 0.7 // bugs tend to be smaller
	case spec.TypeInitiative:
		typeMultiplier = 2.0 // initiatives are umbrella items
	}

	raw := (filePoints + sectionPoints + depPoints + wordPoints) * typeMultiplier

	// Minimum: if spec has any content at all, at least 1 point
	if raw < 1.0 && (est.FileCount > 0 || est.WordCount > 10) {
		raw = 1.0
	}

	// Apply calibration: if we have history, scale relative to project average
	if cal.HasHistory && cal.AvgFiles > 0 && est.FileCount > 0 {
		relativeSize := float64(est.FileCount) / cal.AvgFiles
		// Blend raw heuristic with calibrated signal (60% raw, 40% calibrated)
		calibratedPoints := relativeSize * 5.0 // 5 points = project average
		raw = raw*0.6 + calibratedPoints*0.4
	}

	est.Points = math.Round(raw*10) / 10
	est.Bucket = bucketFromPoints(est.Points)

	return est
}

func bucketFromPoints(points float64) string {
	switch {
	case points < 2:
		return effortTrivial
	case points < 5:
		return effortSmall
	case points < 10:
		return effortMedium
	case points < 20:
		return effortLarge
	default:
		return effortXLarge
	}
}

func countDependencies(s *spec.Spec) int {
	count := 0
	for _, r := range s.Relations {
		if r.Kind == "depends-on" || r.Kind == "parent" {
			count++
		}
	}
	return count
}

func countWords(s string) int {
	return len(strings.Fields(s))
}

func printSingleEstimate(est costEstimate, cal calibrationData) {
	fmt.Printf("Spec:   %s\n", est.Slug)
	fmt.Printf("Title:  %s\n", est.Title)
	fmt.Printf("Type:   %s\n", est.Type)
	fmt.Printf("Status: %s\n", est.Status)
	fmt.Println(strings.Repeat("─", 50))

	fmt.Printf("\nEffort: %s (%.1f points)\n", est.Bucket, est.Points)

	fmt.Println("\nSignals:")
	fmt.Printf("  Files in Changes:  %d\n", est.FileCount)
	fmt.Printf("  Sections filled:   %d\n", est.SectionCount)
	fmt.Printf("  Dependencies:      %d\n", est.DependsCount)
	fmt.Printf("  Word count:        %d\n", est.WordCount)

	if cal.HasHistory {
		fmt.Printf("\nCalibration (from %d completed specs):\n", cal.CompletedCount)
		fmt.Printf("  Avg files/spec:    %.1f\n", cal.AvgFiles)
		fmt.Printf("  Avg words/spec:    %.0f\n", cal.AvgWords)
	}
}

func printEstimateTable(estimates []costEstimate, cal calibrationData) {
	if cal.HasHistory {
		fmt.Printf("Calibrated against %d completed specs\n", cal.CompletedCount)
	}
	fmt.Println()

	// Header
	fmt.Printf("%-8s  %-6s  %-30s  %s\n", "EFFORT", "POINTS", "SPEC", "FILES")
	fmt.Println(strings.Repeat("─", 65))

	totalPoints := 0.0
	for _, est := range estimates {
		title := est.Title
		if len(title) > 28 {
			title = title[:25] + "..."
		}
		label := fmt.Sprintf("%s (%s)", est.Slug, title)
		if len(label) > 30 {
			label = label[:27] + "..."
		}
		fmt.Printf("%-8s  %5.1f   %-30s  %d\n", est.Bucket, est.Points, label, est.FileCount)
		totalPoints += est.Points
	}

	fmt.Println(strings.Repeat("─", 65))
	fmt.Printf("%-8s  %5.1f   %d specs\n", "TOTAL", totalPoints, len(estimates))
}
