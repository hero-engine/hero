---
title: "hero why starts from a promoted intake instead of its spec when both share a slug"
slug: why-intake-slug-tie
type: bug
status: completed
priority: P2
severity: moderate
root_cause_class: code
domain: engineering
size: trivial
created: 2026-10-07
tags: [why, traversal, intake, mail, flaky-test]
completed_at: 2026-10-07T03:16:23Z
---

# hero why: promoted spec wins the slug tie

## Goal

`hero why <slug>` always starts from the real spec when a promoted Mail intake shares its slug. The provenance chain then walks spec → derived_from → intake → mail_source deterministically.

## Kickoff

`traversal.resolveTarget` picks `ORDER BY (repo match) DESC, ingested_at DESC LIMIT 1`. `ingested_at` has one-second resolution, and promotion writes the intake right after its spec, so the start node was whichever won the tie. This surfaced as the flaky `TestPromotionResumesAfterEveryStepAndWritesBodyFreeProvenance`, which blocked the v0.35.1 release build. The fix orders intakes last, with `id` as the final tiebreak. Test with `go test ./internal/traversal -run PromotedSpecWins`.

## Root Cause

A nondeterministic tiebreak on a coarse timestamp across node types that share a key. The spec and its intake both match `key = slug`.

## Fix

`ORDER BY (repo = ?) DESC, (type = 'Intake') ASC, ingested_at DESC, id DESC`. This is the same "work spec beats intake" rule the read-contract work model uses (`workmodel.slugRank`).

## Acceptance Criteria

- **AC-1:** WHEN a promoted intake and its spec share a slug THE SYSTEM SHALL resolve `hero why` to the spec, regardless of ingestion order, and the chain SHALL include both the derived_from and mail_source hops.
- **AC-2:** THE SYSTEM SHALL resolve ties between same-key nodes deterministically.

## Changes

1. `internal/traversal/why.go`: `resolveTarget` order.
2. `internal/traversal/why_test.go`: `TestWhy_PromotedSpecWinsSlugTieWithItsIntake`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: spec wins, full chain | DONE | `TestWhy_PromotedSpecWinsSlugTieWithItsIntake` makes the intake ingested one second later (the RFC 3339 stored format). On the old code it fails with "start node = Intake"; it passes now. The original flaky `TestPromotionResumesAfterEveryStepAndWritesBodyFreeProvenance` passes 5/5 under `-race` |
| 2 | AC-2: deterministic ties | DONE | `id DESC` is the last sort key, so equal timestamps can no longer pick arbitrarily |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | why.go order | DONE | — |
| 2 | test | DONE | — |

### Exercise-the-feature check

- [x] The regression test drives the real graph store and `traversal.Why`. The production test that flaked (Mail promotion → `hero why` provenance) passes repeatedly.

### Excellence Bar self-check

- [x] Yes. It fixes the actual nondeterminism instead of retrying the flaky test.
