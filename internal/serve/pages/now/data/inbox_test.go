package data

import (
	"strings"
	"testing"
	"time"
)

func TestLoadInbox_Empty(t *testing.T) {
	got := LoadInbox(InboxInputs{})
	if got.Total != 0 {
		t.Errorf("Total = %d, want 0", got.Total)
	}
	if got.Rows == nil {
		t.Errorf("Rows must be non-nil even when empty (for SSE refresh diff)")
	}
}

func TestLoadInbox_WithProposals(t *testing.T) {
	got := LoadInbox(InboxInputs{
		Proposals: []*ProposalRow{
			{
				ProposalID:  "p-1",
				SessionID:   "s-1",
				SpecSlug:    "per-feature-smoke-coverage",
				Agent:       "claude-opus",
				AnchorValue: "internal/tasks/runner.go",
				EmittedAt:   time.Now().Add(-14 * time.Minute),
			},
		},
	})
	if got.Total != 1 {
		t.Fatalf("Total = %d, want 1", got.Total)
	}
	if got.Rows[0].Kind != "proposal" {
		t.Errorf("Kind = %q, want proposal", got.Rows[0].Kind)
	}
	if !strings.Contains(string(got.Rows[0].Summary), "claude-opus") {
		t.Errorf("Summary missing agent: %q", got.Rows[0].Summary)
	}
}

func TestPrettyAge(t *testing.T) {
	cases := []struct {
		t    time.Time
		want string
	}{
		{time.Time{}, "just now"},
		{time.Now().Add(-30 * time.Second), "just now"},
		{time.Now().Add(-5 * time.Minute), "5m ago"},
		{time.Now().Add(-2 * time.Hour), "2h ago"},
		{time.Now().Add(-3 * 24 * time.Hour), "3d ago"},
	}
	for _, c := range cases {
		if got := prettyAge(c.t); got != c.want {
			t.Errorf("prettyAge(%v) = %q, want %q", c.t, got, c.want)
		}
	}
}
