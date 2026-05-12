package cli

import "testing"

func TestExtractAskAndSuggestion_HappyPath(t *testing.T) {
	body := `---
session: claude-x
---

## Last user ask

> "lets just move into the logical next phase in the plan" — after
> finishing the previous run.

## Just finished

work happened.

## Proposed next ask

> *"Finish the traversal-queries phases — natural-language routing
> for hero why and hero blocked, plus the MCP registration so agents
> can call them mid-reasoning."*

## Blocked on

Nothing.
`
	ask, sug := extractAskAndSuggestion(body)
	if ask == "" || !contains(ask, "lets just move into the logical next phase") {
		t.Errorf("ask = %q", ask)
	}
	if sug == "" || !contains(sug, "Finish the traversal-queries phases") {
		t.Errorf("suggestion = %q", sug)
	}
}

func TestExtractAskAndSuggestion_PrefersSuggestedNextOverProposedAsk(t *testing.T) {
	body := `## Suggested next prompt

> let's tackle phase 4

## Proposed next ask

> something older and stale
`
	_, sug := extractAskAndSuggestion(body)
	if !contains(sug, "let's tackle phase 4") {
		t.Errorf("suggestion = %q, want phase 4 (suggested next prompt should win)", sug)
	}
}

func TestExtractAskAndSuggestion_EmptyOnNoSections(t *testing.T) {
	body := `## Some other section

just text, no quotes
`
	ask, sug := extractAskAndSuggestion(body)
	if ask != "" {
		t.Errorf("ask = %q, want empty", ask)
	}
	if sug != "" {
		t.Errorf("sug = %q, want empty", sug)
	}
}

func TestFirstQuoteOrText_PrefersBlockquote(t *testing.T) {
	got := firstQuoteOrText(`
> first quoted line
> second quoted line

next paragraph
`)
	if got != "first quoted line second quoted line" {
		t.Errorf("got %q", got)
	}
}

func TestFirstQuoteOrText_FallsBackToParagraph(t *testing.T) {
	got := firstQuoteOrText(`
just a plain sentence.

second paragraph.
`)
	if got != "just a plain sentence." {
		t.Errorf("got %q", got)
	}
}

func TestFirstQuoteOrText_SkipsItalicPlaceholder(t *testing.T) {
	got := firstQuoteOrText(`_(none recorded yet)_`)
	if got != "" {
		t.Errorf("italic placeholder treated as content: %q", got)
	}
}
