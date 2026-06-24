package synthesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSpec(t *testing.T, heroDir, sub, slug, body string) {
	t.Helper()
	dir := filepath.Join(heroDir, sub, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleResolvesAndStampsProvenance(t *testing.T) {
	hero := t.TempDir()
	writeSpec(t, hero, "planning/features", "feat-a", "---\ntitle: Feature A\ntype: feature\nslug: feat-a\ncreated: 2026-01-01\n---\n# Feature A\nbody a\n")
	writeSpec(t, hero, "planning/features", "feat-b", "---\ntitle: Feature B\ntype: feature\nslug: feat-b\ncreated: 2026-02-01\n---\n# Feature B\nbody b\n")

	pkt, err := Assemble(hero, hero, []string{"feat-a", "feat-b"})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(pkt.Specs) != 2 {
		t.Fatalf("Specs = %d, want 2", len(pkt.Specs))
	}
	// No initiative → dominant is the first input.
	if pkt.OutSlug != "feat-a" {
		t.Errorf("OutSlug = %q, want feat-a", pkt.OutSlug)
	}

	fm := pkt.Frontmatter("2026-06-23")
	for _, want := range []string{"type: explainer", "synthesized_from:", "- feat-a", "- feat-b", "last_synthesized: 2026-06-23"} {
		if !strings.Contains(fm, want) {
			t.Errorf("frontmatter missing %q:\n%s", want, fm)
		}
	}
	if strings.Contains(fm, "source_initiative:") {
		t.Error("source_initiative should not be set when no initiative is an input")
	}
}

func TestAssembleInitiativeIsDominant(t *testing.T) {
	hero := t.TempDir()
	writeSpec(t, hero, "planning/features", "feat-a", "---\ntitle: Feature A\ntype: feature\nslug: feat-a\n---\n# A\n")
	writeSpec(t, hero, "planning/initiatives", "big-thing", "---\ntitle: Big Thing\ntype: initiative\nslug: big-thing\n---\n# Big Thing\n")

	pkt, err := Assemble(hero, hero, []string{"feat-a", "big-thing"})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if pkt.OutSlug != "big-thing" {
		t.Errorf("OutSlug = %q, want big-thing (initiative dominates)", pkt.OutSlug)
	}
	if pkt.Title != "Big Thing" {
		t.Errorf("Title = %q, want Big Thing", pkt.Title)
	}
	if !strings.Contains(pkt.Frontmatter("2026-06-23"), "source_initiative: big-thing") {
		t.Error("source_initiative should be set to the initiative slug")
	}
}

func TestAssembleFailsLoudOnUnknownSlug(t *testing.T) {
	hero := t.TempDir()
	writeSpec(t, hero, "planning/features", "feat-a", "---\ntitle: A\ntype: feature\nslug: feat-a\n---\n# A\n")

	pkt, err := Assemble(hero, hero, []string{"feat-a", "does-not-exist"})
	if err == nil {
		t.Fatal("expected error for unknown slug, got nil")
	}
	if pkt != nil {
		t.Error("packet should be nil when a slug is unresolved (write nothing)")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("error should name the unresolved slug, got: %v", err)
	}
}
