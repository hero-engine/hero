---
title: "Feature Knowledge Artifact — the 'How It Works' Entry Type"
type: feature
status: planning
slug: fks-feature-knowledge-artifact
domain: engineering
parent: feature-knowledge-synthesis
priority: medium
size: small
created: 2026-06-23
tags: [knowledge, synthesis, schema, template, provenance]
kind: new
---

# Feature Knowledge Artifact

## Goal

Define the `type: feature` knowledge entry — the durable "how this feature works,
as it exists now" artifact that the rest of the `feature-knowledge-synthesis`
initiative generates and maintains. This spec ships the **contract**: the section
template, the provenance frontmatter, the sacred Developer Notes zone, and the
link policy vs. existing `decision` entries. No generation, no detection — just
the shape everything downstream reads and writes.

## Kickoff

Define a new knowledge artifact type for "how a feature works" explainers.
Deliverable: a documented template + frontmatter contract under
`.hero/knowledge/` (likely a `features/` subdir), recognized by `hero index` and
`hero search` like other knowledge types. Must carry provenance (source spec
slugs + last-synthesized timestamp), a fixed how-it-works section skeleton, and a
**Developer Notes** section flagged as human-owned / never auto-touched. Settle
link-don't-restate vs. `decision` entries. Parent initiative:
`feature-knowledge-synthesis`. Start by reading an existing `decision` entry and
`core/skills/note-capture` for current knowledge conventions.

## Problem

Hero's knowledge base has no artifact for a feature explainer. Today's types —
`decision` (why), `convention` (how-we-do-things), `note` (point thought) — are
all point-in-time records that are *allowed* to be historical. A "how it works"
doc is different: it claims to describe **current reality**, so it needs two
things the other types don't:

1. **Provenance** — which specs it was synthesized from, and when it was last
   refreshed — so a reader can judge staleness and a maintenance pass knows what
   it covers.
2. **An ownership boundary inside the doc** — a zone the (future) auto-amender
   never touches, so human annotations survive regeneration pressure.

Without a settled contract, every downstream spec (#2 synthesizer, #5 amender)
would invent its own shape and they would drift. This spec exists to prevent
that.

## Design

### Location & recognition
- New entries live under `.hero/knowledge/features/<slug>.md` (a sibling of the
  existing `decisions/`, `conventions/`, `notes/` subdirs).
- `hero index` and `hero search` recognize `type: feature` knowledge entries the
  same way they recognize the other knowledge types — no special-casing beyond
  the new subdir + type value. (Note: `type: feature` already exists for *specs*;
  this is a knowledge-base entry, disambiguated by living under
  `.hero/knowledge/` — confirm the indexer keys on path, not just type, and
  resolve the collision explicitly if it doesn't.)

### Frontmatter contract
```yaml
---
title: <human title of the feature>
type: feature
synthesized_from:           # provenance — the spec cluster
  - <spec-slug>
  - <spec-slug>
last_synthesized: <YYYY-MM-DD>
source_initiative: <slug or null>   # set when boundary was an initiative
tags: [...]
---
```

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
- `## Developer Notes` and below is sacred: free-form human tribal knowledge,
  never auto-touched.

### Link-don't-restate policy
The **Related decisions** section references decision-entry slugs; the synthesizer
must link rather than duplicate decision content, so the two artifact types stay
DRY and a decision has one home.

## Acceptance Criteria

- THE SYSTEM SHALL recognize `type: feature` knowledge entries under
  `.hero/knowledge/features/` in `hero index` and return them from `hero search`.
- WHERE a feature knowledge entry exists THE SYSTEM SHALL require
  `synthesized_from` (≥1 spec slug) and `last_synthesized` in its frontmatter,
  and `hero check` SHALL warn on an entry missing either.
- THE SYSTEM SHALL provide the fixed section skeleton (What it is → Developer
  Notes) as a documented template downstream synthesis can fill.
- THE SYSTEM SHALL document `## Developer Notes` as a human-owned zone that
  automated synthesis must never read or write.
- THE SYSTEM SHALL document the link-don't-restate policy: feature entries
  reference `decision` entries by slug rather than restating them.
- IF the indexer keys knowledge type on the `type:` value and collides with the
  `feature` *spec* type THEN THE SYSTEM SHALL disambiguate by knowledge-base path
  so the two never cross-contaminate search results.

## Boundaries / Out of scope

- No generation logic (that's `fks-on-demand-synthesizer`).
- No detection, no trust handshake, no amendment (#3/#4/#5).
- No inline pin markers — Developer Notes is the only ownership zone in v1.

## Dependencies

- None upstream. This is the foundation spec; #2 depends on it.
- Coordinate the frontmatter/Developer-Notes contract with
  `fks-living-doc-amendment` (#5), which must honor it.
