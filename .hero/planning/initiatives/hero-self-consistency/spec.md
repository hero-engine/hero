---
title: "Hero Doesn't Lie — Self-Consistency Between Generated Guidance, Hero's Own Writes, and Hero's Actual Contract"
slug: hero-self-consistency
type: initiative
status: planning
domain: engineering
priority: high
size: large
horizon: now
created: 2026-07-14
tags: [self-consistency, contract, validation, dogfood, correctness]
relations:
  - target: resume-emits-dead-recall-command
    kind: child
  - target: spec-contract-enums-unified
    kind: child
  - target: generated-command-refs-validated
    kind: child
  - target: wire-checks-to-boundaries
    kind: child
  - target: spec-state-axes
    kind: child
  - target: hero-killer-features
    kind: refines
  - target: get-back-on-track
    kind: relates-to
---

# Hero Doesn't Lie — Self-Consistency Between Generated Guidance, Hero's Own Writes, and Hero's Actual Contract

## Goal

Hero's generated guidance, Hero's own writes, and Hero's actual product contract agree — and something gates the disagreement when they drift apart. Done means: one definition of the spec-type and status contract instead of three; zero specs carrying a status Hero itself writes and Hero itself rejects; zero dead command references in generated output; and the checks that already exist running at a boundary instead of waiting for a human to remember them.

## Kickoff

Hero tells the next agent session things that aren't true — a dead command in the cold-start digest, three disagreeing definitions of its own spec-type contract, statuses Hero writes that Hero rejects. Five children fix each lie and gate its return.

**Status:** planning — scope validated against code, no work started.

**Pick up at:** ship child #1 (`resume-emits-dead-recall-command`) standalone today — it's a two-line fix plus a test that currently enforces the lie. Then `/design` child #2, which unblocks #4.

→ `/deliver resume-emits-dead-recall-command`

**Files:** `internal/digest/digest.go:930`, `internal/digest/digest_test.go:190`, `internal/cli/validate.go:88`, `internal/triage/structural.go`
**Skip:** don't add attention/horizon work — `spec-prioritization` owns it and received two reassignments from here.

## Problem

Hero is a context-engineering product. Its primary output is guidance to an AI. When the cold-start digest names a command that doesn't exist, that is a defect in the shipped product surface — not corpus housekeeping. Every item below is a lie that makes the next agent session dumber.

Nothing gates the disagreement. The evidence, all verified in code:

### A. Three incompatible definitions of Hero's own spec-type contract

| Source | Types |
|---|---|
| `internal/cli/validate.go:88` | feature, bug, convention, decision, initiative, explainer (6) |
| `internal/triage/structural.go` | feature, bug, convention, decision, initiative, rule, external, context, note, explainer, intake (11) |
| `core/spec-types/*.md` on disk | bug, chore, epic, feature, initiative, intake, prd, release, sprint (9) |

The intersection is **{feature, bug, initiative} — 3 of ~15**. Three sets, three answers to "what is a spec type?"

`enhancement` exists in **no Go enum** — `grep TypeEnhancement internal/spec/spec.go` returns nothing. Yet 15 specs carry it, the `spec-sizing` skill bands "Feature / Bug / Enhancement" as first-class, and `cli-test-isolation-stray-workspace-boundary` is an actively-delivering `enhancement`. A type that ships work but exists in no enum is the contract failing in both directions at once.

`hero check validate` reports **150 `invalid type`**: context×114, enhancement×15, note×13, rule×2, reference×2, tripwire×1, plan×1. The 114 `context` specs are accepted by `structural.go` and rejected by `validate.go` — the two Go enums disagree *with each other*, not merely with the corpus.

### B. Hero writes statuses Hero rejects

14 `invalid status` total: `handed_off`×9, `handoff`×2, `designed`×2, `delivered`×1.

`internal/peering/handoff.go:212` writes `spec.StatusHandedOff` via `SetFrontmatterField`. `internal/cli/validate.go` `validStatuses` omits it. `hero handoff` is a shipped, CLAUDE.md-documented feature — so the product's own happy path produces specs the product's own validator calls invalid. `regressed` is the same bug with **zero** carriers; `handed_off` is the live one with nine.

### C. The cold-start digest emits a dead command

`internal/digest/digest.go:930` prints ``hero recall <topic>` to dig deeper`. `hero recall` does not exist — it was renamed to `hero search`. `internal/digest/digest_test.go:190` **asserts the string is present**. A passing test enforces the lie. This is the sharpest illustration of the thesis: the safety net is pinned to the defect.

### D. Checks exist but run at no boundary

`hero check validate` is a subcommand that plain `hero check` never invokes (`check.go:54`). `hero check --reconcile` defaults false (`check.go:47`), and `check.go:295` prints "Run 'hero check --reconcile' to auto-fix eligible items" — a reminder to a human, which is verbatim the lossiness that `status-reconciliation` (completed) was built to eliminate. A check nobody runs is a check that doesn't exist.

### E. Wiring the validator today would fire 1019 issues

On the maintainer's own repo: 504 `file not found`, 277 `missing smoke`, 150 `invalid type`, 14 `invalid status`. **781 of those (file-not-found + missing-smoke) are uncalibrated policy, not defects** — a planning spec whose `files:` names unbuilt code is *correct*. A gate that fires 1019 times is not a gate. This is why #4 is medium: the warn-vs-error policy decision is the work.

## Why this is corrective work, and why that's right

Four of five children fix what's broken rather than adding product surface. That ratio resembles a prior GPT-drafted initiative (`no-repeat-surprises`, archived, not in this repo — 11 children, 4 phases, invented metrics, zero verified defects) that was rejected for exactly this shape. The distinction is real and worth stating plainly:

- **That one groomed the corpus.** Its subject was spec hygiene — tidier metadata for its own sake.
- **This one fixes the product.** Hero's primary output is guidance to an AI. A dead command in the cold-start digest is a shipped defect in the thing customers pay for, and three disagreeing type enums are a broken contract in the engine.

Every child here traces to a defect verified in code, not a hypothesis. That is the test `no-repeat-surprises` failed and this passes.

Note also that `get-back-on-track` (giant, P0, `horizon: now`) is itself an unfinished corrective initiative — corrective work is already the house's top priority, not a novel indulgence. This initiative is deliberately scoped `large`, slots beside it, and hands it two reassignments rather than competing with it.

## Specs

Delivery order and rationale live in `plan.md`. Summary:

| # | Slug | Type | Size | Role |
|---|---|---|---|---|
| 1 | `resume-emits-dead-recall-command` | bug | trivial | Kill the dead `hero recall` ref and the test enforcing it. Standalone, ships today. |
| 2 | `spec-contract-enums-unified` | feature | medium | One source of truth for spec types + statuses. Unblocks #4. |
| 3 | `generated-command-refs-validated` | feature | small | Assert every `hero <subcommand>` ref in generated output resolves. Kills the (C) class permanently. |
| 4 | `wire-checks-to-boundaries` | feature | medium | Run the existing validator + reconciler at boundaries Hero already observes. **Hard-depends on #2.** |
| 5 | `spec-state-axes` | feature | large | Already designed at `.hero/planning/features/spec-state-axes/spec.md`. Adopted by reference; parallel from day 1. |

Child #5 is **adopted, not authored here** — its spec already exists and is not moved, restubbed, or redesigned.

## Dependencies

- **#2 → #4 is hard.** You cannot wire a validator that disagrees with the contract it validates. Wiring first would gate the corpus against a definition known to be wrong in three directions.
- **#3 is genuinely independent** — it reads the Cobra command registry only, touching neither the type enum nor the status enum.
- **#1 and #5 are independent** of everything. #1 ships immediately; #5 parallelizes from day 1.

## Acceptance Criteria

- THE SYSTEM SHALL define spec types and statuses in exactly one place, with `internal/cli/validate.go`, `internal/triage/structural.go`, and `core/spec-types/*.md` deriving from it rather than restating it.
- WHEN `hero check validate` runs over this repo THE SYSTEM SHALL report 0 `invalid type` and 0 `invalid status`.
- IF a code path writes a spec status THEN THE SYSTEM SHALL accept that status in every validator that reads it.
- WHEN generated output or an installed instruction file names a `hero <subcommand>` THE SYSTEM SHALL resolve that name against the Cobra command registry and fail the build if it does not exist.
- THE SYSTEM SHALL run the spec validator and the reconciler at a boundary Hero already observes, rather than printing a reminder for a human to run them.
- WHERE an issue class is uncalibrated policy rather than a defect THE SYSTEM SHALL warn rather than error, so the gate stays credible.

## Measures

Countable only. No modelled percentages.

| Measure | Now | Target |
|---|---|---|
| Spec-type contract definitions in disagreement | 3 sets, 3/15 intersection | 1 set |
| `hero check validate` → `invalid type` | 150 | 0 |
| `hero check validate` → `invalid status` | 14 | 0 |
| Specs carrying a status Hero writes and Hero rejects | 11 (`handed_off`×9, `handoff`×2) | 0 |
| Statuses written by one path, rejected by another | 2 (`handed_off`, `regressed`) | 0 |
| Dead command refs in generated output | 1 (3 occurrences) | 0 |
| Checks that exist but run at no boundary | 2 (`validate`, `reconcile`) | 0 |

Two measures are **forbidden** in this initiative's reporting; `plan.md` carries the reasoning. In short: "invalid statuses 3 → 0" is wrong (the real number is 14), and "`hero check validate` 1019 → 0" is wrong (781 are uncalibrated policy). An initiative titled "Hero Doesn't Lie" does not get to ship its own inaccurate claim.

## Boundaries

Explicitly **not** in scope. Named here so they don't creep in:

- **Expectation diff** ("Promised: 4 · Verified: 3 · Changed: 1 · Unexpected scope: none"). Valuable, and nearly free once `spec-state-axes` lands since it is just the two axes rendered. **Follow-on, not a child.**
- **Surprise ledger / correction memory.** The underlying gap is real — `tripwire-system` has no confidence, promotion, or expiry model, so a tripwire today is authored and instantly absolute. Revisit later as a *promotion model for tripwires*, not as a new subsystem.
- **General claim-checking** (spec relations resolve, referenced statuses accurate). #3 is the tiny high-value slice — command refs only. Defer the rest.
- **Attention/horizon work.** Deliberately not here. `spec-prioritization` (P0, planning, parent `get-back-on-track`) already owns the `horizon` field and specs auto-demote at line 28. Two findings were **reassigned there** rather than dropped — see "Reassigned to `spec-prioritization`" below.
- **No evaluation/metrics harness.**

## Reassigned to `spec-prioritization`

Recorded so the findings aren't lost. Neither is a child here.

1. **Deterministic horizon rules.** `superseded` and `handed_off` should never be `now`; a missing horizon means untriaged, not `now`. 26 of 91 planning specs lack a horizon; with the 23 explicit ones, 54% of the backlog claims current attention.
2. **Mechanical-write provenance + dormancy ranking.** Hero's bulk maintenance writes corrupt its own freshness signal by 19x — 7 specs report 3 days fresh by naive git mtime against a true dormancy of 57 days, because `chore(hero): backfill created: on 42 specs` touched 41 files. `events.log` is the honest substrate: it carries actor provenance (`"agent":"human/chet-bellows"`) and is immune to mechanical pollution by construction. But it is a completion ledger (126 `delivery_complete` vs 3 `spec_created`), so closing its coverage gap is the real work. Any dormancy ranking built on git mtime would be silently wrong in the direction that pins stale work to `now`.

## Risks

- **#4 is where this initiative can fail.** The 781 uncalibrated issues (E) are a policy question wearing a bug's clothing. If #4 treats them as defects it produces a gate that fires 1019 times, everyone disables it, and the initiative ships negative value. Warn-vs-error must be settled before any wiring lands.
- **#2 touches `core/spec-types/*.md`, which `hero install` propagates.** Changing the type contract is a harness-facing change. Check propagation before landing.
- **Unifying the enums will reveal, not create, breakage.** 150 specs currently carry a type some validator rejects. Unification forces a decision per type (`context`×114 especially): admit it to the contract or migrate the specs. Deciding 7 type-fates is #2's real cost, not the enum plumbing.
- **`refines` is not a parsed relation kind anywhere in the Go code**, and this initiative's own `refines: hero-killer-features` edge survives only because `relations:` accepts free-form kind strings. A top-level `refines:` key is silently dropped. This is an in-class instance of the initiative's own thesis, found while authoring it (see `plan.md` → "Authoring note").
- **Sizing:** initiative at `large` is within the normal band for its type — no promotion nudge fires. Deliberately smaller than the two `giant` P0 initiatives already in flight.

## Progress

Not started. Child #1 is ready to ship standalone; #5 is already designed and can start in parallel immediately.
