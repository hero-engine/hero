---
title: "`hero upgrade` Refreshes Git Hooks on New Releases"
slug: hero-upgrade-refreshes-hooks
type: feature
status: completed
tags: [upgrade, hooks, git, refresh, install]
created: 2026-04-30
relations:
  - target: pre-commit-auto-stage-next
    kind: extends
  - target: kickoff-prompts-queue
    kind: related
horizon: now
---

## Kickoff

Make `hero upgrade` refresh the installed pre-commit hook so binary upgrades that change the hook script (e.g. adding `hero queue write`) actually take effect — and surface stale-hook drift in `hero check` so users see when their installed hook lags behind the binary.

**Status:** completed — `hero upgrade` now refreshes installed hooks (gated by `--no-hooks`) and `hero check` flags content drift.

**Pick up at:** lived-experience pass — `hero upgrade` should now silently refresh your stale pre-commit hook (the one that doesn't yet have `hero queue write`). `hero check` will warn if a hook is stale before upgrade runs.

→ `hero upgrade --dry-run` (run in this repo to see the refresh detection in action)

**Files:** [internal/cli/upgrade.go:130](internal/cli/upgrade.go:130), [internal/cli/next_hooks.go:194](internal/cli/next_hooks.go:194), [internal/cli/check.go:179](internal/cli/check.go:179), [internal/cli/scan.go:90](internal/cli/scan.go:90)

## Goal

Close the gap that lets installed git hooks lag behind the hero binary. When the hook script changes (new functionality, bug fix, new staged file), `hero upgrade` should refresh it; if the user hasn't upgraded yet, `hero check` should warn that the installed hook is stale.

Discovered while shipping `kickoff-prompts-queue`: the pre-commit hook gained a `hero queue write -q` line, but existing repos with the old hook installed don't pick up the change until they manually run `hero next install-hooks`. Same gap will recur every time the hook script content changes.

## Problem

Today:

- `hero scan` auto-installs the pre-commit hook on first run ([internal/cli/scan.go:98](internal/cli/scan.go:98)).
- `hero upgrade` refreshes `.claude/`, `.opencode/`, `.cursor/` agents/commands/skills — but **not** git hooks.
- `hero check` only checks for **presence** of the managed marker block, not whether its content matches the current binary's `hookScript()` output.

Result: a hero binary upgrade that changes the hook script silently fails to deliver the change to users who already have hero installed. The user has to know to run `hero next install-hooks` themselves. They usually don't.

## Design

### Refresh hooks during `hero upgrade`

In `runUpgrade` ([internal/cli/upgrade.go](internal/cli/upgrade.go)), after the workspace + targets upgrade completes successfully, call into hook refresh:

```go
if !upgradeNoHooks {
    if err := refreshHooksIfPresent(projectRoot, upgradeDryRun, cmd.OutOrStdout()); err != nil {
        // Non-fatal — log and continue. Upgrade primary purpose is
        // workspace files; hooks are a nice-to-have refresh.
        fmt.Fprintf(os.Stderr, "Warning: hook refresh failed: %v\n", err)
    }
}
```

`refreshHooksIfPresent` lives in `next_hooks.go` (alongside `installNextHooksQuiet`):

- Detect git repo via `resolveGitDir`. If not a git repo, skip silently.
- Detect installed managed block via existing `preCommitHookInstalled`. If not installed, **skip** — don't auto-install on upgrade if the user explicitly removed the hook. `hero scan` is the install path; `hero upgrade` is the refresh path.
- If installed:
  - In dry-run mode: compare current hook content to `hookScript("pre-commit")`. Print `would refresh pre-commit hook (content stale)` or `pre-commit hook is current — no refresh needed`.
  - Otherwise: call `installNextHooksQuiet` (idempotent, marker-block-safe) and print `refreshed pre-commit hook`.

### `--no-hooks` flag on `hero upgrade`

Mirror the existing flag on `hero scan`. Default false (refresh by default). Useful for users who manage hooks themselves or have an alternate install model.

### Stale-hook drift detection in `hero check`

Today: `hero check` flags missing pre-commit hook ([internal/cli/check.go:179](internal/cli/check.go:179)). Extend it: when the hook **is** installed, also check whether the managed block content matches `hookScript("pre-commit")`. If not:

```
Pre-commit hook is stale:
  Installed managed block doesn't match the current hero binary's
  hook script. Run 'hero upgrade' (or 'hero next install-hooks')
  to refresh.
```

Implementation detail: extract the managed block from the installed hook file (between `hookMarkerStart` and `hookMarkerEnd`), trim whitespace, compare against `hookScript("pre-commit")` similarly trimmed. Mismatch → stale.

### Don't touch `.gitattributes` or merge driver

The merge driver is registered in `.git/config` (per-clone, not committed). The `.gitattributes` managed block ships with the repo. Both are written by `installNextHooksQuiet` already; the same call refreshes them as part of the same idempotent install path. No additional work for them.

### What's NOT shipping

- **Auto-install on upgrade when hook isn't present.** Respect explicit user removal. `hero scan` is the install entry point; upgrade only refreshes existing installations.
- **Cross-repo hook refresh.** Single-repo only. If the user has multiple repos, each upgrade runs in one repo at a time. Out of scope.
- **Migration of older hook formats.** The marker-block model has been stable since `git-hook-integration` shipped. If/when a hook layout migration is needed, it's its own spec.
- **Refreshing hooks for `hero serve` / daemon installs.** Daemon hook lifecycle is separate; not addressed here.

## Changes

- `internal/cli/upgrade.go` — declare `upgradeNoHooks bool` flag, wire `--no-hooks` in init, call `refreshHooksIfPresent` at end of `runUpgrade` (after the targets loop, before the final summary print).
- `internal/cli/next_hooks.go` — add `refreshHooksIfPresent(projectRoot string, dryRun bool, w io.Writer) error` that detects + refreshes (or reports). Add a helper `currentPreCommitManagedBlock(projectRoot string) (string, error)` returning the installed managed block content or `""` when missing — reused by both upgrade refresh and `hero check`.
- `internal/cli/check.go` — when `preCommitHookInstalled` is true, compare installed managed block to `hookScript("pre-commit")`. On mismatch, print the stale-hook warning and bump the issue counter.
- `internal/cli/upgrade_test.go` — add tests for hook refresh path: present-and-current → no-op (no error, no extra output), present-and-stale → refreshes, absent → skipped, `--no-hooks` → skipped, `--dry-run` → reports without writing.
- `internal/cli/check_test.go` — add tests for stale-hook detection.

No new packages, no new commands, no MCP surface changes.

## Acceptance Criteria

- WHEN `hero upgrade` runs in a git repo with the pre-commit managed block installed THE SYSTEM SHALL refresh the pre-commit hook to match the current binary's `hookScript("pre-commit")` output.
- WHEN `hero upgrade` runs in a git repo without the pre-commit managed block THE SYSTEM SHALL skip hook refresh (do not auto-install on upgrade).
- WHEN `hero upgrade --no-hooks` runs THE SYSTEM SHALL skip hook refresh regardless of installed state.
- WHEN `hero upgrade --dry-run` runs and the installed hook content matches the current binary THE SYSTEM SHALL print "pre-commit hook is current" without modifying any files.
- WHEN `hero upgrade --dry-run` runs and the installed hook content is stale THE SYSTEM SHALL print "would refresh pre-commit hook" without modifying any files.
- WHEN `hero upgrade` runs outside a git repo THE SYSTEM SHALL skip hook refresh silently (no error, no output).
- WHEN `hero check` runs and the pre-commit managed block content does not match `hookScript("pre-commit")` THE SYSTEM SHALL print a stale-hook warning recommending `hero upgrade` (or `hero next install-hooks`) and bump the issue counter.
- WHEN `hero check` runs and the pre-commit managed block content matches the current binary THE SYSTEM SHALL not emit a stale-hook warning.
- THE SYSTEM SHALL preserve `installNextHooksQuiet` as the single source of truth for hook content — `refreshHooksIfPresent` calls it rather than implementing parallel write logic.
- IF hook refresh fails during `hero upgrade` THEN THE SYSTEM SHALL emit a warning to stderr and continue — upgrade's primary job (workspace files) does not block on hook refresh.

## Boundaries

- Does **not** auto-install hooks when none are present — respects explicit user removal. `hero scan` retains the install entry point.
- Does **not** modify `.gitattributes` or merge driver registration independently — those flow through the existing `installNextHooksQuiet` call.
- Does **not** introduce a separate `hero hooks refresh` command — refresh is a side effect of `hero upgrade`, not a standalone verb. (`hero next install-hooks` already serves the explicit-refresh use case.)
- Does **not** address daemon-mode hook lifecycle, multi-repo hook refresh, or hook-format migrations.

## Mission Fit

> "Does this make the next agent session start smarter than the last one ended — and does it raise the floor for everyone, not just the senior dev who already knows what to ask?"

Floor-raising: yes. Today, hook content changes ship to senior users (who know to run `hero next install-hooks`) but silently miss everyone else. Closing that gap means every user gets the new hook behavior on their next `hero upgrade` — without them needing to know hooks exist.

Smarter starts: indirect. The hook is what makes `.hero/NEXT.md` and `.hero/QUEUE.md` travel with commits in the first place. A stale hook means stranded handoff state — the same mission leak `pre-commit-auto-stage-next` already closed, recurring every time the hook content evolves.
