package drive

import (
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/score"
	"github.com/hero-engine/hero/internal/spec"
)

// PauseInfo is the pause payload of a CheckResult.
type PauseInfo struct {
	Category string `json:"category"`
	Reason   string `json:"reason"`
}

// CheckResult is the per-turn verdict `hero goal --check` emits — the
// contract the harness Stop hook consumes. Derived from on-disk state only,
// so a cold process produces the same verdict.
type CheckResult struct {
	Verdict    string     `json:"verdict"`          // "continue" | "pause" | "done"
	Action     string     `json:"action,omitempty"` // on continue: "design" | "deliver"
	Initiative string     `json:"initiative"`
	NextSpec   string     `json:"next_spec,omitempty"`
	Kickoff    string     `json:"kickoff,omitempty"`
	Pause      *PauseInfo `json:"pause,omitempty"`
	Remaining  []string   `json:"remaining"`
	Completed  []string   `json:"completed"`
}

func isCompleted(s *spec.Spec) bool {
	return s.Status == spec.StatusCompleted || s.Status == spec.StatusSuperseded
}

// Children returns an initiative's child specs (those declaring a `parent`
// relation targeting init.Slug), sorted by slug for deterministic order.
func Children(init *spec.Spec, all []*spec.Spec) []*spec.Spec {
	var kids []*spec.Spec
	for _, s := range all {
		if s == nil || s.Slug == init.Slug {
			continue
		}
		for _, r := range s.Relations {
			if r.Kind == "parent" && r.Target == init.Slug {
				kids = append(kids, s)
				break
			}
		}
	}
	sort.Slice(kids, func(i, j int) bool { return kids[i].Slug < kids[j].Slug })
	return kids
}

func depsMet(s *spec.Spec, completed map[string]bool) bool {
	for _, r := range s.Relations {
		if r.Kind == "depends-on" || r.Kind == "depends_on" {
			if !completed[r.Target] {
				return false
			}
		}
	}
	return true
}

// rank maps a hero priority/severity string to a total, case-insensitive
// ordering key: critical=0, high=1, medium=2, low=3, unset/unknown last (99).
// Shared by priority and severity so candidate selection is a pure function of
// on-disk state — no reliance on map iteration or input order.
func rank(v string) int {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 99
	}
}

// lessCandidate orders two ready candidates by (priorityRank, severityRank,
// slug). Slug is the final deterministic tiebreak so ties never depend on
// iteration order. Missing priority/severity (including spec == nil stubs)
// rank last via the empty string.
func lessCandidate(a, b *intended) bool {
	if ra, rb := rank(a.priority()), rank(b.priority()); ra != rb {
		return ra < rb
	}
	if ra, rb := rank(a.severity()), rank(b.severity()); ra != rb {
		return ra < rb
	}
	return a.slug < b.slug
}

// conflictDeliveringSlug returns the slug of a conflicts-with target of ic
// that is currently delivering locally, or "" if none. This is the soft-mutex
// gate: a child is not selectable while a spec it conflicts with is in flight.
// Reads only on-disk status via IsLocallyDelivering, so it stays
// deterministic. Dangling and self targets resolve to a no-op (never panic).
// v1 honors conflicts-with as authored (outbound only); inbound conflicts are
// a documented follow-up.
func conflictDeliveringSlug(ic *intended, all []*spec.Spec) string {
	if ic.spec == nil {
		return ""
	}
	for _, r := range ic.spec.Relations {
		if r.Kind != "conflicts-with" && r.Kind != "conflicts_with" {
			continue
		}
		if r.Target == ic.slug {
			continue // self-conflict is a no-op
		}
		if t := specBySlugDrive(all, r.Target); t != nil && t.IsLocallyDelivering() {
			return t.Slug
		}
	}
	return ""
}

// completedSet returns the slugs of all specs in a completed/superseded
// state. Children archive to .hero/specs/ on completion but remain
// discoverable, so this reflects real verify-gated progress.
func completedSet(all []*spec.Spec) map[string]bool {
	done := map[string]bool{}
	for _, s := range all {
		if isCompleted(s) {
			done[s.Slug] = true
		}
	}
	return done
}

// scoreFn computes a spec's readiness score. Overridable in tests so stage
// classification can be driven deterministically off structure.
var scoreFn = func(s *spec.Spec) int {
	if s == nil {
		return -1
	}
	return score.Score(s, score.DefaultConfig()).Score
}

// intended is one child the initiative intends to ship — either a discovered
// spec or a declared-but-unscaffolded slug (spec == nil).
type intended struct {
	slug string
	spec *spec.Spec
}

func (ic *intended) stage(completed map[string]bool) Stage {
	if completed[ic.slug] {
		return StageDone
	}
	if ic.spec == nil {
		return StageNeedsScaffold
	}
	return ChildStage(ic.spec, scoreFn(ic.spec))
}

func (ic *intended) ready(completed map[string]bool) bool {
	if ic.spec == nil {
		return true // a declared stub has no deps to satisfy yet
	}
	return depsMet(ic.spec, completed)
}

// priority returns the child's declared priority, or "" for a needs-scaffold
// stub (spec == nil) — which ranks last in selection.
func (ic *intended) priority() string {
	if ic.spec == nil {
		return ""
	}
	return ic.spec.Priority
}

// severity returns the child's declared severity, or "" for a stub.
func (ic *intended) severity() string {
	if ic.spec == nil {
		return ""
	}
	return ic.spec.Severity
}

// buildIntended is the authoritative child set: discovered children (via
// `parent` relations) unioned with the slugs the initiative declares in its
// child table. A declared slug with a spec on disk attaches it; one without
// becomes a needs-scaffold entry — so the run can't short-circuit to done.
func buildIntended(init *spec.Spec, all []*spec.Spec) []intended {
	seen := map[string]bool{}
	var out []intended
	for _, k := range Children(init, all) {
		seen[k.Slug] = true
		out = append(out, intended{slug: k.Slug, spec: k})
	}
	for _, slug := range declaredChildSlugs(init) {
		if seen[slug] {
			continue
		}
		seen[slug] = true
		out = append(out, intended{slug: slug, spec: specBySlugDrive(all, slug)})
	}
	return out
}

func specBySlugDrive(all []*spec.Spec, slug string) *spec.Spec {
	for _, s := range all {
		if s != nil && s.Slug == slug {
			return s
		}
	}
	return nil
}

// Check computes the verdict for one turn of an initiative run. It classifies
// each intended child's design→deliver stage, ANDs verify-status with the
// needs_me boundary, and returns the next ACTION (design or deliver) — all
// from on-disk state.
//
// Progressive design: an undesigned child routes to `design` (autonomously),
// never to delivery; the run is not `done` while any intended child —
// including declared-but-unscaffolded ones — is unfinished. A low score means
// "design it," not "pause"; only a genuine design fork (raised by the design
// step) pauses.
//
// promoted is the learning hook (nil = no promotions), consulted only in
// Autonomous mode for Promotable categories.
func Check(init *spec.Spec, all []*spec.Spec, promoted func(PauseCategory) bool) CheckResult {
	mode := ParseMode(init.Autonomy)
	completed := completedSet(all)
	res := CheckResult{Verdict: "done", Initiative: init.Slug}

	intendeds := buildIntended(init, all)
	if len(intendeds) == 0 {
		if isCompleted(init) {
			return res // done
		}
		res.Verdict = "pause"
		res.Pause = &PauseInfo{Category: string(CategoryBlocked), Reason: "initiative has no child specs to run"}
		return res
	}

	// Completed/Remaining reporting stays slug-stable (intendeds preserves
	// Children()/buildIntended order). Priority ordering below is scoped to
	// selection of the next candidate only — it does not reorder these lists.
	var firstRem *intended
	var ready []*intended
	for i := range intendeds {
		ic := &intendeds[i]
		if ic.stage(completed) == StageDone {
			res.Completed = append(res.Completed, ic.slug)
			continue
		}
		res.Remaining = append(res.Remaining, ic.slug)
		if firstRem == nil {
			firstRem = ic
		}
		// A child is selectable when its deps are met AND no conflicts-with
		// target is currently delivering (soft-mutex seam collision).
		if ic.ready(completed) && conflictDeliveringSlug(ic, all) == "" {
			ready = append(ready, ic)
		}
	}

	if len(res.Remaining) == 0 {
		return res // every intended child finished → done
	}

	// Among selectable candidates, pick the min under (priority, severity,
	// slug). Slug is the final tiebreak, keeping the choice reproducible.
	var nextI *intended
	var nextStage Stage
	for _, ic := range ready {
		if nextI == nil || lessCandidate(ic, nextI) {
			nextI = ic
		}
	}
	if nextI != nil {
		nextStage = nextI.stage(completed)
	}

	ctx := RunContext{ActionClassified: true, NextScore: -1, Promoted: promoted}
	if nextI == nil {
		// Nothing selectable. Distinguish the fallback reason off the
		// first-remaining child: ready-on-deps but conflict-excluded is a
		// SeamCollision (name the in-flight spec); otherwise it's blocked on
		// an unmet dependency, as today.
		nextI, nextStage = firstRem, firstRem.stage(completed)
		if firstRem.ready(completed) {
			if conflict := conflictDeliveringSlug(firstRem, all); conflict != "" {
				ctx.SeamBlocked = true
				ctx.SeamConflictSlug = conflict
			} else {
				ctx.Blocked = true
			}
		} else {
			ctx.Blocked = true
		}
	}

	res.NextSpec = nextI.slug
	dec := NeedsMe(nextI.spec, ctx, mode)
	if !dec.Proceed {
		res.Verdict = "pause"
		res.Pause = &PauseInfo{Category: string(dec.Category), Reason: dec.Reason}
		return res
	}
	res.Verdict = "continue"
	res.Action = ActionForStage(nextStage)
	if res.Action == ActionDeliver && nextI.spec != nil {
		res.Kickoff = nextI.spec.Kickoff()
	}
	return res
}

// DryStep is one simulated transition in a DryRun preview.
type DryStep struct {
	Step     int    `json:"step"`
	Verdict  string `json:"verdict"`
	Action   string `json:"action,omitempty"`
	Spec     string `json:"spec,omitempty"`
	Category string `json:"category,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// DryRun previews up to n transitions Check WOULD take, optimistically
// assuming each "continue" child then finishes. Stops on a pause or done.
func DryRun(init *spec.Spec, all []*spec.Spec, n int, promoted func(PauseCategory) bool) []DryStep {
	mode := ParseMode(init.Autonomy)
	intendeds := buildIntended(init, all)
	completed := completedSet(all)

	var steps []DryStep
	for i := 1; i <= n; i++ {
		var firstRem *intended
		var ready []*intended
		allDone := true
		for j := range intendeds {
			ic := &intendeds[j]
			if ic.stage(completed) == StageDone {
				continue
			}
			allDone = false
			if firstRem == nil {
				firstRem = ic
			}
			if ic.ready(completed) && conflictDeliveringSlug(ic, all) == "" {
				ready = append(ready, ic)
			}
		}
		if allDone {
			steps = append(steps, DryStep{Step: i, Verdict: "done"})
			break
		}
		// Same priority-aware, conflict-excluding selection as Check, so the
		// preview matches real behavior. Deterministic (slug final tiebreak).
		var nextI *intended
		var nextStage Stage
		for _, ic := range ready {
			if nextI == nil || lessCandidate(ic, nextI) {
				nextI = ic
			}
		}
		if nextI != nil {
			nextStage = nextI.stage(completed)
		}
		if nextI == nil {
			if firstRem.ready(completed) {
				if conflict := conflictDeliveringSlug(firstRem, all); conflict != "" {
					steps = append(steps, DryStep{Step: i, Verdict: "pause", Category: string(CategorySeamCollision), Reason: seamReason(firstRem.slug, conflict)})
					break
				}
			}
			steps = append(steps, DryStep{Step: i, Verdict: "pause", Category: string(CategoryBlocked), Reason: "remaining children are blocked on dependencies"})
			break
		}
		ctx := RunContext{ActionClassified: true, NextScore: -1, Promoted: promoted}
		dec := NeedsMe(nextI.spec, ctx, mode)
		ds := DryStep{Step: i, Spec: nextI.slug}
		if !dec.Proceed {
			ds.Verdict = "pause"
			ds.Category = string(dec.Category)
			ds.Reason = dec.Reason
			steps = append(steps, ds)
			break
		}
		ds.Verdict = "continue"
		ds.Action = ActionForStage(nextStage)
		steps = append(steps, ds)
		completed[nextI.slug] = true // simulate this child finishing
	}
	return steps
}
