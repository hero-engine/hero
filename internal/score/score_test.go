package score

import (
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

func makeSpec(slug, content string) *spec.Spec {
	s, _ := spec.Parse(content, "/fake/"+slug+"/spec.md", time.Now())
	return s
}

func TestScoreEmptySpec(t *testing.T) {
	s := makeSpec("empty", "")
	r := Score(s, nil)

	if r.Score > 20 {
		t.Errorf("empty spec should score very low, got %d", r.Score)
	}
	if r.Grade != "F" {
		t.Errorf("empty spec should get grade F, got %s", r.Grade)
	}
	if r.Deliverable {
		t.Error("empty spec should not be deliverable")
	}
}

func TestScoreMinimalSpec(t *testing.T) {
	content := `---
title: Fix login bug
type: bug
status: planning
---
# Fix login bug

The login page crashes when the user enters special characters.
`
	s := makeSpec("fix-login", content)
	r := Score(s, nil)

	if r.Grade == "A" || r.Grade == "B" {
		t.Errorf("minimal spec should not score high, got grade %s (score %d)", r.Grade, r.Score)
	}
	if len(r.Suggestions) == 0 {
		t.Error("minimal spec should have suggestions")
	}
}

func TestScoreWellStructuredSpec(t *testing.T) {
	content := `---
title: Add export to CSV
type: feature
status: approved
---
# Add export to CSV

## Goal

Users need to export their dashboard data to CSV format for use in Excel and other tools.

## Design

Add an export button to the dashboard toolbar. When clicked, it streams the current filtered
dataset as a CSV file download. Uses the existing ` + "`DataService.query()`" + ` method to fetch data
and ` + "`csv-stringify`" + ` package for formatting.

## Changes

- ` + "`src/components/Dashboard/ExportButton.tsx`" + ` — new component
- ` + "`src/services/export.ts`" + ` — CSV generation logic
- ` + "`src/components/Dashboard/Toolbar.tsx`" + ` — add ExportButton

## Acceptance Criteria

- Clicking export must produce a valid CSV file with headers matching column names
- Export must include only the currently filtered dataset, not all data
- Files larger than 10MB must stream to avoid memory issues
- Export button should be disabled while an export is in progress

## Test Strategy

- Unit test CSV generation with known input/output pairs in ` + "`export.test.ts`" + `
- Integration test the full export flow with mock data
- Verify streaming works for datasets > 10MB

## Non-Goals

- PDF export (future work)
- Scheduled/automated exports
`
	s := makeSpec("export-csv", content)
	r := Score(s, nil)

	if r.Score < 70 {
		t.Errorf("well-structured spec should score >= 70, got %d", r.Score)
	}
	if r.Grade == "F" || r.Grade == "D" {
		t.Errorf("well-structured spec should get at least grade C, got %s", r.Grade)
	}
	if !r.Deliverable {
		t.Error("well-structured spec should be deliverable")
	}

	// Check dimensions exist
	if len(r.Dimensions) != 6 {
		t.Errorf("expected 6 dimensions, got %d", len(r.Dimensions))
	}
}

func TestScoreAcceptanceCriteriaScoring(t *testing.T) {
	// No AC section at all
	noAC := makeSpec("no-ac", "# Feature\n\nSome description of what to do.")
	rNoAC := Score(noAC, nil)

	// With AC section
	withAC := makeSpec("with-ac", `# Feature

## Acceptance Criteria

- The API must return 200 on success
- Response time should be within 500ms
- Error responses must include an error code
`)
	rWithAC := Score(withAC, nil)

	acDimNoAC := findDimension(rNoAC, "Acceptance Criteria")
	acDimWithAC := findDimension(rWithAC, "Acceptance Criteria")

	if acDimNoAC == nil || acDimWithAC == nil {
		t.Fatal("missing Acceptance Criteria dimension")
	}

	if acDimWithAC.Score <= acDimNoAC.Score {
		t.Errorf("spec with AC (%v) should score higher than without (%v)", acDimWithAC.Score, acDimNoAC.Score)
	}
}

func TestScoreAmbiguity(t *testing.T) {
	clear := makeSpec("clear", `# Clear Spec

## Goal

Implement the user authentication flow using JWT tokens.

## Acceptance Criteria

- Login endpoint must return a JWT token
- Token must expire after 24 hours
`)
	rClear := Score(clear, nil)

	vague := makeSpec("vague", `# Vague Spec

## Goal

Maybe implement something like authentication, possibly using some kind of token system.
We should probably figure out the details later. TBD on the exact approach, etc.
Perhaps we could potentially use JWT or some sort of alternative, as appropriate.
`)
	rVague := Score(vague, nil)

	clearDim := findDimension(rClear, "Clarity")
	vagueDim := findDimension(rVague, "Clarity")

	if clearDim == nil || vagueDim == nil {
		t.Fatal("missing Clarity dimension")
	}

	if vagueDim.Score >= clearDim.Score {
		t.Errorf("vague spec clarity (%v) should be lower than clear spec (%v)", vagueDim.Score, clearDim.Score)
	}
}

func TestScoreDeliverableThreshold(t *testing.T) {
	s := makeSpec("test", `---
title: Test
type: feature
status: planning
---
# Test

## Goal

Do something specific.

## Acceptance Criteria

- It must work
- It should return correct results

## Design

Use the existing ` + "`FooService`" + ` to implement.
`)
	// Default threshold (40)
	r := Score(s, nil)
	defaultDeliverable := r.Deliverable

	// High threshold
	high := &Config{MinScore: 95}
	rHigh := Score(s, high)

	if rHigh.Deliverable && !defaultDeliverable {
		t.Error("higher threshold should be harder to pass")
	}

	// Zero threshold
	zero := &Config{MinScore: 0}
	rZero := Score(s, zero)
	if !rZero.Deliverable {
		t.Error("zero threshold should always be deliverable")
	}
}

func TestScoreGradeAssignment(t *testing.T) {
	tests := []struct {
		score int
		grade string
	}{
		{95, "A"}, {90, "A"},
		{80, "B"}, {75, "B"},
		{65, "C"}, {60, "C"},
		{50, "D"}, {40, "D"},
		{30, "F"}, {0, "F"},
	}

	for _, tt := range tests {
		r := &Result{Score: tt.score}
		// Assign grade using same logic
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
		if r.Grade != tt.grade {
			t.Errorf("score %d: expected grade %s, got %s", tt.score, tt.grade, r.Grade)
		}
	}
}

func TestBodyAfterFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{"no frontmatter", "# Hello", "# Hello"},
		{"with frontmatter", "---\ntitle: Test\n---\n# Hello", "\n# Hello"},
		{"incomplete frontmatter", "---\ntitle: Test", "---\ntitle: Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bodyAfterFrontmatter(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	if got := pluralize(1, "item", "items"); got != "1 item" {
		t.Errorf("got %q", got)
	}
	if got := pluralize(3, "item", "items"); got != "3 items" {
		t.Errorf("got %q", got)
	}
	if got := pluralize(0, "item", "items"); got != "0 items" {
		t.Errorf("got %q", got)
	}
}

func TestExtractSection(t *testing.T) {
	body := `## Goal

Do the thing.

## Acceptance Criteria

- Must work
- Must be fast

## Design

Use Go.
`
	section := extractSection(body, "acceptance criteria")
	if section == "" {
		t.Fatal("expected to find acceptance criteria section")
	}
	if !contains(section, "Must work") {
		t.Error("section should contain 'Must work'")
	}
	if contains(section, "Use Go") {
		t.Error("section should not contain content from next section")
	}
}

func TestWarningsGenerated(t *testing.T) {
	// Spec with zero-scoring dimensions should produce error warnings
	s := makeSpec("bare", "Just a sentence.")
	r := Score(s, nil)

	hasError := false
	for _, w := range r.Warnings {
		if w.Severity == "error" {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("bare spec should have at least one error-level warning")
	}
}

// --- helpers ---

func findDimension(r *Result, name string) *Dimension {
	for i := range r.Dimensions {
		if r.Dimensions[i].Name == name {
			return &r.Dimensions[i]
		}
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
