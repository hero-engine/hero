---
title: "hero doctor: installed harness target table with agent/command/skill counts"
slug: doctor-install-target-table
type: feature
status: delivering
size: small
domain: engineering
priority: medium
created: 2026-07-16
tags: [doctor, install, harness-targets, cli, diagnostics]
relates-to:
  - hero-docs-check-engine-repo-misfire
  - install-json-mode-repair-migrate-parity
delivery_method: manual
---

# hero doctor: installed harness target table with agent/command/skill counts

## Context

`hero doctor` today answers exactly one question: *which hero binary is
running, and does its schema agree with the workspace graph?* It renders four
sections — `Running binary`, `PATH resolution`, `Workspace graph`, `Verdict:`
(`internal/cli/doctor.go:98-145`). CLAUDE.md leans on it hard as the
binary/PATH/schema triage tool.

It cannot answer the adjacent question a user asks just as often: **"did my
install actually land?"** After `hero install`, the only way to confirm is to
`ls` harness directories by hand and know, per target, what should be there —
which almost nobody knows, because per-target layout genuinely differs.

User's framing:

> "hero doctor should probably give a brief table of installed harness targets
> it sees - and the counts of agents, commands and skills - so you can make sure
> its installed properly"

This is an **addition**. The four existing sections and the `Verdict:` line stay
intact and byte-identical.

### Why this is more than a file count

Hero installs to **six** targets (`opencode | cursor | claude | copilot | codex
| generic`), and each materializes the same canonical content into a different
shape. A naive `len(ls .claude/agents)` table would be wrong for four of the six.
The per-target truth, verified against the install code this session:

| Target | agents dest | commands dest | skills dest | root file |
|---|---|---|---|---|
| `claude` | `.claude/agents/*.md` | `.claude/commands/*.md` | `.claude/skills/<n>/SKILL.md` | `CLAUDE.md` |
| `opencode` | `.opencode/agents/*.md` | `.opencode/commands/*.md` | `.opencode/skills/<n>/SKILL.md` | `AGENTS.md` |
| `cursor` | `.cursor/rules/agents/*.md` | `.cursor/rules/commands/*.md` | `.cursor/rules/skills/<n>.md` **(flat)** | `AGENTS.md` |
| `codex` | `.codex/agents/*.toml` **(TOML)** | **no loader** — installed as skills | `.agents/skills/<n>/SKILL.md` **+** `.agents/skills/command-<n>/SKILL.md` | `AGENTS.md` |
| `copilot` | `.github/prompts/agents/*.prompt.md` | `.github/prompts/commands/*.prompt.md` | `.github/skills/<n>/SKILL.md` | `AGENTS.md` |
| `generic` | `.ai/agents/*.md` | `.ai/commands/*.md` | `.ai/skills/<n>/SKILL.md` | `AGENTS.md` |

Sources: `target_claude.go:30-39`, `target_opencode.go:33-42`,
`target_cursor.go:27-33,57`, `target_codex.go:76-114,168-180`,
`target_copilot.go:70-91`, `target_generic.go:25-37`; root-file mapping
`agents_md.go:66` (`nativeInstructionFile`).

### Active tripwire — `harness-changes-cover-all-targets` [high]

This feature sits squarely inside the tripwire. It is harness-facing and the
recurring failure mode it exists to prevent is *"handles Claude, silently broken
for the other five."* A table that hardcodes `.claude/` or treats codex as
"0 commands" is exactly that failure.

**How this design satisfies it:**

1. **No parallel path map.** The per-target dest map is derived from the same
   `internal/install` code that performs the install (see `## Approach`), not
   re-typed in `internal/cli`. Drift is impossible by construction — the same
   discipline `hero-docs-check-engine-repo-misfire` established with
   `EnumerateContent`.
2. **All six targets are first-class**, each with its own expected-count
   derivation. Codex and Cursor are not special cases bolted on; they are cells
   in one table.
3. **The test is a six-row table test** mirroring
   `internal/install/contracts_test.go:22-55`, so a seventh target (or a changed
   layout) fails loudly rather than silently under-reporting.
4. **Codex's no-commands quirk renders as `—`, never `0`** — see `## Approach`.

## Goal

`hero doctor` gains a fifth section, `Installed harness targets`, rendered
between `Workspace graph` and `Verdict:`. It lists one row per installed target
with **expected-vs-actual** counts of agents / commands / skills, plus the
target's root instruction file. Targets with a shortfall are flagged in-section
with an actionable remediation. All six targets are handled correctly, including
codex's absent command loader (renders `—`, not `0`) and its commands-as-skills
rollup. The existing four sections and the `Verdict:` line are unchanged. Done
when a user can run `hero doctor` after `hero install` and confirm at a glance
that every installed target is complete.

## Kickoff

Adds a brief "Installed harness targets" table to `hero doctor` — per-target
expected/actual counts of agents, commands, and skills — so you can confirm an
install landed and spot a partial one.

**Status:** planning — spec just landed, no code yet.

**Pick up at:** write `internal/install/inventory.go` — an exported
`Inventory(projectRoot, domain)` that returns one `TargetInventory` per target in
`UnionTargets(PreviouslyInstalledTargets, detected)`. Derive expected counts
per-target from `EnumerateContent` (codex skills = skills+commands; codex
commands = n/a). Then render it in `doctor.go` before the `Verdict:` line.

→ `.hero/planning/features/doctor-install-target-table/spec.md`

**Files:** `internal/install/manifest.go:37`, `internal/install/contracts.go:69`,
`internal/cli/doctor.go:98`, `internal/cli/docs_check.go:285`
**Skip:** `install.DetectInstalledTargets` (probes legacy `.github/copilot/`, cannot
see copilot — `integrity_test.go:397-407`); a `--json` flag (out of scope, see Boundaries).

## Approach

### 1. Expected counts come from `EnumerateContent`, derived per-target

**Do not reinvent the canonical set.** `hero-docs-check-engine-repo-misfire`
already extracted install's content selection into
`internal/install/manifest.go` — `EnumerateContent(contentFS, domain)
(ContentManifest, error)` (`manifest.go:37`) returns the deduped
`{Agents, Commands, Skills}` name lists that install actually materializes,
routed through the same `selectFlatContent`/`selectSkillContent` selectors the
install functions use. That is the expected-count source. Reuse it verbatim.

Build the FS exactly as `docs_check.go:285-292` does:

```go
domainFS, err := hero.DomainFS(domain)
manifest, err := install.EnumerateContent(hero.OverlayFS(domainFS, hero.CoreFS()), domain)
```

**Counts are domain-dependent.** Resolve the active domain with
`activeDomainForRoot(projectRoot)` (`docs_check.go:275-283`) — already in
`package cli`, directly callable from `doctor.go`. It reads `cfg.Domain` and
falls back to `engineering`.

**The critical part — expected is per-target-derived, not the raw manifest.**
Mapping the raw manifest onto every target would falsely report codex as short
by 29 skills. The derivation:

| Target | expected agents | expected commands | expected skills |
|---|---|---|---|
| `claude`, `opencode`, `cursor`, `copilot`, `generic` | `len(Agents)` | `len(Commands)` | `len(Skills)` |
| `codex` | `len(Agents)` | **n/a** (no loader) | `len(Skills) + len(Commands)` |

Codex's rollup is not a guess — `codexSkillDirNames` (`target_codex.go:145-163`)
is literally `canonicalSkillDirNames(opts)` plus one `codexCommandSkillDir(name)`
per command. The inventory must mirror that function's arithmetic, and the test
asserts the two agree.

### 2. Detection: union of persisted and on-disk, using the real dest paths

**Do not use `install.DetectInstalledTargets`** (`satellite.go:80`). It walks
`targetLayouts`, which probes `.github/copilot/` — a **legacy location the
modern copilot install deliberately no longer creates** and in fact actively
deletes (`target_copilot.go:57-67`). `integrity_test.go:397-407` documents this
as a known probe gap: copilot **cannot** be inferred from disk through that
registry. Building the table on it would make copilot permanently invisible —
the tripwire's exact failure mode, shipped.

Instead, detection derives from the **same per-target dest map used for
counting**, so detection and counting cannot disagree:

- A target is **detected** if any of its real dest dirs exists (for copilot,
  also `.github/copilot-instructions.md`, the file marker —
  cf. `auto_sync.go:74,83-87`, which is the closer-to-correct existing probe but
  is unexported and only `Lstat`s the base dir).
- Union with `PreviouslyInstalledTargets(projectRoot)` (`state.go:224`) via the
  existing `UnionTargets(...)` helper (`integrity.go:63`).

**The union is load-bearing, in both directions:**

- `.hero/install-state.json` is **gitignored** (verified: `.gitignore:69`), so on
  a fresh clone `PreviouslyInstalledTargets` returns nil and on-disk detection
  must carry the whole result.
- Conversely, a target present in `install-state.json` but with an empty/missing
  dest tree is **precisely the broken install the user wants to spot**. It must
  render as a row with `0/35` and a warning — not vanish from the table.

### 3. Rendering — brief, and honest about codex

Only union-resolved targets get rows. Everything else collapses to a single
`not installed:` line, which keeps the section to ~6 lines in the common case.

**Codex's commands cell renders `—`, never `0`.** `0/29` would read as a broken
install and send a user chasing a non-bug; the truth is Codex has no command
loader at any scope (`SlashCommand` is a built-in enum —
`target_codex.go:23-26`) and Hero installs commands as skills instead. `—` plus
one legend line is the honest rendering. A footnote is emitted **only when codex
is in the table**, so the section stays brief for everyone else.

### 4. Verdict integration — an in-section warning, not a changed `Verdict:` line

**Position: the install table does NOT feed the `Verdict:` line.** It carries
its own in-section `WARNING:` block.

The argument is precedent, not preference: the `PATH resolution` section already
does exactly this. A PATH divergence — arguably doctor's single most consequential
finding — emits an in-section `WARNING:` (`doctor.go:119-123`) and leaves
`Verdict:` untouched, because `Verdict:` answers one specific question: *do the
binary and graph schemas agree, and what is the true remediation?*
(`doctorVerdict`, `doctor.go:149-170`). Its two failure branches give
schema-specific advice ("wrong hero binary on PATH", "`hero upgrade` will NOT
help") that would be nonsense for a missing skills dir. Overloading it would
dilute the line CLAUDE.md trains agents to act on, and would churn every
`Verdict:` assertion in `doctor_test.go`.

A partial install is still loud: the row is marked with `!`, and a `WARNING:`
block recommends **`hero upgrade`** to re-materialize the missing content. This
matches the house pattern for "important but not the schema question."

**The shortfall fix is `hero upgrade`, not `hero install`.** Verified against the
CLI: `hero upgrade` "updates agents, commands, and skills to match the installed
hero binary version" over the previously-installed target set
(`install-state.json` ∪ filesystem probe) — exactly the installed-but-short case.
`hero install --repair` is wrong here (it reconciles satellite symlinks/markers,
not content). `hero install --target <t>` is for adding a target that is not
installed at all — that belongs to the **informational** `not installed:` line,
never to a shortfall.

**A shortfall applies only to installed targets.** A target on the `not
installed:` line is a *choice*, not a broken install — it gets no `!`, no
`WARNING:`, and no upgrade nudge. Only a target that is installed yet reports
`actual < expected` for a numeric, applicable kind is a shortfall. (Codex's
`NotApplicable` commands cell is never a shortfall — `—` is not `< expected`.)

### 5. Engine repo — no special case, deliberately

**Position: doctor needs no `isEngineSourceRepo` branch, and adding one would be
a mistake.**

`hero docs check` misfired in this repo because it counted an *installed harness
tree* in a repo whose content authoring source is `core/` + `domains/*` — it was
asking the wrong question of the wrong tree, so `actual: 0` was meaningless.
Doctor is not in that position: it counts installed harness trees, and the engine
repo genuinely **has** them. Verified on disk this session: `.claude/`, `.codex/`,
and `.agents/` are present; `.cursor/`, `.opencode/`, `.ai/`, and
`.github/copilot-instructions.md` are not. So doctor here truthfully reports
claude + codex installed, four targets not installed. That is a correct answer,
not a false alarm.

The gitignored-content nuance (`.gitignore:70-80` ignores `.claude/agents`,
`.claude/commands`, `.claude/skills`, and peers) means a **fresh clone of the
engine repo** has no harness content until someone runs `hero install`. Doctor
then reports "no harness targets installed" — correct and actionable. The empty
state must therefore read as neutral guidance, not an error (see AC-9).

One genuine edge, worth naming because it is doctor's own subject matter: expected
counts come from the **running binary's embedded FS**, so a stale binary in the
engine repo could report expected counts that disagree with local `core/` +
`domains/` source. That composes correctly rather than confusing — the
`PATH resolution` warning directly above the table is the explanation. No extra
handling needed.

## Rendered example

Healthy engine-repo install (engineering domain: 35 agents / 29 commands /
55 skills — the canonical counts `hero-docs-check-engine-repo-misfire` derived):

```text
hero doctor

Running binary
  os.Executable(): /Users/you/go/bin/hero
  version:         v0.25.0
  binary schema:   4

PATH resolution
  `hero` on PATH:  /Users/you/go/bin/hero

Workspace graph
  workspace:       /Users/you/projects/hero/.hero
  graph schema:    4

Installed harness targets
  TARGET     AGENTS   COMMANDS   SKILLS   ROOT FILE
  claude      35/35      29/29    55/55   CLAUDE.md
  codex       35/35          —    84/84   AGENTS.md
  not installed: copilot, cursor, generic, opencode

  codex has no command loader — its 29 commands install as skills under
  .agents/skills/command-<name>/ (55 canonical + 29 commands = 84).

Verdict: OK — binary and graph agree on schema 4.
```

Partial / broken install:

```text
Installed harness targets
  TARGET     AGENTS   COMMANDS   SKILLS   ROOT FILE
  claude      35/35      12/29 !  55/55   CLAUDE.md
  codex        0/35          —     0/84 ! AGENTS.md
  not installed: copilot, cursor, generic, opencode

  WARNING: 2 installed targets are incomplete (marked !) — content is
           missing. Run `hero upgrade` to re-materialize the missing
           agents, commands, and skills.

  codex has no command loader — its 29 commands install as skills under
  .agents/skills/command-<name>/ (55 canonical + 29 commands = 84).
```

The `codex 0/35 ... 0/84` row is the union doing its job: codex is in
`install-state.json` but its tree is gone, so it renders as a flagged row rather
than silently disappearing. Note the fix is `hero upgrade`, not `hero install
--target` — the targets *are* installed, they are just short on content, which is
exactly what `hero upgrade` reconciles. The `not installed:` line below is
untouched: those four targets get no `!` and no warning, because a never-installed
target is a choice, not a broken install.

Nothing installed:

```text
Installed harness targets
  no harness targets installed — run `hero install --target <claude|codex|copilot|cursor|opencode|generic>`
```

## Acceptance Criteria

- **AC-1:** WHEN `hero doctor` runs in a workspace with at least one installed harness target THE SYSTEM SHALL render an `Installed harness targets` section between `Workspace graph` and `Verdict:`, with one row per installed target showing expected-vs-actual counts for agents, commands, and skills plus the target's root instruction file.
- **AC-2:** THE SYSTEM SHALL derive expected counts from `install.EnumerateContent` over the active domain's overlay FS, so the table's expected values equal what `hero install` materializes.
- **AC-3:** WHEN the active domain is resolved THE SYSTEM SHALL use `activeDomainForRoot`, falling back to `engineering` when `hero.json` declares no `domain`.
- **AC-4:** THE SYSTEM SHALL render a correct row for each of the six targets — `claude`, `opencode`, `cursor`, `copilot`, `codex`, `generic` — counting each target's real destination paths (`.claude/skills/<n>/SKILL.md`, `.cursor/rules/skills/<n>.md` flat, `.codex/agents/*.toml`, `.github/prompts/{agents,commands}/*.prompt.md` and `.github/skills/`, `.agents/skills/`, `.ai/`) rather than assuming Claude's layout.
- **AC-5:** WHERE the target is `codex` THE SYSTEM SHALL render the commands cell as `—` and never as a numeric `0`, and SHALL compute expected skills as `len(Skills) + len(Commands)` to account for commands installed as skills under `.agents/skills/command-<name>/`.
- **AC-6:** WHEN `codex` appears in the table THE SYSTEM SHALL emit a one-line footnote explaining that codex has no command loader and its commands install as skills; and WHEN codex is absent THE SYSTEM SHALL omit that footnote.
- **AC-7:** WHERE the target is `copilot` THE SYSTEM SHALL detect the install via `.github/copilot-instructions.md` (a regular file) or its `.github/prompts/` / `.github/skills/` destination dirs, and SHALL NOT rely on `install.DetectInstalledTargets`, whose `targetLayouts` probes the legacy `.github/copilot/` directory.
- **AC-8:** THE SYSTEM SHALL resolve the row set as the union of `install.PreviouslyInstalledTargets` and on-disk detection, so that a fresh clone with a gitignored `install-state.json` still lists targets, and a target recorded in `install-state.json` whose tree is missing renders as a flagged `0/N` row rather than being omitted.
- **AC-9:** IF no harness target is installed THEN THE SYSTEM SHALL render a single neutral line naming `hero install --target <name>` as the remedy, and SHALL NOT render a warning or alter the `Verdict:` line.
- **AC-10:** WHEN an installed target's actual count is below its expected count for a numeric, applicable kind THE SYSTEM SHALL mark the affected row with `!` and emit an in-section `WARNING:` that recommends running `hero upgrade` to re-materialize the missing agents, commands, and skills.
- **AC-11:** IF a target is not installed (appears on the `not installed:` line) THEN THE SYSTEM SHALL NOT mark it, emit a shortfall `WARNING:`, or recommend `hero upgrade` for it — a never-installed target is informational only.
- **AC-12:** THE SYSTEM SHALL leave the `Verdict:` line's text and logic unchanged, so that an incomplete install does not alter the schema verdict.
- **AC-13:** WHEN `hero doctor` runs outside a workspace or where no graph exists THE SYSTEM SHALL preserve its existing early-return behavior and the corresponding `Verdict: cannot compare` output.
- **AC-14:** THE SYSTEM SHALL keep the section brief — targets that are not installed collapse to a single `not installed:` line, and no per-file names are listed.

## Changes

1. **New `internal/install/inventory.go`** — the per-target dest map, detection, and counting. This is the whole correctness surface; it lives in `internal/install` (not `internal/cli`) so it sits next to the install code it must not drift from, matching where `ContentManifest` lives.
   - Exported `type TargetInventory struct { Target Target; RootFile string; Agents, Commands, Skills KindCount }`.
   - `type KindCount struct { Expected, Actual int; NotApplicable bool }` — `NotApplicable` is how codex/commands renders `—` instead of `0`. Do not model it as `Expected: 0`.
   - `func Inventory(projectRoot, domain string) ([]TargetInventory, error)` — resolves the row set via `UnionTargets(PreviouslyInstalledTargets(projectRoot), detected)`, then fills counts per target.
   - An internal `targetInstallPaths(t Target, projectRoot string)` returning the per-kind dest dir + how to count it (flat `*.md` / `*.toml` / `*.prompt.md` files vs. nested `<n>/SKILL.md` dirs). Derive these from the existing target files; do not re-type them from this spec's table without checking the source.
   - Expected derivation per `## Approach` §1. Codex skills expected must mirror `codexSkillDirNames` (`target_codex.go:145-163`).
   - `RootFile` from `nativeInstructionFile(t)` (`agents_md.go:66`) — do not hardcode `CLAUDE.md`/`AGENTS.md` strings.
   - Counting must exclude non-content files (mirror the `isContentReadme` skip used by `selectFlatContent`, `manifest.go:84`).

2. **New `internal/install/inventory_test.go`** — six-row table test mirroring the `cells` pattern at `internal/install/contracts_test.go:22-55`.
   - One case per target asserting dest paths, expected derivation, and counting shape against a harness install (`newInstallHarness`).
   - `TestInventory_CodexCommandsNotApplicable` — codex commands is `NotApplicable`, never `Expected: 0`.
   - `TestInventory_CodexSkillsMatchInstalledDirs` — the pivotal guard: inventory's expected codex skills equals `len(codexSkillDirNames(opts))`, so the rollup can't drift from the installer.
   - `TestInventory_CopilotDetectedFromInstructionsFile` — copilot resolves from a real `runCopilot` install (the regression guard for the `DetectInstalledTargets` trap; note the existing skip at `integrity_test.go:397-407` and do **not** replicate it here).
   - `TestInventory_UnionSurvivesMissingInstallState` — remove `install-state.json`, assert rows still resolve (mirrors `TestCheckIntegrity_FreshCloneWithoutInstallState`, `integrity_test.go:394`).
   - `TestInventory_PersistedTargetWithMissingTreeIsZero` — the other union direction: persisted target, deleted tree, expect a `0/N` row.

3. **`internal/cli/doctor.go`** — gather + render.
   - Extend `doctorInfo` (line 42) with `inventory []install.TargetInventory` and `inventoryErr string`. Keep the struct's "constructed directly in tests" property (line 40-41) — the renderer stays pure.
   - In `runDoctor` (line 54), after the graph block (~line 90), populate the inventory from `projectRoot` + `activeDomainForRoot(projectRoot)`. An inventory error must **not** fail doctor — record it and render a one-line note. Doctor is triage; it must still report binary/PATH/schema when install introspection fails.
   - New `buildInventorySection(info) string`, called from `buildDoctorReport` (line 98) **after** the `Workspace graph` block and **before** `doctorVerdict` (line 143).
   - Note the two early returns at lines 129-137 (no workspace / no graph) return before the verdict; the section is skipped on those paths per AC-13.
   - The shortfall `WARNING:` string must recommend `hero upgrade` (not `hero install`) per AC-10, and must fire only for installed targets with a numeric-kind shortfall — never for a `not installed:` target (AC-11).
   - Column widths sized to the longest target name (`opencode`, 8 chars); align with the existing two-space-indent, aligned-label style of the other sections.

4. **`internal/cli/doctor_test.go`** — extend `TestBuildDoctorReport` (line 8) with subtests, following the existing `base := doctorInfo{...}` + `buildDoctorReport(info)` style.
   - Healthy table renders expected/actual per target; codex commands renders `—`; codex footnote present only when codex is present; no `WARNING:` on a fully healthy install.
   - `shortfall_recommends_hero_upgrade` — an installed target with `actual < expected` marks the row with `!` and the `WARNING:` string contains `hero upgrade` and NOT `hero install` (AC-10).
   - `not_installed_target_no_warning` — a target on the `not installed:` line produces no `!` and no shortfall `WARNING:`, even when other targets are healthy; asserts the upgrade nudge is absent for it (AC-11).
   - `verdict_unchanged_under_shortfall` — `Verdict:` text is byte-identical with and without a shortfall (AC-12).
   - Empty state renders the neutral line; no-workspace/no-graph paths unchanged (AC-13).

5. **`web/docs/src/cli/overview.md`** — docs ship in the same PR.
   - Expand the `## Troubleshooting` section (lines 56-67), which is the **only** place doctor is documented; there is no dedicated doctor page.
   - Add a ` ```text ` sample-output block showing all five sections, and name what each reports. This establishes the first sample-CLI-output block in `web/docs/src/` — ` ```text ` is the consistent choice (`bash` fences are for commands).
   - Document the codex `—` convention explicitly; it is the cell most likely to be misread as a bug.
   - Edit `web/docs/src/` **only** — `web/docs/site/` is mkdocs build output and gitignored (`.gitignore:41-42`).
   - Beware `internal/cli/markdown_drift_test.go:72-75`: it regexes `hero <command>` across the **whole file**, not fence-aware. `hero doctor` resolves fine and the indented count lines won't match, but re-run the test. Escape hatch if needed: `<!-- drift-test:ignore -->` (`markdown_invocations.go:86,105`).
   - Consider adding `doctor` to the `## Workspace Utilities` table (lines 50-54), where it is currently absent.

## Boundaries

- **No `--json` flag.** Doctor has none today, so there is no contract to keep parity with, and the honest scope of "doctor gets JSON" is a **whole-doctor envelope** (binary + PATH + graph + install + verdict), not a table-only one. Minting a JSON contract that covers one of five sections would guarantee a reshape later — exactly the stdout-contract break class that `install-json-mode-repair-migrate-parity` punished three times. Deferred to its own spec (see `## Risks`). Do not add a partial `--json` here.
- **No `DetectLegacyDrift` surfacing.** `cleanup.go:235` answers a different question — "are there dead bytes from an older layout?" — not "did my install land?" It has its own remediation (`hero install` cleans up as a side effect), and folding it in would bloat the section past "brief". Natural follow-up, not this spec.
- **No content validation.** The table counts files; it does not check frontmatter, required fields, or format. That is what `HarnessContract` + `ContractsFor` (`contracts.go:185`) exist for, and wiring a contract validator into doctor is a separate, much larger feature.
- **No global-mode (`~/.claude`, `~/.codex`) introspection.** The table covers the project workspace only. Global installs are a real surface but a distinct question with distinct paths.
- **No changes to `install.DetectInstalledTargets` or `targetLayouts`.** The legacy `.github/copilot/` probe gap (`integrity_test.go:397-407`) is real and affects satellites and `hero upgrade` — but fixing the shared probe registry has blast radius well beyond doctor. This spec routes around it with its own detection and leaves a follow-up.
- **No `Verdict:` line changes.** Per `## Approach` §4.
- **No new `hero install` behavior.** Read-only introspection.

## Risks

- **Expected-count derivation drifting from install.** The single highest risk: if codex's skills rollup is hand-computed and `target_codex.go` later changes, doctor silently reports a false shortfall. Mitigation: `TestInventory_CodexSkillsMatchInstalledDirs` asserts against `codexSkillDirNames` directly rather than a literal.
- **Copilot invisibility.** If the engineer reaches for the obvious-looking `install.DetectInstalledTargets`, copilot silently never appears and the tripwire failure ships looking green. Mitigation: AC-7 plus a copilot detection test built on a real `runCopilot` install.
- **Codex commands rendered as `0`.** Would send users chasing a non-bug. Mitigation: `NotApplicable` modeled in the type (not as `Expected: 0`), plus AC-5 and a dedicated test.
- **Cursor's flat skills.** `installSkillsFlat` writes `<name>.md`, not `<name>/SKILL.md` (`target_cursor.go:33`, `contracts.go:118`). Counting cursor skills as nested dirs yields `0/55` — a false broken-install report for every Cursor user.
- **Section-order regression.** The two early returns at `doctor.go:129-137` return before the verdict. Inserting the section carelessly could either skip it on healthy runs or emit it on no-workspace runs.
- **Stale-binary expected counts** in the engine repo — accepted and explained by the adjacent PATH warning; see `## Approach` §5.
- **Follow-ups this spec deliberately leaves open** (each worth its own item): whole-doctor `--json` using the envelope pattern (`sync_push.go:62-71`, `spec_set_owner.go:79`); the `targetLayouts` copilot probe gap; and promoting the "every `--json` return path emits exactly one object" rule to `.hero/knowledge/conventions/` — an Explore pass confirmed it exists only in spec prose and code comments despite recurring three times.

## Validation

- `go build ./cmd/hero/` and `go vet ./internal/...` clean.
- `go test ./internal/install/... ./internal/cli/...` green — the two packages this touches.
- **Six-target coverage is the gate.** `inventory_test.go` must have a passing case for every one of `claude`, `opencode`, `cursor`, `copilot`, `codex`, `generic`. A `t.Skip` on any target is a failed delivery, not a green one — the tripwire's failure mode is precisely a target quietly going untested.
- Manual, in this repo: `hero doctor` shows `claude` and `codex` rows with `codex` commands as `—`, codex skills `84/84`, and `not installed: copilot, cursor, generic, opencode`. Confirms the engine repo produces no false "broken install" (`## Approach` §5).
- Manual, shortfall: `rm -rf .claude/commands && hero doctor` → claude commands row flagged with `!`, `WARNING:` emitted recommending `hero upgrade` (not `hero install`), `Verdict:` still `OK — binary and graph agree on schema 4` (proves AC-10 + AC-12).
- Manual, not-installed silence: with `.cursor/` absent, cursor appears only on the `not installed:` line — no `!`, no `WARNING:`, no upgrade nudge (proves AC-11).
- Manual, union: `rm .hero/install-state.json && hero doctor` → rows still resolve from disk (proves AC-8's fresh-clone direction).
- `go test ./internal/cli/ -run TestMarkdownDrift` green after the docs edit.
- `hero docs check` still green — this spec changes no agent/command/skill counts, but the docs edit touches a file it scans.
