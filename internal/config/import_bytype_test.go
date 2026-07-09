package config

import (
	"encoding/json"
	"testing"
)

// TestEffectiveFilterForType_MergesOverBase asserts by_type[type]
// overrides base_filter per non-empty field while base fields survive
// where the per-type filter leaves them empty.
func TestEffectiveFilterForType_MergesOverBase(t *testing.T) {
	ic := &ImportConfig{
		BaseFilter: &ImportFilter{Status: "Open", Assignee: "alice"},
		ByType: map[string]*ImportFilter{
			"bug":   {Priority: "High", Status: "New"},
			"story": {Labels: []string{"ready"}},
		},
	}

	bug := ic.EffectiveFilterForType("bug")
	if bug.Status != "New" { // by_type overrides base
		t.Errorf("bug.Status = %q, want New", bug.Status)
	}
	if bug.Priority != "High" {
		t.Errorf("bug.Priority = %q, want High", bug.Priority)
	}
	if bug.Assignee != "alice" { // base survives
		t.Errorf("bug.Assignee = %q, want alice (from base)", bug.Assignee)
	}

	story := ic.EffectiveFilterForType("story")
	if story.Status != "Open" { // base survives (no override)
		t.Errorf("story.Status = %q, want Open (from base)", story.Status)
	}
	if len(story.Labels) != 1 || story.Labels[0] != "ready" {
		t.Errorf("story.Labels = %v, want [ready]", story.Labels)
	}
}

// TestEffectiveFilterForType_CaseInsensitive matches the type key
// case-insensitively.
func TestEffectiveFilterForType_CaseInsensitive(t *testing.T) {
	ic := &ImportConfig{
		ByType: map[string]*ImportFilter{"Bug": {Priority: "Critical"}},
	}
	if got := ic.EffectiveFilterForType("bug"); got.Priority != "Critical" {
		t.Errorf("Priority = %q, want Critical (case-insensitive key match)", got.Priority)
	}
}

// TestEffectiveFilterForType_NoByType falls back to the base filter
// (with its defaults) when no by_type entry exists — the additive
// guarantee for unconfigured types.
func TestEffectiveFilterForType_NoByType(t *testing.T) {
	ic := &ImportConfig{BaseFilter: &ImportFilter{Status: "Open"}}
	got := ic.EffectiveFilterForType("epic")
	base := ic.EffectiveBaseFilter()
	if got.Status != base.Status || got.IssueType != base.IssueType {
		t.Errorf("no-by_type filter should equal base; got %+v, base %+v", got, base)
	}
}

// TestHasByType reports presence correctly, nil-safe.
func TestHasByType(t *testing.T) {
	var nilIC *ImportConfig
	if nilIC.HasByType() {
		t.Error("nil ImportConfig should report HasByType()=false")
	}
	if (&ImportConfig{}).HasByType() {
		t.Error("empty ImportConfig should report HasByType()=false")
	}
	if !(&ImportConfig{ByType: map[string]*ImportFilter{"bug": {}}}).HasByType() {
		t.Error("configured by_type should report HasByType()=true")
	}
}

// TestByType_JSONRoundTrip confirms the by_type block unmarshals from
// hero.json shape.
func TestByType_JSONRoundTrip(t *testing.T) {
	var cfg Config
	if err := json.Unmarshal([]byte(`{
	  "import": {
	    "base_filter": { "status": "Open" },
	    "by_type": {
	      "bug":   { "priority": "High", "status": "Open" },
	      "epic":  { "status": "Active" }
	    }
	  }
	}`), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !cfg.Import.HasByType() {
		t.Fatal("expected by_type to be parsed")
	}
	if cfg.Import.ByType["bug"].Priority != "High" {
		t.Errorf("bug priority = %q, want High", cfg.Import.ByType["bug"].Priority)
	}
	if cfg.Import.ByType["epic"].Status != "Active" {
		t.Errorf("epic status = %q, want Active", cfg.Import.ByType["epic"].Status)
	}
}
