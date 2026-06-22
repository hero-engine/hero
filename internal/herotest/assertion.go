package herotest

import "strings"

// AssertionHint represents a classified criterion mapped to a generic assertion intent.
// Framework adapters use the hint to emit framework-specific assertion code.
type AssertionHint struct {
	Kind       string // "visible", "text_contains", "url", "error", "count", "input", "click", "title", "output", "default"
	Criterion  string // original criterion text
	Selector   string // extracted CSS selector or element reference, if any
	QuotedText string // extracted quoted text, if any
}

// ClassifyAssertion maps a criterion string to an AssertionHint using keyword heuristics.
// This is the shared version of the keyword detection previously inlined in Playwright's
// mapCriterionToAssertion. Each framework adapter translates the returned hint into
// framework-specific assertion code.
func ClassifyAssertion(criterion string) AssertionHint {
	lower := strings.ToLower(criterion)
	hint := AssertionHint{Criterion: criterion}

	switch {
	case containsAny(lower, "url", "navigate to", "redirect", "route"):
		hint.Kind = "url"
	case containsAny(lower, "title"):
		hint.Kind = "title"
	case containsAny(lower, "visible", "display", "show", "appear", "render"):
		hint.Kind = "visible"
		hint.Selector = extractSelector(criterion)
	case containsAny(lower, "text", "contain", "message", "label"):
		hint.Kind = "text_contains"
		hint.QuotedText = extractQuotedText(criterion)
	case containsAny(lower, "count", "number of", "items", "results"):
		hint.Kind = "count"
	case containsAny(lower, "input", "value", "field", "form"):
		hint.Kind = "input"
	case containsAny(lower, "click", "button", "press", "submit"):
		hint.Kind = "click"
	case containsAny(lower, "error", "fail", "invalid", "reject"):
		hint.Kind = "error"
	case containsAny(lower, "json", "output", "return"):
		hint.Kind = "output"
	default:
		hint.Kind = "default"
	}

	return hint
}

// containsAny checks if s contains any of the given substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// extractSelector attempts to extract a CSS selector hint from criterion text.
// Looks for backtick-delimited strings that look like selectors.
func extractSelector(criterion string) string {
	start := strings.Index(criterion, "`")
	if start < 0 {
		return ""
	}
	end := strings.Index(criterion[start+1:], "`")
	if end < 0 {
		return ""
	}
	candidate := criterion[start+1 : start+1+end]
	// Basic heuristic: if it looks like a CSS selector or HTML element
	if strings.HasPrefix(candidate, ".") || strings.HasPrefix(candidate, "#") ||
		strings.HasPrefix(candidate, "[") || strings.Contains(candidate, "-") {
		return candidate
	}
	return ""
}

// extractQuotedText extracts text within single or double quotes.
func extractQuotedText(criterion string) string {
	for _, q := range []string{"\"", "'"} {
		start := strings.Index(criterion, q)
		if start < 0 {
			continue
		}
		end := strings.Index(criterion[start+1:], q)
		if end < 0 {
			continue
		}
		return criterion[start+1 : start+1+end]
	}
	return ""
}
