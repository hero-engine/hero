package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan <slug>",
	Short: "View or manage execution plans for specs",
	Long:  `View the execution plan attached to a spec, if one exists. Plans are created by agents via hero_plan or by /deliver --dry-run.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPlan,
}

func init() {}

func runPlan(cmd *cobra.Command, args []string) error {
	slug := args[0]
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)

	// Find the spec
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	for _, s := range specs {
		if s.Slug == slug {
			planPath := filepath.Join(filepath.Dir(s.Path), "plan.md")
			data, err := os.ReadFile(planPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("No plan exists for %q. One will be created when an agent runs /deliver --dry-run or calls hero_plan.\n", slug)
					return nil
				}
				return fmt.Errorf("reading plan: %w", err)
			}
			fmt.Print(string(data))
			return nil
		}
	}

	return fmt.Errorf("spec %q not found", slug)
}
