package drive

import (
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// Stage is where a child sits in the design→deliver lifecycle, so Drive can
// route it correctly: design it if it isn't designed, deliver it if it is,
// and never declare an initiative done while any child is unfinished.
type Stage int

const (
	StageDone          Stage = iota // completed / superseded
	StageReadyDeliver               // designed + adequately scored, not yet completed
	StageNeedsDesign                // discoverable but not yet designed (stub)
	StageNeedsScaffold              // declared by the initiative but no spec on disk
)

// DesignReadyThreshold is the score below which a discovered child is treated
// as still needing design. Matches the scorer's default deliverable bar.
const DesignReadyThreshold = 40

// Action is the per-turn instruction the harness acts on for a continue verdict.
const (
	ActionDesign  = "design"
	ActionDeliver = "deliver"
)

// ChildStage classifies a DISCOVERED child spec. score < 0 means "unknown"
// (skip the score signal and rely on structure). A child needs design when it
// lacks the structural mark of a real design (no `## Acceptance Criteria`) or
// scores below the readiness threshold — this is the progressive-design guard
// that keeps undesigned stubs out of delivery.
func ChildStage(child *spec.Spec, score int) Stage {
	if isCompleted(child) {
		return StageDone
	}
	if strings.TrimSpace(child.Sections["acceptance criteria"]) == "" {
		return StageNeedsDesign
	}
	if score >= 0 && score < DesignReadyThreshold {
		return StageNeedsDesign
	}
	return StageReadyDeliver
}

// ActionForStage maps a non-done stage to the harness action.
func ActionForStage(s Stage) string {
	if s == StageReadyDeliver {
		return ActionDeliver
	}
	return ActionDesign // needs-design and needs-scaffold both design first
}

// declaredChildSlugs returns the child slugs an initiative declares, via
// spec.DeclaredChildren — the same roster the completion gate
// (spec.InitiativeReadyToComplete) consumes. Sharing one function is what
// keeps `hero goal --check` and `hero spec verify` auto-completion from
// disagreeing about which children remain; when the two derived the roster
// from different sources, a check could report done while declared children
// were still unbuilt.
//
// This is the supplementary completeness signal: a child declared in
// frontmatter or the sequence table but not yet materialized on disk is
// needs-scaffold, not absent — so the run can't short-circuit to done. Parent
// relations remain the spine; this only adds declared-but-undiscovered
// children.
func declaredChildSlugs(init *spec.Spec) []string {
	return spec.DeclaredChildren(init)
}
