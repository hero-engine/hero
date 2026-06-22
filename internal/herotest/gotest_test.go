package herotest

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestGoTestName(t *testing.T) {
	fw := &GoTestFramework{}
	if fw.Name() != "go" {
		t.Errorf("Name() = %q, want %q", fw.Name(), "go")
	}
}

func TestGoTestRegistered(t *testing.T) {
	fw, err := Get("go")
	if err != nil {
		t.Fatalf("Get(go) failed: %v", err)
	}
	if fw.Name() != "go" {
		t.Errorf("registered Name() = %q, want %q", fw.Name(), "go")
	}
}

func TestGoTestTestFilePath(t *testing.T) {
	fw := &GoTestFramework{}

	// Default (nil config)
	got := fw.TestFilePath("login-flow", nil)
	want := filepath.Join(".", "login_flow_test.go")
	if got != want {
		t.Errorf("TestFilePath(nil) = %q, want %q", got, want)
	}

	// Custom test dir
	cfg := &config.TestingConfig{TestDir: "internal/auth"}
	got = fw.TestFilePath("login-flow", cfg)
	want = filepath.Join("internal/auth", "login_flow_test.go")
	if got != want {
		t.Errorf("TestFilePath(custom) = %q, want %q", got, want)
	}
}

func TestGoTestRunCommand(t *testing.T) {
	fw := &GoTestFramework{}

	// Default
	runner, args := fw.RunCommand("login_flow_test.go", nil)
	if runner != "go" {
		t.Errorf("runner = %q, want %q", runner, "go")
	}
	if len(args) < 4 || args[0] != "test" || args[1] != "-run" {
		t.Errorf("args = %v, want [test -run TestLoginFlow -v ./...]", args)
	}
	// Verify pascal-case conversion
	if !strings.HasPrefix(args[2], "Test") {
		t.Errorf("run pattern = %q, want to start with 'Test'", args[2])
	}

	// Custom runner
	cfg := &config.TestingConfig{RunnerCommand: "gotestsum --format standard-verbose"}
	runner, args = fw.RunCommand("test.go", cfg)
	if runner != "gotestsum" {
		t.Errorf("runner = %q, want %q", runner, "gotestsum")
	}
}

func TestGoTestGenerateAssisted(t *testing.T) {
	fw := &GoTestFramework{}
	s := makeSpec("login-flow", "Login Flow", []string{
		"User sees login form",
		"Error shown for bad password",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAssisted(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAssisted failed: %v", err)
	}

	// Check package declaration
	if !strings.Contains(content, "package ") {
		t.Error("missing package declaration")
	}

	// Check import
	if !strings.Contains(content, `import "testing"`) {
		t.Error("missing testing import")
	}

	// Check mode comment
	if !strings.Contains(content, "// Mode: assisted") {
		t.Error("missing mode comment")
	}

	// Check function name
	if !strings.Contains(content, "func TestLoginFlow(t *testing.T)") {
		t.Error("missing TestLoginFlow function")
	}

	// Check t.Skip
	if !strings.Contains(content, "t.Skip(\"TODO: implement\")") {
		t.Error("missing t.Skip call")
	}

	// Check t.Run subtests
	if !strings.Contains(content, "t.Run(") {
		t.Error("missing t.Run subtest")
	}

	// Verify valid Go syntax
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "test.go", content, parser.AllErrors)
	if err != nil {
		t.Errorf("generated Go code does not parse: %v\nContent:\n%s", err, content)
	}
}

func TestGoTestGenerateAssistedNoCriteria(t *testing.T) {
	fw := &GoTestFramework{}
	s := makeSpec("empty", "Empty", nil)

	_, err := fw.GenerateAssisted(s, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty criteria")
	}
}

func TestGoTestGenerateAutonomous(t *testing.T) {
	fw := &GoTestFramework{}
	s := makeSpec("dashboard", "Dashboard", []string{
		"Dashboard displays user metrics",
		"Error message shown for invalid input",
		`Page contains text "Welcome"`,
		"Results count should match expected",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAutonomous(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAutonomous failed: %v", err)
	}

	// Check mode
	if !strings.Contains(content, "// Mode: autonomous") {
		t.Error("missing mode comment")
	}

	// Check assertions present
	if !strings.Contains(content, "t.Errorf") || !strings.Contains(content, "t.Fatal") {
		// At least one assertion style should be present
		if !strings.Contains(content, "t.Skip") && !strings.Contains(content, "t.Errorf") {
			t.Error("missing test assertions")
		}
	}

	// Check quoted text extraction
	if !strings.Contains(content, "Welcome") {
		t.Error("missing extracted quoted text")
	}
}

func TestGoTestGenerateAutonomousEARSEvent(t *testing.T) {
	fw := &GoTestFramework{}
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

func TestGoTestGenerateAutonomousEARSUnwanted(t *testing.T) {
	fw := &GoTestFramework{}
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
	if !strings.Contains(content, "t.Fatal") {
		t.Error("missing t.Fatal for unwanted criterion")
	}
}

func TestGoTestAgentContext(t *testing.T) {
	fw := &GoTestFramework{}
	s := makeSpec("checkout", "Checkout Flow", []string{
		"User can add items to cart",
	})
	criteria := ExtractCriteria(s)

	ctx := fw.AgentContext(s, criteria, nil)

	if !strings.Contains(ctx, "Go") || !strings.Contains(ctx, "testing") {
		t.Error("missing Go testing framework reference")
	}
	if !strings.Contains(ctx, "checkout") {
		t.Error("missing slug in agent context")
	}
	if !strings.Contains(ctx, "User can add items to cart") {
		t.Error("missing criterion in agent context")
	}
	if !strings.Contains(ctx, "t.Run") {
		t.Error("missing t.Run convention")
	}
	if !strings.Contains(ctx, "Table-driven") || !strings.Contains(ctx, "table-driven") {
		// Either capitalization
		if !strings.Contains(strings.ToLower(ctx), "table-driven") {
			t.Error("missing table-driven test convention")
		}
	}
	if !strings.Contains(ctx, "testdata") {
		t.Error("missing testdata directory convention")
	}
}

func TestGoTestAssistedValidSyntax(t *testing.T) {
	fw := &GoTestFramework{}
	s := makeSpec("syntax-check", "Syntax Check", []string{
		"Simple criterion",
		`Criterion with "quotes" and backticks`,
		"WHEN something happens THE SYSTEM SHALL do things",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAssisted(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAssisted failed: %v", err)
	}

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "test.go", content, parser.AllErrors)
	if err != nil {
		t.Errorf("assisted mode generates invalid Go:\n%v\nContent:\n%s", err, content)
	}
}

func TestPackageFromDir(t *testing.T) {
	tests := []struct {
		dir  string
		want string
	}{
		{".", "main"},
		{"", "main"},
		{"internal/auth", "auth"},
		{"my-package", "mypackage"},
	}
	for _, tt := range tests {
		got := packageFromDir(tt.dir)
		if got != tt.want {
			t.Errorf("packageFromDir(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}
