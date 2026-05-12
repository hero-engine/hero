package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImportSource represents a detected documentation/knowledge file that can be imported.
type ImportSource struct {
	Path     string // relative path from project root
	FullPath string // absolute path
	Kind     string // "agents", "instructions", "cursor-rules", "claude", "copilot", "custom-doc"
	Content  string // raw file content
}

// ImportedSection is a parsed section from an imported document.
type ImportedSection struct {
	Source  string // source file path (relative)
	Heading string // the heading text (e.g. "Architecture", "Conventions")
	Level   int    // heading level (1-6)
	Body    string // section content (text under the heading)
}

// ImportResult holds the full result of importing existing knowledge files.
type ImportResult struct {
	Sources  []ImportSource
	Sections []ImportedSection

	// Classified sections — ready to become knowledge entries or enrich existing ones
	Architecture  []ImportedSection // sections about architecture, structure, design patterns
	Conventions   []ImportedSection // sections about coding conventions, patterns, style
	Rules         []ImportedSection // sections about rules, constraints, enforcement
	Commands      []ImportedSection // sections about build/test/run commands
	TechStack     []ImportedSection // sections about technology stack
	ModuleDetails []ImportedSection // sections about specific modules/packages
	Other         []ImportedSection // unclassified sections
}

// DetectImportSources finds all importable documentation files in the project root.
func DetectImportSources(projectRoot string) []ImportSource {
	var sources []ImportSource

	// Single-file documentation
	singleFiles := []struct {
		path string
		kind string
	}{
		{"AGENTS.md", "agents"},
		{"CLAUDE.md", "claude"},
		{".github/copilot-instructions.md", "copilot"},
		{"COPILOT.md", "copilot"},
		{"CURSOR.md", "cursor-rules"},
		{"ARCHITECTURE.md", "custom-doc"},
		{"DESIGN.md", "custom-doc"},
		{"DEVELOPMENT.md", "custom-doc"},
		{"HACKING.md", "custom-doc"},
	}

	for _, sf := range singleFiles {
		full := filepath.Join(projectRoot, sf.path)
		if _, err := os.Stat(full); err == nil {
			content, err := os.ReadFile(full)
			if err == nil && len(content) > 0 {
				sources = append(sources, ImportSource{
					Path:     sf.path,
					FullPath: full,
					Kind:     sf.kind,
					Content:  string(content),
				})
			}
		}
	}

	// Directory-based documentation
	dirs := []struct {
		path string
		kind string
	}{
		{".opencode/instructions", "instructions"},
		{".cursor/rules", "cursor-rules"},
		{"docs", "custom-doc"},
	}

	for _, d := range dirs {
		full := filepath.Join(projectRoot, d.path)
		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".txt") {
				continue
			}
			relPath := filepath.Join(d.path, name)
			fullPath := filepath.Join(full, name)
			content, err := os.ReadFile(fullPath)
			if err != nil || len(content) == 0 {
				continue
			}
			sources = append(sources, ImportSource{
				Path:     relPath,
				FullPath: fullPath,
				Kind:     d.kind,
				Content:  string(content),
			})
		}
	}

	return sources
}

// ParseImportSource parses a single import source into sections by markdown headings.
func ParseImportSource(src ImportSource) []ImportedSection {
	return parseMarkdownSections(src.Path, src.Content)
}

// parseMarkdownSections splits markdown content into sections based on headings.
func parseMarkdownSections(sourcePath, content string) []ImportedSection {
	lines := strings.Split(content, "\n")
	var sections []ImportedSection

	var currentHeading string
	currentLevel := 0
	var bodyLines []string

	flushSection := func() {
		if currentHeading == "" && len(bodyLines) == 0 {
			return
		}
		body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
		if body == "" && currentHeading == "" {
			return
		}
		sections = append(sections, ImportedSection{
			Source:  sourcePath,
			Heading: currentHeading,
			Level:   currentLevel,
			Body:    body,
		})
	}

	reHeading := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	for _, line := range lines {
		if m := reHeading.FindStringSubmatch(line); m != nil {
			flushSection()
			currentLevel = len(m[1])
			currentHeading = strings.TrimSpace(m[2])
			bodyLines = nil
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	flushSection()

	return sections
}

// ClassifyImportedSections categorizes sections by their heading text and content.
// Sub-headings (deeper level than a classified parent) inherit the parent's classification
// unless they independently classify to something other than "other".
func ClassifyImportedSections(sections []ImportedSection) *ImportResult {
	r := &ImportResult{
		Sections: sections,
	}

	// Track parent classification for hierarchy inheritance.
	// parentCat/parentLevel track the most recent "real" (non-other) classified heading.
	parentCat := ""
	parentLevel := 0

	for _, s := range sections {
		cat := classifySection(s)

		// If this heading is at the same or higher level as the parent,
		// it's a new top-level section — update the parent.
		if s.Level > 0 && s.Level <= parentLevel {
			parentCat = cat
			parentLevel = s.Level
		} else if parentLevel == 0 && s.Level > 0 {
			// First heading — set as parent
			parentCat = cat
			parentLevel = s.Level
		} else if s.Level > parentLevel && cat == "other" && parentCat != "" && parentCat != "other" {
			// Sub-heading classified as "other" inherits parent classification
			cat = parentCat
		}

		switch cat {
		case "architecture":
			r.Architecture = append(r.Architecture, s)
		case "conventions":
			r.Conventions = append(r.Conventions, s)
		case "rules":
			r.Rules = append(r.Rules, s)
		case "commands":
			r.Commands = append(r.Commands, s)
		case "techstack":
			r.TechStack = append(r.TechStack, s)
		case "modules":
			r.ModuleDetails = append(r.ModuleDetails, s)
		default:
			r.Other = append(r.Other, s)
		}
	}

	return r
}

// classifySection returns a category string based on the section heading and content.
func classifySection(s ImportedSection) string {
	h := strings.ToLower(s.Heading)
	b := strings.ToLower(s.Body)

	// Architecture / structure / design
	archKeywords := []string{
		"architecture", "design", "structure", "module", "dependency graph",
		"system overview", "topology", "deployment", "service resolution",
		"plugin system", "frontend architecture", "hybrid",
	}
	for _, kw := range archKeywords {
		if strings.Contains(h, kw) {
			return "architecture"
		}
	}

	// Module-specific details (headings that name specific modules/packages)
	modulePatterns := []string{
		"module:", "plugin", "bundled plugin", "cloud plugin",
		"backup plugin", "network plugin",
	}
	for _, mp := range modulePatterns {
		if strings.Contains(h, mp) {
			return "modules"
		}
	}

	// Conventions / patterns / coding style
	conventionKeywords := []string{
		"convention", "coding pattern", "coding style", "style guide",
		"naming convention", "naming pattern",
		"best practice", "guidelines", "localization", "logging",
		"domain class", "controllers", "services", "views", "api controllers",
		"seed data", "database migration", "testing", "security annotation",
		"state management",
	}
	for _, kw := range conventionKeywords {
		if strings.Contains(h, kw) {
			return "conventions"
		}
	}

	// Rules / constraints / enforcement
	ruleKeywords := []string{
		"rule", "constraint", "enforcement", "must", "never", "always",
		"requirement", "mandatory",
	}
	for _, kw := range ruleKeywords {
		if strings.Contains(h, kw) {
			return "rules"
		}
	}
	// Sections with rule-like content in body
	if strings.Contains(h, "convention") || h == "" {
		ruleBodyPhrases := []string{"never ", "always ", "must ", "do not "}
		ruleCount := 0
		for _, phrase := range ruleBodyPhrases {
			ruleCount += strings.Count(b, phrase)
		}
		if ruleCount >= 3 {
			return "rules"
		}
	}

	// Commands / build / test / run
	commandKeywords := []string{
		"command", "build", "run", "test", "deploy", "dev environment",
		"getting started", "setup", "install",
	}
	for _, kw := range commandKeywords {
		if strings.Contains(h, kw) {
			return "commands"
		}
	}

	// Tech stack
	stackKeywords := []string{
		"tech stack", "technology", "stack", "framework", "version",
		"dependencies", "key version",
	}
	for _, kw := range stackKeywords {
		if strings.Contains(h, kw) {
			return "techstack"
		}
	}

	// Sections about specific subsystems (detected from body content patterns)
	if strings.Contains(h, "key") && (strings.Contains(h, "class") || strings.Contains(h, "hierarch")) {
		return "architecture"
	}

	// Headings about specific directories/files typically describe modules
	if strings.Contains(b, "grails-app/") || strings.Contains(b, "src/main/") {
		if len(b) > 200 {
			return "modules"
		}
	}

	return "other"
}

// ImportToEntries converts an ImportResult into GeneratedEntry items for the knowledge base.
func ImportToEntries(ir *ImportResult, heroDir, date string) []GeneratedEntry {
	var entries []GeneratedEntry

	// 1. Architecture sections → context entries
	if len(ir.Architecture) > 0 {
		entry := buildArchitectureEntry(ir.Architecture, heroDir, date)
		if entry.Content != "" {
			entries = append(entries, entry)
		}
	}

	// 2. Convention sections → convention entries
	if len(ir.Conventions) > 0 {
		entry := buildConventionsEntry(ir.Conventions, heroDir, date)
		if entry.Content != "" {
			entries = append(entries, entry)
		}
	}

	// 3. Rule sections → rule entries
	if len(ir.Rules) > 0 {
		entry := buildRulesEntry(ir.Rules, heroDir, date)
		if entry.Content != "" {
			entries = append(entries, entry)
		}
	}

	// 4. Command sections → context entry for dev workflow
	if len(ir.Commands) > 0 {
		entry := buildCommandsEntry(ir.Commands, heroDir, date)
		if entry.Content != "" {
			entries = append(entries, entry)
		}
	}

	// 5. Module details → individual context entries per module/source file
	entries = append(entries, buildModuleEntries(ir.ModuleDetails, ir.Sources, heroDir, date)...)

	return entries
}

func buildArchitectureEntry(sections []ImportedSection, heroDir, date string) GeneratedEntry {
	slug := "architecture-overview"
	path := filepath.Join(heroDir, "knowledge", "context", slug, "spec.md")

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("title: Architecture Overview\n")
	sb.WriteString("type: context\n")
	sb.WriteString("status: active\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", date))
	sb.WriteString("tags: [imported, architecture]\n")
	sb.WriteString("---\n\n")

	for _, s := range sections {
		if s.Heading != "" {
			sb.WriteString(fmt.Sprintf("## %s\n\n", s.Heading))
		}
		if s.Body != "" {
			sb.WriteString(s.Body + "\n\n")
		}
	}

	sb.WriteString(fmt.Sprintf("<!-- Imported from: %s -->\n", sectionSources(sections)))

	return GeneratedEntry{
		Type:    "context",
		Slug:    slug,
		Path:    path,
		Content: sb.String(),
	}
}

func buildConventionsEntry(sections []ImportedSection, heroDir, date string) GeneratedEntry {
	slug := "project-conventions"
	path := filepath.Join(heroDir, "knowledge", "conventions", slug, "spec.md")

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("title: Project Conventions\n")
	sb.WriteString("type: convention\n")
	sb.WriteString("status: active\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", date))
	sb.WriteString("scope: [\"*\"]\n")
	sb.WriteString("tags: [imported, conventions]\n")
	sb.WriteString("---\n\n")

	for _, s := range sections {
		if s.Heading != "" {
			sb.WriteString(fmt.Sprintf("## %s\n\n", s.Heading))
		}
		if s.Body != "" {
			sb.WriteString(s.Body + "\n\n")
		}
	}

	sb.WriteString(fmt.Sprintf("<!-- Imported from: %s -->\n", sectionSources(sections)))

	return GeneratedEntry{
		Type:    "convention",
		Slug:    slug,
		Path:    path,
		Content: sb.String(),
	}
}

func buildRulesEntry(sections []ImportedSection, heroDir, date string) GeneratedEntry {
	slug := "project-rules"
	path := filepath.Join(heroDir, "knowledge", "rules", slug, "spec.md")

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("title: Project Rules\n")
	sb.WriteString("type: rule\n")
	sb.WriteString("status: active\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", date))
	sb.WriteString("scope: [\"*\"]\n")
	sb.WriteString("tags: [imported, rules]\n")
	sb.WriteString("---\n\n")

	for _, s := range sections {
		if s.Heading != "" {
			sb.WriteString(fmt.Sprintf("## %s\n\n", s.Heading))
		}
		if s.Body != "" {
			sb.WriteString(s.Body + "\n\n")
		}
	}

	sb.WriteString(fmt.Sprintf("<!-- Imported from: %s -->\n", sectionSources(sections)))

	return GeneratedEntry{
		Type:    "rule",
		Slug:    slug,
		Path:    path,
		Content: sb.String(),
	}
}

func buildCommandsEntry(sections []ImportedSection, heroDir, date string) GeneratedEntry {
	slug := "dev-workflow"
	path := filepath.Join(heroDir, "knowledge", "context", slug, "spec.md")

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("title: Development Workflow & Commands\n")
	sb.WriteString("type: context\n")
	sb.WriteString("status: active\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", date))
	sb.WriteString("tags: [imported, commands, dev-workflow]\n")
	sb.WriteString("---\n\n")

	for _, s := range sections {
		if s.Heading != "" {
			sb.WriteString(fmt.Sprintf("## %s\n\n", s.Heading))
		}
		if s.Body != "" {
			sb.WriteString(s.Body + "\n\n")
		}
	}

	sb.WriteString(fmt.Sprintf("<!-- Imported from: %s -->\n", sectionSources(sections)))

	return GeneratedEntry{
		Type:    "context",
		Slug:    slug,
		Path:    path,
		Content: sb.String(),
	}
}

// buildModuleEntries generates per-source-file context entries for instruction files
// that describe specific modules (e.g. .opencode/instructions/morpheus-core-agents.md).
func buildModuleEntries(moduleSections []ImportedSection, sources []ImportSource, heroDir, date string) []GeneratedEntry {
	var entries []GeneratedEntry

	// Group module sections by source file
	bySource := map[string][]ImportedSection{}
	for _, s := range moduleSections {
		bySource[s.Source] = append(bySource[s.Source], s)
	}

	// Also create entries for instruction files that are about specific modules
	// (these files as a whole describe a module, not just individual sections)
	for _, src := range sources {
		if src.Kind != "instructions" {
			continue
		}
		// If we already have module sections from this source, skip — they're already grouped
		if _, ok := bySource[src.Path]; ok {
			continue
		}
		// Create an entry for the entire instruction file
		baseName := strings.TrimSuffix(filepath.Base(src.Path), filepath.Ext(filepath.Base(src.Path)))
		slug := "module-" + slugify(baseName)
		path := filepath.Join(heroDir, "knowledge", "context", slug, "spec.md")

		// Extract title from first heading or filename
		title := baseName
		for _, line := range strings.Split(src.Content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimPrefix(line, "# ")
				break
			}
		}

		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("title: %s\n", title))
		sb.WriteString("type: context\n")
		sb.WriteString("status: active\n")
		sb.WriteString(fmt.Sprintf("created: %s\n", date))
		sb.WriteString("tags: [imported, module]\n")
		sb.WriteString("---\n\n")
		sb.WriteString(stripFrontmatter(src.Content))
		sb.WriteString(fmt.Sprintf("\n\n<!-- Imported from: %s -->\n", src.Path))

		entries = append(entries, GeneratedEntry{
			Type:    "context",
			Slug:    slug,
			Path:    path,
			Content: sb.String(),
		})
	}

	// Generate entries for grouped module sections from non-instruction sources
	for source, sects := range bySource {
		baseName := strings.TrimSuffix(filepath.Base(source), filepath.Ext(filepath.Base(source)))
		slug := "module-" + slugify(baseName)
		path := filepath.Join(heroDir, "knowledge", "context", slug, "spec.md")

		title := "Module Details"
		if len(sects) > 0 && sects[0].Heading != "" {
			title = sects[0].Heading
		}

		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("title: %s\n", title))
		sb.WriteString("type: context\n")
		sb.WriteString("status: active\n")
		sb.WriteString(fmt.Sprintf("created: %s\n", date))
		sb.WriteString("tags: [imported, module]\n")
		sb.WriteString("---\n\n")

		for _, s := range sects {
			if s.Heading != "" {
				sb.WriteString(fmt.Sprintf("## %s\n\n", s.Heading))
			}
			if s.Body != "" {
				sb.WriteString(s.Body + "\n\n")
			}
		}

		sb.WriteString(fmt.Sprintf("<!-- Imported from: %s -->\n", source))

		entries = append(entries, GeneratedEntry{
			Type:    "context",
			Slug:    slug,
			Path:    path,
			Content: sb.String(),
		})
	}

	return entries
}

// EnrichOverviewFromImport adds imported tech stack and architecture info
// to the project overview content.
func EnrichOverviewFromImport(overviewContent string, ir *ImportResult) string {
	var additions strings.Builder

	// Add tech stack info from imports
	if len(ir.TechStack) > 0 {
		additions.WriteString("\n## Imported Tech Stack Details\n\n")
		for _, s := range ir.TechStack {
			if s.Body != "" {
				additions.WriteString(s.Body + "\n\n")
			}
		}
	}

	// Add architecture summary from imports (brief reference, full content goes to architecture-overview)
	if len(ir.Architecture) > 0 {
		additions.WriteString("## Architecture Summary\n\n")
		additions.WriteString("Detailed architecture documentation is available in the architecture-overview knowledge entry.\n\n")
		// Include first architecture section as a brief summary
		first := ir.Architecture[0]
		if first.Body != "" {
			summary := truncate(first.Body, 500)
			additions.WriteString(summary + "\n\n")
		}
	}

	if additions.Len() == 0 {
		return overviewContent
	}

	// Insert before the "Current Gaps" section or at end
	gapsMarker := "## Current Gaps"
	if idx := strings.Index(overviewContent, gapsMarker); idx > 0 {
		return overviewContent[:idx] + additions.String() + overviewContent[idx:]
	}

	// Insert before the HTML comment at end
	commentMarker := "<!-- Add project-specific"
	if idx := strings.Index(overviewContent, commentMarker); idx > 0 {
		return overviewContent[:idx] + additions.String() + overviewContent[idx:]
	}

	return overviewContent + additions.String()
}

// stripFrontmatter removes YAML frontmatter (---...---) from markdown content.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	// Find the closing ---
	rest := content[3:]
	idx := strings.Index(rest, "---")
	if idx == -1 {
		return content
	}
	return strings.TrimSpace(rest[idx+3:])
}

// sectionSources returns a comma-separated list of unique source paths.
func sectionSources(sections []ImportedSection) string {
	seen := map[string]bool{}
	var sources []string
	for _, s := range sections {
		if !seen[s.Source] {
			sources = append(sources, s.Source)
			seen[s.Source] = true
		}
	}
	return strings.Join(sources, ", ")
}
