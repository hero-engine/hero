package index

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// DiscoverKnowledge walks .hero/knowledge/** and returns the hand-authored
// files that work-spec Discover does NOT already index — i.e. flat `<name>.md`
// knowledge (typed or untyped) in any subdir. It is deliberately
// layout-agnostic: a knowledge entry surfaces regardless of shape, because
// following the <slug>/spec.md convention is a nudge, not a precondition for
// being found (see the knowledge-surfacing initiative and its ADR).
//
// Dedup rule: spec.md-shaped and three-file knowledge is already loaded by
// spec.Discover into the specs table (reachable via `hero ask`/`search`), so
// those paths — and any sidecar files in a spec-owned directory — are skipped
// here to avoid double-indexing. raw/ has its own graph ingest and is skipped.
func DiscoverKnowledge(heroDir string) ([]*KnowledgeEntry, error) {
	knowledgeDir := filepath.Join(heroDir, "knowledge")
	if info, err := os.Stat(knowledgeDir); err != nil || !info.IsDir() {
		return nil, nil
	}

	// Paths work-spec Discover already owns (spec.md, three-file).
	work, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}
	claimed := make(map[string]bool, len(work))
	for _, s := range work {
		if s != nil {
			claimed[s.Path] = true
		}
	}

	var out []*KnowledgeEntry
	err = filepath.Walk(knowledgeDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			// raw/ is the immutable audit copy with its own graph ingest.
			if info.Name() == "raw" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		if claimed[path] {
			return nil
		}
		// A directory holding a spec.md is spec-owned; skip its sidecar
		// files (delivery-audit.md, retro.md, …) — they belong to the spec.
		if info.Name() != "spec.md" {
			if _, statErr := os.Stat(filepath.Join(filepath.Dir(path), "spec.md")); statErr == nil {
				return nil
			}
		}
		if e := parseKnowledgeFile(knowledgeDir, path); e != nil {
			out = append(out, e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// parseKnowledgeFile builds a KnowledgeEntry from a single file, best-effort:
// frontmatter is used when present, otherwise the title falls back to the first
// H1 or the filename, and the kind is always the first subdir under knowledge/.
// A file is never dropped for being malformed.
func parseKnowledgeFile(knowledgeDir, path string) *KnowledgeEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)

	rel, relErr := filepath.Rel(knowledgeDir, path)
	if relErr != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(rel)
	kind := rel
	if i := strings.IndexByte(rel, '/'); i >= 0 {
		kind = rel[:i]
	} else {
		// File sits directly under knowledge/ with no subdir.
		kind = "knowledge"
	}
	slug := strings.TrimSuffix(rel, ".md")

	var title, ftype, domain string
	var tags, scope, triggers []string
	if s, perr := spec.ParseFile(path); perr == nil && s != nil {
		title = strings.TrimSpace(s.Title)
		ftype = string(s.Type)
		domain = s.Domain
		tags = s.Tags
		scope = s.Scope
		triggers = s.Triggers
	}
	if title == "" {
		title = firstH1(content)
	}
	if title == "" {
		base := filepath.Base(path)
		title = strings.TrimSuffix(base, filepath.Ext(base))
	}

	mt := time.Time{}
	if fi, statErr := os.Stat(path); statErr == nil {
		mt = fi.ModTime()
	}

	return &KnowledgeEntry{
		Slug:       slug,
		Title:      title,
		Kind:       kind,
		Type:       ftype,
		Path:       path,
		Domain:     domain,
		Tags:       tags,
		Scope:      scope,
		Triggers:   triggers,
		Content:    content,
		ModifiedAt: mt,
	}
}

// firstH1 returns the text of the first `# ` heading, ignoring frontmatter.
func firstH1(content string) string {
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(t[2:])
		}
	}
	return ""
}
