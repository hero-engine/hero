package search

import (
	"sort"
	"strings"
	"unicode"
)

// Passage represents a scored passage extracted from a spec file.
type Passage struct {
	Slug    string
	Path    string
	Excerpt string
	Score   float64
}

var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true,
	"has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true,
	"might": true, "shall": true, "can": true, "what": true, "which": true,
	"who": true, "whom": true, "this": true, "that": true, "these": true,
	"those": true, "i": true, "we": true, "you": true, "he": true, "she": true,
	"it": true, "they": true, "to": true, "of": true, "in": true, "for": true,
	"on": true, "with": true, "at": true, "by": true, "from": true, "and": true,
	"or": true, "but": true, "if": true, "as": true, "up": true, "about": true,
	"into": true, "through": true, "during": true, "before": true, "after": true,
	"above": true, "below": true, "between": true, "out": true, "not": true,
	"no": true, "so": true, "than": true, "then": true, "when": true,
	"where": true, "why": true, "how": true,
}

// Tokenize lowercases text, strips punctuation, splits on whitespace, and
// removes stopwords.
func Tokenize(text string) []string {
	// Lowercase and strip punctuation
	var b strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(' ')
		}
	}

	raw := strings.Fields(b.String())
	tokens := make([]string, 0, len(raw))
	for _, t := range raw {
		if !stopwords[t] && t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// ScorePassage computes a TF-IDF-like score: count unique query tokens that
// appear in the passage, normalized by the passage length (in tokens).
func ScorePassage(passage string, queryTokens []string) float64 {
	if len(queryTokens) == 0 {
		return 0
	}

	passageTokens := Tokenize(passage)
	if len(passageTokens) == 0 {
		return 0
	}

	// Build a set of passage tokens for O(1) lookup.
	passageSet := make(map[string]bool, len(passageTokens))
	for _, t := range passageTokens {
		passageSet[t] = true
	}

	// Count unique query tokens that appear in the passage.
	matched := 0
	seen := make(map[string]bool, len(queryTokens))
	for _, qt := range queryTokens {
		if seen[qt] {
			continue
		}
		seen[qt] = true
		if passageSet[qt] {
			matched++
		}
	}

	// Normalize by passage length — longer passages shouldn't dominate.
	return float64(matched) / float64(len(passageTokens))
}

// ExtractPassages splits content into sentences, scores each against
// queryTokens, and returns the top maxPassages sentences by score.
func ExtractPassages(content string, queryTokens []string, maxPassages int) []string {
	if content == "" || maxPassages <= 0 {
		return nil
	}

	// Split on sentence-ending punctuation followed by space or newline.
	// We replace all delimiters with a common separator first.
	normalized := content
	normalized = strings.ReplaceAll(normalized, ".\n", ". ")
	normalized = strings.ReplaceAll(normalized, "! ", ". ")
	normalized = strings.ReplaceAll(normalized, "? ", ". ")
	sentences := strings.Split(normalized, ". ")

	type scored struct {
		text  string
		score float64
	}

	candidates := make([]scored, 0, len(sentences))
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		score := ScorePassage(s, queryTokens)
		candidates = append(candidates, scored{text: s, score: score})
	}

	// Sort descending by score.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	limit := maxPassages
	if limit > len(candidates) {
		limit = len(candidates)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = candidates[i].text
	}
	return result
}

// Confidence returns a human-readable confidence label for a given score.
func Confidence(score float64) string {
	switch {
	case score > 0.6:
		return "high"
	case score > 0.3:
		return "medium"
	default:
		return "low"
	}
}
