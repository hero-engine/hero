---
title: "Merge driver never regenerates .hero/NEXT.md — project file falls through to 'keep ours'"
slug: next-project-file-conflict-not-regenerated
type: bug
status: planning
severity: medium
priority: medium
size: small
domain: engineering
created: 2026-06-03
origin: session
root_cause_class: code
regression-of:
  - next-as-projection
relates-to:
  - next-projection-gate-punts-migration-to-user
  - next-team-mode-per-user-handoff-unmaintained
  - next-merge-driver-not-portable
---

# Merge driver never regenerates .hero/NEXT.md — project file falls through to "keep ours"

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | medium — correctness gap that violates AC-10 of `next-as-projection`, but self-healing on the next checkpoint, so blast radius is one stale turn. |
| **Ease of Fix** | easy — add a by-name route mirroring the two existing branches (`isQueueOutputPath`, `isSnapshotOutputPath`). |
| **Caused by our codebase?** | Yes — a missing branch in `runNextMergeResolve`. |
| **Needs more research?** | No — root cause confirmed against source; the renderer (`projection.NextMD`) and its checkpoint-time invocation are both read and understood. |

### Background
Hero ships a git merge driver (`hero-next`) that resolves conflicts on the projected
handoff files by ignoring both sides of the merge and regenerating from the local graph.
`.gitattributes` marks three paths for the driver: `.hero/next/*.md`, `.hero/NEXT.md`, and
`.hero/QUEUE.md`. The driver correctly regenerates QUEUE.md (→ `RenderQueueSnapshot`),
SNAPSHOT.md (→ `runSnapshotMergeResolve`), and per-user files (→ `projection.UserHandoffMD`).
But the **project-level `.hero/NEXT.md` has no route** — it is silently resolved as "keep ours."

### Analysis
`runNextMergeResolve` dispatches by inspecting the `--output` path git passes (git's `%A`).
There are explicit by-name matches for QUEUE and SNAPSHOT, then a fall-through to
`userFromOutputPath`, which only matches files whose **parent directory is `next`**.
`.hero/NEXT.md` has parent `.hero`, so `userFromOutputPath` returns `""`, and the function
hits the "Unknown projected path — leave file as-is (current side wins), exit 0" branch.
The conflict is "resolved" by keeping the local branch's NEXT.md verbatim — including any
conflict the merge produced — rather than re-projecting from the graph.

### Root Cause
A missing dispatch branch in `runNextMergeResolve` (`internal/cli/next_hooks.go`). The
project-level renderer `projection.NextMD` exists and is invoked at checkpoint time by
`writeProjectedNextMD`, but the merge driver never calls it. There is no `isNextOutputPath`
matcher analogous to `isQueueOutputPath` / `isSnapshotOutputPath`.

### Source
- `internal/cli/next_hooks.go` — `runNextMergeResolve` (dispatch), `userFromOutputPath`
  (the `next`-parent guard that excludes `.hero/NEXT.md`), the "Unknown projected path"
  fall-through.
- `internal/projection/projection.go` — `NextMD` (the renderer that should be called).
- `internal/cli/checkpoint.go` — `writeProjectedNextMD` (the canonical invocation to mirror).

### Fix Direction
Add a by-name route for `.hero/NEXT.md` in `runNextMergeResolve`: detect the NEXT.md output
path with an `isNextOutputPath` matcher (mirroring `isSnapshotOutputPath`), then render
`projection.NextMD(store, opts)` with the same option set `writeProjectedNextMD` uses and
write it unconditionally to `--output`. Place the branch before the `userFromOutputPath`
fall-through.

---

## Problem Statement

When two branches both touch `.hero/NEXT.md` and git produces a merge conflict, the
`hero-next` merge driver is supposed to discard both sides and write a fresh graph
projection to `%A` (AC-10 of `next-as-projection`). For `.hero/NEXT.md` specifically it does
not: it leaves the current side (ours) in place and exits 0. Whatever was on the local
branch — including the unmerged content git staged — wins, and no fresh projection is
written.

This is the one projected file every non-CLI viewer sees on GitHub (the repo-root handoff
briefing), so a stale or conflicted NEXT.md is the most visible failure of the projection
guarantee.

**Reproduction (conceptual — see Test Plan for the automated version):**
1. In a repo with the merge driver installed, create divergent commits on two branches that
   each rewrite `.hero/NEXT.md`.
2. Merge one into the other. Git invokes `hero next merge-resolve --output <abs path to NEXT.md>`.
3. Observe that `.hero/NEXT.md` retains the current-branch content rather than a fresh
   `projection.NextMD` render.

**Self-healing caveat:** the very next `hero next checkpoint` (Stop-hook / pre-commit /
post-merge) calls `writeProjectedNextMD`, which re-projects `.hero/NEXT.md` from the graph.
So the practical window is one turn. The bug is a correctness gap, not data loss.

## Environment Details
- Affects any repo with `hero next install-hooks` run (driver registered in `.git/config`,
  `.gitattributes` marking `.hero/NEXT.md merge=hero-next`).
- Independent of solo/team mode — `.hero/NEXT.md` is the project-level file in both.
- No external dependency; reproducible locally with two branches and a merge.

---

## Root Cause Analysis

`runNextMergeResolve` (`internal/cli/next_hooks.go:123`) routes by `--output` path:

```
internal/cli/next_hooks.go:145   if isQueueOutputPath(nextMergeResolveOutput) { ... RenderQueueSnapshot ... }
internal/cli/next_hooks.go:156   if isSnapshotOutputPath(nextMergeResolveOutput) { ... runSnapshotMergeResolve ... }
internal/cli/next_hooks.go:160   user := userFromOutputPath(nextMergeResolveOutput)
internal/cli/next_hooks.go:161   if user == "" { return nil }   // <-- .hero/NEXT.md lands here, exits 0, ours wins
```

`userFromOutputPath` (`internal/cli/next_hooks.go:250-260`) requires the parent directory to
be `next`:

```
internal/cli/next_hooks.go:255   parent := filepath.Base(filepath.Dir(path))
internal/cli/next_hooks.go:256   if parent != "next" { return "" }
```

For `.hero/NEXT.md` the parent is `.hero`, so this returns `""`. There is no
`isNextOutputPath` check before the fall-through, so the project file is never routed to
`projection.NextMD`. The existing unit test `TestUserFromOutputPath`
(`internal/cli/next_hooks_test.go:21`) already asserts `.hero/NEXT.md` → `""` and documents
the intent (`// project file, not user file`) — confirming the omission is in the dispatch,
not in `userFromOutputPath`, which is behaving as designed.

The renderer is present and correct: `projection.NextMD`
(`internal/projection/projection.go:74`) takes a `*graph.Store` and `NextMDOptions` (RepoKey
required) and returns the full NEXT.md body. The driver already opens the store
(`internal/cli/next_hooks.go:134`) and computes `repoKey` (`internal/cli/next_hooks.go:140`),
so everything needed is in hand at the fall-through point.

**Confirmed** (read in this session): the dispatch order, the `next`-parent guard, the
"Unknown projected path" branch, the `NextMD` signature, and the canonical invocation in
`writeProjectedNextMD`. **Not assumed.**

Note on `.gitattributes` vs. dispatch: `updateGitAttributes`
(`internal/cli/next_hooks.go:729-733`) marks `.hero/next/*.md`, `.hero/NEXT.md`, and
`.hero/QUEUE.md` for the driver — but **not** `.hero/SNAPSHOT.md`. SNAPSHOT has a dispatch
branch but no `.gitattributes` entry (so the driver is never invoked on it today); NEXT.md
has a `.gitattributes` entry but no dispatch branch (so the driver is invoked and silently
keeps ours). This spec fixes the NEXT.md half. The SNAPSHOT `.gitattributes` gap is a
separate observation — see Secondary Defects.

---

## Code Flow (End to End)

Merge-resolve path (the bug):

1. `internal/cli/next_hooks.go:497-509` — `registerMergeDriver` registers
   `hero next merge-resolve --output %A` as the `hero-next` driver in `.git/config`.
2. `internal/cli/next_hooks.go:729-733` — `updateGitAttributes` marks `.hero/NEXT.md merge=hero-next`.
3. Git encounters a conflict on `.hero/NEXT.md` during a merge and invokes
   `hero next merge-resolve --output <abs path to NEXT.md (%A)>`.
4. `internal/cli/next_hooks.go:123` — `runNextMergeResolve` runs; opens config + graph store,
   computes `repoKey`.
5. `internal/cli/next_hooks.go:145` — `isQueueOutputPath` → false.
6. `internal/cli/next_hooks.go:156` — `isSnapshotOutputPath` → false.
7. `internal/cli/next_hooks.go:160` — `userFromOutputPath` → `""` (parent is `.hero`, not `next`).
8. `internal/cli/next_hooks.go:161-166` — fall-through: `return nil`. **File left as-is; ours wins.** BUG.

Contrast — checkpoint path (correct, the behavior to match):

1. `internal/cli/checkpoint.go:288` — `writeProjectedNextMD(nextPath, projectRoot, heroDir)`.
2. `internal/cli/checkpoint.go:289` — opens graph store.
3. `internal/cli/checkpoint.go:297-306` — calls `projection.NextMD` with RepoKey, Branch,
   Vocab, Methodology, HeroDir, ProjectRoot, and roadmap config from `cfg`.
4. `internal/cli/checkpoint.go:310` — writes via `writeProjectedFileIfSemanticChanged`
   (idempotent — skips the write when nothing changed semantically).

---

## Key Files

### Merge driver dispatch
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next_hooks.go` | 123–179 | `runNextMergeResolve` — the dispatch missing the NEXT.md branch. |
| `internal/cli/next_hooks.go` | 183–191 | `isQueueOutputPath` / `isSnapshotOutputPath` — matcher pattern to mirror. |
| `internal/cli/next_hooks.go` | 250–260 | `userFromOutputPath` — `next`-parent guard that excludes `.hero/NEXT.md`. |
| `internal/cli/next_hooks.go` | 725–737 | `updateGitAttributes` — marks `.hero/NEXT.md` for the driver (confirms driver IS invoked on it). |

### Renderer
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/projection/projection.go` | 28–66 | `NextMDOptions` — the option set to populate. |
| `internal/projection/projection.go` | 74–~200 | `NextMD` — the renderer the new branch must call. |

### Canonical invocation to mirror
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/checkpoint.go` | 284–312 | `writeProjectedNextMD` — exact options + config wiring to copy. |

### Tests
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next_hooks_test.go` | 13–31 | `TestUserFromOutputPath` — already asserts `.hero/NEXT.md` → `""`. |
| `internal/cli/next_hooks_test.go` | 287–303 | `TestIsQueueOutputPath` — unit-test pattern to mirror for `isNextOutputPath`. |

---

## Acceptance Criteria

- WHEN `hero next merge-resolve --output <path>/.hero/NEXT.md` runs THE SYSTEM SHALL write a
  fresh `projection.NextMD` render to that path and exit 0 — NOT leave the current side in place.
- WHEN the regenerated `.hero/NEXT.md` is produced THE SYSTEM SHALL leave no conflict markers
  (`<<<<<<<`, `=======`, `>>>>>>>`) in the file.
- THE SYSTEM SHALL render `.hero/NEXT.md` in the merge-resolve path using the same
  `NextMDOptions` (RepoKey, Branch, Vocab, Methodology, HeroDir, ProjectRoot, roadmap config)
  that `writeProjectedNextMD` uses at checkpoint time, so merge-resolve output matches what
  `hero next checkpoint` would produce.
- WHEN the merge driver writes `.hero/NEXT.md` THE SYSTEM SHALL write unconditionally to
  `%A` (the merge driver MUST NOT skip the write the way the idempotent checkpoint path does;
  git requires `%A` to be materialized as the merged result).
- `isNextOutputPath` SHALL match `.hero/NEXT.md` regardless of whether git passes an absolute
  or relative `--output` path (mirror `isSnapshotOutputPath`'s `filepath.Base` comparison) AND
  SHALL NOT match `.hero/next/<user>.md`, `.hero/QUEUE.md`, `.hero/SNAPSHOT.md`, or
  `.hero/next.md.backup`.
- THE existing `TestUserFromOutputPath` assertion (`.hero/NEXT.md` → `""`) SHALL remain
  passing — `userFromOutputPath` is unchanged; the fix is purely a new dispatch branch.
- WHERE the graph is sparse (fresh repo) THE SYSTEM SHALL still write a valid NEXT.md
  (sections render as "Nothing yet.") and exit 0 — no error path that would abort the merge.

---

## Suggested Fix Approach

All changes in `internal/cli/next_hooks.go`. Two edits: a new matcher and a new dispatch branch.

### 1. Add the matcher (mirror `isSnapshotOutputPath`)

**After** `isSnapshotOutputPath` (around line 191), add:

```go
// isNextOutputPath reports whether the merge driver's --output path
// names the project-level NEXT.md. Distinct from userFromOutputPath,
// which only matches per-user files under .hero/next/. Uses
// filepath.Base so it's robust to absolute vs relative --output paths
// git may pass.
func isNextOutputPath(path string) bool {
	return filepath.Base(path) == "NEXT.md"
}
```

(`NEXT.md` is uppercase and unambiguous; `filepath.Base` strips any directory git supplies.
The per-user files are `<user>.md`, never `NEXT.md`, so there is no collision.)

### 2. Add the dispatch branch (before the `userFromOutputPath` fall-through)

**Before** — `internal/cli/next_hooks.go:156-166`:

```go
	if isSnapshotOutputPath(nextMergeResolveOutput) {
		return runSnapshotMergeResolve(projectRoot, heroDir, cfg)
	}

	user := userFromOutputPath(nextMergeResolveOutput)
	if user == "" {
		// Unknown projected path — leave file as-is (current side
		// wins, no conflict markers). Still exit 0 so git treats
		// the merge as successful.
		return nil
	}
```

**After**:

```go
	if isSnapshotOutputPath(nextMergeResolveOutput) {
		return runSnapshotMergeResolve(projectRoot, heroDir, cfg)
	}

	// Project-level NEXT.md: regenerate from the graph, mirroring the
	// option set writeProjectedNextMD uses at checkpoint time. Written
	// unconditionally to --output (git's %A) — the idempotent
	// semantic-change skip used at checkpoint time would leave %A
	// unmaterialized and break the merge.
	if isNextOutputPath(nextMergeResolveOutput) {
		content, err := projection.NextMD(store, projection.NextMDOptions{
			RepoKey:                 repoKey,
			Branch:                  currentBranch(projectRoot),
			Vocab:                   activeVocab(&cfg),
			Methodology:             activeMethodology(&cfg),
			HeroDir:                 heroDir,
			ProjectRoot:             projectRoot,
			RoadmapRecencyDays:      cfg.Roadmap.AmbientRecencyDaysOrDefault(),
			RoadmapStopNaggingHours: cfg.Roadmap.StopNaggingHoursOrDefault(),
		})
		if err != nil {
			return fmt.Errorf("projection: %w", err)
		}
		return os.WriteFile(nextMergeResolveOutput, []byte(content), 0o644)
	}

	user := userFromOutputPath(nextMergeResolveOutput)
	if user == "" {
		// Unknown projected path — leave file as-is (current side
		// wins, no conflict markers). Still exit 0 so git treats
		// the merge as successful.
		return nil
	}
```

**Why:** routes `.hero/NEXT.md` to the same renderer + option set the checkpoint path uses,
so merge-resolve output is byte-equivalent (modulo the `updated:` timestamp) to what
`hero next checkpoint` writes. `os.WriteFile` is the unconditional write the merge driver
needs — it does NOT go through `writeProjectedFileIfSemanticChanged` (which can skip),
matching how the QUEUE and user branches already write.

**Confirm before coding:** `cfg` is already in scope at this point (loaded at
`internal/cli/next_hooks.go:128`), and `currentBranch`, `activeVocab`, `activeMethodology`,
and `cfg.Roadmap.*` are the same helpers `writeProjectedNextMD` uses — all already imported
in package `cli`. `projection` and `os` are already imported in `next_hooks.go`. No new
imports required.

---

## Test Plan

### Existing test review
- `internal/cli/next_hooks_test.go:13-31` `TestUserFromOutputPath` — asserts `.hero/NEXT.md`
  → `""`. Keep as-is; the fix does not touch `userFromOutputPath`.
- `internal/cli/next_hooks_test.go:287-303` `TestIsQueueOutputPath` — the table-driven
  matcher pattern to mirror for `isNextOutputPath`.
- No full graph-backed merge-resolve test exists today (confirmed: no `TestRunNextMergeResolve`,
  no `runSnapshotMergeResolve` test). The QUEUE/SNAPSHOT/user routes are covered only by their
  pure path-matcher unit tests.

### Test changes needed
1. **`TestIsNextOutputPath`** (mirror `TestIsQueueOutputPath`) — table-driven:
   - `"/repo/.hero/NEXT.md"` → true
   - `".hero/NEXT.md"` → true
   - `"NEXT.md"` → true
   - `"/repo/.hero/next/chet-bellows.md"` → false
   - `"/repo/.hero/QUEUE.md"` → false
   - `"/repo/.hero/SNAPSHOT.md"` → false
   - `"/repo/.hero/next.md"` → false (case/exact-name guard)
   - `"/repo/.hero/NEXT.md.backup"` → false
2. **Graph-backed regeneration test** (`TestRunNextMergeResolve_RegeneratesProjectNext`) —
   establishes the missing end-to-end coverage AC-10 asks for:
   - Set up a temp repo with a `.hero/` and an initialized graph store seeded with at least
     one commit/feature so `NextMD` renders non-empty sections.
   - Write a `.hero/NEXT.md` containing fake conflict markers (`<<<<<<<` … `>>>>>>>`).
   - Set `nextMergeResolveOutput` to that path and call `runNextMergeResolve`.
   - Assert: no conflict markers remain; the file starts with the `---` frontmatter block and
     contains the `## Just finished` / `## Next` headers that `NextMD` emits; exit (returned
     error) is nil.
   - If wiring a real graph store in a CLI unit test is heavy, this can instead live as an
     integration test alongside the checkpoint tests; the path-matcher unit test (#1) is the
     mandatory minimum and is what locks the dispatch fix.

### Regression scope
- `runNextMergeResolve` dispatch ordering — confirm QUEUE, SNAPSHOT, and per-user routes
  still resolve to their existing handlers (the new branch sits between SNAPSHOT and the
  user fall-through and matches only basename `NEXT.md`).
- `userFromOutputPath` unchanged — `TestUserFromOutputPath` must stay green.
- No `.gitattributes` change in this spec; `updateGitAttributes` already lists `.hero/NEXT.md`.
- Run `go test ./internal/cli/...` and `go test ./internal/projection/...`.

---

## Secondary Defects
- **SNAPSHOT.md `.gitattributes` gap (separate issue, not fixed here):**
  `updateGitAttributes` (`internal/cli/next_hooks.go:729-733`) marks `.hero/next/*.md`,
  `.hero/NEXT.md`, and `.hero/QUEUE.md` for `merge=hero-next` but **omits `.hero/SNAPSHOT.md`**,
  even though `runNextMergeResolve` has an `isSnapshotOutputPath` branch ready to handle it.
  Net effect: the driver is never invoked on SNAPSHOT.md, so SNAPSHOT.md conflicts resolve by
  whatever git's default text merge does (potentially leaving conflict markers). This is the
  mirror image of the NEXT.md bug (NEXT has the attribute but no dispatch; SNAPSHOT has the
  dispatch but no attribute). Worth its own spec; flagged here for traceability.

---

## Boundaries
- Does NOT change `userFromOutputPath`, the QUEUE route, the per-user route, or
  `.gitattributes`.
- Does NOT fix the SNAPSHOT.md `.gitattributes` omission (see Secondary Defects — separate spec).
- Does NOT address merge-driver portability across machines (`next-merge-driver-not-portable`)
  or team-mode per-user handoff staleness (`next-team-mode-per-user-handoff-unmaintained`).
- Does NOT alter the self-heal checkpoint path; it only makes the merge driver itself correct
  so the file is right at merge time rather than one turn later.

## Risks
- Low. The change is additive (one matcher + one branch) and copies a proven invocation.
- The only behavioral shift is that a previously-untouched `.hero/NEXT.md` merge now gets
  rewritten from the graph. That is the intended AC-10 behavior; the risk is purely that the
  regenerated content differs from a hand-curated NEXT.md — but in projection mode NEXT.md is
  graph-owned by design (any hand edits are already wiped by the next checkpoint).
- Confirm `cfg` and the projection helpers are in scope at the insertion point (they are —
  see Suggested Fix Approach), otherwise the build breaks.

## Validation
- `go build ./...` and `go test ./internal/cli/... ./internal/projection/...` pass.
- New `TestIsNextOutputPath` passes; existing `TestUserFromOutputPath` and
  `TestIsQueueOutputPath` stay green.
- Manual: in a scratch repo with hooks installed, create two branches that both edit
  `.hero/NEXT.md`, merge, and confirm the result is a clean graph projection with no conflict
  markers and frontmatter at the top.

---

## Recap
`.hero/NEXT.md` is marked for the `hero-next` merge driver in `.gitattributes`, but
`runNextMergeResolve` has no dispatch branch for it — it falls through `userFromOutputPath`
(which only matches `.hero/next/<user>.md`) into the "keep ours" exit, so NEXT.md merge
conflicts are never regenerated from the graph, violating AC-10 of `next-as-projection`. The
fix is a small additive `isNextOutputPath` matcher plus a `projection.NextMD` branch mirroring
`writeProjectedNextMD`. Severity is medium: real correctness gap on the most-visible projected
file, but self-healing on the next `hero next checkpoint`.

## Kickoff

Fixes the `hero-next` git merge driver so a conflict on `.hero/NEXT.md` regenerates from the
graph instead of silently keeping the local branch's copy.

**Status:** planning — root cause confirmed; fix is a one-matcher + one-branch addition to the
merge driver dispatch.

**Pick up at:** in `internal/cli/next_hooks.go`, add `isNextOutputPath` (mirror
`isSnapshotOutputPath` at ~line 189) and a dispatch branch calling `projection.NextMD` before
the `userFromOutputPath` fall-through at ~line 160, copying the option set from
`writeProjectedNextMD` in `checkpoint.go:297`. Then add `TestIsNextOutputPath` mirroring
`TestIsQueueOutputPath`.

→ `.hero/planning/bugs/next-project-file-conflict-not-regenerated/spec.md`

**Files:** `internal/cli/next_hooks.go:156`, `internal/cli/next_hooks.go:250`, `internal/cli/checkpoint.go:297`, `internal/projection/projection.go:74`, `internal/cli/next_hooks_test.go:287`
**Skip:** changing `userFromOutputPath` (it's correct — `.hero/NEXT.md` is intentionally not a user file); touching `.gitattributes` (already lists NEXT.md).
