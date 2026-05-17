package spectypes

import "fmt"

// LintIssue is a registry-driven structural validation finding.
// Mirrors triage.StructuralIssue's shape so callers can adapt one to
// the other.
type LintIssue struct {
	Field   string
	Message string
	IsError bool
}

// ValidateTypeAndStatus runs the registry-driven equivalent of the
// legacy triage.ValidateStructure type+status checks. Used by the
// parity test to confirm the registry produces the same verdict as
// the hardcoded validator for every spec in the corpus.
//
// Returns issues sorted by field for stable comparison.
func ValidateTypeAndStatus(reg Registry, typeName, status string) []LintIssue {
	var issues []LintIssue

	if typeName == "" {
		issues = append(issues, LintIssue{Field: "type", Message: "type is required", IsError: true})
		return issues
	}

	rec, ok := reg.Lookup(typeName)
	if !ok {
		var known []string
		for _, r := range reg.All() {
			known = append(known, r.Name)
		}
		issues = append(issues, LintIssue{
			Field:   "type",
			Message: fmt.Sprintf("invalid type: %s (registered: %v)", typeName, known),
			IsError: true,
		})
		return issues
	}

	if status == "" {
		issues = append(issues, LintIssue{Field: "status", Message: "status is required", IsError: true})
		return issues
	}

	// If the type declares a lifecycle, validate against it; otherwise
	// any non-empty status is fine (matches the legacy validator's
	// "default" branch for initiative/external/context/note).
	if len(rec.Lifecycle.States) == 0 {
		return issues
	}

	for _, s := range rec.Lifecycle.States {
		if s == status {
			return issues
		}
	}

	issues = append(issues, LintIssue{
		Field:   "status",
		Message: fmt.Sprintf("invalid status for %s: %s (must be one of: %v)", typeName, status, rec.Lifecycle.States),
		IsError: true,
	})
	return issues
}

// ValidateKind validates a spec's declared kind against the
// registry-declared kind enum for its type. Returns nil if the kind
// is empty (kind is optional) or in the enum.
func ValidateKind(reg Registry, typeName, kind string) []LintIssue {
	if kind == "" {
		return nil
	}
	rec, ok := reg.Lookup(typeName)
	if !ok {
		return nil // ValidateTypeAndStatus catches the bad type
	}
	if !rec.Kind.HasKind() {
		// Type declares no kind enum; any kind value is unexpected.
		return []LintIssue{{
			Field:   "kind",
			Message: fmt.Sprintf("type %s does not declare a kind enum; got %q", typeName, kind),
			IsError: true,
		}}
	}
	for _, v := range rec.Kind.Values {
		if v == kind {
			return nil
		}
	}
	return []LintIssue{{
		Field:   "kind",
		Message: fmt.Sprintf("invalid kind for %s: %s (must be one of: %v)", typeName, kind, rec.Kind.Values),
		IsError: true,
	}}
}
