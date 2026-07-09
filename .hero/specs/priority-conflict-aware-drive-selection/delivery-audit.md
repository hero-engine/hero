# Delivery audit — priority-conflict-aware-drive-selection

**Audited:** `git diff -- internal/` (uncommitted working tree; spec dir untracked)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] Highest-priority ready child selected (priority→severity→slug), not slug-first — `rank`/`lessCandidate` (check.go:66-92), selection loop (check.go:270-281); test `TestCheckSelectsHigherPriorityOverSlug` (check_test.go:174) asserts `zzz`(critical) beats `aaa`(low).
- [✓] Same on-disk state → same verdict — comparator is total + slug-terminated (check.go:82-91); test `TestCheckDeterministic` (check_test.go:243) re-invokes Check 20× and compares `NextSpec`+`Verdict`, and anchors the critical-tie resolves on slug (`bbb`). See note 1 — test is genuine but does not permute input order.
- [✓] Candidate with a delivering `conflicts-with` target is not selected — gate `conflictDeliveringSlug` (check.go:98-121) via `IsLocallyDelivering`; test `TestCheckConflictExcludesDeliveringCandidate` (check_test.go:196) picks `bbb` over conflict-excluded `aaa`.
- [✓] Only otherwise-ready candidate conflict-blocked → pause `SeamCollision` naming the in-flight spec — fallback distinction (check.go:281-296); test `TestCheckSeamCollisionWhenOnlyCandidateConflicts` (check_test.go:216) asserts category + reason contains `seam-peer`.
- [✓] Autonomous still pauses on `SeamCollision` (non-promotable) — plain `pause(...)` branch bypasses `maybePromoted` (needsme.go:186-188); `Promotable()` leaves it in the false default (needsme.go:64-72); tests `TestNeedsMeSeamCollisionPausesEveryMode` (needsme_test.go:168, sets `Promoted→true`, still pauses) + `TestPromotableScope` (needsme_test.go:158).
- [✓] `conflicts-with` retained as distinct relation/edge, both whitelist sites, no degrade to `related` — parser (spec.go:578, normalizes `conflicts_with`→`conflicts-with`) + `graphEdgeForRelation` (graph_ingest.go:228-232, distinct `conflicts_with` edge); tests `TestParseRelations_ConflictsWith` (spec_test.go:217, asserts `related` empty AND 2 conflicts targets — fails on parser revert) + `TestGraphEdgeForRelation_RelatesToMapsToEdge` (graph_ingest_test.go:247, asserts `conflicts_with` edge — fails on graph revert).
- [✓] No priorities + no conflicts → behaves as today incl. Remaining/Completed order — reporting kept slug-stable (check.go:244-246 comment + unchanged append order); test `TestCheckBackwardCompatSlugOrder` (check_test.go:275) asserts `NextSpec=alpha`, `Remaining=[alpha bravo]`, `Completed=[charlie]`.
- [✓] Tests mirror existing structure-driven fixture style — `withPriority`/`withConflict` helpers reuse `mkChild`/`mkInit` (check_test.go:163-172).

## Changes

- [✓] check.go — `rank` helper (total, case-insensitive, default 99) — check.go:66-77.
- [✓] check.go — priority-aware candidate selection (collect ready, pick min under `lessCandidate`) — check.go:249-281; `Children()`/`buildIntended`/Remaining/Completed left slug-stable.
- [✓] spec.go:578 — parser accepts `conflicts-with`/`conflicts_with`, normalizes to canonical.
- [✓] graph_ingest.go:228 — distinct `conflicts_with` edge, no fall-through to `related_to`.
- [✓] needsme.go — `CategorySeamCollision` const (needsme.go:62-65); kept out of `Promotable()` true-set (needsme.go:64-72).
- [✓] needsme.go — `RunContext.SeamBlocked` + `.SeamConflictSlug` (needsme.go:109-114); `seamReason` helper (needsme.go:143-145); plain-pause branch sibling of `Blocked` (needsme.go:186-188).
- [✓] check.go — conflict exclusion + `SeamBlocked`/`Blocked` fallback distinction off first-remaining child (check.go:281-296).
- [✓] DryRun — mirrors selection + conflict + seam logic via shared `lessCandidate`/`seamReason`/`conflictDeliveringSlug` (check.go:336-377). See note 2 — no test exercises the new DryRun paths.
- [✓] Tests — 7 new drive tests, 1 new + 1 extended spec test; `go build`/`go vet`/`go test` all exit 0 (verified fresh, `-count=1`).

## Audit notes

1. **Determinism test is genuine but narrow.** `TestCheckDeterministic` re-invokes Check 20× on identical input and compares output — it is not trivially passing (it anchors the critical-tie-on-slug outcome). It does **not** permute the order of the `all` slice. Input-order independence is nonetheless guaranteed by construction: selection picks the unique min under a strict total order whose terminal key (`slug`) is unique, and the whole selection/conflict path touches only slices (`intendeds`, `Relations`, `all`) — no map iteration. Verified by reading, not by the test. Low risk.

2. **DryRun new logic is untested.** The preview path gained priority-aware selection, the conflict gate, and a `SeamCollision` pause branch (check.go:336-377), but the three existing DryRun tests (`TestDryRunGuidedPreviewsThenDone`, `TestDryRunSupervisedStopsAtFirst`, `TestDryRunSurfacesAction`) predate this change and assert none of it. The code is correct by inspection and reuses the same helpers as the tested Check path (`lessCandidate`, `conflictDeliveringSlug`, `seamReason`), so divergence risk is low — but no test would catch a future regression in the DryRun preview. Not an AC requirement; flagged for the user's awareness.

3. **Conflict gate correctness confirmed.** Uses `IsLocallyDelivering()` (not a hand-rolled status check); self-conflict is skipped (`r.Target == ic.slug`, check.go:112); dangling target is a no-op via the `t != nil` guard in `specBySlugDrive` (check.go:116) — never panics. `SeamBlocked`-vs-`Blocked` distinction keys off `firstRem.ready(completed)`: a dep-blocked-only fallback reports `Blocked` (`TestCheckBlockedNotSeamWhenDepUnmet`), a ready-but-conflict-excluded fallback reports `SeamCollision`. Correct per spec design (distinction is scoped to the first-remaining child).

4. **NeedsMe branch reachability confirmed.** Check sets `ctx.NextScore = -1`; the score-underspecified branch guards on `NextScore >= 0` (needsme.go:195), so it cannot preempt the `SeamBlocked` branch (needsme.go:186), which sits ahead of it and ahead of `Blocked`. No performative DONE marks found.
