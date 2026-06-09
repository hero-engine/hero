package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/index"
	"github.com/spf13/cobra"
)

var conflictsCmd = &cobra.Command{
	Use:   "conflicts <spec-slug>",
	Short: "Check for overlapping in-flight specs and graph-level divergence",
	Long: `Finds other in-flight specs (planning, in-review, or delivering) that touch
the same files as the given spec. Also checks the knowledge graph for nodes
where two teammates pushed different versions of the same entity (graph-level
divergence). Use this before starting delivery to avoid conflicting work.`,
	Args: cobra.ExactArgs(1),
	RunE: runConflicts,
}

func runConflicts(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	slug := args[0]
	found := false

	// --- Spec-level conflicts (file overlap) ---
	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	specConflicts, err := idx.FindConflicts(slug)
	idx.Close()
	if err != nil {
		return fmt.Errorf("finding spec conflicts: %w", err)
	}
	if len(specConflicts) > 0 {
		found = true
		fmt.Printf("Spec conflicts for %s (overlapping files):\n\n", slug)
		for _, c := range specConflicts {
			claimStr := ""
			if c.ClaimedBy != "" {
				claimStr = fmt.Sprintf("  [claimed by %s]", c.ClaimedBy)
			}
			fmt.Printf("  %-30s  %-10s  %-10s  %s%s\n", c.Slug, c.Type, c.Status, c.Title, claimStr)
			fmt.Printf("    overlapping files: %s\n", strings.Join(c.OverlappingFiles, ", "))
		}
		fmt.Printf("\n%d conflict(s) (spec-level) — coordinate before proceeding.\n\n", len(specConflicts))
	}

	// --- Graph-level conflicts (cached from last push response) ---
	if records := loadPushConflictsForSlug(heroDir, slug); len(records) > 0 {
		found = true
		fmt.Printf("Graph conflicts for %s (concurrent edits detected at push time):\n\n", slug)
		for _, r := range records {
			fmt.Printf("  %s %s — %s  (detected %s)\n", r.NodeType, r.NodeKey, r.Reason, r.DetectedAt)
		}
		fmt.Printf("\n%d conflict(s) (graph-level) — run 'hero sync graph pull && hero scan' to reconcile.\n", len(records))
	}

	// --- Graph-level conflicts (bitemporal history — local graph.db) ---
	if store, gerr := graph.Open(heroDir); gerr == nil {
		if gconflicts, gerr := store.FindGraphConflicts(slug); gerr == nil && len(gconflicts) > 0 {
			found = true
			fmt.Printf("\nGraph divergence for %s (multiple clients pushed different versions):\n\n", slug)
			for _, c := range gconflicts {
				fmt.Printf("  %s %s — %d versions from %d client(s):\n", c.NodeType, c.NodeKey, len(c.Versions), countDistinctClients(c.Versions))
				for _, v := range c.Versions {
					cur := ""
					if v.Current {
						cur = " [current]"
					}
					fmt.Printf("    client %s: status=%s at %s%s\n", v.ClientID, v.Status, v.ValidFrom.Format("2006-01-02 15:04"), cur)
				}
			}
			fmt.Printf("\n%d divergent node(s) — run 'hero sync graph pull && hero scan' to reconcile.\n", len(gconflicts))
		}
		store.Close()
	}

	if !found {
		fmt.Printf("No conflicts found for %s.\n", slug)
	}
	return nil
}

func countDistinctClients(versions []graph.GraphConflictVersion) int {
	seen := map[string]bool{}
	for _, v := range versions {
		seen[v.ClientID] = true
	}
	return len(seen)
}

// loadPushConflictsForSlug reads .hero/push_conflicts.json and returns
// records whose node_key matches the slug exactly or as a suffix.
func loadPushConflictsForSlug(heroDir, slug string) []pushConflictRecord {
	data, err := os.ReadFile(filepath.Join(heroDir, "push_conflicts.json"))
	if err != nil {
		return nil
	}
	var all []pushConflictRecord
	if err := json.Unmarshal(data, &all); err != nil {
		return nil
	}
	var matched []pushConflictRecord
	for _, r := range all {
		if r.NodeKey == slug || strings.HasSuffix(r.NodeKey, ":"+slug) {
			matched = append(matched, r)
		}
	}
	return matched
}
