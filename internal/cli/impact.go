package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/impact"
	"github.com/hero-engine/hero/internal/index"
	"github.com/spf13/cobra"
)

var impactCmd = &cobra.Command{
	Use:   "impact <file-path> [file-path...]",
	Short: "Analyze blast radius of changing a file",
	Long:  `Shows which specs, conventions, and decisions are affected if you change the given file(s).`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runImpact,
}

var (
	impactFormatJSON bool
	impactCrossRepo  bool
	impactType       string
)

func init() {
	impactCmd.Flags().BoolVar(&impactFormatJSON, "format", false, "output as JSON")
	impactCmd.Flags().Lookup("format").NoOptDefVal = "true"
	impactCmd.Flags().BoolVar(&impactCrossRepo, "cross-repo", false, "query Hero Cloud for cross-repo callers (federation phase 8)")
	impactCmd.Flags().StringVar(&impactType, "type", "Symbol", "node type for --cross-repo: Symbol, File, Package")
}

func runImpact(cmd *cobra.Command, args []string) error {
	// Cross-repo path: ask the team server for callers across the
	// whole org. Distinct workflow from the local spec-impact —
	// answers a different question ("who else is affected?").
	if impactCrossRepo {
		return runImpactCrossRepo(args)
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

	idx, err := index.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	reports, err := impact.Analyze(idx, args)
	if err != nil {
		return fmt.Errorf("analyzing impact: %w", err)
	}

	if impactFormatJSON {
		out, jsonErr := impact.RenderJSON(reports)
		if jsonErr != nil {
			return fmt.Errorf("rendering JSON: %w", jsonErr)
		}
		fmt.Println(out)
		return nil
	}

	fmt.Print(impact.RenderText(reports))

	// Code-graph blast-radius — additive section. Best-effort; if the
	// graph isn't populated this prints nothing and we don't fail.
	fmt.Println()
	if err := runCodeGraphImpact(args); err != nil {
		// Don't fail the whole impact report on graph absence.
		// Phase 7 federation extends this with cross-repo callers.
		fmt.Fprintf(os.Stderr, "(graph impact unavailable: %v)\n", err)
	}
	return nil
}
