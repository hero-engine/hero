package data

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// fabricateKnowledgeTree creates a `<heroDir>/knowledge/` tree with all
// three entry shapes plus skippable dirs and returns the heroDir. The
// returned tree contains 6 entry-bearing files (excluding hidden /
// underscore-prefixed dirs):
//
//	knowledge/notes/flat-note.md            (flat)
//	knowledge/notes/dir-note/spec.md        (dir-style)
//	knowledge/context/dev-workflow/spec.md  (dir-style)
//	knowledge/rules/project-rules/auto.md   (nested .md)
//	knowledge/rules/project-rules/spec.md   (dir-style under nested dir)
//	knowledge/decisions/loose-decision.md   (flat)
//	knowledge/.cache/hidden.md              (SKIPPED — dotfile)
//	knowledge/notes/_draft/spec.md          (SKIPPED — underscore)
func fabricateKnowledgeTree(t *testing.T) string {
	t.Helper()
	heroDir := t.TempDir()
	root := filepath.Join(heroDir, "knowledge")

	mk := func(rel, body string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}

	mk("notes/flat-note.md", "# Flat note\n\nbody\n")
	mk("notes/dir-note/spec.md", "# Dir note\n\nbody\n")
	mk("context/dev-workflow/spec.md", "---\ntitle: Dev workflow\n---\nbody\n")
	mk("rules/project-rules/auto.md", "# Auto rule\n\nbody\n")
	mk("rules/project-rules/spec.md", "# Project rules\n\nbody\n")
	mk("decisions/loose-decision.md", "# Loose decision\n\nbody\n")
	mk(".cache/hidden.md", "# Hidden\n")
	mk("notes/_draft/spec.md", "# Draft\n")

	return heroDir
}

func TestCollectKnowledgeFiles_AllThreeShapes(t *testing.T) {
	heroDir := fabricateKnowledgeTree(t)
	root := filepath.Join(heroDir, "knowledge")

	files := collectKnowledgeFiles(root)

	type kindSlug struct{ kind, slug string }
	got := map[kindSlug]bool{}
	for _, f := range files {
		got[kindSlug{f.Kind, f.Slug}] = true
	}

	want := []kindSlug{
		{"notes", "flat-note"},
		{"notes", "dir-note"},
		{"context", "dev-workflow"},
		{"rules", "auto"},
		{"rules", "project-rules"},
		{"decisions", "loose-decision"},
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("missing entry %s/%s in result", w.kind, w.slug)
		}
	}
	if len(files) != len(want) {
		t.Errorf("unexpected entry count: got %d (%v), want %d", len(files), files, len(want))
	}
}

func TestCollectKnowledgeFiles_SkipsHiddenAndUnderscoreDirs(t *testing.T) {
	heroDir := fabricateKnowledgeTree(t)
	root := filepath.Join(heroDir, "knowledge")

	for _, f := range collectKnowledgeFiles(root) {
		if filepath.Base(filepath.Dir(f.Path)) == "_draft" {
			t.Errorf("underscore-prefixed dir was walked: %v", f)
		}
		if filepath.Base(filepath.Dir(filepath.Dir(f.Path))) == ".cache" {
			t.Errorf("dotfile dir was walked: %v", f)
		}
	}
}

func TestCollectKnowledgeFiles_FlatWinsOnCollision(t *testing.T) {
	heroDir := t.TempDir()
	root := filepath.Join(heroDir, "knowledge")
	mk := func(rel, body string) {
		full := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(full), 0o755)
		_ = os.WriteFile(full, []byte(body), 0o644)
	}
	mk("notes/dup.md", "FLAT")
	mk("notes/dup/spec.md", "DIR")

	files := collectKnowledgeFiles(root)
	if len(files) != 1 {
		t.Fatalf("expected 1 entry after dedup, got %d: %+v", len(files), files)
	}
	if files[0].Slug != "dup" || files[0].Kind != "notes" {
		t.Errorf("unexpected entry: %+v", files[0])
	}
	if filepath.Base(files[0].Path) != "dup.md" {
		t.Errorf("flat shape should win; got path %s", files[0].Path)
	}
}

func TestLoadEntry_FlatStillResolves(t *testing.T) {
	heroDir := fabricateKnowledgeTree(t)
	e := LoadEntry(heroDir, "flat-note")
	if e == nil {
		t.Fatalf("LoadEntry returned nil for flat slug")
	}
	if e.Slug != "flat-note" || e.Domain != "notes" {
		t.Errorf("unexpected entry: %+v", e)
	}
}

func TestLoadEntry_DirStyleResolves(t *testing.T) {
	heroDir := fabricateKnowledgeTree(t)
	e := LoadEntry(heroDir, "dev-workflow")
	if e == nil {
		t.Fatalf("LoadEntry returned nil for dir-style slug 'dev-workflow'")
	}
	if e.Slug != "dev-workflow" {
		t.Errorf("slug = %q, want dev-workflow", e.Slug)
	}
	if e.Domain != "context" {
		t.Errorf("Domain = %q, want context", e.Domain)
	}
	if e.Title != "Dev workflow" {
		t.Errorf("Title = %q, want %q", e.Title, "Dev workflow")
	}
}

func TestLoadEntry_NestedResolves(t *testing.T) {
	heroDir := fabricateKnowledgeTree(t)
	e := LoadEntry(heroDir, "auto")
	if e == nil {
		t.Fatalf("LoadEntry returned nil for nested slug 'auto'")
	}
	if e.Slug != "auto" || e.Domain != "rules" {
		t.Errorf("unexpected entry: slug=%q domain=%q", e.Slug, e.Domain)
	}
}

func TestLoadEntry_MissingSlugReturnsNil(t *testing.T) {
	heroDir := fabricateKnowledgeTree(t)
	if e := LoadEntry(heroDir, "no-such-slug"); e != nil {
		t.Errorf("LoadEntry(missing) = %+v, want nil", e)
	}
}

func TestLoadCorpus_CountsAllThreeShapes(t *testing.T) {
	heroDir := fabricateKnowledgeTree(t)
	c := LoadCorpus(CorpusInputs{HeroDir: heroDir})

	// 6 entry-bearing files in the fabricated tree (see fabricateKnowledgeTree).
	if c.TotalEntries != 6 {
		t.Errorf("TotalEntries = %d, want 6 (%+v)", c.TotalEntries, slugList(c.Entries))
	}

	// Every entry the loader can resolve should appear in the corpus.
	for _, slug := range []string{
		"flat-note", "dir-note", "dev-workflow", "auto", "project-rules", "loose-decision",
	} {
		found := false
		for _, e := range c.Entries {
			if e.Slug == slug {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("corpus missing slug %q (got %v)", slug, slugList(c.Entries))
		}
	}
}

func TestLoadCorpus_CountMatchesLoadEntry(t *testing.T) {
	heroDir := fabricateKnowledgeTree(t)
	c := LoadCorpus(CorpusInputs{HeroDir: heroDir})

	for _, entry := range c.Entries {
		if got := LoadEntry(heroDir, entry.Slug); got == nil {
			t.Errorf("corpus listed slug %q but LoadEntry returned nil", entry.Slug)
		}
	}
}

func slugList(entries []CorpusEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Slug)
	}
	sort.Strings(out)
	return out
}
