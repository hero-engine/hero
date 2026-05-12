package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestStatus_AllLinked(t *testing.T) {
	s := &spec.Spec{
		Slug:  "test-spec",
		Title: "Test Spec",
		Sections: map[string]string{
			"acceptance criteria": `- WHEN user logs in THE SYSTEM SHALL create a session
  verified_by: internal/auth/auth_test.go::TestCreateSession
- THE SYSTEM SHALL log failed attempts
  verified_by: internal/auth/auth_test.go::TestLogFailedAttempts`,
		},
	}

	r := Status(s)
	if r.Total != 2 {
		t.Fatalf("expected 2 criteria, got %d", r.Total)
	}
	if r.Linked != 2 {
		t.Errorf("expected 2 linked, got %d", r.Linked)
	}
	if r.Coverage != 100 {
		t.Errorf("expected 100%% coverage, got %.0f%%", r.Coverage)
	}
}

func TestStatus_SomeUnlinked(t *testing.T) {
	s := &spec.Spec{
		Slug: "test-spec",
		Sections: map[string]string{
			"acceptance criteria": `- WHEN user logs in THE SYSTEM SHALL create a session
  verified_by: auth_test.go::TestCreateSession
- THE SYSTEM SHALL log failed attempts`,
		},
	}

	r := Status(s)
	if r.Linked != 1 {
		t.Errorf("expected 1 linked, got %d", r.Linked)
	}
	if r.Total != 2 {
		t.Errorf("expected 2 total, got %d", r.Total)
	}
}

func TestStatus_NoCriteria(t *testing.T) {
	s := &spec.Spec{Slug: "empty", Sections: map[string]string{}}
	r := Status(s)
	if r.Total != 0 {
		t.Errorf("expected 0 criteria, got %d", r.Total)
	}
}

func TestLink(t *testing.T) {
	dir := t.TempDir()

	// Create a test file so validation passes
	testFile := filepath.Join(dir, "auth_test.go")
	os.WriteFile(testFile, []byte("package auth"), 0o644)

	specPath := filepath.Join(dir, "spec.md")
	content := `---
title: Test
status: completed
---

## Acceptance Criteria

- WHEN user logs in THE SYSTEM SHALL create a session
- THE SYSTEM SHALL log failed attempts
`
	os.WriteFile(specPath, []byte(content), 0o644)

	err := Link(specPath, dir, 1, "auth_test.go::TestCreateSession")
	if err != nil {
		t.Fatal(err)
	}

	updated, _ := os.ReadFile(specPath)
	if !strings.Contains(string(updated), "verified_by: auth_test.go::TestCreateSession") {
		t.Error("expected verified_by annotation in spec")
	}
}

func TestLink_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	os.WriteFile(specPath, []byte("## Acceptance Criteria\n- criterion 1\n"), 0o644)

	err := Link(specPath, dir, 1, "nonexistent_test.go::TestFoo")
	if err == nil {
		t.Error("expected error for missing test file")
	}
}

func TestDetectRunner(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"internal/auth/auth_test.go", "go"},
		{"e2e/login.spec.ts", "npx"},
		{"tests/test_auth.py", "pytest"},
		{"unknown.txt", ""},
	}

	for _, tt := range tests {
		runner, _ := detectRunner(spec.TestLink{File: tt.file, Name: "test"})
		if runner != tt.want {
			t.Errorf("detectRunner(%q) = %q, want %q", tt.file, runner, tt.want)
		}
	}
}

func TestRenderText(t *testing.T) {
	r := &ContractReport{
		Slug:  "csv-export",
		Total: 3,
		Linked: 2,
		Coverage: 66.67,
		Criteria: []CriterionStatus{
			{Index: 1, Kind: "event", Raw: "WHEN export exceeds 10MB", Linked: true,
				VerifiedBy: []spec.TestLink{{File: "test.go", Name: "TestStream"}}},
			{Index: 2, Kind: "ubiquitous", Raw: "THE SYSTEM SHALL log", Linked: true,
				VerifiedBy: []spec.TestLink{{File: "test.go", Name: "TestLog"}}},
			{Index: 3, Kind: "freeform", Raw: "support CSV format", Linked: false},
		},
	}
	out := RenderText(r)
	if !strings.Contains(out, "csv-export") {
		t.Error("expected slug in output")
	}
	if !strings.Contains(out, "UNLINKED") {
		t.Error("expected UNLINKED for criterion 3")
	}
	if !strings.Contains(out, "2/3") {
		t.Error("expected coverage fraction")
	}
}

func TestRenderCheckText_NoTests(t *testing.T) {
	out := RenderCheckText(nil)
	if !strings.Contains(out, "No linked tests") {
		t.Error("expected 'No linked tests' message")
	}
}
