package drive

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestParseMode(t *testing.T) {
	cases := map[string]AutonomyMode{
		"":           Supervised,
		"supervised": Supervised,
		"nonsense":   Supervised,
		"guided":     Guided,
		"GUIDED":     Guided,
		"autonomous": Autonomous,
		" Autonomous ": Autonomous,
	}
	for in, want := range cases {
		if got := ParseMode(in); got != want {
			t.Errorf("ParseMode(%q) = %v, want %v", in, got, want)
		}
	}
}

// base is a context that, on its own, would proceed in Guided/Autonomous.
func base() RunContext {
	return RunContext{
		VerifyVerdict:        "PASS",
		VerifyStuckThreshold: 2,
		NextScore:            80,
		ScoreThreshold:       60,
		ActionClassified:     true,
		HardCap:              0,
	}
}

func TestNeedsMeTaxonomy(t *testing.T) {
	at := &spec.Spec{Slug: "next-spec"}
	tests := []struct {
		name     string
		mutate   func(*RunContext)
		mode     AutonomyMode
		wantPause bool
		wantCat  PauseCategory
	}{
		{"guided proceeds on clean", func(c *RunContext) {}, Guided, false, CategoryNone},
		{"autonomous proceeds on clean", func(c *RunContext) {}, Autonomous, false, CategoryNone},
		{"underspecified pauses", func(c *RunContext) { c.NextScore = 40 }, Guided, true, CategoryUnderspecified},
		{"score unknown does not underspecify", func(c *RunContext) { c.NextScore = -1 }, Guided, false, CategoryNone},
		{"design fork pauses", func(c *RunContext) { c.DesignFork = true }, Guided, true, CategoryDesignFork},
		{"blocked pauses", func(c *RunContext) { c.Blocked = true }, Guided, true, CategoryBlocked},
		{"ambiguous pick pauses", func(c *RunContext) { c.AmbiguousPick = true }, Guided, true, CategoryAmbiguousPick},
		{"verify fail under threshold = rework (proceed)", func(c *RunContext) { c.VerifyVerdict = "FAIL"; c.VerifyFailCount = 1 }, Guided, false, CategoryNone},
		{"verify fail at threshold = stuck", func(c *RunContext) { c.VerifyVerdict = "FAIL"; c.VerifyFailCount = 2 }, Guided, true, CategoryVerifyStuck},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := base()
			tc.mutate(&ctx)
			got := NeedsMe(at, ctx, tc.mode)
			if got.Proceed == tc.wantPause {
				t.Fatalf("Proceed=%v, want pause=%v (%+v)", got.Proceed, tc.wantPause, got)
			}
			if tc.wantPause && got.Category != tc.wantCat {
				t.Errorf("Category=%q, want %q", got.Category, tc.wantCat)
			}
		})
	}
}

func TestNeedsMeSupervisedAlwaysPauses(t *testing.T) {
	at := &spec.Spec{Slug: "x"}
	got := NeedsMe(at, base(), Supervised) // base() would proceed in guided
	if got.Proceed {
		t.Fatal("supervised must pause at every boundary")
	}
	if got.Category != CategorySupervised {
		t.Errorf("Category=%q, want Supervised", got.Category)
	}
}

func TestNeedsMeIrreversibleAlwaysPausesEveryMode(t *testing.T) {
	at := &spec.Spec{Slug: "deploy"}
	for _, mode := range []AutonomyMode{Supervised, Guided, Autonomous} {
		ctx := base()
		ctx.Irreversible = true
		// Even with a promotion that would otherwise auto-proceed:
		ctx.Promoted = func(PauseCategory) bool { return true }
		got := NeedsMe(at, ctx, mode)
		if got.Proceed {
			t.Fatalf("mode %v: irreversible action must pause", mode)
		}
		if got.Category != CategoryIrreversible {
			t.Errorf("mode %v: Category=%q, want Irreversible", mode, got.Category)
		}
	}
}

func TestNeedsMeUnknownActionPauses(t *testing.T) {
	at := &spec.Spec{Slug: "x"}
	ctx := base()
	ctx.ActionClassified = false
	got := NeedsMe(at, ctx, Autonomous)
	if got.Proceed || got.Category != CategoryUnknown {
		t.Fatalf("unclassified action must pause as Unknown, got %+v", got)
	}
}

func TestNeedsMeHardCapAndBoundary(t *testing.T) {
	at := &spec.Spec{Slug: "x"}
	// Consecutive-proceed cap.
	ctx := base()
	ctx.HardCap = 3
	ctx.ConsecutiveProceeds = 3
	if got := NeedsMe(at, ctx, Autonomous); got.Proceed || got.Category != CategoryHardCap {
		t.Fatalf("hard cap must pause, got %+v", got)
	}
	// Initiative boundary.
	ctx2 := base()
	ctx2.AtInitiativeBoundary = true
	if got := NeedsMe(at, ctx2, Autonomous); got.Proceed || got.Category != CategoryHardCap {
		t.Fatalf("initiative boundary must pause, got %+v", got)
	}
}

func TestNeedsMeAutonomousPromotionProceeds(t *testing.T) {
	at := &spec.Spec{Slug: "x"}
	ctx := base()
	ctx.NextScore = 40 // would pause Underspecified
	ctx.Promoted = func(c PauseCategory) bool { return c == CategoryUnderspecified }

	// Autonomous + promoted => proceed.
	if got := NeedsMe(at, ctx, Autonomous); !got.Proceed {
		t.Errorf("autonomous with promoted Underspecified should proceed, got %+v", got)
	}
	// Guided ignores promotions => still pauses.
	if got := NeedsMe(at, ctx, Guided); got.Proceed {
		t.Errorf("guided must ignore promotions and pause, got %+v", got)
	}
	// Non-promotable category never auto-proceeds even if Promoted says yes.
	ctx2 := base()
	ctx2.Blocked = true
	ctx2.Promoted = func(PauseCategory) bool { return true }
	if got := NeedsMe(at, ctx2, Autonomous); got.Proceed {
		t.Errorf("Blocked is non-promotable; must pause, got %+v", got)
	}
}

func TestPromotableScope(t *testing.T) {
	promotable := []PauseCategory{CategoryDesignFork, CategoryUnderspecified, CategoryAmbiguousPick, CategorySeamDetected}
	for _, c := range promotable {
		if !c.Promotable() {
			t.Errorf("%q should be promotable", c)
		}
	}
	never := []PauseCategory{CategoryIrreversible, CategoryHardCap, CategoryUnknown, CategoryVerifyStuck, CategoryBlocked, CategorySupervised, CategorySeamCollision}
	for _, c := range never {
		if c.Promotable() {
			t.Errorf("%q must never be promotable", c)
		}
	}
}

// TestNeedsMeSeamDetectedPromotable: a detected (heuristic, unauthored) seam
// pauses in Guided naming the overlap + in-flight spec, is promotable in
// Autonomous, and always yields to an authored SeamCollision when both fire.
func TestNeedsMeSeamDetectedPromotable(t *testing.T) {
	at := &spec.Spec{Slug: "candidate"}
	mk := func() RunContext {
		c := base()
		c.SeamDetected = true
		c.SeamDetectedSlug = "in-flight-peer"
		c.SeamDetectedFiles = []string{"src/shared.go"}
		return c
	}

	// Guided pauses and names the overlapping file + in-flight spec.
	got := NeedsMe(at, mk(), Guided)
	if got.Proceed || got.Category != CategorySeamDetected {
		t.Fatalf("guided must pause SeamDetected, got %+v", got)
	}
	if !strings.Contains(got.Reason, "in-flight-peer") || !strings.Contains(got.Reason, "src/shared.go") {
		t.Errorf("reason=%q, want it to name the in-flight spec and file", got.Reason)
	}

	// Autonomous + promoted proceeds (the deliberate contrast with the
	// non-promotable authored SeamCollision).
	promo := mk()
	promo.Promoted = func(cat PauseCategory) bool { return cat == CategorySeamDetected }
	if got := NeedsMe(at, promo, Autonomous); !got.Proceed {
		t.Errorf("autonomous+promoted should proceed past SeamDetected, got %+v", got)
	}
	// Autonomous WITHOUT the promotion still pauses.
	if got := NeedsMe(at, mk(), Autonomous); got.Proceed {
		t.Errorf("autonomous without promotion must pause SeamDetected, got %+v", got)
	}

	// Authored SeamCollision wins when both signals are set, even with a
	// blanket promotion.
	both := mk()
	both.SeamBlocked = true
	both.SeamConflictSlug = "authored-peer"
	both.Promoted = func(PauseCategory) bool { return true }
	if got := NeedsMe(at, both, Autonomous); got.Proceed || got.Category != CategorySeamCollision {
		t.Errorf("authored SeamCollision must win over detected, got %+v", got)
	}
}

// TestNeedsMeSeamCollisionPausesEveryMode: a seam collision is a real
// obstacle — it pauses in every mode, and in Guided/Autonomous surfaces the
// SeamCollision category naming the in-flight spec even when a promotion would
// otherwise auto-proceed.
func TestNeedsMeSeamCollisionPausesEveryMode(t *testing.T) {
	at := &spec.Spec{Slug: "candidate"}
	for _, mode := range []AutonomyMode{Supervised, Guided, Autonomous} {
		ctx := base()
		ctx.SeamBlocked = true
		ctx.SeamConflictSlug = "in-flight-peer"
		// A promotion that would auto-proceed a promotable category must not
		// relax SeamCollision.
		ctx.Promoted = func(PauseCategory) bool { return true }
		got := NeedsMe(at, ctx, mode)
		if got.Proceed {
			t.Fatalf("mode %v: seam collision must pause", mode)
		}
		if mode == Supervised {
			continue // supervised pauses first with its own category
		}
		if got.Category != CategorySeamCollision {
			t.Errorf("mode %v: category=%q, want SeamCollision", mode, got.Category)
		}
		if !strings.Contains(got.Reason, "in-flight-peer") {
			t.Errorf("mode %v: reason=%q, want it to name in-flight-peer", mode, got.Reason)
		}
	}
}
