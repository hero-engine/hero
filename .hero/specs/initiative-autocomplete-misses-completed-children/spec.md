---
title: "Initiative auto-complete misses already-completed children — no standalone parent re-check strands every finished initiative"
slug: initiative-autocomplete-misses-completed-children
type: bug
status: completed
severity: high
priority: high
size: medium
domain: engineering
root_cause_class: design
created: 2026-07-09
tags: [verify, initiative, autocomplete, lifecycle, completion, reconcile, status-drift]
relations:
  - target: cst-initiative-premature-autocomplete
    kind: related
  - target: flat-named-spec-discovery
    kind: related
completed_at: 2026-07-10T01:32:37Z
---

# Initiative auto-complete misses already-completed children

## Symptom

An initiative whose **every declared child is `completed` and archived** does not
flip to `status: completed`. It stays `planning` forever, with no command that
recovers it.

Two live, on-disk reproductions in this repo:

1. **`content-remediation`** (`.hero/planning/initiatives/content-remediation/spec.md`)
   — declares 8 children in a block-style `child:` list. All 8 now resolve to
   `status: completed` specs archived under `.hero/specs/**`. The initiative is
   still `status: planning`. Its own Progress note records that
   `autoCompleteParentIfReady` did not fire after the last child
   (`token-efficiency-pass`) verified, and did not fire after a
   `hero check --reconcile` pass either — root cause was left undiagnosed there.

2. **`cold-start-trust-hardening`** (`.hero/planning/initiatives/cold-start-trust-hardening/spec.md`)
   — inconsistent split state: frontmatter has `completed_at: 2026-06-23T19:57:04Z`
   set while `status: planning`. This is a **secondary** defect (see below), not a
   second instance of the primary bug.

**Impact.** Every completed initiative silently strands in `planning`. Rollups,
velocity, and any "what's shipped" view undercount; the graph's initiative→child
completion state is wrong; and `/drive` / snapshot logic that keys off initiative
status reads a false picture. There is no recovery short of a hand-edit — which
the tooling is explicitly supposed to own.

## Root Cause

**Classification: `design`** (a one-shot trigger with no idempotent re-check),
with a secondary `code`/`data` element (status ↔ `completed_at` can split with no
repair path).

### The definitive, reproduced finding

`autoCompleteParentIfReady` (`internal/cli/verify.go:624`) is **only reachable as
an in-process side-effect of verifying a not-yet-completed child**. It has exactly
one caller — `runVerify` at `internal/cli/verify.go:182` — and that call happens
only after a child spec passes its own delivery gates in the *same* `hero spec
verify` process (guarded by `result.Result == "PASS" || "FORCED"` at
`verify.go:174`).

The enumeration/roster logic inside the function is **not** the blocker. I proved
this by running the function's exact discovery + roster + child-count logic
against the live `.hero` tree (read-only, via a throwaway test since deleted):

```
parent found: slug=content-remediation type=initiative status=planning
parent.Relations (11): ... 8 with kind="child" ...
declaredCount=8 declaredComplete=true
childCount(via parent rel)=8 allDone=true
```

- The block-style `child:` list **does** parse into `parent.Relations` with
  `Kind:"child"` — `internal/spec/spec.go:613-641` lists `child` among the
  relation-producing keys and falls back to `parseScalarListBlock`
  (`spec.go:795`) for the newline `- item` form. (This was fixed under
  `cst-relation-frontmatter-fail-loud`; it is working.)
- All 8 children carry a `{target: content-remediation, kind: parent}` relation
  and are `completed`.
- So the roster gate (`verify.go:660-673`: `declaredCount>0 && !declaredComplete`)
  and the child-count gate (`verify.go:675-691`: `childCount==0 || !allDone`)
  both **pass right now**. If `autoCompleteParentIfReady` were invoked today with
  any child as `target`, it would complete and archive `content-remediation`.

**Nothing invokes it.** That is the bug:

- The only trigger is verifying a child that is *not yet* completed. Once all
  children are completed **and archived**, re-running `hero spec verify <child>`
  hits the early return at `verify.go:103-111`
  (`target.Status == StatusCompleted && isAlreadyInSpecsDir(...)` → print
  "already completed and archived" → `return nil`) **before** reaching line 182.
  So the trigger cannot be re-fired by re-verifying a child.
- `hero spec verify <initiative>` does not help: an initiative in `planning` hits
  the pre-delivery lifecycle guard at `verify.go:118` and refuses to run; even
  with `--force` it would run *delivery gates on the initiative itself*
  (Completion Ledger + audit report on the initiative), and would **never**
  consult children — `autoCompleteParentIfReady` only ever inspects the verified
  spec's *parent*, never its children.
- `hero check --reconcile` does not help: the reconcile package
  (`internal/reconcile/reconcile.go`) only (a) moves a `completed`-but-stuck-in-
  `planning/` spec to `specs/`, and (b) promotes `planning → delivering` on git
  evidence. Its `StatusDelivering` branch explicitly declines to auto-complete
  ("auto-complete is dangerous, so we skip this for now", `reconcile.go:143-145`).
  It never derives an initiative's completion from its children.

**Conclusion:** parent auto-complete has a single, one-shot, in-process trigger
and **no idempotent standalone re-check**. If that trigger is missed at the exact
instant the last child completes — for *any* reason — the initiative is stranded
permanently with no recovery command.

### Why the trigger was missed for `content-remediation` (contributing, not the assigned defect)

Because the logic succeeds against today's tree, the miss must have been a
**transient discovery/timing condition at the moment `token-efficiency-pass`
verified** — e.g. a sibling child that was momentarily not discoverable (the
open, distinct `flat-named-spec-discovery` bug: "flat-named spec files are
invisible to discovery — verify can't resolve initiative children"), or a sibling
not yet `completed`, making `declaredComplete=false` and the roster gate `continue`
at that instant. That transient cause is **not** this bug and must not be folded
in. This bug is that a *transient* miss becomes a *permanent* strand because there
is no re-check. The fix must hold regardless of why the first trigger was missed.

### Secondary defect — `completed_at` / `status` split (`cold-start-trust-hardening`)

The canonical writer keeps the two together: `updateFrontmatterStatus`
(`internal/cli/complete.go:300-312`) stamps `completed_at` in the *same* write
when it sets `status: completed`. The split on `cold-start-trust-hardening`
(`completed_at` set, `status: planning`) is the residue of the **old**
`cst-initiative-premature-autocomplete` bug: that bug auto-completed the whole
initiative (stamping `completed_at` + archiving), and it was then "restored
manually" (see its Progress note, `cold-start-trust-hardening/spec.md:160`). The
manual un-complete reset `status` to `planning` but left `completed_at` behind.

The real, still-present gap this exposes: **there is no supported "reopen" that
clears `completed_at` when a spec leaves `completed`, and no invariant check that
flags `completed_at set && status != completed`.** So any future manual restore —
or a half-applied auto-complete that errors between the status write
(`verify.go:476` → `complete.go:88`) and the archive — leaves a silent split that
nothing repairs.

## Suggested Fix Approach

Two coordinated changes; keep them surgical. **Do not hand-edit any initiative's
`status:` frontmatter** — `hero spec verify` (and the reconcile path below) must
own the transition.

### 1. Add an idempotent standalone parent re-check (primary)

Extract the child-aware completion decision so it can run **without** a child
being verified in-process, and wire it into `hero check --reconcile`.

- **New surface:** a reconcile-time check that, for every initiative in
  `planning`/`delivering`, runs the same roster + child-count evaluation currently
  living in `autoCompleteParentIfReady` (`verify.go:656-691`) and, when satisfied,
  completes + archives the initiative via the existing `completeAndArchive`
  (`verify.go:467`). This makes completion recoverable and idempotent, not a
  one-shot side-effect.
  - Reuse the existing gate logic verbatim — it is already correct (proven above).
    Factor `verify.go:656-691` into a shared helper (e.g.
    `initiativeReadyToComplete(parent, allSpecs) bool`) called by both
    `autoCompleteParentIfReady` and the new reconcile check, so the two paths can
    never diverge.
  - Prefer surfacing it in `internal/reconcile/reconcile.go` so `hero check` shows
    it as a finding and `hero check --reconcile` applies it (consistent with how
    "completed-but-stuck-in-planning" is already handled at
    `reconcile.go:27-36,73-79`). The `StatusDelivering`/initiative branch
    (`reconcile.go:143-145`) is the natural home — replace the "skip for now" note
    with the child-roster-driven completion (initiatives only; still never
    auto-complete a leaf feature/bug from git evidence).

- **Also remove the re-trigger dead end (optional but recommended):** the early
  return at `verify.go:103-111` for an already-completed+archived child could,
  before returning, run the parent re-check so that re-verifying any child of a
  stranded initiative repairs it. If the reconcile path above lands, this is
  belt-and-suspenders; pick one primary surface and note the other.

### 2. Enforce the `status` ↔ `completed_at` invariant (secondary)

- In the reconcile pass, add a finding for `completed_at` set while
  `status != completed`: either clear `completed_at` (if the spec genuinely
  reopened) or, when a re-check now completes the initiative, let the standard
  completed-write reconcile the pair. Do not silently leave the split.
- Ensure any "reopen"/status-away-from-completed write path clears `completed_at`
  in the same write (mirror of `updateFrontmatterStatus`'s stamp-on-complete at
  `complete.go:308-310`).

### 3. Reconcile the two already-stranded initiatives (as part of delivery, tool-owned)

- After the fix lands, run `hero check --reconcile` (or `hero spec verify` on a
  child, if that path is chosen) and confirm it flips **`content-remediation`** to
  `completed` and archives it to `.hero/specs/content-remediation/`. Do **not**
  hand-edit its frontmatter.
- For **`cold-start-trust-hardening`**: its 10 children are already delivered per
  its Progress note; let the same reconcile path complete it, which will reconcile
  the stale `completed_at`. If it should remain open for an unrelated reason,
  clear the orphaned `completed_at` via the invariant repair rather than leaving
  the split. Decide during delivery based on whether all 10 declared children
  resolve to `completed`.

## Acceptance Criteria

- WHEN an initiative declares its children as a block-style `child:` list AND
  every declared child resolves to a `completed`, archived spec, THE SYSTEM SHALL
  flip the initiative to `status: completed` and archive it — **even if no child
  is being verified in the current process** (i.e. via a standalone re-check such
  as `hero check --reconcile`).
- WHEN all of an initiative's children are already `completed` and archived and
  the initiative is still `planning`, THE SYSTEM SHALL provide at least one
  idempotent command that completes it, and re-running that command SHALL be a
  safe no-op once the initiative is completed.
- THE SYSTEM SHALL NOT auto-complete an initiative that has any declared child
  which is unmaterialized or not `completed` (the `cst-initiative-premature-autocomplete`
  guarantee must be preserved — verify `TestVerify_UnmaterializedInitiativeChild`
  still passes).
- THE SYSTEM SHALL NOT leave a spec with `completed_at` set while
  `status != completed`; the reconcile pass SHALL detect and repair the split.
- A regression test SHALL sit alongside `TestVerify_UnmaterializedInitiativeChild`
  proving the inverse case: an initiative whose declared block-style `child:`
  roster is fully completed **and already archived** (no child verified
  in-process) is completed by the standalone re-check.

## Test Plan

### Existing test review
- `internal/cli/verify_test.go:707` `TestVerify_InitiativeAutoComplete_FlowStyleRelations`
  — covers the **in-process** trigger: verifying the last child (`child-two`)
  completes `content-remediation`. Confirms the roster/child parse works. It does
  **not** cover the already-archived-children re-check gap.
- `internal/cli/verify_test.go:956` `TestVerify_UnmaterializedInitiativeChild`
  — guards against premature completion (the `cst-initiative-premature-autocomplete`
  fix). Must stay green after this change.

### Test changes needed
1. **New regression test** (alongside `TestVerify_UnmaterializedInitiativeChild`):
   build an initiative with a block-style `child:` roster where **all** children
   are `status: completed` under `specs/**` and none is verified in-process; run
   the new standalone re-check (`hero check --reconcile`, or the extracted helper
   directly) and assert the initiative is completed + archived to
   `specs/<slug>/spec.md`.
2. **Idempotency test:** running the re-check again is a no-op (no error, no
   double-archive).
3. **Invariant test:** a spec with `completed_at` set and `status: planning` is
   detected by reconcile and repaired (completed via roster, or `completed_at`
   cleared).
4. **Negative test guard:** re-confirm an initiative with one non-completed /
   unmaterialized declared child is **not** completed by the new path.

### Regression scope
- `internal/reconcile/` gains initiative-completion behavior — audit that it does
  not complete leaf features/bugs, and that its dry-run (`hero check` without
  `--reconcile`) only *reports*.
- The shared helper extraction must not change `autoCompleteParentIfReady`'s
  in-process behavior — `TestVerify_InitiativeAutoComplete_FlowStyleRelations`
  and the premature-autocomplete test are the guards.
- `go test ./...` including `internal/cli` and `internal/reconcile`.

## Kickoff

Fix: an initiative whose every declared block-style `child:` entry is `completed`
and archived never flips to `completed`, because `autoCompleteParentIfReady`
(`internal/cli/verify.go:624`, sole caller `verify.go:182`) only runs as an
in-process side-effect of verifying a not-yet-completed child. Once children are
completed+archived, re-verifying a child early-returns at `verify.go:103-111`
before the check, and neither `hero spec verify <initiative>` nor
`hero check --reconcile` re-runs child-driven completion. The roster/child-count
logic itself is correct (reproduced: it would complete `content-remediation`
today if invoked) — the gap is that **nothing invokes it a second time**.

**Do this:**
1. Extract `verify.go:656-691` into a shared `initiativeReadyToComplete` helper.
2. Wire it into `internal/reconcile/reconcile.go` (initiative branch,
   `reconcile.go:143-145`) so `hero check --reconcile` completes+archives a
   satisfied initiative via `completeAndArchive`. Keep `hero check` (no flag)
   report-only.
3. Add reconcile repair for the `completed_at` set / `status != completed` split.
4. Add the inverse regression test next to `TestVerify_UnmaterializedInitiativeChild`.
5. Run `hero check --reconcile` to complete the two stranded initiatives
   (`content-remediation`, `cold-start-trust-hardening`) — **do not hand-edit
   their status frontmatter**.

**Pick up at:** `internal/cli/verify.go:624-700` and
`internal/reconcile/reconcile.go`.

→ `.hero/planning/bugs/initiative-autocomplete-misses-completed-children/spec.md`

**Preserve:** `TestVerify_UnmaterializedInitiativeChild` and
`TestVerify_InitiativeAutoComplete_FlowStyleRelations` must stay green — this is
the inverse of `cst-initiative-premature-autocomplete`; do not reopen that hole.

## Validation

- `hero check --reconcile` flips `content-remediation` to `completed` and archives
  it to `.hero/specs/content-remediation/`; re-running is a clean no-op.
- After delivery, no spec in the tree has `completed_at` set while
  `status != completed` (spot-check `cold-start-trust-hardening`).
- `TestVerify_UnmaterializedInitiativeChild` and
  `TestVerify_InitiativeAutoComplete_FlowStyleRelations` pass; new inverse-case
  test passes.
- `go build ./... && go test ./...` green.
- Sanity: an initiative with a deliberately unmaterialized declared child is still
  refused (no regression of the premature-autocomplete guard).

## Root Cause Analysis (evidence index)

| Claim | Status | Evidence |
|---|---|---|
| `autoCompleteParentIfReady` has one caller, in-process only | read | `verify.go:182`, sole caller; grep confirms no others |
| Block-style `child:` parses into `Kind:"child"` relations | read + reproduced | `spec.go:613-641`, `parseScalarListBlock` `spec.go:795`; live parse showed 8 child relations |
| Roster + child-count gates pass for `content-remediation` today | reproduced | read-only run: `declaredCount=8 declaredComplete=true childCount=8 allDone=true` |
| Completed+archived child re-verify early-returns before the check | read | `verify.go:103-111` returns before `verify.go:182` |
| `hero check --reconcile` never completes an initiative from children | read | `reconcile.go:27-36,73-79,143-145` (delivering branch explicitly skips auto-complete) |
| `completed_at` written with `status` on the canonical path | read | `complete.go:300-312` (`updateFrontmatterStatus` stamps in same write) |
| `cold-start` split is manual-restore residue of the old premature-autocomplete bug | read | `cold-start-trust-hardening/spec.md:160` Progress note |
| Children all `completed`, archived, carry `parent`→`content-remediation` | read | `find` over `.hero/specs/**` frontmatter |

**Could it be reproduced?** Yes — the stuck end-state is live on disk, and the
roster/child logic was executed read-only against the real tree to prove it would
succeed if invoked, isolating the defect to the missing standalone trigger.

## Completion Ledger

Delivered 2026-07-09. `go build ./...`, `go vet` (cli/reconcile/spec), and
`go test ./...` (86 packages) green. Cold audit: **SHIP (noteworthy)** — see
`delivery-audit.md`. The fix adds an idempotent standalone parent re-check and
single-sources the completion decision so the in-process and reconcile paths
cannot diverge. Live reconcile flipped `content-remediation` → completed+archived
and cleared `cold-start-trust-hardening`'s orphaned `completed_at`.

### Acceptance Criteria

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | Fully-completed archived block-style `child:` roster → initiative completed+archived even with no child verified in-process | DONE | `reconcile.go` `FindingInitiativeComplete` + `check.go` apply → `completeAndArchive`; `TestReconcile_InitiativeCompleteFromArchivedChildren`, `TestCheckReconcile_CompletesArchivedChildrenInitiative`; live: content-remediation planning→completed |
| 2 | ≥1 idempotent command completes it; re-run is a safe no-op | DONE | `hero check --reconcile`; `TestReconcile_InitiativeComplete_Idempotent` + live re-run clean no-op |
| 3 | Never auto-complete an initiative with any unmaterialized/non-completed declared child (premature-autocomplete guard preserved) | DONE | roster gate verbatim in `spec.InitiativeReadyToComplete`; `TestReconcile_NegativeGuard_UnmaterializedChild` + `TestVerify_UnmaterializedInitiativeChild` (still green) |
| 4 | Never leave `completed_at` set while `status != completed`; reconcile detects + repairs | DONE | `FindingOrphanCompletedAt` + `clearCompletedAt`; `TestReconcile_OrphanCompletedAt`, `TestCheckReconcile_ClearsOrphanCompletedAt`; live: cold-start orphan cleared, 0 orphans repo-wide |
| 5 | Inverse-case regression test alongside `TestVerify_UnmaterializedInitiativeChild` | DONE | `TestCheckReconcile_CompletesArchivedChildrenInitiative` in `verify_test.go` — fails if fix reverted |

### Fix items

| # | Item | Status | Evidence |
|---|---|---|---|
| 1 | Extract gate into shared predicate; wire idempotent standalone re-check into reconcile; dry-run report-only, `--reconcile` applies | DONE | `spec.InitiativeReadyToComplete` (new `initiative_complete.go`) called by both `autoCompleteParentIfReady` + `Reconcile`. Deviation: predicate placed in `spec` (not `cli`) because `reconcile` cannot import `cli`. Chose reconcile as the single primary surface (did not also add the verify.go early-return re-trigger). |
| 2 | Enforce `status`↔`completed_at` invariant; completion wins over clearing | DONE | `FindingOrphanCompletedAt` finding + `clearCompletedAt` repair; completion path stamps both in same write |
| 3 | Reconcile the two stranded initiatives, tool-owned (no hand-edit) | DONE | `hero check --reconcile`: content-remediation → completed+archived; cold-start orphan cleared, left open (its declared children don't all resolve to completed) |

### Exercise-the-feature check

- [x] Exercised end-to-end: built `hero` from `./cmd/hero`, ran `hero check` (dry-run —
  confirmed report-only, no disk mutation) then `hero check --reconcile` against this
  repo's live `.hero`. Observed `content-remediation` flip planning→completed and archive
  to `.hero/specs/content-remediation/spec.md`; `cold-start-trust-hardening` orphaned
  `completed_at` cleared (status left `planning` — its children don't all resolve to
  completed); repo-wide orphan sweep = 0; second `--reconcile` a clean no-op.
- Note: the same `--reconcile` run also archived ~25 pre-existing completed-but-stuck
  specs and recovered a second stranded initiative (`hero-surface-polish`) — legitimate
  existing reconcile behavior, committed separately from the code fix.
