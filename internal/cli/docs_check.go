package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/config"
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
var docsCheckPublic bool
var docsCheckProduction string
var docsCheckExpectedRevision string

func init() {
	docsCheckCmd.Flags().BoolVar(&docsCheckInvocations, "invocations", false,
		"also scan markdown surfaces for stale `hero <command>` invocations")
	docsCheckCmd.Flags().BoolVar(&docsCheckPublic, "public", false,
		"validate public claims, config examples, dependency bounds, and revision markers")
	docsCheckCmd.Flags().StringVar(&docsCheckProduction, "production", "",
		"crawl a deployed surface and verify revision parity: docs, landing, or all")
	docsCheckCmd.Flags().StringVar(&docsCheckExpectedRevision, "expected-revision", "",
		"exact source revision required by --production")
	docsCmd.AddCommand(docsCheckCmd)
}

func runDocsCheck(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()

	var agentCount, commandCount, skillCount int
	var err error
	if isEngineSourceRepo(projectRoot) {
		// Engine source repo: agents/commands/skills live as authored
		// source under core/ + domains/<domain>/ (and are mirrored into
		// .claude/, .codex/, web/docs/site/), not as an installed harness
		// tree. Count the canonical deduped install set for the active
		// domain via the same enumeration `hero install` uses, so the
		// checker's numbers can never diverge from what install copies.
		domain := activeDomainForRoot(projectRoot)
		agentCount, commandCount, skillCount, err = canonicalInstallCounts(domain)
		if err != nil {
			return err
		}
		fmt.Println("Documentation freshness check")
		fmt.Println("=============================")
		fmt.Println()
		fmt.Printf("Engine source repo detected — counting canonical install set for domain %q (core + domain, deduped).\n", domain)
		fmt.Println()
	} else {
		// Installed workspace: count the harness tree hero install wrote.
		agentCount = countMDFiles(filepath.Join(projectRoot, "agents"))
		commandCount = countMDFiles(filepath.Join(projectRoot, "commands"))
		// Skills are directories containing SKILL.md, not flat .md files.
		skillCount = countSkillDirs(filepath.Join(projectRoot, "skills"))

		fmt.Println("Documentation freshness check")
		fmt.Println("=============================")
		fmt.Println()
	}

	fmt.Printf("Actual counts:\n")
	fmt.Printf("  agents:   %d\n", agentCount)
	fmt.Printf("  commands: %d\n", commandCount)
	fmt.Printf("  skills:   %d\n", skillCount)
	if docsCheckPublic {
		inventory, inventoryErr := canonicalMCPInventory(projectRoot)
		if inventoryErr != nil {
			return inventoryErr
		}
		fmt.Printf("  MCP tools (total): %d\n", inventory.Total)
		fmt.Printf("  MCP tools (default profile): %d\n", inventory.Default)
		for _, profile := range inventory.Profiles {
			fmt.Printf("  MCP tools (profile %s): %d\n", profile.Name, profile.Count)
		}
	}
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
				label    string
				dir      string
				skillDir bool
			}
			dirs := []dirCheck{
				{"agents", filepath.Join(projectRoot, "agents"), false},
				{"commands", filepath.Join(projectRoot, "commands"), false},
				{"skills", filepath.Join(projectRoot, "skills"), true},
			}
			for _, dc := range dirs {
				var missing []string
				if dc.skillDir {
					missing = findUnmentionedSkills(dc.dir, contentLower)
				} else {
					missing = findUnmentioned(dc.dir, contentLower)
				}
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

	if docsCheckPublic || docsCheckProduction != "" {
		fmt.Println("--- Public documentation contract ---")
		fmt.Println()
		failures := publicDocsIssues(projectRoot)
		if docsCheckPublic {
			executable, executableErr := os.Executable()
			if executableErr != nil {
				failures = append(failures, fmt.Sprintf("resolve current Hero executable for quickstart exercise: %v", executableErr))
			} else {
				failures = append(failures, publicQuickstartIssues(executable, projectRoot)...)
			}
		}
		if len(failures) == 0 {
			fmt.Println("  Public claims, config examples, dependency bounds, and revision templates are consistent.")
		} else {
			for _, failure := range failures {
				fmt.Printf("  %s\n", failure)
			}
			issues += len(failures)
		}
		fmt.Println()
	}

	if docsCheckProduction != "" {
		fmt.Println("--- Production parity ---")
		fmt.Println()
		failures := productionPublicIssues(nil, docsCheckProduction, docsCheckExpectedRevision, productionBaseURLs())
		if len(failures) == 0 {
			fmt.Printf("  %s surface matches source revision %s.\n", docsCheckProduction, docsCheckExpectedRevision)
		} else {
			for _, failure := range failures {
				fmt.Printf("  %s\n", failure)
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

// isEngineSourceRepo reports whether root is the hero engine's own source
// tree, where agents/commands/skills live as authored source under core/
// and domains/<domain>/ rather than as an installed harness. The signature
// — core/, domains/, and .goreleaser.yaml all present at root — is unique
// to this repo; an installed workspace has none of these three.
func isEngineSourceRepo(root string) bool {
	for _, marker := range []string{"core", "domains", ".goreleaser.yaml"} {
		if _, err := os.Stat(filepath.Join(root, marker)); err != nil {
			return false
		}
	}
	return true
}

// activeDomainForRoot resolves the active domain pack for the repo at root:
// hero.json's domain if set, else the "engineering" default.
func activeDomainForRoot(root string) string {
	if cfg, err := config.Load(root); err == nil {
		return cfg.PrimaryDomain()
	}
	return "engineering"
}

// canonicalInstallCounts returns the deduped agent/command/skill counts the
// install pipeline would materialize for domain, built from the same
// embedded content FS `hero install` reads (core overlaid by the active
// domain). Reusing install.EnumerateContent guarantees the checker's counts
// equal what install copies.
func canonicalInstallCounts(domain string) (agents, commands, skills int, err error) {
	domainFS, derr := hero.DomainFS(domain)
	if derr != nil {
		return 0, 0, 0, fmt.Errorf("resolving domain %q content: %w", domain, derr)
	}
	manifest, merr := install.EnumerateContent(hero.OverlayFS(domainFS, hero.CoreFS()), domain)
	if merr != nil {
		return 0, 0, 0, fmt.Errorf("enumerating canonical install set: %w", merr)
	}
	return len(manifest.Agents), len(manifest.Commands), len(manifest.Skills), nil
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

// countSkillDirs returns the number of skill directories under dir. Each skill
// is a subdirectory containing a SKILL.md file.
func countSkillDirs(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); err == nil {
			count++
		}
	}
	return count
}

// findUnmentionedSkills returns names of skill directories not mentioned in the
// lowercased readme content.
func findUnmentionedSkills(dir, readmeLower string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var missing []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); err != nil {
			continue
		}
		if !strings.Contains(readmeLower, strings.ToLower(e.Name())) {
			missing = append(missing, e.Name())
		}
	}
	return missing
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
