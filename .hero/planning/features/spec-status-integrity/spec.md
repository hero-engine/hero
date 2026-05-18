---
title: Spec Status Integrity — Graph-Verified Delivery Claims
slug: spec-status-integrity
type: feature
status: delivering
status_verified: "2026-04-29 by hero ac record: 4/6 ACs passing (AC-5 pre-commit hook + AC-6 auto-downgrade-on-regression deferred to a follow-on phase)"
priority: P0
tags: [process-integrity, validation, anti-drift, foundational]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: parent
  - target: project-charter
    kind: sibling
  - target: acceptance-criteria-graph
    kind: depends-on
  - target: v2-delivery-audit-2026-04-28
    kind: motivated-by
mission_alignment: |
  When the corpus lies about itself, every downstream context-injection
  inherits the lie. An agent told "this spec is completed" trusts that
  framing; if the code doesn't match, the agent makes wrong decisions.
  Truthful spec status is a precondition for the corpus being a useful
  context source. This feature makes "delivered" a verifiable state, not
  a markdown claim.
principles_check: |
  Serves #4 (sessions end making everyone smarter — but only if the
  corpus they enrich is honest). Risks #1 (it just works) if validation
  becomes ceremonial; mitigated by making it run automatically on
  commit/scan, never asking the user to "validate." Risks #5 if it
  surfaces as a separate validation CLI; mitigated by piggybacking on
  existing `hero check` and `hero scan`.
horizon: now
smoke:
  script: scripts/smoke/spec-status-integrity.sh
  expects: [spec-status-integrity:AC-1, spec-status-integrity:AC-4]
  runs_on: [commit-touches:internal/integrity/*.go, commit-touches:internal/cli/check*.go, nightly]
---

## Goal

A spec cannot claim `status: completed` (or claim phase ✅ in a phased
plan, or claim "delivered" in a body block) without the graph backing
the claim with passing acceptance criteria and present implementation.
Lying becomes structurally expensive, not a habit.

## Why now

The recovery audit found at least three drift modes and at least three
specs lying about delivery:

1. **`auto-capture learnings`** — frontmatter says `status: completed`;
   no implementation exists anywhere
2. **`graph-schema-simplification`** — phase 7c commit message claims
   "schema simplification"; schema is unchanged
3. **`graph-memory` phased-plan table** marks all 10 phases ✅ shipped;
   reality averages ~60% across them

These aren't moral failings. They're failures of a process that lets
self-reported status drift from reality. The fix is to make the
"delivered" claim a graph-verified state derived from passing ACs +
present code, not a frontmatter assertion the author types.

## Surface

### Frontmatter validation rules

`status: completed` is rejected by `hero check` unless:

1. The spec has `## Acceptance criteria` with at least 1 parsed AC
2. **All** of those Criteria have status `passing` in the graph (per
   `acceptance-criteria-graph` ingest)
3. Each AC has at least one `satisfied_by` commit edge OR a recent
   passing run-result

If any of those fail, `hero check` proposes a downgrade to one of:

- `delivering` — work in progress, some ACs passing
- `partial` — some ACs passing, others failing or proposed
- `planning` — no ACs passing yet
- `blocked` — explicit user override (requires comment)

### Phased plan tables

If a spec contains a phased-plan table with ✅ checkmarks, `hero check`
parses each row and verifies the corresponding child specs (or named
phase delivery) actually passes the rules above. Misleading checkmarks
are flagged; auto-fix proposes downgrade to ⚠️ or ❌.

This catches the `graph-memory` phased-plan-fraud pattern directly.

### Commit-message claim audit

Optional pre-commit hook: parses commit messages for "delivers", "fixes",
"completes", "ships <feature>" patterns. Resolves the named feature in
the graph. If the commit doesn't actually contain code that satisfies
that feature's ACs, the hook warns (but doesn't block — too noisy).

`hero check commits --since 30d` runs the same audit asynchronously and
produces a "claim drift" report.

### Status truthfulness report

`hero check status` produces a workspace-wide report:

```
Specs claiming `completed`:                   42
  Verified by passing ACs:                    27 ✅
  Lying (no implementation found):             2 🔻 [list with paths]
  Partial (some ACs failing):                  8 ⚠️ [list]
  No ACs (cannot verify either way):           5 ❓ [list]
```

Surfaced in `hero status`, `hero pulse`, dashboard health page.

### Auto-downgrade tooling

`hero check status --auto-fix` rewrites frontmatter for the lying-spec
cases — flips `completed` to `partial` or `planning`, writes a comment
field with the verifier's evidence ("downgraded by hero check on
2026-04-28: ACs 2, 5, 7 of 7 are not passing"). User reviews the diff
and commits. This is the first pass of recovery applied to existing
specs.

## Acceptance criteria

**AC-1:** ✅ **passing** (commit `8f938d5`, 2026-04-28).
`hero check status` exits non-zero on any spec with
`status: completed` and at least one `failing`, `regressed`, or
`proposed` AC. Verified end-to-end: temporarily marked
master-ingest-restore as completed with one regressed AC →
`hero check status; echo $?` → exit 1, lying verdict surfaced.
Restoring AC-2 to passing returns exit 0. Unit tests in
`internal/integrity/status_test.go` cover lying, partial, verified,
unverifiable, non-completed-skipped, and sort-order verdicts.

**AC-2:** `hero check status --auto-fix` correctly downgrades the three
known liars from the v2 audit (`auto-capture`,
`graph-schema-simplification` phase claim, `graph-memory` phased-plan
✅ row for any phase whose ACs aren't passing). Verified by reviewing
the resulting diff.

**AC-3:** Phased-plan checkmark parsing: any ✅ in a `| Phase | ... |
Status |` table column is verified against the named child spec or
phase. Misleading ✅ flagged with row number. Verified on the
`graph-memory` spec.

**AC-4:** ✅ **passing** (commit `8f938d5`, 2026-04-28). One-line
status truthfulness summary surfaced in `hero check` default output:
`Status truthfulness: 0/68 verified, 68 unverifiable` on the live
hero corpus today. The summary line is generated by
`statusSummaryLine()`; `hero check` runs it after the existing drift
+ knowledge passes and bumps the issue counter when lying or partial
specs exist. (Note: spec said `hero status`; `hero check` is the
real audit verb. `hero status` could grow the same line in a future
pass — captured as a follow-up.)

**AC-5:** Pre-commit hook (opt-in via `hero hooks setup
--include status-truth`) runs `hero check status` on the spec touched
in the commit; fails commit if `status: completed` is added without
all ACs passing.

**AC-6:** Specs with `status: completed` retain that status across
re-scans as long as ACs remain passing. If any AC regresses, status
auto-downgrades to `regressed` (new status) and a graph event fires
that surfaces in `hero recap`.

## Approach

**Phase 1 — validation rules** (~1 day): Build the validator on top of
the AC graph from `acceptance-criteria-graph`. Wire into `hero check`.
Fixture-based tests.

**Phase 2 — phased-plan parser** (~½ day): Markdown table parser for
phased-plan rows; map row to child spec or named phase; check
delivery.

**Phase 3 — auto-fix tooling** (~1 day): `--auto-fix` for downgrade.
Generates a diff for user review. Apply to current hero workspace as
the first dogfood pass.

**Phase 4 — auto-downgrade-on-regression** (~½ day): Watch AC status
flips; downgrade `completed` → `regressed` automatically; emit graph
event.

**Phase 5 — commit message audit** (~½ day): Async claim-drift report.
Defer the pre-commit hook to opt-in; default off.

## Dependencies

- **`acceptance-criteria-graph` must ship first.** Without
  Criterion-as-graph-node, there's nothing to verify against.
- **`project-charter`** ships first or in parallel — `hero check`
  already needs to be enforcing the mission/principles fields.

## Out of scope

- LLM-based "does the code actually do what the spec describes"
  semantic validation — too noisy for v1
- Cross-repo claim audit (this initiative's commit claims a feature
  in another repo) — defer
- Backdating: rewriting commit history to remove false claims — never

## Open questions

- Should `regressed` be a permanent terminal state requiring explicit
  re-completion, or should it auto-flip back to `completed` when ACs
  go green again? Lean: auto-flip back, but emit an event so the
  cycle is visible in `hero recap`.
- Should the audit retro-apply to specs older than 30 days, or only
  recent ones? Lean: full corpus on first run, then incremental.
- How loud should `hero status` be about lying specs? Lean: one-line
  summary by default, full list behind `hero check status`.
