package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestResolveSessionID_FlagOverridesStdin(t *testing.T) {
	stdin := bytes.NewBufferString(`{"session_id":"from-stdin"}`)
	got := resolveSessionID(stdin, "from-flag")
	if got != "from-flag" {
		t.Errorf("flag should win, got %q", got)
	}
}

func TestResolveSessionID_StdinParse(t *testing.T) {
	payload := `{"session_id":"sess-X","transcript_path":"/tmp/t","cwd":"/repo","source":"compact","hook_event_name":"SessionStart"}`
	got := resolveSessionID(bytes.NewBufferString(payload), "")
	if got != "sess-X" {
		t.Errorf("got %q want sess-X", got)
	}
}

func TestResolveSessionID_StdinMalformed(t *testing.T) {
	// Malformed JSON returns "" — caller renders the fallback path.
	got := resolveSessionID(bytes.NewBufferString("not json"), "")
	if got != "" {
		t.Errorf("malformed stdin should fall back to empty, got %q", got)
	}
}

func TestWriteEnvelope_ValidJSONShape(t *testing.T) {
	var buf bytes.Buffer
	writeEnvelope(&buf, "hello world")

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("envelope is not valid JSON: %v\n%s", err, buf.String())
	}
	hso, ok := parsed["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("hookSpecificOutput missing: %#v", parsed)
	}
	if hso["hookEventName"] != "SessionStart" {
		t.Errorf("hookEventName = %v", hso["hookEventName"])
	}
	if hso["additionalContext"] != "hello world" {
		t.Errorf("additionalContext = %v", hso["additionalContext"])
	}
}

func TestWriteEnvelope_EmptyContextIsValid(t *testing.T) {
	var buf bytes.Buffer
	writeEnvelope(&buf, "")
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("empty envelope not parseable: %v", err)
	}
	hso := parsed["hookSpecificOutput"].(map[string]any)
	if hso["additionalContext"] != "" {
		t.Errorf("empty additionalContext expected, got %v", hso["additionalContext"])
	}
}

func TestApproxTokens(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 0},
		{"abcd", 1},
		{strings.Repeat("a", 6000), 1500},
	}
	for _, c := range cases {
		if got := approxTokens(c.in); got != c.want {
			t.Errorf("approxTokens(len=%d) = %d, want %d", len(c.in), got, c.want)
		}
	}
}

func TestEnforceTokenCap_DropsWorkingTreeFirst(t *testing.T) {
	parts := handoffParts{
		header:         "**Session:** sess-A · started now · 5m\n**Branch:** main · clean\n**Active spec:** none\n",
		whatYouWere:    "doing important work",
		activeSpecBody: strings.Repeat("body\n", 1000),
		kickoff:        "kickoff text",
		workingTree:    strings.Repeat("- some-file.go\n", 100),
		nextAction:     "ship it",
	}
	md := assembleFullHandoff(parts)
	got := enforceTokenCap(md, parts)
	if approxTokens(got) > compactHandoffTokenHardCap {
		t.Errorf("token cap exceeded: %d > %d", approxTokens(got), compactHandoffTokenHardCap)
	}
	// Header + active spec slug area + Next action are always preserved.
	if !strings.Contains(got, "## Hero session handoff") {
		t.Error("header dropped")
	}
	if !strings.Contains(got, "ship it") {
		t.Error("next action dropped")
	}
}

func TestAssembleFullHandoff_NoActiveSpec(t *testing.T) {
	parts := handoffParts{
		header:      "**Session:** sess-X · started now · 1m\n**Branch:** main · clean\n**Active spec:** none (exploratory session)\n",
		whatYouWere: "",
		kickoff:     "what was the user thinking about",
	}
	md := assembleFullHandoff(parts)
	if !strings.Contains(md, "exploratory session") {
		t.Error("expected exploratory-session indicator")
	}
	if !strings.Contains(md, "what was the user thinking about") {
		t.Error("expected kickoff preserved")
	}
	if strings.Contains(md, "### Active spec — full content") {
		t.Error("active spec section should be omitted when no spec")
	}
}

func TestIndentQuoteLines(t *testing.T) {
	in := "line one\nline two\nline three"
	got := indentQuoteLines(in)
	want := "line one\n> line two\n> line three"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestPickNextAction_NoSessionNoSpec(t *testing.T) {
	// pickNextAction with empty sessionID and nil spec returns "" so
	// the renderer prints the documented fallback.
	got := pickNextAction(t.TempDir(), "", nil)
	if got != "" {
		t.Errorf("expected empty next action; got %q", got)
	}
}
