package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/embeddings"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/index"
	"github.com/spf13/cobra"
)

var embeddingsCmd = &cobra.Command{
	Use:   "embeddings",
	Short: "Manage the semantic embeddings index",
	Long: `View status or rebuild the vector embedding index used for hybrid
(BM25 + semantic) search. The index lives inside index.db alongside
the FTS5 tables.

Subcommands:
  status    Print chunk counts per corpus, index size, model availability
  refresh   Refresh only missing or changed chunks
  rebuild   Full re-embed (wipe and rebuild from scratch)`,
}

var embeddingsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print embedding index status",
	RunE:  runEmbeddingsStatus,
}

var embeddingsRebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Wipe and rebuild the embedding index from scratch",
	RunE:  runEmbeddingsRebuild,
}

var (
	embeddingsRefreshDeadline time.Duration
	embeddingsRefreshQuiet    bool
)

var embeddingsRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh missing or changed embedding chunks",
	Long: `Refresh missing or changed embedding chunks.

Refresh is intrinsically stale-only: --if-stale is accepted to make hook and
automation intent explicit, but omitting it performs the same hash-aware pass.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !embeddingsRefreshQuiet && rootCmd.PersistentPreRun != nil {
			rootCmd.PersistentPreRun(cmd, args)
		}
	},
	RunE: runEmbeddingsRefresh,
}

func init() {
	embeddingsCmd.AddCommand(embeddingsStatusCmd)
	embeddingsCmd.AddCommand(embeddingsRefreshCmd)
	embeddingsCmd.AddCommand(embeddingsRebuildCmd)
	embeddingsRefreshCmd.Flags().Bool("if-stale", false, "refresh only missing or changed chunks (refresh is always stale-only)")
	embeddingsRefreshCmd.Flags().DurationVar(&embeddingsRefreshDeadline, "deadline", defaultIncrementalScanDeadline, "maximum refresh duration")
	embeddingsRefreshCmd.Flags().BoolVarP(&embeddingsRefreshQuiet, "quiet", "q", false, "suppress output and normalize failures for hooks")
}

func runEmbeddingsStatus(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Config status.
	fmt.Printf("Embeddings enabled: %v\n", cfg.IsEmbeddingsEnabled())
	fmt.Printf("Model: %s\n", cfg.EmbeddingsModel())
	fmt.Printf("Scope: %v\n", cfg.EmbeddingsScope())

	// Model availability.
	model, err := embeddings.LoadModelFromConfig(cfg.EmbeddingsModel())
	if err != nil {
		fmt.Printf("Model status: error loading (%v)\n", err)
	} else if model == nil {
		fmt.Printf("Model status: not installed (install to ~/.hero/models/embeddings/%s/)\n", cfg.EmbeddingsModel())
	} else {
		fmt.Printf("Model status: loaded (dim=%d)\n", model.Dim())
	}

	// Index stats.
	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	store, err := embeddings.OpenStorage(idx.RawDB())
	if err != nil {
		return fmt.Errorf("opening embedding storage: %w", err)
	}

	stats, err := store.Stats()
	if err != nil {
		return fmt.Errorf("querying stats: %w", err)
	}

	fmt.Printf("\nIndex stats:\n")
	fmt.Printf("  Total chunks: %d\n", stats.Total)
	for corpus, count := range stats.ByCorpus {
		fmt.Printf("  %-14s %d\n", corpus+":", count)
	}

	return nil
}

func runEmbeddingsRebuild(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Load the embedding model.
	model, err := embeddings.LoadModelFromConfig(cfg.EmbeddingsModel())
	if err != nil {
		return fmt.Errorf("loading embedding model: %w", err)
	}
	if model == nil {
		return fmt.Errorf("no embedding model installed at ~/.hero/models/embeddings/%s/ -- cannot rebuild", cfg.EmbeddingsModel())
	}

	// Open index.db.
	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	indexDB := idx.RawDB()

	// Wipe existing chunks. Table may not exist yet (OpenStorage creates it
	// during Refresh), so ignore errors here.
	_, _ = indexDB.Exec("DELETE FROM vec_chunks")

	// Open graph.db for event/code corpora (best-effort).
	var graphDB *sql.DB
	if gstore, gErr := graph.Open(heroDir); gErr == nil {
		graphDB = gstore.DB()
		defer gstore.Close()
	}

	scope := cfg.EmbeddingsScope()
	fmt.Printf("Rebuilding embeddings (model=%s, scope=%v)...\n", cfg.EmbeddingsModel(), scope)

	stats, err := embeddings.Refresh(heroDir, model, indexDB, graphDB, scope)
	if err != nil {
		return fmt.Errorf("embedding refresh failed: %w", err)
	}

	fmt.Printf("Rebuild complete: %s\n", stats)
	return nil
}

type embeddingPhaseOutcome struct {
	Stats       *embeddings.RefreshStats
	Skipped     bool
	Unavailable bool
	Reason      string
}

func runEmbeddingPhase(
	ctx context.Context,
	cfg config.Config,
	heroDir string,
	indexDB *sql.DB,
	graphDB *sql.DB,
) (embeddingPhaseOutcome, error) {
	if !cfg.IsEmbeddingsEnabled() {
		return embeddingPhaseOutcome{Skipped: true, Reason: "embeddings are disabled"}, nil
	}

	model, err := embeddings.LoadModelFromConfig(cfg.EmbeddingsModel())
	if err != nil {
		return embeddingPhaseOutcome{}, fmt.Errorf("loading embedding model: %w", err)
	}
	if model == nil {
		outcome := embeddingPhaseOutcome{
			Unavailable: true,
			Reason:      fmt.Sprintf("embedding model %q is unavailable", cfg.EmbeddingsModel()),
		}
		return outcome, fmt.Errorf("%s", outcome.Reason)
	}

	stats, err := embeddings.RefreshContext(ctx, heroDir, model, indexDB, graphDB, cfg.EmbeddingsScope())
	outcome := embeddingPhaseOutcome{Stats: stats}
	if err != nil {
		return outcome, err
	}
	if stats.Unavailable {
		var unavailable []string
		for corpus, corpusStats := range stats.Corpora {
			if corpusStats.Outcome == "unavailable" {
				unavailable = append(unavailable, corpus)
			}
		}
		sort.Strings(unavailable)
		outcome.Unavailable = true
		outcome.Reason = fmt.Sprintf("embedding source unavailable for corpus: %s", strings.Join(unavailable, ", "))
		return outcome, fmt.Errorf("%s", outcome.Reason)
	}
	return outcome, nil
}

func runWorkspaceEmbeddingPhase(ctx context.Context, cfg config.Config, heroDir string) (embeddingPhaseOutcome, error) {
	if !cfg.IsEmbeddingsEnabled() {
		return embeddingPhaseOutcome{Skipped: true, Reason: "embeddings are disabled"}, nil
	}

	indexPath := filepath.Join(heroDir, index.IndexFileName)
	var indexDB *sql.DB
	if _, err := os.Stat(indexPath); err == nil {
		indexDB, err = sql.Open("sqlite", indexPath+"?_pragma=busy_timeout(5000)")
		if err != nil {
			return embeddingPhaseOutcome{}, fmt.Errorf("opening search index: %w", err)
		}
	} else if os.IsNotExist(err) {
		idx, openErr := index.Open(heroDir)
		if openErr != nil {
			return embeddingPhaseOutcome{}, fmt.Errorf("opening search index: %w", openErr)
		}
		indexDB = idx.RawDB()
	} else {
		return embeddingPhaseOutcome{}, fmt.Errorf("stat search index: %w", err)
	}
	defer indexDB.Close()

	var graphDB *sql.DB
	var graphStore *graph.Store
	if scopeNeedsGraph(cfg.EmbeddingsScope()) {
		if _, err := os.Stat(filepath.Join(heroDir, graph.FileName)); err == nil {
			graphStore, err = graph.Open(heroDir)
			if err != nil {
				return embeddingPhaseOutcome{}, fmt.Errorf("opening graph: %w", err)
			}
			defer graphStore.Close()
			graphDB = graphStore.DB()
		}
	}
	return runEmbeddingPhase(ctx, cfg, heroDir, indexDB, graphDB)
}

func scopeNeedsGraph(scope []string) bool {
	for _, corpus := range scope {
		if corpus == "event" || corpus == "code" {
			return true
		}
	}
	return false
}

func runEmbeddingsRefresh(cmd *cobra.Command, args []string) (retErr error) {
	if embeddingsRefreshQuiet {
		defer func() {
			if retErr != nil {
				retErr = nil
			}
		}()
	}
	if embeddingsRefreshDeadline <= 0 {
		return fmt.Errorf("embedding refresh deadline must be greater than zero")
	}
	// RefreshContext always compares stored hashes before embedding. Read the
	// flag so the accepted hook contract remains explicit even though there is
	// no force-reembed mode on this command.
	if _, err := cmd.Flags().GetBool("if-stale"); err != nil {
		return fmt.Errorf("reading --if-stale: %w", err)
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

	ctx, cancel := context.WithTimeout(cmd.Context(), embeddingsRefreshDeadline)
	defer cancel()
	outcome, err := runWorkspaceEmbeddingPhase(ctx, cfg, heroDir)
	if err != nil {
		return fmt.Errorf("embedding refresh failed: %w", err)
	}
	if embeddingsRefreshQuiet {
		return nil
	}
	if outcome.Skipped {
		fmt.Fprintf(cmd.OutOrStdout(), "Embedding refresh skipped: %s\n", outcome.Reason)
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Embedding refresh complete: %s\n", outcome.Stats)
	return nil
}
