package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/spf13/cobra"
)

// userMsg / asstMsg build transcript JSONL lines for the capture tests.
func userMsg(text string) string {
	return `{"type":"user","message":{"role":"user","content":` + jsonStr(text) + `}}`
}
func asstMsg(text string) string {
	return `{"type":"assistant","message":{"role":"assistant","content":` + jsonStr(text) + `}}`
}
func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// TestIsTrivialOpener covers the triviality filter directly: greeting/ack
// openers are trivial; whole-message matters, not prefix; substance
// signals and length keep a message.
func TestIsTrivialOpener(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"hi", true},
		{"hey there", true},
		{"thanks!", true},
		{"ok", true},
		{"go ahead", true},
		{"sure", true},
		{"lgtm", true},
		{"perfect", true},
		{"", true},
		// Substance keeps these:
		{"ok now do X", false},                                            // ack prefix but real request after
		{"add login rate limiting", false},                                // imperative verb
		{"why is this failing?", false},                                   // question mark
		{"look at internal/cli/checkpoint.go", false},                     // file path
		{"can you help me reason through the whole redesign here", false}, // >=6 words
		{"fix it", false},                                                 // imperative verb, short but substantive
	}
	for _, c := range cases {
		if got := isTrivialOpener(c.msg); got != c.want {
			t.Errorf("isTrivialOpener(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}

// TestOpeningWindowGoal_SkipsGreeting is the mandatory "greeting skipped /
// goal in message 2 captured" test: the opener is a greeting, the real
// goal is the second message, and the window must capture the second.
func TestOpeningWindowGoal_SkipsGreeting(t *testing.T) {
	env := newTestEnv(t)
	tp := writeTranscript(t, env.dir, strings.Join([]string{
		userMsg("hey can you help"),
		asstMsg("sure, what do you need?"),
		userMsg("add login rate limiting to stop credential stuffing"),
	}, "\n")+"\n")

	got := openingWindowGoalFromTranscript(tp)
	if !strings.Contains(got, "add login rate limiting") {
		t.Errorf("window goal did not capture the substantive 2nd message: %q", got)
	}
	if strings.Contains(got, "hey can you help") {
		t.Errorf("window goal wrongly included the trivial greeting opener: %q", got)
	}
}

// TestOpeningWindowGoal_GoalInThirdMessage proves the window reaches
// message 3 past two trivial openers.
func TestOpeningWindowGoal_GoalInThirdMessage(t *testing.T) {
	env := newTestEnv(t)
	tp := writeTranscript(t, env.dir, strings.Join([]string{
		userMsg("hi"),
		userMsg("thanks"),
		userMsg("refactor the digest scoring to use a half-life decay"),
	}, "\n")+"\n")

	got := openingWindowGoalFromTranscript(tp)
	if !strings.Contains(got, "refactor the digest scoring") {
		t.Errorf("window goal did not reach the substantive 3rd message: %q", got)
	}
}

// TestOpeningWindowGoal_AllTrivialFallsBackToFirst proves the floor never
// empties: when every opener is trivial, the raw first message is kept.
func TestOpeningWindowGoal_AllTrivialFallsBackToFirst(t *testing.T) {
	env := newTestEnv(t)
	tp := writeTranscript(t, env.dir, strings.Join([]string{
		userMsg("hi"),
		userMsg("thanks"),
		userMsg("ok"),
	}, "\n")+"\n")

	got := openingWindowGoalFromTranscript(tp)
	if got != "hi" {
		t.Errorf("all-trivial fallback = %q, want raw first message %q", got, "hi")
	}
}

// TestOpeningWindowGoal_Truncates proves the assembled window is truncated
// at compactHandoffKickoffCap.
func TestOpeningWindowGoal_Truncates(t *testing.T) {
	env := newTestEnv(t)
	long := "implement " + strings.Repeat("very-detailed-requirement ", 100)
	tp := writeTranscript(t, env.dir, userMsg(long)+"\n")
	got := openingWindowGoalFromTranscript(tp)
	if len(got) > compactHandoffKickoffCap+len("…") {
		t.Errorf("window goal not truncated: len=%d cap=%d", len(got), compactHandoffKickoffCap)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated window should end with ellipsis: %q", got[len(got)-10:])
	}
}

// TestOpeningWindowGoal_MissingTranscript proves a missing/unreadable
// transcript degrades to "" (best-effort).
func TestOpeningWindowGoal_MissingTranscript(t *testing.T) {
	if got := openingWindowGoalFromTranscript("/no/such/transcript.jsonl"); got != "" {
		t.Errorf("missing transcript should yield empty, got %q", got)
	}
}

// TestGoalMarkerFromTranscript covers the marker grep: the LAST
// `<!-- hero:goal: … -->` in an assistant message wins; absent → "".
func TestGoalMarkerFromTranscript(t *testing.T) {
	env := newTestEnv(t)
	t.Run("last marker wins", func(t *testing.T) {
		tp := writeTranscript(t, env.dir, strings.Join([]string{
			userMsg("add rate limiting"),
			asstMsg("first guess <!-- hero:goal: early goal -->"),
			asstMsg("refined <!-- hero:goal: stop credential-stuffing on login -->"),
		}, "\n")+"\n")
		got := goalMarkerFromTranscript(tp)
		if got != "stop credential-stuffing on login" {
			t.Errorf("marker = %q, want the LAST marker", got)
		}
	})

	t.Run("absent marker yields empty", func(t *testing.T) {
		tp := writeTranscript(t, env.dir, strings.Join([]string{
			userMsg("add rate limiting"),
			asstMsg("no marker here"),
		}, "\n")+"\n")
		if got := goalMarkerFromTranscript(tp); got != "" {
			t.Errorf("no marker should yield empty, got %q", got)
		}
	})

	t.Run("marker in user message is ignored", func(t *testing.T) {
		// Only ASSISTANT messages carry markers; a user echoing the
		// syntax must not be picked up.
		tp := writeTranscript(t, env.dir, userMsg("don't capture <!-- hero:goal: fake -->")+"\n")
		if got := goalMarkerFromTranscript(tp); got != "" {
			t.Errorf("user-message marker should be ignored, got %q", got)
		}
	})
}

// TestAutoEmitSessionGoal_MarkerOverridesWindow drives the real capture
// path: a transcript with a greeting opener, a substantive message, and a
// later marker. After capture, the goal is the marker (overriding the
// window), source=marker.
func TestAutoEmitSessionGoal_MarkerOverridesWindow(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	tp := writeTranscript(t, env.dir, strings.Join([]string{
		userMsg("hey"),
		userMsg("add rate limiting to login"),
		asstMsg("<!-- hero:goal: stop credential-stuffing -->"),
	}, "\n")+"\n")
	payload := `{"session_id":"s-goal","transcript_path":"` + tp + `"}`

	cmd := &cobra.Command{RunE: runNextCheckpoint}
	cmd.SetIn(strings.NewReader(payload))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	checkpointQuiet = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	defer store.Close()
	repoKey := gitutil.RepoKey(env.dir)
	user := nextUserSlug(cfg)
	domain := graph.DomainFor(cfg, graph.IntrinsicActive)
	goal, _ := handoff.LatestGoal(store, user, repoKey, domain)
	if goal == nil {
		t.Fatal("no goal recorded after checkpoint")
	}
	if goal.Text != "stop credential-stuffing" {
		t.Errorf("goal text = %q, want the marker text", goal.Text)
	}
	if goal.Source != handoff.GoalSourceMarker {
		t.Errorf("goal source = %q, want marker", goal.Source)
	}
	// The last-ask auto-emit is unchanged — the LAST user message wins.
	ask, _ := handoff.LatestAsk(store, user, repoKey, domain)
	if ask == nil || !strings.Contains(ask.Text, "add rate limiting to login") {
		t.Errorf("last ask = %+v, want the last user message", ask)
	}
}

// TestRunNextGoal_SetAndGet covers the manual command: set with arg, read
// with no arg, and that manual supersedes a marker/window goal.
func TestRunNextGoal_SetAndGet(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	repoKey := gitutil.RepoKey(env.dir)
	user := nextUserSlug(cfg)
	domain := graph.DomainFor(cfg, graph.IntrinsicActive)

	// Seed an auto-window goal, then a marker goal, to prove manual wins.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("open graph: %v", err)
	}
	_ = handoff.RecordGoal(store, repoKey, handoff.SessionGoal{User: user, Domain: domain, Text: "auto opener", Source: handoff.GoalSourceAutoWindow})
	_ = handoff.RecordGoal(store, repoKey, handoff.SessionGoal{User: user, Domain: domain, Text: "marker goal", Source: handoff.GoalSourceMarker})
	store.Close()

	// Set the manual goal via the command.
	setCmd := &cobra.Command{RunE: runNextGoal}
	setCmd.SetOut(bytes.NewBuffer(nil))
	if err := setCmd.RunE(setCmd, []string{"the", "real", "session", "goal"}); err != nil {
		t.Fatalf("runNextGoal(set): %v", err)
	}

	// Read it back via the command (no args).
	var out bytes.Buffer
	getCmd := &cobra.Command{RunE: runNextGoal}
	getCmd.SetOut(&out)
	if err := getCmd.RunE(getCmd, nil); err != nil {
		t.Fatalf("runNextGoal(get): %v", err)
	}
	if !strings.Contains(out.String(), "the real session goal") {
		t.Errorf("get output = %q, want the manual goal", out.String())
	}

	// And it agrees with LatestGoal, manual source, superseding the marker.
	store, err = graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("reopen graph: %v", err)
	}
	defer store.Close()
	goal, _ := handoff.LatestGoal(store, user, repoKey, domain)
	if goal == nil || goal.Text != "the real session goal" {
		t.Fatalf("manual goal did not supersede: %+v", goal)
	}
	if goal.Source != handoff.GoalSourceManual {
		t.Errorf("goal source = %q, want manual", goal.Source)
	}
}

// TestRunNextGoal_EmptyPrintsHint proves bare `hero next goal` on an empty
// goal prints the none-recorded hint, not a crash.
func TestRunNextGoal_EmptyPrintsHint(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	var out bytes.Buffer
	cmd := &cobra.Command{RunE: runNextGoal}
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("runNextGoal: %v", err)
	}
	if !strings.Contains(out.String(), "no session goal recorded") {
		t.Errorf("empty get = %q, want the none-recorded hint", out.String())
	}
}
