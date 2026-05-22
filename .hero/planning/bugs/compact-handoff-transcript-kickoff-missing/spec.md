---
title: Compact handoff kickoff never reads transcript_path — silent fallback to placeholder
slug: compact-handoff-transcript-kickoff-missing
type: bug
status: completed
severity: medium
root_cause_class: code
priority: medium
tags: [next, compact-handoff, kickoff, transcript]
created: 2026-05-21
relates-to: [next-compact-handoff, compact-handoff-test-coverage]
---

# Compact handoff kickoff never reads `transcript_path` — silent fallback to placeholder

## Kickoff

The compact handoff promises an "Original kickoff (this session)" section. When no `UserAsk` graph node exists for the session, the [next-compact-handoff spec](../../features/next-compact-handoff/spec.md) says fall back to the first user prompt from `transcript_path` (the JSONL transcript Claude Code points to in the SessionStart payload). That fallback was never implemented.

**Status:** planning — diagnosis complete, fix not yet written.

**Pick up at:** thread `TranscriptPath` from `resolveSessionID` through to a new `firstUserAskWithTranscriptFallback(heroDir, sessionID, transcriptPath)`. When the graph lookup returns empty, open the transcript (bounded read), scan for the first `role: "user"` record, extract `content`, truncate at `compactHandoffKickoffCap`. Errors return empty — the section gracefully shows `_No kickoff recorded for this session._` exactly as today.

→ [internal/cli/next_compact_handoff.go:310](../../../internal/cli/next_compact_handoff.go:310)

**Files to touch:** `internal/cli/next_compact_handoff.go`, `internal/cli/next_compact_handoff_test.go`.

## Issue

Surfaced by the engineer during delivery of [compact-handoff-test-coverage](../../features/compact-handoff-test-coverage/spec.md):

> "Transcript-path fallback is not implemented. `next_compact_handoff.go:firstUserAskForSession` returns "" when no UserAsk is in the graph for the session. The spec text says it should 'fall back to the first user prompt from `transcript_path`.' Today it just renders the `_No kickoff recorded for this session._` placeholder."

Real-world impact: any session that compacts before emitting an explicit `UserAsk` event — which is most sessions in practice, because `hero next ask` emission isn't routine — gets an empty kickoff. The compact handoff loses one of its most useful sections (the framing question that started the work) for the common case.

The MVP smoke test against this Hero repo's own state showed this directly: `Original kickoff (this session): _No kickoff recorded for this session._`

## Investigation

### The current code path

`internal/cli/next_compact_handoff.go:310` calls:

```go
p.kickoff = firstUserAskForSession(heroDir, sessionID)
```

`firstUserAskForSession` at line 462–500 queries the graph:

```go
SELECT COALESCE(json_extract(props, '$.text'), '') AS text, valid_from
  FROM nodes
 WHERE type = ? AND valid_to IS NULL
   AND COALESCE(json_extract(props, '$.session_id'), '') = ?
 ORDER BY valid_from ASC
 LIMIT 1
```

When the query returns no rows, the function returns `""`. There is no fallback path.

### Why the fallback was never wired

The SessionStart payload type at line 149 *does* parse `transcript_path`:

```go
type sessionStartPayload struct {
    SessionID      string `json:"session_id"`
    TranscriptPath string `json:"transcript_path"`
    CWD            string `json:"cwd"`
    Source         string `json:"source"`
    HookEventName  string `json:"hook_event_name"`
}
```

But `resolveSessionID` at line 164 returns only the `session_id` string — `TranscriptPath` is read off stdin and immediately discarded. No code path threads it forward.

### Transcript format (Claude Code)

Claude Code transcripts are JSONL files. Each line is a JSON object representing one turn. The relevant shape (per Claude Code docs and observed transcripts):

```jsonl
{"type":"user","timestamp":"2026-05-21T...","message":{"role":"user","content":"<text or content blocks>"}}
{"type":"assistant","timestamp":"...","message":{"role":"assistant","content":"..."}}
{"type":"system","timestamp":"...","content":"..."}
```

The first user message is the kickoff prompt. `content` may be a plain string OR an array of content blocks (text + tool use); when it's an array, the first text block's text is the prompt.

Codex's transcript format is similar JSONL but field shapes differ slightly. For this fix, we target Claude Code's shape and gracefully no-op if parsing fails (so Codex sessions still render the empty placeholder rather than crashing — and Codex parity follows in a separate ticket).

### Root cause class — code

This is a logic-not-implemented gap. The spec described the behavior; the engineer wrote the graph path; the transcript path was never written. No deeper design issue.

### Why it didn't surface in MVP testing

The test that *does* exist (`TestKickoffFromTranscript_WhenNoUserAsk` in `internal/cli/next_compact_handoff_test.go`, added during the coverage delivery) asserts the **current** behavior — graph-empty returns empty string. It doesn't assert the transcript-fallback behavior, because the engineer correctly noted that the production code doesn't implement it. The test is a documentation marker for the gap.

## Suggested Fix Approach

### 1. Thread `transcript_path` forward

Change `resolveSessionID` to return a `payloadContext` struct instead of just a string:

```go
type payloadContext struct {
    SessionID      string
    TranscriptPath string
}

func resolveSessionContext(stdin io.Reader, override string) payloadContext { ... }
```

Update call sites in `runNextCompactHandoff` and any test that calls `resolveSessionID` directly. Keep a `resolveSessionID` wrapper for tests that only care about the ID, to minimize test churn.

### 2. New kickoff resolver with fallback

Replace the `firstUserAskForSession(heroDir, sessionID)` call site with:

```go
p.kickoff = kickoffForSession(heroDir, sessionID, transcriptPath)
```

Where `kickoffForSession`:

1. Calls existing `firstUserAskForSession` first.
2. If empty AND `transcriptPath != ""` AND file exists AND is readable, opens it.
3. Scans line-by-line (bounded — stop after the first user message OR after a bytes/lines cap to prevent runaway on a malformed transcript). Use the same 64 KiB / 1000-line caps the safety contract already imposes elsewhere.
4. Extracts `content` from the first record whose `type == "user"` or `message.role == "user"`. Handles both string-content and content-blocks shapes (take the first text block).
5. Trims, truncates at `compactHandoffKickoffCap`, returns.
6. Any parse error → return empty string. Never crash.

### 3. Tests to add

- `TestKickoffForSession_FallbackToTranscript` — graph empty, transcript has a `role: "user"` first line → returns the text.
- `TestKickoffForSession_TranscriptStringContent` — `content` is plain string.
- `TestKickoffForSession_TranscriptContentBlocks` — `content` is `[{"type":"text","text":"..."}]`.
- `TestKickoffForSession_MalformedTranscriptReturnsEmpty` — invalid JSONL → empty, no crash.
- `TestKickoffForSession_MissingTranscriptReturnsEmpty` — `transcript_path` set but file missing → empty, no crash.
- `TestKickoffForSession_GraphHitWinsOverTranscript` — graph has a UserAsk → returns that, transcript not consulted.

Also update `TestKickoffFromTranscript_WhenNoUserAsk` (currently asserts empty for missing graph) to assert the transcript-derived value.

### 4. Safety constraints (unchanged)

- Bounded reads. 64 KiB or 1000 lines, whichever first.
- All transcript errors → empty string, never panic, never block.
- Must not call into the graph store if `sessionID == ""`.
- Must not open `transcriptPath` if it's empty.

## Boundaries

- **Out of scope:** Codex transcript format support. Codex's JSONL shape differs; a separate ticket will handle it when Codex installer also lands.
- **Out of scope:** caching parsed transcripts. Each compaction reads the file fresh; compactions are infrequent.
- **Out of scope:** richer kickoff extraction (e.g., "first non-trivial user message"). First user message wins; if it's empty, we just return empty.

## Acceptance Criteria

- [ ] `firstUserAskForSession` (or its replacement) accepts `transcriptPath` and falls back to it when the graph is empty.
- [ ] Plain-string `content` and content-blocks `content` both extract correctly.
- [ ] Malformed transcript / missing file / empty `transcript_path` all return empty without crash.
- [ ] Graph hit short-circuits — transcript is not opened when a UserAsk exists for the session.
- [ ] Six new tests cover the cases listed above.
- [ ] Updated test `TestKickoffFromTranscript_WhenNoUserAsk` now asserts the actual fallback behavior, not the placeholder.
- [ ] Smoke test against this repo with a real Claude Code transcript produces a non-empty kickoff for a session that has no UserAsk in the graph.
- [ ] Always-exit-0 contract unchanged. Defer-panic still wraps `runCompactHandoff`.

## Changes

- `internal/cli/next_compact_handoff.go` — added `payloadContext` struct, `resolveSessionContext` (keeping `resolveSessionID` as a thin wrapper), `kickoffForSession`, `firstUserAskFromTranscript`, `extractTranscriptText`; threaded `TranscriptPath` through `buildHandoff` → `buildHandoffParts`.
- `internal/cli/next_compact_handoff_test.go` — added eight tests for the transcript fallback (`TestKickoffForSession_*`); updated `TestKickoffFromTranscript_WhenNoUserAsk` to assert the recovered prompt; updated existing `buildHandoff` / `buildHandoffParts` call sites to the new signatures.
