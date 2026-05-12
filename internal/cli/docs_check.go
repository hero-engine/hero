package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Documentation utilities",
}

var docsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate documentation freshness against actual file counts",
	Long: `Compares numeric claims in README.md (e.g. "22 commands", "33 specialist agents",
"37 skills") against the actual .md files in agents/, commands/, and skills/ directories.
Also checks that each agent, command, and skill filename is mentioned in the README.`,
	RunE: runDocsCheck,
}

func init() {
	docsCmd.AddCommand(docsCheckCmd)
}

func runDocsCheck(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()

	// Count actual .md files in each directory.
	agentCount := countMDFiles(filepath.Join(projectRoot, "agents"))
	commandCount := countMDFiles(filepath.Join(projectRoot, "commands"))
	skillCount := countMDFiles(filepath.Join(projectRoot, "skills"))

	fmt.Println("Documentation freshness check")
	fmt.Println("=============================")
	fmt.Println()
	fmt.Printf("Actual counts:\n")
	fmt.Printf("  agents:   %d\n", agentCount)
	fmt.Printf("  commands: %d\n", commandCount)
	fmt.Printf("  skills:   %d\n", skillCount)
	fmt.Println()

	// Check all documentation files.
	docFiles := []string{"README.md", "GETTING-STARTED.md"}
	issues := 0

	for _, docFile := range docFiles {
		docPath := filepath.Join(projectRoot, docFile)
		docBytes, err := os.ReadFile(docPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("%s not found — skipping.\n\n", docFile)
				continue
			}
			return fmt.Errorf("reading %s: %w", docFile, err)
		}
		content := string(docBytes)

		fmt.Printf("--- %s ---\n\n", docFile)

		// Numeric claim checks.
		claimPatterns := []struct {
			pattern *regexp.Regexp
			label   string
			actual  int
		}{
			{regexp.MustCompile(`(\d+)\s+commands`), "commands", commandCount},
			{regexp.MustCompile(`(\d+)\s+(?:specialist\s+)?agents`), "agents", agentCount},
			{regexp.MustCompile(`(\d+)\s+skills`), "skills", skillCount},
		}

		fmt.Println("Claim validation:")
		for _, cp := range claimPatterns {
			matches := cp.pattern.FindAllStringSubmatch(content, -1)
			if len(matches) == 0 {
				continue
			}
			for _, m := range matches {
				claimed := 0
				fmt.Sscanf(m[1], "%d", &claimed)
				if claimed != cp.actual {
					issues++
					fmt.Printf("  %-10s  claims %d, actual %d  ← MISMATCH\n", cp.label, claimed, cp.actual)
				} else {
					fmt.Printf("  %-10s  claims %d, actual %d  ✓\n", cp.label, claimed, cp.actual)
				}
			}
		}

		// Mention checks (only for README — it's the comprehensive reference).
		if docFile == "README.md" {
			contentLower := strings.ToLower(content)
			type dirCheck struct {
				label string
				dir   string
			}
			dirs := []dirCheck{
				{"agents", filepath.Join(projectRoot, "agents")},
				{"commands", filepath.Join(projectRoot, "commands")},
				{"skills", filepath.Join(projectRoot, "skills")},
			}
			for _, dc := range dirs {
				missing := findUnmentioned(dc.dir, contentLower)
				if len(missing) > 0 {
					issues += len(missing)
					fmt.Printf("\n  Unmentioned %s:\n", dc.label)
					for _, name := range missing {
						fmt.Printf("    %s\n", name)
					}
				}
			}
		}
		fmt.Println()
	}

	if issues == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	fmt.Printf("%d issue(s) found.\n", issues)
	os.Exit(1)
	return nil
}

// countMDFiles returns the number of .md files in a directory (non-recursive).
func countMDFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count
}

// findUnmentioned returns basenames (without .md) of files in dir that are not
// mentioned anywhere in the lowercased readme content.
func findUnmentioned(dir, readmeLower string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var missing []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".md")
		if !strings.Contains(readmeLower, strings.ToLower(base)) {
			missing = append(missing, base)
		}
	}
	return missing
}
