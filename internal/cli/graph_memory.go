package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/hero-engine/hero/internal/codescan"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/knowledge"
	"github.com/hero-engine/hero/internal/nextdoc"
	"github.com/hero-engine/hero/internal/sessions"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// Subcommands of `hero graph` that operate on the unified knowledge
// graph (graph.db). The pre-existing positional form `hero graph <slug>`
// still works — cobra dispatches to a subcommand only when the first
// arg matches one of these names.

var graphStatsJSON bool

var graphStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show node and edge counts in the knowledge graph",
	Long: `Reports current node counts by type, edge counts by type, and the
append-only history depth. Reads .hero/graph.db.`,
	RunE: runGraphStats,
}

var graphReingestCmd = &cobra.Command{
	Use:   "reingest [subgraph]",
	Short: "Re-populate a subgraph from its source of truth",
	Long: `Re-runs the ingest path for one or more subgraphs:

  code   re-scans the codebase → Repo/Package/File/Symbol nodes
         plus belongs_to/defines/imports edges
  work   re-reads planning specs, sessions, git log → Feature/
         Initiative/Decision/Session/Commit/Person nodes plus
         belongs_to/depends_on/authored_by/touches/mentions edges
  all    runs every ingest path in the right order

If omitted, defaults to "all".`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGraphReingest,
}

func init() {
	graphStatsCmd.Flags().BoolVar(&graphStatsJSON, "json", false, "emit stats as JSON")
	graphCmd.AddCommand(graphStatsCmd)
	graphCmd.AddCommand(graphReingestCmd)
}

func runGraphStats(cmd *cobra.Command, args []string) error {
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

	stats, err := store.Stats()
	if err != nil {
		return fmt.Errorf("computing stats: %w", err)
	}

	if graphStatsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(stats)
	}

	fmt.Printf("Graph: %s\n", store.Path())
	fmt.Printf("  schema_version: %s\n", stats.SchemaVersion)
	fmt.Printf("  install_id:     %s\n\n", stats.InstallID)

	fmt.Printf("Current nodes (%d):\n", stats.TotalNodes)
	for _, t := range sortedKeys(stats.NodesByType) {
		fmt.Printf("  %-12s %d\n", t, stats.NodesByType[t])
	}
	if len(stats.NodesByScope) > 0 {
		fmt.Println()
		fmt.Println("Nodes by scope:")
		for _, sc := range sortedKeys(stats.NodesByScope) {
			fmt.Printf("  %-8s %d\n", sc, stats.NodesByScope[sc])
		}
	}
	if len(stats.NodesByRepo) > 0 {
		fmt.Println()
		fmt.Println("Nodes by repo:")
		for _, r := range sortedKeys(stats.NodesByRepo) {
			fmt.Printf("  %-20s %d\n", r, stats.NodesByRepo[r])
		}
	}
	if len(stats.NodesByUnit) > 0 {
		fmt.Println()
		fmt.Println("Nodes by unit:")
		for _, u := range sortedKeys(stats.NodesByUnit) {
			fmt.Printf("  %-20s %d\n", u, stats.NodesByUnit[u])
		}
	}
	fmt.Printf("\nCurrent edges (%d):\n", stats.TotalEdges)
	for _, t := range sortedKeys(stats.EdgesByType) {
		fmt.Printf("  %-12s %d\n", t, stats.EdgesByType[t])
	}
	fmt.Printf("\nHistory rows: %d nodes, %d edges (append-only)\n",
		stats.HistoryRows.Nodes, stats.HistoryRows.Edges)
	return nil
}

func runGraphReingest(cmd *cobra.Command, args []string) error {
	subgraph := "all"
	if len(args) > 0 {
		subgraph = args[0]
	}
	switch subgraph {
	case "code", "work", "all":
	default:
		return fmt.Errorf("unknown subgraph %q (want code | work | all)", subgraph)
	}

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

	if subgraph == "code" || subgraph == "all" {
		if err := reingestCode(cfg, projectRoot, store); err != nil {
			return err
		}
	}
	if subgraph == "work" || subgraph == "all" {
		if err := reingestWork(cfg, projectRoot, heroDir, store); err != nil {
			return err
		}
	}
	return nil
}

func reingestCode(cfg config.Config, projectRoot string, store *graph.Store) error {
	codeDir := cfg.CodeDir(projectRoot)
	prevChecksums, err := codescan.LoadChecksums(codeDir)
	if err != nil {
		return fmt.Errorf("loading checksums: %w", err)
	}
	prevCache, err := codescan.LoadScanCache(codeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load scan cache, re-parsing all files: %v\n", err)
		prevCache = nil
	}
	scanner := codescan.NewScannerWithMode(cfg.CodeScan, projectRoot, resolveParser(cfg.CodeScan.Parser))
	result, err := scanner.Scan(prevChecksums, prevCache)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}
	summary, err := codescan.WriteGraph(result, store, activeDomain(cfg))
	if err != nil {
		return fmt.Errorf("writing code subgraph: %w", err)
	}
	fmt.Printf("code: %d packages, %d files, %d symbols, %d edges\n",
		summary.Packages, summary.Files, summary.Symbols, summary.Edges)
	return nil
}

// reingestWork drives the three work-subgraph ingest paths in dependency
// order: specs first (so sessions can resolve mentions edges), then
// sessions, then git log (so touches edges can resolve File targets
// already created by codescan).
func reingestWork(cfg config.Config, projectRoot, heroDir string, store *graph.Store) error {
	repoKey := filepath.Base(projectRoot)

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	specSummary, err := spec.WriteGraph(specs, repoKey, graph.DomainFor(cfg, graph.IntrinsicActive), store)
	if err != nil {
		return fmt.Errorf("writing spec subgraph: %w", err)
	}
	fmt.Printf("specs: %d nodes, %d edges\n", specSummary.Specs, specSummary.Edges)

	sessList, err := sessions.List(heroDir)
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}
	sessSummary, err := sessions.WriteGraph(sessList, repoKey, store)
	if err != nil {
		return fmt.Errorf("writing sessions subgraph: %w", err)
	}
	fmt.Printf("sessions: %d nodes, %d edges\n", sessSummary.Sessions, sessSummary.Edges)

	gitSummary, err := gitutil.WriteGitLogGraph(projectRoot, repoKey, 0, store)
	if err != nil {
		return fmt.Errorf("writing git-log subgraph: %w", err)
	}
	fmt.Printf("git: %d commits, %d persons, %d issues, %d edges\n",
		gitSummary.Commits, gitSummary.Persons, gitSummary.Issues, gitSummary.Edges)

	rawSummary, err := knowledge.WriteRawGraph(heroDir, repoKey, store)
	if err != nil {
		return fmt.Errorf("writing raw-doc subgraph: %w", err)
	}
	if rawSummary.Documents > 0 {
		fmt.Printf("raw: %d documents\n", rawSummary.Documents)
	}

	nextSummary, err := nextdoc.WriteGraph(heroDir, repoKey, store)
	if err != nil {
		return fmt.Errorf("writing NEXT.md subgraph: %w", err)
	}
	if nextSummary.Sessions > 0 || nextSummary.Attempts > 0 {
		fmt.Printf("next: %d sessions, %d attempts, %d edges\n",
			nextSummary.Sessions, nextSummary.Attempts, nextSummary.Edges)
	}

	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
