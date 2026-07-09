---
title: "Priority- and conflict-aware next-spec selection in the /drive judge"
slug: priority-conflict-aware-drive-selection
type: enhancement
status: completed
size: medium
priority: high
domain: engineering
tags: [drive, autonomy, judge, sequencing, relations]
created: 2026-07-09
completed_at: 2026-07-09T18:28:17Z
---

# Priority- and conflict-aware next-spec selection in the /drive judge

## Context

The `/drive` judge (`drive.Check` in `internal/drive/check.go`) picks the next
child spec of an initiative to work on. Today it selects purely by
**slug-alphabetical order** among dependency-ready children. In real
autonomous drive runs this has forced manual override twice over, for two
distinct reasons:

1. **Priority inversion.** `Children()` sorts strictly by slug (check.go:49) and
   the selection loop picks the first ready child in that order
   (check.go:168–183). `spec.Spec` already carries `Priority` and `Severity`
   fields (spec.go:160–161), but `Check` never reads them. A P2 `size:large`
   child gets selected ahead of undelivered P0 safety specs purely because its
   slug sorts earlier.

2. **Overlap / seam collision.** An initiative's overlap seams and
   wave-ordering guardrails live only in **prose** (the child-table narrative,
   `[overlap]` tags). The judge parses only structured `parent` and
   `depends-on` relations, so it will happily select a child that edits the
   same code region as another spec that is **actively delivering in a parallel
   session** — there is no ordering relation between them, only a "don't run
   these two at once" constraint the judge can't see.

`Check` is a deliberately **pure, cold-startable function of on-disk state** —
the contract note at check.go:16 says "a cold process produces the same
verdict." **This determinism is a hard design constraint.** The fix must NOT
introduce an LLM/agent reasoning step into the selection loop. The LLM authors
the structured inputs (priority, `conflicts-with` relations) at
`/compose` / `/design` time; the judge only reads on-disk state and stays
reproducible.

## Kickoff

Makes `/drive` pick the next child by priority (not just alphabetical slug) and
stops it from starting a spec that collides with one already delivering.

**Status:** planning — spec just landed, no code yet. Design is grounded against
the real code; touch-points are exact.

**Pick up at:** start with Part A (priority tiebreak) — introduce a
`priorityRank`/`severityRank` helper and sort *ready candidates* by
`(priorityRank, severityRank, slug)` at selection time in `check.go`, leaving
`Children()` slug order intact for reporting. Then Part B (the
`conflicts-with` soft-mutex + `SeamCollision` pause). Determinism must hold:
same on-disk state → same verdict.

→ `.hero/planning/features/priority-conflict-aware-drive-selection/spec.md`

**Files:** `internal/drive/check.go:49,168-194`, `internal/drive/needsme.go:51-75,131-181`, `internal/spec/spec.go:578,1474`, `internal/spec/graph_ingest.go:224`
**Skip:** adding an LLM/agent step into the selection loop (breaks determinism, explicitly rejected); designing the deferred `wave` ordinal (redundant once priority lands).

## Problem

Two structured signals the judge should honor are either present-but-ignored
(priority/severity) or not yet expressible (a soft mutex between overlapping
specs). Both gaps degrade to the same failure: the judge selects the wrong
next child, and a human has to override the autonomous run.

## Goal

Among dependency-ready children, `/drive` selects the **highest-priority** ready
candidate (priority, then severity, then slug) instead of the
alphabetically-first, and **refuses to start** a candidate that is soft-mutexed
(`conflicts-with`) against a spec currently delivering — pausing with a
dedicated, non-promotable `SeamCollision` category that names the in-flight
conflicting spec. `Check` stays a pure function of on-disk state: same inputs →
same verdict, no wall-clock or LLM input in the loop. When no priorities and no
`conflicts-with` relations are present, behavior is byte-for-byte what it is
today.

## Approach

Two independent parts plus one explicitly deferred idea.

### Part A — Priority-aware tiebreak

- Introduce a **priority→rank** and **severity→rank** mapping (none exists in
  the codebase today). A small helper, e.g. in `internal/drive/check.go`:
  `critical=0, high=1, medium=2, low=3, unset/unknown=last (e.g. 99)`. Same
  mapping reused for severity.
- Change the **selection** step so that, among dependency-ready candidates, the
  next candidate is chosen by ordering key `(priorityRank, severityRank, slug)`
  rather than by first-in-slug-order. Slug remains the **final deterministic
  tiebreak** so the function stays reproducible.
- **Scope of the sort — decide and state explicitly.** `Children()` slug order
  (check.go:49) is also consumed by `buildIntended` ordering and by the
  `Remaining` / `Completed` report lists. **Recommendation (least invasive):**
  apply the priority ordering **only to selection of the next candidate** —
  i.e. when the loop is choosing `nextI` among ready children — and leave
  `Children()`, `buildIntended`, and the `Remaining` / `Completed` reporting
  order **stable on slug**. This keeps the reported lists visually stable and
  confines the behavior change to the one decision that matters. Do not reorder
  the whole child list.
- Concretely: the current loop (check.go:170–183) sets `nextI` to the *first*
  ready intended in iteration order. Instead, collect all ready intendeds, then
  pick the min under the `(priorityRank, severityRank, slug)` comparator. The
  `firstRem` fallback (first remaining child, used when nothing is ready) stays
  slug-ordered — it is a "nothing selectable" signal, not a real pick.

### Part B — `conflicts-with` soft-mutex relation + `SeamCollision` pause

- Introduce a new relation kind **`conflicts-with`** — a *soft* mutex, distinct
  from the *hard* ordering of `depends-on`. Two specs linked by
  `conflicts-with` must not be delivered concurrently, but there is **no
  ordering requirement** between them.
- **Whitelist in TWO places or the kind silently degrades to `related`:**
  1. the accepted-kinds switch in the parser at **spec.go:578** (add
     `conflicts-with` / `conflicts_with` so the relation is accepted and forms
     an edge instead of being dropped);
  2. the normalization switch in `graphEdgeForRelation` at
     **graph_ingest.go:224** (map `conflicts-with` to a distinct edge type,
     e.g. `conflicts_with`, so it doesn't fall through to `related_to`).
- **Gate rule in `Check`:** a child is **not selectable** if any of its
  `conflicts-with` targets is currently in a delivering state. Use the existing
  predicate **`spec.IsLocallyDelivering`** (spec.go:1474) — do not hand-roll a
  status check. Scope for v1: **local delivering only.** Peer-delivering states
  (the active-delivering invariant discussion around spec.go:1434–1474) are a
  documented follow-up unless trivial to include.
- **`SeamCollision` pause category.** If a conflicting-but-blocked child is the
  *only* otherwise-ready candidate, the verdict is `pause` with a new pause
  category `SeamCollision`. It is a **real obstacle, not a human-judgment
  fork**, so it must be **non-promotable** — even Autonomous mode pauses (same
  class as Blocked / Irreversible / HardCap). The pause reason must **name the
  conflicting in-flight spec** so the operator knows what to wait for.
- **Wiring `Check` → pause category.** Seam collision is a *sibling* of the
  existing "nothing selectable" branch (check.go:190–194), which sets
  `ctx.Blocked = true` when `nextI == nil` (every remaining child blocked on
  unmet deps). `Check` must distinguish:
  - **Blocked** — the first-remaining fallback is blocked on an unmet
    *dependency*.
  - **SeamCollision** — the first-remaining fallback is ready on deps but
    excluded because a `conflicts-with` target is delivering.
  Add a new `RunContext` field (e.g. `SeamBlocked bool` plus a field carrying
  the conflicting spec's slug for the reason string) and a corresponding
  `NeedsMe` branch, mirroring the existing `ctx.Blocked` handling.

### Part C — DEFERRED (out of scope)

A `wave` ordinal sorting before slug. Likely redundant once the priority
tiebreak lands (Wave A = P0, Wave D = P2, so priority already encodes the
intended order). Mentioned as a possible follow-up; **not designed here.**

## Changes

1. **`internal/drive/check.go` — priority/severity rank helper (Part A).**
   - Add a `priorityRank(string) int` (and reuse for severity, or a shared
     `rank(string) int`) implementing `critical=0, high=1, medium=2, low=3,
     unset/unknown=last`. Keep it total and case-insensitive against the string
     values `spec.Spec.Priority` / `.Severity` carry (spec.go:160–161).

2. **`internal/drive/check.go` — priority-aware candidate selection (Part A).**
   - In the selection loop (check.go:168–183), instead of taking the *first*
     ready intended, gather all ready intendeds and select the minimum under
     the `(priorityRank(spec.Priority), rank(spec.Severity), slug)` comparator.
   - Leave `Children()` (check.go:49), `buildIntended`, and the `Remaining` /
     `Completed` append order **slug-stable**. Document in a comment that the
     priority ordering is scoped to *selection only*, for determinism and
     stable reporting.
   - `nextStage` must still be computed from the chosen candidate.

3. **`internal/spec/spec.go:578` — accept `conflicts-with` in the parser
   (Part B).**
   - Extend the accepted-kinds switch to include `conflicts-with` (and the
     underscore variant `conflicts_with`), normalizing to the canonical
     `conflicts-with`. Without this the relation is dropped / degrades to
     `related`.

4. **`internal/spec/graph_ingest.go:224` — map the edge (Part B).**
   - Add a `case "conflicts-with", "conflicts_with":` to `graphEdgeForRelation`
     returning a distinct edge type (e.g. `"conflicts_with"`), so it does not
     fall through to `related_to`.

5. **`internal/drive/needsme.go:51–62` — add the `SeamCollision` category
   (Part B).**
   - Add `CategorySeamCollision PauseCategory = "SeamCollision"` to the const
     block with a comment: soft-mutex collision with an in-flight conflicting
     spec — a real obstacle, non-promotable.

6. **`internal/drive/needsme.go:68 (`Promotable`) — keep it non-promotable
   (Part B).**
   - Do **not** add `SeamCollision` to the `Promotable()` true-set. It stays in
     the default (false) branch alongside Blocked / Irreversible / HardCap, so
     Autonomous mode still pauses.

7. **`internal/drive/needsme.go` — `RunContext` + `NeedsMe` branch (Part B).**
   - Add a `SeamBlocked bool` field (and a companion field for the conflicting
     spec slug, e.g. `SeamConflictSlug string`) to `RunContext`
     (needsme.go:87–123).
   - Add a branch in `NeedsMe` (needsme.go:131–181) that returns
     `pause(CategorySeamCollision, ...)` naming the in-flight conflicting spec.
     Place it among the taxonomy categories (Guided + Autonomous) — but because
     it is non-promotable it must **not** route through `maybePromoted`; return
     a plain `pause(...)` so it pauses in every mode. Order it as a sibling of
     the `ctx.Blocked` branch (needsme.go:165–167).

8. **`internal/drive/check.go:168–194` — compute conflict exclusion and set the
   seam context (Part B).**
   - When evaluating whether an intended child is selectable, in addition to
     `ready(completed)`, exclude it if any of its `conflicts-with` targets
     resolves (via the existing `all []*spec.Spec` set) to a spec where
     `IsLocallyDelivering()` is true.
   - When `nextI == nil` (nothing selectable), distinguish the two fallback
     reasons: if the first-remaining child is ready-on-deps but conflict-
     excluded, set `ctx.SeamBlocked = true` and `ctx.SeamConflictSlug =
     <delivering conflict slug>`; otherwise set `ctx.Blocked = true` as today.
     `NeedsMe` then produces `SeamCollision` vs `Blocked` accordingly.

9. **Tests (Part A + Part B).**
   - `internal/drive/check_test.go`: add fixtures mirroring the existing
     structure-driven `scoreFn`-override style — (a) two ready children with
     differing priorities, assert the higher-priority one is selected though
     its slug sorts later; (b) a candidate whose `conflicts-with` target is
     delivering is not selected; (c) the only-ready-candidate-is-conflict-
     blocked case yields `pause` / `SeamCollision` naming the in-flight spec;
     (d) determinism: same on-disk state produces the same verdict across
     repeated calls; (e) backward-compat: no priorities + no `conflicts-with`
     reproduces today's slug-order selection and today's `Remaining`/`Completed`
     ordering.
   - `internal/drive/needsme_test.go`: add cases asserting `SeamCollision`
     pauses in **all** modes including Autonomous (non-promotable), and that
     the reason string includes the conflicting slug.
   - `internal/spec` parser/graph tests (matching the package's existing test
     style): a `conflicts-with` relation round-trips through the parser without
     degrading to `related`, and materializes the distinct edge.

### Delivered files

- `internal/drive/check.go` — `rank`/`lessCandidate`/`conflictDeliveringSlug`
  helpers, `intended.priority()`/`.severity()`, priority-aware +
  conflict-excluding selection in `Check` and `DryRun`, `SeamBlocked`/`Blocked`
  fallback distinction.
- `internal/drive/needsme.go` — `CategorySeamCollision`, `RunContext.SeamBlocked`
  + `.SeamConflictSlug`, `seamReason` helper, plain-`pause` SeamCollision branch
  alongside `Blocked`.
- `internal/spec/spec.go` — accept `conflicts-with` / `conflicts_with` in the
  frontmatter relation switch (normalize to canonical `conflicts-with`).
- `internal/spec/graph_ingest.go` — `conflicts-with` → distinct `conflicts_with`
  edge in `graphEdgeForRelation`.
- `internal/drive/check_test.go`, `internal/drive/needsme_test.go`,
  `internal/spec/spec_test.go`, `internal/spec/graph_ingest_test.go` — coverage.

## Implementation Notes

Exact touch-points, all verified against the current tree:

- **`internal/drive/check.go:49`** — `Children()` sorts kids by slug. **Leave
  as-is** (reporting/determinism anchor).
- **`internal/drive/check.go:168–183`** — the selection loop that sets `nextI`
  to the first ready intended. Replace "first ready" with "min ready under the
  priority comparator." Add the `conflicts-with` exclusion to the selectable
  test.
- **`internal/drive/check.go:189–194`** — the `RunContext` construction and the
  `nextI == nil` fallback that sets `ctx.Blocked = true`. Add the
  `SeamBlocked` / `SeamConflictSlug` distinction here.
- **`internal/drive/needsme.go:51–62`** — `PauseCategory` const block; add
  `CategorySeamCollision`.
- **`internal/drive/needsme.go:68`** — `Promotable()`; leave `SeamCollision` in
  the false branch.
- **`internal/drive/needsme.go:87–123`** — `RunContext` struct; add
  `SeamBlocked` + `SeamConflictSlug`.
- **`internal/drive/needsme.go:131–181`** — `NeedsMe`; add the plain-`pause`
  `SeamCollision` branch (not via `maybePromoted`).
- **`internal/spec/spec.go:578`** — accepted-kinds switch in the frontmatter
  parser; add `conflicts-with` / `conflicts_with`.
- **`internal/spec/graph_ingest.go:224`** — `graphEdgeForRelation`; add the
  `conflicts-with` case returning a distinct edge type (don't fall through to
  `related_to`).
- **`internal/spec/spec.go:1474`** — `IsLocallyDelivering()`; the exact
  predicate the conflict gate must call. Do not hand-roll a `Status ==
  StatusDelivering` check.
- Priority/severity string values live on `spec.Spec.Priority` /
  `spec.Spec.Severity` (spec.go:160–161): the strings `critical|high|medium|low`
  (possibly empty).

## Boundaries

- **No LLM/agent step in the selection loop.** The judge stays a pure function
  of on-disk state. Priority and `conflicts-with` are authored upstream at
  `/compose` / `/design` time.
- **No `wave` ordinal** (Part C) — deferred, not designed here.
- **No reordering of `Remaining` / `Completed` reporting** or of `Children()` /
  `buildIntended`. The priority ordering is confined to next-candidate
  selection.
- **Peer-delivering conflict states out of scope for v1** — local delivering
  only via `IsLocallyDelivering`. Note peer-delivering as a follow-up.
- No new CLI surface, no changes to `/compose` authoring UX (authors already
  write relations); this spec is the judge-side consumption only.

## Risks

- **Determinism regression.** The single biggest risk. The priority comparator
  must be total and end in `slug` so ties never depend on map iteration or
  input order. The conflict gate reads only `IsLocallyDelivering` (on-disk
  status) — no time or process state. Cover with the "same state → same
  verdict across repeated calls" test.
- **Silent relation degradation.** If only one of the two whitelist sites
  (spec.go:578, graph_ingest.go:224) is updated, `conflicts-with` degrades to
  `related` and the gate silently never fires. The round-trip parser test is
  the guard.
- **Backward compatibility.** Specs with no priority and no `conflicts-with`
  must select exactly as today. The empty-priority rank (last) plus slug
  tiebreak must reproduce pure slug order when all priorities are empty.
- **Wrong pause category.** Conflating `SeamCollision` with `Blocked` would let
  Autonomous mode's promotion path (if Blocked were ever promoted) skip a real
  seam. Keeping `SeamCollision` non-promotable and distinct is the safeguard.
- **Self-conflict / dangling targets.** A `conflicts-with` pointing at a
  non-existent or self slug should be a no-op, not a panic — resolve targets
  defensively against the `all` set.

## Validation

- `go test ./internal/drive/... ./internal/spec/...` passes, including the new
  fixtures.
- **Acceptance-criteria coverage:** every criterion below has a corresponding
  test (priority selection, determinism, conflict exclusion, `SeamCollision`
  pause naming the in-flight spec, non-promotability, relation round-trip,
  backward-compat).
- Manual: construct a small initiative with a P0 late-slug child and a P2
  early-slug child, run `hero goal --check` (the `Check` surface), confirm the
  P0 is selected. Mark a conflicting spec `delivering`, confirm the verdict
  flips to `pause` / `SeamCollision` naming it.
- Determinism check: run the judge twice against a frozen workspace and diff
  the JSON verdicts — identical.

## Acceptance Criteria

- WHEN multiple dependency-ready children exist THE SYSTEM SHALL select the one
  with the highest priority (then severity, then slug), not the
  alphabetically-first.
- THE SYSTEM SHALL produce the same verdict for the same on-disk state (no
  wall-clock or LLM input in the selection loop).
- IF a candidate has a `conflicts-with` target whose status is delivering THEN
  THE SYSTEM SHALL NOT select that candidate.
- IF the only otherwise-ready candidate is conflict-blocked THEN THE SYSTEM
  SHALL pause with category `SeamCollision` naming the in-flight conflicting
  spec.
- WHILE running in Autonomous mode THE SYSTEM SHALL still pause on
  `SeamCollision` (the category is non-promotable).
- WHEN a `conflicts-with` relation is parsed THE SYSTEM SHALL retain it as a
  distinct relation/edge and SHALL NOT degrade it to `related` (both whitelist
  sites updated).
- WHERE no priorities and no `conflicts-with` relations are present THE SYSTEM
  SHALL behave exactly as today — slug-order selection and unchanged
  `Remaining` / `Completed` ordering.
- THE SYSTEM SHALL cover the above with tests mirroring the existing
  structure-driven `scoreFn`-override fixture style in
  `internal/drive/check_test.go` and `internal/drive/needsme_test.go`.

## Completion Ledger

Delivered 2026-07-09. `go build ./...`, `go vet ./internal/drive/... ./internal/spec/...`,
and `go test ./...` (86 packages) all green. Cold audit: **SHIP (noteworthy)**, high
confidence — `.hero/planning/features/priority-conflict-aware-drive-selection/delivery-audit.md`.

### Acceptance Criteria

| Criterion | Status | Evidence |
|---|---|---|
| Select highest priority (then severity, then slug), not slug-first | DONE | `check.go` `rank`/`lessCandidate` + selection loop; `TestCheckSelectsHigherPriorityOverSlug` |
| Same on-disk state → same verdict | DONE | total comparator, slug final tiebreak; `TestCheckDeterministic` (20×) + `TestCheckDeterministicUnderInputPermutation` (3 slice orderings) |
| Candidate with delivering `conflicts-with` target NOT selected | DONE | `conflictDeliveringSlug` gate; `TestCheckConflictExcludesDeliveringCandidate` |
| Only-ready candidate conflict-blocked → pause `SeamCollision` naming in-flight spec | DONE | `check.go` seam fallback + `needsme.go` branch; `TestCheckSeamCollisionWhenOnlyCandidateConflicts` |
| Autonomous mode still pauses on `SeamCollision` (non-promotable) | DONE | plain `pause` (not `maybePromoted`); `TestNeedsMeSeamCollisionPausesEveryMode`, `TestPromotableScope` |
| `conflicts-with` retained as distinct edge, not degraded to `related` (both sites) | DONE | `spec.go:578` parser + `graph_ingest.go` edge; `TestParseRelations_ConflictsWith`, graph edge test |
| No priorities + no conflicts → behaves exactly as today | DONE | `TestCheckBackwardCompatSlugOrder`; existing drive tests still green |
| Tests mirror existing `scoreFn`-override fixture style | DONE | reused `mkChild`/`mkStub`/`mkInit`, pinned `scoreFn` |

### Changes

| # | Change | Status | Evidence |
|---|---|---|---|
| 1 | `check.go` — rank helper (critical=0…low=3, unset=99, case-insensitive) | DONE | `rank` shared by priority + severity |
| 2 | `check.go` — priority-aware candidate selection; lists stay slug-stable | DONE | selection loop picks min under `(priority,severity,slug)`; `Children`/`buildIntended`/`Remaining`/`Completed` slug-stable |
| 3 | `spec.go:578` — accept `conflicts-with`/`conflicts_with`, normalize | DONE | parser switch, `TestParseRelations_ConflictsWith` |
| 4 | `graph_ingest.go` — distinct `conflicts_with` edge, no fall-through | DONE | `graphEdgeForRelation` case + graph test |
| 5 | `needsme.go` — add `CategorySeamCollision` | DONE | const block w/ comment |
| 6 | `Promotable()` — keep SeamCollision non-promotable | DONE | default false branch; `TestPromotableScope` |
| 7 | `RunContext` `SeamBlocked`/`SeamConflictSlug` + `NeedsMe` plain-pause branch | DONE | branch bypasses `maybePromoted`; `TestNeedsMeSeamCollisionPausesEveryMode` |
| 8 | `check.go` — conflict exclusion + SeamBlocked-vs-Blocked distinction | DONE | `conflictDeliveringSlug` gate; `TestCheckConflictExcludesDeliveringCandidate` |
| 9 | Tests (Part A + Part B) + `DryRun` parity coverage | DONE | 7 drive tests + 2 spec tests + `TestDryRun{PreviewsHigherPriorityFirst,SeamCollisionPause}` |

Also delivered: `DryRun` mirrors the priority-aware, conflict-excluding selection via a shared
`seamReason` helper (`TestDryRunPreviewsHigherPriorityFirst`, `TestDryRunSeamCollisionPause`).

**Exercise-the-feature:** pure-function library change, no UI — behavior exercised directly through
`Check` / `DryRun` / `NeedsMe` / relation parsing. `go test ./...` → `86 packages ok, 0 failed`.

**Scoping decisions (v1):** outbound-only conflicts (honors a child's own `conflicts-with`, no
inbound scan — spec permits); local-delivering only via `IsLocallyDelivering` (peer-delivering is a
documented follow-up).
