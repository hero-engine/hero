---
title: "Next-step engine — one deterministic primary action per work item"
slug: next-step-engine
type: feature
status: delivering
priority: high
domain: engineering
size: small
created: 2026-10-06
parent: hero-read-contract
depends-on: [work-item-model]
---

# Next-step engine

## Goal

Compute each `WorkItem`'s `next`: one primary action, a chat-sendable slash command, a phase, enabled/reason and extras. It follows `read-contract-v1`'s table, keeping the requester's rules. Ship the new `/verify` workflow it points to for every harness.

## Kickoff

Implement `workmodel.NextFor` / `ApplyNext` from `read-contract-v1`'s next-step table, and add `core/commands/verify.md`. Hero's install pipeline renders it for all 8 harness targets. Test with `go test ./internal/workmodel ./internal/install ./internal/cli`.

## Design

- `internal/workmodel/next.go`: a pure function of the item, its spec and the corpus. The order is: closed types, then finished (Verify only within `recently_done` and not passed), then regressed, handed-off, delivering, and finally per type: initiative/epic (Compose or Drive), decision (Decide), bug (Diagnose or Fix), feature (Design or Deliver). Deliver steps are disabled with "waits on <slugs>" while unmet dependencies remain. Extras never include the other half of Diagnose⇄Deliver.
- `core/commands/verify.md`: run the cold audit when it is missing, stale or for another spec (independent reviewer only), then `hero spec verify <slug>`. Never `--force` unasked. It is listed in engineering routing (natural-language row plus the both-surfaces table).

## Acceptance Criteria

- **AC-1:** WHEN next steps are computed THE SYSTEM SHALL produce the `read-contract-v1` table's action, label, command, phase and enabled/reason for every row, including null for closed work, verified work, and work finished before `recently_done`.
- **AC-2:** WHILE a deliver step has unmet `depends-on`/`blocks` targets THE SYSTEM SHALL disable it with reason `waits on <slugs>` and phase state `waiting`.
- **AC-3:** THE SYSTEM SHALL never offer Diagnose and Deliver together, never offer Deliver for finished work, only show Delivered via a passed verify, use only slash commands, and serialize empty extras as `[]`.
- **AC-4:** THE SYSTEM SHALL ship a `/verify` workflow in core commands, so every install target renders it and engineering routing lists it.

## Changes

1. `internal/workmodel/next.go` with `next_test.go`.
2. `core/commands/verify.md`.
3. `domains/engineering/routing.md` and the regenerated `domains/engineering/AGENTS.md`; prompt baselines regenerated (one more command file installed).

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: every table row, including nulls | DONE | `TestNextStepTable`: 17 cases covering stub, ready, blocked, delivering, HOLD attention, undiagnosed/diagnosed/blocked bug, regressed, handed-off, initiative with and without children, decision, recent unverified (Verify), old unverified (null), verified (null) and superseded (null) |
| 2 | AC-2: disabled with "waits on" | DONE | The `blocked` and `blockedbug` cases assert `enabled=false`, the reason, and `waiting` |
| 3 | AC-3: invariants | DONE | `TestNextStepInvariants` runs over every seeded item |
| 4 | AC-4: /verify for every target | DONE | `core/commands/verify.md`. The canonical routing-reference test resolves `/verify` against real surfaces. The roster test (`TestEngineeringAgentsMdRosterComplete`) and the all-target install, contract and routing matrices pass. Prompt baselines regenerated |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | next.go + tests | DONE | Real-corpus probe: 513 items with build and next steps in ~20 ms |
| 2 | verify.md | DONE | Independent-reviewer gate and no unasked `--force` |
| 3 | routing, pack AGENTS.md, baselines | DONE | Pack regenerated via `HERO_REGEN_PACK_AGENTS=1`; baselines via `-update-baseline` |

### Exercise-the-feature check

- [x] Real-corpus probe over this repo's 513 specs (`ApplyNext`): actions are deliver 19, design 72, diagnose 2, drive 10, verify (recent only) and null for the rest, all well-formed. The tools that surface them come next.

### Excellence Bar self-check

- [x] Yes. One table-driven pure function, with the requester's invariants tested over every item.
