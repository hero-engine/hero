package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hero-engine/hero/internal/install"
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
Also checks that each agent, command, and skill filename is mentioned in the README.

With --invocations, additionally scans markdown surfaces (commands/, skills/,
agents/, web/docs/src/, top-level docs, and the rendered AGENTS.md template)
for ` + "`hero <command>`" + ` invocations and verifies each resolves against
the cobra command tree. Lines marked with <!-- drift-test:ignore --> are
skipped; .hero/specs/ and .hero/planning/ are excluded entirely.`,
	RunE: runDocsCheck,
}

var docsCheckInvocations bool

func init() {
	docsCheckCmd.Flags().BoolVar(&docsCheckInvocations, "invocations", false,
		"also scan markdown surfaces for stale `hero <command>` invocations")
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

	if docsCheckInvocations {
		fmt.Println("--- CLI invocation drift ---")
		fmt.Println()
		failures := scanInvocationDrift(projectRoot)
		if len(failures) == 0 {
			fmt.Println("  All `hero <command>` references in markdown resolve cleanly.")
		} else {
			for _, f := range failures {
				fmt.Printf("  %s:%d  `%s`  →  %s\n", f.File, f.Line, f.Raw, f.err)
			}
			issues += len(failures)
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

// invocationFailure is a single drift hit reported by --invocations.
type invocationFailure struct {
	Invocation
	err error
}

// scanInvocationDrift walks every markdown surface (mirrors the surface
// set in markdown_drift_test.go) and returns one entry per invocation
// that fails to resolve against rootCmd.
func scanInvocationDrift(projectRoot string) []invocationFailure {
	var invs []Invocation
	for _, d := range []string{"commands", "skills", "agents", "web/docs/src"} {
		invs = append(invs, walkMarkdownDirForInvocations(projectRoot, d)...)
	}
	for _, f := range []string{"README.md", "AGENTS.md", "GETTING-STARTED.md"} {
		abs := filepath.Join(projectRoot, f)
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		invs = append(invs, ExtractInvocations(f, data)...)
	}
	rendered := install.RenderAgentsMdBodyForDriftTest()
	invs = append(invs, ExtractInvocations("<rendered:internal/install/agents_md.go>", rendered)...)

	var failures []invocationFailure
	for _, inv := range invs {
		if err := ValidateInvocation(rootCmd, inv); err != nil {
			failures = append(failures, invocationFailure{Invocation: inv, err: err})
		}
	}
	return failures
}

// walkMarkdownDirForInvocations recursively reads .md files under
// projectRoot/dir (skipping excluded paths) and extracts invocations.
// Errors are swallowed silently — this is a best-effort CLI scan.
func walkMarkdownDirForInvocations(projectRoot, dir string) []Invocation {
	abs := filepath.Join(projectRoot, dir)
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil
	}
	var out []Invocation
	_ = filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(projectRoot, path)
		if relErr != nil {
			rel = path
		}
		relSlash := filepath.ToSlash(rel)
		if isExcludedDir(relSlash) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		out = append(out, ExtractInvocations(relSlash, data)...)
		return nil
	})
	return out
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
