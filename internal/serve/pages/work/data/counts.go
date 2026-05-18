package data

import (
	"github.com/hero-engine/hero/internal/spec"
)

// CountsInputs is the per-request input bundle for the page hero
// subhead counts.
type CountsInputs struct {
	ProjectRoot string
	HeroDir     string
}

// LoadCounts computes total / delivering / blocked + sprint state for
// the page hero subhead. SprintState falls back to "No active sprint"
// — the real sprint pipeline lives in sibling specs.
func LoadCounts(in CountsInputs) PageCounts {
	specs := loadSpecsBest(in.HeroDir)
	out := PageCounts{
		SprintState: "No active sprint",
	}
	for _, s := range specs {
		// Skip non-work types from the headline tally.
		switch s.Type {
		case spec.TypeContext, spec.TypeNote, spec.TypeExternal,
			spec.TypeConvention, spec.TypeRule, spec.TypeTripwire:
			continue
		}
		out.Total++
		switch s.Status {
		case spec.StatusDelivering, spec.StatusInReview:
			out.Delivering++
		}
		if isBlocked(s) {
			out.Blocked++
		}
	}
	return out
}
