// Package errpattern manages a catalog of common error patterns accumulated
// from diagnose sessions. Patterns are stored as markdown files with YAML-like
// frontmatter under .hero/knowledge/error-patterns/.
package errpattern

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Pattern represents a cataloged error pattern.
type Pattern struct {
	ID        string   `yaml:"id"`       // slug-like identifier
	PatternRe string   `yaml:"pattern"`  // regex to match error text
	Stack     []string `yaml:"stack"`    // e.g. ["go", "postgres"]
	Severity  string   `yaml:"severity"` // common, rare, critical
	Files     []string `yaml:"files"`    // relevant file paths
	Symptom   string   `yaml:"-"`        // parsed from markdown body
	RootCause string   `yaml:"-"`        // parsed from markdown body
	Fix       string   `yaml:"-"`        // parsed from markdown body
	Path      string   `yaml:"-"`        // file path on disk
}

// LoadPatterns reads all .md files from .hero/knowledge/error-patterns/ and
// parses their frontmatter and body sections.
func LoadPatterns(heroDir string) ([]Pattern, error) {
	dir := filepath.Join(heroDir, "knowledge", "error-patterns")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var patterns []Pattern
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		p := parsePattern(string(data))
		p.Path = path
		if p.ID == "" {
			p.ID = strings.TrimSuffix(e.Name(), ".md")
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}

// MatchByFile returns patterns whose Files field overlaps with the given file paths.
func MatchByFile(patterns []Pattern, filePaths []string) []Pattern {
	pathSet := make(map[string]bool, len(filePaths))
	for _, fp := range filePaths {
		pathSet[fp] = true
	}

	var matched []Pattern
	for _, p := range patterns {
		for _, f := range p.Files {
			if pathSet[f] {
				matched = append(matched, p)
				break
			}
		}
	}
	return matched
}

// MatchByError returns patterns whose PatternRe regex matches the error text.
func MatchByError(patterns []Pattern, errorText string) []Pattern {
	var matched []Pattern
	for _, p := range patterns {
		if p.PatternRe == "" {
			continue
		}
		re, err := regexp.Compile(p.PatternRe)
		if err != nil {
			continue
		}
		if re.MatchString(errorText) {
			matched = append(matched, p)
		}
	}
	return matched
}

// SavePattern writes a pattern file to .hero/knowledge/error-patterns/{id}.md.
func SavePattern(heroDir string, p Pattern) error {
	dir := filepath.Join(heroDir, "knowledge", "error-patterns")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating error-patterns dir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("id: " + p.ID + "\n")
	if p.PatternRe != "" {
		sb.WriteString("pattern: " + p.PatternRe + "\n")
	}
	if len(p.Stack) > 0 {
		sb.WriteString("stack: [" + strings.Join(p.Stack, ", ") + "]\n")
	}
	if p.Severity != "" {
		sb.WriteString("severity: " + p.Severity + "\n")
	}
	if len(p.Files) > 0 {
		sb.WriteString("files: [" + strings.Join(p.Files, ", ") + "]\n")
	}
	sb.WriteString("---\n\n")

	if p.Symptom != "" {
		sb.WriteString("## Symptom\n\n" + p.Symptom + "\n\n")
	}
	if p.RootCause != "" {
		sb.WriteString("## Root Cause\n\n" + p.RootCause + "\n\n")
	}
	if p.Fix != "" {
		sb.WriteString("## Fix\n\n" + p.Fix + "\n\n")
	}

	path := filepath.Join(dir, p.ID+".md")
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// FormatPatterns returns a markdown-formatted summary of the given patterns.
func FormatPatterns(patterns []Pattern) string {
	if len(patterns) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Known Error Patterns\n\n")
	for _, p := range patterns {
		sb.WriteString("### " + p.ID)
		if p.Severity != "" {
			sb.WriteString(" [" + p.Severity + "]")
		}
		sb.WriteString("\n")
		if p.PatternRe != "" {
			sb.WriteString("Pattern: `" + p.PatternRe + "`\n")
		}
		if len(p.Stack) > 0 {
			sb.WriteString("Stack: " + strings.Join(p.Stack, ", ") + "\n")
		}
		if len(p.Files) > 0 {
			sb.WriteString("Files: " + strings.Join(p.Files, ", ") + "\n")
		}
		if p.Symptom != "" {
			sb.WriteString("\n**Symptom:** " + p.Symptom + "\n")
		}
		if p.RootCause != "" {
			sb.WriteString("\n**Root Cause:** " + p.RootCause + "\n")
		}
		if p.Fix != "" {
			sb.WriteString("\n**Fix:** " + p.Fix + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// parsePattern parses a markdown file into a Pattern, extracting frontmatter
// fields and body sections.
func parsePattern(content string) Pattern {
	var p Pattern
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		p.Symptom = extractSection(content, "Symptom")
		p.RootCause = extractSection(content, "Root Cause")
		p.Fix = extractSection(content, "Fix")
		return p
	}

	// Find closing ---
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx < 0 {
		return p
	}

	// Parse frontmatter key: value pairs
	for _, line := range lines[1:endIdx] {
		key, val, ok := parseKV(line)
		if !ok {
			continue
		}
		switch key {
		case "id":
			p.ID = val
		case "pattern":
			p.PatternRe = val
		case "severity":
			p.Severity = val
		case "stack":
			p.Stack = parseList(val)
		case "files":
			p.Files = parseList(val)
		}
	}

	body := strings.Join(lines[endIdx+1:], "\n")
	p.Symptom = extractSection(body, "Symptom")
	p.RootCause = extractSection(body, "Root Cause")
	p.Fix = extractSection(body, "Fix")

	return p
}

// parseKV splits "key: value" and returns (key, value, true).
func parseKV(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	return key, val, true
}

// parseList parses "[a, b, c]" into []string{"a", "b", "c"}.
func parseList(val string) []string {
	val = strings.TrimPrefix(val, "[")
	val = strings.TrimSuffix(val, "]")
	parts := strings.Split(val, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// extractSection extracts the text under a ## heading until the next ## or EOF.
func extractSection(body, heading string) string {
	marker := "## " + heading
	idx := strings.Index(body, marker)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(marker):]
	// Find next ## heading
	nextIdx := strings.Index(rest, "\n## ")
	if nextIdx >= 0 {
		rest = rest[:nextIdx]
	}
	return strings.TrimSpace(rest)
}
