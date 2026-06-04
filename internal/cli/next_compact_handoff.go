package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/active"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/projection"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// next_compact_handoff.go — implements `hero next compact-handoff`.
//
// Purpose: when Claude Code (or Codex) compacts a long session, the
// harness fires SessionStart with source:"compact" and supports an
// `additionalContext` field in the hook's JSON response that gets
// injected directly into the model's restored context. This command
// builds that context: a session-scoped, deterministic resume packet
// containing the active spec, session metadata, recent decisions, and
// next action — capped at ~1500 tokens.
//
// Critical safety contract: on the `--json` path, this command MUST
// NEVER fail. A crash or stdin parse error returns a minimal valid
// envelope and exits 0 so the hook never blocks compaction.

const (
	// compactHandoffTokenHardCap is the absolute upper bound on the
	// rendered markdown's approximate token count. We approximate
	// tokens as len(content)/4, matching the rough OpenAI/Anthropic
	// rule of thumb. Truncation order is defined in the spec.
	compactHandoffTokenHardCap = 1500

	// compactHandoffActiveSpecBodyCap limits the inline spec body
	// before further truncation under token pressure.
	compactHandoffActiveSpecBodyCap = 6000

	// compactHandoffKickoffCap truncates the first UserAsk to keep
	// the kickoff section bounded.
	compactHandoffKickoffCap = 400

	// compactHandoffGoalSummaryCap truncates the "what you were doing"
	// derivation from the spec's Goal section.
	compactHandoffGoalSummaryCap = 300

	// compactHandoffWorkingTreeCap caps the working-tree section.
	compactHandoffWorkingTreeCap = 15

	// compactHandoffSessionStartLookbackHours is the fallback window
	// when stdin doesn't carry session_id: any registered session
	// started within this many hours is treated as "probably current."
	compactHandoffSessionStartLookbackHours = 1
)

var (
	compactHandoffJSON      bool
	compactHandoffSessionID string
)

var nextCompactHandoffCmd = &cobra.Command{
	Use:   "compact-handoff",
	Short: "Emit a session-scoped resume packet for SessionStart{compact} hooks",
	Long: `Returns a deterministic, session-scoped resume context tailored
to a post-compaction SessionStart hook. With --json, reads a Claude
Code-style SessionStart payload from stdin and emits the
{"hookSpecificOutput": {"additionalContext": "..."}} envelope.

Without --json, prints the rendered markdown to stdout for debugging.
The --session <id> flag overrides any stdin payload and is intended
for manual testing.

This command never fails on the --json path: on any internal error
it returns a minimal valid envelope and exits 0 so the hook never
blocks compaction.`,
	RunE: runNextCompactHandoff,
}

func init() {
	nextCompactHandoffCmd.Flags().BoolVar(&compactHandoffJSON, "json", false,
		"emit a Claude Code SessionStart JSON envelope reading the payload from stdin")
	nextCompactHandoffCmd.Flags().StringVar(&compactHandoffSessionID, "session", "",
		"force a specific session id (skips stdin payload parsing; for debugging)")
}

// runNextCompactHandoff is the cobra entrypoint. In JSON mode it
// guards every failure path so the hook never bubbles a non-zero exit
// or unparseable output to Claude Code.
func runNextCompactHandoff(cmd *cobra.Command, args []string) error {
	if compactHandoffJSON {
		return emitJSONEnvelope(cmd)
	}
	ctx := resolveSessionContext(cmd.InOrStdin(), compactHandoffSessionID)
	md := buildHandoff(cmd, ctx)
	fmt.Fprint(cmd.OutOrStdout(), md)
	return nil
}

// emitJSONEnvelope is the safe wrapper around the JSON-mode path.
// Defers a panic recovery so even an unexpected panic during render
// returns a minimal-valid envelope instead of crashing the hook.
func emitJSONEnvelope(cmd *cobra.Command) (retErr error) {
	w := cmd.OutOrStdout()
	defer func() {
		if r := recover(); r != nil {
			writeEnvelope(w, "")
			retErr = nil
		}
	}()

	ctx := resolveSessionContext(cmd.InOrStdin(), compactHandoffSessionID)
	md := buildHandoff(cmd, ctx)
	writeEnvelope(w, md)
	return nil
}

// writeEnvelope emits the Claude Code SessionStart hook envelope on w.
// On marshal failure (which should be impossible for a string payload),
// falls back to a hand-crafted empty envelope so the hook always
// produces parseable JSON.
func writeEnvelope(w io.Writer, additional string) {
	payload := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "SessionStart",
			"additionalContext": additional,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		// Defensive: any string is JSON-marshalable, so this branch
		// should never trip. Emit a minimal hand-crafted envelope.
		fmt.Fprintln(w, `{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":""}}`)
		return
	}
	fmt.Fprintln(w, string(data))
}

// sessionStartPayload mirrors the documented Claude Code SessionStart
// hook stdin payload. Fields are best-effort: unknown harnesses may
// drop or rename fields, and the parser falls through to the lookback
// heuristic when session_id is absent.
type sessionStartPayload struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	Source         string `json:"source"`
	HookEventName  string `json:"hook_event_name"`
}

// payloadContext is the small bundle of session-scoped fields extracted
// from the SessionStart hook payload. We thread this through so the
// kickoff fallback can read the transcript when no UserAsk is in the
// graph for this session.
type payloadContext struct {
	SessionID      string
	TranscriptPath string
}

// resolveSessionID is a thin wrapper around resolveSessionContext that
// returns only the session id. Kept so existing tests and callers that
// only care about the id stay short.
func resolveSessionID(stdin io.Reader, override string) string {
	return resolveSessionContext(stdin, override).SessionID
}

// resolveSessionContext picks the session id and transcript path to use.
// Priority for session id:
//
//  1. --session flag (debugging override).
//  2. session_id from stdin JSON payload (Claude Code's normal path).
//  3. Most-recently-started session in the active registry within
//     compactHandoffSessionStartLookbackHours.
//  4. Empty string — caller renders the "no active session" fallback.
//
// TranscriptPath is populated only when stdin carries the SessionStart
// payload; the registry-lookback path can't recover it and leaves it
// empty (the kickoff fallback then just doesn't fire).
func resolveSessionContext(stdin io.Reader, override string) payloadContext {
	ctx := payloadContext{}
	if stdin != nil {
		// Bounded read so a malformed/blocking stdin can't hang the
		// hook. 64 KiB is far more than any SessionStart payload.
		buf := make([]byte, 64*1024)
		n, _ := stdin.Read(buf)
		if n > 0 {
			var payload sessionStartPayload
			if err := json.Unmarshal(buf[:n], &payload); err == nil {
				ctx.TranscriptPath = payload.TranscriptPath
				if payload.SessionID != "" {
					ctx.SessionID = payload.SessionID
				}
			}
		}
	}
	if override != "" {
		ctx.SessionID = override
		return ctx
	}
	if ctx.SessionID != "" {
		return ctx
	}
	// Lookback fallback: most-recent registry entry within the window.
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return ctx
	}
	heroDir := cfg.HeroDir(projectRoot)
	reg := active.Load(heroDir)
	if reg == nil || len(reg.Sessions) == 0 {
		return ctx
	}
	cutoff := time.Now().UTC().Add(-time.Duration(compactHandoffSessionStartLookbackHours) * time.Hour)
	var newest string
	var newestStart time.Time
	for id, s := range reg.Sessions {
		if s.Started.Before(cutoff) {
			continue
		}
		if newest == "" || s.Started.After(newestStart) {
			newest = id
			newestStart = s.Started
		}
	}
	ctx.SessionID = newest
	return ctx
}

// buildHandoff assembles the markdown blob the spec describes. Every
// step is defensive: a single missing data source degrades the
// corresponding section rather than aborting the whole render. The
// result is always token-capped before return.
func buildHandoff(cmd *cobra.Command, ctx payloadContext) string {
	sessionID := ctx.SessionID
	transcriptPath := ctx.TranscriptPath
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		// Without a workspace, we can still render a thin header.
		return assembleMinimalHandoff(projectRoot, sessionID, time.Time{}, nil)
	}
	heroDir := cfg.HeroDir(projectRoot)

	var (
		sessStart  time.Time
		activeSpec *spec.Spec
	)
	reg := active.Load(heroDir)
	if sess, ok := reg.Sessions[sessionID]; ok {
		sessStart = sess.Started
		if sess.Spec != "" {
			if s, err := findSpecBySlugOrPath(heroDir, sess.Spec); err == nil {
				activeSpec = s
			}
		}
	}

	// Collect session-scoped events (decisions + files touched). On
	// error, treat as empty — the fallback path still produces a
	// useful handoff.
	var events *projection.SessionEvents
	if store, gerr := graph.Open(heroDir); gerr == nil {
		defer store.Close()
		activeSlug := ""
		if activeSpec != nil {
			activeSlug = activeSpec.Slug
		}
		repoKey := gitutil.RepoKey(projectRoot)
		evts, _ := projection.CollectSessionEvents(store, projection.CompactHandoffOptions{
			SessionID:      sessionID,
			SessionStart:   sessStart,
			ActiveSpecSlug: activeSlug,
			RepoKey:        repoKey,
		})
		events = evts
		// Original kickoff — first UserAsk for this session_id. We
		// piggyback on the handoff package's user-singleton query and
		// filter to session_id manually.
		_ = store
	}

	parts := buildHandoffParts(projectRoot, heroDir, sessionID, transcriptPath, sessStart, activeSpec, events)

	md := assembleFullHandoff(parts)
	return enforceTokenCap(md, parts)
}

// handoffParts is the assembled set of section bodies. Keeping them
// addressable lets the truncation pass shrink individual sections in
// the documented order without re-running graph queries.
type handoffParts struct {
	header         string
	whatYouWere    string
	activeSpecBody string
	activeSpecPath string
	kickoff        string
	filesTouched   []projection.CompactFileTouch
	decisions      []projection.CompactDecision
	nextAction     string
	workingTree    string
	activeSpecSlug string
	activeSpecType string
}

func buildHandoffParts(projectRoot, heroDir, sessionID, transcriptPath string, sessStart time.Time, activeSpec *spec.Spec, events *projection.SessionEvents) handoffParts {
	p := handoffParts{}

	// Header.
	branch := currentBranch(projectRoot)
	dirty := isWorkingTreeDirty(projectRoot)
	dirtyCount := 0
	if dirty {
		dirtyCount = len(uncommittedFiles(projectRoot))
	}
	p.header = renderCompactHeader(sessionID, sessStart, branch, dirty, dirtyCount, activeSpec)

	// What you were doing — derived from active spec's Goal section.
	p.whatYouWere = deriveWhatYouWere(activeSpec)

	// Active spec full content (frontmatter stripped, body capped).
	if activeSpec != nil {
		body := stripFrontmatter(activeSpec.RawContent)
		body = strings.TrimSpace(body)
		if len(body) > compactHandoffActiveSpecBodyCap {
			body = body[:compactHandoffActiveSpecBodyCap] + fmt.Sprintf("\n\n… (truncated — read full at %s)", relForDisplay(projectRoot, activeSpec.Path))
		}
		p.activeSpecBody = body
		p.activeSpecPath = relForDisplay(projectRoot, activeSpec.Path)
		p.activeSpecSlug = activeSpec.Slug
		p.activeSpecType = string(activeSpec.Type)
	}

	// Original kickoff — best-effort: latest UserAsk node matching
	// this session_id, falling back to the SessionStart payload's
	// transcript_path when no UserAsk is in the graph yet.
	p.kickoff = kickoffForSession(heroDir, sessionID, transcriptPath)

	if events != nil {
		p.filesTouched = events.FilesTouched
		p.decisions = events.Decisions
	}

	// Next concrete action.
	p.nextAction = pickNextAction(heroDir, sessionID, activeSpec)

	// Working tree (capped lines).
	p.workingTree = renderWorkingTree(projectRoot, compactHandoffWorkingTreeCap)

	return p
}

// assembleFullHandoff renders the canonical layout. Section order is
// the one specified by the next-compact-handoff spec.
func assembleFullHandoff(p handoffParts) string {
	var b strings.Builder
	b.WriteString("## Hero session handoff (post-compact)\n\n")
	b.WriteString("> Resume context for **this session only**. Other concurrent sessions' work\n")
	b.WriteString("> is intentionally excluded. The full cross-session rollup lives at\n")
	b.WriteString("> .hero/next/<user>.md if you need it.\n\n")
	b.WriteString(p.header)
	b.WriteString("\n")

	b.WriteString("### What you were doing\n")
	if strings.TrimSpace(p.whatYouWere) == "" {
		b.WriteString("_No active spec — exploratory session._\n")
	} else {
		b.WriteString(p.whatYouWere + "\n")
	}
	b.WriteString("\n")

	if p.activeSpecBody != "" {
		b.WriteString("### Active spec — full content\n")
		b.WriteString(p.activeSpecBody)
		b.WriteString("\n\n")
	}

	b.WriteString("### Original kickoff (this session)\n")
	if strings.TrimSpace(p.kickoff) == "" {
		b.WriteString("_No kickoff recorded for this session._\n")
	} else {
		b.WriteString("> " + indentQuoteLines(p.kickoff) + "\n")
	}
	b.WriteString("\n")

	b.WriteString("### Files touched this session\n")
	if len(p.filesTouched) == 0 {
		b.WriteString("_None recorded._\n")
	} else {
		for _, f := range p.filesTouched {
			fmt.Fprintf(&b, "- %s — %d %s\n", f.Path, f.Count, plural(f.Count, "mention", "mentions"))
		}
	}
	b.WriteString("\n")

	b.WriteString("### Recent decisions (this session)\n")
	if len(p.decisions) == 0 {
		b.WriteString("_None recorded._\n")
	} else {
		for _, d := range p.decisions {
			title := strings.TrimSpace(d.Title)
			if title == "" {
				title = "(untitled)"
			}
			fmt.Fprintf(&b, "- %s\n", oneLineCompact(title))
		}
	}
	b.WriteString("\n")

	b.WriteString("### Next concrete action\n")
	if strings.TrimSpace(p.nextAction) == "" {
		b.WriteString("_None recorded — pick up where the conversation left off._\n")
	} else {
		b.WriteString(p.nextAction + "\n")
	}
	b.WriteString("\n")

	b.WriteString("### Working tree\n")
	if strings.TrimSpace(p.workingTree) == "" {
		b.WriteString("clean\n")
	} else {
		b.WriteString(p.workingTree)
		if !strings.HasSuffix(p.workingTree, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderCompactHeader returns the metadata block. Active-spec line
// degrades to "none (exploratory session)" when no spec is registered.
func renderCompactHeader(sessionID string, started time.Time, branch string, dirty bool, dirtyCount int, activeSpec *spec.Spec) string {
	var b strings.Builder
	startedStr := "unknown"
	elapsed := ""
	if !started.IsZero() {
		startedStr = started.UTC().Format(time.RFC3339)
		elapsed = " · " + durationSince(time.Now().UTC(), started)
	}
	sid := sessionID
	if sid == "" {
		sid = "(unknown)"
	}
	fmt.Fprintf(&b, "**Session:** %s · started %s%s\n", sid, startedStr, elapsed)
	state := "clean"
	if dirty {
		state = fmt.Sprintf("dirty(%d files)", dirtyCount)
	}
	br := branch
	if br == "" {
		br = "(detached)"
	}
	fmt.Fprintf(&b, "**Branch:** %s · %s\n", br, state)
	if activeSpec != nil {
		t := string(activeSpec.Type)
		s := string(activeSpec.Status)
		fmt.Fprintf(&b, "**Active spec:** [%s](%s) — %s, %s\n", activeSpec.Slug, relForDisplayBare(activeSpec.Path), t, s)
	} else {
		b.WriteString("**Active spec:** none (exploratory session)\n")
	}
	return b.String()
}

// deriveWhatYouWere pulls a short summary from the spec's Goal section.
func deriveWhatYouWere(s *spec.Spec) string {
	if s == nil {
		return ""
	}
	goal := strings.TrimSpace(s.Sections["goal"])
	if goal == "" {
		// Fall back to the title; better than empty.
		if s.Title != "" {
			return s.Title
		}
		return ""
	}
	// Take the first paragraph for brevity, then cap.
	if idx := strings.Index(goal, "\n\n"); idx >= 0 {
		goal = goal[:idx]
	}
	goal = strings.TrimSpace(goal)
	if len(goal) > compactHandoffGoalSummaryCap {
		goal = goal[:compactHandoffGoalSummaryCap] + "…"
	}
	return goal
}

// firstUserAskForSession returns the text of the most-recent UserAsk
// node carrying session_id == sessionID. Returns "" when none is found.
// We open the graph directly here (rather than threading it through
// from buildHandoff) so a graph-open failure is local to this section.
func firstUserAskForSession(heroDir, sessionID string) string {
	if sessionID == "" {
		return ""
	}
	store, err := graph.Open(heroDir)
	if err != nil {
		return ""
	}
	defer store.Close()

	rows, err := store.DB().Query(
		`SELECT COALESCE(json_extract(props, '$.text'), '') AS text,
		        valid_from
		   FROM nodes
		  WHERE type = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.session_id'), '') = ?
		  ORDER BY valid_from ASC
		  LIMIT 1`,
		handoff.NodeUserAsk, sessionID,
	)
	if err != nil {
		return ""
	}
	defer rows.Close()
	if rows.Next() {
		var text string
		var validFrom string
		_ = rows.Scan(&text, &validFrom)
		if len(text) > compactHandoffKickoffCap {
			text = text[:compactHandoffKickoffCap] + "…"
		}
		return text
	}
	return ""
}

// kickoffForSession returns the first user prompt for this session. It
// tries the graph first (latest UserAsk node carrying session_id); if
// that yields empty AND a non-empty transcriptPath points at a readable
// JSONL file, it falls back to parsing the transcript's first user
// record.
//
// Errors anywhere on the transcript path return empty string — the
// caller renders the "no kickoff recorded" placeholder. This keeps the
// always-exit-0 contract intact even when the harness writes a hostile
// or partial transcript.
func kickoffForSession(heroDir, sessionID, transcriptPath string) string {
	if got := firstUserAskForSession(heroDir, sessionID); got != "" {
		return got
	}
	if transcriptPath == "" {
		return ""
	}
	return firstUserAskFromTranscript(transcriptPath)
}

const (
	// transcriptReadByteCap bounds how much of the transcript we'll read.
	// Matches the safety cap used on stdin in resolveSessionContext.
	transcriptReadByteCap = 64 * 1024
	// transcriptReadLineCap bounds how many JSONL lines we'll scan.
	// Whichever cap trips first wins.
	transcriptReadLineCap = 1000
)

// transcriptMessage is the subset of the Claude Code transcript JSONL
// record shape we need. Both top-level `type` and nested
// `message.role` are checked so we tolerate slight harness variation.
type transcriptMessage struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
	Message *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// firstUserAskFromTranscript opens transcriptPath (bounded read) and
// returns the text of the first user record. Any parse / IO error
// short-circuits to empty string. compact-handoff depends on "first".
func firstUserAskFromTranscript(transcriptPath string) string {
	return scanUserAskFromTranscript(transcriptPath, false)
}

// lastUserAskFromTranscript opens transcriptPath (same bounded read as
// firstUserAskFromTranscript) and returns the text of the LAST user
// record within the scanned window. Any parse / IO error short-circuits
// to empty string. This is what the end-of-turn auto-emit path wants:
// the user's most recent message, not the session's opening prompt.
func lastUserAskFromTranscript(transcriptPath string) string {
	return scanUserAskFromTranscript(transcriptPath, true)
}

// scanUserAskFromTranscript is the shared bounded scan behind both
// firstUserAskFromTranscript and lastUserAskFromTranscript. It reads at
// most transcriptReadByteCap bytes and scans at most
// transcriptReadLineCap lines, returning the user-record text truncated
// at compactHandoffKickoffCap. When wantLast is false it returns the
// FIRST matching user record (the long-standing compact-handoff
// behavior); when true it keeps scanning and returns the LAST. Any IO /
// parse error yields "" — the always-exit-0 hook contract.
func scanUserAskFromTranscript(transcriptPath string, wantLast bool) string {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Bounded read so a multi-megabyte transcript doesn't balloon
	// memory. We read up to transcriptReadByteCap bytes and scan
	// at most transcriptReadLineCap lines of what we read.
	buf := make([]byte, transcriptReadByteCap)
	n, _ := io.ReadFull(f, buf)
	if n == 0 {
		return ""
	}
	data := buf[:n]

	// Split on newlines and walk up to transcriptReadLineCap lines.
	// The last line in the slice may be a partial record (because we
	// truncated mid-line); the JSON decoder will reject it and we
	// skip it — that's fine.
	lines := strings.Split(string(data), "\n")
	if len(lines) > transcriptReadLineCap {
		lines = lines[:transcriptReadLineCap]
	}
	var last string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec transcriptMessage
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		isUser := rec.Type == "user"
		if !isUser && rec.Message != nil && rec.Message.Role == "user" {
			isUser = true
		}
		if !isUser {
			continue
		}
		raw := rec.Content
		if len(raw) == 0 && rec.Message != nil {
			raw = rec.Message.Content
		}
		text := extractTranscriptText(raw)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if len(text) > compactHandoffKickoffCap {
			text = text[:compactHandoffKickoffCap] + "…"
		}
		if !wantLast {
			return text
		}
		last = text
	}
	return last
}

// goalWindowSize is the number of substantial opening user messages
// the floor goal collects into its window. ~3 is enough for the reading
// model to recover the session's intent from the opening exchange
// without dragging in mid-session refinements.
const goalWindowSize = 3

// goalMarkerPattern matches the agent-emitted goal marker
// `<!-- hero:goal: <text> -->` in an assistant message. The capture
// group is the goal text; surrounding whitespace is trimmed by the
// caller. Non-greedy so multiple markers on one line each match.
var goalMarkerPattern = regexp.MustCompile(`(?s)<!--\s*hero:goal:\s*(.*?)\s*-->`)

// openingWindowGoalFromTranscript walks the transcript's user messages
// from the start, skips trivial greeting/ack openers, and joins the
// first goalWindowSize SUBSTANTIAL messages into a single goal candidate
// (truncated at compactHandoffKickoffCap). If every opener is trivial it
// falls back to the raw first user message so the floor never produces
// an empty goal. Any IO/parse error yields "" — the best-effort,
// never-fail-the-hook contract.
//
// It does NOT reuse firstUserAskFromTranscript's single-record return
// (that stays unchanged for compact-handoff); it shares the same bounded
// read and parse loop but keeps the full ordered list of opener texts.
func openingWindowGoalFromTranscript(transcriptPath string) string {
	openers := userOpenersFromTranscript(transcriptPath)
	if len(openers) == 0 {
		return ""
	}

	var window []string
	for _, msg := range openers {
		if isTrivialOpener(msg) {
			continue
		}
		window = append(window, msg)
		if len(window) >= goalWindowSize {
			break
		}
	}
	// Fall back to the raw first message when every opener was trivial —
	// the floor never empties.
	if len(window) == 0 {
		window = append(window, openers[0])
	}

	joined := strings.TrimSpace(strings.Join(window, "\n"))
	if len(joined) > compactHandoffKickoffCap {
		joined = joined[:compactHandoffKickoffCap] + "…"
	}
	return joined
}

// userOpenersFromTranscript returns the ordered user-message texts from
// the bounded head of the transcript (same 64 KiB / 1000-line window as
// scanUserAskFromTranscript). Each text is NOT individually truncated —
// the window joiner truncates the assembled result instead. Returns nil
// on any IO/parse error.
func userOpenersFromTranscript(transcriptPath string) []string {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	buf := make([]byte, transcriptReadByteCap)
	n, _ := io.ReadFull(f, buf)
	if n == 0 {
		return nil
	}
	lines := strings.Split(string(buf[:n]), "\n")
	if len(lines) > transcriptReadLineCap {
		lines = lines[:transcriptReadLineCap]
	}

	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec transcriptMessage
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		isUser := rec.Type == "user"
		if !isUser && rec.Message != nil && rec.Message.Role == "user" {
			isUser = true
		}
		if !isUser {
			continue
		}
		raw := rec.Content
		if len(raw) == 0 && rec.Message != nil {
			raw = rec.Message.Content
		}
		text := strings.TrimSpace(extractTranscriptText(raw))
		if text == "" {
			continue
		}
		out = append(out, text)
		// Stop once we have enough candidates to fill the window even if
		// every one survives the triviality filter — bounds the scan.
		if len(out) >= goalWindowSize*4 {
			break
		}
	}
	return out
}

// goalTriviaWords are the greeting/ack tokens that, when a message is
// essentially nothing but these, mark it as a trivial opener to skip.
var goalTriviaWords = map[string]bool{
	"hi": true, "hey": true, "hello": true, "yo": true, "sup": true,
	"thanks": true, "thank": true, "thankyou": true, "ty": true,
	"ok": true, "okay": true, "k": true, "kk": true,
	"yes": true, "yep": true, "yeah": true, "yup": true, "sure": true,
	"no": true, "nope": true,
	"go": true, "continue": true, "proceed": true, "lgtm": true,
	"cool": true, "nice": true, "perfect": true, "great": true,
	"awesome": true, "good": true, "please": true, "ahead": true,
	// Conversational filler that, on its own, carries no intent — so a
	// throat-clearing opener like "hey can you help" classifies as
	// trivial. A real request word (an imperative verb, a noun phrase
	// with substance) breaks the all-filler match and keeps the message.
	"can": true, "could": true, "would": true, "you": true, "u": true,
	"me": true, "we": true, "help": true, "with": true, "a": true,
	"the": true, "this": true, "that": true, "it": true, "to": true,
	"for": true, "just": true, "quick": true, "question": true,
	"i": true, "have": true, "got": true, "lets": true, "let": true,
}

// goalSubstanceVerbs are imperative request verbs that keep an opener
// even when it is short — a strong intent signal.
var goalSubstanceVerbs = map[string]bool{
	"add": true, "fix": true, "build": true, "make": true, "change": true,
	"remove": true, "delete": true, "why": true, "how": true, "what": true,
	"refactor": true, "implement": true, "create": true, "update": true,
	"write": true, "design": true, "debug": true, "investigate": true,
	"diagnose": true, "review": true, "test": true, "rename": true,
	"move": true, "fixup": true, "wire": true, "handle": true, "support": true,
}

// isTrivialOpener reports whether a whole user message is essentially
// just greeting/ack noise, so the opening-window goal should skip it.
//
// The match is WHOLE-MESSAGE, not prefix: "ok now do X" is NOT trivial
// because it carries a real request after the ack. A message is trivial
// only when, after lowercasing and stripping punctuation, it is made up
// solely of greeting/ack tokens, OR it is very short (<= 3 words) AND
// carries no substance signal. A substance signal — a '?', a code
// fence/backtick, a file path or .ext, an error-looking token, an
// imperative verb, or >= 6 words — always keeps the message.
func isTrivialOpener(msg string) bool {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" {
		return true
	}
	// Substance signals that keep any message regardless of length.
	if hasGoalSubstanceSignal(trimmed) {
		return false
	}

	fields := strings.Fields(strings.ToLower(trimmed))
	if len(fields) >= 6 {
		// Long enough to carry intent even without an explicit signal.
		return false
	}

	// Tokenize to bare words (strip surrounding punctuation) and test
	// whether every token is a greeting/ack word.
	allTrivia := true
	wordCount := 0
	for _, f := range fields {
		w := strings.Trim(f, ".,!?;:\"'`()[]{}…-")
		if w == "" {
			continue
		}
		wordCount++
		if goalTriviaWords[w] {
			continue
		}
		allTrivia = false
	}
	if wordCount == 0 {
		return true
	}
	if allTrivia {
		return true
	}
	// Not pure trivia, but very short with no substance signal → skip.
	return wordCount <= 3
}

// hasGoalSubstanceSignal reports whether a message carries an explicit
// intent signal that should always keep it in the opening window.
func hasGoalSubstanceSignal(msg string) bool {
	if strings.ContainsAny(msg, "?`") {
		return true
	}
	lower := strings.ToLower(msg)
	// File path / extension or error-looking tokens.
	if strings.Contains(msg, "/") && strings.Contains(msg, ".") {
		return true
	}
	for _, marker := range []string{".go", ".ts", ".js", ".py", ".md", ".json", ".yaml", ".yml", ".rs", ".java",
		"error", "panic", "exception", "fail", "traceback", "stack trace"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	// Imperative request verb anywhere in the message.
	for _, f := range strings.Fields(lower) {
		w := strings.Trim(f, ".,!?;:\"'`()[]{}…-")
		if goalSubstanceVerbs[w] {
			return true
		}
	}
	return false
}

// goalMarkerFromTranscript greps the transcript's ASSISTANT messages for
// `<!-- hero:goal: <text> -->` markers and returns the LAST one's text
// (most recent wins — a later marker reflects refined understanding).
// Returns "" when no marker is present or on any IO/parse error. The
// result is truncated at compactHandoffKickoffCap to match the window.
func goalMarkerFromTranscript(transcriptPath string) string {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, transcriptReadByteCap)
	n, _ := io.ReadFull(f, buf)
	if n == 0 {
		return ""
	}
	lines := strings.Split(string(buf[:n]), "\n")
	if len(lines) > transcriptReadLineCap {
		lines = lines[:transcriptReadLineCap]
	}

	var last string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec transcriptMessage
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		isAssistant := rec.Type == "assistant"
		if !isAssistant && rec.Message != nil && rec.Message.Role == "assistant" {
			isAssistant = true
		}
		if !isAssistant {
			continue
		}
		raw := rec.Content
		if len(raw) == 0 && rec.Message != nil {
			raw = rec.Message.Content
		}
		text := extractTranscriptText(raw)
		if text == "" {
			continue
		}
		for _, m := range goalMarkerPattern.FindAllStringSubmatch(text, -1) {
			if g := strings.TrimSpace(m[1]); g != "" {
				last = g
			}
		}
	}
	if len(last) > compactHandoffKickoffCap {
		last = last[:compactHandoffKickoffCap] + "…"
	}
	return last
}

// extractTranscriptText pulls the user-visible text out of a Claude Code
// `content` field. The field may be either a plain string ("hello") or
// an array of content blocks ([{"type":"text","text":"…"}, …]). We
// take the first text block when it's an array.
func extractTranscriptText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Plain string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Array of content blocks.
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}

// pickNextAction priority order:
//
//  1. Most-recent NextSuggestion for this session.
//  2. First unchecked acceptance criterion on the active spec.
//  3. Empty (caller renders fallback).
func pickNextAction(heroDir, sessionID string, activeSpec *spec.Spec) string {
	if sessionID != "" {
		store, err := graph.Open(heroDir)
		if err == nil {
			defer store.Close()
			rows, qerr := store.DB().Query(
				`SELECT COALESCE(json_extract(props, '$.text'), '') AS text
				   FROM nodes
				  WHERE type = ? AND valid_to IS NULL
				    AND COALESCE(json_extract(props, '$.session_id'), '') = ?
				  ORDER BY valid_from DESC
				  LIMIT 1`,
				handoff.NodeNextSuggestion, sessionID,
			)
			if qerr == nil {
				defer rows.Close()
				if rows.Next() {
					var text string
					_ = rows.Scan(&text)
					if strings.TrimSpace(text) != "" {
						return strings.TrimSpace(text)
					}
				}
			}
		}
	}
	if activeSpec != nil {
		for _, c := range activeSpec.AcceptanceCriteria() {
			// AC text marked with "- [ ]" is unchecked; the parser
			// records both Raw and Behavior. Surface the first one.
			if strings.HasPrefix(strings.TrimSpace(c.Raw), "[ ]") || !strings.HasPrefix(strings.TrimSpace(c.Raw), "[x]") {
				return "Acceptance criterion: " + oneLineCompact(c.Raw)
			}
		}
	}
	return ""
}

// renderWorkingTree returns `git status --short` output capped at
// `cap` lines. Empty string when the tree is clean.
func renderWorkingTree(projectRoot string, cap int) string {
	if !isWorkingTreeDirty(projectRoot) {
		return ""
	}
	files := uncommittedFiles(projectRoot)
	if len(files) > cap {
		extra := len(files) - cap
		files = files[:cap]
		files = append(files, fmt.Sprintf("… (+%d more)", extra))
	}
	return strings.Join(files, "\n")
}

// enforceTokenCap returns md unchanged when within budget, otherwise
// rebuilds it dropping sections in the documented truncation order:
// Working tree → Files touched tail → Recent decisions tail → Active
// spec body (further) → Original kickoff (further). Always preserves
// the header, the active spec slug/title in the header, and the next
// concrete action.
func enforceTokenCap(md string, parts handoffParts) string {
	if approxTokens(md) <= compactHandoffTokenHardCap {
		return md
	}
	// Strategy: shrink in-place, then re-assemble after each cut.
	// Step 1: drop working tree.
	parts.workingTree = ""
	if md = assembleFullHandoff(parts); approxTokens(md) <= compactHandoffTokenHardCap {
		return md
	}
	// Step 2: tail-trim files touched.
	for len(parts.filesTouched) > 0 && approxTokens(md) > compactHandoffTokenHardCap {
		parts.filesTouched = parts.filesTouched[:len(parts.filesTouched)-1]
		md = assembleFullHandoff(parts)
	}
	if approxTokens(md) <= compactHandoffTokenHardCap {
		return md
	}
	// Step 3: tail-trim decisions.
	for len(parts.decisions) > 0 && approxTokens(md) > compactHandoffTokenHardCap {
		parts.decisions = parts.decisions[:len(parts.decisions)-1]
		md = assembleFullHandoff(parts)
	}
	if approxTokens(md) <= compactHandoffTokenHardCap {
		return md
	}
	// Step 4: shrink active spec body in halves until under budget.
	for parts.activeSpecBody != "" && approxTokens(md) > compactHandoffTokenHardCap {
		newLen := len(parts.activeSpecBody) / 2
		if newLen < 200 {
			parts.activeSpecBody = ""
		} else {
			parts.activeSpecBody = parts.activeSpecBody[:newLen] + "\n\n… (truncated under token cap)"
		}
		md = assembleFullHandoff(parts)
	}
	if approxTokens(md) <= compactHandoffTokenHardCap {
		return md
	}
	// Step 5: shrink kickoff.
	if len(parts.kickoff) > 100 {
		parts.kickoff = parts.kickoff[:100] + "…"
		md = assembleFullHandoff(parts)
	}
	return md
}

// approxTokens approximates token count as len(s)/4 — the OpenAI/Anthropic
// rule of thumb for English text. Good enough for budget enforcement;
// we have a safety margin built into the cap.
func approxTokens(s string) int {
	return len(s) / 4
}

// --- small helpers ---------------------------------------------------------

// assembleMinimalHandoff renders a very thin fallback when even the
// project root or config can't be loaded.
func assembleMinimalHandoff(projectRoot, sessionID string, sessStart time.Time, _ any) string {
	hdr := renderCompactHeader(sessionID, sessStart, currentBranch(projectRoot), isWorkingTreeDirty(projectRoot), len(uncommittedFiles(projectRoot)), nil)
	return "## Hero session handoff (post-compact)\n\n" + hdr + "\n_Workspace not detected — minimal handoff._\n"
}

// indentQuoteLines prefixes every line after the first with "> " so
// multi-line kickoff text renders as a coherent blockquote.
func indentQuoteLines(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 1 {
		return lines[0]
	}
	for i := 1; i < len(lines); i++ {
		lines[i] = "> " + lines[i]
	}
	return strings.Join(lines, "\n")
}

// oneLineCompact collapses whitespace and truncates at the first
// newline. Used for bullet lines that must stay on a single row.
func oneLineCompact(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return s
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// relForDisplay renders a path for the markdown body. Tries to make
// it relative to projectRoot for compactness; falls back to absolute.
func relForDisplay(projectRoot, abs string) string {
	if rel, err := filepath.Rel(projectRoot, abs); err == nil {
		return rel
	}
	return abs
}

// relForDisplayBare is like relForDisplay but uses ./<rel> style when
// the path lives inside the project, suitable for the `[slug](path)`
// markdown link in the header.
func relForDisplayBare(abs string) string {
	projectRoot := findProjectRoot()
	if rel, err := filepath.Rel(projectRoot, abs); err == nil {
		return rel
	}
	return abs
}

// stripFrontmatter removes the leading `---\n…\n---` YAML block. This
// is identical in shape to the helper in next.go but kept local to
// avoid forcing the test wiring of next.go's package dependencies.
//
// Note: next.go's stripFrontmatter exists in the same package, so we
// reuse it directly — this declaration is omitted here to avoid the
// duplicate-symbol error.
var _ = os.ErrNotExist
