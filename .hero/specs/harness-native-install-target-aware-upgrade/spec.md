---
title: "Harness-Native Install — Target-Aware Instruction Files and Upgrade"
slug: harness-native-install-target-aware-upgrade
type: feature
status: completed
domain: engineering
priority: P0
size: large
tags: [install, upgrade, harness, agents-md, claude-md, target-aware]
created: 2026-07-12
relations:
  - target: single-source-install-p1-agents-md
    kind: supersedes
  - target: single-source-install
    kind: parent
  - target: install-target-emits-both-claude-and-agents-md
    kind: related
supersedes: [single-source-install-p1-agents-md, install-target-emits-both-claude-and-agents-md]
completed_at: 2026-07-12T19:59:17Z
---

# Harness-Native Install — Target-Aware Instruction Files and Upgrade

## Goal

`hero install` writes **each harness the instruction file it natively
reads, and nothing else**. Claude Code gets `CLAUDE.md`; every other
target (`codex`, `opencode`, `cursor`, `copilot`, `generic`, `aider`,
…) gets `AGENTS.md`. Installing multiple targets that include Claude
produces **both** files, each carrying the same Hero-managed block
body. `hero upgrade` regenerates the managed region **only in the
native files of the targets that were actually installed** — it never
conjures a `CLAUDE.md` for a repo where Claude was never a target.

The Hero-managed-region mechanics from `single-source-install-p1`
(versioned markers, byte-for-byte preservation outside markers,
idempotent regeneration, shared body generator) are **kept unchanged**.
The only two things that change: **which files each target writes**,
and **making upgrade target-aware via persisted install state**.

## Kickoff

Reverses P1's "AGENTS.md is the one root file" thesis: harness-native
instead. `--target claude` → CLAUDE.md only; every other target →
AGENTS.md only; multi-target-with-claude → both. Upgrade reads the
persisted installed-target set and regenerates only those targets'
native files.

**Status:** delivered — implemented, 86-pkg suite green, cold audit SHIP (noteworthy). Pending `hero spec verify`.

**Pick up at:** nothing to implement. Harness-native mapping is live via
`installNativeInstructionFile` (`agents_md.go`); `runClaude` no longer writes
AGENTS.md; cursor/copilot/generic now emit AGENTS.md; upgrade reads
`install-state.json` `targets` (∪ filesystem detect) with backfill; orphan
pruning is opt-in (`--prune-orphaned-instruction-files`) and never deletes a
file with user content. Open follow-ups: (1) re-send the hero-code peer advisory
when auth works; (2) decide whether `hero init` should suppress its
project-context AGENTS.md for a Claude-only repo.

→ `.hero/planning/features/harness-native-install-target-aware-upgrade/spec.md`

**Files:** `internal/install/target_claude.go:40`, `internal/install/claude_md.go`, `internal/install/state.go:138`, `internal/cli/upgrade.go:371`, `internal/cli/install.go`
**Skip:** the CLAUDE.md→AGENTS.md symlink/@import shim (P1's earlier draft, rejected); always-both dual-managed-block (current Model-B, being replaced).

## Decision & Rationale

**This spec is the source of truth for Hero's install/upgrade model.
It deliberately reverses `single-source-install-p1-agents-md`.** Future
`hero why` traversals on install behavior should land here.

### The decision

Hero is **harness-native, target-aware**:

- **Mapping rule:** Claude is the *only* target whose native root
  instruction file is `CLAUDE.md`. **Every other target uses
  `AGENTS.md`.** There is no single forced canonical root file.
- `hero install --target claude` → `CLAUDE.md` only.
- `hero install --target <non-claude>` → `AGENTS.md` only.
- `hero install` with 2+ targets where one is Claude → **both** files,
  same managed-block body in each.
- `hero upgrade` regenerates the managed region **only in the native
  files of previously-installed targets**, read from persisted state.
  If Claude was never a target, upgrade never creates `CLAUDE.md`.

### Why this reverses P1

P1 (`single-source-install-p1-agents-md`, status: delivering, P0)
declared **"AGENTS.md as the only root instruction file"** — every
harness, including Claude, gets `AGENTS.md`, and `CLAUDE.md` is at most
a same-body companion. That thesis is wrong for three reasons:

1. **No phantom instruction files.** A Claude-only install must never
   litter an `AGENTS.md` into a repo whose user only runs Claude Code
   (and vice versa). P1's model produces files no installed harness
   reads. Harness-native means the file set matches the harness set.
2. **Upgrade must be faithful to what was installed.** Regeneration has
   to be driven by the *actual* previously-installed target set, not by
   a blanket "always maintain AGENTS.md (and maybe CLAUDE.md)" rule.
   This requires install to **persist** the target set and upgrade to
   **read** it.
3. **Each harness reads its native file — no shim fragility.** No
   `CLAUDE.md → AGENTS.md` symlink and no `@AGENTS.md` import. Both are
   brittle across filesystems (Windows without Developer Mode), across
   harnesses that don't honor `@import`, and under version control.
   Writing each harness its own real file removes an entire failure
   class.

### Rejected models (named, so future sessions don't re-propose them)

- **(A) Symlink / `@import` shim** — `CLAUDE.md` is a symlink or a
  one-line `@AGENTS.md` import pointing at `AGENTS.md`. Rejected:
  filesystem-fragile (Windows), harness-fragile (not every tool
  resolves `@import`), and confusing under git. This was an *earlier
  draft inside P1* before P1 settled on same-body dual files.
- **(B) Dual-managed-block, always both** — the model **currently in
  the code**: `internal/install/claude_md.go` writes a full `CLAUDE.md`
  for `--target claude`, and `runClaude` (`target_claude.go:40-48`)
  *also* writes `AGENTS.md`. Every target gets both files. Rejected:
  produces phantom files (the exact bug
  `install-target-emits-both-claude-and-agents-md`), and gives upgrade
  no way to know which files a repo legitimately owns.

Harness-native keeps the **good part of B** (real, independent, same-body
managed-block files — no shim) while fixing B's core defect (writing
files for targets that were never installed).

### What this spec resolves

Directly resolves the bug
`install-target-emits-both-claude-and-agents-md`: after this lands,
`hero install --target claude` emits `CLAUDE.md` only.

## Problem

### Current behavior (Model B, in the code today)

- `runClaude` (`internal/install/target_claude.go:40-48`) writes **both**
  `AGENTS.md` (via `installAgentsMd`, lines 40-44) **and** `CLAUDE.md`
  (via `installClaudeMd`, lines 46-48). So `--target claude` in a clean
  repo emits two files. This is the observed bug.
- `runCodex` (`target_codex.go:88`) and `runOpenCode`
  (`target_opencode.go:47`) write `AGENTS.md` — correct for the new
  model, keep as-is.
- `runCursor`, `runCopilot`, `runGeneric` write **no** root instruction
  file at all (grep confirms only `claude`, `codex`, `opencode` call
  `installAgentsMd`). Under the new mapping rule these three must emit
  `AGENTS.md`. This is a pre-existing coverage gap the harness-native
  model closes — and honors the `harness-changes-cover-all-targets`
  tripwire (all six targets get defined behavior, not just Claude).

### Upgrade is not target-aware for instruction files

`hero upgrade` resolves its target set via `resolveUpgradeTargets`
(`internal/cli/upgrade.go:371`) → `detectInstalledTargets` →
`install.DetectInstalledTargets` (filesystem probe of harness content
dirs `.claude/`, `.codex/`, …) with a `version.json` `LastInstall`
fallback. It then invokes `install.Run` per target, which under Model B
writes both files for Claude. There is no persisted, authoritative
"these targets were installed → maintain exactly these instruction
files" signal driving instruction-file regeneration. That's what makes
upgrade unfaithful.

### Existing installs already have both files

Because Model B is live, real repos (including this one — see
`.hero/install-state.json` recording `claude` + `codex`) already have
**both** `CLAUDE.md` and `AGENTS.md` on disk. The new model must not
destructively delete either. Migration safety is the top risk (see
below).

## Design

### Kept from P1 (do not redesign)

The managed-region machinery is correct and stays exactly as-is. This
spec changes *which files it is applied to*, not how it works:

- Versioned markers `<!-- hero:managed-start v=… -->` /
  `<!-- hero:managed-end -->`; block at the top after the H1.
- User content outside the markers preserved byte-for-byte.
- Idempotent regeneration (second run → zero filesystem writes;
  `installManagedMarkdown`, `agents_md.go:238` short-circuits when
  `newContent == existing`).
- Hand-edit refusal without `--force-managed`; `hero check` drift
  warning surface.
- **Same body generator** for `CLAUDE.md` and `AGENTS.md`
  (`defaultSections` → `newAgentsMdBodySection` +
  `snapshot.NewPointerSection`, `agents_md.go:68`). Both files get
  byte-identical managed bodies.

### The per-target instruction-file mapping (all six targets defined)

| Target | Native root instruction file | Installer change |
|---|---|---|
| `claude` | `CLAUDE.md` | **Stop writing AGENTS.md.** Drop `installAgentsMd` call in `runClaude` (`target_claude.go:40-44`); keep `installClaudeMd`. |
| `codex` | `AGENTS.md` | No change — already writes AGENTS.md. |
| `opencode` | `AGENTS.md` | No change — already writes AGENTS.md. |
| `cursor` | `AGENTS.md` | **Add** `installAgentsMd`. (Cursor Agent mode reads AGENTS.md natively; `.cursor/rules/` content dirs are separate and out of scope here.) |
| `copilot` | `AGENTS.md` | **Add** `installAgentsMd`. (Copilot reads AGENTS.md; `.github/copilot-instructions.md` is a separate content surface, out of scope.) |
| `generic` | `AGENTS.md` | **Add** `installAgentsMd`. |

Multi-target composition falls out naturally: each target writes its own
native file, so `--target claude --target codex` yields `CLAUDE.md`
(from `runClaude`) + `AGENTS.md` (from `runCodex`), both idempotent,
both same body. Two non-Claude targets both write the same root
`AGENTS.md` (idempotent second write).

Concretely, replace the ad-hoc per-target calls with a single helper
that consults the mapping so no target can silently miss coverage:

- Add `nativeInstructionFile(t Target) string` in `install` returning
  `"CLAUDE.md"` for `TargetClaude`, `"AGENTS.md"` for all others.
- Each `run<Target>` calls one `installNativeInstructionFile(opts,
  result)` that dispatches on the mapping, rather than hand-wiring
  `installAgentsMd`/`installClaudeMd`. This makes "cover all six
  targets" structurally enforced and testable.

### Persisted installed-target state — store decision

**Recommendation: reuse `.hero/install-state.json` `targets` map. Do
NOT add a field to `hero.json`, and do NOT use the contract registry.**

Rationale:

- `install-state.json` **already exists** (`internal/install/state.go`),
  already has a `targets map[string]TargetState` populated on every
  project-mode install via `RecordTargetInstall` (`state.go:138`, called
  at `install.go:187`), already carries a `schema_version` for
  compatible evolution, and already records `installed_at` /
  `last_updated_at` / per-target `hero_version`. The live file on this
  repo already lists `claude` and `codex`. **The set of installed
  targets is therefore already persisted** — the missing piece is that
  *upgrade doesn't read it* for instruction-file decisions.
- `hero.json` is user-facing project config (tracker, team, conventions).
  Install bookkeeping does not belong there.
- `install-contract-registry-foundation` (completed) is a per-`(target,
  kind)` *output-shape validator* registry — it proves rendered files
  match a declared contract. It is not an install-state store and is the
  wrong home.

Schema (no breaking change; `targets` already exists):

- **Field:** `targets` (`map[string]TargetState`) in
  `install-state.json`. A key present ⇒ that target was installed and
  its native instruction file must be maintained.
- **Written when:** every successful project-mode install, via the
  existing `RecordTargetInstall`. No change needed to the write path —
  it already accumulates targets across installs and never drops one
  except on explicit uninstall (correct "previously-installed"
  semantics).
- **Read by upgrade:** new logic in `resolveUpgradeTargets`
  (`upgrade.go:371`) reads `install-state.json` `targets` as the
  authoritative previously-installed set (union with the existing
  filesystem `DetectInstalledTargets` probe, so a repo whose state file
  predates this work still resolves targets). The resolved set drives
  which `run<Target>` runs, and therefore which native instruction file
  is regenerated.

### Upgrade instruction-file policy (precise algorithm)

On `hero upgrade` (after computing the target set `T`):

1. **Resolve `T`** = `install-state.json` `targets` ∪ filesystem
   `DetectInstalledTargets` (content-dir probe). `--target` narrows `T`
   to the requested subset (validated against the six known targets, as
   today).
2. **If `T` is empty and no state file exists → backfill** (see below),
   producing an inferred `T`, persist it, then continue.
3. For each `t ∈ T`, run the target installer. Because `runClaude` now
   writes only `CLAUDE.md` and non-Claude installers write only
   `AGENTS.md`, upgrade regenerates exactly the native files of `T`.
4. **Faithfulness guarantee:** `claude ∉ T` ⇒ `runClaude` never runs ⇒
   `CLAUDE.md` is never created. `claude ∈ T` ⇒ `CLAUDE.md`'s managed
   region is regenerated.
5. **Never orphan, never delete:** if an instruction file exists on disk
   that isn't implied by `T` (e.g. a legacy Model-B `AGENTS.md` left by a
   Claude-only install), upgrade **maintains its managed region if
   already Hero-managed** (so it doesn't rot) but **does not delete it**
   and **does not create new files** for un-inferred targets. Deletion of
   such an orphan is opt-in only (see migration safety).

### Backfill for pre-state installs

A repo installed before `install-state.json` carried a usable `targets`
set (empty map, or no file) has no persisted set. Upgrade must infer it:

1. **Content-dir probe first (authoritative for the target SET):**
   `DetectInstalledTargets` walks `.claude/`, `.codex/`, `.opencode/`,
   `.cursor/rules/`, `.github/copilot/`, `.ai/` for real content. Any
   detected dir ⇒ that target was installed.
2. **Instruction-file presence (secondary signal, for file maintenance
   only):** `CLAUDE.md` present ⇒ Claude was a target; `AGENTS.md`
   present ⇒ at least one non-Claude target was.
3. **Persist** the inferred set into `install-state.json` `targets`
   (with `installed_at`/`last_updated_at` = now, `hero_version` = binary
   version), then proceed with the upgrade using that set.

**Ambiguous cases — explicit handling:**

- **Both `CLAUDE.md` and `AGENTS.md` present** (the common Model-B legacy
  state): trust the content-dir probe for the SET. If only `.claude/`
  content exists (a Claude-only Model-B install that wrongly wrote
  `AGENTS.md`), infer `T = {claude}`. The stray `AGENTS.md` is **not**
  adopted as evidence of a non-Claude target and is **not** deleted — it
  is maintained-if-managed per the orphan rule, or left for the opt-in
  prune. Prevents inferring a phantom target from a phantom file.
- **Only a legacy Hero-managed `CLAUDE.md` stub** (entire content inside
  Hero markers, no content dirs): infer `T = {claude}`; regenerate its
  managed region in place (the existing `installManagedMarkdown` handles
  legacy → versioned marker migration).
- **Neither file, no content dirs:** `T` is empty → upgrade stamps the
  version only (existing `upgrade.go:133-147` "no target detected" path).
  Nothing is created.
- **`AGENTS.md` present, no `CLAUDE.md`, only non-Claude content dirs:**
  infer `T` = the detected non-Claude targets; regenerate `AGENTS.md`
  only. Claude is not inferred.

### Migration safety — no destructive deletes

**Invariant: this migration never deletes a user's `AGENTS.md` or
`CLAUDE.md`.** The managed region continues to be maintained in whatever
instruction files exist. An install/upgrade under the new model against
a Model-B repo (both files present) leaves both files in place and keeps
both managed regions current. This is the safe default and satisfies the
"preserve existing files" requirement.

**Orphaned-file pruning is explicit opt-in only:**

- Add a flag `--prune-orphaned-instruction-files` (default **off**) to
  `hero install` and `hero upgrade`.
- When set, and only when set, an instruction file whose target is not in
  the resolved set `T` **and** whose content is *entirely* Hero-managed
  (whole file inside the markers — reuse the P1 "Hero-managed only"
  detection) may be deleted. A file with any user content outside the
  markers is **never** deleted, even with the flag.
- Without the flag, orphans are preserved and their managed region kept
  current (never rotting).

This means a user who ran Claude-only under Model B (and thus has a
phantom `AGENTS.md`) sees no data loss on upgrade; if they want the
phantom gone, they opt in explicitly, and even then only a
fully-Hero-managed phantom is removed.

### `hero check` and the tripwire note

- `hero check` keeps the P1 managed-region hand-edit drift warning
  (`check.go` already surfaces managed-block drift). Add a check signal:
  an instruction file exists on disk for a target **not** in the
  persisted `targets` set → surface as an informational note
  ("`AGENTS.md` present but no non-Claude target recorded; run
  `hero install --prune-orphaned-instruction-files` to remove, or ignore
  to keep"). Informational, not a failure.
- **Tripwire note:** the `harness-changes-cover-all-targets` tripwire
  currently states "AGENTS.md is the canonical instruction surface …
  CLAUDE.md is the Claude-only view, fed by the CLAUDE.md → AGENTS.md
  symlink / @AGENTS.md import." This spec **intentionally revises** that
  direction (harness-native, no shim). The tripwire text is being updated
  separately to match — do **not** treat the current wording as a blocker
  for this spec. This design still honors the tripwire's **other**,
  load-bearing intent: a harness-facing change must define behavior for
  **all six** install targets, not just Claude — which the per-target
  mapping table above does explicitly.

## Acceptance Criteria

- WHEN `hero install --target claude` runs in a clean repo THE SYSTEM
  SHALL create `CLAUDE.md` and SHALL NOT create `AGENTS.md`.
- WHEN `hero install --target <non-claude>` (codex, opencode, cursor,
  copilot, generic) runs in a clean repo THE SYSTEM SHALL create
  `AGENTS.md` and SHALL NOT create `CLAUDE.md`.
- WHEN `hero install` runs with multiple targets including claude THE
  SYSTEM SHALL create both `CLAUDE.md` and `AGENTS.md`, each containing
  the same Hero-managed block body.
- WHEN `hero install` completes THE SYSTEM SHALL persist the set of
  installed targets in `.hero/install-state.json`.
- WHEN `hero upgrade` runs THE SYSTEM SHALL regenerate the managed region
  only in the native instruction files of previously-installed targets.
- IF claude was never a previously-installed target THEN `hero upgrade`
  SHALL NOT create `CLAUDE.md`.
- WHEN `hero upgrade` runs and claude WAS a previously-installed target
  THE SYSTEM SHALL regenerate `CLAUDE.md`'s managed region.
- WHEN no persisted target state exists (pre-state install) THE SYSTEM
  SHALL infer the prior target set from existing content dirs and
  instruction files, persist the inferred set, and proceed without
  creating files for un-inferred targets.
- THE SYSTEM SHALL preserve all user content outside the managed markers
  in every instruction file, byte-for-byte.
- THE SYSTEM SHALL be idempotent: a second install/upgrade run against
  unchanged state produces zero filesystem changes.
- THE SYSTEM SHALL NOT auto-delete an existing `AGENTS.md` or `CLAUDE.md`
  created under a prior install model; deletion is opt-in only via
  `--prune-orphaned-instruction-files`, and even then only for a file
  whose entire content is Hero-managed.
- WHEN `--prune-orphaned-instruction-files` is set AND an instruction
  file's target is not in the resolved set AND the file is entirely
  Hero-managed THE SYSTEM SHALL delete it; WHEN the same file contains
  any user content outside the markers THE SYSTEM SHALL preserve it.
- THE SYSTEM SHALL define instruction-file behavior for all six install
  targets (claude, codex, opencode, cursor, copilot, generic), not
  claude alone.

## Changes

- `internal/install/agents_md.go` — add `nativeInstructionFile(Target)
  string` (claude → CLAUDE.md, else AGENTS.md) and
  `installNativeInstructionFile(opts, result)` dispatching on it; both
  files continue to share `defaultSections`.
- `internal/install/target_claude.go` — remove the `installAgentsMd`
  block (lines 40-44); route CLAUDE.md through
  `installNativeInstructionFile` (keeps `installClaudeMd` semantics).
- `internal/install/target_cursor.go`, `target_copilot.go`,
  `target_generic.go` — add `installNativeInstructionFile` so these
  three emit `AGENTS.md` (they emit no root file today).
- `internal/install/target_codex.go`, `target_opencode.go` — switch to
  `installNativeInstructionFile` for consistency (behavior unchanged —
  still AGENTS.md).
- `internal/install/claude_md.go` — keep `installClaudeMd` managed-block
  behavior; ensure it is reached only for `TargetClaude`.
- `internal/install/state.go` — no schema change; confirm
  `RecordTargetInstall` remains the authoritative writer of `targets`.
  Add a helper `PreviouslyInstalledTargets(projectRoot) []Target`
  reading `install-state.json` `targets` for upgrade to consume, plus
  an inference/backfill helper `InferInstalledTargets(projectRoot)`
  combining content-dir probe + instruction-file presence, and a
  persist path for the inferred set.
- `internal/cli/upgrade.go` — `resolveUpgradeTargets` reads persisted
  `targets` (union with filesystem detection); wire backfill for the
  empty-state case; add `--prune-orphaned-instruction-files`; implement
  the orphan-maintain-not-delete policy.
- `internal/cli/install.go` — add `--prune-orphaned-instruction-files`;
  update `--target` help text if needed (set unchanged).
- `internal/cli/check.go` — add the informational orphan-instruction-file
  signal; keep the managed-region drift warning.
- `internal/install/*_test.go` — per-target table tests asserting the
  exact file set for each of the six targets (single, and multi-target
  combinations including/excluding claude); upgrade target-awareness
  tests (claude-never-installed → no CLAUDE.md; claude-installed →
  CLAUDE.md regenerated); backfill/inference tests for each ambiguous
  case (both files, stub-only, neither, agents-only); prune opt-in tests
  (managed-only deleted, user-content preserved); idempotency.
- `docs/cli/*.md` — document the harness-native model, the per-target
  file mapping, the persisted-target-set behavior, the migration story,
  and `--prune-orphaned-instruction-files`.
- `cmd/hero/...` — bump help text for `install` and `upgrade` where
  behavior changes.

### Delivered (actual files touched)

- `internal/install/agents_md.go` — added `nativeInstructionFile`,
  `installNativeInstructionFile`, `instructionFileIsHeroManagedOnly`,
  `ApplyOrphanInstructionFilePolicy` (+ `InstructionFileOrphanAction`
  constants); tightened `resolveAgentsMdPath` global branch to `""` for
  non-codex/opencode targets.
- `internal/install/target_claude.go` — dropped the `installAgentsMd`
  block; routes CLAUDE.md through `installNativeInstructionFile`.
- `internal/install/target_codex.go`, `target_opencode.go` — switched to
  `installNativeInstructionFile`.
- `internal/install/target_cursor.go`, `target_copilot.go`,
  `target_generic.go` — emit AGENTS.md via `installNativeInstructionFile`
  (generic replaced the legacy `installInstructionsMd` stub; copilot keeps
  its separate `.github/copilot-instructions.md`).
- `internal/install/state.go` — added `PreviouslyInstalledTargets`,
  `InferInstalledTargets`, `claudeMdIsHeroManaged`,
  `PersistInferredTargets`.
- `internal/cli/upgrade.go` — union of persisted+detected targets, backfill,
  `--prune-orphaned-instruction-files`, orphan maintain-not-delete
  (`handleOrphanedInstructionFiles`, `unionTargets`), help text.
- `internal/cli/install.go` — `--prune-orphaned-instruction-files`, harness-
  native help text, post-install prune wiring.
- `internal/cli/check.go` — informational orphan-instruction-file signal
  (`detectOrphanInstructionFiles`, `fileExists`).
- `internal/cli/helpers_test.go` — reset the two new flags.
- `internal/install/harness_native_test.go` (new),
  `internal/cli/upgrade_target_aware_test.go` (new) — coverage.
- Updated `internal/install/harness_smoke_test.go`,
  `internal/install/install_test.go` — old "both files for claude" model
  assertions flipped to harness-native.
- `web/docs/src/cli/server-and-mcp.md` — harness-native model, per-target
  mapping, persisted-target behavior, migration + prune flag.
- `cmd/hero/main.go` — thin wrapper (26 lines); the `install`/`upgrade`
  help surface lives in the cobra definitions in `internal/cli`, updated
  there.

## Boundaries

- **Not in scope:** changing the *content* Hero generates into the
  managed region — same body generator as P1, unchanged.
- **Not in scope:** the symlink / `@import` shim — rejected model (A),
  not implemented.
- **Not in scope:** harness-specific *content* surfaces beyond the root
  instruction file (`.cursor/rules/*.mdc`, `.github/copilot-instructions.md`,
  Aider `CONVENTIONS.md`). Those are separate content dirs; this spec
  governs the root instruction file only. The mapping rule assigns those
  targets `AGENTS.md` as their root instruction file per the product
  decision.
- **Not in scope:** global-mode (`~/.claude`, `~/.codex`) instruction-file
  changes — project mode is the target; global-mode path (`resolveClaudePaths`
  / `resolveAgentsMdPath` ModeGlobal branches) is left as-is unless a
  test surfaces a regression.
- **Not in scope:** removing `install-state.json` `HostCapabilities` /
  P2 mode fields — untouched.

## Cross-repo coordination (pending)

The hero-code peer holds a handed-off bug, `hihcp-agents-md-harness-agnostic`
("Produce Harness-Agnostic AGENTS.md, **Demote CLAUDE.md**"), which encodes the
now-retired convergence model. An advisory `hero peer call hero-code` to flag
this reversal was composed and validated (dry-run envelope OK,
call_id `18c19fd781c3…`) but the **live dispatch failed with a 401 auth error**
in this environment — it was **not delivered**. **Action still owed:** re-send
the advisory when peer auth is available, so hero-code doesn't deliver the
"demote CLAUDE.md" model in its own repo. Under the new model, CLAUDE.md is
Claude's co-equal native instruction file, not a demoted shim.

## Completion Ledger

All 13 acceptance criteria and all 10 Changes items: **DONE**. Cold audit
verdict SHIP (noteworthy, high confidence) —
[delivery-audit.md](delivery-audit.md).

**Acceptance criteria (13/13 DONE):** per-target file set for all six targets
(`TestHarnessNative_PerTargetFileSet`); claude→CLAUDE.md-only, non-claude→AGENTS.md-only;
multi-target→both same body (`_MultiTargetIncludingClaude`, `_SameManagedBody`);
install persists targets (`_InstallPersistsTargets`); upgrade regenerates only
prev-installed targets' native files, claude-never→no CLAUDE.md, claude-was→regenerated
(`TestUpgrade_ClaudeNeverInstalled_NoCLAUDEMd`, `_ClaudeInstalled_RegeneratesCLAUDEMd`);
backfill inference for all four ambiguous cases (`TestInferInstalledTargets_*`,
`TestUpgrade_Backfill_StubOnly`); user content preserved byte-for-byte; idempotent
(`_Idempotent`); no auto-delete + prune opt-in managed-only
(`TestOrphanPolicy_*`, `TestUpgrade_OrphanAgentsMd_MaintainedByDefault`, `_OrphanPrune_OptIn`);
all-six-targets coverage via `installNativeInstructionFile` chokepoint.

**Changes (10/10 DONE):** `nativeInstructionFile`/`installNativeInstructionFile`
(`agents_md.go`); `target_claude.go` drops AGENTS.md write; cursor/copilot/generic
now emit AGENTS.md (closed a no-root-file gap); codex/opencode routed through
chokepoint (unchanged behavior); `state.go` adds PreviouslyInstalled/Infer/PersistInferred;
`upgrade.go` persisted∪detected + backfill + `--prune-orphaned-instruction-files`;
`install.go` flag; `check.go` informational orphan note; docs → `web/docs/src/cli/`;
help text in cobra defs.

**Validation:** `go build ./cmd/hero` OK · `go test ./...` 86 packages, 0 FAIL ·
exercised end-to-end with the built binary in temp dirs (never the real tree).

**Noteworthy (per audit):** (1) claude+codex isn't byte-identical because codex
appends its pre-existing codex-specific addendum (out of scope per Boundaries) —
the same-body test uses claude+cursor. (2) `hero init`'s project-context AGENTS.md
is a separate surface, not in scope. (3) The hero-code peer advisory is still owed
(401 auth) — see Cross-repo coordination above.

## Risks

- **Backfill mis-inference (top risk).** The installed-target state store
  and its backfill are the highest-stakes surface. If backfill
  over-infers a target from a phantom Model-B file, upgrade could keep
  maintaining (or, with the prune flag, delete) a file the user didn't
  intend. Mitigation: content-dir probe is authoritative for the SET;
  instruction-file presence only ever triggers *maintain-if-managed*,
  never *create* and never *delete-without-opt-in*. Every ambiguous case
  is enumerated in the design and must have a dedicated test.
- **Rollback implication.** If this change ships and must be reverted, a
  repo installed under the new model (Claude-only, `CLAUDE.md` only) that
  rolls back to Model-B code will, on the next install/upgrade, re-emit a
  phantom `AGENTS.md`. That is non-destructive (both files carry the same
  body) and self-heals when the new model is re-applied — but note it in
  the release. No persisted-state format change blocks rollback:
  `targets` already exists in the shipped schema, so an older binary
  reading the file simply ignores the semantics.
- **Multi-target write ordering.** When two non-Claude targets both write
  root `AGENTS.md`, the second write must be a no-op (idempotent). Relies
  on `installManagedMarkdown`'s `newContent == existing` short-circuit —
  covered, but must be asserted in a multi-target test.
- **Tripwire divergence window.** Until the `harness-changes-cover-all-targets`
  tripwire text is updated, its "AGENTS.md is canonical / CLAUDE.md is a
  shim" wording contradicts this spec. Resolved out-of-band; flagged here
  so a reviewer doesn't block on it.

## Mission Fit

> "Does this make the next agent session start smarter than the last one
> ended — and does it raise the floor for everyone?"

Yes. Every harness reads exactly the instruction file it natively looks
for, containing the same Hero-managed context, with no phantom files to
confuse the model about which file is authoritative. Upgrade stays
faithful to what was installed, so context doesn't silently drift or
sprout files no harness reads. The floor rises for every multi-harness
team — and for the Claude-only user whose repo stays clean.
