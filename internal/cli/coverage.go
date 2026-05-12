package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/coverage"
	"github.com/spf13/cobra"
)

var (
	coverageAll     bool
	coverageFormat  string
	coverageTestDir string
)

var coverageCmd = &cobra.Command{
	Use:   "coverage [slug]",
	Short: "Report which acceptance criteria have test coverage",
	Long: `Analyzes a spec's acceptance criteria against test files in the project.
For each criterion, extracts keywords and searches test files for matching
test names, assertion text, and describe-block labels.

All analysis is local and heuristic — no LLM calls, no test execution.

Examples:
  hero coverage csv-export         # one spec
  hero coverage --all              # all specs with criteria
  hero coverage csv-export --format json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCoverage,
}

func init() {
	coverageCmd.Flags().BoolVar(&coverageAll, "all", false, "analyze all specs with acceptance criteria")
	coverageCmd.Flags().StringVar(&coverageFormat, "format", "", "output format: text (default), json")
	coverageCmd.Flags().StringVar(&coverageTestDir, "test-dir", "", "override test file discovery root")
}

func runCoverage(cmd *cobra.Command, args []string) error {
	if !coverageAll && len(args) == 0 {
		return fmt.Errorf("specify a spec slug or use --all")
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

	if coverageAll {
		reports, err := coverage.AnalyzeAll(projectRoot, heroDir, coverageTestDir)
		if err != nil {
			return err
		}
		if len(reports) == 0 {
			fmt.Println("No specs with acceptance criteria found.")
			return nil
		}

		hasGaps := false
		for _, r := range reports {
			if coverageFormat == "json" {
				out, err := coverage.FormatJSON(r)
				if err != nil {
					return err
				}
				fmt.Println(out)
			} else {
				fmt.Print(coverage.FormatText(r))
				fmt.Println()
			}
			if r.Gaps > 0 {
				hasGaps = true
			}
		}

		if hasGaps {
			os.Exit(1)
		}
		return nil
	}

	slug := args[0]
	r, err := coverage.Analyze(projectRoot, heroDir, slug, coverageTestDir)
	if err != nil {
		return err
	}

	if coverageFormat == "json" {
		out, err := coverage.FormatJSON(r)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		fmt.Print(coverage.FormatText(r))
	}

	if r.ExitCode != 0 {
		os.Exit(r.ExitCode)
	}
	return nil
}
