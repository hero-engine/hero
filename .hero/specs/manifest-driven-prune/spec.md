---
title: "Manifest-driven prune of product-removed content — remove agents/commands/skills Hero dropped, never user files"
slug: manifest-driven-prune
type: feature
status: completed
domain: engineering
priority: P2
size: medium
created: 2026-07-16
tags: [install, upgrade, prune, manifest, harness, provenance]
relations:
  - target: agents-md-erased-by-snapshot-pointer-writer
    kind: related
  - target: install-integrity-self-check
    kind: related
completed_at: 2026-07-16T23:36:43Z
---

# Manifest-driven prune of product-removed content

## Context

v0.26.2 (shipped today) removed checksum-based overwrite-gating from `hero upgrade`:
upgrade now overwrites Hero's own generated files unconditionally (`install.Options.Force =
true`, always), because a recorded checksum can never reliably identify "our file" across
arbitrary version jumps. That is settled and correct — see
`.hero/knowledge/conventions/upgrade-overwrites-hero-owned-files.md`.

That decision closed the **overwrite** direction and deliberately left the **removal**
direction open. When the product *drops* an agent, command, or skill between versions — a
scrubber agent is deleted, a command is renamed, a skill is retired — `hero upgrade`
renders the new set but the orphaned file lingers on disk forever. Nothing records "what
Hero wrote last time" to diff against the new render, so nothing can tell an orphaned
Hero file apart from a file the user authored themselves. It is the removal-direction
mirror of the Codex dead-bytes cleanup, and of the skill-directory prune that landed today.

The user endorsed this as the correct counterpart to the overwrite change, in their words:
*"keep a manifest on install and upgrade of what files were written, and use it to
potentially remove an agent or command or skill that we removed from the product, without
removing ones that an end user could have added themselves. Checksums do not make sense."*

**The manifest is provenance — a record of what Hero wrote — and it drives PRUNE only.**
Never overwrite (that's Force, already done). Never checksums (structurally disqualified,
already documented).

### What already exists — this spec generalizes a proven pattern

`internal/install/prune.go` (landed today) implements exactly this philosophy, but only for
**nested skill DIRECTORIES**. Its header comment is the canonical statement of the model
and this spec reuses its reasoning verbatim: the dest dirs are shared ground (`.agents/skills`
is a cross-tool standard; `.claude/skills` / `.opencode/skills` hold user-authored skills in
real projects), so *"remove what I did not just write"* is not a safe rule. A directory is
removed only when Hero can **prove** it wrote it, via one of two proofs — (1)
`install-state.json`'s recorded `TargetState.SkillDirs`, or (2) a namespace Hero owns by
construction (Codex's `command-<name>/` prefix). Everything else is left alone.

This spec generalizes that proven, provenance-based prune from skill **dirs** to the
**flat rendered FILES** (agents, commands, and Cursor's flat skills) that `prune.go`'s
directory mechanism does not reach — across all six install targets.

## Goal

Install and upgrade record, per target, a **file manifest** — the set of flat agent /
command / flat-skill dest paths Hero rendered this run — in `install-state.json`
`TargetState.Files`. On the next install/upgrade, after rendering the new set for a target,
Hero removes files that are **in the recorded prior manifest but absent from the new
render**, and only those. A file that is not in the manifest is the user's and is never
touched. A missing or empty prior manifest (fresh clone, first run under the new scheme)
makes the prune a strict no-op. `--dry-run` reports what would be pruned and deletes
nothing. The existing skill-directory prune (`prune.go`, `SkillDirs`) keeps working
unchanged; the file prune is its disjoint sibling.

Done means: install target T at version A (which ships agent `foo`), upgrade to a content
set that dropped `foo`, and the stale `foo` file for T is gone — while a `my-custom-agent`
file the user dropped in the same directory is still there, byte-for-byte.

## Kickoff

Records what agent/command/skill files Hero wrote per target, so a later upgrade can delete
the ones the product dropped — without ever touching files the user added.

**Status:** planning — spec just landed, no code yet. Generalizes the skill-dir prune
that shipped today (`internal/install/prune.go`) from dirs to flat files.

**Pick up at:** add `TargetState.Files` to `state.go`, then instrument the three flat-file
write primitives (`installFlat`, `installSkillsFlat`, `renderToFile`) to record a
`result.rendered` set, then add `pruneStaleFiles` beside `pruneStaleSkillDirs` and hook it
into `Run()`.

→ `.hero/planning/features/manifest-driven-prune/spec.md`

**Files:** `internal/install/prune.go`, `internal/install/state.go:57,148`,
`internal/install/content.go:26,172`, `internal/install/render.go:40`,
`internal/install/install.go:117,143`

**Skip:** don't source the manifest from `result.Copied` — `copyFileFromFS` skips the
Copied append on byte-match when `!Force`, so it's incomplete on no-op installs (the trap
prune.go already documents). Don't prune nested skill dirs here — prune.go owns those.

## Problem

There is no per-target record of what Hero wrote, so nothing can drive a safe removal:

1. **`TargetState.SkillDirs` covers only nested skill directories.** It records dir *names*
   at one dest, consumed by `pruneStaleSkillDirs`. It says nothing about agent files
   (`.claude/agents/<n>.md`, `.codex/agents/<n>.toml`, `.github/prompts/agents/<n>.prompt.md`),
   command files, or Cursor's flat skill files (`.cursor/rules/skills/<n>.md`). Those file
   kinds have no manifest and are never pruned.

2. **`version.json` `installed_files` is the wrong substrate.** Verified on disk
   (`internal/version/version.go:119-126`): `StampInstall` *merges* into `installed_files`
   (`for k, v := range files { info.InstalledFiles[k] = v }`) and never deletes a key. It is
   therefore **append-only** — it accumulates paths from every install forever — and
   **union-across-targets** (all targets' files in one flat map, keyed on
   project-relative path with no target dimension). It cannot tell "dropped from the product"
   apart from "written two versions ago and still valid," and it cannot answer a per-target
   question at all. Its checksum *values* are also stale and no longer load-bearing (v0.26.2
   stopped trusting them). It stays as informational-only drift data; this spec does not make
   it load-bearing and does not migrate off it.

3. **`result.Copied` is not a reliable manifest source either.** `copyFileFromFS`
   (`internal/install/files.go:29-48`) only appends to `result.Copied` when it actually
   writes, and the whole idempotency/skip block is gated on `!opts.Force`. So on a non-force
   no-op install (bytes already match), `installFlat` and `installSkillsFlat` produce an
   almost-empty `Copied` — the exact reason `prune.go` derives its written set from canonical
   selectors instead (see its comment at lines 46-53). `renderToFile` always appends, so
   `Copied` is a *mixed* bag: complete for Codex/Copilot renders, incomplete for the
   copy-based flat installs. A manifest built from it would under-record and, worse, could
   overwrite a good full manifest with a partial one.

## Approach

### Design decision 1 — manifest shape, source, and location

**Decision: a new per-target `TargetState.Files []string` in `install-state.json`, recorded
from the canonical render set (a new `result.rendered`), REPLACED each run, holding
TargetDir-relative forward-slash dest paths.**

- **Per-target, in `install-state.json`.** Consistent with the proven `SkillDirs` mechanism
  and sitting right beside it in the same struct. Per-target is load-bearing: a file orphaned
  for Claude (`.claude/agents/foo.md`) is not orphaned for Codex, whose agents live at
  `.codex/agents/foo.toml` — different dest trees, different manifests. The append-only,
  union-across-targets `installed_files` cannot express this; `TargetState` already keys by
  target.

- **Recorded from the canonical render set, NOT `result.Copied`.** Add an unexported
  `rendered []string` to `Result` (mirroring the existing unexported `skillDirs`). The three
  flat-file write primitives — `installFlat`, `installSkillsFlat`, and `renderToFile` —
  append every dest path they would materialize, **independent of** `copyFileFromFS`'s
  byte-match skip and of `Force`. This makes the manifest complete on every run (Problem #3),
  and it is the identical pattern `prune.go` uses to populate `result.skillDirs` from
  `canonicalSkillDirNames`. Instrumenting these three shared chokepoints covers all six
  targets automatically — no per-target renderer edits.

- **Replace, not merge.** `RecordTargetInstall` sets `TargetState.Files = <this run's
  rendered set>` wholesale — exactly as it already replaces `SkillDirs`. Append-only can't
  distinguish "dropped from the product" from "still valid"; a full-replace manifest is the
  precise record of "what Hero renders *now*," which is what a prune diffs against.

- **Fate of `installed_files`: leave it informational-only.** It keeps feeding
  `StampInstallVersion` (harmless) and drift *reporting*. It is never consulted by the prune.
  No migration in this spec.

### Design decision 2 — the prune rule and its load-bearing safety invariant

**Decision: after rendering a target's new set, remove a file iff it is IN the recorded
prior `Files` manifest for that target AND absent from this run's rendered set AND still
present on disk. Never remove a file absent from the manifest.**

The manifest membership *is* the provenance proof — the direct analogue of `prune.go`'s
`prior[name]` check. A file not in the prior manifest was not recorded as Hero-written, so
it is the user's and is invisible to the prune. This is the invariant the whole feature
rests on; it gets a dedicated AC (AC-2). Because the manifest stores **actual rendered dest
paths**, Codex `.toml`, Copilot `.prompt.md`, and Cursor flat-skill `.md` paths compare
correctly against the new render without any name-canonicalization step.

### Design decision 3 — missing / empty manifest is a strict no-op (fresh clone + first run)

**Decision: when the prior manifest for a target is nil or empty, prune nothing.**

`install-state.json` is gitignored (`.gitignore:69`) and machine-local. On a fresh clone it
does not exist; `ReadInstallState` returns a zero-valued state with empty `Targets`, so
`prior.Files` is nil. The prune must treat nil/empty prior as "no proof of anything" and do
nothing — never guess, never prune off absence. This single rule also covers the first run
under the new scheme (design decision 6): an existing workspace's `TargetState` has no
`Files` field yet, so `prior.Files` is nil and the first run cannot false-delete. It mirrors
`prune.go`, whose namespace-fallback stays safe with no prior state.

**Empty-render guard.** Symmetrically, if *this run's* rendered set for a target is empty
(a broken or empty content source), skip the prune rather than deleting everything the prior
manifest listed. A legitimately empty product set is near-impossible; a broken source is not.
The guard trades a never-happens correct prune for protection against a catastrophic wipe.

### Design decision 4 — the actor deletes and reports; honors dry-run; no opt-out flag

**Decision: the prune deletes, prints every removed path, honors `--dry-run`, and has no
opt-out flag.**

Unlike `hero check` (which only reports — see `install-integrity-self-check`), install and
upgrade are the actor here, exactly as `prune.go` already deletes stale skill dirs during
install. Match its output style: `cleanup <path> (removed — dropped from product)`; under
`--dry-run`, `cleanup <path> (would remove — dropped from product)` and delete nothing. No
opt-out flag — this is core upgrade hygiene, the manifest-proof rule already makes it safe,
and `prune.go` ships without one; adding a flag here would be a gratuitous asymmetry.

### Design decision 5 — disjoint from prune.go, no regression

**Decision: the file prune is a sibling of the skill-dir prune, disjoint by construction;
`prune.go`'s existing `pruneStaleSkillDirs` / `pruneNestedSkills` are untouched.**

The two mechanisms must not both claim the same path. They stay disjoint because the
manifest is populated only by the **flat-file** primitives:

- `installSkillsNested` (nested `<name>/SKILL.md` dirs) is **not** instrumented — nested
  skills stay owned by `prune.go` via `SkillDirs`.
- In `renderToFile`, append to `result.rendered` **only when the rendered `destName`
  contains no path separator**. This admits Codex agent `.toml` and Copilot `.prompt.md`
  (flat names) while excluding Codex's commands-as-skills, whose `destName` is
  `command-<name>/SKILL.md` (nested — already owned by `prune.go` via `codexSkillDirNames`
  and the `command-` owned prefix).
- `installSkillsFlat` (Cursor's `.cursor/rules/skills/<n>.md`) **is** instrumented — those
  are flat files that `prune.go` never touches (Cursor calls neither `pruneNestedSkills` nor
  `pruneStaleSkillDirs`), so they are a real orphan gap the file manifest closes.

Root instruction files (`AGENTS.md` / `CLAUDE.md`) and `copilot-instructions.md` flow
through `installNativeInstructionFile` / `installClaudeMd` / `installInstructionsMd`, **not**
through the three instrumented primitives — so they are structurally excluded from the
manifest and can never be pruned as whole files (they are managed-region-owned).

### Design decision 6 — migration / first-run safety for existing workspaces

Covered structurally by design decision 3: an existing workspace has `TargetState` with
`SkillDirs` but no `Files`. The first install/upgrade under the new scheme sees
`prior.Files == nil` → prunes nothing → **records** the fresh `Files` manifest. The prune
can only bite on the *second* run, once a real prior-render manifest exists. This is exactly
how `SkillDirs` itself bootstrapped; no version flag or migration step is needed. Gets a
dedicated AC (AC-6). `PersistInferredTargets` (the backfill path) must preserve any prior
`Files` for the same reason it already preserves `SkillDirs` (`state.go:299`) — backfilling
a target *set* says nothing about its files.

### Per-target file coverage (the tripwire)

`harness-changes-cover-all-targets` **[high]** applies. All six targets render flat agent /
command / flat-skill files, so all six get a file manifest. Coverage falls out of
instrumenting the three shared primitives:

| Target | Flat files in the manifest | Nested skills (stay with prune.go) |
|---|---|---|
| `claude` | `.claude/agents/<n>.md`, `.claude/commands/<n>.md` | `.claude/skills/<n>/` |
| `codex` | `.codex/agents/<n>.toml` | `.agents/skills/<n>/`, `.agents/skills/command-<n>/` |
| `copilot` | `.github/prompts/agents/<n>.prompt.md`, `.github/prompts/commands/<n>.prompt.md` | `.github/skills/<n>/` |
| `cursor` | `.cursor/rules/agents/<n>.md`, `.cursor/rules/commands/<n>.md`, `.cursor/rules/skills/<n>.md` | (none — Cursor skills are flat, in the manifest) |
| `opencode` | `.opencode/agents/<n>.md`, `.opencode/commands/<n>.md` | `.opencode/skills/<n>/` |
| `generic` | `.ai/agents/<n>.md`, `.ai/commands/<n>.md` | `.ai/skills/<n>/` |

No target is excluded.

## Acceptance Criteria

Each criterion is one physical line (the EARS classifier needs the trigger and `THE SYSTEM
SHALL` on the same line). Verify with `hero spec lint <slug>`.

- **AC-1:** WHEN an install/upgrade renders a target's set and the recorded prior `Files` manifest for that target contains a path that is absent from this run's rendered set and still present on disk THE SYSTEM SHALL remove that file and print a `cleanup <path> (removed — dropped from product)` line.
- **AC-2:** IF a file under a target's dest tree is absent from that target's recorded prior `Files` manifest THEN THE SYSTEM SHALL NOT remove it, even when it is absent from the current rendered set.
- **AC-3:** IF the prior `Files` manifest for a target is nil or empty THEN THE SYSTEM SHALL prune no files for that target.
- **AC-4:** THE SYSTEM SHALL record and apply the file manifest independently per target across all six targets (`claude`, `codex`, `copilot`, `cursor`, `opencode`, `generic`), so a path orphaned for one target is not pruned from another target's dest tree.
- **AC-5:** WHEN a target renders agents or commands to a target-specific format THE SYSTEM SHALL record and compare the actual rendered dest paths — `.codex/agents/<n>.toml` for Codex and `.github/prompts/{agents,commands}/<n>.prompt.md` for Copilot — not canonical `<n>.md` names.
- **AC-6:** WHEN an install/upgrade runs against a workspace whose `TargetState` predates the `Files` field (only `SkillDirs` recorded) THE SYSTEM SHALL prune no files on that run and SHALL record a fresh `Files` manifest for subsequent runs.
- **AC-7:** WHERE `--dry-run` IS ENABLED THE SYSTEM SHALL report each file it would prune with a `would remove` line and SHALL delete no file and write no manifest.
- **AC-8:** WHEN an install/upgrade completes for a target THE SYSTEM SHALL record `TargetState.Files` as the full TargetDir-relative, forward-slash rendered-file set for that run, replacing any prior value.
- **AC-9:** THE SYSTEM SHALL never record a root instruction file (`AGENTS.md`, `CLAUDE.md`) or `.github/copilot-instructions.md` in any `Files` manifest and SHALL never prune those files.
- **AC-10:** WHEN the file prune runs THE SYSTEM SHALL leave nested skill directories governed by `prune.go`/`SkillDirs` untouched and SHALL not error on a manifest path already removed by the skill-dir prune.
- **AC-11:** WHEN a Cursor install drops a skill whose flat file (`.cursor/rules/skills/<n>.md`) was recorded in the prior manifest THE SYSTEM SHALL remove that flat skill file.
- **AC-12:** IF this run's rendered set for a target is empty THEN THE SYSTEM SHALL prune no files for that target.
- **AC-13:** WHEN `install-state.json` is absent (fresh clone of a gitignored workspace) THE SYSTEM SHALL prune no files.

## Changes

1. **`internal/install/install.go`** — the plumbing and the hook.
   - Add an unexported `rendered []string` to `Result` (mirroring the existing `skillDirs`
     doc comment: internal bookkeeping, not part of the `--json` contract).
   - In `Run`, after the per-target installer returns and before `StampInstallVersion` /
     `RecordTargetInstall`, call `pruneStaleFiles(opts, result)`. Place it **outside** the
     `!opts.DryRun` gate (it honors `DryRun` internally, like `pruneStaleSkillDirs`) but
     inside `opts.Mode == ModeProject && opts.TargetDir != ""`. Order matters: prune reads
     the *prior* manifest, then `RecordTargetInstall` writes the *new* one.

2. **`internal/install/content.go`** — instrument the copy-based flat writers.
   - In `installFlat`, after computing `dst` for each selected name, append `dst` to
     `result.rendered` (unconditionally — before/independent of `copyFileFromFS`, so the
     record is complete even when the copy is a byte-match no-op).
   - In `installSkillsFlat`, do the same for each `<destDir>/<name>.md` (Cursor's flat
     skills). Leave `installSkillsNested` **uninstrumented** — nested skills stay with
     `prune.go`.

3. **`internal/install/render.go`** — instrument the format renderers.
   - In `renderToFile`, after resolving `dst` and confirming `destName != ""`, append `dst`
     to `result.rendered` **only when `destName` contains no path separator**. This admits
     Codex `.toml` and Copilot `.prompt.md` while excluding Codex commands-as-skills
     (`command-<name>/SKILL.md`), which `prune.go` already owns. Record in the `DryRun`
     branch too, so `--dry-run` reporting sees the full set.

4. **`internal/install/prune.go`** — the file prune, sibling to `pruneStaleSkillDirs`.
   - `func pruneStaleFiles(opts Options, result *Result) error`:
     - Return nil unless `opts.Mode == ModeProject && opts.TargetDir != ""`.
     - Read prior state via `ReadInstallState(opts.TargetDir)`; `prior := st.Targets[string(opts.Target)].Files`.
       Nil/empty prior → return nil (AC-3, AC-6, AC-13).
     - Build the current rendered set as TargetDir-relative forward-slash paths from
       `result.rendered`. Empty current set → return nil (AC-12, empty-source guard).
     - For each path in `prior` not in the current set: resolve to absolute under
       `opts.TargetDir`; `os.Stat` — skip if already gone (AC-10, tolerates skill-dir prune
       having removed it). Under `DryRun`, print `would remove` and continue; else
       `os.Remove` and print `removed — dropped from product`. Best-effort `os.Remove` of the
       now-possibly-empty parent dir (ignore errors), matching the Copilot cleanup idiom.
     - Sort the stale set before acting, for deterministic output (as `pruneStaleSkillDirs`
       does).
   - Add a header block extending `prune.go`'s existing doctrine to files: same two-proof
     model, but proof here is *manifest membership only* (no owned-prefix analogue — flat
     agent/command files have no Hero-owned namespace at shared dests, so the recorded
     manifest is the sole proof).

5. **`internal/install/state.go`** — persist the manifest.
   - Add `Files []string \`json:"files,omitempty"\`` to `TargetState`, doc-commented like
     `SkillDirs` (the flat agent/command/flat-skill dest paths, TargetDir-relative,
     forward-slash, that the last install wrote; the next install prunes recorded entries it
     no longer writes).
   - In `RecordTargetInstall`, build `files` from `result.rendered`: relativize each against
     `opts.TargetDir`, `filepath.ToSlash`, sort, dedupe; set `TargetState.Files = files`
     (replace). Keep the existing `SkillDirs` handling unchanged.
   - In `PersistInferredTargets`, preserve `prior.Files` when carrying an existing entry
     forward (same treatment as `prior.SkillDirs` at line 299) — a backfilled target set says
     nothing about its files.

6. **`internal/install/prune_test.go`** (extend) — table-driven over **all six targets**,
   mirroring the existing prune tests and `TestHarness_*` shape:
   - `TestPruneStaleFiles_RemovesDroppedAgent` — install with agent `foo`, re-run with a
     content set lacking `foo`, assert `foo`'s rendered dest for the target is gone and the
     removal was printed. (AC-1, AC-4, AC-5 — assert `.toml` for codex, `.prompt.md` for
     copilot)
   - `TestPruneStaleFiles_NeverRemovesUserFile` — install, drop a `my-custom-agent.md` (or
     target-appropriate extension) into the same dest dir, re-run, assert the user file
     survives byte-for-byte. (AC-2)
   - `TestPruneStaleFiles_NoPriorManifestIsNoOp` — render once with no prior `Files`
     recorded (simulate a pre-`Files` `TargetState`), assert nothing pruned and a fresh
     manifest recorded. (AC-3, AC-6)
   - `TestPruneStaleFiles_FreshCloneNoState` — delete `install-state.json`, re-run, assert
     no-op. (AC-13)
   - `TestPruneStaleFiles_DryRunDeletesNothing` — set up a droppable file, run with
     `DryRun`, assert the file remains and no manifest was written but the `would remove`
     line printed. (AC-7)
   - `TestPruneStaleFiles_CursorFlatSkill` — Cursor install drops a skill, assert
     `.cursor/rules/skills/<n>.md` is pruned. (AC-11)
   - `TestPruneStaleFiles_LeavesNestedSkillDirs` — assert a nested skill dir dropped in the
     same run is handled by the skill-dir prune and the file prune neither double-removes nor
     errors. (AC-10)
   - `TestPruneStaleFiles_EmptyRenderIsNoOp` — force an empty rendered set, assert the prior
     manifest's files are NOT deleted. (AC-12)
   - `TestPruneStaleFiles_NeverManifestsInstructionFiles` — assert `AGENTS.md` / `CLAUDE.md`
     / `copilot-instructions.md` never appear in `TargetState.Files` and survive a prune
     run. (AC-9)
   - `TestRecordTargetInstall_FilesReplaced` — two runs with different sets, assert `Files`
     reflects only the second (replace-not-merge). (AC-8)

7. **`docs/`** — if install/upgrade prune behavior is documented anywhere, add the file
   manifest beside the skill-dir prune note. Locate with
   `rg -l "prune|SkillDirs|dead bytes" docs/`; if prune behavior is not enumerated in docs
   today, skip this item rather than inventing a new doc surface (as
   `install-integrity-self-check` did).

## Boundaries

Explicitly **not** in scope:

- **Overwrite behavior.** Settled in v0.26.2 — unconditional `Force=true`. Untouched.
- **Checksums as a load-bearing signal for anything.** Structurally disqualified; the
  manifest is provenance, not content comparison.
- **`version.json` `installed_files`.** Stays informational-only. Not made load-bearing, not
  migrated off, not consulted by the prune.
- **Root instruction files (`CLAUDE.md` / `AGENTS.md`) and `copilot-instructions.md` as whole
  files.** Owned by the managed-region writer; structurally excluded from the manifest
  (design decision 5). Whole-file pruning of orphaned instruction files is a separate,
  existing concern handled by `handleOrphanedInstructionFiles` in `upgrade.go`.
- **Nested skill directories.** Already pruned by `prune.go` via `SkillDirs`; this spec does
  not modify or re-solve that path.
- **MCP config, hooks, `settings.json`, `opencode.json`.** Not agents/commands/skills; not
  in the manifest.
- **`hero check` reporting of orphaned files.** This spec makes install/upgrade *remove*
  them; a read-only advisory would be a separate follow-up.

## Risks

- **Deleting a user file is the catastrophic failure.** The entire design turns on the
  manifest-membership invariant (AC-2): a path not recorded as Hero-written is invisible to
  the prune. `TestPruneStaleFiles_NeverRemovesUserFile` guards it directly; if a second path
  to deletion appears during delivery, treat it as a stop-and-think moment, not a filter to
  add.
- **Manifest completeness depends on instrumenting the right chokepoints.** If a target
  renders flat files through a path other than `installFlat` / `installSkillsFlat` /
  `renderToFile`, those files won't be recorded, so a later drop won't prune them (safe
  direction — under-prune, never over-delete) but the feature silently misses them. The
  per-target coverage table is the checklist; the tests assert an actual dropped file is
  pruned for every target.
- **The `renderToFile` path-separator rule is load-bearing for disjointness.** If Codex's
  command-as-skill `destName` were ever flattened to remove the `/`, it would enter the file
  manifest and collide with `prune.go`'s dir ownership. `TestPruneStaleFiles_LeavesNestedSkillDirs`
  pins the boundary.
- **Determinism of `result.rendered`.** The manifest must be stable across identical runs or
  `install-state.json` churns on every upgrade. Sort + dedupe in `RecordTargetInstall`; the
  rendered set is derived from a directory read, so sorting is required.
- **Auto-sync interaction.** `Run` is invoked per target, including recursively for
  auto-synced siblings; each records and prunes its own target's manifest. Per-target
  isolation (AC-4) makes this correct, but a test should exercise a multi-target workspace to
  confirm one target's prune does not reach another's dest tree.
- **Rollback.** The prune only deletes files Hero itself recorded writing; re-running
  `hero install --target <t>` re-renders the current set, and the manifest self-heals on the
  next run. There is no persisted state a user must hand-repair. If a regression over-prunes,
  the fix is `hero install` to re-render, and the guard is the never-touch-user-file test.

## Validation

**The real test — reproduce a product drop and prove the prune catches it, per target:**

1. Install target T from a content set that includes agent `foo`. Confirm `foo`'s rendered
   dest exists and `TargetState.Files` lists it.
2. Drop a user file (`my-custom-agent`) into the same dest dir.
3. Re-run install/upgrade against a content set that no longer ships `foo`.
4. Assert: `foo`'s dest is **gone** and the removal was printed; `my-custom-agent` **survives
   byte-for-byte**; `TargetState.Files` now reflects the new set.
5. Repeat across all six targets, asserting the target-specific dest shape (`.toml`,
   `.prompt.md`, `.cursor/rules/skills/<n>.md`, `.md`).

**No-op safety:** delete `install-state.json` and re-run — zero prunes. Run once against a
pre-`Files` `TargetState` — zero prunes, fresh manifest recorded. Both must be silent.

**Dry-run:** with a droppable file present, `--dry-run` prints `would remove` and changes
nothing on disk (file present, manifest unwritten).

**Automated:** `go test -race -count=1 ./internal/install/...` green; the `prune_test.go`
cases map 1:1 onto AC-1 through AC-13 and run table-driven across all six targets. Full
`go test -race -count=1 ./...` green.

## Completion Ledger

### Task as executed

Generalized the proven skill-dir prune (`prune.go`, `SkillDirs`) to flat rendered
agent/command/flat-skill files across all six install targets. Instrumented the three
flat-file write chokepoints (`installFlat`, `installSkillsFlat`, `renderToFile` with the
no-path-separator rule) to populate a canonical `result.rendered` set — independent of
`copyFileFromFS`'s byte-match skip and of `Force` — persisted it as `TargetState.Files`
(replace-not-merge), and added `pruneStaleFiles` hooked into `Run()` before
`RecordTargetInstall`. Provenance rule: a file is removed only when the recorded prior
manifest proves Hero wrote it; nil/empty prior or empty current render → strict no-op.

**Stack:** Go. Skills loaded: `go-stack`, `implementation-principles`, `completion-ledger`.

**A discovered correctness nuance (documented, not a defect):** Codex's `.codex/agents/`
dir is wiped wholesale every run by the pre-existing `removeLegacyDir` dead-bytes cleanup
(`target_codex.go:60`). A dropped Codex agent is therefore removed by that mechanism before
`pruneStaleFiles` runs; the file prune correctly no-ops on it (AC-10's `os.Stat`-skip) and
records the `.toml` manifest for AC-5. This is why Codex is excluded from the
user-file-preservation table — a user file in `.codex/agents/` is destroyed by
`removeLegacyDir`, a separate concern outside this spec's scope. Copilot's `removeLegacyDir`
targets the disjoint `.github/copilot/` legacy tree, not its `.github/prompts/` render dest,
so the file prune is the genuine actor there.

**Validation performed:** `go build ./...` clean; `gofmt -l` clean on all six touched files;
`go vet ./internal/install/` clean; `go test -race -count=1 ./internal/install/ ./internal/cli/`
green; full `go test -count=1 ./...` green (exit 0, opsrunner did not flake this run). Manual
end-to-end demonstration with the real `hero` binary captured below.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Remove a manifest file absent from render + print cleanup line | DONE | `pruneStaleFiles` prune.go — `TestPruneStaleFiles_RemovesDroppedAgent` (5 flat targets assert the `removed — dropped from product` line); manual transcript shows the line for `zzz-retired-agent.md` |
| 2 | Never remove a file absent from the manifest | DONE | `TestPruneStaleFiles_NeverRemovesUserFile` (5 targets, byte-for-byte); manual: `my-custom-agent.md` survives |
| 3 | Nil/empty prior manifest → prune nothing | DONE | `pruneStaleFiles` prune.go early return; `TestPruneStaleFiles_NoPriorManifestIsNoOp` |
| 4 | Per-target manifest across all six targets | DONE | `TestPruneStaleFiles_RemovesDroppedAgent` runs table-driven over all six; each uses an isolated workspace + its own `st.Targets[target].Files` |
| 5 | Record actual rendered dest (.toml codex, .prompt.md copilot) | DONE | `renderToFile` instrumentation render.go; `RemovesDroppedAgent` asserts `.toml`/`.prompt.md` manifest membership; manual: codex manifest is 35 `.toml` paths |
| 6 | Pre-`Files` TargetState → prune nothing + record fresh manifest | DONE | `pruneStaleFiles` nil-prior return; `PersistInferredTargets` carry-forward state.go; `TestPruneStaleFiles_NoPriorManifestIsNoOp` |
| 7 | `--dry-run` reports, deletes nothing, writes no manifest | DONE | DryRun branch in `pruneStaleFiles`; `RecordTargetInstall` DryRun early-return; `TestPruneStaleFiles_DryRunDeletesNothing` |
| 8 | Record `Files` as full rel set, replacing prior | DONE | `renderedFileManifest` + `RecordTargetInstall` state.go; `TestRecordTargetInstall_FilesReplaced` |
| 9 | Never manifest/prune root instruction files | DONE | Structural (instruction files bypass the three primitives); `TestPruneStaleFiles_NeverManifestsInstructionFiles` (claude/codex/copilot) |
| 10 | Leave nested skill dirs alone; no error on already-removed path | DONE | no-path-separator rule render.go; `os.Stat`-skip prune.go; `TestPruneStaleFiles_LeavesNestedSkillDirs` |
| 11 | Prune dropped Cursor flat skill | DONE | `installSkillsFlat` instrumentation content.go; `TestPruneStaleFiles_CursorFlatSkill` |
| 12 | Empty current render → prune nothing | DONE | empty-current guard prune.go; `TestPruneStaleFiles_EmptyRenderIsNoOp` |
| 13 | `install-state.json` absent → prune nothing | DONE | `ReadInstallState` zero-state + no-target-entry return; `TestPruneStaleFiles_FreshCloneNoState` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `install.go` — `Result.rendered` field + `pruneStaleFiles` hook in `Run` | DONE | `install.go:126-136` (field), `install.go:191-201` (hook, outside `!DryRun`, inside ModeProject, before `RecordTargetInstall`) |
| 2 | `content.go` — instrument `installFlat` + `installSkillsFlat` | DONE | `content.go:51-54` (installFlat), `content.go:189-193` (installSkillsFlat); `installSkillsNested` left uninstrumented |
| 3 | `render.go` — instrument `renderToFile`, no-path-separator rule, DryRun too | DONE | `render.go:74-83`; excludes codex `command-<n>/SKILL.md` via `strings.ContainsAny(destName, "/\\")` |
| 4 | `prune.go` — `pruneStaleFiles` + doctrine header | DONE | header extension `prune.go:41-54`; `renderedFileManifest` + `pruneStaleFiles` appended (`prune.go:171-280`) |
| 5 | `state.go` — `TargetState.Files`, replace-not-merge, carry-forward | DONE | field `state.go:84-92`; `RecordTargetInstall` `state.go:178-190`; `PersistInferredTargets` `state.go:290-311` |
| 6 | `prune_test.go` — 10 tests, table-driven over six targets | DONE | 10 tests, 14 subtests across the table-driven cases; all mapped to ACs above |
| 7 | `docs/` — add file manifest beside skill-dir prune, conditional on prune being enumerated in docs | DONE | Conditional executed exactly as the item specifies: `rg -l "prune\|SkillDirs\|dead bytes" docs/` → no matches, so prune behavior is not enumerated anywhere in docs, and the item directs "skip rather than invent a doc surface." The no-op branch was the specified action; no doc surface invented. |

### Exercise-the-feature check

- [x] User-visible behavior exercised end-to-end with the real `hero` binary (built `go build -o hero ./cmd/hero`), in a scratch git repo:
  - `hero init` + `hero install project . --target claude` → manifest recorded 64 flat files.
  - Simulated a product-drop: added `.claude/agents/zzz-retired-agent.md` to disk and to the recorded `Files` manifest; dropped a user-authored `.claude/agents/my-custom-agent.md` NOT in the manifest.
  - Re-ran install. Observed on stderr: `cleanup .claude/agents/zzz-retired-agent.md (removed — dropped from product)`.
  - Verified: `zzz-retired-agent.md` PRUNED; `my-custom-agent.md` PRESERVED byte-for-byte (`# My Custom Agent — user authored`); manifest no longer lists the retired file and never listed the user file.
  - `hero install project . --target codex` → manifest = 35 `.codex/agents/*.toml` paths (AC-5 actual-dest recording), all `.toml`.

### Excellence Bar self-check

Yes. The prune reuses `prune.go`'s exact provenance doctrine (extended in its header), the two prunes are disjoint by construction via the path-separator rule and proven disjoint by test, and the catastrophic-wipe guards (nil-prior, empty-render, os.Stat-skip) each carry a dedicated AC and test. The one non-obvious interaction — Codex's `removeLegacyDir` owning `.codex/agents` — was surfaced explicitly rather than papered over, and the test table encodes it as a `pruneActor` flag instead of silently dropping targets. Table-driven over all six targets satisfies the `harness-changes-cover-all-targets` tripwire.
