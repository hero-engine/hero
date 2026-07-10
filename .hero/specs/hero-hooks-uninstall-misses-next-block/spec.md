---
title: "`hero hooks uninstall --host=all` leaves `hero next` managed block in pre-commit"
slug: hero-hooks-uninstall-misses-next-block
type: bug
status: completed
severity: low
root_cause_class: design
priority: medium
tags: [hooks, install, uninstall, next, dx]
created: 2026-05-21
relates-to: [next-compact-handoff, compact-handoff-test-coverage]
completed_at: 2026-05-22T00:25:27Z
---

# `hero hooks uninstall --host=all` leaves `hero next` managed block in pre-commit

## Kickoff

`hero hooks uninstall --host=all` removes Hero's tracker-related git hooks and the Claude Code host-tool hook, but **does not** remove the `# >>> hero next hooks (managed) >>>` block that `hero next install-hooks` writes into `.git/hooks/pre-commit` (and the equivalent `.gitattributes` merge-driver block). After uninstall, the user thinks Hero is gone from their git hooks; in reality the projection auto-stage block survives.

**Status:** planning — diagnosis complete, fix not yet written.

**Pick up at:** decide whether to **unify** (move next-hooks logic into `internal/hooks/` so one uninstall covers everything) or **coordinate** (have `hero hooks uninstall` invoke a new `internal/cli/next_hooks.go:uninstallNextHooks` function). Recommendation below leans toward coordinate — smaller change, preserves the existing `hero next install-hooks` ergonomic.

→ [internal/cli/hooks.go](../../../internal/cli/hooks.go), [internal/cli/next_hooks.go](../../../internal/cli/next_hooks.go)

**Files to touch:** `internal/cli/next_hooks.go` (add uninstall), `internal/cli/hooks.go` or `internal/cli/host_hooks.go` (call the new uninstall), plus tests.

## Issue

Surfaced by the engineer during delivery of [compact-handoff-test-coverage](../../features/compact-handoff-test-coverage/spec.md):

> "`hero hooks uninstall --host=all` does NOT remove the `# >>> hero next hooks (managed) >>>` block that `hero init` installs. Those are two different hook systems with different markers... the user-facing semantics of 'uninstall --host=all removes git hooks' is ambiguous in the presence of two parallel git-hook installers."

User-visible impact: someone running `hero hooks uninstall --host=all` (e.g. uninstalling Hero from a project, or troubleshooting hook behavior) reasonably expects Hero's git hooks to be gone. The pre-commit hook still fires Hero on every commit because the managed block survives.

## Investigation

### Three parallel hook installers exist

| Installer | Package | Marker | Wired via |
|---|---|---|---|
| **General git hooks** (tracker/spec-related) | `internal/hooks/install.go` | `# Hero git hook` | `hero hooks install` (no flag) |
| **`hero next` projection hooks** | `internal/cli/next_hooks.go` | `# >>> hero next hooks (managed) >>>` and `# >>> hero next merge driver (managed) >>>` | `hero next install-hooks` |
| **Claude Code host-tool hook** | `internal/hooks/claude_settings.go` | `added_by_hero: true` JSON field | `hero hooks install --host=claude` |

The general installer at `internal/hooks/install.go:118` *does* have an `Uninstall` function that strips the `# Hero git hook` block. The next-hooks installer at `internal/cli/next_hooks.go` has only an install path — there is no `uninstallNextHooks` function.

### Why the split exists historically

These installers serve different purposes:
- General hooks call `hero hook <event>` for tracker/spec auto-routing on `post-checkout`, `post-merge`, `post-commit`, `prepare-commit-msg`.
- `hero next` hooks call `hero next checkpoint`, `hero queue write`, `hero index --if-stale` in `pre-commit`, plus `git add` the projected NEXT files. They also register a `hero-next` git merge driver in `.git/config` and add `merge=hero-next` directives in `.gitattributes`.

The two installers coexist correctly *during install* — they use distinct marker blocks within the same hook file, and `mergeMarkerBlock` preserves content outside markers. The bug is purely on the uninstall side.

### Two reasonable fixes

**Option A — Unify.** Move all git-hook install/uninstall into `internal/hooks/`. One install command, one uninstall, no separate `hero next install-hooks`. Pro: simpler mental model. Con: larger refactor; need to migrate the merge-driver registration too; `hero next install-hooks` is referenced in docs and probably user shell history.

**Option B — Coordinate.** Add `uninstallNextHooks` to `internal/cli/next_hooks.go` that strips the two managed blocks (`# >>> hero next hooks (managed) >>>` and the `.gitattributes` block) and removes the `hero-next` merge driver from `.git/config`. Wire `hero hooks uninstall` (and `--host=all`) to call it. Pro: small, surgical, preserves `hero next install-hooks` as a granular operation. Con: keeps the split.

**Recommendation: Option B.** The bug is real but small. The split has been working fine for install; only uninstall is broken. A targeted uninstall function plus a one-call coordination from `runHooksUninstall` closes the gap without churning the entire install architecture.

### Status reporting

`hero hooks status` (general installer) reports the general hook state. It does not report next-hooks state. Same fix opportunity: extend `runHooksStatus` (or have `host_hooks.go` extend it via the existing wrapper) to also report whether the next-hooks managed block is installed.

### Root cause class — design

Two independent installers with no shared uninstall coordination. Each install path was correct in isolation; the user-facing command `hero hooks uninstall` was never updated to know about the second installer.

## Suggested Fix Approach

### 1. New `uninstallNextHooks` function in `internal/cli/next_hooks.go`

```go
// uninstallNextHooks strips the hero-next managed blocks from
// .git/hooks/pre-commit, .git/hooks/post-merge, and .gitattributes.
// Also removes the hero-next merge driver from .git/config.
// Idempotent — no-op when nothing is installed.
func uninstallNextHooks(projectRoot string) (removed []string, err error)
```

Behavior:
- For each hook file (`pre-commit`, `post-merge`): read, strip the marker block, write back. If the resulting file body is just `#!/bin/sh\nset -e\n` (the shebang we installed) with no other content, remove the file entirely so we don't leave dead hook stubs.
- For `.gitattributes`: strip the `gaMarkerStart`/`gaMarkerEnd` block. If the file is empty after, remove it.
- For `.git/config`: run `git config --unset-all merge.hero-next.driver` and `git config --unset-all merge.hero-next.name`. Best-effort — ignore "key not found" errors.
- Return the list of paths actually modified, for the caller to print.

### 2. Wire into `hero hooks uninstall`

In `internal/cli/hooks.go`, `runHooksUninstall` already removes the general `# Hero git hook` block. After that succeeds, also call `uninstallNextHooks(projectRoot)`. Print each removal.

In `internal/cli/host_hooks.go`, `runHostHooksUninstall` doesn't need a change — when `--host=all`, the wrapper already chains `runHooksUninstall` (which now includes next-hooks) → `runHostHooksUninstall` (host-tool removal).

### 3. Status reporting

Extend `runHooksStatus` to also report next-hooks installation state. Use the existing `preCommitHookInstalled(projectRoot)` and a new equivalent for `.gitattributes` / merge driver. Output style mirrors the existing per-hook lines:

```
  hero next pre-commit block: yes
  hero-next merge driver:     yes
```

### 4. Tests to add

- `TestUninstallNextHooks_RemovesPreCommitBlock`
- `TestUninstallNextHooks_RemovesGitAttributesBlock`
- `TestUninstallNextHooks_UnregistersMergeDriver`
- `TestUninstallNextHooks_PreservesUserContentOutsideMarkers`
- `TestUninstallNextHooks_IdempotentNoOpWhenNotInstalled`
- `TestRunHooksUninstall_AlsoRemovesNextHooks` — install both, run general uninstall, assert both blocks gone
- `TestHostHooksUninstall_AllRemovesNextHooks` — install everything via `--host=all`, uninstall via `--host=all`, assert all three (general / next / claude) gone
- Update the existing `TestHostHooksUninstall_AllRemovesGitAndClaude` to additionally assert next-hooks removal

### 5. Documentation

Update the help text on `hooks uninstallCmd.Short` to mention that next-hooks are included. The current text is "Remove Hero git hooks from .git/hooks/" — accurate but ambiguous about which Hero git hooks. Tighten to "Remove all Hero git hooks (tracker hooks, hero-next projection hooks, merge driver) from .git/hooks/ and .gitattributes."

## Boundaries

- **Out of scope:** unifying the two installer packages (Option A). Defer to a separate refactor spec if it ever becomes worth doing.
- **Out of scope:** removing `hero next install-hooks` as a command. It stays as a granular path.
- **Out of scope:** auto-cleaning orphaned hook files Hero never created. We only touch files Hero owns (marker blocks).
- **Out of scope:** uninstall hooks in worktrees other than the primary. `--git-dir` resolution remains as it is.

## Acceptance Criteria

- [ ] `uninstallNextHooks(projectRoot)` exists in `internal/cli/next_hooks.go`, idempotent, returns the list of modified paths.
- [ ] Removes the managed block from `.git/hooks/pre-commit` and `.git/hooks/post-merge`. Removes empty hook files. Preserves user content outside markers.
- [ ] Strips the `.gitattributes` managed block. Removes the file if empty after.
- [ ] Unregisters the `hero-next` merge driver from `.git/config`.
- [ ] `hero hooks uninstall` calls `uninstallNextHooks` after removing the general `# Hero git hook` block.
- [ ] `hero hooks status` reports next-hooks installation state alongside the general hook state.
- [ ] `hero hooks uninstall --host=all` removes general hooks + next-hooks + claude SessionStart{compact}, leaving no Hero-managed git or host-tool hook content behind.
- [ ] Eight tests added/updated per the list above; all green.
- [ ] No regression in the existing `hero next install-hooks` behavior. `TestNextInstallHooks_*` (or equivalent) still passes.
- [ ] Help text on `hero hooks uninstall` updated to name what's removed.

## Changes

- `internal/cli/next_hooks.go` — added `uninstallNextHooks` (returns the list of modified paths), `stripManagedBlockFromHook`, `stripGitAttributesBlock`, `isHookStubOnly`, `isGitConfigKeyMissing`, and `nextMergeDriverRegistered`.
- `internal/cli/hooks.go` — `runHooksUninstall` now calls `uninstallNextHooks` after the general uninstall and prints each removal; `runHooksStatus` reports `hero next pre-commit block` and `hero-next merge driver` state; `hooksUninstallCmd.Short` updated to name all three removal surfaces.
- `internal/cli/install_hooks_test.go` — added seven uninstall tests (`TestUninstallNextHooks_*`, `TestRunHooksUninstall_AlsoRemovesNextHooks`, `TestHostHooksUninstall_AllRemovesNextHooks`).
- `internal/cli/host_hooks_test.go` — extended `TestHostHooksUninstall_AllRemovesGitAndClaude` to install the hero-next block and assert it's removed alongside the general/claude hooks.
