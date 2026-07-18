package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/sitegen"
	"github.com/spf13/cobra"
)

var (
	pagesOutDir   string
	pagesSiteName string
)

var publishPagesCmd = &cobra.Command{
	Use:   "pages",
	Short: "Render the knowledge graph as a static HTML site (deployable to GitHub Pages)",
	Long: `Renders the unified knowledge graph as a self-contained static site:
landing page, per-feature/initiative/decision/note pages, full
activity feed. No JS framework, no build step — embedded templates
+ a single stylesheet produce HTML that deploys cleanly to GitHub
Pages, Netlify, S3, or anywhere else that serves static files.

By default writes to ./site. Re-running with the same graph state
produces byte-identical output, so gh-pages diffs stay readable.`,
	RunE: runPublishPages,
}

func init() {
	publishPagesCmd.Flags().StringVar(&pagesOutDir, "output", "site", "output directory for the generated site")
	publishPagesCmd.Flags().StringVar(&pagesSiteName, "site-name", "", "site name shown in header (default: repo basename)")
	publishCmd.AddCommand(publishPagesCmd)
}

func runPublishPages(cmd *cobra.Command, args []string) error {
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

	repoKey := graphRepoKey(projectRoot)
	gen := &sitegen.Generator{
		Store:    store,
		RepoKey:  repoKey,
		OutDir:   pagesOutDir,
		SiteName: pagesSiteName,
	}
	summary, err := gen.Generate()
	if err != nil {
		return fmt.Errorf("generating site: %w", err)
	}

	fmt.Printf("Wrote %s/ → index (%d), features (%d), initiatives (%d), decisions (%d), notes (%d), activity (%d)\n",
		pagesOutDir,
		summary.Index, summary.Features, summary.Initiatives,
		summary.Decisions, summary.Notes, summary.Activity,
	)
	fmt.Printf("Preview locally: open %s/index.html\n", pagesOutDir)
	fmt.Printf("Deploy: push %s/ to your gh-pages branch\n", pagesOutDir)
	return nil
}
