package herotest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestVitestName(t *testing.T) {
	fw := &VitestFramework{}
	if fw.Name() != "vitest" {
		t.Errorf("Name() = %q, want %q", fw.Name(), "vitest")
	}
}

func TestVitestRegistered(t *testing.T) {
	fw, err := Get("vitest")
	if err != nil {
		t.Fatalf("Get(vitest) failed: %v", err)
	}
	if fw.Name() != "vitest" {
		t.Errorf("registered Name() = %q, want %q", fw.Name(), "vitest")
	}
}

func TestVitestTestFilePath(t *testing.T) {
	fw := &VitestFramework{}

	// Default (nil config)
	got := fw.TestFilePath("login-flow", nil)
	want := filepath.Join("tests", "login-flow.test.ts")
	if got != want {
		t.Errorf("TestFilePath(nil) = %q, want %q", got, want)
	}

	// Custom test dir
	cfg := &config.TestingConfig{TestDir: "src/__tests__"}
	got = fw.TestFilePath("login-flow", cfg)
	want = filepath.Join("src/__tests__", "login-flow.test.ts")
	if got != want {
		t.Errorf("TestFilePath(custom) = %q, want %q", got, want)
	}
}

func TestVitestRunCommand(t *testing.T) {
	fw := &VitestFramework{}

	// Default
	runner, args := fw.RunCommand("tests/login-flow.test.ts", nil)
	if runner != "npx" {
		t.Errorf("runner = %q, want %q", runner, "npx")
	}
	if len(args) != 3 || args[0] != "vitest" || args[1] != "run" || args[2] != "tests/login-flow.test.ts" {
		t.Errorf("args = %v, want [vitest run tests/login-flow.test.ts]", args)
	}

	// Custom runner
	cfg := &config.TestingConfig{RunnerCommand: "pnpm vitest run"}
	runner, args = fw.RunCommand("test.ts", cfg)
	if runner != "pnpm" {
		t.Errorf("runner = %q, want %q", runner, "pnpm")
	}
}

func TestVitestGenerateAssisted(t *testing.T) {
	fw := &VitestFramework{}
	s := makeSpec("login-flow", "Login Flow", []string{
		"User sees login form",
		"Error shown for bad password",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAssisted(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAssisted failed: %v", err)
	}

	// Check vitest import
	if !strings.Contains(content, "import { describe, it, expect } from 'vitest'") {
		t.Error("missing vitest import")
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

func TestVitestGenerateAssistedNoCriteria(t *testing.T) {
	fw := &VitestFramework{}
	s := makeSpec("empty", "Empty", nil)

	_, err := fw.GenerateAssisted(s, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty criteria")
	}
}

func TestVitestGenerateAutonomous(t *testing.T) {
	fw := &VitestFramework{}
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

	// Check vitest import
	if !strings.Contains(content, "import { describe, it, expect } from 'vitest'") {
		t.Error("missing vitest import")
	}

	// Check mode
	if !strings.Contains(content, "// Mode: autonomous") {
		t.Error("missing mode comment")
	}

	// Check assertions present
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

func TestVitestGenerateAutonomousEARSEvent(t *testing.T) {
	fw := &VitestFramework{}
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

func TestVitestGenerateAutonomousEARSUnwanted(t *testing.T) {
	fw := &VitestFramework{}
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

func TestVitestAgentContext(t *testing.T) {
	fw := &VitestFramework{}
	s := makeSpec("checkout", "Checkout Flow", []string{
		"User can add items to cart",
	})
	criteria := ExtractCriteria(s)

	ctx := fw.AgentContext(s, criteria, nil)

	if !strings.Contains(ctx, "Vitest") || !strings.Contains(ctx, "vitest") {
		t.Error("missing framework name in agent context")
	}
	if !strings.Contains(ctx, "checkout") {
		t.Error("missing slug in agent context")
	}
	if !strings.Contains(ctx, "User can add items to cart") {
		t.Error("missing criterion in agent context")
	}
	if !strings.Contains(ctx, "vi.mock") {
		t.Error("missing vi.mock convention")
	}
	if !strings.Contains(ctx, "describe") {
		t.Error("missing describe convention")
	}
	if !strings.Contains(ctx, "ESM") {
		t.Error("missing ESM note")
	}
}
