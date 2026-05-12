package triage

import (
	"strings"
	"unicode"

	"github.com/hero-engine/hero/internal/spec"
)

// ConventionConflict describes a contradiction between two convention/rule specs.
type ConventionConflict struct {
	TargetSlug string
	Subject    string // what the conflict is about
	OurRule    string // our imperative
	TheirRule  string // conflicting imperative
}

// imperativeKeywords are the trigger words that mark an imperative sentence.
var imperativeKeywords = []string{
	"use", "never", "always", "don't", "do not",
	"must", "should", "avoid", "prefer", "require",
}

// ExtractImperatives returns sentences from content that contain an imperative keyword.
// Sentences longer than 200 characters are discarded.
func ExtractImperatives(content string) []string {
	// Split into sentences on '.', '!', '\n' boundaries.
	sentences := splitSentences(content)

	var result []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" || len(s) > 200 {
			continue
		}
		lower := strings.ToLower(s)
		for _, kw := range imperativeKeywords {
			if containsWord(lower, kw) {
				result = append(result, s)
				break
			}
		}
	}
	return result
}

// splitSentences breaks content into sentences on '.', '!', '?', and newlines.
func splitSentences(content string) []string {
	var sentences []string
	var sb strings.Builder

	runes := []rune(content)
	for i, r := range runes {
		if r == '\n' || r == '!' || r == '?' {
			s := strings.TrimSpace(sb.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			sb.Reset()
		} else if r == '.' {
			// Only split on period if followed by space or end
			if i+1 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '\n') {
				s := strings.TrimSpace(sb.String())
				if s != "" {
					sentences = append(sentences, s)
				}
				sb.Reset()
			} else {
				sb.WriteRune(r)
			}
		} else {
			sb.WriteRune(r)
		}
	}
	if s := strings.TrimSpace(sb.String()); s != "" {
		sentences = append(sentences, s)
	}
	return sentences
}

// containsWord checks whether s contains word w as a whole word (not a substring).
func containsWord(s, w string) bool {
	idx := strings.Index(s, w)
	if idx < 0 {
		return false
	}
	// Check left boundary
	if idx > 0 {
		prev := rune(s[idx-1])
		if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
			return false
		}
	}
	// Check right boundary
	end := idx + len(w)
	if end < len(s) {
		next := rune(s[end])
		if unicode.IsLetter(next) || unicode.IsDigit(next) {
			return false
		}
	}
	return true
}

// extractSubject returns the first significant word after an imperative keyword
// in the sentence, used as the "subject" for conflict matching.
func extractSubject(sentence string) string {
	lower := strings.ToLower(sentence)

	// Try multi-word keywords first (longest match)
	multiWord := []string{"do not", "don't"}
	for _, kw := range multiWord {
		if idx := strings.Index(lower, kw); idx >= 0 {
			rest := strings.TrimSpace(sentence[idx+len(kw):])
			if w := firstWord(rest); w != "" {
				return strings.ToLower(w)
			}
		}
	}

	singleWord := []string{"never", "always", "avoid", "prefer", "require", "must", "should", "use"}
	for _, kw := range singleWord {
		if idx := strings.Index(lower, kw); idx >= 0 {
			rest := strings.TrimSpace(sentence[idx+len(kw):])
			if w := firstWord(rest); w != "" {
				return strings.ToLower(w)
			}
		}
	}

	return ""
}

// firstWord returns the first alphabetic token from s.
func firstWord(s string) string {
	fields := strings.Fields(s)
	for _, f := range fields {
		// Strip punctuation
		clean := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, f)
		if clean != "" {
			return clean
		}
	}
	return ""
}

// isNegativeImperative returns true when the sentence expresses a prohibition
// (never/avoid/don't/do not) for the subject.
func isNegativeImperative(sentence string) bool {
	lower := strings.ToLower(sentence)
	for _, neg := range []string{"never", "avoid", "don't", "do not"} {
		if containsWord(lower, neg) {
			return true
		}
	}
	return false
}

// FindConflicts checks a convention/rule spec against existing ones for prescription conflicts.
// Only runs for type == convention or type == rule.
func FindConflicts(candidate *spec.Spec, corpus []*spec.Spec) []ConventionConflict {
	if candidate.Type != spec.TypeConvention && candidate.Type != spec.TypeRule {
		return nil
	}

	// Gather all content sections of candidate.
	ourContent := allContent(candidate)
	ourImperatives := ExtractImperatives(ourContent)

	var conflicts []ConventionConflict

	for _, s := range corpus {
		if s.Slug == candidate.Slug {
			continue
		}
		if s.Type != spec.TypeConvention && s.Type != spec.TypeRule {
			continue
		}

		theirContent := allContent(s)
		theirImperatives := ExtractImperatives(theirContent)

		for _, ours := range ourImperatives {
			ourSubject := extractSubject(ours)
			if ourSubject == "" {
				continue
			}
			ourNeg := isNegativeImperative(ours)

			for _, theirs := range theirImperatives {
				theirSubject := extractSubject(theirs)
				if theirSubject == "" || theirSubject != ourSubject {
					continue
				}
				theirNeg := isNegativeImperative(theirs)

				// Conflict: one says "use X", the other says "never/avoid X"
				if ourNeg != theirNeg {
					conflicts = append(conflicts, ConventionConflict{
						TargetSlug: s.Slug,
						Subject:    ourSubject,
						OurRule:    ours,
						TheirRule:  theirs,
					})
					// One conflict per (candidate-imperative, corpus-spec) pair is enough
					goto nextSpec
				}
			}
		}
	nextSpec:
	}

	return conflicts
}

// allContent concatenates all section content of a spec for imperative extraction.
func allContent(s *spec.Spec) string {
	var parts []string
	for _, v := range s.Sections {
		parts = append(parts, v)
	}
	return strings.Join(parts, "\n")
}
