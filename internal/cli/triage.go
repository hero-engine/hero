package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/triage"
	"github.com/spf13/cobra"
)

var triageCmd = &cobra.Command{
	Use:   "triage [slug]",
	Short: "Check specs for duplicates, conflicts, and structural issues",
	Long: `Runs structural, duplicate, conflict, and orphan-relation checks on specs.

Without a slug argument, every spec in the workspace is triaged.
With a slug argument, only that spec is triaged against the rest of the corpus.

Exit codes:
  0  No issues
  1  One or more errors found
  2  Warnings only (when --no-prompt is set)`,
	RunE: runTriage,
}

var (
	triageFix      bool
	triageNoPrompt bool
	triageJSON     bool
)

func init() {
	triageCmd.Flags().BoolVar(&triageFix, "fix", false, "auto-fix fixable issues (adds missing created date from file mtime)")
	triageCmd.Flags().BoolVar(&triageNoPrompt, "no-prompt", false, "non-interactive mode; exit 2 on warnings-only")
	triageCmd.Flags().BoolVar(&triageJSON, "json", false, "output results as JSON")
}

func runTriage(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	allSpecs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	if len(allSpecs) == 0 {
		fmt.Println("No specs found.")
		return nil
	}

	opts := triage.Options{
		DuplicateThreshold: 0.80,
		TagOverlapMin:      3,
	}

	var results []triage.Result

	if len(args) > 0 {
		// Triage a single named spec
		slug := args[0]
		var candidate *spec.Spec
		var corpus []*spec.Spec
		for _, s := range allSpecs {
			if s.Slug == slug {
				candidate = s
			} else {
				corpus = append(corpus, s)
			}
		}
		if candidate == nil {
			return fmt.Errorf("spec %q not found", slug)
		}
		results = append(results, triage.Triage(candidate, corpus, opts))
	} else {
		// Triage every spec against the rest
		for i, candidate := range allSpecs {
			corpus := make([]*spec.Spec, 0, len(allSpecs)-1)
			for j, s := range allSpecs {
				if j != i {
					corpus = append(corpus, s)
				}
			}
			results = append(results, triage.Triage(candidate, corpus, opts))
		}
	}

	// --fix: repair fixable structural issues before output
	if triageFix {
		for _, r := range results {
			if err := applyTriageFixes(r, allSpecs); err != nil {
				fmt.Fprintf(os.Stderr, "warning: fix failed for %s: %v\n", r.Slug, err)
			}
		}
	}

	// Output
	if triageJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
	} else {
		printTriageResults(results)
	}

	// Exit code logic
	hasErrors := false
	hasWarnings := false
	for _, r := range results {
		for _, iss := range r.Issues {
			if iss.Kind == "error" {
				hasErrors = true
			} else {
				hasWarnings = true
			}
		}
	}

	if hasErrors {
		os.Exit(1)
	}
	if hasWarnings && triageNoPrompt {
		os.Exit(2)
	}

	return nil
}

// printTriageResults writes human-readable triage output.
func printTriageResults(results []triage.Result) {
	anyIssues := false
	for _, r := range results {
		if len(r.Issues) == 0 {
			continue
		}
		anyIssues = true
		fmt.Printf("%s\n", r.Slug)
		for _, iss := range r.Issues {
			prefix := "⚠"
			if iss.Kind == "error" {
				prefix = "✗"
			}
			fmt.Printf("  %s [%s] %s\n", prefix, iss.Check, iss.Message)
		}
	}

	// Summary line
	errorCount := 0
	warnCount := 0
	passCount := 0
	for _, r := range results {
		hasErr := false
		for _, iss := range r.Issues {
			if iss.Kind == "error" {
				hasErr = true
				errorCount++
			} else {
				warnCount++
			}
		}
		if !hasErr && len(r.Issues) == 0 {
			passCount++
		}
	}

	if !anyIssues {
		fmt.Printf("✓ All %d spec(s) passed triage.\n", len(results))
	} else {
		fmt.Printf("\n%d spec(s) checked — %d error(s), %d warning(s)\n",
			len(results), errorCount, warnCount)
	}
}

// applyTriageFixes handles the --fix path for a single triage result.
// Currently fixes: missing "created" date (writes mtime as YYYY-MM-DD to frontmatter).
func applyTriageFixes(r triage.Result, allSpecs []*spec.Spec) error {
	for _, iss := range r.Issues {
		if iss.Check != "structural" || iss.Kind != "warning" {
			continue
		}
		// Only fix the "created date is missing" warning
		if iss.Message != "created date is missing (fixable with --fix)" {
			continue
		}

		// Find the spec file path
		var targetSpec *spec.Spec
		for _, s := range allSpecs {
			if s.Slug == r.Slug {
				targetSpec = s
				break
			}
		}
		if targetSpec == nil {
			continue
		}

		data, err := os.ReadFile(targetSpec.Path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", targetSpec.Path, err)
		}

		dateStr := targetSpec.ModifiedAt.Format("2006-01-02")
		updated := spec.SetFrontmatterField(string(data), "created", dateStr)

		if err := os.WriteFile(targetSpec.Path, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", targetSpec.Path, err)
		}
		fmt.Printf("  fixed: %s — added created: %s\n", r.Slug, dateStr)
	}
	return nil
}
