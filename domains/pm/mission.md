---
title: Hero PM — Product Management Vertical Charter
type: mission
scope: vertical
vertical: pm
inherits: ../../.hero/mission.md
locked_at: 2026-05-16
locked_by: chet-bellows
version: 1
---

## Mission

**Hero PM is the product-management vertical of Hero — the sidekick
brain for AI-driven product work.**

It rides on [Core Hero](../../.hero/mission.md) and adds the spec
types, agents, skills, commands, and vocabulary that turn the core
engine into a complete intake + discovery + roadmap + PRD + story +
handoff toolkit for product managers.

The core mission applies unchanged: the model in the PM's harness
starts cold; Hero captures everything that happens during the work
and injects it back automatically; sessions start smart, end smarter,
the floor rises for everyone — first PM hire and seasoned director
alike.

What this vertical adds: the **shape** of the work, in product terms.
*Decide what's worth building*. *Define it clearly enough that
engineering can execute without ambiguity*. *Make tradeoffs visible*
— what's being deferred, why, what changing course costs. *Maintain
alignment* across eng, sales, leadership, and customers about what's
coming and why. *Learn from what shipped* — validate the thing moved
a needle and feed back into priorities.

## What this vertical brings

| Layer | What's in it |
|---|---|
| **Spec types** | **Shared** registered spec types: `feature`, `epic`, `initiative` — used by both PM and engineering under the unified type model. **PM-led** registered spec types: `prd`, `intake`. The PM-common type is `feature` (rendered as "Story" / "Scope" / "Card" depending on active vocabulary); engineering picks up the same feature via the owner-flip workflow. |
| **Agents** | `pm-delivery-lead`, `pm-investigator`, `product-strategist`, `discovery-researcher`, `prd-author`, `story-writer`, `roadmap-curator`, `intake-triager`, `duplicate-detector`, `prioritization-strategist`, `handoff-coordinator`, `pm-reviewer` (P0), plus P1/P2 expansions — see [`agents/`](agents/) |
| **Skills** | writing (`story-writing-invest`, `acceptance-criteria-ears`, `prd-structure`, `pitch-writing-shape-up`, …), frameworks (`prioritization-frameworks`, `opportunity-solution-trees-torres`, `metrics-design`), process (`continuous-discovery-cadence`, `sprint-planning`, `cycle-planning`), curation (`intake-classification`, `duplicate-detection`, `dependency-mapping`, `evidence-synthesis`), cross-domain (`handoff-protocol`, `cross-domain-graph-query`), operational (`pm-preset-detection`) — see [`skills/`](skills/) |
| **Commands** | `/refine`, `/triage`, `/roadmap`, `/prioritize`, `/prd`, `/pitch`, `/handoff`, `/discover`, `/metrics`, `/release-notes`, plus reused `/why`, `/search`, `/note` — see [`commands/`](commands/) |
| **Methodology layers** | Roadmap layer (Horizon) + delivery layer (Continuous flow / Sprint / Cycle / Phased) + optional milestone overlay. Switched in `hero.json` under `pm.presets`. Independent of vocabulary preset (`vocabulary:` in `hero.json`). |
| **Vocabulary** | `default` / `agile-scrum` / `shape-up` / `kanban` / `jira` / `linear` — display-name presets that render canonical `feature / epic / initiative` as "Story / Epic / Theme", "Scope / Pitch / Bet", etc. User-overridable via `vocabulary_overrides`. |
| **Trackers** | Jira, Linear, GitHub — read/write via the active vocabulary's `tracker_mappings` block (no separate registry hardcoded switch). |
| **Killer demo** | A Jira epic becomes a Hero `epic`; a PM refines a child `feature`; the user clicks "Hand off to engineering" and `owner` flips `pm → engineering` on the **same** feature; engineering's `engineer` agent picks it up in the same workspace; the right-rail "linked engineering work" disappears entirely because it's the same artifact. The brand interaction is the moment the feature transitions surfaces without changing identity. |

## How it specializes the core

The core mission test asks: *"Does this make the next agent session
start smarter than the last one ended?"* For Hero PM, "next agent
session" means *the next product session — triaging inbound feedback,
shaping a PRD, ranking the next quarter, refining a story for
delivery, presenting the roadmap, debriefing what shipped.* Every
Hero PM feature must answer: does a product session start with the
right context loaded — open initiatives in flight, recent decisions
about tradeoffs, intake clustered by theme, stories ready to hand
off, what engineering shipped this week, what didn't move the needle
last quarter?

## Vocabulary additions (PM-specific)

These extend (never override) the core vocabulary.

- **PRD** — product requirement document; the largest authoring
  artifact. Pitch-shaped by default under cycle preset; ten-section
  shape under sprint/flow presets.
- **feature** — the canonical dev-ready unit (`type: feature`). PM
  authors features by default; engineering may originate `bug`,
  `chore`, etc. as their own work types. Vocabulary preset renders
  the display name (Story / Scope / Card / Issue / Feature).
- **epic** — mid-tier grouping; a set of features that go together to
  achieve a large outcome with many parts. Shared between PM and
  engineering. Optional parent initiative.
- **initiative** — coarse-grained aspirational bet; anchors PRDs;
  carries prioritization weight rolled up from linked intake.
- **intake** — inbound feedback / request / signal; promoted-to
  or merged-with an initiative, or rejected with reason.
- **handoff (owner flip)** — the PM → engineering boundary crossing.
  Under the unified type model, the same artifact carries through; only
  the `owner` field flips (`pm → engineering`), recorded bitemporally
  in `owner_history`. No new spec is created; no separate graph edge
  is written.
- **kind** — first-class sub-type field. Canonical values are
  methodology-neutral (`theme, delivery, bet, milestone` on `epic`;
  `now, next, later` on `initiative`). Vocabulary preset translates
  canonical to display.
- **preset** — two independent systems. **Methodology preset**
  (`pm.presets` in `hero.json`: horizon / sprint / cycle / phased)
  drives lifecycle behavior. **Vocabulary preset** (`vocabulary` in
  `hero.json`) drives display names only.
- **discovery** — Teresa Torres-style continuous research; reduces
  uncertainty before authoring.
- **intake** — inbound signal capture; the Productboard-shaped
  surface.

## Anti-patterns specific to this vertical

In addition to core anti-patterns, Hero PM must never become:

- **A tracker replacement.** Jira/Linear/GitHub are systems of record
  for org-state (assignee, sprint, workflow status). Hero is the
  working surface — content (PRD body, AC, story description) wins
  locally; org-state wins from the tracker. The integration layer
  routes by field.
- **A roadmap presentation tool.** Roadmaps live in the working
  surface where the PM authors, prioritizes, and reconciles them
  against delivery — not in a static slide deck.
- **A PM-replacement.** The PM owns the decisions. Hero feeds them
  context, drafts, and pattern recognition. Direct mirror of Hero
  Code's "we don't write code" — we don't make the call.
- **A methodology cult.** Continuous Discovery, Shape Up, RICE, and
  the rest are *tools*, surfaced as skills. The pack supports them
  via layered presets; it does not force a methodology on a team.
- **A roadmap that lies.** Stale `now` items, `committed` items with
  no live delivery, "shipped" claims with no graph edge — these are
  failures. `roadmap-curator` reads live engineering delivery state
  from the cross-domain graph; the roadmap reconciles to reality.

## Silo-tearing — the differentiator

No existing PM tool owns engineering. Hero does. The differentiator
is that PM artifacts are natively visible inside engineering surfaces,
and engineering artifacts are natively visible inside PM surfaces —
through the shared knowledge graph, not via sync.

- `hero search` returns cross-owner results — search "csv export"
  from a PM session and engineering-owned specs of the same type
  show up alongside PM-owned ones; the `owner` field makes the
  boundary visible without splitting the type.
- `hero why <spec>` walks feature → epic → initiative plus the
  bitemporal `owner_history` — from any session, asking "why does
  this spec exist" walks the lineage back through both the parent
  hierarchy and the cross-domain ownership transitions.
- `/diagnose` runs on a `bug` regardless of owner —
  closing the loop from "shipped" → "regression filed" → "the bet
  that produced both" using the same artifact type.
- The PM Handoff stream pulls live delivery status from engineering
  via `owner_history` queries — no separate sync; no separate edge
  kind; same graph, same session, same spec.

## Why this vertical is the first non-engineering pack

Hero PM is the **first non-engineering domain** to land. It earns
this slot for three reasons:

1. **Reuses existing tracker integrations.** Jira/Linear/GitHub
   already work; PM ships on the same providers. No new integration
   surface in v1.
2. **Validates the multi-domain architecture end-to-end.** PM
   exercises every platform primitive — pack loading, spec-type
   registry, domain routing, dashboard view registry, scan
   pluggability, and the cross-domain graph (via the killer demo).
3. **The handoff is the brand interaction.** Story → feature is the
   single most visible cross-domain moment in the product. Shipping
   it first proves the thesis.

## Inheritance discipline

When [`.hero/mission.md`](../../.hero/mission.md) (the core charter)
changes, this vertical charter is reviewed for compatibility within
the same PR. Vertical charter cannot weaken or contradict the core;
it can only specialize.
