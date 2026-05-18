package data

import (
	"testing"
	"time"
)

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

func TestLoadAgents_PopulatesRunningFromLedger(t *testing.T) {
	started := time.Now().Add(-12 * time.Minute)
	got := LoadAgents(AgentsInputs{
		LiveSessions: func() []SessionRow {
			return []SessionRow{
				{
					ID:        "sess-1",
					Agent:     "claude-opus",
					Spec:      "hero-now-home-followups",
					Status:    "live",
					StartedAt: started,
					CostUSD:   0.42,
					ToolCalls: 7,
				},
			}
		},
	})

	if got.RunningCount != 1 {
		t.Fatalf("RunningCount = %d, want 1", got.RunningCount)
	}
	if got.Running == nil {
		t.Fatalf("Running == nil; want populated from ledger")
	}
	if got.Running.Name != "claude-opus" {
		t.Errorf("Running.Name = %q, want %q", got.Running.Name, "claude-opus")
	}
	if got.Running.SpecSlug != "hero-now-home-followups" {
		t.Errorf("Running.SpecSlug = %q, want %q", got.Running.SpecSlug, "hero-now-home-followups")
	}
	if got.Running.SpecHref != "/work/spec/hero-now-home-followups" {
		t.Errorf("Running.SpecHref = %q", got.Running.SpecHref)
	}
	if got.Running.OpenHref != "/agents/session/sess-1" {
		t.Errorf("Running.OpenHref = %q", got.Running.OpenHref)
	}
	if got.Running.ToolCalls != 7 {
		t.Errorf("Running.ToolCalls = %d, want 7", got.Running.ToolCalls)
	}
	if got.Running.Cost != "$0.42" {
		t.Errorf("Running.Cost = %q, want %q", got.Running.Cost, "$0.42")
	}
}

func TestLoadAgents_LedgerEmptyKeepsEmptyState(t *testing.T) {
	got := LoadAgents(AgentsInputs{
		LiveSessions: func() []SessionRow { return nil },
	})
	if got.Running != nil {
		t.Errorf("expected nil Running when ledger empty, got %+v", got.Running)
	}
	if got.RunningCount != 0 {
		t.Errorf("RunningCount = %d, want 0", got.RunningCount)
	}
}

func TestLoadAgents_LedgerFiltersDoneAndFailed(t *testing.T) {
	got := LoadAgents(AgentsInputs{
		LiveSessions: func() []SessionRow {
			return []SessionRow{
				{ID: "a", Agent: "x", Status: "done"},
				{ID: "b", Agent: "y", Status: "failed"},
				{ID: "c", Agent: "z", Status: "live"},
			}
		},
	})
	if got.RunningCount != 1 {
		t.Errorf("RunningCount = %d, want 1 (only the live row qualifies)", got.RunningCount)
	}
	if got.Running == nil || got.Running.Name != "z" {
		t.Errorf("expected Running.Name = z, got %+v", got.Running)
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
