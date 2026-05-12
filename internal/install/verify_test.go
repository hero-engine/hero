package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// verify_test.go — coverage for `hero verify-install`.

// TestVerify_CleanSymlinkInstall_NoIssues asserts a freshly-installed
// project (P2 layout with symlinks pointing at canonical) reports clean.
func TestVerify_CleanSymlinkInstall_NoIssues(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.Run(TargetClaude, nil)

	report, err := RunVerify(h.TargetDir)
	if err != nil {
		t.Fatalf("RunVerify: %v\n%s", err, report.StringReport())
	}
	if !report.Clean {
		t.Errorf("expected clean report, got issues:\n%s", report.StringReport())
	}
	if report.HasErrors() {
		t.Errorf("expected no errors, got:\n%s", report.StringReport())
	}
}

// TestVerify_RegularDir_ReportsExpectedSymlink seeds a legacy-multi-harness-shape
// state (regular directory full of files instead of symlink to
// canonical) and asserts verify flags it.
func TestVerify_RegularDir_ReportsExpectedSymlink(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Seed legacy .claude/agents as a regular dir.
	legacyDir := filepath.Join(h.TargetDir, ".claude", "agents")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "engineer.md"), []byte("rendered\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := RunVerify(h.TargetDir)
	if err != nil {
		t.Fatalf("RunVerify: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Code == "expected_symlink" && strings.Contains(issue.Path, ".claude/agents") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected an expected_symlink issue for .claude/agents; got:\n%s", report.StringReport())
	}
}

// TestVerify_BrokenSymlink_ReportsError places a symlink with a
// non-existent target alongside a working content dir, and asserts
// verify flags the broken symlink as an error.
func TestVerify_BrokenSymlink_ReportsError(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero", "skills", "spec-format"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.TargetDir, ".hero/skills/spec-format/SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	claudeDir := filepath.Join(h.TargetDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Working commands dir so the target is detected as installed.
	if err := os.MkdirAll(filepath.Join(claudeDir, "commands"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Broken agents symlink — target dir doesn't exist.
	if err := os.Symlink("../.hero/agents", filepath.Join(claudeDir, "agents")); err != nil {
		t.Skip("symlinks unavailable on this host")
	}

	report, err := RunVerify(h.TargetDir)
	if err != nil {
		t.Fatalf("RunVerify: %v", err)
	}

	if !report.HasErrors() {
		t.Errorf("expected error-severity issue for broken symlink; got:\n%s", report.StringReport())
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Code == "broken_symlink" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected broken_symlink code, got:\n%s", report.StringReport())
	}
}

// TestVerify_SymlinkEscape_ReportsError places a symlink pointing
// outside the project root.
func TestVerify_SymlinkEscape_ReportsError(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a sibling target outside the project root.
	externalDir, err := os.MkdirTemp("", "external-target-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(externalDir)
	if err := os.WriteFile(filepath.Join(externalDir, "x.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	claudeDir := filepath.Join(h.TargetDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalDir, filepath.Join(claudeDir, "agents")); err != nil {
		t.Skip("symlinks unavailable on this host")
	}

	report, err := RunVerify(h.TargetDir)
	if err != nil {
		t.Fatalf("RunVerify: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Code == "symlink_escape" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected symlink_escape issue; got:\n%s", report.StringReport())
	}
	if !report.HasErrors() {
		t.Errorf("symlink_escape should be error-severity")
	}
}

// TestVerify_DriftedRendered_ReportsPerFile asserts the rendered-mode
// drift check: a harness dir is a regular directory whose files differ
// from the canonical equivalents.
func TestVerify_DriftedRendered_ReportsPerFile(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Canonical content.
	canonicalAgents := filepath.Join(h.TargetDir, ".hero", "agents")
	if err := os.MkdirAll(canonicalAgents, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonicalAgents, "engineer.md"), []byte("canonical v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Rendered (drifted) copy in .claude/.
	legacyDir := filepath.Join(h.TargetDir, ".claude", "agents")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "engineer.md"), []byte("stale v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := RunVerify(h.TargetDir)
	if err != nil {
		t.Fatalf("RunVerify: %v", err)
	}

	foundDrift := false
	for _, issue := range report.Issues {
		if issue.Code == "drifted_rendered" && strings.Contains(issue.Path, "engineer.md") {
			foundDrift = true
		}
	}
	if !foundDrift {
		t.Errorf("expected drifted_rendered issue for engineer.md; got:\n%s", report.StringReport())
	}
}

// TestVerify_NoTargetsDetected_ReturnsClean asserts a project with no
// harness directories at all returns a clean report (nothing to
// verify).
func TestVerify_NoTargetsDetected_ReturnsClean(t *testing.T) {
	h := newInstallHarness(t)
	if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := RunVerify(h.TargetDir)
	if err != nil {
		t.Fatalf("RunVerify: %v", err)
	}
	if !report.Clean {
		t.Errorf("expected clean (no targets, nothing to verify); got:\n%s", report.StringReport())
	}
}
