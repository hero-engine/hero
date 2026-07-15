---
title: "Spec Contract Enums Unified — One Definition of Hero's Types and Statuses"
slug: spec-contract-enums-unified
type: feature
status: planning
domain: engineering
priority: high
size: medium
horizon: now
created: 2026-07-14
parent: hero-self-consistency
tags: [contract, enums, spec-types, statuses, self-consistency]
---

# Spec Contract Enums Unified — One Definition of Hero's Types and Statuses

## Goal

Hero has exactly one definition of what a spec type is and what a spec status is. `internal/cli/validate.go`, `internal/triage/structural.go`, and `core/spec-types/*.md` derive from that single source rather than each restating it. `hero check validate` reports 0 `invalid type` and 0 `invalid status` on this repo, and no code path writes a status that another code path rejects.

## Kickoff

Hero has three disagreeing answers to "what is a spec type?" — a 6-type validator, an 11-type triage enum, and 9 files on disk. They intersect on 3. This makes one.

**Status:** planning — three definitions located and diffed; the per-type fate decisions are the open work.

**Pick up at:** decide the fate of each of the 7 orphan types before writing plumbing — `context`×114 and `enhancement`×15 are the load-bearing calls (admit to contract vs. migrate corpus). The enum plumbing is easy; the decisions are the spec.

→ `hero check validate 2>&1 | rg 'invalid type|invalid status' | sort | uniq -c | sort -rn`

**Files:** `internal/cli/validate.go:88`, `internal/triage/structural.go`, `internal/spec/spec.go`, `internal/peering/handoff.go:212`, `core/spec-types/`
**Skip:** don't wire the validator to a boundary here — that's `wire-checks-to-boundaries`, which hard-depends on this.

## Context

Parent initiative: `hero-self-consistency`. This child addresses findings A and B, and **unblocks child #4** (`wire-checks-to-boundaries`) — you cannot wire a validator that disagrees with the contract it validates.

Three sources define the type contract today:

| Source | Types | Count |
|---|---|---|
| `internal/cli/validate.go:88` | feature, bug, convention, decision, initiative, explainer | 6 |
| `internal/triage/structural.go` | feature, bug, convention, decision, initiative, rule, external, context, note, explainer, intake | 11 |
| `core/spec-types/*.md` | bug, chore, epic, feature, initiative, intake, prd, release, sprint | 9 |

Intersection: **{feature, bug, initiative} — 3 of ~15**.

`enhancement` exists in **no Go enum** — `grep TypeEnhancement internal/spec/spec.go` returns nothing. Yet 15 specs carry it, the `spec-sizing` skill bands it as first-class alongside Feature and Bug, and `cli-test-isolation-stray-workspace-boundary` is an actively-delivering `enhancement`.

`hero check validate` reports **150 `invalid type`**: context×114, enhancement×15, note×13, rule×2, reference×2, tripwire×1, plan×1. The 114 `context` specs are accepted by `structural.go` and rejected by `validate.go` — the two Go enums disagree with each other.

Statuses have the same shape of failure. `internal/peering/handoff.go:212` writes `spec.StatusHandedOff` via `SetFrontmatterField`; `internal/cli/validate.go` `validStatuses` (lines 101–111) omits it. `hero handoff` is shipped and CLAUDE.md-documented, so the product's happy path produces specs the product's validator rejects: 14 `invalid status` — `handed_off`×9, `handoff`×2, `designed`×2, `delivered`×1.

## Approach

The plumbing is not the work. **The per-type fate decisions are the work**, and they should be made explicitly and recorded, not absorbed silently into an enum edit.

1. **Single source of truth in Go.** The type and status enums already live in `internal/spec/spec.go` as `Type`/`Status` constants. Make that the canonical registry — exported, iterable, with a validity predicate. `validate.go` and `structural.go` both consume it. Neither restates a list.
2. **Derive, don't duplicate, on disk.** `core/spec-types/*.md` is documentation of the contract, not a second contract. Either generate the expected file set from the registry or add a test asserting the directory matches it. Do not hand-maintain a third list.
3. **Decide each orphan type explicitly.** For each of the 7 rejected types, one of two outcomes — admit to the contract, or migrate the corpus. Record the decision and its rationale. Notes on the load-bearing ones:
   - **`context` (×114)** — the largest class by far, and already accepted by `structural.go`. Migrating 114 specs to fit a 6-type enum is almost certainly the wrong direction; the enum is more likely wrong than the corpus. Decide deliberately, not by whichever list you edited first.
   - **`enhancement` (×15)** — carried by shipped work and treated as first-class by `spec-sizing`. Either it enters the enum or `spec-sizing` and 15 specs are wrong. Do not resolve this by deleting the band.
   - `note`×13, `rule`×2, `reference`×2, `tripwire`×1, `plan`×1 — smaller, but each needs a call.
4. **Close the status gap.** Add `handed_off` to `validStatuses`. `regressed` is the same defect with zero carriers — fix it in the same pass rather than leaving a known-broken write path armed. Then audit every `SetFrontmatterField(..., "status", ...)` call site to confirm each written status is accepted by every validator that reads it.
5. **Make divergence a test failure.** The reason three lists drifted is that nothing compared them. A test that fails when a type exists in one place and not another is what keeps this fixed.

## Changes

1. Establish the canonical registry in `internal/spec/spec.go`
   - Export an iterable set of valid `Type` and `Status` values plus validity predicates.
   - Add `TypeEnhancement` and any other type the fate decisions admit.
   - Add `StatusHandedOff` to the valid set (the constant already exists — only the validator's list omits it).
2. Rewrite `internal/cli/validate.go` to consume the registry
   - Delete the local `validTypes` map (line 88) and `validStatuses` map (lines 101–111); call the registry predicates instead.
   - Preserve the existing type/status compatibility checks (e.g. the convention-status rule at line 117).
3. Rewrite `internal/triage/structural.go` to consume the registry
   - Delete its local 11-type list; call the registry.
   - Any type it accepted that the registry does not must be resolved by a fate decision, not dropped silently.
4. Bind `core/spec-types/*.md` to the registry
   - Add a test asserting the file set matches the registry exactly — no file without a type, no type without a file.
   - Resolve the disk-only names (`chore`, `epic`, `prd`, `release`, `sprint`) via the same fate decisions.
   - **`hero install` propagates `core/spec-types/`** — see Risks.
5. Audit status write paths
   - `rg -n 'SetFrontmatterField.*"status"'` across `internal/`; confirm each written status passes the registry predicate.
   - `internal/peering/handoff.go:212` is the known live case.
6. Migrate or admit the corpus, per the fate decisions
   - For types decided "migrate": rewrite the carrying specs' frontmatter.
   - For types decided "admit": no corpus change.
   - Either way `hero check validate` reaches 0 `invalid type` / 0 `invalid status`.
7. Add the anti-drift test
   - Assert the registry is the only definition — a new type added in one place and not another fails the build.

## Acceptance Criteria

- THE SYSTEM SHALL define valid spec types and statuses in exactly one location, consumed by `internal/cli/validate.go` and `internal/triage/structural.go`.
- WHEN `hero check validate` runs over this repo THE SYSTEM SHALL report 0 `invalid type`.
- WHEN `hero check validate` runs over this repo THE SYSTEM SHALL report 0 `invalid status`.
- WHEN `hero handoff` writes `handed_off` to a spec THE SYSTEM SHALL accept that spec as valid.
- IF a spec type is present in `core/spec-types/` but absent from the registry (or the reverse) THEN THE SYSTEM SHALL fail the test suite.
- IF any code path writes a spec status THEN THE SYSTEM SHALL accept that status in every validator that reads it.
- THE SYSTEM SHALL record an explicit decision for each of the 7 currently-invalid types: admitted to the contract, or migrated in the corpus.

## Boundaries

- **Do not wire the validator to any boundary.** That is `wire-checks-to-boundaries` (child #4), which hard-depends on this spec landing first.
- Do not address `file not found` (504) or `missing smoke` (277) issues. Those are uncalibrated policy and belong to #4's warn-vs-error decision.
- Do not touch the `status` field's delivery-vs-verification axis collapse — that is `spec-state-axes` (child #5), running in parallel. Coordinate on `regressed`, which both specs touch: this spec only adds it to the valid set; #5 decides whether it should be a status at all.
- Do not refactor the frontmatter parser.

## Risks

- **`core/spec-types/*.md` is propagated by `hero install`.** That makes this a harness-facing change and puts tripwire `harness-changes-cover-all-targets` [high] in play. `hero install` writes across six targets (`opencode | cursor | claude | copilot | codex | generic`). Verify how `core/spec-types/` propagates before landing, and cover every target it reaches — not just Claude.
- **The 114 `context` specs are the decision that can go badly.** Reflexively "fixing the corpus to match the validator" would rewrite 114 specs to satisfy the narrower of two enums that already disagree. The validator is at least as likely to be wrong. Decide on merits.
- **Overlap with `spec-state-axes` on `regressed`.** Both specs touch it. Agree the split before either lands: this one makes the write/read contract consistent; #5 decides whether verification health belongs in `status` at all.
- **Unification reveals breakage rather than creating it.** Expect the 150 count to be politically larger than it looks — each is a real spec someone wrote.
- **Sizing:** `medium` is right if the fate decisions are made once and applied mechanically. If `context`×114 turns into a migration with per-spec judgment, this spec has become `large` — bump `size:` rather than absorbing it silently.

## Validation

- `hero check validate` reports 0 `invalid type` and 0 `invalid status`.
- `go test ./internal/spec/ ./internal/cli/ ./internal/triage/` passes.
- The anti-drift test fails when a type is added to the registry but not `core/spec-types/`, and when a type is added to `core/spec-types/` but not the registry — verify both directions by temporarily breaking each.
- Round-trip `hero handoff` on a scratch spec and confirm the resulting spec validates clean.
- Confirm the fate decision for each of the 7 types is recorded in the spec or a linked decision, not just in the diff.
