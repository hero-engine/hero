---
title: "Drive judge FindConflicts backstop — detect undeclared file overlaps against in-flight specs"
slug: drive-judge-findconflicts-backstop
type: enhancement
status: completed
size: medium
priority: high
domain: engineering
tags: [drive, judge, conflicts, index, sequencing, autonomy]
created: 2026-07-09
relates-to: [priority-conflict-aware-drive-selection, compose-emits-conflicts-and-priority]
completed_at: 2026-07-10T00:46:25Z
---

# Drive judge FindConflicts backstop — detect undeclared file overlaps against in-flight specs

## Kickoff

Adds a machine backstop to the `/drive` judge: when the judge picks a child to
deliver, it now also checks the index for **undeclared** file overlaps against
specs that are currently delivering, and pauses (softly) on any seam nobody
wrote down as a `conflicts-with`.

**Status:** planning — spec just landed, no code yet. Third and final piece of
the conflict-aware-drive chain (pieces 1 and 2 shipped).

**Pick up at:** extend `drive.Check`'s signature with a nil-safe `detect`
callback and add the promotable `CategorySeamDetected` category + `RunContext`
fields, then wire both callers (`hero goal --check`, MCP `hero_goal`) through a
shared `internal/driveio` detector seam backed by a delivering-filtered
`FindConflicts`. Start in `internal/drive/needsme.go` (category + Promotable +
ctx fields + NeedsMe branch), then `check.go` (signature + dedup), then the seam
and callers. nil callback must reproduce piece-1 behavior byte-for-byte.

→ `.hero/planning/features/drive-judge-findconflicts-backstop/spec.md`

**Files:** `internal/drive/check.go:229`, `internal/drive/needsme.go:51`,
`internal/index/index.go:1262`, `internal/cli/goal.go:103`,
`internal/serve/mcp_tools.go:1274`
**Skip:** don't write a second overlap detector (reuse `index.FindConflicts`);
don't make the detected gate hard/non-promotable (that re-introduces the
override tax); don't auto-author the missing `conflicts-with` from the judge
path (read-only verdict).

## Context

This is the third and final piece of a shipped chain that hardens `/drive`'s
autonomous next-spec selection against concurrent seam collisions:

- **Piece 1 — `priority-conflict-aware-drive-selection`** (completed, in
  `.hero/specs/`): taught the drive judge to **honor** authored `conflicts-with`
  relations and priority/severity ordering. It added the soft-mutex gate
  (`conflictDeliveringSlug`), the non-promotable `CategorySeamCollision`, and the
  `RunContext.SeamBlocked` / `SeamConflictSlug` signals. It scoped the gate to
  **local** delivering specs (`spec.IsLocallyDelivering`) and to **outbound**
  `conflicts-with` relations (v1).
- **Piece 2 — `compose-emits-conflicts-and-priority`** (planning, in
  `.hero/specs/`): teaches `/compose` to **write** those signals — reciprocal
  `conflicts-with` and priority — onto child stubs.

Both pieces depend on the **author** remembering to name every overlap seam.
That is the same forget-the-guardrail failure the chain set out to fix, just
moved one level up: if the human (or the composer) never declares that child B
touches the same file as in-flight spec A, the judge honors a relation that was
never written and happily runs them concurrently.

This spec closes that gap with a **machine backstop**. Hero already has a
whole-file overlap engine (`index.FindConflicts`) surfaced as `hero conflicts`.
We reuse it inside the judge: when the judge selects a candidate child, it asks
the index whether that candidate's files overlap any currently-delivering spec.
An overlap that is **not** already an authored `conflicts-with` becomes a
distinct, softer pause — so a seam nobody wrote down can't silently slip through,
but the judge doesn't grind to a halt on every incidental same-file touch.

## Goal

When the `/drive` judge picks a child to deliver, it detects whole-file overlaps
between that child and any locally-delivering spec via the existing
`index.FindConflicts` engine, injected into `drive.Check` as a nil-safe
callback. An overlap that is **not** already declared as an authored
`conflicts-with` pauses the run under a new, promotable `CategorySeamDetected`
that names the overlapping file(s) and the in-flight spec. The pause is softer
than the authored `CategorySeamCollision`: it pauses in Supervised/Guided but is
promotable in Autonomous mode. When the detector callback is nil (tests, cold
paths), `Check` behaves byte-for-byte as it did after piece 1 — same on-disk
state produces the same verdict. Both callers (`hero goal --check` and MCP
`hero_goal`) wire the same detector and produce identical verdicts; the existing
parity test stays green.

## Approach

### The central purity problem, and the injection that solves it

`index.FindConflicts` reads the SQLite index DB. `drive.Check` /
`drive.NeedsMe` are **pure, cold-startable** functions — determinism is a hard
constraint inherited from piece 1 (`check.go:17-19`, `needsme.go:1-8`). All I/O
is pre-computed by the caller and handed into the pure predicate via
`RunContext`. So `FindConflicts` **cannot** run inside `Check`.

The clean fit — and the design we adopt — mirrors the injection pattern already
in `Check`'s signature (`promoted func(PauseCategory) bool`): **extend `Check`
with an injected, nil-safe detector callback.**

```go
// DetectedConflict is one delivering spec whose files overlap the candidate.
// Lives in package drive so Check stays decoupled from internal/index.
type DetectedConflict struct {
    Slug  string   // the in-flight (delivering) spec
    Files []string // the overlapping file paths (deterministic order)
}

func Check(
    init *spec.Spec,
    all []*spec.Spec,
    promoted func(PauseCategory) bool,
    detect func(candidateSlug string) []DetectedConflict, // nil = today's behavior
) CheckResult
```

- The two real callers supply a `detect` closure backed by
  `index.FindConflicts`, filtered to **delivering** status.
- Tests and any cold path pass `nil`. A nil callback means the detected gate is
  skipped entirely → the verdict is byte-for-byte identical to piece 1. This is
  the backward-compat and determinism anchor.

**Reconciling the asymmetry (state it plainly):** the *authored* gate stays
**inside** `Check` — it is relation-based, computed over the `all` slice via
`conflictDeliveringSlug` + `IsLocallyDelivering`, and needs no external I/O. The
*detected* gate is **injected** via the callback because it needs the index.
This keeps `Check` deterministic (the callback is a pure function of on-disk
index state), preserves cold-start (nil callback = unchanged), and reuses the
established injection convention rather than coupling `drive` to `internal/index`.

> **Refinement over the original sketch:** the callback returns
> `[]DetectedConflict` (slug **plus** overlapping files), not just `[]string` of
> slugs. The acceptance criteria require the pause to *name the overlapping
> file(s)* — so the files must travel with the slug. `FindConflicts` already
> returns `OverlappingFiles` per result, so this is free.

### HARD vs SOFT — the real judgment call (decision: distinct SOFTER category)

Authored `conflicts-with` is a **deliberate** human statement of intent. Piece
1's hard, non-promotable `CategorySeamCollision` is correct for it: even
Autonomous mode pauses, because the human explicitly said "these must not run
together."

Detected file-overlap is **heuristic** and **whole-file-granular**. Two specs
editing different functions of the same large file (`mcp_tools.go`, say — a
3000-line file already touched by many specs) will false-positive. A **hard**
pause on every detected same-file touch would re-introduce a manual-override tax
in the *opposite* direction from the one the chain fixed: instead of silently
running colliding work, the judge would pause on noise, and an autonomous run
would stall constantly. That is exactly what an autonomous run does not want.

**Decision:** add a **distinct, softer** category, `CategorySeamDetected`, that:

1. Names the overlapping file(s) and the in-flight spec in the pause reason.
2. Pauses in Supervised and Guided modes (Supervised pauses at every boundary
   regardless; Guided pauses on all taxonomy categories).
3. **Is promotable** in Autonomous mode — so the learning layer can let a user
   who has seen enough false positives auto-proceed. This is the deliberate
   contrast with the non-promotable authored `CategorySeamCollision`.

Wiring: add `CategorySeamDetected` to the `PauseCategory` const block, add it to
the **true set** of `Promotable()` (opposite of `SeamCollision`), add
`RunContext` field(s) for the detected slug + files, and a `NeedsMe` branch
routed through `maybePromoted` (so Autonomous can promote it).

*Justification (one paragraph):* The authored gate encodes intent, so it is a
real obstacle and must never be relaxed — hard and non-promotable is right.
The detected gate encodes a *suspicion* derived from whole-file granularity,
which is inherently noisy; treating a suspicion as a hard obstacle trades one
silent-failure mode (unflagged collisions) for a louder failure mode (constant
false pauses that make Autonomous mode useless). A distinct promotable category
lets the signal surface every time by default, while giving the learning layer a
per-user escape valve once the human confirms the noise is tolerable — which is
precisely the division of labor the promotion machinery was built for.

### Where the detected gate runs relative to selection (design B)

The authored gate excludes conflicting candidates from the `ready` set *during
selection* (`check.go:261`), and only surfaces as a pause via the
first-remaining fallback (`check.go:284-300`). The detected gate does **not**
work that way. It must not silently steer selection away from overlapping work,
because that would hide the signal and defeat the "surface it" goal.

Instead: run detection on the **already-selected** candidate. After `Check`
picks `nextI` (its normal priority/severity/slug selection), it calls
`detect(nextI.slug)`, dedups the result against the candidate's authored
`conflicts-with` targets, and if any overlap remains, sets
`ctx.SeamDetected = true` plus the slug/files. `NeedsMe` then evaluates the
pause. This keeps selection unchanged (the authored gate still owns
selectability) and layers detection as a post-selection check that feeds a
promotable pause.

### Authored wins; dedup (structural invariant + explicit subtraction)

When both gates would fire for the same in-flight target, the authored hard
`SeamCollision` takes precedence — detected is only for overlaps **not** already
declared.

Two facts make this clean:

1. **Structural invariant:** a candidate that reaches selection (is in `ready`)
   has *no* authored `conflicts-with` target currently delivering — the authored
   gate already excluded it (`check.go:261`). So by construction, a
   *selected* candidate's detected overlaps are never also authored-and-
   delivering. Where the authored gate *does* fire (first-remaining fallback,
   nothing selectable), `NeedsMe` checks `ctx.SeamBlocked` (authored) **before**
   `ctx.SeamDetected`, so the authored `SeamCollision` is emitted and the
   detected branch is never reached for that candidate.
2. **Explicit subtraction (belt-and-suspenders):** `Check` still filters the
   detector's returned overlaps to exclude any whose slug is in the candidate's
   authored `conflicts-with` target set, so the guarantee does not rely solely
   on the selection invariant. The dedup happens **in `Check`**, because `Check`
   already knows the candidate's authored relations — the callback returns raw
   overlaps, `Check` subtracts the authored ones and applies `SeamDetected` to
   the remainder.

### Detector home — the `internal/driveio` seam (decision)

Both callers (`internal/cli/goal.go`, `internal/serve/mcp_tools.go`) must build
the *same* `detect` closure, and the parity test demands they stay identical.
The logic is: `FindConflicts(slug)` → keep only delivering-status results → map
to `[]drive.DetectedConflict`.

**Decision: a new tiny `internal/driveio` seam.** It imports both `drive` and
`index` and exposes a single builder, e.g.
`func Detector(idx *index.DB) func(string) []drive.DetectedConflict`. Both
callers use it, so the FindConflicts→delivering-filter→map logic lives in
exactly one place and the two callers are parity-identical by construction.

Why this over the alternatives:

- **Put the adapter in `internal/drive`** → makes `drive` import `index`,
  defeating the whole point of the callback (keeping the pure predicate package
  free of the sqlite-backed index dependency). Rejected.
- **Put it in `internal/index`** → makes low-level infrastructure import
  higher-level policy (`drive`); a layering smell, and index is meant to stay
  infrastructure-only. Rejected.
- **Inline the closure in each caller** → duplicates the filter+map in two
  packages, and the only guard against drift is the parity test. Workable but
  fragile. Rejected in favor of the seam.

The seam keeps `drive` pure (no `index` import), keeps `index` infrastructure-
only (no `drive` import), and centralizes the shared logic. New-package overhead
is one small file.

The delivering-filter itself is cleanest as a new index method
`func (idx *DB) FindDeliveringConflicts(slug string) ([]ConflictResult, error)`
(reuses the `FindConflicts` SQL but narrows the status `IN` clause to
`'delivering'`), since `FindConflicts` today returns
`planning|in-review|delivering` (`index.go:1270`) and we need delivering-only to
match the authored gate's `IsLocallyDelivering` scope. The `driveio.Detector`
builder calls it and maps the results.

### Pre-existing caller asymmetry to be aware of (do not widen)

The MCP `hero_goal` path currently calls `drive.Check(init, all, nil)`
(`mcp_tools.go:1274`) — it passes `nil` for `promoted`, unlike the CLI which
passes `promo.IsPromoted` (`goal.go:103`). That promotions gap is **out of scope
here**; do not fix it as a side effect. But note: to wire the detector, the MCP
handler needs an `*index.DB` in scope (it already uses `idx` elsewhere in
`mcp_tools.go`), and the CLI goal handler currently has **no** index handle and
must open one. Both wire the *same* `driveio.Detector`, so their detected-gate
verdicts stay identical even though their promotions wiring differs today.

## Changes

1. **`internal/drive/needsme.go` — new category, promotability, context fields,
   pause branch.**
   - Add `CategorySeamDetected PauseCategory = "SeamDetected"` to the const block
     (`needsme.go:51-66`), with a comment contrasting it against
     `CategorySeamCollision`: detected = heuristic whole-file overlap not
     authored as a relation; soft; **promotable**.
   - Add `CategorySeamDetected` to the **true** set in `Promotable()`
     (`needsme.go:72-79`) — alongside `DesignFork`, `Underspecified`,
     `AmbiguousPick`. (Contrast: `SeamCollision` stays in the default/false set.)
   - Add `RunContext` fields (`needsme.go:88-134`): `SeamDetected bool`,
     `SeamDetectedSlug string`, `SeamDetectedFiles []string`. Document that these
     name a detected overlap not already declared as `conflicts-with`.
   - Add a `NeedsMe` branch in the taxonomy section (`needsme.go:178-201`),
     placed **after** the `ctx.SeamBlocked` (authored, hard) check and routed
     through `maybePromoted(CategorySeamDetected, ...)` so Autonomous can promote
     it. Ordering guarantees authored wins.
   - Add a `seamDetectedReason(candidate, conflictSlug string, files []string)`
     helper next to `seamReason` (`needsme.go:143`) that names the candidate, the
     in-flight spec, and the overlapping file(s) (and a count if >1). Deterministic.

2. **`internal/drive/check.go` — signature, detection call, dedup.**
   - Extend `Check`'s signature (`check.go:229`) with
     `detect func(candidateSlug string) []DetectedConflict` as the final param.
     Document nil = piece-1 behavior.
   - Define the `DetectedConflict` struct (`Slug string`, `Files []string`) in
     package drive (top of `check.go` or a small `detect.go` in the package,
     **no** `index` import).
   - After the candidate `nextI` is chosen (`check.go:279-281`), and only when
     `detect != nil` and `nextI` came from the normal `ready` selection (not the
     firstRem authored-fallback), call `detect(nextI.slug)`, subtract the
     candidate's authored `conflicts-with` target slugs (dedup), and if any
     overlap remains set `ctx.SeamDetected`, `ctx.SeamDetectedSlug` (deterministic
     first), `ctx.SeamDetectedFiles`.
   - Add a small helper `authoredConflictTargets(ic *intended) map[string]bool`
     (or reuse the relation walk from `conflictDeliveringSlug`) so the dedup set
     is computed once.
   - **`DryRun` stays authored-only for v1** (`check.go:329`): do **not** wire
     the detector into `DryRun`. `DryRun` optimistically simulates children
     finishing (`check.go:391`), and the index-backed detector cannot reflect
     those simulated completions — wiring it would produce a misleading preview.
     Document this as a boundary; detected-overlap preview in dry-run is a
     follow-up. `DryRun`'s signature is unchanged.

3. **`internal/index/index.go` — delivering-filtered overlap query.**
   - Add `func (idx *DB) FindDeliveringConflicts(slug string) ([]ConflictResult, error)`
     next to `FindConflicts` (`index.go:1262`): same query shape, but the status
     `IN` clause is narrowed to `('delivering')` (vs.
     `('planning','in-review','delivering')` at `index.go:1270`). Reuses
     `ConflictResult` (`index.go:1310`). No second detector — this is a scoped
     variant of the existing engine.

4. **`internal/driveio/detector.go` — new shared seam.**
   - New package `internal/driveio` exposing
     `func Detector(idx *index.DB) func(string) []drive.DetectedConflict`. The
     returned closure calls `idx.FindDeliveringConflicts(slug)`, maps each
     `ConflictResult` to `drive.DetectedConflict{Slug, Files: OverlappingFiles}`,
     and returns them (empty/nil on none or on query error — a detector failure
     must not crash the run; log/swallow consistent with existing
     `conflicts, _ := idx.FindConflicts(...)` usage at `mcp_tools.go:2647`).
   - This is the single source of the filter+map logic both callers share.

5. **`internal/cli/goal.go` — wire the detector into `hero goal --check`.**
   - In the `goalCheck` case (`goal.go:98-107`), open an `*index.DB` (the handler
     has `heroDir`; follow the index-open pattern used elsewhere in `internal/cli`),
     build `driveio.Detector(idx)`, and pass it as the new `Check` arg:
     `drive.Check(init, all, promo.IsPromoted, driveio.Detector(idx))`.
   - Leave `drive.DryRun(...)` (`goal.go:113`) unchanged (authored-only preview).

6. **`internal/serve/mcp_tools.go` — wire the detector into MCP `hero_goal`.**
   - In the `check` case (`mcp_tools.go:1274`), build `driveio.Detector(idx)`
     from the `idx` already available in the file and pass it:
     `drive.Check(init, all, nil, driveio.Detector(idx))`. (Leave the `nil`
     promoted arg as-is — the promotions gap is out of scope.)
   - Leave the `dryRun` case (`mcp_tools.go:1276`) unchanged.

7. **Tests** (mirror existing fixture styles):
   - `internal/drive/check_test.go` + `internal/drive/needsme_test.go`: use the
     existing structure-driven approach (`scoreFn`-override pattern) and inject a
     **fake** `detect` callback (a plain Go func returning canned
     `[]DetectedConflict`) — no index needed. New cases:
     - detected-pause naming (reason contains the overlapping file(s) + in-flight
       slug);
     - promotable-in-Autonomous (promoted → proceed; Guided → pause; Supervised →
       pauses as boundary);
     - authored-wins dedup (a candidate whose detected overlap is also its
       authored `conflicts-with` target emits `SeamCollision` once, never
       `SeamDetected`);
     - nil-callback determinism / backward-compat (same fixture, `detect == nil`,
       identical verdict to piece 1).
   - `internal/index/index_test.go`: add coverage for `FindDeliveringConflicts`
     mirroring `TestFindConflicts` (`index_test.go:237`) — assert planning/
     in-review specs are excluded and delivering ones included.
   - `internal/serve/mcp_test.go`: extend the parity fixture (`mcp_test.go:1133`)
     with an undeclared overlap so both callers, with the detector wired, produce
     the same `SeamDetected` verdict — guarding caller parity.

## Acceptance Criteria

- WHEN the injected detector reports an in-flight (delivering) spec whose files
  overlap the selected candidate AND that overlap is not already an authored
  `conflicts-with`, THE SYSTEM SHALL pause with `SeamDetected`, naming the
  overlapping file(s) and the in-flight spec.
- WHERE the run is Autonomous AND `SeamDetected` has been promoted, THE SYSTEM
  SHALL proceed past a `SeamDetected` pause (it is promotable), in contrast to
  the non-promotable authored `SeamCollision`.
- IF an overlap is already declared as an authored `conflicts-with`, THEN THE
  SYSTEM SHALL report it once as `SeamCollision` (authored wins) and SHALL NOT
  also emit `SeamDetected` for it.
- WHERE the detector callback is nil, THE SYSTEM SHALL behave byte-for-byte as
  piece 1 — the same on-disk state produces the same verdict.
- THE SYSTEM SHALL reuse the `index.FindConflicts` engine (via a delivering-
  filtered variant) and SHALL NOT introduce a second overlap detector.
- THE SYSTEM SHALL scope detected overlaps to locally-delivering specs only,
  consistent with the authored gate's `IsLocallyDelivering` scope.
- WHEN either caller (`hero goal --check` or MCP `hero_goal`) runs `Check` with
  the detector wired, THE SYSTEM SHALL produce identical verdicts for the same
  on-disk state (the existing parity test stays green).
- WHILE previewing via `DryRun`, THE SYSTEM SHALL evaluate the authored gate only
  and SHALL NOT apply the detected gate (v1 boundary).

## Boundaries

- Do **not** change the shipped authored `conflicts-with` semantics, and do
  **not** re-open piece 1's outbound-only v1 decision.
- Do **not** break `Check`'s purity or cold-start determinism — a nil callback
  must equal today's behavior exactly.
- Reuse `FindConflicts`; introduce **no** second overlap detector.
- No `wave` ordinal or any new sequencing hierarchy.
- **No auto-authoring** of the missing reciprocal `conflicts-with` from the judge
  path — `hero goal --check` is a read-only verdict, and mutating specs as a side
  effect of a verdict is wrong. (Right homes for auto-authoring are noted under
  Future work, out of scope here.)
- **Local-delivering only** for v1 — peer-delivering specs are not consulted by
  the detected gate (consistent with the authored gate). Peer scope is a
  follow-up.
- `DryRun` stays authored-only; detected-overlap preview in dry-run is a
  follow-up.
- Do **not** widen the pre-existing MCP `promoted: nil` gap (`mcp_tools.go:1274`).

### Future work (explicitly out of scope)

- Auto-authoring the missing reciprocal `conflicts-with` on detection so the next
  run sees it declared and reviewable — the right homes are a `hero conflicts
  --fix` writer or a compose-time check, **not** the judge path.
- Extending the detected gate to peer-delivering specs.
- Wiring the detector into `DryRun` (needs a way to reflect simulated
  completions, or an explicit "as-of-now" preview caveat).
- Closing the MCP `hero_goal` promotions gap.

## Risks

- **Whole-file false positives.** The overlap engine is file-granular; two specs
  editing different regions of the same large file will trigger `SeamDetected`.
  This is *by design* handled via the softer, promotable category — but the
  pause reason must be clear enough that a human immediately sees it's a same-
  file, maybe-not-real overlap. Keep the reason concrete (name the file).
- **Determinism regression.** The detector must be a pure function of on-disk
  index state. The nil-callback determinism test is the guard; keep it. Any
  nondeterminism (map iteration in the map→slice step) must be eliminated —
  `FindConflicts` already `ORDER BY s.slug, ft2.file_path`, so preserve that
  order end-to-end.
- **Caller parity drift.** Two callers building the closure independently could
  diverge. The `driveio.Detector` seam plus the extended parity fixture mitigate
  this; do not inline the filter+map in either caller.
- **Index freshness.** `FindConflicts` reads `files_touched`, populated from each
  spec's `## Changes` section (`spec.go:189,431`). A child only has a file
  footprint once **designed**; a bare stub has none. So the detector is effective
  for designed-but-not-yet-delivered children and **weak for undesigned stubs**.
  This is acceptable: by the time the judge would route a child to **deliver**
  (as opposed to design), it is designed and has a footprint. State this in
  the delivered code's doc comment so the limitation isn't rediscovered.
- **New package.** `internal/driveio` is new; keep it to a single small file with
  one exported builder to avoid it becoming a junk drawer.

## Validation

- `go test ./internal/drive/...` — new detected-pause, promotable-in-Autonomous,
  authored-wins dedup, and nil-callback determinism cases pass; all existing
  drive tests stay green.
- `go test ./internal/index/...` — `FindDeliveringConflicts` excludes planning/
  in-review and includes delivering.
- `go test ./internal/serve/...` — the extended `hero_goal` parity fixture
  (`mcp_test.go:1133`) passes with an undeclared overlap producing `SeamDetected`
  on both surfaces.
- `go test ./internal/cli/...` — `hero goal --check` emits a `SeamDetected` pause
  for a fixture with an undeclared overlap against a delivering spec, and does
  **not** emit one when the overlap is authored as `conflicts-with` (that stays
  `SeamCollision`).
- Manual: construct two specs sharing a file in `## Changes`, set one to
  `delivering`, run `hero goal --check` on an initiative that would select the
  other → observe a `SeamDetected` pause naming the file and the delivering spec.
  Add the reciprocal `conflicts-with` → observe it becomes `SeamCollision`
  (authored wins), reported once.

## Implementation Notes — exact touch-points

- **Overlap engine (reuse, don't duplicate):**
  `internal/index/index.go:1262` — `FindConflicts(slug)`; `:1310` —
  `ConflictResult{Slug, Title, Type, Status, Path, ClaimedBy, OverlappingFiles}`.
  Note: `FindConflicts` returns `planning|in-review|delivering` (`:1270`); add
  `FindDeliveringConflicts` narrowing to delivering-only.
- **File footprint source:** `internal/spec/spec.go:189` — `Spec.FilesTouched`;
  `:431` — populated from the `## Changes` section. (Limitation: stubs have no
  footprint until designed.)
- **Pure judge (signature + RunContext build + dedup):**
  `internal/drive/check.go:229` — `Check(init, all, promoted, detect)`;
  candidate selection at `:272-281`; RunContext construction at `:283`;
  authored-fallback at `:284-300`; `DryRun` at `:329` (unchanged).
- **Category + Promotable + RunContext + NeedsMe branch:**
  `internal/drive/needsme.go:51-66` (const block; add `CategorySeamDetected`),
  `:72-79` (`Promotable()` true-set), `:88-134` (`RunContext` fields), `:143`
  (`seamReason` neighbor for `seamDetectedReason`), `:178-201` (taxonomy branch,
  after `SeamBlocked`, via `maybePromoted`).
- **Caller wirings (both, guarded by parity test):**
  `internal/cli/goal.go:103` — `hero goal --check`; `internal/serve/mcp_tools.go:1274`
  — MCP `hero_goal` (currently passes `nil` promoted — leave as-is);
  `internal/serve/mcp_test.go:1133` — parity fixture to extend.
- **New shared seam:** `internal/driveio/detector.go` —
  `Detector(idx) func(string) []drive.DetectedConflict`.
- **Existing engine coverage reference:** `internal/index/index_test.go:237` —
  `TestFindConflicts` (mirror for `FindDeliveringConflicts`).

## Completion Ledger

Delivered 2026-07-09. Piece 3 of the conflict-aware-drive chain. `go build ./...`,
`go vet` (drive/index/driveio/cli/serve), and `go test ./...` all green. Purity
invariant verified: `go list -deps ./internal/drive/` → **0** `internal/index`
imports (the `driveio` seam keeps `drive` index-free). Cold audit: see
`delivery-audit.md`.

### Acceptance Criteria

| Criterion | Status | Evidence |
|---|---|---|
| Undeclared delivering overlap → `SeamDetected` naming file(s) + in-flight spec | DONE | `check.go` detect+dedup, `needsme.go` branch + `seamDetectedReason`; `TestCheckSeamDetectedPauseNamesOverlap`, CLI `TestGoalCheckDetectsUndeclaredSeam` (end-to-end, live sqlite) |
| Autonomous+promoted proceeds past `SeamDetected` (promotable) | DONE | `Promotable()` true-set + `maybePromoted` branch; `TestCheckSeamDetectedPromotableAcrossModes`, `TestNeedsMeSeamDetectedPromotable` |
| Authored `conflicts-with` → `SeamCollision` once, never also `SeamDetected` | DONE | structural exclusion + `authoredConflictTargets` subtraction; `TestCheckAuthoredWinsOverDetected`, CLI `TestGoalCheckAuthoredConflictStaysSeamCollision` |
| nil detector → byte-for-byte piece-1 verdict | DONE | detect block guarded by `detect != nil`; `TestCheckNilDetectorMatchesPiece1` (10-run determinism) |
| Reuse `FindConflicts`; no second detector | DONE | `FindDeliveringConflicts` shares private `findConflicts(slug, statuses...)` helper with `FindConflicts` |
| Scope to locally-delivering only | DONE | `findConflicts(slug, "delivering")`; `TestFindDeliveringConflicts` (planning/in-review excluded) |
| Both callers wired → identical verdicts (parity green) | DONE | both use `driveio.Detector(idx)`; `TestMCP_ToolGoal_CheckParity` green + `TestMCP_ToolGoal_CheckDetectsUndeclaredSeam` |
| DryRun authored-only, no detected gate | DONE | DryRun signature unchanged + boundary comment; existing DryRun tests green |

### Changes

| # | Change | Status | Evidence |
|---|---|---|---|
| 1 | `needsme.go` — category, Promotable, ctx fields, branch, reason | DONE | `CategorySeamDetected` + Promotable true-set + 3 RunContext fields + `seamDetectedReason` + NeedsMe branch after SeamBlocked via `maybePromoted` |
| 2 | `check.go` — signature, `DetectedConflict`, detection, dedup, DryRun comment | DONE | struct (no index import), final `detect` param, post-selection block, `authoredConflictTargets`, DryRun untouched |
| 3 | `index.go` — `FindDeliveringConflicts` | DONE | sibling to `FindConflicts`, DRY via shared `findConflicts`; preserves `ORDER BY s.slug, ft2.file_path` |
| 4 | `internal/driveio/detector.go` — new shared seam | DONE | single-file package, `Detector(idx)` filter+map, nil on error |
| 5 | `goal.go` — wire detector into `hero goal --check` | DONE | index.Open + `driveio.Detector(idx)`; DryRun path unchanged |
| 6 | `mcp_tools.go` — wire detector into MCP `hero_goal` | DONE | detector wired; `nil` promoted left as-is (gap out of scope); dryRun unchanged |
| 7 | Tests (drive, index, serve, CLI) | DONE | 5 drive Check cases + NeedsMe case + Promotable update + `FindDeliveringConflicts` + MCP seam test + 2 CLI end-to-end |

### Exercise-the-feature check

- [x] End-to-end via the real `hero goal drive --check` cobra command against on-disk
  fixtures with a live sqlite index: `TestGoalCheckDetectsUndeclaredSeam` observes
  `verdict: pause` / `category: SeamDetected` naming the in-flight spec + overlapping
  file; the authored variant observes `SeamCollision`. `TestMCP_ToolGoal_CheckDetectsUndeclaredSeam`
  exercises the same through MCP dispatch. Both green. Purity invariant test confirms
  `internal/drive` does not import `internal/index`.
