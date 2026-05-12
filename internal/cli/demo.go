package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/demos"
	"github.com/hero-engine/hero/internal/herotest"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Record and manage video demos for delivered specs",
	Long:  `Record video demos by running spec tests with Playwright video capture enabled.`,
}

// ---------- demo record ----------

var demoRecordAll bool

var demoRecordCmd = &cobra.Command{
	Use:   "record [<slug>]",
	Short: "Record a video demo for a spec",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDemoRecord,
}

// ---------- demo list ----------

var demoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List specs with demo recording status",
	RunE:  runDemoList,
}

// ---------- demo show ----------

var demoShowCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Show demo recording details for a spec",
	Args:  cobra.ExactArgs(1),
	RunE:  runDemoShow,
}

// ---------- demo clean ----------

var demoCleanCmd = &cobra.Command{
	Use:   "clean [<slug>]",
	Short: "Remove demo recordings",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDemoClean,
}

func init() {
	demoRecordCmd.Flags().BoolVar(&demoRecordAll, "all", false, "record demos for all specs with test files")

	demoCmd.AddCommand(demoRecordCmd)
	demoCmd.AddCommand(demoListCmd)
	demoCmd.AddCommand(demoShowCmd)
	demoCmd.AddCommand(demoCleanCmd)
}

func runDemoRecord(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	frameworkName := "playwright"
	if cfg.Demos != nil {
		frameworkName = cfg.Demos.FrameworkOrDefault()
	}

	fw, err := demos.Get(frameworkName)
	if err != nil {
		return err
	}

	if demoRecordAll {
		return recordAllDemos(projectRoot, cfg, fw)
	}

	if len(args) == 0 {
		return fmt.Errorf("provide a spec slug or use --all")
	}

	slug := args[0]
	return recordDemo(projectRoot, cfg, fw, slug)
}

func recordDemo(projectRoot string, cfg config.Config, fw demos.DemoFramework, slug string) error {
	testFile := herotest.TestFilePath(slug, cfg.Testing)
	if !herotest.TestFileExists(projectRoot, slug, cfg.Testing) {
		return fmt.Errorf("no test file found for %q at %s (run: hero test generate %s)", slug, testFile, slug)
	}

	fmt.Printf("Recording demo for %s...\n", slug)

	result, err := fw.Record(slug, testFile, cfg.Demos, cfg.Testing, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  warning: %s\n", err)
	}

	if result != nil {
		fmt.Printf("  Status: %s\n", result.Status)
		fmt.Printf("  Videos: %d\n", len(result.Videos))
		for _, v := range result.Videos {
			fmt.Printf("    %s (%d bytes)\n", v.Path, v.SizeBytes)
		}
		dir := fw.VideoDir(slug, cfg.Demos, projectRoot)
		fmt.Printf("  Output: %s\n", dir)
	}

	return nil
}

func recordAllDemos(projectRoot string, cfg config.Config, fw demos.DemoFramework) error {
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	var recorded int
	for _, s := range specs {
		if !herotest.TestFileExists(projectRoot, s.Slug, cfg.Testing) {
			continue
		}

		if err := recordDemo(projectRoot, cfg, fw, s.Slug); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: %s — %s\n", s.Slug, err)
		} else {
			recorded++
		}
	}

	if recorded == 0 {
		fmt.Println("No specs with test files found. Run: hero test generate --all")
	} else {
		fmt.Printf("\nRecorded %d demo(s).\n", recorded)
	}
	return nil
}

func runDemoList(cmd *cobra.Command, args []string) error {
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

	manifests, _ := demos.ListDemos(projectRoot, cfg.Demos)
	manifestMap := make(map[string]*demos.Manifest)
	for _, m := range manifests {
		manifestMap[m.Slug] = m
	}

	fmt.Printf("%-30s  %-12s  %-10s  %-10s  %s\n", "Spec", "Status", "Tests", "Demo", "Videos")
	fmt.Println(strings.Repeat("─", 85))

	for _, s := range specs {
		if !s.IsWorkSpec() {
			continue
		}

		hasTests := herotest.TestFileExists(projectRoot, s.Slug, cfg.Testing)
		testMarker := "  "
		if hasTests {
			testMarker = "ok"
		}

		demoMarker := "  "
		videoCount := ""
		if m, ok := manifestMap[s.Slug]; ok {
			demoMarker = m.Status
			videoCount = fmt.Sprintf("%d", len(m.Videos))
		}

		fmt.Printf("  %-28s  %-12s  %-10s  %-10s  %s\n",
			truncate(s.Slug, 28), s.Status, testMarker, demoMarker, videoCount)
	}
	return nil
}

func runDemoShow(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	slug := args[0]

	frameworkName := "playwright"
	if cfg.Demos != nil {
		frameworkName = cfg.Demos.FrameworkOrDefault()
	}

	fw, err := demos.Get(frameworkName)
	if err != nil {
		return err
	}

	dir := fw.VideoDir(slug, cfg.Demos, projectRoot)
	m, err := demos.ReadManifest(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no demo found for %q (run: hero demo record %s)", slug, slug)
		}
		return err
	}

	fmt.Printf("Demo: %s\n", m.Slug)
	if m.Title != "" {
		fmt.Printf("Title: %s\n", m.Title)
	}
	fmt.Printf("Recorded: %s\n", m.RecordedAt)
	fmt.Printf("Status: %s\n", m.Status)
	fmt.Printf("Test file: %s\n", m.TestFile)
	fmt.Printf("Videos (%d):\n", len(m.Videos))
	for _, v := range m.Videos {
		fmt.Printf("  %s (%d bytes)\n", v.Path, v.SizeBytes)
	}
	fmt.Printf("Directory: %s\n", dir)
	return nil
}

func runDemoClean(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(args) == 0 {
		// Clean all
		if err := demos.CleanAll(projectRoot, cfg.Demos); err != nil {
			return err
		}
		fmt.Println("Removed all demo recordings.")
		return nil
	}

	slug := args[0]

	frameworkName := "playwright"
	if cfg.Demos != nil {
		frameworkName = cfg.Demos.FrameworkOrDefault()
	}

	fw, err := demos.Get(frameworkName)
	if err != nil {
		return err
	}

	if err := fw.Clean(slug, cfg.Demos, projectRoot); err != nil {
		return err
	}

	fmt.Printf("Removed demo recordings for %s.\n", slug)
	return nil
}
