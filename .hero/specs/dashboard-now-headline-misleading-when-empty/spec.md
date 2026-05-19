---
title: Now headline reads "no agent running · since 19h ago" — composes two empty signals into a false story
slug: dashboard-now-headline-misleading-when-empty
type: bug
status: completed
severity: medium
root_cause_class: design
priority: medium
tags: [dashboard, ux, now-page, empty-state, headline]
created: 2026-05-19
---

# Now headline reads "no agent running · since 19h ago" — composes two empty signals into a false story

## Issue

Reporter: 277887514+chet-bellows@users.noreply.github.com — observed live in browser on 2026-05-19.

Symptom: the `/now` page hero subhead reads **"no agent running ·
since 19h ago"**. Read in isolation, that strongly implies "this
workspace has had no agent activity for 19 hours" — which the user
correctly identifies as false (they were doing heavy work that day).

The two halves are independently honest signals but they were never
meant to share a sentence:

- **"no agent running"** — there is no live agent session right now.
  Correct given the live-session ledger is empty.
- **"since 19h ago"** — the most recent event in `.hero/events.log` was
  19h ago. Correct given the workspace doesn't emit `delivery_complete`
  on normal flows (see `dashboard-delivery-events-never-emitted`),
  so the event log is sparse and stale.

Joining them with " · " makes the page lead with a sentence whose
implied meaning is wrong — and it's the **page-hero subhead**, the
most prominent text on the dashboard's landing screen.

## Investigation

### How the subhead is composed

`internal/serve/pages/now/page.go:502-525 — buildPageHero`:

```go
parts := []string{}
switch inboxCount {
case 0:
    // Skip — empty inbox tells its own story in the section below.
case 1:
    parts = append(parts, "<strong>1 needs your input</strong>")
default:
    parts = append(parts, fmt.Sprintf("<strong>%d need your input</strong>", inboxCount))
}
switch runningCount {
case 0:
    parts = append(parts, "<strong>no agent running</strong>")
case 1:
    parts = append(parts, "<strong>1 agent running</strong>")
default:
    parts = append(parts, fmt.Sprintf("<strong>%d agents running</strong>", runningCount))
}
if lastActive != "" {
    parts = append(parts, "since "+template.HTMLEscapeString(lastActive))
}
subhead := strings.Join(parts, `<span class="dot-sep">·</span>`)
```

`lastActive` comes from `agents.LastActivePretty`, populated at
`internal/serve/pages/now/data/agents.go:89-91`:

```go
if len(events) > 0 {
    out.LastActivePretty = prettyAgeChip(events[0].Timestamp)
}
```

`events` is the 24h scrape of `events.log` (line 47). So
`lastActive` reflects "when was the last event emitted, ever, in
this workspace" — not "when did an agent last run".

`prettyAgeChip` at `agents.go:219-227`:

```go
if d := time.Since(t); d < 24*time.Hour {
    return prettyAge(t)            // "19h ago"
}
return fmt.Sprintf("%s %s", t.Format("Mon"), t.Format("3:04pm"))
```

When the freshest event is between 1m and ~24h old, we return a
relative chip like `"19h ago"`.

### Why "since 19h ago" is meaningless next to "no agent running"

The subhead semantics today are:

> "Right now: N proposals; X agents running; **and the last event in
> the log was Y ago**."

The intended reading (per the spec author's `case 0: skip` comment for
inbox) is: each chip is a *fact* and `·` separates them. But:

- When `runningCount > 0`, "since Y ago" reads as "the running session
  started Y ago" — that's the running session's age. Fine.
- When `runningCount == 0`, "since Y ago" reads as "since the last
  agent run", because "no agent running · since Y ago" is a single
  English clause where "since" naturally binds to the preceding
  predicate. That makes the dashboard headline assert that nothing
  has happened for Y time — when in fact Y is only the time since
  the last *emitted* event, and a workspace that emits sparsely
  (see the delivery-event gap) routinely has Y > 12h while plenty
  of work happened.

This is a composition bug, not a data bug. Each chip is fine alone;
the join produces a false implication.

### `subheadPlainText` has the same shape

`internal/serve/pages/now/page.go:424-446 — subheadPlainText` (used
on the `event: hero` SSE channel) mirrors the same join. So even if
the subhead is updated live, the false-implication persists.

### Reproduction (no special state required)

1. Open a workspace whose `events.log` has a few entries older than
   1h and no live agent session (e.g. this workspace at the time of
   the report).
2. `hero serve` → `/now`.
3. Observe the page-hero subhead: `no agent running · since <Nh> ago`.
4. Note that the inbox section below renders "Nothing waiting on you"
   (also empty). So the headline + body together imply a quiet
   workspace, when in reality the user just shipped many specs.

### Root cause

`buildPageHero` joins independent signals with `·` and includes the
"since X ago" clause whenever `lastActive` is non-empty, regardless
of whether the preceding chip ("no agent running") leaves the reader
to bind "since" against the wrong predicate. The subhead's English
grammar is doing work the data model didn't account for.

### Severity

**Medium.** Cosmetic but conspicuous — it's the dashboard's largest
text. Doesn't cost data, but actively misleads the user about
workspace state at first glance. Workaround: ignore the headline and
trust the metric strip below. (Which has its own data bugs — see the
sibling specs.)

Caused by our codebase.

## Code Flow (End to End)

1. `internal/serve/pages/now/page.go:240` — `hero := buildPageHero(...,
   agents.LastActivePretty)`.
2. `internal/serve/pages/now/data/agents.go:47, 89-91` —
   `LastActivePretty` set from the most recent event in `events.log`
   (24h window).
3. `internal/serve/pages/now/page.go:502-525` — chips composed and
   joined with `<span class="dot-sep">·</span>`.
4. Rendered into the page-hero fragment.

The `event: hero` SSE refresh uses `subheadPlainText` at line 424 —
same composition logic.

## Key Files

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/pages/now/page.go` | 424–446 | subheadPlainText (SSE channel) |
| `internal/serve/pages/now/page.go` | 502–525 | buildPageHero (initial render) |
| `internal/serve/pages/now/data/agents.go` | 47–91, 219–227 | LastActivePretty computation |

## Secondary Defects

1. **`LastActivePretty` claims to mean "last agent activity"** (per
   the comment at `agents.go:105-107`) but actually means "last
   event of any type in the 24h window". Outbound peer calls, spec
   updates, decision-made events all count. The name implies a
   richer filter than the code applies.

2. **When `events.log` is completely silent in the 24h window**,
   `LastActivePretty` is empty and the subhead becomes
   `"no agent running"` standalone — fine. The problematic case is
   the partial-silence band where one stale event is still inside
   the 24h window.

3. **No empty-inbox chip** — the `case 0: skip` comment notes "empty
   inbox tells its own story in the section below". That's a fair
   design choice, but it means a workspace with 0 inbox + 0 running
   + 1 stale event gets a one-chip-plus-tail subhead like
   `"no agent running · since 19h ago"` — there is no contextual
   anchor to bind "since" to anything other than the empty agent
   state. Adding even one positive chip (e.g. inbox count, or a
   "last shipped Xd ago" chip from a different source) would
   disambiguate.

## Notes

- The user's deeper concern is "the headline is doing too much work
  with too little data". Fixing the data side (events emitted on
  real lifecycle events — see `dashboard-delivery-events-never-
  emitted`) makes `LastActivePretty` more useful and reduces the
  frequency this composition shows up. But the composition bug is
  independent: even when events ARE flowing, the same "no agent
  running · since X ago" can render on a quiet weekend morning and
  imply something stronger than it should.

- A simpler subhead that just lists what's happening NOW
  ("N proposals · X agents running") and reserves "since Y ago" for
  the running-agent case would avoid this entire failure mode.

## Acceptance Criteria

- WHEN `runningCount == 0` THE SYSTEM SHALL NOT append the
  "since X ago" clause to the page-hero subhead.
- WHEN `runningCount > 0` THE SYSTEM SHALL render "since X ago" only
  if X is the timestamp of the actual running session (not the most
  recent log event).
- IF the inbox is empty AND no agents are running AND the events log
  is silent in the last 24h THEN THE SYSTEM SHALL render a single
  truthful subhead like "no live activity right now" rather than
  composing two empty signals.
- WHEN computing `LastActivePretty` for a running session THE SYSTEM
  SHALL pull from the session's `LastActiveAt` (the live ledger) and
  not from `events.log` at all.
- THE SYSTEM SHALL apply the same composition logic in both
  `buildPageHero` (initial render) and `subheadPlainText` (SSE
  channel) so the live update doesn't reintroduce the misleading
  string.

## Goal

The Now page's headline tells a true story regardless of how sparse
the event log is. "Since X ago" never modifies "no agent running".

## Boundaries

- Not in scope: changing what `events.log` records (covered by
  `dashboard-delivery-events-never-emitted`).
- Not in scope: changing the inbox / metric-strip / sections below
  the hero.
- Not in scope: redesigning the page-hero across all home pages —
  this is a Now-page-specific fix.

## Risks

- Tests pinning the subhead string need to be updated; check
  `internal/serve/pages/now/page_test.go` for sensitivity.
- The `event: hero` SSE channel publishes the plain-text string; any
  client that string-matches it must tolerate the new wording.
- Removing "since" entirely when no agent is running loses a small
  amount of "is anything happening here?" affordance — make sure
  the Since-you-were-here feed section still surfaces last-event
  information for users who need it.

## Validation

1. Workspace with running agent (live session present):
   - Subhead reads "1 agent running · since <session start>".
2. Workspace with no agent, fresh events:
   - Subhead reads "no agent running" with no trailing clause.
3. Workspace with no agent, stale events (e.g. 19h ago):
   - Subhead reads "no agent running" — same as fresh.
4. Workspace with inbox + no agent:
   - Subhead reads "3 need your input · no agent running".
5. Regression tests:
   - Pin the new behaviour with a table-driven test covering all
     combinations of (inboxCount, runningCount, lastActive).
   - Update `subheadPlainText` tests to match.

## Changes

- `internal/serve/pages/now/page.go` — `buildPageHero` and `subheadPlainText` now share the same composition rule: the "since X ago" tail renders only when `runningCount > 0`. When inbox and running are both zero the subhead collapses to a single truthful "no live activity right now" instead of joining two empty signals.
- `internal/serve/pages/now/page_test.go` — table-driven `TestSubheadPlainText_Cases` covers every combination of inbox / running / lastActive, plus `TestBuildPageHero_NoMisleadingSinceClause` pins the HTML render to the same rule.

## Scope notes

- AC 4 (running-agent age from the live ledger, not events.log) is satisfied by the existing `agents.go:105-106` logic that already prefers `LastActiveAt` from the running session. Since the new composition only renders "since X ago" when a session is running, that source is the one in play — no separate code change required.
- The headline now does *less work* with the data it has: when the events log is sparse (the underlying cause flagged by `dashboard-delivery-events-never-emitted`, fixed in item 2 of this sprint), the subhead no longer compounds the underemission into a false-quiet claim.

## Recap

The Now page-hero subhead joins inbox / running / since-X-ago chips
with `·`. When no agent is running, the trailing "since X ago" binds
to "no agent running" in English and asserts a quiet workspace that
may be anything but. Drop the trailing clause unless an agent is
actually running, and source running-agent ages from the live ledger
rather than the (often stale) event log.
