package triage

import (
	"fmt"

	"github.com/hero-engine/hero/internal/spec"
)

// Issue is a single triage finding for a spec.
type Issue struct {
	Kind    string // "error" or "warning"
	Check   string // "structural", "duplicate", "conflict", "orphan"
	Message string
	Target  string  // related slug, if applicable
	Score   float64 // similarity score for duplicates
}

// Result holds all triage findings for a single spec.
type Result struct {
	Slug   string
	Passed bool
	Issues []Issue
}

// Options controls triage behaviour.
type Options struct {
	DuplicateThreshold float64 // default 0.80
	TagOverlapMin      int     // default 3
}

// Triage runs all checks on candidate against the corpus.
func Triage(candidate *spec.Spec, corpus []*spec.Spec, opts Options) Result {
	if opts.DuplicateThreshold <= 0 {
		opts.DuplicateThreshold = 0.80
	}
	if opts.TagOverlapMin <= 0 {
		opts.TagOverlapMin = 3
	}

	var issues []Issue

	// 1. Structural checks
	structIssues := ValidateStructure(candidate)
	for _, si := range structIssues {
		kind := "warning"
		if si.IsError {
			kind = "error"
		}
		issues = append(issues, Issue{
			Kind:    kind,
			Check:   "structural",
			Message: si.Message,
		})
	}

	// 2. Duplicate detection
	dupes := FindDuplicates(candidate, corpus, opts.DuplicateThreshold, opts.TagOverlapMin)
	for _, d := range dupes {
		issues = append(issues, Issue{
			Kind:    "warning",
			Check:   "duplicate",
			Message: fmt.Sprintf("possible duplicate of %s (similarity %.0f%%)", d.Slug, d.Similarity*100),
			Target:  d.Slug,
			Score:   d.Similarity,
		})
	}

	// 3. Convention/rule conflict detection
	conflicts := FindConflicts(candidate, corpus)
	for _, c := range conflicts {
		issues = append(issues, Issue{
			Kind:    "warning",
			Check:   "conflict",
			Message: fmt.Sprintf("conflicts with %s on subject %q: %q vs %q", c.TargetSlug, c.Subject, c.OurRule, c.TheirRule),
			Target:  c.TargetSlug,
		})
	}

	// 4. Orphaned relations
	slugIndex := make(map[string]bool, len(corpus))
	for _, s := range corpus {
		slugIndex[s.Slug] = true
	}
	for _, rel := range candidate.Relations {
		if !slugIndex[rel.Target] {
			issues = append(issues, Issue{
				Kind:    "warning",
				Check:   "orphan",
				Message: fmt.Sprintf("relation %s:%s — target spec %q not found in corpus", rel.Kind, rel.Target, rel.Target),
				Target:  rel.Target,
			})
		}
	}

	passed := true
	for _, iss := range issues {
		if iss.Kind == "error" {
			passed = false
			break
		}
	}

	return Result{
		Slug:   candidate.Slug,
		Passed: passed,
		Issues: issues,
	}
}
