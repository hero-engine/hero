package serve

import (
	"strings"
	"testing"
)

func TestRenderEnvelopeText_RoundTrip(t *testing.T) {
	e := envelope{
		RefID:     "spec:hero-ask:full",
		ExpandVia: "hero_expand",
		Kind:      "spec",
		Slug:      "hero-ask",
		Scope:     "full",
		Summary:   "Adds hero ask <query> for natural-language Q&A. Status: planning.",
	}
	rendered := renderEnvelopeText(e)
	if !strings.HasPrefix(rendered, "[hero envelope]\n") {
		t.Fatalf("envelope must open with marker, got: %q", rendered[:30])
	}
	if !strings.HasSuffix(rendered, "[/hero envelope]") {
		t.Fatalf("envelope must close with marker, got tail: %q", rendered[len(rendered)-30:])
	}

	parsed, ok := parseEnvelopeText(rendered)
	if !ok {
		t.Fatalf("parse failed")
	}
	if parsed != e {
		t.Fatalf("round-trip mismatch:\n got: %+v\nwant: %+v", parsed, e)
	}
}

func TestRenderEnvelopeText_DefaultExpandVia(t *testing.T) {
	out := renderEnvelopeText(envelope{RefID: "spec:x:full", Summary: "s"})
	if !strings.Contains(out, "expand_via: hero_expand") {
		t.Fatalf("expand_via default missing: %q", out)
	}
}

func TestRenderEnvelopeText_FlattensSummaryNewlines(t *testing.T) {
	out := renderEnvelopeText(envelope{
		RefID:   "spec:x:full",
		Summary: "line one\nline two\nline three",
	})
	parsed, _ := parseEnvelopeText(out)
	if strings.Contains(parsed.Summary, "\n") {
		t.Fatalf("summary should be single-line: %q", parsed.Summary)
	}
}

func TestParseEnvelopeText_NoEnvelope(t *testing.T) {
	if _, ok := parseEnvelopeText("just regular content"); ok {
		t.Fatalf("parse should fail when no envelope present")
	}
}

func TestArgCompact_Variants(t *testing.T) {
	cases := []struct {
		name string
		args map[string]interface{}
		want bool
	}{
		{"absent", nil, false},
		{"empty", map[string]interface{}{}, false},
		{"bool true", map[string]interface{}{"compact": true}, true},
		{"bool false", map[string]interface{}{"compact": false}, false},
		{"string true", map[string]interface{}{"compact": "true"}, true},
		{"string false", map[string]interface{}{"compact": "false"}, false},
		{"string TRUE caseless", map[string]interface{}{"compact": "TRUE"}, true},
		{"unrecognised", map[string]interface{}{"compact": 1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := argCompact(tc.args); got != tc.want {
				t.Fatalf("argCompact(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
