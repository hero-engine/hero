// Package governance defines the classification, subject, principal,
// policy, audit, and retrieval contracts shared between hero (CLI) and
// hero-cloud (enforcement engine). The vocabulary lives here; the
// enforcement implementation lives in hero-cloud.
package governance

import "errors"

// Classification is an ordered sensitivity level on a graph node.
// Higher values are more sensitive. Egress decisions compare the maximum
// classification in a result set against the caller's clearance.
type Classification int

// Built-in classification levels. Organizations may register additional
// levels between or after these via RegisterClassification.
const (
	// Public content may be shared without restriction.
	Public Classification = 100
	// Internal content is restricted to org members; the team default.
	Internal Classification = 200
	// Confidential content is restricted to scoped subsets of org members.
	Confidential Classification = 300
	// Restricted content is the most sensitive built-in level (e.g. policy
	// nodes themselves, audit events).
	Restricted Classification = 400
)

// Compare returns -1 if a is less sensitive than b, 0 if equal, and +1
// if a is more sensitive than b.
func (a Classification) Compare(b Classification) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// Max returns the more-sensitive of a and b. Used by the
// enrichment-inherits-max rule: a node derived from inputs takes the
// maximum classification of its inputs.
func Max(a, b Classification) Classification {
	if a.Compare(b) >= 0 {
		return a
	}
	return b
}

// ErrClassificationAlreadyRegistered is returned by RegisterClassification
// when a level with the given ordinal value already exists.
var ErrClassificationAlreadyRegistered = errors.New("governance: classification ordinal already registered")

// RegisterClassification registers an organization-defined classification
// level with the given name and ordinal value. Ordinals fit between or
// after the built-ins by choosing a value in the appropriate range.
//
// The registry itself lives in the enforcement engine; this signature is
// the contract every implementation honors. The default contracts build
// holds no registry state, so this returns nil and performs no work —
// implementations override the binding at startup.
func RegisterClassification(_ string, _ Classification) error {
	return nil
}
