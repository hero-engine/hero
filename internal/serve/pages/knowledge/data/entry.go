package data

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/serve/mdrender"
)

// Entry is one knowledge-base entry resolved by slug. Body is the
// markdown content beneath the frontmatter; BodyHTML is a best-effort
// rendering of that markdown into safe HTML (paragraphs, headings,
// lists, code blocks, inline code, links). Relations carry forward
// frontmatter `relates-to` / `depends-on` / `supersedes` so the
// detail-view footer can render linked chips.
type Entry struct {
	Slug          string
	Kind          string // "note" | "convention" | "decision" | …
	Domain        string // directory name (notes, conventions, …)
	Title         string
	Type          string
	Status        string
	Path          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CreatedPretty string
	UpdatedPretty string
	Body          string
	BodyHTML      template.HTML
	Relations     []EntryRelation
}

// EntryRelation is one frontmatter cross-link from this entry to
// another. Kind is "relates-to" | "depends-on" | "supersedes" | … —
// preserved verbatim so the template can render whatever was authored.
type EntryRelation struct {
	Kind   string
	Target string
}

// LoadEntry resolves a knowledge entry by slug. Walks the
// `.hero/knowledge/` tree via collectKnowledgeFiles, covering three
// shapes: `<kind>/<slug>.md` (flat), `<kind>/<slug>/spec.md` (dir-style),
// and `<kind>/<nested>/<slug>.md` (one level deeper). Also handles
// loose `<slug>.md` files directly under the knowledge/ root for
// backwards-compatibility with the pre-v5 layout. Returns nil when the
// slug doesn't resolve. heroDir empty also returns nil.
func LoadEntry(heroDir, slug string) *Entry {
	if heroDir == "" || slug == "" {
		return nil
	}
	root := filepath.Join(heroDir, "knowledge")
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}

	// Subdir-scoped lookups (flat, dir-style, nested) via the shared
	// walk. Flat shape wins on (kind, slug) collisions per collectKnowledgeFiles.
	for _, ef := range collectKnowledgeFiles(root) {
		if ef.Slug == slug {
			return readEntry(ef.Path, ef.Kind, ef.Slug, ef.ModTime)
		}
	}

	// Backwards-compat: loose markdown files directly under knowledge/.
	loose := filepath.Join(root, slug+".md")
	if st, err := os.Stat(loose); err == nil && !st.IsDir() {
		return readEntry(loose, "", slug, st.ModTime())
	}
	return nil
}

// readEntry parses the file at path into an Entry. dir is the
// knowledge subdirectory name (e.g. "notes"); empty for loose files at
// the knowledge/ root. slug is supplied explicitly because dir-style
// entries (`<kind>/<slug>/spec.md`) need the parent dir name, not the
// basename, as the slug.
func readEntry(path, dir, slug string, modTime time.Time) *Entry {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	text := string(raw)
	front, body := splitFrontmatter(text)
	fm := parseFrontmatterFields(front)
	if slug == "" {
		slug = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	e := &Entry{
		Slug:          slug,
		Kind:          singularizeKind(dir),
		Domain:        dir,
		Path:          path,
		UpdatedAt:     modTime,
		UpdatedPretty: prettyAge(modTime),
	}
	if v := fm["title"]; v != "" {
		e.Title = v
	} else if t := firstHeading(body); t != "" {
		e.Title = t
	} else {
		e.Title = humanize(slug)
	}
	if v := fm["type"]; v != "" {
		e.Type = v
	}
	if v := fm["status"]; v != "" {
		e.Status = v
	}
	if v := fm["created"]; v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			e.CreatedAt = t
			e.CreatedPretty = t.Format("2006-01-02")
		}
	}
	if e.CreatedPretty == "" {
		e.CreatedAt = modTime
		e.CreatedPretty = modTime.Format("2006-01-02")
	}

	for _, kind := range []string{"relates-to", "depends-on", "supersedes"} {
		for _, target := range splitList(fm[kind]) {
			e.Relations = append(e.Relations, EntryRelation{Kind: kind, Target: target})
		}
	}

	e.Body = strings.TrimSpace(body)
	e.BodyHTML = mdrender.Render(e.Body)
	return e
}

// splitFrontmatter separates a leading `---`-delimited YAML block from
// the body. When no frontmatter is present the whole input is the body
// and frontmatter is empty.
func splitFrontmatter(text string) (front, body string) {
	if !strings.HasPrefix(text, "---") {
		return "", text
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", text
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n")
		}
	}
	return "", text
}

// parseFrontmatterFields is a thin top-level key:value extractor.
// Nested YAML is ignored — we only need flat keys (title, type,
// status, created, relates-to, depends-on, supersedes).
func parseFrontmatterFields(front string) map[string]string {
	out := map[string]string{}
	if front == "" {
		return out
	}
	for _, line := range strings.Split(front, "\n") {
		raw := line
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Skip indented continuation lines.
		if len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\t') {
			continue
		}
		colon := strings.Index(trimmed, ":")
		if colon < 0 {
			continue
		}
		k := strings.TrimSpace(trimmed[:colon])
		v := strings.TrimSpace(trimmed[colon+1:])
		v = strings.Trim(v, `"'`)
		out[k] = v
	}
	return out
}

// splitList parses a YAML scalar list — either `[a, b, c]` or `a, b`
// — into a trimmed slice. Empty input returns nil.
func splitList(val string) []string {
	val = strings.TrimSpace(val)
	val = strings.Trim(val, "[]")
	if val == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(val, ",") {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// firstHeading returns the first H1 found in body content, without
// the leading `# `. Returns empty when no heading is present.
func firstHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		}
	}
	return ""
}
