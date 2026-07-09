package data

import (
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// KnowledgeInputs is the per-request input bundle for the Knowledge
// Stats section.
type KnowledgeInputs struct {
	HeroDir string
}

// Knowledge is what the partial renders. All counts default to zero
// when HeroDir is empty or unreadable.
type Knowledge struct {
	Decisions   int
	Conventions int
	Notes       int
	Captures    int // rules + context — operator-curated knowledge

	LastCapturedAt       time.Time
	LastCapturedAtPretty string
	LastCapturedSlug     string
	LastCapturedKind     string

	// KnowledgeHref is "/knowledge" — surfaced so the partial can link
	// without hardcoding the route.
	KnowledgeHref string
}

// LoadKnowledge counts knowledge artifacts under the .hero directory.
// Per the spec it's a thin tally — no body parsing.
func LoadKnowledge(in KnowledgeInputs) Knowledge {
	out := Knowledge{KnowledgeHref: "/knowledge"}
	if in.HeroDir == "" {
		return out
	}
	specs, err := spec.Discover(in.HeroDir)
	if err != nil {
		return out
	}
	var lastCaptured *spec.Spec
	for _, s := range specs {
		if s == nil {
			continue
		}
		switch s.Type {
		case spec.TypeDecision:
			out.Decisions++
		case spec.TypeConvention:
			out.Conventions++
		case spec.TypeNote:
			out.Notes++
		case spec.TypeRule, spec.TypeContext, spec.TypeExplainer:
			out.Captures++
		default:
			continue
		}
		if lastCaptured == nil || s.ModifiedAt.After(lastCaptured.ModifiedAt) {
			lastCaptured = s
		}
	}
	if lastCaptured != nil {
		out.LastCapturedAt = lastCaptured.ModifiedAt
		out.LastCapturedAtPretty = lastCaptured.ModifiedAt.Format("2006-01-02 15:04")
		out.LastCapturedSlug = lastCaptured.Slug
		out.LastCapturedKind = string(lastCaptured.Type)
	}
	return out
}
