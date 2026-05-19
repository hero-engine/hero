package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Graph schema migration tooling",
	Long: `Inspect and revert graph schema migrations.

Subverbs:
  schema rollback v3   revert the domain-scoped-knowledge-graph migration`,
}

var schemaRollbackCmd = &cobra.Command{
	Use:   "rollback <version>",
	Short: "Revert the named schema migration",
	Args:  cobra.ExactArgs(1),
	RunE:  runSchemaRollback,
}

var schemaRollbackDryRun bool

func init() {
	schemaCmd.AddCommand(schemaRollbackCmd)
	adminCmd.AddCommand(schemaCmd)
	schemaRollbackCmd.Flags().BoolVar(&schemaRollbackDryRun, "dry-run", false, "report what would be discarded without changing the database")
}

func runSchemaRollback(cmd *cobra.Command, args []string) error {
	version := args[0]
	if version != "v3" {
		return fmt.Errorf("rollback: only v3 is supported (got %q)", version)
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

	nodes, edges, err := store.NonEngineeringRowCount()
	if err != nil {
		return fmt.Errorf("counting non-engineering rows: %w", err)
	}

	if schemaRollbackDryRun {
		fmt.Println("Schema rollback v3 (dry-run)")
		fmt.Println()
		fmt.Println("Would drop the `domain` column from nodes and edges.")
		fmt.Printf("Non-engineering rows that would lose their tag:\n")
		fmt.Printf("  nodes: %d\n", nodes)
		fmt.Printf("  edges: %d\n", edges)
		fmt.Println()
		fmt.Println("Re-run without --dry-run to apply.")
		return nil
	}

	if nodes > 0 || edges > 0 {
		fmt.Printf("Warning: %d nodes and %d edges currently carry a non-engineering domain.\n", nodes, edges)
		fmt.Println("Their domain tags will be discarded by the rollback.")
	}

	if err := store.RollbackV3(); err != nil {
		return fmt.Errorf("rollback: %w", err)
	}
	fmt.Println("Rolled back schema to v2.")
	fmt.Println("The binary still expects v3 — re-open will re-apply the migration.")
	return nil
}
