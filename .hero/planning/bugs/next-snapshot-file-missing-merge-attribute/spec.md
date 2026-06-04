---
title: SNAPSHOT.md is missing its merge-driver attribute — its merge-resolve handler is dead code and the file gets raw conflict markers
slug: next-snapshot-file-missing-merge-attribute
type: bug
status: planning
severity: medium
priority: P2
domain: engineering
created: 2026-06-03
origin: session
root_cause_class: code
tags: [next-md, projection, merge-driver, gitattributes, snapshot]
relations:
  - target: next-merge-driver-not-portable
    kind: relates-to
  - target: next-project-file-conflict-not-regenerated
    kind: relates-to
  - target: next-as-projection
    kind: regression-of
---

# SNAPSHOT.md is missing its merge-driver attribute — handler is dead code, file gets raw conflict markers

> Session-originated bug (surfaced during the NEXT-subsystem deep-research pass). No `tracker_id`.

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | medium — `.hero/SNAPSHOT.md` is a tracked, graph-projected file, but `.gitattributes` never routes it to the `hero-next` merge driver, so a conflicting merge writes raw `<<<<<<<` markers into it for *every* clone (installed driver or not). Self-heals on the next `hero next checkpoint` projection, so blast radius is one stale/dirty turn. |
| **Ease of Fix** | trivial — add `.hero/SNAPSHOT.md merge=%s` to `updateGitAttributes`. The merge-resolve handler it needs already exists. |
| **Caused by our codebase?** | Yes — `internal/cli/next_hooks.go` `updateGitAttributes` omits the SNAPSHOT line. |
| **Needs more research?** | No — confirmed against source. |

### Background
`runNextMergeResolve` already routes a SNAPSHOT.md `--output` path to `runSnapshotMergeResolve`, which regenerates `.hero/SNAPSHOT.md` from the graph + repo shape ([next_hooks.go:156-157](internal/cli/next_hooks.go:156), [:187-198](internal/cli/next_hooks.go:187)). But that handler is **only reached if git actually invokes the `hero-next` driver on SNAPSHOT.md** — which requires a `merge=hero-next` attribute in `.gitattributes`. `updateGitAttributes` ([next_hooks.go:725](internal/cli/next_hooks.go:725)) writes that attribute for `.hero/next/*.md`, `.hero/NEXT.md`, and `.hero/QUEUE.md` — but **not** `.hero/SNAPSHOT.md`. So the snapshot branch in the merge driver is dead code, and SNAPSHOT.md falls through to git's default text merge → raw conflict markers.

### Root Cause
**Code defect — a missing line.** `updateGitAttributes` enumerates three of the four projected files. The fourth (`SNAPSHOT.md`) was given a merge-resolve handler (`isSnapshotOutputPath` / `runSnapshotMergeResolve`) but never added to the `.gitattributes` managed block, so the handler is unreachable. This is the inverse of the sibling bug `next-project-file-conflict-not-regenerated` (there the attribute exists but the handler's dispatch is missing; here the handler exists but the attribute is missing).

### Source
- `internal/cli/next_hooks.go:725` — `updateGitAttributes`: writes `.hero/next/*.md`, `.hero/NEXT.md`, `.hero/QUEUE.md` — omits `.hero/SNAPSHOT.md`.
- `internal/cli/next_hooks.go:156-157` — `runNextMergeResolve` routes `isSnapshotOutputPath` → `runSnapshotMergeResolve`.
- `internal/cli/next_hooks.go:187-198` — `isSnapshotOutputPath` / `runSnapshotMergeResolve` (the ready, currently-unreachable handler).
- `.gitattributes` — managed block currently lists three files, not SNAPSHOT.md.

---

## Problem Statement

`.hero/SNAPSHOT.md` is a tracked, every-turn-regenerated projection (`projectSnapshot` runs each checkpoint). Because `.gitattributes` doesn't mark it `merge=hero-next`, any branch merge that touches it produces standard git conflict markers instead of a regenerated file. The dedicated merge-resolve code path that would prevent this exists but is never invoked.

### Reproduction (inferred — not run)
1. On two branches, let `hero next checkpoint` produce divergent `.hero/SNAPSHOT.md` content.
2. Merge one into the other.
3. Observe `<<<<<<<`/`=======`/`>>>>>>>` markers in `.hero/SNAPSHOT.md` (no `hero-next` driver fired), even on a clone where `hero install` registered the driver.

---

## Acceptance Criteria

- **AC-1 — SNAPSHOT.md is routed to the merge strategy.** After `hero next install-hooks` (or whatever installs `.gitattributes`), the managed block includes a merge attribute for `.hero/SNAPSHOT.md`, consistent with the strategy chosen for the other projected files.
- **AC-2 — Its merge-resolve handler is reachable.** A simulated conflicting merge on `.hero/SNAPSHOT.md` resolves to a fresh `runSnapshotMergeResolve` regeneration (or, under a `union` strategy, to a marker-free concatenation that the next checkpoint cleans) — never raw conflict markers.
- **AC-3 — Strategy consistency.** The SNAPSHOT.md attribute uses the *same* merge strategy as `NEXT.md` / `QUEUE.md` / `next/*.md`. If `next-merge-driver-not-portable` switches those to built-in `merge=union`, SNAPSHOT.md moves with them.

---

## Suggested Fix Approach

Add the missing line to `updateGitAttributes` ([next_hooks.go:725](internal/cli/next_hooks.go:725)) so the managed block emits a merge attribute for `.hero/SNAPSHOT.md` alongside the other three. One line in the format string / attribute list.

**Coordinate with `next-merge-driver-not-portable`:** that spec decides whether the projected files use the custom `hero-next` driver, built-in `merge=union`, or a hybrid. This fix must use whatever strategy that spec lands — do not hardcode `merge=hero-next` if the sibling spec is moving the block to `union`. If this spec is delivered first, use the current `hero-next` strategy and leave a note that the portability spec will sweep all four lines together; if delivered after, match the chosen strategy. Either way, the four projected files must end up on **one** strategy, not a mix.

Because the change is to the managed-block writer, existing installs need a re-run of the installer (`hero next install-hooks` / `hero install`) to pick up the new line — note this in delivery; the portability spec likely addresses the same re-render concern.

---

## Test Plan

1. **Attribute emitted** — call `updateGitAttributes` against a temp repo; assert the managed block contains a `.hero/SNAPSHOT.md merge=<strategy>` line matching the other three.
2. **Handler reachable** — simulate the merge driver invocation with a SNAPSHOT.md `--output` path (mirror the existing QUEUE/SNAPSHOT/user merge-resolve tests in `next_hooks_test.go`); assert `runSnapshotMergeResolve` regenerates the file and no markers remain.
3. **Strategy consistency** — assert all four projected-file lines in the managed block use the same merge strategy token.

---

## Kickoff

You're fixing a one-line omission in Hero's NEXT/projection merge wiring. Read this spec first: `.hero/planning/bugs/next-snapshot-file-missing-merge-attribute/spec.md`.

**The bug:** `.hero/SNAPSHOT.md` is a tracked, every-turn graph projection, but `updateGitAttributes` ([internal/cli/next_hooks.go:725](internal/cli/next_hooks.go:725)) never adds its `merge=hero-next` attribute — even though `runNextMergeResolve` already has a `runSnapshotMergeResolve` handler ready ([next_hooks.go:156](internal/cli/next_hooks.go:156)). So the handler is dead code and SNAPSHOT.md gets raw conflict markers on any conflicting merge.

**The fix:** add the `.hero/SNAPSHOT.md` line to the `updateGitAttributes` managed block, using the SAME merge strategy as the other three projected files. **Check the sibling spec `next-merge-driver-not-portable` first** — if it has moved (or is moving) the block to built-in `merge=union`, use `union` here too; do not leave a mix. Add the three tests in the Test Plan (attribute emitted, handler reachable, strategy consistency). Note in the delivery that existing installs must re-run `hero next install-hooks` to pick up the new line.

**Do NOT** also fix the inverse `.hero/NEXT.md` dispatch gap here — that's owned by `next-project-file-conflict-not-regenerated`.

---

## Recap
`.hero/SNAPSHOT.md` has a graph-regenerating merge-resolve handler that is never reached because `updateGitAttributes` omits its `.gitattributes` merge attribute — so the file gets raw conflict markers on conflicting merges and self-heals only on the next checkpoint. Trivial `code` fix (add one line), but it must adopt whatever merge strategy `next-merge-driver-not-portable` settles on so all four projected files stay consistent. Severity: medium.
