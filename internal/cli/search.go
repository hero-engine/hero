package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/retrieval"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the unified knowledge graph",
	Long: `Searches across every node type in the knowledge graph: Features,
Decisions, Notes, Attempts, Commits, Symbols, and more. Returns
ranked, compact results that beat raw grep on cross-subgraph
questions.

Filter flags (--type, --status, --tag, --since, --list,
--cross-repo) and --specs route to the legacy FTS5 spec-only index
for backward compatibility.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if searchListOnly {
			return nil
		}
		return cobra.MinimumNArgs(1)(cmd, args)
	},
	RunE: runSearch,
}

var (
	searchByFile    bool
	searchType       string
	searchStatus     string
	searchTag        string
	searchSince      string
	searchListOnly   bool
	searchCrossRepo  bool
	searchSpecsOnly  bool
	searchBudget     int
	searchJSON       bool
	searchSubproject string
	searchSemantic        bool
	searchHybrid          bool
	searchIncludeSuperseded bool
	searchKnowledge         bool
)

func init() {
	searchCmd.Flags().BoolVar(&searchByFile, "file", false, "search by file path instead of content (FTS5 path)")
	searchCmd.Flags().StringVar(&searchType, "type", "", "filter by spec type (FTS5 path)")
	searchCmd.Flags().StringVar(&searchStatus, "status", "", "filter by status (FTS5 path)")
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "filter by tag (FTS5 path)")
	searchCmd.Flags().StringVar(&searchSince, "since", "", "filter by creation date YYYY-MM-DD (FTS5 path)")
	searchCmd.Flags().BoolVar(&searchListOnly, "list", false, "list specs matching filters without a text query (FTS5 path)")
	searchCmd.Flags().BoolVar(&searchCrossRepo, "cross-repo", false, "search across all configured repos (FTS5 path)")
	searchCmd.Flags().BoolVar(&searchSpecsOnly, "specs", false, "force the legacy FTS5 spec-only search")
	searchCmd.Flags().BoolVar(&searchKnowledge, "knowledge", false, "search the hand-authored .hero/knowledge/** corpus instead of work specs (--type filters by knowledge kind)")
	searchCmd.Flags().IntVar(&searchBudget, "budget", 800, "token budget for graph search results")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "emit JSON (graph search only)")
	searchCmd.Flags().StringVar(&searchSubproject, "subproject", "", "filter by subproject scope (e.g. engines/mlx); 'all' disables. Default: active scope from cwd")
	searchCmd.Flags().BoolVar(&searchSemantic, "semantic", false, "vector-only semantic search (requires embedding model)")
	searchCmd.Flags().BoolVar(&searchHybrid, "hybrid", false, "hybrid BM25+vector search (default when embeddings available)")
	searchCmd.Flags().BoolVar(&searchIncludeSuperseded, "include-superseded", false, "skip the rank de-weight on specs carrying superseded_by (the [SUPERSEDED → slug] marker is shown either way)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Self-heal index drift so newly-created specs (or specs edited
	// outside the indexing path) surface in search results without
	// requiring a manual `hero index`. Spec: index-staleness-auto-refresh.
	if _, err := index.RefreshIfStale(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: index refresh failed: %v\n", err)
	}

	// --file, --list, and --cross-repo are FTS5-specific modes that have no
	// unified-retrieval equivalent yet. Handle them with the direct FTS5 path.
	if searchByFile || searchListOnly || searchCrossRepo {
		return runSearchFTS(heroDir, cfg, projectRoot, args)
	}

	// All other paths (plain text, filter flags, --specs) go through the
	// unified retrieval layer. The retrieval package routes internally:
	//   - Filters or Types present → FTS5
	//   - Plain text, no filters   → graph-first, FTS5 fallback
	q := retrieval.Query{
		Text:              strings.Join(args, " "),
		Limit:             searchBudget,
		SemanticOK:        searchSemantic || searchHybrid,
		IncludeSuperseded: searchIncludeSuperseded,
	}
	// --knowledge scopes to the isolated knowledge corpus; --type filters by
	// knowledge kind (battlecards, decisions, …). Spec: knowledge-surfacing.
	if searchKnowledge {
		q.KnowledgeOnly = true
		q.SemanticOK = false
		if searchType != "" {
			q.Types = []string{searchType}
		}
		ret, err := retrieval.New(heroDir)
		if err != nil {
			return fmt.Errorf("opening retrieval layer: %w", err)
		}
		defer ret.Close()
		results, err := ret.Retrieve(q)
		if err != nil {
			return fmt.Errorf("searching: %w", err)
		}
		if len(results) == 0 {
			if searchJSON {
				fmt.Println("[]")
				return nil
			}
			fmt.Println("No knowledge found.")
			return nil
		}
		return printFTSResults(results, searchJSON)
	}
	if searchSpecsOnly || hasFilters() {
		q.Types = nil
		q.Filters = make(map[string]string)
		if searchType != "" {
			q.Types = []string{searchType}
			q.Filters["type"] = searchType
		}
		if searchStatus != "" {
			q.Filters["status"] = searchStatus
		}
		if searchTag != "" {
			q.Filters["tag"] = searchTag
		}
		if searchSince != "" {
			q.Filters["since"] = searchSince
		}
		// Resolve subproject filter: explicit flag wins, else default
		// to active scope from cwd. "all" disables filtering.
		sp := resolveSubprojectFilter(searchSubproject)
		if sp != "" && sp != "all" {
			q.Filters["subproject"] = sp
		}
		maybePrintScopeHint(cmd.ErrOrStderr(), searchSubproject, sp)
		// Ensure the routing goes to FTS5 even when no specific filter was set
		// (i.e. --specs alone with no other flags).
		if len(q.Filters) == 0 && len(q.Types) == 0 {
			q.Filters = map[string]string{"_specs_only": "1"}
		}
	}

	ret, err := retrieval.New(heroDir)
	if err != nil {
		return fmt.Errorf("opening retrieval layer: %w", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(q)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	if len(results) == 0 {
		if searchJSON {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No results found.")
		return nil
	}

	// Graph/hybrid/vector results use the compact markdown format; FTS5
	// results use the tabular format.
	switch results[0].Source {
	case "graph", "hybrid", "vector":
		return printGraphResults(results, strings.Join(args, " "), searchBudget, searchJSON, gitutil.RepoKey(projectRoot))
	default:
		return printFTSResults(results, searchJSON)
	}
}

// runSearchFTS handles the FTS5-specific modes: --file, --list, --cross-repo.
// These don't map to a unified text-search Query so they bypass retrieval.
func runSearchFTS(heroDir string, cfg config.Config, projectRoot string, args []string) error {
	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	query := strings.Join(args, " ")
	var results []index.SearchResult

	if searchListOnly {
		results, err = idx.ListFiltered(searchType, searchStatus, searchTag, searchSince)
	} else if searchByFile {
		results, err = idx.SearchByFile(query)
	} else {
		results, err = idx.Search(query)
	}
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	if searchCrossRepo {
		repos := cfg.ResolveAllRepos(projectRoot)
		for alias, rs := range repos {
			if !rs.Accessible {
				continue
			}
			repoHeroDir := cfg.HeroDir(rs.Path)
			repoIdx, openErr := index.Open(repoHeroDir)
			if openErr != nil {
				continue
			}
			var repoResults []index.SearchResult
			if searchListOnly {
				repoResults, _ = repoIdx.ListFiltered(searchType, searchStatus, searchTag, searchSince)
			} else if searchByFile {
				repoResults, _ = repoIdx.SearchByFile(query)
			} else {
				repoResults, _ = repoIdx.Search(query)
			}
			repoIdx.Close()
			for i := range repoResults {
				repoResults[i].Slug = alias + "/" + repoResults[i].Slug
			}
			results = append(results, repoResults...)
		}
	}

	if len(results) == 0 {
		if searchJSON {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No results found.")
		return nil
	}

	if searchJSON {
		fmt.Print("[")
		for i, r := range results {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf(`{"type":%q,"key":%q,"title":%q,"status":%q,"snippet":%q}`,
				r.Type, r.Slug, r.Title, r.Status, r.Snippet)
		}
		fmt.Println("]")
		return nil
	}

	for _, r := range results {
		claimStr := ""
		if r.ClaimedBy != "" {
			claimStr = fmt.Sprintf("  [%s]", r.ClaimedBy)
		}
		domainStr := ""
		if r.Domain != "" && r.Domain != "engineering" {
			// Only surface non-engineering tags so engineering-only
			// workspaces see no visual change.
			domainStr = fmt.Sprintf("  {%s}", r.Domain)
		}
		fmt.Printf("%-30s  %-10s  %-10s  %s%s%s\n", r.Slug, r.Type, r.Status, r.Title, domainStr, claimStr)
		if r.Snippet != "" {
			fmt.Printf("  %s\n", r.Snippet)
		}
	}
	fmt.Printf("\n%d result(s)\n", len(results))
	return nil
}

// printFTSResults formats FTS5-sourced retrieval.Results in the tabular layout,
// or emits a JSON array when asJSON is true.
func printFTSResults(results []retrieval.Result, asJSON bool) error {
	if asJSON {
		fmt.Print("[")
		for i, r := range results {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf(`{"type":%q,"key":%q,"title":%q,"status":%q,"snippet":%q}`,
				r.Type, r.Key, r.Title, r.Status, r.Snippet)
		}
		fmt.Println("]")
		return nil
	}

	for _, r := range results {
		claimStr := ""
		if r.ClaimedBy != "" {
			claimStr = fmt.Sprintf("  [%s]", r.ClaimedBy)
		}
		fmt.Printf("%-30s  %-10s  %-10s  %s%s\n", r.Key, r.Type, r.Status, r.Title, claimStr)
		if r.Snippet != "" {
			fmt.Printf("  %s\n", r.Snippet)
		}
	}
	fmt.Printf("\n%d result(s)\n", len(results))
	return nil
}

// printGraphResults formats graph-sourced retrieval.Results in the markdown
// list layout, budget-limited to approximately the given token count.
// localRepo is the remote-origin key for this workspace; results with a
// different non-empty Repo get a "[repo]" label so cross-repo provenance
// is visible at a glance.
func printGraphResults(results []retrieval.Result, topic string, budget int, asJSON bool, localRepo string) error {
	if asJSON {
		fmt.Print("[")
		for i, r := range results {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf(`{"type":%q,"key":%q,"title":%q,"score":%v}`,
				r.Type, r.Key, r.Title, r.Score)
		}
		fmt.Println("]")
		return nil
	}

	fmt.Printf("<!-- hero search: %q, %d matches, budget=%d -->\n\n", topic, len(results), budget)
	used := 0
	dropped := 0
	for _, r := range results {
		line := fmt.Sprintf("- **%s** _(%s, `%s`)_", r.Title, r.Type, r.Key)
		if r.Repo != "" && r.Repo != localRepo {
			line += fmt.Sprintf(" [%s]", r.Repo)
		}
		if r.Snippet != "" {
			line += " — " + r.Snippet
		}
		tok := (len(line) + 3) / 4
		if used+tok > budget {
			dropped++
			continue
		}
		fmt.Println(line)
		used += tok
	}
	if dropped > 0 {
		fmt.Printf("\n_…+%d more — refine query or run `hero search --specs` for FTS5 spec-only results_\n", dropped)
	}
	return nil
}

func hasFilters() bool {
	return searchType != "" || searchStatus != "" || searchTag != "" || searchSince != ""
}
