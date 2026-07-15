---
title: "Spec State Axes — Separate Delivery Lifecycle from Verification Health"
slug: spec-state-axes
type: feature
status: planning
domain: engineering
priority: high
size: large
horizon: now
created: 2026-07-14
relations:
  - target: hero-self-consistency
    kind: parent
  - target: acceptance-criteria-graph
    kind: depends-on
  - target: spec-status-integrity
    kind: refines
  - target: living-contract
    kind: related
  - target: tripwire-system
    kind: related
  - target: status-reconciliation
    kind: related
  - target: spec-prioritization
    kind: related
---

# Spec State Axes — Separate Delivery Lifecycle from Verification Health

## Context

`internal/integrity/regression.go` → `AutoDowngradeRegressions()` overwrites a
spec's frontmatter `status` from `completed` to `regressed` when a Criterion
node in the graph goes failing or regressed. The code comment calls this "the
bridge logic that closes spec-status-integrity AC-6" — it was deliberate.

This spec argues the choice was wrong, and corrects it additively.

Delivery lifecycle and verification health are two independent facts sharing one
field. Because they share it, recording the present ("an AC is red") destroys the
past ("this shipped"). "Shipped then broke" becomes indistinguishable from "never
shipped." A later session reading `status: planning` cannot tell whether the work
was never done or was done and later broke — and those two situations call for
completely different next actions.

Three things found while reading the code make the case stronger than "the
semantics are muddy":

**1. The downgrade erases the drift it exists to report.**
`CheckCompletedSpecs` skips any spec whose status is not `completed`
(`internal/integrity/status.go:94`). `AutoDowngradeRegressions` also skips
non-completed specs (`internal/integrity/regression.go:70`). So the moment a
spec is downgraded to `regressed`, it disappears from the truthfulness report
*and* from the downgrade scanner. The mechanism meant to surface a lie makes the
lie invisible. It is a one-way ratchet with no reset — the doc comment at
`internal/spec/spec.go:50-51` claims "Reset to completed once the regressed AC is
passing again," and **no such code exists**.

**2. `regressed` is not a valid status.** `internal/triage/structural.go:38-43`
lists `workStatuses` as `{planning, in-review, delivering, completed}`.
`core/spec-types/feature.md:9` declares `states: [planning, refined, ready,
delivering, in-review, completed]`. Neither includes `regressed`. So
`hero ac record` writes a value that the very next `hero check` flags as
`invalid status for feature: regressed`. The two halves of the system
contradict each other today.

**3. The two downgrade paths disagree with each other.**
`AutoDowngradeRegressions` maps a red AC → `regressed`. `SuggestStatus`
(`internal/integrity/status.go:180-188`), which drives
`hero check status --auto-fix`, maps the same evidence → `planning` (lying) or
`delivering` (partial). Two commands, same input, different destructive rewrite.

Corpus scan: `rg -l "^status: regressed" .hero` returns **zero** files. No spec
in this repo currently carries the value — consistent with it being rejected by
the validator and never reset once set.

### Mission fit

*"Does this make the next agent session start smarter than the last one ended?"*
Yes. A session that can distinguish "this shipped and later broke" from "this
never shipped" makes materially better decisions — the first is a regression to
investigate against a known-good commit, the second is unfinished work to pick
up. That distinction is currently destroyed on write, and destroyed
irreversibly. This is the clearest possible case of the mission test.

## Goal

`status` keeps exactly one meaning: **delivery lifecycle** — where the work sits
on the planning → completed path. Verification health becomes a **separate state
derived from the acceptance-criteria graph**, computed on read, never authored,
never persisted. A completed spec whose AC goes red stays `completed` and is
separately reported as verification-failing, with the evidence and its age
attached. Delivery history is never destroyed in order to record present health.

Done means: `AutoDowngradeRegressions` no longer writes to any spec file; the
two axes render side by side in CLI, MCP, and the dashboard in plain English;
and `spec-status-integrity` AC-6's intent (a completed claim is never shown
unqualified while an AC is red) is honored by surfacing rather than by mutation.

## Kickoff

Splits spec `status` into two axes: `status` stays delivery lifecycle, and
verification health becomes a read-only state derived from the AC graph — so a
completed spec that breaks stays `completed` instead of being rewritten.

**Status:** planning — spec just landed, no code yet. Design decisions are
settled; see Approach.

**Pick up at:** start with the pure-read conversion — add `VerificationState`
+ `Evaluate()` in `internal/integrity/`, mapping from the existing `Verdict`
enum (the compute path already exists in `verifySpec`). Then gut
`AutoDowngradeRegressions` to `DetectRegressions` (no file writes) and fix its
one call site at `internal/cli/ac.go:233`. Rendering is Phase 3.

→ `.hero/planning/features/spec-state-axes/spec.md`

**Files:** `internal/integrity/regression.go:34`, `internal/integrity/status.go:91`,
`internal/cli/ac.go:233`, `internal/acceptance/query.go:14`

**Skip:** don't add a `verification:` frontmatter field — derived-on-read is the
decided approach, and a persisted copy would be a git-visible fact whose evidence
is machine-local. Don't design legacy evidence migration; the no-AC path is
already safe.

## Problem

One field carries two orthogonal facts:

| Question | Axis | Source of truth |
|---|---|---|
| How far along the delivery path is this? | delivery lifecycle | human/agent authored in frontmatter |
| Does the evidence say it currently works? | verification health | Criterion nodes in the graph |

These vary independently. A spec can be `delivering` with all ACs green (tests
written first), or `completed` with ACs red (shipped, then broke). Collapsing
them means every write to one clobbers the other, and the collapse is lossy in
exactly the direction that matters most: **the durable fact (it shipped) is
overwritten by the volatile fact (a test is red right now)**.

The volatile fact is also cheaply recomputable — it lives in `graph.db` and is
recomputed on every `hero ac record`. The durable fact is not recomputable; once
`status: completed` is overwritten, the information is gone from the file. The
system is currently destroying the irreplaceable to record the reproducible.

## Approach

### Decision 1 — Verification health is derived on read, not persisted

Three options were considered.

**(a) Derived on read from the graph. ← CHOSEN**
Computed from Criterion nodes at query time. No new field, no writes.

**(b) Managed frontmatter field** (`verification: failing`, stamped by code,
mirroring the `auto_downgraded` precedent at `internal/integrity/autofix.go:183`).

**(c) Materialized projection** — computed once, cached as a prop on the Spec
node in the graph.

Chosen **(a)**, for reasons that are specific rather than aesthetic:

- **The evidence is machine-local; a persisted verdict would not be.** Criterion
  status lives only in `graph.db`, which is gitignored
  (`.gitignore:15-18`) and rebuilt by `hero scan`. Writing `verification: failing`
  into `spec.md` commits a claim to git whose evidence is *not* in git. The next
  reader on another machine gets an assertion they cannot falsify, refresh, or
  audit. That is strictly worse than no field: it recreates the original defect
  (a stale state field lying about reality) on a new axis.
- **It cannot drift, because it is not a copy.** Option (b) and (c) both create a
  second representation of a fact, which means a sync problem, which means a
  reconciler. Option (a) has no second representation.
- **The compute path already exists and is already tested.** `verifySpec()`
  (`internal/integrity/status.go:137`) already derives exactly this from
  `acceptance.ListBySpec`. This decision is mostly *deleting* the write path, not
  building a read path.
- **It satisfies the hard constraints for free.** No new mandatory frontmatter
  field; no corpus rewrite; no eager file rewrites; reads tolerate missing fields
  indefinitely — because there is no field.
- **No LLM call anywhere.** The derivation is a pure fold over Criterion node
  statuses.

The cost, stated honestly: verification health is unavailable when `graph.db` is
absent, and `hero status` gains a graph read it does not perform today. Both are
acceptable. Graph-absent resolves to `unknown` — which is the honest answer and
is already the existing `VerdictUnverifiable` behavior. The graph read is a
single indexed query per spec and is skippable via a flag (see Changes item 6).

### Decision 2 — The state set is four values, and it already exists

The existing `Verdict` enum (`internal/integrity/status.go:22-35`) already
computes this. Do not invent a parallel vocabulary — repoint the existing one.
`VerificationState` is the public axis; `Verdict` stays as the internal
computation and its wire values are preserved for back-compat.

| State | Meaning | Derivation |
|---|---|---|
| `passing` | Every AC passing. | `Passing == Total`, `Total > 0` |
| `failing` | At least one AC failing or regressed. Concrete evidence of breakage. | `Failing > 0 \|\| Regressed > 0` |
| `partial` | No failures, but at least one AC not yet passing. No evidence either way. | `Total > 0`, no failures, `Passing < Total` |
| `unknown` | No Criterion nodes, or no graph. Cannot judge. | `Total == 0` or store unavailable |

**Why `regressed` is deliberately NOT a verification state.** This is the
load-bearing simplification. "Regressed" means *failing now, but it shipped* —
and "it shipped" is `status: completed`, which is the **delivery** axis. Encoding
regression on the verification axis would smuggle the delivery axis back into the
field we just freed. Regression is not a state; it is the **pair**:

```
status: completed  +  verification: failing   → regressed (shipped, now broken)
status: delivering +  verification: failing   → work in progress, not yet green
```

The pair carries what the single field could not. That *is* the fix. Adding a
distinct `regressed` verification state would re-merge the axes on the first day.

**Why `legacy-corroborated` is cut.** Its only purpose would be legacy evidence
migration, which is an explicit non-goal (see Boundaries) — and the premise is
false anyway: a spec with no ACs yields `VerdictUnverifiable`, and
`SuggestStatus()` returns `""` for it (`internal/integrity/status.go:187`), so no
downgrade ever fires on that path. It is already safe. There is nothing to
corroborate.

**Why `partial` and `unknown` stay distinct.** They imply different actions:
`partial` means ACs exist and some are unproven — run the tests. `unknown` means
no ACs exist — write some. Collapsing them would lose an actionable distinction,
and both already exist in the `Verdict` enum, so keeping them costs nothing and
preserves the `hero check status` contract.

### Decision 3 — What replaces the status overwrite

`AutoDowngradeRegressions()` becomes `DetectRegressions()`: same detection, same
returned findings, **zero side effects**. The rename is the point — the function
was never a "downgrader," it was a detector with a destructive rendering
attached.

**How `spec-status-integrity` AC-6's intent is still honored.** AC-6's intent was
that *a spec must not silently keep claiming `completed` while an AC is red*. The
operative word is **silently**. The frontmatter rewrite was the chosen mechanism,
not the goal — and as established in Context, it fails at the goal: after the
rewrite, the spec is skipped by both scanners and the problem goes quiet. The
intent is honored better by surfacing:

1. `hero check` already flags this via `Report.HasIssues()`
   (`internal/integrity/status.go:73`) — and will now keep flagging it, because
   the spec stays `completed` and stays in scope of `CheckCompletedSpecs`.
2. `hero ac record` keeps printing the regression loudly
   (`internal/cli/ac.go:262-267`) — it now prints instead of writing.
3. The verification axis renders adjacent to `status` everywhere the spec is
   listed, so a `completed` claim is never displayed unqualified while red.
4. MCP `hero_blocked` (`internal/serve/mcp_tools.go:3180`) already joins failing
   ACs and needs no change.

Net: the claim is qualified at every read, permanently, instead of being
destructively rewritten once and then forgotten. That is a strict improvement on
AC-6's intent, delivered by removal.

### Decision 4 — Evidence provenance and staleness

Derived state can go stale: `verification: passing` computed from a test run 90
days ago is not the same claim as one from this morning. Make the dependency
explicit by **carrying the evidence with the verdict** rather than by inventing a
staleness state.

`acceptance.Record` already writes `last_run_at`, `last_pass_at`, and
`last_run_id` onto every Criterion node (`internal/acceptance/record.go:102-107`).
**These are written today and read by nothing** — `criterionFromNode`
(`internal/acceptance/query.go:231`) does not project them. Projecting them is a
free win that requires no new write path.

`VerificationState` is returned inside a struct that carries:
- the contributing Criterion keys, split by status;
- `EvidenceAsOf` — the **oldest** `last_run_at` across the spec's ACs (the verdict
  is only as fresh as its stalest input);
- `LastRunIDs` — provenance for "which run said so."

**Staleness policy is explicitly out of scope.** This spec carries the evidence
age; it does not decide what age is "too old" or what to do about it. That policy
belongs to `attention-earned-not-assumed`, which owns `EffectiveHorizon` defaults
and superseded-vs-now rules. Rendering shows the age; it does not judge it.

### Decision 5 — `status: regressed` interpretation and migration

`StatusRegressed` becomes **deprecated, read-only, never written**.

Interpretation is unambiguous because the writer was: `AutoDowngradeRegressions`
only ever set `regressed` on a spec whose prior status was `completed`
(`internal/integrity/regression.go:70`). So the value means, exactly and always:

```
status: regressed  ≡  status: completed + (verification was failing at write time)
```

Reading rule: `regressed` normalizes to `completed` for every lifecycle purpose.
The parenthetical half is discarded on read — not because it is unimportant, but
because it is **recomputed from the graph anyway**, and the recomputed value is
current where the frontmatter value is a snapshot of unknown age. The legacy
value contributes nothing and is harmless.

Migration is opt-in and near-free: this corpus has zero such specs; downstream
installs may have some. `hero admin migrate-regressed` (dry-run by default)
rewrites `status: regressed` → `status: completed`. Nothing is eager. Specs that
are never migrated read correctly forever via the normalization rule.

### Tripwire: `harness-changes-cover-all-targets` — satisfied by non-applicability

This design introduces **no harness-facing surface**. It is internal Go plus
CLI/MCP/dashboard rendering. Specifically:

- No instruction-file content (`CLAUDE.md`, `AGENTS.md`, or any per-target
  equivalent) is added or changed.
- No routing guidance, agent guidance, or skill guidance about how to interpret
  or set the verification axis is added. **None is needed** — this is precisely
  why Decision 2's no-jargon requirement is load-bearing: the rendered output
  ("shipped — 2 of 7 ACs failing") is self-describing plain English. An agent
  reading it needs no instruction file to interpret it.
- No core behavior is gated on a Claude-only hook. The axis is computed and
  rendered by the Go engine, identically for all six install targets
  (`opencode | cursor | claude | copilot | codex | generic`).

If a future slice *does* want agent guidance on this axis, the rule holds:
author it harness-agnostically in `.hero/` and let `hero install` propagate to
all six targets. Nothing in this design creates a reason to do so.

## Acceptance Criteria

- WHEN an AC flips to failing or regressed on a spec whose status is `completed`
  THE SYSTEM SHALL leave the spec's frontmatter `status` field unmodified.
- THE SYSTEM SHALL NOT write to any spec file during `hero ac record`.
- WHEN a completed spec has at least one failing or regressed AC THE SYSTEM SHALL
  report it as `status: completed` with verification state `failing`, on every
  surface that lists the spec.
- WHEN a spec with a failing AC is listed by `hero check status` THE SYSTEM SHALL
  continue to report it on every subsequent run, rather than dropping it from
  scope after the first detection.
- THE SYSTEM SHALL derive verification state solely from Criterion nodes in the
  graph, with no authored frontmatter field as input.
- IF the graph store is unavailable or the spec has no Criterion nodes THEN THE
  SYSTEM SHALL report verification state `unknown` and SHALL NOT downgrade,
  rewrite, or flag the spec.
- THE SYSTEM SHALL return, alongside every non-`unknown` verification state, the
  contributing AC keys and the oldest `last_run_at` across those ACs.
- WHEN a spec's frontmatter carries the deprecated `status: regressed` THE SYSTEM
  SHALL interpret it as `completed` for all lifecycle purposes and SHALL derive
  verification state independently from the graph.
- THE SYSTEM SHALL NOT write `status: regressed` to any spec file.
- WHERE `hero admin migrate-regressed` IS invoked without `--apply` THE SYSTEM
  SHALL report the specs it would rewrite and change nothing.
- THE SYSTEM SHALL preserve every existing valid `status` value and every
  existing CLI and MCP command contract; new output fields are additive.
- THE SYSTEM SHALL render verification state in plain language, without the terms
  "lying" or "unverifiable", on all user-facing surfaces.
- THE SYSTEM SHALL make no LLM call in any code path that derives verification
  state.

## Changes

Ordered to keep the tree green at every step: the destructive write dies first
(Phase 1), the read model lands second (Phase 2), rendering last (Phase 3).

### Phase 1 — Stop the destruction

1. **`internal/integrity/regression.go`** — convert `AutoDowngradeRegressions` to
   a pure read.
   - Rename to `DetectRegressions(specs []*spec.Spec, store *graph.Store) ([]RegressionFinding, error)`.
   - Drop the `dryRun` parameter — the function no longer writes, so every call
     is a dry run.
   - Rename `RegressionDowngrade` → `RegressionFinding`; drop `OldStatus` /
     `NewStatus` fields (there is no transition anymore); keep `Slug`, `Path`,
     `RegressedACs`.
   - Delete `applyRegressionDowngrade` entirely.
   - Rewrite the package doc comment: this function detects; it does not mutate.
     Explicitly note that AC-6 is honored by surfacing (reference Decision 3).
2. **`internal/cli/ac.go:233`** — update the call site.
   - Call `integrity.DetectRegressions(specs, store)`.
   - Change the render at `:262-267` from
     `"🔻 %s: completed → regressed (%s)"` to
     `"⚠ %s — shipped, now failing: %s"`. No transition arrow; nothing transitioned.
   - Update the AC-6 comment block at `:226-231` to describe detection, not
     downgrade.
   - **Fix the pre-existing JSON bug**: the `--json` branch at `:235-241` returns
     before reporting downgrades, so today it silently rewrites files and reports
     nothing. Add `Regressions []integrity.RegressionFinding` to the JSON payload
     struct. (In scope: it is the same call site, and the bug is only visible now
     that the write is gone.)
3. **`internal/integrity/autofix.go`** — stop `--auto-fix` from rewinding shipped
   specs.
   - `SuggestStatus` (`internal/integrity/status.go:180-188`) currently maps
     `VerdictLying` → `StatusPlanning` and `VerdictPartial` → `StatusDelivering`.
     Both destroy delivery history for exactly the reason this spec exists.
     Change both to return `""` (no downgrade).
   - `PlanFixes` consequently emits `Skipped: true` for these verdicts. Update
     `skipReasonFor` with honest text: lying → `"AC evidence is red; status
     reflects delivery, not health — see verification state"`; partial → `"ACs
     not all proven; no delivery-lifecycle change warranted"`.
   - Leave `ApplyFix` and `rewriteFrontmatterStatus` in place — they are still
     used by other status-writing paths and are not the defect.
4. **Tests** — `internal/integrity/regression_test.go`,
   `internal/integrity/autofix_test.go`: replace assertions that a file was
   rewritten with assertions that **no file was rewritten** and the finding was
   returned. Add a regression test asserting `status: completed` survives a
   failing AC verbatim (byte-for-byte file compare).

### Phase 2 — The derived read model

5. **`internal/acceptance/query.go`** — project the provenance that is already
   written.
   - Add `LastRunAt time.Time`, `LastPassAt time.Time`, `LastRunID string` to the
     `Criterion` struct (`:14-20`).
   - Populate them in `criterionFromNode` (`:231`) from the `last_run_at` /
     `last_pass_at` / `last_run_id` node props written at
     `internal/acceptance/record.go:102-107`. Missing props → zero values; this
     must not error for specs recorded before the props existed.
6. **`internal/integrity/verification.go`** (new file, existing package) — the
   public axis.
   - `type VerificationState string` with `VerificationPassing`,
     `VerificationFailing`, `VerificationPartial`, `VerificationUnknown`.
   - `type Verification struct { State VerificationState; Total, Passing, Failing,
     Regressed, Open int; FailingKeys, OpenKeys []string; EvidenceAsOf time.Time;
     LastRunIDs []string }`.
   - `func Evaluate(s *spec.Spec, store *graph.Store) (Verification, error)` —
     reuse the `verifySpec` fold (`internal/integrity/status.go:137`); do not
     duplicate the counting logic. A `nil` store returns
     `{State: VerificationUnknown}` and a `nil` error — graph-absent is not an
     error condition.
   - `EvidenceAsOf` = **oldest** non-zero `LastRunAt` across the spec's criteria.
     Zero when no AC has ever been run.
   - `func VerificationFor(v Verdict) VerificationState` — total mapping:
     `VerdictVerified`→`passing`, `VerdictLying`→`failing`,
     `VerdictPartial`→`partial`, `VerdictUnverifiable`→`unknown`. Keeps `Verdict`
     as the internal compute and `VerificationState` as the public vocabulary, so
     the existing `hero check status` JSON contract is untouched.
   - `func IsRegressed(s *spec.Spec, v Verification) bool` — the pair predicate:
     `s.Status.IsDelivered() && v.State == VerificationFailing`. This is the only
     place "regressed" is defined, and it is defined as a conjunction of two axes,
     never as a stored value.
7. **`internal/spec/spec.go`** — deprecate `StatusRegressed`.
   - Retag the `StatusRegressed` const (`:49-52`) as deprecated: read-only, never
     written, retained for back-compat. **Delete the false claim** in the comment
     that it is "Reset to completed once the regressed AC is passing again" — no
     such code has ever existed.
   - Add `func (s Status) Normalize() Status` — maps `StatusRegressed` →
     `StatusCompleted`; identity for everything else.
   - Add `func (s Status) IsDelivered() bool` — `Normalize() == StatusCompleted`.
   - Do **not** normalize inside `parseFrontmatter`; the raw authored value must
     round-trip so a spec Hero did not write is never silently altered on read.
8. **`internal/triage/structural.go:38-43`** — resolve the live contradiction.
   - Add `spec.StatusRegressed: true` to `workStatuses` so legacy specs stop
     failing `hero check` with `invalid status for feature: regressed`. Add a
     comment marking it accepted-for-read-only, never written.
   - Leave the hardcoded error message at `:100` listing only the four canonical
     states — `regressed` is tolerated, not offered.
   - Do **not** add `regressed` to `core/spec-types/feature.md` or `bug.md`
     lifecycle states. The registry declares what the lifecycle *is*; `regressed`
     is not part of it and never was. Tolerance belongs in the reader, not the
     schema.
9. **Tests** — `internal/integrity/verification_test.go` (new): table-driven over
   the four states, plus nil-store → `unknown`, plus `EvidenceAsOf` picking the
   oldest input. `internal/spec/spec_test.go`: `Normalize` / `IsDelivered` over
   every status value including `regressed`.

### Phase 3 — Render both axes, no jargon

Principle: **silence when healthy.** The verification axis renders only when it
is `failing` or `partial`. A column that is blank on 90% of rows is noise; a
trailing qualifier that appears only when it means something is signal. No
surface gains a mandatory column.

10. **`internal/cli/status.go`** — `hero status`.
    - `printSpecGroup` row (`:283`) and the completed-spec row (`:166`): append a
      verification qualifier when non-green, e.g.
      `⚠ shipped, 2 of 7 ACs failing (last run 12d ago)`.
    - Bucket by `s.Status.Normalize()` at `:132-145` so legacy `regressed` specs
      land in the `completed` group rather than an unlabeled bucket.
    - Add `--no-verify` to skip the graph read for callers that want the current
      pure-file behavior and speed.
11. **`internal/cli/check_status.go`** — de-jargon the loudest surface.
    - `glyphForVerdict` (`:169-181`) and `printStatusReport` (`:229-283`): render
      `VerificationState`, not `Verdict`. Replace `"lying"` with
      `"shipped, ACs failing"`; replace `"no-ACs"` / `"unverifiable"` with
      `"no ACs recorded"`.
    - `statusSummaryLine` (`:186-203`): `"Status truthfulness: 3/7 verified, 2
      lying, 1 partial"` → `"Verification: 3 passing, 2 failing, 1 partial, 1 no
      ACs (of 7 completed)"`.
    - Show `EvidenceAsOf` age on failing findings. Report the age; do not judge it
      (staleness policy is `attention-earned-not-assumed`'s).
    - `--json` keeps emitting `verdict` verbatim and **adds** `verification`.
      Additive only.
12. **`internal/cli/list.go`** — `hero queue` / `hero list`.
    - `renderSpecsTable` (`:253-272`): keep the `STATUS` column rendering
      `s.Status.Normalize()`. Do **not** add a `VERIFY` column — mostly-blank.
    - `renderSpecText` (`:286`): append the qualifier when non-green.
    - `renderSpecs` JSON: add an additive `verification` object.
13. **`internal/serve/mcp_tools.go`** — mirror the CLI.
    - `formatStatusOutput` (`:1734-1767`): its status `switch` at `:1750` silently
      drops `regressed` and the `handed_*` states on the floor today. Bucket by
      `Normalize()` and add a `default` case so no spec is ever silently
      invisible. Append the same qualifier as the CLI at `:1767`.
    - `hero_check` / `toolCheck` (`:289`): additive `verification` in the payload.
    - `hero_blocked` (`:3180-3277`) needs no change — it already joins failing ACs
      directly.
    - **Note for the engineer**: `internal/cli/status.go:283` and
      `internal/serve/mcp_tools.go:1767` are structurally identical and
      independently maintained. Extract the qualifier formatter into one shared
      helper rather than writing it twice.
14. **`internal/serve/pages/work/data/roadmap.go`** — the dashboard.
    - `statusFor` (`:388-409`): bucket by `Normalize()`; drop the
      `regressed`→`blocked` special case at `:399-400`.
    - `isBlocked` (`:489-500`): stop string-matching `status == regressed`. Use
      `integrity.IsRegressed(s, v)` — the two-axis pair predicate.
    - `card.Bars` (`:342-347`) and `ChildRow.Progress` (`:373-377`) are both
      stubbed with explicit "follow-on" comments awaiting exactly this data. Fill
      them with the AC pass counts from `Verification`. This is the reserved slot;
      use it.
15. **`internal/cli/admin_migrate_regressed.go`** (new) — opt-in migration.
    - `hero admin migrate-regressed`, dry-run by default, `--apply` to write.
    - Rewrites `status: regressed` → `status: completed` via the existing
      `spec.SetFrontmatterField`. Stamps a `migrated_from: regressed` note
      adjacent to the status line, following the `auto_downgraded` precedent
      (`internal/integrity/autofix.go:183`): UTC date + reason, replace-not-append,
      idempotent.
    - Prints "0 specs to migrate" and exits zero on this corpus. That is the
      expected result here; the command exists for downstream installs.
16. **`internal/spec/select.go`** — selection consistency.
    - `statusRank` (`:261`): rank via `Normalize()`; `regressed` inherits
      `completed`'s rank.
    - `isClosedStatus` (`:305-311`): unchanged in effect, but route through
      `Normalize()` so `regressed` is explicitly closed rather than implicitly
      open by omission. Today a legacy `regressed` spec silently survives queue
      filtering — that is an accident, not a decision.
    - `internal/spec/horizon_migrate.go:73-74` maps `regressed` → `HorizonNow`.
      Leave it. Horizon policy belongs to `attention-earned-not-assumed`; touching
      it here would cross a boundary this spec drew.

## Boundaries

Explicit non-goals. Each is out of scope for a specific reason, not by oversight.

- **Legacy evidence migration is not designed here, because there is nothing to
  migrate.** A prior draft claimed absent ACs cause legacy downgrades. This is
  **false and verified false**: no ACs yields `VerdictUnverifiable`
  (`internal/integrity/status.go:149`), and `SuggestStatus()` returns `""` for it
  (`:187`), so no downgrade fires. The path is already safe. (Migration of
  existing `status: regressed` values **is** in scope — see Changes item 15. That
  is a different thing.)
- **Attention and horizon policy** — `EffectiveHorizon` defaults and
  superseded-vs-now rules belong to `attention-earned-not-assumed`. This spec
  carries evidence age; it does not decide what age is too old or what to do
  about it.
- **Completion-receipt UX** — out of scope.
- **Any new mandatory frontmatter field on every spec** — explicitly rejected by
  Decision 1. `migrated_from` (item 15) appears only on the specs that are
  migrated, and only when the user opts in.
- **Reopening `spec-status-integrity`, `tripwire-system`, or
  `acceptance-criteria-graph`** — all completed. Their implementations are
  preserved; this spec corrects semantics *around* them and reuses their evidence
  substrate. No cloning.
- **`internal/reconcile/` (`status-reconciliation`)** — the git-evidence
  reconciler is a third evidence source with its own axis question. Not touched.
  Worth a follow-on to check it does not have the same collapse.
- **A `hero verify` command or new verification write path** — the axis is
  derived. `hero ac record` remains the only writer of AC evidence.
- **Fixing the two divergent glyph vocabularies** (`statusGlyph` at
  `internal/cli/deliver.go:251` for criterion status vs `glyphForVerdict` at
  `internal/cli/check_status.go:169` for spec verdict — both use `✅`/`⚠️`, they
  diverge on failure with `❌` vs `🔻`). Pre-existing; a cleanup, not this spec's
  problem. Do not let it grow this change.

## Risks

- **`hero status` gains a graph read.** It is currently pure-file. Mitigated by
  `--no-verify` (item 10) and by the read being a single indexed query per spec.
  If `hero status` latency regresses measurably on a large corpus, fall back to
  rendering the axis only in `hero check status` and drop item 10 — the axis is
  still correct, just less visible.
- **`graph.db` is machine-local and gitignored, so verification state differs per
  machine.** This is pre-existing (criterion status already has this property) and
  is an argument *for* Decision 1, not against it — but it means two developers
  can legitimately see different verification states for the same spec. The
  `EvidenceAsOf` provenance is what makes that intelligible rather than
  mysterious. Do not "fix" this by persisting the verdict.
- **Removing `SuggestStatus`'s downgrades (item 3) changes `--auto-fix` behavior
  to a no-op for lying/partial verdicts.** This is intended — those downgrades are
  the same defect on a second path. But if any workflow depends on `--auto-fix`
  rewinding specs, it will go quiet. Grep for callers in CI config before
  landing; the Explore pass found none outside `internal/cli/check_status.go`.
- **Silence-when-healthy could make the axis too invisible.** If a completed spec
  is green, nothing renders — a user may not realize the axis exists. Accepted:
  `hero check status` always shows the full breakdown, so the axis is
  discoverable there. Revisit only if users report confusion.
- **Rollback**: Phases 1 and 2 are pure deletions and additions with no data
  migration and no file writes — revert the commit and the prior behavior returns
  intact. Phase 3 is rendering-only. Item 15 is the only writer, is opt-in, and
  its rewrite is idempotent and reversible by hand. **There is no irreversible
  step in this spec** — which is itself the strongest evidence that the axis split
  is the right shape.

## Validation

- `go test ./internal/integrity/... ./internal/acceptance/... ./internal/spec/...`
- **The load-bearing test**: seed a spec at `status: completed` with a passing AC;
  record a failing run via `acceptance.Record`; assert (a) `spec.md` is unchanged
  **byte-for-byte**, (b) `Evaluate()` returns `VerificationFailing`, (c)
  `IsRegressed()` is true, (d) `EvidenceAsOf` equals the failing run's timestamp.
  This single test is the spec.
- **The self-erasure test**: run the detection twice on the same red completed
  spec; assert the finding is returned **both** times. Under the old code the
  second run returned nothing because the first had rewritten the status out of
  scope. This asserts AC-6's intent is now actually met.
- Legacy interpretation: a fixture spec at `status: regressed` must load, must
  `Normalize()` to `completed`, must pass `hero check` structural validation, and
  must derive its verification state from the graph independently of the
  frontmatter value.
- Migration: `hero admin migrate-regressed` on this corpus prints "0 specs" and
  exits zero. On a fixture with one `regressed` spec, dry-run changes nothing and
  `--apply` is idempotent across two runs.
- Manual: `hero check status` output contains neither "lying" nor
  "unverifiable". Grep the rendering tests for both strings as a guard.
- Contract: capture `hero check status --json` and `hero_check` MCP output before
  and after; diff must be **additive only** — no removed or retyped fields.
- Confirm no LLM call exists in the `Evaluate` path (it is a pure fold; assert by
  inspection that `internal/integrity/` imports no LLM client).
