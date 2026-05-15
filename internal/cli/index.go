package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	indexIfStale bool
	indexQuiet   bool
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Rebuild the spec corpus index",
	Long: `Re-indexes specs and knowledge entries in the hero workspace.

By default, runs a full rebuild — clears the index and re-walks every
spec from disk. Use --if-stale (-s) to do a surgical sync instead:
diff disk against the index and only re-index specs whose mtime is
newer than the stored stamp (plus add new specs and remove orphans).
Surgical sync is what skills, hooks, and read-side tools call.`,
	RunE: runIndex,
}

func init() {
	indexCmd.Flags().BoolVarP(&indexIfStale, "if-stale", "s", false, "diff disk against the index and only update what's drifted (cheap)")
	indexCmd.Flags().BoolVarP(&indexQuiet, "quiet", "q", false, "suppress non-error output (for hooks)")
}

func runIndex(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if indexIfStale {
		stats, err := index.RefreshIfStale(heroDir)
		if err != nil {
			return fmt.Errorf("refreshing index: %w", err)
		}
		// Regenerate the peer manifest after a stale refresh too —
		// convention edits may have changed publish status.
		if mErr := peering.GenerateAndWriteManifest(projectRoot); mErr != nil && !indexQuiet {
			fmt.Fprintf(os.Stderr, "  warning: peer manifest regen failed: %v\n", mErr)
		}
		if !indexQuiet {
			if stats.IsClean() {
				fmt.Printf("Index is current (%d specs scanned in %dms)\n", stats.Scanned, stats.DurationMS)
			} else {
				fmt.Printf("Indexed %d, Updated %d, Removed %d (%d scanned in %dms)\n",
					stats.Indexed, stats.Updated, stats.Removed, stats.Scanned, stats.DurationMS)
			}
		}
		return nil
	}

	if !indexQuiet {
		fmt.Printf("Indexing specs in %s...\n", heroDir)
	}

	stats, err := index.Rebuild(heroDir)
	if err != nil {
		return fmt.Errorf("rebuilding index: %w", err)
	}

	// Regenerate the peer manifest as part of the full rebuild.
	if mErr := peering.GenerateAndWriteManifest(projectRoot); mErr != nil && !indexQuiet {
		fmt.Fprintf(os.Stderr, "  warning: peer manifest regen failed: %v\n", mErr)
	}

	if indexQuiet {
		return nil
	}

	fmt.Printf("Indexed %d specs (%d features, %d bugs, %d conventions, %d decisions, %d initiatives, %d rules, %d external, %d context)\n",
		stats.TotalSpecs, stats.Features, stats.Bugs, stats.Conventions, stats.Decisions, stats.Initiatives,
		stats.Rules, stats.External, stats.Context)
	fmt.Printf("  %d planning, %d in-review, %d delivering, %d completed\n",
		stats.Planning, stats.InReview, stats.Delivering, stats.Completed)
	if stats.Active > 0 || stats.Accepted > 0 {
		fmt.Printf("  %d active conventions, %d accepted decisions\n", stats.Active, stats.Accepted)
	}
	fmt.Printf("  %d files tracked, %d approach docs, %d root causes\n",
		stats.FilesTracked, stats.DecisionDocs, stats.RootCauses)
	if stats.Claims > 0 {
		fmt.Printf("  %d claimed specs\n", stats.Claims)
	}

	// Subproject scope coverage — only meaningful when the workspace
	// actually declares subprojects.
	if missing, total := countSpecsMissingScope(heroDir); total > 0 && missing > 0 {
		fmt.Printf("  %d/%d specs lack a subproject: declaration\n", missing, total)
		fmt.Printf("  Run 'hero spec stamp-scope --all' to back-fill, or '--from-cwd <slug>' for a single spec.\n")
	}

	return nil
}

// countSpecsMissingScope returns (missing, total) where total counts
// specs whose path lives under a declared subproject prefix and missing
// counts those of those whose Subproject frontmatter is empty. Returns
// (0, 0) when no subprojects are declared.
func countSpecsMissingScope(heroDir string) (missing, total int) {
	projectRoot := filepath.Dir(heroDir)
	subs, err := install.LoadSubprojects(heroDir)
	if err != nil || subs == nil || len(subs.Subprojects) == 0 {
		return 0, 0
	}
	declared := subs.DeclaredPaths()
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return 0, 0
	}
	for _, s := range specs {
		// Only count specs that live under a declared subproject path.
		dir := filepath.Dir(s.Path)
		scope := workspace.MatchScope(projectRoot, dir, declared)
		if scope == workspace.RootScope {
			continue
		}
		total++
		if s.Subproject == "" {
			missing++
		}
	}
	return missing, total
}
