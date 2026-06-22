package herotest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestJestName(t *testing.T) {
	fw := &JestFramework{}
	if fw.Name() != "jest" {
		t.Errorf("Name() = %q, want %q", fw.Name(), "jest")
	}
}

func TestJestRegistered(t *testing.T) {
	fw, err := Get("jest")
	if err != nil {
		t.Fatalf("Get(jest) failed: %v", err)
	}
	if fw.Name() != "jest" {
		t.Errorf("registered Name() = %q, want %q", fw.Name(), "jest")
	}
}

func TestJestTestFilePath(t *testing.T) {
	fw := &JestFramework{}

	// Default (nil config) — __tests__ directory
	got := fw.TestFilePath("login-flow", nil)
	want := filepath.Join("__tests__", "login-flow.test.ts")
	if got != want {
		t.Errorf("TestFilePath(nil) = %q, want %q", got, want)
	}

	// Custom test dir
	cfg := &config.TestingConfig{TestDir: "src/tests"}
	got = fw.TestFilePath("login-flow", cfg)
	want = filepath.Join("src/tests", "login-flow.test.ts")
	if got != want {
		t.Errorf("TestFilePath(custom) = %q, want %q", got, want)
	}
}

func TestJestRunCommand(t *testing.T) {
	fw := &JestFramework{}

	// Default
	runner, args := fw.RunCommand("__tests__/login-flow.test.ts", nil)
	if runner != "npx" {
		t.Errorf("runner = %q, want %q", runner, "npx")
	}
	if len(args) != 3 || args[0] != "jest" || args[1] != "__tests__/login-flow.test.ts" || args[2] != "--no-coverage" {
		t.Errorf("args = %v, want [jest __tests__/login-flow.test.ts --no-coverage]", args)
	}

	// Custom runner
	cfg := &config.TestingConfig{RunnerCommand: "yarn jest"}
	runner, args = fw.RunCommand("test.ts", cfg)
	if runner != "yarn" {
		t.Errorf("runner = %q, want %q", runner, "yarn")
	}
}

func TestJestGenerateAssisted(t *testing.T) {
	fw := &JestFramework{}
	s := makeSpec("login-flow", "Login Flow", []string{
		"User sees login form",
		"Error shown for bad password",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAssisted(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAssisted failed: %v", err)
	}

	// Jest has NO explicit import (globals)
	if strings.Contains(content, "import {") {
		t.Error("jest should not have explicit import statement")
	}

	// Check Jest header comment
	if !strings.Contains(content, "// Jest test file") {
		t.Error("missing Jest header comment")
	}

	// Check mode comment
	if !strings.Contains(content, "// Mode: assisted") {
		t.Error("missing mode comment")
	}

	// Check describe block
	if !strings.Contains(content, "describe('login-flow'") {
		t.Error("missing describe block")
	}

	// Check it.skip
	if !strings.Contains(content, "it.skip(") {
		t.Error("missing it.skip()")
	}

	// Check TODO placeholder
	if !strings.Contains(content, "// TODO: implement") {
		t.Error("missing TODO placeholder")
	}
}

func TestJestGenerateAssistedNoCriteria(t *testing.T) {
	fw := &JestFramework{}
	s := makeSpec("empty", "Empty", nil)

	_, err := fw.GenerateAssisted(s, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty criteria")
	}
}

func TestJestGenerateAutonomous(t *testing.T) {
	fw := &JestFramework{}
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

	// No explicit import
	if strings.Contains(content, "import {") {
		t.Error("jest should not have explicit import statement")
	}

	// Check mode
	if !strings.Contains(content, "// Mode: autonomous") {
		t.Error("missing mode comment")
	}

	// Check assertions
	if !strings.Contains(content, "expect(") {
		t.Error("missing expect assertion")
	}
	if !strings.Contains(content, "toBeTruthy") {
		t.Error("missing toBeTruthy for 'displays'")
	}
	if !strings.Contains(content, "toThrow") {
		t.Error("missing toThrow for 'fail'")
	}

	// Check quoted text extraction
	if !strings.Contains(content, "Welcome") {
		t.Error("missing extracted quoted text 'Welcome'")
	}

	// Check count assertion
	if !strings.Contains(content, "toHaveLength") {
		t.Error("missing toHaveLength for 'count'")
	}
}

func TestJestGenerateAutonomousEARSEvent(t *testing.T) {
	fw := &JestFramework{}
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

func TestJestGenerateAutonomousEARSUnwanted(t *testing.T) {
	fw := &JestFramework{}
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
	if !strings.Contains(content, "toThrow") {
		t.Error("missing toThrow for unwanted criterion")
	}
}

func TestJestAgentContext(t *testing.T) {
	fw := &JestFramework{}
	s := makeSpec("checkout", "Checkout Flow", []string{
		"User can add items to cart",
	})
	criteria := ExtractCriteria(s)

	ctx := fw.AgentContext(s, criteria, nil)

	if !strings.Contains(ctx, "Jest") || !strings.Contains(ctx, "jest") {
		t.Error("missing framework name in agent context")
	}
	if !strings.Contains(ctx, "checkout") {
		t.Error("missing slug in agent context")
	}
	if !strings.Contains(ctx, "User can add items to cart") {
		t.Error("missing criterion in agent context")
	}
	if !strings.Contains(ctx, "jest.mock") {
		t.Error("missing jest.mock convention")
	}
	if !strings.Contains(ctx, "__tests__") {
		t.Error("missing __tests__ directory convention")
	}
}

func TestJestVsVitestDifferences(t *testing.T) {
	jfw := &JestFramework{}
	vfw := &VitestFramework{}

	// Directory defaults differ
	jestPath := jfw.TestFilePath("my-feature", nil)
	vitestPath := vfw.TestFilePath("my-feature", nil)

	if !strings.Contains(jestPath, "__tests__") {
		t.Error("jest default dir should be __tests__")
	}
	if !strings.Contains(vitestPath, "tests") && !strings.Contains(vitestPath, "__tests__") {
		t.Error("vitest default dir should be tests")
	}
	if jestPath == vitestPath {
		t.Error("jest and vitest should have different default test paths")
	}

	// Import differences
	s := makeSpec("diff-check", "Diff Check", []string{"Something works"})
	criteria := ExtractCriteria(s)

	vitestContent, _ := vfw.GenerateAssisted(s, criteria, nil)
	jestContent, _ := jfw.GenerateAssisted(s, criteria, nil)

	if !strings.Contains(vitestContent, "import { describe, it, expect } from 'vitest'") {
		t.Error("vitest should have explicit import")
	}
	if strings.Contains(jestContent, "import {") {
		t.Error("jest should not have explicit import")
	}
}
