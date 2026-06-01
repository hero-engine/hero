---
title: ContentFS Legacy Fallback Removal — Cut Engineering Over to domains/engineering/
slug: contentfs-legacy-fallback-removal
type: feature
status: completed
priority: P1
tags: [platform, domains, embed, refactor, cleanup]
created: 2026-05-19
relations:
  - target: hero-domains
    kind: parent
  - target: domain-plugin-architecture
    kind: follows
  - target: install-core-domain-merge
    kind: depends_on
horizon: next
smoke: deferred
completed_at: 2026-05-19T16:59:21Z
---

## Kickoff

Follow-up to the now-completed `domain-plugin-architecture` spec
(`.hero/specs/domain-plugin-architecture/spec.md`). That spec's B1
delivery deliberately left `ContentFS()` wired to `legacyContent` (the
root-level `agents/`, `commands/`, `skills/` embed) instead of cutting
over to `domains/engineering/`. The decision is recorded in
[content.go:37-54](content.go:37) and in the parent spec's "Decision —
ContentFS legacy fallback retained (B1, 2026-05-17)" section.

This spec finishes that cutover: reconcile the two surfaces, drop the
legacy fallback, and make engineering go through the same domain-pack
path as `pm` and `sales`.

**Blocked on:** `install-core-domain-merge`. That spec fixes the real
behavior gap underneath (today's install pipeline reads only one FS
and never merges `hero.CoreFS()` with the active domain, so the
universal `core/agents`, `core/commands`, `core/skills` layer is
embedded but never installed). Doing the cutover before the core-merge
ships would freeze a buggy install shape — engineering and other
domains would still be missing core content — into the "consistent"
target this spec defines. Land core-merge first; then this becomes a
clean wiring refactor.

**Status:** planning — no code written yet. Parent spec is completed
and archived; do not edit it.

**Pick up at:** `/deliver contentfs-legacy-fallback-removal`

**Files:** content.go (lines 16-54 and 59-80), domains/engineering/**,
agents/**, commands/**, skills/**, internal/install/install.go (root
asset wiring), AGENTS.md.

**Skip:** Any changes to the `pm` or `sales` domain packs. Any change
to `CoreFS()` / `CoreVocabulariesFS()` / `CoreMethodologiesFS()` /
`CoreSpecTypesFS()` — those already go through the domain-pack model.

## Problem

`domain-plugin-architecture` introduced a domain-pack model where every
vertical (engineering, pm, sales) is supposed to be a sibling under
`domains/<name>/`. For pm and sales that's true today; for engineering
it isn't.

Two surfaces currently coexist:

1. **Root-level `agents/`, `commands/`, `skills/`** — the
   actively-maintained engineering source. Embedded as `legacyContent`.
   `ContentFS()` returns this. `DomainFS("engineering")` also returns
   this.
2. **`domains/engineering/agents/`, `commands/`, `skills/`,
   `spec-types/`** — a scaffolded mirror. Embedded as
   `engineeringContent`. Used only by `DomainSpecTypesFS("engineering")`
   today; the agents/commands/skills subtrees are present in the embed
   but never read.

The duplication has three costs:

- **Drift risk.** Anyone editing engineering content has to know to
  edit root-level dirs, not `domains/engineering/`. There's no
  enforcement; the embeds compile cleanly even if the two surfaces
  diverge.
- **Asymmetry in the domain interface.** Engineering doesn't behave
  like a domain — it short-circuits to `legacyContent` inside both
  `DomainFS` and `DomainSpecTypesFS`. Anything that reasons over
  domains uniformly (pack-listing tools, `hero domain switch`, future
  third-party-pack support) has to special-case `engineering`.
- **Two embeds for the same content.** Every binary carries the
  engineering tree twice — once at root, once under `domains/engineering/`.

The parent spec deferred this on purpose: B1 was the PM embed cutover,
not an engineering content migration. The work to do it cleanly is
this spec.

## Goal

Make `engineering` a real domain pack: one source of truth under
`domains/engineering/`, one embed, one code path through
`DomainFS` / `DomainSpecTypesFS`. Remove `legacyContent` and the
root-level `agents/` / `commands/` / `skills/` dirs, or make them
generated artifacts of `hero install`.

## Design

### Phase 1 — Parity check (no code changes shipped)

Before cutting anything over, prove the two surfaces match
bit-for-bit. Add a one-shot parity check (test or script):

- Walk `agents/`, `commands/`, `skills/` at the repo root and
  `domains/engineering/agents/`, `domains/engineering/commands/`,
  `domains/engineering/skills/` in parallel.
- Compare file sets (no missing or extra files on either side).
- Compare file contents byte-for-byte.
- Fail loudly with a per-file diff if they don't match.

If they don't match (likely), reconcile by treating root as
authoritative and syncing into `domains/engineering/`. The mirror is
explicitly the side that's allowed to be incomplete per the
`legacyContent` comment, so root wins on any conflict.

This phase ends when the parity check passes in CI.

### Phase 2 — Cut ContentFS() over to domains/engineering/

Change [content.go:56-61](content.go:56) so `ContentFS()` returns
`fs.Sub(engineeringContent, "domains/engineering")` instead of
`legacyContent`.

At this point the root dirs are still embedded and still on disk, but
nothing reads from them at runtime. Keep the parity check in CI so a
regression is impossible during this window.

### Phase 3 — Collapse the engineering branch in DomainFS

Today [content.go:69-72](content.go:69) returns `legacyContent`
directly for `engineering` (and the empty default). After Phase 2,
change it to take the same `fs.Sub(engineeringContent, ...)` path the
other domains take.

Same for [content.go:135-149](content.go:135):
`DomainSpecTypesFS("engineering")` is already correct (it reads from
`engineeringContent`), so this is just removing the special case from
`DomainFS`, not `DomainSpecTypesFS`.

After this, the function bodies of `DomainFS` and `DomainSpecTypesFS`
have no `if domain == "engineering"` early return — the switch handles
it like any other pack.

### Phase 4 — Drop the legacyContent embed

Delete the `//go:embed agents commands skills` directive and the
`legacyContent` var entirely. The legacy comment block at
[content.go:37-54](content.go:37) goes away with it.

### Phase 5 — Remove or regenerate the root-level dirs

Two options here, decided in delivery:

- **5a (preferred): delete `agents/`, `commands/`, `skills/` from the
  repo root.** They're no longer embedded and no longer read. The
  canonical location is `domains/engineering/...`. Engineers editing
  content edit there.
- **5b (fallback if tooling outside Hero depends on root paths):
  make root dirs generated.** `hero install` writes them from
  `domains/engineering/` at install time; add a `.gitignore` entry so
  they don't get committed.

Pick 5a unless we find a concrete external consumer that needs the
root layout. Anything inside this repo that reads from `agents/`,
`commands/`, `skills/` directly (bypassing the embed) should be
switched to the canonical path during this phase.

### AGENTS.md

The root `AGENTS.md` is in scope too — same treatment. The parent spec
called for moving it to `domains/engineering/AGENTS.md`; do that move
here and update any references.

## Changes

- `content.go` — `ContentFS()` now returns
  `fs.Sub(engineeringContent, "domains/engineering")` via
  `DomainFS("engineering")`. The `legacyContent` var and its
  `//go:embed agents commands skills` directive are deleted. The
  `engineering`-specific early return in `DomainFS` is gone; empty
  and `"engineering"` both flow through the same switch path that
  serves `pm` and `sales`. Package doc-comment retained (still
  accurate; no legacy-fallback language remained after the cutover).
- `content_test.go` — `TestEmbeddedAgents/SkillsFrontmatter` drops
  the `legacy/root` case (the embed is gone). `TestDomainFS_DefaultAndEmpty`
  loses its "legacy fallback contract" wording.
- `parity_check_test.go` — added in the Phase 1 sync commit as the
  fail-loud parity gate; removed in the Phase 5 commit once the root
  surfaces themselves are gone.
- `domains/engineering/agents/`, `domains/engineering/commands/`,
  `domains/engineering/skills/` — synced from root to bit-for-bit
  parity. 36 files added (core-tier agents/commands/skills that had
  been authored at root) and 3 commands updated to the newer root
  copy. Root wins on conflict per the spec.
- `domains/engineering/AGENTS.md` — overwritten with the newer root
  copy.
- `agents/`, `commands/`, `skills/`, `AGENTS.md` at repo root —
  removed (5a). Audit found no external consumer of root paths.
- `internal/cli/markdown_drift_test.go` — surfaces re-pointed from
  root `commands/`/`skills/`/`agents/` to `domains/engineering/...`
  equivalents. Test intentionally not widened to scan `core/` or
  sibling packs (pm/sales) — both carry pre-existing invocation drift
  out of scope here.
- `README.md`, `CLAUDE.md` — repository-layout sections updated to
  point at `domains/engineering/` and document the `core/` overlay
  introduced by `install-core-domain-merge`.

## Acceptance Criteria

- WHEN the parity-check tool runs against the repo BEFORE the cutover
  THE SYSTEM SHALL report zero diffs between root `agents/`,
  `commands/`, `skills/` and the equivalents under
  `domains/engineering/`.
- WHEN `ContentFS()` is called THE SYSTEM SHALL return a filesystem
  rooted at `domains/engineering/` (verifiable by listing a known
  file path that exists only after the cutover).
- WHEN `DomainFS("engineering")` and `DomainFS("")` are called THE
  SYSTEM SHALL return the same filesystem as `ContentFS()`, served
  through the same code path used for `pm` and `sales` (no
  `engineering`-specific early return remains in `DomainFS`).
- WHEN the binary is built THE SYSTEM SHALL NOT contain a
  `legacyContent` embed; the root-level `agents/`, `commands/`,
  `skills/` are either absent from the repo (5a) or absent from the
  embed surface and produced only by `hero install` (5b).
- WHEN `hero install` runs on a project whose `hero.json` has no
  `"domain"` key THE SYSTEM SHALL install engineering content and
  match the post-install file tree of an explicit `--domain engineering`
  install byte-for-byte.
- WHEN `hero init --domain engineering` runs in a fresh workspace
  THE SYSTEM SHALL produce agents, commands, and skills sourced from
  `domains/engineering/`.
- WHEN `hero domain switch engineering` runs on a workspace currently
  on `pm` THE SYSTEM SHALL restore engineering agents/commands/skills
  using the same code path that handles `hero domain switch pm`.
- WHEN `hero domain list` runs THE SYSTEM SHALL list `engineering`
  using the same mechanism that surfaces `pm` and `sales` (no
  hardcoded engineering entry).
- THE SYSTEM SHALL continue to pass all existing install, init, and
  domain-switch tests with no behavioral changes visible to a user
  whose project was created before the cutover.

## Boundaries

- Does **not** change the content of any agent, command, or skill —
  this is a wiring refactor, not a content edit.
- Does **not** touch `pm` or `sales` domain packs.
- Does **not** change `CoreVocabulariesFS()` /
  `CoreMethodologiesFS()` / `CoreSpecTypesFS()` — those already go
  through the domain-pack model.
- Does **not** introduce third-party / on-disk domain packs (still
  deferred per the parent spec).
- Does **not** revisit the `domain-plugin-architecture` spec — that
  one is completed and archived.
- Does **not** address the gap that `hero.CoreFS()` is not currently
  merged into install output. That's owned by the prerequisite spec
  `install-core-domain-merge`, which **must land first** — see the
  Kickoff "Blocked on" line.

## Risks & Mitigations

- **Drift between root and `domains/engineering/` at the moment of
  cutover.** The Phase 1 parity check is the mitigation; the cutover
  doesn't ship until it passes.
- **External tooling (CI, docs, IDE configs, downstream installers)
  that reads from root-level paths.** Audit during Phase 5; if any
  hard dependency turns up, fall back to 5b (generated root dirs).
- **`hero upgrade` on older projects.** Projects without a `domain`
  key default to engineering; that path needs to keep working through
  the cutover. Covered by the existing install/upgrade tests plus the
  default-domain acceptance criterion above.

## Implementation notes

- **Parity scope.** The naive read of Phase 1 is "root agents/ ==
  domains/engineering/agents/". Reality is messier: root was the union
  of core-tier and engineering-tier content (e.g. `convention-author`,
  `kickoff-prompt`, `session-primer` live in `core/` post the
  install-core-domain-merge cutover). The install pipeline already
  layers core under domain via `OverlayFS(domain, core)` with domain
  winning, so pre-cutover the root copy was the one users saw. To
  preserve byte-for-byte install parity, root was synced into
  `domains/engineering/` wholesale — including files that duplicate
  `core/`. Those duplicates can be pruned later as a content cleanup;
  the wiring refactor deliberately did not touch them.
- **5a vs 5b decision.** Picked 5a (delete root dirs) because the
  audit (`grep -rn '"agents/"\|"commands/"\|"skills/"' internal/ cmd/`)
  turned up only embed-relative reads (paths inside the FS the embed
  serves), not OS-level reads of the repo's root directories. The
  Makefile bootstrap comment mentions root paths but only in the
  context of `./hero install` writing to `.hero/`. No external
  consumer was found.
- **AGENTS.md move.** Root `AGENTS.md` was newer than
  `domains/engineering/AGENTS.md` (168 lines of diff). Synced root →
  domain per the "root wins" rule and deleted from root. Not embedded
  by the install pipeline (`installInstructionsMd` generates AGENTS.md
  fresh per harness target), so its move is documentation-only.
- **Drift test scope.** `TestMarkdownInvocationsResolveAgainstRootCmd`
  previously walked root `commands/`/`skills/`/`agents/` plus root
  `AGENTS.md`. After the cutover, those surfaces moved under
  `domains/engineering/`. The test was repointed but intentionally
  not widened to scan `core/`, `domains/pm/`, or `domains/sales/`
  too. All three carry pre-existing drift (e.g. `hero event` calls
  that no longer resolve) which is out of scope for this wiring
  refactor and would turn the test red on landing.
- **`hero domain switch` semantics.** Verified post-cutover that
  `hero domain switch engineering` from a PM workspace flows through
  the same `internal/cli/domain.go` code path as switching to PM or
  sales — no `engineering`-special branch remains. The "switch leaves
  A-only files behind" behavior carried over from
  `install-core-domain-merge` is unchanged.
- **Pre-cutover install snapshot diff.** Built the binary at HEAD
  (pre-cutover) and again at the final commit, ran
  `hero install project /tmp/... --target claude --force` with both
  default and `--domain engineering` flags. All four scratch installs
  produced byte-identical output (`diff -r` reports zero diffs across
  every pair). This is the load-bearing evidence that the no-behavior-
  change contract held.
