package context

import "strings"

// ApproxTokens estimates the token count of a string using the heuristic:
// token_count ≈ word_count * 1.3
func ApproxTokens(text string) int {
	words := len(strings.Fields(text))
	return int(float64(words) * 1.3)
}

// defaultPriority is the order entries are kept when truncating.
var defaultPriority = []string{"tripwire", "rule", "convention", "decision", "past_work", "external"}

// TruncateByPriority returns the largest prefix of entries (by priority) that
// fits within maxTokens.  Entries are selected in priority order; within each
// type bucket the original relative order is preserved.
//
// If priority is nil or empty the default order is used:
//
//	rule → convention → decision → past_work → external
//
// Any type not listed in priority is treated as lower priority than all listed
// types and is appended (in original order) after the listed buckets.
func TruncateByPriority(entries []ContextEntry, maxTokens int, priority []string) []ContextEntry {
	if maxTokens <= 0 || len(entries) == 0 {
		return entries
	}

	order := priority
	if len(order) == 0 {
		order = defaultPriority
	}

	// Build an index: type → ordered list of entries.
	buckets := make(map[string][]ContextEntry)
	for _, e := range entries {
		buckets[e.Type] = append(buckets[e.Type], e)
	}

	// Collect types that are NOT in the explicit priority list.
	listed := make(map[string]bool, len(order))
	for _, t := range order {
		listed[t] = true
	}

	// Build the full ordered sequence: listed types first, then unlisted.
	var ordered []ContextEntry
	for _, t := range order {
		ordered = append(ordered, buckets[t]...)
	}
	// Append unlisted types in their original relative order.
	for _, e := range entries {
		if !listed[e.Type] {
			ordered = append(ordered, e)
		}
	}

	// Greedily add entries until the budget is exhausted.
	used := 0
	var result []ContextEntry
	for _, e := range ordered {
		cost := ApproxTokens(e.Title + " " + e.Body)
		if used+cost > maxTokens {
			break
		}
		result = append(result, e)
		used += cost
	}

	return result
}
