package drive

import "testing"

func TestPromotionStateMachine(t *testing.T) {
	dir := t.TempDir()
	p, err := LoadPromotions(dir, "alice")
	if err != nil {
		t.Fatal(err)
	}
	// Below threshold: not promoted.
	for i := 0; i < PromoteThreshold-1; i++ {
		p.RecordOutcome(CategoryDesignFork, OutcomeApproved)
	}
	if p.IsPromoted(CategoryDesignFork) {
		t.Fatalf("should not promote before %d approvals", PromoteThreshold)
	}
	// Hitting the threshold promotes.
	p.RecordOutcome(CategoryDesignFork, OutcomeApproved)
	if !p.IsPromoted(CategoryDesignFork) {
		t.Fatalf("should promote at %d approvals", PromoteThreshold)
	}
	// An edit/redirect demotes and resets.
	p.RecordOutcome(CategoryDesignFork, OutcomeRedirected)
	if p.IsPromoted(CategoryDesignFork) {
		t.Error("redirect must demote")
	}

	// Persistence round-trip.
	for i := 0; i < PromoteThreshold; i++ {
		p.RecordOutcome(CategoryUnderspecified, OutcomeApproved)
	}
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	p2, _ := LoadPromotions(dir, "alice")
	if !p2.IsPromoted(CategoryUnderspecified) {
		t.Error("promotion should persist across reload")
	}
	if got := p2.PromotedList(); len(got) != 1 || got[0] != string(CategoryUnderspecified) {
		t.Errorf("PromotedList = %v, want [Underspecified]", got)
	}

	// Reset re-enables pausing.
	p2.Reset(CategoryUnderspecified)
	if p2.IsPromoted(CategoryUnderspecified) {
		t.Error("Reset must clear the promotion")
	}
}

func TestGuardrailCategoriesNeverPromote(t *testing.T) {
	p := &Promotions{User: "x", Categories: map[string]*CategoryTrust{}}
	for _, cat := range []PauseCategory{CategoryIrreversible, CategoryHardCap, CategoryUnknown, CategoryVerifyStuck, CategoryBlocked} {
		for i := 0; i < PromoteThreshold*3; i++ {
			p.RecordOutcome(cat, OutcomeApproved)
		}
		if p.IsPromoted(cat) {
			t.Errorf("%q must never promote, even after many approvals", cat)
		}
		if _, tracked := p.Categories[string(cat)]; tracked {
			t.Errorf("%q should not even be tracked", cat)
		}
	}
}
