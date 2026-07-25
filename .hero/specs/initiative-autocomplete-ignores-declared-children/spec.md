---
title: "Initiative auto-completion ignores declared-but-unscaffolded children"
slug: initiative-autocomplete-ignores-declared-children
type: bug
status: completed
domain: engineering
size: medium
priority: high
severity: high
created: 2026-07-25
tags: [hero-core, initiative, auto-complete, verify, drive, goal-check, roster, governance]
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code (pensive-rosalind-d30330)
  call_id: 18c592748b3f74b8ca24dc4552100441
  handed_off_at: 2026-07-25T15:37:36Z
  at_commit: c0e6a2f6
  reason: "Driving the hero-ops initiative from hero-code, a governance initiative auto-completed with its safety-critical financial-action gate unbuilt. Reproduced 3x. Black-box evidence from the consumer side; root cause is in hero's own auto-complete logic."
delivery_method: manual
completed_at: 2026-07-25T16:26:06Z
---

# Initiative auto-completion ignores declared-but-unscaffolded children

## Kickoff

**Pick up at:** delivered — all eight acceptance criteria implemented and
tested; closing gates (cold audit + `hero spec verify`) are the remaining step.

Paste into a fresh session if the closing gates still need to run:

> `initiative-autocomplete-ignores-declared-children` is implemented. The
> declared-child roster now lives in one place — `spec.DeclaredChildren`
> (`internal/spec/declared_children.go`) returns the de-duplicated union of
> frontmatter `child` relations and `## Child Specs & Sequence`
> table links — and is consumed by both the completion gate
> (`internal/spec/initiative_complete.go`) and drive's `declaredChildSlugs`
> (`internal/drive/stage.go`), so `hero spec verify` auto-completion and
> `hero goal --check` cannot disagree. `children:` (plural) parses to `child`
> edges and `child-of:`/`child_of:` to `parent`; unrecognized frontmatter keys
> are retained on `Spec.UnknownKeys` and `hero check` warns on near-misses.
> Run `go test ./internal/spec/... ./internal/drive/... ./internal/cli/...
> ./internal/reconcile/...`, then the cold delivery audit, then
> `hero spec verify initiative-autocomplete-ignores-declared-children`.

## Summary

`hero spec verify <leaf>` flips a parent initiative to `status: completed` and
prints `Initiative "<parent>" auto-completed — all children delivered` when only
a subset of the initiative's declared children have actually been delivered. The
completion decision counts only children that exist as spec files **on disk** and
declare the initiative as their parent; it ignores children the initiative
*declared* in frontmatter but that have not yet been scaffolded.

The reproduction that surfaced this (hero-code, `hero v0.27.2-1-gca6006e`) is the
worst possible shape: a **governance** initiative auto-completed at 1-of-4
children, and one of the three undelivered children was the financial-action
gate — the invariant that purchasing domains/servers always requires explicit
human confirmation. Hero reported the initiative as delivered while its hard
financial invariant did not exist in code.

An existing guard (the "roster gate" in `InitiativeReadyToComplete`) was written
specifically to prevent this. It does not fire because the declared roster never
reaches it. This is a starved-guard bug, not a missing-guard bug.

## Issue

Reported cross-repo via `hero peer call` (spec-out) from hero-code while driving
the `hero-ops-governance` initiative. Observed 3× in one session:

1. `hero-ops-governance/spec.md` declared four children in frontmatter:
   `children: [hero-ops-blast-radius-tiers, hero-ops-financial-action-gate, hero-ops-earned-autonomy, hero-ops-governance-gate]`
2. Only `hero-ops-blast-radius-tiers.md` existed as a file. The other three were
   declared-but-unscaffolded.
3. Delivering + `hero spec verify hero-ops-blast-radius-tiers` → the parent
   auto-completed at **1-of-4** children.
4. Reopen, scaffold a second leaf, deliver → auto-completed again at 2-of-4.
5. Only after scaffolding **all** leaves up front did completion behave correctly
   (4-of-4). This confirmed the count keys off on-disk files, not the declaration.

The consumer also flagged that this silently defeats `/drive`, whose contract
says done is reached only when *every intended child (including ones the
initiative declared but hadn't scaffolded)* is verified — and that
`hero goal <init> --check` agrees with the false completion because the parent is
already `completed`.

## Root Cause Analysis

Two compounding defects. The first triggers the bug; the second is why it stays
broken after an obvious parse fix and why `verify` and `goal --check` disagree.

### Defect 1 — `children:` (plural) is silently dropped by the relation parser

`internal/spec/spec.go:633` is the frontmatter case that turns relation keys into
`Relation` edges:

```go
case "relates-to", "depends-on", "depends_on", "supersedes", "parent",
     "child", "initiative", "conflicts-with", "conflicts_with":
```

It accepts `child` (singular) but **not** `children` (plural). The reproduction
used `children: [...]`. That key matches no relation case and no tracker prefix,
so it falls through to `default` and is dropped with no error and no warning.
Result: `parent.Relations` contains **zero** `child` edges.

The value itself parses fine — `parseList` (`spec.go:1018`) handles the inline
`[a, b, c, d]` flow array. The gap is purely the key name.

### Defect 2 — the completion roster gate is starved, then skips itself

`internal/spec/initiative_complete.go` (`InitiativeReadyToComplete`) has two
gates:

- **Roster gate** (`:39-52`): every declared `child`/`child-of` relation must
  resolve to a *materialized, completed* spec, else return false. This is exactly
  the guard that should block a 1-of-4 completion.
- **Child-count gate** (`:54-72`): at least one on-disk spec must declare this
  initiative as parent, and all such specs must be completed.

The roster gate reads `declaredCount` from `parent.Relations`. With Defect 1
dropping `children:`, `declaredCount == 0`, so the roster gate at `:50`
(`if declaredCount > 0 && !declaredComplete`) is skipped entirely. Only the
child-count gate runs — and with one delivered on-disk child (`blast-radius`
declaring `parent: hero-ops-governance`), `childCount == 1`, `allDone == true`,
so it returns true and the initiative completes. The message at
`internal/cli/verify.go:196` then prints as if all children shipped.

### Defect 3 (structural) — two different definitions of "declared children"

Even if Defect 1 is patched, the completion path and the drive/`goal --check`
path derive the declared-child roster from **different sources**, so they can
disagree by construction:

| Path | "Declared children" source |
|---|---|
| Completion gate (`InitiativeReadyToComplete`) | frontmatter `child`/`child-of` **relations** |
| Drive check (`internal/drive/stage.go:declaredChildSlugs`) | `## Child Specs & Sequence` **body markdown table**, via `childLinkRe` regex on `[slug](slug/spec.md)` links — frontmatter relations are ignored for the stub signal |

`declaredChildSlugs` never reads frontmatter relations; `InitiativeReadyTo
Complete` never reads the body table. An initiative that declares children in
one place but not the other makes the two systems disagree about
remaining/completed. This is the structural reason the consumer saw
`hero goal --check` agree with a false completion. The consumer's instinct —
"whatever decides auto-completion should agree with whatever `hero goal --check`
uses" — is correct and must be enforced in code, not left to authoring
discipline.

## Code Flow (End to End)

1. `hero spec verify <leaf>` passes its gates and archives the leaf.
2. `autoCompleteParentIfReady` (`internal/cli/verify.go:624`) walks the leaf's
   `parent`/`child-of` relations, discovers the parent via `spec.Discover`, and
   calls `spec.InitiativeReadyToComplete(parent, allSpecs)` (`:647`).
3. Because the parent's `children:` never became relations, the roster gate is
   skipped; the child-count gate passes on the single delivered leaf; the
   predicate returns true.
4. `completeAndArchive` flips the initiative to `completed` and the message at
   `:196` prints.
5. Independently, `hero goal <init> --check` (drive `check.go` → `buildIntended`
   → `declaredChildSlugs`) reads only the body table; with the parent already
   `completed`, the run reads as done. The two paths never reconciled a roster.

## Key Files

- `internal/spec/spec.go:633` — relation-key case; add plural/underscore aliases.
- `internal/spec/initiative_complete.go` — roster gate; feed it the unified roster.
- `internal/drive/stage.go:declaredChildSlugs` (`:66-90`) + `childLinkRe` (`:58`)
  — body-table roster source to fold into the shared function.
- `internal/drive/check.go:buildIntended` (`:221`) — consumer that must switch to
  the shared roster.
- `internal/cli/verify.go:196,624` — auto-complete message + call site (behavior
  unchanged, but message becomes truthful once the gate is fed correctly).
- `internal/reconcile/reconcile.go:102` — the second `InitiativeReadyToComplete`
  caller; must stay correct via the shared predicate.

## Goal

Auto-completion of an initiative must consult the **complete** set of children
the initiative declares — merging frontmatter relations and the body-table
roster — and must **block** completion while any declared child is not a
materialized, completed spec. The completion gate and `hero goal --check` must
compute the same roster so an autonomous `/drive` run can never declare victory
with declared children unbuilt.

## Suggested Fix Approach

### 1. One declared-children authority (fixes Defect 3)

Add a single exported function in `internal/spec` — `DeclaredChildren(init *Spec)
[]string` — that returns the **union**, de-duplicated and slug-normalized, of:

- frontmatter child relations (`Kind == "child"` or `"child-of"`), and
- `## Child Specs & Sequence` body-table links (relocate `childLinkRe` and the
  body-scan from `drive/stage.go` into `spec`, or have `spec` expose it and drive
  call it).

Both `InitiativeReadyToComplete` and drive's `buildIntended`/`declaredChildSlugs`
consume this one function. After this, the completion gate and `goal --check`
read from the same roster by construction.

### 2. Accept the plural / underscore aliases (fixes Defect 1)

In `spec.go:633`, add `children`, `child_of`, `child-of` to the case and
normalize them to `child`, mirroring the existing `depends_on → depends-on` and
`initiative → parent` normalization at `:639-646`. `children:` is the form
first-use sessions naturally reach for; it must form edges, not drop.

### 3. Roster gate uses the unified roster and blocks on any gap (fixes Defect 2)

Rework `InitiativeReadyToComplete`'s roster gate to iterate
`DeclaredChildren(parent)` rather than only `parent.Relations`. Any declared
child that is not a materialized, completed spec (missing file, or present but
not `completed`/`superseded`) → return false. This is the explicit answer to the
consumer's question **"block, warn, or something else?"** for a
declared-but-unscaffolded child: **block**. A governance initiative with an
unbuilt financial-action gate must never read as done. Retain the child-count
gate as a secondary guard (≥1 completed materialized child), but the unified
roster gate is now authoritative and correctly fed.

### 4. Stop silently dropping near-miss relation keys (defense-in-depth)

Add a `hero check` warning when frontmatter contains an unrecognized key that is
a near-miss of a known relation key (e.g. `subspecs`, `child_specs`, `depends`,
`parents`, `relates`). Now that `children` is accepted, this warning covers the
residual variants so the next natural spelling can't silently reintroduce a false
completion. This is a warning, not a hard error — it names the key and the likely
intended relation.

## Changes

- `internal/spec/spec.go` — add `children`/`child_of`/`child-of` aliases at the
  relation case.
- `internal/spec/initiative_complete.go` — introduce/consume `DeclaredChildren`;
  roster gate iterates the unified roster and blocks on any unsatisfied entry.
- `internal/spec/` (new or existing file) — `DeclaredChildren` + relocated
  `childLinkRe` body-table scan.
- `internal/drive/stage.go` + `internal/drive/check.go` — `declaredChildSlugs`
  and `buildIntended` delegate to `spec.DeclaredChildren`.
- `internal/cli/check.go` (or wherever `hero check` warnings are emitted) —
  near-miss relation-key warning.
- Tests as listed under **Validation**.

## Acceptance Criteria

- **AC-1:** WHEN an initiative declares children via `children:` (plural, inline
  `[a, b]` or block-style) in frontmatter THE SYSTEM SHALL parse each entry into
  a `child` relation edge normalized identically to the singular `child:` form.
- **AC-2:** WHEN `hero spec verify <leaf>` completes a child AND the parent
  initiative has any declared child — from frontmatter relations OR the
  `## Child Specs & Sequence` body table — that is not a materialized, completed
  spec THE SYSTEM SHALL NOT auto-complete the parent and SHALL NOT print the
  `auto-completed — all children delivered` message.
- **AC-3:** WHEN every child an initiative declares (the union of frontmatter
  relations and body-table links) resolves to a materialized, completed spec THE
  SYSTEM SHALL auto-complete the parent exactly once and print the message.
- **AC-4:** THE SYSTEM SHALL compute the declared-child roster from a single
  shared function consumed by both `InitiativeReadyToComplete` and drive's child-set
  builder, so `hero spec verify` auto-completion and `hero goal <init> --check`
  never disagree on which children remain.
- **AC-5:** WHEN `hero goal <init> --check` runs against an initiative with
  declared-but-unscaffolded children THE SYSTEM SHALL report those slugs as
  remaining (needs-scaffold) and SHALL NOT report the run as done.
- **AC-6:** WHEN `hero check` encounters frontmatter containing an unrecognized
  key that is a near-miss of a known relation key THE SYSTEM SHALL emit a warning
  naming the key and the likely intended relation and SHALL NOT silently drop it.
- **AC-7:** WHEN an initiative declares children only via the existing singular
  `child:` block form THE SYSTEM SHALL preserve current correct completion
  behavior (regression guard).
- **AC-8:** THE SYSTEM SHALL keep the reconcile re-check (`hero check
  --reconcile`) and the in-process `hero spec verify` side-effect using the same
  shared predicate, so both honor the unified declared-child roster.

## Boundaries

- **In scope:** the completion predicate, its declared-child roster source, the
  `children:` alias, drive/`goal --check` roster unification, and the
  near-miss-key warning.
- **Out of scope:** changing how `/drive` scaffolds or designs children;
  redefining what "completed" means for a leaf; tracker-status auto-completion;
  the reconcile path's leaf auto-complete heuristics beyond routing through the
  shared predicate.
- **Not a workaround request.** The known workaround (scaffold every declared
  child before delivering any) is explicitly rejected as the fix. Correctness must
  not depend on authoring order.

## Risks

- **Over-blocking:** an initiative that legitimately lists exploratory children it
  no longer intends to build would now never auto-complete. Mitigation: blocking
  is the safe default for a governance-grade invariant; the operator removes a
  child from the declaration (or marks it `superseded`, which the roster gate
  accepts) to intentionally drop it. Document this in the completion message /
  `hero check` output.
- **Roster relocation regression:** moving `childLinkRe` out of drive could change
  drive behavior subtly. Mitigation: AC-7 + existing drive tests as regression
  guards; keep the regex and its matching semantics byte-identical.
- **Double-source de-dup:** a child declared in both frontmatter and the body
  table must count once. Mitigation: `DeclaredChildren` de-dupes by normalized
  slug; covered by a dedicated test.

## Validation

| Test | Asserts |
|---|---|
| `children:` inline `[a,b]` → relations | AC-1: plural inline forms `child` edges |
| `children:` block-style list → relations | AC-1: plural block form parses |
| 1-of-4 declared (frontmatter), 1 on disk → `InitiativeReadyToComplete == false` | AC-2: starved-roster case blocked |
| 4-of-4 declared + all completed → `== true`, fires once | AC-3 |
| Same initiative through completion gate and `goal --check` → identical remaining set | AC-4 |
| `goal --check` with declared-but-unscaffolded child → slug in `remaining`, not done | AC-5 |
| `hero check` on spec with `subspecs:`/`child_specs:` key → warning emitted, not dropped | AC-6 |
| Existing singular `child:` block initiative → completion unchanged | AC-7 regression |
| Child declared in both frontmatter and body table → counted once | Risk: de-dup |
| `reconcile` path uses shared predicate on a starved roster → does not complete | AC-8 |

Run `go test ./internal/spec/... ./internal/drive/... ./internal/cli/... ./internal/reconcile/...`
plus a `go build ./cmd/hero` smoke.

## Notes

- The message at `verify.go:196` needs no change — once the gate is fed the full
  roster, "all children delivered" becomes true whenever it prints.
- This is a starved-guard bug. The roster gate's original author already reasoned
  the correct behavior ("delivering a single child would wrongly complete an
  initiative whose other children are unbuilt stubs", `initiative_complete.go:36-38`);
  the fix delivers the input that reasoning assumed.

## Investigation History

- 2026-07-25 — Received via `hero peer call` spec-out from hero-code
  (peer_id `cd8dd06d-…`), call `18c592748b3f74b8ca24dc4552100441`. Consumer
  supplied black-box reproduction (3×). Native root-cause investigation on
  hero's source (this workspace, at `b909d4c`) located: Defect 1 at
  `internal/spec/spec.go:633` (plural key dropped), Defect 2 at
  `internal/spec/initiative_complete.go:39-52` (roster gate skipped when
  `declaredCount == 0`), Defect 3 — the completion gate and
  `internal/drive/stage.go:declaredChildSlugs` derive "declared children" from
  different sources (frontmatter relations vs. `## Child Specs & Sequence` body
  table), which is why `hero spec verify` and `hero goal --check` disagreed.

## Completion Ledger

**Task as executed.** Starved-guard fix in three parts: make `children:`
(plural) parse to `child` edges, unify the declared-child roster behind one
`spec.DeclaredChildren` function that both the completion gate and drive's
child-set builder consume, and stop silently dropping near-miss relation keys.

**Stack:** Go (detected: `go.mod`, `internal/` package layout). Skills: go-stack,
implementation-principles, testing-and-validation.

**Validation performed.**
- `go build ./...` — clean.
- `go vet ./internal/spec/... ./internal/drive/... ./internal/cli/... ./internal/reconcile/...` — clean.
- `go test ./...` (whole repo) — green, no regressions.
- `gofmt -l` on all touched files — clean.
- **Pre-fix falsification:** temporarily reverted the `children:` alias and the
  frontmatter arm of `DeclaredChildren`; 8 of the new tests failed
  (`TestParsePluralChildrenInline`, `TestParsePluralChildrenBlock`,
  `TestDeclaredChildrenUnionsBothSources`,
  `TestDeclaredChildrenNormalizesPathTargets`,
  `TestInitiativeBlockedByUnscaffoldedChild`,
  `TestNearMissNotReportedForAcceptedPluralChildren`,
  `TestCheckHonorsFrontmatterDeclaredChildren`,
  `TestReconcile_NegativeGuard_PluralChildrenRoster`). Separately reintroduced
  roster divergence in drive only; `TestCheckAndCompletionGateAgreeOnRoster`
  failed with `rosters disagree: goal --check remaining-empty=true,
  completion gate=false`. The tests fail on the bug and pass on the fix.

**Deviations from the Suggested Fix Approach** (each deliberate, see notes):
1. §2 proposed normalizing `child-of:`/`child_of:` to `child`. That inverts the
   edge — every consumer in the tree (`cli/verify.go:626`,
   `snapshot/rollup.go:315,758`, `snapshot/release.go:56`,
   `initiative_complete.go:60`) reads the `child-of` *kind* as "X is my parent,"
   and normalizing the key to `child` would break parent discovery in
   `hero spec verify`. Implemented as `child-of:`/`child_of:` → `parent`, which
   is what the key means and what those consumers expect. `DeclaredChildren`
   still accepts a `child-of` *kind* on an initiative's own relations, so the
   pre-existing roster-gate behavior is unchanged.
2. The child-count gate now treats `superseded` as finished alongside
   `completed` (new `childFinished` helper). Without this the escape hatch the
   Risks section documents ("mark it `superseded` to intentionally drop a
   child") does not actually work — the roster gate would accept it and the
   child-count gate would still block forever.
3. `childTableSlugs`' fallback to "any section whose header starts with child"
   now sorts candidate section keys. The relocated code picked one by Go map
   order; a completion gate must not depend on map iteration order.

**Post-audit corrections.** The cold audit (`delivery-audit.md`, verdict SHIP)
surfaced one defect and two coverage gaps; all three were fixed in-session
before verify, and the full suite re-run green:
1. `DeclaredChildren` originally carried the old gate's `Kind == "child-of"`
   arm forward. That reads an initiative's own *parent* as a child — tolerable
   while the roster only over-blocked a completion, but the unification newly
   feeds it to drive's `buildIntended`, where a sub-initiative would list its
   parent as an intended child and never reach `done`. Now `child` only
   (`declared_children.go:46-57`); no authored form is lost, since the
   `child-of:`/`child_of:` keys normalize to `parent` at parse time. Test:
   `TestDeclaredChildrenIgnoresChildOfKind`.
2. AC-6's `hero check` wiring had tests only at the classifier level. Added
   `TestCheck_NearMissRelationKeyWarning` and
   `TestCheck_NoNearMissWarningForAcceptedKeys` at the CLI surface, mirroring
   the adjacent `TestCheck_WikilinkEdgeWarning`.
3. Deviation 3's determinism had no test. Added
   `TestChildTableSlugsFallbackIsDeterministic` (50 calls, stable roster).

**Risks / follow-ups.**
- `blocks:` and `related:` are documented relation *kinds* but are not accepted
  as top-level frontmatter shorthands, so they still parse to nothing. The new
  `hero check` warning now names them (it fires on 4 real specs in this repo),
  but accepting them as shorthands is a separate, out-of-scope change.
- `drive.Children` matches `r.Target == init.Slug` without `normalizeRelTarget`,
  so a path-form `parent:` target isn't discovered there. Pre-existing, untouched.
- Classifier residual risk noted by the audit: `agent:` sits at edit distance 2
  from `parent` and would be a plausible future false positive in an
  agent-oriented tool. Measured today at zero false positives across all 34
  distinct unknown frontmatter keys in the corpus.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `children:` (plural, inline or block) parses to `child` relation edges normalized identically to `child:` | DONE | `internal/spec/spec.go:639-660` — key added to the relation case, normalized to `child`. Tests: `TestParsePluralChildrenInline`, `TestParsePluralChildrenBlock` (asserts byte-identical relations vs. the singular form) |
| 2 | `hero spec verify <leaf>` must NOT auto-complete the parent, nor print the message, while any declared child (frontmatter OR body table) is not a materialized completed spec | DONE | `internal/spec/initiative_complete.go:47-52` blocks; `internal/cli/verify.go:196` prints only when `InitiativeCompleted != ""`, which requires the predicate to pass — no change needed there. Tests: `TestInitiativeBlockedByUnscaffoldedChild` (1-of-4, 2-of-4, and materialized-but-unfinished) |
| 3 | When every declared child (union of both sources) is a materialized completed spec, auto-complete exactly once and print | DONE | `initiative_complete.go:22-28` returns false for an already-completed initiative, so the fire is one-shot. Test: `TestInitiativeCompletesWhenFullRosterDelivered` (blocks on a table-only child, completes when it lands, refuses to re-complete) |
| 4 | Roster computed from a single shared function consumed by both `InitiativeReadyToComplete` and drive's child-set builder | DONE | `spec.DeclaredChildren` (`internal/spec/declared_children.go:32`) is the sole source; consumed at `initiative_complete.go:47` and `drive/stage.go:70` (which `check.go:228 buildIntended` calls). Test: `TestCheckAndCompletionGateAgreeOnRoster` — 5 scenarios asserting `goal --check` remaining-empty ≡ completion gate verdict |
| 5 | `hero goal <init> --check` reports declared-but-unscaffolded slugs as remaining and not done | DONE | Exercised live with the real binary on the reported initiative shape — see Exercise check |
| 6 | `hero check` warns on an unrecognized frontmatter key that is a near-miss of a relation key, naming key + likely relation; no silent drop | DONE | `Spec.UnknownKeys` populated at `internal/spec/spec.go:757`; classifier `spec.NearMissRelationKey` (`internal/spec/relation_keys.go:61`); `hero check` section at `internal/cli/check.go:531-555`. Tests: classifier — `TestNearMissRelationKey` (hits + two negative sets), `TestUnknownKeysRecordedNotDropped`, `TestNearMissNotReportedForAcceptedPluralChildren`; CLI surface — `TestCheck_NearMissRelationKeyWarning`, `TestCheck_NoNearMissWarningForAcceptedKeys` |
| 7 | Singular `child:` block form preserves current correct completion behavior (regression guard) | DONE | Test: `TestInitiativeSingularChildBehaviorUnchanged` (blocks on unfinished, completes on full). Pre-existing `TestReconcile_InitiativeCompleteFromArchivedChildren` and `TestReconcile_NegativeGuard_UnmaterializedChild` still pass unmodified |
| 8 | `hero check --reconcile` and the in-process `hero spec verify` side-effect use the same shared predicate and honor the unified roster | DONE | `internal/reconcile/reconcile.go:102` and `internal/cli/verify.go:647` both call `spec.InitiativeReadyToComplete`; unchanged call sites, corrected predicate. Test: `TestReconcile_NegativeGuard_PluralChildrenRoster`. Also exercised live — see Exercise check |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `internal/spec/spec.go` — `children`/`child_of`/`child-of` aliases at the relation case | DONE | `:639-660`. Also added `Spec.UnknownKeys` (`:236-241`) + population (`:702,757`) for AC-6 |
| 2 | `internal/spec/initiative_complete.go` — consume `DeclaredChildren`; roster gate blocks on any unsatisfied entry | DONE | `:35-52` roster gate rewritten; `:81 childFinished` shared by both gates |
| 3 | `internal/spec/` new file — `DeclaredChildren` + relocated `childLinkRe` body-table scan | DONE | `internal/spec/declared_children.go` — regex kept byte-identical to drive's; frontmatter-first order, slug-normalized, de-duped, self-reference dropped |
| 4 | `internal/drive/stage.go` + `internal/drive/check.go` — delegate to `spec.DeclaredChildren` | DONE | `stage.go:70` now a delegating wrapper (regex + `regexp` import removed). `check.go` needed no edit — `buildIntended:228` already routes through `declaredChildSlugs`, so it consumes the shared roster transitively |
| 5 | `internal/cli/check.go` — near-miss relation-key warning | DONE | Report section `:531-555`, collector `specsWithNearMissRelationKeys:824`; adds a `relation-key-near-miss` health row |
| 6 | Tests as listed under Validation | DONE | `internal/spec/declared_children_test.go` (12 tests), `internal/spec/relation_keys_test.go` (3), `internal/drive/check_test.go` (+2), `internal/cli/check_test.go` (+2), `internal/reconcile/reconcile_test.go` (+1). Every row in the Validation table is covered — including the de-dup risk case and, after the audit, the `hero check` row at the CLI level rather than only at the classifier level |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end with a real binary
  (`go build -o /tmp/hero-nm ./cmd/hero`) against a scratch workspace
  reproducing the reported shape — `hero-ops-governance` declaring four
  children via `children:`, with only `blast-radius-tiers` on disk and
  completed:
  - `hero goal hero-ops-governance --check` →
    `"verdict": "continue"`, `"action": "design"`,
    `"remaining": ["hero-ops-financial-action-gate", "hero-ops-earned-autonomy", "hero-ops-governance-gate"]`,
    `"completed": ["blast-radius-tiers"]`. Not `done` (AC-5).
  - `hero check --reconcile` → initiative left at `status: planning`, no
    completion finding (AC-8).
- [x] The `hero check` warning was exercised against this repo's real corpus
  (338 specs): it fires on exactly 4 specs — `superseded-by:` on
  `jira-connection-onboarding-misleads-agents` and `related:` on three others —
  all genuine silent drops. Signal, not noise (AC-6).

### Excellence Bar self-check

Yes. The fix removes the *class* of bug rather than patching the reported
instance: one function owns the roster, so the two consumers cannot drift again,
and every unrecognized frontmatter key is now retained and screened instead of
vanishing. The tests were falsified against the pre-fix code rather than merely
observed to pass, and the AC-4 invariant is asserted as an equivalence between
the two paths rather than by inspection. The three deviations from the suggested
approach are each documented above with the reason — the `child-of` one in
particular would have shipped an inverted edge if followed literally.
