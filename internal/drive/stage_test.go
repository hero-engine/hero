package drive

import (
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestChildStageClassification(t *testing.T) {
	completed := &spec.Spec{Slug: "c", Status: spec.StatusCompleted, Sections: map[string]string{"acceptance criteria": "x"}}
	if ChildStage(completed, 100) != StageDone {
		t.Error("completed → StageDone")
	}
	stub := &spec.Spec{Slug: "s", Status: spec.StatusPlanning, Sections: map[string]string{"goal": "g"}}
	if ChildStage(stub, 100) != StageNeedsDesign {
		t.Error("no AC → StageNeedsDesign even with a high score")
	}
	designed := &spec.Spec{Slug: "d", Status: spec.StatusPlanning, Sections: map[string]string{"acceptance criteria": "- THE SYSTEM SHALL x"}}
	if ChildStage(designed, 80) != StageReadyDeliver {
		t.Error("AC + good score → StageReadyDeliver")
	}
	if ChildStage(designed, 20) != StageNeedsDesign {
		t.Error("AC but score below threshold → StageNeedsDesign")
	}
	if ChildStage(designed, -1) != StageReadyDeliver {
		t.Error("AC + unknown score (-1) → StageReadyDeliver (score skipped)")
	}
}

func TestActionForStage(t *testing.T) {
	if ActionForStage(StageReadyDeliver) != ActionDeliver {
		t.Error("ready → deliver")
	}
	for _, s := range []Stage{StageNeedsDesign, StageNeedsScaffold} {
		if ActionForStage(s) != ActionDesign {
			t.Errorf("%v → design", s)
		}
	}
}

func TestCheckRoutesDesignForUndesignedChild(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init, mkStub("stub-child", "drive")}
	res := Check(init, all, nil)
	if res.Verdict != "continue" {
		t.Fatalf("verdict=%q, want continue", res.Verdict)
	}
	if res.Action != ActionDesign {
		t.Errorf("action=%q, want design (undesigned child must not go to delivery)", res.Action)
	}
	if res.Kickoff != "" {
		t.Errorf("a design action must not carry a delivery kickoff, got %q", res.Kickoff)
	}
}

func TestCheckRoutesDeliverForDesignedChild(t *testing.T) {
	init := mkInit("drive", "guided")
	all := []*spec.Spec{init, mkChild("ready-child", "drive", spec.StatusPlanning)}
	res := Check(init, all, nil)
	if res.Verdict != "continue" || res.Action != ActionDeliver {
		t.Fatalf("want continue+deliver, got verdict=%q action=%q", res.Verdict, res.Action)
	}
	if res.Kickoff == "" {
		t.Error("deliver action should carry the child kickoff")
	}
}

func initWithChildTable(slug, autonomy, tableBody string) *spec.Spec {
	return &spec.Spec{
		Slug: slug, Type: spec.TypeInitiative, Status: spec.StatusPlanning, Autonomy: autonomy,
		Sections: map[string]string{"child specs & sequence": tableBody},
	}
}

func TestCheckNoShortCircuitOnDeclaredButUnscaffoldedChild(t *testing.T) {
	// Initiative declares two children in its table; only one exists on disk
	// (and it's completed). The other is declared-but-unscaffolded.
	table := "1. **[done-child](done-child/spec.md)** — first\n2. **[ghost-child](ghost-child/spec.md)** — not yet scaffolded\n"
	init := initWithChildTable("drive", "guided", table)
	done := mkChild("done-child", "drive", spec.StatusCompleted)
	all := []*spec.Spec{init, done}

	res := Check(init, all, nil)
	if res.Verdict == "done" {
		t.Fatalf("must NOT short-circuit to done while ghost-child is unscaffolded, got %+v", res)
	}
	if res.Verdict != "continue" || res.Action != ActionDesign {
		t.Fatalf("should route to design the missing child, got verdict=%q action=%q", res.Verdict, res.Action)
	}
	if res.NextSpec != "ghost-child" {
		t.Errorf("next=%q, want ghost-child (the unscaffolded declared child)", res.NextSpec)
	}
}

func TestDeclaredChildSlugs(t *testing.T) {
	init := initWithChildTable("drive", "", "- **[alpha](alpha/spec.md)** x\n- [bravo](bravo/spec.md) y\n- [external](../../features/other/spec.md) ignore\n")
	got := declaredChildSlugs(init)
	if len(got) != 2 || got[0] != "alpha" || got[1] != "bravo" {
		t.Fatalf("declaredChildSlugs = %v, want [alpha bravo] (external link ignored)", got)
	}
}

func TestDryRunSurfacesAction(t *testing.T) {
	init := mkInit("drive", "guided")
	// Slugs chosen so the stub sorts first (a- < b-): design then deliver.
	all := []*spec.Spec{init, mkStub("a-stub", "drive"), mkChild("b-ready", "drive", spec.StatusPlanning)}
	steps := DryRun(init, all, 5, nil)
	if len(steps) != 3 {
		t.Fatalf("steps=%d want 3: %+v", len(steps), steps)
	}
	if steps[0].Action != ActionDesign || steps[0].Spec != "a-stub" {
		t.Errorf("step1=%+v, want design a-stub", steps[0])
	}
	if steps[1].Action != ActionDeliver || steps[1].Spec != "b-ready" {
		t.Errorf("step2=%+v, want deliver b-ready", steps[1])
	}
	if steps[2].Verdict != "done" {
		t.Errorf("step3=%+v, want done", steps[2])
	}
}
