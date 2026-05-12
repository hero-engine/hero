package herotest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// makeSpec creates a test spec with the given slug, title, and acceptance criteria.
func makeSpec(slug, title string, criteria []string) *spec.Spec {
	var lines []string
	for _, c := range criteria {
		lines = append(lines, "- "+c)
	}
	sections := map[string]string{
		"acceptance criteria": strings.Join(lines, "\n"),
	}
	return &spec.Spec{
		Slug:     slug,
		Title:    title,
		Type:     spec.TypeFeature,
		Status:   spec.StatusDelivering,
		Sections: sections,
	}
}

func TestRegistryGetPlaywright(t *testing.T) {
	fw, err := Get("playwright")
	if err != nil {
		t.Fatalf("Get(playwright) failed: %v", err)
	}
	if fw.Name() != "playwright" {
		t.Errorf("Name() = %q, want %q", fw.Name(), "playwright")
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	_, err := Get("cypress")
	if err == nil {
		t.Fatal("Get(cypress) should fail, got nil error")
	}
	if !strings.Contains(err.Error(), "unknown test framework") {
		t.Errorf("error = %q, want to contain 'unknown test framework'", err.Error())
	}
}

func TestExtractCriteria(t *testing.T) {
	s := makeSpec("test-slug", "Test", []string{
		"User can log in",
		"Dashboard shows metrics",
		"Error message displayed on failure",
	})
	criteria := ExtractCriteria(s)
	if len(criteria) != 3 {
		t.Fatalf("got %d criteria, want 3", len(criteria))
	}
	if criteria[0] != "User can log in" {
		t.Errorf("criteria[0] = %q, want %q", criteria[0], "User can log in")
	}
	if criteria[2] != "Error message displayed on failure" {
		t.Errorf("criteria[2] = %q, want %q", criteria[2], "Error message displayed on failure")
	}
}

func TestExtractCriteriaBulletStar(t *testing.T) {
	s := &spec.Spec{
		Slug: "star-bullets",
		Sections: map[string]string{
			"acceptance criteria": "* First item\n* Second item",
		},
	}
	criteria := ExtractCriteria(s)
	if len(criteria) != 2 {
		t.Fatalf("got %d criteria, want 2", len(criteria))
	}
	if criteria[0] != "First item" {
		t.Errorf("criteria[0] = %q, want %q", criteria[0], "First item")
	}
}

func TestExtractCriteriaEmpty(t *testing.T) {
	s := &spec.Spec{
		Slug:     "no-criteria",
		Sections: map[string]string{},
	}
	criteria := ExtractCriteria(s)
	if len(criteria) != 0 {
		t.Errorf("got %d criteria, want 0", len(criteria))
	}
}

func TestCriterionToTestName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"User can log in", "user can log in"},
		{"Show `#dashboard` element", "show #dashboard element"},
		{strings.Repeat("x", 100), strings.Repeat("x", 80)},
		{"It's a \"test\" with 'quotes'", "its a test with quotes"},
	}
	for _, tt := range tests {
		got := CriterionToTestName(tt.input)
		if got != tt.want {
			t.Errorf("CriterionToTestName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- PlaywrightFramework tests ---

func TestPlaywrightTestFilePath(t *testing.T) {
	fw := &PlaywrightFramework{}

	// Default (nil config)
	got := fw.TestFilePath("my-feature", nil)
	want := filepath.Join("e2e", "my-feature.spec.ts")
	if got != want {
		t.Errorf("TestFilePath(nil cfg) = %q, want %q", got, want)
	}

	// Custom test dir
	cfg := &config.TestingConfig{TestDir: "tests/integration"}
	got = fw.TestFilePath("my-feature", cfg)
	want = filepath.Join("tests/integration", "my-feature.spec.ts")
	if got != want {
		t.Errorf("TestFilePath(custom dir) = %q, want %q", got, want)
	}
}

func TestPlaywrightRunCommand(t *testing.T) {
	fw := &PlaywrightFramework{}

	// Default
	runner, args := fw.RunCommand("e2e/test.spec.ts", nil)
	if runner != "npx" {
		t.Errorf("runner = %q, want %q", runner, "npx")
	}
	if len(args) < 3 || args[0] != "playwright" || args[1] != "test" || args[2] != "e2e/test.spec.ts" {
		t.Errorf("args = %v, want [playwright test e2e/test.spec.ts]", args)
	}

	// Custom runner + config
	cfg := &config.TestingConfig{
		RunnerCommand: "yarn playwright test",
		ConfigPath:    "playwright.config.ts",
	}
	runner, args = fw.RunCommand("e2e/test.spec.ts", cfg)
	if runner != "yarn" {
		t.Errorf("runner = %q, want %q", runner, "yarn")
	}
	// Should contain config arg
	found := false
	for i, a := range args {
		if a == "--config" && i+1 < len(args) && args[i+1] == "playwright.config.ts" {
			found = true
		}
	}
	if !found {
		t.Errorf("args %v missing --config playwright.config.ts", args)
	}
}

func TestGenerateAssisted(t *testing.T) {
	fw := &PlaywrightFramework{}
	s := makeSpec("login-page", "Login Page", []string{
		"User sees login form",
		"Error shown for invalid credentials",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAssisted(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAssisted failed: %v", err)
	}

	// Check imports
	if !strings.Contains(content, "import { test, expect } from '@playwright/test'") {
		t.Error("missing playwright import")
	}
	// Check mode comment
	if !strings.Contains(content, "// Mode: assisted") {
		t.Error("missing mode comment")
	}
	// Check video conditional
	if !strings.Contains(content, "PWVIDEO") {
		t.Error("missing PWVIDEO conditional")
	}
	// Check describe block
	if !strings.Contains(content, "test.describe('login-page'") {
		t.Error("missing test.describe block")
	}
	// Check TODO placeholders
	if !strings.Contains(content, "// TODO: implement") {
		t.Error("missing TODO placeholder")
	}
	// Check test.skip
	if !strings.Contains(content, "test.skip()") {
		t.Error("missing test.skip()")
	}
}

func TestGenerateAssistedNoCriteria(t *testing.T) {
	fw := &PlaywrightFramework{}
	s := makeSpec("empty", "Empty", nil)

	_, err := fw.GenerateAssisted(s, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty criteria")
	}
}

func TestGenerateAutonomous(t *testing.T) {
	fw := &PlaywrightFramework{}
	s := makeSpec("dashboard", "Dashboard", []string{
		"Dashboard displays user metrics",
		"URL should contain /dashboard",
		"Error message shown for invalid input",
		"Click the submit button",
		`Page contains text "Welcome"`,
		"Form input field is pre-filled",
		"Results count should match expected",
	})
	criteria := ExtractCriteria(s)
	cfg := &config.TestingConfig{BaseURL: "http://localhost:3000"}

	content, err := fw.GenerateAutonomous(s, criteria, cfg)
	if err != nil {
		t.Fatalf("GenerateAutonomous failed: %v", err)
	}

	// Check mode
	if !strings.Contains(content, "// Mode: autonomous") {
		t.Error("missing mode comment")
	}
	// Check page.goto with baseURL
	if !strings.Contains(content, "page.goto('http://localhost:3000')") {
		t.Error("missing page.goto with baseURL")
	}
	// Check heuristic mappings are present
	if !strings.Contains(content, "toBeVisible") {
		t.Error("missing toBeVisible assertion for 'displays'")
	}
	if !strings.Contains(content, "toHaveURL") {
		t.Error("missing toHaveURL assertion for 'URL should contain'")
	}
	// Check quoted text extraction
	if !strings.Contains(content, "Welcome") {
		t.Error("missing extracted quoted text 'Welcome'")
	}
}

func TestGenerateAutonomousDefaultFallback(t *testing.T) {
	fw := &PlaywrightFramework{}
	s := makeSpec("misc", "Misc", []string{
		"Something completely unmappable happens in the system",
	})
	criteria := ExtractCriteria(s)

	content, err := fw.GenerateAutonomous(s, criteria, nil)
	if err != nil {
		t.Fatalf("GenerateAutonomous failed: %v", err)
	}

	// Default branch should produce test.skip()
	if !strings.Contains(content, "test.skip()") {
		t.Error("unmappable criterion should produce test.skip()")
	}
}

func TestAgentContext(t *testing.T) {
	fw := &PlaywrightFramework{}
	s := makeSpec("checkout", "Checkout Flow", []string{
		"User can add items to cart",
		"Payment form validates card number",
	})
	criteria := ExtractCriteria(s)
	cfg := &config.TestingConfig{BaseURL: "http://localhost:8080"}

	ctx := fw.AgentContext(s, criteria, cfg)

	if !strings.Contains(ctx, "checkout") {
		t.Error("missing slug in agent context")
	}
	if !strings.Contains(ctx, "http://localhost:8080") {
		t.Error("missing base URL in agent context")
	}
	if !strings.Contains(ctx, "User can add items to cart") {
		t.Error("missing criterion in agent context")
	}
	if !strings.Contains(ctx, "Playwright Test") {
		t.Error("missing framework name in agent context")
	}
}

// --- Helper function tests ---

func TestContainsAny(t *testing.T) {
	if !containsAny("hello world", "world", "foo") {
		t.Error("containsAny should find 'world'")
	}
	if containsAny("hello world", "foo", "bar") {
		t.Error("containsAny should not find 'foo' or 'bar'")
	}
}

func TestExtractSelector(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Show `#dashboard` element", "#dashboard"},
		{"Show `.main-nav` component", ".main-nav"},
		{"Show `[data-testid]` attribute", "[data-testid]"},
		{"Show `my-component` element", "my-component"}, // contains hyphen
		{"No backticks here", ""},
		{"Single `backtick only", ""},
		{"Show `foo` something", ""}, // no selector-like chars
	}
	for _, tt := range tests {
		got := extractSelector(tt.input)
		if got != tt.want {
			t.Errorf("extractSelector(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractQuotedText(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`Contains "Welcome" text`, "Welcome"},
		{`Contains 'Hello' text`, "Hello"},
		{`No quotes here`, ""},
		{`Single "quote only`, ""},
	}
	for _, tt := range tests {
		got := extractQuotedText(tt.input)
		if got != tt.want {
			t.Errorf("extractQuotedText(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeTS(t *testing.T) {
	got := escapeTS(`it's a "test"`)
	want := `it\'s a "test"`
	if got != want {
		t.Errorf("escapeTS = %q, want %q", got, want)
	}
}

func TestEscapeRegex(t *testing.T) {
	got := escapeRegex("http://example.com/path")
	if !strings.Contains(got, `\.`) {
		t.Errorf("escapeRegex should escape dots: %q", got)
	}
	// Check other special chars
	got2 := escapeRegex("a+b*c?d")
	if !strings.Contains(got2, `\+`) || !strings.Contains(got2, `\*`) || !strings.Contains(got2, `\?`) {
		t.Errorf("escapeRegex should escape +*?: %q", got2)
	}
}

// --- Generate orchestration tests ---

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock spec file in a specs directory
	specDir := filepath.Join(tmpDir, ".hero", "planning", "features", "login-flow")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specContent := `---
title: Login Flow
type: feature
status: delivering
created: 2025-01-01
---
# Login Flow

## Acceptance Criteria

- User sees login form
- Error shown for bad password
`
	specPath := filepath.Join(specDir, "spec.md")
	if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := spec.ParseFile(specPath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	// Generate in autonomous mode
	testFile, err := Generate(tmpDir, s, nil, "autonomous")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check file was created
	absPath := filepath.Join(tmpDir, testFile)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Fatalf("test file not created at %s", absPath)
	}

	// Read and verify content
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "import { test, expect }") {
		t.Error("generated file missing playwright import")
	}
	if !strings.Contains(content, "// Mode: autonomous") {
		t.Error("generated file missing mode comment")
	}
}

func TestGenerateAssistedMode(t *testing.T) {
	tmpDir := t.TempDir()

	s := makeSpec("my-feature", "My Feature", []string{"Feature works correctly"})
	s.CreatedAt = time.Now()
	s.ModifiedAt = time.Now()

	testFile, err := Generate(tmpDir, s, nil, "assisted")
	if err != nil {
		t.Fatalf("Generate(assisted) failed: %v", err)
	}

	absPath := filepath.Join(tmpDir, testFile)
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "// Mode: assisted") {
		t.Error("expected assisted mode")
	}
	if !strings.Contains(content, "// TODO: implement") {
		t.Error("assisted mode should have TODO placeholders")
	}
}

func TestGenerateAgentMode(t *testing.T) {
	tmpDir := t.TempDir()

	s := makeSpec("agent-test", "Agent Test", []string{"Agent criterion"})

	testFile, err := Generate(tmpDir, s, nil, "agent")
	if err != nil {
		t.Fatalf("Generate(agent) failed: %v", err)
	}

	// Agent mode writes the context markdown, not a TS file
	absPath := filepath.Join(tmpDir, testFile)
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Test Requirements") {
		t.Error("agent mode should produce context with 'Test Requirements'")
	}
}

func TestGenerateInvalidMode(t *testing.T) {
	tmpDir := t.TempDir()
	s := makeSpec("bad", "Bad", []string{"criterion"})

	_, err := Generate(tmpDir, s, nil, "bogus")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestGenerateNoCriteria(t *testing.T) {
	tmpDir := t.TempDir()
	s := makeSpec("empty", "Empty", nil)

	_, err := Generate(tmpDir, s, nil, "autonomous")
	if err == nil {
		t.Fatal("expected error for no criteria")
	}
}

func TestTestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Not existing yet
	if TestFileExists(tmpDir, "my-slug", nil) {
		t.Error("should not exist before generation")
	}

	// Create the file
	dir := filepath.Join(tmpDir, "e2e")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "my-slug.spec.ts"), []byte("test"), 0o644)

	if !TestFileExists(tmpDir, "my-slug", nil) {
		t.Error("should exist after creation")
	}
}

func TestTestFilePath(t *testing.T) {
	got := TestFilePath("my-slug", nil)
	want := filepath.Join("e2e", "my-slug.spec.ts")
	if got != want {
		t.Errorf("TestFilePath = %q, want %q", got, want)
	}

	cfg := &config.TestingConfig{TestDir: "custom-tests"}
	got = TestFilePath("my-slug", cfg)
	want = filepath.Join("custom-tests", "my-slug.spec.ts")
	if got != want {
		t.Errorf("TestFilePath(custom) = %q, want %q", got, want)
	}
}

func TestRunArgs(t *testing.T) {
	cmd, args, err := RunArgs("my-slug", nil)
	if err != nil {
		t.Fatalf("RunArgs failed: %v", err)
	}
	if cmd != "npx" {
		t.Errorf("cmd = %q, want npx", cmd)
	}
	if len(args) < 3 {
		t.Fatalf("args too short: %v", args)
	}
	if args[0] != "playwright" {
		t.Errorf("args[0] = %q, want playwright", args[0])
	}
}
