package drive

import (
	"regexp"
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

var childLinkRe = regexp.MustCompile(`\[([a-z0-9][a-z0-9-]*)\]\((?:\./)?([a-z0-9-]+)/spec\.md\)`)

// declaredChildSlugs parses the child slugs an initiative declares in its
// `## Child Specs` / sequence table (links of the form `[slug](slug/spec.md)`).
// This is the supplementary completeness signal: a child named in the table
// but not yet materialized on disk is needs-scaffold, not absent — so the run
// can't short-circuit to done. Parent relations remain the spine; this only
// adds declared-but-undiscovered children.
func declaredChildSlugs(init *spec.Spec) []string {
	body := init.Sections["child specs & sequence"]
	if body == "" {
		// Fall back to any section whose header starts with "child".
		for k, v := range init.Sections {
			if strings.HasPrefix(k, "child") {
				body = v
				break
			}
		}
	}
	if body == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range childLinkRe.FindAllStringSubmatch(body, -1) {
		slug := m[1]
		if slug == m[2] && !seen[slug] { // link text matches the folder slug
			seen[slug] = true
			out = append(out, slug)
		}
	}
	return out
}
