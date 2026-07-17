package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/score"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var scoreJSON bool

var scoreCmd = &cobra.Command{
	Use:   "score <slug-or-path>",
	Short: "Score a spec's quality and readiness for delivery",
	Long: `Evaluates a spec across 6 dimensions:
  - Acceptance Criteria (25%) — measurable, testable criteria
  - Scope Clarity (20%) — goal, non-goals, design sections
  - Technical Specificity (20%) — references to files, packages, APIs
  - Test Strategy (15%) — testing plan or verification approach
  - Structure (10%) — document organization and detail
  - Clarity (10%) — absence of vague or ambiguous language

Returns a score (0-100), grade (A-F), and actionable suggestions.`,
	Args: cobra.ExactArgs(1),
	RunE: runScore,
}

func init() {
	scoreCmd.Flags().BoolVar(&scoreJSON, "json", false, "output as JSON")
}

func runScore(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)

	s, err := resolveSpec(args[0], heroDir)
	if err != nil {
		return err
	}

	// Build score config from hero.json if available
	scoreCfg := score.DefaultConfig()
	if cfg.Score != nil && cfg.Score.MinScore > 0 {
		scoreCfg.MinScore = cfg.Score.MinScore
	}

	result := score.Score(s, scoreCfg)

	if scoreJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	printScoreResult(result, scoreCfg)
	return nil
}

// resolveSpec finds a spec by directory, file path, or slug.
func resolveSpec(arg, heroDir string) (*spec.Spec, error) {
	// Directory: resolve to the spec inside it (single-file spec.md,
	// otherwise a three-file requirements.md layout).
	if info, err := os.Stat(arg); err == nil && info.IsDir() {
		if p := filepath.Join(arg, "spec.md"); fileExists(p) {
			return spec.ParseFile(p)
		}
		if fileExists(filepath.Join(arg, "requirements.md")) {
			return spec.ParseThreeFile(arg)
		}
		return nil, fmt.Errorf("no spec.md or requirements.md in %s", arg)
	}

	// Try as a direct file path
	if strings.HasSuffix(arg, ".md") || strings.Contains(arg, string(filepath.Separator)) {
		return spec.ParseFile(arg)
	}

	// Try as a slug — search planning/ and specs/
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	for _, s := range specs {
		if s.Slug == arg {
			return s, nil
		}
	}

	return nil, fmt.Errorf("spec %q not found (try a slug or file path)", arg)
}

func printScoreResult(r *score.Result, cfg *score.Config) {
	// Header
	fmt.Printf("\n  Spec:  %s\n", r.Slug)
	fmt.Printf("  Score: %d/100  Grade: %s", r.Score, r.Grade)
	if r.Deliverable {
		fmt.Printf("  (deliverable)\n")
	} else {
		fmt.Printf("  (below minimum %d)\n", cfg.MinScore)
	}

	// Dimensions
	fmt.Printf("\n  %-25s %5s  %s\n", "DIMENSION", "SCORE", "DETAILS")
	fmt.Printf("  %s\n", strings.Repeat("─", 70))
	for _, d := range r.Dimensions {
		marker := " "
		if d.Score == 0 {
			marker = "✗"
		} else if d.Score >= 75 {
			marker = "✓"
		} else if d.Score < 50 {
			marker = "!"
		}
		fmt.Printf("  %s %-23s %5.0f  %s\n", marker, d.Name, d.Score, d.Details)
	}

	// Warnings
	if len(r.Warnings) > 0 {
		fmt.Printf("\n  Warnings:\n")
		for _, w := range r.Warnings {
			icon := "⚠"
			if w.Severity == "error" {
				icon = "✗"
			}
			fmt.Printf("    %s %s\n", icon, w.Message)
		}
	}

	// Suggestions
	if len(r.Suggestions) > 0 {
		fmt.Printf("\n  Suggestions:\n")
		for _, s := range r.Suggestions {
			fmt.Printf("    → %s\n", s)
		}
	}

	fmt.Println()
}
