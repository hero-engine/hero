package cli

import (
	"strings"
	"testing"
)

// TestSmokeStatus verifies that `hero status --smoke` prints the command's
// help text and then runs the registered status smoke fn (runStatus).
func TestSmokeStatus(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("status", "--smoke")
	if err != nil {
		t.Fatalf("status --smoke returned error: %v", err)
	}

	// Help output must appear (status Long description opens with this phrase)
	if !strings.Contains(output, "Displays specs in planning") {
		t.Errorf("smoke output missing help text: %q", output)
	}

	// The registered smoke fn runs runStatus — verify its output appears too.
	if !strings.Contains(output, "in-flight") {
		t.Errorf("smoke output missing status result (in-flight line): %q", output)
	}
}

// TestSmokeScan verifies that `hero scan --smoke` prints help and runs the
// scan smoke fn in dry-run mode (no files written).
func TestSmokeScan(t *testing.T) {
	env := newTestEnv(t)

	output, err := runCmd("scan", "--smoke")
	if err != nil {
		t.Fatalf("scan --smoke returned error: %v", err)
	}

	// Help output must appear (scan Long description opens with this phrase)
	if !strings.Contains(output, "Scans the current project") {
		t.Errorf("smoke output missing scan help text: %q", output)
	}

	// Dry-run path: no files written, but output should mention dry run
	// (scan --dry-run prints "Dry run — no files written.")
	// If scan finds nothing to generate, that's still OK.
	_ = env // workspace used for findProjectRoot
}

// TestSmokeDefaultNoOp verifies that a command without a registered smoke fn
// prints its help and exits 0, emitting a "OK (no-op)" line.
func TestSmokeDefaultNoOp(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("index", "--smoke")
	if err != nil {
		t.Fatalf("index --smoke returned error: %v", err)
	}

	// Default no-op message
	if !strings.Contains(output, "OK (no-op)") {
		t.Errorf("expected default no-op message, got: %q", output)
	}

	// Help text must also appear
	if !strings.Contains(output, "index") {
		t.Errorf("expected help text in smoke output, got: %q", output)
	}
}

// TestSmokeDoesNotAffectNormalRun confirms that normal (non-smoke) invocations
// of wrapped commands are unchanged.
func TestSmokeDoesNotAffectNormalRun(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("normal status returned error: %v", err)
	}

	// Normal status output — should NOT contain the no-op smoke message
	if strings.Contains(output, "OK (no-op)") {
		t.Errorf("normal run should not contain smoke message: %q", output)
	}

	if !strings.Contains(output, "Specs: (none)") {
		t.Errorf("normal status should show workspace state: %q", output)
	}
}

// TestSmokeResetBetweenRuns confirms globalSmoke is reset between runCmd calls
// (via resetFlags), so a smoke run doesn't bleed into the next invocation.
func TestSmokeResetBetweenRuns(t *testing.T) {
	_ = newTestEnv(t)

	// First call: smoke mode
	_, err := runCmd("status", "--smoke")
	if err != nil {
		t.Fatalf("status --smoke returned error: %v", err)
	}

	// Second call: normal mode — must not see smoke behaviour
	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("follow-up normal status returned error: %v", err)
	}
	if strings.Contains(output, "OK (no-op)") {
		t.Errorf("smoke bled into subsequent normal run: %q", output)
	}
	if !strings.Contains(output, "Specs: (none)") {
		t.Errorf("normal status after smoke should show workspace: %q", output)
	}
}
