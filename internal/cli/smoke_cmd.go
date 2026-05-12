package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// SmokeRunRecord is the result of a single smoke script execution.
// Written to .hero/smoke/last-run.json after each invocation.
type SmokeRunRecord struct {
	Slug       string    `json:"slug"`
	Script     string    `json:"script"`
	Status     string    `json:"status"` // pass | fail | skip | deferred
	DurationMS int64     `json:"duration_ms"`
	Timestamp  time.Time `json:"ts"`
	Output     string    `json:"output,omitempty"`
	Error      string    `json:"error,omitempty"`
}

var (
	smokeRunAll   bool
	smokeRunArea  string
	smokeRunSince string
)

var smokeCmd = &cobra.Command{
	Use:   "smoke [feature-slug]",
	Short: "Run per-feature smoke verification",
	Long: `Run smoke verification for one or more features.

  hero smoke <slug>              run a single feature's smoke script
  hero smoke --all               run all features that have a smoke script
  hero smoke --since <ref>       run smokes for features touched since <git-ref>
  hero smoke --area <area>       run smokes for features in an area
  hero smoke status              show last-run status for all smokes`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSmokeCommand,
}

var smokeStatusSubCmd = &cobra.Command{
	Use:   "status",
	Short: "Show last-run smoke status for all features",
	RunE:  runSmokeStatusCommand,
}

func init() {
	smokeCmd.Flags().BoolVar(&smokeRunAll, "all", false, "run all features with a smoke script")
	smokeCmd.Flags().StringVar(&smokeRunArea, "area", "", "run smokes for features in this area")
	smokeCmd.Flags().StringVar(&smokeRunSince, "since", "", "run smokes for features touched since <git-ref>")
	smokeCmd.AddCommand(smokeStatusSubCmd)
}

func runSmokeCommand(cmd *cobra.Command, args []string) error {
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

	switch {
	case len(args) == 1:
		return runSmokeSlug(args[0], specs, projectRoot, heroDir)
	case smokeRunSince != "":
		return runSmokeSince(smokeRunSince, specs, projectRoot, heroDir)
	case smokeRunArea != "":
		return runSmokeArea(smokeRunArea, specs, projectRoot, heroDir)
	case smokeRunAll:
		return runSmokeAll(specs, projectRoot, heroDir)
	default:
		return cmd.Help()
	}
}

func runSmokeSlug(slug string, specs []*spec.Spec, projectRoot, heroDir string) error {
	for _, s := range specs {
		if s.Slug != slug {
			continue
		}
		record := execSmoke(s, projectRoot)
		printSmokeRecord(record)
		if err := saveSmokeResults(heroDir, []SmokeRunRecord{record}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save smoke result: %v\n", err)
		}
		if record.Status == "fail" {
			return fmt.Errorf("smoke failed for %s", slug)
		}
		return nil
	}
	return fmt.Errorf("spec %q not found", slug)
}

func runSmokeAll(specs []*spec.Spec, projectRoot, heroDir string) error {
	var eligible []*spec.Spec
	for _, s := range specs {
		if s.Smoke != nil && !s.Smoke.Deferred && s.Smoke.Script != "" {
			eligible = append(eligible, s)
		}
	}
	if len(eligible) == 0 {
		fmt.Println("No feature smokes to run (all deferred or no smoke scripts configured).")
		return nil
	}
	return runSmokeMultiple(eligible, projectRoot, heroDir)
}

func runSmokeSince(ref string, specs []*spec.Spec, projectRoot, heroDir string) error {
	changed := filesChangedSince(projectRoot, ref)
	if len(changed) == 0 {
		fmt.Printf("No files changed since %s.\n", ref)
		return nil
	}
	fmt.Printf("Files changed since %s: %d\n", ref, len(changed))

	var toRun []*spec.Spec
	for _, s := range specs {
		if s.Smoke == nil || s.Smoke.Deferred || s.Smoke.Script == "" {
			continue
		}
		if smokeTriggeredBy(s, changed, projectRoot) {
			toRun = append(toRun, s)
		}
	}
	if len(toRun) == 0 {
		fmt.Printf("No smoke scripts triggered by changes since %s.\n", ref)
		return nil
	}
	return runSmokeMultiple(toRun, projectRoot, heroDir)
}

func runSmokeArea(area string, specs []*spec.Spec, projectRoot, heroDir string) error {
	areaLow := strings.ToLower(area)
	var toRun []*spec.Spec
	for _, s := range specs {
		if s.Smoke == nil || s.Smoke.Deferred || s.Smoke.Script == "" {
			continue
		}
		if strings.Contains(strings.ToLower(s.Slug), areaLow) {
			toRun = append(toRun, s)
			continue
		}
		for _, tag := range s.Tags {
			if strings.ToLower(tag) == areaLow {
				toRun = append(toRun, s)
				break
			}
		}
	}
	if len(toRun) == 0 {
		fmt.Printf("No smoke scripts found for area %q.\n", area)
		return nil
	}
	return runSmokeMultiple(toRun, projectRoot, heroDir)
}

func runSmokeMultiple(specs []*spec.Spec, projectRoot, heroDir string) error {
	var records []SmokeRunRecord
	failed := 0
	for _, s := range specs {
		record := execSmoke(s, projectRoot)
		records = append(records, record)
		printSmokeRecord(record)
		if record.Status == "fail" {
			failed++
		}
	}
	if err := saveSmokeResults(heroDir, records); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save smoke results: %v\n", err)
	}
	fmt.Printf("\n%d smoke(s) ran: %d passed, %d failed\n",
		len(records), len(records)-failed, failed)
	if failed > 0 {
		return fmt.Errorf("%d smoke(s) failed", failed)
	}
	return nil
}

// execSmoke runs the smoke script for s and returns a populated SmokeRunRecord.
func execSmoke(s *spec.Spec, projectRoot string) SmokeRunRecord {
	record := SmokeRunRecord{
		Slug:      s.Slug,
		Timestamp: time.Now().UTC(),
	}
	if s.Smoke != nil {
		record.Script = s.Smoke.Script
	}

	if s.Smoke == nil || s.Smoke.Deferred {
		record.Status = "deferred"
		return record
	}

	scriptPath := filepath.Join(projectRoot, s.Smoke.Script)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		record.Status = "fail"
		record.Error = fmt.Sprintf("smoke script not found: %s", s.Smoke.Script)
		return record
	}

	start := time.Now()
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = projectRoot
	out, err := cmd.CombinedOutput()
	record.DurationMS = time.Since(start).Milliseconds()
	record.Output = string(out)
	if err != nil {
		record.Status = "fail"
		record.Error = err.Error()
	} else {
		record.Status = "pass"
	}
	return record
}

func printSmokeRecord(r SmokeRunRecord) {
	glyph := "✅"
	switch r.Status {
	case "fail":
		glyph = "❌"
	case "deferred", "skip":
		glyph = "⊘ "
	}
	durStr := ""
	if r.DurationMS > 0 {
		durStr = fmt.Sprintf("  (%dms)", r.DurationMS)
	}
	fmt.Printf("%s %-40s  %s%s\n", glyph, r.Slug, r.Status, durStr)
	if r.Error != "" {
		fmt.Printf("   error: %s\n", r.Error)
	}
}

// smokeTriggeredBy reports whether the changed files match any commit-touches:
// glob in s.Smoke.RunsOn.  Falls back to matching the spec's planning dir.
func smokeTriggeredBy(s *spec.Spec, changedFiles []string, projectRoot string) bool {
	var globs []string
	for _, ro := range s.Smoke.RunsOn {
		if after, ok := strings.CutPrefix(ro, "commit-touches:"); ok {
			globs = append(globs, after)
		}
	}

	// Fallback: the spec's own planning directory (frontmatter edit → re-run).
	if len(globs) == 0 {
		if rel, err := filepath.Rel(projectRoot, filepath.Dir(s.Path)); err == nil {
			globs = append(globs, rel+"/*")
		}
	}

	for _, f := range changedFiles {
		for _, g := range globs {
			if ok, _ := filepath.Match(g, f); ok {
				return true
			}
			// Match path prefixes for directory-level globs.
			dir := strings.TrimSuffix(g, "/*")
			if strings.HasPrefix(f, dir+"/") || f == dir {
				return true
			}
		}
	}
	return false
}

func saveSmokeResults(heroDir string, records []SmokeRunRecord) error {
	smokeDir := filepath.Join(heroDir, "smoke")
	if err := os.MkdirAll(smokeDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(smokeDir, "last-run.json"), data, 0o644)
}

func loadSmokeResults(heroDir string) ([]SmokeRunRecord, error) {
	path := filepath.Join(heroDir, "smoke", "last-run.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []SmokeRunRecord
	return records, json.Unmarshal(data, &records)
}

func runSmokeStatusCommand(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	records, err := loadSmokeResults(heroDir)
	if err != nil {
		return fmt.Errorf("loading smoke results: %w", err)
	}
	if len(records) == 0 {
		fmt.Println("No smoke runs found. Run 'hero smoke --all' to start.")
		return nil
	}

	fmt.Printf("%-40s  %-8s  %-10s  %s\n", "Feature", "Status", "Duration", "Last Run")
	fmt.Println(strings.Repeat("-", 80))
	for _, r := range records {
		dur := fmt.Sprintf("%dms", r.DurationMS)
		ts := r.Timestamp.Local().Format("2006-01-02 15:04")
		fmt.Printf("%-40s  %-8s  %-10s  %s\n", r.Slug, r.Status, dur, ts)
		if r.Status == "fail" && r.Error != "" {
			fmt.Printf("  └─ %s\n", r.Error)
		}
	}
	return nil
}

// filesChangedSince returns files modified between <ref> and HEAD.
func filesChangedSince(projectRoot, ref string) []string {
	out, err := exec.Command("git", "-C", projectRoot, "diff", "--name-only", ref+"..HEAD").Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}
