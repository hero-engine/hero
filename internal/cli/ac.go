package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/integrity"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// `hero ac` umbrella for acceptance-criteria operations. The Phase-2
// surface is intentionally minimal: `record` ingests a run-result
// JSON. `status` and `history` queries arrive in Phase 5.

var acCmd = &cobra.Command{
	Use:   "ac",
	Short: "Acceptance-criteria graph operations",
	Long: `Acceptance-criteria graph operations.

  hero ac list <spec-slug>      list ACs for a spec (markdown or --json)
  hero ac record <run.json>     ingest run results, flip Criterion status
  hero ac status [--feature X]  pass-rate per spec across the corpus
  hero ac history <ac-key>      timeline of every status flip for an AC`,
}

var acListCmd = &cobra.Command{
	Use:   "list <spec-slug>",
	Short: "List acceptance criteria for a spec from the graph",
	Long: `Reads Criterion nodes for the given spec slug from the graph and
prints them — markdown by default, JSON with --json. Designed to feed
e2e suite harnesses (which need the AC list without duplicating it
from spec.md).

Output (--json):
  [{"key":"<slug>:AC-N","ac_id":"AC-N","statement":"...","status":"..."}, ...]`,
	Args: cobra.ExactArgs(1),
	RunE: runAcList,
}

var acListJSON bool

var acRecordCmd = &cobra.Command{
	Use:   "record <run.json>",
	Short: "Ingest a run-result JSON file and flip Criterion status",
	Long: `Reads a run-result JSON file and applies its results to the
acceptance-criteria graph: each entry flips the matching Criterion
node's status (bitemporally — the prior row is invalidated and a new
current row is inserted), and a satisfied_by / breaks edge is wired
to the named commit when one is supplied.

Schema:
  [
    {"ac": "<spec-slug>:AC-N", "status": "pass|fail",
     "ts": "2026-04-28T22:30:00Z", "sha": "abc123"},
    ...
  ]

Status values: pass, fail (and skip — silently no-op'd).
Pass-after-fail flips to "passing"; fail-after-pass flips to
"regressed".

Unknown AC keys are reported in the summary but never fail the
batch — stale payloads are tolerated.`,
	Args: cobra.ExactArgs(1),
	RunE: runAcRecord,
}

var acRecordJSON bool

var (
	acStatusFeature string
	acStatusJSON    bool
	acHistoryJSON   bool
)

var acStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "AC pass-rate per spec across the corpus",
	Long: `Reads every Criterion node and rolls up status counts per parent
spec. Default output is markdown with one row per spec; --json emits
the raw rollup. --feature <slug> narrows to one spec.

Designed as a quick "what's red right now" overview without leaving
the terminal. For specific failures, hero blocked already joins
failing ACs into the dependency tree.`,
	RunE: runAcStatus,
}

var acHistoryCmd = &cobra.Command{
	Use:   "history <spec-slug>:<AC-N>",
	Short: "Show every recorded status flip for one AC",
	Long: `Walks the bitemporal Criterion rows for the given AC key (oldest
first) and prints each [valid_from, valid_to) interval and its
status. The current row has an open-ended interval.

Example:
  hero ac history next-as-projection:AC-3
  proposed   2026-04-28T03:54:52Z → 2026-04-28T04:13:58Z
  passing    2026-04-28T04:13:58Z → (current)`,
	Args: cobra.ExactArgs(1),
	RunE: runAcHistory,
}

func init() {
	acRecordCmd.Flags().BoolVar(&acRecordJSON, "json", false, "emit machine-readable summary")
	acListCmd.Flags().BoolVar(&acListJSON, "json", false, "emit JSON array suitable for piping into scripts")
	acStatusCmd.Flags().StringVar(&acStatusFeature, "feature", "", "narrow to one spec slug")
	acStatusCmd.Flags().BoolVar(&acStatusJSON, "json", false, "emit JSON rollup")
	acHistoryCmd.Flags().BoolVar(&acHistoryJSON, "json", false, "emit JSON timeline")
	acCmd.AddCommand(acListCmd)
	acCmd.AddCommand(acRecordCmd)
	acCmd.AddCommand(acStatusCmd)
	acCmd.AddCommand(acHistoryCmd)
	rootCmd.AddCommand(acCmd)
}

func runAcList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	criteria, err := acceptance.ListBySpec(store, args[0])
	if err != nil {
		return fmt.Errorf("listing criteria: %w", err)
	}
	if len(criteria) == 0 {
		if acListJSON {
			fmt.Println("[]")
			return nil
		}
		fmt.Printf("No acceptance criteria found for %q.\n", args[0])
		fmt.Println("(`hero scan` ingests Criterion nodes from `## Acceptance criteria` blocks.)")
		return nil
	}

	if acListJSON {
		// Emit one JSON object per AC with the fields scripts need.
		// Inline-marshal to preserve key order — most consumers parse
		// the array shape, but stable keys help diffing.
		fmt.Println("[")
		for i, c := range criteria {
			sep := ","
			if i == len(criteria)-1 {
				sep = ""
			}
			data, _ := json.Marshal(struct {
				Key       string `json:"key"`
				ACID      string `json:"ac_id"`
				Statement string `json:"statement"`
				Status    string `json:"status"`
				Parent    string `json:"parent"`
			}{
				Key: c.Key, ACID: c.ACID, Statement: c.Statement,
				Status: c.Status, Parent: c.Parent,
			})
			fmt.Printf("  %s%s\n", string(data), sep)
		}
		fmt.Println("]")
		return nil
	}

	fmt.Printf("Acceptance criteria for `%s` (%d):\n\n", args[0], len(criteria))
	for _, c := range criteria {
		fmt.Printf("  %s  %s — %s\n", statusGlyph(c.Status), c.ACID, summarize(c.Statement, 90))
	}
	return nil
}

func runAcRecord(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	results, err := acceptance.LoadRunResults(args[0])
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Println("Run-result file is empty — no Criterion changes.")
		return nil
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(projectRoot)
	summary, err := acceptance.Record(results, repoKey, store)
	if err != nil {
		return fmt.Errorf("recording results: %w", err)
	}

	// Phase 4: refresh File→Criterion participation edges so any new
	// satisfied_by edges this run created get joined against the
	// existing Commit→touches→File edges. Best-effort: a participation
	// failure shouldn't fail the whole record (the Criterion flips
	// already landed), but it should surface in the summary.
	part, partErr := acceptance.ComputeParticipation(store, repoKey)

	// spec-status-integrity AC-6: any AC flip to failing/regressed on
	// a parent spec that claims status: completed gets the spec auto-
	// downgraded to status: regressed. Best-effort — a downgrade
	// failure (e.g. spec.md path not writable) is logged but doesn't
	// fail the record itself.
	specs, _ := spec.Discover(heroDir)
	downgrades, dgErr := integrity.AutoDowngradeRegressions(specs, store, false)

	if acRecordJSON {
		out := struct {
			Summary       *acceptance.RecordSummary       `json:"summary"`
			Participation acceptance.ParticipationSummary `json:"participation"`
		}{Summary: summary, Participation: part}
		return json.NewEncoder(os.Stdout).Encode(out)
	}

	fmt.Printf("Recorded %d run results from %s.\n", len(results), args[0])
	fmt.Printf("  Status flips:   %d\n", summary.Criteria)
	fmt.Printf("  No-ops:         %d\n", summary.NoOps)
	if summary.Unknown > 0 {
		fmt.Printf("  Unknown ACs:    %d (stale or never-ingested)\n", summary.Unknown)
	}
	if summary.SatisfiedBy > 0 {
		fmt.Printf("  satisfied_by:   %d\n", summary.SatisfiedBy)
	}
	if summary.Breaks > 0 {
		fmt.Printf("  breaks:         %d (regressions)\n", summary.Breaks)
	}
	if partErr != nil {
		fmt.Fprintf(os.Stderr, "  warning: participation join failed: %v\n", partErr)
	} else if part.Edges > 0 {
		fmt.Printf("  participates_in:%d new (across %d files)\n", part.Edges, part.Touched)
	}
	if dgErr != nil {
		fmt.Fprintf(os.Stderr, "  warning: regression downgrade failed: %v\n", dgErr)
	} else if len(downgrades) > 0 {
		fmt.Printf("\nAuto-downgraded %d spec(s) due to AC regression:\n", len(downgrades))
		for _, dg := range downgrades {
			fmt.Printf("  🔻 %s: completed → regressed (%s)\n",
				dg.Slug, strings.Join(dg.RegressedACs, ", "))
		}
	}
	return nil
}

func runAcStatus(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	rows, err := acceptance.StatusByFeature(store, acStatusFeature)
	if err != nil {
		return fmt.Errorf("status query: %w", err)
	}

	if acStatusJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
	}

	if len(rows) == 0 {
		if acStatusFeature != "" {
			fmt.Printf("No acceptance criteria found for %q.\n", acStatusFeature)
		} else {
			fmt.Println("No acceptance criteria in graph. (Run `hero scan` first.)")
		}
		return nil
	}

	fmt.Printf("%-40s  %5s  %4s %4s %4s %4s  %s\n",
		"Spec", "Total", "Pass", "Fail", "Reg", "Prop", "Pass-rate")
	fmt.Println(strings.Repeat("-", 90))
	for _, r := range rows {
		rate := ""
		if r.Total > 0 {
			rate = fmt.Sprintf("%3.0f%%", r.PassRate()*100)
		}
		fmt.Printf("%-40s  %5d  %4d %4d %4d %4d  %s\n",
			truncate(r.Parent, 40), r.Total, r.Passing, r.Failing, r.Regressed, r.Proposed, rate)
	}
	return nil
}

func runAcHistory(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	entries, err := acceptance.History(store, args[0])
	if err != nil {
		return fmt.Errorf("history query: %w", err)
	}

	if acHistoryJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(entries)
	}

	if len(entries) == 0 {
		fmt.Printf("No history for %q. Check the AC key — format is <spec-slug>:AC-N.\n", args[0])
		return nil
	}

	fmt.Printf("History for %s:\n\n", args[0])
	for _, e := range entries {
		to := "(current)"
		if e.ValidTo != "" {
			to = e.ValidTo
		}
		fmt.Printf("  %s  %s → %s\n", statusGlyph(e.Status), e.ValidFrom, to)
		fmt.Printf("                                         %s\n", e.Status)
	}
	return nil
}

// (truncate is defined in session.go and shared with the ac status table.)
