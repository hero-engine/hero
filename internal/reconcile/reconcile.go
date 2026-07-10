// Package reconcile compares spec status fields against git evidence
// and produces findings about status drift.
package reconcile

import (
	"strings"

	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/spec"
)

// FindingKind classifies the repair a finding calls for, so the applier
// (`hero check --reconcile`) can pick the right action rather than inferring
// it from the status pair alone.
type FindingKind string

const (
	// FindingStatusDrift is a git-evidence promotion (planning → delivering).
	FindingStatusDrift FindingKind = "status-drift"
	// FindingCompletedStuck is a completed spec still living under planning/
	// that should be moved to specs/.
	FindingCompletedStuck FindingKind = "completed-stuck"
	// FindingInitiativeComplete is an initiative whose every declared child is
	// already completed; it should be completed and archived even though no
	// child was verified in the current process.
	FindingInitiativeComplete FindingKind = "initiative-complete"
	// FindingOrphanCompletedAt is a spec carrying a `completed_at:` stamp while
	// its status is not completed; the orphaned timestamp should be cleared.
	FindingOrphanCompletedAt FindingKind = "orphan-completed-at"
)

// Finding represents a single status mismatch between a spec's declared
// status and what git evidence (or its child roster) suggests it should be.
type Finding struct {
	Kind            FindingKind
	Spec            *spec.Spec
	CurrentStatus   spec.Status
	SuggestedStatus spec.Status
	Evidence        string
}

// CanAutoFix returns true if this finding can be safely auto-fixed.
// We auto-promote planning → delivering, auto-move completed specs out of
// planning, complete initiatives whose children are all done, and clear
// orphaned completed_at stamps.
func (f Finding) CanAutoFix() bool {
	switch f.Kind {
	case FindingInitiativeComplete, FindingOrphanCompletedAt:
		return true
	}
	if f.CurrentStatus == spec.StatusPlanning && f.SuggestedStatus == spec.StatusDelivering {
		return true
	}
	// Completed specs stuck in planning/ should be moved
	if f.CurrentStatus == spec.StatusCompleted && f.SuggestedStatus == spec.StatusCompleted {
		return true
	}
	return false
}

// Reconcile compares work specs against git evidence and returns findings
// for specs whose status appears to be out of date.
//
// Only work specs (features, bugs, initiatives) with FilesTouched are checked.
// Knowledge specs are skipped — they don't follow the planning→delivering→completed
// lifecycle.
//
// The function never modifies any files. Callers decide what to do with findings.
func Reconcile(heroDir, projectRoot string) []Finding {
	if !gitutil.IsRepo(projectRoot) {
		return nil
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil
	}

	// Get all changed files (branch + uncommitted) once
	changedFiles := gitutil.AllChangedFiles(projectRoot)
	changedSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	var findings []Finding

	for _, s := range specs {
		// Only check work specs
		if !s.IsWorkSpec() && s.Type != spec.TypeInitiative {
			continue
		}

		// Standalone initiative completion re-check: an initiative whose every
		// declared child is already a completed, archived spec should be
		// completed and archived even though no child was verified in this
		// process. This is the idempotent recovery path for the one-shot
		// in-process trigger in `hero spec verify`. Initiatives only — never
		// auto-complete a leaf feature/bug from git or roster evidence.
		if s.Type == spec.TypeInitiative &&
			(s.Status == spec.StatusPlanning || s.Status == spec.StatusDelivering) &&
			spec.InitiativeReadyToComplete(s, specs) {
			findings = append(findings, Finding{
				Kind:            FindingInitiativeComplete,
				Spec:            s,
				CurrentStatus:   s.Status,
				SuggestedStatus: spec.StatusCompleted,
				Evidence:        "every declared child is completed — initiative should be completed and archived",
			})
			continue
		}

		// Invariant: a spec must not carry a completed_at stamp while its
		// status is not completed. When the roster re-check above completes an
		// initiative, the completed-write reconciles the pair; anything left
		// here is a genuine split (e.g. a manual reopen) whose orphaned
		// timestamp should be cleared.
		if !s.CompletedAt.IsZero() && s.Status != spec.StatusCompleted {
			findings = append(findings, Finding{
				Kind:            FindingOrphanCompletedAt,
				Spec:            s,
				CurrentStatus:   s.Status,
				SuggestedStatus: s.Status,
				Evidence:        "completed_at is set but status is not completed — orphaned timestamp should be cleared",
			})
			continue
		}

		// Check for completed specs stuck in planning/
		if s.Status == spec.StatusCompleted {
			if isInPlanning(s.Path) {
				findings = append(findings, Finding{
					Kind:            FindingCompletedStuck,
					Spec:            s,
					CurrentStatus:   spec.StatusCompleted,
					SuggestedStatus: spec.StatusCompleted,
					Evidence:        "status is completed but spec is still in planning/ — should be moved to specs/",
				})
			}
			continue
		}

		// Skip specs with no files listed — we can't do git-based inference
		if len(s.FilesTouched) == 0 {
			continue
		}

		finding := checkSpec(s, projectRoot, changedSet)
		if finding != nil {
			findings = append(findings, *finding)
		}
	}

	return findings
}

// checkSpec evaluates a single spec against git evidence.
func checkSpec(s *spec.Spec, projectRoot string, changedFiles map[string]bool) *Finding {
	// Count how many of the spec's files have been changed
	matchCount := 0
	for _, f := range s.FilesTouched {
		normalized := gitutil.NormalizeFilePath(projectRoot, f)
		if changedFiles[normalized] {
			matchCount++
		}
	}

	// Also check if spec has a claim (someone is working on it)
	hasClaim := s.ClaimedBy != ""

	switch s.Status {
	case spec.StatusPlanning:
		// If files have been touched or someone claimed it → should be delivering
		if matchCount > 0 {
			evidence := pluralize(matchCount, "file", "files") + " from the Changes section " + haveHas(matchCount) + " been modified"
			return &Finding{
				Kind:            FindingStatusDrift,
				Spec:            s,
				CurrentStatus:   s.Status,
				SuggestedStatus: spec.StatusDelivering,
				Evidence:        evidence,
			}
		}
		if hasClaim {
			return &Finding{
				Kind:            FindingStatusDrift,
				Spec:            s,
				CurrentStatus:   s.Status,
				SuggestedStatus: spec.StatusDelivering,
				Evidence:        "claimed by " + s.ClaimedBy + " but still in planning",
			}
		}

	case spec.StatusInReview:
		// If files changed → still active, probably should be delivering
		// But in-review → delivering is a demotion, so just note it
		if matchCount > 0 && !hasClaim {
			// Don't suggest demotion, but note the activity
			return nil
		}

	case spec.StatusDelivering:
		// Could check if all files are merged to default branch → suggest completed
		// But auto-complete is dangerous, so we skip this for now
		return nil
	}

	return nil
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return itoa(n) + " " + plural
}

func haveHas(n int) string {
	if n == 1 {
		return "has"
	}
	return "have"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// isInPlanning returns true if the spec path is under a planning/ directory.
func isInPlanning(path string) bool {
	return strings.Contains(path, "/planning/") || strings.Contains(path, "\\planning\\")
}
