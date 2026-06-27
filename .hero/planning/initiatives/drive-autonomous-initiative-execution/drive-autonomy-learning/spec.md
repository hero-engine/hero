---
title: "Rubber-stamp learning — promote always-approved pauses to auto-proceed"
slug: drive-autonomy-learning
type: feature
status: planning
priority: medium
horizon: next
tags: [drive, learning, autonomy, feedback, events, floor-raising]
created: 2026-06-27
relations:
  - target: drive-autonomous-initiative-execution
    kind: parent
  - target: drive-pause-resume
    kind: depends-on
---

# Rubber-stamp learning — promote always-approved pauses to auto-proceed

## Goal

Make the autonomy boundary self-tuning: when a user approves a given
pause-category unchanged enough times in a row, **promote that category to
auto-proceed** for that user/initiative — and the moment the user edits or
redirects at a pause, **keep stopping there**. This is the piece that
actually drains the friction the user named ("mostly just say yes, do the
next"), and it is the mission-fit core: it captures one person's judgment
and raises the floor so the next session — or a junior — inherits it.

## Kickoff

Record every pause → outcome as an event (reuse the existing event/feed
infra, [internal/serve/team_coordination.go](../../../../../internal/serve/team_coordination.go)
and `hero_event`): `{user, initiative, category, outcome: approved-unchanged
| edited | redirected}`. Maintain a per-(user, category[, initiative])
promotion state: K consecutive `approved-unchanged` → `promoted`; any
`edited`/`redirected` → demote and reset. `NeedsMe`
([needs-me-predicate](../needs-me-predicate/spec.md)) consults promotion
state in `autonomous` mode only, and **never** for hard-pause guardrails
(irreversible, hard cap). Surface promotions so they're visible and
reversible. Start with K conservative (e.g. 3) and config-tunable.

## Problem

A static `needs_me()` is either too chatty (you still rubber-stamp) or too
bold (it guesses wrong). The signal to fix this is already free: every "yep,
next" the user types is a labeled training example. Nothing captures or acts
on it today, so the boundary can't improve and the user keeps approving the
same categories forever.

## Design

### Outcome capture

At each pause resolution (from [drive-pause-resume](../drive-pause-resume/spec.md)),
classify the human's response:

- `approved-unchanged` — proceeded as recommended, no edits.
- `edited` — changed the spec/plan/decision before continuing.
- `redirected` — chose a different option / stopped the run.

Emit as an event so the history is auditable and travels with the corpus.

### Promotion state

Per `(user, category)` (optionally scoped to an initiative or initiative
*kind*):

```
streak++ on approved-unchanged
if streak >= K  -> state = promoted
on edited|redirected -> state = demoted, streak = 0
```

Stored alongside other learned state; queryable and resettable
(`hero drive trust --list` / `--reset <category>` or equivalent).

### How `NeedsMe` uses it

Only in `autonomous` mode, `NeedsMe` treats a `promoted` category as
proceed-eligible *for that user*. Guardrails are exempt: `Irreversible` and
the hard cap are never promotable. Promotions are per-user (it's *their*
judgment), and shareable later via the corpus so a teammate or junior
inherits the senior's "this is fine" patterns — the floor-raising payoff.

### Visibility & control

- A promotion emits a one-line notice the first time it suppresses a pause
  ("auto-proceeding `Underspecified` for you — approved 3× running; reply
  'ask me again' to undo").
- `--dry-run` on `hero goal` shows which upcoming transitions are
  auto-proceeding due to promotions vs. base policy.

## Acceptance Criteria

- WHEN a user approves a pause-category unchanged K consecutive times, THE
  SYSTEM SHALL promote that category to auto-proceed for that user.
- WHEN a user edits or redirects at a previously-promoted category, THE
  SYSTEM SHALL demote it and resume pausing there.
- THE SYSTEM SHALL NOT promote `Irreversible` pauses or the hard-cap pause
  under any circumstances.
- WHILE `autonomy` is `supervised` or `guided`, THE SYSTEM SHALL ignore
  promotions (they apply only in `autonomous`).
- THE SYSTEM SHALL make promotions inspectable and individually resettable.
- THE SYSTEM SHALL record each pause outcome as an event in the corpus.

## Test Plan

- Unit: streak state machine (promote at K, demote on edit/redirect, reset).
- Unit: `NeedsMe` honors promotion only in `autonomous`, never for
  guardrails.
- Integration: simulate K approvals → next same-category transition
  auto-proceeds; then an edit → it pauses again.
- Visibility: promotion notice fires once on first suppression; dry-run
  lists promoted transitions.

## Risks

- **Over-promotion erodes safety** — a user who reflexively approves trains
  the system to stop asking about things that mattered. Mitigation:
  guardrails are never promotable; promotions are per-category (not global);
  K is conservative; demotion is instant on any edit.
- **Cold-start sparsity** — little history early. Mitigation: until promoted,
  behavior is just the base policy; no regression.
- **Cross-initiative leakage** — a promotion learned on trivial work
  applied to risky work. Mitigation: scope promotions by initiative kind/
  size where signal allows; default to per-initiative until confidence.
