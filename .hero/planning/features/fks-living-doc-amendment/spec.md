---
title: "Living-Doc Amendment — Keep Explainers Current Without Clobbering Humans"
type: feature
status: completed
slug: fks-living-doc-amendment
domain: engineering
parent: feature-knowledge-synthesis
priority: medium
size: large
created: 2026-06-23
completed_at: 2026-06-23
tags: [knowledge, synthesis, amendment, freshness, explainer]
kind: new
relations:
  - target: fks-on-demand-synthesizer
    kind: depends-on
  - target: synthesis-maintenance
    kind: related
---

# Living-Doc Amendment

## Goal

Keep an explainer current as later specs land — by **amending** it (add new
behavior, correct what's now contradicted, expand provenance) rather than
regenerating from scratch — while keeping the human sovereign: the
**Developer Notes** section is never touched, and human edits/deletions to the
generated body survive because amendment operates on the *current on-disk*
content, not a fresh rebuild.

## Kickoff

Build explainer amendment for feature-knowledge-synthesis. Add staleness
detection (`hero synthesize --stale`): an explainer is stale when a completed
spec belonging to its cluster (child of `source_initiative`, or related to a
`synthesized_from` member) appeared/completed after `last_synthesized`. Add
`hero synthesize --amend <slug>`: re-assemble the expanded cluster, split off
and preserve the `## Developer Notes` section verbatim, regenerate-or-scaffold
the generated body **from the current body + the new material** (amend, not
rebuild), expand `synthesized_from`, bump `last_synthesized`. Depends on #2.
The automatic trigger on `synthesis-maintenance`'s `OnWrite` hook is **deferred**
until that feature lands (it's unbuilt) — `--stale`/`--amend` are the manual
entry points now.

## Problem

An explainer claims to describe current reality, so it rots the moment a later
spec changes the feature. The naive fix — regenerate from the cluster — destroys
two things the user explicitly wanted preserved (from the `/discover` session):

1. **Human edits to the generated body.** If someone corrected a generated
   paragraph, a from-scratch regen overwrites the correction. Amendment must
   start from the *current* content so human edits persist.
2. **The Developer Notes zone.** Human tribal knowledge below `## Developer
   Notes` must never be read or written by synthesis.

And free human deletion only "sticks" because we amend rather than regenerate —
there is nothing to resurrect a deleted line from (no tombstones needed), exactly
because the source of truth for amendment is the on-disk doc, not the specs alone.

## Design

### Staleness detection (`--stale`)
For each explainer, find completed specs that belong to its cluster but are not
in `synthesized_from`:
- children of `source_initiative` (if set), and
- specs with a relation edge to any `synthesized_from` member,
filtered to `completed` and (modified/completed) after `last_synthesized`.
Report `{ explainer, newSlugs }`. Nothing is written.

### Amendment (`--amend <slug>`)
1. Read the existing explainer; **split** it into the generated body (above
   `## Developer Notes`) and the Developer Notes section (kept verbatim).
2. Re-assemble the **expanded** cluster (`synthesized_from` ∪ newSlugs) via #2.
3. Produce the amended body:
   - **LLM path:** prompt with the *current generated body* + the new material,
     instructed to **add new behavior and strike/correct only what the new specs
     contradict**, leaving everything else intact. (Amend, not rewrite.)
   - **No-key path:** write an amend-scaffold = current body + the new material in
     a guidance comment for an agent/human to fold in.
4. Re-attach Developer Notes verbatim. Update frontmatter: `synthesized_from`
   expanded, `last_synthesized` bumped. Git holds the prior version.

### Ownership invariants
- `## Developer Notes` is never read or written by amendment.
- The system only **adds or strikes/annotates** generated content (visible in the
  diff); it never silently deletes. Humans may hard-delete anything.
- Amendment reads the current on-disk doc, so human edits/deletions persist.

## Acceptance Criteria

- WHEN a completed spec joins an explainer's cluster after `last_synthesized`
  THE SYSTEM SHALL report that explainer as stale via `hero synthesize --stale`,
  naming the new specs.
- WHEN a user runs `hero synthesize --amend <slug>` THE SYSTEM SHALL preserve the
  existing `## Developer Notes` section verbatim in the output.
- WHEN amending THE SYSTEM SHALL base the amended body on the current on-disk
  content (so human edits persist), not a from-scratch rebuild.
- WHEN amending THE SYSTEM SHALL expand `synthesized_from` to include the joined
  specs and set `last_synthesized` to the amendment date.
- THE SYSTEM SHALL NOT read or write content below `## Developer Notes` during
  amendment.

## Boundaries / Out of scope

- **Automatic OnWrite trigger is deferred** — `synthesis-maintenance` (the hook
  owner) is unbuilt. `--stale`/`--amend` are manual/headless entry points; wiring
  the hook is a follow-up once that feature lands.
- No inline pin markers (still deferred to the initiative's out-of-scope list).
- No tombstones — unnecessary under amend-not-regenerate.

## Dependencies

- **Depends on `fks-on-demand-synthesizer` (#2)** — reuses assembly + render.
- **Related to `synthesis-maintenance`** — the eventual `OnWrite` trigger owner.

## Delivery notes

- 2026-06-23 — Delivered. `internal/synthesize/amend.go`: `StaleExplainers`
  (cluster membership via relations + shared parent, gated on completed +
  newer than `last_synthesized`), `SplitDeveloperNotes`/`StripFrontmatter`,
  `AmendPrompt`/`RenderAmended`/`AmendScaffold`, `AmendTargets`. CLI
  `hero synthesize --stale` (lists stale explainers + the new specs) and
  `--amend <slug>` (LLM amends the current body + new material, or scaffolds
  when no key; Developer Notes spliced back verbatim; `synthesized_from`
  expanded; `last_synthesized` bumped). Verified e2e on the real repo: the
  cold-start explainer flagged stale (2 new cst specs), `--amend` expanded
  provenance and preserved Developer Notes. Unit tests cover split/strip,
  staleness (stale + current), and dev-notes preservation.
- **Deferred (honest scope):** the automatic `OnWrite` trigger — its owner,
  `synthesis-maintenance`, is unbuilt. `--stale`/`--amend` are the manual /
  headless entry points until that lands.
- Human edits persist because amendment reads the *current on-disk* body;
  free deletion sticks (no tombstones) under amend-not-regenerate.
