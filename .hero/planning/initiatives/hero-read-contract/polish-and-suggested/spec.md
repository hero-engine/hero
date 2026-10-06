---
title: "Polish and suggested lists in hero_work"
slug: polish-and-suggested
type: feature
status: delivering
priority: medium
domain: engineering
size: small
created: 2026-10-06
parent: hero-read-contract
depends-on: [hero-work-tool]
---

# Polish and suggested

## Goal

Give clients Hero's view of loose ends on recent work and of what to do next, with every entry carrying a next step.

## Kickoff

Implement `workmodel.Polish` and `workmodel.Suggested` per `read-contract-v1`, and fill `hero_work.polish` / `.suggested`. Test with `go test ./internal/workmodel -run 'Polish|Suggested' ./internal/serve -run ToolWork`.

## Design

**Polish** (recently_done items only, in item path order):
- `weak_verify`: verify is not passed. Next is the item's own (Verify).
- `open_followups`: unfinished non-bug specs related in either direction, created on or after the item's completion day. Next is the first follow-up's.
- `bug_against_recent`: unfinished related bugs. Next is the first bug's.
- `rated_worse` is reserved.

**Suggested**:
- Up to 3 `queue` picks: ready items by priority (then path), then undesigned stubs whose next is Design or Diagnose.
- Reasons follow the next step ("ready to fix, critical priority", "needs Design").
- When fewer than 3 items are ready or in progress, an Explore item (`/discover`, slug null) is appended.

`PolishItem` / `SuggestedItem` live in `workmodel`; `serve` aliases them.

## Acceptance Criteria

- **AC-1:** WHEN `hero_work` runs THE SYSTEM SHALL list polish only for recently_done items, with kinds weak_verify, open_followups (created on or after completion) and bug_against_recent, each with a reason and a slash-command next step, and never `rated_worse` in v1.
- **AC-2:** THE SYSTEM SHALL suggest at most three queue picks (ready by priority, then undesigned stubs), with reasons that match each pick's next step.
- **AC-3:** WHILE fewer than three items are ready or in progress THE SYSTEM SHALL end `suggested` with an Explore item (`slug: null`, `/discover`), and SHALL omit it otherwise.

## Changes

1. `internal/workmodel/polish.go` + `polish_test.go`.
2. `internal/serve/mcp_tools_read_contract.go`: aliases; `toolWork` fills both lists; `TestToolWorkShape` updated.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: polish kinds | DONE | `TestPolishKinds`: weak_verify → `/verify recent`; bug → `/diagnose recentbug`; follow-up created on the completion day counted; an older related spec excluded; no rated_worse; only recent items. Real repo: 1 weak_verify (`real-exercise.log`) |
| 2 | AC-2: picks | DONE | `TestSuggested`: ready first, at most 3, slash next steps, reason derived from the next step. Real repo: "ready to fix / drive / deliver", by priority |
| 3 | AC-3: Explore when thin | DONE | `TestSuggested`: Explore exactly when thin, and absent once the backlog is padded past the threshold; `TestToolWorkShape` asserts it through the tool |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | polish.go + tests | DONE | Found and fixed in the real exercise: the suggested reason said "deliver" for an initiative whose step is Drive |
| 2 | tool wiring | DONE | — |

### Exercise-the-feature check

- [x] Real `hero mcp` `hero_work` on this repo (`real-exercise.log`).

### Excellence Bar self-check

- [x] Yes. Deterministic rules with reasons that match the action.
