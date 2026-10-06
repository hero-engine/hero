---
title: "Work-item model — one WorkItem for every read tool"
slug: work-item-model
type: feature
status: delivering
priority: critical
domain: engineering
size: medium
created: 2026-10-06
parent: hero-read-contract
depends-on: [read-contract-v1]
delivery_method: manual
---

# Work-item model

## Goal

A single deterministic Go model, `internal/workmodel`, that turns the spec corpus into the read contract's `WorkItem`s. `hero_work`, `hero_spec` and the next-step engine all render it, so there is exactly one interpretation of lanes, verify state and revisions.

## Kickoff

Implement `internal/workmodel`. `Build(specs, Options{Now, RecentDays})` returns `[]Item`, with JSON matching `WorkItem` in `read-contract-v1` (`next` is left nil for `next-step-engine`). It derives:
- the work-type filter;
- normalized priority and severity;
- lane, designed, unmet dependencies, initiative progress and verify state;
- the per-item revision.

All of it uses `internal/spec` (sections, relations, `IsFinished`, `DeclaredChildren`, `ParseLedger`, `FindAuditReport`) exactly as `read-contract-v1` defines. Test with `go test ./internal/workmodel`.

## Design

- `type Item` mirrors `WorkItem`: `slug, title, revision, type, status, priority, severity, size, path (repo-relative), parent, progress {done,total}|null, lane, created_at, updated_at, completed_at|null, tracker {id,url}|null, verify {state,audit}|null, next *NextStep` (the `NextStep` struct is defined here; the engine fills it).
- **Work types:** feature, bug, enhancement, initiative, epic, plus decisions whose parent is an initiative.
- **Normalization:**
  - priority and severity via `read-contract-v1`'s synonym table (e.g. `P1`/`major`→high, `moderate`→medium); anything else null.
  - `updated_at` is the file mtime; `completed_at` is `CompletedAt`, else null.
  - `tracker` is `{id, url:null}` when a `TrackerID` exists.
- **Designed, unmet dependencies and lanes:** exactly the `read-contract-v1` rules. Relation kinds `depends-on`, raw `depends_on` (from `relations:` blocks) and `blocks` count as dependency edges. A missing target counts as unmet.
- **Verify state:** per `read-contract-v1`, from `FindAuditReport` (slug-validated; staleness ignored for finished specs, since verify rewrites them after the audit), `ParseLedger`, and finished status. It is `null` for decisions.
- **Revision:** 16 hex characters of SHA-256 over the raw file bytes, the sorted `slug:status` of every related or declared-child spec, the audit verdict plus report mtime, and the derived lane and verify state.
- **Helpers exported for the engine:** `Designed`, `UnmetDeps`, plus an index lookup.
- Deterministic output order: `path` ascending.

## Acceptance Criteria

- **AC-1:** WHEN `Build` runs over a corpus THE SYSTEM SHALL return one item per work spec (feature, bug, enhancement, initiative, epic, and initiative-child decision) and none for knowledge types, ordered by path.
- **AC-2:** THE SYSTEM SHALL assign each item exactly one lane by the `read-contract-v1` first-match rules (recently_done, in_progress, ready, designed, none), honoring `RecentDays` and unmet `depends-on`/`blocks` edges.
- **AC-3:** THE SYSTEM SHALL derive `verify` as passed, partial, failed or not_run with the audit verdict from the spec's own validated report, and null for decisions.
- **AC-4:** THE SYSTEM SHALL normalize priority and severity to critical, high, medium, low or null, and report initiative `progress` as finished declared children over declared children.
- **AC-5:** WHEN a spec's file, a related spec's status, or its audit report changes THE SYSTEM SHALL change that item's `revision`; otherwise the revision SHALL stay identical across builds.

## Changes

1. `internal/workmodel/model.go`: types, `Build`, normalization, lanes, verify, progress.
2. `internal/workmodel/revision.go`: item revision.
3. `internal/workmodel/model_test.go`: table tests over a temp corpus.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: one item per work spec, knowledge excluded, path order | DONE | `IsWorkItem` and `Build` in `model.go`. `TestBuildIncludesOnlyWorkSpecsInPathOrder`: 11 work specs including an initiative-child decision; a convention and a parentless decision are excluded; path order; decoded title; repo-relative path |
| 2 | AC-2: one lane per item by first-match rules, with RecentDays and unmet deps | DONE | `Lane`, `UnmetDeps`, `Designed`. `TestLanes` covers all five lanes. `TestRecentDaysWindow`. Audit round 1: `TestEdgeCasesFromAudit` covers raw `depends_on`, `blocks`, a missing target, the CompletedAt→mtime fallback, and a childless initiative |
| 3 | AC-3: verify state and audit from the validated report; null for decisions | DONE | `VerifyOf`. Round 2: signed-off SKIPPED/BLOCKED rows count as passed, matching `hero spec verify` Gate 1. `TestVerifyState` covers passed, partial, failed, not_run, and a null decision. Audit round 1: a finished spec newer than its SHIP audit still passes (`TestEdgeCasesFromAudit`). On the real 513-spec corpus, passed went from 4 to 118 |
| 4 | AC-4: priority/severity normalization; initiative progress | DONE | `normalizeLevel` implements `read-contract-v1`'s amended synonym table exactly. `TestNormalizationAndProgress`: P1→high, moderate→medium, unset→null, tracker, progress 1/3, parent, JSON nulls |
| 5 | AC-5: revision changes with file, related status, audit; otherwise stable | DONE | `revision.go`. Round 2: `NewCorpus` never lets a promoted intake shadow the spec sharing its slug, so related status resolves to the live spec (`TestRound2AuditCases`). `TestRevisionChangesOnlyWithInputs`. Audit round 1: the revision moves when an item ages out of `recently_done` (`TestEdgeCasesFromAudit`) |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | model.go | DONE | Also defines the `NextStep`/`Phase`/`Extra` types for the engine, plus the `BuildOne` and `Corpus` helpers |
| 2 | revision.go | DONE | — |
| 3 | model_test.go | DONE | 6 tests over a real `spec.Discover` corpus in a temp dir |

### Exercise-the-feature check

- [x] The tests build real spec files on disk and parse them with `spec.Discover`, the same path the MCP tools use. There is no user-facing surface until `hero-work-tool`.

### Excellence Bar self-check

- [x] Yes. One pure model encodes `read-contract-v1` exactly, with explicit null semantics for the JSON contract.
