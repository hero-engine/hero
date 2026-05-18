package data

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/serve/mdrender"
	"github.com/hero-engine/hero/internal/spec"
)

// SpecDetail is one work spec resolved by slug, ready for the detail
// template. Wraps the parsed spec.Spec with rendered markdown and a
// link list of child specs (for initiatives).
type SpecDetail struct {
	Slug       string
	Title      string
	Type       string
	Status     string
	Horizon    string
	Path       string
	Goal       string
	BodyHTML   template.HTML
	Relations  []SpecDetailRelation
	IsInitiative bool
}

// SpecDetailRelation is one frontmatter cross-link on a spec.
type SpecDetailRelation struct {
	Kind   string
	Target string
}

// LoadSpec resolves a work spec by slug. Search order matches the
// project's lifecycle: completed specs first, then in-flight planning
// directories. Returns nil when the slug doesn't resolve anywhere.
func LoadSpec(heroDir, slug string) *SpecDetail {
	if heroDir == "" || slug == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(heroDir, "specs", slug, "spec.md"),
		filepath.Join(heroDir, "planning", "features", slug, "spec.md"),
		filepath.Join(heroDir, "planning", "bugs", slug, "spec.md"),
		filepath.Join(heroDir, "planning", "initiatives", slug, "spec.md"),
	}
	for _, path := range candidates {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			parsed, err := spec.ParseFile(path)
			if err != nil {
				return nil
			}
			return projectSpec(parsed)
		}
	}
	// Three-file layout fallback — requirements.md/design.md/tasks.md.
	dirs := []string{
		filepath.Join(heroDir, "specs", slug),
		filepath.Join(heroDir, "planning", "features", slug),
		filepath.Join(heroDir, "planning", "bugs", slug),
		filepath.Join(heroDir, "planning", "initiatives", slug),
	}
	for _, dir := range dirs {
		req := filepath.Join(dir, "requirements.md")
		if st, err := os.Stat(req); err == nil && !st.IsDir() {
			parsed, err := spec.ParseThreeFile(dir)
			if err != nil {
				return nil
			}
			return projectSpec(parsed)
		}
	}
	return nil
}

// projectSpec adapts a parsed spec.Spec into the detail-template
// payload. Strips the spec's frontmatter from RawContent so the
// rendered body doesn't show the YAML.
func projectSpec(s *spec.Spec) *SpecDetail {
	body := stripFrontmatter(s.RawContent)
	out := &SpecDetail{
		Slug:         s.Slug,
		Title:        s.Title,
		Type:         string(s.Type),
		Status:       string(s.Status),
		Horizon:      string(s.Horizon),
		Path:         s.Path,
		Goal:         s.Sections["goal"],
		BodyHTML:     mdrender.Render(body),
		IsInitiative: s.Type == spec.TypeInitiative,
	}
	if out.Title == "" {
		out.Title = humanizeSlug(s.Slug)
	}
	for _, r := range s.Relations {
		out.Relations = append(out.Relations, SpecDetailRelation{Kind: r.Kind, Target: r.Target})
	}
	return out
}

// stripFrontmatter drops a leading `---`-delimited YAML block so the
// rendered detail body shows only the prose.
func stripFrontmatter(content string) string {
	if len(content) < 4 || content[:3] != "---" {
		return content
	}
	// Find the closing ---. Look for `\n---` followed by EOF or newline.
	end := -1
	for i := 3; i < len(content)-3; i++ {
		if content[i] == '\n' && content[i+1] == '-' && content[i+2] == '-' && content[i+3] == '-' {
			// Confirm the closing delimiter ends the line.
			j := i + 4
			if j >= len(content) || content[j] == '\n' || content[j] == '\r' {
				end = j
				break
			}
		}
	}
	if end < 0 {
		return content
	}
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[end:]
}

// humanizeSlug is a local copy of the corpus helper — kept independent
// so this package doesn't reach across into knowledge/data.
func humanizeSlug(slug string) string {
	if slug == "" {
		return ""
	}
	out := []rune(slug)
	if out[0] >= 'a' && out[0] <= 'z' {
		out[0] -= 32
	}
	for i := range out {
		if out[i] == '-' {
			out[i] = ' '
		}
	}
	return string(out)
}
