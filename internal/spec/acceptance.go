package spec

import (
	"regexp"
	"strings"
)

// AcceptanceCriterion is one parsed `AC-N: <statement>` entry from a
// spec's `## Acceptance criteria` section.
//
// Unlike the EARS-flavored Criterion type, this is the form the
// `acceptance-criteria-graph` feature ingests: a stable ID we can key
// graph nodes by, plus a statement that's the verifiable promise.
type AcceptanceCriterion struct {
	ID        string // e.g. "AC-1"
	Statement string // text following the ID on the same line, trimmed
}

// acIDPattern matches the leading `AC-N` token at the start of a line,
// with optional surrounding markdown (`**…**`) and bullet prefixes
// (`- `, `* `). The captured group is the numeric N. The trailing
// punctuation (`:` or end-of-bold) is consumed but not captured.
//
// Examples that match:
//
//	**AC-1:** statement here
//	AC-2: bare line
//	- **AC-3:** bullet bold
//	- AC-4: bullet plain
//	* **AC-5** end-bold without colon
var acIDPattern = regexp.MustCompile(`^\s*(?:[-*]\s+)?\*{0,2}AC-(\d+)\*{0,2}\s*:?\s*\**\s*`)

// ParseAcceptanceCriteria scans every section of s whose lowercased
// heading starts with "acceptance criteria" (covers `Acceptance
// criteria`, `Acceptance Criteria`, `Acceptance criteria (build-out-
// as-we-go set)`, etc.) and extracts `AC-N: <text>` entries.
//
// Both the bullet form (`- **AC-1:** …`) and the paragraph form
// (`**AC-1:** …\n…\n…`) are recognized. For paragraph entries, lines
// continuing the statement (until the next `AC-N:` or a blank line
// followed by a non-AC paragraph) are joined with a single space.
//
// Returns nil if the spec has no acceptance-criteria section or the
// section is empty.
func (s *Spec) ParseAcceptanceCriteria() []AcceptanceCriterion {
	if s == nil {
		return nil
	}
	var sections []string
	for name, body := range s.Sections {
		lower := strings.ToLower(name)
		if lower == "acceptance criteria" || strings.HasPrefix(lower, "acceptance criteria") {
			sections = append(sections, body)
		}
	}
	if len(sections) == 0 {
		return nil
	}

	var out []AcceptanceCriterion
	seen := map[string]bool{}
	for _, body := range sections {
		for _, ac := range parseACBlock(body) {
			if seen[ac.ID] {
				continue
			}
			seen[ac.ID] = true
			out = append(out, ac)
		}
	}
	return out
}

// parseACBlock walks a section body and emits one AcceptanceCriterion
// per `AC-N` entry. Paragraph-style entries collect continuation lines
// until the next AC-N line or a blank gap.
func parseACBlock(body string) []AcceptanceCriterion {
	lines := strings.Split(body, "\n")
	var out []AcceptanceCriterion
	var current *AcceptanceCriterion
	var blankRun int

	flush := func() {
		if current == nil {
			return
		}
		current.Statement = normalizeStatement(current.Statement)
		if current.Statement != "" || current.ID != "" {
			out = append(out, *current)
		}
		current = nil
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)

		if m := acIDPattern.FindStringSubmatchIndex(line); m != nil {
			flush()
			n := line[m[2]:m[3]]
			rest := strings.TrimSpace(line[m[1]:])
			current = &AcceptanceCriterion{
				ID:        "AC-" + n,
				Statement: rest,
			}
			blankRun = 0
			continue
		}

		if current == nil {
			continue
		}

		if trimmed == "" {
			// One blank line still allows the paragraph to continue
			// (markdown soft-paragraph break). Two blank lines in a
			// row close the paragraph.
			blankRun++
			if blankRun >= 2 {
				flush()
			}
			continue
		}
		blankRun = 0
		if current.Statement != "" {
			current.Statement += " "
		}
		current.Statement += trimmed
	}
	flush()
	return out
}

// normalizeStatement collapses runs of whitespace, trims leading
// markdown decorators that survived the regex, and strips a trailing
// period if the statement ends in one (matches how Spec authors write
// either form interchangeably).
func normalizeStatement(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "*")
	s = strings.TrimSpace(s)
	// Collapse multiple spaces to one.
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}
