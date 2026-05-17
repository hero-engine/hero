package cli

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

// TestDisplayType_NilVocabPreservesLiteral verifies that the
// engineering / legacy fall-through path (no vocab/methodology set in
// hero.json) renders the canonical type literal unchanged. This is the
// load-bearing invariant for B6: existing engineering output must be
// byte-identical to pre-B6 behavior.
func TestDisplayType_NilVocabPreservesLiteral(t *testing.T) {
	cases := []string{"feature", "bug", "initiative", "convention", "decision", "rule", "note", "context"}
	for _, c := range cases {
		if got := displayType(nil, c); got != c {
			t.Errorf("displayType(nil, %q) = %q, want %q (engineering output must be unchanged)", c, got, c)
		}
	}
}

// TestActiveVocab_NoConfigReturnsNil verifies that activeVocab returns
// nil when neither vocabulary nor methodology is set in hero.json.
// Returning nil is what preserves engineering-identical rendering.
func TestActiveVocab_NoConfigReturnsNil(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	cfg := &config.Config{}
	if v := activeVocab(cfg); v != nil {
		t.Errorf("activeVocab({}) = %v, want nil", v)
	}
}

// TestActiveVocab_ExplicitWins covers precedence rule 1: explicit
// cfg.Vocabulary takes priority over any inference.
func TestActiveVocab_ExplicitWins(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	cfg := &config.Config{Vocabulary: "agile-scrum"}
	v := activeVocab(cfg)
	if v == nil {
		t.Fatalf("activeVocab(agile-scrum) = nil, want non-nil")
	}
	if v.Name != "agile-scrum" {
		t.Errorf("v.Name = %q, want %q", v.Name, "agile-scrum")
	}
}

// TestActiveVocab_MethodologyDerivesVocab covers precedence rule 2:
// when methodology is set without explicit vocabulary, the vocabulary
// is auto-derived from the methodology's aligned_vocabulary field
// (scrum → agile-scrum).
func TestActiveVocab_MethodologyDerivesVocab(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	cfg := &config.Config{Methodology: "scrum"}
	v := activeVocab(cfg)
	if v == nil {
		t.Fatalf("activeVocab(methodology=scrum) = nil, want non-nil")
	}
	if v.Name != "agile-scrum" {
		t.Errorf("v.Name = %q, want %q (methodology should derive vocab via aligned_vocabulary)",
			v.Name, "agile-scrum")
	}
}

// TestDisplayType_AgileScrumStory verifies that a workspace running
// agile-scrum renders engineering's "feature" type as "Story" — the
// canonical user-facing example from the sprint spec acceptance
// criteria.
func TestDisplayType_AgileScrumStory(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	v := activeVocab(&config.Config{Vocabulary: "agile-scrum"})
	if v == nil {
		t.Fatalf("activeVocab(agile-scrum) returned nil")
	}
	if got := displayType(v, "feature"); got != "Story" {
		t.Errorf("displayType(agile-scrum, feature) = %q, want %q", got, "Story")
	}
	if got := displayType(v, "bug"); got != "Bug" {
		t.Errorf("displayType(agile-scrum, bug) = %q, want %q", got, "Bug")
	}
}

// TestDisplayType_ShapeUpScope verifies shape-up vocabulary renders
// feature as the Shape Up term ("Scope" or whatever the YAML
// declares).
func TestDisplayType_ShapeUpScope(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	v := activeVocab(&config.Config{Vocabulary: "shape-up"})
	if v == nil {
		t.Fatalf("activeVocab(shape-up) returned nil")
	}
	got := displayType(v, "feature")
	if got == "feature" || got == "Feature" {
		t.Errorf("displayType(shape-up, feature) = %q, expected shape-up-specific term (not the canonical literal)", got)
	}
}

// TestDialectLine covers the rendered header line used by
// `hero status` and `hero dashboard` to surface the active layers.
func TestDialectLine(t *testing.T) {
	resetVocabCacheForTesting()
	t.Cleanup(resetVocabCacheForTesting)

	if got := dialectLine(nil); got != "" {
		t.Errorf("dialectLine(nil) = %q, want empty", got)
	}
	if got := dialectLine(&config.Config{}); got != "" {
		t.Errorf("dialectLine({}) = %q, want empty (engineering workspace must be quiet)", got)
	}
	gotVocab := dialectLine(&config.Config{Vocabulary: "agile-scrum"})
	if !strings.Contains(gotVocab, "agile-scrum") {
		t.Errorf("dialectLine(vocab=agile-scrum) = %q, want it to contain %q", gotVocab, "agile-scrum")
	}
	gotMeth := dialectLine(&config.Config{Methodology: "scrum"})
	if !strings.Contains(gotMeth, "scrum") {
		t.Errorf("dialectLine(methodology=scrum) = %q, want it to contain %q", gotMeth, "scrum")
	}
	gotBoth := dialectLine(&config.Config{Vocabulary: "agile-scrum", Methodology: "scrum"})
	if !strings.Contains(gotBoth, "agile-scrum") || !strings.Contains(gotBoth, "scrum") {
		t.Errorf("dialectLine(both) = %q, want both names", gotBoth)
	}
}

// TestCanonicalize verifies the flat-type-literal → (type, kind)
// mapping used to bridge engineering's flat types to vocabulary YAML's
// (type, kind) pairs.
func TestCanonicalize(t *testing.T) {
	cases := []struct {
		in       string
		wantType string
		wantKind string
	}{
		{"feature", "spec", "feature"},
		{"bug", "spec", "bug"},
		{"chore", "spec", "chore"},
		{"initiative", "roadmap-item", ""},
		{"convention", "convention", ""},
		{"decision", "decision", ""},
		{"note", "note", ""},
		{"FEATURE", "spec", "feature"},
		{" feature ", "spec", "feature"},
	}
	for _, c := range cases {
		gotT, gotK := canonicalize(c.in)
		if gotT != c.wantType || gotK != c.wantKind {
			t.Errorf("canonicalize(%q) = (%q, %q), want (%q, %q)",
				c.in, gotT, gotK, c.wantType, c.wantKind)
		}
	}
}
