package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/herotest"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Generate and run tests from spec acceptance criteria",
	Long:  `Generate, run, and manage end-to-end tests derived from Hero spec acceptance criteria.`,
}

// ---------- test generate ----------

var (
	testGenerateAll  bool
	testGenerateMode string
)

var testGenerateCmd = &cobra.Command{
	Use:   "generate [<slug>]",
	Short: "Generate test files from spec acceptance criteria",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTestGenerate,
}

// ---------- test run ----------

var testRunAll bool

var testRunCmd = &cobra.Command{
	Use:   "run [<slug>]",
	Short: "Run tests for a specific spec or all specs",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTestRun,
}

// ---------- test list ----------

var testListCmd = &cobra.Command{
	Use:   "list",
	Short: "List specs with test coverage status",
	RunE:  runTestList,
}

// ---------- test show ----------

var testShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Print the generated test file content",
	Args:  cobra.ExactArgs(1),
	RunE:  runTestShow,
}

func init() {
	testGenerateCmd.Flags().BoolVar(&testGenerateAll, "all", false, "generate tests for all delivering/completed specs")
	testGenerateCmd.Flags().StringVar(&testGenerateMode, "mode", "", "override generation mode (agent, assisted, autonomous)")

	testRunCmd.Flags().BoolVar(&testRunAll, "all", false, "run tests for all specs with test files")

	testCmd.AddCommand(testGenerateCmd)
	testCmd.AddCommand(testRunCmd)
	testCmd.AddCommand(testListCmd)
	testCmd.AddCommand(testShowCmd)
}

func runTestGenerate(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	if testGenerateAll {
		return generateAllTests(projectRoot, cfg, specs)
	}

	if len(args) == 0 {
		return fmt.Errorf("provide a spec slug or use --all")
	}

	slug := args[0]
	s := findSpec(slug, specs)
	if s == nil {
		return fmt.Errorf("spec %q not found", slug)
	}

	testFile, err := herotest.Generate(projectRoot, s, cfg.Testing, testGenerateMode)
	if err != nil {
		return err
	}

	fmt.Printf("Generated %s\n", testFile)
	return nil
}

func generateAllTests(projectRoot string, cfg config.Config, specs []*spec.Spec) error {
	var generated int
	for _, s := range specs {
		if !s.IsWorkSpec() {
			continue
		}
		if s.Status != spec.StatusDelivering && s.Status != spec.StatusCompleted {
			continue
		}

		criteria := herotest.ExtractCriteria(s)
		if len(criteria) == 0 {
			continue
		}

		testFile, err := herotest.Generate(projectRoot, s, cfg.Testing, testGenerateMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warning: %s — %s\n", s.Slug, err)
			continue
		}
		fmt.Printf("  Generated %s\n", testFile)
		generated++
	}

	if generated == 0 {
		fmt.Println("No specs with acceptance criteria found.")
	} else {
		fmt.Printf("\nGenerated %d test file(s).\n", generated)
	}
	return nil
}

func runTestRun(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if testRunAll {
		return runAllTests(projectRoot, cfg)
	}

	if len(args) == 0 {
		return fmt.Errorf("provide a spec slug or use --all")
	}

	slug := args[0]

	if !herotest.TestFileExists(projectRoot, slug, cfg.Testing) {
		return fmt.Errorf("no test file found for %q (run: hero test generate %s)", slug, slug)
	}

	runner, runArgs, err := herotest.RunArgs(slug, cfg.Testing)
	if err != nil {
		return err
	}

	fmt.Printf("Running tests for %s...\n\n", slug)

	c := exec.Command(runner, runArgs...)
	c.Dir = projectRoot
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}
	return nil
}

func runAllTests(projectRoot string, cfg config.Config) error {
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	var testFiles []string
	for _, s := range specs {
		if herotest.TestFileExists(projectRoot, s.Slug, cfg.Testing) {
			testFiles = append(testFiles, herotest.TestFilePath(s.Slug, cfg.Testing))
		}
	}

	if len(testFiles) == 0 {
		fmt.Println("No test files found. Run: hero test generate --all")
		return nil
	}

	// Run all test files in a single Playwright invocation
	frameworkName := "playwright"
	if cfg.Testing != nil {
		frameworkName = cfg.Testing.FrameworkOrDefault()
	}

	runner := "npx"
	args := []string{"playwright", "test"}
	if cfg.Testing != nil && cfg.Testing.RunnerCommand != "" {
		parts := strings.Fields(cfg.Testing.RunnerCommand)
		if len(parts) > 0 {
			runner = parts[0]
			args = parts[1:]
		}
	}
	args = append(args, testFiles...)
	if cfg.Testing != nil && cfg.Testing.ConfigPath != "" {
		args = append(args, "--config", cfg.Testing.ConfigPath)
	}

	fmt.Printf("Running %d test file(s) with %s...\n\n", len(testFiles), frameworkName)

	c := exec.Command(runner, args...)
	c.Dir = projectRoot
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}
	return nil
}

func runTestList(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	fmt.Printf("%-30s  %-12s  %-10s  %s\n", "Spec", "Status", "Tests", "Path")
	fmt.Println(strings.Repeat("─", 80))

	var withTests, withCriteria int
	for _, s := range specs {
		if !s.IsWorkSpec() {
			continue
		}

		criteria := herotest.ExtractCriteria(s)
		if len(criteria) == 0 {
			continue
		}
		withCriteria++

		testFile := herotest.TestFilePath(s.Slug, cfg.Testing)
		hasTests := herotest.TestFileExists(projectRoot, s.Slug, cfg.Testing)

		marker := "  "
		if hasTests {
			marker = "ok"
			withTests++
		}

		fmt.Printf("  %-28s  %-12s  %-10s  %s\n",
			truncate(s.Slug, 28), s.Status, marker, testFile)
	}

	fmt.Printf("\n%d/%d specs have generated tests.\n", withTests, withCriteria)
	return nil
}

func runTestShow(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	slug := args[0]
	testFile := herotest.TestFilePath(slug, cfg.Testing)
	absPath := filepath.Join(projectRoot, testFile)

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no test file for %q (run: hero test generate %s)", slug, slug)
		}
		return err
	}

	fmt.Printf("# %s\n\n", testFile)
	fmt.Print(string(data))
	return nil
}

// findSpec finds a spec by slug in the list.
func findSpec(slug string, specs []*spec.Spec) *spec.Spec {
	for _, s := range specs {
		if s.Slug == slug {
			return s
		}
	}
	return nil
}
