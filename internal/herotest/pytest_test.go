package herotest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestPytestName(t *testing.T) {
	fw := &PytestFramework{}
	if fw.Name() != "pytest" {
		t.Errorf("Name() = %q, want %q", fw.Name(), "pytest")
	}
}

func TestPytestRegistered(t *testing.T) {
	fw, err := Get("pytest")
	if err != nil {
		t.Fatalf("Get(pytest) failed: %v", err)
	}
	if fw.Name() != "pytest" {
		t.Errorf("registered Name() = %q, want %q", fw.Name(), "pytest")
	}
}

func TestPytestTestFilePath(t *testing.T) {
	fw := &PytestFramework{}

	// Default (nil config) — note test_ prefix and underscore conversion
	got := fw.TestFilePath("login-flow", nil)
	want := filepath.Join("tests", "test_login_flow.py")
	if got != want {
		t.Errorf("TestFilePath(nil) = %q, want %q", got, want)
	}

	// Custom test dir
	cfg := &config.TestingConfig{TestDir: "src/tests"}
	got = fw.TestFilePath("login-flow", cfg)
	want = filepath.Join("src/tests", "test_login_flow.py")
	if got != want {
		t.Errorf("TestFilePath(custom) = %q, want %q", got, want)
	}
}

func TestPytestRunCommand(t *testing.T) {
	fw := &PytestFramework{}

	// Default
	runner, args := fw.RunCommand("tests/test_login_flow.py", nil)
	if runner != "pytest" {
		t.Errorf("runner = %q, want %q", runner, "pytest")
	}
	if len(args) != 2 || args[0] != "tests/test_login_flow.py" || args[1] != "-v" {
		t.Errorf("args = %v, want [tests/test_login_flow.py -v]", args)
	}

	// Custom runner
	cfg := &config.TestingConfig{RunnerCommand: "python -m pytest"}
	runner, args = fw.RunCommand("test.py", cfg)
	if runner != "python" {
		t.Errorf("runner = %q, want %q", runner, "python")
	}
}

func TestPytestGenerateAssisted(t *testing.T) {
	fw := &PytestFramework{}
	s := makeSpec("login-flow", "Login Flow", []string{
		"User sees login form",
		"Error shown for bad password",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAssisted(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAssisted failed: %v", err)
	}

	// Check import pytest
	if !strings.Contains(content, "import pytest") {
		t.Error("missing pytest import")
	}

	// Check docstring
	if !strings.Contains(content, `"""Tests for login-flow."""`) {
		t.Error("missing module docstring")
	}

	// Check mode comment
	if !strings.Contains(content, "# Mode: assisted") {
		t.Error("missing mode comment")
	}

	// Check @pytest.mark.skip decorator
	if !strings.Contains(content, "@pytest.mark.skip") {
		t.Error("missing @pytest.mark.skip decorator")
	}

	// Check test_ prefix on functions
	if !strings.Contains(content, "def test_") {
		t.Error("missing test_ prefix on function name")
	}

	// Check TODO reason
	if !strings.Contains(content, "TODO: implement") {
		t.Error("missing TODO in skip reason")
	}

	// Check pass body
	if !strings.Contains(content, "    pass") {
		t.Error("missing pass body")
	}

	// Verify snake_case naming
	if !strings.Contains(content, "def test_user_sees_login_form") {
		t.Errorf("expected snake_case function name 'test_user_sees_login_form', got:\n%s", content)
	}
}

func TestPytestGenerateAssistedNoCriteria(t *testing.T) {
	fw := &PytestFramework{}
	s := makeSpec("empty", "Empty", nil)

	_, err := fw.GenerateAssisted(s, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty criteria")
	}
}

func TestPytestGenerateAutonomous(t *testing.T) {
	fw := &PytestFramework{}
	s := makeSpec("dashboard", "Dashboard", []string{
		"Dashboard displays user metrics",
		"Operation should fail on bad data",
		`Page contains text "Welcome"`,
		"Results count should match expected",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAutonomous(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAutonomous failed: %v", err)
	}

	// Check import
	if !strings.Contains(content, "import pytest") {
		t.Error("missing pytest import")
	}

	// Check mode
	if !strings.Contains(content, "# Mode: autonomous") {
		t.Error("missing mode comment")
	}

	// Check assert statements (Python uses assert, not expect)
	if !strings.Contains(content, "assert ") {
		t.Error("missing assert statement")
	}

	// Check visibility assertion
	if !strings.Contains(content, "assert result is not None") {
		t.Error("missing 'assert result is not None' for visible criterion")
	}

	// Check error assertion
	if !strings.Contains(content, "pytest.raises") {
		t.Error("missing pytest.raises for error criterion")
	}

	// Check quoted text
	if !strings.Contains(content, "Welcome") {
		t.Error("missing extracted quoted text 'Welcome'")
	}

	// Check count assertion
	if !strings.Contains(content, "len(items)") {
		t.Error("missing len(items) for count criterion")
	}
}

func TestPytestGenerateAutonomousEARSEvent(t *testing.T) {
	fw := &PytestFramework{}
	s := makeSpec("event-test", "Event Test", []string{
		"WHEN the user clicks submit THE SYSTEM SHALL display a confirmation",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAutonomous(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAutonomous failed: %v", err)
	}

	if !strings.Contains(content, "EARS:event") {
		t.Error("missing EARS:event comment")
	}
	if !strings.Contains(content, "Trigger:") {
		t.Error("missing trigger setup comment")
	}
}

func TestPytestGenerateAutonomousEARSUnwanted(t *testing.T) {
	fw := &PytestFramework{}
	s := makeSpec("unwanted-test", "Unwanted Test", []string{
		"IF the session expires THEN THE SYSTEM SHALL redirect to login",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAutonomous(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAutonomous failed: %v", err)
	}

	if !strings.Contains(content, "EARS:unwanted") {
		t.Error("missing EARS:unwanted comment")
	}
	if !strings.Contains(content, "pytest.raises") {
		t.Error("missing pytest.raises for unwanted criterion")
	}
}

func TestPytestAgentContext(t *testing.T) {
	fw := &PytestFramework{}
	s := makeSpec("checkout", "Checkout Flow", []string{
		"User can add items to cart",
	})
	criteria := ExtractCriteria(s)

	ctx := fw.AgentContext(s, criteria, nil)

	if !strings.Contains(ctx, "pytest") {
		t.Error("missing framework name in agent context")
	}
	if !strings.Contains(ctx, "checkout") {
		t.Error("missing slug in agent context")
	}
	if !strings.Contains(ctx, "User can add items to cart") {
		t.Error("missing criterion in agent context")
	}
	if !strings.Contains(ctx, "fixture") || !strings.Contains(ctx, "Fixture") {
		// Either capitalization
		if !strings.Contains(strings.ToLower(ctx), "fixture") {
			t.Error("missing fixture convention")
		}
	}
	if !strings.Contains(ctx, "parametrize") {
		t.Error("missing parametrize convention")
	}
	if !strings.Contains(ctx, "conftest.py") {
		t.Error("missing conftest.py convention")
	}
}

func TestPytestFileNameConversion(t *testing.T) {
	fw := &PytestFramework{}

	// Hyphens should be converted to underscores in file names
	got := fw.TestFilePath("my-feature-thing", nil)
	want := filepath.Join("tests", "test_my_feature_thing.py")
	if got != want {
		t.Errorf("TestFilePath hyphen conversion = %q, want %q", got, want)
	}
}

func TestEscapePython(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`it's a test`, `it\'s a test`},
		{`has "quotes"`, `has \"quotes\"`},
		{`back\slash`, `back\\slash`},
		{"simple", "simple"},
	}
	for _, tt := range tests {
		got := escapePython(tt.input)
		if got != tt.want {
			t.Errorf("escapePython(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
