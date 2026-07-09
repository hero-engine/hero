---
title: "hero spec verify runs delivery gates on planning drafts (no lifecycle guard)"
type: bug
status: completed
slug: cst-verify-lifecycle-scoping
domain: engineering
priority: high
severity: medium
created: 2026-06-23
tags: [verify, delivery-gate, lifecycle, dx, cold-start]
parent: cold-start-trust-hardening
completed_at: 2026-06-23T19:57:04Z
---

# hero spec verify runs delivery gates on planning drafts

## Problem

`hero spec verify <slug>` runs the full delivery gates — Gate 1 Completion Ledger and Gate 2 Delivery Audit — against a spec in **any** status, including `planning`. A freshly drafted spec therefore FAILs with cryptic messages:

```
Completion Ledger   FAIL  no Completion Ledger section found in spec
Delivery Audit      FAIL  no audit report found (expected delivery-audit.md in spec directory)
```

A planning-stage draft has no business carrying a Completion Ledger or a delivery-audit.md — those are artifacts of the *delivery* phase. The failure misleads the user into thinking something is broken or missing, and can send them down a rabbit hole hand-producing delivery artifacts for a spec that hasn't been implemented yet.

**Reproduced live:** running `hero spec verify cold-start-trust-hardening` on the freshly-created planning initiative produced exactly these two FAILs. This was also observed in an external first-use session (the `candy` project), where the agent reacted by hand-writing a Completion Ledger and spawning a cold delivery audit on a planning draft — pure wasted motion.

## Root cause

**Classification:** missing precondition guard (lifecycle scoping).

`runVerify` in `internal/cli/verify.go:68-137` resolves the spec, handles the already-completed-and-archived case (`:95-103`), then runs all four gates **unconditionally** (`:107-121`). There is no check of `target.Status` before the gates run. Delivery gates are designed to run at the *end* of delivery — `feature-delivery-lead` only invokes `hero spec verify` after the ledger is DONE and the cold audit returns SHIP — but nothing stops a user (or agent) from running it on a `planning` draft and getting a confusing gate failure.

This is not an install or packaging problem: the `delivery-audit` skill ships correctly (`domains/engineering/skills/delivery-audit/SKILL.md`); it is invoked as a cold subagent, not a standalone agent. The gap is purely the missing status guard.

## Suggested Fix Approach

Add a lifecycle guard in `runVerify`, after the already-completed check (`verify.go:103`) and before the gates run (`:105`):

- If the spec's status is a **pre-delivery** state (`planning` or `draft`) and `--force` was not passed, do **not** run the gates. Return a clear, lifecycle-aware message naming the status and pointing at `/deliver`, e.g.:
  `spec "<slug>" is in planning status — delivery gates (Completion Ledger, audit report) only apply once delivery has started. Run /deliver to begin implementation, then verify; or pass --force to run the gates anyway.`
- `--force` bypasses the guard (consistent with its existing "bypass failed gates" semantics) so the old behavior remains reachable.
- In `--json` mode, emit a structured result with `Result: "SKIPPED"` and a `lifecycle` gate row, mirroring the existing JSON-exits-0 convention.
- Add a small helper `isPreDeliveryStatus(spec.Status) bool` covering `StatusPlanning` and `StatusDraft`.

Statuses that have entered delivery (`delivering`, `in-review`, `handed_back`, `completed`) are unaffected — gates still run exactly as before. The existing 16 verify tests all use `status: delivering`, so they are unaffected.

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL NOT run delivery gates (Completion Ledger, Delivery Audit) for a spec whose status is `planning` or `draft`; instead it SHALL return a single clear message naming the status and directing the user to `/deliver`.
- AC-2: THE SYSTEM SHALL bypass the lifecycle guard and run the gates when `--force` is passed.
- AC-3: THE SYSTEM SHALL run delivery gates unchanged for specs in `delivering`, `in-review`, `handed_back`, and `completed` statuses (no regression).
- AC-4: In `--json` mode for a pre-delivery spec, THE SYSTEM SHALL emit a structured result with `Result: "SKIPPED"` and a `lifecycle` gate, exiting 0.
- AC-5: A regression test SHALL cover a `planning`-status spec verify (guarded) and the `--force` bypass.

## Changes

- `internal/cli/verify.go` — add `isPreDeliveryStatus` helper and the lifecycle guard in `runVerify`.
- `internal/cli/verify_test.go` — add regression tests for the planning guard and the `--force` bypass.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | No delivery gates for planning/draft; clear lifecycle message | DONE | guard + `isPreDeliveryStatus` in `internal/cli/verify.go`; test `TestVerify_PlanningStatusGuarded` |
| 2 | `--force` bypasses the guard | DONE | guard condition `&& !verifyForce`; test `TestVerify_PlanningStatusForceBypass` |
| 3 | delivering/in-review/handed_back/completed unaffected | DONE | guard matches only planning/draft; existing 16 verify tests (all `delivering`) stay green |
| 4 | JSON mode emits `SKIPPED` + `lifecycle` gate, exit 0 | DONE | guard JSON branch; test `TestVerify_PlanningStatusJSON` |
| 5 | Regression tests for guard + force bypass | DONE | 3 tests in `internal/cli/verify_test.go` |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | `internal/cli/verify.go` — helper + lifecycle guard | DONE | `isPreDeliveryStatus` + guard in `runVerify` |
| 2 | `internal/cli/verify_test.go` — regression tests | DONE | 3 tests added |

### Exercise-the-feature check

- [x] Exercised: built the fixed binary and ran `hero spec verify cold-start-trust-hardening` (a real planning spec) — it now returns the lifecycle guard message instead of cryptic "no Completion Ledger" gate failures. Full suite green via `go test ./...`.

### Excellence Bar self-check

- [x] yes — surgical guard mirroring existing `--force`/JSON conventions; no regression; tests cover guard, force bypass, and JSON paths.

## Kickoff

**Pick up at:** implement the lifecycle guard in `internal/cli/verify.go`.

Cold-start prompt:
> Fix `hero spec verify` running delivery gates on planning drafts. In `internal/cli/verify.go`, `runVerify` runs all four gates unconditionally after resolving the spec (around line 105). Add a guard: if `target.Status` is `spec.StatusPlanning` or `spec.StatusDraft` and `--force` was not passed, return a clear lifecycle message ("spec is in planning status — delivery gates only apply once delivery starts; run /deliver, or --force to override") instead of running the gates. In `--json` mode emit `Result: "SKIPPED"` with a `lifecycle` gate and exit 0. Add `isPreDeliveryStatus(spec.Status) bool`. Existing verify tests use `status: delivering` and must stay green. Add regression tests in `internal/cli/verify_test.go` for the planning guard and the `--force` bypass. Verify with `go build ./... && go test ./internal/cli/`.

Part of the `cold-start-trust-hardening` initiative (delivery order #2).
