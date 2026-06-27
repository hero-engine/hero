package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func TestIntakeCaptureCreatesSpec(t *testing.T) {
	env := newTestEnv(t)

	out, err := runCmd("intake", "let users export to CSV")
	if err != nil {
		t.Fatalf("intake capture error: %v", err)
	}
	if !strings.Contains(out, "Created intake") {
		t.Errorf("output missing 'Created intake': %q", out)
	}

	specPath := filepath.Join(env.heroDir, "planning", "intake", "let-users-export-to-csv", "spec.md")
	content := readFile(t, specPath)
	if !strings.Contains(content, "type: intake") {
		t.Error("spec.md missing type: intake")
	}
	if !strings.Contains(content, "status: planning") {
		t.Error("spec.md missing status: planning")
	}
	if !strings.Contains(content, "## Signal") {
		t.Error("spec.md missing Signal section")
	}
}

func TestIntakeList(t *testing.T) {
	newTestEnv(t)

	if _, err := runCmd("intake", "first-idea"); err != nil {
		t.Fatalf("capture: %v", err)
	}
	out, err := runCmd("intake", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "first-idea") {
		t.Errorf("list missing captured intake: %q", out)
	}
	if !strings.Contains(out, "planning") {
		t.Errorf("list missing status: %q", out)
	}
}

func TestIntakePromoteCreatesFeatureWithProvenance(t *testing.T) {
	env := newTestEnv(t)

	if _, err := runCmd("intake", "csv-export"); err != nil {
		t.Fatalf("capture: %v", err)
	}
	out, err := runCmd("intake", "promote", "csv-export")
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if !strings.Contains(out, "Promoted intake") {
		t.Errorf("output missing 'Promoted intake': %q", out)
	}

	// New feature spec carries a derived_from relation back to the intake.
	featurePath := filepath.Join(env.heroDir, "planning", "features", "csv-export", "spec.md")
	feature := readFile(t, featurePath)
	if !strings.Contains(feature, "type: feature") {
		t.Error("promoted spec missing type: feature")
	}
	if !strings.Contains(feature, "kind: derived_from") || !strings.Contains(feature, "target: csv-export") {
		t.Errorf("promoted spec missing derived_from relation:\n%s", feature)
	}

	// Intake is now terminal-promoted and records where it went.
	intakePath := filepath.Join(env.heroDir, "planning", "intake", "csv-export", "spec.md")
	intake := readFile(t, intakePath)
	if !strings.Contains(intake, "status: promoted") {
		t.Errorf("intake not marked promoted:\n%s", intake)
	}
	if !strings.Contains(intake, "promoted_to: csv-export") {
		t.Errorf("intake missing promoted_to:\n%s", intake)
	}
}

func TestIntakePromoteBugType(t *testing.T) {
	env := newTestEnv(t)

	if _, err := runCmd("intake", "flaky-thing"); err != nil {
		t.Fatalf("capture: %v", err)
	}
	if _, err := runCmd("intake", "promote", "flaky-thing", "--type", "bug"); err != nil {
		t.Fatalf("promote --type bug: %v", err)
	}
	bugPath := filepath.Join(env.heroDir, "planning", "bugs", "flaky-thing", "spec.md")
	if !strings.Contains(readFile(t, bugPath), "type: bug") {
		t.Error("promoted spec is not a bug")
	}
}

func TestIntakeReject(t *testing.T) {
	env := newTestEnv(t)

	if _, err := runCmd("intake", "stale-idea"); err != nil {
		t.Fatalf("capture: %v", err)
	}
	if _, err := runCmd("intake", "reject", "stale-idea"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	intakePath := filepath.Join(env.heroDir, "planning", "intake", "stale-idea", "spec.md")
	if !strings.Contains(readFile(t, intakePath), "status: rejected") {
		t.Error("intake not marked rejected")
	}
}

// TestIntakeAbsentFromStatusWorkBuckets is the no-leak guard at the CLI
// surface: a planning-status intake must surface only in the dedicated
// pre-commitment section, never the work "Planning" bucket.
func TestIntakeAbsentFromStatusWorkBuckets(t *testing.T) {
	newTestEnv(t)

	if _, err := runCmd("intake", "brewing-idea"); err != nil {
		t.Fatalf("capture: %v", err)
	}
	out, err := runCmd("status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, "Intake — pre-commitment") {
		t.Errorf("status missing pre-commitment section:\n%s", out)
	}

	// The intake slug must not appear under the work "Planning" header.
	if planningIdx := strings.Index(out, "Planning"); planningIdx >= 0 {
		intakeIdx := strings.Index(out, "Intake — pre-commitment")
		brewingIdx := strings.Index(out, "brewing-idea")
		if brewingIdx >= 0 && intakeIdx >= 0 && brewingIdx < intakeIdx {
			t.Errorf("intake 'brewing-idea' appeared before the pre-commitment section (leaked into work buckets):\n%s", out)
		}
	}
}
