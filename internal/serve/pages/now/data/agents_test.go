package data

import "testing"

func TestLoadAgents_EmptyWorkspace(t *testing.T) {
	got := LoadAgents(AgentsInputs{})
	if got.Running != nil {
		t.Errorf("expected nil Running on empty input, got %+v", got.Running)
	}
	if got.Today.Sessions == nil {
		t.Errorf("Today.Sessions must be non-nil (empty slice ok)")
	}
	if got.RunningCount != 0 {
		t.Errorf("RunningCount = %d, want 0", got.RunningCount)
	}
}

func TestShortenEventType(t *testing.T) {
	cases := map[string]string{
		"spec_created":      "created",
		"spec_updated":      "updated",
		"delivery_complete": "delivery_complete", // fall-through default
	}
	for in, want := range cases {
		if got := shortenEventType(in); got != want {
			t.Errorf("shortenEventType(%q) = %q, want %q", in, got, want)
		}
	}
}
