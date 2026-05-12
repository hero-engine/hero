package serve

import (
	"fmt"
	"strings"
)

// envelope contains the structured header attached to two-tier MCP
// responses. The header is text, not JSON, because Hero MCP tools
// return strings via ToolContent. The block is parseable by simple
// line scanning and does not collide with markdown that follows.
//
// Phase 1 contract (additive, opt-in):
//   - When a tool is called WITHOUT `compact: true`, behavior is
//     unchanged. The full body is returned exactly as before.
//   - When called WITH `compact: true`, the tool returns the envelope
//     text alone — no full body. The summary plus the ref_id is
//     enough for the model to decide whether to drill in via
//     hero_expand.
//
// Tools that have not been migrated to two-tier responses simply
// ignore the `compact` parameter and return their normal output.
type envelope struct {
	RefID     string
	ExpandVia string
	Kind      string
	Slug      string
	Scope     string
	Summary   string
}

// renderEnvelopeText builds the text representation. Format:
//
//	[hero envelope]
//	ref_id: <id>
//	expand_via: hero_expand
//	kind: <kind>
//	slug: <slug>
//	scope: <scope>
//	summary: <single-line summary; multi-line summaries are joined>
//	[/hero envelope]
func renderEnvelopeText(e envelope) string {
	summary := strings.ReplaceAll(strings.TrimSpace(e.Summary), "\n", " ")
	expand := e.ExpandVia
	if expand == "" {
		expand = "hero_expand"
	}
	var b strings.Builder
	b.WriteString("[hero envelope]\n")
	fmt.Fprintf(&b, "ref_id: %s\n", e.RefID)
	fmt.Fprintf(&b, "expand_via: %s\n", expand)
	if e.Kind != "" {
		fmt.Fprintf(&b, "kind: %s\n", e.Kind)
	}
	if e.Slug != "" {
		fmt.Fprintf(&b, "slug: %s\n", e.Slug)
	}
	if e.Scope != "" {
		fmt.Fprintf(&b, "scope: %s\n", e.Scope)
	}
	fmt.Fprintf(&b, "summary: %s\n", summary)
	b.WriteString("[/hero envelope]")
	return b.String()
}

// parseEnvelopeText extracts an envelope back from a text result. Used
// only by tests (round-trip verification). Returns ok=false when no
// envelope header is present.
func parseEnvelopeText(s string) (envelope, bool) {
	const open = "[hero envelope]"
	const close = "[/hero envelope]"
	i := strings.Index(s, open)
	if i < 0 {
		return envelope{}, false
	}
	j := strings.Index(s[i:], close)
	if j < 0 {
		return envelope{}, false
	}
	block := s[i+len(open) : i+j]
	out := envelope{}
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "ref_id":
			out.RefID = v
		case "expand_via":
			out.ExpandVia = v
		case "kind":
			out.Kind = v
		case "slug":
			out.Slug = v
		case "scope":
			out.Scope = v
		case "summary":
			out.Summary = v
		}
	}
	return out, true
}

// argCompact reads the optional boolean `compact` argument. Accepts
// bool, string ("true"/"false"), or absence. Defaults to false (Phase
// 1 backwards-compat default).
func argCompact(args map[string]interface{}) bool {
	if v, ok := args["compact"]; ok {
		switch x := v.(type) {
		case bool:
			return x
		case string:
			return strings.EqualFold(x, "true")
		}
	}
	return false
}
