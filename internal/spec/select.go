package spec

import (
	"slices"
	"sort"
	"strings"
	"time"
)

// Filter narrows a population of specs by composable criteria. The
// zero value matches every spec, so callers only set the fields they
// care about. Selector applies the filter, then ranks the survivors.
type Filter struct {
	// Types limits to these spec types. Empty matches all.
	Types []Type
	// Statuses limits to these lifecycle states. Empty matches all open
	// (non-completed/-archived/-abandoned/-superseded) specs by default
	// when ExcludeClosedDefault is true.
	Statuses []Status
	// Horizons limits to these horizon values. Empty matches all.
	Horizons []Horizon
	// Tags requires the spec to carry every tag in this slice. Empty
	// matches all.
	Tags []string
	// Ready selects only specs eligible for `hero queue` — open status
	// and all hard dependencies completed. Mutually exclusive with
	// Blocked.
	Ready bool
	// Blocked selects specs with at least one unmet hard dependency.
	// Complement of Ready.
	Blocked bool
	// Pinned selects only specs with `pinned: true` in frontmatter.
	Pinned bool
	// MineUser, when non-empty, filters to specs touched by this git
	// user identifier. Touch is determined by ClaimedBy match for v1;
	// commit-author scanning is a follow-up.
	MineUser string
	// StaleDays selects specs with no modification activity for at
	// least this many days. Zero means no staleness filter.
	StaleDays int
	// ExcludeClosedDefault, when true, excludes completed / archived /
	// abandoned / superseded specs unless Statuses explicitly names one.
	// Default true (caller opts in to seeing closed specs).
	ExcludeClosedDefault bool
	// Subproject, when non-empty and not "all", restricts to specs whose
	// frontmatter `subproject:` field matches exactly. Empty or "all"
	// disables subproject filtering.
	Subproject string
}

// Sort describes how a filtered population is ordered.
type Sort string

const (
	// SortRecency — most recently modified first. Default.
	SortRecency Sort = "recency"
	// SortStatus — delivering > planning > others, then alpha.
	SortStatus Sort = "status"
	// SortAlpha — alphabetical by slug.
	SortAlpha Sort = "alpha"
	// SortPriority — pinned > status > horizon > recency. The ranking
	// used by `hero queue`.
	SortPriority Sort = "priority"
)

// Selector applies a filter and sort over a population of specs and
// returns the result. The population is typically the output of
// spec.Discover. Selector is pure — it does no IO and no DB access —
// so the same logic powers `hero list`, `hero queue`, the MCP tools,
// and the QUEUE.md projection without ranking drift.
type Selector struct {
	Filter Filter
	Sort   Sort
	Limit  int // 0 means unlimited
	// Now is the reference time for staleness checks. Zero means
	// time.Now() at Apply time.
	Now time.Time
}

// Apply runs the selector against the given population and returns
// the surviving specs in ranked order.
func (sel Selector) Apply(all []*Spec) []*Spec {
	now := sel.Now
	if now.IsZero() {
		now = time.Now()
	}

	// Build a lookup from slug -> *Spec so we can resolve dependency
	// targets when the Ready / Blocked filters are applied.
	bySlug := make(map[string]*Spec, len(all))
	for _, s := range all {
		bySlug[s.Slug] = s
	}

	out := make([]*Spec, 0, len(all))
	for _, s := range all {
		if !filterMatches(s, sel.Filter, bySlug, now) {
			continue
		}
		out = append(out, s)
	}

	rankSpecs(out, sel.Sort)

	if sel.Limit > 0 && len(out) > sel.Limit {
		out = out[:sel.Limit]
	}
	return out
}

// IsReady reports whether s is eligible for `hero queue` — open
// status and all `depends-on` relations resolve to completed targets.
// Targets that don't resolve in the population are treated as missing
// (not blocking) since absent specs aren't a dependency the user can
// satisfy. Knowledge entries (notes, contexts, conventions, explainers,
// …) are never ready: they carry no delivery lifecycle, so the queue
// must not nag them for a `## Kickoff` section.
func IsReady(s *Spec, bySlug map[string]*Spec) bool {
	if s.IsKnowledge() || s.IsPreCommitment() {
		return false
	}
	if !isOpenStatus(s.Status) {
		return false
	}
	for _, rel := range s.Relations {
		if !isHardDependency(rel.Kind) {
			continue
		}
		target, ok := bySlug[rel.Target]
		if !ok {
			continue
		}
		if !isClosedStatus(target.Status) {
			return false
		}
	}
	return true
}

// IsBlocked is the complement of IsReady: at least one hard
// dependency is unmet.
func IsBlocked(s *Spec, bySlug map[string]*Spec) bool {
	if !isOpenStatus(s.Status) {
		return false
	}
	for _, rel := range s.Relations {
		if !isHardDependency(rel.Kind) {
			continue
		}
		target, ok := bySlug[rel.Target]
		if !ok {
			continue
		}
		if !isClosedStatus(target.Status) {
			return true
		}
	}
	return false
}

func filterMatches(s *Spec, f Filter, bySlug map[string]*Spec, now time.Time) bool {
	if len(f.Types) > 0 && !slices.Contains(f.Types, s.Type) {
		return false
	}
	if len(f.Statuses) > 0 {
		if !slices.Contains(f.Statuses, s.Status) {
			return false
		}
	} else if f.ExcludeClosedDefault && isClosedStatus(s.Status) {
		return false
	}
	if len(f.Horizons) > 0 && !slices.Contains(f.Horizons, s.EffectiveHorizon()) {
		return false
	}
	if len(f.Tags) > 0 {
		for _, want := range f.Tags {
			if !slices.Contains(s.Tags, want) {
				return false
			}
		}
	}
	if f.Pinned && !s.Pinned {
		return false
	}
	if f.MineUser != "" && !strings.EqualFold(s.ClaimedBy, f.MineUser) {
		return false
	}
	if f.StaleDays > 0 {
		cutoff := now.AddDate(0, 0, -f.StaleDays)
		ref := s.ModifiedAt
		if ref.IsZero() {
			ref = s.CreatedAt
		}
		if !ref.Before(cutoff) {
			return false
		}
	}
	if f.Ready && !IsReady(s, bySlug) {
		return false
	}
	if f.Blocked && !IsBlocked(s, bySlug) {
		return false
	}
	if f.Subproject != "" && f.Subproject != "all" && s.Subproject != f.Subproject {
		return false
	}
	return true
}

func rankSpecs(specs []*Spec, sortKey Sort) {
	switch sortKey {
	case SortAlpha:
		sort.SliceStable(specs, func(i, j int) bool {
			return specs[i].Slug < specs[j].Slug
		})
	case SortStatus:
		sort.SliceStable(specs, func(i, j int) bool {
			si, sj := statusRank(specs[i].Status), statusRank(specs[j].Status)
			if si != sj {
				return si < sj
			}
			return specs[i].Slug < specs[j].Slug
		})
	case SortPriority:
		sort.SliceStable(specs, func(i, j int) bool {
			return priorityLess(specs[i], specs[j])
		})
	case SortRecency, "":
		sort.SliceStable(specs, func(i, j int) bool {
			return refTime(specs[i]).After(refTime(specs[j]))
		})
	}
}

// priorityLess implements the `hero queue` ranking: pinned first,
// then status (delivering > planning > others), then horizon
// (now > next > later), then recency.
func priorityLess(a, b *Spec) bool {
	if a.Pinned != b.Pinned {
		return a.Pinned
	}
	if sa, sb := statusRank(a.Status), statusRank(b.Status); sa != sb {
		return sa < sb
	}
	if ha, hb := horizonRank(a.EffectiveHorizon()), horizonRank(b.EffectiveHorizon()); ha != hb {
		return ha < hb
	}
	return refTime(a).After(refTime(b))
}

// statusRank — lower is higher priority. Open work-spec statuses
// rank ahead of knowledge specs which rank ahead of closed states.
func statusRank(s Status) int {
	switch s {
	case StatusDelivering:
		return 0
	case StatusInReview:
		return 1
	case StatusPlanning:
		return 2
	case StatusRegressed:
		return 3
	case StatusActive, StatusAccepted:
		return 4
	case StatusDraft, StatusProposed:
		return 5
	case StatusCompleted:
		return 9
	case StatusSuperseded:
		return 10
	}
	return 6
}

func horizonRank(h Horizon) int {
	switch h {
	case HorizonNow:
		return 0
	case HorizonNext:
		return 1
	case HorizonSomeday:
		return 2
	case HorizonParking:
		return 3
	}
	return 4
}

func refTime(s *Spec) time.Time {
	if !s.ModifiedAt.IsZero() {
		return s.ModifiedAt
	}
	return s.CreatedAt
}

// isOpenStatus reports whether the spec is in an open / actionable
// state for queue purposes. Closed statuses are excluded from the
// ready set.
func isOpenStatus(s Status) bool {
	return !isClosedStatus(s)
}

// isClosedStatus reports whether the spec is in a terminal or
// non-actionable lifecycle state.
func isClosedStatus(s Status) bool {
	switch s {
	case StatusCompleted, StatusSuperseded:
		return true
	}
	return false
}

// isHardDependency reports whether a relation kind blocks delivery
// when its target is not yet closed. The set is conservative: only
// the kinds with a clear "this must be done first" semantic count.
func isHardDependency(kind string) bool {
	switch kind {
	case "depends-on", "depends_on", "blocked-by", "blocked_by":
		return true
	}
	return false
}
