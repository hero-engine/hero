package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/coverage"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var (
	verifyJSON      bool
	verifySkipTests bool
	verifyForce     bool
)

var verifyCmd = &cobra.Command{
	Use:   "verify <spec-slug>",
	Short: "Verify delivery gates and complete a spec",
	Long: `Checks four delivery gates before marking a spec completed:

  Gate 1: Completion Ledger — present, all AC rows DONE (or signed-off)
  Gate 2: Delivery Audit — audit report exists with SHIP verdict
  Gate 3: Test Coverage — AC-to-test mapping (advisory, not blocking)
  Gate 4: Build & Tests — test command passes

If all hard gates pass, flips status to completed and archives the spec.
If any gate fails, reports the failures and does NOT change status.

Use --force to bypass gates (logged visibly). Use --skip-tests to skip Gate 4.
Use --json for machine-readable output.`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

func init() {
	verifyCmd.Flags().BoolVar(&verifyJSON, "json", false, "output JSON result")
	verifyCmd.Flags().BoolVar(&verifySkipTests, "skip-tests", false, "skip Gate 4 (build & tests)")
	verifyCmd.Flags().BoolVar(&verifyForce, "force", false, "bypass failed gates (logged visibly)")
}

// GateResult is the outcome of a single verification gate.
type GateResult struct {
	Name    string   `json:"name"`
	Result  string   `json:"result"` // PASS, FAIL, ADVISORY, SKIPPED
	Details []string `json:"details"`
}

// VerifyResult is the overall verification outcome.
type VerifyResult struct {
	Slug                string       `json:"slug"`
	Result              string       `json:"result"` // PASS, FAIL, FORCED
	Gates               []GateResult `json:"gates"`
	Archived            bool         `json:"archived"`
	ACStatusUpdates     int          `json:"ac_status_updates,omitempty"`
	InitiativeCompleted string       `json:"initiative_completed,omitempty"`
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

	target, hint := spec.ResolveOrHint(args[0], specs)
	if target == nil {
		if hint != "" {
			return fmt.Errorf("%s", hint)
		}
		return fmt.Errorf("spec %q not found", args[0])
	}

	// If already completed and archived, report and exit
	if target.Status == spec.StatusCompleted && isAlreadyInSpecsDir(target.Path, heroDir) {
		if verifyJSON {
			r := VerifyResult{Slug: target.Slug, Result: "PASS", Archived: true}
			r.Gates = append(r.Gates, GateResult{Name: "already-completed", Result: "PASS", Details: []string{"spec is already completed and archived"}})
			return outputJSON(r)
		}
		fmt.Printf("\n  Spec %s is already completed and archived.\n\n", target.Slug)
		return nil
	}

	result := VerifyResult{Slug: target.Slug}

	// Gate 1: Completion Ledger
	gate1 := checkLedger(target)
	result.Gates = append(result.Gates, gate1)

	// Gate 2: Delivery Audit
	gate2 := checkAudit(target)
	result.Gates = append(result.Gates, gate2)

	// Gate 3: Test Coverage (advisory)
	gate3 := checkCoverage(projectRoot, heroDir, target, cfg)
	result.Gates = append(result.Gates, gate3)

	// Gate 4: Build & Tests
	gate4 := checkTests(projectRoot, cfg)
	result.Gates = append(result.Gates, gate4)

	// Determine overall result
	hardFails := 0
	for _, g := range result.Gates {
		if g.Result == "FAIL" {
			hardFails++
		}
	}

	if hardFails == 0 {
		result.Result = "PASS"
	} else if verifyForce {
		result.Result = "FORCED"
	} else {
		result.Result = "FAIL"
	}

	// Ledger → AC graph writeback: flip DONE criteria to passing.
	if gate1.Result == "PASS" {
		ledger := spec.ParseLedger(target)
		if len(ledger.ACRows) > 0 {
			acUpdates := recordLedgerToGraph(target.Slug, ledger, heroDir, projectRoot)
			result.ACStatusUpdates = acUpdates
		}
	}

	// Archive on PASS or FORCED. Skip the gate check in autoArchive
	// because verify already checked the gates (PASS) or the user
	// explicitly bypassed them (FORCED).
	if result.Result == "PASS" || result.Result == "FORCED" {
		moved, err := completeAndArchive(target.Path, heroDir, true)
		if err != nil {
			return fmt.Errorf("completing spec: %w", err)
		}
		result.Archived = moved || isAlreadyInSpecsDir(target.Path, heroDir)

		// Auto-complete parent initiative if all children are done.
		if slug := autoCompleteParentIfReady(target, heroDir); slug != "" {
			result.InitiativeCompleted = slug
		}
	}

	if verifyJSON {
		return outputJSON(result)
	}

	printVerifyReport(result, hardFails)
	if result.ACStatusUpdates > 0 {
		fmt.Printf("  AC graph: %d criteria flipped to passing\n", result.ACStatusUpdates)
	}
	if result.InitiativeCompleted != "" {
		fmt.Printf("  Initiative %q auto-completed — all children delivered\n", result.InitiativeCompleted)
	}
	if result.ACStatusUpdates > 0 || result.InitiativeCompleted != "" {
		fmt.Println()
	}
	if result.Result == "FAIL" {
		return fmt.Errorf("verification failed: %d gate(s) did not pass", hardFails)
	}
	return nil
}

// checkLedger verifies Gate 1: Completion Ledger present and clean.
func checkLedger(s *spec.Spec) GateResult {
	gate := GateResult{Name: "Completion Ledger"}
	ledger := spec.ParseLedger(s)

	if !ledger.Found {
		gate.Result = "FAIL"
		gate.Details = append(gate.Details, "no Completion Ledger section found in spec")
		return gate
	}
	gate.Details = append(gate.Details, "ledger found")

	// Check AC rows
	allDone := true
	acCount := len(ledger.ACRows)
	doneCount := 0
	for _, row := range ledger.ACRows {
		switch row.Status {
		case spec.LedgerDone:
			doneCount++
		case spec.LedgerSkipped, spec.LedgerBlocked:
			if row.SignedOff {
				doneCount++
			} else {
				allDone = false
				gate.Details = append(gate.Details,
					fmt.Sprintf("AC-%d is %s (not signed-off): %s", row.Index, row.Status, row.Note))
			}
		case spec.LedgerPartial:
			allDone = false
			gate.Details = append(gate.Details,
				fmt.Sprintf("AC-%d is PARTIAL: %s", row.Index, row.Note))
		default:
			allDone = false
			gate.Details = append(gate.Details,
				fmt.Sprintf("AC-%d has unrecognized status: %s", row.Index, row.Status))
		}
	}

	if acCount > 0 {
		gate.Details = append([]string{gate.Details[0],
			fmt.Sprintf("%d/%d AC rows DONE", doneCount, acCount)}, gate.Details[1:]...)
	}

	// Check Changes rows
	for _, row := range ledger.ChangesRows {
		if row.Status != spec.LedgerDone && !(row.SignedOff && (row.Status == spec.LedgerSkipped || row.Status == spec.LedgerBlocked)) {
			allDone = false
			gate.Details = append(gate.Details,
				fmt.Sprintf("Changes item %d is %s: %s", row.Index, row.Status, row.Note))
		}
	}
	if len(ledger.ChangesRows) > 0 {
		changesOK := 0
		for _, row := range ledger.ChangesRows {
			if row.Status == spec.LedgerDone || (row.SignedOff && (row.Status == spec.LedgerSkipped || row.Status == spec.LedgerBlocked)) {
				changesOK++
			}
		}
		gate.Details = append(gate.Details,
			fmt.Sprintf("%d/%d Changes rows DONE", changesOK, len(ledger.ChangesRows)))
	}

	// Exercise-the-feature is advisory — it nudges toward regression
	// tests but does not block the gate from passing.
	if ledger.ExerciseChecked && ledger.ExerciseDetail != "" {
		gate.Details = append(gate.Details, "exercise-the-feature: checked with detail")
	} else if ledger.ExerciseChecked && ledger.ExerciseDetail == "" {
		gate.Details = append(gate.Details,
			"ADVISORY: exercise-the-feature checked but no detail — consider a regression test")
	} else {
		gate.Details = append(gate.Details,
			"ADVISORY: exercise-the-feature not checked — consider a regression test for this behavior")
	}

	if allDone {
		gate.Result = "PASS"
	} else {
		gate.Result = "FAIL"
	}
	return gate
}

// checkAudit verifies Gate 2: Delivery audit report exists with SHIP verdict.
func checkAudit(s *spec.Spec) GateResult {
	gate := GateResult{Name: "Delivery Audit"}
	audit := spec.FindAuditReport(s)

	if !audit.Found {
		gate.Result = "FAIL"
		gate.Details = append(gate.Details, "no audit report found (expected delivery-audit.md in spec directory)")
		return gate
	}

	gate.Details = append(gate.Details, fmt.Sprintf("audit report found at %s", audit.Path))

	if audit.Verdict == "SHIP" {
		gate.Result = "PASS"
		surface := audit.Surface
		if surface == "" {
			surface = "unknown"
		}
		gate.Details = append(gate.Details, fmt.Sprintf("verdict: SHIP (%s)", surface))
	} else if audit.Verdict == "HOLD" {
		gate.Result = "FAIL"
		gate.Details = append(gate.Details, "verdict: HOLD — audit found issues that need resolution")
	} else {
		gate.Result = "FAIL"
		gate.Details = append(gate.Details, fmt.Sprintf("verdict: %q — expected SHIP or HOLD", audit.Verdict))
	}

	return gate
}

// checkCoverage verifies Gate 3: Test coverage for acceptance criteria (advisory).
func checkCoverage(projectRoot, heroDir string, s *spec.Spec, cfg config.Config) GateResult {
	gate := GateResult{Name: "Test Coverage"}

	testDir := ""
	if cfg.Testing != nil && cfg.Testing.TestDir != "" {
		testDir = cfg.Testing.TestDir
	}

	report, err := coverage.Analyze(projectRoot, heroDir, s.Slug, testDir)
	if err != nil {
		gate.Result = "ADVISORY"
		gate.Details = append(gate.Details, fmt.Sprintf("could not analyze coverage: %v", err))
		return gate
	}

	if report.Total == 0 {
		gate.Result = "ADVISORY"
		gate.Details = append(gate.Details, "no acceptance criteria to check coverage for")
		return gate
	}

	gate.Details = append(gate.Details,
		fmt.Sprintf("%d/%d criteria have test coverage (%d strong, %d weak)",
			report.Covered, report.Total, report.StrongCount, report.WeakCount))

	if report.Gaps > 0 {
		gate.Result = "ADVISORY"
		for _, c := range report.Criteria {
			if !c.Covered {
				gate.Details = append(gate.Details,
					fmt.Sprintf("gap: AC-%d %q — %s",
						c.Index, truncateForVerify(c.Raw, 60), c.Detail))
			}
		}
	} else {
		gate.Result = "PASS"
	}

	return gate
}

// checkTests verifies Gate 4: Build & tests pass.
func checkTests(projectRoot string, cfg config.Config) GateResult {
	gate := GateResult{Name: "Build & Tests"}

	if verifySkipTests {
		gate.Result = "SKIPPED"
		gate.Details = append(gate.Details, "skipped via --skip-tests")
		return gate
	}

	if cfg.Verify != nil && !cfg.Verify.RunTestsOrDefault() {
		gate.Result = "SKIPPED"
		gate.Details = append(gate.Details, "disabled via verify.run_tests=false in hero.json")
		return gate
	}

	testCmd := detectTestCommand(projectRoot, cfg)
	if testCmd == "" {
		gate.Result = "SKIPPED"
		gate.Details = append(gate.Details, "no test command detected — set verify.test_command in hero.json")
		return gate
	}

	gate.Details = append(gate.Details, fmt.Sprintf("running: %s", testCmd))

	cmd := exec.Command("sh", "-c", testCmd)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		gate.Result = "FAIL"
		// Truncate output to last 30 lines for readability
		lines := strings.Split(string(output), "\n")
		if len(lines) > 30 {
			lines = lines[len(lines)-30:]
		}
		gate.Details = append(gate.Details, fmt.Sprintf("FAILED: %v", err))
		gate.Details = append(gate.Details, strings.Join(lines, "\n"))
	} else {
		gate.Result = "PASS"
		// Count packages/tests from output
		summary := summarizeTestOutput(string(output))
		if summary != "" {
			gate.Details = append(gate.Details, summary)
		} else {
			gate.Details = append(gate.Details, "tests passed")
		}
	}

	return gate
}

// detectTestCommand determines the test command from config or stack detection.
func detectTestCommand(projectRoot string, cfg config.Config) string {
	// Check config override first
	if cfg.Verify != nil && cfg.Verify.TestCommandOrDefault() != "" {
		return cfg.Verify.TestCommandOrDefault()
	}

	// Stack detection
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
		return "go build ./... && go test ./..."
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "package.json")); err == nil {
		return "npm test"
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "pyproject.toml")); err == nil {
		return "pytest"
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "Cargo.toml")); err == nil {
		return "cargo test"
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "build.gradle")); err == nil {
		return "./gradlew test"
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "pom.xml")); err == nil {
		return "mvn test"
	}

	return ""
}

// summarizeTestOutput extracts a summary line from test output.
func summarizeTestOutput(output string) string {
	lines := strings.Split(output, "\n")
	okCount := 0
	failCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ok ") || strings.HasPrefix(trimmed, "ok\t") {
			okCount++
		}
		if strings.HasPrefix(trimmed, "FAIL") {
			failCount++
		}
	}
	if okCount > 0 || failCount > 0 {
		return fmt.Sprintf("%d packages ok, %d failed", okCount, failCount)
	}
	return ""
}

// completeAndArchive flips status to completed and archives.
// skipGateCheck bypasses the ledger/audit gate in autoArchive
// (used when verify itself already checked the gates or --force was used).
func completeAndArchive(specPath, heroDir string, skipGateCheck bool) (bool, error) {
	// Read the current spec to check status
	s, err := spec.ParseFile(specPath)
	if err != nil {
		return false, fmt.Errorf("parsing spec: %w", err)
	}

	// Flip status to completed if not already
	if s.Status != spec.StatusCompleted {
		if err := updateFrontmatterStatus(specPath, "completed"); err != nil {
			return false, fmt.Errorf("updating status: %w", err)
		}
	}

	// Use the existing auto-archive path. Skip the gate check because
	// verify already validated the gates (or --force was used).
	return autoArchiveIfCompletedOpt(specPath, heroDir, skipGateCheck)
}

// printVerifyReport outputs a human-readable verification report.
func printVerifyReport(r VerifyResult, hardFails int) {
	fmt.Printf("\n  Delivery Gate Report: %s\n", r.Slug)
	fmt.Printf("  %s\n", strings.Repeat("=", 60))

	for _, g := range r.Gates {
		symbol := "PASS"
		switch g.Result {
		case "PASS":
			symbol = "PASS"
		case "FAIL":
			symbol = "FAIL"
		case "ADVISORY":
			symbol = "ADVISORY"
		case "SKIPPED":
			symbol = "SKIPPED"
		}

		fmt.Printf("\n  %-35s %s\n", g.Name, symbol)
		for _, d := range g.Details {
			prefix := "    "
			switch {
			case strings.HasPrefix(d, "gap:") || strings.Contains(d, "PARTIAL") ||
				strings.Contains(d, "SKIPPED") || strings.Contains(d, "BLOCKED") ||
				strings.Contains(d, "FAIL") || strings.Contains(d, "not checked") ||
				strings.Contains(d, "no detail"):
				prefix = "    ! "
			default:
				prefix = "    "
			}
			fmt.Printf("%s%s\n", prefix, d)
		}
	}

	fmt.Printf("\n  %s\n", strings.Repeat("-", 60))

	switch r.Result {
	case "PASS":
		fmt.Printf("  Result: PASS — all gates satisfied\n")
		if r.Archived {
			fmt.Printf("  -> Status flipped to completed\n")
			fmt.Printf("  -> Archived to specs/%s/\n", r.Slug)
		}
	case "FORCED":
		fmt.Printf("  Result: FORCED — %d gate(s) bypassed\n", hardFails)
		fmt.Printf("  WARNING: gates were bypassed with --force\n")
		if r.Archived {
			fmt.Printf("  -> Status flipped to completed (FORCED)\n")
			fmt.Printf("  -> Archived to specs/%s/\n", r.Slug)
		}
	case "FAIL":
		fmt.Printf("  Result: FAIL — %d gate(s) did not pass\n", hardFails)
		fmt.Printf("  -> Status NOT changed. Fix the failures and re-run hero verify.\n")
	}
	fmt.Println()
}

func outputJSON(r VerifyResult) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// extractCriteriaItems pulls list items from a section. Retained for
// backward compatibility — used by deliver_test.go.
func extractCriteriaItems(section string) []string {
	var items []string
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 3 {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			items = append(items, strings.TrimSpace(trimmed[2:]))
		} else if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
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

// truncateForVerify shortens s to max chars. Separate from the
// truncateStr in automations.go to avoid redeclaration.
func truncateForVerify(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// recordLedgerToGraph converts DONE ledger rows into acceptance.RunResult
// entries and records them to the AC graph, flipping Criterion nodes to
// "passing". Returns the number of criteria updated.
func recordLedgerToGraph(slug string, ledger spec.LedgerResult, heroDir, projectRoot string) int {
	var results []acceptance.RunResult
	for _, row := range ledger.ACRows {
		if row.Status != spec.LedgerDone {
			continue
		}
		results = append(results, acceptance.RunResult{
			AC:     fmt.Sprintf("%s:AC-%d", slug, row.Index),
			Status: "pass",
		})
	}
	if len(results) == 0 {
		return 0
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return 0
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(projectRoot)
	summary, err := acceptance.Record(results, repoKey, store)
	if err != nil {
		return 0
	}
	return summary.Criteria
}

// autoCompleteParentIfReady checks whether the completed spec's parent
// initiative has all children done. If so, it auto-completes and archives
// the parent. Returns the parent slug if completed, empty string otherwise.
func autoCompleteParentIfReady(target *spec.Spec, heroDir string) string {
	for _, rel := range target.Relations {
		if rel.Kind != "parent" && rel.Kind != "child-of" {
			continue
		}
		parentSlug := normalizeVerifyParentTarget(rel.Target)

		allSpecs, err := spec.Discover(heroDir)
		if err != nil {
			continue
		}

		var parent *spec.Spec
		for _, s := range allSpecs {
			if s.Slug == parentSlug {
				parent = s
				break
			}
		}
		if parent == nil || parent.Type != spec.TypeInitiative {
			continue
		}
		if parent.Status == spec.StatusCompleted {
			continue
		}

		allDone := true
		childCount := 0
		for _, s := range allSpecs {
			for _, r := range s.Relations {
				if (r.Kind == "parent" || r.Kind == "child-of") &&
					normalizeVerifyParentTarget(r.Target) == parentSlug {
					childCount++
					if s.Status != spec.StatusCompleted {
						allDone = false
					}
					break
				}
			}
		}
		if childCount == 0 || !allDone {
			continue
		}

		if _, err := completeAndArchive(parent.Path, heroDir, true); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not auto-complete initiative %q: %v\n", parentSlug, err)
			continue
		}
		return parentSlug
	}
	return ""
}

// normalizeVerifyParentTarget converts a parent reference to a slug.
// Handles both slug format ("hero-cli") and relative-path format
// ("../../initiatives/hero-cli/spec.md").
func normalizeVerifyParentTarget(target string) string {
	if !strings.Contains(target, "/") {
		return target
	}
	dir := filepath.Dir(target)
	return filepath.Base(dir)
}
