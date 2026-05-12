package cli

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check spec frontmatter, relations, and file paths",
	Long: `Validates all specs in the workspace for common issues:
  - Missing required frontmatter (title, type, status)
  - Invalid type or status values
  - Broken relation targets (slug not found)
  - File paths in Changes section that don't exist`,
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	if len(specs) == 0 {
		fmt.Println("No specs found.")
		return nil
	}

	// Build slug index for relation checking
	slugIndex := make(map[string]bool)
	for _, s := range specs {
		slugIndex[s.Slug] = true
	}

	totalIssues := 0
	for _, s := range specs {
		issues := validateSpec(s, slugIndex, projectRoot)
		if len(issues) > 0 {
			fmt.Printf("%s (%s):\n", s.Slug, s.Path)
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue)
			}
			totalIssues += len(issues)
		}
	}

	if totalIssues == 0 {
		fmt.Printf("All %d specs are valid.\n", len(specs))
	} else {
		fmt.Printf("\nFound %d issue(s) across %d spec(s).\n", totalIssues, len(specs))
	}

	return nil
}

// validateSpec checks a single spec for issues and returns a list of problem descriptions.
func validateSpec(s *spec.Spec, slugIndex map[string]bool, projectRoot string) []string {
	var issues []string

	// Check required frontmatter
	if s.Title == "" {
		issues = append(issues, "missing title")
	}
	if s.Type == "" {
		issues = append(issues, "missing type")
	}
	if s.Status == "" {
		issues = append(issues, "missing status")
	}

	// Validate type
	validTypes := map[spec.Type]bool{
		spec.TypeFeature:    true,
		spec.TypeBug:        true,
		spec.TypeConvention: true,
		spec.TypeDecision:   true,
		spec.TypeInitiative: true,
	}
	if s.Type != "" && !validTypes[s.Type] {
		issues = append(issues, fmt.Sprintf("invalid type: %q", s.Type))
	}

	// Validate status
	validStatuses := map[spec.Status]bool{
		spec.StatusPlanning:   true,
		spec.StatusInReview:   true,
		spec.StatusDelivering: true,
		spec.StatusCompleted:  true,
		spec.StatusDraft:      true,
		spec.StatusActive:     true,
		spec.StatusProposed:   true,
		spec.StatusAccepted:   true,
		spec.StatusSuperseded: true,
	}
	if s.Status != "" && !validStatuses[s.Status] {
		issues = append(issues, fmt.Sprintf("invalid status: %q", s.Status))
	}

	// Check status/type compatibility
	if s.Type == spec.TypeConvention && s.Status != "" {
		conventionStatuses := map[spec.Status]bool{
			spec.StatusDraft: true, spec.StatusActive: true, spec.StatusSuperseded: true,
		}
		if !conventionStatuses[s.Status] {
			issues = append(issues, fmt.Sprintf("convention should use draft/active/superseded status, not %q", s.Status))
		}
	}
	if s.Type == spec.TypeDecision && s.Status != "" {
		decisionStatuses := map[spec.Status]bool{
			spec.StatusProposed: true, spec.StatusAccepted: true, spec.StatusSuperseded: true,
		}
		if !decisionStatuses[s.Status] {
			issues = append(issues, fmt.Sprintf("decision should use proposed/accepted/superseded status, not %q", s.Status))
		}
	}

	// Check relations point to valid slugs
	for _, rel := range s.Relations {
		if !slugIndex[rel.Target] {
			issues = append(issues, fmt.Sprintf("relation %s:%s — target spec %q not found", rel.Kind, rel.Target, rel.Target))
		}
	}

	// Work specs must declare smoke coverage (or defer it explicitly).
	// Knowledge types (convention, decision, …) are exempt.
	if s.IsWorkSpec() && s.Smoke == nil {
		issues = append(issues, "missing smoke: field — add 'smoke: deferred' or a full smoke: block")
	}

	// Check files in Changes section exist
	for _, f := range s.FilesTouched {
		fullPath := f
		if !isAbsPath(f) {
			fullPath = projectRoot + "/" + f
		}
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("file not found: %s", f))
		}
	}

	return issues
}

func isAbsPath(p string) bool {
	return len(p) > 0 && p[0] == '/'
}
