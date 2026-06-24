---
title: "Feature Knowledge Synthesis — Auto-Generated 'How It Works' Entries from Shipped Work"
slug: feature-knowledge-synthesis
type: initiative
status: planning
domain: engineering
size: large
priority: medium
created: 2026-06-23
horizon: next
tags: [knowledge, synthesis, discoverability, cold-start, initiative, onboarding, dx]
child:
  - fks-feature-knowledge-artifact
  - fks-on-demand-synthesizer
  - fks-cluster-detection
  - fks-trust-handshake
  - fks-living-doc-amendment
relations:
  - target: synthesis-maintenance
    kind: related
  - target: timely-briefs
    kind: related
---

## Vision

When a developer ships a feature across many specs over weeks, the next person
who asks "how does this thing work?" should find a single, current, readable
explainer in the knowledge base — not be told to read eight specs in commit
order and reconstruct it themselves. Hero already records **why** each change
happened (decisions) and **how we do things** (conventions). It does not record
**how a shipped feature works, end to end, as it exists now**. This initiative
closes that gap, and makes the explainer appear *because work shipped*, not
because someone remembered to write it.

## Goal

Recognize when a coherent feature has shipped — whether wrapped in a formal
initiative or assembled from a loose cluster of specs — and synthesize a
durable `explainer` knowledge entry describing how it works, with
provenance back to the specs it came from. Generation happens automatically,
behind a one-keystroke trust handshake; the entry stays current by being
amended (not regenerated) as later related specs land; and a human can always
delete or annotate it, with a sacred Developer Notes zone the synthesizer never
touches.

## Problem

The trigger for this initiative was a real question: a developer spent ~2 weeks
on a feature spanning many specs, and a teammate later asked whether there was a
"how this works" summary in knowledge — there wasn't, and reading every spec was
too tedious to bother. Investigation of this repo confirms the gap is
structural, not just discipline:

1. **The knowledge taxonomy is skewed away from feature explainers.** Live
   counts in `.hero/knowledge/`: ~20 `type: decision`, ~17 `type: note`, ~6
   `type: convention`, **1 `type: feature`**. Hero captures the *why* and the
   *how-we-do-things* well; it captures the *how-a-feature-works-now* almost not
   at all.
2. **Auto-capture is keyed to the wrong unit.** `knowledge.auto_capture` (on by
   default) fires at the end of a *single* workflow — `/retro` runs per-spec
   (`core/commands/retro.md`), the `auto-knowledge-capture` skill triggers at
   workflow end. A feature spanning 8 specs produces, at best, 8 disconnected
   decision captures and never a bird's-eye synthesis. No step owns the whole.
3. **Auto-capture writes the wrong artifact type.** Retro is told to capture
   *conventions, decisions, rules* — "how it works" is not on that list, so even
   when it fires it does not produce the wanted artifact.
4. **The natural trigger is unused.** An initiative has a completion gate
   (`in-flight → shipped` when children complete), the obvious "a feature just
   shipped" signal — and nothing hooks it. But keying *only* on initiative close
   would miss the originating case, where the work was never wrapped in a formal
   initiative. Boundary detection must also handle loose clusters.

The through-line: Hero knows when work *finishes* and knows which specs *relate*,
but never turns that into a standing description of the thing that was built.

## Guiding principles

- **Synthesis happens because work shipped, not because someone remembered.**
  The boundary triggers it; the developer never has to think to ask.
- **Provenance over authority.** A "how it works" doc claims to describe current
  reality, so it must always name the specs it was built from and when it was
  last synthesized. Stale-but-honest beats confident-and-unattributed.
- **Auto, but with a recoverable handshake.** Generation is automatic; a wrong
  boundary detection is undone in one keystroke, not lived with as a bad doc.
- **The doc is a living artifact, amended not regenerated.** Later specs add to
  it and strike what they contradict; git holds the history. Regeneration is
  avoided precisely so human deletions stick.
- **The human is sovereign over the doc.** Free deletion, plus a sacred
  Developer Notes zone the synthesizer never reads or writes.
- **Reuse the substrate, don't rebuild it.** Detection rides Hero's existing
  relation graph; amendment rides `synthesis-maintenance`'s `OnWrite` hook.

## Boundary vs. `synthesis-maintenance` (read before assuming overlap)

These two are adjacent and must not be conflated:

- **`synthesis-maintenance`** does *node-level write-through coherence*: on every
  graph write, integrate that one node against the substrate (dedup, write
  `supersedes`/`related` edges). It keeps the graph *connected*. It never
  produces a new human-readable artifact.
- **This initiative** does *feature-level synthesis*: collapse a *cluster* of
  completed specs into one new `explainer` entry. It produces a new
  artifact.

They compose: `fks-living-doc-amendment` (#5) should fire from
`synthesis-maintenance`'s `OnWrite(node, prev)` hook when the new node relates to
a feature that already has a synthesized entry — rather than inventing a parallel
trigger. Wire to it; don't duplicate it.

## Specs

Child stubs below are each actionable as a `/design` input. Slugs are listed in
`child:` frontmatter. **#1 and #2 are fully designed now** (`fks-…/spec.md`
exist); #3–#5 are sequenced stubs, deliberately left for a `/design` pass once
the engine has taught us what a good synthesis looks like.

### fks-feature-knowledge-artifact  *(materialized — designed)*
Define the `explainer` knowledge entry: its template ("how it works"
structure — purpose, surfaces/entry points, key flows, data shape, gotchas),
its provenance frontmatter (source specs + last-synthesized timestamp), and the
sacred **Developer Notes** section. Settles the link policy vs. `decision`
entries (reference captured decisions, don't restate them). Size: small.

### fks-on-demand-synthesizer  *(materialized — designed)*
`hero synthesize <slugs…>` (+ MCP surface): read the named specs and the git
diff across their delivery window, emit one feature knowledge entry in the #1
shape. Exercised manually first — zero detection risk — so we learn what "good"
is before automating the trigger. The spine of the initiative. Size: medium.

### fks-cluster-detection  *(delivered)*
Infer feature boundaries Hero isn't told about: score coherent spec clusters from
graph signals (shared relations, co-touched files, time window, author) and a
completeness gate (don't synthesize a half-shipped feature). Emit candidates with
confidence. The risky, novel part; worthless without #2 behind it. Also wire the
explicit signal — the initiative/epic `completed` gate — as a zero-false-positive
candidate source. Size: medium–large.

### fks-trust-handshake  *(delivered)*
Turn candidates into the autonomy the user asked for: on detection, surface
"feature X shipped across these specs — I drafted an entry; keep doing this
automatically / let me review each / turn it off," and persist the choice in
`hero.json` (mirror the existing `knowledge.auto_capture` flag). Low-confidence
candidates always route through review regardless of mode. This is what makes
auto-generation safe against detection false positives. Size: small–medium.

### fks-living-doc-amendment  *(stub — design last)*
Keep the entry current: when a later spec joins a known cluster, **amend
surgically** — add new behavior, strike/correct what it contradicts — rather than
regenerate. Fire from `synthesis-maintenance`'s `OnWrite` hook. Honor human
sovereignty: the **Developer Notes** section is never touched; free human
deletion sticks (guaranteed by amend-not-regenerate — nothing resurrects deleted
content); the synthesizer may only strike-through/annotate generated content
(visible), while humans may hard-delete (git holds history). The hard 20%;
sequenced last because it needs real entries in the wild to calibrate. Size:
large.

## Dependencies

- `fks-on-demand-synthesizer` (#2) depends on `fks-feature-knowledge-artifact`
  (#1) — the engine writes into the artifact the first spec defines.
- `fks-cluster-detection` (#3) depends on #2 — detection is worthless without a
  good generator behind it.
- `fks-trust-handshake` (#4) depends on #3 — nothing to hand-shake over until
  candidates exist.
- `fks-living-doc-amendment` (#5) depends on #1 (the Developer Notes / provenance
  contract) and coordinates with `synthesis-maintenance` (the `OnWrite` hook).

## Cross-cutting concerns & shared risks

- **Detection false positives.** Inferred clusters will sometimes name a
  non-feature. Mitigated not by perfect detection but by the cheap-to-be-wrong
  handshake (#4) — a bad candidate is dismissed in one keystroke.
- **Stale confident docs.** A "how it works" doc that silently rots is worse than
  none. Provenance stamping (#1) + amend-on-new-spec (#5) are the mitigation;
  both are load-bearing, not optional polish.
- **Human-edit reconciliation.** Auto-amendment must never clobber a hand-edited
  doc. Resolved by the Developer Notes sacred zone + amend-not-regenerate; a
  per-section "manual — don't auto-touch" marker is a possible #5 extension.
- **Feature-vs-decision duplication.** Feature entries will reference decisions
  already captured. Settle a link-don't-restate policy in #1.

## Out of scope / deferred

- **Inline pin markers** (`<!-- pin -->`) letting a human fence an untouchable
  caveat *inline* next to a generated claim — deferred until the Developer Notes
  footer proves too coarse in practice.
- **Tombstones / suppression memory** for deletions — unnecessary while we
  amend-not-regenerate; only needed if a full-regeneration path is ever added.
- **Cross-feature linking** (feature A depends on feature B) and surfacing
  feature entries in `hero snapshot` — valuable, but only once entries exist.

## Recommended delivery order

1. **fks-feature-knowledge-artifact** — the contract everything writes into.
2. **fks-on-demand-synthesizer** — ships value immediately (point it at the
   feature that prompted this and get the doc today); de-risks everything below.
3. **fks-cluster-detection** — once we know what good output looks like.
4. **fks-trust-handshake** — turns detection from a liability into the asked-for
   autonomy.
5. **fks-living-doc-amendment** — last; needs real entries to calibrate, and
   reuses `synthesis-maintenance`'s hook.

## Progress

- 2026-06-23 — **Delivered #3 `fks-cluster-detection`.** `Detect` + `hero
  synthesize --detect`: explicit (completed-initiative children) + inferred
  (graph + file-overlap, completeness-gated, dedup'd) candidates with explainable
  confidence. E2e on the real repo exposed an over-clustering bug (hub files
  chained ~180 specs into one blob); fixed with a hub-file guard + size cap. Now
  returns clean candidates incl. the dogfood `fks` pair. Tests green.
- 2026-06-23 — **Designed #3 `fks-cluster-detection`** (full spec materialized).
  Folded in a delivered learning from #2: time-window clustering swept in
  unrelated same-day commits, so detection is specced graph-first (shared
  parent/relations/co-touched files dominate; time is a weak tiebreaker), with a
  completeness gate and explainable confidence scoring. Emits candidates only;
  synthesis stays in #2, the prompt in #4. Also produced the first real explainer
  (dogfood: `cold-start-trust-hardening`) via the agent-fills path.
- 2026-06-23 — **Delivered #2 `fks-on-demand-synthesizer`.** New
  `internal/synthesize` package + `hero synthesize <slug...>` CLI and
  `hero_synthesize` MCP tool. Per the "both" decision: CLI assembles specs +
  git-diff window + referenced decisions deterministically; prose is generated
  by the in-session agent (MCP, no key) by default, or by the CLI's LLM path
  when a provider key is present (for headless / the future #3/#4 auto-trigger).
  Fail-loud on unresolved slugs; scaffold fallback with the material when no key.
  Verified e2e on the real repo. #3 (cluster detection) is next.
- 2026-06-23 — **Delivered #1 `fks-feature-knowledge-artifact`.** Added the
  first-class `explainer` knowledge type (`.hero/knowledge/explainers/<slug>/spec.md`),
  wired through type recognition, `IsKnowledge`, graph node type, both
  `validTypes` allowlists, the knowledge stats counter, and four work-rollup
  exclusion switches; `hero check --knowledge` now validates `synthesized_from`
  + `last_synthesized` provenance. Format doc at
  `core/skills/explainer-format/SKILL.md`. Verified end-to-end with the dev
  binary; full `internal/spec`/`cli`/`serve` suites green. Verification corrected
  the on-disk layout (dir/`spec.md`, not flat) and surfaced a pre-existing
  cross-type queue-advisory issue (flagged separately). #2 is next.
- 2026-06-23 — Initiative drafted from a `/discover` session on the "no how-it-
  works summary after a multi-week feature" gap. Resolved three design forks:
  inferred clusters (not initiative-close only), auto-with-handshake autonomy,
  living-doc amendment with a sacred Developer Notes zone + free human deletion.
  Boundary carved vs. `synthesis-maintenance` (node-coherence) and related to it
  + `timely-briefs`. #1 and #2 designed in full; #3–#5 left as sequenced stubs to
  avoid over-designing ahead of the engine's learnings.
