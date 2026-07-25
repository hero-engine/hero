package drive

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

// Pin the scorer so stage classification in these tests is driven purely by
// structure (AC presence), not the real scorer's verdict on tiny fixtures.
func init() { scoreFn = func(*spec.Spec) int { return 100 } }

// mkChild builds a *designed* child (has acceptance criteria → ready-to-deliver).
func mkChild(slug, parent string, status spec.Status, deps ...string) *spec.Spec {
	rels := []spec.Relation{{Kind: "parent", Target: parent}}
	for _, d := range deps {
		rels = append(rels, spec.Relation{Kind: "depends-on", Target: d})
	}
	return &spec.Spec{
		Slug: slug, Type: spec.TypeFeature, Status: status,
		Relations: rels,
		Sections: map[string]string{
			"kickoff":             "kickoff for " + slug,
			"acceptance criteria": "- THE SYSTEM SHALL do " + slug,
		},
	}
}

// mkStub builds an *undesigned* child (no acceptance criteria → needs-design).
func mkStub(slug, parent string, deps ...string) *spec.Spec {
	rels := []spec.Relation{{Kind: "parent", Target: parent}}
	for _, d := range deps {
		rels = append(rels, spec.Relation{Kind: "depends-on", Target: d})
	}
	return &spec.Spec{
		Slug: slug, Type: spec.TypeFeature, Status: spec.StatusPlanning,
		Relations: rels,
		Sections:  map[string]string{"goal": "do " + slug},
	}
}

func mkInit(slug, autonomy string) *spec.Spec {
	return &spec.Spec{Slug: slug, Type: spec.TypeInitiative, Status: spec.StatusPlanning, Autonomy: autonomy}
}

func TestCheckDoneWhenAllChildrenCompleted(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		mkChild("a", "drive", spec.StatusCompleted),
		mkChild("b", "drive", spec.StatusCompleted),
	}
	res := Check(init, all, nil, nil)
	if res.Verdict != "done" {
		t.Fatalf("verdict=%q, want done", res.Verdict)
	}
	if len(res.Completed) != 2 || len(res.Remaining) != 0 {
		t.Errorf("completed=%v remaining=%v", res.Completed, res.Remaining)
	}
}

func TestCheckContinueGuided(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		mkChild("a", "drive", spec.StatusCompleted),
		mkChild("b", "drive", spec.StatusPlanning),
	}
	res := Check(init, all, nil, nil)
	if res.Verdict != "continue" {
		t.Fatalf("verdict=%q, want continue", res.Verdict)
	}
	if res.NextSpec != "b" {
		t.Errorf("next=%q, want b", res.NextSpec)
	}
	if res.Kickoff != "kickoff for b" {
		t.Errorf("kickoff=%q, want child's kickoff", res.Kickoff)
	}
}

func TestCheckPauseSupervised(t *testing.T) {
	init := mkInit("drive", "supervised")
	all := []*spec.Spec{init, mkChild("b", "drive", spec.StatusPlanning)}
	res := Check(init, all, nil, nil)
	if res.Verdict != "pause" || res.Pause == nil || res.Pause.Category != string(CategorySupervised) {
		t.Fatalf("want supervised pause, got %+v", res)
	}
}

func TestCheckBlockedWhenDepsUnmet(t *testing.T) {
	init := mkInit("drive", "autonomous")
	all := []*spec.Spec{init,
		mkChild("b", "drive", spec.StatusPlanning, "a"), // depends on a, which is not completed
	}
	res := Check(init, all, nil, nil)
	if res.Verdict != "pause" || res.Pause == nil || res.Pause.Category != string(CategoryBlocked) {
		t.Fatalf("want blocked pause, got %+v", res)
	}
}

func TestCheckNoChildrenPauses(t *testing.T) {
	init := mkInit("drive", "guided")
	res := Check(init, []*spec.Spec{init}, nil, nil)
	if res.Verdict != "pause" || res.Pause == nil {
		t.Fatalf("initiative with no children should pause, got %+v", res)
	}
}

func TestDryRunGuidedPreviewsThenDone(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		mkChild("a", "drive", spec.StatusPlanning),
		mkChild("b", "drive", spec.StatusPlanning),
	}
	steps := DryRun(init, all, 5, nil)
	// 2 continues (a, b) then a done step.
	if len(steps) != 3 {
		t.Fatalf("steps=%d, want 3 (%+v)", len(steps), steps)
	}
	if steps[0].Verdict != "continue" || steps[0].Spec != "a" {
		t.Errorf("step1=%+v, want continue a", steps[0])
	}
	if steps[1].Verdict != "continue" || steps[1].Spec != "b" {
		t.Errorf("step2=%+v, want continue b", steps[1])
	}
	if steps[2].Verdict != "done" {
		t.Errorf("step3=%+v, want done", steps[2])
	}
}

// TestDryRunPreviewsHigherPriorityFirst: DryRun uses the same priority-aware
// selection as Check — the higher-priority child previews first even though
// its slug sorts later.
func TestDryRunPreviewsHigherPriorityFirst(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		withPriority(mkChild("aaa", "drive", spec.StatusPlanning), "low"),
		withPriority(mkChild("zzz", "drive", spec.StatusPlanning), "critical"),
	}
	steps := DryRun(init, all, 5, nil)
	// zzz (critical) first, then aaa (low), then done.
	if len(steps) != 3 {
		t.Fatalf("steps=%d, want 3 (%+v)", len(steps), steps)
	}
	if steps[0].Verdict != "continue" || steps[0].Spec != "zzz" {
		t.Errorf("step1=%+v, want continue zzz (critical beats slug-earlier low)", steps[0])
	}
	if steps[1].Verdict != "continue" || steps[1].Spec != "aaa" {
		t.Errorf("step2=%+v, want continue aaa", steps[1])
	}
	if steps[2].Verdict != "done" {
		t.Errorf("step3=%+v, want done", steps[2])
	}
}

// TestDryRunSeamCollisionPause: DryRun yields a SeamCollision pause when the
// only otherwise-ready candidate's conflicts-with target is delivering,
// naming the in-flight spec — mirroring Check.
func TestDryRunSeamCollisionPause(t *testing.T) {
	init := mkInit("drive", "autonomous")
	all := []*spec.Spec{init,
		withConflict(mkChild("aaa", "drive", spec.StatusPlanning), "seam-peer"),
		{Slug: "seam-peer", Type: spec.TypeFeature, Status: spec.StatusDelivering},
	}
	steps := DryRun(init, all, 5, nil)
	if len(steps) != 1 {
		t.Fatalf("steps=%d, want 1 (%+v)", len(steps), steps)
	}
	if steps[0].Verdict != "pause" || steps[0].Category != string(CategorySeamCollision) {
		t.Fatalf("step1=%+v, want pause/SeamCollision", steps[0])
	}
	if !strings.Contains(steps[0].Reason, "seam-peer") {
		t.Errorf("reason=%q, want it to name the in-flight spec seam-peer", steps[0].Reason)
	}
}

func TestDryRunSupervisedStopsAtFirst(t *testing.T) {
	init := mkInit("drive", "supervised")
	all := []*spec.Spec{init,
		mkChild("a", "drive", spec.StatusPlanning),
		mkChild("b", "drive", spec.StatusPlanning),
	}
	steps := DryRun(init, all, 3, nil)
	if len(steps) != 1 || steps[0].Verdict != "pause" {
		t.Fatalf("supervised dry-run should pause at first step, got %+v", steps)
	}
}

func TestChildrenSortedAndScoped(t *testing.T) {
	init := mkInit("drive", "")
	all := []*spec.Spec{init,
		mkChild("charlie", "drive", spec.StatusPlanning),
		mkChild("alpha", "drive", spec.StatusPlanning),
		mkChild("other-kid", "someone-else", spec.StatusPlanning),
	}
	kids := Children(init, all)
	if len(kids) != 2 || kids[0].Slug != "alpha" || kids[1].Slug != "charlie" {
		t.Fatalf("Children=%v, want [alpha charlie]", slugsOf(kids))
	}
}

func slugsOf(ss []*spec.Spec) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = s.Slug
	}
	return out
}

// withPriority tags a child's priority (Part A selection input).
func withPriority(s *spec.Spec, priority string) *spec.Spec {
	s.Priority = priority
	return s
}

// withConflict adds a conflicts-with relation to a child (Part B soft-mutex).
func withConflict(s *spec.Spec, target string) *spec.Spec {
	s.Relations = append(s.Relations, spec.Relation{Kind: "conflicts-with", Target: target})
	return s
}

// TestCheckSelectsHigherPriorityOverSlug: two dependency-ready children; the
// higher-priority one is selected even though its slug sorts later.
func TestCheckSelectsHigherPriorityOverSlug(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		withPriority(mkChild("aaa", "drive", spec.StatusPlanning), "low"),
		withPriority(mkChild("zzz", "drive", spec.StatusPlanning), "critical"),
	}
	res := Check(init, all, nil, nil)
	if res.Verdict != "continue" {
		t.Fatalf("verdict=%q, want continue (%+v)", res.Verdict, res)
	}
	if res.NextSpec != "zzz" {
		t.Errorf("next=%q, want zzz (critical beats slug-earlier low)", res.NextSpec)
	}
}

// TestCheckConflictExcludesDeliveringCandidate: a ready candidate whose
// conflicts-with target is delivering is not selected; a clean ready
// candidate is chosen instead.
func TestCheckConflictExcludesDeliveringCandidate(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		// "aaa" would win on slug order but conflicts with a delivering spec.
		withConflict(mkChild("aaa", "drive", spec.StatusPlanning), "seam-peer"),
		mkChild("bbb", "drive", spec.StatusPlanning),
		{Slug: "seam-peer", Type: spec.TypeFeature, Status: spec.StatusDelivering},
	}
	res := Check(init, all, nil, nil)
	if res.Verdict != "continue" {
		t.Fatalf("verdict=%q, want continue (%+v)", res.Verdict, res)
	}
	if res.NextSpec != "bbb" {
		t.Errorf("next=%q, want bbb (aaa is conflict-excluded)", res.NextSpec)
	}
}

// TestCheckSeamCollisionWhenOnlyCandidateConflicts: the only otherwise-ready
// candidate is conflict-blocked → pause / SeamCollision naming the in-flight
// spec.
func TestCheckSeamCollisionWhenOnlyCandidateConflicts(t *testing.T) {
	init := mkInit("drive", "autonomous")
	all := []*spec.Spec{init,
		withConflict(mkChild("aaa", "drive", spec.StatusPlanning), "seam-peer"),
		{Slug: "seam-peer", Type: spec.TypeFeature, Status: spec.StatusDelivering},
	}
	res := Check(init, all, nil, nil)
	if res.Verdict != "pause" || res.Pause == nil {
		t.Fatalf("want pause, got %+v", res)
	}
	if res.Pause.Category != string(CategorySeamCollision) {
		t.Errorf("category=%q, want SeamCollision", res.Pause.Category)
	}
	if !strings.Contains(res.Pause.Reason, "seam-peer") {
		t.Errorf("reason=%q, want it to name the in-flight spec seam-peer", res.Pause.Reason)
	}
}

// TestCheckBlockedNotSeamWhenDepUnmet: first-remaining child blocked on an
// unmet dependency stays Blocked, not SeamCollision.
func TestCheckBlockedNotSeamWhenDepUnmet(t *testing.T) {
	init := mkInit("drive", "autonomous")
	all := []*spec.Spec{init,
		mkChild("aaa", "drive", spec.StatusPlanning, "missing-dep"),
	}
	res := Check(init, all, nil, nil)
	if res.Verdict != "pause" || res.Pause == nil || res.Pause.Category != string(CategoryBlocked) {
		t.Fatalf("want blocked pause, got %+v", res)
	}
}

// TestCheckDeterministic: same on-disk state yields an identical verdict
// across repeated calls (no wall-clock or iteration-order dependence).
func TestCheckDeterministic(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		withPriority(mkChild("aaa", "drive", spec.StatusPlanning), "medium"),
		withPriority(mkChild("bbb", "drive", spec.StatusPlanning), "critical"),
		withPriority(mkChild("ccc", "drive", spec.StatusPlanning), "critical"),
	}
	first := Check(init, all, nil, nil)
	for i := 0; i < 20; i++ {
		got := Check(init, all, nil, nil)
		if got.NextSpec != first.NextSpec || got.Verdict != first.Verdict {
			t.Fatalf("call %d differs: %+v vs %+v", i, got, first)
		}
	}
	// Tie between two criticals resolves on slug: bbb < ccc.
	if first.NextSpec != "bbb" {
		t.Errorf("next=%q, want bbb (critical tie broken by slug)", first.NextSpec)
	}
}

// TestCheckDeterministicUnderInputPermutation: the min-under-comparator
// selection must not depend on the order of the `all` slice. The same logical
// spec set in different orderings yields an identical verdict.
func TestCheckDeterministicUnderInputPermutation(t *testing.T) {
	init := mkInit("drive", "guided")
	// Two criticals (tie → slug picks bbb) plus a slug-earlier low that must
	// lose on priority; conflict-excluded ddd exercises the gate path too.
	base := func() []*spec.Spec {
		return []*spec.Spec{
			withPriority(mkChild("aaa", "drive", spec.StatusPlanning), "low"),
			withPriority(mkChild("bbb", "drive", spec.StatusPlanning), "critical"),
			withPriority(mkChild("ccc", "drive", spec.StatusPlanning), "critical"),
			withConflict(withPriority(mkChild("ddd", "drive", spec.StatusPlanning), "critical"), "seam-peer"),
			{Slug: "seam-peer", Type: spec.TypeFeature, Status: spec.StatusDelivering},
		}
	}
	orderings := [][]int{
		{0, 1, 2, 3, 4},
		{4, 3, 2, 1, 0},
		{2, 4, 0, 3, 1},
	}
	var want string
	for i, order := range orderings {
		kids := base()
		all := []*spec.Spec{init}
		for _, idx := range order {
			all = append(all, kids[idx])
		}
		res := Check(init, all, nil, nil)
		if res.Verdict != "continue" {
			t.Fatalf("ordering %d: verdict=%q, want continue (%+v)", i, res.Verdict, res)
		}
		if i == 0 {
			want = res.NextSpec
		} else if res.NextSpec != want {
			t.Fatalf("ordering %d: next=%q, want %q — selection depends on slice order", i, res.NextSpec, want)
		}
	}
	// bbb wins: critical (ties with ccc/ddd), ddd is conflict-excluded, and
	// the bbb/ccc tie breaks on slug.
	if want != "bbb" {
		t.Errorf("next=%q, want bbb", want)
	}
}

// --- Detected-seam backstop (piece 3): injected detector callback ---

// TestCheckSeamDetectedPauseNamesOverlap: a selected candidate whose files
// overlap a delivering spec (per the injected detector) — and that overlap is
// NOT authored — pauses with SeamDetected naming the file(s) + in-flight spec.
func TestCheckSeamDetectedPauseNamesOverlap(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init, mkChild("aaa", "drive", spec.StatusPlanning)}
	detect := func(slug string) []DetectedConflict {
		if slug == "aaa" {
			return []DetectedConflict{{Slug: "in-flight-peer", Files: []string{"internal/shared/util.go"}}}
		}
		return nil
	}
	res := Check(init, all, nil, detect)
	if res.Verdict != "pause" || res.Pause == nil {
		t.Fatalf("want pause, got %+v", res)
	}
	if res.Pause.Category != string(CategorySeamDetected) {
		t.Errorf("category=%q, want SeamDetected", res.Pause.Category)
	}
	if !strings.Contains(res.Pause.Reason, "in-flight-peer") || !strings.Contains(res.Pause.Reason, "internal/shared/util.go") {
		t.Errorf("reason=%q, want it to name the in-flight spec and the overlapping file", res.Pause.Reason)
	}
}

// TestCheckSeamDetectedSkipsAuthoredPicksUndeclared: the dedup subtracts any
// detected overlap already authored as conflicts-with, then surfaces the first
// remaining (undeclared) overlap — proving the subtraction and the
// deterministic slug-order "first" together.
func TestCheckSeamDetectedSkipsAuthoredPicksUndeclared(t *testing.T) {
	init := mkInit("drive", "guided")
	// aaa authors conflicts-with "authored-peer" (not delivering → stays
	// selectable). The detector reports both authored-peer and an undeclared
	// ghost-peer; only ghost-peer should surface.
	all := []*spec.Spec{init,
		withConflict(mkChild("aaa", "drive", spec.StatusPlanning), "authored-peer"),
	}
	detect := func(slug string) []DetectedConflict {
		return []DetectedConflict{
			{Slug: "authored-peer", Files: []string{"src/authored.go"}},
			{Slug: "ghost-peer", Files: []string{"src/ghost.go"}},
		}
	}
	res := Check(init, all, nil, detect)
	if res.Verdict != "pause" || res.Pause == nil || res.Pause.Category != string(CategorySeamDetected) {
		t.Fatalf("want SeamDetected pause, got %+v", res)
	}
	if strings.Contains(res.Pause.Reason, "authored-peer") {
		t.Errorf("authored overlap must be subtracted, reason names it: %q", res.Pause.Reason)
	}
	if !strings.Contains(res.Pause.Reason, "ghost-peer") || !strings.Contains(res.Pause.Reason, "src/ghost.go") {
		t.Errorf("reason=%q, want the undeclared ghost-peer overlap", res.Pause.Reason)
	}
}

// TestCheckSeamDetectedPromotableAcrossModes: SeamDetected is promotable —
// Autonomous+promoted proceeds, Guided pauses (ignores promotions), Supervised
// pauses at the boundary (its own category, before the taxonomy).
func TestCheckSeamDetectedPromotableAcrossModes(t *testing.T) {
	detect := func(slug string) []DetectedConflict {
		if slug == "aaa" {
			return []DetectedConflict{{Slug: "peer", Files: []string{"src/a.go"}}}
		}
		return nil
	}
	mkAll := func(init *spec.Spec) []*spec.Spec {
		return []*spec.Spec{init, mkChild("aaa", "drive", spec.StatusPlanning)}
	}

	// Autonomous + promoted SeamDetected → proceed.
	initA := mkInit("drive", "autonomous")
	promoted := func(c PauseCategory) bool { return c == CategorySeamDetected }
	if res := Check(initA, mkAll(initA), promoted, detect); res.Verdict != "continue" {
		t.Fatalf("autonomous+promoted should proceed past SeamDetected, got %+v", res)
	}

	// Guided → pause even with a blanket promotion (Guided ignores promotions).
	initG := mkInit("drive", "guided")
	if res := Check(initG, mkAll(initG), func(PauseCategory) bool { return true }, detect); res.Verdict != "pause" || res.Pause.Category != string(CategorySeamDetected) {
		t.Fatalf("guided should pause SeamDetected, got %+v", res)
	}

	// Supervised → pause at the boundary (CategorySupervised wins first).
	initS := mkInit("drive", "supervised")
	if res := Check(initS, mkAll(initS), nil, detect); res.Verdict != "pause" || res.Pause.Category != string(CategorySupervised) {
		t.Fatalf("supervised should pause at boundary, got %+v", res)
	}
}

// TestCheckAuthoredWinsOverDetected: when the candidate's overlap is authored
// AND the target is delivering, the authored gate excludes it from selection
// and the fallback emits SeamCollision — never SeamDetected — even if the
// detector also reports the same overlap.
func TestCheckAuthoredWinsOverDetected(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		withConflict(mkChild("aaa", "drive", spec.StatusPlanning), "peer"),
		{Slug: "peer", Type: spec.TypeFeature, Status: spec.StatusDelivering},
	}
	detect := func(slug string) []DetectedConflict {
		return []DetectedConflict{{Slug: "peer", Files: []string{"src/shared.go"}}}
	}
	res := Check(init, all, nil, detect)
	if res.Verdict != "pause" || res.Pause == nil {
		t.Fatalf("want pause, got %+v", res)
	}
	if res.Pause.Category != string(CategorySeamCollision) {
		t.Errorf("category=%q, want SeamCollision (authored wins, never SeamDetected)", res.Pause.Category)
	}
}

// TestCheckNilDetectorMatchesPiece1: a nil detector never emits SeamDetected
// and is deterministic across runs; an empty detector yields the identical
// verdict — the piece-1 backward-compat / determinism anchor.
func TestCheckNilDetectorMatchesPiece1(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		mkChild("aaa", "drive", spec.StatusPlanning),
		withPriority(mkChild("zzz", "drive", spec.StatusPlanning), "critical"),
	}
	first := Check(init, all, nil, nil)
	if first.Pause != nil && first.Pause.Category == string(CategorySeamDetected) {
		t.Fatalf("nil detector must never emit SeamDetected, got %+v", first)
	}
	for i := 0; i < 10; i++ {
		got := Check(init, all, nil, nil)
		if got.Verdict != first.Verdict || got.NextSpec != first.NextSpec {
			t.Fatalf("run %d differs from first: %+v vs %+v", i, got, first)
		}
	}
	// An empty (no-op) detector must produce the same verdict as nil.
	noop := func(string) []DetectedConflict { return nil }
	if got := Check(init, all, nil, noop); got.Verdict != first.Verdict || got.NextSpec != first.NextSpec {
		t.Fatalf("empty detector diverged from nil: %+v vs %+v", got, first)
	}
}

// TestCheckBackwardCompatSlugOrder: no priorities + no conflicts-with must
// reproduce today's slug-order selection AND today's Remaining/Completed
// ordering.
func TestCheckBackwardCompatSlugOrder(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init,
		mkChild("charlie", "drive", spec.StatusCompleted),
		mkChild("alpha", "drive", spec.StatusPlanning),
		mkChild("bravo", "drive", spec.StatusPlanning),
	}
	res := Check(init, all, nil, nil)
	if res.NextSpec != "alpha" {
		t.Errorf("next=%q, want alpha (slug-first when no priorities)", res.NextSpec)
	}
	if len(res.Remaining) != 2 || res.Remaining[0] != "alpha" || res.Remaining[1] != "bravo" {
		t.Errorf("remaining=%v, want [alpha bravo] slug-ordered", res.Remaining)
	}
	if len(res.Completed) != 1 || res.Completed[0] != "charlie" {
		t.Errorf("completed=%v, want [charlie]", res.Completed)
	}
}

// TestCheckHonorsFrontmatterDeclaredChildren covers AC-5: a child declared in
// the initiative's frontmatter but never scaffolded is remaining, and the run
// is not done. Before the roster unification `declaredChildSlugs` read only
// the body table, so a frontmatter-only declaration was invisible here.
func TestCheckHonorsFrontmatterDeclaredChildren(t *testing.T) {
	init := mkInit("gov", "guided")
	init.Relations = []spec.Relation{
		{Kind: "child", Target: "blast-radius-tiers"},
		{Kind: "child", Target: "financial-action-gate"},
	}
	all := []*spec.Spec{init, mkChild("blast-radius-tiers", "gov", spec.StatusCompleted)}

	res := Check(init, all, nil, nil)
	if res.Verdict == "done" {
		t.Fatalf("must not be done with financial-action-gate unbuilt, got %+v", res)
	}
	if len(res.Remaining) != 1 || res.Remaining[0] != "financial-action-gate" {
		t.Fatalf("Remaining = %v, want [financial-action-gate]", res.Remaining)
	}
	if res.Action != ActionDesign {
		t.Errorf("action = %q, want design (the child has no spec on disk yet)", res.Action)
	}
}

// TestCheckAndCompletionGateAgreeOnRoster is the AC-4 invariant, asserted
// end-to-end rather than by inspection: for the same initiative, `goal --check`
// reports nothing remaining exactly when the completion gate is willing to
// auto-complete. Both now read spec.DeclaredChildren, so they cannot drift.
func TestCheckAndCompletionGateAgreeOnRoster(t *testing.T) {
	newInit := func() *spec.Spec {
		s := mkInit("gov", "guided")
		s.Relations = []spec.Relation{{Kind: "child", Target: "alpha"}}
		// alpha is declared in frontmatter only, bravo in the table only —
		// the exact split that let the two paths derive different rosters.
		s.Sections = map[string]string{
			"child specs & sequence": "1. **[bravo](bravo/spec.md)**\n",
		}
		return s
	}

	cases := []struct {
		name string
		all  func(init *spec.Spec) []*spec.Spec
	}{
		{"nothing scaffolded", func(i *spec.Spec) []*spec.Spec { return []*spec.Spec{i} }},
		{"frontmatter child only", func(i *spec.Spec) []*spec.Spec {
			return []*spec.Spec{i, mkChild("alpha", "gov", spec.StatusCompleted)}
		}},
		{"table child only", func(i *spec.Spec) []*spec.Spec {
			return []*spec.Spec{i, mkChild("bravo", "gov", spec.StatusCompleted)}
		}},
		{"one delivered one open", func(i *spec.Spec) []*spec.Spec {
			return []*spec.Spec{i, mkChild("alpha", "gov", spec.StatusCompleted), mkChild("bravo", "gov", spec.StatusDelivering)}
		}},
		{"both delivered", func(i *spec.Spec) []*spec.Spec {
			return []*spec.Spec{i, mkChild("alpha", "gov", spec.StatusCompleted), mkChild("bravo", "gov", spec.StatusCompleted)}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			init := newInit()
			all := c.all(init)
			checkSaysDone := len(Check(init, all, nil, nil).Remaining) == 0
			gateSaysDone := spec.InitiativeReadyToComplete(init, all)
			if checkSaysDone != gateSaysDone {
				t.Fatalf("rosters disagree: goal --check remaining-empty=%v, completion gate=%v",
					checkSaysDone, gateSaysDone)
			}
		})
	}
}
