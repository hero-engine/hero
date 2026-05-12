---
title: Multi-Harness Install Collision — Second Target Refuses Identical Canonical Content
type: bug
status: delivering
severity: medium
priority: high
created: 2026-05-12
tags: [install, canonical, idempotency, onboarding]
---

# Multi-Harness Install Collision — Second Target Refuses Identical Canonical Content

## Issue

A user installing Hero into more than one AI tool in the same project hits a
hard failure on the second `hero install`:

```
$ cd <project-with-.hero/>
$ hero install project . --target claude   # succeeds
$ hero install project . --target codex
installing canonical agents: refusing to overwrite .hero/agents/api-engineer.md
(use --force to replace)
```

The first install materializes the canonical content tree at
`.hero/agents/`, `.hero/commands/`, and `.hero/skills/`. The second install,
on a different harness target, re-runs the same canonical materialization
step against the same destinations and the existence check fails.

User-facing impact: blocks the multi-harness install flow that is the
documented onboarding path. Every user installing Hero into a second AI
tool hits this — the current workaround is to pass `--force` on every
install after the first. The workaround is safe in the common case
(byte-identical content from the same `hero` binary) but it is undocumented
friction at the most visible step in onboarding.

Reporter: internal (Hero team), via `GETTING-STARTED.md` and
`docs/getting-started/project-setup.md`. No tracker configured.

## Investigation

### Code flow (end to end)

1. `cmd/hero/...` parses `hero install project . --target codex` and calls
   `install.Run(opts)` with `Mode: ModeProject`, `TargetDir: "."`,
   `Force: false`.
2. `internal/install/install.go` `Run` dispatches to the per-target
   installer (e.g. `target_codex.go`), and as part of project-mode setup
   calls `installCanonical(opts, result)`.
3. `internal/install/canonical.go:97` `installCanonical` materializes
   embedded source content into `.hero/agents`, `.hero/commands`, and
   `.hero/skills` whenever the kind's resolved canonical path lives
   inside `.hero/`. Its docstring (lines 83–96) explicitly promises:
   *"Re-running this is idempotent when content is unchanged."*
4. For each kind, materialization calls `installFlat` /
   `installSkillsNested` in `internal/install/content.go`, which iterates
   embedded files and calls `copyFileFromFS` per file.
5. `internal/install/files.go:14` `copyFileFromFS`, on its second
   invocation against an existing destination, hits:
   ```go
   if _, err := os.Stat(dst); err == nil && !opts.Force {
       result.Skipped = append(result.Skipped, dst)
       return fmt.Errorf("refusing to overwrite %s (use --force to replace)", dst)
   }
   ```
   The check is existence-only — no content comparison — so it fires even
   when the source bytes equal the destination bytes.
6. The error bubbles back through `installCanonical` as
   `installing canonical agents: refusing to overwrite ...` and aborts
   the second install.

### Reproduction

```sh
mkdir tmp && cd tmp
hero init                          # create .hero/ workspace
hero install project . --target claude   # OK
hero install project . --target codex    # FAILS with the message above
```

### Key files and lines

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/install/files.go` | 14–49 | `copyFileFromFS` — existence-only guard. Site of the fix. |
| `internal/install/canonical.go` | 83–126 | `installCanonical` — declares the idempotency contract the current `copyFileFromFS` violates. |
| `internal/install/content.go` | (callers) | `installFlat` / `installSkillsNested` — call `copyFileFromFS` per embedded file. |
| `internal/install/install_test.go` | 120–178 | `TestRunRefusesOverwrite` / `TestRunForceOverwrite` — existing tests; the first must be retargeted onto a content-drift scenario when the behavior changes. |

### Root cause

This is a **regression by omission** under the
`root-cause-classification` skill: the idempotency contract was written
into the canonical docstring but never implemented at the file-copy
layer. `copyFileFromFS` was authored as a strict overwrite guard
(existence-only), which is the right behavior for cases where the
destination is *user-edited* (config files, prompts, etc.) but the
wrong behavior for canonical content that the installer itself owns
and that is intentionally re-materialized on every invocation.

The two requirements collided silently because the guard was added
generically — not scoped to "user-owned" vs "canonical-owned"
destinations — and content equivalence was never considered.

### Severity

- **Criticality**: medium. No data loss, no broken installs once the
  user knows about `--force`. But it gates the documented multi-harness
  flow and presents as a scary error to first-time users.
- **Frequency**: hits every user with more than one AI tool. Onboarding
  docs (`GETTING-STARTED.md`, `docs/getting-started/project-setup.md`)
  explicitly walk users through the multi-harness path.
- **Ease of fix**: easy. Single function, ~10 lines, plus a test
  update.
- **Caused by our code**: yes.
- **Workaround**: `--force`; safe when content is byte-identical, which
  it is for the same `hero` binary.

## Goal

A second `hero install` against the same project with a different
`--target` succeeds without `--force`, provided the canonical content
on disk is byte-identical to what the install would write. Content
drift (a user has hand-edited a canonical file, or two different
`hero` binaries disagree on canonical content) continues to trigger
the existing refusal so users do not silently lose changes.

## Changes

1. **Make `copyFileFromFS` content-aware in `internal/install/files.go`.**
   - When the destination exists and `--force` is not set, read the
     source bytes (via `srcFS.Open` / `io.ReadAll`) and the destination
     bytes (via `os.ReadFile`).
   - If `bytes.Equal(src, dst)`, treat the operation as a successful
     no-op: do NOT append to `result.Skipped`, do NOT append to
     `result.Copied` (or append to a new `result.Unchanged` slice if
     that fits the existing `Result` shape — check `internal/install/install.go`
     first), and return `nil`.
   - If the bytes differ, preserve the current behavior verbatim:
     append to `result.Skipped` and return the
     `refusing to overwrite %s (use --force to replace)` error.
   - Keep the `DryRun` and `MkdirAll`/`os.Create`/`io.Copy` paths
     unchanged; only the existence-check branch is modified.

2. **Update the canonical docstring in `internal/install/canonical.go`
   only if the wording in lines 83–96 no longer reads cleanly after
   the fix.** The current "Re-running this is idempotent when content
   is unchanged" statement becomes accurate without edits, so leave it
   alone unless a reviewer asks for a tightening.

3. **Update `TestRunRefusesOverwrite` in
   `internal/install/install_test.go` (lines 120–148).**
   - The existing test runs `Run(opts)` twice against an unchanged
     source, which under the new contract MUST succeed. Re-target the
     test to mutate one of the destination canonical files between
     runs (e.g. append a byte to `.opencode/agents/engineer.md` after
     the first install) so the second `Run` is truly a drift case
     that should refuse.

4. **Add a new test `TestRunIdempotentReinstall` (or similar) covering
   the bug scenario directly.**
   - Run the project install twice against the same source with
     `Force: false`. Assert the second `Run` returns `nil` error.
   - Cross-target variant: first install with `--target claude`, second
     with `--target codex` (or `--target opencode`) on the same
     `targetDir`. Assert both succeed without `--force`.

5. **Smoke-check the broader callers of `copyFileFromFS`** (grep for
   `copyFileFromFS` across `internal/install/`) and confirm none of
   them rely on the existence-only semantics for *user-owned* files.
   If any do (e.g. a target installer that writes a user-editable
   config), surface that as a `## Risks` note for the delivery agent
   rather than expanding scope here.

## Boundaries

- Do NOT add a `--force` default. The flag still exists for the
  drift case.
- Do NOT change `mergeJSONFromData` in `files.go` — that path handles
  merge semantics for config files and is not implicated.
- Do NOT introduce a new "canonical-only" code path. The fix is at
  the file primitive, not the caller layer.
- Do NOT touch the symlink behavior used by per-target installers —
  the bug is in the canonical materialization, not the linking step.
- Do NOT add a content-hash manifest or any state file. Compare
  in-process bytes.
- Out of scope: the larger question of whether canonical materialization
  should be skipped entirely after the first install. That would be a
  separate design (and would prevent legitimate updates when the `hero`
  binary upgrades). The fix here keeps re-materialization on every
  install but makes it correct.

## Risks

- **Performance**: `copyFileFromFS` now reads both source and
  destination bytes when the destination exists. For the canonical
  tree this is dozens of small files — negligible. Note in the test
  if anyone wires this into a hot path.
- **Embedded FS read cost**: `srcFS.Open` may be called twice if the
  no-op branch falls through to the copy branch. Structure the fix
  so the source bytes are read once and reused for the equality check
  and (if needed) the subsequent write.
- **Race window**: between the equality check and the write, a third
  party could modify the file. Acceptable — same race as the existing
  `os.Stat` check, no worse.
- **Tests touching shared canonical destinations**: confirm no test in
  `internal/install/` relies on the second-call error message in a way
  that survives the fix beyond `TestRunRefusesOverwrite`. Quick
  `rg "refusing to overwrite" internal/install` before landing.

## Validation

- `go test ./internal/install/...` passes, including the updated
  `TestRunRefusesOverwrite` and the new idempotent-reinstall test.
- Manual smoke: in a clean throwaway directory, run
  `hero init && hero install project . --target claude && hero install project . --target codex`
  and confirm the second install exits zero without `--force`.
- Drift smoke: after the two successful installs, edit one file under
  `.hero/agents/`, then run `hero install project . --target claude`
  again. Confirm it still refuses with the
  `refusing to overwrite ... (use --force to replace)` message.
- `--force` smoke: with drift in place, re-run with `--force` and
  confirm the file is overwritten.

## Acceptance Criteria

- WHEN a user runs `hero install project . --target <X>` followed by
  `hero install project . --target <Y>` against the same project with
  the same `hero` binary THE SYSTEM SHALL complete both invocations
  successfully without `--force`.
- WHEN `copyFileFromFS` finds a destination whose bytes equal the
  source bytes and `Force` is false THE SYSTEM SHALL return nil
  without writing.
- IF `copyFileFromFS` finds a destination whose bytes differ from the
  source bytes and `Force` is false THEN THE SYSTEM SHALL append the
  destination to `result.Skipped` and return the
  `refusing to overwrite %s (use --force to replace)` error.
- WHERE `Force` IS ENABLED THE SYSTEM SHALL overwrite the destination
  regardless of content equality.
- THE SYSTEM SHALL preserve all existing passing tests in
  `internal/install/install_test.go` (with `TestRunRefusesOverwrite`
  updated to use a content-drift fixture).

## Kickoff

Fixes the second-`hero install` collision so multi-harness onboarding
works without `--force`.

**Status:** delivering — code change landed, tests added, full suite
green, manual smokes pass. Spec ready to mark complete after commit.

**Pick up at:** confirm the commit looks right, then
`hero spec complete .hero/planning/bugs/multi-harness-install-collision/spec.md`
and push. If something looks off, the fix is contained to
`internal/install/files.go` `copyFileFromFS` — revert that function
and the three install_test.go tests
(`TestRunRefusesOverwriteOnDrift`, `TestRunIdempotentReinstall`,
`TestRunIdempotentAcrossTargets`).

**What changed:**
- `internal/install/files.go` — `copyFileFromFS` now reads source
  bytes once, compares to the existing destination when one exists,
  and returns nil silently if equal. Drift case unchanged.
- `internal/install/install_test.go` — renamed
  `TestRunRefusesOverwrite` to `TestRunRefusesOverwriteOnDrift` (now
  mutates the file between runs to create real drift); added
  `TestRunIdempotentReinstall` and `TestRunIdempotentAcrossTargets`.

**Validation done:**
- `go test ./...` — full suite passes
- `go vet ./...` — clean
- Manual: `hero init && hero install --target claude && hero install --target codex` — second install succeeds without `--force`
- Manual: edit a canonical file, re-install — refuses with the right error
- Manual: same drift + `--force` — overwrites cleanly

→ `hero spec complete .hero/planning/bugs/multi-harness-install-collision/spec.md`

**Skip:** adding a content-hash manifest, gating canonical
materialization on first-install-only, or relaxing the guard for
non-canonical destinations.
