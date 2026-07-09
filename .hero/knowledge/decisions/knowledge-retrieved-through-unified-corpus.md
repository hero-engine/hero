---
title: Hand-Authored Knowledge Is Retrieved Through the Unified Corpus, Keyed by Directory
type: decision
status: proposed
created: 2026-07-06
tags: [knowledge, retrieval, hero-ask, hero-search, ingest, architecture, decision]
relations:
  - target: knowledge-content-retrieval
    kind: decided-in
  - target: hero-ask
    kind: related
  - target: unified-retrieval-layer
    kind: related
  - target: sales-pack-reality-sync
    kind: informed-by
---

# Hand-Authored Knowledge Is Retrieved Through the Unified Corpus, Keyed by Directory

## Decision

Hand-authored files under `.hero/knowledge/**/*.md` are made content-retrievable
by **ingesting them into Hero's existing unified retrieval corpus** under a
`category=knowledge` marker, keyed by their containing directory, and exposing
them through the **surfaces that already promise knowledge** — `hero ask`
(primary) and `hero search --knowledge` (opt-in).

We do **not** build a separate `hero knowledge find` command, and we do **not**
loosen the `nonWorkFlatTypes` boundary that keeps knowledge out of work-spec
discovery.

**Capture is layout-agnostic.** The ingest keys on *location + content*, never on
file shape. Flat `foo.md`, `<slug>/spec.md`, and three-file layouts are all
captured; typed, untyped, and domain-invented subdirs are all captured;
malformed files are indexed with what reads rather than dropped. The recent
work-spec hardening that *nudges* authors toward `<slug>/spec.md` (so a work spec
can hold audit/retro children) is deliberately **not** imposed on knowledge —
most knowledge is a single file and should stay one if the author wants.
Following slug/spec is a nudge, never a precondition for being found; any such
nudge lives in `hero check --knowledge`, not in retrievability.

Delivered as the [[knowledge-surfacing]] initiative: P1
[[knowledge-content-retrieval]] (layout-agnostic ingest + pull) and P2
[[knowledge-context-injection]] (push/file-scoped injection).

## Context

Hero stores knowledge in two shapes and only one is retrievable today:

- **spec-shaped** (`<slug>/spec.md`, `type: convention|decision|context|rule`) —
  loaded by `spec.Discover` (which loads `spec.md` type-agnostically), indexed
  into `fts_specs`, reachable by `hero ask --type` and `hero search`. This is
  what the completed `hero-ask` feature was designed and tested against.
- **flat-file** (`.hero/knowledge/<subdir>/*.md`) — the *canonical* home that
  `knowledge.auto_capture`, `/note`, and domain agents actually write to
  (e.g. sales `battlecards/<competitor>.md`, `playbooks/<slug>.md`). Reachable
  by **no** content-search surface: `spec.Discover` skips these via
  `nonWorkFlatTypes` (`internal/spec/spec.go:1161-1196`), and the knowledge-graph
  ingest covers only `raw/` (`internal/knowledge/graph_ingest.go:31`).

Verified against a live binary on this repo's own `.hero/`:
`hero ask "what is the peer manifest publish boundary"` returns *"No knowledge
found"* even though `.hero/knowledge/decisions/peer-manifest-publish-boundary.md`
says exactly that; 60 flat knowledge files are invisible to content search.

The deeper finding: whether a captured entry ever surfaces is decided by an
**accidental authoring layout**, not by its type or intent. Entries authored as
`<slug>/spec.md` (all `context/`, `rules/`, `tripwires/`, `explainers/`, and 12
of 17 `notes/`) are indexed and reach *every* surface; entries authored as flat
`<name>.md` (all 23 `decisions/`, all 4 `conventions/`, 5 `notes/`) reach none.
And retrieval is not only *pull* — the same `specs` / `convention_scopes` tables
back the **push** surfaces (`hero_context`/`BuildContext` file-scoped injection,
`hero relevant`/`BuildNudge`, `drift`, `impact`, `hero anchor`). So a flat
`conventions/*.md` whose whole purpose is to constrain code edits never injects
when the model touches the files it governs. Capture→surface is broken most
where it matters most.

This surfaced during `sales-pack-reality-sync`, whose AC#6 fell back to
path-based `ls` lookup because no content surface reached the battlecard
directory. The workaround is honest but can't rank, can't answer content
questions, and requires the caller to already know the exact path. The gap is
engine-wide (pm/sales/chat/engineering), so it earns its own decision.

Two constraints frame the design:
1. `hero ask` was *specified* to answer against "Knowledge and Specs" and already
   carries `--type convention|decision|context|rule`. The surface exists; it is
   under-fed, not missing.
2. `nonWorkFlatTypes` exists for a real reason: keeping knowledge out of
   **work** discovery (`hero list`, `hero queue`, `verify` child-resolution).
   `sales-pack-reality-sync` AC#7 literally depends on a template battlecard
   *not* appearing in `hero list`.

## Options considered

1. **New `hero knowledge find` surface.**
   A dedicated command with its own walk/rank over `.hero/knowledge/`.
   - Pros: leaves existing indexes untouched; explicit, discoverable verb.
   - Cons: fragments the unified retrieval facade the engine already committed
     to (`unified-retrieval-layer`: "Every caller that needs ranked content goes
     through here"); duplicates ranking + passage-extraction that `hero ask`
     already has; leaves `hero ask`'s "query against Knowledge" promise still
     broken; a third surface users must learn.

2. **Loosen `nonWorkFlatTypes` so Discover picks up knowledge.**
   Let flat knowledge flow through the existing spec index by removing the
   exclusion.
   - Pros: smallest diff; reuses `RefreshIfStale` directly.
   - Cons: regresses the exact boundary that keeps knowledge out of work
     discovery — battlecards/playbooks/notes would reappear in `hero list`,
     `hero queue`, and `verify` child-resolution, breaking `sales-pack-reality-
     sync` AC#7 and the `flat-named-spec-discovery` invariants. Wrong lever.

3. **Ingest flat knowledge into the unified corpus under `category=knowledge`,
   keyed by directory; expose via `hero ask` + `hero search --knowledge`.**
   A knowledge ingest (sibling to the `raw/` graph ingest, or a category-aware
   extension of the index) walks `.hero/knowledge/**` (excluding `raw/`),
   indexes each file with `category=knowledge` and a `kind` from its subdirectory
   (`battlecards`, `playbooks`, `decisions`, …), and self-heals on change.
   Untyped files are covered because categorization is by directory, not
   frontmatter. Work surfaces filter `category=work`; knowledge surfaces filter
   `category=knowledge`.
   - Pros: honors the unified-retrieval decision (one facade, one ranker);
     fixes `hero ask`'s existing promise instead of inventing surface area;
     reaches untyped template-authored files (the battlecard case) without
     forcing frontmatter into agent-written content; preserves the work/knowledge
     boundary by construction; uniform across all domains.
   - Cons: needs a persisted `category` discriminator on index rows (small
     schema/migration); two ingest walkers to maintain (work-spec + knowledge)
     until they're unified; `kind`-vs-`type` vocabulary needs a mapping rule.

## Decision

Option 3. Ingest into the unified corpus under `category=knowledge`, keyed by
directory, exposed through `hero ask` and `hero search --knowledge`.

Option 1 is rejected for fragmenting the retrieval facade; Option 2 is rejected
for regressing the work/knowledge boundary. Option 3 is the only one that both
fixes retrieval and keeps knowledge out of work discovery.

## Consequences

- A `category` (work|knowledge) discriminator is added to the retrieval corpus;
  default work surfaces (`hero list`, `hero queue`, `hero search`, verify
  child-resolution) filter to `category=work`.
- `.hero/knowledge/**/*.md` (excluding `raw/`) is indexed with a `kind` derived
  from the first subdir segment. Known kinds (`decisions`→decision,
  `conventions`→convention, `context`, `rules`→rule) map onto the existing
  `--type` vocabulary; free-form kinds (`battlecards`, `playbooks`) are searchable
  by their raw subdir name.
- `hero ask` now reaches the flat-knowledge corpus, closing the gap between its
  spec ("query against Knowledge and Specs") and its behavior.
- `hero search` gains an opt-in `--knowledge` scope; without it, work-search
  results are unchanged.
- The corpus is the shared substrate for both *pull* and *push*: once flat
  code-scoped knowledge (conventions/decisions/rules) has its `scope:` globs in
  `convention_scopes`, the existing injection surfaces — `hero_context` /
  `BuildContext`, `hero relevant` / `BuildNudge`, `drift`, `impact` — light up
  with no per-surface change, because they all read `FindConventionsForFiles`.
  This is the higher-value half: knowledge reaching the model *at edit time*, not
  only when someone asks. Free-form knowledge (battlecards/playbooks) has no
  code scope and stays pull-only, so injection remains signal rather than noise.
- The sales pack replaces `ls .hero/knowledge/battlecards/`-style path lookup
  with the content surface, retiring the `sales-pack-reality-sync` AC#6
  deviation.
- `nonWorkFlatTypes` stays as-is; the work/knowledge separation is enforced by
  the `category` marker, not by the discovery exclusion alone.
- Semantic/vector retrieval of knowledge is not delivered here but becomes a
  transport swap once knowledge lives in the corpus (rides the existing
  embeddings path).
