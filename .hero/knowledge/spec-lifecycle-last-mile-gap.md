---
title: Spec lifecycle last-mile gap — detection without prevention
type: decision
created: 2026-06-11
tags: [lifecycle, architecture, acceptance-criteria]
---

# Spec Lifecycle Last-Mile Gap

Hero's lifecycle contracts follow a consistent anti-pattern: every contract has **detection** (in `hero check`) but **zero transition-time enforcement**. The gap was discovered during dogfooding when `hero check` surfaced 120 issues across 5 dimensions on 2026-05-18.

## Key architectural insight

The AC graph infrastructure is fully built (`acceptance.Record()`, `ListBySpec()`, `StatusByFeature()`, bitemporal history) but the forward-looking writeback from delivery was never wired up. The Completion Ledger is parsed by `hero verify` (Gate 1) but the DONE rows are never fed to `acceptance.Record()`. This means the truthfulness verifier (`internal/integrity/status.go`) always returns `VerdictUnverifiable` — not because it's broken, but because its input is empty.

## Design decision

Rather than adding more detection, the fix pattern is: promote existing detection into transition-time gates. `hero verify` becomes the single path to `completed` (both manual and async), and the verify command handles the AC writeback + initiative auto-complete as post-gate side effects. This keeps the gate logic in one place rather than spreading it across deliver, complete, and async runner.

## Related

- `spec-lifecycle-hygiene-breakdown` — the P0 bug that identified all 5 symptoms
- `spec-status-integrity` — the AC graph backfill feature (handles historical data)
- `spec-completion-loop` — the forward-looking fix (handles new deliveries)
