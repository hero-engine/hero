// Package score provides heuristic quality scoring for specs.
// It evaluates completeness, clarity, and deliverability without
// requiring an LLM — scoring is fast, deterministic, and local.
package score

import (
	"regexp"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// Dimension represents a single scoring dimension with its result.
type Dimension struct {
	Name    string  `json:"name"`
	Score   float64 `json:"score"`   // 0-100
	Weight  float64 `json:"weight"`  // how much this dimension counts
	Details string  `json:"details"` // human-readable explanation
}

// Warning represents a specific quality issue found in the spec.
type Warning struct {
	Severity string `json:"severity"` // "error", "warning", "info"
	Message  string `json:"message"`
}

// Result holds the complete scoring output for a spec.
type Result struct {
	Slug       string      `json:"slug"`
	Score      int         `json:"score"`       // 0-100 weighted total
	Grade      string      `json:"grade"`       // A/B/C/D/F
	Dimensions []Dimension `json:"dimensions"`
	Warnings   []Warning   `json:"warnings"`
	Suggestions []string   `json:"suggestions,omitempty"`
	Deliverable bool       `json:"deliverable"` // above minimum threshold
}

// Config controls scoring behavior.
type Config struct {
	MinScore int `json:"min_score"` // minimum score to allow delivery (default 40)
}

// DefaultConfig returns default scoring configuration.
func DefaultConfig() *Config {
	return &Config{MinScore: 40}
}

// Score evaluates a spec and returns a quality score.
func Score(s *spec.Spec, cfg *Config) *Result {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	content := s.RawContent
	body := bodyAfterFrontmatter(content)

	r := &Result{
		Slug: s.Slug,
	}

	// Run each scoring dimension
	r.Dimensions = []Dimension{
		scoreAcceptanceCriteria(body),
		scoreScopeClarity(body, s),
		scoreTechnicalSpecificity(body),
		scoreTestStrategy(body),
		scoreStructure(body, s),
		scoreAmbiguity(body),
	}

	// Compute weighted total
	totalWeight := 0.0
	weightedSum := 0.0
	for _, d := range r.Dimensions {
		totalWeight += d.Weight
		weightedSum += d.Score * d.Weight
	}
	if totalWeight > 0 {
		r.Score = int(weightedSum / totalWeight)
	}

	// Assign grade
	switch {
	case r.Score >= 90:
		r.Grade = "A"
	case r.Score >= 75:
		r.Grade = "B"
	case r.Score >= 60:
		r.Grade = "C"
	case r.Score >= 40:
		r.Grade = "D"
	default:
		r.Grade = "F"
	}

	r.Deliverable = r.Score >= cfg.MinScore

	// Generate warnings from dimension scores
	for _, d := range r.Dimensions {
		if d.Score == 0 {
			r.Warnings = append(r.Warnings, Warning{
				Severity: "error",
				Message:  d.Name + ": " + d.Details,
			})
		} else if d.Score < 50 {
			r.Warnings = append(r.Warnings, Warning{
				Severity: "warning",
				Message:  d.Name + ": " + d.Details,
			})
		}
	}

	// Generate suggestions
	r.Suggestions = generateSuggestions(r.Dimensions, body)

	return r
}

// bodyAfterFrontmatter strips YAML frontmatter and returns the body.
func bodyAfterFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return content
	}
	return parts[2]
}

// scoreAcceptanceCriteria checks for measurable, testable acceptance criteria.
// Weight: 25 (most important dimension)
func scoreAcceptanceCriteria(body string) Dimension {
	d := Dimension{Name: "Acceptance Criteria", Weight: 25}

	lower := strings.ToLower(body)

	// Look for acceptance criteria section
	hasSection := strings.Contains(lower, "## acceptance criteria") ||
		strings.Contains(lower, "## acceptance") ||
		strings.Contains(lower, "## success criteria") ||
		strings.Contains(lower, "## done when") ||
		strings.Contains(lower, "## definition of done")

	if !hasSection {
		// Check for inline criteria patterns
		bulletCriteria := countBulletCriteria(body)
		if bulletCriteria >= 2 {
			d.Score = 50
			d.Details = "found inline criteria but no dedicated section"
			return d
		}
		d.Score = 0
		d.Details = "no acceptance criteria section found"
		return d
	}

	// Extract the section content
	section := extractSection(body, "acceptance criteria", "success criteria", "done when", "definition of done")

	// Count criteria items (bullets or numbered items)
	items := countListItems(section)
	if items == 0 {
		d.Score = 20
		d.Details = "acceptance criteria section exists but has no items"
		return d
	}
	if items == 1 {
		d.Score = 40
		d.Details = "only 1 acceptance criterion — consider adding more"
		return d
	}

	// Check for measurability (numbers, comparisons, specific outcomes)
	measurable := countMeasurableItems(section)
	ratio := float64(measurable) / float64(items)

	if ratio >= 0.5 && items >= 3 {
		d.Score = 100
		d.Details = pluralize(items, "criterion", "criteria") + ", " + pluralize(measurable, "is measurable", "are measurable")
	} else if items >= 3 {
		d.Score = 75
		d.Details = pluralize(items, "criterion", "criteria") + " but few are measurable/testable"
	} else {
		d.Score = 60
		d.Details = pluralize(items, "criterion", "criteria")
	}

	return d
}

// scoreScopeClarity checks whether the spec has bounded scope.
// Weight: 20
func scoreScopeClarity(body string, s *spec.Spec) Dimension {
	d := Dimension{Name: "Scope Clarity", Weight: 20}

	lower := strings.ToLower(body)
	score := 0.0

	// Has a goal/problem statement?
	if strings.Contains(lower, "## goal") || strings.Contains(lower, "## problem") ||
		strings.Contains(lower, "## objective") || strings.Contains(lower, "## summary") {
		score += 30
	}

	// Has explicit non-goals or out-of-scope?
	if strings.Contains(lower, "non-goal") || strings.Contains(lower, "out of scope") ||
		strings.Contains(lower, "## scope") || strings.Contains(lower, "not in scope") {
		score += 25
	}

	// Has a design/approach section?
	if strings.Contains(lower, "## design") || strings.Contains(lower, "## approach") ||
		strings.Contains(lower, "## solution") || strings.Contains(lower, "## proposed") ||
		strings.Contains(lower, "## implementation") || strings.Contains(lower, "## changes") {
		score += 25
	}

	// Has files/changes listed?
	if strings.Contains(lower, "## changes") || strings.Contains(lower, "## files") ||
		len(s.FilesTouched) > 0 {
		score += 20
	}

	if score == 0 {
		d.Details = "no goal, scope, or design sections found"
	} else if score < 50 {
		d.Details = "partial scope definition — missing goal, non-goals, or design sections"
	} else if score < 75 {
		d.Details = "good scope definition"
	} else {
		d.Details = "well-bounded scope with goal, non-goals, and design"
	}

	d.Score = score
	return d
}

// scoreTechnicalSpecificity checks for concrete references to files, packages, APIs.
// Weight: 20
func scoreTechnicalSpecificity(body string) Dimension {
	d := Dimension{Name: "Technical Specificity", Weight: 20}

	// Count code references (backtick-wrapped terms)
	codeRefs := countCodeRefs(body)

	// Count file path references
	fileRefs := countFileRefs(body)

	// Count code blocks
	codeBlocks := strings.Count(body, "```")

	total := codeRefs + fileRefs*2 + codeBlocks*3

	switch {
	case total >= 15:
		d.Score = 100
		d.Details = "highly specific — references concrete code, files, and APIs"
	case total >= 8:
		d.Score = 75
		d.Details = "good specificity — references some concrete elements"
	case total >= 3:
		d.Score = 50
		d.Details = "moderate specificity — few concrete references"
	case total >= 1:
		d.Score = 25
		d.Details = "low specificity — mostly abstract description"
	default:
		d.Score = 0
		d.Details = "no technical references — purely abstract"
	}

	return d
}

// scoreTestStrategy checks for testing plan or verification approach.
// Weight: 15
func scoreTestStrategy(body string) Dimension {
	d := Dimension{Name: "Test Strategy", Weight: 15}

	lower := strings.ToLower(body)

	hasTestSection := strings.Contains(lower, "## test") || strings.Contains(lower, "## verification") ||
		strings.Contains(lower, "## validation") || strings.Contains(lower, "## how to verify")

	hasTestMentions := strings.Contains(lower, "unit test") || strings.Contains(lower, "integration test") ||
		strings.Contains(lower, "e2e test") || strings.Contains(lower, "test coverage") ||
		strings.Contains(lower, "test case") || strings.Contains(lower, "test file") ||
		strings.Contains(lower, "_test.go") || strings.Contains(lower, ".test.ts") ||
		strings.Contains(lower, ".spec.ts") || strings.Contains(lower, "pytest")

	hasVerifyMentions := strings.Contains(lower, "verify by") || strings.Contains(lower, "verified by") ||
		strings.Contains(lower, "validate by") || strings.Contains(lower, "confirm by") ||
		strings.Contains(lower, "run the") || strings.Contains(lower, "check that")

	if hasTestSection {
		d.Score = 100
		d.Details = "dedicated test/verification section"
	} else if hasTestMentions {
		d.Score = 60
		d.Details = "mentions testing but no dedicated section"
	} else if hasVerifyMentions {
		d.Score = 40
		d.Details = "mentions verification but no structured test plan"
	} else {
		d.Score = 0
		d.Details = "no testing or verification strategy"
	}

	return d
}

// scoreStructure checks overall document structure.
// Weight: 10
func scoreStructure(body string, s *spec.Spec) Dimension {
	d := Dimension{Name: "Structure", Weight: 10}

	// Count h2 sections
	h2Count := strings.Count(body, "\n## ")
	if strings.HasPrefix(body, "## ") {
		h2Count++
	}

	// Check frontmatter completeness
	fmScore := 0
	if s.Title != "" {
		fmScore += 20
	}
	if s.Type != "" {
		fmScore += 10
	}
	if s.Status != "" {
		fmScore += 10
	}

	// Word count
	words := len(strings.Fields(body))

	structScore := float64(fmScore)

	if h2Count >= 4 {
		structScore += 30
	} else if h2Count >= 2 {
		structScore += 20
	} else if h2Count >= 1 {
		structScore += 10
	}

	if words >= 200 {
		structScore += 30
	} else if words >= 100 {
		structScore += 20
	} else if words >= 50 {
		structScore += 10
	}

	d.Score = structScore
	if structScore >= 80 {
		d.Details = "well-structured with multiple sections and adequate detail"
	} else if structScore >= 50 {
		d.Details = "basic structure — could use more sections or detail"
	} else {
		d.Details = "minimal structure — spec is too short or lacks organization"
	}

	return d
}

// scoreAmbiguity detects vague language that leads to poor delivery.
// Weight: 10
func scoreAmbiguity(body string) Dimension {
	d := Dimension{Name: "Clarity", Weight: 10}

	lower := strings.ToLower(body)

	// Vague phrases that indicate ambiguity
	vaguePatterns := []string{
		"somehow", "maybe", "perhaps", "possibly",
		"should probably", "might need", "could potentially",
		"as needed", "as appropriate", "if necessary",
		"etc.", "and so on", "and more",
		"something like", "some kind of", "some sort of",
		"tbd", "to be determined", "to be decided",
		"not sure", "unclear", "figure out",
	}

	vagueCount := 0
	for _, phrase := range vaguePatterns {
		vagueCount += strings.Count(lower, phrase)
	}

	words := len(strings.Fields(body))
	if words == 0 {
		d.Score = 0
		d.Details = "spec body is empty"
		return d
	}

	// Ratio of vague phrases to total words
	ratio := float64(vagueCount) / float64(words) * 100

	switch {
	case vagueCount == 0:
		d.Score = 100
		d.Details = "no ambiguous language detected"
	case ratio < 0.5:
		d.Score = 80
		d.Details = pluralize(vagueCount, "vague phrase", "vague phrases") + " — minor ambiguity"
	case ratio < 1.0:
		d.Score = 50
		d.Details = pluralize(vagueCount, "vague phrase", "vague phrases") + " — moderate ambiguity"
	default:
		d.Score = 20
		d.Details = pluralize(vagueCount, "vague phrase", "vague phrases") + " — significant ambiguity"
	}

	return d
}

// --- helpers ---

var (
	listItemRe    = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+|^[\s]*\d+[.)]\s+`)
	measurableRe  = regexp.MustCompile(`(?i)\b(must|shall|should|returns?|produces?|fails?|passes?|within|at least|at most|no more than|exactly|\d+\s*(ms|seconds?|minutes?|bytes?|%))\b`)
	codeRefRe     = regexp.MustCompile("`[^`]+`")
	fileRefRe     = regexp.MustCompile(`(?i)[\w/.-]+\.(go|ts|tsx|js|jsx|py|java|rs|rb|groovy|yaml|yml|json|toml|sql|proto|graphql)`)
	bulletCriteriaRe = regexp.MustCompile(`(?im)^[\s]*[-*+]\s+.*(must|should|shall|verify|confirm|ensure|check|expect|return|produce|fail|pass)`)
)

func countListItems(s string) int {
	return len(listItemRe.FindAllStringIndex(s, -1))
}

func countMeasurableItems(s string) int {
	lines := strings.Split(s, "\n")
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") || (len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9')) {
			if measurableRe.MatchString(trimmed) {
				count++
			}
		}
	}
	return count
}

func countCodeRefs(s string) int {
	return len(codeRefRe.FindAllStringIndex(s, -1))
}

func countFileRefs(s string) int {
	return len(fileRefRe.FindAllStringIndex(s, -1))
}

func countBulletCriteria(s string) int {
	return len(bulletCriteriaRe.FindAllStringIndex(s, -1))
}

func extractSection(body string, headings ...string) string {
	lower := strings.ToLower(body)
	for _, h := range headings {
		target := "## " + h
		idx := strings.Index(lower, target)
		if idx < 0 {
			continue
		}
		// Find from this heading to the next ## or end
		rest := body[idx:]
		nextH2 := strings.Index(rest[3:], "\n## ")
		if nextH2 >= 0 {
			return rest[:nextH2+3]
		}
		return rest
	}
	return ""
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	s := ""
	v := n
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	if s == "" {
		s = "0"
	}
	return s + " " + plural
}

func generateSuggestions(dims []Dimension, body string) []string {
	var sugs []string
	for _, d := range dims {
		switch d.Name {
		case "Acceptance Criteria":
			if d.Score < 50 {
				sugs = append(sugs, "Add an ## Acceptance Criteria section with measurable, testable criteria")
			} else if d.Score < 75 {
				sugs = append(sugs, "Make acceptance criteria more measurable — use specific outcomes, numbers, or verifiable conditions")
			}
		case "Scope Clarity":
			if d.Score < 30 {
				sugs = append(sugs, "Add ## Goal and ## Design sections to define what this spec does and how")
			}
			if d.Score < 50 && !strings.Contains(strings.ToLower(body), "non-goal") {
				sugs = append(sugs, "Consider adding non-goals or out-of-scope section to bound the work")
			}
		case "Technical Specificity":
			if d.Score < 50 {
				sugs = append(sugs, "Reference specific files, packages, or APIs that will be affected")
			}
		case "Test Strategy":
			if d.Score < 50 {
				sugs = append(sugs, "Add a ## Test Strategy or ## Verification section describing how to validate the implementation")
			}
		case "Structure":
			if d.Score < 50 {
				sugs = append(sugs, "Add more detail — spec is too brief for reliable delivery")
			}
		case "Clarity":
			if d.Score < 50 {
				sugs = append(sugs, "Replace vague language (TBD, maybe, etc.) with concrete decisions")
			}
		}
	}
	return sugs
}
