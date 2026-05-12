// Package integrity implements graph-verified spec status checks —
// the structural answer to "specs that lied about being completed."
//
// A spec claiming `status: completed` is verified by checking every
// Criterion node belonging to that spec has status `passing`. The
// possible verdicts (verified / lying / partial / unverifiable) plus
// the per-AC breakdown drive `hero check status`, the one-line
// summary in `hero check`, and the auto-downgrade tooling that
// arrives in Phase 3.
package integrity

import (
	"sort"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/spec"
)

// Verdict is the outcome of checking a single completed spec.
type Verdict string

const (
	// VerdictVerified — spec claims completed; all ACs pass.
	VerdictVerified Verdict = "verified"
	// VerdictLying — spec claims completed; at least one AC is
	// failing or regressed (concrete evidence of breakage).
	VerdictLying Verdict = "lying"
	// VerdictPartial — spec claims completed; some ACs pass, others
	// are still proposed/unknown (no concrete failure, but no
	// evidence the work is done).
	VerdictPartial Verdict = "partial"
	// VerdictUnverifiable — spec claims completed; the spec has no
	// Criterion nodes at all. Cannot judge either way.
	VerdictUnverifiable Verdict = "unverifiable"
)

// Finding describes the integrity check result for one spec.
type Finding struct {
	Slug    string
	Title   string
	Path    string
	Verdict Verdict
	// ACBreakdown counts criteria by current status so callers can
	// render concise summaries ("2 of 7 ACs failing").
	Total          int
	Passing        int
	Failing        int
	Regressed      int
	ProposedOrOpen int // proposed + unknown + retired
	// FailingKeys lists the Criterion keys that hold up the verdict
	// (failing or regressed). Empty for verified / partial / unverifiable.
	FailingKeys []string
	// OpenKeys lists Criterion keys still in proposed/unknown — the
	// "no evidence of pass yet" set. Drives the partial verdict.
	OpenKeys []string
}

// Report is the workspace-wide rollup.
type Report struct {
	Findings []Finding

	// Aggregated counts across all completed specs.
	Verified     int
	Lying        int
	Partial      int
	Unverifiable int
}

// HasIssues reports whether the report contains any spec that's
// lying or partial — the cases that warrant downgrade. Unverifiable
// specs aren't issues per se (they predate the AC graph).
func (r Report) HasIssues() bool {
	return r.Lying > 0 || r.Partial > 0
}

// Total returns the count of completed specs evaluated.
func (r Report) Total() int {
	return r.Verified + r.Lying + r.Partial + r.Unverifiable
}

// CheckCompletedSpecs returns one Finding per spec whose status is
// `completed`, plus aggregate counters. Skips non-completed specs
// (their status is allowed to be whatever).
//
// The verifier consumes the AC graph from
// `acceptance-criteria-graph` Phase 1+2; if that hasn't run yet (or
// the spec has no `## Acceptance criteria` block), the verdict is
// `unverifiable` rather than an error — graph-less specs predate
// this verification layer.
func CheckCompletedSpecs(specs []*spec.Spec, store *graph.Store) (*Report, error) {
	report := &Report{}
	for _, s := range specs {
		if s.Status != spec.StatusCompleted {
			continue
		}
		f, err := verifySpec(s, store)
		if err != nil {
			return nil, err
		}
		report.Findings = append(report.Findings, f)
		switch f.Verdict {
		case VerdictVerified:
			report.Verified++
		case VerdictLying:
			report.Lying++
		case VerdictPartial:
			report.Partial++
		case VerdictUnverifiable:
			report.Unverifiable++
		}
	}
	sort.SliceStable(report.Findings, func(i, j int) bool {
		// Stable order: lying first (loudest signal), then partial,
		// then unverifiable, then verified.
		return verdictRank(report.Findings[i].Verdict) < verdictRank(report.Findings[j].Verdict) ||
			(verdictRank(report.Findings[i].Verdict) == verdictRank(report.Findings[j].Verdict) &&
				report.Findings[i].Slug < report.Findings[j].Slug)
	})
	return report, nil
}

func verdictRank(v Verdict) int {
	switch v {
	case VerdictLying:
		return 0
	case VerdictPartial:
		return 1
	case VerdictUnverifiable:
		return 2
	case VerdictVerified:
		return 3
	}
	return 4
}

func verifySpec(s *spec.Spec, store *graph.Store) (Finding, error) {
	f := Finding{
		Slug:  s.Slug,
		Title: s.Title,
		Path:  s.Path,
	}
	criteria, err := acceptance.ListBySpec(store, s.Slug)
	if err != nil {
		return f, err
	}
	f.Total = len(criteria)
	if f.Total == 0 {
		f.Verdict = VerdictUnverifiable
		return f, nil
	}
	for _, c := range criteria {
		switch c.Status {
		case "passing":
			f.Passing++
		case "failing":
			f.Failing++
			f.FailingKeys = append(f.FailingKeys, c.Key)
		case "regressed":
			f.Regressed++
			f.FailingKeys = append(f.FailingKeys, c.Key)
		default:
			f.ProposedOrOpen++
			f.OpenKeys = append(f.OpenKeys, c.Key)
		}
	}
	switch {
	case f.Failing > 0 || f.Regressed > 0:
		f.Verdict = VerdictLying
	case f.Passing == f.Total:
		f.Verdict = VerdictVerified
	default:
		f.Verdict = VerdictPartial
	}
	return f, nil
}

// SuggestStatus maps a verdict to the spec status the auto-fix would
// downgrade to. Verified specs stay completed.
func SuggestStatus(v Verdict) spec.Status {
	switch v {
	case VerdictLying:
		return spec.StatusPlanning
	case VerdictPartial:
		return spec.StatusDelivering
	default:
		return ""
	}
}
