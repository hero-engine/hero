package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/score"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <spec-slug>",
	Short: "Verify implementation against spec acceptance criteria",
	Long: `Reviews a spec's acceptance criteria and produces a verification checklist.

For each acceptance criterion, reports whether it can be verified from the
current codebase state. Use after manual delivery to check your work before
marking the spec as complete.

Outputs the acceptance criteria, files that should have been touched, and
a structured prompt suitable for AI-assisted review.`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

func runVerify(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Find spec
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == args[0] {
			target = s
			break
		}
	}
	if target == nil {
		return fmt.Errorf("spec %q not found", args[0])
	}

	// Score the spec first
	scoreCfg := score.DefaultConfig()
	if cfg.Score != nil && cfg.Score.MinScore > 0 {
		scoreCfg.MinScore = cfg.Score.MinScore
	}
	result := score.Score(target, scoreCfg)

	// Extract acceptance criteria
	acSection := ""
	for name, content := range target.Sections {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "acceptance") || strings.Contains(lower, "success criteria") ||
			strings.Contains(lower, "done when") || strings.Contains(lower, "definition of done") {
			acSection = content
			break
		}
	}

	// Extract test strategy
	testSection := ""
	for name, content := range target.Sections {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "test") || strings.Contains(lower, "verification") ||
			strings.Contains(lower, "validation") {
			testSection = content
			break
		}
	}

	// Print verification report
	fmt.Printf("\n  Verification Report: %s\n", target.Slug)
	fmt.Printf("  %s\n", strings.Repeat("═", 60))

	// Spec quality
	fmt.Printf("\n  Spec Quality: %d/100 (Grade %s)\n", result.Score, result.Grade)

	// Acceptance criteria
	if acSection != "" {
		fmt.Printf("\n  Acceptance Criteria:\n")
		criteria := extractCriteriaItems(acSection)
		if len(criteria) > 0 {
			for i, c := range criteria {
				fmt.Printf("    %d. [ ] %s\n", i+1, c)
			}
		} else {
			fmt.Printf("    (section exists but no list items found)\n")
		}
	} else {
		fmt.Printf("\n  ⚠ No acceptance criteria section found in spec.\n")
	}

	// Files that should have been touched
	if len(target.FilesTouched) > 0 {
		fmt.Printf("\n  Expected File Changes:\n")
		for _, f := range target.FilesTouched {
			// Check if file exists
			if _, err := os.Stat(f); err == nil {
				fmt.Printf("    ✓ %s (exists)\n", f)
			} else {
				fmt.Printf("    ? %s (not found — may need creating or path differs)\n", f)
			}
		}
	}

	// Test strategy
	if testSection != "" {
		fmt.Printf("\n  Test Strategy:\n")
		items := extractCriteriaItems(testSection)
		if len(items) > 0 {
			for _, item := range items {
				fmt.Printf("    • %s\n", item)
			}
		} else {
			// Print the section as-is, trimmed
			for _, line := range strings.Split(strings.TrimSpace(testSection), "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					fmt.Printf("    %s\n", trimmed)
				}
			}
		}
	}

	// Generate AI verification prompt
	fmt.Printf("\n  %s\n", strings.Repeat("─", 60))
	fmt.Printf("  AI Verification Prompt (copy to agent):\n")
	fmt.Printf("  %s\n\n", strings.Repeat("─", 60))

	var prompt strings.Builder
	prompt.WriteString(fmt.Sprintf("Verify the implementation of spec '%s'.\n\n", target.Slug))
	prompt.WriteString(fmt.Sprintf("Spec: %s\n\n", target.Path))

	if acSection != "" {
		prompt.WriteString("Check each acceptance criterion:\n")
		criteria := extractCriteriaItems(acSection)
		for i, c := range criteria {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
		}
		prompt.WriteString("\n")
	}

	if len(target.FilesTouched) > 0 {
		prompt.WriteString("Expected files changed:\n")
		for _, f := range target.FilesTouched {
			prompt.WriteString(fmt.Sprintf("- %s\n", f))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("For each criterion: examine the code, run relevant tests if possible, and report PASS/FAIL with evidence.\n")
	prompt.WriteString("At the end, give an overall PASS/FAIL verdict.\n")

	fmt.Print(prompt.String())
	fmt.Println()

	// Auto-archive when the spec is already at status: completed. This
	// closes the manual /deliver loop: the model flips status to
	// completed during delivery, runs `hero verify` for the AC report,
	// and the verify call moves the spec under specs/ without anyone
	// having to remember `hero spec complete`. No-op when status hasn't
	// flipped yet, so calling verify mid-delivery is still safe.
	moved, err := autoArchiveIfCompleted(target.Path, heroDir)
	if err != nil {
		return fmt.Errorf("auto-archive: %w", err)
	}
	if moved {
		fmt.Printf("Auto-archived: spec moved to specs/%s/\n", target.Slug)
	}

	return nil
}

// extractCriteriaItems pulls list items from a section.
func extractCriteriaItems(section string) []string {
	var items []string
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 3 {
			continue
		}
		// Match bullet or numbered list items
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			items = append(items, strings.TrimSpace(trimmed[2:]))
		} else if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			// Numbered items like "1. text" or "1) text"
			for i := 1; i < len(trimmed); i++ {
				if trimmed[i] == '.' || trimmed[i] == ')' {
					if i+1 < len(trimmed) && trimmed[i+1] == ' ' {
						items = append(items, strings.TrimSpace(trimmed[i+2:]))
					}
					break
				}
				if trimmed[i] < '0' || trimmed[i] > '9' {
					break
				}
			}
		}
	}
	return items
}
