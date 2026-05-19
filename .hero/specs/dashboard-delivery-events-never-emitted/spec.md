---
title: hero spec complete never emits delivery_complete event — shipped-spec counts always understated
slug: dashboard-delivery-events-never-emitted
type: bug
status: completed
severity: high
root_cause_class: design
priority: high
tags: [dashboard, events, lifecycle, metrics, spec-lifecycle]
created: 2026-05-19
---

# hero spec complete never emits delivery_complete event — shipped-spec counts always understated

## Issue

Reporter: 277887514+chet-bellows@users.noreply.github.com — observed live in browser on 2026-05-19.

Symptoms:

- `/now` "specs shipped this week" reads **2**.
- The user reports working on 20+ specs the previous day.
- `/work` "22 specs delivering" simultaneously confirms substantial
  active work — so the workspace is not idle.
- The two numbers disagree on the same source of truth (what does
  "delivered" mean?) — and the smaller number is wrong.

Confirmed against the events log live (2026-05-19):

```bash
$ grep -ho '"type":"[^"]*"' .hero/events.log | sort | uniq -c | sort -rn
   8 "type":"peer.call.invoked"
   7 "type":"peer.call.completed"
   5 "type":"files_modified"
   5 "type":"delivery_complete"
   1 "type":"workspace.peer_id_minted"
   1 "type":"spec_created"
   1 "type":"decision_made"
```

5 `delivery_complete` events **total in the entire log** (Apr-26 →
May-19), 2 of them inside the trailing 7-day window — and the events
log goes back nearly a month. Meanwhile, browsing `.hero/specs/`
reveals dozens of completed specs that left no delivery_complete trace.

## Investigation

### What the dashboard counts

`internal/serve/pages/now/data/metrics.go:307-309`:

```go
func isDeliveryCompleteEvent(t string) bool {
    return t == "delivery_complete"
}
```

`internal/serve/pages/now/data/metrics.go:287-299`:

```go
func countCompletedSince(heroDir string, since time.Duration) int {
    if heroDir == "" {
        return 0
    }
    events := readEventsBest(heroDir, time.Now().Add(-since), 0)
    count := 0
    for _, e := range events {
        if isDeliveryCompleteEvent(e.Type) {
            count++
        }
    }
    return count
}
```

So "specs shipped this week" = number of `delivery_complete` events in
the trailing 7d. Same definition used in `pages/work/data/shipped.go:45`.

### Who actually emits `delivery_complete`

`grep -rn delivery_complete internal/cli/ cmd/` returns **zero**
producers. The only places that write the event type are:

- `internal/feed/feed.go:28` — declares it as a valid type.
- `internal/cli/event.go:27` — lists it in the CLI help text.
- `internal/serve/mcp_tools_def.go:446` — declares it as an
  acceptable event type for the `hero event` MCP tool.

So the **only** way `delivery_complete` ever lands in `events.log` is
if some external caller (an agent, a script, a human at the CLI)
manually invokes `hero event delivery_complete <message> --slug <s>`.

The five emissions in this workspace's log all came from `agent:
mcp/hero` (an agent that knows to call it) — confirmed via
`grep delivery_complete .hero/events.log`. Human-driven completions
(`hero spec complete <path>`) leave no event.

### What `hero spec complete` actually does

`internal/cli/complete.go:31-110 — runComplete`:

1. Reads the spec at the given path.
2. Validates it's a work spec.
3. Updates frontmatter `status: completed` (unless already completed).
4. Moves the spec file from `planning/` → `specs/` (via `git mv` when
   possible).
5. Re-indexes the corpus via `index.Rebuild`.
6. Optionally posts to the tracker if configured.

**Nowhere in this list is an event written.** Same for
`autoArchiveIfCompleted` at line 122 — silent move + reindex.

### What about `/deliver`?

The `/deliver` slash command produces a spec, runs the work, and
eventually flips status. The completion event is only emitted if the
delivering agent voluntarily calls `hero event delivery_complete ...`
themselves. The slash-command machinery does not enforce or auto-emit
it. The two delivery_complete events in the 7-day window came from
`mcp/hero` calling the MCP tool explicitly.

### Spec move emits the wrong event type

`internal/cli/spec_move.go:217`:

```go
_ = feed.AppendEvent(logPath, evt)
```

The event written has `Type = "spec_updated"` (or similar) — never
`delivery_complete`. So even when the file moves to `specs/` (a strong
signal the spec is complete), the dashboard's "shipped" counter
doesn't budge.

### Reproduction (no special state required)

1. In any hero workspace, run `hero spec complete .hero/planning/
   features/some-spec/spec.md`.
2. Confirm status flipped to `completed` and the file moved to
   `.hero/specs/some-spec/spec.md`.
3. `grep delivery_complete .hero/events.log` — the new completion is
   absent.
4. Reload `/now` — "specs shipped this week" did not increment.

### Root cause

`delivery_complete` is the dashboard's sole signal for "spec was
shipped", but no part of the spec-completion lifecycle emits it.
Emission is opt-in via the `hero event` CLI (or the MCP tool), and
most agents / humans don't know to call it. Result: the shipped count
chronically reads near-zero regardless of actual throughput.

This is a **design** classification — the contract between writers and
readers was never closed. The spec-status `completed` transition is
the canonical signal of "shipped"; the dashboard reads a separate
event stream that nobody is required to write to.

### Severity

**High.** The two most important dashboard metrics — "specs shipped
this week" and the My-week / Hero-ROI tabs derived from the same
count — are silently broken on every workspace. Users have no way to
notice until they manually compare to git history or the specs/
directory.

Caused by our codebase. Workaround: emit `hero event delivery_complete
...` manually after every `hero spec complete` (nobody does).

## Code Flow (End to End)

The producer side that *should* emit but doesn't:

1. `internal/cli/complete.go:31-110 — runComplete` — flips status,
   moves file, re-indexes, syncs tracker. **No event.**
2. `internal/cli/complete.go:122-143 — autoArchiveIfCompleted` — same
   path via /deliver completion hook. **No event.**
3. `internal/cli/spec_move.go:200-220` — emits `spec_updated`/
   `files_modified` (not `delivery_complete`) when files move.

The consumer side that reads it:

4. `internal/serve/pages/now/data/metrics.go:280-299` — counts
   `delivery_complete` events in the trailing 7d → "specs shipped this
   week".
5. `internal/serve/pages/now/data/metrics.go:113-146 — myWeekTiles` —
   same count drives the My-week tab.
6. `internal/serve/pages/work/data/shipped.go:21-45` — recently-shipped
   timeline pulls the same event type.
7. `internal/serve/metrics/metrics.go:81-153` — `SpecsDelivered` in the
   serve metrics package also reads `delivery_complete`.

The Work page's "22 specs delivering" reads spec frontmatter directly
(`internal/serve/pages/work/data/counts.go:31-32`), which is why it
doesn't have the same blind spot — it counts `status: delivering` /
`status: in-review` files. But "shipped" relies on the event log.

## Key Files

### Reader side (consumes delivery_complete)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/pages/now/data/metrics.go` | 75–146, 280–325 | "specs shipped" tile + My-week sparkline |
| `internal/serve/pages/work/data/shipped.go` | 21–60 | recently-shipped timeline |
| `internal/serve/metrics/metrics.go` | 81–153 | metrics package SpecsDelivered counter |
| `internal/serve/pages/now/data/agents.go` | 51, 201–206 | session count uses delivery_complete |
| `internal/serve/pages/now/data/changes.go` | 159, 252, 279 | feed icon kind + dedup |

### Producer side (should emit, doesn't)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/complete.go` | 31–143 | spec complete + auto-archive — no event emission |
| `internal/cli/spec_move.go` | 200–220 | spec move — emits wrong type |
| `internal/cli/verify.go` | 172 | auto-archive on `hero verify` — same gap |

### Event substrate
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/feed/feed.go` | 28, 221 | declares delivery_complete as a valid type |
| `internal/cli/event.go` | 27, 43–80 | the CLI command that emits |
| `internal/serve/mcp_tools_def.go` | 446 | MCP tool surface |

## Secondary Defects

1. **`hero spec complete` doesn't emit `spec.status_changed` either.**
   Even if delivery_complete is a higher-level signal, status
   transitions are normally announced via `spec.status_changed` (the
   change-feed dedup table at `changes.go:151` knows about it). Both
   are missing from the completion path.

2. **`internal/cli/verify.go:172`'s auto-archive runs silently** when a
   spec reaches `status: completed` through any path. Same gap as
   `runComplete`.

3. **`isDeliveryEvent` at `metrics.go:318-325`** includes
   `agent_session_started` and `agent_session_ended`, but those are
   never emitted in this workspace either (zero in the log). The "Hero
   active sessions" tile at `agents.go:51` over-relies on the same
   under-emitted event family.

4. **MCP tool description in `mcp_tools_def.go:446`** doesn't tell the
   model when to emit `delivery_complete` vs. when not to. An LLM
   reading the tool spec will emit the event inconsistently.

5. **Sparkline in `metrics.go:91`** plots a hardcoded fake history
   (`[1,0,1,2,0,1,shipped]`) — even if the count is fixed, the spark
   is theatrical. Out of scope for this spec.

## Notes

- The user spec corpus already contains `spec-status-integrity`
  ("Graph-Verified Delivery Claims") — that initiative is the natural
  home for closing this gap. This bug spec narrowly targets the
  shipped-count regression rather than rebuilding the lifecycle.
- An alternative design would have the dashboard compute "shipped this
  week" from spec frontmatter (`status: completed` + git log of the
  spec file's mtime in the trailing window) and ignore the event log
  entirely. That removes the dependency on a publisher altogether but
  is more invasive. Stated as a fix-direction option below.

## Acceptance Criteria

- WHEN `hero spec complete` flips a spec's status to `completed` THE
  SYSTEM SHALL append a `delivery_complete` event to `.hero/events.log`
  with `slug=<spec slug>`, `agent=human/<gitUserName>` (or the agent
  identity from `$HERO_AGENT`), and a message summarizing the
  completion.
- WHEN `autoArchiveIfCompleted` runs as part of `/deliver` or
  `hero verify` THE SYSTEM SHALL emit the same `delivery_complete`
  event (idempotent — never duplicates on a re-run for the same spec).
- WHEN a spec is moved from `planning/` → `specs/` by any code path
  THE SYSTEM SHALL also emit a `spec.status_changed` event with the
  new status, so the changes feed picks it up.
- IF the spec is already `completed` AND `events.log` already contains
  a `delivery_complete` event for that slug within the last 24h THEN
  THE SYSTEM SHALL skip emission to avoid duplicates.
- WHERE the dashboard is reading "specs shipped this week" THE SYSTEM
  SHALL fall back to counting `specs/` directory files whose
  frontmatter `status: completed` AND whose file mtime is within the
  trailing 7d window WHEN the event log is empty (so old workspaces
  with no historical events still surface a non-zero count after the
  fix lands).

## Goal

Every spec that transitions to `completed` leaves a single
`delivery_complete` event in `events.log`, regardless of which command
or agent triggered the transition. The dashboard's shipped-spec
counts reflect actual workspace throughput on every page render.

## Boundaries

- Not in scope: changing the metrics-tab UI, sparkline rendering, or
  the My-week / ROI tiles' computation. Once the producer is fixed,
  the consumers light up.
- Not in scope: rebuilding `agent_session_started/ended` emission for
  the agent-session counts — that's its own spec under
  `agent-outposts` / `hero-team-server`.
- Not in scope: cross-workspace event aggregation (cloud / team
  features) — solo-mode is the target here.

## Risks

- Idempotency matters: a re-run of `hero spec complete` on an
  already-completed spec must not double-count. The proposed dedup
  window (24h, same slug) needs a quick test pass.
- The `human/gitUserName` agent string changes audit-trail attribution
  for completions. Confirm with the `spec-status-integrity` design
  author that this is the intended identity (vs. the OS user or the
  active session ID).
- Adding an event emission inside `runComplete` means a re-index that
  was previously cheap now also writes to the log. Re-verify
  performance for workspaces with very large event logs.
- The dashboard's fallback to "scan `specs/` mtime" must not double-
  count when both the event AND the file exist; document the precedence
  rule (event log wins).

## Validation

1. Fresh workspace:
   - Create a feature spec under `planning/`.
   - Run `hero spec complete .hero/planning/features/.../spec.md`.
   - Confirm `events.log` gains a `delivery_complete` entry for that
     slug.
   - Reload `/now` — "specs shipped this week" increments by 1.
2. Idempotency:
   - Re-run `hero spec complete` on the same spec.
   - Confirm no duplicate entry is written (or a `spec_already_
     completed` event is written instead).
3. `/deliver` integration:
   - Run the full `/deliver` flow on a small spec.
   - Confirm `autoArchiveIfCompleted` emits exactly one
     `delivery_complete` event.
4. Mtime-fallback path:
   - In a workspace with no `delivery_complete` events but several
     specs under `specs/` with recent mtime, confirm "specs shipped
     this week" returns the file-count fallback.
5. Regression tests:
   - Cover `runComplete` happy path emits the event.
   - Cover idempotency within 24h.
   - Cover the dashboard tile returns non-zero when only file-mtime
     fallback is available.

## Changes

- `internal/cli/complete.go` — new `emitCompletionEvents` helper wired into `runComplete` (all three branches: nothing-to-do backfill, status-flipped move, and standard happy path) and `autoArchiveIfCompleted` (both the already-in-specs and freshly-moved branches). 24h idempotency via `hasRecentDeliveryComplete`. Emits both `delivery_complete` and a companion `spec.status_changed` event.
- `internal/cli/complete_test.go` — regression tests for: event emission on first complete, idempotency on re-run, backfill on already-complete spec, HERO_AGENT env propagation.
- `internal/feed/feed.go` — added `spec.status_changed` to `ValidTypes` and a `shortType` label for it. Consumer code already understood the type; the gate was only blocking ad-hoc writes.
- `internal/serve/pages/now/data/metrics.go` — `countCompletedSince` falls back to `countCompletedSpecsByMtime` (specs/ files with `status: completed` and recent file mtime) when the event log returns zero. Event log takes precedence.
- `internal/serve/pages/now/data/metrics_test.go` — tests for both fallback path (no events log) and event-log precedence (events present, files ignored).

## Recap

The Now dashboard's "specs shipped" tile (and several siblings) counts
`delivery_complete` events, but **nothing in the codebase emits that
event during a normal spec completion** — not `hero spec complete`,
not `autoArchiveIfCompleted`, not the `/deliver` flow. The signal is
opt-in via a CLI command that the user never knows to call. Wire
emission into the canonical completion path (with idempotent dedup)
and add a file-mtime fallback for historic workspaces.
