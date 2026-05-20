package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKnowledge_HappyPath(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")

	mustWriteSpec := func(kind, slug, ftype string) {
		t.Helper()
		p := filepath.Join(heroDir, "knowledge", kind, slug)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "spec.md"),
			[]byte("---\ntitle: "+slug+"\ntype: "+ftype+"\nstatus: active\n---\n# "+slug+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWriteSpec("decisions", "d1", "decision")
	mustWriteSpec("conventions", "c1", "convention")
	mustWriteSpec("notes", "n1", "note")
	mustWriteSpec("notes", "n2", "note")

	out := LoadKnowledge(KnowledgeInputs{HeroDir: heroDir})
	if out.Decisions != 1 {
		t.Errorf("Decisions = %d, want 1", out.Decisions)
	}
	if out.Conventions != 1 {
		t.Errorf("Conventions = %d, want 1", out.Conventions)
	}
	if out.Notes != 2 {
		t.Errorf("Notes = %d, want 2", out.Notes)
	}
	if out.LastCapturedSlug == "" {
		t.Error("expected LastCapturedSlug non-empty")
	}
	if out.KnowledgeHref != "/knowledge" {
		t.Errorf("KnowledgeHref = %q, want /knowledge", out.KnowledgeHref)
	}
}

func TestLoadKnowledge_NoHeroDir(t *testing.T) {
	out := LoadKnowledge(KnowledgeInputs{})
	if out.Decisions+out.Conventions+out.Notes+out.Captures != 0 {
		t.Errorf("all counts should be 0, got d=%d c=%d n=%d cap=%d",
			out.Decisions, out.Conventions, out.Notes, out.Captures)
	}
	if out.KnowledgeHref != "/knowledge" {
		t.Errorf("KnowledgeHref = %q, want /knowledge", out.KnowledgeHref)
	}
}
