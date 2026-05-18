---
type: decision
status: accepted
title: Snapshot Architecture — Projector, Surfaces, Lifecycle, Archives, Pointer Discovery
created: 2026-05-18
shipped-in: v0.10.0
shipped-commits:
  - 5110b1a
relates-to:
  - next-as-projection-architecture
  - project-snapshot
tags: [decision, post-hoc, architecture, snapshot, projection, surfaces, archives]
---

# Snapshot Architecture — Projector, Surfaces, Lifecycle, Archives, Pointer Discovery

## Kickoff

Post-hoc decision record for `hero snapshot` (v0.10.0). Captures *why*
the snapshot subsystem is shaped the way it is — projector framework,
inferred surfaces with override layer, six-stage lifecycle, three-trigger
archive model, discoverable pointer (never auto-injected), and strict
archive isolation. The original delivery spec
(`.hero/specs/project-snapshot/spec.md`, 1,759 lines) carries the full
design narrative; this record exists so a future maintainer extending or
modifying the subsystem doesn't have to reverse-engineer the choices.

**Status:** accepted — design shipped in commit `5110b1a` on 2026-05-18.

**Pick up at:** read this record before adding a new surface-detection
signal, a new archive trigger, a new lifecycle stage, or any change that
would push snapshot content into session context automatically. Cross-
check the original delivery spec for full rationale on individual
sub-decisions.

→ `.hero/planning/decisions/snapshot-architecture/spec.md`

**Files:** `internal/snapshot/projector.go`,
`internal/snapshot/detect.go`, `internal/snapshot/surfaces.go`,
`internal/snapshot/stage.go`, `internal/snapshot/archive.go`,
`internal/snapshot/pointers.go`, `internal/cli/snapshot.go`,
`.hero/specs/project-snapshot/spec.md`

**Skip:** treating archives as searchable peers of specs/knowledge,
auto-injecting snapshot bytes into `/resume` or `/prime` by default,
adding a fixed-interval archive trigger (replaced by staleness cutoff),
demanding a single `release_target:` field (resolution chain handles
heterogeneous projects).

## Context

By mid-v0.10 the project graph held everything a fresh agent or new
teammate needed to answer "where are we across the board" — in-flight
specs, completions, initiatives, blockers, ranked queue — but no
artifact rolled it up *along the surface axis* ("the CLI," "hero serve,"
"the docs site"). The senior dev's mental model — "core is shipped, serve
is shipping v1, docs is scaffolded, landing just landed, mcp is mature,
domain packs vary" — existed nowhere on disk. A fresh session had to
open mission.md, NEXT.md, QUEUE.md (49KB), run `hero status` (200+ flat
entries), scan `internal/serve/pages/` and `web/` to *guess* the surface
list, and finally ask the senior dev. 10+ minutes, missed the cross-
cutting shape, and Step 6 (ask the senior dev) was exactly what Hero is
built to make unnecessary.

Forces:

- **The cold-start surface for project shape was missing.** NEXT.md
  covers per-user/per-session handoff; mission.md covers doctrine;
  QUEUE.md is a firehose. None answer "what does this project consist
  of and where is each piece in its lifecycle."
- **The graph was already authoritative.** Adding a hand-maintained
  document would have asked the senior dev to keep two truths in sync —
  the exact failure mode `next-as-projection` had just resolved for the
  handoff artifact.
- **Surfaces were the only new axis.** Every other input — specs,
  initiatives, queue, blockers, recent commits — was already in the
  graph. The new contribution was the surface dimension and the rollup
  over it.
- **Cross-cutting trajectory was missing too.** A live mirror answers
  "now"; nothing answered "how did we get here." Without that, "session
  starts smarter than the last one ended" caps at point-in-time state
  with no historical context.
- **A projector framework had just landed.** `next-as-projection`
  (shipped one commit prior) gave Hero a working pattern for graph-
  projected, merge-driver-managed, Stop-hook-rewritten artifacts. The
  cost of a second projector was now marginal.

## Decision

A projector subsystem (`internal/snapshot/`) that emits `.hero/SNAPSHOT.md`
on every Stop hook, paired with a `.hero/snapshots/YYYY-MM-DD.md`
archive trail under strict isolation invariants. The design has six
load-bearing pieces:

### 1. Projector framework, not a hand-maintained document

`internal/snapshot/projector.go` exposes a single `Project()` entry
point that (a) loads specs and the optional override, (b) discovers
shipped git tags, (c) builds the snapshot, (d) renders markdown, (e)
writes `SNAPSHOT.md` (skipped if content hash matches), (f) ensures the
pointer line lives in NEXT.md / AGENTS.md, (g) evaluates archive
triggers, (h) applies retention. All steps after the SNAPSHOT.md write
are best-effort — pointer / archive failures log but do not fail the
projector. The snapshot is **tracked in git** as project state.

### 2. Inference-first surfaces, override-layer for human intent

Surfaces are *detected* every projector run from repo structure
(top-level dirs like `internal/`, `cmd/`, `web/<surface>/`,
`domains/<pack>/`), package manifests (`go.mod`, `Cargo.toml`,
`package.json`, `mkdocs.yml`, `wrangler.toml`, `pyproject.toml`), and
`hero.json` paths. Each candidate carries a confidence score and a
rationale of which signals fired. `.hero/surfaces.yaml` is an **optional
override layer** — never the source of truth — carrying only deltas
(renames, ignores, additions, per-surface field overrides). The merge
rule: `detected ∪ override-added − override-ignored`, with override
fields winning per surface. `hero snapshot detect --explain` prints
each detected surface with the signals that fired so the user can debug
inference quickly.

### 3. Six-stage lifecycle taxonomy

Pure-function derivation in `internal/snapshot/stage.go`:
`concept → scaffolded → building → shipping-v1 → shipped → maturing`.
Rules read SpecsByStatus, ReleaseDone/ReleaseTotal, LastTouched,
HasShippedTag. Graceful degradation when no release signal resolves
(any in-flight → `building`; shipped-tag-but-no-in-flight → `maturing`;
otherwise `scaffolded`). Overridable per-surface via `stage:` in the
override layer; pinned stages render with a `pinned` badge.

### 4. Layered release-target resolution

Priority chain per spec: explicit `release_target:` frontmatter → parent
initiative frontmatter (cascades) → tracker integration
(Jira fixVersion / GitHub milestone / Linear cycle) → git-tag heuristic
on the configurable pattern (default `v[0-9]*`) → **none, hide the
column for that surface and emit a one-line footnote**. A spec without a
resolved release target is *excluded* from rollup counts (not bucketed
into a phantom v1).

### 5. Three-trigger archive model

`internal/snapshot/archive.go` writes byte-for-byte copies of the live
`SNAPSHOT.md` (plus prepended archive frontmatter and a mandatory
banner line) to `.hero/snapshots/YYYY-MM-DD[--<slug>].md`. Three triggers:

- **Milestone** — release tag matching the configured pattern, initiative
  spec flipping to `completed`, sprint completion (soft-gated on sprint
  integration).
- **Manual** — `hero snapshot archive [--label <slug>]` or
  `hero_snapshot archive: true`.
- **Staleness safety net** — when `now − last_archive_date >
  snapshot.archive.staleness_cutoff` (default `monthly`) AND no
  milestone fired on the same projector run, write an `auto-staleness`
  archive. Suppressed by any concurrent milestone trigger so the two
  never double-archive on the same day.

Archives are **immutable** — the projector never re-reads or rewrites
them after initial write; same-day re-fires without explicit `--label`
are no-ops. Default retention is `keep all`; optional `last-N` and `none`
exist. First-run safety: `newReleaseTags` and `newlyCompletedInitiatives`
both return nothing when no archive exists, so installing the projector
on a repo with years of historical git tags does **not** retroactively
write one archive per tag.

### 6. Six isolation invariants for archives

Archives carry `historical: true` + `not_current: true` frontmatter and a
mandatory banner line ("This is a point-in-time snapshot, not current
state"). `Read()` refuses tampered files. `.hero/snapshots/` is excluded
from the default search index, skipped by `auto-knowledge-capture` /
`note-capture` / auto-memory, omitted from `/resume` and `/prime`
cold-start bundles, and *only* rendered in serve at
`/project/snapshots/<date>` (never in listings or homepages). Archives
are reachable only through six intentional history-query surfaces
enumerated in the delivery spec.

### 7. Discoverable, never pushed: the pointer model

The projector writes one extra line into `.hero/NEXT.md` and
`AGENTS.md` — exactly:
`Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).` — between
managed markers (`<!-- >>> hero snapshot pointer (managed) >>> -->` /
`<!-- <<< hero snapshot pointer (managed) <<< -->`). The writer is
idempotent (skips when markers or the pointer line already appear).
Snapshot bytes are **never auto-injected** into session context.
Consumers pull via the rendered file, the `hero_snapshot` MCP tool, the
`hero snapshot` CLI, the `/project` serve home, or — opt-in — `/resume`
and `/prime`.

### 8. Five access surfaces

`.hero/SNAPSHOT.md` (file, tracked, GitHub-rendered) ·
`hero_snapshot` (MCP tool with `compact / section / surface / at /
history / archive / label` flags) ·
`hero snapshot` (CLI: bare invocation + `--json / --section / --project`
plus `detect / assign / archive / history / show / diff` subcommands) ·
`/project` (serve home, peer of Now / Work / Knowledge / Agents /
People; surface detail at `/project/surface/<id>`; archive read-only
view at `/project/snapshots/<date>`) ·
the pointer line in NEXT/AGENTS as the discoverability anchor.

### 9. Stop-hook integration on top of next-as-projection

`projectSnapshot()` in `internal/cli/checkpoint.go` runs after the
NEXT / user-handoff / local-state writes in the same Stop-hook firing.
A single graph read transaction is shared so the combined warm
checkpoint stays well under budget (measured ~190ms on the hero repo's
234-spec graph against a 250ms target). `.hero/SNAPSHOT.md` also
inherits the `hero-next` merge driver in `.gitattributes`.

## Alternatives considered

### A1 — Hand-maintained `SNAPSHOT.md` (rejected — explicit in the delivery spec)

Treat snapshot like a `## Surfaces` section of mission.md: the senior
dev writes it; readers trust it; nothing auto-updates. Pro: zero
infrastructure. Con: this is exactly the failure mode `next-as-projection`
had just resolved for NEXT.md — drift between document and reality,
audit-invisible until somebody checks against git. The delivery spec
makes the parallel explicit ("Same machinery as next-as-projection") and
notes the projector pattern was already proven by the time snapshot
shipped. Rejected on the same grounds NEXT.md hand-editing was rejected.

### A2 — Declarative `surfaces.yaml` as the source of truth (rejected — explicit in the delivery spec, Sub-section 1)

Earlier drafts (referenced in the delivery spec) evaluated three options
for surface modeling: register surface as a spec type, derive purely
from directory structure, or declare in a YAML file. Option B
(declarative YAML) was chosen *in the earlier draft* on the grounds that
pure derivation is too lossy. The shipped design splits that conclusion:
*existence* (which surfaces are there) is high-confidence inference;
*intent* (what a surface IS) is the override layer's job. The shipped
spec says it directly: "Inference is high-confidence on the latter and
only fuzzy on the former. The merge model captures both: inference
handles detection; override carries human intent where it matters." A
zero-config project gets a working surface list the first time the
projector runs.

### A3 — Surface as a new spec type (rejected — explicit in the delivery spec, Sub-section 3)

Treat each surface as a `surface`-type spec with frontmatter for intent,
owner, paths, stage. Pro: uniform with the rest of the spec model;
inherits frontmatter validation, search, and rendering for free. Con:
the spec-type registry (in-flight in v0.10) covers nine canonical
work-tracking + knowledge types; surfaces are a *facet on existing
types*, not a type of work themselves. Adding a tenth type for surfaces
would create a spec with no acceptance criteria, no lifecycle, no
delivery — a category violation. Rejected; surfaces stay a peer concept
with their own (optional) declaration file, forward-compatible to a
registry record if the boundary ever shifts.

### A4 — Auto-inject snapshot into every session (rejected — explicit in the delivery spec)

Earlier drafts proposed `auto_context.include_snapshot` to push (some
fraction of) the snapshot into every fresh session's context window. Pro:
the cold-start surface is, by definition, valuable for cold starts. Con:
the snapshot is heavier than NEXT (full surfaces table, multiple rollups,
health footer), most sessions don't need the full shape, and auto-
injection ratchets context cost on every fresh session whether the
session uses the information or not. Rejected in favor of the
*discoverable* model: the pointer line in NEXT/AGENTS announces the
artifact exists; consumers pull when useful.

### A5 — Fixed-interval archive cadence (rejected — explicit in the delivery spec)

`snapshot.archive.interval: weekly | biweekly | monthly` would write an
archive on a clock schedule regardless of project activity. Pro: simple
mental model. Con: active projects already accumulate milestone archives
(release tags, initiative completions) and would double up; quiet
projects need *some* coverage but don't need a Tuesday tick. Rejected
in favor of the staleness-cutoff safety net which fires *only when no
other archive has been written in the cutoff window* — milestone-active
projects never hit it, quiet projects still get periodic trajectory data
without the user touching a knob, one default ships.

### A6 — Single `release_target:` field (rejected — explicit in the delivery spec, Sub-section 4a)

Originally the spec assumed every spec would carry an explicit
`release_target:` frontmatter field and the rollup column would simply
read it. The shipped design rejects this: many projects already encode
release scope in the tracker or in git tags, and demanding a single
canonical field would force every project to migrate its existing model.
The priority chain (frontmatter → tracker → tags → graceful hide) picks
up whatever signal the project already uses and quietly hides the
column when no signal resolves — a polite footnote instead of a broken
column.

### A7 — Co-locate archives under `.hero/reports/` (rejected — explicit in the delivery spec)

The hero-code peer repo established `.hero/reports/snapshot-YYYY-MM-DD.md`
as the location for executive-report output. Co-locating under
`.hero/reports/` would have given one folder for "things generated for
human consumption over time." Rejected because *reports are authored*
(hand-curated narrative, varying templates, audience-specific) while
*snapshots are projected* (same projector output, deterministic shape,
written by the system without prose decisions). Mixing them forces every
reader (and every list view) to filter by kind. Dedicated
`.hero/snapshots/` matches the precedent of `.hero/specs/` vs
`.hero/planning/` vs `.hero/knowledge/` — each top-level corpus folder
maps to one artifact category.

### A8 — Extend `hero_status` instead of adding `hero_snapshot` (rejected — explicit in the delivery spec, Sub-section 4)

Fold the surface-shape rollup into the existing `hero_status` MCP tool.
Rejected because `hero_status` returns a flat by-status grouping
(delivering / planning / completed); adding surface-shape would either
break the schema or balloon the response. The two compose:
`hero_status` answers "what's in flight," `hero_snapshot` answers
"what's the project shape."

### A9 — Treat archives as indexable peers of specs and knowledge (rejected — explicit in the delivery spec, "Archive containment")

Let archives flow into the default search index, `/resume` and `/prime`
cold-start bundles, auto-knowledge-capture, and the live `/project`
listings. Rejected because the worst-case outcome is exactly the failure
the delivery spec calls out: a six-month-old archive row claiming
"hero-serve: building" surfaces in `hero search` and the model treats it
as current state. The six isolation invariants exist specifically to
make that failure mode impossible.

## Consequences

### What this enables

- **Zero-config snapshot.** A fresh Hero project gets a useful surfaces
  list on first projector run with no human authoring. Override only
  when inference is wrong.
- **Surface awareness everywhere snapshot is consumed.** The new
  `surface:` frontmatter field on specs becomes a queryable facet for
  any future tool that wants to group by surface — not just snapshot.
- **Trajectory queries.** `hero snapshot diff <date> <date>`,
  `/project/snapshots/<date>`, and `hero_snapshot at: <date>` make
  "how has the project shape changed" answerable without manual
  bookkeeping.
- **Cross-projector reuse.** The pointer-line pattern
  (`pointers.EnsurePointer`) is a reusable shape any future
  discoverable-not-pushed artifact can adopt. Same for the
  trigger-evaluator framework and the isolation-invariant pattern.
- **Pre-release rollup works on heterogeneous projects.** The release-
  target priority chain means a project that uses *any one* of the four
  signal sources gets a working column; a project that uses none gets a
  polite footnote.

### What this locks in

- **Snapshot is always Stop-hook coupled.** The combined NEXT +
  SNAPSHOT projection runs on every assistant turn end. The ~250ms
  budget governs both projectors together — adding a third rollup that
  shares the Stop hook will need to fit inside the same window or
  trigger a re-architecture (e.g. async projection).
- **Six lifecycle stages are committed surface vocabulary.** Adding a
  seventh stage is a breaking change for any consumer that switches on
  the enum (templates, CLI output, override-file validation, future
  shape-diff tooling). The taxonomy was chosen empirically from the
  hero-engine repo's own surfaces; other repos may find it imperfect
  but the pin-via-override escape hatch is the intended remedy.
- **Archive isolation is contractual.** Every future contributor adding
  a discovery surface must check whether archives must be excluded.
  The six invariants are listed explicitly in the delivery spec; new
  default-discovery paths (a new search command, a new context
  injector, a new cold-start skill) need an explicit decision on
  whether they leak archive bodies and the answer is almost certainly
  "no, exclude them."
- **`SNAPSHOT.md` is tracked git state.** Branches diverge on it; the
  `hero-next` merge driver resolves cleanly for users who've run
  `hero install`. Users who haven't (fresh checkouts, CI) get standard
  conflict markers and rely on the next Stop-hook firing to self-heal.
- **`.hero/snapshots/` grows unbounded by default.** Markdown is cheap;
  default `retention: all` keeps the trajectory. Long-running projects
  will accumulate dozens of files per year. Retention knobs exist
  (`last-N`, `none`) but the default favors history over disk frugality.

### Operational properties future maintainers need to know

- **Failure surface for the projector is broad but bounded.** Pointer
  failure, archive failure, retention failure: all log to stderr and
  continue. The only fatal path is the SNAPSHOT.md write itself.
  `projectSnapshot()` in `internal/cli/checkpoint.go` wraps everything
  in a non-fatal call so the broader checkpoint never fails because of
  snapshot-side issues.
- **Content-hash skip means quiet projects produce no file churn.**
  `writeIfChanged()` short-circuits when bytes match the existing file;
  a graph that hasn't moved produces zero writes per Stop hook.
- **Surface inference runs on every projector pass.** No caching. On
  large repos (1000s of top-level dirs) this could become measurable;
  for current scales (hero-engine itself) it's not. If profile pressure
  appears, the natural fix is invalidation on `.hero/surfaces.yaml`
  mutation + repo-root mtime, not a cache.
- **`hero check` is the validation surface.** It parses every archive,
  asserts isolation flags, asserts the banner line, confirms the
  search-exclude registration. New archive-shape contracts should add
  assertions there.
- **The `projector_version` frontmatter field hedges shape evolution.**
  Bump it when the render shape changes; readers can detect mismatches
  and render older archives with a badge instead of misinterpreting
  them as current shape.
- **MCP tool keeps one name, not three.** Tool discovery is expensive;
  flag discovery is free. Adding a new dimension (e.g. per-team
  rollup) should land as a flag on `hero_snapshot`, not as a new tool.

## Open follow-ups

Noticed during the reverse-engineer; not blockers, worth recording so
they aren't lost.

1. **The "(unassigned)" row hint promises `hero snapshot assign` as an
   interactive helper.** `internal/cli/snapshot.go` registers
   `snapshotAssignCmd`. Whether the interactive prompt UX matches what
   the rendered hint promises (per-spec walk, accept-or-skip flow)
   should be verified against actual user feedback. The delivery spec
   describes the intended shape; the in-code implementation should be
   checked against it before any larger UX investment.
2. **Tracker integration step of the release-resolution chain depends
   on tracker adapters that may not all be present.** The delivery spec
   names Jira fixVersion, GitHub milestone, Linear cycle. If a project
   uses a tracker Hero doesn't yet adapt to, the chain falls through to
   step 3 (git tags) — by design, and graceful, but a "tracker-agnostic
   release field" config was deferred. Reconsider when a real project
   requests it.
3. **`hero pulse` and `hero snapshot` overlap on intent.** Both
   summarize project state. The delivery spec is explicit that pulse is
   *narrative weekly* and snapshot is *structural rollup* — three
   different surfaces (pulse, snapshot, NEXT) for three different
   questions. As pulse is implemented, the boundary needs to stay
   crisp: pulse may *read* archives via the explicit history-query
   surfaces but must not surface archive bodies via default-discovery
   paths.
4. **Initiative-completion archive ordering.** Open question #2 in the
   delivery spec: should the milestone archive be written *before* or
   *after* the initiative status flip propagates through the projector?
   Currently *after*. Revisit if users find the resulting archive
   misleading.
5. **Snapshot bytes are never injected — but the delivery spec mentions
   `/resume` and `/prime` may optionally include the snapshot in
   cold-start bundles.** Whether that optional inclusion is wired
   anywhere yet, or is purely documented intent, should be checked
   before treating "snapshot can land in cold start" as a real path. As
   of v0.10, the only confirmed surfaces are the explicit pull paths
   (file, MCP, CLI, serve).
6. **Three Knowledge notes were captured alongside delivery** —
   `discoverable-not-pushed-artifacts`,
   `historical-artifact-isolation`,
   `inference-first-with-override-layer` (see commit `5110b1a`). These
   are the reusable patterns the snapshot design exercises. Future
   architecture work should treat them as available primitives, not
   reinvented one-offs.

## Provenance and gaps

Sources mined for this record:

- Commit `5110b1a` body (the most complete current artifact prior to
  this decision spec; reproduced almost in full here).
- `.hero/specs/project-snapshot/spec.md` (1,759-line delivery spec —
  the design narrative is dense and authoritative; this record
  summarizes and quotes selectively).
- `internal/snapshot/` package source (projector.go, detect.go,
  surfaces.go, stage.go, archive.go, pointers.go, retention.go,
  release.go, rollup.go, render_markdown.go, render_json.go, diff.go).
- `internal/cli/snapshot.go` (CLI surface).
- `internal/cli/checkpoint.go` (Stop-hook integration via
  `projectSnapshot()`).
- `.gitattributes` (merge driver registration).

**Rationale gaps to flag:** None material. The delivery spec is
unusually thorough about *why* each design choice was made (each
sub-section under "Approach" has an explicit "why option X is wrong"
paragraph). The alternatives in this record draw directly from those
paragraphs; the only "inferred" framings are mild reorderings, never
invented motivations.
