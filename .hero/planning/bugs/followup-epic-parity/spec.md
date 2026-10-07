---
title: "Work model treats epics inconsistently and gives containers a misleading verify state"
slug: followup-epic-parity
type: bug
status: delivering
priority: P3
severity: low
root_cause_class: code
domain: engineering
size: trivial
created: 2026-10-07
tags: [workmodel, read-contract, epic]
---

# Epic parity and container verify

## Goal

Model epics exactly like initiatives in the read contract. Give containers (initiative/epic) `verify: null`, since they are never audited themselves.

## Kickoff

`internal/workmodel` applied initiative rules (children, designed, progress, lane, revision, decision children) to `TypeInitiative` only, while the next-step engine treated epics as containers. That lets lane and next step disagree for an epic. Containers also reported verify `not_run`/`partial`, so a recently finished one produced a `weak_verify` polish entry with no action. Add `IsContainer` and use it everywhere. Test with `go test ./internal/workmodel -run EpicParity`.

## Root Cause

Type checks named `spec.TypeInitiative` directly in five places. `VerifyOf` exempted only decisions.

## Fix

`IsContainer(s)` (initiative or epic) is used in `IsWorkItem` (decision parents), `buildItem` (progress), `Designed`, `Lane` (started) and `Revision` (children). `VerifyOf` returns nil for containers. The `read-contract-v1` amendment of 2026-10-07 is additive within v1.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL model an epic like an initiative: children (declared and parent edges), designed, progress, lane, Compose/Drive next step, and decisions under it as work items.
- **AC-2:** THE SYSTEM SHALL report `verify: null` for initiatives and epics, and every polish entry SHALL carry a next step.

## Changes

1. `internal/workmodel/model.go`, `revision.go`: `IsContainer`.
2. `internal/workmodel/next_test.go`: `TestEpicParityAndContainerVerify`.
3. `.hero/specs/hero-read-contract/read-contract-v1/spec.md`: amendment.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: epic parity | DONE | `TestEpicParityAndContainerVerify`: an epic with declared and parent-edge children has progress 1/3, in_progress and Drive; its decision child is an item; a childless epic gets Compose. It fails on the old code |
| 2 | AC-2: container verify null | DONE | Same test: `big`, `init` and `doneinit` have verify null, and every polish entry has a next step. The golden schema test still passes (`verify:null` is already in v1) |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | IsContainer | DONE | — |
| 2 | test | DONE | — |
| 3 | decision amendment | DONE | Additive within v1; hero-harness notified |

### Exercise-the-feature check

- [x] Covered through `Build`/`ApplyNext`/`Polish` over a real `spec.Discover` corpus. The real repo has no epics yet, so nothing changes for them there.

### Excellence Bar self-check

- [x] Yes. One predicate replaces five scattered type checks.
