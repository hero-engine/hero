// Package metrics computes the canonical ROI metric set the People & ROI
// home renders. The same package is intended to be reused by the
// executive-report skill and any future `hero roi` CLI so the formula
// lives in exactly one place.
//
// All functions are nil-safe by design: an empty events log, missing
// hero dir, or absent propose store yields zero/empty values, never an
// error. The People home calls these on every render — they must not
// panic and must degrade gracefully.
//
// Scope: this is the v1 shape needed for the ROI Overview's headline
// tiles (Money chain) plus the basic Throughput/Quality scalars. The
// full per-section family files (autonomy.go, velocity.go, …) called
// out in the spec are intentionally folded into a single file here to
// keep the surface small and verifiable; later splits can carve out
// pieces without changing the public API.
package metrics

import (
	"path/filepath"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

// Window selects a time range for a computation.
type Window struct {
	Since time.Time
	Until time.Time // zero means "now"
}

// Last returns a Window covering the trailing duration ending now.
func Last(d time.Duration) Window {
	now := time.Now().UTC()
	return Window{Since: now.Add(-d), Until: now}
}

// Coefficients holds the configurable knobs that turn raw counts into
// the saved-hours estimator output. Defaults are conservative per the
// spec.
type Coefficients struct {
	NoEdit         float64 // c_no_edit
	WithEdit       float64 // c_with_edit
	ImportTriage   float64 // c_import_triage
	Diagnosis      float64 // c_diagnosis
	DesignReview   float64 // c_design_review
	ContextLookup  float64 // c_context_lookup
	ScheduledRun   float64 // c_scheduled_run
	HourlyCost     float64 // c_hourly_cost ($/h loaded)
}

// DefaultCoefficients are the spec's published defaults. Per the spec
// these are calibrated per workspace via hero.json; the People home
// loads from config and falls back here when unset.
func DefaultCoefficients() Coefficients {
	return Coefficients{
		NoEdit:        1.5,
		WithEdit:      0.5,
		ImportTriage:  0.1,
		Diagnosis:     1.0,
		DesignReview:  0.5,
		ContextLookup: 0.05,
		ScheduledRun:  0.25,
		HourlyCost:    150.0,
	}
}

// Counts is the substrate the saved-hours estimator multiplies through.
// Each field is a window-scoped count derived from .hero/events.log,
// the propose store, or the scheduled-tasks log. Zero values are valid
// and contribute zero to the estimator.
type Counts struct {
	ProposalsMergedNoEdit    int
	ProposalsMergedWithEdit  int
	AutoImportedSpecs        int
	AutoDiagnosedBugs        int
	AutoReviewedSpecs        int
	KnowledgeInjections      int
	ScheduledJobsExecuted    int

	// SpecsDelivered is the count of `delivery_complete` events in the
	// window — fed into the Throughput tab and the page-hero subhead.
	SpecsDelivered int
}

// HoursSaved applies the spec's published formula. Returns 0 when the
// counts are all zero (empty workspace).
func HoursSaved(c Counts, k Coefficients) float64 {
	return float64(c.ProposalsMergedNoEdit)*k.NoEdit +
		float64(c.ProposalsMergedWithEdit)*k.WithEdit +
		float64(c.AutoImportedSpecs)*k.ImportTriage +
		float64(c.AutoDiagnosedBugs)*k.Diagnosis +
		float64(c.AutoReviewedSpecs)*k.DesignReview +
		float64(c.KnowledgeInjections)*k.ContextLookup +
		float64(c.ScheduledJobsExecuted)*k.ScheduledRun
}

// DollarsSaved converts hours into the team's loaded-cost dollar value.
func DollarsSaved(hours float64, k Coefficients) float64 {
	return hours * k.HourlyCost
}

// NetValue is dollars saved minus the Hero API spend in the window.
func NetValue(hoursSaved float64, apiSpend float64, k Coefficients) float64 {
	return DollarsSaved(hoursSaved, k) - apiSpend
}

// ROIMultiple is net value divided by API spend. Per the spec, zero API
// spend returns a sentinel (math.Inf) so the renderer can show "—";
// callers normalise via FormatROIMultiple.
func ROIMultiple(netValue, apiSpend float64) float64 {
	if apiSpend <= 0 {
		// Sentinel — render layer turns this into "—".
		return 0
	}
	return netValue / apiSpend
}

// LoadCounts reads .hero/events.log under heroDir and tallies the
// event types we know how to credit. Empty heroDir or missing log
// returns zero counts (not an error).
//
// Mapping (best-effort, conservative):
//   - delivery_complete     → SpecsDelivered (one per merged spec)
//   - spec_created          → AutoImportedSpecs (proxy: every created
//                             spec counts as one import-triage save)
//   - decision_made         → AutoDiagnosedBugs (proxy for design-time
//                             decisions captured via /diagnose or
//                             /decide)
//
// Proposal-related counts (ProposalsMergedNoEdit / WithEdit) and
// KnowledgeInjections require log sources that don't yet exist in
// .hero/events.log — they default to zero here and will be wired when
// the propose-lifecycle log + context-injection log land in their own
// specs. The hours-saved estimator returns zero contribution for those
// terms in the meantime, which matches the spec's empty-state behavior.
func LoadCounts(heroDir string, w Window) Counts {
	if heroDir == "" {
		return Counts{}
	}
	logPath := filepath.Join(heroDir, "events.log")
	evts, err := feed.ReadEvents(logPath, feed.Filter{Since: w.Since})
	if err != nil {
		return Counts{}
	}
	var c Counts
	until := w.Until
	for _, e := range evts {
		if !until.IsZero() && e.Timestamp.After(until) {
			continue
		}
		switch e.Type {
		case "delivery_complete":
			c.SpecsDelivered++
		case "spec_created":
			c.AutoImportedSpecs++
		case "decision_made":
			c.AutoDiagnosedBugs++
		}
	}
	return c
}

// PendingProposalCount is a tiny helper for the page-hero subhead: the
// number of proposals still awaiting human action across the supplied
// snapshot. The snapshot is provided by the serve layer (it owns the
// store); here we just consume a count to keep this package free of a
// runtime dep on internal/propose.
func PendingProposalCount(snapshotCount int) int {
	if snapshotCount < 0 {
		return 0
	}
	return snapshotCount
}
