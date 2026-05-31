---
title: Superseded Is Genealogy, Not Status
type: decision
status: proposed
created: 2026-05-30
tags: [retrieval, indexing, frontmatter, archive, genealogy, principle]
relations:
  - target: superseded-specs-soft-archive
    kind: decided-in
---

# Superseded Is Genealogy, Not Status

## Decision

"Was this spec replaced by another?" is a **genealogy** fact, modeled as
a dedicated frontmatter field (`superseded_by: <slug>`), not a value of
the lifecycle `status:` enum.

The existing `StatusSuperseded` enum value remains for backward
compatibility, but new specs and the new `hero supersede` command write
to `superseded_by:` and leave `status:` alone (typically `completed`).

## Rationale

Status and genealogy are orthogonal:

- **Status** says where the spec is in its delivery lifecycle (planning,
  delivering, completed, regressed, ...).
- **Genealogy** says how this spec relates to other specs over time
  (parent, child, supersedes, replaced-by, ...).

Conflating them loses information. A `completed` spec can later be
superseded without un-completing. A `planning` spec can be superseded by
a new direction. A `convention` is normally `active` and can still be
superseded by a newer convention also `active`. Forcing one enum to
carry both axes means you can't represent "this shipped, AND it was
later replaced" cleanly.

Separating them also gives the indexer a single, unambiguous column to
predicate on (`WHERE superseded_by != ''`), keeps the YAML
self-documenting (`superseded_by: <slug>` tells the reader the
replacement; `status: superseded` does not), and means hand-editors only
touch one field.

## Consequences

- Retrieval (search, context-injection) treats `superseded_by != ''` as
  the de-weight / annotate signal — not status.
- The `supersedes:` relation kind continues to exist; the new
  `superseded_by:` field is the *forward* reference, and the indexer
  derives the inverse `supersedes:` edge automatically on ingest. Authors
  only edit one side.
- Render-time banners and search-result annotations carry the
  replacement slug — so even when a stale superseded spec leaks into the
  context window, the model sees where to go instead.
- Graph genealogy is preserved: the superseded node stays in the graph
  with all its edges; only ranking and rendering change. `hero why`
  continues to traverse `supersedes` edges so the trail to "how we got
  here" stays intact.

## Related

- `superseded-specs-soft-archive` — the feature spec that implements
  this decision end-to-end.
- `historical-artifact-isolation` — the broader principle that
  historical artifacts must be isolated from default discovery without
  losing the ability to walk back to them on demand. Superseded specs
  are a specific application of that principle: isolated from default
  retrieval ranking, reachable via `hero why` and explicit
  `--include-superseded` queries.
