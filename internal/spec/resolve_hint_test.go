package spec

import (
	"strings"
	"testing"
)

// specWithSlug builds a minimal spec fixture with a slug and type.
func specWithSlug(slug string, t Type) *Spec {
	return &Spec{Slug: slug, Type: t, Sections: map[string]string{}}
}

// initiativeWith builds an initiative fixture whose children section is keyed
// exactly as parseSections would key it (lowercased heading, body trimmed).
func initiativeWith(slug, childrenKey, childrenBody string) *Spec {
	s := specWithSlug(slug, TypeInitiative)
	s.Sections[childrenKey] = strings.TrimSpace(childrenBody)
	return s
}

func TestResolveOrHint(t *testing.T) {
	const childTable = `| Slug | Title | Priority |
|---|---|---|
| configurable-reranking | Configurable reranking | P1 |
| query-expansion | Query expansion | P1 |`

	const childTableLinked = `| Spec | Notes |
|---|---|
| [project-charter](../../charter/spec.md) | the charter |`

	const childTableNoPipes = `Slug | Title
--- | ---
loose-child | A child without outer pipes`

	tests := []struct {
		name       string
		slug       string
		specs      []*Spec
		wantSlug   string // expected resolved spec slug; "" means nil
		wantHint   string // exact hint when deterministic; "" means assert empty
		hintSubstr []string
	}{
		{
			name:     "exact match",
			slug:     "retrieval-quality",
			specs:    []*Spec{specWithSlug("retrieval-quality", TypeInitiative)},
			wantSlug: "retrieval-quality",
			wantHint: "",
		},
		{
			name:     "case-only mismatch resolves silently",
			slug:     "Retrieval-Quality",
			specs:    []*Spec{specWithSlug("retrieval-quality", TypeInitiative)},
			wantSlug: "retrieval-quality",
			wantHint: "",
		},
		{
			name: "unmaterialized initiative child",
			slug: "configurable-reranking",
			specs: []*Spec{
				initiativeWith("retrieval-quality", "children", childTable),
			},
			wantSlug:   "",
			hintSubstr: []string{"retrieval-quality", "/design", "configurable-reranking"},
		},
		{
			name: "children heading variant with markdown-link row",
			slug: "project-charter",
			specs: []*Spec{
				initiativeWith("some-initiative", "children — six features", childTableLinked),
			},
			wantSlug:   "",
			hintSubstr: []string{"some-initiative", "/design"},
		},
		{
			name: "leading/trailing-pipe tolerance (no outer pipes)",
			slug: "loose-child",
			specs: []*Spec{
				initiativeWith("some-initiative", "children", childTableNoPipes),
			},
			wantSlug:   "",
			hintSubstr: []string{"some-initiative", "/design"},
		},
		{
			name:     "fuzzy near-miss single suggestion",
			slug:     "configurable-rerankin",
			specs:    []*Spec{specWithSlug("configurable-reranking", TypeFeature)},
			wantSlug: "",
			wantHint: "no spec `configurable-rerankin` found — did you mean `configurable-reranking`?",
		},
		{
			name: "multiple suggestions capped at 3",
			slug: "report",
			specs: []*Spec{
				specWithSlug("reports", TypeFeature), // dist 1
				specWithSlug("export", TypeFeature),  // dist 2 — 4th by distance, dropped
				specWithSlug("resort", TypeFeature),  // dist 1
				specWithSlug("deport", TypeFeature),  // dist 1
			},
			wantSlug: "",
			// Exact hint asserts the cap (export, the only dist-2 candidate, is
			// dropped) AND the stable distance ordering. A substring check would
			// still pass if the cap broke and export leaked as a 4th suggestion.
			wantHint: "no spec `report` found — did you mean `reports`, `resort`, `deport`?",
		},
		{
			// Regression for flat-named-spec-discovery: once a flat child is
			// discovered, its slug is both an exact-match spec AND a first-column
			// entry in its initiative's children table. Step 1 (exact match) must
			// win so verify resolves the real spec instead of misfiring the
			// step-3 "hasn't been designed yet" hint. Guards the step ordering.
			name: "discovered child resolves despite being in children table",
			slug: "configurable-reranking",
			specs: []*Spec{
				initiativeWith("retrieval-quality", "children", childTable),
				specWithSlug("configurable-reranking", TypeFeature),
			},
			wantSlug: "configurable-reranking",
			wantHint: "",
		},
		{
			name: "initiative-child beats fuzzy on tie",
			slug: "configurable-reranking",
			specs: []*Spec{
				initiativeWith("retrieval-quality", "children", childTable),
				// a near-miss spec that would otherwise fuzzy-match
				specWithSlug("configurable-rerankinX", TypeFeature),
			},
			wantSlug:   "",
			hintSubstr: []string{"child of initiative `retrieval-quality`", "/design"},
		},
		{
			name:     "no signal falls through to empty hint",
			slug:     "totally-unrelated-xyz",
			specs:    []*Spec{specWithSlug("alpha", TypeFeature), specWithSlug("beta", TypeFeature)},
			wantSlug: "",
			wantHint: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, hint := ResolveOrHint(tc.slug, tc.specs)

			gotSlug := ""
			if got != nil {
				gotSlug = got.Slug
			}
			if gotSlug != tc.wantSlug {
				t.Errorf("resolved slug = %q, want %q (hint=%q)", gotSlug, tc.wantSlug, hint)
			}

			if len(tc.hintSubstr) > 0 {
				for _, sub := range tc.hintSubstr {
					if !strings.Contains(hint, sub) {
						t.Errorf("hint %q missing substring %q", hint, sub)
					}
				}
				return
			}

			if hint != tc.wantHint {
				t.Errorf("hint = %q, want %q", hint, tc.wantHint)
			}
		})
	}
}

func TestFuzzySuggestions_ShortSlugGuard(t *testing.T) {
	// A 2-char slug at distance 2 from every candidate should be suppressed:
	// the closest distance (2) is not strictly less than the slug length (2).
	specs := []*Spec{specWithSlug("foo", TypeFeature), specWithSlug("bar", TypeFeature)}
	if got := fuzzySuggestions("xy", specs); got != nil {
		t.Errorf("expected no suggestions for short over-matching slug, got %v", got)
	}

	// A 2-char slug at distance 1 (strictly < length) should still suggest.
	specs2 := []*Spec{specWithSlug("ab", TypeFeature)}
	if got := fuzzySuggestions("ax", specs2); len(got) != 1 || got[0] != "ab" {
		t.Errorf("expected [ab] for close short slug, got %v", got)
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"r-01", "r-01-foo", 4},
	}
	for _, c := range cases {
		if got := levenshtein(c.a, c.b); got != c.want {
			t.Errorf("levenshtein(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
