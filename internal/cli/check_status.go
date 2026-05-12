package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/integrity"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// `hero check status` — workspace-wide truthfulness audit of every
// `status: completed` spec. Phase-1 surface of spec-status-integrity:
// list lying / partial / unverifiable specs and exit non-zero if any
// concrete failures (lying or partial) are present.
//
// `--auto-fix` is Phase-3 work; not implemented here. The verb just
// reports today; it'll grow the rewrite path later.

var checkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Audit `status: completed` claims against the AC graph",
	Long: `Walks every spec with status: completed and verifies the claim
against the acceptance-criteria graph. A spec is:

  verified     — every Criterion node belonging to the spec has
                  status: passing.
  lying        — at least one Criterion is failing or regressed.
                  Concrete evidence the claim is wrong.
  partial      — some Criteria pass; others are still proposed or
                  unknown (no evidence of completion either way).
  unverifiable — spec has no Criterion nodes (predates AC graph or
                  has no acceptance-criteria block).

Exits non-zero when any spec is lying or partial. Verified and
unverifiable specs never produce a non-zero exit by themselves.`,
	RunE: runCheckStatus,
}

var (
	checkStatusVerbose bool
	checkStatusAutoFix bool
	checkStatusDryRun  bool
)

func init() {
	checkStatusCmd.Flags().BoolVar(&checkStatusVerbose, "verbose", false, "show verified + unverifiable specs too")
	checkStatusCmd.Flags().BoolVar(&checkStatusAutoFix, "auto-fix", false, "rewrite lying/partial status frontmatter with the verifier's verdict")
	checkStatusCmd.Flags().BoolVar(&checkStatusDryRun, "dry-run", false, "with --auto-fix: print the fix plan without writing")
	checkCmd.AddCommand(checkStatusCmd)
}

func runCheckStatus(cmd *cobra.Command, args []string) error {
	report, err := buildStatusReport()
	if err != nil {
		return err
	}
	printStatusReport(report, checkStatusVerbose)

	// Phase 2: phased-plan checkmark audit. Walks every spec body for
	// markdown tables with a Status column, flags ones whose ✅ rows
	// disagree with frontmatter status or AC graph state.
	phasedFindings, err := buildPhasedPlanFindings(report)
	if err != nil {
		return err
	}
	if len(phasedFindings) > 0 {
		printPhasedPlanFindings(phasedFindings)
	}

	if checkStatusAutoFix {
		return runAutoFix(cmd, report)
	}
	if report.HasIssues() || len(phasedFindings) > 0 {
		// Non-zero exit for any concrete drift. Cobra translates a
		// returned error into a non-zero exit; the message is the
		// summary the user just read above.
		return fmt.Errorf("status truthfulness: %d lying, %d partial, %d phased-plan inconsistencies",
			report.Lying, report.Partial, len(phasedFindings))
	}
	return nil
}

func buildPhasedPlanFindings(report *integrity.Report) ([]integrity.PhasedPlanFinding, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	finds := map[string]integrity.Finding{}
	if report != nil {
		for _, f := range report.Findings {
			finds[f.Slug] = f
		}
	}
	return integrity.CheckPhasedPlans(specs, finds), nil
}

func printPhasedPlanFindings(findings []integrity.PhasedPlanFinding) {
	fmt.Println()
	fmt.Println("Phased-plan inconsistencies")
	fmt.Println("---------------------------")
	for _, f := range findings {
		fmt.Printf("⚠️  %s — %s\n", f.Slug, f.Inconsistency)
		fmt.Printf("    %d/%d phases ✅, %d pending, %d other (table at line %d)\n",
			f.Shipped, f.Total, f.Pending, f.Other, f.HeaderLine)
		fmt.Printf("    %s\n", f.Path)
	}
}

// runAutoFix executes Phase-3 auto-downgrade. Prints the planned
// rewrites; in --dry-run mode stops there. Otherwise writes the
// frontmatter changes and prints a summary.
func runAutoFix(cmd *cobra.Command, report *integrity.Report) error {
	plan := integrity.PlanFixes(report)
	actionable := 0
	for _, a := range plan {
		if !a.Skipped {
			actionable++
		}
	}

	fmt.Println()
	if actionable == 0 {
		fmt.Println("Auto-fix: nothing actionable. Verified and unverifiable specs are skipped.")
		return nil
	}

	if checkStatusDryRun {
		fmt.Printf("Auto-fix plan (--dry-run, no files written): %d spec(s)\n\n", actionable)
	} else {
		fmt.Printf("Auto-fix: rewriting %d spec(s)\n\n", actionable)
	}

	written := 0
	for _, a := range plan {
		if a.Skipped {
			continue
		}
		fmt.Printf("  %s  %s → %s   (%s)\n", glyphForVerdict(a.Verdict), a.Slug, a.NewStatus, a.Reason)
		fmt.Printf("    %s\n", a.Path)
		if checkStatusDryRun {
			continue
		}
		if err := integrity.ApplyFix(a); err != nil {
			fmt.Printf("    ERROR: %v\n", err)
			continue
		}
		written++
	}

	fmt.Println()
	if checkStatusDryRun {
		fmt.Println("Dry-run complete. Re-run without --dry-run to apply.")
		return nil
	}
	fmt.Printf("Wrote %d spec file(s). Review the diff with `git diff` and commit when ready.\n", written)
	return nil
}

func glyphForVerdict(v integrity.Verdict) string {
	switch v {
	case integrity.VerdictLying:
		return "🔻"
	case integrity.VerdictPartial:
		return "⚠️ "
	case integrity.VerdictUnverifiable:
		return "❓"
	case integrity.VerdictVerified:
		return "✅"
	}
	return "  "
}

// statusSummaryLine returns the one-line summary surfaced inside the
// default `hero check` output. Empty string when no completed specs
// exist (early-stage projects).
func statusSummaryLine(report *integrity.Report) string {
	if report.Total() == 0 {
		return ""
	}
	parts := []string{
		fmt.Sprintf("%d/%d verified", report.Verified, report.Total()),
	}
	if report.Lying > 0 {
		parts = append(parts, fmt.Sprintf("%d lying", report.Lying))
	}
	if report.Partial > 0 {
		parts = append(parts, fmt.Sprintf("%d partial", report.Partial))
	}
	if report.Unverifiable > 0 {
		parts = append(parts, fmt.Sprintf("%d unverifiable", report.Unverifiable))
	}
	return "Status truthfulness: " + strings.Join(parts, ", ")
}

func buildStatusReport() (*integrity.Report, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}
	store, err := graph.Open(heroDir)
	if err != nil {
		return nil, fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	return integrity.CheckCompletedSpecs(specs, store)
}

func printStatusReport(report *integrity.Report, verbose bool) {
	fmt.Println("Status truthfulness audit")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Printf("Specs claiming `completed`:    %d\n", report.Total())
	fmt.Printf("  Verified by passing ACs:     %d ✅\n", report.Verified)
	if report.Lying > 0 {
		fmt.Printf("  Lying (failing/regressed):   %d 🔻\n", report.Lying)
	}
	if report.Partial > 0 {
		fmt.Printf("  Partial (open ACs):          %d ⚠️\n", report.Partial)
	}
	if report.Unverifiable > 0 {
		fmt.Printf("  No ACs (cannot verify):      %d ❓\n", report.Unverifiable)
	}
	fmt.Println()

	for _, f := range report.Findings {
		switch f.Verdict {
		case integrity.VerdictLying:
			fmt.Printf("🔻 %s — lying (%d/%d passing, %d failing)\n",
				f.Slug, f.Passing, f.Total, f.Failing+f.Regressed)
			for _, k := range f.FailingKeys {
				fmt.Printf("     failing AC: %s\n", k)
			}
		case integrity.VerdictPartial:
			fmt.Printf("⚠️  %s — partial (%d/%d passing, %d still open)\n",
				f.Slug, f.Passing, f.Total, f.ProposedOrOpen)
			if len(f.OpenKeys) > 0 && len(f.OpenKeys) <= 4 {
				for _, k := range f.OpenKeys {
					fmt.Printf("     open AC: %s\n", k)
				}
			} else if len(f.OpenKeys) > 4 {
				for _, k := range f.OpenKeys[:3] {
					fmt.Printf("     open AC: %s\n", k)
				}
				fmt.Printf("     … and %d more\n", len(f.OpenKeys)-3)
			}
		case integrity.VerdictUnverifiable:
			if verbose {
				fmt.Printf("❓ %s — unverifiable (no ACs)\n", f.Slug)
			}
		case integrity.VerdictVerified:
			if verbose {
				fmt.Printf("✅ %s — verified (%d ACs passing)\n", f.Slug, f.Total)
			}
		}
	}

	if report.HasIssues() {
		fmt.Println()
		fmt.Println("Run `hero check status --auto-fix` to rewrite lying/partial")
		fmt.Println("frontmatter with the verifier's verdict (use --dry-run to preview).")
	}
}
