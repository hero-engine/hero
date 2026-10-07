---
title: "A hero folder that is a symlink is walked as empty — Hero finds no specs"
slug: symlinked-hero-dir-walks-nothing
type: bug
status: delivering
priority: P1
severity: moderate
root_cause_class: logic
domain: engineering
size: small
created: 2026-10-07
tags: [filesystem, discover, watch, release-blocker]
---

# A symlinked hero folder is walked as empty

## Goal

When a project's hero folder (`.hero`) is a symlink to a directory elsewhere, Hero works exactly as if it were a real directory. Today it silently finds nothing.

## Kickoff

Found by the v0.35.3 pre-release audit. The user chose to hold the release until it is fixed. `filepath.Walk` and `filepath.WalkDir` never follow their root, so every walk rooted at the hero folder sees a single non-directory entry and stops. Fix those walks with one shared helper. Verify with `go test ./...` and a real `hero list` in a symlinked workspace.

## Root Cause

`filepath.Walk(root)` calls `os.Lstat(root)`. For a symlinked root, that describes the link (not a directory), so the walk never descends. Walks rooted *below* the hero folder, such as `.hero/knowledge`, are fine, because the OS resolves the symlinked parent component. Exactly four walks are rooted at the hero folder itself:

| Site | Effect |
|---|---|
| `spec.Discover` (`internal/spec/spec.go`) | Finds zero specs: `hero list` prints "No specs match", and every spec-reading command and MCP tool sees an empty workspace |
| `watch.Scan` (`internal/watch/watch.go`) | The watcher's snapshot is empty, so no change events |
| `validateSpecSources` (`internal/embeddings/chunker.go`) | Spec validation before embedding checks nothing |
| `lastTouchedAt` (`internal/serve/projectpage/data/identity.go`) | Reports the symlink's own mtime, not the newest spec's |

One adjacent site, found by the audit, has the same cause one level down. `install.FindNestedHeroDirs` (the "leftover standalone workspaces" migration hint) used `d.IsDir()` from `WalkDir`, so a nested workspace whose `.hero` is a symlink was not listed. That listing also feeds `hero install satellites --migrate-nested --apply`, which moves files out of a nested `.hero` and deletes it. For a linked `.hero` that would gut the link target (often another checkout's live workspace) and remove only the link. So once the hint lists it, the migration must refuse it.

Hero never creates a symlinked hero folder itself (satellites deliberately get none), so only hand-made layouts are affected.

## Fix

New `internal/fsutil` with `Walk` and `WalkDir`. When the root is a symlink that resolves, they walk the target but report every path under the original root. Callers that compare against or relativize to the hero dir therefore behave unchanged, including `Spec.Archived` stamping, `hiddenHeroPath` and the snapshots skip. A plain root or a broken link behaves exactly like `filepath.Walk`/`WalkDir`. The four sites switch to the helper.

## Acceptance Criteria

- **AC-1:** WHEN the hero folder is a symlink to a directory THE SYSTEM SHALL discover its specs with paths under the hero folder, and stamp `Archived` correctly.
- **AC-2:** WHEN the hero folder is a symlink THE SYSTEM SHALL include its files in the watch snapshot, spec-source validation and last-touched time.
- **AC-3:** WHEN the walk root is not a symlink, or is a broken symlink, THE SYSTEM SHALL behave exactly like `filepath.Walk`/`WalkDir`.
- **AC-5:** WHEN a nested workspace's `.hero` is a symlink to a directory THE SYSTEM SHALL list it in the nested-workspace migration hint. A `.hero` that links to a file or dangles SHALL NOT be listed.
- **AC-6:** WHEN a nested `.hero` is a symlink THE SYSTEM SHALL refuse to plan or apply its migration (`ErrLinkedNestedHero`, naming the link target), leave the target and the link untouched, and still migrate the other nested workspaces.
- **AC-4:** WHEN a user runs `hero list` in a project whose `.hero` is a symlink THE SYSTEM SHALL list its specs.

## Changes

1. `internal/fsutil/walk.go` (new: `Walk`, `WalkDir`) and `walk_test.go`.
2. `internal/spec/spec.go` (`Discover`) and `discover_symlink_test.go`.
3. `internal/watch/watch.go` (`Scan`) and `watch_test.go`.
4. `internal/embeddings/chunker.go` (`validateSpecSources`) and `chunker_test.go`.
5. `internal/serve/projectpage/data/identity.go` (`lastTouchedAt`) and `identity_test.go`.
6. `internal/install/satellite_detect.go` (`FindNestedHeroDirs`, `isSymlinkToDir`) and `satellite_detect_test.go`.
7. `internal/install/satellite_migrate.go` (`ErrLinkedNestedHero`, refused in `PlanMigration`, so `ApplyMigration` refuses too), `internal/cli/install_satellites.go` (apply skips and continues), `satellite_migrate_apply_test.go` and `install_satellites_linked_test.go`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: Discover through a symlink | DONE | `TestDiscoverFollowsSymlinkedHeroDir` checks the paths stay under the link and that `Archived` is a=false, b=true. With the old walker it fails: "discovered 0 specs" |
| 2 | AC-2: watch / validate / last-touched | DONE | `TestScan_followsSymlinkedHeroDir`, `TestValidateSpecSourcesFollowsSymlinkedHeroDir` (an unreadable spec.md must be reported, which proves the walk descended) and `TestLastTouchedAtFollowsSymlinkedHeroDir` (a distinctive future mtime, so the link's own mtime cannot pass). All three fail with the old walkers |
| 3 | AC-3: unchanged otherwise | DONE | `TestWalkPlainRootUnchanged` and `TestWalkBrokenSymlinkRootReportsLikeFilepathWalk`. The full suite passes |
| 5 | AC-5: nested symlinked .hero in migration hint | DONE | `TestFindNestedHeroDirsFollowsSymlinkedHero`: a link to a dir is listed; a link to a file and a dangling link are not. With the old code it fails: "got [], want [apps/web]" |
| 6 | AC-6: migration refuses a linked nested .hero | DONE | `TestMigrationRefusesLinkedNestedHero` checks that plan and apply both return `ErrLinkedNestedHero` and that the target and link are intact. `TestMigrateNestedApplySkipsLinkedHero` runs the real `install satellites --migrate-nested --apply --yes --force`: it prints "Skipped apps/web", leaves the target spec and the link in place, and migrates the real `engines/mlx`. Without the guard, apply moved the target's spec and deleted the link, which confirms the audit's blocker was real |
| 4 | AC-4: hero list end-to-end | DONE | A freshly built binary in a temp repo whose `.hero` is a symlink: `hero list` shows `a`, and `hero status` counts 1 upcoming and 1 completed. Before the fix: "No specs match" |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | fsutil | DONE | `TestWalkDescendsSymlinkedRoot` and `TestWalkDirDescendsSymlinkedRoot` |
| 2 | Discover | DONE | — |
| 3 | watch | DONE | — |
| 4 | embeddings | DONE | — |
| 5 | identity | DONE | — |
| 6 | nested-workspace hint | DONE | — |
| 7 | migration guard | DONE | — |

### Exercise-the-feature check

- [x] Ran the real `hero list` and `hero status` binary in a symlinked workspace (AC-4). Each consumer test was falsified against the old walkers.

### Excellence Bar self-check

- [x] Yes. Every walk rooted at the hero folder is covered (enumerated from all `filepath.Walk`/`WalkDir` call sites), and walks rooted below it were shown to be unaffected.
