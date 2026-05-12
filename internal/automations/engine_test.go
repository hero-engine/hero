package automations

import (
	"testing"
)

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		filter  map[string]string
		payload map[string]string
		want    bool
	}{
		{nil, nil, true},
		{map[string]string{"type": "Bug"}, map[string]string{"type": "Bug"}, true},
		{map[string]string{"type": "Bug"}, map[string]string{"type": "Feature"}, false},
		{map[string]string{"type": "Bug", "priority": "Critical"}, map[string]string{"type": "Bug", "priority": "Critical"}, true},
		{map[string]string{"type": "Bug", "priority": "Critical"}, map[string]string{"type": "Bug"}, false},
		{map[string]string{}, map[string]string{"anything": "goes"}, true},
	}

	for _, tt := range tests {
		got := matchesFilter(tt.filter, tt.payload)
		if got != tt.want {
			t.Errorf("matchesFilter(%v, %v) = %v, want %v", tt.filter, tt.payload, got, tt.want)
		}
	}
}

func TestResolveArgs(t *testing.T) {
	tests := []struct {
		template string
		payload  map[string]string
		want     string
	}{
		{"{{tracker_id}}", map[string]string{"tracker_id": "PROJ-123"}, "PROJ-123"},
		{"{{issue.tracker_id}}", map[string]string{"tracker_id": "PROJ-123"}, "PROJ-123"},
		{"diagnose {{tracker_id}} --priority {{priority}}", map[string]string{"tracker_id": "BUG-42", "priority": "high"}, "diagnose BUG-42 --priority high"},
		{"no vars here", nil, "no vars here"},
	}

	for _, tt := range tests {
		got := ResolveArgs(tt.template, tt.payload)
		if got != tt.want {
			t.Errorf("ResolveArgs(%q, %v) = %q, want %q", tt.template, tt.payload, got, tt.want)
		}
	}
}

func TestMatch(t *testing.T) {
	e := &Engine{
		automations: []AutomationStatus{
			{Config: AutomationConfig{
				Name:    "auto-diagnose",
				Enabled: true,
				Trigger: TriggerConfig{Type: "tracker", Event: "issue_created", Filter: map[string]string{"type": "Bug"}},
				Action:  ActionConfig{Command: "diagnose", Args: "{{tracker_id}}"},
			}},
			{Config: AutomationConfig{
				Name:    "weekly-check",
				Enabled: true,
				Trigger: TriggerConfig{Type: "schedule", Event: "0 9 * * 1"},
				Action:  ActionConfig{Command: "check"},
			}},
			{Config: AutomationConfig{
				Name:    "disabled",
				Enabled: false,
				Trigger: TriggerConfig{Type: "tracker", Event: "issue_created"},
				Action:  ActionConfig{Command: "diagnose"},
			}},
		},
	}

	// Should match auto-diagnose
	matches := e.Match("tracker", "issue_created", map[string]string{"type": "Bug", "tracker_id": "PROJ-1"})
	if len(matches) != 1 || matches[0].Name != "auto-diagnose" {
		t.Errorf("expected 1 match (auto-diagnose), got %d", len(matches))
	}

	// Should not match — wrong event
	matches = e.Match("tracker", "issue_closed", map[string]string{"type": "Bug"})
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}

	// Should not match disabled
	matches = e.Match("tracker", "issue_created", map[string]string{})
	if len(matches) != 0 {
		t.Errorf("expected 0 matches (disabled filtered), got %d", len(matches))
	}
}
