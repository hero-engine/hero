package cli

import (
	"database/sql"
	"fmt"
	"os"

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

func init() {
	embeddingsCmd.AddCommand(embeddingsStatusCmd)
	embeddingsCmd.AddCommand(embeddingsRebuildCmd)
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
