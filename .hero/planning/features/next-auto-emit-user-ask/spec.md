---
title: "Auto-Emit UserAsk on End-of-Turn Checkpoint"
slug: next-auto-emit-user-ask
type: feature
status: planning
priority: high
severity: high
size: small
domain: engineering
created: 2026-06-03
origin: session
relates-to:
  - next-as-projection
  - handoff-one-call-simplification
  - next-unconditional-commit-staging
root_cause_class: design
---

# Auto-Emit UserAsk on End-of-Turn Checkpoint

## Context

This is the root cause of the user's #1 felt pain — handoff drift/staleness
that has persisted "for two years."

Hero's `next-as-projection` work (now archived in `.hero/specs/next-as-projection/`)
made the projected handoff files structurally fresh: NEXT.md, `.hero/next/<user>.md`,
and `.hero/next/<user>.local.md` are total-rewritten from the graph on **every**
end-of-turn checkpoint. The Stop hook runs `hero next checkpoint --quiet`
(`internal/install/claude_hooks.go:25`, `claudeHookEvents = ["Stop", "PreCompact", "SessionStart"]`
at `:50`), so the *render* never goes stale.

But the most valuable handoff content — the user's last ask, the suggested next
step, in-session reflections — only enters the graph when the **agent manually**
runs `hero next ask` / `hero next suggest` / `hero next reflection`. There is no
auto-emit. So the projection is a perfect re-render of a **stale graph**:
structurally fresh, semantically stale. The "Last user ask" section in
`.hero/next/<user>.md` shows whatever the agent last bothered to type — or the
`_(none recorded — hero next ask "..." to set)_` placeholder forever.

The fix folds **auto-emit of `UserAsk`** into the end-of-turn checkpoint path so
freshness of the single highest-frequency field stops depending on agent
discipline. The enabling functions already exist in
`internal/cli/next_compact_handoff.go` — this is a re-wire, not new capability.

This spec is Phase-1 of a small cluster of handoff-freshness fixes:
`handoff-one-call-simplification` (collapse the manual field-grab surface) and
`next-unconditional-commit-staging` (make projected files travel with commits).

## Goal

After an end-of-turn Stop checkpoint that carries a transcript payload
(`session_id` + `transcript_path` on stdin, as Claude Code provides), a `UserAsk`
graph node reflecting the user's **last** message exists in the graph and renders
in `.hero/next/<user>.md` — with **no** manual `hero next ask`. When no transcript
payload is present (other harnesses, the git post-commit fallback), checkpoint is
a clean no-op for auto-emit: it never errors, never hangs, and falls back to
whatever ask is already in the graph. Singleton supersession guarantees re-emit
(or a same-turn manual `hero next ask`) produces exactly one node, not duplicates.

## Root Cause

The drift is structural: **projection and emission are split, and only projection
is automatic.**

- **Projection (automatic, every turn):** `writeCheckpoint()` in
  `internal/cli/checkpoint.go` total-rewrites the handoff files from the graph.
  Confirmed by reading the whole function (`checkpoint.go:90-213`): it performs
  **zero** graph mutations — no `RecordAsk`, `RecordSuggestion`, or
  `RecordReflection` calls. It only *reads* the graph (via
  `writeProjectedNextMD`, `writeUserHandoffFile`, `projectSnapshot`).

- **Emission (manual, agent-discipline-gated):** The only callers of
  `handoff.RecordAsk` / `RecordSuggestion` / `RecordReflection` are the manual
  `hero next ask|suggest|reflection` commands (`internal/cli/next_handoff.go:175-266`),
  plus the round-trip `ingest` path and the projection→migration paths. Nothing
  on the end-of-turn path emits.

Because the checkpoint never emits, the graph's UserAsk node is only as fresh as
the last time the agent *chose* to type `hero next ask`. The projection faithfully
re-renders that stale node every turn. The user experiences a handoff that "never
updates itself."

The fix is dead-center on Hero's mission: the model is stateless and only knows
what someone thinks to inject — auto-emit removes the "someone has to think to
inject the user's own ask" step.

## Source

Confirmed against source this session (all line numbers verified):

| File | Lines | What it proves |
|------|-------|----------------|
| `internal/cli/checkpoint.go` | 64-73 | `runNextCheckpoint` — cobra entry the Stop hook calls. Does **not** read stdin. |
| `internal/cli/checkpoint.go` | 90-213 | `writeCheckpoint` — projects from graph, **zero** `Record*` calls. |
| `internal/install/claude_hooks.go` | 25, 50 | Stop hook command is `hero next checkpoint --quiet`; events include `Stop`. |
| `internal/cli/hook.go` | 94-104 | git `post-commit` fallback also calls `writeCheckpoint()` — no transcript available there. |
| `internal/handoff/handoff.go` | 46-58, 101-131 | `UserAsk` is a per-`(user,repo,domain)` singleton; `RecordAsk` upserts by `singletonKey` (supersedes prior); empty text clears. |
| `internal/handoff/handoff.go` | 95-99 | `singletonKey(user, domain)` — supersession key. |
| `internal/cli/next_handoff.go` | 175-266 | `runNextSuggest/Ask/Reflection` — the **only** manual callers of `Record*`. No auto-emit exists. |
| `internal/cli/next_compact_handoff.go` | 150-156 | `sessionStartPayload` struct — parses `session_id` + `transcript_path` from Stop/SessionStart stdin. |
| `internal/cli/next_compact_handoff.go` | 186-235 | `resolveSessionContext` — bounded 64 KiB stdin read, JSON-unmarshal, returns `payloadContext{SessionID, TranscriptPath}`. **Reusable.** |
| `internal/cli/next_compact_handoff.go` | 556-636 | `firstUserAskFromTranscript` + bounded-read caps (`transcriptReadByteCap=64KiB`, `transcriptReadLineCap=1000`); `extractTranscriptText` (542-664). Takes the **first** user record — the new variant takes the **last**. **Reusable.** |
| `internal/projection/user_handoff.go` | 64-73 | "Last user ask" render — reads `handoff.LatestAsk`, shows placeholder when none. The surface auto-emit feeds. |
| `internal/projection/user_handoff.go` | 150-186 | `PickUserSuggestion` — mechanical floor for suggested-next (top open Feature → Initiative → empty). Untouched by this spec. |
| `internal/cli/next.go` | 91-101 | `nextUserSlug(cfg)` — the user slug auto-emit records under. |

## Approach

Re-wire existing functions; add no new capability.

The Stop hook already pipes Claude Code's `{session_id, transcript_path, ...}`
JSON to `hero next checkpoint` via stdin — but `runNextCheckpoint` ignores it.
The transcript-parsing machinery that `compact-handoff` uses to recover the
*first* user message is exactly what auto-emit needs, except it wants the *last*.

1. **Read the Stop-hook payload in the checkpoint path.** `runNextCheckpoint`
   (`checkpoint.go:64`) gains access to `cmd.InOrStdin()` and calls the existing
   `resolveSessionContext(cmd.InOrStdin(), "")` to get `{SessionID, TranscriptPath}`.
   This is the *same* bounded-read, never-hang, JSON-tolerant parser
   `compact-handoff` already trusts.

2. **Extract the LAST user message.** Add `lastUserAskFromTranscript(path)` next
   to `firstUserAskFromTranscript` (`next_compact_handoff.go:580`) — identical
   bounded-read discipline (`transcriptReadByteCap`/`transcriptReadLineCap`,
   `extractTranscriptText`), but it keeps scanning and returns the **last**
   matching user record instead of returning on the first. Same length cap
   (`compactHandoffKickoffCap = 400`, `next_compact_handoff.go:49`) so a giant
   paste is truncated identically to the existing path.

3. **Emit `UserAsk`.** When the transcript yields a non-empty last user message,
   open the graph and call `handoff.RecordAsk(store, repoKey, handoff.UserAsk{
   User: nextUserSlug(cfg), Domain: graph.DomainFor(cfg, graph.IntrinsicActive),
   Text: lastAsk, SessionID: ctx.SessionID})`. Singleton supersession
   (`handoff.go:101-131`) overwrites the prior node cleanly — no duplicate nodes.
   Best-effort: a graph-open or record error logs to stderr and the checkpoint
   continues (same non-fatal contract as `writeUserHandoffFile` /
   `projectSnapshot` already use, `checkpoint.go:199-210`).

4. **Order: emit before projecting.** The auto-emit must run *before*
   `writeUserHandoffFile` so the just-recorded `UserAsk` lands in the same
   checkpoint's `.hero/next/<user>.md` render — not one turn late.

5. **No-payload path is a structural no-op.** When stdin carries nothing parseable
   (cursor/codex/generic harnesses, the `hook.go:94` post-commit fallback),
   `resolveSessionContext` returns an empty `TranscriptPath`; auto-emit reads no
   transcript, records nothing, and the projection falls back to whatever
   `LatestAsk` already holds. No error, no hang.

Boundaries below spell out what auto-emit explicitly does **not** do
(NextSuggestion, SessionReflection).

## Acceptance Criteria

- WHEN `hero next checkpoint` runs with a Stop-hook stdin payload carrying a readable `transcript_path` THE SYSTEM SHALL record a `UserAsk` graph node whose text is the last user message from that transcript, with no manual `hero next ask`.
- WHEN that checkpoint completes THE SYSTEM SHALL render the recorded ask in the "Last user ask" section of `.hero/next/<user>.md` in the same checkpoint pass (not one turn later).
- IF the stdin payload carries no `transcript_path` (other harnesses, git post-commit fallback) THEN THE SYSTEM SHALL skip auto-emit as a no-op, leave any existing `UserAsk` untouched, and not error or hang.
- IF the transcript file is missing, unreadable, or malformed JSONL THEN THE SYSTEM SHALL skip auto-emit and complete the checkpoint normally (always-exit-0 / never-fail-the-hook contract preserved).
- WHILE auto-emit reads the transcript THE SYSTEM SHALL bound the read to 64 KiB and 1000 lines so an oversized or blocking transcript never hangs the Stop hook.
- WHEN the same user ask is emitted twice (auto-emit after a same-turn manual `hero next ask`, or auto-emit on consecutive turns with the same last message) THE SYSTEM SHALL keep exactly one current `UserAsk` node via singleton supersession — no duplicate nodes.
- THE SYSTEM SHALL truncate the recorded ask text to the existing kickoff cap (`compactHandoffKickoffCap = 400`) so a large paste does not bloat the graph node.
- THE SYSTEM SHALL leave `NextSuggestion` (floor + optional agent ceiling) and `SessionReflection` (agent-emitted) behavior unchanged.

## Changes

1. **`internal/cli/next_compact_handoff.go` — add `lastUserAskFromTranscript`.**
   - Clone `firstUserAskFromTranscript` (`:580-636`) into a `last` variant: same
     `os.Open`, same bounded `io.ReadFull` into a `transcriptReadByteCap` buffer,
     same line-split capped at `transcriptReadLineCap`, same `extractTranscriptText`.
   - Instead of `return text` on the first user record, store it in a `last`
     variable and keep scanning; return `last` after the loop.
   - Apply the same `compactHandoffKickoffCap` truncation to the returned text.
   - Factor the shared scan into a helper if it reduces duplication, but do not
     change `firstUserAskFromTranscript`'s behavior (compact-handoff depends on
     "first").

2. **`internal/cli/checkpoint.go` — wire auto-emit into the checkpoint entry.**
   - In `runNextCheckpoint` (`:64`), before calling `writeCheckpoint()`, call a
     new `autoEmitUserAsk(cmd.InOrStdin())` (or thread the resolved
     `payloadContext` into `writeCheckpoint`).
   - `autoEmitUserAsk` resolves `ctx := resolveSessionContext(stdin, "")`; if
     `ctx.TranscriptPath == ""` return immediately (no-op).
   - Read `lastUserAskFromTranscript(ctx.TranscriptPath)`; if empty, return.
   - Load config, open the graph (`graph.Open(heroDir)`), derive
     `user := nextUserSlug(cfg)`, `repoKey := gitutil.RepoKey(projectRoot)`,
     `domain := graph.DomainFor(cfg, graph.IntrinsicActive)`.
   - Call `handoff.RecordAsk(store, repoKey, handoff.UserAsk{User: user,
     Domain: domain, Text: lastAsk, SessionID: ctx.SessionID})`.
   - All errors are non-fatal: log to stderr, continue. The auto-emit must run
     **before** the `writeUserHandoffFile` projection so the new ask renders this
     turn.

3. **No change to `internal/handoff/handoff.go`.** `RecordAsk` singleton
   supersession is already correct-by-construction for re-emit.

4. **No change to `internal/projection/user_handoff.go`.** The "Last user ask"
   render already reads `LatestAsk`; it just starts getting fresh data.

5. **Documentation/skill touch (optional, low-priority).** The
   `next-handoff-emit` skill currently frames UserAsk emission as an agent
   discipline. A one-line note that UserAsk is now auto-emitted on checkpoint
   (agent emission becomes a manual override, not a requirement) keeps the skill
   honest. Defer if out of scope for the first delivery.

## Test Plan

Existing coverage to build on (`internal/cli/next_compact_handoff_test.go`):
- `writeTranscript(dir, lines...)` helper (`:863-874`) writes a JSONL transcript
  and returns its path — reuse directly.
- `TestKickoffForSession_FallbackToTranscript` / `_MalformedTranscriptReturnsEmpty`
  / `_MissingTranscriptReturnsEmpty` (`:874-929`) already prove the bounded-read
  and malformed-input no-op behavior for the *first*-message path. Mirror them
  for *last*.
- `internal/cli/checkpoint_test.go` for the checkpoint integration harness.

New tests:

1. **`lastUserAskFromTranscript` unit tests** (in `next_compact_handoff_test.go`):
   - Multi-user-message transcript → returns the **last** user message (assert it
     differs from what `firstUserAskFromTranscript` returns on the same fixture).
   - Malformed JSONL → empty string, no panic.
   - Missing path → empty string.
   - Oversized transcript (> 64 KiB, > 1000 lines) → returns within bounds, never
     hangs; assert the returned text is from within the scanned window.
   - Text longer than `compactHandoffKickoffCap` → truncated with the `…` marker.

2. **Auto-emit integration test** (checkpoint-level):
   - Build a temp workspace + graph, write a transcript fixture with a known last
     user message, invoke the checkpoint with that stdin payload, then assert a
     `UserAsk` node exists in the graph (via `handoff.LatestAsk`) with the
     expected text, **and** that `.hero/next/<user>.md` renders it (no
     `_(none recorded…)_` placeholder).

3. **No-payload no-op test:**
   - Invoke checkpoint with empty stdin (no `transcript_path`). Assert: no error,
     checkpoint succeeds, and any pre-existing `UserAsk` is unchanged (and no new
     node created).

4. **Singleton-supersession test:**
   - Manually `RecordAsk` a value, then run auto-emit with a transcript whose last
     message differs. Assert exactly one current (`valid_to IS NULL`) `UserAsk`
     node for the `(user, repo, domain)` triple afterward, carrying the
     auto-emitted text.

Regression scope:
- `compact-handoff` must be unaffected — `firstUserAskFromTranscript` behavior
  unchanged; run its existing tests.
- Checkpoint projection paths (`writeProjectedNextMD`, `writeUserHandoffFile`,
  `projectSnapshot`) must still run when auto-emit no-ops or errors.
- The git `post-commit` → `writeCheckpoint()` path (`hook.go:104`) must still
  succeed with no stdin (it passes none today; auto-emit no-ops).

## Boundaries

- **NOT auto-deriving the *good* suggested-next.** `NextSuggestion` keeps its
  existing two-tier design: `PickUserSuggestion` (`user_handoff.go:163-186`) is
  the deterministic mechanical floor (top open Feature → Initiative → empty), and
  the agent's optional `hero next suggest` remains the quality ceiling. Synthesizing
  a *good* next-step needs model judgment and cannot be mechanically derived from
  the transcript — out of scope here.
- **NOT touching `SessionReflection`.** Reflections are low-frequency judgment
  calls; they stay agent-emitted via `hero next reflection`.
- **NOT changing the manual `hero next ask` command.** It remains available as an
  explicit override; auto-emit simply removes the *requirement* to run it.
- **NOT adding new stdin/transcript capability.** This is strictly a re-wire of
  `resolveSessionContext` + a `last`-variant of `firstUserAskFromTranscript`.
- **NOT summarizing or paraphrasing the user message.** Auto-emit records the
  verbatim last message (truncated at the existing cap). Summarization is a
  separate, higher-cost concern.

## Risks

- **Harness-specific stdin.** Only Claude Code's Stop hook delivers
  `session_id` + `transcript_path`. cursor/codex/generic harnesses and the git
  post-commit fallback (`hook.go:94-104`) provide nothing. The no-payload path
  **must** be a silent no-op — verified design above. Test #3 guards this.
- **Bounded, non-blocking stdin/transcript read.** A huge or blocking
  stdin/transcript must never hang the Stop hook. Reuse the existing
  64 KiB / 1000-line caps verbatim — do not introduce an unbounded read. Test #1
  (oversized) guards this. The bar is the existing
  "never fail, return minimal valid envelope" contract
  (`next_compact_handoff.go:33`).
- **Double-emit harmlessness.** If the agent already ran `hero next ask` this
  turn, auto-emit overwrites with the (same or last) message. Singleton
  supersession (`handoff.go:101-131`) makes this a clean upsert — no duplicate
  nodes. Test #4 guards this.
- **Privacy/noise.** The last user message could be a large paste or contain
  sensitive content. Truncation at `compactHandoffKickoffCap = 400` bounds size
  and matches the existing kickoff behavior; this spec does not add redaction
  (note it as a possible follow-up if the corpus federates via Cloud).
- **Last-vs-first subtlety.** The "last user message" in a Claude Code transcript
  at Stop time is the most recent human turn — which is what the next session
  most wants to resume from. Confirm against a real transcript that the final
  user record is the human's latest ask and not a tool-result echo;
  `extractTranscriptText` + the `type=="user"`/`message.role=="user"` filter
  (`next_compact_handoff.go:614-617`) already screens for genuine user records.

## Validation

- Unit + integration tests above all pass.
- Manual: in a Claude Code session, send a distinctive ask, let the turn end
  (Stop hook fires `hero next checkpoint --quiet`), then `cat .hero/next/<user>.md`
  and confirm the "Last user ask" section shows the distinctive ask with **no**
  manual `hero next ask` run.
- Manual no-op: run `hero next checkpoint --quiet < /dev/null` and confirm exit 0,
  successful checkpoint, and an unchanged `UserAsk`.
- `go test ./internal/cli/... ./internal/handoff/... ./internal/projection/...`
  green; `go build ./...` clean.

## Kickoff

Makes the user's last ask land in the handoff automatically at end-of-turn, so
`.hero/next/<user>.md` stops showing a stale ask (or the "none recorded"
placeholder) unless the agent remembers to type `hero next ask`.

**Status:** planning — spec just landed; transcript-parsing reuse confirmed
available in `next_compact_handoff.go`. No code yet.

**Pick up at:** add `lastUserAskFromTranscript` next to
`firstUserAskFromTranscript` (clone the bounded scan, return the *last* user
record instead of the first), then wire `autoEmitUserAsk(cmd.InOrStdin())` into
`runNextCheckpoint` *before* `writeCheckpoint`'s projection so the recorded ask
renders the same turn. Singleton supersession needs no handoff.go change.

→ `.hero/planning/features/next-auto-emit-user-ask/spec.md`

**Files:** `internal/cli/next_compact_handoff.go:580` (clone target),
`internal/cli/checkpoint.go:64` (wire point),
`internal/handoff/handoff.go:101` (RecordAsk, no change),
`internal/projection/user_handoff.go:64` (render target),
`internal/cli/next_compact_handoff_test.go:863` (test helpers)
**Skip:** auto-deriving a *good* NextSuggestion — needs model judgment, keep the
mechanical floor + agent ceiling. Don't touch SessionReflection. Don't change
`firstUserAskFromTranscript` — compact-handoff depends on "first".
