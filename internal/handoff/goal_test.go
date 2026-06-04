package handoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goalRepo = "repo-goal"

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "alice.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

// TestRecordGoal_RoundTrip records a goal and reads it back via
// LatestGoal, including the source.
func TestRecordGoal_RoundTrip(t *testing.T) {
	store := openTestStore(t)
	if err := RecordGoal(store, goalRepo, SessionGoal{
		User: "alice", Text: "add rate limiting to login", Source: GoalSourceAutoWindow, SessionID: "s1",
	}); err != nil {
		t.Fatalf("RecordGoal: %v", err)
	}
	got, err := LatestGoal(store, "alice", goalRepo, "engineering")
	if err != nil {
		t.Fatalf("LatestGoal: %v", err)
	}
	if got == nil {
		t.Fatal("LatestGoal = nil, want a row")
	}
	if got.Text != "add rate limiting to login" {
		t.Errorf("Text = %q", got.Text)
	}
	if got.Source != GoalSourceAutoWindow {
		t.Errorf("Source = %q, want %q", got.Source, GoalSourceAutoWindow)
	}
	if got.SessionID != "s1" {
		t.Errorf("SessionID = %q", got.SessionID)
	}
}

// TestRecordGoal_EmptyClears proves empty text invalidates the row.
func TestRecordGoal_EmptyClears(t *testing.T) {
	store := openTestStore(t)
	_ = RecordGoal(store, goalRepo, SessionGoal{User: "bob", Text: "x", Source: GoalSourceManual})
	if err := RecordGoal(store, goalRepo, SessionGoal{User: "bob", Text: ""}); err != nil {
		t.Fatalf("RecordGoal(empty): %v", err)
	}
	got, _ := LatestGoal(store, "bob", goalRepo, "engineering")
	if got != nil {
		t.Errorf("after clear, LatestGoal = %+v, want nil", got)
	}
}

// TestRecordGoal_PriorityLadder is the mandatory priority-ladder
// enforcement test: a lower-priority source can never clobber a higher
// one; a higher source overrides a lower one; same-source rewrites
// refresh; and the full ordering auto-window < auto-embed < marker <
// manual holds.
func TestRecordGoal_PriorityLadder(t *testing.T) {
	// pairwise: each (incoming, existing) combination behaves per the
	// ladder. We use distinct text per write so we can tell which won.
	type step struct {
		source string
		text   string
		wantTo string // expected current text after this write
	}

	t.Run("higher overrides lower in ascending order", func(t *testing.T) {
		store := openTestStore(t)
		steps := []step{
			{GoalSourceAutoWindow, "window-goal", "window-goal"},
			{GoalSourceAutoEmbed, "embed-goal", "embed-goal"},
			{GoalSourceMarker, "marker-goal", "marker-goal"},
			{GoalSourceManual, "manual-goal", "manual-goal"},
		}
		for _, s := range steps {
			if err := RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: s.text, Source: s.source}); err != nil {
				t.Fatalf("RecordGoal(%s): %v", s.source, err)
			}
			got, _ := LatestGoal(store, "u", goalRepo, "engineering")
			if got == nil || got.Text != s.wantTo {
				t.Fatalf("after %s write: got %+v, want text %q", s.source, got, s.wantTo)
			}
		}
	})

	t.Run("lower cannot clobber higher", func(t *testing.T) {
		store := openTestStore(t)
		// Establish a manual goal (top of ladder).
		if err := RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "MANUAL", Source: GoalSourceManual}); err != nil {
			t.Fatalf("seed manual: %v", err)
		}
		// Every lower source must be a no-op.
		for _, src := range []string{GoalSourceAutoWindow, GoalSourceAutoEmbed, GoalSourceMarker} {
			if err := RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "LOWER-" + src, Source: src}); err != nil {
				t.Fatalf("RecordGoal(%s): %v", src, err)
			}
			got, _ := LatestGoal(store, "u", goalRepo, "engineering")
			if got == nil || got.Text != "MANUAL" {
				t.Errorf("%s clobbered the manual goal: got %+v, want MANUAL", src, got)
			}
			if got.Source != GoalSourceManual {
				t.Errorf("%s changed the source: got %q, want manual", src, got.Source)
			}
		}
	})

	t.Run("marker overrides window but not manual", func(t *testing.T) {
		store := openTestStore(t)
		_ = RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "WINDOW", Source: GoalSourceAutoWindow})
		_ = RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "MARKER", Source: GoalSourceMarker})
		if got, _ := LatestGoal(store, "u", goalRepo, "engineering"); got == nil || got.Text != "MARKER" {
			t.Fatalf("marker did not override window: %+v", got)
		}
		// Manual then marker: manual must hold.
		_ = RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "MANUAL", Source: GoalSourceManual})
		_ = RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "MARKER2", Source: GoalSourceMarker})
		if got, _ := LatestGoal(store, "u", goalRepo, "engineering"); got == nil || got.Text != "MANUAL" {
			t.Fatalf("marker clobbered manual: %+v", got)
		}
	})

	t.Run("same-source refresh updates text", func(t *testing.T) {
		store := openTestStore(t)
		_ = RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "v1", Source: GoalSourceAutoWindow})
		_ = RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "v2", Source: GoalSourceAutoWindow})
		if got, _ := LatestGoal(store, "u", goalRepo, "engineering"); got == nil || got.Text != "v2" {
			t.Fatalf("same-source refresh did not update: %+v", got)
		}
	})
}

// TestRecordGoal_SameSourceIdempotent proves a same-source rewrite with
// identical content does not create a new row (content-hash stable).
func TestRecordGoal_SameSourceIdempotent(t *testing.T) {
	store := openTestStore(t)
	write := func() {
		if err := RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "stable goal", Source: GoalSourceMarker}); err != nil {
			t.Fatalf("RecordGoal: %v", err)
		}
	}
	write()
	first, _ := LatestGoal(store, "u", goalRepo, "engineering")
	write()
	second, _ := LatestGoal(store, "u", goalRepo, "engineering")
	if first == nil || second == nil {
		t.Fatal("nil goal")
	}
	if first.UpdatedAt != second.UpdatedAt {
		t.Errorf("idempotent rewrite created a new row: %q != %q", first.UpdatedAt, second.UpdatedAt)
	}
}

// TestRecordGoal_DistinctFromUserAsk proves the goal and the last-ask
// are separate singletons that do not clobber each other — the core
// structural invariant of the design.
func TestRecordGoal_DistinctFromUserAsk(t *testing.T) {
	store := openTestStore(t)
	if err := RecordAsk(store, goalRepo, UserAsk{User: "u", Text: "the latest refinement"}); err != nil {
		t.Fatalf("RecordAsk: %v", err)
	}
	if err := RecordGoal(store, goalRepo, SessionGoal{User: "u", Text: "the durable goal", Source: GoalSourceAutoWindow}); err != nil {
		t.Fatalf("RecordGoal: %v", err)
	}
	ask, _ := LatestAsk(store, "u", goalRepo, "engineering")
	goal, _ := LatestGoal(store, "u", goalRepo, "engineering")
	if ask == nil || ask.Text != "the latest refinement" {
		t.Errorf("ask = %+v, want 'the latest refinement'", ask)
	}
	if goal == nil || goal.Text != "the durable goal" {
		t.Errorf("goal = %+v, want 'the durable goal'", goal)
	}
}

// TestGoalSourcePriority pins the ladder ordering.
func TestGoalSourcePriority(t *testing.T) {
	cases := []struct {
		src  string
		want int
	}{
		{GoalSourceAutoWindow, 0},
		{GoalSourceAutoEmbed, 1},
		{GoalSourceMarker, 2},
		{GoalSourceManual, 3},
		{"bogus", 0}, // unknown sorts as the floor — never clobbers
	}
	for _, c := range cases {
		if got := goalSourcePriority(c.src); got != c.want {
			t.Errorf("goalSourcePriority(%q) = %d, want %d", c.src, got, c.want)
		}
	}
	// Strict ascending.
	if !(goalSourcePriority(GoalSourceAutoWindow) < goalSourcePriority(GoalSourceAutoEmbed) &&
		goalSourcePriority(GoalSourceAutoEmbed) < goalSourcePriority(GoalSourceMarker) &&
		goalSourcePriority(GoalSourceMarker) < goalSourcePriority(GoalSourceManual)) {
		t.Error("priority ladder is not strictly ascending")
	}
}

// TestParseGoalSection_RoundTripsSourceFraming proves the ingest parser
// recovers the source framing from the rendered section so the goal
// round-trips cross-machine without masquerading as manual.
func TestParseGoalSection_RoundTripsSourceFraming(t *testing.T) {
	cases := []struct {
		body       string
		wantText   string
		wantSource string
	}{
		{"Session opened with — add login rate limiting", "add login rate limiting", GoalSourceAutoWindow},
		{"Likely goal — fix the redis race", "fix the redis race", GoalSourceAutoEmbed},
		{"stop credential-stuffing", "stop credential-stuffing", GoalSourceMarker},
	}
	for _, c := range cases {
		text, source := parseGoalSection(c.body)
		if text != c.wantText {
			t.Errorf("parseGoalSection(%q) text = %q, want %q", c.body, text, c.wantText)
		}
		if source != c.wantSource {
			t.Errorf("parseGoalSection(%q) source = %q, want %q", c.body, source, c.wantSource)
		}
	}
}

// TestIngestUserFile_RoundTripsGoal proves the goal travels through the
// markdown ingest path (file → graph), the cross-machine federation
// medium.
func TestIngestUserFile_RoundTripsGoal(t *testing.T) {
	store := openTestStore(t)
	md := strings.Join([]string{
		"---", "user: alice", "---", "",
		"# alice's handoff", "",
		"## Session goal", "",
		"> Session opened with — add rate limiting to the login endpoint", "",
		"## Last user ask", "",
		"> cap it at 5 a minute", "",
	}, "\n")
	path := writeTempFile(t, md)
	if err := IngestUserFile(store, goalRepo, "engineering", path, "", true); err != nil {
		t.Fatalf("IngestUserFile: %v", err)
	}
	goal, _ := LatestGoal(store, "alice", goalRepo, "engineering")
	if goal == nil || goal.Text != "add rate limiting to the login endpoint" {
		t.Fatalf("goal not ingested: %+v", goal)
	}
	if goal.Source != GoalSourceAutoWindow {
		t.Errorf("ingested goal source = %q, want auto-window", goal.Source)
	}
	// And the last ask still ingests independently.
	ask, _ := LatestAsk(store, "alice", goalRepo, "engineering")
	if ask == nil || ask.Text != "cap it at 5 a minute" {
		t.Errorf("ask not ingested alongside goal: %+v", ask)
	}
}
