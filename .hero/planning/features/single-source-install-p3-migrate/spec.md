---
title: "Single-Source Install P3 — hero install --migrate for legacy multi-harness installs"
slug: single-source-install-p3-migrate
type: feature
status: completed
status_verified: "2026-05-12 by go test ./internal/install/... -count=1 — 5 new TestMigrate_* tests pass + full install suite. Dogfooded on example codebase: 32 drifted agent files + 22 commands + 44 skill conflicts (mix of flat .md and nested SKILL.md layouts) all reconciled cleanly. Newest mtime wins (Claude versions, more recently re-installed). Result: example codebase goes from `.claude/skills`/44 + `.opencode/skills`/79 (mixed flat+nested duplicates) → single canonical `.hero/skills`/44 with both `.claude/skills` and `.opencode/skills` as symlinks pointing at it. 28KB user-authored AGENTS.md fully preserved with Hero managed block inserted."
priority: P0
tags: [install, migrate, drift, example codebase, multi-harness]
created: 2026-05-12
relations:
  - target: single-source-install
    kind: parent
  - target: single-source-install-p2-canonical-tree
    kind: follows
horizon: now
---

## Goal

A single command — `hero install --migrate` — that takes a
legacy-multi-harness-shape project (multiple harness directories full of drifted
physical copies of the same agent/command/skill content) and converts
it into the P2 canonical-tree layout (one canonical copy in
`.hero/{agents,commands,skills}/` or the user's configured override;
every harness directory is a directory symlink pointing into it).

Auto-detects which harness targets are installed, reconciles content
drift via newest-mtime wins, promotes winners to canonical, then
re-runs each target's install with `--force` to materialize the
symlinks.

## Problem

Paperboy as a snapshot of the bad outcome:

- `.claude/agents/` had 34 regular-file agent definitions
- `.opencode/agents/` had 34 regular-file agent definitions
- 14 of those 34 pairs had drifted content (different timestamps,
  different bytes)
- `.claude/skills/` had 44 flat `.md` files
- `.opencode/skills/` had 79 files — a mix of legacy flat `*.md`
  AND nested `<name>/SKILL.md` directories, both layouts holding the
  same skill names with stale duplicates accumulating across installs

The user did nothing wrong. This is what happens when
`hero install --target X` runs at different points in time against
different harness targets; each install renders its own physical
copy, and the copies drift as Hero's templates evolve.

P2 introduced the canonical-tree layout to prevent this from
happening going forward. P3 cleans up projects that already have
the bad state.

## Design

### `hero install --migrate`

A new install flag. Usage:

```
hero install project <path> --migrate [--dry-run]
```

`--target` is auto-detected from filesystem presence — no per-target
flag required.

Flow:

1. **Detect** installed harness targets via the existing
   `DetectInstalledTargets(targetDir)` — scans for `.claude/`,
   `.opencode/`, `.codex/`, `.cursor/`, `.github/copilot/`, `.ai/`,
   `.openhands/`, each with at least one of `agents/`/`commands/`/
   `skills/` present.

2. **Gather candidates** for each kind (agents, commands, skills)
   across all detected harness dirs PLUS canonical. Skills directory
   scan handles both layouts: flat `<name>.md` AND nested
   `<name>/SKILL.md` register under the same name key, so when both
   exist for the same skill they drift-compete naturally.

3. **Reconcile drift** — for each filename group with multiple
   distinct content hashes, pick the file with the newest mtime as
   winner. Record a `MigrationConflict` describing every candidate
   with mtime + a short content hash. (Files with identical content
   are not "drift" — newest mtime wins for consistency, but no
   conflict reported.)

4. **Promote winners to canonical** — copy each winner into the
   resolved canonical path (`.hero/{agents,commands,skills}/` by
   default, or the user's `content.*_path` override from
   `hero.json`). Skills always written in `<name>/SKILL.md` directory
   layout regardless of input layout.

5. **Re-run install per target** with `Force=true` and a new
   `SkipCanonicalRender=true` flag — so the per-target installer
   creates the harness-dir symlinks pointing at canonical, but does
   NOT clobber the just-promoted winner content by re-rendering from
   embedded source.

6. **Report** — a `MigrationReport` with detected targets, conflict
   list (each conflict gets a full candidate list for diagnostics),
   promoted-files count per kind, per-target install results, and
   any errors. Renderable via `report.StringReport()`.

### `--dry-run`

Reports everything (detected targets, conflicts with full diagnostics,
which files would be promoted) without writing to the filesystem.
Useful for "what would migrate look like?" before committing.

### Idempotency

Re-running migrate against an already-migrated project: detected
targets resolve harness dirs that are symlinks (Stat follows symlinks
so they still count as installed). The scan skips symlink sources, so
candidates come only from canonical (already correct). No drift
reported. Per-target install is a no-op. Report says "No drift
detected."

### Why mtime-wins instead of interactive prompts

Heuristic that resolves 99% of real drift correctly: drifted copies
across harness dirs are usually different generations of the same
Hero content rendered at different times. The newer install ran more
recently → has newer Hero content. Newest mtime → most recent Hero
version → correct winner.

For the rare case where mtime is misleading (user manually edited an
older copy), the report makes the choice visible (full candidate list
with hashes), and the user can re-edit the canonical after migration
if needed. We don't add interactive prompts for the first release —
the dry-run output gives the user everything they need to spot a
wrong winner before running for real.

## Acceptance Criteria

- WHEN `hero install --migrate <path>` runs in a project with no
  installed harness targets THE SYSTEM SHALL exit with a clear
  "nothing to migrate" error rather than silently doing nothing
- WHEN `hero install --migrate <path>` runs in a project with two
  or more installed harness targets holding identical agent files
  THE SYSTEM SHALL promote one copy to canonical without reporting
  it as a conflict, and replace every harness dir with a symlink
- WHEN `hero install --migrate <path>` finds drifted copies of the
  same filename across harness dirs (different content) THE SYSTEM
  SHALL select the file with the newest mtime as the winner,
  promote it to canonical, and record a conflict entry naming every
  candidate with its mtime + short content hash
- WHEN `hero install --migrate <path>` runs across mixed layouts
  (e.g. flat `skills/<name>.md` and nested `skills/<name>/SKILL.md`
  in different harness dirs) THE SYSTEM SHALL treat both as
  candidates for the same skill, pick the newest, and write the
  winner to canonical in `<name>/SKILL.md` directory layout
- WHEN `hero install --migrate --dry-run <path>` runs THE SYSTEM
  SHALL emit the full reconciliation report without writing any
  files or creating any symlinks
- WHEN `hero install --migrate <path>` completes successfully THE
  SYSTEM SHALL leave each detected harness directory as a symlink
  pointing at the resolved canonical content
- WHEN `hero install --migrate <path>` runs against an
  already-migrated project (all harness dirs are symlinks already)
  THE SYSTEM SHALL report no drift and produce zero new filesystem
  changes
- THE SYSTEM SHALL preserve user-authored content in AGENTS.md and
  CLAUDE.md throughout migration (P1's managed-block-injection
  policy applies to whatever AGENTS.md / CLAUDE.md state existed
  pre-migrate)

## Changes

- `internal/install/migrate.go` (new) — RunMigrate entry point;
  candidate gathering across kinds and sources; drift detection via
  sha256-derived short hashes; newest-mtime winner selection;
  canonical-destination resolver; per-target re-run with
  SkipCanonicalRender; MigrationReport + StringReport renderer
- `internal/install/install.go` — Options gains
  `SkipCanonicalRender bool` so RunMigrate can prevent per-target
  installs from clobbering just-promoted winner content
- `internal/cli/install.go` — `--migrate` flag, short-circuit path
  in `runInstall` that constructs Options and calls
  `install.RunMigrate`
- `internal/install/migrate_test.go` (new) — TestMigrate_NoDrift_OneCanonical,
  TestMigrate_DriftedCopies_NewestWins,
  TestMigrate_DryRun_MakesNoChanges,
  TestMigrate_NoHarnessesDetected_Errors,
  TestMigrate_Idempotent_AfterFirstRun

## Boundaries

- **Not in scope:** orphan-harness-directory cleanup (e.g., `.cursor/`
  exists from a prior install but the user no longer uses Cursor).
  Migration migrates whatever's there; the user can `rm -rf .cursor/`
  manually if they want to clean up. A future feature could add
  `--prune-orphans` if real demand emerges.

- **Not in scope:** interactive conflict resolution. Newest mtime
  wins automatically. The dry-run output makes the winner visible
  so users can preview before committing.

- **Not in scope:** content merging within a file. If `.claude/agents/
  engineer.md` and `.opencode/agents/engineer.md` differ, migrate
  picks one — it doesn't try to merge their content line-by-line.

- **Not in scope:** automated rollback. If migration goes wrong, the
  user can recover from git (assuming their harness dirs were
  committed). For uncommitted state we'd need a backup-tarball
  feature; deferred until real failures are observed.

- **Not in scope:** migrating projects without a `.hero/` workspace.
  `hero install --migrate` requires `.hero/` to already exist (run
  `hero init` first). The errors out with a clear message.

## Mission Fit

> "Does this make the next agent session start smarter than the
> last one ended — and does it raise the floor for everyone?"

Yes — directly for legacy-multi-harness-shape projects, which were the canary
for this entire initiative. Before P3, the only way out of the bad
state was for the user to manually delete `.claude/`, `.opencode/`,
etc. directories and re-install. After P3, a single command does
the right thing automatically: newest content wins, canonical
established, symlinks everywhere, user content preserved. The
ecosystem floor rises specifically for the multi-harness teams
where the legacy state is most likely to have accumulated.
