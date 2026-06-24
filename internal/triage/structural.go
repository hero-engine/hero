package triage

import "github.com/hero-engine/hero/internal/spec"

// StructuralIssue represents a structural validation problem.
type StructuralIssue struct {
	Field   string
	Message string
	IsError bool // true=error (blocks import), false=warning
}

// validTypes is the set of recognised spec types.
var validTypes = map[spec.Type]bool{
	spec.TypeFeature:    true,
	spec.TypeBug:        true,
	spec.TypeConvention: true,
	spec.TypeDecision:   true,
	spec.TypeInitiative: true,
	spec.TypeRule:       true,
	spec.TypeExternal:   true,
	spec.TypeContext:    true,
	spec.TypeNote:       true,
	spec.TypeExplainer:  true,
}

// workStatuses are valid statuses for feature/bug specs.
var workStatuses = map[spec.Status]bool{
	spec.StatusPlanning:   true,
	spec.StatusInReview:   true,
	spec.StatusDelivering: true,
	spec.StatusCompleted:  true,
}

// conventionStatuses are valid statuses for convention/rule specs.
var conventionStatuses = map[spec.Status]bool{
	spec.StatusDraft:      true,
	spec.StatusActive:     true,
	spec.StatusSuperseded: true,
}

// decisionStatuses are valid statuses for decision specs.
var decisionStatuses = map[spec.Status]bool{
	spec.StatusProposed:   true,
	spec.StatusAccepted:   true,
	spec.StatusSuperseded: true,
}

// ValidateStructure checks required frontmatter fields and valid enum values.
func ValidateStructure(s *spec.Spec) []StructuralIssue {
	var issues []StructuralIssue

	// title required
	if s.Title == "" {
		issues = append(issues, StructuralIssue{
			Field:   "title",
			Message: "title is required",
			IsError: true,
		})
	}

	// type required and must be valid
	if s.Type == "" {
		issues = append(issues, StructuralIssue{
			Field:   "type",
			Message: "type is required",
			IsError: true,
		})
	} else if !validTypes[s.Type] {
		issues = append(issues, StructuralIssue{
			Field:   "type",
			Message: "invalid type: " + string(s.Type) + " (must be one of: feature, bug, convention, decision, initiative, rule, external, context, note)",
			IsError: true,
		})
	}

	// status required and must be valid for the type
	if s.Status == "" {
		issues = append(issues, StructuralIssue{
			Field:   "status",
			Message: "status is required",
			IsError: true,
		})
	} else {
		switch s.Type {
		case spec.TypeFeature, spec.TypeBug:
			if !workStatuses[s.Status] {
				issues = append(issues, StructuralIssue{
					Field:   "status",
					Message: "invalid status for " + string(s.Type) + ": " + string(s.Status) + " (must be one of: planning, in-review, delivering, completed)",
					IsError: true,
				})
			}
		case spec.TypeConvention, spec.TypeRule:
			if !conventionStatuses[s.Status] {
				issues = append(issues, StructuralIssue{
					Field:   "status",
					Message: "invalid status for " + string(s.Type) + ": " + string(s.Status) + " (must be one of: draft, active, superseded)",
					IsError: true,
				})
			}
		case spec.TypeDecision:
			if !decisionStatuses[s.Status] {
				issues = append(issues, StructuralIssue{
					Field:   "status",
					Message: "invalid status for decision: " + string(s.Status) + " (must be one of: proposed, accepted, superseded)",
					IsError: true,
				})
			}
		default:
			// initiative, external, context, note — any non-empty status is fine
		}
	}

	// created is a warning if zero (fixable by --fix)
	if s.CreatedAt.IsZero() {
		issues = append(issues, StructuralIssue{
			Field:   "created",
			Message: "created date is missing (fixable with --fix)",
			IsError: false,
		})
	}

	return issues
}
