---
title: "CI NEXT.md drift gate red again — projection stamps a live `branch:` line the gate doesn't ignore"
slug: next-drift-gate-branch-line-drift
type: bug
status: planning
domain: engineering
priority: high
severity: high
root_cause_class: design
size: small
created: 2026-07-13
tags: [ci, next-projection, drift-gate, flaky-gate, release-blocker, recurrence]
related: [next-drift-gate-unwinnable, next-as-projection, next-as-projection-architecture]
---

# CI NEXT.md drift gate red again — the `branch:` frontmatter line drifts

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — reds out the `Test` job on `main` for essentially every commit, masking the real test signal and blocking release confidence. Not a product defect. |
| **Ease of Fix** | easy — delete one 3-line emission block in the projection; belt-and-suspenders is a 1-line gate + 1-line normalize change. |
| **Caused by our codebase?** | Yes — the NEXT.md projector bakes the live git branch into a byte-exact-gated committed file. |
| **Needs more research?** | No — root cause reproduced end-to-end locally with the gate's exact command. |

### Background
This is a **recurrence / variant** of the completed spec `next-drift-gate-unwinnable`
(v0.24.x). That fix neutralized two volatile inputs to the byte-gated `.hero/NEXT.md`
(the `updated:` wall-clock timestamp — ignored in the gate via `-I'^updated: '` and
normalized in `normalizeUpdatedFrontmatter`; and the churny size-drift count — removed
from the projection). It missed a third volatile line with the **identical defect
shape**: `branch:`. The gate is red again on new `main` commits (runs `29277637764`,
`29274092561`, and the in-flight v0.25.1 run `~29282543046`).

### Analysis
`.hero/NEXT.md` carries a `branch:` frontmatter line stamped from the projecting
machine's **live** git checkout (`git rev-parse --abbrev-ref HEAD`). That value is
environment-local — it is not derivable from committed state. A developer (or a
concurrent session) who projects NEXT.md on a feature branch bakes that branch name
into the committed file. When CI checks out `main` and re-projects, it stamps
`branch: main`, which differs. The drift gate ignores `updated:` but **not** `branch:`,
so the diff is non-empty and the gate fails.

### Root Cause
`internal/projection/projection.go:90-92` unconditionally emits `branch: <opts.Branch>`
into the NEXT.md frontmatter, where `opts.Branch = currentBranch(projectRoot)` /
`gitutil.CurrentBranch(projectRoot)` = the live checkout branch. This is an
environment-local value placed inside a file that CI gates **byte-for-byte** (modulo
the `updated:` line). The deeper design fault is the same one the prior spec's
rejected "Alternative A" warned about: a byte-exact gate that must **enumerate every
volatile line** re-breaks the moment a new volatile line appears — which is exactly
what happened here. The `updated:` line was handled two ways (gate `-I` and semantic
normalize); `branch:` was handled neither, so it drifts.

### Source
- `internal/projection/projection.go:90-92` — emits the `branch:` line (the defect).
- `internal/cli/checkpoint.go:481` + `internal/cli/next_project.go:59` — populate
  `opts.Branch` from the live git branch.
- `internal/cli/checkpoint.go:623-665` — `writeProjectedFileIfSemanticChanged` /
  `normalizeUpdatedFrontmatter`: the idempotence guard normalizes only `updated:`, so
  a branch change is treated as a **semantic** change → the file is rewritten (with a
  fresh timestamp too), guaranteeing the on-disk diff.
- `.github/workflows/test.yml:58` — the gate ignores only `-I'^updated: '`.

### Fix Direction
Root-cause: **stop emitting the `branch:` line from the committed NEXT.md projection**
(mirrors how the prior fix removed the size-drift count). The `branch:` field is
documented `// current branch (frontmatter only)` and is consumed by **nothing** —
removing it is safe and loses no capability (live branch is always available via
`git status`/`git branch`). If branch context is wanted for handoff, its correct home
is the **gitignored** per-user local-state file (`.hero/next/<user>.local.md`), not the
committed shared projection. Do **not** rely on enumerating volatile lines in the gate —
the prior spec already rejected that as fragile, and this bug is the proof.

---

## Problem Statement

The `Test` GitHub Actions workflow's **"NEXT.md projection drift gate"**
(`.github/workflows/test.yml:43-64`) fails on `main` with:

> `::error::.hero/NEXT.md drifted from hero next checkpoint output.`

`go test -race -count=1 ./...` passes (86 packages). Only the drift gate is red,
across multiple recent `main` commits from concurrent/parallel sessions.

### Reproduction (definitive, local)
Current `HEAD` is `7a02a00` ("chore(hero): carry along concurrent-session workspace
state", v0.25.1). Its committed `.hero/NEXT.md` frontmatter reads
`branch: fix/mcp-transport-closes-midsession-supersede` — a branch that belonged to a
**different, concurrent session** (the one that produced `3c38bcc`,
"fix(serve): stop hero mcp daemon dying mid-session"). Running the projector on `main`
and then the gate's **exact** diff command:

```
$ git diff --exit-code -I'^updated: ' -- .hero/NEXT.md
...
-updated: 2026-07-13T20:28:08Z
+updated: 2026-07-13T23:45:02Z
 repo: hero-engine/hero
-branch: fix/mcp-transport-closes-midsession-supersede
+branch: main
...
EXIT CODE: 1
```

The `-I'^updated: '` filter suppresses a hunk only when **all** its changed lines match
the regex. Because the `branch:` line also changed, the hunk is not suppressed → exit 1
→ gate fails. The `branch:` line is the load-bearing divergence.

### Why concurrent sessions make it worse (not the fundamental cause)
The branch drift fires for **any** commit whose NEXT.md was projected on a branch other
than the one CI checks out — i.e. essentially every feature-branch commit. The
concurrent-session angle is an **amplifier**: a session on branch `A` projects NEXT.md
(stamping `branch: A`), and that file gets carried into a commit that lands on `main`
(exactly what `7a02a00`'s own message describes: "carry along concurrent-session
workspace state"). So even commits that land on `main` carry a foreign branch value and
drift when CI re-projects. Removing the line eliminates both cases at once.

---

## Environment Details
- CI: `.github/workflows/test.yml`, `ubuntu-latest`, Go `1.26.x`. The gate builds
  `./hero`, runs `./hero scan` then `./hero next checkpoint --quiet`, then byte-diffs
  `.hero/NEXT.md` ignoring `updated:`.
- `actions/checkout@v6` on a push to `main` leaves the checkout on branch `main`, so
  `git rev-parse --abbrev-ref HEAD` → `main` in CI.
- `next.projected = true` for this repo, so `hero next checkpoint` rewrites NEXT.md
  wholesale from the graph via `writeProjectedNextMD`.

---

## Root Cause Analysis

**Confirmed (read + reproduced):**

1. **The projector emits a live branch line.**
   `internal/projection/projection.go:90-92`:
   ```go
   if opts.Branch != "" {
       fmt.Fprintf(&b, "branch: %s\n", opts.Branch)
   }
   ```
   `opts.Branch` is set to the live checkout branch at both call sites:
   - `internal/cli/checkpoint.go:481` — `Branch: currentBranch(projectRoot)`
   - `internal/cli/next_project.go:59` — `Branch: gitutil.CurrentBranch(projectRoot)`
   Both resolve `git rev-parse --abbrev-ref HEAD`.

2. **Nothing consumes `branch:`.** The `NextMDOptions.Branch` field is documented
   `// current branch (frontmatter only)` (`projection.go:30`). A grep for any parser
   of `branch:` out of NEXT.md frontmatter returns nothing. Removal is behavior-safe.

3. **The idempotence guard doesn't protect the branch line.**
   `writeProjectedFileIfSemanticChanged` (`checkpoint.go:623-631`) compares existing vs
   new content after `normalizeUpdatedFrontmatter`, which normalizes **only** the
   `updated:` line (`checkpoint.go:659-663`). A differing `branch:` therefore reads as a
   semantic change → the file is rewritten (with a fresh `updated:` timestamp too),
   guaranteeing on-disk drift for CI to catch.

4. **The gate ignores only `updated:`.** `.github/workflows/test.yml:58` uses
   `git diff --exit-code -I'^updated: '`. `branch:` is not in the ignore set.

**Design-level cause:** an environment-local value (live branch) lives inside a
byte-exact-gated committed file. The prior spec (`next-drift-gate-unwinnable`) explicitly
**rejected** the "normalize/strip every volatile line in the gate" approach as fragile —
"must enumerate every volatile line; re-breaks when a new derived metric is added." This
bug is that prediction coming true: the `updated:` line was patched, the next volatile
line (`branch:`) re-broke the gate. Note the prior spec's own Evidence section even saw
the `branch:` line ("a `branch:` line (worktree artifact; would be `main` in CI)") but
dismissed it on the false assumption that the committed value is also `main`.

---

## Code Flow (End to End)

1. A developer or agent session runs `hero next checkpoint` (or the pre-commit hook does)
   while checked out on branch `A` (feature branch, or a concurrent session's branch).
2. `internal/cli/checkpoint.go:481` (or `internal/cli/next_project.go:59`) sets
   `opts.Branch = currentBranch()` → `A`.
3. `internal/projection/projection.go:90-92` writes `branch: A` into NEXT.md frontmatter.
4. `internal/cli/checkpoint.go:492` → `writeProjectedFileIfSemanticChanged` writes the
   file (branch `A` differs from prior → semantic change → rewrite).
5. The commit lands on `main` carrying `branch: A` in `.hero/NEXT.md` (directly, or via
   concurrent-session workspace state being carried along — cf. commit `7a02a00`).
6. CI (`test.yml`) checks out `main`, builds `./hero`, runs `./hero scan` then
   `./hero next checkpoint --quiet`.
7. On the `main` checkout, step 2 now yields `main`; step 3 writes `branch: main`;
   step 4 rewrites the file (branch differs → semantic change).
8. `test.yml:58` `git diff --exit-code -I'^updated: '` sees the `branch: A` → `branch: main`
   change (not covered by the `updated:` ignore) → exit 1.
9. `test.yml:59` emits `::error::.hero/NEXT.md drifted…` and the job fails.

---

## Key Files

### Projection (defect origin)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/projection/projection.go` | 30, 90-92 | Emits the `branch:` frontmatter line; the `Branch` field is `frontmatter only` (unconsumed) |

### Branch stamping (inputs)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/checkpoint.go` | 481, 1136-1140 | `Branch: currentBranch(projectRoot)`; `currentBranch` = `git rev-parse --abbrev-ref HEAD` (returns literal `HEAD` when detached) |
| `internal/cli/next_project.go` | 59 | `Branch: gitutil.CurrentBranch(projectRoot)` (the `hero next checkpoint` path) |
| `internal/gitutil/gitutil.go` | 95-105 | `CurrentBranch` returns `""` when detached (diverges from `checkpoint.go`'s variant) |

### Idempotence guard (fails to shield branch)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/checkpoint.go` | 623-665 | `writeProjectedFileIfSemanticChanged` + `normalizeUpdatedFrontmatter` normalize only `updated:`, not `branch:` |

### CI gate
| File | Lines | Relevance |
|------|-------|-----------|
| `.github/workflows/test.yml` | 43-64 | The drift gate; ignores only `-I'^updated: '` |

---

## Secondary Defects

1. **Latent `session:` volatility (same class).** `projection.go:87-89` emits
   `session: <opts.SessionID>` when set. `SessionID` is a per-session value
   (`checkpoint.go:137/177/201` set it from `ctx.SessionID`). It is not normalized by
   `normalizeUpdatedFrontmatter` and not ignored by the gate. It currently avoids drift
   only because the `next checkpoint` path reads it back from the existing file
   (`next_project.go:68-69 readSessionFromExistingNext`) and the committed file has no
   session line. If a checkpoint path ever stamps a fresh differing `SessionID`, the gate
   breaks identically. The recommended fix (see below) should treat `session:` the same
   as `branch:`.

2. **Two divergent `currentBranch` implementations.** `checkpoint.go:1136-1140` returns
   the literal string `HEAD` on detached HEAD; `gitutil.CurrentBranch` returns `""`.
   Harmless once the line is removed, but a latent inconsistency worth collapsing to one
   helper.

---

## Notes
- Fix must not break the **handoff magic** (automatic capture/load continuity). Removing
  the `branch:` line is safe on that axis: nothing reads it, and capture/load of the
  managed region and per-user handoff files is unaffected.
- Tracker type is `none` (`.hero/hero.json`) — no tracker posting applies.
- Mission-fit: a permanently-red `Test` job means "is CI green?" answers nothing, so no
  session can trust the pipeline signal — fixing it directly serves "every session starts
  as smart as where the last one left off."

---

## Root Cause Classification

- **Class: `design`** — an environment-local value (live git branch) embedded in a
  byte-exact-gated committed file; the byte-gate design cannot survive per-environment
  values. Fix surface is `code`. Concurrency is an **amplifier**, not the root (the drift
  fires for any feature-branch commit even single-threaded), so this is **not** a `race`.
- **Severity: high** — reds out `main`'s Test job on effectively every commit, masks the
  real test signal, blocks release confidence. Not a product/runtime defect.

---

## Suggested Fix Approach

### Change 1 — Remove the `branch:` line from the projection (root cause)
**File:** `internal/projection/projection.go` (~line 90-92, inside `NextMD`)

**Before**
```go
	if opts.SessionID != "" {
		fmt.Fprintf(&b, "session: %s\n", opts.SessionID)
	}
	if opts.Branch != "" {
		fmt.Fprintf(&b, "branch: %s\n", opts.Branch)
	}
	b.WriteString("---\n\n")
```

**After**
```go
	// Deliberately omit environment-local frontmatter (branch, session):
	// NEXT.md is byte-gated in CI (test.yml drift gate), and these values
	// differ between the committer's checkout and CI's `main` checkout,
	// producing unwinnable drift. Live branch is available via `git status`;
	// per-session context belongs in the gitignored .hero/next/<user>.local.md.
	// See next-drift-gate-branch-line-drift (recurrence of next-drift-gate-unwinnable).
	b.WriteString("---\n\n")
```

**Why:** The `branch:` (and `session:`) lines are the only remaining
environment-local values in the frontmatter. Neither is consumed by any code
(`Branch` is documented `frontmatter only`). Removing them at the source eliminates the
drift permanently and defuses the concurrent-session "carry a foreign branch" case —
without the fragile "enumerate volatile lines in the gate" pattern the prior spec rejected.

> Note: `opts.Branch` / `opts.SessionID` and their two call-site assignments
> (`checkpoint.go:481`, `next_project.go:59-60`) become inert. Leave them in place for a
> minimal, surgical fix (consistent with the prior spec's judgment call on inert
> `NextMDOptions` fields), or remove them as a small follow-on scrub.

### Change 2 (belt-and-suspenders) — teach the idempotence guard + gate to ignore these lines
Only needed if the team wants to **keep** `branch:`/`session:` in the file for human
context. Preferred is Change 1 (removal). If kept:

**File:** `internal/cli/checkpoint.go` — `normalizeUpdatedFrontmatter` (~line 659-663)

**Before**
```go
	for i, line := range lines {
		if strings.HasPrefix(line, "updated:") {
			lines[i] = placeholder
		}
	}
```

**After**
```go
	for i, line := range lines {
		if strings.HasPrefix(line, "updated:") ||
			strings.HasPrefix(line, "branch:") ||
			strings.HasPrefix(line, "session:") {
			lines[i] = placeholder + ":" + strings.SplitN(line, ":", 2)[0]
		}
	}
```
(use a per-key placeholder so two different keys don't collapse equal)

**File:** `.github/workflows/test.yml` (line 58)

**Before**
```sh
          if ! git diff --exit-code -I'^updated: ' -- .hero/NEXT.md; then
```
**After**
```sh
          if ! git diff --exit-code -I'^updated: ' -I'^branch: ' -I'^session: ' -- .hero/NEXT.md; then
```

**Why (and why this is the fallback, not the primary):** This keeps the human-visible
branch/session context but re-adopts the exact enumerate-every-volatile-line pattern the
prior spec rejected as fragile — the next new volatile frontmatter key re-breaks the gate.
Prefer Change 1.

---

## Test Plan

### Existing test review
- `internal/cli/checkpoint_test.go:341-416` — `normalizeUpdatedFrontmatter` byte-stability
  / idempotence tests (two consecutive projections byte-identical). These currently only
  exercise timestamp stability.
- `internal/cli/handoff_continuity_test.go:357-416` — handoff continuity + idempotence
  round-trip.
- `internal/projection/projection_test.go` — `TestNextMD_RoadmapShape_NeverEmitted`
  (regression guard from the prior fix) — pattern to mirror for an "absent line" guard.

### Test changes needed
1. **New guard (projection):** `TestNextMD_NoEnvironmentLocalFrontmatter` — call
   `projection.NextMD` with `Branch: "feature/x"` and `SessionID: "sess-123"` set, assert
   the output frontmatter contains **no** `branch:` and no `session:` line. This is the
   direct regression guard for Change 1 and locks the fix.
2. **Cross-branch idempotence (integration):** simulate the CI scenario — project NEXT.md
   with `Branch: "feature/x"`, then re-project with `Branch: "main"`, assert the two
   outputs are byte-identical after ignoring `updated:` (i.e. the gate's diff would be
   empty). This reproduces the reported failure and proves it fixed.
3. **Gate purpose preserved (negative):** hand-edit NEXT.md's managed region stale
   (simulate a hook-less commit), run the projection + gate diff, assert it still diverges
   — the gate must still catch a genuinely stale/missing projection (mirror prior spec AC-4).

### Regression scope
- Anything that reads NEXT.md frontmatter — verified **none** reads `branch:`/`session:`.
- The pre-commit hook + merge driver (`next-merge-driver-*`, `pre-commit-auto-stage-next`)
  operate on the managed region / whole file, not the branch line — unaffected.
- Handoff continuity round-trip (A→B) — unaffected; no consumer of the removed lines.
- Real acceptance: push to `main` and confirm the `Test` job goes green
  (`gh run list --workflow=Test --branch main`).

---

## Kickoff

**Pick up at: DIAGNOSED — ready to fix.** The CI "NEXT.md projection drift gate"
(`.github/workflows/test.yml:43-64`) is red again on `main`. Root cause is fully
reproduced: `internal/projection/projection.go:90-92` stamps a live
`branch: <git rev-parse --abbrev-ref HEAD>` line into the byte-gated `.hero/NEXT.md`.
The committer/concurrent-session's branch (e.g. `fix/mcp-transport-closes-midsession-supersede`
in HEAD `7a02a00`) gets baked in; CI re-projects on `main` → `branch: main`; the gate
ignores `updated:` but not `branch:`, so `git diff --exit-code -I'^updated: '` returns
exit 1. This is a recurrence of the completed spec `next-drift-gate-unwinnable`, which
fixed the `updated:` timestamp and size-drift count but left `branch:` (same defect
shape); the prior spec even rejected the "ignore volatile lines in the gate" approach as
fragile — this bug is that prediction realized.

**Do this:** apply Change 1 — delete the `branch:` (and latent `session:`) emission in
`internal/projection/projection.go` `NextMD` (~line 87-92). Nothing consumes these
(`Branch` is `// frontmatter only`); live branch is available via `git status`, and
per-session context belongs in the gitignored `.hero/next/<user>.local.md`. Add the
regression guard `TestNextMD_NoEnvironmentLocalFrontmatter` (assert no `branch:`/`session:`
line even when the options set them) plus a cross-branch idempotence test (project with
`Branch: feature/x` then `Branch: main`, assert byte-identical modulo `updated:`).

**Verify:**
```
go build -o ./hero ./cmd/hero && ./hero scan >/dev/null 2>&1 || true
./hero next checkpoint --quiet
git diff --exit-code -I'^updated: ' -- .hero/NEXT.md   # must be exit 0
go test ./internal/projection/... ./internal/cli/...
```
Then commit (stage `.hero/NEXT.md` + `.hero/next/*.md` per the handoff-travels-with-commits
rule) and confirm the `Test` job turns green on the next push to `main` — that green run is
the real acceptance. Do **not** re-adopt the enumerate-volatile-lines gate hack (Change 2)
unless the team explicitly wants to keep the branch line visible; removal is the root-cause
fix. Deferred scrub: collapse the two `currentBranch` helpers
(`checkpoint.go:1136` vs `gitutil.CurrentBranch`) and drop the now-inert `Branch`/`SessionID`
options + call-site assignments.
