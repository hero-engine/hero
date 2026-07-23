package spec

import (
	"testing"
	"time"
)

// TestPredicateCategoriesMutuallyExclusive locks the three-category
// partition: a spec type is never both work and knowledge, never both
// pre-commitment and either, and intake is pre-commitment only. Initiative
// is deliberately none of the three (it is a container, not in-flight
// work) — so this asserts mutual exclusivity, not full coverage.
func TestPredicateCategoriesMutuallyExclusive(t *testing.T) {
	allTypes := []Type{
		TypeFeature, TypeBug, TypeConvention, TypeDecision, TypeInitiative,
		TypeRule, TypeExternal, TypeContext, TypeNote, TypeTripwire,
		TypeExplainer, TypeIntake,
	}
	for _, ty := range allTypes {
		s := &Spec{Type: ty}
		n := 0
		if s.IsWorkSpec() {
			n++
		}
		if s.IsKnowledge() {
			n++
		}
		if s.IsPreCommitment() {
			n++
		}
		if n > 1 {
			t.Errorf("type %q matches %d categories, want at most 1", ty, n)
		}
	}
}

func TestIsPreCommitment(t *testing.T) {
	if !(&Spec{Type: TypeIntake}).IsPreCommitment() {
		t.Error("intake should be pre-commitment")
	}
	// Pre-commitment must be disjoint from work and knowledge.
	intake := &Spec{Type: TypeIntake}
	if intake.IsWorkSpec() {
		t.Error("intake must not be a work spec (would leak into rollups)")
	}
	if intake.IsKnowledge() {
		t.Error("intake must not be knowledge")
	}
	for _, ty := range []Type{TypeFeature, TypeBug, TypeNote, TypeInitiative} {
		if (&Spec{Type: ty}).IsPreCommitment() {
			t.Errorf("type %q should not be pre-commitment", ty)
		}
	}
}

// TestIntakeNotReady is the queue no-leak guard: an open-status intake
// must never be eligible for `hero queue` (IsReady false), unlike a
// feature in the same planning status.
func TestIntakeNotReady(t *testing.T) {
	bySlug := map[string]*Spec{}

	intake := &Spec{Slug: "an-idea", Type: TypeIntake, Status: StatusPlanning}
	if IsReady(intake, bySlug) {
		t.Error("intake with planning status must not be ready (queue leak)")
	}

	feature := &Spec{Slug: "real-work", Type: TypeFeature, Status: StatusPlanning}
	if !IsReady(feature, bySlug) {
		t.Error("a planning feature with no deps should be ready (sanity check)")
	}

	initiative := &Spec{Slug: "container", Type: TypeInitiative, Status: StatusPlanning}
	if !IsReady(initiative, bySlug) {
		t.Error("a planning initiative with no deps should retain queue eligibility")
	}

	note := &Spec{Slug: "reference", Type: TypeNote, Status: StatusPlanning}
	if IsReady(note, bySlug) {
		t.Error("knowledge must not become queue-eligible")
	}
}

// TestParseHonorsIntakeFrontmatterType confirms an explicit `type: intake`
// in frontmatter wins, so a discovered intake is modeled as TypeIntake
// rather than falling back to the path/feature default.
func TestParseHonorsIntakeFrontmatterType(t *testing.T) {
	content := `---
title: Export to CSV
type: intake
status: planning
---
# Export to CSV

## Signal

Users keep asking.
`
	s, err := Parse(content, "/project/.hero/planning/intake/csv-export/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if s.Type != TypeIntake {
		t.Errorf("Type = %q, want intake", s.Type)
	}
	if !s.IsPreCommitment() {
		t.Error("parsed intake should be pre-commitment")
	}
}
