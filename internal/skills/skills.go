package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a parsed .hero/skills/<name>.md file.
type Skill struct {
	Title   string
	Slug    string
	Version int
	Tags    []string
	Author  string
	Created string
	Steps   []Step
	Params  []Param
	Notes   string
	Path    string
}

// Step represents a single step in a skill workflow.
type Step struct {
	Index int
	Kind  string // "hero", "shell", "prompt"
	Raw   string // original line
	Cmd   string // command to run (shell or hero command)
	Text  string // for prompt steps
}

// Param represents a parameter declaration in a skill.
type Param struct {
	Name        string
	Description string
}

// ParseSkillFile reads and parses a .hero/skills/<name>.md file.
func ParseSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	skill := &Skill{
		Path: path,
		Slug: skillSlugFromPath(path),
	}

	content := string(data)
	body := skill.parseFrontmatter(content)
	skill.parseBody(body)

	return skill, nil
}

// Discover finds all skill files in the given skillsDir.
func Discover(skillsDir string) ([]*Skill, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var skills []*Skill
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(skillsDir, name)
		skill, err := ParseSkillFile(path)
		if err != nil {
			continue // skip unparseable files
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

// InterpolateParams replaces {{param}} placeholders with actual values.
// Returns an error if a placeholder has no corresponding value in params.
func InterpolateParams(text string, params map[string]string) (string, error) {
	result := text
	// Find all {{...}} placeholders
	for {
		start := strings.Index(result, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end < 0 {
			break
		}
		end += start
		name := strings.TrimSpace(result[start+2 : end])
		val, ok := params[name]
		if !ok {
			return "", fmt.Errorf("missing value for parameter %q", name)
		}
		result = result[:start] + val + result[end+2:]
	}
	return result, nil
}

// parseFrontmatter extracts YAML-like frontmatter delimited by ---.
// Returns the body content after the frontmatter.
func (s *Skill) parseFrontmatter(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	first := strings.TrimSpace(lines[0])
	if first != "---" {
		return content
	}

	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 {
		return content
	}

	for i := 1; i < closeIdx; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])

		switch key {
		case "title":
			s.Title = val
		case "slug":
			s.Slug = val
		case "version":
			var v int
			fmt.Sscanf(val, "%d", &v)
			s.Version = v
		case "tags":
			s.Tags = parseSkillList(val)
		case "author":
			s.Author = val
		case "created":
			s.Created = val
		}
	}

	return strings.Join(lines[closeIdx+1:], "\n")
}

// parseBody parses the body for ## Steps, ## Parameters, and ## Notes sections.
func (s *Skill) parseBody(body string) {
	lines := strings.Split(body, "\n")

	type section struct {
		name  string
		lines []string
	}

	var sections []section
	var current *section

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if current != nil {
				sections = append(sections, *current)
			}
			name := strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			current = &section{name: name}
		} else if current != nil {
			current.lines = append(current.lines, line)
		} else if s.Title == "" && strings.HasPrefix(trimmed, "# ") {
			s.Title = strings.TrimPrefix(trimmed, "# ")
		}
	}
	if current != nil {
		sections = append(sections, *current)
	}

	for _, sec := range sections {
		switch sec.name {
		case "steps":
			s.Steps = parseSteps(sec.lines)
		case "parameters", "params":
			s.Params = parseParams(sec.lines)
		case "notes":
			s.Notes = strings.TrimSpace(strings.Join(sec.lines, "\n"))
		}
	}
}

// parseSteps parses numbered list items from the Steps section.
func parseSteps(lines []string) []Step {
	var steps []Step
	index := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Match numbered list items: "1. ...", "2. ...", etc.
		rest, ok := stripNumberedPrefix(trimmed)
		if !ok {
			continue
		}

		index++
		step := Step{
			Index: index,
			Raw:   trimmed,
		}

		if strings.HasPrefix(rest, "Run: ") {
			step.Kind = "shell"
			step.Cmd = strings.TrimPrefix(rest, "Run: ")
		} else if strings.HasPrefix(rest, "Prompt agent: ") {
			step.Kind = "prompt"
			step.Text = strings.TrimPrefix(rest, "Prompt agent: ")
		} else if isHeroCommand(rest) {
			step.Kind = "hero"
			step.Cmd = rest
		} else {
			step.Kind = "prompt"
			step.Text = rest
		}

		steps = append(steps, step)
	}
	return steps
}

// stripNumberedPrefix strips a leading "N. " or "N) " from a line.
// Returns (rest, true) if matched, ("", false) otherwise.
func stripNumberedPrefix(line string) (string, bool) {
	for i, ch := range line {
		if ch >= '0' && ch <= '9' {
			continue
		}
		if i == 0 {
			return "", false
		}
		if ch == '.' || ch == ')' {
			rest := strings.TrimSpace(line[i+1:])
			return rest, true
		}
		return "", false
	}
	return "", false
}

// isHeroCommand returns true if a string looks like a hero sub-command invocation.
func isHeroCommand(s string) bool {
	heroCommands := []string{
		"hero context", "hero ask", "hero check", "hero claim", "hero complete",
		"hero conflicts", "hero diff", "hero do", "hero graph", "hero index",
		"hero knowledge", "hero new", "hero note", "hero nudge", "hero prime",
		"hero pull", "hero replay", "hero report", "hero scan", "hero search",
		"hero serve", "hero skill", "hero sprint", "hero status", "hero sync",
		"hero triage", "hero validate", "hero watch", "hero wiki", "hero pulse",
		"hero link", "hero uninstall", "hero upgrade", "hero install", "hero init",
		"hero cost", "hero dashboard", "hero import", "hero connect", "hero hooks",
		"hero hook", "hero mock",
	}
	for _, cmd := range heroCommands {
		if strings.HasPrefix(s, cmd) {
			return true
		}
	}
	return false
}

// parseParams parses parameter declarations from lines like:
// - `name` — description
func parseParams(lines []string) []Param {
	var params []Param
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		rest := strings.TrimSpace(trimmed[1:])

		// Extract backtick-quoted name
		if !strings.HasPrefix(rest, "`") {
			continue
		}
		closeBacktick := strings.Index(rest[1:], "`")
		if closeBacktick < 0 {
			continue
		}
		name := rest[1 : closeBacktick+1]
		after := strings.TrimSpace(rest[closeBacktick+2:])

		// Strip leading em-dash or regular dash separator
		desc := after
		if strings.HasPrefix(desc, "—") {
			desc = strings.TrimSpace(desc[len("—"):])
		} else if strings.HasPrefix(desc, "-") {
			desc = strings.TrimSpace(desc[1:])
		}

		params = append(params, Param{Name: name, Description: desc})
	}
	return params
}

// parseSkillList parses a comma-separated or bracket-enclosed list.
func parseSkillList(val string) []string {
	val = strings.TrimSpace(val)
	val = strings.Trim(val, "[]")
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// skillSlugFromPath returns the skill slug from a file path (basename without .md).
func skillSlugFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".md")
}
