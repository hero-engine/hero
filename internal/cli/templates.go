package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/templates"
	"github.com/spf13/cobra"
)

var templatesForce bool

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Show learned spec authoring patterns",
	Long: `Analyzes completed specs to discover section structures, criteria
density, and frontmatter patterns per spec type.

Subcommands:
  hero templates               — summary table of discovered patterns
  hero templates show <type>   — full detail for one type
  hero templates refresh       — re-analyze corpus and write pattern files`,
	RunE: runTemplatesList,
}

var templatesShowCmd = &cobra.Command{
	Use:   "show <type>",
	Short: "Show full pattern detail for a spec type",
	Args:  cobra.ExactArgs(1),
	RunE:  runTemplatesShow,
}

var templatesRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Re-analyze the spec corpus and generate learned templates",
	RunE:  runTemplatesRefresh,
}

func init() {
	templatesRefreshCmd.Flags().BoolVar(&templatesForce, "force", false, "regenerate even if corpus is below threshold")
	templatesCmd.AddCommand(templatesShowCmd)
	templatesCmd.AddCommand(templatesRefreshCmd)
}

func runTemplatesList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	patterns, err := templates.AnalyzeCorpus(heroDir)
	if err != nil {
		return err
	}

	if len(patterns) == 0 {
		fmt.Println("No completed specs found.")
		return nil
	}

	total := 0
	for _, p := range patterns {
		total += p.CorpusSize
	}
	fmt.Printf("Learned templates (from %d completed specs):\n\n", total)

	for _, t := range []spec.Type{"feature", "bug", "convention", "decision", "initiative"} {
		p, ok := patterns[t]
		if !ok {
			continue
		}
		if p.CorpusSize < templates.MinCorpusSize {
			fmt.Printf("  %-14s %2d specs  — below threshold (%d), using static template\n", t, p.CorpusSize, templates.MinCorpusSize)
		} else {
			sectionsAbove := 0
			for _, s := range p.Sections {
				if s.Frequency >= templates.SectionThreshold {
					sectionsAbove++
				}
			}
			criteriaStr := fmt.Sprintf("avg %.1f criteria", p.CriteriaCount.Mean)
			if p.CriteriaCount.Max == 0 {
				criteriaStr = "(no criteria)"
			}
			earsStr := ""
			if p.EARSRatio > 0 {
				earsStr = fmt.Sprintf("  %.0f%% EARS", p.EARSRatio*100)
			}
			fmt.Printf("  %-14s %2d specs  %d sections  %s%s\n", t, p.CorpusSize, sectionsAbove, criteriaStr, earsStr)
		}
	}

	return nil
}

func runTemplatesShow(cmd *cobra.Command, args []string) error {
	specType := spec.Type(args[0])
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	patterns, err := templates.AnalyzeCorpus(heroDir)
	if err != nil {
		return err
	}

	p, ok := patterns[specType]
	if !ok {
		return fmt.Errorf("no completed specs of type %q found", specType)
	}

	if p.CorpusSize < templates.MinCorpusSize {
		fmt.Printf("Type %q has only %d completed spec(s) — below threshold of %d.\n", specType, p.CorpusSize, templates.MinCorpusSize)
		return nil
	}

	fmt.Print(templates.GenerateLearnedTemplate(p))
	return nil
}

func runTemplatesRefresh(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	patterns, err := templates.AnalyzeCorpus(heroDir)
	if err != nil {
		return err
	}

	knowledgeDir := cfg.KnowledgeDir(projectRoot)
	written, err := templates.WritePatternFiles(knowledgeDir, patterns)
	if err != nil {
		return err
	}

	if written == 0 {
		fmt.Println("No spec types have enough completed specs to generate learned templates.")
		fmt.Printf("Need at least %d completed specs per type.\n", templates.MinCorpusSize)
	} else {
		fmt.Printf("Generated %d learned template(s) in %s/templates/\n", written, knowledgeDir)
	}

	return nil
}
