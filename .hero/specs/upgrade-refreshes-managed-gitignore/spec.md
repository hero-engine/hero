---
title: "hero upgrade never refreshes the managed .gitignore block, so new entries miss existing workspaces"
slug: upgrade-refreshes-managed-gitignore
type: bug
status: completed
priority: P2
severity: low
root_cause_class: code
domain: engineering
size: trivial
created: 2026-10-06
tags: [gitignore, upgrade, mcp, peer-request]
completed_at: 2026-10-06T23:08:21Z
---

# hero upgrade refreshes the managed .gitignore block

## Goal

Make entries added to Hero's managed root `.gitignore` block reach workspaces that were initialized before those entries existed, on `hero upgrade`.

## Kickoff

Peer hero-harness reported `.hero/mcp-*.pid` files showing up as untracked changes (Mail `mail_2a0ea87cdef292b91b0035db`). The leak itself was fixed by contributor PRs #10 (release on watchdog exit), #11 (stale reaper) and #12 (managed entries). This repo had independently built the same three fixes and dropped them in favor of those PRs. #12 rolls the entries out only when `hero init` is re-run. Add a `hero upgrade` refresh of an existing managed block. Verify with `go test ./internal/cli -run Gitignore`.

## Problem

Existing workspaces (e.g. hero-harness, with 4 untracked pidfiles) run `hero upgrade` after updating Hero, not `hero init`. So the new `.hero/mcp-*.pid*` and `.hero/mcp-debug.log` entries from #12 never reach them.

## Root Cause

`ensureManagedGitignoreBlock` is called only from `hero init`. The upgrade path refreshes hooks (`refreshHooksIfPresent`) but not the managed `.gitignore` block.

## Fix

`refreshManagedGitignoreIfPresent(projectRoot, dryRun)` re-renders the block only when the root `.gitignore` already contains Hero's markers. It never creates the block (`hero init` is the opt-in) and writes nothing in dry-run. It is called on both `hero upgrade` completion paths, beside the hook refresh.

## Acceptance Criteria

- **AC-1:** WHEN `hero upgrade` completes in a workspace whose root `.gitignore` already contains Hero's managed block THE SYSTEM SHALL refresh that block to the current entries while preserving content outside the markers.
- **AC-2:** IF the root `.gitignore` has no managed block, or the upgrade is a dry-run, THEN THE SYSTEM SHALL write nothing.

## Changes

1. `internal/cli/init.go`: `refreshManagedGitignoreIfPresent`.
2. `internal/cli/upgrade.go`: call it on both completion paths.
3. `internal/cli/init_gitignore_test.go`: `TestManagedGitignoreCoversMCPPidfilesAndUpgradeRefresh`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: upgrade refreshes an existing block, keeps outside content | DONE | `refreshManagedGitignoreIfPresent`, called after the hook refresh in both completion paths of `upgrade.go`. `TestManagedGitignoreCoversMCPPidfilesAndUpgradeRefresh` asserts user entries are kept and the #12 entries (`.hero/mcp-*.pid`, `.hero/mcp-*.pid.*`, `.hero/mcp-debug.log`) are added |
| 2 | AC-2: no block or dry-run writes nothing | DONE | Same test: dry-run leaves the bytes unchanged, and a workspace with no `.gitignore` gets none |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | init.go refresh helper | DONE | Reuses #12's `managedGitignoreEntries` and the existing `ensureManagedGitignoreBlock` |
| 2 | upgrade.go call sites | DONE | Both the no-target and main completion paths |
| 3 | test | DONE | `internal/cli/init_gitignore_test.go` |

### Exercise-the-feature check

- [x] Covered by the CLI test above, which drives the real helper against temp workspaces. The pidfile behaviour itself (release, reaper, entries) is #10–#12's, already merged with their own tests.

### Excellence Bar self-check

- [x] Yes. It's the minimal remaining delta after adopting the contributors' fixes, and it follows the existing refresh-only-what-was-opted-into pattern.

## Final validation

`go test ./... -count=1` and `go vet ./...` pass on the rebased branch.
