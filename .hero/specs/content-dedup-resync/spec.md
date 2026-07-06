---
title: Content Dedup & Resync — Single-Master core/engineering Content with CI Parity Gate
slug: content-dedup-resync
type: bug
status: completed
priority: P1
size: medium
domain: engineering
tags: [content, dedup, drift, core, engineering, overlay, ci]
created: 2026-07-05
relations:
  - target: hero-content-audit
    kind: follows
  - target: core-vertical-layering
    kind: related
mission_alignment: |
  Drifted duplicate content ships stale — and in two cases actively
  wrong — guidance into every install: the next session starts dumber
  than the content's best version. Single-mastering the duplicated
  files and gating parity in CI makes every future content fix land
  once, for all domains and all six harness targets.
completed_at: 2026-07-06T18:42:26Z
---

# Content Dedup & Resync — Single-Master core/engineering Content with CI Parity Gate

## Context

The hero-content-audit (completed 2026-07-05, report at
`.hero/specs/hero-content-audit/audit-report.md`, theme T1) found **34
same-named file pairs** duplicated between `core/` and
`domains/engineering/`: 13 skills, 17 commands, 4 agents.

- **20 pairs are byte-identical** (8 skills, 9 commands, 3 agents). Under
  `OverlayFS(domain, core)` (internal/cli/install.go:188-200, domain wins),
  the engineering copies are pure dead weight — deleting them changes
  nothing because the overlay falls through to the identical core file.
- **14 pairs have forked**, all classified **accidental drift** (fork
  origin: commit `92c94aa`; every divergence is one-sided). Because each
  copy is live for a different audience, stale content ships in both
  directions:
  - Engineering installs (the default) get `spec-format` and
    `context-injection` copies that never received `d7fd9e9` — they still
    teach the deprecated `status: superseded` hand-edit and never see
    superseded-spec handling.
  - pm/sales installs get core copies frozen at v0.8.0: `import.md`
    instructs `hero import --preset/--jql` (a live bug — those flags belong
    to `hero sync import`), `session-primer` uses nonexistent `hero status
    --delivering/--claimed` flags, and `agent-reliability` / `next-md` /
    `next-handoff-emit` lack two generations of universal rules (including
    "never hand-edit `.hero/next/<user>.local.md` — the checkpoint wipes it").

There is no sync mechanism and no CI check. Full per-pair evidence:
`findings-skills.md §(b)` (skills matrix), `findings-commands.md §D`
(commands table), `findings-agents.md` (agents).

**Key engine fact that shapes the design:** the overlay is *file-level*.
A domain pack cannot carry a partial delta — its copy either fully shadows
the core file or doesn't exist. So "keep a thin engineering layer" is not
an available mechanism; every fork is a full second master. The only stable
states are (a) core-only single master, or (b) a deliberate, annotated,
full fork. This spec drives every current pair to (a) and makes (b)
impossible to create silently.

## Goal

`domains/engineering/` contains zero same-named copies of core content
files; every universal improvement stranded on one side of a fork is merged
into the core master; the two live bugs (core `import.md`, core
`session-primer`) are fixed by that merge; and a CI test fails any future
PR that adds a domain-pack file shadowing a core file without an explicit
intentional-fork annotation. Installs for all three domains and all six
targets ship the merged content.

## Kickoff

Collapse the 34 duplicated core↔engineering content files to single
masters in core, merging stranded improvements both ways, then add a CI
parity test so drift can't recur silently.

**Status:** delivering — all merges, deletions, annotations, and the parity
test are done; full suite green; awaiting cold audit + verify gate.

**Pick up at:** cold audit → `hero spec verify content-dedup-resync`, then
commit (34 deletions + 16 core edits + content_parity_test.go).

→ `.hero/planning/features/content-dedup-resync/spec.md` · test:
`content_parity_test.go`

## Approach

Single-master in core. For each forked pair, merge the union of both
copies' improvements into the core file, scoping any genuinely
engineering-specific sentence in place ("when the engineering pack is
installed…") rather than keeping a second master. Then delete every
engineering duplicate — identical and merged alike. The overlay guarantees
engineering installs keep receiving the file via fallthrough (this is
existing, tested behavior — `overlay_test.go`).

Intentional cross-domain overrides do exist and stay: pm's `discover.md`
and `handoff.md` (full rewrites, classified intentional) and chat's four
command rewrites (dead pack, out of scope here). The parity test therefore
needs an annotation mechanism, not a blanket ban: a domain file that
shadows a core path must carry a `core_fork: <one-line reason>` frontmatter
key (name final at implementation; commands use plain `description:`
frontmatter so the key slots in beside it). Unannotated shadow = test
failure. Annotated-but-identical = also a failure (pointless copy).

Direction of merge for the 14 forked pairs (from the audit matrices):

| File | Merge direction | What moves |
|---|---|---|
| commands: capture, convention, note | eng → core | subproject-workspace guard |
| command: decide | eng → core | `hero_anchor` tripwire step |
| commands: handoff, prime, resume | eng → core | QUEUE.md refresh / queue-surfacing steps |
| command: import | eng → core | **`hero sync import` fix (live bug in core)** |
| agent: session-primer | eng → core | **`hero list --status/--mine` fix (live bug in core)** |
| skill: agent-reliability | eng → core | grounding check, Two-Reading Rule, persistence rules (universal); scope the `engineer.md`/Completion-Ledger cross-ref sentence in place |
| skills: next-md, next-handoff-emit | eng → core | `0cfe403` `.local.md` wipe warning (engine behavior, universal) |
| skill: context-injection | core is ahead | nothing to merge — delete eng copy |
| skill: spec-format | **both ways into core** | from eng: folder-per-spec rationale, `slug:` authority, `completed_at`, Mockups section (scope Mockups to "packs that ship `/mock`"); core already has supersede genealogy |

All these deltas are domain-neutral (QUEUE.md, tripwires, subprojects, and
the NEXT machinery are core engine features), so merging them into core is
correct for pm/sales too — it *upgrades* those installs.

The parity test lives beside the existing overlay tests (repo root
`content_test.go` / `overlay_test.go` territory): walk every embedded
domain FS, and for each path that also exists in `CoreFS()`, require
either byte-equality-is-absent (i.e. the file shouldn't exist — fail with
"delete the redundant copy") or a `core_fork:` annotation with a nonempty
reason. Grandfather list: pm `discover.md`, pm `handoff.md` (annotate them
in this change).

## Changes

1. **Merge the forked pairs into core** per the direction table above —
   9 command files, 1 agent file, 4 skill files under `core/`. Each merge
   is a targeted edit porting the named delta, not a wholesale copy;
   `spec-format` is the one true two-way merge.
2. **Annotate the two intentional pm overrides** — add `core_fork:` +
   reason to `domains/pm/commands/discover.md` and
   `domains/pm/commands/handoff.md` frontmatter.
3. **Delete all 34 engineering duplicates** — `domains/engineering/`
   copies of: 13 skills (agent-reliability, auto-knowledge-capture,
   context-injection, convention-writing, documentation-practices,
   executive-report, knowledge-flywheel, next-handoff-emit, next-md,
   note-capture, nudge-awareness, project-context-generation, spec-format),
   17 commands (blocked, capture, check, convention, decide, discover,
   docs, drive, handoff, hero, import, note, prime, resume, retro, scan,
   why), 4 agents (convention-author, documentation-engineer,
   project-context-builder, session-primer).
4. **Add the parity test** — new test walking `AvailableDomains()` FSes
   against `CoreFS()`: unannotated shadow fails; annotated identical copy
   fails; annotation without reason fails.
5. **Fix collateral references** — grep for content that names the deleted
   paths (e.g. anything pointing at `domains/engineering/skills/spec-format`)
   and repoint to core; run `hero docs check` and fix README/docs counts
   that the deletions change.

## Boundaries

- **No content-quality rewrites.** Verbosity cuts, phantom-reference
  fixes inside pm/sales bodies, and harness-agnosticism scoping belong to
  audit follow-ups #2/#3/#6/#8. This spec only moves/merges/deletes
  existing text (plus the one-line scoping sentences the merges require).
- **Core commands that dangle on pm/sales installs** (core `decide.md`
  delegating to engineering-only agents, etc.) are follow-up #4
  (`core-commands-domain-neutral`) — pre-existing, not made worse here.
- **Where content directories live** stays with `core-vertical-layering`;
  this spec changes file *count*, not layout.
- **The chat pack** is untouched (follow-up #10).
- **`generateEngineeringAgentsMdBody`** (Go fallback for AGENTS.md) is not
  in the duplicate set — AGENTS.md files are per-domain by design.

## Risks

- **Hidden consumers of the engineering paths.** A test, embed directive,
  or docs count may assert the engineering copies exist (`hero docs check`
  compares README counts against directory listings — skill/command/agent
  counts will change). Mitigation: Changes step 5 + full `go test ./...`
  (the goreleaser hook runs it anyway).
- **Overlay fallthrough assumption.** The whole design rests on
  engineering installs receiving core files for paths the pack no longer
  ships. `overlay_test.go` covers this, but Validation includes a live
  per-domain, per-target install check — per the
  `harness-changes-cover-all-targets` tripwire, verify propagation on all
  six targets, not just claude.
- **Merge fidelity.** A missed delta silently re-ships stale guidance —
  the exact failure this spec fixes. Mitigation: after merging, `diff` each
  deleted engineering copy against the new core master and account for
  every removed line (either merged or explicitly rejected with a reason
  noted in the PR description).
- **Annotation key design.** `core_fork:` must not collide with harness
  frontmatter consumers (opencode reads agent frontmatter directly).
  Verify unknown keys are ignored by all six targets before finalizing the
  name.

## Acceptance Criteria

- THE SYSTEM SHALL contain zero files under `domains/engineering/` whose relative path also exists under `core/`.
- WHEN `hero install` runs with `--domain engineering` THE SYSTEM SHALL install every previously-duplicated file from its core master, on all six targets (opencode, cursor, claude, copilot, codex, generic).
- WHEN `hero install` runs with `--domain pm` or `--domain sales` THE SYSTEM SHALL install core copies containing the merged improvements (subproject guard, `hero sync import`, `hero list` session-primer fixes, `.local.md` wipe warning, QUEUE.md steps).
- THE SYSTEM SHALL ship a core `spec-format` skill containing both the supersede genealogy (from core) and folder-per-spec/`slug:`/`completed_at` content (from engineering).
- IF a domain-pack file shadows a core path without a `core_fork:` annotation THEN THE SYSTEM SHALL fail the parity test.
- IF an annotated domain-pack shadow is byte-identical to its core file THEN THE SYSTEM SHALL fail the parity test.
- WHEN the full test suite runs THE SYSTEM SHALL pass, including `hero docs check` against the updated content counts.

## Completion Ledger

| # | AC / Change item | Status | Evidence |
|---|---|---|---|
| AC1 | Zero engineering files shadowing core paths | DONE | `comm -12` over both trees: empty; parity test passes |
| AC2 | Engineering installs ship every previously-duplicated file from core, six targets | DONE | install matrix (tree-built binary): content sentinels found on all targets, with renames (copilot `*.prompt.md`, codex `command-*/SKILL.md`) verified by content. Exception documented: cursor installs no skills at all — pre-existing target bug (`target_cursor.go:33` uses `installFlat` on nested skill dirs), byte-identical behavior with the pre-change release binary; filed as background task. Cursor's previously-duplicated commands/agents install correctly from core |
| AC3 | pm/sales installs get merged improvements | DONE | matrix sentinels on pm/sales × all targets: `hero sync import`, `hero list --status delivering`, `.local.md` machine-state-only warning, supersede genealogy |
| AC4 | Core spec-format carries both supersede genealogy and folder-per-spec/`slug:`/`completed_at` | DONE | both sentinel families present in installed copies; merge diff shows old core content 100% retained, exactly 1 eng line replaced (legacy supersedes row) |
| AC5 | Unannotated shadow fails parity test | DONE | fixture exercise: unannotated copy of note.md → FAIL; removed → pass |
| AC6 | Annotated content-identical shadow fails parity test | DONE | comparison strips the `core_fork:` line first (cold-audit recommendation applied — an annotation-only diff is still a pointless copy); fixture exercise: annotated copy differing only by annotation → FAIL; removed → pass |
| AC7 | Full suite passes incl. docs check vs updated counts | DONE | `go test ./...` exit 0. `hero docs check` output is byte-identical before/after this change (its 2 mismatches are pre-existing: it counts root-level agents//skills/ dirs absent since the domains refactor — audit F14; nothing it measures changed here) |
| C1 | Merge forked pairs into core (14 files) | DONE | 11 verbatim ports (zero diff vs old eng copies), agent-reliability ported + 1 sentence scoped, spec-format two-way merged, context-injection core-already-ahead |
| C2 | Annotate pm discover/handoff with `core_fork:` | DONE | both frontmatter blocks carry annotation + reason; parity test passes them |
| C3 | Delete all 34 engineering duplicates | DONE | `git rm` 17 commands + 4 agents + 13 skill dirs; embed directives still valid (dirs non-empty); build green |
| C4 | Parity test | DONE | `content_parity_test.go` — walks AvailableDomains vs CoreFS; both failure branches fixture-exercised |
| C5 | Collateral refs + docs counts | DONE | repo-wide grep for all 34 deleted paths (excluding .hero/, web/docs): zero references; docs check unchanged (see AC7) |

Exercise-the-feature check: install matrix above is the live exercise —
18 real `hero install` runs from a tree-built binary with content
sentinel verification.

## Validation

- `comm -12 <(cd core && find . -name '*.md' | sort) <(cd domains/engineering && find . -name '*.md' | sort)` returns only AGENTS.md-adjacent expected rows (i.e. no agents/commands/skills overlap).
- Per-pair diff audit: for each of the 14 forks, the deleted engineering
  copy diffed against the new core master shows no unaccounted lines.
- Scripted install of each domain × each of the six targets into a temp
  dir; assert the merged files land and contain a sentinel line from each
  merged delta (e.g. the `hero sync import` invocation in import.md).
- New parity test fails when a test fixture adds an unannotated shadow;
  passes on the annotated pm pair.
- `go test ./...` green.
