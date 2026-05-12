package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectImportSources(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"AGENTS.md":                         "# Project\n\n## Architecture\nSome arch info\n",
		"CLAUDE.md":                         "# Claude instructions\n\nDo this and that.\n",
		".opencode/instructions/module1.md": "# Module 1\n\nModule 1 details.\n",
		".opencode/instructions/module2.md": "# Module 2\n\nModule 2 details.\n",
		".cursor/rules/style.md":            "# Style\n\nCode style rules.\n",
		"main.go":                           "package main\n",
	})

	sources := DetectImportSources(dir)

	kinds := map[string]int{}
	for _, s := range sources {
		kinds[s.Kind]++
	}

	if kinds["agents"] != 1 {
		t.Errorf("expected 1 agents source, got %d", kinds["agents"])
	}
	if kinds["claude"] != 1 {
		t.Errorf("expected 1 claude source, got %d", kinds["claude"])
	}
	if kinds["instructions"] != 2 {
		t.Errorf("expected 2 instructions sources, got %d", kinds["instructions"])
	}
	if kinds["cursor-rules"] != 1 {
		t.Errorf("expected 1 cursor-rules source, got %d", kinds["cursor-rules"])
	}
}

func TestDetectImportSourcesEmpty(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"main.go": "package main\n",
	})

	sources := DetectImportSources(dir)
	if len(sources) != 0 {
		t.Errorf("expected 0 sources for clean project, got %d", len(sources))
	}
}

func TestDetectImportSourcesSkipsNonMarkdown(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		".opencode/instructions/readme.md":  "# Info\nSome info.\n",
		".opencode/instructions/config.yml": "key: value\n",
		".opencode/instructions/binary.bin": "\x00\x01\x02",
	})

	sources := DetectImportSources(dir)

	// Should find the .md file, and .txt is also allowed, but not .yml or .bin
	if len(sources) != 1 {
		t.Errorf("expected 1 source, got %d: %v", len(sources), sourcePaths(sources))
	}
}

func TestParseImportSource(t *testing.T) {
	src := ImportSource{
		Path: "AGENTS.md",
		Kind: "agents",
		Content: `## Project: Morpheus

morpheus is a multi-module gradle project

## Tech Stack
- **Framework**: groovy on grails 6.2.1
- **Backend**: tomcat 9, jdk 17

## Architecture

### Module Dependency Graph
morpheus-ui is the main runtime

### Dual Plugin System
Morpheus has two plugin systems

## Conventions
- always respect the groovy patterns
- never push without permission

## Commands

### Build
- ` + "`./gradlew build`" + ` - full project build

### Test
- ` + "`./gradlew :morpheus-core:test`" + ` - run unit tests
`,
	}

	sections := ParseImportSource(src)

	if len(sections) == 0 {
		t.Fatal("expected sections, got none")
	}

	// Find by heading
	headings := map[string]bool{}
	for _, s := range sections {
		headings[s.Heading] = true
	}

	expected := []string{
		"Project: Morpheus",
		"Tech Stack",
		"Architecture",
		"Module Dependency Graph",
		"Dual Plugin System",
		"Conventions",
		"Commands",
		"Build",
		"Test",
	}
	for _, h := range expected {
		if !headings[h] {
			t.Errorf("missing heading %q in parsed sections", h)
		}
	}
}

func TestParseImportSourceEmpty(t *testing.T) {
	src := ImportSource{Path: "empty.md", Content: ""}
	sections := ParseImportSource(src)
	if len(sections) != 0 {
		t.Errorf("expected 0 sections for empty content, got %d", len(sections))
	}
}

func TestParseImportSourceNoHeadings(t *testing.T) {
	src := ImportSource{
		Path:    "plain.md",
		Content: "Just some plain text\nwith multiple lines\nand no headings.\n",
	}
	sections := ParseImportSource(src)
	if len(sections) != 1 {
		t.Errorf("expected 1 section for text without headings, got %d", len(sections))
	}
	if sections[0].Heading != "" {
		t.Errorf("expected empty heading, got %q", sections[0].Heading)
	}
}

func TestClassifyImportedSections(t *testing.T) {
	sections := []ImportedSection{
		{Source: "AGENTS.md", Heading: "Architecture", Level: 2, Body: "System arch details"},
		{Source: "AGENTS.md", Heading: "Module Dependency Graph", Level: 3, Body: "morpheus-ui depends on..."},
		{Source: "AGENTS.md", Heading: "Tech Stack", Level: 2, Body: "Go 1.22, React 18"},
		{Source: "AGENTS.md", Heading: "Conventions", Level: 2, Body: "Always use service response pattern"},
		{Source: "AGENTS.md", Heading: "Coding Patterns", Level: 2, Body: "Controllers use traits"},
		{Source: "AGENTS.md", Heading: "Rules", Level: 2, Body: "Never push without permission"},
		{Source: "AGENTS.md", Heading: "Build", Level: 3, Body: "./gradlew build"},
		{Source: "AGENTS.md", Heading: "Test", Level: 3, Body: "./gradlew test"},
		{Source: "AGENTS.md", Heading: "Commands", Level: 2, Body: "Build and test commands"},
		{Source: "AGENTS.md", Heading: "Structure", Level: 2, Body: "morpheus-ui contains grails-app/ with controllers, services, views..."},
	}

	result := ClassifyImportedSections(sections)

	if len(result.Architecture) == 0 {
		t.Error("expected architecture sections")
	}
	if len(result.TechStack) == 0 {
		t.Error("expected tech stack sections")
	}
	if len(result.Conventions) == 0 {
		t.Error("expected convention sections")
	}
	if len(result.Rules) == 0 {
		t.Error("expected rule sections")
	}
	if len(result.Commands) == 0 {
		t.Error("expected command sections")
	}
}

func TestClassifySectionArchitecture(t *testing.T) {
	tests := []struct {
		heading string
		want    string
	}{
		{"Architecture", "architecture"},
		{"System Overview", "architecture"},
		{"Module Dependency Graph", "architecture"},
		{"Deployment Topology", "architecture"},
		{"Frontend Architecture", "architecture"},
		{"Dual Plugin System", "architecture"},
		{"Design Patterns", "architecture"},
	}
	for _, tt := range tests {
		s := ImportedSection{Heading: tt.heading, Body: "some content"}
		got := classifySection(s)
		if got != tt.want {
			t.Errorf("classifySection(heading=%q) = %q, want %q", tt.heading, got, tt.want)
		}
	}
}

func TestClassifySectionConventions(t *testing.T) {
	tests := []struct {
		heading string
		want    string
	}{
		{"Conventions", "conventions"},
		{"Coding Patterns", "conventions"},
		{"Naming Conventions", "conventions"},
		{"Controllers", "conventions"},
		{"Services", "conventions"},
		{"Domain Classes", "conventions"},
		{"Testing", "conventions"},
		{"Localization", "conventions"},
		{"Logging", "conventions"},
		{"Security Annotations", "conventions"},
	}
	for _, tt := range tests {
		s := ImportedSection{Heading: tt.heading, Body: "some content"}
		got := classifySection(s)
		if got != tt.want {
			t.Errorf("classifySection(heading=%q) = %q, want %q", tt.heading, got, tt.want)
		}
	}
}

func TestClassifySectionCommands(t *testing.T) {
	tests := []struct {
		heading string
		want    string
	}{
		{"Commands", "commands"},
		{"Build", "commands"},
		{"Run", "commands"},
		{"Test", "commands"},
		{"Dev environment tips", "commands"},
	}
	for _, tt := range tests {
		s := ImportedSection{Heading: tt.heading, Body: "some content"}
		got := classifySection(s)
		if got != tt.want {
			t.Errorf("classifySection(heading=%q) = %q, want %q", tt.heading, got, tt.want)
		}
	}
}

func TestImportToEntries(t *testing.T) {
	ir := &ImportResult{
		Sources: []ImportSource{
			{Path: "AGENTS.md", Kind: "agents"},
		},
		Architecture: []ImportedSection{
			{Source: "AGENTS.md", Heading: "Architecture", Body: "System architecture details"},
		},
		Conventions: []ImportedSection{
			{Source: "AGENTS.md", Heading: "Conventions", Body: "Always use ServiceResponse"},
		},
		Rules: []ImportedSection{
			{Source: "AGENTS.md", Heading: "Rules", Body: "Never push without permission"},
		},
		Commands: []ImportedSection{
			{Source: "AGENTS.md", Heading: "Build", Body: "./gradlew build"},
		},
	}

	heroDir := "/tmp/test/.hero"
	entries := ImportToEntries(ir, heroDir, "2026-01-01")

	// Should have: architecture-overview, project-conventions, project-rules, dev-workflow
	slugs := map[string]bool{}
	for _, e := range entries {
		slugs[e.Slug] = true
	}

	if !slugs["architecture-overview"] {
		t.Error("missing architecture-overview entry")
	}
	if !slugs["project-conventions"] {
		t.Error("missing project-conventions entry")
	}
	if !slugs["project-rules"] {
		t.Error("missing project-rules entry")
	}
	if !slugs["dev-workflow"] {
		t.Error("missing dev-workflow entry")
	}
}

func TestImportToEntriesInstructionFiles(t *testing.T) {
	ir := &ImportResult{
		Sources: []ImportSource{
			{Path: ".opencode/instructions/morpheus-core-agents.md", Kind: "instructions",
				Content: "# Module: morpheus-core\n\nCore module details.\n\n## Structure\nDir layout...\n"},
			{Path: ".opencode/instructions/morpheus-ui-agents.md", Kind: "instructions",
				Content: "# Project: Morpheus\n\nUI module details.\n"},
		},
	}

	heroDir := "/tmp/test/.hero"
	entries := ImportToEntries(ir, heroDir, "2026-01-01")

	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries for instruction files, got %d", len(entries))
	}

	slugs := map[string]bool{}
	for _, e := range entries {
		slugs[e.Slug] = true
	}

	if !slugs["module-morpheus-core-agents"] {
		t.Errorf("missing module-morpheus-core-agents, got slugs: %v", slugKeys(slugs))
	}
	if !slugs["module-morpheus-ui-agents"] {
		t.Errorf("missing module-morpheus-ui-agents, got slugs: %v", slugKeys(slugs))
	}
}

func TestImportToEntriesEmpty(t *testing.T) {
	ir := &ImportResult{}
	entries := ImportToEntries(ir, "/tmp/.hero", "2026-01-01")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty import, got %d", len(entries))
	}
}

func TestEnrichOverviewFromImport(t *testing.T) {
	overview := `---
title: Project Overview
type: context
---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go |

## Current Gaps

- No tests
`

	ir := &ImportResult{
		TechStack: []ImportedSection{
			{Heading: "Tech Stack", Body: "- **Framework**: Grails 6.2.1\n- **Backend**: Tomcat 9"},
		},
		Architecture: []ImportedSection{
			{Heading: "Architecture", Body: "Multi-module Gradle project with dual plugin system"},
		},
	}

	enriched := EnrichOverviewFromImport(overview, ir)

	if !strings.Contains(enriched, "Grails 6.2.1") {
		t.Error("enriched overview should contain imported tech stack")
	}
	if !strings.Contains(enriched, "architecture-overview") {
		t.Error("enriched overview should reference architecture entry")
	}
	// The imported sections should appear before "Current Gaps"
	gapsIdx := strings.Index(enriched, "## Current Gaps")
	grailsIdx := strings.Index(enriched, "Grails 6.2.1")
	if gapsIdx < 0 || grailsIdx < 0 || grailsIdx > gapsIdx {
		t.Error("imported content should appear before Current Gaps section")
	}
}

func TestEnrichOverviewFromImportEmpty(t *testing.T) {
	overview := "Some content"
	ir := &ImportResult{}
	result := EnrichOverviewFromImport(overview, ir)
	if result != overview {
		t.Error("empty import should not modify overview")
	}
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"---\ntitle: Test\n---\n# Hello\nWorld", "# Hello\nWorld"},
		{"# No frontmatter\nJust content", "# No frontmatter\nJust content"},
		{"---\nonly opening\n# Still content", "---\nonly opening\n# Still content"},
	}

	for _, tt := range tests {
		got := stripFrontmatter(tt.input)
		if got != tt.want {
			t.Errorf("stripFrontmatter(%q) = %q, want %q", tt.input[:20], got, tt.want)
		}
	}
}

func TestSectionSources(t *testing.T) {
	sections := []ImportedSection{
		{Source: "AGENTS.md"},
		{Source: "AGENTS.md"},
		{Source: ".opencode/instructions/foo.md"},
	}

	result := sectionSources(sections)
	if !strings.Contains(result, "AGENTS.md") {
		t.Error("should contain AGENTS.md")
	}
	if !strings.Contains(result, "foo.md") {
		t.Error("should contain foo.md")
	}
	// AGENTS.md should appear only once
	if strings.Count(result, "AGENTS.md") != 1 {
		t.Error("AGENTS.md should appear only once")
	}
}

func TestFullImportPipeline(t *testing.T) {
	dir := newScanDir(t, map[string]string{
		"AGENTS.md": `## Project: TestApp

TestApp is a Go web service.

## Tech Stack
- **Language**: Go 1.22
- **Framework**: Gin
- **Database**: PostgreSQL

## Architecture

### Service Layer
All business logic lives in the service layer under internal/services/.

### API Layer
REST endpoints are defined in internal/api/ using Gin handlers.

## Conventions
- Always return structured errors
- Use context.Context for all service methods
- Log with zerolog

## Rules
- Never commit secrets
- Always run tests before pushing

## Commands

### Build
` + "```" + `
go build ./...
` + "```" + `

### Test
` + "```" + `
go test ./...
` + "```" + `
`,
		"main.go": "package main\n",
	})

	// Run the full pipeline
	sources := DetectImportSources(dir)
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	var allSections []ImportedSection
	for _, src := range sources {
		allSections = append(allSections, ParseImportSource(src)...)
	}

	ir := ClassifyImportedSections(allSections)
	ir.Sources = sources

	heroDir := filepath.Join(dir, ".hero")
	os.MkdirAll(heroDir, 0o755)

	entries := ImportToEntries(ir, heroDir, "2026-01-01")

	// Should produce architecture, conventions, rules, commands entries
	types := map[string]int{}
	for _, e := range entries {
		types[e.Type]++
		t.Logf("  entry: type=%s slug=%s", e.Type, e.Slug)
	}

	if types["context"] < 2 { // architecture-overview + dev-workflow
		t.Errorf("expected at least 2 context entries, got %d", types["context"])
	}
	if types["convention"] < 1 {
		t.Errorf("expected at least 1 convention entry, got %d", types["convention"])
	}
	if types["rule"] < 1 {
		t.Errorf("expected at least 1 rule entry, got %d", types["rule"])
	}

	// Verify content quality
	for _, e := range entries {
		if e.Slug == "architecture-overview" {
			if !strings.Contains(e.Content, "Service Layer") {
				t.Error("architecture entry should contain Service Layer heading")
			}
			if !strings.Contains(e.Content, "API Layer") {
				t.Error("architecture entry should contain API Layer heading")
			}
		}
		if e.Slug == "project-conventions" {
			if !strings.Contains(e.Content, "structured errors") {
				t.Error("conventions entry should contain structured errors")
			}
		}
		if e.Slug == "dev-workflow" {
			if !strings.Contains(e.Content, "go build") {
				t.Error("dev-workflow entry should contain go build command")
			}
		}
	}

}

// helpers

func sourcePaths(sources []ImportSource) []string {
	var paths []string
	for _, s := range sources {
		paths = append(paths, s.Path)
	}
	return paths
}

func slugKeys(m map[string]bool) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
