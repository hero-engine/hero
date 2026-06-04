---
title: "Harden CLI test isolation against stray hero workspaces"
slug: cli-test-isolation-stray-workspace-boundary
type: enhancement
status: planning
domain: engineering
size: small
created: 2026-06-03
tags: [test-infrastructure, workspace, locate, ci-reliability]
---

# Harden CLI test isolation against stray hero workspaces

## Context

During the v0.15.3 release, `make test` failed: 17 tests in the `internal/cli`
package failed and the package hit a 10-minute timeout panic (`cli.test` ran
the full 600s). No code change caused this. The actual cause was a leaked
workspace at `/private/tmp/.hero` — a full tree (`graph.db` 98KB, `index.db`
245KB, `knowledge/`, `planning/`, `peer-manifest.yaml`, dated Jun 2–3) sitting
in the macOS temp root. Removing it made the full 95-package suite pass in
seconds.

The defect is structural: the CLI test suite is hostage to any stray
`/tmp/.hero`. On macOS `t.TempDir()` returns a path under `/private/tmp/...`,
so the workspace upward-walk ascends past the test's temp dir and discovers
`/private/tmp/.hero`, making commands believe they are inside a real
workspace. Tests that assert "no hero workspace" then fail, and commands that
walk the tree (`scan`) hang on the large stray tree until the test timeout
fires.

Root cause, verified in code:

1. `internal/cli/root.go:226` `findProjectRoot()` calls
   `workspace.LocateFromCWD()`.
2. `internal/workspace/locate.go:159` `LocateFromCWD()` calls `Locate(cwd)`
   with no options, so the parent walk ascends all the way to filesystem root.
3. `Locate(startDir, opts...)` (locate.go:85) walks up from `startDir`
   checking each directory for `.hero/` via `isHeroRoot` (locate.go:172) and
   for satellite markers. The walk ascends past the test temp dir and finds
   `/private/tmp/.hero`.
4. The harness (`internal/cli/helpers_test.go`): `newTestEnv` (line 26) and
   `newTestEnvEmpty` (line 87) both `t.TempDir()` + `os.Chdir(dir)` but set no
   boundary. `runCmd` (line 160) executes via the shared `rootCmd`.
   `TestScanRequiresWorkspace` (scan_test.go:183) calls `newTestEnvEmpty(t)`
   then asserts `scan` returns an error containing "no hero workspace" — which
   fails when the stray workspace is discovered above the temp dir.

The key seam already exists. `Locate` supports `WithStopAt(dir)`
(locate.go:62–70, `LocateOption` at line 56, honored at line 145:
`if stopAt != "" && dir == stopAt { break }`). The boundary is inclusive — it
inspects the `stopAt` dir itself, then stops before its parent.
**`LocateFromCWD` simply never passes it.** There is currently no
environment-variable boundary anywhere in `workspace/` or `root.go`. So the
fix is wiring an existing mechanism plus giving tests a way to set the
boundary — not building new infrastructure.

Blast radius of `LocateFromCWD` / `workspace.Locate` callers (7 files):
`internal/workspace/locate.go`, `internal/cli/{context,status,install_satellites,note,install,root}.go`.

## Goal

The `internal/cli` test suite is immune to a stray `.hero` workspace existing
anywhere above the test's temp directory. A CLI command run from a test temp
dir never discovers a workspace above the test boundary; `*RequiresWorkspace`
commands report "no hero workspace" even when a stray `.hero` sits in a parent
of the working directory. Production workspace discovery is unchanged when no
boundary is configured. A new regression test — creating a stray `.hero` in a
parent of the test temp dir — fails on today's code and passes after the fix.

## Kickoff

Stops the CLI test suite from discovering a stray `/tmp/.hero` by wiring an
env-var boundary into the workspace upward-walk and setting it from the test
harness.

**Status:** planning — spec just landed, no code yet. Boundary machinery
(`WithStopAt`) already exists; `LocateFromCWD` never passes it.

**Pick up at:** add a `HERO_WORKSPACE_BOUNDARY` env read inside
`LocateFromCWD` (locate.go:159) that forwards to `WithStopAt`, then have
`newTestEnv`/`newTestEnvEmpty` set it via `t.Setenv` to the temp dir. Add the
parent-stray regression test last.

→ `.hero/planning/features/cli-test-isolation-stray-workspace-boundary/spec.md`

**Files:** `internal/workspace/locate.go:85,145,159`, `internal/cli/root.go:226`, `internal/cli/helpers_test.go:26,87`, `internal/cli/scan_test.go:183`
**Skip:** building new boundary infra — `WithStopAt` already exists. Changing prod discovery semantics — out of scope unless clearly safe.

## Problem

Two layers must be addressed.

**Layer 1 — fix the test isolation.** The harness gives the upward walk no
boundary, so it can escape the test's temp root and find `/private/tmp/.hero`.
This is the direct cause of the 17 failures and the timeout.

**Layer 2 — harden so this class can't recur.** There is no regression test
that would have caught this. The bug is invisible on a clean machine and only
surfaces when a dev (or CI runner) has left a stray `.hero` in the temp root —
exactly the kind of environment-dependent flakiness that erodes trust in the
suite.

## Approach

### Mechanism choice — env-var boundary read in `LocateFromCWD`

Add an environment variable, `HERO_WORKSPACE_BOUNDARY`, read inside
`LocateFromCWD`. When set and non-empty, `LocateFromCWD` forwards it as
`WithStopAt(boundary)` to `Locate`. When unset (the production default),
behavior is byte-for-byte unchanged — `Locate(cwd)` with no options, walking to
filesystem root exactly as today.

`findProjectRoot` (root.go:226) calls `LocateFromCWD` first, so it inherits the
boundary automatically for its primary path. Its second loop — the `.git`
upward-walk fallback (root.go:234–244) — must get the same boundary treatment
for consistency (see Layer 2; `TestFindProjectRoot_WithGitDir` was among the
failures).

The harness (`newTestEnv`, `newTestEnvEmpty`) sets the variable via
`t.Setenv(...)` to the test's temp dir. Because the boundary is inclusive, a
boundary set to the temp dir lets `Locate` inspect the temp dir itself (so a
workspace the test legitimately created there is still found) while refusing to
ascend into `/private/tmp` and beyond.

#### Why this approach over the alternatives

- **Threading `WithStopAt` through an injectable locator.** Cleaner in theory,
  but the CLI calls `LocateFromCWD` through `findProjectRoot` and six other
  call sites that just want a workspace; injecting a locator means plumbing a
  seam through all of them or introducing a package-level hook. Heavier blast
  radius for a test-only need. Rejected as overbuilt for the mission.
- **Harness creates a sentinel `.hero` in the temp dir.** Would make
  `newTestEnvEmpty` no longer empty, defeating the very assertion
  (`TestScanRequiresWorkspace`) we need to keep honest. Rejected.
- **Env-var boundary (chosen).** Smallest seam. Zero production behavior change
  when unset. One read site (`LocateFromCWD`), one mirror in the `.git`
  fallback, two harness lines. The variable name doubles as future-proofing: a
  CI job or container that wants to bound discovery can set it without code
  changes.

#### Test-parallelism implication (must call out)

`t.Setenv` forbids `t.Parallel` in the same test or any parent. Today the
`internal/cli` tests do **not** use `t.Parallel` (they share `rootCmd` and a
mutex-guarded stdout capture in `captureStdout`, helpers_test.go:138–157, which
already serializes them). So adding `t.Setenv` to `newTestEnv` /
`newTestEnvEmpty` costs nothing today. The change does, however, **lock in**
that `internal/cli` tests cannot become parallel without removing the env-var
boundary first. This is an acceptable constraint — the shared `rootCmd` already
prevents parallelism — but it must be documented in a comment on the harness
helpers so a future contributor who tries to add `t.Parallel` understands why
it fails.

### Production-boundary question — raise, don't decide

Should production `LocateFromCWD` also stop at a sensible boundary (never
ascend above `$HOME` or `os.TempDir()`)? There is a real argument that a
`hero` command run from a deep temp path discovering an unrelated `.hero` is a
latent prod footgun, not just a test problem. But the mission-fit here is test
reliability, not prod semantics, and changing the prod upward-walk is a
behavior change with its own blast radius across the 7 caller files. **Default
to NOT changing production discovery** in this spec. Capture the question in
Boundaries so it can be picked up as a separate, deliberately-scoped decision.

## Changes

1. **Add the env-var boundary read in `internal/workspace/locate.go`.**
   - In `LocateFromCWD` (line 159), read `os.Getenv("HERO_WORKSPACE_BOUNDARY")`.
   - If non-empty, call `Locate(cwd, WithStopAt(boundary))`; otherwise call
     `Locate(cwd)` exactly as today.
   - Add a doc comment naming the variable, stating it is unset in production
     and exists to bound the upward walk (primarily for tests and bounded CI
     environments). Document that the boundary is inclusive (matches
     `WithStopAt` semantics at line 62–70).
   - Define the variable name as a package constant (e.g.
     `EnvWorkspaceBoundary = "HERO_WORKSPACE_BOUNDARY"`) so the harness and any
     future caller reference the same symbol rather than a string literal.

2. **Mirror the boundary in `findProjectRoot`'s `.git` fallback,
   `internal/cli/root.go`.**
   - The primary path (line 227, `LocateFromCWD`) already inherits the boundary
     via change 1. The `.git` upward-walk fallback (lines 234–244) does not.
   - Read the same `HERO_WORKSPACE_BOUNDARY` (via the `workspace` package
     constant) and, when set, break the `.git` loop once `dir == boundary`
     (inclusive — check the boundary dir, then stop), mirroring the
     `WithStopAt` semantics. When unset, the loop is unchanged.
   - This keeps `TestFindProjectRoot_WithGitDir` honest: a stray `.git` above
     the boundary must not be discovered either.

3. **Set the boundary in the CLI test harness, `internal/cli/helpers_test.go`.**
   - In `newTestEnv` (line 26): after `t.TempDir()` (line 29) and before the
     `os.Chdir`, call `t.Setenv(workspace.EnvWorkspaceBoundary, dir)`.
   - In `newTestEnvEmpty` (line 87): same — `t.Setenv(...)` to the temp `dir`.
   - Add a short comment on both helpers explaining the boundary prevents the
     upward walk from escaping the temp dir into `/tmp/.hero`, and noting that
     `t.Setenv` is why these tests (and the `internal/cli` package generally)
     must not call `t.Parallel`.

4. **Add the parent-stray regression test in `internal/cli`.**
   - New test (e.g. `TestLocate_StrayWorkspaceAboveBoundary` in
     `helpers_test.go`'s package, or a focused `locate_isolation_test.go`).
   - Construct a parent temp dir, create a full-enough `.hero/` inside it
     (enough that `isHeroRoot` returns true), create a child dir, `chdir` into
     the child, set the boundary to the child via the harness path, and assert
     a `*RequiresWorkspace`-style command (e.g. `scan`) still returns an error
     containing "no hero workspace".
   - This is the test that would have caught the original bug: it must **fail**
     against today's unbounded `LocateFromCWD` and **pass** after change 1.
   - Keep it non-parallel (it uses `t.Setenv`).

5. **Defensive note on the timeout dimension (no new infra).**
   - The boundary fixes the root cause of the 10-minute hang — `scan` was
     walking the huge stray tree. With the boundary in place the hang cannot
     recur from this cause, so do **not** add a per-package `go test` timeout
     as the fix.
   - Add a one-line comment in the harness (or a brief note in the
     `internal/cli` test docs if one exists) recording that the boundary is
     what keeps the suite from hanging on stray-tree discovery, so a future
     reader understands the coupling.

## Acceptance Criteria

- WHEN a CLI command runs from a test temp dir THE SYSTEM SHALL NOT discover a
  `.hero` workspace located above the test boundary.
- IF a stray `.hero` exists in a parent of the test working directory THEN THE
  SYSTEM SHALL still report "no hero workspace" for `*RequiresWorkspace`
  commands.
- THE SYSTEM SHALL leave production workspace-discovery behavior unchanged when
  `HERO_WORKSPACE_BOUNDARY` is unset.
- WHEN `HERO_WORKSPACE_BOUNDARY` is set THE SYSTEM SHALL forward it to
  `Locate` as an inclusive `WithStopAt` boundary so the boundary directory
  itself is still inspected.
- IF a stray `.git` directory exists above the boundary THEN
  `findProjectRoot`'s fallback walk SHALL NOT ascend past the boundary to
  discover it.
- WHEN the suite runs with a stray `.hero` planted in a parent temp dir THE
  SYSTEM SHALL complete `TestScanRequiresWorkspace` and the new
  parent-stray regression test as passing.
- THE SYSTEM SHALL not introduce `t.Parallel` into `internal/cli` tests that
  set the boundary via `t.Setenv`.

## Mission-fit

Hero's premise is that the floor rises for everyone — agents and humans inherit
a workspace they can trust. A test suite that silently breaks because some dev
left a `/tmp/.hero` behind violates that contract at the most basic level:
green should mean green, on every machine, regardless of a contributor's stray
local state. CI shouldn't be hostage to temp-dir pollution. This change makes
the `internal/cli` suite deterministic with respect to the surrounding
filesystem — a small, well-bounded hardening that removes a whole class of
"works on my machine / mysteriously red in release" flakiness.

## Boundaries

- **Not changing production workspace-discovery semantics.** `LocateFromCWD`
  with no boundary set still walks to filesystem root. The question of whether
  prod should bound the walk at `$HOME`/`os.TempDir()` is explicitly deferred
  to a separate decision spec — surfaced here, not decided here.
- **Not adding a per-package `go test` timeout** as the remedy. The boundary
  removes the cause of the hang; a timeout would only mask a different future
  hang and is out of scope.
- **Not refactoring the seven `LocateFromCWD`/`Locate` callers** beyond the env
  read in `LocateFromCWD` and the mirrored boundary in `findProjectRoot`'s
  `.git` fallback. No injectable-locator plumbing.
- **Not making `internal/cli` tests parallel.** This spec locks in
  non-parallelism via `t.Setenv`; it does not attempt to remove the shared
  `rootCmd`/stdout-mutex coupling that already serializes them.
- **Not cleaning up the stray `/tmp/.hero`** as a code concern — that was a
  one-time manual cleanup. The fix makes the suite immune regardless of whether
  such a tree exists.

## Risks

- **Boundary too tight breaks legitimately-nested tests.** If any existing CLI
  test relies on discovering a workspace in a parent of its `chdir` target
  (e.g. satellite-marker tests that walk up to a linked root), setting the
  boundary to the immediate temp dir could break them. Mitigation: the boundary
  is set to the test's top-level `t.TempDir()` root, not a nested child, so
  walks within the temp tree still succeed; audit satellite/monorepo tests
  during delivery and, if any set up a parent-root relationship, set their
  boundary to the appropriate ancestor rather than the leaf.
- **Env-var leakage between tests.** `t.Setenv` restores the prior value on
  cleanup, so there is no cross-test leakage as long as every helper that
  chdirs also sets the boundary. Both `newTestEnv` and `newTestEnvEmpty` must
  set it; any test that builds its own temp dir without these helpers must set
  the boundary itself or it remains exposed to the original bug.
- **`findProjectRoot` fallback divergence.** If the `.git` fallback boundary
  read is forgotten, `TestFindProjectRoot_WithGitDir` stays vulnerable. Change
  2 is not optional polish — it is required for the `.git` path to match the
  `.hero` path.
- **Rollback implication.** This change is additive and gated on an env var
  that is unset in production. Reverting it restores the prior (buggy) test
  behavior with zero production impact — there is no data migration, no
  persisted state, and no on-disk format change. Rollback is a clean revert of
  the three source files plus the new test.

## Validation

- Run `go test ./internal/cli/...` on a clean machine — full pass, fast.
- Reproduce the original failure mode: create `$(go env GOTMPDIR ||
  echo /tmp)/.hero` (a minimal `.hero/` dir is enough for `isHeroRoot`), run
  `go test ./internal/cli/...` against the **pre-fix** code and confirm
  `TestScanRequiresWorkspace` (and the new regression test) fail; then run
  against the **post-fix** code and confirm they pass. Remove the planted
  `.hero` afterward.
- Confirm the new `TestLocate_StrayWorkspaceAboveBoundary` fails on the
  unbounded `LocateFromCWD` (sanity that it actually exercises the bug) and
  passes after change 1.
- Run the full `make test` (95 packages) and confirm no regressions and no
  timeout.
- Confirm production unchanged: run a real `hero status` from a deep path with
  `HERO_WORKSPACE_BOUNDARY` unset and verify it still resolves the actual
  workspace exactly as before.
