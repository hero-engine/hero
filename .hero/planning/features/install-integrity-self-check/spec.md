---
title: "Install integrity is self-verifiable — hero check detects a damaged, stale, or incomplete install"
slug: install-integrity-self-check
type: feature
status: planning
domain: engineering
priority: P1
size: medium
created: 2026-07-15
tags: [install, hero-check, managed-region, harness, integrity]
relations:
  - target: agents-md-erased-by-snapshot-pointer-writer
    kind: related
  - target: codex-install-broken
    kind: related
---

# Install integrity is self-verifiable — `hero check` detects a damaged, stale, or incomplete install

## Context

The snapshot pointer writer silently replaced the entire `hero:managed` region of
`AGENTS.md` with a pointer-only section on any ordinary `hero snapshot --project` or
`hero next checkpoint` run — 232 lines to a 7-line stub. Five of six harness targets
(codex, opencode, cursor, copilot, generic) started completely cold as vanilla coding
agents. It ran for two months, ate the file twice (May 31, Jun 9), and produced **zero
symptoms**. It was found by accident.

The eraser is fixed and guarded by `TestHarness_InstalledContentSurvivesOrdinaryCommands`
(`internal/install/harness_smoke_test.go:330`), table-driven over all six targets and
gating CI. But that guard only protects *this repo's* CI. In a user's repo, nothing
detects a damaged install: not a future writer with the same bug, not a hand-edit, not a
partial or failed install, not a version upgrade leaving stale content.

**The detection gap is the durable problem; the eraser was one instance of it.** This
spec closes the gap.

### Reviving two items that were designed, then skipped

`codex-install-broken` (completed 2026-06-09) designed both of the following and SKIPPED
both. Their triage-away as "P3 priority" and "lower priority" is a direct cause of the
two-month blindness — of its six fixes, the three that shipped all WRITE content and the
three that were cut all DETECT problems.

- **Its Fix 4** — "Add `hero check` advisories for Codex completeness." SKIPPED, "P3
  priority; separate delivery pass."
- **Its Fix 6** — "Auto-reinstall on stale managed regions." SKIPPED, "Lower priority;
  upgrade path improvement." Original text: *"`hero upgrade` and `hero install` should
  detect when the on-disk managed region is missing expected sections and auto-regenerate.
  The version stamp (`v=dev`) in the marker already exists for this purpose but isn't
  checked against the current content hash."*

They are one idea: **know what install should have produced, compare to what is on disk,
report the delta.** Treat their revival as the point of this spec.

### Mission fit

> *"Does this make the next agent session start smarter than the last one ended — and does
> it raise the floor for everyone, not just the senior dev who already knows what to ask?"*

A silently gutted `AGENTS.md` means every session in five of six harnesses starts cold.
That is the exact failure Hero exists to prevent, and nothing currently notices. It raises
the floor specifically: the senior dev eventually notices Codex acting dumb and reinstalls;
everyone else just gets a worse agent forever, and never learns why.

## Goal

`hero check` gains an `install-integrity` category that, for **each installed target**,
re-renders the managed body install would produce right now, compares it to the body on
disk in that target's native instruction file, and reports the delta as a `fail`
(damaged — expected sections missing) or `warn` (stale — content differs), naming the
target, the file, and the repair command. It stays **silent** for targets that were never
installed, and it **never** flags or touches user content outside the managed markers.
`hero check` reports only; `hero install` / `hero upgrade` remain the sole repair path.

Done means: reintroduce the eraser, run `hero snapshot --project`, run `hero check` — and
`hero check` says the install is damaged, for all five AGENTS.md targets and for CLAUDE.md.

## Kickoff

Adds an `install-integrity` check to `hero check` so a gutted or stale `AGENTS.md` /
`CLAUDE.md` gets reported instead of silently making every agent session start cold.

**Status:** planning — spec just landed, no code yet.

**Pick up at:** add `internal/install/integrity.go` with `CheckIntegrity(projectRoot)`,
re-rendering the body via `defaultSections` + `managed.Writer.RenderBody` and comparing
against `managed.FindManagedRegion(onDisk).Body`. Compare **bodies, not regions** — the
`v=` marker stamp changes on every version bump and would false-positive.

→ `.hero/planning/features/install-integrity-self-check/spec.md`

**Files:** `internal/install/agents_md.go:66,214`, `internal/managed/marker.go:82`,
`internal/managed/region.go:136`, `internal/cli/check.go:107,133`, `internal/install/state.go:206`

**Skip:** don't use `version.json` `installed_files` hashes as the oracle — root
instruction files are not in that map (240 entries, zero of them). Don't auto-repair from
`hero check`.

## Problem

There is no oracle. Nothing in the codebase can answer "is the install on disk what
install would produce?" Concretely:

1. **`hero check` has no install category at all.** Its closest neighbor,
   `detectOrphanInstructionFiles` (`internal/cli/check.go:107`), only asks *does a root
   instruction file exist whose target isn't recorded* — a file-presence question. A
   7-line stub passes it: the file exists.
2. **The `v=` version stamp is never read.** `<!-- hero:managed-start v=dev -->` records
   the writing binary's version, not the content. It cannot distinguish 232 correct lines
   from a 7-line stub written by the same binary.
3. **Install-time tests assert at the one moment the file cannot be wrong.** The class of
   test that was missing — *"installed content survives ordinary use"* — now exists, but
   only in this repo's CI. Nothing ships to users.

## Approach

### Design decision 1 — the oracle for "should": re-render in-memory

**Decision: re-render the managed body via the same `defaultSections` install uses, and
compare it to the on-disk body extracted by `managed.FindManagedRegion`.**

Three candidates were considered. The evidence is decisive.

| Oracle | Verdict |
|---|---|
| **Re-render in-memory** | **Chosen.** Self-maintaining, structurally correct, mechanism already exists. |
| **Persisted content hashes** (`version.json` `installed_files`) | **Disqualified — structurally blind to the actual failure.** |
| **`v=` marker version stamp** | **Disqualified — encodes version, not content.** |

**Why the hash oracle is disqualified, concretely.** `.hero/version.json` `installed_files`
holds 240 sha256 entries and **not one of them is a root instruction file** — no
`AGENTS.md`, no `CLAUDE.md`. Verified:

```
$ python3 -c "import json; f=json.load(open('.hero/version.json'))['installed_files']; \
    print(len(f), [k for k in f if 'AGENTS.md' in k or 'CLAUDE.md' in k])"
240 []
```

The one file that actually got gutted is the one file the hash map does not cover. This
is not an oversight to patch — it is **structurally correct**, and it is why the hash
oracle can never work here. Root instruction files are *partial-file* writes: Hero owns
the managed region, the user owns everything outside it. A whole-file hash would go stale
the moment a user legitimately edits their own prose above the markers, which is a
guaranteed false positive and a direct violation of design constraint 6 below. Hashing is
right for the 240 files Hero owns end-to-end; it is wrong for the two it co-owns.

Two further nails: hashes **go stale across versions** (every content change to a domain
pack invalidates the recorded hash, so the oracle reports "damaged" on a perfectly healthy
upgrade), and `version.json`'s `last_install` records a *single* target (`"target":
"codex"`), not a set — so it cannot answer per-target questions at all.

**Why the `v=` stamp is disqualified.** It records the binary version that wrote the
region. Both the 232-line region and the 7-line stub were written by the same binary and
carry the same `v=dev`. Fix 6's original text proposed checking the stamp "against the
current content hash" — but there is no content hash to check it against, for the reason
above. The stamp is a red herring; it should be left alone.

**Why re-render wins.** It is *self-maintaining* — it always reflects the currently
embedded content, so it cannot go stale across versions the way a persisted hash does. It
requires no new persisted state, so it works on a fresh clone. And the mechanism is
**already built and already exported**:

- `managed.Writer.RenderBody(ctx)` (`internal/managed/region.go:136`) — renders the body
  with no filesystem writes. Already exposed for exactly this class of caller ("callers
  that need to see the body without writing").
- `managed.FindManagedRegion(content) Region` (`internal/managed/marker.go:82`) — extracts
  the on-disk body. Its `Region.Body` doc comment already reads: *"This is what callers
  compare against newly-generated content to detect user edits inside the region."*
  **The comparison surface was designed for this and never wired up.**
- `install.defaultSections(opts, path)` (`internal/install/agents_md.go:214`) — the
  canonical section list, shared by AGENTS.md and CLAUDE.md.

**Compare bodies, not regions — and not `PlanContent`.** `PlanContent` returns
`(existing, next)` full-file content and is tempting (`existing != next` ⟺ region
differs, since `InsertManagedRegion` preserves outside content byte-for-byte). But `next`
embeds a freshly rendered marker line carrying the *current binary's* `v=` stamp. On any
version bump `existing != next` for every healthy install — a false positive on 100% of
upgrades. Compare `managed.FindManagedRegion(onDisk).Body` against `RenderBody(ctx)`,
which excludes the markers entirely and sidesteps the stamp.

### Design decision 2 — detect vs. auto-repair: check reports, install repairs

**Decision: `hero check` REPORTS. `hero install` / `hero upgrade` REPAIR. `hero check`
writes no files.**

Fix 6 said "auto-regenerate," and that instinct should be declined:

- **`hero check` is read-oriented by contract.** Every one of its ~15 categories reports
  and suggests; none mutate. A health command that silently rewrites root instruction
  files during a routine `hero check` is a surprise, and — given the incident that
  motivated this spec is *"a command wrote a file nobody expected it to write"* — shipping
  a second command that writes files nobody expected would be a poor lesson to draw.
- **The precedent already exists in this file.** `satellite-drift` (`check.go:478`) warns
  and prints `run 'hero install --repair'`. This check follows the identical shape.
- **Repair is already solved.** `hero install project . --target <t>` and `hero upgrade`
  already re-render the managed region idempotently — `managed.Writer.Write` is a no-op
  when content matches (`region.go:120`). Fix 6's "auto-regenerate" is, in effect,
  *already true*; what was missing was that nothing ever **told** you to run it. **The
  entire delta is detection.** Keeping repair out of scope is not deferral — the repair
  side is done.

The advisory therefore ends in the exact command to run, per target.

### Design decision 3 — artifacts in scope

**In scope: the root instruction file's managed region, per target.** That is the proven
failure and the highest-value signal — it is the file every harness reads first.

Two checks over that one artifact:

1. **Section presence** — every section `defaultSections` would render non-empty must have
   its `SectionTitle()` H2 present in the on-disk body. Catches the gutting: the 7-line
   stub retained only `## Project snapshot` and lost `## Hero — Spec-Driven AI Engineering`
   (or the pack's H1 title) and `## Hero Binary & MCP Surface`. Enumerated from
   `defaultSections`, not hardcoded, so it stays correct as packs change.
2. **Body equality** — the on-disk body must equal the re-rendered body. Catches stale
   content after an upgrade and hand-edits inside the markers.

Missing agents/commands/skills directories are **out of scope** — deliberately. The
detection gap that cost two months was the *instruction file*, and `codex-install-broken`
failed precisely by bundling eight problems under one heading until its P0 hid among its
P1s. Directory-completeness advisories are a reasonable follow-up spec; they are not this
one.

Stale skill dirs are **already solved** — `internal/install/prune.go` (landed today)
removes dirs whose source is gone, using the `SkillDirs` set persisted in
`install-state.json` as proof the dir is Hero's to remove. Do not re-solve.

### Design decision 4 — per-target coverage is mandatory

The `harness-changes-cover-all-targets` tripwire **[high]** applies. Six targets:
`opencode | cursor | claude | copilot | codex | generic`. The mapping is
`nativeInstructionFile(t)` (`internal/install/agents_md.go:66`): `claude` → `CLAUDE.md`,
the other five → `AGENTS.md`.

The check **must** route through `nativeInstructionFile` rather than reconstructing the
mapping — that function is documented as "the single source of truth for the
harness-native install mapping," and the original incident is a direct case study in what
hardcoding costs: `agentsMDPathOrEmpty` hardcoded the literal `"AGENTS.md"`, which is the
*only* reason `CLAUDE.md` escaped the eraser. That reprieve was an accident of a hardcoded
string, not a design.

The original bug was filed as **Codex-only when it hit five targets**. Do not repeat that
framing: this is not a Codex check, it is an install-integrity check that iterates targets.

### Design decision 5 — only-installed targets

An advisory must never nag about a target that was never installed. Resolve the installed
set as the **union** of `install.PreviouslyInstalledTargets(root)` (persisted
`install-state.json` `targets`) and `install.InferInstalledTargets(root)` (content-dir
probe + `claudeMdIsHeroManaged`), mirroring `resolveUpgradeTargets`
(`internal/cli/upgrade.go:413-420`) so check and upgrade agree on what "installed" means.
This honors the `harness-native-install-target-aware-upgrade` decision.

**The union is load-bearing, not belt-and-braces.** `.hero/install-state.json` is
**gitignored** (`.gitignore:69`) and therefore machine-local. On a fresh clone it does not
exist, so `PreviouslyInstalledTargets` returns `nil` and a persisted-state-only resolver
sees *zero* installed targets — the check would go silent on exactly the machine most
likely to have a cold install. `InferInstalledTargets` reconstructs the set from on-disk
content dirs and works on a fresh clone.

**Note the existing gap this must not copy:** `detectOrphanInstructionFiles`
(`check.go:107`) uses `PreviouslyInstalledTargets` *alone*. On a fresh clone with a healthy
install, that resolves to nil → both `CLAUDE.md` and `AGENTS.md` read as orphaned → a false
`orphan-instruction-files` warning. That is a pre-existing bug in an adjacent check; see
Boundaries.

### Design decision 6 — false-positive discipline

**Only the managed region is Hero's.** Users legitimately hand-edit outside the markers,
and `managed.InsertManagedRegion` preserves that content byte-for-byte. An advisory that
cries wolf gets ignored — which reproduces the original failure in a new costume, since an
ignored warning and no warning are the same warning.

The chosen mechanism gives this property structurally rather than by discipline:
`FindManagedRegion(content).Body` returns *only* the bytes between the markers. Content
outside them is never read, never compared, never mentioned. There is no code path in this
design that can see user prose, so there is no code path that can flag it.

Two further false-positive guards:

- **Region absent entirely** on an installed target → report `damaged` (the file exists but
  Hero's block is gone). Region absent on a file for a **never-installed** target → silent
  (design decision 5).
- **`v=` stamp excluded** from comparison (design decision 1) — no upgrade false positives.

## Acceptance Criteria

Each criterion is written on a single physical line: the EARS classifier requires the
trigger and `THE SYSTEM SHALL` to appear on the same line, so wrapping a bullet before the
behavior clause silently demotes it to freeform. Verify with `hero spec lint <slug>`.

- **AC-1:** WHEN `hero check` runs against an installed target whose native instruction file's managed region is missing one or more sections that `defaultSections` renders non-empty THE SYSTEM SHALL emit an `install-integrity` row with status `fail` naming the target, the file, and the missing section titles.
- **AC-2:** WHEN `hero check` runs against an installed target whose managed body differs from the re-rendered body but retains every expected section THE SYSTEM SHALL emit an `install-integrity` row with status `warn` describing the content as stale.
- **AC-3:** WHEN `hero check` runs against an installed target whose managed body is byte-identical to the re-rendered body THE SYSTEM SHALL emit no `install-integrity` row.
- **AC-4:** WHEN an `install-integrity` row is emitted THE SYSTEM SHALL include the exact repair command `hero install project . --target <target>` for the affected target.
- **AC-5:** THE SYSTEM SHALL resolve each target's instruction file through `nativeInstructionFile`, mapping `claude` → `CLAUDE.md` and each of `codex`, `opencode`, `cursor`, `copilot`, `generic` → `AGENTS.md`, for all six targets.
- **AC-6:** IF a target is not in the union of `PreviouslyInstalledTargets` and `InferInstalledTargets` THEN THE SYSTEM SHALL emit no `install-integrity` row for that target, including when that target's native instruction file exists on disk.
- **AC-7:** WHILE content outside the `hero:managed` markers differs from any prior state THE SYSTEM SHALL emit no `install-integrity` row on account of that content and SHALL leave it unmodified.
- **AC-8:** WHEN the on-disk managed region's `v=` version stamp differs from the running binary's version but the body is otherwise identical THE SYSTEM SHALL emit no `install-integrity` row.
- **AC-9:** WHEN `hero check` completes THE SYSTEM SHALL have written no files at `CLAUDE.md`, `AGENTS.md`, or any other install-managed path, regardless of findings.
- **AC-10:** IF an installed target's native instruction file exists but contains no `hero:managed` region THEN THE SYSTEM SHALL emit an `install-integrity` row with status `fail`.

## Changes

1. **`internal/install/integrity.go`** (new) — the oracle. Exported so `hero check` and
   future callers share one definition of "correct install."
   - `type IntegrityFinding struct { Target Target; File string; Kind IntegrityKind; MissingSections []string; RepairCmd string }`
   - `type IntegrityKind int` — `IntegrityDamaged` (region absent, or expected sections
     missing) and `IntegrityStale` (body differs, sections all present).
   - `func CheckIntegrity(projectRoot string) ([]IntegrityFinding, error)`:
     - Resolve installed targets: union of `PreviouslyInstalledTargets(projectRoot)` and
       `InferInstalledTargets(projectRoot)`, deduped via the `unionTargets` shape in
       `internal/cli/upgrade.go:475` (lift/share rather than duplicate).
     - For each target: build the same `Options` install uses, resolve the path via
       `nativeInstructionFile(t)` joined to `projectRoot`, and construct
       `managed.Writer{File: path, Sections: defaultSections(opts, path)}`.
     - `want, err := writer.RenderBody(ctx)` — no filesystem write.
     - Read the file. Absent → `IntegrityDamaged`. `managed.FindManagedRegion(content)`;
       `!Present` → `IntegrityDamaged` (AC-10).
     - Expected section titles: iterate `defaultSections(opts, path)`, call `Render(ctx)`,
       skip sections rendering empty (matching `renderBody`'s skip at `region.go:148`),
       collect `SectionTitle()`. Any expected title whose `## <title>` heading is absent
       from `region.Body` → `IntegrityDamaged` with `MissingSections` populated.
     - Else `region.Body != want` → `IntegrityStale`. Else no finding.
     - `RepairCmd` = `fmt.Sprintf("hero install project . --target %s", t)`.
   - **Read-only by construction:** this file must not import a write path. No
     `Writer.Write`, no `os.WriteFile`.

2. **`internal/cli/check.go`** — surface the findings.
   - In `runCheck`, after the `orphan-instruction-files` block (~line 518), call
     `install.CheckIntegrity(projectRoot)`.
   - Any `IntegrityDamaged` → `addRow("install-integrity", "fail", ...)` — this is the
     "agents are running cold" signal and belongs in the failing bucket, not the advisory
     one. `IntegrityStale` only → `addRow("install-integrity", "warn", ...)`.
   - Message names each affected target + file, missing sections when known, and the repair
     command. Mixed findings → one row at `fail`, message covering both.
   - Findings flow into the existing `healthJSONRow` output via `addRow`; no new JSON
     plumbing.

3. **`internal/install/integrity_test.go`** (new) — table-driven over **all six targets**
   per the tripwire, mirroring `TestHarness_InstalledContentSurvivesOrdinaryCommands`'s
   shape (`harness_smoke_test.go:330`):
   - `TestCheckIntegrity_CleanInstallIsSilent` — install, check, expect zero findings.
     (AC-3, AC-5)
   - `TestCheckIntegrity_DetectsGuttedRegion` — install, overwrite the region with the
     7-line pointer-only stub from the incident, expect `IntegrityDamaged` naming the
     missing sections. (AC-1)
   - `TestCheckIntegrity_DetectsStaleBody` — install, mutate one line inside the region,
     expect `IntegrityStale`. (AC-2)
   - `TestCheckIntegrity_SilentOnNeverInstalledTarget` — write an `AGENTS.md` with no
     install, expect zero findings. (AC-6)
   - `TestCheckIntegrity_IgnoresUserContentOutsideMarkers` — install, append and prepend
     user prose outside the markers, expect zero findings and the prose unmodified byte-
     for-byte. (AC-7)
   - `TestCheckIntegrity_IgnoresVersionStampDrift` — install, rewrite the marker's `v=` to
     a different version leaving the body intact, expect zero findings. (AC-8)
   - `TestCheckIntegrity_MissingRegion` — install, strip the markers and body entirely,
     expect `IntegrityDamaged`. (AC-10)
   - `TestCheckIntegrity_WritesNothing` — snapshot every file's mtime+bytes under the
     fixture, run `CheckIntegrity`, assert nothing changed. (AC-9)
   - `TestCheckIntegrity_FreshCloneWithoutInstallState` — install, delete
     `.hero/install-state.json` (simulating the gitignored file on a fresh clone), assert
     targets still resolve via `InferInstalledTargets` and a gutted region is still
     detected. Guards the union in design decision 5.

4. **`internal/install/harness_smoke_test.go`** — one integration assertion tying the guard
   to the advisory: extend or add alongside
   `TestHarness_InstalledContentSurvivesOrdinaryCommands` so that after an ordinary command
   runs, `CheckIntegrity` also reports clean. Ties "content survives" to "and we'd notice
   if it didn't."

5. **`docs/`** — document the `install-integrity` category wherever `hero check`'s
   categories are listed. Locate with `rg -l "satellite-drift|orphan-instruction-files" docs/`;
   if categories are not enumerated in docs today, skip this item rather than inventing a
   new doc surface.

## Boundaries

Bundling eight problems under one heading is precisely what made `codex-install-broken`
fail — its P0 hid among its P1s and every detection item got triaged away. This spec is
**one idea**: know what install should have produced, compare to disk, report the delta.

Explicitly **not** in scope:

- **MCP reachability in the Codex sandbox** (`codex-install-broken`'s Fix 5) — unrelated
  failure mode, needs research into Codex `setup_steps`, needs its own bug spec.
- **hero-code embedded install-on-open** — being handled in the hero-code repo.
- **Auto-repair from `hero check`** — design decision 2. Repair already works; check
  reports.
- **Missing agents/commands/skills directory advisories** — reasonable follow-up, separate
  spec. The instruction file is the proven failure.
- **Stale skill dir pruning** — already solved by `internal/install/prune.go`.
- **The `detectOrphanInstructionFiles` false positive on fresh clones** (design decision 5)
  — a real, pre-existing bug in an adjacent check. Do **not** fix it here; the new check
  must simply not copy the mistake. Worth its own small bug spec.
- **Changing the `v=` stamp's meaning or contents** — it records the writing binary's
  version. Leave it alone; this design does not need it.
- **Making `managed.Writer.Write` merge with on-disk sections** — rejected in
  `agents-md-erased-by-snapshot-pointer-writer`: section IDs are not emitted into rendered
  output and cannot be recovered from it. `TestWritePointerOnly_ReplacesEntireManagedRegion`
  pins the replace-not-merge semantics deliberately.

## Risks

- **False positives destroy the feature.** An advisory that fires on a healthy install gets
  ignored, and an ignored warning is indistinguishable from the two months of silence this
  spec exists to end. The two known false-positive sources are the `v=` stamp (design
  decision 1, AC-8) and never-installed targets (design decision 5, AC-6). Both have
  dedicated tests. If a third source surfaces during delivery, treat it as a stop-and-think
  moment, not a filter to bolt on.
- **Non-determinism in `RenderBody`.** The oracle assumes the body renders deterministically
  from embedded content. If any section injects a timestamp, absolute path, or map-iteration
  order, every healthy install reports stale. `snapshotPointerRelativePath(opts, filePath)`
  is path-derived and worth verifying explicitly during delivery. **Verify determinism
  first** — render twice in a test and compare. If a section is non-deterministic, that is
  a genuine finding to surface, not to paper over.
- **Pack-dependent body.** `loadPackAgentsMdBody` resolves through a chain
  (override → pack on disk → Go fallback). If the check's `Options` construction resolves a
  different link in that chain than the install did, it reports stale on a healthy install.
  Construct `Options` the same way the install path does; this is the highest-risk detail in
  the change.
- **Fresh clone with no `install-state.json`.** Gitignored (`.gitignore:69`). The union
  resolver handles it; `TestCheckIntegrity_FreshCloneWithoutInstallState` guards it.
- **Severity choice may be contentious.** `fail` for a gutted region is deliberate — it is
  the "every agent session starts cold" signal. If it proves noisy in practice, the fix is
  to tighten detection, not to demote the row to `warn`.

## Validation

**The real test — reproduce the original incident and prove the check catches it:**

1. Reintroduce the eraser (restore the `agentsPath` argument to `EnsurePointer` and point
   it at the root instruction file).
2. `hero install project . --target codex` — confirm `AGENTS.md` is ~232 lines.
3. `hero snapshot --project` — confirm it collapses to the 7-line stub.
4. `hero check` — **must report `install-integrity: fail`**, naming the missing sections and
   the repair command. This is the check the last two months did not have.
5. Repeat for `--target claude` / `CLAUDE.md` — proving the check is not accidentally
   AGENTS.md-only the way the eraser was accidentally AGENTS.md-only.
6. Revert the eraser; `hero check` must go silent again.

**Automated:** `go test -race -count=1 ./...` green. The nine `integrity_test.go` cases map
1:1 onto AC-1 through AC-10 and run table-driven across all six targets.

**False-positive validation (the discipline check):** on this repo, after a clean
`make install`, `hero check` must emit **zero** `install-integrity` rows — including with
uncommitted user prose above the markers in `CLAUDE.md`. Run it twice in a row; the second
run must also be silent (determinism).

**Repair round-trip:** with the check reporting `fail`, run the exact command from the
advisory message and confirm `hero check` goes silent. If the printed command does not
repair the finding, the advisory is lying — that is a bug, not a doc nit.
