package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Spec lifecycle: scaffold, mock, demo, deliver, verify, complete",
	Long: `Subverbs that operate on a spec — the full lifecycle from
scaffold to delivery, plus design artifacts and quality gates.

  spec new      scaffold a new spec from template
  spec mock     list / open / serve design mockups
  spec demo     record and manage video demos
  spec claim    claim a spec (set claimed_by)
  spec unclaim  release a spec
  spec claims   list current claims
  spec plan     view or manage execution plans
  spec score    score a spec's quality and readiness
  spec deliver  start delivery of a spec
  spec verify   verify implementation against acceptance criteria
  spec complete mark a spec completed
  spec contract criteria-to-test traceability
  spec lint     classify acceptance criteria (EARS coverage)`,
}

var (
	specLintAll bool
)

var specLintCmd = &cobra.Command{
	Use:   "lint [<slug>]",
	Short: "Classify acceptance criteria (EARS vs freeform) and report coverage",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSpecLint,
}

func init() {
	specLintCmd.Flags().BoolVar(&specLintAll, "all", false, "lint every work spec in the workspace")
	specCmd.AddCommand(specLintCmd)

	// Subverbs migrated from top-level commands. Variable names
	// retained for low-churn; only Use strings + parent registration
	// changed.
	specCmd.AddCommand(newCmd)
	specCmd.AddCommand(mockCmd)
	specCmd.AddCommand(demoCmd)
	specCmd.AddCommand(claimCmd)
	specCmd.AddCommand(unclaimCmd)
	specCmd.AddCommand(claimsCmd)
	specCmd.AddCommand(planCmd)
	specCmd.AddCommand(scoreCmd)
	specCmd.AddCommand(deliverCmd)
	specCmd.AddCommand(verifyCmd)
	specCmd.AddCommand(completeCmd)
	specCmd.AddCommand(contractCmd)
	specCmd.AddCommand(diagnoseCmd)
}

func runSpecLint(cmd *cobra.Command, args []string) error {
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

	var targets []*spec.Spec
	switch {
	case specLintAll:
		for _, s := range specs {
			if s.IsWorkSpec() {
				targets = append(targets, s)
			}
		}
	case len(args) == 1:
		s := findSpec(args[0], specs)
		if s == nil {
			return fmt.Errorf("spec %q not found", args[0])
		}
		targets = []*spec.Spec{s}
	default:
		return fmt.Errorf("provide a spec slug or use --all")
	}

	var totalCriteria, totalEARS, totalFreeform int
	for _, s := range targets {
		criteria := s.AcceptanceCriteria()
		if len(criteria) == 0 {
			fmt.Printf("%s — no acceptance criteria section\n", s.Slug)
			continue
		}

		kindCounts := make(map[spec.CriterionKind]int)
		for _, c := range criteria {
			kindCounts[c.Kind]++
			totalCriteria++
			if c.Kind.IsEARS() {
				totalEARS++
			} else {
				totalFreeform++
			}
		}

		fmt.Printf("%s (%d criteria)\n", s.Slug, len(criteria))
		ears := len(criteria) - kindCounts[spec.CriterionFreeform]
		fmt.Printf("  %s %d EARS  (%s)\n", okMarker(ears > 0), ears, formatKindBreakdown(kindCounts))
		if freeform := kindCounts[spec.CriterionFreeform]; freeform > 0 {
			fmt.Printf("  %s %d freeform\n", warnMarker(), freeform)
			for _, c := range criteria {
				if c.Kind == spec.CriterionFreeform {
					fmt.Printf("     - %q\n", truncate(c.Raw, 80))
				}
			}
		}
		fmt.Println()
	}

	if specLintAll {
		fmt.Printf("Totals: %d criteria across %d spec(s) — %d EARS, %d freeform",
			totalCriteria, len(targets), totalEARS, totalFreeform)
		if totalCriteria > 0 {
			ratio := float64(totalEARS) / float64(totalCriteria) * 100
			fmt.Printf(" (%.0f%% EARS)", ratio)
		}
		fmt.Println()
	}

	return nil
}

func okMarker(ok bool) string {
	if ok {
		return "✓"
	}
	return "·"
}

func warnMarker() string { return "⚠" }

func formatKindBreakdown(counts map[spec.CriterionKind]int) string {
	var parts []string
	for _, k := range []spec.CriterionKind{
		spec.CriterionEvent, spec.CriterionState, spec.CriterionUbiquitous,
		spec.CriterionOptional, spec.CriterionUnwanted,
	} {
		if n := counts[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, k))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return joinWith(parts, ", ")
}

func joinWith(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += sep + p
	}
	return out
}
