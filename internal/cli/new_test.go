package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew_RequiresArg(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("spec", "new")
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestNew_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("spec", "new", "my-feature")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestNew_Feature(t *testing.T) {
	env := newTestEnv(t)

	output, err := runCmd("spec", "new", "csv-export")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created feature spec") {
		t.Errorf("output should mention creation, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "planning", "features", "csv-export", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "title: Csv Export") {
		t.Error("expected title in frontmatter")
	}
	if !strings.Contains(content, "type: feature") {
		t.Error("expected type: feature")
	}
	if !strings.Contains(content, "status: planning") {
		t.Error("expected status: planning")
	}
	if !strings.Contains(content, "## Goal") {
		t.Error("expected Goal section")
	}
	if !strings.Contains(content, "## Changes") {
		t.Error("expected Changes section")
	}

	today := time.Now().Format("2006-01-02")
	if !strings.Contains(content, "created: "+today) {
		t.Errorf("expected today's date in created field")
	}
}

func TestNew_Bug(t *testing.T) {
	env := newTestEnv(t)

	output, err := runCmd("spec", "new", "null-pointer", "--type", "bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created bug spec") {
		t.Errorf("output should mention bug, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "planning", "bugs", "null-pointer", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: bug") {
		t.Error("expected type: bug")
	}
	if !strings.Contains(content, "## Problem") {
		t.Error("expected Problem section for bug")
	}
	if !strings.Contains(content, "## Steps to Reproduce") {
		t.Error("expected Steps to Reproduce section")
	}
}

func TestNew_Initiative(t *testing.T) {
	env := newTestEnv(t)

	output, err := runCmd("spec", "new", "v2-migration", "--type", "initiative")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created initiative spec") {
		t.Errorf("output should mention initiative, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "planning", "initiatives", "v2-migration", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: initiative") {
		t.Error("expected type: initiative")
	}
	if !strings.Contains(content, "## Vision") {
		t.Error("expected Vision section for initiative")
	}
}

func TestNew_Convention(t *testing.T) {
	env := newTestEnv(t)

	output, err := runCmd("spec", "new", "error-handling", "--type", "convention")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created convention spec") {
		t.Errorf("output should mention convention, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "conventions", "error-handling", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: convention") {
		t.Error("expected type: convention")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("expected status: draft")
	}
	if !strings.Contains(content, "## Rule") {
		t.Error("expected Rule section for convention")
	}
}

func TestNew_Decision(t *testing.T) {
	env := newTestEnv(t)

	output, err := runCmd("spec", "new", "use-postgres", "--type", "decision")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created decision spec") {
		t.Errorf("output should mention decision, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "decisions", "use-postgres", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: decision") {
		t.Error("expected type: decision")
	}
	if !strings.Contains(content, "status: proposed") {
		t.Error("expected status: proposed")
	}
	if !strings.Contains(content, "## Decision") {
		t.Error("expected Decision section")
	}
}

func TestNew_AlreadyExists(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	_, err := runCmd("spec", "new", "csv-export")
	if err == nil {
		t.Fatal("expected error for existing spec")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want 'already exists'", err.Error())
	}
}

func TestNew_InvalidType(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("spec", "new", "my-spec", "--type", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !strings.Contains(err.Error(), "unknown spec type") {
		t.Errorf("error = %q, want 'unknown spec type'", err.Error())
	}
}

func TestNew_InvalidSlug(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("spec", "new", "has spaces")
	if err == nil {
		t.Fatal("expected error for slug with spaces")
	}
	if !strings.Contains(err.Error(), "invalid slug") {
		t.Errorf("error = %q, want 'invalid slug'", err.Error())
	}

	_, err = runCmd("spec", "new", "has/slash")
	if err == nil {
		t.Fatal("expected error for slug with slashes")
	}
}

func TestSlugToTitle(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"csv-export", "Csv Export"},
		{"fix-null-pointer", "Fix Null Pointer"},
		{"simple", "Simple"},
		{"a-b-c", "A B C"},
	}

	for _, tt := range tests {
		got := slugToTitle(tt.slug)
		if got != tt.want {
			t.Errorf("slugToTitle(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}

func TestNew_InteractiveFeature(t *testing.T) {
	env := newTestEnv(t)

	// Simulate interactive input: title, tags, claimed_by
	oldStdin := newStdin
	newStdin = strings.NewReader("My CSV Export Feature\nexport, data, csv\nalice\n")
	defer func() { newStdin = oldStdin }()

	output, err := runCmd("spec", "new", "csv-interactive", "--interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created feature spec") {
		t.Errorf("output should mention creation, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "planning", "features", "csv-interactive", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "title: My CSV Export Feature") {
		t.Errorf("expected custom title, got: %s", content)
	}
	if !strings.Contains(content, "tags: [export, data, csv]") {
		t.Errorf("expected tags, got: %s", content)
	}
	if !strings.Contains(content, "claimed_by: alice") {
		t.Errorf("expected claimed_by, got: %s", content)
	}
}

func TestNew_InteractiveDefaults(t *testing.T) {
	env := newTestEnv(t)

	// Empty title falls back to default, no tags, no claimed_by
	oldStdin := newStdin
	newStdin = strings.NewReader("\n\n\n")
	defer func() { newStdin = oldStdin }()

	output, err := runCmd("spec", "new", "default-test", "--interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created feature spec") {
		t.Errorf("expected creation output, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "planning", "features", "default-test", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "title: Default Test") {
		t.Errorf("expected default title, got: %s", content)
	}
	if !strings.Contains(content, "tags: []") {
		t.Errorf("expected empty tags, got: %s", content)
	}
	// Should NOT have claimed_by when empty
	if strings.Contains(content, "claimed_by") {
		t.Errorf("should not have claimed_by when empty, got: %s", content)
	}
}

func TestNew_InteractiveConvention(t *testing.T) {
	env := newTestEnv(t)

	// Convention: title, tags (no claimed_by prompt for conventions)
	oldStdin := newStdin
	newStdin = strings.NewReader("API Error Format\napi, errors\n")
	defer func() { newStdin = oldStdin }()

	output, err := runCmd("spec", "new", "api-errors", "--type", "convention", "--interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created convention spec") {
		t.Errorf("expected convention creation, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "knowledge", "conventions", "api-errors", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "title: API Error Format") {
		t.Errorf("expected custom title, got: %s", content)
	}
	if !strings.Contains(content, "status: draft") {
		t.Errorf("expected draft status, got: %s", content)
	}
	if !strings.Contains(content, "scope: [\"*\"]") {
		t.Errorf("expected scope, got: %s", content)
	}
}

func TestNew_CustomTemplate(t *testing.T) {
	env := newTestEnv(t)

	// Create a custom feature template
	templateContent := `# {{.Title}}

## Business Context

<!-- Why does the business need this? -->

## User Stories

<!-- As a <user>, I want <goal> so that <reason>. -->

## Technical Design

<!-- Architecture and implementation approach. -->

## Rollout Plan

<!-- How will this be rolled out? Feature flags, A/B test, etc. -->

## Metrics

<!-- How will success be measured? -->
`
	templatePath := filepath.Join(env.heroDir, "knowledge", "templates", "feature.md")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("WriteFile template: %v", err)
	}

	output, err := runCmd("spec", "new", "custom-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Created feature spec") {
		t.Errorf("output should mention creation, got: %s", output)
	}

	specPath := filepath.Join(env.heroDir, "planning", "features", "custom-feature", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	// Should have standard frontmatter
	if !strings.Contains(content, "type: feature") {
		t.Error("expected type: feature in frontmatter")
	}
	if !strings.Contains(content, "status: planning") {
		t.Error("expected status: planning in frontmatter")
	}
	// Should have custom sections
	if !strings.Contains(content, "## Business Context") {
		t.Error("expected Business Context section from custom template")
	}
	if !strings.Contains(content, "## User Stories") {
		t.Error("expected User Stories section from custom template")
	}
	if !strings.Contains(content, "## Rollout Plan") {
		t.Error("expected Rollout Plan section from custom template")
	}
	if !strings.Contains(content, "## Metrics") {
		t.Error("expected Metrics section from custom template")
	}
	// Should NOT have default sections
	if strings.Contains(content, "## Goal") {
		t.Error("should not have default Goal section when using custom template")
	}
	// Title placeholder should be resolved
	if !strings.Contains(content, "# Custom Feature") {
		t.Error("expected title to be resolved from placeholder")
	}
}

func TestNew_CustomTemplateBug(t *testing.T) {
	env := newTestEnv(t)

	templateContent := `# {{.Title}}

## Symptoms

<!-- What does the user see? -->

## Environment

<!-- OS, browser, version, etc. -->

## Investigation Log

<!-- Step-by-step investigation notes. -->
`
	templatePath := filepath.Join(env.heroDir, "knowledge", "templates", "bug.md")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("WriteFile template: %v", err)
	}

	_, err := runCmd("spec", "new", "custom-bug", "--type", "bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "planning", "bugs", "custom-bug", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: bug") {
		t.Error("expected type: bug")
	}
	if !strings.Contains(content, "## Symptoms") {
		t.Error("expected custom Symptoms section")
	}
	if !strings.Contains(content, "## Investigation Log") {
		t.Error("expected custom Investigation Log section")
	}
	// Default sections should NOT appear
	if strings.Contains(content, "## Problem") {
		t.Error("should not have default Problem section when using custom template")
	}
}

func TestNew_CustomTemplateInteractive(t *testing.T) {
	env := newTestEnv(t)

	templateContent := `# {{.Title}}

## Overview

<!-- Brief overview. -->

## Custom Section

<!-- Project-specific section. -->
`
	templatePath := filepath.Join(env.heroDir, "knowledge", "templates", "feature.md")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("WriteFile template: %v", err)
	}

	oldStdin := newStdin
	newStdin = strings.NewReader("My Custom Feature\ncustom, test\nbob\n")
	defer func() { newStdin = oldStdin }()

	_, err := runCmd("spec", "new", "custom-interactive", "--interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "planning", "features", "custom-interactive", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "title: My Custom Feature") {
		t.Errorf("expected interactive title, got: %s", content)
	}
	if !strings.Contains(content, "tags: [custom, test]") {
		t.Errorf("expected tags, got: %s", content)
	}
	if !strings.Contains(content, "claimed_by: bob") {
		t.Errorf("expected claimed_by, got: %s", content)
	}
	if !strings.Contains(content, "## Custom Section") {
		t.Error("expected custom section from template")
	}
	if !strings.Contains(content, "# My Custom Feature") {
		t.Error("expected resolved title in body")
	}
}

func TestNew_NoCustomTemplateFallback(t *testing.T) {
	env := newTestEnv(t)

	// No custom template exists — should use built-in
	_, err := runCmd("spec", "new", "fallback-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	specPath := filepath.Join(env.heroDir, "planning", "features", "fallback-feature", "spec.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec file not found: %v", err)
	}

	content := string(data)
	// Should have default sections
	if !strings.Contains(content, "## Goal") {
		t.Error("expected default Goal section when no custom template exists")
	}
	if !strings.Contains(content, "## Changes") {
		t.Error("expected default Changes section")
	}
}

func TestLoadCustomTemplate(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	templatesDir := filepath.Join(heroDir, "knowledge", "templates")
	os.MkdirAll(templatesDir, 0o755)

	// Write a template
	os.WriteFile(filepath.Join(templatesDir, "feature.md"), []byte("# {{.Title}}\n\nCustom body\n"), 0o644)

	body, ok := loadCustomTemplate(heroDir, "feature")
	if !ok {
		t.Fatal("expected to find custom template")
	}
	if !strings.Contains(body, "Custom body") {
		t.Errorf("body missing content: %q", body)
	}

	// Non-existent template
	_, ok = loadCustomTemplate(heroDir, "bug")
	if ok {
		t.Error("should not find bug template when it doesn't exist")
	}
}

func TestApplyTemplatePlaceholders(t *testing.T) {
	body := "# {{.Title}}\n\nCreated on {{.Date}}\n"
	result := applyTemplatePlaceholders(body, "My Feature", "2026-04-12")

	if !strings.Contains(result, "# My Feature") {
		t.Errorf("title not replaced: %q", result)
	}
	if !strings.Contains(result, "Created on 2026-04-12") {
		t.Errorf("date not replaced: %q", result)
	}
}
