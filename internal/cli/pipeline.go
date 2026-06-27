package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var (
	pipelineType string
	pipelineJSON bool
	pipelineRun  string
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Show batch pipeline status across import/diagnose/deliver stages",
	Long: `Displays work specs (bugs, features) grouped by pipeline stage:

  Imported    — specs imported from tracker, not yet investigated
  Diagnosed   — bugs with investigation/fix plan, ready for approval
  Approved    — specs ready for delivery (have changes/design section)
  Delivering  — specs currently being implemented
  Completed   — delivered and archived
  Blocked     — specs missing information or with quality issues

Use --type bug or --type feature to filter.
Use --run diagnose to async-diagnose all imported bugs.
Use --run deliver to async-deliver all approved specs.
Use --run all to diagnose then deliver.`,
	RunE: runPipeline,
}

func init() {
	pipelineCmd.Flags().StringVar(&pipelineType, "type", "", "filter by spec type (bug, feature)")
	pipelineCmd.Flags().BoolVar(&pipelineJSON, "json", false, "output as JSON")
	pipelineCmd.Flags().StringVar(&pipelineRun, "run", "", "advance pipeline: diagnose, deliver, or all")
}

// PipelineStage represents a group of specs in a pipeline stage.
type PipelineStage struct {
	Name  string          `json:"name"`
	Count int             `json:"count"`
	Specs []PipelineEntry `json:"specs,omitempty"`
}

// PipelineEntry represents a single spec in the pipeline.
type PipelineEntry struct {
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Type           string `json:"type"`
	TrackerID      string `json:"tracker_id,omitempty"`
	DeliveryMethod string `json:"delivery_method,omitempty"`
	ClaimedBy      string `json:"claimed_by,omitempty"`
}

func runPipeline(cmd *cobra.Command, args []string) error {
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

	// Filter to work specs
	var work []*spec.Spec
	for _, s := range specs {
		if !s.IsWorkSpec() {
			continue
		}
		if pipelineType != "" && string(s.Type) != pipelineType {
			continue
		}
		work = append(work, s)
	}

	// If --run, advance the pipeline
	if pipelineRun != "" {
		return runPipelineAdvance(projectRoot, heroDir, work)
	}

	// Classify into stages
	stages := classifyPipeline(work)

	if pipelineJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(stages)
	}

	printPipeline(stages)
	return nil
}

func runPipelineAdvance(projectRoot, heroDir string, specs []*spec.Spec) error {
	switch pipelineRun {
	case "diagnose":
		// Set flags and delegate to diagnose --batch --async
		diagnoseBatch = true
		diagnoseAsync = true
		return runDiagnose(diagnoseCmd, nil)

	case "deliver":
		// Set flags and delegate to deliver --batch --async
		deliverBatch = true
		deliverAsync = true
		return runDeliver(deliverCmd, nil)

	case "all":
		// First diagnose, then deliver
		fmt.Println("Phase 1: Diagnosing imported bugs...")
		diagnoseBatch = true
		diagnoseAsync = true
		if err := runDiagnose(diagnoseCmd, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warning: diagnose phase: %v\n", err)
		}

		fmt.Println()
		fmt.Println("Phase 2: Delivering approved specs...")
		deliverBatch = true
		deliverAsync = true
		if err := runDeliver(deliverCmd, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warning: deliver phase: %v\n", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown --run mode %q (use: diagnose, deliver, all)", pipelineRun)
	}
}

func classifyPipeline(specs []*spec.Spec) []PipelineStage {
	buckets := map[string][]PipelineEntry{
		"imported":   {},
		"diagnosed":  {},
		"approved":   {},
		"delivering": {},
		"completed":  {},
	}

	for _, s := range specs {
		entry := PipelineEntry{
			Slug:           s.Slug,
			Title:          s.Title,
			Type:           string(s.Type),
			TrackerID:      s.TrackerID,
			DeliveryMethod: s.DeliveryMethod,
			ClaimedBy:      s.ClaimedBy,
		}

		switch s.Status {
		case spec.StatusCompleted:
			buckets["completed"] = append(buckets["completed"], entry)
		case spec.StatusDelivering:
			buckets["delivering"] = append(buckets["delivering"], entry)
		case spec.StatusInReview:
			buckets["delivering"] = append(buckets["delivering"], entry)
		default:
			// draft/planning/active — classify by readiness
			stage := classifyReadiness(s)
			buckets[stage] = append(buckets[stage], entry)
		}
	}

	// Build ordered stages
	order := []string{"imported", "diagnosed", "approved", "delivering", "completed"}
	var stages []PipelineStage
	for _, name := range order {
		entries := buckets[name]
		// Sort by slug for stability
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Slug < entries[j].Slug
		})
		stages = append(stages, PipelineStage{
			Name:  name,
			Count: len(entries),
			Specs: entries,
		})
	}
	return stages
}

// classifyReadiness determines if a draft/planning spec is imported, diagnosed, or approved.
func classifyReadiness(s *spec.Spec) string {
	// Has changes/design section → approved (ready to deliver)
	_, hasChanges := s.Sections["changes"]
	_, hasDesign := s.Sections["design"]
	_, hasApproach := s.Sections["approach"]
	_, hasSolution := s.Sections["solution"]
	_, hasImplementation := s.Sections["implementation"]

	if hasChanges || hasDesign || hasApproach || hasSolution || hasImplementation {
		// Has investigation findings → diagnosed (for bugs)
		_, hasInvestigation := s.Sections["investigation"]
		_, hasRootCause := s.Sections["root cause"]
		_, hasFix := s.Sections["suggested fix approach"]

		if s.Type == spec.TypeBug && (hasInvestigation || hasRootCause || hasFix) {
			return "diagnosed"
		}
		return "approved"
	}

	// Bug with investigation but no fix plan
	if s.Type == spec.TypeBug {
		_, hasInvestigation := s.Sections["investigation"]
		_, hasRootCause := s.Sections["root cause"]
		if hasInvestigation || hasRootCause {
			return "diagnosed"
		}
	}

	return "imported"
}

func printPipeline(stages []PipelineStage) {
	total := 0
	for _, s := range stages {
		total += s.Count
	}

	fmt.Printf("\n  Pipeline Status (%d specs)\n", total)
	fmt.Printf("  %s\n\n", strings.Repeat("═", 55))

	// Visual pipeline
	for _, s := range stages {
		icon := stageIcon(s.Name)
		bar := ""
		if total > 0 {
			width := s.Count * 30 / total
			if s.Count > 0 && width == 0 {
				width = 1
			}
			bar = strings.Repeat("█", width)
		}
		fmt.Printf("  %s %-12s %3d  %s\n", icon, s.Name, s.Count, bar)
	}

	// Detail for non-completed stages
	fmt.Println()
	for _, s := range stages {
		if s.Count == 0 || s.Name == "completed" {
			continue
		}
		fmt.Printf("  %s %s (%d):\n", stageIcon(s.Name), s.Name, s.Count)
		for _, e := range s.Specs {
			tracker := ""
			if e.TrackerID != "" {
				tracker = " [" + e.TrackerID + "]"
			}
			method := ""
			if e.DeliveryMethod == "manual" {
				method = " (manual)"
			}
			claimed := ""
			if e.ClaimedBy != "" {
				claimed = " → " + e.ClaimedBy
			}
			title := e.Title
			if title == "" {
				title = e.Slug
			}
			fmt.Printf("    %s %s%s%s%s\n", e.Type[:1], title, tracker, method, claimed)
		}
		fmt.Println()
	}

	// Suggest next actions
	for _, s := range stages {
		switch s.Name {
		case "imported":
			if s.Count > 0 {
				fmt.Printf("  → %d specs ready for diagnosis: hero pipeline --run diagnose\n", s.Count)
			}
		case "diagnosed":
			if s.Count > 0 {
				fmt.Printf("  → %d specs diagnosed, review and approve for delivery\n", s.Count)
			}
		case "approved":
			if s.Count > 0 {
				fmt.Printf("  → %d specs ready for delivery: hero pipeline --run deliver\n", s.Count)
			}
		case "delivering":
			if s.Count > 0 {
				fmt.Printf("  → %d specs in delivery: hero verify <slug> when done\n", s.Count)
			}
		}
	}
	fmt.Println()
}

func stageIcon(name string) string {
	switch name {
	case "imported":
		return "○"
	case "diagnosed":
		return "◐"
	case "approved":
		return "●"
	case "delivering":
		return "▶"
	case "completed":
		return "✓"
	default:
		return "?"
	}
}
