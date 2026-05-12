package context

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ContextEntry is a unified representation of a single knowledge item.
type ContextEntry struct {
	Type  string // "convention", "rule", "decision", "past_work", "external", "risk"
	Slug  string
	Title string
	Body  string
	Path  string
}

// Format dispatches to the appropriate formatter based on the format string.
// Valid formats: "json", "yaml", "compact", "opencode", "claude", "cursorrules", "pipe".
// Any other value (including empty string) produces Markdown.
func Format(format string, entries []ContextEntry, files []string) string {
	switch format {
	case "json":
		return FormatJSON(entries, files)
	case "yaml":
		return FormatYAML(entries, files)
	case "compact":
		return FormatCompact(entries, files)
	case "opencode":
		return FormatOpenCode(entries, files)
	case "claude":
		return FormatClaude(entries, files)
	case "cursorrules":
		return FormatCursorRules(entries, files)
	case "pipe":
		return FormatPipe(entries, files)
	default:
		return FormatMarkdown(entries, files)
	}
}

// FormatMarkdown renders context as human-readable Markdown.
func FormatMarkdown(entries []ContextEntry, files []string) string {
	if len(entries) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("## Relevant context from spec corpus\n\n")

	sections := []struct {
		typeKey string
		heading string
	}{
		{"tripwire", "### Tripwires (do not violate)"},
		{"rule", "### Rules (hard constraints)"},
		{"convention", "### Conventions to follow"},
		{"decision", "### Decisions that apply"},
		{"past_work", "### Past work in this area"},
		{"risk", "### Known risks in this area"},
		{"external", "### External references"},
	}

	for _, sec := range sections {
		var group []ContextEntry
		for _, e := range entries {
			if e.Type == sec.typeKey {
				group = append(group, e)
			}
		}
		if len(group) == 0 {
			continue
		}
		sb.WriteString(sec.heading)
		sb.WriteString("\n")
		for _, e := range group {
			body := e.Body
			if body == "" {
				body = e.Title
			}
			if e.Path != "" {
				fmt.Fprintf(&sb, "- **%s**: %s (path: %s)\n", e.Slug, body, e.Path)
			} else {
				fmt.Fprintf(&sb, "- **%s**: %s\n", e.Slug, body)
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "Context generated for %d file(s): %s\n", len(files), strings.Join(files, ", "))

	return sb.String()
}

// jsonKnowledgeEntry is the JSON wire format for a single entry.
type jsonKnowledgeEntry struct {
	Type  string `json:"type"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
	Path  string `json:"path,omitempty"`
}

// jsonOutput is the top-level JSON structure.
type jsonOutput struct {
	Files       []string             `json:"files"`
	Knowledge   []jsonKnowledgeEntry `json:"knowledge"`
	GeneratedAt string               `json:"generated_at"`
}

// FormatJSON renders context as a JSON object.
func FormatJSON(entries []ContextEntry, files []string) string {
	knowledge := make([]jsonKnowledgeEntry, 0, len(entries))
	for _, e := range entries {
		knowledge = append(knowledge, jsonKnowledgeEntry{
			Type:  e.Type,
			Slug:  e.Slug,
			Title: e.Title,
			Body:  e.Body,
			Path:  e.Path,
		})
	}

	if files == nil {
		files = []string{}
	}

	out := jsonOutput{
		Files:       files,
		Knowledge:   knowledge,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// yamlEscape returns a YAML-safe single-quoted string.
// Single quotes in the value are escaped by doubling them.
func yamlEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// FormatYAML renders context as hand-rendered YAML (no external library).
func FormatYAML(entries []ContextEntry, files []string) string {
	var sb strings.Builder

	sb.WriteString("generated_at: ")
	sb.WriteString(yamlEscape(time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString("\n")

	sb.WriteString("files:\n")
	for _, f := range files {
		fmt.Fprintf(&sb, "  - %s\n", yamlEscape(f))
	}

	sb.WriteString("knowledge:\n")
	for _, e := range entries {
		fmt.Fprintf(&sb, "  - type: %s\n", yamlEscape(e.Type))
		fmt.Fprintf(&sb, "    slug: %s\n", yamlEscape(e.Slug))
		fmt.Fprintf(&sb, "    title: %s\n", yamlEscape(e.Title))
		if e.Body != "" {
			fmt.Fprintf(&sb, "    body: %s\n", yamlEscape(e.Body))
		}
		if e.Path != "" {
			fmt.Fprintf(&sb, "    path: %s\n", yamlEscape(e.Path))
		}
	}

	return sb.String()
}

// stripMarkdown removes common Markdown markup from text, collapsing
// bold/italic markers, headers, and excess whitespace.
func stripMarkdown(s string) string {
	// Remove bold/italic markers
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", " ")

	// Remove heading markers (# ## ### etc at line start)
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, "#")
		if trimmed != line {
			line = strings.TrimSpace(trimmed)
		}
		out = append(out, line)
	}
	s = strings.Join(out, " ")

	// Collapse whitespace
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// FormatCompact strips Markdown and collapses whitespace to a dense plain-text block.
func FormatCompact(entries []ContextEntry, files []string) string {
	if len(entries) == 0 {
		return ""
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "files: %s\n", strings.Join(files, ", "))

	for _, e := range entries {
		body := stripMarkdown(e.Body)
		if body == "" {
			body = stripMarkdown(e.Title)
		}
		if e.Path != "" {
			fmt.Fprintf(&sb, "[%s] %s: %s (path: %s)\n", e.Type, e.Slug, body, e.Path)
		} else {
			fmt.Fprintf(&sb, "[%s] %s: %s\n", e.Type, e.Slug, body)
		}
	}

	return sb.String()
}

// FormatOpenCode wraps context in a <hero_context> XML element understood by OpenCode.
func FormatOpenCode(entries []ContextEntry, files []string) string {
	var sb strings.Builder

	sb.WriteString("<hero_context>\n")
	sb.WriteString("<files>\n")
	for _, f := range files {
		fmt.Fprintf(&sb, "  <file>%s</file>\n", xmlEscape(f))
	}
	sb.WriteString("</files>\n")

	sb.WriteString("<knowledge>\n")
	for _, e := range entries {
		fmt.Fprintf(&sb, "  <entry type=%q slug=%q>\n", e.Type, e.Slug)
		fmt.Fprintf(&sb, "    <title>%s</title>\n", xmlEscape(e.Title))
		if e.Body != "" {
			fmt.Fprintf(&sb, "    <body>%s</body>\n", xmlEscape(e.Body))
		}
		if e.Path != "" {
			fmt.Fprintf(&sb, "    <path>%s</path>\n", xmlEscape(e.Path))
		}
		sb.WriteString("  </entry>\n")
	}
	sb.WriteString("</knowledge>\n")

	fmt.Fprintf(&sb, "<generated_at>%s</generated_at>\n", time.Now().UTC().Format(time.RFC3339))
	sb.WriteString("</hero_context>\n")

	return sb.String()
}

// FormatClaude renders context as a typed XML block suitable for Claude's system prompt.
func FormatClaude(entries []ContextEntry, files []string) string {
	var sb strings.Builder

	sb.WriteString("<context>\n")
	sb.WriteString("  <source>hero</source>\n")
	fmt.Fprintf(&sb, "  <generated_at>%s</generated_at>\n", time.Now().UTC().Format(time.RFC3339))

	sb.WriteString("  <files>\n")
	for _, f := range files {
		fmt.Fprintf(&sb, "    <file>%s</file>\n", xmlEscape(f))
	}
	sb.WriteString("  </files>\n")

	sections := []struct {
		typeKey string
		tag     string
	}{
		{"tripwire", "tripwires"},
		{"rule", "rules"},
		{"convention", "conventions"},
		{"decision", "decisions"},
		{"past_work", "past_work"},
		{"risk", "risks"},
		{"external", "external_references"},
	}

	for _, sec := range sections {
		var group []ContextEntry
		for _, e := range entries {
			if e.Type == sec.typeKey {
				group = append(group, e)
			}
		}
		if len(group) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "  <%s>\n", sec.tag)
		for _, e := range group {
			fmt.Fprintf(&sb, "    <item slug=%q>\n", e.Slug)
			fmt.Fprintf(&sb, "      <title>%s</title>\n", xmlEscape(e.Title))
			if e.Body != "" {
				fmt.Fprintf(&sb, "      <body>%s</body>\n", xmlEscape(e.Body))
			}
			if e.Path != "" {
				fmt.Fprintf(&sb, "      <path>%s</path>\n", xmlEscape(e.Path))
			}
			sb.WriteString("    </item>\n")
		}
		fmt.Fprintf(&sb, "  </%s>\n", sec.tag)
	}

	sb.WriteString("</context>\n")

	return sb.String()
}

// FormatCursorRules renders context in the .cursorrules convention format.
func FormatCursorRules(entries []ContextEntry, files []string) string {
	var sb strings.Builder

	sb.WriteString("# Hero context\n\n")

	fmt.Fprintf(&sb, "Generated for: %s\n\n", strings.Join(files, ", "))

	sections := []struct {
		typeKey string
		heading string
	}{
		{"tripwire", "## Tripwires (do not violate)"},
		{"rule", "## Rules (hard constraints)"},
		{"convention", "## Conventions"},
		{"decision", "## Decisions"},
		{"past_work", "## Past work"},
		{"risk", "## Known risks"},
		{"external", "## External references"},
	}

	for _, sec := range sections {
		var group []ContextEntry
		for _, e := range entries {
			if e.Type == sec.typeKey {
				group = append(group, e)
			}
		}
		if len(group) == 0 {
			continue
		}
		sb.WriteString(sec.heading)
		sb.WriteString("\n\n")
		for _, e := range group {
			body := e.Body
			if body == "" {
				body = e.Title
			}
			fmt.Fprintf(&sb, "### %s\n", e.Slug)
			fmt.Fprintf(&sb, "%s\n", body)
			if e.Path != "" {
				fmt.Fprintf(&sb, "_Source: %s_\n", e.Path)
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// pipeEntry is the per-line JSON structure for the pipe format.
type pipeEntry struct {
	Type  string `json:"type"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
	Path  string `json:"path,omitempty"`
}

// FormatPipe renders context as newline-delimited JSON (one entry per line).
func FormatPipe(entries []ContextEntry, files []string) string {
	var sb strings.Builder

	for _, e := range entries {
		pe := pipeEntry{
			Type:  e.Type,
			Slug:  e.Slug,
			Title: e.Title,
			Body:  e.Body,
			Path:  e.Path,
		}
		b, err := json.Marshal(pe)
		if err != nil {
			continue
		}
		sb.Write(b)
		sb.WriteString("\n")
	}

	return sb.String()
}

// xmlEscape escapes the five predefined XML entities.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
