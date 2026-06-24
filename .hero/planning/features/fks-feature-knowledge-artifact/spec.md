---
title: "Explainer Knowledge Type — the 'How It Works' Entry"
type: feature
status: completed
slug: fks-feature-knowledge-artifact
domain: engineering
parent: feature-knowledge-synthesis
priority: medium
size: small
created: 2026-06-23
completed_at: 2026-06-23
tags: [knowledge, synthesis, schema, template, provenance, explainer]
kind: new
---

# Explainer Knowledge Type

## Goal

Add a new first-class knowledge type — **`explainer`** — the durable "how this
feature works, as it exists now" artifact that the rest of the
`feature-knowledge-synthesis` initiative generates and maintains. This spec ships
the **contract**: the new type, its `.hero/knowledge/explainers/` home, the
section template, the provenance frontmatter, the sacred Developer Notes zone,
and the link policy vs. existing `decision` entries. No generation, no detection
— just the shape everything downstream reads and writes.

## Kickoff

Add an `explainer` knowledge type to the Hero Go codebase. It must be recognized
as knowledge (not a work spec) by indexing, search, and stats, and live under
`.hero/knowledge/explainers/`. Carry provenance frontmatter (`synthesized_from`
spec slugs + `last_synthesized` date) and a fixed how-it-works section skeleton
ending in a **Developer Notes** section flagged human-owned. Touch points (from
code audit): `TypeExplainer` const, `typeFromPath`, `IsKnowledge`,
`graphTypeFor`, the knowledge stats counter, and a `hero check` validator for the
provenance fields. Parent initiative: `feature-knowledge-synthesis`.

## Problem

Hero's knowledge base has no artifact for a feature explainer, and the obvious
shortcut — reusing `type: feature` — collides destructively with work specs
(confirmed by code audit, 2026-06-23):

- Type derives from path/frontmatter, and `typeFromPath` (`internal/spec/spec.go`)
  has no `/knowledge/...` check for this, so a `type: feature` knowledge file
  falls through to the **work-spec** `TypeFeature` default.
- `IsKnowledge()` (`internal/spec/spec.go`) does **not** include `TypeFeature`, so
  such an entry is classified as *work* — it would leak into `hero queue`,
  `hero search --type feature`, and never appear in knowledge counters.
- All specs share one `specs` table, so a work feature and a knowledge feature
  are indistinguishable to search and filters.

Today's knowledge types (`decision`, `convention`, `note`) are all point-in-time
records *allowed* to be historical. An explainer is different: it claims to
describe **current reality**, so it needs two things they don't:

1. **Provenance** — which specs it was synthesized from, and when last refreshed.
2. **An ownership boundary inside the doc** — a zone the (future) auto-amender
   never touches, so human annotations survive regeneration pressure.

Without a settled contract, every downstream spec (#2 synthesizer, #5 amender)
would invent its own shape and they would drift. This spec prevents that.

## Design

### A new type, not an overloaded one
Add `TypeExplainer Type = "explainer"` to the type enum. Wire it through the five
sites the audit identified so it behaves as a first-class knowledge type:
- `typeFromPath` — recognize `.hero/knowledge/explainers/` → `TypeExplainer`.
- `statusFromPath` — default explainers to `StatusActive` (like other knowledge).
- `IsKnowledge()` — include `TypeExplainer` so it's classed as knowledge, not work.
- `graphTypeFor()` — map `TypeExplainer` → `"Explainer"` node type.
- knowledge stats counter — count explainers in the knowledge section.

### Location & recognition
- Entries live at `.hero/knowledge/explainers/<slug>/spec.md`, a sibling of
  `decisions/`, `conventions/`, `notes/`.
- `hero index` / `hero search` recognize them via the new type; `--type explainer`
  filters cleanly, with no work-feature contamination.

### Frontmatter contract
```yaml
---
title: <human title of the feature>
type: explainer
synthesized_from:           # provenance — the spec cluster
  - <spec-slug>
  - <spec-slug>
last_synthesized: <YYYY-MM-DD>
source_initiative: <slug or null>   # set when boundary was an initiative
tags: [...]
---
```
Add `synthesized_from []string` and `last_synthesized string` to the `Spec`
struct + `parseFrontmatter` so the check can validate them structurally.

### Section skeleton (the "how it works" template)
A fixed, ordered set of sections the synthesizer fills:
1. **What it is** — one-paragraph purpose.
2. **Surfaces / entry points** — commands, MCP tools, files, UI a user touches.
3. **How it works** — the key flows, in order.
4. **Data & state** — what it reads/writes/persists.
5. **Gotchas** — non-obvious constraints, sharp edges.
6. **Related decisions** — *links* to existing `decision` entries (slugs), not
   restatements.
7. **Developer Notes** — **human-owned. The synthesizer never reads or writes
   below this heading.**

### Ownership boundary
- Everything above `## Developer Notes` is synthesizer-owned (generated, later
  amendable).
- `## Developer Notes` and below is sacred human tribal knowledge, never
  auto-touched.

### Link-don't-restate policy
**Related decisions** references decision-entry slugs; the synthesizer links
rather than duplicates decision content, so the two types stay DRY.

## Acceptance Criteria

- THE SYSTEM SHALL define an `explainer` knowledge type recognized under
  `.hero/knowledge/explainers/` by `hero index`, returned by `hero search`, and
  filterable via `--type explainer`.
- THE SYSTEM SHALL classify `explainer` entries as knowledge (`IsKnowledge()`
  true) and exclude them from work rollups — the `now`/`work` page tallies,
  roadmap, and in-flight strips. (Note: kickoff-less knowledge entries of every
  type still surface in `hero queue`'s advisory list today; that is pre-existing
  cross-type behavior, tracked separately — see Delivery notes.)
- THE SYSTEM SHALL map `explainer` to an `Explainer` graph node type, distinct
  from `Feature`.
- WHERE an explainer entry exists THE SYSTEM SHALL require `synthesized_from`
  (≥1 spec slug) and `last_synthesized` in its frontmatter, and `hero check`
  SHALL warn on an entry missing either.
- THE SYSTEM SHALL count explainers in the knowledge stats counter.
- THE SYSTEM SHALL provide the fixed section skeleton (What it is → Developer
  Notes) as a documented template downstream synthesis can fill.
- THE SYSTEM SHALL document `## Developer Notes` as a human-owned zone automated
  synthesis must never read or write, and the link-don't-restate policy for
  `decision` references.

## Boundaries / Out of scope

- No generation logic (that's `fks-on-demand-synthesizer`).
- No detection, no trust handshake, no amendment (#3/#4/#5).
- No inline pin markers — Developer Notes is the only ownership zone in v1.

## Dependencies

- None upstream. Foundation spec; #2 depends on it.
- Coordinate the frontmatter / Developer-Notes contract with
  `fks-living-doc-amendment` (#5), which must honor it.

## Delivery notes

- 2026-06-23 — Delivered. `explainer` type wired through `TypeExplainer`,
  `typeFromPath`/`statusFromPath`, `IsKnowledge`, `graphTypeFor`, both
  `validTypes` allowlists (`validate.go`, `triage/structural.go`), the knowledge
  stats counter, and four work-rollup exclusion switches (counts/inflight/
  roadmap/themes). Provenance check added to `hero check --knowledge`. Provenance
  fields (`synthesized_from`, `last_synthesized`) parse in inline + block-list
  form. Format documented in `core/skills/explainer-format/SKILL.md`. Unit tests
  in `internal/spec/spec_test.go`; full `internal/spec` suite green.
- **Layout correction (found in verification):** `spec.Discover` only loads
  `<dir>/spec.md` (and three-file dirs), never flat `<slug>.md`. Explainers
  therefore live at `.hero/knowledge/explainers/<slug>/spec.md`, not flat. Specs
  #1/#2 and the skill corrected.
- **Pre-existing follow-up (out of scope):** kickoff-less knowledge entries of
  *every* type (note/context and now explainer) appear in `hero queue`'s advisory
  "no `## Kickoff`" list. Verified identical for an existing note. Excluding all
  knowledge from that advisory is a separate cross-type change, flagged for its
  own spec.
