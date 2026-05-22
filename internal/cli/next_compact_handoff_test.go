package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/active"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/projection"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
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

// --- compact-handoff integration fixture & tests --------------------------
//
// The fixture below is intentionally kept local to this test file. It
// builds the smallest "populated session" that exercises every section
// of the assembled handoff: an active spec on disk, a session-id-tagged
// UserAsk + NextSuggestion + two Decisions + three Attempts touching
// distinct files, plus a dirty working tree. The integration test then
// calls buildHandoff and asserts every section's content.

type compactHandoffFixture struct {
	env         *testEnv
	sessionID   string
	specSlug    string
	specPath    string
	sessStart   time.Time
}

// setupCompactHandoffFixture stands up the full fixture and returns a
// handle. The caller is on the fixture's chdir thanks to newTestEnv.
func setupCompactHandoffFixture(t *testing.T) *compactHandoffFixture {
	t.Helper()

	env := newTestEnv(t)
	// Need a real git repo so the working-tree section has data.
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	// Configure a user so git commit could work if needed; not required
	// but harmless and makes `git status` output stable.
	_ = exec.Command("git", "-C", env.dir, "config", "user.email", "t@t.invalid").Run()
	_ = exec.Command("git", "-C", env.dir, "config", "user.name", "test").Run()

	const sessionID = "test-session"
	const specSlug = "fixture-spec"

	// Drop a fixture spec into .hero/planning/features/fixture-spec/spec.md.
	specDir := filepath.Join(env.heroDir, "planning", "features", specSlug)
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("mkdir spec: %v", err)
	}
	specPath := filepath.Join(specDir, "spec.md")
	specContent := `---
title: "Fixture Spec"
slug: fixture-spec
type: feature
status: delivering
priority: medium
---

# Fixture Spec

## Goal

Ship the fixture workflow so the integration test has a real spec to chew on. The Goal section is intentionally short so the 300-char excerpt assertion is deterministic.

## Design

A populated session needs a spec, decisions, and a kickoff. The design fills the body so the spec-body excerpt has meaningful content.

## Acceptance Criteria

- [ ] First criterion intentionally unchecked so pickNextAction can fall back to it.
- [x] Second criterion already done.
`
	if err := os.WriteFile(specPath, []byte(specContent), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	// Active-session registry entry pointing at the fixture spec.
	sessStart := time.Now().UTC().Add(-1 * time.Hour)
	reg := active.Load(env.heroDir)
	reg.Sessions[sessionID] = active.Session{
		Spec:    specSlug,
		Command: "test",
		Started: sessStart,
	}
	if err := reg.Save(env.heroDir); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	// Populate the graph with session-tagged events.
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	// UserAsk — the original kickoff.
	if _, err := store.UpsertNode(&graph.Node{
		Type: handoff.NodeUserAsk, Key: "ask-1", Repo: "test-repo", Domain: "engineering", ContentHash: "h-ask-1",
		Props: map[string]any{
			"text":       "build the fixture workflow",
			"session_id": sessionID,
		},
	}); err != nil {
		t.Fatalf("UpsertNode UserAsk: %v", err)
	}
	// Two Decisions with explicit valid_from to lock ordering — the
	// store uses second-precision RFC3339, so we space them by a
	// minute to be unambiguous.
	now := time.Now().UTC()
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Decision", Key: "dec-old", Repo: "test-repo", Domain: "engineering", ContentHash: "h-dec-old",
		ValidFrom: now.Add(-2 * time.Minute).Format(time.RFC3339),
		Props: map[string]any{
			"title":      "use a fixture",
			"session_id": sessionID,
		},
	}); err != nil {
		t.Fatalf("UpsertNode Decision old: %v", err)
	}
	if _, err := store.UpsertNode(&graph.Node{
		Type: "Decision", Key: "dec-new", Repo: "test-repo", Domain: "engineering", ContentHash: "h-dec-new",
		ValidFrom: now.Add(-1 * time.Minute).Format(time.RFC3339),
		Props: map[string]any{
			"title":      "co-locate fixture in test file",
			"session_id": sessionID,
		},
	}); err != nil {
		t.Fatalf("UpsertNode Decision new: %v", err)
	}
	// Three Attempts touching distinct file paths.
	for i, body := range []string{
		"editing internal/cli/next_compact_handoff.go to add hook",
		"verified internal/hooks/claude_settings.go layout",
		"updated internal/projection/compact_handoff.go query",
	} {
		if _, err := store.UpsertNode(&graph.Node{
			Type: "Attempt", Key: "att-" + string(rune('1'+i)), Repo: "test-repo", Domain: "engineering",
			ContentHash: "h-att-" + string(rune('1'+i)),
			Props: map[string]any{
				"body":       body,
				"session_id": sessionID,
			},
		}); err != nil {
			t.Fatalf("UpsertNode Attempt: %v", err)
		}
	}
	// NextSuggestion — drives the next-concrete-action section.
	if _, err := store.UpsertNode(&graph.Node{
		Type: handoff.NodeNextSuggestion, Key: "ns-1", Repo: "test-repo", Domain: "engineering", ContentHash: "h-ns-1",
		Props: map[string]any{
			"text":       "run go test on the new fixture",
			"session_id": sessionID,
		},
	}); err != nil {
		t.Fatalf("UpsertNode NextSuggestion: %v", err)
	}
	store.Close()

	// Working tree: write two dirty files so `git status --short` is non-empty.
	if err := os.WriteFile(filepath.Join(env.dir, "dirty-a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write dirty-a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(env.dir, "dirty-b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write dirty-b: %v", err)
	}

	return &compactHandoffFixture{
		env:       env,
		sessionID: sessionID,
		specSlug:  specSlug,
		specPath:  specPath,
		sessStart: sessStart,
	}
}

// TestAssembleFullHandoff_PopulatedSession — the integration test.
// Drives the full buildHandoff pipeline against a populated fixture and
// asserts every documented section appears with the expected content.
func TestAssembleFullHandoff_PopulatedSession(t *testing.T) {
	f := setupCompactHandoffFixture(t)

	// Sanity-check the config path resolves under the fixture root.
	if _, err := config.Load(f.env.dir); err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	md := buildHandoff(&cobra.Command{}, payloadContext{SessionID: f.sessionID})
	if md == "" {
		t.Fatal("buildHandoff returned empty markdown")
	}

	// Header — tolerant on the exact timestamp; assert shape and session id.
	headerRE := regexp.MustCompile(`(?m)^\*\*Session:\*\* test-session · started \S+ · \S+`)
	if !headerRE.MatchString(md) {
		t.Errorf("header line not in expected shape:\n%s", excerpt(md, 0, 400))
	}

	// Active spec line links the fixture spec slug.
	activeRE := regexp.MustCompile(`\*\*Active spec:\*\* \[fixture-spec\]`)
	if !activeRE.MatchString(md) {
		t.Errorf("active spec line missing fixture-spec link:\n%s", excerpt(md, 0, 600))
	}

	// "What you were doing" section is rendered.
	if !strings.Contains(md, "### What you were doing") {
		t.Error("missing 'What you were doing' section")
	}
	// The derived excerpt should mention the Goal text and be ≤300 chars.
	whatYou := sectionBody(md, "### What you were doing")
	if !strings.Contains(whatYou, "fixture workflow") {
		t.Errorf("'What you were doing' didn't pull Goal: %q", whatYou)
	}
	if len(whatYou) > compactHandoffGoalSummaryCap+10 { // +10 for ellipsis slack
		t.Errorf("'What you were doing' over cap: len=%d", len(whatYou))
	}

	// Active spec full content — frontmatter stripped.
	if !strings.Contains(md, "### Active spec — full content") {
		t.Error("missing 'Active spec — full content' section")
	}
	specBody := sectionBody(md, "### Active spec — full content")
	if strings.Contains(specBody, "title: \"Fixture Spec\"") {
		t.Error("frontmatter leaked into active spec body")
	}
	if !strings.Contains(specBody, "# Fixture Spec") {
		t.Error("active spec body missing the fixture spec H1")
	}

	// Original kickoff — UserAsk text.
	kickoff := sectionBody(md, "### Original kickoff (this session)")
	if !strings.Contains(kickoff, "build the fixture workflow") {
		t.Errorf("kickoff missing UserAsk text: %q", kickoff)
	}

	// Files touched — all three files, deduplicated.
	files := sectionBody(md, "### Files touched this session")
	for _, want := range []string{
		"internal/cli/next_compact_handoff.go",
		"internal/hooks/claude_settings.go",
		"internal/projection/compact_handoff.go",
	} {
		if !strings.Contains(files, want) {
			t.Errorf("files-touched missing %s; got:\n%s", want, files)
		}
	}

	// Recent decisions — both present, newest-first.
	decisions := sectionBody(md, "### Recent decisions (this session)")
	idxNew := strings.Index(decisions, "co-locate fixture in test file")
	idxOld := strings.Index(decisions, "use a fixture")
	if idxNew < 0 || idxOld < 0 {
		t.Errorf("decisions missing entries: new=%d old=%d\n%s", idxNew, idxOld, decisions)
	} else if !(idxNew < idxOld) {
		t.Errorf("decisions not ordered newest-first: new@%d old@%d", idxNew, idxOld)
	}

	// Next concrete action — reflects the NextSuggestion text.
	next := sectionBody(md, "### Next concrete action")
	if !strings.Contains(next, "run go test on the new fixture") {
		t.Errorf("next-action missing NextSuggestion text: %q", next)
	}

	// Working tree — both dirty files appear.
	wt := sectionBody(md, "### Working tree")
	if !strings.Contains(wt, "dirty-a.txt") || !strings.Contains(wt, "dirty-b.txt") {
		t.Errorf("working tree missing dirty files: %q", wt)
	}

	// Total output within token cap.
	if approxTokens(md) > compactHandoffTokenHardCap {
		t.Errorf("assembled handoff exceeds token cap: %d > %d", approxTokens(md), compactHandoffTokenHardCap)
	}
}

// --- truncation cascade ---------------------------------------------------

// TestEnforceTokenCap_FullCascade walks each of the 5 truncation steps
// with a progressively larger overage, asserting that the cap is respected
// at every step and that earlier sections survive a moderate overage.
func TestEnforceTokenCap_FullCascade(t *testing.T) {
	t.Run("step1_dropWorkingTree", func(t *testing.T) {
		parts := handoffParts{
			header:      "**Session:** s · started now · 1m\n**Branch:** main · clean\n**Active spec:** none\n",
			whatYouWere: "x",
			workingTree: strings.Repeat("- some-file.go\n", 1500), // ~25KB working tree alone
			nextAction:  "ship it",
		}
		md := assembleFullHandoff(parts)
		if approxTokens(md) <= compactHandoffTokenHardCap {
			t.Skipf("precondition: input must exceed cap; got %d tokens", approxTokens(md))
		}
		got := enforceTokenCap(md, parts)
		if approxTokens(got) > compactHandoffTokenHardCap {
			t.Errorf("step1: cap not enforced (%d > %d)", approxTokens(got), compactHandoffTokenHardCap)
		}
		if strings.Contains(got, "- some-file.go") {
			t.Error("step1: working-tree entries should be dropped")
		}
		if !strings.Contains(got, "ship it") {
			t.Error("step1: next action should survive")
		}
	})

	t.Run("step2_trimFilesTouched", func(t *testing.T) {
		parts := makeFilesTouchedHeavy(40)
		md := assembleFullHandoff(parts)
		if approxTokens(md) <= compactHandoffTokenHardCap {
			t.Skipf("precondition: must exceed cap")
		}
		got := enforceTokenCap(md, parts)
		if approxTokens(got) > compactHandoffTokenHardCap {
			t.Errorf("step2: cap not enforced (%d > %d)", approxTokens(got), compactHandoffTokenHardCap)
		}
		if !strings.Contains(got, "ship it") {
			t.Error("step2: next action should survive")
		}
	})

	t.Run("step3_trimDecisions", func(t *testing.T) {
		parts := makeDecisionsHeavy(40)
		md := assembleFullHandoff(parts)
		if approxTokens(md) <= compactHandoffTokenHardCap {
			t.Skipf("precondition: must exceed cap")
		}
		got := enforceTokenCap(md, parts)
		if approxTokens(got) > compactHandoffTokenHardCap {
			t.Errorf("step3: cap not enforced (%d > %d)", approxTokens(got), compactHandoffTokenHardCap)
		}
		if !strings.Contains(got, "ship it") {
			t.Error("step3: next action should survive")
		}
	})

	t.Run("step4_shrinkSpecBody", func(t *testing.T) {
		parts := handoffParts{
			header:         "**Session:** s · started now · 1m\n**Branch:** main · clean\n**Active spec:** [slug](p)\n",
			activeSpecSlug: "slug",
			activeSpecBody: strings.Repeat("active spec body content that is verbose ", 4000),
			nextAction:     "ship it",
		}
		md := assembleFullHandoff(parts)
		if approxTokens(md) <= compactHandoffTokenHardCap {
			t.Skipf("precondition: must exceed cap")
		}
		got := enforceTokenCap(md, parts)
		if approxTokens(got) > compactHandoffTokenHardCap {
			t.Errorf("step4: cap not enforced (%d > %d)", approxTokens(got), compactHandoffTokenHardCap)
		}
		if !strings.Contains(got, "ship it") {
			t.Error("step4: next action should survive")
		}
	})

	t.Run("step5_shrinkKickoff", func(t *testing.T) {
		// Very large kickoff and no other shrinkables; final step must
		// reduce kickoff itself to drop us under the cap (or close to it).
		parts := handoffParts{
			header:     "**Session:** s · started now · 1m\n**Branch:** main · clean\n**Active spec:** none\n",
			kickoff:    strings.Repeat("user-said-this-and-then-said-more ", 2000),
			nextAction: "ship it",
		}
		md := assembleFullHandoff(parts)
		if approxTokens(md) <= compactHandoffTokenHardCap {
			t.Skipf("precondition: must exceed cap")
		}
		got := enforceTokenCap(md, parts)
		// Note: step 5 only shrinks the kickoff to 100 chars max — on a
		// 65KB kickoff that's a huge reduction. We still assert "under cap"
		// because the only remaining material is small.
		if approxTokens(got) > compactHandoffTokenHardCap {
			t.Errorf("step5: cap not enforced (%d > %d)", approxTokens(got), compactHandoffTokenHardCap)
		}
		if !strings.Contains(got, "ship it") {
			t.Error("step5: next action should survive")
		}
	})
}

// makeFilesTouchedHeavy returns parts where the files-touched list is
// the dominant size contributor.
func makeFilesTouchedHeavy(n int) handoffParts {
	parts := handoffParts{
		header:     "**Session:** s · started now · 1m\n**Branch:** main · clean\n**Active spec:** none\n",
		nextAction: "ship it",
	}
	for i := 0; i < n; i++ {
		path := strings.Repeat("very/long/path/segment/", 10) + "file" + string(rune('a'+(i%26))) + ".go"
		parts.filesTouched = append(parts.filesTouched, projection.CompactFileTouch{
			Path:  path,
			Count: i + 1,
		})
	}
	return parts
}

// makeDecisionsHeavy returns parts where decisions dominate size.
func makeDecisionsHeavy(n int) handoffParts {
	parts := handoffParts{
		header:     "**Session:** s · started now · 1m\n**Branch:** main · clean\n**Active spec:** none\n",
		nextAction: "ship it",
	}
	for i := 0; i < n; i++ {
		title := strings.Repeat("verbose decision title segment ", 30)
		parts.decisions = append(parts.decisions, projection.CompactDecision{Title: title})
	}
	return parts
}

// TestEnforceTokenCap_PreservesInvariants — absurdly over-budget input
// must still leave the header, active spec slug, and next-action line
// intact in the final output.
func TestEnforceTokenCap_PreservesInvariants(t *testing.T) {
	parts := handoffParts{
		header:         "**Session:** sess-X · started 2026-01-01T00:00:00Z · 1m\n**Branch:** main · clean\n**Active spec:** [fixture-spec](p) — feature, delivering\n",
		whatYouWere:    "doing important work",
		activeSpecSlug: "fixture-spec",
		activeSpecBody: strings.Repeat("body line that is verbose and contributes to size\n", 10000),
		kickoff:        strings.Repeat("user spoke at length ", 5000),
		workingTree:    strings.Repeat("- file.go\n", 3000),
		nextAction:     "run the absolutely critical next concrete action",
	}
	md := assembleFullHandoff(parts)
	got := enforceTokenCap(md, parts)

	if !strings.Contains(got, "**Session:** sess-X") {
		t.Error("session header dropped under extreme overage")
	}
	if !strings.Contains(got, "fixture-spec") {
		t.Error("active spec slug dropped under extreme overage")
	}
	if !strings.Contains(got, "run the absolutely critical next concrete action") {
		t.Error("next concrete action dropped under extreme overage")
	}
}

// --- safety contract ------------------------------------------------------

// TestRunCompactHandoff_PanicRecoveryReturnsValidEnvelope — emitJSONEnvelope
// has a deferred recover. Force a panic in the middle of envelope
// emission and assert the writer still received a valid JSON envelope
// and no error bubbled up.
func TestRunCompactHandoff_PanicRecoveryReturnsValidEnvelope(t *testing.T) {
	// Direct unit test on the recover path: emitJSONEnvelope's defer
	// catches panics. We trigger one by passing a malformed cobra
	// command that panics on OutOrStdout. Easier path: call a helper
	// that wraps emitJSONEnvelope's recover with a synthetic panic.
	// Since we can't easily inject a panic into the real pipeline
	// without changing production code, we cover the contract by
	// exercising writeEnvelope directly on a panicking writer.

	// First sanity-check: writeEnvelope on an empty additional yields a
	// parseable envelope.
	var buf bytes.Buffer
	writeEnvelope(&buf, "")
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("empty envelope unparseable: %v", err)
	}

	// Now exercise the real recover via a panic-injecting wrapper. We
	// build a cobra command whose stdin reader panics on Read, which
	// surfaces inside resolveSessionID via emitJSONEnvelope.
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetIn(panicReader{})

	// Save and restore the package-level flag.
	prevJSON := compactHandoffJSON
	defer func() { compactHandoffJSON = prevJSON }()
	compactHandoffJSON = true

	if err := emitJSONEnvelope(cmd); err != nil {
		t.Fatalf("emitJSONEnvelope returned error despite recover: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected an envelope on stdout even after panic")
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("post-panic envelope unparseable: %v\n%s", err, out.String())
	}
	hso, ok := parsed["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("envelope shape wrong: %#v", parsed)
	}
	if hso["hookEventName"] != "SessionStart" {
		t.Errorf("hookEventName = %v", hso["hookEventName"])
	}
}

// panicReader is an io.Reader whose Read panics — used to drive the
// emitJSONEnvelope recover path.
type panicReader struct{}

func (panicReader) Read(p []byte) (int, error) {
	panic("synthetic panic from test reader")
}

// TestRunCompactHandoff_AlwaysExitsZeroOnBadStdin — feed malformed
// JSON on stdin to emitJSONEnvelope and assert it does not return an
// error AND still produces a valid envelope.
func TestRunCompactHandoff_AlwaysExitsZeroOnBadStdin(t *testing.T) {
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetIn(bytes.NewBufferString("{ this is not valid json"))

	prevJSON := compactHandoffJSON
	prevSess := compactHandoffSessionID
	defer func() {
		compactHandoffJSON = prevJSON
		compactHandoffSessionID = prevSess
	}()
	compactHandoffJSON = true
	compactHandoffSessionID = ""

	if err := emitJSONEnvelope(cmd); err != nil {
		t.Fatalf("emitJSONEnvelope returned error on bad stdin: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected an envelope even with bad stdin")
	}
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("envelope unparseable: %v\n%s", err, out.String())
	}
	hso, ok := parsed["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("envelope missing hookSpecificOutput: %#v", parsed)
	}
	if _, ok := hso["additionalContext"]; !ok {
		t.Error("envelope missing additionalContext field")
	}
}

// --- content-extraction helpers ------------------------------------------

// TestExtractGoalSection_PreservesIntent — deriveWhatYouWere should
// pull the Goal section's first paragraph and cap at 300 chars, not
// silently truncate mid-word in a way that loses meaning.
func TestExtractGoalSection_PreservesIntent(t *testing.T) {
	tmp := t.TempDir()
	specFile := filepath.Join(tmp, "spec.md")
	content := `---
title: "T"
type: feature
status: planning
---

## Goal

Build the thing so it works. First paragraph contains the core intent.

Second paragraph has additional detail that should be dropped.

## Design

Out of scope.
`
	if err := os.WriteFile(specFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := spec.ParseFile(specFile)
	if err != nil {
		t.Fatalf("spec.ParseFile: %v", err)
	}
	got := deriveWhatYouWere(s)
	if !strings.Contains(got, "core intent") {
		t.Errorf("expected first paragraph intent; got %q", got)
	}
	if strings.Contains(got, "Second paragraph") {
		t.Errorf("second paragraph should be dropped; got %q", got)
	}
	if len(got) > compactHandoffGoalSummaryCap {
		t.Errorf("excerpt over cap: len=%d cap=%d", len(got), compactHandoffGoalSummaryCap)
	}
}

// TestStripFrontmatter_HandlesAllShapes — stripFrontmatter (from next.go,
// shared in the cli package) must handle the three real shapes: with
// trailing newline, without, and content with no frontmatter at all.
func TestStripFrontmatter_HandlesAllShapes(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{
			name: "withTrailingNewline",
			in:   "---\ntitle: x\n---\n\n# Body\n",
			want: "\n# Body\n",
		},
		{
			name: "noFrontmatter",
			in:   "# Just a body\n",
			want: "# Just a body\n",
		},
		{
			name: "unterminatedFrontmatter",
			in:   "---\ntitle: x\nno closing here",
			want: "---\ntitle: x\nno closing here",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripFrontmatter(c.in)
			if got != c.want {
				t.Errorf("stripFrontmatter(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestTruncateSpecBody_AppendsReadFullSuffix — when the active spec
// body exceeds compactHandoffActiveSpecBodyCap, buildHandoffParts
// appends "… (truncated — read full at <path>)" to the body slot.
// We test this on the parts struct (before enforceTokenCap) because
// the global token cap may further rewrite the suffix downstream.
func TestTruncateSpecBody_AppendsReadFullSuffix(t *testing.T) {
	f := setupCompactHandoffFixture(t)

	// Overwrite the fixture spec with a body that exceeds the 6KB cap.
	bigBody := `---
title: "T"
type: feature
status: planning
---

# Big
` + strings.Repeat("filler line that pads the body beyond the 6KB inline cap so the truncation branch fires deterministically\n", 200)
	if err := os.WriteFile(f.specPath, []byte(bigBody), 0o644); err != nil {
		t.Fatal(err)
	}

	// Parse the spec and call buildHandoffParts directly.
	s, err := spec.ParseFile(f.specPath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	parts := buildHandoffParts(f.env.dir, f.env.heroDir, f.sessionID, "", f.sessStart, s, nil)
	if !strings.Contains(parts.activeSpecBody, "truncated — read full at") {
		t.Errorf("expected read-full suffix on oversize spec body; got:\n%s", parts.activeSpecBody)
	}
	if !strings.Contains(parts.activeSpecBody, "fixture-spec") {
		t.Errorf("expected suffix to reference the spec path; got:\n%s", parts.activeSpecBody)
	}
}

// TestKickoffFromTranscript_WhenNoUserAsk — when no UserAsk is present
// in the graph for this session but a transcript_path is available
// from the SessionStart payload, kickoffForSession falls back to the
// first user record in the JSONL transcript. The assembled handoff
// then renders the recovered prompt instead of the empty placeholder.
func TestKickoffFromTranscript_WhenNoUserAsk(t *testing.T) {
	env := newTestEnv(t)
	if err := exec.Command("git", "init", "-q", env.dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	const sessionID = "no-ask-session"
	// Register a session with no spec — guarantees no graph events.
	reg := active.Load(env.heroDir)
	reg.Sessions[sessionID] = active.Session{
		Spec:    "",
		Command: "test",
		Started: time.Now().UTC().Add(-30 * time.Minute),
	}
	if err := reg.Save(env.heroDir); err != nil {
		t.Fatal(err)
	}

	// Write a fake Claude Code transcript with a first user message.
	tp := filepath.Join(env.dir, "transcript.jsonl")
	const kickoffText = "fix the kickoff fallback"
	jsonl := `{"type":"user","message":{"role":"user","content":"` + kickoffText + `"}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":"sure"}}` + "\n"
	if err := os.WriteFile(tp, []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}

	md := buildHandoff(&cobra.Command{}, payloadContext{SessionID: sessionID, TranscriptPath: tp})
	if !strings.Contains(md, kickoffText) {
		t.Errorf("expected kickoff text recovered from transcript; got:\n%s", md)
	}
	if strings.Contains(md, "_No kickoff recorded for this session._") {
		t.Errorf("placeholder should not appear when transcript supplied a kickoff; got:\n%s", md)
	}
}

// --- small helpers --------------------------------------------------------

// sectionBody returns the text between heading and the next blank-line +
// "###" heading. Tolerant: trims surrounding whitespace.
func sectionBody(md, heading string) string {
	idx := strings.Index(md, heading)
	if idx < 0 {
		return ""
	}
	rest := md[idx+len(heading):]
	// Find the next H3 (or end of doc).
	if next := strings.Index(rest, "\n### "); next >= 0 {
		rest = rest[:next]
	}
	return strings.TrimSpace(rest)
}

func excerpt(s string, start, end int) string {
	if end > len(s) {
		end = len(s)
	}
	if start < 0 {
		start = 0
	}
	return s[start:end]
}

// --- kickoffForSession transcript-fallback tests --------------------------

// writeTranscript writes a JSONL transcript and returns its path. Used
// by the kickoff-fallback tests below.
func writeTranscript(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "transcript.jsonl")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	return p
}

// TestKickoffForSession_FallbackToTranscript — graph empty, transcript
// has a user record → returns the text.
func TestKickoffForSession_FallbackToTranscript(t *testing.T) {
	env := newTestEnv(t)
	tp := writeTranscript(t, env.dir,
		`{"type":"user","message":{"role":"user","content":"do the thing"}}`+"\n")
	got := kickoffForSession(env.heroDir, "sess-A", tp)
	if got != "do the thing" {
		t.Errorf("got %q; want %q", got, "do the thing")
	}
}

// TestKickoffForSession_TranscriptStringContent — `content` is a plain
// string at the top level (no nested message).
func TestKickoffForSession_TranscriptStringContent(t *testing.T) {
	env := newTestEnv(t)
	tp := writeTranscript(t, env.dir,
		`{"type":"user","content":"plain string content"}`+"\n")
	got := kickoffForSession(env.heroDir, "sess-A", tp)
	if got != "plain string content" {
		t.Errorf("got %q; want %q", got, "plain string content")
	}
}

// TestKickoffForSession_TranscriptContentBlocks — `content` is an array
// of content blocks; the first text block's text wins.
func TestKickoffForSession_TranscriptContentBlocks(t *testing.T) {
	env := newTestEnv(t)
	tp := writeTranscript(t, env.dir,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"block-one"},{"type":"text","text":"block-two"}]}}`+"\n")
	got := kickoffForSession(env.heroDir, "sess-A", tp)
	if got != "block-one" {
		t.Errorf("got %q; want %q", got, "block-one")
	}
}

// TestKickoffForSession_MalformedTranscriptReturnsEmpty — invalid JSONL
// at the head of the file does not crash; the loop simply skips broken
// lines and returns empty when no valid user record is found.
func TestKickoffForSession_MalformedTranscriptReturnsEmpty(t *testing.T) {
	env := newTestEnv(t)
	tp := writeTranscript(t, env.dir,
		"this is not json\nneither is this\n{ partial\n")
	got := kickoffForSession(env.heroDir, "sess-A", tp)
	if got != "" {
		t.Errorf("expected empty on malformed transcript; got %q", got)
	}
}

// TestKickoffForSession_MissingTranscriptReturnsEmpty — transcript_path
// set but file missing → empty, no crash.
func TestKickoffForSession_MissingTranscriptReturnsEmpty(t *testing.T) {
	env := newTestEnv(t)
	got := kickoffForSession(env.heroDir, "sess-A", filepath.Join(env.dir, "does-not-exist.jsonl"))
	if got != "" {
		t.Errorf("expected empty on missing transcript; got %q", got)
	}
}

// TestKickoffForSession_GraphHitWinsOverTranscript — when a UserAsk
// node exists in the graph for the session, the transcript is not
// consulted.
func TestKickoffForSession_GraphHitWinsOverTranscript(t *testing.T) {
	env := newTestEnv(t)
	const sessionID = "graph-wins-session"
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	if _, err := store.UpsertNode(&graph.Node{
		Type: handoff.NodeUserAsk, Key: "ask-graph", Repo: "test-repo", Domain: "engineering",
		ContentHash: "h-ask-graph",
		Props: map[string]any{
			"text":       "from the graph",
			"session_id": sessionID,
		},
	}); err != nil {
		t.Fatalf("UpsertNode UserAsk: %v", err)
	}
	store.Close()

	// Transcript has a different prompt. Should NOT be read.
	tp := writeTranscript(t, env.dir,
		`{"type":"user","message":{"role":"user","content":"from the transcript"}}`+"\n")

	got := kickoffForSession(env.heroDir, sessionID, tp)
	if got != "from the graph" {
		t.Errorf("graph hit should win; got %q", got)
	}
}

// TestKickoffForSession_SkipsAssistantBeforeUser — the loop must walk
// past non-user records (system / assistant) and return the first user
// message it finds, not the first record overall.
func TestKickoffForSession_SkipsAssistantBeforeUser(t *testing.T) {
	env := newTestEnv(t)
	tp := writeTranscript(t, env.dir,
		`{"type":"system","content":"session init"}`+"\n"+
			`{"type":"assistant","message":{"role":"assistant","content":"warmup"}}`+"\n"+
			`{"type":"user","message":{"role":"user","content":"actual user prompt"}}`+"\n")
	got := kickoffForSession(env.heroDir, "sess-A", tp)
	if got != "actual user prompt" {
		t.Errorf("got %q; want %q", got, "actual user prompt")
	}
}

// TestKickoffForSession_TruncatesAtCap — the transcript-derived text is
// truncated at compactHandoffKickoffCap exactly like the graph path.
func TestKickoffForSession_TruncatesAtCap(t *testing.T) {
	env := newTestEnv(t)
	long := strings.Repeat("x", compactHandoffKickoffCap+50)
	tp := writeTranscript(t, env.dir,
		`{"type":"user","message":{"role":"user","content":"`+long+`"}}`+"\n")
	got := kickoffForSession(env.heroDir, "sess-A", tp)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis suffix on truncated text; got len=%d", len(got))
	}
	// length: cap + 1 rune for the ellipsis (3 bytes in UTF-8)
	if len(got) > compactHandoffKickoffCap+4 {
		t.Errorf("over cap: len=%d cap=%d", len(got), compactHandoffKickoffCap)
	}
}

// TestKickoffForSession_EmptyTranscriptPath — no transcript path → empty,
// regardless of file existence.
func TestKickoffForSession_EmptyTranscriptPath(t *testing.T) {
	env := newTestEnv(t)
	got := kickoffForSession(env.heroDir, "sess-A", "")
	if got != "" {
		t.Errorf("expected empty on no transcript path; got %q", got)
	}
}


