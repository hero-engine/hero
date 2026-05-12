package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/drift"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var driftCmd = &cobra.Command{
	Use:   "drift [slug]",
	Short: "Detect drift between spec and code",
	Long:  `Reports when implementation has diverged from what a spec promised — missing files, renamed files, unaddressed criteria, and boundary violations.`,
	RunE:  runDrift,
}

var (
	driftInFlight  bool
	driftInit      string
	driftSince     string
	driftFormatJSON bool
)

func init() {
	driftCmd.Flags().BoolVar(&driftInFlight, "in-flight", false, "analyze all delivering specs")
	driftCmd.Flags().StringVar(&driftInit, "initiative", "", "analyze all child specs of an initiative")
	driftCmd.Flags().StringVar(&driftSince, "since", "", "only count drift since this git ref")
	driftCmd.Flags().BoolVar(&driftFormatJSON, "format", false, "output as JSON (--format json)")

	// Support --format json as a string flag
	driftCmd.Flags().Lookup("format").NoOptDefVal = "true"
}

func runDrift(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Open index for convention drift checking (best-effort)
	idx, _ := index.Open(heroDir)
	if idx != nil {
		defer idx.Close()
	}

	var reports []*drift.Report

	switch {
	case driftInFlight:
		reports, err = drift.AnalyzeAllWithIndex(heroDir, projectRoot, driftSince, idx)
		if err != nil {
			return fmt.Errorf("analyzing in-flight specs: %w", err)
		}

	case driftInit != "":
		reports, err = drift.AnalyzeInitiativeWithIndex(heroDir, projectRoot, driftInit, driftSince, idx)
		if err != nil {
			return fmt.Errorf("analyzing initiative: %w", err)
		}

	default:
		if len(args) == 0 {
			return fmt.Errorf("provide a spec slug, or use --in-flight / --initiative <id>")
		}
		slug := args[0]
		s, findErr := findSpecBySlugOrPath(heroDir, slug)
		if findErr != nil {
			return findErr
		}
		reports = []*drift.Report{drift.AnalyzeWithIndex(s, projectRoot, driftSince, idx)}
	}

	if driftFormatJSON {
		out, jsonErr := drift.RenderJSON(reports)
		if jsonErr != nil {
			return fmt.Errorf("rendering JSON: %w", jsonErr)
		}
		fmt.Println(out)
	} else {
		fmt.Print(drift.RenderText(reports))
	}

	exitCode := drift.AggregateExitCode(reports)
	if exitCode > 0 {
		os.Exit(exitCode)
	}
	return nil
}

func findSpecBySlugOrPath(heroDir, slug string) (*spec.Spec, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	for _, s := range specs {
		if s.Slug == slug {
			return s, nil
		}
	}

	// Try direct path
	direct := filepath.Join(heroDir, "planning", "features", slug, "spec.md")
	if s, err := spec.ParseFile(direct); err == nil {
		return s, nil
	}
	direct = filepath.Join(heroDir, "planning", "bugs", slug, "spec.md")
	if s, err := spec.ParseFile(direct); err == nil {
		return s, nil
	}

	return nil, fmt.Errorf("spec %q not found", slug)
}
