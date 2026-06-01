---
title: Project Snapshot — Always-Fresh "Where Are We Across the Board" Surface
slug: project-snapshot
type: feature
status: completed
priority: P0
tags: [snapshot, surface, projection, context, cold-start, serve, mcp]
created: 2026-05-18
relations:
  - target: next-as-projection
    kind: relates-to
  - target: hero-now-home
    kind: relates-to
  - target: hero-work-home
    kind: relates-to
  - target: hero-surface-polish-v2
    kind: relates-to
  - target: spec-type-registry
    kind: relates-to
  - target: timely-briefs
    kind: relates-to
  - target: hero-pulse
    kind: relates-to
  - target: synthesis-maintenance
    kind: relates-to
depends-on:
  - next-as-projection
horizon: now
smoke: deferred
mission_alignment: |
  This spec is the cold-start "where are we" artifact a fresh
  session — or a fresh teammate — needs and currently cannot get
  without spelunking. NEXT.md already projects per-user / per-session
  handoff state; the project graph already knows in-flight specs,
  initiatives, completions, and blockers. What's missing is the
  cross-cutting *project-shaped* rollup: which surfaces exist in
  this codebase, what stage each is at, what's shipping now, and
  what's next across all of them at once. Without it, a fresh
  session has to ask "what is this project?" before it can ask
  "what should I do?" — and the answer to the first question lives
  in the senior dev's head, not the corpus.

  Mission-fit: "Does this make the next agent session start smarter
  than the last one ended — and does it raise the floor for everyone,
  not just the senior dev who already knows what to ask?" Yes on
  both counts. Senior devs get a one-call rollup; juniors and new
  teammates get a substitute for the institutional knowledge they
  don't have yet. The history dimension extends this from
  point-in-time to *trajectory*: a new session can see "here's
  where we were a month ago, here's where we are now, here's the
  delta" — the same omniscient-start promise applied to project
  arc, not just project state. That is directly the pitch.
principles_check: |
  Serves #3 (sessions start omniscient) directly — Snapshot is the
  first thing a fresh session reads after Mission and NEXT. Serves
  #5 (raise the floor) — the snapshot is identically useful whether
  you've been on the project for two years or two minutes. Risks #1
  (it just works) if the surface declaration becomes a chore the
  user has to maintain; mitigated by deriving as much as possible
  and asking the user to declare only the minimum the graph cannot
  infer (what a surface IS, not what's happening on it). Risks
  competing with `hero-now-home` and `hero-pulse`; we compose with
  both — Now is the personal surface, Pulse is the narrative
  weekly, Snapshot is the project-shape rollup. Three distinct
  audiences, three distinct artifacts, one shared graph.
completed_at: 2026-05-18T19:25:38Z
---

## Goal

A perpetually fresh, single-surface artifact — `.hero/SNAPSHOT.md`
plus paired MCP / CLI / serve readers — *and a time-snapped archive
trail under `.hero/snapshots/`* — that together answer, in one place,
**where the project is now** and **how it got here**:

1. **What surfaces exist in this project.** A `surface` is a
   coherent shipping unit a user can name aloud: "the CLI," "hero
   serve," "the docs site," "the landing page," "the MCP server,"
   "the chat domain pack." For hero-engine itself the snapshot lists
   at least: `core` (CLI + engine), `serve` (hero serve daemon and
   web companion), `mcp` (MCP server tools), `docs` (web/docs
   mkdocs site), `landing` (web/landing static site), plus each
   domain pack (`domains/engineering`, `domains/pm`, `domains/qa`,
   `domains/sales`, etc.).
2. **What stage each surface is at.** A six-state lifecycle:
   `concept → scaffolded → building → shipping-v1 → shipped →
   maturing`. Per surface, projected from spec frontmatter + git
   activity + tags.
3. **For pre-release surfaces, how complete vs. initial-release scope.**
   Resolved through a priority chain (explicit `release_target:`
   frontmatter → tracker integration → git-tag heuristic), with
   `done / total` counts and the next gating spec. When no release
   signal resolves for any spec on a surface, the column is hidden
   and a footnote explains how to add one.
4. **For shipped surfaces, releases / initiatives in flight.**
   Rolled up from `initiative` specs that target the surface.
5. **What was recently completed, project-wide.** Last N days,
   deduplicated by surface, rendered as a tight list — not a
   per-spec firehose.
6. **What's next, top-ranked across all surfaces.** Pulled from the
   existing ranked queue, but grouped by surface so the reader sees
   "two next-up items in serve, one in mcp, four in core."
7. **Open risks and blockers across the board.** From `hero
   blocked`, stale-in-flight specs, failing acceptance criteria,
   open bugs aged past a threshold.

The snapshot must be readable as: a rendered markdown file, an MCP
tool call, a CLI command, and a hero-serve page. It is *discovered*
via a one-line pointer the projector writes into NEXT.md and
AGENTS.md; it is *never* auto-injected into session context.
Consumers pull when they need it.

In addition, the snapshot has a **history dimension**: opt-in
milestone / manual / staleness triggers preserve dated, immutable
copies under `.hero/snapshots/YYYY-MM-DD.md` so the project carries
a *trajectory*, not just a current-state mirror. A fresh agent
session can read "here's where we were, here's where we are, here's
the delta" — the same composition pattern that powers NEXT.md, but
extended across time.

The design generalizes beyond hero-engine. Any project Hero
manages — including those with one surface, ten surfaces, or
none-declared (fall back to "project root is the only surface") —
must produce a useful snapshot with zero ceremony.

### Unifying principle: high-quality artifacts, zero-config inference, explicit pull

Every design decision in this spec follows one principle:

> **Hero produces high-quality artifacts with zero-config inference;
> consumers pull when useful; overrides exist for when inference
> is wrong; nothing gets forced into context.**

Concretely, this principle shows up in five places below — the
surface list is *inferred* from repo structure rather than
declared up-front; the release target is *resolved through a
priority chain* rather than demanded as a single field; the
snapshot is *discoverable* via a pointer in NEXT/AGENTS rather
than pushed into every session; archive cadence is *triggered by
milestones* with a staleness safety net rather than fixed
intervals; archives themselves are *strictly isolated* from
default discovery so historical data can never poison live
context. The user gets a working snapshot the first time they
install Hero in a repo. They get override knobs only when
inference is wrong. They never get heavy artifacts injected into
context without an explicit pull.

## Why this is mission-critical

A fresh session today, when asked "where are we on Hero?", has to:

1. Open `.hero/mission.md` (5KB of doctrine — useful but not status).
2. Open `.hero/NEXT.md` (per-session handoff — what *I* did, not
   what's true project-wide).
3. Open `.hero/QUEUE.md` (49KB of ranked specs — firehose, no
   grouping).
4. Run `hero status` (lists 87 in-flight + 114 completed — flat,
   ungrouped, no surface awareness).
5. Scan `internal/serve/pages/` and `web/` to guess what surfaces
   even exist.
6. Ask the senior dev.

Steps 1–5 take 10+ minutes and miss the cross-cutting shape. Step 6
is what we built Hero to make unnecessary. The senior dev's mental
model — "we have a CLI, a serve daemon, a docs site, a landing
page, an MCP server, and domain packs; serve is shipping v1, docs
just scaffolded, landing just landed, mcp is mature" — exists
nowhere on disk. Snapshot is that model, captured once, refreshed
always, readable in 30 seconds.

This is the **principle #3 surface for project shape**, the same
way NEXT.md is the principle #3 surface for session handoff.

## Composition with existing surfaces

Snapshot is a **rollup over the same graph** that powers NEXT, the
serve homes, and `hero status`. It introduces no new data sources;
the new contribution is the **`surface` axis** and the cross-cutting
projection over it.

| Existing artifact | Audience | What it answers | Relationship to Snapshot |
|---|---|---|---|
| `.hero/mission.md` | Anyone | Why this project exists | Snapshot's header links to it; no overlap |
| `.hero/NEXT.md` | Fresh session | What happened last + what's next *for the user/session* | Snapshot is the **project** complement; NEXT is the **session** complement |
| `.hero/QUEUE.md` | Anyone needing the full backlog | Every ready spec ranked | Snapshot's "Next up" pulls the top N from queue; queue stays the firehose |
| `hero status` | CLI users | Flat list of in-flight + completed specs | Snapshot adds surface grouping; both stay |
| Hero serve "Now" home | Personal cold-start | What *I* am working on / picked up | Snapshot is the **project shape** view; Now is the **personal** view |
| Hero serve "Work" home | Spec + delivery surface | Roadmap, Horizons, Kanban | Snapshot is a **shape** view; Work is a **detail** view |
| `hero pulse` | Anyone needing weekly narrative | Sprint / week summary in prose | Snapshot is the **structural** rollup; Pulse is the **narrative** complement |
| `.hero/snapshots/YYYY-MM-DD.md` | Trajectory readers / cold-start sessions | What the project looked like at a point in time | Same projector as `SNAPSHOT.md`, written to a dated path; **immutable** once written |
| `.hero/reports/` | Generators of one-shot artifacts (e.g. executive-report skill) | Curated narrative reports authored on demand | Different category: `reports/` is for *authored* outputs that explain or summarize for an audience; `snapshots/` is for *projected* time-snaps of the same structural rollup |

The decision boundary: **anything answerable by "what surfaces
exist and what shape are they in" belongs to Snapshot.** Anything
narrative, anything per-user, anything per-spec belongs elsewhere.

**Archives are deliberately not peers.** `.hero/snapshots/*.md`
are historical trajectory data and are intentionally *not*
indexed alongside specs / knowledge / live snapshot. They are
excluded from the default search index, skipped by
auto-knowledge-capture, omitted from /resume and /prime
cold-start bundles, and unreachable from default serve listings.
See the "Archive containment" sub-section below for the full
isolation invariants. The composition rule: archives compose
with the live snapshot through explicit history-querying
surfaces (`hero snapshot history|show|diff`, `hero_snapshot
at: <date>`, `/project/snapshots/<date>`) — never through the
default-discovery surfaces.

**Why `.hero/snapshots/` and not `.hero/reports/`.** The
hero-code repo established `.hero/reports/snapshot-YYYY-MM-DD.md`
as the location for executive-report outputs. Two paths considered:

- **Co-locate under `.hero/reports/`.** Pro: one folder for "things
  generated for human consumption over time." Con: conflates two
  fundamentally different artifacts. Reports are *authored* —
  hand-curated narrative, varying templates, audience-specific.
  Snapshots are *projected* — same projector output, deterministic
  shape, written by the system without prose decisions. Mixing them
  forces every reader (and every list view) to filter by kind.
- **Dedicated `.hero/snapshots/`.** Pro: clean separation between
  authored reports and projected time-snaps; lets each category
  evolve independent retention, indexing, and UI; matches the
  precedent of `.hero/specs/` vs `.hero/planning/` vs `.hero/knowledge/`
  — each top-level corpus folder maps to one artifact category.

**Decision: dedicated `.hero/snapshots/`.** Reports become a
category (currently one resident: executive-report), snapshots
become a category, and the `/project` serve home renders the
snapshot timeline; reports get their own surface treatment owned
by whatever skill produced them. The hero-code naming pattern
(`snapshot-YYYY-MM-DD.md`) is honored — same date format, same
ordering — minus the redundant `snapshot-` prefix since the
folder itself is named `snapshots/`.

## Approach

### Open question resolutions

#### 1. Surface modeling — **inference-first; `surfaces.yaml` is an override layer, not the source of truth**

Surfaces are **inferred automatically** from repo structure on every
projector run. The user authors nothing to get a working surface list.
`.hero/surfaces.yaml` is an *override layer* that exists only when
inference is wrong — and is created lazily when the user actually
needs to override something.

**Detection signals (in priority order):**

- Top-level directory shape: `internal/`, `cmd/`, `web/<surface>/`,
  `crates/<surface>/`, `apps/<surface>/`, `packages/<surface>/`,
  `domains/<pack>/`.
- Presence of package manifests: `go.mod`, `Cargo.toml`,
  `package.json`, `mkdocs.yml`, `wrangler.toml`, `pyproject.toml`.
- Paths declared in `hero.json` (when present).
- Naming hints: a `cli/` or `mcp/` folder is recognized as its own
  surface; an `internal/serve/` is recognized as the serve surface;
  etc.

Each detection emits a candidate surface with a `confidence`
score and a `rationale` listing which signals fired. The detector
infers an initial stage from the same signals (e.g. presence of a
shipped tag, count of completed specs on declared paths) using the
same six-stage taxonomy as before.

**Override semantics.** `.hero/surfaces.yaml` carries only the
deltas — never the full surface list. Supported override shapes:

- **Rename** an inferred surface id: `detected_id → preferred_id`.
- **Exclude** an inferred surface from the snapshot: `ignore: [id]`.
- **Add** a surface inference missed: full surface entry under
  `additions:`.
- **Field overrides** per surface: pinned `stage:`, `owner:`,
  custom `intent:` line, custom `paths:` list. These win over
  inferred values for the named surface.

The projector merges: **detected ∪ override-added − override-ignored**,
with override fields winning per surface.

**Zero-config experience.** A fresh Hero project gets a sensible
surface list with no human authoring. The example file shipped
with `hero init` is annotated `# This file is optional — Hero
infers your surfaces automatically. Edit only when inference is
wrong.`

**New CLI:** `hero snapshot detect [--explain]` — prints the
inferred surface list with rationale. `--explain` shows which
signals fired for each detected surface so the user can see
exactly why something was (or wasn't) inferred.

**Why option B from the original analysis is wrong.** The original
spec evaluated "declarative YAML file as source of truth" against
"register surface as a spec type" and "derive from directory
structure." Option B (declarative) was chosen on the grounds that
pure derivation is too lossy. That conclusion stands for *intent*
(what a surface IS) — but not for *existence* (which surfaces are
there). Inference is high-confidence on the latter and only fuzzy
on the former. The merge model captures both: inference handles
detection; override carries human intent where it matters.

Consistent with `next-as-projection`: declarative *anchors* live
in a separate file, projected content regenerates every turn.
Here the anchor file is a delta-only override layer rather than a
full registry. Same shape, lighter touch.

#### 2. Freshness model — **projected, mirroring NEXT.md exactly**

Same machinery as `next-as-projection`:

- The rendered file (`.hero/SNAPSHOT.md`) is a **total-rewrite
  projection** emitted on every Stop hook (and on the existing
  watched events: spec status changes, completions, commits,
  initiative changes, surface-declaration changes).
- The file is **tracked in git** (it's project state, not user state).
- A `hero-snapshot` git merge driver — added to `.gitattributes` —
  resolves merge conflicts by ignoring both sides and regenerating
  from the local graph. Same model as the `hero-next` driver in the
  projection spec.
- Surface inference runs on every projector pass; results feed the
  same Stop-hook render as everything else. `.hero/surfaces.yaml`
  (when present) is read as an override layer; the projector never
  writes to it.
- Header timestamp + content hash let stale-detection tools warn if
  the file is older than its source graph.

Staleness is the failure mode the user named first. A projected
file with merge-driver resolution and total-rewrite-on-events is
the strongest answer; we have it working for NEXT and re-using it
costs nothing.

The archive write path piggybacks on the same projector run:
after the live `SNAPSHOT.md` is written, the projector evaluates
the configured archive triggers (milestone / manual / staleness
safety net) and, if any fire, writes the same rendered bytes to a
dated file under `.hero/snapshots/` with the archive frontmatter
prepended. Archive writes are append-only and never replace an
existing dated file (same-day re-fires are no-ops unless an
explicit `--label` is provided).

#### 3. Surface as a first-class concept — **inferred from repo; overrides via `.hero/surfaces.yaml`**

Surface modeling is covered in detail under sub-section 1 above
(inference-first, override layer). The short version, retained
here for cross-referencing:

- Surfaces are **inferred** from repo structure and package
  manifests on every projector pass.
- `.hero/surfaces.yaml` is an **override layer**, not the source
  of truth — used only for renames, excludes, additions, and
  per-surface field overrides.
- Specs and initiatives attach to surfaces via a `surface: <id>`
  frontmatter field (rolled up by the projector). A spec without
  `surface:` is bucketed into the surface its file paths most
  likely belong to, with low confidence — and surfaces in the
  `(unassigned)` row when no inference resolves.
- Surfaces are **not** a new spec type. `spec-type-registry`
  covers nine canonical work-tracking types plus knowledge
  types; surfaces are a *facet on existing types*, not a type.
  This is consistent with the registry's posture on `vocabulary`
  and `methodology` as peer concepts rather than registry
  records.

Forward-compatible: if real need ever justifies promoting
surfaces to spec-type-registry records, the override schema is
trivially mappable to a registry shape.

#### 4. Access surfaces — **file + MCP + CLI + serve + discoverability pointer; no auto-injection**

Snapshot is a **discoverable artifact, not a pushed one.** The
projector never injects snapshot content into a session by
default. Consumers pull when useful. The five access surfaces:

| Surface | Path / handle | When to use |
|---|---|---|
| Rendered markdown file | `.hero/SNAPSHOT.md` | Anyone opens the repo; GitHub renders it; non-CLI viewers get it for free; tracked in git |
| MCP tool | `hero_snapshot` (new) | Agents asking "where are we across the board" — explicit programmatic pull |
| CLI command | `hero snapshot` | Terminal users; pipes into other commands; `--json` for scripting |
| Hero serve page | `/project` (new top-level home) | Browser users; clickable surfaces and drill-throughs |
| Discoverability pointer | One-line link in `NEXT.md` and `AGENTS.md` | The only thing the projector writes outside `.hero/SNAPSHOT.md` itself — *"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."* |

The pointer line is what makes Snapshot *discoverable* without
being *pushed*. A fresh agent reading NEXT or AGENTS sees the
link, knows the artifact exists, and can choose to read it. The
projector writes the pointer line on every Stop hook (idempotent
— inserted once, never duplicated).

**Optional pull from `/resume` and `/prime`.** Those two skills'
documentation is updated to mention that they may *optionally*
include the snapshot in cold-start bundles. That is an explicit
pull moment, not a push. Skills that don't include it work the
same as before; skills that do include it get the structural
view alongside their normal cold-start. Default behavior of
`/resume` and `/prime` is unchanged — they don't pull unless
explicitly configured to.

**Why a new top-level serve home and not part of Now / Work.** Now
is *personal* ("what am I working on"). Work is the *spec firehose*
("every spec, kanban, roadmap, graph"). Snapshot is *project shape*
("what is this project, where is each piece"). Three different
questions; three different surfaces. Snapshot lives at `/project`
in the surface shell next to the existing Now / Work / Knowledge /
Agents / People homes.

**MCP tool vs extending `hero_status`.** New tool. `hero_status`
returns a flat by-status grouping (delivering / planning /
completed); folding surface-shape rollup into it would either
break the existing schema or balloon the response. Keep
`hero_status` flat and add `hero_snapshot` for the shape view.
The two compose: `hero_status` answers "what's in flight,"
`hero_snapshot` answers "what's the project shape."

**No auto-injection.** Earlier drafts of this spec proposed
`auto_context.include_snapshot` to inject a portion of the
snapshot into every fresh session. That direction is removed.
The snapshot is heavier than NEXT and most sessions don't need
the full surface rollup. Discovery via the NEXT/AGENTS pointer
plus explicit pull via `/resume`, `/prime`, MCP, or direct file
read covers every real need without forcing snapshot bytes into
every context window.

#### 4a. Release modeling — **layered resolution with graceful degradation**

The "% to next milestone" column needs a release target per spec.
Earlier drafts assumed a single field (`release_target:`) carried
this. That assumption is wrong — many projects already encode
release scope in their tracker, in git tags, or in initiative
groupings. Hero should pick up whichever signal the project
already uses; only when none exists should the column disappear.

**Resolution priority chain (highest wins):**

1. **Explicit spec frontmatter.** `release_target: <name>` on the
   spec itself wins outright. If the spec doesn't declare one,
   check the parent initiative's frontmatter — initiative values
   cascade to child specs unless overridden.
2. **Tracker integration.** When tracker integration is
   configured, pull the release name from the tracker record:
   - Jira: `fixVersion`.
   - GitHub: milestone.
   - Linear: cycle.
   The tracker lookup happens via the existing tracker adapter;
   no new credentials, no new round-trips beyond what tracker
   sync already does.
3. **Git tag heuristic.** Configurable tag pattern (default
   `v*`). For each spec, associate it to the nearest *later* tag
   based on the commit range that touched the spec's files.
   This is a fallback for projects that release via tags but
   don't (yet) declare release targets in tracker or frontmatter.
4. **None.** If no signal resolves for any spec on a surface,
   hide the "% to next milestone" column for that surface and
   emit a footnote: *"No release model declared. Add
   `release_target:` on specs or initiatives, or configure
   tracker integration, to enable release rollup."*

**`release_target:` is optional everywhere.** Accepted on both
spec and initiative frontmatter; never required. The priority
chain means a project that uses any one of the four signals gets
a working rollup; a project that uses none gets a polite
footnote, not a broken column.

**Graceful degradation invariants:**

- A spec with no resolved release target is excluded from
  release-rollup counts (not silently bucketed into a phantom
  v1).
- A surface where *every* spec lacks a resolved release target
  hides the column entirely for that surface row, not just on
  the snapshot but on `/project/surface/<id>` as well.
- The footnote appears once per snapshot regardless of how many
  surfaces are unrolled.

This is the same "inference + override + graceful degradation"
shape as surface modeling. The user opts into a release model by
declaring one anywhere; Hero finds it; if no model exists, the
column quietly disappears rather than demanding configuration.

#### 5. Content sections — concrete structure

```markdown
# Project Snapshot — <project name>

> <mission one-liner — pulled from .hero/mission.md or
> frontmatter `mission:` field>

_Last refreshed: <RFC3339 timestamp> · projected from graph rev <hash>_

## Surfaces

| Surface | Stage | Path(s) | Initial-release | Last touched | Driver spec |
|---|---|---|---|---|---|
| core | shipped | `cmd/hero`, `internal/` | — | 2d ago | — |
| serve | shipping-v1 | `internal/serve/`, `web/companion/` | 18/22 (82%) | 1d ago | hero-surface-polish-v2 |
| mcp | maturing | `internal/serve/mcp/` | — | 5d ago | mcp-tool-filtering |
| docs | building | `web/docs/` | 3/12 (25%) | 4d ago | hero-docs-site |
| landing | shipped | `web/landing/` | — | 6d ago | — |
| domains/engineering | shipped | `domains/engineering/` | — | 1d ago | hero-pm |
| domains/sales | concept | `domains/sales/` | 0/8 (0%) | 14d ago | hero-sales |
| (unassigned) | — | — | — | — | 12 specs without surface |

## Active initiatives

- **Hero Surface Polish** (surface: serve) — 2/3 specs done; in flight: hero-surface-polish-v2
- **Web Surfaces Restructure** (surface: docs, landing) — 4/4 specs done · COMPLETED 2026-05-15
- **Cross-Repo Peering** (surface: core, mcp) — 1/5 specs done; in flight: cross-repo-peering
- **Hero Domains** (surface: domains/*) — 6/12 specs in flight

## Recently completed (last 14 days)

- **serve** — Now home polish follow-ups, surface polish v1
- **core** — Pre-commit auto-stage NEXT files; spec status integrity wins
- **landing** — Hero Landing Page shipped
- **mcp** — MCP tool filtering, two-tier responses

## Next up across surfaces

1. **serve** — `hero-surface-polish-v2` (P0, delivering)
2. **core** — `spec-type-registry` (P0, delivering)
3. **mcp** — `mcp-tool-filtering` followups (P1, delivering)
4. **core** — `cross-repo-peering` finishers (P1, delivering)
5. **docs** — `hero-docs-site` (P1, planning)

## Open risks & blockers

- **Blocked specs (3):** `hero-pm` (waits on `spec-type-registry`),
  `hero-qa` (waits on `spec-type-registry`),
  `agent-outposts` (waits on `hero-governance`).
- **Stale-in-flight (2):** `e2e-validation` (no commit in 21d),
  `inline-propose-output-mode` (no commit in 17d).
- **Aged open bugs (1):** `scan-enrichment-unbounded-loop` (open 23d).
- **Unassigned specs (12) — no `surface:` declared.** Run
  `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces declared: 7 · Specs covered: 75/87 (86%)
- Last surface-declaration edit: 2026-05-12
- Projection generation: ~120ms · Source nodes: 412
```

Refinements baked in:

- The `Surfaces` table is **the densest section** — the one a
  reader scans first. Six columns are the cap; more becomes noise.
- The `(unassigned)` row is **always present** when non-empty so the
  user knows when their declarations have drifted from reality.
- `Snapshot health` at the bottom is the meta-line — surfaces
  declared, coverage %, last-declared timestamp. Makes drift visible.
- Each surface row is a **deep link** in the serve renderer
  (`/project/surface/<id>`); the markdown version uses plain text
  so it renders cleanly on GitHub.

#### 6. Snapshot archives — time-snapped trajectory

`SNAPSHOT.md` answers *now*. Archives answer *over time*. Same
projector, same render, written to a dated, immutable path so the
project carries a trajectory the live mirror cannot.

**Storage & path.** Archives live at
`.hero/snapshots/YYYY-MM-DD.md`, optionally
`.hero/snapshots/YYYY-MM-DD--<slug>.md` when milestone-tagged
(e.g. `2026-05-18--v1-release.md`). One file per archive, plain
markdown, ordered by filename. Rationale for choosing
`.hero/snapshots/` over the hero-code precedent of `.hero/reports/`
is in the composition section above — the short version is
*reports are authored, snapshots are projected; mixing categories
forces every reader to filter*.

**Trigger model.** Milestone events and manual invocations are
the primary triggers. A configurable staleness safety net catches
quiet projects so they still accumulate trajectory data.

1. **Milestone.** Automatic archive on events that mark a project
   inflection. Detected events for v1:
   - **Release tag.** A new git tag matching the configured
     pattern (default `v[0-9]*`) pushed on the local repo. Archive
     written with `label = <tag name>` (e.g.
     `2026-05-18--v1.0.0.md`). Trigger fires on the
     `git post-commit` / `git post-receive` hook path the snapshot
     projector already listens to.
   - **Initiative completion.** Any `initiative` spec flipping to
     `status: completed`. Archive written with
     `label = <initiative-slug>`. Detected via the existing
     spec-status-change event stream.
   - **Sprint end.** When a sprint spec (per the in-flight
     `sprint` skill / `spec-type-registry`) flips to `completed`,
     archive with `label = sprint-<slug>`. Soft-gated on whether
     sprint integration is present; absent means no-op.
2. **Manual.** `hero snapshot archive [--label <slug>]` CLI
   command and an `archive: true` parameter on the
   `hero_snapshot` MCP tool. Always writes regardless of
   milestones or last-archive timestamp.
3. **Staleness safety net.** Config field
   `snapshot.archive.staleness_cutoff` with values `weekly |
   biweekly | monthly | quarterly | off`, **default `monthly`**.
   On every Stop-hook projector run, after the live file is
   written, the projector checks `now − last_archive_date`
   against the cutoff. If the cutoff is exceeded *and* no
   milestone fired on the same run, the projector writes a
   safety-net archive labeled `auto-staleness`. The check uses
   the max `snapshot_date` from existing archive frontmatter
   (no separate state file). Active projects archive often from
   milestones and never hit the cutoff; quiet projects still
   get periodic trajectory data without the user touching a
   knob.

**Why no fixed-interval trigger.** Earlier drafts proposed
`snapshot.archive.interval: weekly | biweekly | monthly` as a
primary trigger. That field is **removed**. The replacement —
`staleness_cutoff` — looks superficially similar but is
fundamentally different: it fires *only when no other archive
has been written in the cutoff window*, rather than firing on
schedule. Milestone-active projects never hit it; idle projects
still get coverage. One knob, one default, no v2 plan for a
default flip.

When multiple triggers fire on the same calendar day, the first
write wins; subsequent triggers append a numeric suffix
(`YYYY-MM-DD--<slug>-2.md`) only when an explicit `--label` is
provided. Otherwise milestone / staleness triggers that match an
existing same-day file are no-ops (idempotent). The staleness
check is suppressed when a milestone fires on the same
projector run, so the two never double-archive.

**What gets archived.** The archive file is **byte-for-byte the
same rendered markdown** as `SNAPSHOT.md` at the moment of
archiving — same projector, same template, same surfaces table,
same recent-completed and next-up sections. Do not invent a
parallel format. The only difference: archives carry a small
frontmatter block prepended at write time, which `SNAPSHOT.md`
does not:

```yaml
---
snapshot_date: 2026-05-18
trigger: milestone       # milestone | manual | staleness
label: "v1.0.0"          # optional, e.g. "v1.0.0", "hero-pm", "auto-staleness"
git_commit: 424bb36abc…  # HEAD sha at archive time
projector_version: 1     # bumps if rendered shape changes
historical: true         # isolation flag — never filter-back into live discovery
not_current: true        # isolation flag — pair with historical for explicit semantics
---
```

This frontmatter records archive provenance and carries the
two isolation flags (`historical: true`, `not_current: true`)
that any future indexer or tool can use to filter explicitly.
The flags are part of the file format; the writer refuses to
emit an archive without them. Archives are *not* registered
with the default search index. Frontmatter is traceable to the
commit the archive reflects, and forward-compatible if the
projector output schema evolves.

**Retention.** Default **keep all**. Markdown is cheap and
trajectory is the value. Optional `hero.json` field
`snapshot.archive.retention: all | last-N | none`
(default `all`). `last-N` keeps the most-recent N by
`snapshot_date` and deletes older ones at the end of each Stop
hook. `none` is equivalent to keeping nothing on disk — included
for completeness; not recommended. Compression / sub-directory
archival of older snapshots is **explicitly out of scope for v1**;
note it as a v2 follow-up if disk usage becomes a concern.

**Immutability.** Archives are **read-only artifacts**. Once
written, the projector never reads from them, never re-projects
them, never updates them. The only write paths are (a) initial
archive write and (b) explicit user `rm`. The `hero check`
command warns if an archive's content hash drifts from a
recompute on the same commit (signals tampering or a stale
projector run).

**Consumption surfaces.**

| Surface | Access shape |
|---|---|
| File | `.hero/snapshots/*.md` — plain markdown, searchable by the existing hero index, viewable on GitHub or any editor |
| MCP | `hero_snapshot` gains optional `at: <date \| latest>` (default `latest`); new `hero_snapshot` `history: true` flag returns the enumerated archive list `[{date, trigger, label, git_commit, path}]` |
| CLI | `hero snapshot history` — list archives with date, trigger, label; `hero snapshot show <date>` — render a specific archive to stdout; `hero snapshot diff <date-a> <date-b>` — text diff between two archives (or `<date>` vs `latest` / live) |
| Serve | `/project` home gains a **timeline strip** — clickable list of archived snapshots ordered newest-first, each opening a read-only view. **v2-deferred**: a "diff vs N weeks ago" comparison view rendered inline. |

The MCP tool keeps **one tool name** — `hero_snapshot` — with
flags rather than splitting into `hero_snapshot_list` /
`hero_snapshot_at` / etc. Tool discovery is expensive; flag
discovery is free; the agent reads one tool description and gets
all three call shapes.

#### 6a. Archive containment — isolation invariants

Archives are *historical trajectory data*. The risk model: a
six-month-old archive row claiming `hero-serve: building`
surfaces in `hero search` and the model treats it as current
state. That is the worst-case outcome of this whole feature —
agents acting on stale shape data and confidently telling the
user the wrong thing about where the project is.

The rule: **archives are reachable only through explicit
history-querying surfaces.** Default-discovery paths
(`hero search` without a flag, `/resume`, `/prime`,
auto-knowledge-capture, the live snapshot, the `/project` home
listings) never expose archive bodies.

**Isolation invariants** (each one is a design contract; the
reviewer should be able to verify each independently):

1. **Search index exclusion.** `.hero/snapshots/` is excluded
   from the default Hero search index via a `[search]
   exclude_paths` entry (or the equivalent in the existing
   index config). Opting in via `hero search --include-history`
   is the only path that returns archive bodies.
2. **Frontmatter flags.** Every archive file carries
   `historical: true` and `not_current: true` in its
   frontmatter. Any future indexer / tool can filter on these
   flags without inspecting paths or names.
3. **Banner line in body.** The projector prepends a banner to
   every archive body at write time:
   `> **Historical archive captured {date}.** This is a
   point-in-time snapshot, not current state. For live state
   see [SNAPSHOT.md](../SNAPSHOT.md).` The banner is part of
   the archive file format. The user cannot suppress it; the
   projector refuses to write an archive without it.
4. **Auto-capture subsystems skip archives.**
   `auto-knowledge-capture`, `note-capture`, and the
   auto-memory system must skip `.hero/snapshots/`. The skip is
   a documented requirement of those subsystems and is enforced
   by the same `[search] exclude_paths` config they already
   consult.
5. **Cold-start skills exclude archives.** `/resume` and
   `/prime` skills must not include archive content in their
   cold-start bundles. Only `SNAPSHOT.md` (the live file) is
   eligible. The skill documentation makes this explicit.
6. **Serve isolation.** The `/project` home renders archive
   *bodies* only when the user navigates directly to a specific
   archive URL (`/project/snapshots/<date>`). Archive bodies
   must not appear in any listing, homepage, or relevance feed.
   The timeline strip on `/project` shows only date + trigger +
   label + brief delta summary; clicking opens the full body
   on its dedicated URL.

**Intentional history-query surfaces** (the only ways to reach
archive content):

- `hero snapshot history` — list archives.
- `hero snapshot show <date>` — render one archive.
- `hero snapshot diff <a> <b>` — compare two archives.
- `hero_snapshot` MCP with `at: <date>` or `history: true`.
- `/project/snapshots/<date>` serve route (direct navigation).
- `hero search --include-history` — opt-in extended search.

Anything outside that list must not surface archive content.
This rule is the *single most important* invariant in the spec
— it's the difference between archives being a useful
trajectory and archives being a slow-motion footgun.

#### 7. Composition with existing surfaces — explicitly mapped

See the table in **"Composition with existing surfaces"** above. The
rule when adding new sections to Snapshot in the future: if the
question is **"what surfaces exist and how are they shaped,"** it
belongs here; otherwise it belongs in Now / Work / Pulse / NEXT / a
spec detail page.

### Surface stage taxonomy

Six lifecycle stages per surface, ordered:

| Stage | Definition | Derivation rule |
|---|---|---|
| `concept` | Surface declared; no shipping specs yet | Zero specs with `surface: <id>` AND `status: delivering` or `completed` |
| `scaffolded` | Directory exists, initial structure landed, no user-visible feature yet | At least one `completed` spec with `surface: <id>` AND no `release: v1` spec yet |
| `building` | Multiple specs in flight toward a v1 release | At least one `delivering` spec AND `<50%` of any declared v1 release scope complete |
| `shipping-v1` | Most of v1 scope complete; near release | `>=50%` of v1 release scope complete |
| `shipped` | v1 released; no incomplete initial-release specs | All specs with `release_target: v1` are `completed` AND no in-flight v1 spec |
| `maturing` | Shipped; only follow-up / polish / extension specs in flight | All `delivering` specs are `release_target: v2+` or `release_target: none` |

The user can pin a surface's stage in `surfaces.yaml`
(`stage: shipping-v1`) to override the derivation when the
heuristic doesn't fit. Pinned stages render with a `pinned` badge
in the serve view.

### Data flow

```
repo structure / manifests    ──┐ (surface inference)
.hero/surfaces.yaml           ──┤ (override layer, optional)
spec frontmatter (surface:)   ──┤                            ──► .hero/SNAPSHOT.md (live)
release signals (chain):      ──┤                            ──► pointer line appended to NEXT.md / AGENTS.md
  spec/initiative frontmatter ──┤                            ──► .hero/snapshots/YYYY-MM-DD.md
  tracker fixVersion/etc      ──┼──► internal/snapshot/  ──►    (archive — milestone / manual / staleness)
  git tag heuristic           ──┤    (projector)            ──► hero_snapshot (MCP)
spec status / lifecycle       ──┤                            ──► hero snapshot (CLI)
initiative rollups            ──┤                            ──► /project (serve, incl. timeline strip)
git commit activity           ──┤                            ──► .hero/reports/  (peer category, not written by snapshot)
hero blocked / aged-stale     ──┤
hero queue (top N)            ──┘
```

The projector is a peer of `internal/handoff/` (NEXT) and
`internal/queue/` (QUEUE) — same shape, same lifecycle, same
trigger model. Surface inference runs first; the override layer
(if `.hero/surfaces.yaml` exists) merges on top. Release
resolution runs the priority chain per spec. The live snapshot
renders; the NEXT/AGENTS pointer line is written (idempotent);
then the archive evaluator decides whether to fire a dated
archive. Archives apply the isolation invariants on write
(banner line, `historical: true` + `not_current: true`
frontmatter, no search-index registration) and are immutable
once written.

## Changes

### New package: `internal/snapshot/`

1. **`internal/snapshot/snapshot.go`** — the `Snapshot` struct and
   projection entry point. Exposes:
   - `Build(ctx, store, opts) (*Snapshot, error)` — assembles the
     full snapshot from the graph.
   - `Render(s *Snapshot, format) ([]byte, error)` — renders to
     markdown (default), JSON, or the compact MCP form.
   - `Project(ctx, store) error` — total-rewrites `.hero/SNAPSHOT.md`
     from the live graph; called by the Stop hook and the file
     watcher.
2. **`internal/snapshot/surfaces.go`** — surface inference +
   override loader. Inference scans repo structure
   (top-level dirs, package manifests, `hero.json` paths,
   naming hints) and emits a candidate surface list with
   confidence + rationale per surface. When `.hero/surfaces.yaml`
   exists, parses the override layer (renames, excludes,
   additions, field overrides), validates against the allowed
   stage enum, and merges with the inferred set
   (detected ∪ added − ignored, overrides win per field).
   Reports config errors with file:line. The merged surface list
   is what every other rollup uses.
2a. **`internal/snapshot/detect.go`** — pure-function inference
    rules; takes a repo snapshot and emits `[]CandidateSurface
    {id, paths, signals, confidence}`. Easy to table-test.
2b. **`internal/snapshot/release.go`** — release-target
    resolver implementing the priority chain (explicit
    frontmatter → tracker → git-tag heuristic → none). Per-spec
    resolution feeds the rollup; per-surface "any signal
    resolved?" determines column visibility.
3. **`internal/snapshot/stage.go`** — pure-function stage
   derivation per surface from the rollup inputs (specs by status,
   release scope, last commit). Easy to unit-test.
4. **`internal/snapshot/rollup.go`** — surface-keyed rollups:
   active initiatives, recent completions, next up, blockers,
   last-touched. Reuses `internal/queue/`, `internal/blocked/`,
   and `internal/feed/` rather than re-querying the store.
5. **`internal/snapshot/render_markdown.go`** — markdown template
   for `.hero/SNAPSHOT.md`. Plain text deep-link references; no
   HTML; renders cleanly on GitHub.
6. **`internal/snapshot/render_json.go`** — structured JSON for
   MCP / scripting.
7. **`internal/snapshot/snapshot_test.go`** — golden tests over a
   fixture workspace.
8. **`internal/snapshot/surfaces_test.go`** — surface declaration
   loader / validator tests.
9. **`internal/snapshot/stage_test.go`** — exhaustive table-driven
   tests over the six-stage derivation rules.

### Surface override file (optional)

10. **`.hero/surfaces.yaml`** — *override layer only*, not the
    source of truth. Created lazily, only when inference is
    wrong. Hero ships **no seeded entries** in this file for
    hero-engine itself — inference covers `core`, `serve`,
    `mcp`, `docs`, `landing`, and each `domains/<pack>`
    automatically from the existing repo shape. If a specific
    surface (e.g. domains naming) needs renaming or intent
    refinement, it goes here.

    Schema (validated by the loader):

    ```yaml
    version: 1
    # Optional file. Hero infers surfaces from your repo automatically.
    # Use this only to override inferred surfaces, exclude them, or
    # declare ones inference missed.
    renames:                        # optional; rewrite an inferred id
      - from: domains-engineering
        to: domains/engineering
    ignore:                         # optional; drop an inferred surface
      - id: scratch
    additions:                      # optional; declare a missed surface
      - id: serve-companion
        name: Web Companion
        intent: Companion shell embedded in hero serve.
        paths:
          - web/companion/
        stage: shipping-v1          # optional pin
        owner: chet-bellows         # optional freeform
        release_targets:
          v1:
            description: "First public-use companion release"
            scope_tag: surface-polish
    overrides:                      # optional; per-surface field overrides
      - id: serve
        stage: shipping-v1          # pin stage (overrides inferred)
        owner: chet-bellows
        intent: |
          Local daemon, web companion, MCP server, file watcher.
    ```

11. **`.hero/surfaces.example.yaml`** — annotated example shipped
    in scaffolding (`hero init` / `hero scan`) so projects that
    want overrides have a template. The header comment makes
    explicit: *"This file is optional. Hero infers your surfaces
    automatically. Edit only when inference is wrong."*

### Spec frontmatter facet

12. **`internal/spec/spec.go`** — add `Surface string` field on
    `Spec`; parse from frontmatter `surface:` key; validate
    against the loaded surface registry at lint time. A spec
    declaring `surface:` not in `.hero/surfaces.yaml` produces a
    structural lint warning (NOT an error — surfaces are
    organizational, not blocking). Add `ReleaseTarget string`
    field parsing `release_target:` for v1-scope rollup.
13. **`internal/triage/structural.go`** — add the soft warning for
    unknown surface IDs.

### Projector wiring

14. **`internal/snapshot/projector.go`** — wires `Project()` into:
    - The Stop hook (alongside NEXT.md projection).
    - The file watcher (on spec status flips and
      `.hero/surfaces.yaml` edits).
    - The `git post-commit` hook so newly-completed specs flip
      stage in real time.
    - **The pointer writer** — after each live `Project()`,
      call `pointers.Ensure(rendered, store)` which idempotently
      inserts the single-line pointer
      *"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."*
      into both `.hero/NEXT.md` and `AGENTS.md`. The pointer is
      inserted once (detected via exact-line match), never
      duplicated, and never adjusted in subsequent runs.
    - **The archive evaluator** — after the live file + pointer
      writes complete, call
      `archive.MaybeWrite(rendered, triggers, store)` which
      evaluates the milestone / manual / staleness-cutoff
      triggers and writes a dated archive if any fire. The
      archive writer applies the isolation invariants on write
      (prepends the banner line, sets
      `historical: true` + `not_current: true` in frontmatter,
      registers the file with the search-exclusion config rather
      than the index).
15. **Stop-hook integration in the existing hook script** — call
    `hero snapshot --project` after `hero next checkpoint`. Keep
    NEXT and SNAPSHOT projection sequential (NEXT first; SNAPSHOT
    second; SNAPSHOT can reference NEXT-projection data without a
    cyclic dependency).
16. **`.gitattributes`** — register the `hero-snapshot` merge
    driver for `.hero/SNAPSHOT.md`. Driver definition mirrors the
    `hero-next` driver from the projection spec: ignore both
    sides, regenerate from local graph. Archives under
    `.hero/snapshots/` use **no** merge driver — they are
    immutable, so conflicts on the same dated file indicate
    something pathological (two branches archived a different
    state on the same day) and should be resolved by hand.

### Archive subsystem

15a. **`internal/snapshot/pointers.go`** — idempotent pointer
     writer. `Ensure(rendered []byte, store)` inserts the single
     line *"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."*
     into `.hero/NEXT.md` and `AGENTS.md` (relative to each
     file's location) under a stable marker. Detects existing
     pointers via exact-line match; no-op when already present.
     Unit-tested for: insert, no-op-on-second-call,
     no-modification-of-other-lines, correct relative path per
     file location.

16a. **`internal/snapshot/archive.go`** — archive write + trigger
     evaluator. Exposes:
     - `MaybeWrite(rendered []byte, cfg, store) (*Archive, bool, error)` —
       evaluates configured triggers and writes
       `.hero/snapshots/<date>[--<slug>].md` with the prepended
       frontmatter
       (`snapshot_date`, `trigger`, `label`, `git_commit`,
       `projector_version`, `historical: true`,
       `not_current: true`) and the historical-archive banner
       line at the top of the body; returns the archive record
       + a `wrote` bool. The banner is part of the
       file-format contract: the writer refuses to emit an
       archive without it, and tests assert it appears at the
       top of the body bytes.
     - `List(store) ([]Archive, error)` — enumerates
       `.hero/snapshots/*.md`, parses frontmatter, returns
       sorted-newest-first.
     - `Read(date string) (*Archive, error)` — reads one archive
       by date or `date--slug` filename.
     - `Diff(a, b *Archive) (string, error)` — pure text diff
       (reuses `internal/diff/` or stdlib equivalent).
16b. **`internal/snapshot/archive_triggers.go`** — pure-function
     trigger evaluation. Inputs: existing archive list, current
     time, recent spec-status changes, recent git tags,
     `hero.json` config. Outputs: `[]TriggerHit` (each carrying a
     trigger kind + optional label). Easy to unit-test.
16c. **`internal/snapshot/retention.go`** — applies the
     `snapshot.archive.retention` policy at the end of each
     archive write: `all` is a no-op; `last-N` deletes oldest
     archives beyond the cap; `none` deletes everything (logged
     loudly).
16d. **`internal/snapshot/archive_test.go`** — covers trigger
     evaluator table-tests, retention application, idempotent
     same-day rewrites, frontmatter round-trip.

### CLI surface

17. **`internal/cli/snapshot.go`** — new top-level command:
    - `hero snapshot` — print the rendered markdown to stdout.
    - `hero snapshot --json` — structured JSON.
    - `hero snapshot --project` — rewrite `.hero/SNAPSHOT.md`
      from graph; idempotent. Also re-emits the pointer line in
      `NEXT.md` / `AGENTS.md` (no-op if already present).
    - `hero snapshot --section <name>` — print just one section
      (e.g. `surfaces`, `initiatives`, `next`); convenience for
      scripts that want to grab the table.
    - `hero snapshot detect [--explain]` — print the inferred
      surface list (after override merge) to stdout. With
      `--explain`, prints the signals that fired for each
      detected surface so the user can see why something was
      (or wasn't) inferred.
    - `hero snapshot assign` — interactive helper that lists
      `(unassigned)` specs and prompts the user to pick a
      surface for each; rewrites frontmatter in place.
    - `hero snapshot archive [--label <slug>]` — write an
      archive immediately, regardless of staleness-cutoff;
      trigger recorded as `manual`. Output: path of the written
      archive.
    - `hero snapshot history [--json]` — list archives ordered
      newest-first with date, trigger, label, git_commit.
    - `hero snapshot show <date>` — render a specific archive
      to stdout (date alone or `date--slug` accepted).
    - `hero snapshot diff <date-a> <date-b>` — text diff between
      two archives; `latest` and `live` are valid synonyms for
      the newest archive and the current `.hero/SNAPSHOT.md`
      respectively.
18. **`internal/cli/root.go`** — register the command + completion
    hints.

### MCP surface

19. **`internal/serve/mcp/tools/snapshot.go`** (or wherever new
    MCP tools land per the in-flight `mcp-server-refactor`) —
    register `hero_snapshot` tool with:
    - `compact: bool` — two-tier per existing MCP convention.
      Compact returns `{summary, ref_id, surfaces_count,
      shipping_count}` plus a ref_id for `hero_expand`.
    - `section: string` — optional; one of `surfaces`,
      `initiatives`, `recent`, `next`, `risks`, `all` (default).
    - `surface: string` — optional; restrict to one surface.
    - `at: string` — optional; `latest` (default) returns the
      live `.hero/SNAPSHOT.md`; a `YYYY-MM-DD` or `YYYY-MM-DD--slug`
      returns the matching archive.
    - `history: bool` — when true, returns the enumerated archive
      list `[{date, trigger, label, git_commit, path}]` instead
      of a single snapshot body. Mutually exclusive with `at`.
    - `archive: bool` — when true, writes an immediate manual
      archive (equivalent to `hero snapshot archive`) and returns
      the new archive record. Optional `label: string` accompanies.
20. **MCP tool documentation entry** — short description so the
    agent picks it correctly for "what's the state of the project."

### Serve page

21. **`internal/serve/pages/project/page.go`** (new home, mirrors
    the structure of `internal/serve/pages/now/`).
22. **`internal/serve/pages/project/data/snapshot.go`** —
    project-page data fetcher; calls `snapshot.Build()` and adapts
    for templates.
23. **`internal/serve/pages/project/templates/page.html`** — the
    surfaces table, recent completions, next up, risks, health,
    and (when archives exist) a **timeline strip** listing
    archived snapshots newest-first with date, trigger, label;
    each entry is a link to the archive view.
24. **`internal/serve/pages/project/templates/surface-detail.html`** —
    per-surface drill-in at `/project/surface/<id>`; includes
    every spec declaring `surface: <id>`, every initiative, recent
    commits to declared paths, blockers scoped to the surface.
24a. **`internal/serve/pages/project/templates/archive.html`** —
     read-only archive view at `/project/snapshots/<date>` (or
     `/project/snapshots/<date>--<slug>`); renders the dated
     markdown plus a header showing trigger / label / git_commit
     and a "back to current snapshot" link. Explicit
     `data-readonly` marker so the page can never be edited
     in-place.
24b. **`internal/serve/pages/project/data/archives.go`** —
     fetches the archive list for the timeline strip and individual
     archive bodies for the read-only view. Pure read path; no
     writes.
25. **`internal/serve/shell/templates/top-nav.html` (or wherever
    the shell nav lives)** — add `Project` tab between `Now` and
    `Work`. (Pair with the surface-polish-v2 spec — that one is
    fixing detail-route 404s; the new tab piggybacks on the same
    nav refresh.)
26. **`internal/serve/pages/project/page_test.go`** — 200 on the
    home + the surface detail route; 404 on unknown surface.

### Discoverability pointer (no auto-inject)

27. **`internal/snapshot/pointers.go`** — described above (item
    15a). Ensures the one-line pointer
    *"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."*
    lives in `.hero/NEXT.md` and `AGENTS.md`. This is the **only
    thing the projector writes outside `.hero/SNAPSHOT.md`
    itself.** No auto-injection of snapshot bytes anywhere;
    consumers pull via MCP / CLI / file open / serve / opt-in
    `/resume` and `/prime`.
27a. **`skills/resume.md` / `skills/prime.md`** — documentation
     updates noting that both skills may *optionally* include
     the live snapshot in their cold-start bundles. Default
     behavior unchanged — the option exists for users who want
     project shape in their cold-start; archive bodies are
     **never** eligible.
28. **`internal/config/config.go`** — add the archive +
    isolation config block. No `auto_context.include_snapshot`
    field (removed from this design).
    - `snapshot.archive.staleness_cutoff` — string enum
      `weekly | biweekly | monthly | quarterly | off`; default
      `monthly`.
    - `snapshot.archive.milestones` — bool, default `true` in
      v1. (Milestones are the primary trigger; default-on.)
    - `snapshot.archive.release_tag_pattern` — string, default
      `v[0-9]*`; only consulted when milestones is true.
    - `snapshot.archive.retention` — string enum
      `all | last-N | none`; default `all`.
    - `snapshot.archive.retention_count` — int, default `0`;
      only consulted when retention is `last-N`.
28a. **`internal/index/config.go`** (or wherever the existing
     search-index config lives) — add `.hero/snapshots/` to the
     default `exclude_paths` list. Archives are reachable only
     via `hero search --include-history`. Auto-knowledge-capture,
     note-capture, and the auto-memory subsystems consult this
     same exclusion list, so they pick up the exclusion
     automatically.

### Surfacing in other commands

29. **`internal/cli/prime.go`** — `/prime` *optionally* includes
    the rendered surfaces section when the user has opted in via
    config; default behavior unchanged. Archive bodies remain
    out of scope for `/prime` cold-start bundles.
30. **`internal/cli/check.go`** — `hero check` validates
    `.hero/surfaces.yaml` parses (when present; the file is
    optional so absence is not a warning). Validates
    `.hero/snapshots/` entries: parses each archive's
    frontmatter, asserts `historical: true` and
    `not_current: true` are set, warns on archives whose
    `git_commit` is unreachable from the current repo, warns on
    archives missing required frontmatter fields, asserts the
    banner line is present at the top of the body, confirms
    `.hero/snapshots/` is registered in the search index's
    `exclude_paths` list, and (when retention is `last-N`)
    confirms the on-disk count matches the configured cap.

### Documentation

31. **`docs/cli/snapshot.md`** (or wherever CLI docs land in
    web/docs) — full usage; surface inference signals; override
    schema for `surfaces.yaml`; release resolution chain;
    staleness-cutoff cadence; archive isolation invariants;
    history-querying surfaces.
32. **`AGENTS.md` / project context** — one paragraph noting that
    Snapshot is the project-shape rollup and how it composes with
    NEXT / QUEUE / serve homes.

### Tests

33. **Golden snapshot test** — fixture workspace at
    `internal/snapshot/testdata/` with N inferred surfaces, M
    specs across statuses; assert the rendered markdown matches
    a golden file byte-for-byte.
34. **Stage derivation table-test** — exhaust every stage
    transition rule from the table above.
35. **Surface detection table-test** — given fixture repo
    structures (Go single-module, monorepo with `apps/*`, Rust
    `crates/*`, JS workspaces with `packages/*`, mkdocs-only
    repo), assert the inferred surface list and the rationale
    signals match expected outputs. Includes the `--explain`
    output path.
35a. **Surfaces override merge test** — start from an inferred
     set; apply renames, ignores, additions, and field
     overrides; assert the merged result matches expected.
     Verifies override-field-wins and ignore-removes-from-set.
35b. **Release resolution chain test** — for a spec with no
     signal, with explicit frontmatter only, with tracker
     fixVersion only, with git-tag heuristic only, and with
     multiple signals present, assert the resolver picks the
     right source per the priority chain.
35c. **Pointer-write idempotency test** — `pointers.Ensure`
     inserts the pointer line into NEXT.md / AGENTS.md on first
     call; second call is a no-op (no duplicate insertion); no
     other lines are modified.
36. **Projector idempotency test** — call `Project()` twice with
    no graph changes; second call produces byte-identical output
    (or no-write if the content-hash matches — same pattern as
    `next-noop-writes`).
37. **Surfaces.yaml round-trip test** — override schema parses,
    invalid schemas produce file:line errors.
38. **Merge-driver test** — simulate a conflict on
    `.hero/SNAPSHOT.md`; the driver resolves to local-graph
    regeneration.
39. **MCP compact / expand test** — verify the two-tier response
    pattern works for `hero_snapshot`.
40. **Archive trigger evaluator table-test** — exhaust the
    milestone / manual / staleness-cutoff decision rules across
    realistic archive-list / clock / event-stream inputs; assert
    correct `[]TriggerHit` shape. Includes the
    "milestone-fires-suppresses-staleness" case.
41. **Archive idempotency test** — fire two same-day staleness
    triggers; second one is a no-op and produces no new file.
42. **Archive immutability test** — call `Project()` after an
    archive has been written; assert the archive's bytes and
    mtime do not change.
43. **Archive frontmatter round-trip test** — write an archive,
    read it back, assert `snapshot_date / trigger / label /
    git_commit / projector_version` parse correctly.
44. **Retention test** — `last-N` policy deletes the right
    archives; `all` is a no-op; `none` clears everything.
45. **MCP `at:` + `history:` test** — `at: <date>` returns the
    archive body; `history: true` returns the enumerated list;
    `archive: true` writes a manual archive and returns the
    record.
46. **Serve archive view test** — 200 on `/project/snapshots/<date>`
    for an existing archive; 404 for a missing one; archive
    template renders the frontmatter header.
47. **Archive isolation tests** — one per invariant:
    a. Search index excludes `.hero/snapshots/`: index a workspace
       with archives, run `hero search "<term from archive>"`,
       assert no archive paths appear; re-run with
       `--include-history`, assert they appear.
    b. Archive frontmatter carries `historical: true` and
       `not_current: true` on every write; absence fails
       validation.
    c. Banner line is present at the top of every archive body;
       the writer refuses to emit an archive without it.
    d. `/resume` and `/prime` cold-start bundles do not contain
       archive body bytes (assert via skill-output capture).
    e. `/project` home renders the timeline strip (date + trigger
       + label only) but never archive body content.
    f. `auto-knowledge-capture` and `note-capture` skip
       `.hero/snapshots/` (assert via dry-run output).
48. **Detect CLI test** — `hero snapshot detect` returns the
    inferred surface list as text; `--explain` enriches each
    entry with signal rationale.

## Boundaries

- **Not** introducing a new spec type. Surfaces are inferred;
  the optional `.hero/surfaces.yaml` override layer is a peer
  YAML, not a spec.
- **Not** changing existing spec statuses, lifecycles, or
  frontmatter schemas beyond adding the optional `surface:` and
  `release_target:` fields. `release_target:` is also accepted
  on initiative frontmatter (cascades to child specs).
- **Not** rewriting NEXT.md, QUEUE.md, `hero status`, or any
  serve home. Snapshot composes with them; the projector writes
  exactly one new line (the SNAPSHOT pointer) into NEXT.md /
  AGENTS.md and nothing else.
- **Not** auto-injecting snapshot content into any session by
  default — or ever. The earlier `auto_context.include_snapshot`
  proposal is removed. Discovery via the pointer line and pull
  via explicit MCP / CLI / `/resume` / `/prime` covers the need.
- **Not** writing per-surface ownership / RACI tooling. The
  `owner:` field in the override file is a freeform string for
  v1 display only; no validation, no team-coordination glue.
- **Not** rendering surface-specific charts or burndowns. Stage
  + counts only in v1; visualizations belong in `hero pulse` or
  the serve Work home.
- **Not** requiring `.hero/surfaces.yaml` to exist. Inference
  produces a working surface list with no user authoring; the
  override file is created lazily, only when inference is wrong.
- **Not** federating snapshot across peer repos in v1. Each
  repo's snapshot is local to its own graph. Cross-repo rollup
  belongs to the in-flight `cross-repo-peering` initiative.
- **Not** changing the existing `spec-type-registry` design.
  Snapshot reads what the registry produces; the registry is
  unaware of surfaces.
- **Not** firing archive writes on a fixed interval. Milestones
  and manual invocations are the primary triggers; the
  staleness-cutoff is a safety net for quiet projects, not a
  schedule.
- **Not** demanding a single field for release modeling. The
  resolution chain accepts explicit frontmatter, tracker
  records, or git-tag heuristics; surfaces with no signal hide
  the column gracefully and emit a footnote.
- **Not** treating archives as indexable peers of specs or
  knowledge. Archives are historical trajectory data and are
  excluded from the default search index, the cold-start
  bundles, the auto-capture subsystems, and the default serve
  listings. They are reachable only through the explicit
  history-querying surfaces enumerated in "Archive containment."
- **Not** re-projecting or auto-updating archives. Archives are
  immutable once written. The only mutation paths are initial
  write and explicit `rm`.
- **Not** compressing or sub-directory-archiving older snapshots
  in v1. Plain markdown files at the top level of
  `.hero/snapshots/`. If size becomes a concern, compression is a
  v2 follow-up.
- **Not** building a rich diff UI for archives in v1. CLI text
  diff and read-only archive view ship; the inline "diff vs N
  weeks ago" comparison view in serve is **v2-deferred**.
- **Not** federating archives across peer repos. Same posture as
  the live snapshot — cross-repo trajectory rollup is owned by
  `cross-repo-peering`.
- **Not** indexing archive bodies into the live spec graph.
  Archives are not nodes in the relationship graph; they are
  historical artifacts, not first-class work-tracking entities.
  Archives are also excluded from the default search index
  (see "Archive containment" for the full isolation contract);
  the only opt-in is `hero search --include-history`.

## Acceptance Criteria

- WHEN the projector runs, THE SYSTEM SHALL detect candidate
  surfaces from repo structure and package manifests without
  requiring `.hero/surfaces.yaml` to exist.
- WHERE `.hero/surfaces.yaml` declares an override, THE SYSTEM
  SHALL apply it on top of the detected surface set (renames,
  ignores, additions, and per-surface field overrides).
- WHEN the user runs `hero snapshot detect --explain`, THE
  SYSTEM SHALL print each detected surface with the signals
  that triggered detection.
- WHEN `.hero/surfaces.yaml` is present, THE SYSTEM SHALL parse
  it as an override layer and surface file:line errors for
  malformed entries; absence is not an error.
- THE SYSTEM SHALL emit `.hero/SNAPSHOT.md` containing, in order:
  header, mission one-liner, timestamp, surfaces table, active
  initiatives, recent completions, next up, risks & blockers,
  snapshot health.
- WHEN a Stop hook fires, THE SYSTEM SHALL total-rewrite
  `.hero/SNAPSHOT.md` from the live graph after rewriting NEXT.md.
- WHEN a git merge produces a conflict on `.hero/SNAPSHOT.md`, THE
  SYSTEM SHALL resolve via the `hero-snapshot` merge driver by
  regenerating from the local graph.
- THE SYSTEM SHALL derive each surface's stage from its specs,
  release-target completions, and last-touched timestamp per the
  six-stage taxonomy.
- WHERE a surface declares an explicit `stage:` in the
  `.hero/surfaces.yaml` override layer, THE SYSTEM SHALL use the
  declared stage AND render a `pinned` badge in the serve view.
- THE SYSTEM SHALL resolve a spec's release target by checking,
  in order: explicit `release_target:` frontmatter on the spec,
  then on the parent initiative, then the configured tracker
  integration (Jira fixVersion / GitHub milestone / Linear
  cycle), then the git-tag heuristic.
- IF no release signal resolves for any spec in a surface, THEN
  THE SYSTEM SHALL hide the "% to next milestone" column for
  that surface and surface a "no release model declared"
  footnote on the snapshot.
- WHEN one or more specs lack a `surface:` frontmatter field, THE
  SYSTEM SHALL render an `(unassigned)` row in the surfaces table
  with the count and a hint to run `hero snapshot assign`.
- WHEN `hero snapshot` is invoked on the CLI, THE SYSTEM SHALL
  print the rendered markdown to stdout in under 500ms on a
  workspace with up to 500 specs.
- WHEN `hero snapshot --json` is invoked, THE SYSTEM SHALL emit a
  structured JSON document containing all sections.
- WHEN the `hero_snapshot` MCP tool is invoked with `compact:
  true`, THE SYSTEM SHALL return a summary + ref_id; subsequent
  `hero_expand` SHALL return the full content.
- WHEN a browser opens `/project`, THE SYSTEM SHALL return HTTP
  200 and render the surfaces table, active initiatives, recent
  completions, next-up, risks, and snapshot-health sections.
- WHEN a browser opens `/project/surface/<id>` for a declared
  surface, THE SYSTEM SHALL return HTTP 200 and render the
  per-surface drill-in (specs declaring this surface, initiatives,
  recent commits to declared paths, scoped blockers).
- WHEN a browser opens `/project/surface/<id>` for an undeclared
  surface, THE SYSTEM SHALL return HTTP 404 via the existing
  not-found template.
- WHEN the snapshot projector emits, THE SYSTEM SHALL ensure
  `.hero/NEXT.md` and `AGENTS.md` contain a one-line pointer
  to `SNAPSHOT.md`, inserted exactly once and never duplicated.
- THE SYSTEM SHALL NOT inject snapshot content into any session
  context by default.
- THE SYSTEM SHALL skip rewriting `.hero/SNAPSHOT.md` when the
  newly computed content hash equals the file's existing
  content hash (no-op write).
- WHEN a spec declares `surface:` not present in the merged
  inferred + override surface set, THE SYSTEM SHALL emit a
  structural lint warning (not error) naming the unknown id and
  listing known surfaces.
- THE SYSTEM SHALL re-project `.hero/SNAPSHOT.md` when
  `.hero/surfaces.yaml` is created, modified, or deleted,
  observed via the existing hero serve file watcher.
- THE SYSTEM SHALL include a "snapshot health" footer with
  inferred-surface count, override-applied count, specs-covered
  ratio, and last override-file edit timestamp (when present).
- WHEN `hero snapshot assign` is invoked, THE SYSTEM SHALL list
  unassigned specs and accept a surface id per spec, rewriting
  the spec's frontmatter to add the assigned `surface:` field.
- WHEN the snapshot projector runs AND
  `now − last_archive > snapshot.archive.staleness_cutoff` AND
  no milestone trigger fires on the same projector run, THE
  SYSTEM SHALL write a staleness-safety archive labeled
  `auto-staleness` to `.hero/snapshots/YYYY-MM-DD.md`
  containing the same rendered bytes as the live
  `.hero/SNAPSHOT.md` plus an archive frontmatter block
  (`snapshot_date`, `trigger`, `label`, `git_commit`,
  `projector_version`, `historical: true`, `not_current: true`).
- WHERE `snapshot.archive.staleness_cutoff` is `off`, THE
  SYSTEM SHALL only archive on milestone or manual triggers.
- WHEN an `initiative` spec transitions to `status: completed`
  AND `snapshot.archive.milestones` is enabled, THE SYSTEM
  SHALL write a milestone archive with `trigger: milestone`
  and `label: <initiative-slug>`.
- WHEN a git tag matching the configured
  `snapshot.archive.release_tag_pattern` is created AND
  `snapshot.archive.milestones` is enabled, THE SYSTEM SHALL
  write a milestone archive with `trigger: milestone` and
  `label: <tag-name>`.
- WHEN the user runs `hero snapshot archive` (with or without
  `--label`), THE SYSTEM SHALL write an archive with
  `trigger: manual` regardless of staleness-cutoff or
  last-archive timestamp.
- WHEN the `hero_snapshot` MCP tool is invoked with
  `archive: true`, THE SYSTEM SHALL write a manual archive and
  return the archive record (`{date, trigger, label,
  git_commit, path}`).
- THE SYSTEM SHALL render archive files as read-only and
  SHALL NOT modify, re-project, or overwrite an archive after
  its initial write; same-day re-triggers without an explicit
  `--label` SHALL be no-ops.
- WHERE `snapshot.archive.retention` is `last-N`, THE SYSTEM
  SHALL delete archives older than the N most-recent at the end
  of each successful archive write, preserving the newest N by
  `snapshot_date`.
- WHEN `hero snapshot history` is invoked, THE SYSTEM SHALL
  list all archives newest-first with date, trigger, label,
  git_commit; `--json` SHALL emit the same data as a structured
  list.
- WHEN `hero snapshot show <date>` is invoked for an existing
  archive, THE SYSTEM SHALL print its rendered markdown to
  stdout; for a missing archive, exit non-zero with a clear
  error.
- WHEN `hero snapshot diff <date-a> <date-b>` is invoked for two
  existing archives, THE SYSTEM SHALL emit a text diff between
  them; `latest` and `live` SHALL be valid synonyms.
- WHEN the `hero_snapshot` MCP tool is invoked with `at:
  <date>`, THE SYSTEM SHALL return the matching archive's
  rendered body; with `at: latest` (default) or unset, THE
  SYSTEM SHALL return the live `.hero/SNAPSHOT.md` body.
- WHEN the `hero_snapshot` MCP tool is invoked with
  `history: true`, THE SYSTEM SHALL return the enumerated
  archive list (mutually exclusive with `at`).
- WHEN a browser opens `/project` AND one or more archives
  exist, THE SYSTEM SHALL render a timeline strip listing
  archives newest-first with clickable entries.
- WHEN a browser opens `/project/snapshots/<date>` for an
  existing archive, THE SYSTEM SHALL return HTTP 200 and render
  the archive in a read-only view with a header showing
  trigger / label / git_commit.
- WHEN a browser opens `/project/snapshots/<date>` for a
  non-existent archive, THE SYSTEM SHALL return HTTP 404 via
  the existing not-found template.
- THE SYSTEM SHALL exclude `.hero/snapshots/` from the default
  Hero search index.
- WHEN `hero search` runs without `--include-history`, THE
  SYSTEM SHALL exclude archive bodies from results.
- THE SYSTEM SHALL prepend a "historical archive captured
  {date}" banner line to every archive file body and refuse to
  suppress it.
- THE SYSTEM SHALL set `historical: true` and `not_current:
  true` in every archive file's frontmatter.
- THE SYSTEM SHALL NOT include archive content in `/resume` or
  `/prime` cold-start bundles; only the live `SNAPSHOT.md` is
  eligible.
- THE SYSTEM SHALL skip `.hero/snapshots/` in
  `auto-knowledge-capture`, `note-capture`, and the auto-memory
  subsystems.
- THE SYSTEM SHALL render archive bodies in hero serve only at
  `/project/snapshots/<date>` and SHALL NOT include archive
  body content in listing or homepage views.

## Risks

- **Inference miss / false positive.** Detection might miss a
  real surface (e.g. an unusual layout) or invent a phantom
  surface (e.g. a `scratch/` folder that's not really
  shipping). Mitigation: the override layer's `additions:` and
  `ignore:` fields fix both. `hero snapshot detect --explain`
  makes the failure mode debuggable — the user sees exactly
  which signals fired (or didn't). Inference confidence is
  reported per surface in the snapshot health footer.
- **Stage derivation heuristic is imperfect.** A surface in
  active polish may look `maturing` to the heuristic but feel
  `shipping-v1` to the user. Mitigation: pin via `stage:` in
  the override file; the rendered badge makes pinned-vs-derived
  visible.
- **Snapshot size growth.** As surfaces accumulate, the file
  grows. Mitigation: the surfaces table is bounded by surface
  count, not spec count; the "Recently completed" and "Next up"
  sections are capped at N items each; the file is a rollup, not
  a firehose.
- **Release model absent on real projects.** Some projects
  don't declare release scope anywhere — no frontmatter, no
  tracker, no tags. Mitigation: the "% to next milestone"
  column gracefully disappears with a polite footnote rather
  than blocking the snapshot. The footnote text instructs the
  user how to add a model when they're ready.
- **Archive isolation leak.** A future contributor wires
  archive content into a default-discovery path (search,
  cold-start, knowledge capture) and historical state poisons
  live context. Mitigation: the isolation invariants are
  documented as a hard contract; tests (#47a-f) cover each
  invariant; `hero check` validates the search-exclude config;
  the banner line ensures even a leaked archive body announces
  itself as historical.
- **Merge-driver setup fragility.** New users may not have
  `.gitattributes` configured for the `hero-snapshot` driver.
  Mitigation: `hero install` registers the driver; `hero check`
  detects missing registration and warns; the existing
  `hero-next` install path already handles this for NEXT and
  we extend it.
- **Two projectors firing per Stop hook.** NEXT + SNAPSHOT could
  combine to noticeable delay. Mitigation: target combined
  projection under 250ms on a 500-spec workspace; share a
  single graph read transaction; profile and optimize before
  surface inference adds material cost.
- **Surface schema lock-in.** Once `surfaces.yaml` is in many
  projects, schema changes are breaking. Mitigation: schema
  `version: 1` field is mandatory; loader treats unknown future
  fields as warnings, not errors; reserve `version: 2` for
  breaking changes.
- **Composition with the in-flight surface-polish-v2 spec.**
  v2 is adding per-item detail routes and reshaping the top nav.
  Snapshot adds a new `/project` nav entry. Coordinate the nav
  template edits to land in one PR or as immediate follow-up so
  the surface polish doesn't have to re-render.
- **`spec-type-registry` overlap.** The registry doesn't know
  about surfaces. If a future refactor folds surfaces into the
  registry, the `surface:` frontmatter field stays valid; only
  the storage location shifts. Forward-compatible by design.
- **Mission test failure mode.** If snapshot generation breaks
  silently, fresh sessions read a stale rollup and start *less*
  smart, not more. Mitigation: the projector logs to
  `events.log`; `hero check` validates last-projection
  timestamp; serve home shows a "stale" badge if the file's
  timestamp is older than the last graph mutation.
- **Archive proliferation.** Weekly archives over a multi-year
  project means ~50 files/year per repo. Markdown is cheap, but
  serve-home rendering of a long timeline can become noisy.
  Mitigation: timeline strip caps default render at the most
  recent 12 archives with a "show all" expander; retention
  config (`last-N`) is available for users who want to bound
  on disk too.
- **Same-day re-fire ambiguity.** Two milestone events on the
  same day (e.g. an initiative completion AND a release tag)
  could fight over the same filename. Mitigation: first write
  wins; same-day re-fires are no-ops unless an explicit
  `--label` is provided, in which case `--<slug>` disambiguates
  via the filename suffix.
- **Projector-version drift.** If the projector render shape
  changes, old archives reflect an older shape. Mitigation:
  the `projector_version` frontmatter field records the
  rendering version at write time; readers can detect
  mismatched-version archives and render with a "rendered with
  older projector" badge.
- **Archive write failure on hook path.** A failed archive
  write must not block the live `SNAPSHOT.md` projection or
  the Stop hook. Mitigation: archive writes are wrapped in
  recoverable error handling; failures log to `events.log` and
  surface in `hero check`; the live projection always completes
  even when archive writing fails.
- **Reports vs snapshots category confusion.** Users may not
  immediately understand why `.hero/reports/snapshot-*.md`
  (executive-report skill output) and `.hero/snapshots/*.md`
  (this spec) coexist. Mitigation: the docs entry explicitly
  contrasts them — *reports are authored; snapshots are
  projected* — and the executive-report skill description will
  be updated (out of scope here) to point at this distinction.
- **Staleness-cutoff surprise on first install.** A fresh
  install with `staleness_cutoff: monthly` would, on first
  projector run, find `last_archive_date` unset and could
  immediately write a safety-net archive. Mitigation: the
  cutoff check uses `last_archive_date` defaulting to *now* on
  first run — first archives are milestone- or manually
  triggered, not retroactive. The cutoff only fires after the
  first projector run sets the baseline.

## Validation

- `go build ./... && go test ./...` passes.
- Manual: on a fresh checkout (no `.hero/surfaces.yaml`), run
  `hero snapshot detect --explain` — verify the inferred
  surface list covers `core`, `serve`, `mcp`, `docs`,
  `landing`, and each `domains/<pack>` with sensible
  rationales.
- Manual: run `hero snapshot` — verify the rendered markdown
  shows the inferred surfaces, the `(unassigned)` row if any,
  recent completions over the last 14 days, next-up across
  surfaces, and the health footer.
- Manual: `hero snapshot --json | jq .` — confirm structured
  shape.
- Manual: open `/project` in hero serve — confirm the table
  renders, surfaces are clickable, and a known surface
  drills into a detail page.
- Manual: create `.hero/surfaces.yaml` with an `ignore: [scratch]`
  entry; confirm the file watcher re-projects and the surface
  drops from the list.
- Manual: verify `.hero/NEXT.md` and `AGENTS.md` contain the
  pointer line *"Project shape: see [SNAPSHOT.md]..."* after
  the projector runs; verify a second projector run does not
  duplicate the line.
- Manual: confirm no snapshot content is auto-injected into a
  fresh session (only the pointer line in NEXT/AGENTS); the
  session must explicitly read SNAPSHOT.md or call
  `hero_snapshot` to see the body.
- Manual: simulate a merge conflict by editing `.hero/SNAPSHOT.md`
  on two branches; confirm the merge driver resolves cleanly.
- Manual: verify release resolution — create one spec with
  explicit `release_target: v1`, one with no signal at all,
  configure a tracker fixVersion for a third; confirm the
  snapshot rolls each up via the correct chain step.
- Manual: with no release signal on any spec, confirm the
  surface's "% to next milestone" column hides and the
  footnote appears.
- Validate that `hero check` flags unknown `surface:`
  declarations, missing isolation flags on archives, and
  archives without the banner line.
- Coverage: `internal/snapshot/` ≥ 80% test coverage on
  stage-derivation and rollup logic.
- Manual: set `snapshot.archive.staleness_cutoff: weekly` in
  `hero.json`, fast-forward last-archive-date by 8 days, run
  `hero snapshot --project`; confirm a new file appears at
  `.hero/snapshots/<today>.md` with `trigger: staleness` and
  `label: auto-staleness`.
- Manual: with milestones enabled, flip an initiative spec to
  `completed`, confirm an archive appears with
  `trigger: milestone` and `label: <initiative-slug>`. Confirm
  the same run does NOT also write a staleness archive (the
  same-run suppression invariant).
- Manual: run `hero snapshot archive --label v1-ship`, confirm
  the archive lands at `.hero/snapshots/<today>--v1-ship.md`.
- Manual: re-run `hero snapshot --project` immediately after an
  archive write; confirm the archive's bytes and mtime are
  unchanged (immutability).
- Manual: open an archive and verify the historical banner is
  at the top of the body and frontmatter contains
  `historical: true` + `not_current: true`.
- Manual: run `hero search "<term in archive>"` — confirm
  archive paths are absent from results; re-run with
  `--include-history` — confirm they appear.
- Manual: open `/project` and verify the timeline strip lists
  archives newest-first with date / trigger / label only (no
  body content); click an entry and verify
  `/project/snapshots/<date>` renders the archive in read-only
  mode with the banner visible.
- Manual: confirm `/resume` and `/prime` cold-start bundles
  do not contain archive body bytes; only `SNAPSHOT.md` is
  eligible when explicitly enabled.
- Manual: `hero snapshot history` lists the new archives;
  `hero snapshot show <date>` renders one; `hero snapshot diff
  <date-a> <date-b>` produces a sensible text diff.
- Manual: set `snapshot.archive.retention: last-2`, archive
  three times, confirm only the two most-recent files remain
  on disk.

## Open Questions

The following points are *deliberately deferred* — not blockers
for v1, but worth flagging so they don't slip silently.

Resolved by the five revisions and no longer questions:
~~Default interval cadence in v2~~ (interval removed),
~~Cross-archive search default~~ (decided: exclude from default,
opt-in via `--include-history`), ~~Auto-inject opt-in design~~
(removed entirely).

1. **Should `hero pulse` consume archives as input?** Pulse
   generates weekly narrative; archives capture point-in-time
   structural state. A future pulse implementation could diff
   the most-recent archive against the live snapshot to anchor
   its narrative. Out of scope for this spec; flagged for the
   pulse owner. (Note: pulse must respect the same isolation
   invariants — it can *read* archives via the explicit
   history-query surfaces, not via default discovery paths.)
2. **Initiative-completion archive timing.** Should the milestone
   archive be written *before* or *after* the initiative status
   flip propagates through the projector? Writing after means
   the archive reflects the completed state; writing before
   means it reflects the moment-of-completion. Currently
   specified as *after* (post-projection); revisit if users find
   the resulting archive misleading.
3. **Diff format in v1.** Plain text diff is specified; a
   structured "what changed about the project shape" diff
   (added surface, surface stage moved, X specs completed since
   last archive) would be more useful but requires comparing
   the projected structures, not the rendered markdown. v2
   follow-up: structured-diff renderer.
4. **What happens if the live `SNAPSHOT.md` is missing when an
   archive trigger fires?** Specified: the projector renders the
   current state in-memory and writes both the live file and
   the archive. Should the archive write be gated on the live
   write succeeding first? Currently yes; revisit if that
   ordering causes archive gaps.
5. **Inference confidence threshold.** Detection emits a
   confidence score per candidate; should low-confidence
   candidates (e.g. < 0.3) be omitted from the default render
   and surfaced only via `--explain`? v1 ships with no
   threshold — every candidate is rendered. Revisit if false
   positives are common on real-world layouts.
6. **Tracker fixVersion adapter scope.** Release resolution step
   2 reads from tracker integrations that already sync to Hero.
   For trackers Hero does not yet integrate with (e.g. custom
   internal systems), the chain falls through to step 3 (git
   tags). Whether to add a generic "tracker-agnostic release
   field" config is deferred until a project actually requests
   it.

## Kickoff

Project-shape rollup with time-snapped trajectory:
`.hero/SNAPSHOT.md` (live, projected) plus
`.hero/snapshots/YYYY-MM-DD.md` (immutable, isolated archives —
milestone + manual triggers, staleness-cutoff safety net).
Surfaces are *inferred* from repo shape; `.hero/surfaces.yaml`
is an optional override layer. Release scope resolves through
a priority chain (spec frontmatter → tracker → git tags →
graceful hide). Snapshot is discovered via a pointer line in
NEXT/AGENTS — never auto-injected. Archives are excluded from
default search, cold-start bundles, and listings.

**Status:** completed — landed in 12 phases on 2026-05-18.
Combined `hero next checkpoint` warm time on this repo's 234-spec
graph: ~190ms, well under the 250ms target.

**Pick up at:** future v2 work — structured shape-diff renderer,
inline "diff vs N weeks ago" serve view, compressed archive
storage. All v2-deferred. Implementation lives under
`internal/snapshot/` (detect, surfaces, release, stage, rollup,
render_markdown, render_json, pointers, archive,
archive_triggers, retention, diff, projector), CLI under
`internal/cli/snapshot.go`, MCP tool in
`internal/serve/mcp_tools_snapshot.go`, serve home in
`internal/serve/pages/project/`. The projector fires from
`hero next checkpoint` after the NEXT/handoff projections, so
SNAPSHOT.md regenerates on every Stop hook automatically.

→ `.hero/planning/features/project-snapshot/spec.md`

**Files:** `internal/snapshot/`,
`internal/handoff/handoff.go` (pattern reference),
`internal/cli/checkpoint.go` (Stop-hook integration point),
`.hero/surfaces.yaml` (new — optional override layer),
`.hero/snapshots/` (new dir for archives, excluded from index),
`internal/spec/spec.go` (add `Surface` + `ReleaseTarget`
fields), `internal/index/config.go` (add archive path to
exclude_paths), `internal/serve/pages/now/` (template parallel
for `/project`).

**Skip:** registering `surface` as a new spec type. Authoring a
full `surfaces.yaml` up front — inference handles it.
Auto-injecting snapshot bytes anywhere — discovery via the
pointer line only. Treating archives as indexable peers — they
are strictly isolated historical artifacts. Cross-repo
surface rollup — owned by `cross-repo-peering`. Burndown
visualizations — owned by `hero pulse`. Compressed archive
storage, structured shape-diff renderer, and inline serve
"diff vs N weeks ago" view — all v2-deferred.
