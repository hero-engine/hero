package herotest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestXCTestName(t *testing.T) {
	fw := &XCTestFramework{}
	if fw.Name() != "xctest" {
		t.Errorf("Name() = %q, want %q", fw.Name(), "xctest")
	}
}

func TestXCTestRegistered(t *testing.T) {
	fw, err := Get("xctest")
	if err != nil {
		t.Fatalf("Get(xctest) failed: %v", err)
	}
	if fw.Name() != "xctest" {
		t.Errorf("registered Name() = %q, want %q", fw.Name(), "xctest")
	}
}

func TestXCTestTestFilePath(t *testing.T) {
	fw := &XCTestFramework{}

	// Default (nil config)
	got := fw.TestFilePath("login-flow", nil)
	want := filepath.Join("Tests", "AppTests", "LoginFlowTests.swift")
	if got != want {
		t.Errorf("TestFilePath(nil) = %q, want %q", got, want)
	}

	// Custom module
	cfg := &config.TestingConfig{TestDir: "MyModule"}
	got = fw.TestFilePath("login-flow", cfg)
	want = filepath.Join("Tests", "MyModuleTests", "LoginFlowTests.swift")
	if got != want {
		t.Errorf("TestFilePath(custom) = %q, want %q", got, want)
	}
}

func TestXCTestRunCommand(t *testing.T) {
	fw := &XCTestFramework{}

	// Default
	runner, args := fw.RunCommand("Tests/AppTests/LoginFlowTests.swift", nil)
	if runner != "swift" {
		t.Errorf("runner = %q, want %q", runner, "swift")
	}
	if len(args) < 3 || args[0] != "test" || args[1] != "--filter" {
		t.Errorf("args = %v, want [test --filter ...]", args)
	}

	// Custom runner
	cfg := &config.TestingConfig{RunnerCommand: "xcodebuild test -scheme MyApp"}
	runner, args = fw.RunCommand("test.swift", cfg)
	if runner != "xcodebuild" {
		t.Errorf("runner = %q, want %q", runner, "xcodebuild")
	}
}

func TestXCTestGenerateAssisted(t *testing.T) {
	fw := &XCTestFramework{}
	s := makeSpec("login-flow", "Login Flow", []string{
		"User sees login form",
		"Error shown for bad password",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAssisted(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAssisted failed: %v", err)
	}

	// Check XCTest import
	if !strings.Contains(content, "import XCTest") {
		t.Error("missing XCTest import")
	}

	// Check mode comment
	if !strings.Contains(content, "// Mode: assisted") {
		t.Error("missing mode comment")
	}

	// Check class structure
	if !strings.Contains(content, "final class LoginFlowTests: XCTestCase") {
		t.Error("missing class declaration")
	}

	// Check test methods use camelCase
	if !strings.Contains(content, "func test") {
		t.Error("missing test method")
	}

	// Check XCTSkip
	if !strings.Contains(content, "try XCTSkip") {
		t.Error("missing XCTSkip call")
	}

	// Check TODO placeholder
	if !strings.Contains(content, "TODO: implement") {
		t.Error("missing TODO placeholder")
	}

	// Check throws annotation (required for XCTSkip)
	if !strings.Contains(content, "() throws {") {
		t.Error("missing throws annotation on test method")
	}
}

func TestXCTestGenerateAssistedNoCriteria(t *testing.T) {
	fw := &XCTestFramework{}
	s := makeSpec("empty", "Empty", nil)

	_, err := fw.GenerateAssisted(s, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty criteria")
	}
}

func TestXCTestGenerateAutonomous(t *testing.T) {
	fw := &XCTestFramework{}
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

	// Check XCTest import
	if !strings.Contains(content, "import XCTest") {
		t.Error("missing XCTest import")
	}

	// Check mode
	if !strings.Contains(content, "// Mode: autonomous") {
		t.Error("missing mode comment")
	}

	// Check assertions present
	if !strings.Contains(content, "XCTAssert") {
		t.Error("missing XCTAssert assertion")
	}

	// Check quoted text extraction
	if !strings.Contains(content, "Welcome") {
		t.Error("missing extracted quoted text")
	}
}

func TestXCTestGenerateAutonomousEARSEvent(t *testing.T) {
	fw := &XCTestFramework{}
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

func TestXCTestGenerateAutonomousEARSUnwanted(t *testing.T) {
	fw := &XCTestFramework{}
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
	if !strings.Contains(content, "XCTAssertThrowsError") {
		t.Error("missing XCTAssertThrowsError for unwanted criterion")
	}
}

func TestXCTestAgentContext(t *testing.T) {
	fw := &XCTestFramework{}
	s := makeSpec("checkout", "Checkout Flow", []string{
		"User can add items to cart",
	})
	criteria := ExtractCriteria(s)

	ctx := fw.AgentContext(s, criteria, nil)

	if !strings.Contains(ctx, "XCTest") {
		t.Error("missing framework name in agent context")
	}
	if !strings.Contains(ctx, "checkout") {
		t.Error("missing slug in agent context")
	}
	if !strings.Contains(ctx, "User can add items to cart") {
		t.Error("missing criterion in agent context")
	}
	if !strings.Contains(ctx, "Swift Testing") {
		t.Error("missing Swift Testing alternative mention")
	}
	if !strings.Contains(ctx, "XCTAssertTrue") {
		t.Error("missing assertion conventions")
	}
}

func TestSlugToPascal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"login-flow", "LoginFlow"},
		{"my-feature", "MyFeature"},
		{"single", "Single"},
		{"a-b-c", "ABC"},
	}
	for _, tt := range tests {
		got := slugToPascal(tt.input)
		if got != tt.want {
			t.Errorf("slugToPascal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
