package cli

import (
	"fmt"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// `hero spec horizon-migrate` — Phase 5 of spec-prioritization.
// One-shot bulk classifier: walks the planning corpus, applies the
// heuristics in spec.PlanHorizonMigration, and either prints the
// proposed plan (--dry-run) or writes the horizon: ... frontmatter
// field to each spec.

var (
	horizonMigrateDryRun bool
)

var specHorizonMigrateCmd = &cobra.Command{
	Use:   "horizon-migrate",
	Short: "Bulk-tag every spec with a proposed horizon (now/next/someday/parking)",
	Long: `Walks every spec in the planning corpus and proposes a horizon
based on heuristics:

  - Already has a valid horizon → skipped
  - status: completed / delivering / in-review / regressed → now
  - tags or slug match marketing/sales/launch/positioning/etc. → someday
  - tags include v2-recovery → now
  - everything else → next (default)

By default writes the proposals into each spec's frontmatter.
Use --dry-run to preview without writing.

After this runs once on a fresh repo, use:
  hero spec promote <slug>   to bump horizon up
  hero spec park <slug>      to demote to parking with a reason`,
	RunE: runSpecHorizonMigrate,
}

func init() {
	specHorizonMigrateCmd.Flags().BoolVar(&horizonMigrateDryRun, "dry-run", false, "print the migration plan without writing")
	specCmd.AddCommand(specHorizonMigrateCmd)
}

func runSpecHorizonMigrate(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	plan := spec.PlanHorizonMigration(specs)
	if len(plan) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No specs found.")
		return nil
	}

	// Group by proposed horizon for the human view.
	groups := map[spec.Horizon][]spec.HorizonProposal{}
	skipped := 0
	for _, p := range plan {
		if p.Skip {
			skipped++
			continue
		}
		groups[p.Proposed] = append(groups[p.Proposed], p)
	}

	header := "Migration plan:"
	if horizonMigrateDryRun {
		header = "Migration plan (--dry-run):"
	}
	fmt.Fprintln(cmd.OutOrStdout(), header)
	fmt.Fprintln(cmd.OutOrStdout())

	for _, h := range []spec.Horizon{spec.HorizonNow, spec.HorizonNext, spec.HorizonSomeday, spec.HorizonParking} {
		ps := groups[h]
		if len(ps) == 0 {
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  → %s (%d):\n", h, len(ps))
		for _, p := range ps {
			fmt.Fprintf(cmd.OutOrStdout(), "      %-44s  %s\n", p.Slug, p.Reason)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}
	if skipped > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  (skipped %d already at their proposed horizon)\n\n", skipped)
	}

	written, alsoSkipped, err := spec.ApplyHorizonProposals(plan, horizonMigrateDryRun)
	if err != nil {
		return err
	}
	if horizonMigrateDryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "Would write to %d spec(s). Re-run without --dry-run to apply.\n", written)
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote horizon to %d spec(s); skipped %d.\n", written, alsoSkipped)
	fmt.Fprintln(cmd.OutOrStdout(), "Review with `git diff` and commit when ready.")
	return nil
}
