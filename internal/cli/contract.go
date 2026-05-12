package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/contract"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var contractCmd = &cobra.Command{
	Use:   "contract <slug>",
	Short: "Living contract — criteria-to-test traceability",
	Long:  `Show contract status, link criteria to tests, or check for regressions.`,
	RunE:  runContractStatus,
}

var contractLinkCmd = &cobra.Command{
	Use:   "link <slug> <criterion-index> <file>::<testname>",
	Short: "Link a criterion to a test",
	Args:  cobra.ExactArgs(3),
	RunE:  runContractLink,
}

var contractCheckCmd = &cobra.Command{
	Use:   "check [--slug <slug>]",
	Short: "Run linked tests and report regressions",
	RunE:  runContractCheck,
}

var contractCheckSlug string

func init() {
	contractCheckCmd.Flags().StringVar(&contractCheckSlug, "slug", "", "check only this spec")
	contractCmd.AddCommand(contractLinkCmd)
	contractCmd.AddCommand(contractCheckCmd)
}

func runContractStatus(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provide a spec slug")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	s, err := findSpecBySlugOrPath(heroDir, args[0])
	if err != nil {
		return err
	}

	report := contract.Status(s)
	fmt.Print(contract.RenderText(report))

	if report.Linked < report.Total {
		os.Exit(1)
	}
	return nil
}

func runContractLink(cmd *cobra.Command, args []string) error {
	slug := args[0]
	idxStr := args[1]
	testRef := args[2]

	criterionIdx, err := strconv.Atoi(idxStr)
	if err != nil {
		return fmt.Errorf("criterion index must be a number, got %q", idxStr)
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	s, err := findSpecBySlugOrPath(heroDir, slug)
	if err != nil {
		return err
	}

	if err := contract.Link(s.Path, projectRoot, criterionIdx, testRef); err != nil {
		return err
	}

	fmt.Printf("Linked criterion %d to %s\n", criterionIdx, testRef)
	return nil
}

func runContractCheck(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)

	var allResults []contract.RegressionResult

	if contractCheckSlug != "" {
		s, err := findSpecBySlugOrPath(heroDir, contractCheckSlug)
		if err != nil {
			return err
		}
		allResults = contract.Check(s, projectRoot)
	} else {
		specs, err := spec.Discover(heroDir)
		if err != nil {
			return fmt.Errorf("discovering specs: %w", err)
		}
		for _, s := range specs {
			if s.Status != spec.StatusCompleted {
				continue
			}
			results := contract.Check(s, projectRoot)
			allResults = append(allResults, results...)
		}
	}

	if cmd.Flags().Changed("format") {
		data, _ := json.MarshalIndent(allResults, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Print(contract.RenderCheckText(allResults))
	}

	for _, r := range allResults {
		if !r.Passed {
			os.Exit(1)
		}
	}
	return nil
}
