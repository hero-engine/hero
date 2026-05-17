# Hero PM — Spec Types

Spec-type schemas in this directory are the **PM-led** artifact definitions,
loaded by the spec-type registry (per primitive #2, `spec-type-registry`)
when the active domain in `hero.json` is `pm`. Under the **unified spec-type
model** (`.hero/planning/features/unified-spec-type-model/spec.md`),
three of the work types are **shared** across domains and live in
`core/spec-types/`: `feature`, `epic`, and `initiative`.

Each schema declares the artifact's lifecycle, the canonical `kind` set
(per Decision 2 of the unified model), required frontmatter fields,
optional preset-conditional fields, and the section structure the
authoring agents target. A `## Tasks` section is parsed identically
to `## Acceptance Criteria` on every work spec-type.

| Type | Location | Purpose | Canonical `kind` |
|---|---|---|---|
| `feature` | `core/spec-types/` | Dev-ready unit (shared PM ↔ engineering; owner-flip handoff) | `feature, bug, chore, refactor, perf, infra, security, ux` |
| `epic` | `core/spec-types/` | Mid-tier grouping (shared) | `theme, delivery, bet, milestone` |
| `initiative` | `core/spec-types/` | Coarse aspirational bet (PM-led, shared visibility) | `now, next, later` (or quarter string) |
| `prd` | `domains/pm/spec-types/` | Product requirement doc (PM-led) | `pitch, ten-section, lightweight` |
| `intake` | `domains/pm/spec-types/` | Inbound feedback / request / signal (PM-led) | `customer, support, sales, internal, competitive` |

Display names render via the active vocabulary preset (e.g. `feature`
renders as "Story" under `agile-scrum`, "Scope" under `shape-up`, "Card"
under `kanban`). The canonical type/kind pair is methodology-neutral.

### Preset-conditional fields

The schemas declare preset-conditional fields per
[hero-pm spec](../../../.hero/planning/features/hero-pm/spec.md)
§Methodology layers. These are written when the corresponding preset
is active in `hero.json` `pm.presets`, and ignored (but preserved)
otherwise. No data migration on preset switch.

### Content vs org-state field tagging

Each field in a PM spec-type is tagged as **content** (Hero wins on
conflict) or **org-state** (tracker wins on conflict), per the
[tracker-fronting decision](../../../.hero/knowledge/decisions/tracker-fronting-and-local-first.md).
The integration layer routes by tag when reconciling local writes
with tracker webhooks/polls.

### OKR (deferred)

`okr` is deferred to v2; may belong in a separate `strategy` domain.
See spec.md unknown #1.
