package digest

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/handoff"
)

// TestHandoffSection_GoalAboveLastAsk proves the Goal line renders ABOVE
// the Last ask line in "Where you left off", framed by source.
func TestHandoffSection_GoalAboveLastAsk(t *testing.T) {
	store := openTestStore(t)
	if err := handoff.RecordGoal(store, "test", handoff.SessionGoal{
		User: "alice", Domain: "engineering", Text: "GOAL stop credential-stuffing on login", Source: handoff.GoalSourceMarker,
	}); err != nil {
		t.Fatalf("RecordGoal: %v", err)
	}
	if err := handoff.RecordAsk(store, "test", handoff.UserAsk{
		User: "alice", Domain: "engineering", Text: "ASK cap it at 5 a minute",
	}); err != nil {
		t.Fatalf("RecordAsk: %v", err)
	}
	sec, err := handoffSection(store, Options{RepoKey: "test", User: "alice", Domain: "engineering"}, 300)
	if err != nil {
		t.Fatalf("handoffSection: %v", err)
	}
	joined := strings.Join(sec.Lines, "\n")
	goalIdx := strings.Index(joined, "Goal: GOAL stop credential-stuffing")
	// With a goal above it, the ask line reads as the refinement ("Latest:").
	askIdx := strings.Index(joined, "Latest: ASK cap it at 5 a minute")
	if goalIdx < 0 {
		t.Fatalf("no Goal line:\n%s", joined)
	}
	if askIdx < 0 {
		t.Fatalf("no Latest line:\n%s", joined)
	}
	if goalIdx > askIdx {
		t.Errorf("Goal must render above the latest-ask line (goal=%d ask=%d):\n%s", goalIdx, askIdx, joined)
	}
}

// TestHandoffSection_GoalSoftFraming proves an auto-window goal is softly
// framed in the brief.
func TestHandoffSection_GoalSoftFraming(t *testing.T) {
	store := openTestStore(t)
	_ = handoff.RecordGoal(store, "test", handoff.SessionGoal{
		User: "alice", Domain: "engineering", Text: "add login rate limiting", Source: handoff.GoalSourceAutoWindow,
	})
	sec, err := handoffSection(store, Options{RepoKey: "test", User: "alice", Domain: "engineering"}, 300)
	if err != nil {
		t.Fatalf("handoffSection: %v", err)
	}
	joined := strings.Join(sec.Lines, "\n")
	if !strings.Contains(joined, "Goal (session opened with): add login rate limiting") {
		t.Errorf("auto-window goal not softly framed:\n%s", joined)
	}
}

// TestHandoffSection_GoalOmittedWhenEqualToAsk proves the Goal line is
// suppressed when goal == latest ask.
func TestHandoffSection_GoalOmittedWhenEqualToAsk(t *testing.T) {
	store := openTestStore(t)
	same := "only one message this session"
	_ = handoff.RecordGoal(store, "test", handoff.SessionGoal{User: "alice", Domain: "engineering", Text: same, Source: handoff.GoalSourceAutoWindow})
	_ = handoff.RecordAsk(store, "test", handoff.UserAsk{User: "alice", Domain: "engineering", Text: same})
	sec, err := handoffSection(store, Options{RepoKey: "test", User: "alice", Domain: "engineering"}, 300)
	if err != nil {
		t.Fatalf("handoffSection: %v", err)
	}
	joined := strings.Join(sec.Lines, "\n")
	if strings.Contains(joined, "Goal") {
		t.Errorf("Goal line should be omitted when equal to last ask:\n%s", joined)
	}
	if !strings.Contains(joined, "Last ask: "+same) {
		t.Errorf("last ask should still render:\n%s", joined)
	}
}

// TestHandoffSection_GoalEmptyNoLine proves no goal → no Goal line.
func TestHandoffSection_GoalEmptyNoLine(t *testing.T) {
	store := openTestStore(t)
	_ = handoff.RecordAsk(store, "test", handoff.UserAsk{User: "alice", Domain: "engineering", Text: "just an ask"})
	sec, err := handoffSection(store, Options{RepoKey: "test", User: "alice", Domain: "engineering"}, 300)
	if err != nil {
		t.Fatalf("handoffSection: %v", err)
	}
	joined := strings.Join(sec.Lines, "\n")
	if strings.Contains(joined, "Goal") {
		t.Errorf("no Goal line expected when no goal recorded:\n%s", joined)
	}
}

// TestHandoffSection_EmptyUserUnaffected proves User=="" → section
// unaffected (no goal query, empty section).
func TestHandoffSection_EmptyUserUnaffected(t *testing.T) {
	store := openTestStore(t)
	_ = handoff.RecordGoal(store, "test", handoff.SessionGoal{User: "alice", Domain: "engineering", Text: "x", Source: handoff.GoalSourceManual})
	sec, err := handoffSection(store, Options{RepoKey: "test", User: ""}, 300)
	if err != nil {
		t.Fatalf("handoffSection: %v", err)
	}
	if len(sec.Lines) != 0 {
		t.Errorf("empty User should yield no lines, got %v", sec.Lines)
	}
}
