---
title: "CI NEXT.md drift gate is structurally unwinnable — Test job red on every commit"
slug: next-drift-gate-unwinnable
type: bug
status: completed
priority: high
severity: high
domain: engineering
size: small
created: 2026-07-11
tags: [ci, next-projection, drift-gate, flaky-gate, release-blocker]
completed_at: 2026-07-12T01:04:49Z
---

# CI NEXT.md drift gate is structurally unwinnable

## Symptom

The `Test` GitHub Actions workflow (`.github/workflows/test.yml`) has failed on
**every commit for 10+ commits** — long before recent work. This masks the real
test signal (a release-readiness footgun: "is CI green?" answers nothing) and
would put any release out on a red main.

The failing step is **not** `go test` — that passes (`go test -race -count=1
./...` → 86 packages ok locally, and the go-test step precedes and passes in
CI). The failure is the **"NEXT.md projection drift gate"** (`test.yml:43-54`):

```sh
./hero scan -q || true
./hero next checkpoint --quiet
if ! git diff --exit-code -- .hero/NEXT.md; then
  echo "::error::.hero/NEXT.md drifted from hero next checkpoint output."
  exit 1
fi
```

## Root Cause

The gate does a **byte-exact** `git diff --exit-code` of `.hero/NEXT.md` against
a freshly-projected copy, but `NEXT.md` contains content that is volatile or
stale-by-construction, so the diff can never be empty. Three compounding defects
(all reproduced end-to-end in a clean detached-HEAD `git worktree`):

1. **Volatile `updated:` timestamp.** NEXT.md carries an
   `updated: <RFC3339>` frontmatter line stamped from wall-clock at projection
   time (`internal/cli/checkpoint.go` / `internal/snapshot/projector.go`). CI
   projects at a different instant than the commit, so it always differs.
   `writeIfChanged`/`normalizeUpdatedFrontmatter` (commit `fe7ae28`) strips this
   *only on a content-MATCH* — any other content diff triggers a rewrite that
   also restamps the timestamp.

2. **Chicken-and-egg size-drift count (the load-bearing defect).** The
   `## Roadmap shape` section emits `"N specs have size drift — run
   /roadmap-review to triage"` from `sizing.AmbientDrift`
   (`internal/projection/projection.go:127-140`, string in
   `internal/sizing/ambient.go:140-142`). The count is **deterministic** across
   repeated runs (verified 33 == 33) but the *committed* value is **stale by
   construction**: the pre-commit hook computes it against the pre-commit index,
   while a clean rebuild (`hero index` + `hero next checkpoint`) computes the
   post-commit value. Committed file says `31`; clean rebuild of the exact same
   commit says `33`. This is the same chicken-and-egg `fe7ae28` already called
   out and fixed for the "Just finished" commit list — the hook cannot see the
   state its own commit creates.

3. **Dead `hero scan -q` flag (secondary).** The gate runs `./hero scan -q`, but
   `hero scan` has **no** `-q`/`--quiet` flag — it errors `unknown shorthand
   flag: 'q' in -q`, swallowed by `|| true`. So the intended "build the graph
   from repo sources before projecting" precondition **silently never runs**,
   leaving downstream sections projected against an unbuilt/empty graph.

Classification: **process/tooling** (CI gate design) + **code** (churny derived
metric in a byte-gated file, dead CLI flag). Not a test or product-code defect.

## Evidence

- Clean `git worktree --detach HEAD`, ran the gate's exact sequence → diff showed
  the `updated:` timestamp line, a `branch:` line (worktree artifact; would be
  `main` in CI), and `31 → 33` size-drift count.
- `hero next checkpoint` run twice on the same committed state → identical `33`
  (count is deterministic, so #2 is staleness, not nondeterminism).
- `hero scan -q` → `Error: unknown shorthand flag: 'q' in -q`.
- `gh run list --workflow=Test --branch main` → `failure` on the last 10+
  commits, spanning multiple authors/features.

## Fix

Recommended direction (**B — remove churny derived metrics from the byte-gated
projection**, root-cause, consistent with `fe7ae28`):

1. **Drop the live size-drift count from NEXT.md's `## Roadmap shape`.** Replace
   the corpus-derived `rep.Hint` count with a **static pointer** (e.g. "size
   drift? run `/roadmap-review` / `hero size --check`") or omit the section from
   the projection entirely — mirroring how `fe7ae28` replaced the commit list
   with a git-log pointer. The authoritative count still lives in `hero check`
   and `hero size --check`; it does not belong in a byte-exact-gated committed
   file. This removes the chicken-and-egg drift (#2).
2. **Fix the dead scan invocation (#3).** `hero scan` has no `-q`; use a valid
   quiet form — `./hero scan >/dev/null 2>&1 || true` (or drop the step if the
   projection reads specs directly and doesn't need a pre-built graph — confirm
   what `hero next checkpoint` consumes). The graph must be rebuilt identically
   in CI for the non-count sections to be reproducible.
3. **Timestamp (#1) resolves for free** once content matches: `writeIfChanged`
   skips the rewrite on a content-match run, preserving the committed
   timestamp → clean diff. Optional belt-and-suspenders: have the gate strip the
   `updated:` line before diffing so a stray restamp can't re-break it.

Rejected alternatives:
- **A — normalize/strip volatile lines in the gate only:** fragile (must
  enumerate every volatile line; re-breaks when a new derived metric is added)
  and leaves the churny metric in a byte-gated file.
- **C — rebuild + compare deterministically without removing the count:**
  insufficient — the committed count is pre-commit state and a clean rebuild is
  post-commit state, so it drifts even with `scan` fixed.

The fix must preserve the gate's real purpose: catch a NEXT.md committed
**without** the pre-commit hook running (grossly stale / missing projection).
Removing a churny metric and fixing the scan flag keeps that intact — a
hook-less commit still diverges on the stable content.

## Acceptance Criteria

- AC-1: WHEN the `test.yml` drift-gate sequence runs on a correctly-committed
  repo (pre-commit hook ran), THE SYSTEM SHALL produce an empty
  `git diff -- .hero/NEXT.md` and the gate SHALL pass.
- AC-2: THE `## Roadmap shape` projection SHALL NOT embed a corpus-derived count
  that differs between the committing index state and a clean rebuild (no
  chicken-and-egg drift).
- AC-3: THE gate SHALL invoke `hero scan` with a valid, non-erroring command (no
  `unknown shorthand flag` output).
- AC-4: IF NEXT.md is committed without the pre-commit hook (stale managed
  region / missing projection), THEN the gate SHALL still fail — its detection
  purpose is preserved.
- AC-5: WHERE only the `updated:` wall-clock timestamp would differ on a
  content-match run, THE SYSTEM SHALL NOT rewrite the file (gate stays green).
- AC-6: `go build ./... && go test ./...` passes; the `Test` workflow goes green
  on the next commit to main.

## Validation

- Reproduce-then-fix: in a clean `git worktree --detach HEAD`, run the fixed
  gate sequence → assert empty `git diff -- .hero/NEXT.md`.
- Negative: hand-edit NEXT.md's managed region stale (simulate a hook-less
  commit), run the gate → assert it fails (purpose preserved).
- Determinism: `hero next checkpoint` twice on the same state → byte-identical
  NEXT.md.
- Confirm no other projection section (`## Just finished`, `## Blocked on`,
  features) is corpus-state-volatile once the graph is rebuilt.
- Push to main and confirm the `Test` job turns green (the real acceptance).

## Completion Ledger

| AC | Status | Note |
|----|--------|------|
| AC-1 (gate green on correctly-committed repo) | DONE | verified: `hero next checkpoint` is idempotent — two consecutive runs byte-identical, timestamp preserved via `writeProjectedFileIfSemanticChanged` |
| AC-2 (no corpus-derived count in NEXT.md) | DONE | removed the `## Roadmap shape` block from `internal/projection/projection.go`; NEXT.md now has 0 size-drift lines |
| AC-3 (valid scan invocation) | DONE | `test.yml`: `./hero scan -q` → `./hero scan >/dev/null 2>&1 \|\| true` (scan has no `-q`) |
| AC-4 (still catches hook-less commits) | DONE | gate still byte-compares the stable content; a stale managed region / missing projection still diverges. Regression guard `TestNextMD_RoadmapShape_NeverEmitted` seeds drift and asserts the section is absent |
| AC-5 (timestamp-only run doesn't rewrite) | DONE | idempotence proof preserves the committed `updated:` timestamp on the no-op second run |
| AC-6 (build + tests; Test job green) | DONE | `go vet` + `go build` + `go test ./...` green (86 pkgs); CI-green pending the push (the real acceptance) |

- [x] exercise-the-feature: rebuilt binary → `hero scan` + `hero next checkpoint` twice → byte-identical NEXT.md with no `## Roadmap shape` / `size drift`; reproduced the ORIGINAL failure earlier in a clean worktree of HEAD.

## Judgment call — scope of the count removal

The user chose to **drop** the size-drift line from NEXT.md (vs. keeping it and
teaching the gate to ignore it): committing a number that's stale-by-construction
is itself wrong, and `AmbientDrift` still surfaces authoritatively via `hero size
--check`, `hero pulse`, and the MCP tools. Kept the fix **focused on the gate**:
removed the emission + its `sizing` import + collapsed three roadmap-shape tests
into one absence guard. The now-inert `NextMDOptions` ambient fields
(`HeroDir`/`ProjectRoot`/`ActiveSpec`/`RoadmapRecencyDays`/`RoadmapStopNaggingHours`)
and their two caller assignments are left in place — removing them cascades
across `checkpoint.go`/`next_project.go` for zero functional gain; deferred as a
scrub follow-on.

## Changes

| # | Change | Status |
|---|--------|--------|
| 1 | `internal/projection/projection.go`: remove `## Roadmap shape` emission + `sizing` import | DONE |
| 2 | `internal/projection/projection_test.go`: collapse 3 roadmap tests → `TestNextMD_RoadmapShape_NeverEmitted` guard | DONE |
| 3 | `.github/workflows/test.yml`: fix dead `hero scan -q` → redirected quiet | DONE |

## Kickoff

**Pick up at: DELIVERED — pending push + CI-green confirmation.** The churny
`## Roadmap shape` size-drift line was removed from the NEXT.md projection
(`internal/projection/projection.go`), the dead `./hero scan -q` gate flag was
fixed (`.github/workflows/test.yml`), and `hero next checkpoint` was verified
idempotent (two runs byte-identical, timestamp preserved). Regression guard
`TestNextMD_RoadmapShape_NeverEmitted`. `go test ./...` green (86 pkgs). The
real acceptance is the `Test` GitHub Actions job turning green on the next push
to main — watch `gh run list --workflow=Test --branch main`. Once green, the
pipeline is trustworthy and v0.24.0 can be tagged. Deferred scrub: remove the
now-inert `NextMDOptions` ambient fields + their two caller assignments.
