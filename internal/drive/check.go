package drive

import (
	"sort"

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
	Verdict    string     `json:"verdict"` // "continue" | "pause" | "done"
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

// Check computes the verdict for one turn of an initiative run. It ANDs the
// children's verify-status (completed == verify-gated) with the needs_me
// boundary, deriving everything from on-disk state.
//
// v1 signal scope (hero-goal-command): mode, child verify-status, and
// dependency-readiness drive the verdict. Richer needs_me signals (readiness
// score, design-fork, irreversible-action, verify-stuck fail counts) await
// their detectors / the run-ledger in later specs; until then they default
// to "unknown → safe" and simply do not fire.
// promoted is the learning hook: reports whether a pause-category has been
// promoted to auto-proceed for the current user (nil = no promotions).
// Consulted only in Autonomous mode and only for Promotable categories.
func Check(init *spec.Spec, all []*spec.Spec, promoted func(PauseCategory) bool) CheckResult {
	mode := ParseMode(init.Autonomy)
	kids := Children(init, all)
	completed := completedSet(all)
	res := CheckResult{Verdict: "done", Initiative: init.Slug}

	var pending []*spec.Spec
	for _, k := range kids {
		if isCompleted(k) {
			res.Completed = append(res.Completed, k.Slug)
		} else {
			res.Remaining = append(res.Remaining, k.Slug)
			pending = append(pending, k)
		}
	}

	if len(kids) == 0 {
		// No children: the run condition is the initiative itself verifying.
		if isCompleted(init) {
			return res // done
		}
		res.Verdict = "pause"
		res.Pause = &PauseInfo{Category: string(CategoryBlocked), Reason: "initiative has no child specs to run"}
		return res
	}
	if len(pending) == 0 {
		return res // every child verified → done
	}

	verdict, next, pause := step(pending, completed, mode, promoted)
	res.Verdict = verdict
	if next != nil {
		res.NextSpec = next.Slug
		if verdict == "continue" {
			res.Kickoff = next.Kickoff()
		}
	}
	res.Pause = pause
	return res
}

// step picks the next ready pending child and runs needs_me against it,
// returning the verdict, the chosen spec, and a pause payload when paused.
// Shared by Check (one turn) and DryRun (simulated turns).
func step(pending []*spec.Spec, completed map[string]bool, mode AutonomyMode, promoted func(PauseCategory) bool) (string, *spec.Spec, *PauseInfo) {
	var next *spec.Spec
	blocked := false
	for _, p := range pending {
		if depsMet(p, completed) {
			next = p
			break
		}
	}
	if next == nil {
		// Every remaining child is blocked on an unmet dependency.
		next = pending[0]
		blocked = true
	}

	ctx := RunContext{
		ActionClassified: true,
		NextScore:        -1, // readiness-score detector deferred
		Blocked:          blocked,
		Promoted:         promoted,
	}
	dec := NeedsMe(next, ctx, mode)
	if dec.Proceed {
		return "continue", next, nil
	}
	return "pause", next, &PauseInfo{Category: string(dec.Category), Reason: dec.Reason}
}

// DryStep is one simulated transition in a DryRun preview.
type DryStep struct {
	Step     int    `json:"step"`
	Verdict  string `json:"verdict"`
	Spec     string `json:"spec,omitempty"`
	Category string `json:"category,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// DryRun previews up to n transitions Check WOULD take from the current
// state, optimistically assuming each "continue" child then completes. It
// stops early on a pause or when the run would be done. No disk writes.
func DryRun(init *spec.Spec, all []*spec.Spec, n int, promoted func(PauseCategory) bool) []DryStep {
	mode := ParseMode(init.Autonomy)
	kids := Children(init, all)
	completed := completedSet(all)

	// Working pending list (copy; we mutate the simulated completed set).
	var pending []*spec.Spec
	for _, k := range kids {
		if !isCompleted(k) {
			pending = append(pending, k)
		}
	}

	var steps []DryStep
	for i := 1; i <= n; i++ {
		if len(pending) == 0 {
			steps = append(steps, DryStep{Step: i, Verdict: "done"})
			break
		}
		verdict, next, pause := step(pending, completed, mode, promoted)
		ds := DryStep{Step: i, Verdict: verdict}
		if next != nil {
			ds.Spec = next.Slug
		}
		if pause != nil {
			ds.Category = pause.Category
			ds.Reason = pause.Reason
		}
		steps = append(steps, ds)
		if verdict != "continue" {
			break // a pause halts the preview, as it would the run
		}
		// Optimistically advance: mark next done, drop from pending.
		completed[next.Slug] = true
		out := pending[:0]
		for _, p := range pending {
			if p.Slug != next.Slug {
				out = append(out, p)
			}
		}
		pending = out
	}
	return steps
}
