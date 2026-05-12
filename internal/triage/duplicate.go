package triage

import (
	"strings"
	"unicode"

	"github.com/hero-engine/hero/internal/spec"
)

// DuplicateCandidate holds a potential duplicate slug and its similarity score.
type DuplicateCandidate struct {
	Slug       string
	Similarity float64
}

// stopwords are common English words stripped during title normalisation.
var stopwords = map[string]bool{
	"a": true, "an": true, "the": true,
	"and": true, "or": true, "but": true, "nor": true,
	"in": true, "on": true, "at": true, "to": true, "for": true,
	"of": true, "with": true, "by": true, "from": true, "up": true,
	"is": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "being": true, "have": true, "has": true, "had": true,
	"do": true, "does": true, "did": true, "will": true, "would": true,
	"shall": true, "should": true, "may": true, "might": true, "can": true,
	"could": true, "it": true, "its": true, "this": true, "that": true,
	"as": true, "if": true, "into": true, "not": true, "no": true,
}

// NormalizeTitle lowercases, removes stopwords, and strips punctuation.
func NormalizeTitle(title string) string {
	// Lowercase
	title = strings.ToLower(title)

	// Remove punctuation, keeping spaces and alphanumeric
	var sb strings.Builder
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune(' ')
		}
	}
	title = sb.String()

	// Filter stopwords
	words := strings.Fields(title)
	var kept []string
	for _, w := range words {
		if !stopwords[w] {
			kept = append(kept, w)
		}
	}

	return strings.Join(kept, " ")
}

// LevenshteinSimilarity returns a 0.0–1.0 similarity score between a and b.
// Similarity = 1 - (levenshteinDistance / max(len(a), len(b))).
// Strings are capped at 50 runes each before comparison for performance.
func LevenshteinSimilarity(a, b string) float64 {
	ra := []rune(a)
	rb := []rune(b)

	// Cap length for performance
	if len(ra) > 50 {
		ra = ra[:50]
	}
	if len(rb) > 50 {
		rb = rb[:50]
	}

	la, lb := len(ra), len(rb)

	if la == 0 && lb == 0 {
		return 1.0
	}
	if la == 0 || lb == 0 {
		return 0.0
	}

	// O(n*m) DP using two rows
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			if del < ins {
				curr[j] = del
			} else {
				curr[j] = ins
			}
			if sub < curr[j] {
				curr[j] = sub
			}
		}
		prev, curr = curr, prev
	}

	dist := prev[lb]
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}

	return 1.0 - float64(dist)/float64(maxLen)
}

// TagOverlap counts how many tags are shared between two specs.
func TagOverlap(a, b []string) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := make(map[string]bool, len(a))
	for _, t := range a {
		set[strings.ToLower(t)] = true
	}
	count := 0
	for _, t := range b {
		if set[strings.ToLower(t)] {
			count++
		}
	}
	return count
}

// FindDuplicates checks a spec against all existing specs for duplicate candidates.
// threshold is the normalised-title similarity threshold (default 0.80).
// tagOverlapMin is the minimum shared-tag count to flag (default 3); type must also match.
// Self (same slug) is skipped.
func FindDuplicates(candidate *spec.Spec, corpus []*spec.Spec, threshold float64, tagOverlapMin int) []DuplicateCandidate {
	if threshold <= 0 {
		threshold = 0.80
	}
	if tagOverlapMin <= 0 {
		tagOverlapMin = 3
	}

	normCandidate := NormalizeTitle(candidate.Title)

	var results []DuplicateCandidate

	for _, s := range corpus {
		if s.Slug == candidate.Slug {
			continue
		}

		// Title similarity
		normOther := NormalizeTitle(s.Title)
		sim := LevenshteinSimilarity(normCandidate, normOther)
		if sim >= threshold {
			results = append(results, DuplicateCandidate{Slug: s.Slug, Similarity: sim})
			continue
		}

		// Tag overlap (same type required)
		if s.Type == candidate.Type && TagOverlap(candidate.Tags, s.Tags) >= tagOverlapMin {
			// Use title sim as score even if below threshold
			results = append(results, DuplicateCandidate{Slug: s.Slug, Similarity: sim})
		}
	}

	return results
}
