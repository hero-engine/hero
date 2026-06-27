package drive

import (
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func mkChild(slug, parent string, status spec.Status, deps ...string) *spec.Spec {
	rels := []spec.Relation{{Kind: "parent", Target: parent}}
	for _, d := range deps {
		rels = append(rels, spec.Relation{Kind: "depends-on", Target: d})
	}
	return &spec.Spec{
		Slug: slug, Type: spec.TypeFeature, Status: status,
		Relations: rels,
		Sections:  map[string]string{"kickoff": "kickoff for " + slug},
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
	res := Check(init, all)
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
	res := Check(init, all)
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
	res := Check(init, all)
	if res.Verdict != "pause" || res.Pause == nil || res.Pause.Category != string(CategorySupervised) {
		t.Fatalf("want supervised pause, got %+v", res)
	}
}

func TestCheckBlockedWhenDepsUnmet(t *testing.T) {
	init := mkInit("drive", "autonomous")
	all := []*spec.Spec{init,
		mkChild("b", "drive", spec.StatusPlanning, "a"), // depends on a, which is not completed
	}
	res := Check(init, all)
	if res.Verdict != "pause" || res.Pause == nil || res.Pause.Category != string(CategoryBlocked) {
		t.Fatalf("want blocked pause, got %+v", res)
	}
}

func TestCheckNoChildrenPauses(t *testing.T) {
	init := mkInit("drive", "guided")
	res := Check(init, []*spec.Spec{init})
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
	steps := DryRun(init, all, 5)
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

func TestDryRunSupervisedStopsAtFirst(t *testing.T) {
	init := mkInit("drive", "supervised")
	all := []*spec.Spec{init,
		mkChild("a", "drive", spec.StatusPlanning),
		mkChild("b", "drive", spec.StatusPlanning),
	}
	steps := DryRun(init, all, 3)
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
