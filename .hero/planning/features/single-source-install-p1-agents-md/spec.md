---
title: "Single-Source Install P1 — AGENTS.md as the Only Root Instruction File"
slug: single-source-install-p1-agents-md
type: feature
status: superseded
priority: P0
tags: [install, agents-md, claude-md, instructions, migration]
created: 2026-05-11
relations:
  - target: single-source-install
    kind: parent
  - target: harness-instruction-file-survey
    kind: motivated-by
horizon: now
superseded_by: harness-native-install-target-aware-upgrade
# superseded_reason: Reverses P1's AGENTS.md-as-only-root-file convergence. New direction is harness-native/target-aware: --target claude emits CLAUDE.md only, others AGENTS.md, upgrade respects previously-installed targets.
---

> # ⛔ SUPERSEDED — DO NOT IMPLEMENT
> This spec's core thesis ("**AGENTS.md as the only root instruction file**",
> write both files for `--target claude`) has been **reversed**. Hero's install
> model is now **harness-native, target-aware**: `--target claude` → CLAUDE.md
> only; other targets → AGENTS.md; upgrade respects previously-installed targets.
>
> **Authoritative spec:** [`harness-native-install-target-aware-upgrade`](../harness-native-install-target-aware-upgrade/spec.md).
> The managed-region *mechanics* below (versioned markers, preserve-user-content,
> idempotence, hand-edit refusal) remain correct and are reused by the new spec —
> only the per-target file mapping and upgrade behavior changed. Kept for genealogy
> (`hero why`); do not deliver as written.

## Goal

The project root has **one** instruction file: `AGENTS.md`. Hero owns
a clearly-delimited region inside it; the user owns everything else.
Every supported harness reads `AGENTS.md` either natively or via a
thin shim (symlink or `@import`). `hero install` and `hero upgrade`
manage the Hero region idempotently and never touch user content.

## Problem

Today, every Hero-installed project has at least:

- `AGENTS.md` (sometimes — user-authored project guide, varies by
  project)
- `CLAUDE.md` (Hero-managed stub today, ~30 lines, says "this project
  uses Hero")

…and depending on harnesses, may accumulate `.cursorrules`,
`.windsurfrules`, `.github/copilot-instructions.md`, and others, each
echoing similar boilerplate.

These files:
- Drift over time as Hero versions change.
- Confuse the user about which one to edit when adding project
  conventions.
- Cause harness sessions to see inconsistent context depending on
  which file the harness happens to read.
- Don't compose with user-authored project documentation — example codebase's
  28KB hand-written `AGENTS.md` is invisible to Hero today.

Per the harness survey (`.hero/knowledge/harness-instruction-file-survey.md`),
`AGENTS.md` is the cross-harness convergence point: native in ~30
harnesses including Codex, opencode, Cursor (Agent mode), Copilot,
Windsurf, Amp, Junie, Roo, OpenHands, goose, Gemini CLI, Zed, Warp,
and Factory. Claude Code is the conspicuous holdout (issue #6235 open
since Aug 2025), but Anthropic's official workaround is a symlink or
`@AGENTS.md` import inside `CLAUDE.md`.

## Design

### `AGENTS.md` structure

```markdown
# <project name> — AI Agent Development Guide

<!-- hero:managed-start v=0.7.1 -->

## Hero Workflow (managed by Hero — do not edit directly)

This project uses Hero for spec-driven AI engineering workflows.

- **Specs** live in `.hero/specs/` (active) and `.hero/planning/` (proposed).
- **Knowledge** lives in `.hero/knowledge/`.
- **Conventions** are enforced — see `.hero/conventions/` if present.
- **Session start:** run `hero resume` (or invoke the `resume` skill) to
  load focused context.
- **Before context fills:** run `hero handoff` to write a fresh `NEXT.md`.

### Available commands

- `/design` — produce a spec for a feature or change
- `/deliver` — implement an approved spec
- `/diagnose` — investigate a bug, produce a fix spec
- `/review` — review code, PRs, or architecture decisions
- `/handoff` — refresh NEXT.md
[…dynamically generated from installed commands]

### Active mission

<!-- pulled from .hero/mission.md if present -->

<!-- hero:managed-end -->

## <user-authored sections>

<whatever the user has — Hero never touches anything outside the
managed region.>
```

The managed region:

- Starts with `<!-- hero:managed-start v=<hero-version> -->` and ends
  with `<!-- hero:managed-end -->`. These exact strings are the lookup
  keys for the parser.
- Lives at the **top** of the file by default (after the H1 title if
  one exists) so the model sees Hero conventions first when context
  is truncated.
- If a file already exists with no managed region, Hero inserts the
  block at the top, leaving all existing content below.
- The version stamp lets `hero upgrade` detect when the managed region
  needs regenerating (different Hero version → rewrite the block;
  same version → no-op).

### Harness coverage strategy

The simpler model (post-dogfood revision): every harness instruction
file gets the same managed-block treatment. No symlinks, no `@import`
shims, no special-cased per-harness shim logic. Same body content
rendered into each file; user content outside the markers is
preserved byte-for-byte.

| Harness | File | Hero's action |
|---|---|---|
| Codex, opencode, Copilot, Windsurf, Amp, Junie, Roo, OpenHands, goose, Gemini CLI, Zed, Warp, Factory | `AGENTS.md` (root) | Inject the managed block (canonical body). |
| Claude Code | `CLAUDE.md` (root) | Inject the **same** managed block. CLAUDE.md and AGENTS.md are independent files with the same managed content — each is self-contained, each independently editable for user-content outside the markers. |
| Cursor (Manual mode) | `.cursor/rules/*.mdc` | out of scope for P1 — Cursor's Agent mode reads AGENTS.md, so P1 covers Agent mode use; Manual mode left for P2. |
| Aider | `CONVENTIONS.md` (configurable via `.aider.conf.yml`) | Future: managed block in `CONVENTIONS.md`. Not in P1. |
| Cline | `.clinerules` | Future: managed block. Not in P1. |

### Migration and the user-authored-file principle

**Principle (refined after dogfood feedback):** Hero never modifies
user-authored content. But "install Hero" must mean Hero actually
works — silently failing because the user happened to have a custom
CLAUDE.md sitting there is a worse outcome than touching the file at
all. So Hero may add a small, clearly-marked managed block to make
its content reachable, while keeping every byte of user content
intact.

Detection is deterministic and stateless: a file is "Hero-managed
only" when its *entire* content is inside Hero markers. Anything
else has user content, which is preserved verbatim — but Hero is
allowed (and expected) to add or update its own managed-region
footer so the harness reaches AGENTS.md.

**For `CLAUDE.md`, the three cases:**

1. **CLAUDE.md does not exist.**
   - If Claude Code is an installed target, create a `CLAUDE.md →
     AGENTS.md` symlink (or `@AGENTS.md` shim on Windows).
   - Otherwise, do nothing — there's no reason to create a CLAUDE.md
     just-in-case.

2. **CLAUDE.md exists, content is entirely a legacy Hero-managed
   stub** (recognized by `<!-- hero:managed -->` marker as the
   *only* meaningful content, ignoring whitespace and blank lines).
   - Delete it.
   - Replace with `CLAUDE.md → AGENTS.md` symlink (or `@AGENTS.md`
     shim on Windows).
   - Migration is safe; no user content was present.

3. **CLAUDE.md exists with any user-authored content** (no Hero
   marker, or Hero marker plus content outside it).
   - **User content is preserved byte-for-byte.** Hero never modifies
     anything outside its managed markers.
   - **Hero inserts (or updates in place) a managed block containing
     the same body as AGENTS.md.** The block uses versioned markers
     so it's idempotent and identifiable. Same managed-region
     pattern as AGENTS.md — same code path, same semantics.
   - First-time injection lands at the top of the file (after the H1
     if any). Subsequent regenerations replace the block in place
     wherever it currently sits.
   - One escape hatch:
     - `--no-touch-claude-md`: skip CLAUDE.md entirely. Niche; for
       users who want absolute file-immutability semantics on this
       specific file. Other harnesses still get the content via
       AGENTS.md.

The same logic applies to other harness-specific instruction files
(`.cursorrules`, `.windsurfrules`, `.github/copilot-instructions.md`,
`CONVENTIONS.md`):

- Doesn't exist + target installed → create shim/symlink to AGENTS.md.
- Exists as Hero-managed stub → migrate / replace with shim.
- Exists with user content → preserve user content + inject a managed
  import block (using whatever import syntax the harness supports —
  Cursor `@`, Aider `read:`, etc.).

This is stateless: Hero re-evaluates on every install/upgrade by
looking at the actual file content, not at remembered decisions.
A user who later wants Hero to take over their CLAUDE.md just has
to empty or delete it — Hero will then create the shim on next
install. A user who wants to fully opt out can pass
`--no-touch-claude-md`.

### `hero install` and `hero upgrade` behavior

**On `hero install`** (root or per-target):

1. Resolve install root. Detect existing `AGENTS.md` and `CLAUDE.md`.
2. If `AGENTS.md` doesn't exist, create it with the H1 title (from
   project name) and the managed region.
3. If `AGENTS.md` exists and has the managed region with the current
   Hero version, no-op on the file content.
4. If `AGENTS.md` exists and has a stale-version managed region,
   regenerate the managed region (preserving everything outside it).
5. If `AGENTS.md` exists and has no managed region, insert the
   managed region at the top.
6. For each installed harness target that needs a shim, create the
   symlink or `@import` file as appropriate for the host filesystem.

**On `hero upgrade`** (after Hero version bump):

- Same logic as `hero install`, but always regenerates the managed
  region (since version-bump implies new content). Idempotent if
  upgrade was already applied.

**Both commands:**

- Are non-destructive to user content outside the managed region.
- Are idempotent: re-running with no state change produces no
  filesystem changes.
- Print a summary: created N files, updated M files, no changes for
  K files.

### Detection of user edits inside the managed region

When `hero install` or `hero upgrade` runs and the managed region
contains content that doesn't match what the current Hero version
would generate (i.e., the user edited inside the markers), the
default behavior is:

1. Print a warning: `AGENTS.md managed region has been edited by hand;
   refusing to overwrite. Move your edits outside the markers, then
   re-run.`
2. Exit with non-zero status from install/upgrade.
3. `hero install --force-managed` (or `--reset-managed-region`)
   bypasses this — useful for CI or recovery.

`hero check` reports drift in the managed region as a warning so it's
visible during normal workflow.

### Configurable insertion position

A future enhancement (out of scope for P1 implementation but reserved
in the design): `.hero/config.json` could expose
`agentsMdManagedPosition: "top" | "bottom"` so users who want their
project context first can have it. Default is `top`. Not part of P1.

## Acceptance Criteria

- WHEN `hero install` runs in a project with no existing `AGENTS.md`
  THE SYSTEM SHALL create `AGENTS.md` containing the managed region
  at the top, with a project-titled H1 header
- WHEN `hero install` runs in a project with an existing user-authored
  `AGENTS.md` (no Hero marker) THE SYSTEM SHALL insert the managed
  region at the top of the existing file, preserving all existing
  content verbatim below
- WHEN `hero install` runs in a project with an `AGENTS.md` containing
  a managed region at the current Hero version THE SYSTEM SHALL make
  no changes to the file
- WHEN `hero upgrade` runs against an `AGENTS.md` with a stale-version
  managed region THE SYSTEM SHALL regenerate the managed region in
  place, preserving content outside the markers verbatim
- WHEN `hero install` runs in a project with a legacy Hero-managed
  `CLAUDE.md` (containing the existing `<!-- hero:managed -->` marker
  as its only meaningful content) THE SYSTEM SHALL delete the old
  `CLAUDE.md` and create a `CLAUDE.md → AGENTS.md` symlink (or
  `@AGENTS.md` shim on Windows)
- WHEN an existing `CLAUDE.md` contains user-authored content (no
  Hero marker, or marker plus content outside it) THE SYSTEM SHALL
  preserve every byte of user content verbatim AND insert (or
  update in place) a versioned Hero-managed block containing the
  same managed body that AGENTS.md gets
- THE SYSTEM SHALL never write content outside its own managed
  markers in any user-authored file (AGENTS.md, CLAUDE.md, or
  future harness instruction files)
- WHEN the Hero managed block already exists in CLAUDE.md THE
  SYSTEM SHALL replace it in place (wherever it currently sits)
  rather than always returning it to a fixed position
- THE SYSTEM SHALL use the same managed-region implementation for
  AGENTS.md and CLAUDE.md, sharing the body generator and
  managed-region semantics (no per-file special-casing of marker
  format, regeneration logic, or hand-edit detection)
- WHEN a user explicitly invokes `hero install --no-touch-claude-md`
  THE SYSTEM SHALL skip CLAUDE.md handling entirely, leaving any
  existing file byte-identical and creating no new file (Claude
  Code won't see Hero content via CLAUDE.md but AGENTS.md is still
  written normally)
- WHEN the host filesystem does not support symlinks (Windows without
  Developer Mode) THE SYSTEM SHALL write `CLAUDE.md` as a one-line
  file containing `@AGENTS.md` instead of a symlink
- WHEN a user has edited content inside the managed region of
  `AGENTS.md` THE SYSTEM SHALL refuse to regenerate the region without
  `--force-managed`, print a warning identifying the edited region,
  and exit non-zero
- WHEN `hero check` runs and the managed region has been hand-edited
  THE SYSTEM SHALL surface this as a warning signal in the check
  report
- THE SYSTEM SHALL be idempotent: running `hero install` twice in
  succession against the same state produces zero filesystem changes
  on the second run

## Changes

- `internal/install/agents_md.go` (new) — managed-region parser,
  renderer, insert/upgrade logic, migration from CLAUDE.md
- `internal/install/agents_md_test.go` (new) — table-driven tests
  covering: no file, user-authored no marker, managed at current
  version, managed at stale version, hand-edited managed region,
  legacy CLAUDE.md migration, user-content-in-CLAUDE.md refusal,
  Windows shim path
- `internal/install/install.go` — wire AGENTS.md generation into
  install flow; remove (or deprecate) `installClaudeMd`; route
  CLAUDE.md shim creation through new path
- `internal/install/satellite.go` — update `targetLayouts` so the
  Codex/OpenCode/Generic group's `MarkerFile` is `AGENTS.md`
  (already true) and Claude's is now also `AGENTS.md` with a
  shim policy
- `internal/cli/upgrade.go` — wire managed-region regeneration into
  upgrade flow
- `internal/cli/check.go` — add a check signal for hand-edited
  managed regions
- `docs/cli/*.md` — document the new model, CLAUDE.md → AGENTS.md
  migration story
- `cmd/hero/...` — bump help text for `install`, `upgrade`,
  `check` where their behavior changes

## Boundaries

- **Not in scope:** moving agent/command/skill content into `.hero/`
  — that's Phase 2.
- **Not in scope:** rendered-copy fallback for harnesses that need
  it — that's Phase 2.
- **Not in scope:** the `--migrate` command for legacy messy installs
  with drifted copies — that's Phase 3.
- **Not in scope:** `hero verify-install` drift detection for
  rendered copies — that's Phase 4.
- **Not in scope:** any automated movement of user content from an
  existing non-Hero CLAUDE.md (or any other user-authored harness
  file) into AGENTS.md. User-authored harness instruction files are
  preserved permanently; coexistence with AGENTS.md is the supported
  configuration. The `--replace-claude-md` escape hatch is the only
  way to opt back into Hero-managed CLAUDE.md.
- **Not in scope:** Cursor `.cursor/rules/*.mdc` handling — Cursor's
  Agent mode reads AGENTS.md, so P1 is sufficient for Agent mode
  users; Manual mode users wait for P2.
- **Not in scope:** changing the *content* Hero generates into the
  managed region — this is a packaging change, not a content
  redesign. The current Hero-managed boilerplate plus a few
  dynamically-generated bits (command list, mission) is sufficient
  for P1.

## Mission Fit

> "Does this make the next agent session start smarter than the
> last one ended — and does it raise the floor for everyone?"

Yes. Today, the model sees inconsistent project context depending on
which harness is in the loop (CLAUDE.md vs AGENTS.md vs others, with
different generation timestamps). After P1, every harness sees the
same instruction file, with the same managed Hero context, and the
same user content. The floor rises for anyone working in a project
with more than one harness installed (which is the increasingly common
case for teams), and the maintenance cost drops to "edit AGENTS.md"
for everyone.
