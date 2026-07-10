// Package drive holds the autonomy-boundary logic for `/drive` — the
// deterministic predicate that decides, at each transition in an autonomous
// initiative run, whether to proceed or pause for the human. It is a
// sibling (on a different axis) to the intake committed-work predicate:
// where that classifies *is this real work*, needs_me classifies *does
// advancing need a human*. Pure and unit-testable in isolation; all I/O
// happens in the caller and is passed in via RunContext.
package drive

import (
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// AutonomyMode is the per-initiative `/drive` policy that tunes how
// aggressively needs_me proceeds. Parsed from the `autonomy:` frontmatter
// field via ParseMode.
type AutonomyMode int

const (
	// Supervised pauses at every spec boundary — behavioral parity with
	// today's hand-approved flow. The default for an unset field.
	Supervised AutonomyMode = iota
	// Guided proceeds only on the proceed-silently set; pauses on every
	// taxonomy category.
	Guided
	// Autonomous behaves like Guided but additionally proceeds on
	// categories the learning layer has promoted (see drive-autonomy-learning).
	// Hard-pause guardrails are never relaxed.
	Autonomous
)

// ParseMode maps the `autonomy:` frontmatter value to a mode. Unknown or
// empty values default to Supervised (the safe, today's-behavior choice).
func ParseMode(v string) AutonomyMode {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "guided":
		return Guided
	case "autonomous":
		return Autonomous
	default:
		return Supervised
	}
}

// PauseCategory names why a transition needs the human.
type PauseCategory string

const (
	CategoryNone          PauseCategory = ""
	CategoryDesignFork    PauseCategory = "DesignFork"     // ≥2 viable approaches with material tradeoffs
	CategoryUnderspecified PauseCategory = "Underspecified" // candidate next spec scores below threshold
	CategoryIrreversible  PauseCategory = "Irreversible"   // migration/delete/deploy/external-send — HARD pause
	CategoryVerifyStuck   PauseCategory = "VerifyStuck"     // verify still FAILs after N rework passes
	CategoryBlocked       PauseCategory = "Blocked"         // dependency/externally blocked
	CategoryAmbiguousPick PauseCategory = "AmbiguousPick"   // queue near-tie between specs of different intent
	CategoryHardCap       PauseCategory = "HardCap"         // initiative boundary or N-consecutive-proceeds cap
	CategoryUnknown       PauseCategory = "Unknown"         // unclassifiable transition — conservative pause
	CategorySupervised    PauseCategory = "Supervised"      // supervised mode pauses at every boundary
	// CategorySeamCollision is a soft-mutex collision: the candidate
	// conflicts-with a spec that is currently delivering. A real obstacle,
	// non-promotable — even Autonomous mode pauses.
	CategorySeamCollision PauseCategory = "SeamCollision"
	// CategorySeamDetected is a heuristic whole-file overlap between the
	// candidate and a currently-delivering spec that nobody authored as a
	// conflicts-with relation. Unlike CategorySeamCollision (a deliberate,
	// authored statement of intent — hard, non-promotable), this is a
	// file-granular *suspicion*, so it is soft and **promotable**: it surfaces
	// every time by default but Autonomous can auto-proceed once the learning
	// layer has accepted the noise.
	CategorySeamDetected PauseCategory = "SeamDetected"
)

// Promotable reports whether a category may ever be auto-proceeded by the
// learning layer in Autonomous mode. The hard-pause guardrails and real
// obstacles (Irreversible, HardCap, Unknown, VerifyStuck, Blocked,
// SeamCollision) are never promotable; only the human-judgment categories are.
func (c PauseCategory) Promotable() bool {
	switch c {
	case CategoryDesignFork, CategoryUnderspecified, CategoryAmbiguousPick, CategorySeamDetected:
		return true
	default:
		return false
	}
}

// Decision is the predicate's verdict for one transition.
type Decision struct {
	Proceed  bool
	Category PauseCategory // populated when !Proceed
	Reason   string        // human-readable, feeds the pause question
}

// RunContext carries the already-computed signals needs_me reasons over.
// The caller runs score/verify/blocked lookups and the action classifier;
// the predicate stays pure.
type RunContext struct {
	// VerifyVerdict for the just-finished spec: "PASS", "FAIL", or "".
	VerifyVerdict string
	// VerifyFailCount is consecutive verify failures on the current spec.
	VerifyFailCount int
	// VerifyStuckThreshold is the N at which repeated FAILs stop being
	// rework and become a VerifyStuck pause. <=0 disables (never stuck).
	VerifyStuckThreshold int

	// NextScore is the candidate next spec's readiness score; <0 = unknown.
	NextScore int
	// ScoreThreshold is the minimum acceptable readiness score.
	ScoreThreshold int

	Blocked       bool // next work is dependency/externally blocked
	DesignFork    bool // design surfaced ≥2 viable approaches
	AmbiguousPick bool // queue near-tie of different intent

	// SeamBlocked marks that the only otherwise-ready candidate is excluded
	// because a conflicts-with target is currently delivering (soft-mutex
	// seam collision). Distinct from Blocked (unmet dependency).
	SeamBlocked bool
	// SeamConflictSlug names the in-flight conflicting spec, for the reason.
	SeamConflictSlug string

	// SeamDetected marks that the selected candidate whole-file-overlaps a
	// currently-delivering spec that was NOT declared as a conflicts-with
	// relation — a heuristic seam nobody wrote down. Softer and promotable,
	// unlike the authored SeamBlocked. Set by Check only when a detector
	// callback is injected; nil detector leaves it false (piece-1 behavior).
	SeamDetected bool
	// SeamDetectedSlug names the in-flight (delivering) spec whose files
	// overlap the candidate; SeamDetectedFiles are the overlapping paths
	// (index order). Both feed the pause reason.
	SeamDetectedSlug  string
	SeamDetectedFiles []string

	// Irreversible: the pending action touches an irreversible /
	// outward-facing surface. Always pauses, every mode.
	Irreversible bool
	// ActionClassified is false when the pending action shape is unknown —
	// treated as Irreversible-adjacent: conservative pause.
	ActionClassified bool

	// ConsecutiveProceeds counts proceeds since the last pause; HardCap
	// forces a pause once it reaches the cap.
	ConsecutiveProceeds int
	HardCap             int // 0 disables the consecutive-proceed cap
	// AtInitiativeBoundary forces a pause regardless of mode.
	AtInitiativeBoundary bool

	// Promoted, when non-nil, reports whether a category has been promoted
	// to auto-proceed for this user (supplied by the learning layer).
	// Consulted only in Autonomous mode and only for Promotable categories.
	Promoted func(PauseCategory) bool
}

func pause(cat PauseCategory, reason string) Decision {
	return Decision{Proceed: false, Category: cat, Reason: reason}
}

// seamReason renders the SeamCollision pause reason, naming both the candidate
// and the in-flight conflicting spec. Shared by NeedsMe and DryRun so the two
// surfaces stay byte-identical.
func seamReason(name, conflict string) string {
	return fmt.Sprintf("%s conflicts with %s, which is currently delivering — pausing so they don't run concurrently", name, conflict)
}

// seamDetectedReason renders the SeamDetected pause reason: a heuristic
// whole-file overlap between the candidate and an in-flight (delivering) spec
// that nobody authored as a conflicts-with. Names the candidate, the in-flight
// spec, and the overlapping file(s) — with a count when more than one — so a
// human immediately sees it's a same-file, maybe-not-real seam. Deterministic:
// the caller supplies files in index order.
func seamDetectedReason(candidate, conflictSlug string, files []string) string {
	var where string
	switch len(files) {
	case 0:
		where = ""
	case 1:
		where = " on " + files[0]
	default:
		where = fmt.Sprintf(" on %d files (%s)", len(files), strings.Join(files, ", "))
	}
	return fmt.Sprintf("%s overlaps%s with %s, which is currently delivering, but nobody declared a conflicts-with — pausing to surface the undeclared seam", candidate, where, conflictSlug)
}

func proceed() Decision { return Decision{Proceed: true} }

// NeedsMe reports whether advancing past `at` needs the human, given the
// run's autonomy mode and observable context. Conservative: anything
// unknown pauses. Hard-pause guardrails are enforced first and are never
// relaxed by mode or promotion.
func NeedsMe(at *spec.Spec, ctx RunContext, mode AutonomyMode) Decision {
	name := "the next step"
	if at != nil && at.Slug != "" {
		name = at.Slug
	}

	// --- Hard-pause guardrails (mode-independent, never relaxed) ---
	if ctx.Irreversible {
		return pause(CategoryIrreversible, fmt.Sprintf("%s touches an irreversible or outward-facing action — always your call", name))
	}
	if !ctx.ActionClassified {
		return pause(CategoryUnknown, fmt.Sprintf("the pending action for %s could not be classified — pausing to be safe", name))
	}
	if ctx.AtInitiativeBoundary {
		return pause(CategoryHardCap, "reached an initiative boundary — pausing for a checkpoint")
	}
	if ctx.HardCap > 0 && ctx.ConsecutiveProceeds >= ctx.HardCap {
		return pause(CategoryHardCap, fmt.Sprintf("reached the %d-step cap — pausing so the run never runs unbounded", ctx.HardCap))
	}

	// --- Supervised: pause at every boundary (parity with today) ---
	if mode == Supervised {
		return pause(CategorySupervised, fmt.Sprintf("supervised mode pauses at every spec boundary (before %s)", name))
	}

	// --- Taxonomy categories (Guided + Autonomous) ---
	if ctx.VerifyVerdict == "FAIL" && ctx.VerifyStuckThreshold > 0 && ctx.VerifyFailCount >= ctx.VerifyStuckThreshold {
		return maybePromoted(CategoryVerifyStuck,
			fmt.Sprintf("verify has failed %d times on %s — needs your eyes, not another retry", ctx.VerifyFailCount, name), ctx, mode)
	}
	// SeamCollision is a real obstacle, not a human-judgment fork — return a
	// plain pause (never via maybePromoted) so it pauses in every mode,
	// Autonomous included.
	if ctx.SeamBlocked {
		return pause(CategorySeamCollision, seamReason(name, ctx.SeamConflictSlug))
	}
	// SeamDetected is evaluated AFTER SeamBlocked so an authored collision
	// always wins. It is a heuristic (whole-file) suspicion, so — unlike the
	// authored SeamCollision — it routes through maybePromoted: it pauses in
	// Guided but Autonomous can proceed once the category has been promoted.
	if ctx.SeamDetected {
		return maybePromoted(CategorySeamDetected,
			seamDetectedReason(name, ctx.SeamDetectedSlug, ctx.SeamDetectedFiles), ctx, mode)
	}
	if ctx.Blocked {
		return maybePromoted(CategoryBlocked, fmt.Sprintf("%s is blocked on an unmet dependency", name), ctx, mode)
	}
	if ctx.DesignFork {
		return maybePromoted(CategoryDesignFork, fmt.Sprintf("%s has a real design fork — your decision", name), ctx, mode)
	}
	if ctx.NextScore >= 0 && ctx.NextScore < ctx.ScoreThreshold {
		return maybePromoted(CategoryUnderspecified,
			fmt.Sprintf("%s scores %d (below the %d threshold) — likely underspecified", name, ctx.NextScore, ctx.ScoreThreshold), ctx, mode)
	}
	if ctx.AmbiguousPick {
		return maybePromoted(CategoryAmbiguousPick, "the next pick is ambiguous between specs of different intent", ctx, mode)
	}

	// Proceed-silently set: ready, scored, deps met, verify not stuck.
	return proceed()
}

// maybePromoted returns a pause unless the run is Autonomous, the category is
// Promotable, and the learning layer has promoted it — in which case the run
// proceeds. Guided always pauses here.
func maybePromoted(cat PauseCategory, reason string, ctx RunContext, mode AutonomyMode) Decision {
	if mode == Autonomous && cat.Promotable() && ctx.Promoted != nil && ctx.Promoted(cat) {
		return proceed()
	}
	return pause(cat, reason)
}
