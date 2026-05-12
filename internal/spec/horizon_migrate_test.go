package spec

import "testing"

func TestProposeHorizon_PreservesValidExisting(t *testing.T) {
	s := &Spec{Slug: "x", Horizon: HorizonSomeday}
	got, _ := proposeHorizon(s)
	if got != HorizonSomeday {
		t.Errorf("got %q, want someday (preserve existing)", got)
	}
}

func TestProposeHorizon_StatusCompletedBecomesNow(t *testing.T) {
	s := &Spec{Slug: "x", Status: StatusCompleted}
	got, _ := proposeHorizon(s)
	if got != HorizonNow {
		t.Errorf("got %q, want now", got)
	}
}

func TestProposeHorizon_DeliveringBecomesNow(t *testing.T) {
	s := &Spec{Slug: "x", Status: StatusDelivering}
	got, _ := proposeHorizon(s)
	if got != HorizonNow {
		t.Errorf("got %q, want now", got)
	}
}

func TestProposeHorizon_MarketingTagBecomesSomeday(t *testing.T) {
	s := &Spec{Slug: "x", Status: StatusPlanning, Tags: []string{"marketing", "launch"}}
	got, reason := proposeHorizon(s)
	if got != HorizonSomeday {
		t.Errorf("got %q, want someday (tag-based)", got)
	}
	if reason == "" {
		t.Error("reason empty")
	}
}

func TestProposeHorizon_DistributionSlugBecomesSomeday(t *testing.T) {
	s := &Spec{Slug: "hero-distribution", Status: StatusPlanning}
	got, _ := proposeHorizon(s)
	if got != HorizonSomeday {
		t.Errorf("got %q, want someday (slug-based)", got)
	}
}

func TestProposeHorizon_RecoveryTagBecomesNow(t *testing.T) {
	s := &Spec{Slug: "x", Status: StatusPlanning, Tags: []string{"v2-recovery"}}
	got, _ := proposeHorizon(s)
	if got != HorizonNow {
		t.Errorf("got %q, want now (recovery tag)", got)
	}
}

func TestProposeHorizon_DefaultIsNext(t *testing.T) {
	s := &Spec{Slug: "boring-feature", Status: StatusPlanning, Tags: []string{"feature"}}
	got, _ := proposeHorizon(s)
	if got != HorizonNext {
		t.Errorf("got %q, want next (default)", got)
	}
}

func TestPlanHorizonMigration_SkipsAlreadyValidWithoutChange(t *testing.T) {
	specs := []*Spec{
		{Slug: "a", Horizon: HorizonNow},
		{Slug: "b", Status: StatusCompleted},
	}
	plan := PlanHorizonMigration(specs)
	if len(plan) != 2 {
		t.Fatalf("len = %d, want 2", len(plan))
	}
	if !plan[0].Skip {
		t.Errorf("'a' should be skipped (already at proposed value)")
	}
	if plan[1].Skip {
		t.Errorf("'b' should not be skipped — needs new horizon")
	}
}

func TestEffectiveHorizon_UnsetDefaultsToNow(t *testing.T) {
	s := &Spec{}
	if s.EffectiveHorizon() != HorizonNow {
		t.Errorf("unset = %q, want now", s.EffectiveHorizon())
	}
}

func TestEffectiveHorizon_PreservesValid(t *testing.T) {
	s := &Spec{Horizon: HorizonSomeday}
	if s.EffectiveHorizon() != HorizonSomeday {
		t.Errorf("got %q, want someday", s.EffectiveHorizon())
	}
}

func TestIsActiveHorizon_NowAndNextOnly(t *testing.T) {
	cases := []struct {
		h    Horizon
		want bool
	}{
		{HorizonNow, true},
		{HorizonNext, true},
		{HorizonSomeday, false},
		{HorizonParking, false},
		{"", true}, // empty defaults to now → active
	}
	for _, tc := range cases {
		s := &Spec{Horizon: tc.h}
		if got := s.IsActiveHorizon(); got != tc.want {
			t.Errorf("IsActiveHorizon(%q) = %v, want %v", tc.h, got, tc.want)
		}
	}
}
