package spec

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// HorizonProposal is one row of the bulk-migration plan: which
// horizon a heuristic suggests for an existing spec, plus the
// reasoning so the user can review the diff.
type HorizonProposal struct {
	Slug      string
	Path      string
	Current   Horizon // empty when unset
	Proposed  Horizon
	Reason    string  // human-readable why
	Skip      bool    // true when current already matches proposal
}

// PlanHorizonMigration applies a series of heuristics to every spec
// and returns one proposal per spec. Heuristics, in priority order:
//
//  1. Already has a valid horizon → skip (no proposal change).
//  2. status: completed → horizon: now (completed work is by
//     definition was-actively-now).
//  3. status: delivering / in-review → horizon: now.
//  4. Tags include marketing / sales / launch / positioning /
//     distribution / community / content → horizon: someday.
//     Lots of speculative future thinking lives here.
//  5. Slug contains those same prefixes → horizon: someday.
//  6. status: planning + recovery / get-back-on-track tag →
//     horizon: now.
//  7. Default: horizon: next (concrete, not yet started, but in the
//     queue).
//
// Returns proposals sorted by slug. Empty input returns nil.
func PlanHorizonMigration(specs []*Spec) []HorizonProposal {
	out := make([]HorizonProposal, 0, len(specs))
	for _, s := range specs {
		if s == nil || s.Slug == "" {
			continue
		}
		p := HorizonProposal{
			Slug:    s.Slug,
			Path:    s.Path,
			Current: s.Horizon,
		}
		p.Proposed, p.Reason = proposeHorizon(s)
		if p.Current == p.Proposed {
			p.Skip = true
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Slug < out[j].Slug
	})
	return out
}

// proposeHorizon is the heuristic core. Returns (horizon, reason).
func proposeHorizon(s *Spec) (Horizon, string) {
	if IsValidHorizon(s.Horizon) {
		return s.Horizon, "already set"
	}

	switch s.Status {
	case StatusCompleted:
		return HorizonNow, "status: completed (was active now-work)"
	case StatusDelivering, StatusInReview:
		return HorizonNow, fmt.Sprintf("status: %s (in flight)", s.Status)
	case StatusRegressed:
		return HorizonNow, "status: regressed (active issue)"
	}

	// Speculative-future tag/slug patterns. The recovery audit named
	// these as the noise drowning the actionable signal.
	speculative := []string{
		"marketing", "sales", "launch", "positioning",
		"distribution", "community", "content-engine",
		"landing-page", "demo-content", "playbook",
	}
	for _, t := range s.Tags {
		t = strings.ToLower(t)
		for _, kw := range speculative {
			if strings.Contains(t, kw) {
				return HorizonSomeday, "tag matches speculative-future ('" + kw + "')"
			}
		}
	}
	slug := strings.ToLower(s.Slug)
	for _, kw := range speculative {
		if strings.Contains(slug, kw) {
			return HorizonSomeday, "slug matches speculative-future ('" + kw + "')"
		}
	}

	// Recovery-tagged work that's still in planning is now-work.
	for _, t := range s.Tags {
		t = strings.ToLower(t)
		if strings.Contains(t, "v2-recovery") || strings.Contains(t, "recovery") {
			return HorizonNow, "tag indicates recovery work"
		}
	}

	return HorizonNext, "default — concrete, queued, not started"
}

// ApplyHorizonProposals writes the proposed horizon to each spec's
// frontmatter unless dryRun is true. Skipped proposals (current ==
// proposed) are no-ops. Returns (written, skipped, error).
func ApplyHorizonProposals(proposals []HorizonProposal, dryRun bool) (int, int, error) {
	written := 0
	skipped := 0
	for _, p := range proposals {
		if p.Skip {
			skipped++
			continue
		}
		if dryRun {
			written++
			continue
		}
		data, err := os.ReadFile(p.Path)
		if err != nil {
			return written, skipped, fmt.Errorf("read %s: %w", p.Path, err)
		}
		updated := SetFrontmatterField(string(data), "horizon", string(p.Proposed))
		if updated == string(data) {
			// SetFrontmatterField wasn't able to insert (no frontmatter
			// at all, e.g.). Skip rather than error out.
			skipped++
			continue
		}
		if err := os.WriteFile(p.Path, []byte(updated), 0o644); err != nil {
			return written, skipped, fmt.Errorf("write %s: %w", p.Path, err)
		}
		written++
	}
	return written, skipped, nil
}
