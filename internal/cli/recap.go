package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/recap"
	"github.com/spf13/cobra"
)

var recapCmd = &cobra.Command{
	Use:   "recap",
	Short: "Activity digest grouped by spec",
	Long:  `Shows recent git activity grouped by spec rather than by commit. Answers "what happened since yesterday?" with spec-level context.`,
	RunE:  runRecap,
}

var (
	recapSince      string
	recapFormat     string
	recapCrossRepo  bool
	recapSubproject string
)

func init() {
	recapCmd.Flags().StringVar(&recapSince, "since", "", "time window: duration (24h, 2d, 1w) or ISO date (YYYY-MM-DD). Default: 24h")
	recapCmd.Flags().StringVar(&recapFormat, "format", "", "output format: text (default) or json")
	recapCmd.Flags().BoolVar(&recapCrossRepo, "cross-repo", false, "aggregate activity from all configured repos")
	recapCmd.Flags().StringVar(&recapSubproject, "subproject", "", "filter to specs in this subproject scope; 'all' disables. Default: active scope from cwd")
}

func runRecap(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	since, err := recap.ParseSince(recapSince)
	if err != nil {
		return err
	}

	r, err := recap.Build(heroDir, projectRoot, since)
	if err != nil {
		return fmt.Errorf("building recap: %w", err)
	}

	// Cross-repo: aggregate recaps from all configured repos
	if recapCrossRepo {
		repos := cfg.ResolveAllRepos(projectRoot)
		for alias, rs := range repos {
			if !rs.Accessible {
				continue
			}
			repoHeroDir := cfg.HeroDir(rs.Path)
			repoRecap, buildErr := recap.Build(repoHeroDir, rs.Path, since)
			if buildErr != nil {
				continue
			}
			for i := range repoRecap.Specs {
				repoRecap.Specs[i].Slug = alias + "/" + repoRecap.Specs[i].Slug
			}
			r.Specs = append(r.Specs, repoRecap.Specs...)
			r.Knowledge = append(r.Knowledge, repoRecap.Knowledge...)
			r.Unmatched = append(r.Unmatched, repoRecap.Unmatched...)
		}
	}

	// Filter by subproject — explicit flag wins, else default to cwd scope.
	subproject := resolveSubprojectFilter(recapSubproject)
	maybePrintScopeHint(cmd.ErrOrStderr(), recapSubproject, subproject)
	if subproject != "" && subproject != "all" {
		filtered := r.Specs[:0]
		for _, sa := range r.Specs {
			if sa.Subproject == subproject {
				filtered = append(filtered, sa)
			}
		}
		r.Specs = filtered
	}

	switch recapFormat {
	case "json":
		out, jsonErr := recap.RenderJSON(r)
		if jsonErr != nil {
			return fmt.Errorf("rendering JSON: %w", jsonErr)
		}
		fmt.Println(out)
	default:
		fmt.Print(recap.RenderText(r))
	}

	return nil
}
