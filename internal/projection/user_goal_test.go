package projection

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/handoff"
)

// TestUserHandoffMD_SessionGoalAboveLastAsk proves the goal renders in a
// `## Session goal` section ABOVE `## Last user ask`.
func TestUserHandoffMD_SessionGoalAboveLastAsk(t *testing.T) {
	store := openTestStore(t)
	if err := handoff.RecordGoal(store, "repo-x", handoff.SessionGoal{
		User: "alice", Text: "GOAL_durable_intent", Source: handoff.GoalSourceMarker,
	}); err != nil {
		t.Fatal(err)
	}
	if err := handoff.RecordAsk(store, "repo-x", handoff.UserAsk{
		User: "alice", Text: "ASK_latest_refinement",
	}); err != nil {
		t.Fatal(err)
	}
	body, err := UserHandoffMD(store, UserHandoffOptions{User: "alice", RepoKey: "repo-x"})
	if err != nil {
		t.Fatalf("UserHandoffMD: %v", err)
	}
	goalIdx := strings.Index(body, "## Session goal")
	askIdx := strings.Index(body, "## Last user ask")
	if goalIdx < 0 {
		t.Fatalf("no ## Session goal section:\n%s", body)
	}
	if goalIdx > askIdx {
		t.Errorf("Session goal must render ABOVE Last user ask (goal=%d ask=%d):\n%s", goalIdx, askIdx, body)
	}
	if !strings.Contains(body, "GOAL_durable_intent") {
		t.Errorf("goal text missing:\n%s", body)
	}
}

// TestUserHandoffMD_GoalFramingBySource proves auto sources are softly
// framed and marker/manual are asserted.
func TestUserHandoffMD_GoalFramingBySource(t *testing.T) {
	cases := []struct {
		source     string
		wantPrefix string // substring that must appear; "" means asserted (no soft prefix)
		soft       bool
	}{
		{handoff.GoalSourceAutoWindow, "Session opened with —", true},
		{handoff.GoalSourceAutoEmbed, "Likely goal —", true},
		{handoff.GoalSourceMarker, "", false},
		{handoff.GoalSourceManual, "", false},
	}
	for _, c := range cases {
		store := openTestStore(t)
		if err := handoff.RecordGoal(store, "repo-x", handoff.SessionGoal{
			User: "alice", Text: "GOAL_" + c.source, Source: c.source,
		}); err != nil {
			t.Fatal(err)
		}
		body, err := UserHandoffMD(store, UserHandoffOptions{User: "alice", RepoKey: "repo-x"})
		if err != nil {
			t.Fatalf("UserHandoffMD: %v", err)
		}
		goalSection := h2Section(body, "## Session goal")
		if c.soft {
			if !strings.Contains(goalSection, c.wantPrefix) {
				t.Errorf("source %q should be soft-framed with %q:\n%s", c.source, c.wantPrefix, goalSection)
			}
		} else {
			if strings.Contains(goalSection, "Session opened with") || strings.Contains(goalSection, "Likely goal") {
				t.Errorf("source %q should be asserted (no soft prefix):\n%s", c.source, goalSection)
			}
		}
	}
}

// TestUserHandoffMD_GoalOmittedWhenEqualToLastAsk proves the goal section
// is suppressed when the goal text equals the last ask (single-message
// session).
func TestUserHandoffMD_GoalOmittedWhenEqualToLastAsk(t *testing.T) {
	store := openTestStore(t)
	same := "the one and only message this session"
	_ = handoff.RecordGoal(store, "repo-x", handoff.SessionGoal{User: "alice", Text: same, Source: handoff.GoalSourceAutoWindow})
	_ = handoff.RecordAsk(store, "repo-x", handoff.UserAsk{User: "alice", Text: same})
	body, err := UserHandoffMD(store, UserHandoffOptions{User: "alice", RepoKey: "repo-x"})
	if err != nil {
		t.Fatalf("UserHandoffMD: %v", err)
	}
	if strings.Contains(body, "## Session goal") {
		t.Errorf("goal section should be omitted when equal to last ask:\n%s", body)
	}
}

// TestUserHandoffMD_GoalOmittedWhenEmpty proves no goal → no section.
func TestUserHandoffMD_GoalOmittedWhenEmpty(t *testing.T) {
	store := openTestStore(t)
	_ = handoff.RecordAsk(store, "repo-x", handoff.UserAsk{User: "alice", Text: "just an ask"})
	body, err := UserHandoffMD(store, UserHandoffOptions{User: "alice", RepoKey: "repo-x"})
	if err != nil {
		t.Fatalf("UserHandoffMD: %v", err)
	}
	if strings.Contains(body, "## Session goal") {
		t.Errorf("goal section should be omitted when no goal:\n%s", body)
	}
}

// TestNextMD_DoesNotRenderSessionGoal is the mandatory per-user-only
// assertion: the project-level NextMD must NEVER carry the session goal
// (it is user state, not project state).
func TestNextMD_DoesNotRenderSessionGoal(t *testing.T) {
	store := openTestStore(t)
	if err := handoff.RecordGoal(store, "repo-x", handoff.SessionGoal{
		User: "alice", Text: "GOAL_must_not_leak_into_project_next", Source: handoff.GoalSourceManual,
	}); err != nil {
		t.Fatal(err)
	}
	body, err := NextMD(store, NextMDOptions{RepoKey: "repo-x"})
	if err != nil {
		t.Fatalf("NextMD: %v", err)
	}
	if strings.Contains(body, "GOAL_must_not_leak_into_project_next") {
		t.Errorf("project NEXT.md leaked the per-user session goal:\n%s", body)
	}
	if strings.Contains(body, "Session goal") {
		t.Errorf("project NEXT.md must not carry a Session goal section:\n%s", body)
	}
}

// h2Section returns the body of a `## <header>` section up to the next
// `## ` header or EOF.
func h2Section(doc, header string) string {
	idx := strings.Index(doc, header)
	if idx < 0 {
		return ""
	}
	rest := doc[idx+len(header):]
	if next := strings.Index(rest, "\n## "); next >= 0 {
		rest = rest[:next]
	}
	return rest
}
