package tracker

import (
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

// Tests for the slice-5 size-mapping plumbing:
//   - per-adapter defaults exist and produce the documented bands
//   - MapSize / ReverseMapSize round-trip cleanly for known tiers
//   - out-of-band tracker values fail explicitly (not silently)
//   - PlanSizePull / PlanSizePush implement the non-destructive rules
//
// No live tracker calls; the adapter instances here are constructed
// without HTTP because MapSize/ReverseMapSize do not touch the wire.

func TestSupportsHierarchy_Defaults(t *testing.T) {
	t.Setenv("X", "tok")
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	if !j.SupportsHierarchy() {
		t.Error("jira should report supports_hierarchy=true (epics)")
	}
	l, _ := newLinear("ENG", "tok", "")
	if !l.SupportsHierarchy() {
		t.Error("linear should report supports_hierarchy=true (projects)")
	}
	g, _ := newGitHub("acme/widgets", "tok", "")
	if g.SupportsHierarchy() {
		t.Error("github should report supports_hierarchy=false (basic issues)")
	}
}

func TestTypeSupportsHierarchy(t *testing.T) {
	cases := []struct {
		typ  string
		want bool
	}{
		{"jira", true},
		{"linear", true},
		{"github", false},
		{"none", false},
		{"", false},
		{"trello", false},
	}
	for _, c := range cases {
		if got := TypeSupportsHierarchy(c.typ); got != c.want {
			t.Errorf("TypeSupportsHierarchy(%q) = %v, want %v", c.typ, got, c.want)
		}
	}
}

func TestJira_MapSize_RoundTrip(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	for _, tier := range []string{"trivial", "small", "medium", "large", "x-large", "giant"} {
		raw, err := j.MapSize(tier)
		if err != nil {
			t.Errorf("MapSize(%q) errored: %v", tier, err)
			continue
		}
		gotTier, err := j.ReverseMapSize(raw)
		if err != nil {
			t.Errorf("ReverseMapSize(%q) errored: %v", raw, err)
			continue
		}
		if gotTier != tier {
			t.Errorf("round-trip %q → %q → %q", tier, raw, gotTier)
		}
	}
}

func TestLinear_MapSize_RoundTrip(t *testing.T) {
	l, _ := newLinear("ENG", "tok", "")
	for _, tier := range []string{"trivial", "small", "medium", "large", "x-large", "giant"} {
		raw, err := l.MapSize(tier)
		if err != nil {
			t.Errorf("MapSize(%q) errored: %v", tier, err)
			continue
		}
		gotTier, _ := l.ReverseMapSize(raw)
		if gotTier != tier {
			t.Errorf("round-trip %q → %q → %q", tier, raw, gotTier)
		}
	}
}

func TestGitHub_MapSize_LabelPrefix(t *testing.T) {
	g, _ := newGitHub("acme/widgets", "tok", "")
	raw, err := g.MapSize("large")
	if err != nil {
		t.Fatalf("MapSize: %v", err)
	}
	if raw != "size/large" {
		t.Errorf("MapSize(large) = %q, want size/large", raw)
	}
	tier, err := g.ReverseMapSize("size/x-large")
	if err != nil || tier != "x-large" {
		t.Errorf("ReverseMapSize(size/x-large) = (%q, %v), want (x-large, nil)", tier, err)
	}
	// Bare tier (no prefix) should also work.
	tier, err = g.ReverseMapSize("giant")
	if err != nil || tier != "giant" {
		t.Errorf("ReverseMapSize(giant) = (%q, %v), want (giant, nil)", tier, err)
	}
	// Unknown tier — fail loud.
	if _, err := g.ReverseMapSize("size/nope"); err == nil {
		t.Error("expected ReverseMapSize(size/nope) to error")
	}
}

func TestMapSize_UnknownTier(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	if _, err := j.MapSize("enormous"); err == nil {
		t.Error("expected MapSize(enormous) to error")
	}
}

func TestReverseMapSize_OutOfBand(t *testing.T) {
	// 7 points falls between the medium [3,5] and large [8,8] bands
	// in the default Jira mapping — should not silently bucket. With
	// strict per-band matching, 7 fails to map and the caller treats
	// it as a conflict.
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	_, err := j.ReverseMapSize("7")
	if err == nil {
		t.Error("expected ReverseMapSize(7) to error on out-of-band value")
	}
}

func TestReverseMapSize_GiantUnbounded(t *testing.T) {
	// The giant band has nil max — any value >= 20 should map to giant.
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	for _, v := range []string{"20", "21", "100", "9999"} {
		tier, err := j.ReverseMapSize(v)
		if err != nil {
			t.Errorf("ReverseMapSize(%q) errored: %v", v, err)
			continue
		}
		if tier != "giant" {
			t.Errorf("ReverseMapSize(%q) = %q, want giant", v, tier)
		}
	}
}

func TestExtractTrackerSize_NumericField(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	issue := &Issue{CustomFields: map[string]string{"story_points": "8"}}
	raw, tier := ExtractTrackerSize(j, issue)
	if raw != "8" {
		t.Errorf("raw = %q, want 8", raw)
	}
	if tier != "large" {
		t.Errorf("tier = %q, want large", tier)
	}
}

func TestExtractTrackerSize_Label(t *testing.T) {
	g, _ := newGitHub("acme/widgets", "tok", "")
	issue := &Issue{Labels: []string{"bug", "size/medium", "frontend"}}
	raw, tier := ExtractTrackerSize(g, issue)
	if raw != "size/medium" {
		t.Errorf("raw = %q, want size/medium", raw)
	}
	if tier != "medium" {
		t.Errorf("tier = %q, want medium", tier)
	}
}

func TestExtractTrackerSize_Missing(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	raw, tier := ExtractTrackerSize(j, &Issue{})
	if raw != "" || tier != "" {
		t.Errorf("expected empty, got raw=%q tier=%q", raw, tier)
	}
}

// --- PlanSizePull ---

func TestPlanSizePull_NoMapping_Noop(t *testing.T) {
	// A custom adapter with no shipped default → noop regardless of
	// inputs. We simulate by clearing the default via a stub.
	plan := PlanSizePull(nil, &Issue{}, "")
	if plan.Action != SizeSyncNoop {
		t.Errorf("Action = %s, want noop", plan.Action)
	}
}

func TestPlanSizePull_SeedLocal(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	issue := &Issue{CustomFields: map[string]string{"story_points": "3"}}
	plan := PlanSizePull(j, issue, "")
	if plan.Action != SizeSyncSeedLocal {
		t.Errorf("Action = %s, want seed-local", plan.Action)
	}
	if plan.WriteValue != "medium" {
		t.Errorf("WriteValue = %q, want medium", plan.WriteValue)
	}
}

func TestPlanSizePull_Agree_Noop(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	issue := &Issue{CustomFields: map[string]string{"story_points": "13"}}
	plan := PlanSizePull(j, issue, "x-large")
	if plan.Action != SizeSyncNoop {
		t.Errorf("Action = %s, want noop on agreement", plan.Action)
	}
}

func TestPlanSizePull_Conflict(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	issue := &Issue{CustomFields: map[string]string{"story_points": "8"}} // → large
	plan := PlanSizePull(j, issue, "medium")
	if plan.Action != SizeSyncConflict {
		t.Errorf("Action = %s, want conflict", plan.Action)
	}
	if plan.TrackerTier != "large" {
		t.Errorf("TrackerTier = %q, want large", plan.TrackerTier)
	}
}

func TestPlanSizePull_TrackerEmpty_Noop(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	plan := PlanSizePull(j, &Issue{}, "small")
	if plan.Action != SizeSyncNoop {
		t.Errorf("Action = %s, want noop when tracker has no value", plan.Action)
	}
}

// --- PlanSizePush ---

func TestPlanSizePush_LocalUnset_Noop(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	plan := PlanSizePush(j, &Issue{}, "")
	if plan.Action != SizeSyncNoop {
		t.Errorf("Action = %s, want noop when local size is unset", plan.Action)
	}
}

func TestPlanSizePush_TrackerEmpty_Push(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	plan := PlanSizePush(j, &Issue{}, "large")
	if plan.Action != SizeSyncPushToTracker {
		t.Errorf("Action = %s, want push-to-tracker", plan.Action)
	}
	if plan.WriteValue != "8" {
		t.Errorf("WriteValue = %q, want 8", plan.WriteValue)
	}
}

func TestPlanSizePush_Agree_Noop(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	issue := &Issue{CustomFields: map[string]string{"story_points": "8"}}
	plan := PlanSizePush(j, issue, "large")
	if plan.Action != SizeSyncNoop {
		t.Errorf("Action = %s, want noop on agreement", plan.Action)
	}
}

func TestPlanSizePush_ConflictNonDestructive(t *testing.T) {
	// Tracker carries a human-set value that maps to a different
	// tier than local — must not silently overwrite.
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	issue := &Issue{CustomFields: map[string]string{"story_points": "13"}} // x-large
	plan := PlanSizePush(j, issue, "medium")
	if plan.Action != SizeSyncConflict {
		t.Errorf("Action = %s, want conflict", plan.Action)
	}
	if plan.TrackerTier != "x-large" {
		t.Errorf("TrackerTier = %q, want x-large", plan.TrackerTier)
	}
}

// --- ConfiguredSizeMapping override ---

func TestConfiguredSizeMapping_OverridesDefault(t *testing.T) {
	j, _ := newJira("PROJ", "tok", "u@example.com", "https://j.example.com")
	min2 := float64(2)
	max5 := float64(5)
	j.configuredSizeMapping = &config.SizeMappingConfig{
		Field: "story_points",
		Thresholds: map[string][]*float64{
			"small": {&min2, &max5}, // override: 2-5 → small (default has 2-2)
		},
	}
	// 5 points should now map to small (per the override) instead of
	// the default medium [3, 5].
	tier, err := j.ReverseMapSize("5")
	if err != nil {
		t.Fatalf("ReverseMapSize: %v", err)
	}
	if tier != "small" {
		t.Errorf("ReverseMapSize(5) = %q, want small (per override)", tier)
	}
}
