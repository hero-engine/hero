package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/wiki"
	"github.com/spf13/cobra"
)

var wikiSyncAll bool

var wikiSyncCmd = &cobra.Command{
	Use:   "wiki [spec-path]",
	Short: "Sync specs to the configured wiki target",
	Long: `Pushes spec content to an external wiki (e.g. GitHub Wiki).

With a spec path, syncs that single spec. With --all, syncs all completed specs.
Requires sync.target to be configured in hero.json.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWikiSync,
}

func init() {
	wikiSyncCmd.Flags().BoolVar(&wikiSyncAll, "all", false, "sync all completed specs")
}

func runWikiSync(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	if cfg.Sync == nil || cfg.Sync.Target == "none" || cfg.Sync.Target == "" {
		return fmt.Errorf("no wiki sync target configured — set sync.target in hero.json")
	}

	var syncer wiki.Syncer
	if cfg.Sync.Target == "confluence" {
		syncer, err = wiki.NewConfluence(cfg.Confluence)
	} else {
		syncer, err = wiki.New(cfg.Sync, cfg.Tracker)
	}
	if err != nil {
		return fmt.Errorf("initializing wiki syncer: %w", err)
	}

	if wikiSyncAll {
		return runWikiSyncAll(syncer, heroDir)
	}

	if len(args) == 0 {
		return fmt.Errorf("provide a spec path or use --all")
	}

	return runWikiSyncOne(syncer, args[0])
}

func runWikiSyncOne(syncer wiki.Syncer, specPath string) error {
	s, err := spec.ParseFile(specPath)
	if err != nil {
		return fmt.Errorf("parsing spec %s: %w", specPath, err)
	}

	content, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("reading spec %s: %w", specPath, err)
	}

	pageName, err := syncer.SyncSpec(s, string(content))
	if err != nil {
		return fmt.Errorf("syncing spec: %w", err)
	}

	fmt.Printf("Synced %s to %s page: %s\n", s.Slug, syncer.Name(), pageName)
	return nil
}

func runWikiSyncAll(syncer wiki.Syncer, heroDir string) error {
	allSpecs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	// Filter to completed specs only
	var completed []*spec.Spec
	for _, s := range allSpecs {
		if s.Status == spec.StatusCompleted || s.Status == spec.StatusAccepted || s.Status == spec.StatusActive {
			completed = append(completed, s)
		}
	}

	if len(completed) == 0 {
		fmt.Println("No completed specs to sync.")
		return nil
	}

	pages, err := syncer.SyncAll(completed)
	if err != nil {
		return fmt.Errorf("syncing specs: %w", err)
	}

	fmt.Printf("Synced %d spec(s) to %s:\n", len(pages), syncer.Name())
	for _, p := range pages {
		fmt.Printf("  %s\n", p)
	}
	return nil
}
