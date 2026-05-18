package snapshot

import (
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// StageInputs is the bundle of per-surface signals stage derivation
// uses. Pure-function: given the same StageInputs, DeriveStage
// returns the same Stage.
type StageInputs struct {
	// SpecsByStatus is a count of specs assigned to this surface,
	// keyed by canonical Status string.
	SpecsByStatus map[spec.Status]int
	// ReleaseDone / ReleaseTotal report the count of resolved
	// release-target completions for this surface. When Total is
	// zero no release signal has resolved, so the stage rule that
	// depends on "% of v1 release scope" falls back.
	ReleaseDone  int
	ReleaseTotal int
	// LastTouched is the most recent commit / spec mutation timestamp
	// attributed to this surface. Zero when no activity.
	LastTouched time.Time
	// HasShippedTag is true when the surface has an associated
	// release tag (e.g. v1.0.0 git tag). Used as a tie-breaker for
	// the maturing vs shipped distinction.
	HasShippedTag bool
}

// DeriveStage applies the six-stage taxonomy from the spec. Caller
// is responsible for pinning when the override layer set Stage on
// the surface — this function never reads pin state.
//
// Rules (first match wins):
//   - concept: zero specs delivering or completed
//   - scaffolded: at least one completed spec AND no v1 release scope declared
//   - building: at least one delivering spec AND <50% of v1 scope complete
//   - shipping-v1: >=50% of v1 scope complete
//   - shipped: all v1 specs completed AND no in-flight v1 spec
//   - maturing: shipped AND only follow-up specs in flight (no v1 in flight)
func DeriveStage(in StageInputs) Stage {
	delivering := in.SpecsByStatus[spec.StatusDelivering]
	inReview := in.SpecsByStatus[spec.StatusInReview]
	completed := in.SpecsByStatus[spec.StatusCompleted]

	// concept: nothing's moved.
	if delivering+inReview+completed == 0 {
		return StageConcept
	}

	// If no release signal resolved, we degrade gracefully:
	//   - any completion + no in-flight → maturing (post-ship cleanup)
	//   - any in-flight → building (work happening, no scope claim)
	//   - completion-only-but-no-release → scaffolded
	if in.ReleaseTotal == 0 {
		if delivering+inReview > 0 {
			return StageBuilding
		}
		if in.HasShippedTag {
			return StageMaturing
		}
		return StageScaffolded
	}

	// Release-aware rules.
	if in.ReleaseDone >= in.ReleaseTotal && delivering+inReview == 0 {
		// All v1 work landed; nothing in flight.
		// Distinguish "shipped" (just landed) from "maturing" (long-tail polish).
		// Heuristic: if a shipped tag exists, call it maturing; else shipped.
		if in.HasShippedTag {
			return StageMaturing
		}
		return StageShipped
	}

	if in.ReleaseTotal > 0 {
		ratio := float64(in.ReleaseDone) / float64(in.ReleaseTotal)
		if ratio >= 0.5 {
			return StageShippingV1
		}
	}

	if delivering+inReview > 0 {
		return StageBuilding
	}

	return StageScaffolded
}
