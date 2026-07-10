---
title: "Knowledge Surfacing — Everything Captured Is Retrievable and Fed at the Right Time"
slug: knowledge-surfacing
type: initiative
status: completed
priority: P2
size: large
domain: engineering
created: 2026-07-06
tags: [knowledge, retrieval, context-injection, ingest, capture-loop, all-domains]
relations:
  - target: hero-ask
    kind: builds-on
  - target: unified-retrieval-layer
    kind: builds-on
  - target: sales-pack-reality-sync
    kind: informed-by
  - target: content-remediation
    kind: related
  - target: knowledge-retrieved-through-unified-corpus
    kind: decided-in
child:
  - knowledge-content-retrieval
  - knowledge-context-injection
  - inline-flow-relations-dropped
  - flat-tripwire-trigger-parity
mission_alignment: |
  Hero's pitch is that captured institutional knowledge makes every session
  start smarter. Today that is only true for knowledge that happens to be
  authored as <slug>/spec.md; the flat files that /note, auto_capture, and
  every domain agent actually write vanish — unreachable by ask, search, or
  context injection. Capture without retrieval is dead weight. This initiative
  makes the capture→surface loop whole: everything that belongs in ask, search,
  or context is captured regardless of layout, and fed to the model at the
  moments that matter.
autonomy: guided
completed_at: 2026-07-10T06:05:07Z
---

# Knowledge Surfacing — Everything Captured Is Retrievable and Fed at the Right Time

## Goal

Run until `hero verify knowledge-surfacing` reports PASS — every child
(`knowledge-content-retrieval`, then `knowledge-context-injection`) designed,
delivered, and verified — or a `needs_me` pause is raised. Order is forced: P2
`depends-on` P1, so P1 ships first. Pause on any design fork (the ingest-seam
choice, the `category`-column schema migration), any irreversible action, or a
stuck gate.

## Context

Investigation during `sales-pack-reality-sync` surfaced a gap far wider than the
sales pack. Verified against a live binary on this repo's own `.hero/`:

- Whether a captured knowledge entry ever surfaces is decided by an **accidental
  authoring layout**, not by its type or intent. `<slug>/spec.md` entries (all
  `context/`, `rules/`, `tripwires/`, `explainers/`, 12 of 17 `notes/`) are
  indexed and reach every surface. Flat `<name>.md` entries (all 23 `decisions/`,
  all 4 `conventions/`, 5 `notes/`) reach **none**.
- Retrieval is not only *pull*. The same `specs` / `convention_scopes` tables
  back the **push** surfaces — `hero_context` / `BuildContext` (file-scoped
  injection), `hero relevant` / `BuildNudge`, `drift`, `impact`, `hero anchor`.
  So a flat `conventions/*.md` whose whole purpose is to constrain code edits
  never injects when the model touches the files it governs.
- 60 flat `.hero/knowledge/**/*.md` files are invisible to content search;
  `hero ask` on a flat decision returns "No knowledge found."

The decision on how to close this is recorded in the ADR
[[knowledge-retrieved-through-unified-corpus]]: ingest flat knowledge into the
existing unified corpus under `category=knowledge`, keyed by directory, exposed
through the surfaces that already promise it (`hero ask`, `hero search`,
`hero_context`) — not a new `hero knowledge find`, and not by loosening the
`nonWorkFlatTypes` boundary that keeps knowledge out of work-spec discovery.

## Guiding principle — capture is layout-agnostic; conventions are a nudge, not a gate

We recently hardened work-spec discovery to **prefer** the `<slug>/spec.md`
shape (`flat-named-spec-discovery`, `nonWorkFlatTypes`) — deliberately, because a
*work spec* can grow children (audits, retros, three-file splits) and needs a
directory to hold them. That nudge is right for work.

It is explicitly **not** imposed on knowledge. A battlecard, a one-line
decision, a captured note — most knowledge is a single file and should stay a
single file if the author wants. Following the slug/spec convention is a *nudge*,
never a *precondition for being found*. Therefore:

- **Capture keys on location + content, never on shape.** Any `*.md` anywhere
  under `.hero/knowledge/**` (except `raw/`, which has its own ingest) is
  captured — flat `foo.md`, `foo/spec.md`, three-file `foo/requirements.md`,
  typed, untyped, or in an arbitrary domain-invented subdir (`battlecards/`,
  `playbooks/`, `personas/`, whatever a pack creates tomorrow).
- **Best-effort and lossy-tolerant.** No frontmatter → derive title from the
  first `# H1` or the filename. Unknown subdir → `kind` is the subdir name. A
  malformed file is indexed with what can be read, not dropped. The bar is
  "captured and findable," not "well-formed."
- **The safety net, not the enforcer.** Retrieval never punishes a
  convention miss. If `hero check --knowledge` wants to nudge an author toward
  slug/spec for something that would benefit from children, that is a separate,
  advisory surface — it must not gate retrievability.

The test of this initiative: drop *any* `.md` with useful content into
`.hero/knowledge/` in *any* shape, and it is retrievable by `hero ask`/`search`
and — if it declares a code `scope:` — injected by `hero_context` on file-touch.

## Phases (children)

1. **`knowledge-content-retrieval`** (P1 — foundation + pull). A layout-agnostic
   knowledge ingest into the existing corpus (`category=knowledge`, `kind` from
   subdir, dedup against what Discover already indexes for spec.md-shaped
   knowledge); `hero ask` and `hero search --knowledge` reach it. Ships value on
   its own. Repoints the sales pack's `ls`-based battlecard lookup to the
   content surface.

2. **`knowledge-context-injection`** (P2 — push / the payoff). Feeds
   code-scoped knowledge (`conventions`/`decisions`/`rules`) `scope:` globs into
   `convention_scopes` so `BuildContext` / `BuildNudge` / `drift` / `impact`
   inject flat knowledge on file-touch, at parity with spec.md-shaped
   conventions. Free-form knowledge with no scope stays pull-only so injection
   remains signal, not noise. Depends on P1.

## Success criteria (initiative-level)

- WHERE a `.hero/knowledge/**/*.md` file exists in **any** layout (flat,
  slug/spec, or three-file) and carries content, THE SYSTEM SHALL make it
  retrievable by `hero ask` and `hero search --knowledge`.
- WHERE such a file has no frontmatter `type:`, THE SYSTEM SHALL still capture it
  (title from H1/filename, `kind` from subdir).
- WHERE such a file declares a code `scope:`, WHEN a matching file is edited,
  THE SYSTEM SHALL inject it via `hero_context`.
- THE SYSTEM SHALL NOT surface any knowledge file in `hero list`, `hero queue`,
  or `verify` child-resolution (the work/knowledge boundary is intact).
- THE SYSTEM SHALL treat capture uniformly across all domains (pm/sales/chat/
  engineering) with no domain-specific code path.

## Out of scope

- Semantic/vector retrieval of knowledge (rides the embeddings path once
  knowledge is in the corpus).
- A cold-start knowledge digest in NEXT/QUEUE/SNAPSHOT and `/prime` (natural
  follow-on; tracked separately).
- Changing where or in what shape knowledge is authored — this initiative makes
  *every* shape retrievable rather than forcing one.
- Loosening `nonWorkFlatTypes` or merging knowledge into work discovery.
