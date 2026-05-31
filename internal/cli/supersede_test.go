package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// TestDetectSupersedeCandidates_SlugSuffix exercises the high-confidence
// slug-suffix heuristic: foo-v1 / foo-v2 / foo all pair into the
// newest version. Covers the AC: "WHEN hero supersede --scan is run
// THE SYSTEM SHALL write a candidate report ... and never mutate any
// spec."
func TestDetectSupersedeCandidates_SlugSuffix(t *testing.T) {
	specs := []*spec.Spec{
		mkSpec("hero-surface-polish-v1"),
		mkSpec("hero-surface-polish-v2"),
		mkSpec("unrelated-thing"),
	}
	cands := detectSupersedeCandidates(specs)
	if len(cands) != 1 {
		t.Fatalf("got %d candidates, want 1 (v1→v2)", len(cands))
	}
	c := cands[0]
	if c.OldSlug != "hero-surface-polish-v1" || c.NewSlug != "hero-surface-polish-v2" {
		t.Errorf("got pair %s→%s, want hero-surface-polish-v1→hero-surface-polish-v2",
			c.OldSlug, c.NewSlug)
	}
	if c.Heuristic != "slug-suffix" {
		t.Errorf("heuristic = %q, want slug-suffix", c.Heuristic)
	}
	if c.Confidence != "high" {
		t.Errorf("confidence = %q, want high", c.Confidence)
	}
}

// TestDetectSupersedeCandidates_BodyMention covers the "X replaces Y"
// body-mention heuristic. Only mentions whose target slug exists in the
// corpus produce a pair.
func TestDetectSupersedeCandidates_BodyMention(t *testing.T) {
	old := mkSpec("legacy-mode")
	newer := mkSpec("legacy-mode-removal")
	newer.RawContent = "This spec supersedes `legacy-mode` by removing the old code path entirely."
	cands := detectSupersedeCandidates([]*spec.Spec{old, newer})
	var found bool
	for _, c := range cands {
		if c.OldSlug == "legacy-mode" && c.NewSlug == "legacy-mode-removal" {
			found = true
			if c.Heuristic != "body-mention" {
				t.Errorf("heuristic = %q, want body-mention", c.Heuristic)
			}
		}
	}
	if !found {
		t.Errorf("expected body-mention pair legacy-mode→legacy-mode-removal in %+v", cands)
	}
}

// TestDetectSupersedeCandidates_SkipsAlreadySuperseded ensures the
// scan doesn't re-propose a pair for a spec that already carries the
// superseded_by field — that work is done.
func TestDetectSupersedeCandidates_SkipsAlreadySuperseded(t *testing.T) {
	old := mkSpec("foo-v1")
	old.SupersededBy = "foo-v2"
	newer := mkSpec("foo-v2")
	cands := detectSupersedeCandidates([]*spec.Spec{old, newer})
	for _, c := range cands {
		if c.OldSlug == "foo-v1" {
			t.Errorf("scan re-proposed already-superseded spec: %+v", c)
		}
	}
}

// TestFindSupersedeCycle catches the cycle case: A→B→A. The supersede
// command refuses to write when this would happen.
//
// Covers AC: "IF hero supersede <old> --by <new> is run AND following
// the existing supersede chain from <new> reaches <old> THEN THE
// SYSTEM SHALL refuse the operation as a cycle."
func TestFindSupersedeCycle(t *testing.T) {
	a := mkSpec("a")
	b := mkSpec("b")
	b.SupersededBy = "a" // existing chain B → A

	bySlug := map[string]*spec.Spec{"a": a, "b": b}

	// Attempting to set A.superseded_by = B should detect the cycle:
	// walking from B, we reach A.
	hit := findSupersedeCycle(bySlug, "b", "a")
	if hit != "a" {
		t.Errorf("findSupersedeCycle = %q, want %q (cycle through a)", hit, "a")
	}

	// No cycle: walking from a non-superseded spec returns "".
	if hit := findSupersedeCycle(bySlug, "a", "b"); hit != "" {
		t.Errorf("findSupersedeCycle on plain chain = %q, want \"\"", hit)
	}
}

// TestAppendSupersedesRelation_Idempotent verifies the helper that
// updates the new spec's `supersedes:` relation when a pair is set up.
// Running it twice with the same old-slug must not duplicate the entry.
func TestAppendSupersedesRelation_Idempotent(t *testing.T) {
	dir := t.TempDir()
	specPath := dir + "/spec.md"
	initial := `---
title: New
type: feature
status: planning
---
## Goal
x
`
	if err := os.WriteFile(specPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := appendSupersedesRelation(specPath, "old-one"); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := appendSupersedesRelation(specPath, "old-one"); err != nil {
		t.Fatalf("idempotent append: %v", err)
	}

	// Re-parse and confirm the relation appears exactly once.
	parsed, err := spec.ParseFile(specPath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	count := 0
	for _, rel := range parsed.Relations {
		if rel.Kind == "supersedes" && rel.Target == "old-one" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("supersedes relation count = %d, want 1 (idempotent)", count)
	}
}

// mkSpec builds a minimal Spec value for test fixtures.
func mkSpec(slug string) *spec.Spec {
	return &spec.Spec{
		Slug:       slug,
		Title:      strings.Title(strings.ReplaceAll(slug, "-", " ")),
		Type:       spec.TypeFeature,
		Status:     spec.StatusCompleted,
		ModifiedAt: time.Now(),
		CreatedAt:  time.Now(),
	}
}
