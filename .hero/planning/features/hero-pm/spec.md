---
title: Hero PM — Product Management Domain Pack
slug: hero-pm
type: feature
status: planning
priority: P0
tags: [platform, domains, product-management, roadmap, content-pack]
created: 2026-05-15
relations:
  - target: hero-domains
    kind: parent
depends-on:
  - domain-plugin-architecture
  - spec-type-registry
  - domain-routing-and-agents
  - dashboard-view-registry
  - inline-propose-output-mode
  - scan-pluggability
  - domain-scoped-knowledge-graph
horizon: next
smoke: deferred
---

> **Status: awaiting platform primitives.** This stub is a `/design`-ready
> brief, not a complete design. Implementation is blocked on work items
> 1–6 of the parent `hero-domains` initiative plus the inline-propose
> primitive (#4b): `domain-plugin-architecture`, `spec-type-registry`,
> `domain-routing-and-agents`, `dashboard-view-registry`,
> `inline-propose-output-mode`, `scan-pluggability`,
> `domain-scoped-knowledge-graph`. `/design hero-pm` can resolve the
> open questions below and produce the full design in parallel with
> primitive delivery, but no code lands until the primitives are in
> place.

## Kickoff

First non-engineering Hero domain pack. PM-shaped spec types (PRD, story, epic, roadmap-item, intake-item) with methodology-preset overlays, PM agents and workflows, seven dashboard views (Roadmap default + Story queue + PRD editor + Intake funnel + Story detail + Handoff stream + Chat). IDE-style layout: left nav, tabbed center pane, bottom strip (artifact-state actions + chat input), toggleable right panel (chat with sticky ambient smarts at top). Reuses existing tracker integrations (Jira/Linear/GitHub). Killer demo: a Jira epic becomes a Hero story, `/design` on the story produces an engineering `feature` spec, and the handoff edge surfaces in the Handoff stream live.

**Status:** planning — design pass advanced 2026-05-16. Strategic frame locked (layered presets, five PM principles, silo-tearing thesis). Layout grammar locked (design vs housekeeping modes; bottom strip = verbs; ambient = smarts; chat as right panel + Chat tab for design mode). Research brief + mockup brief + agent/skill/command pack design + handoff-to-hero-code written (27 agents, 32 skills, 22 commands; 13 P0 / 9 P1 / 5 P2). Methodology coaching ships as skills, not agents. Still blocked on primitives 1–6; `/design` still required to lock the canonical Changes section.

**Pick up at:** Run `/design hero-pm` to resolve the open questions and produce the canonical Changes section. The research brief, mockup brief, agent-pack design, and handoff-to-hero-code are sibling files — all four must be incorporated by the design pass, not re-derived.

→ `/design hero-pm`

**Files:** .hero/planning/features/hero-pm/spec.md, .hero/planning/features/hero-pm/research-brief.md, .hero/planning/features/hero-pm/mockup-brief.md, .hero/planning/features/hero-pm/agent-pack-design.md, .hero/planning/features/hero-pm/handoff-to-hero-code.md, .hero/planning/initiatives/hero-domains/spec.md
**Skip:** New tracker integrations in v1 (reuse Jira/Linear/GitHub only). OKRs as a PM spec type (defer to v2; may belong in a separate `strategy` domain). Methodology as a "mode" (not a mode — a layered preset). Forced point estimation.

## Goal

Ship the first non-engineering Hero domain pack: Product Management.
Provides PM-shaped spec types (PRD, story, epic, roadmap-item), PM
agents and workflows, PM-specific dashboard views, and onboarding that
imports an existing roadmap or tracker into a queryable spec corpus.
Reuses existing tracker integrations (Jira/Linear/GitHub) — no new
integrations in v1. Success means a PM can take a Jira epic into Hero,
refine it into stories with acceptance criteria, hand a story off to
engineering as a `feature` spec, and see the handoff edge appear in the
knowledge graph.

## Artifact types

The pack declares the following spec types via the spec-type registry.
Each type has its own lifecycle and frontmatter schema. Artifacts are
universal — every PM team has them. Process layers (see Methodology
layers below) overlay methodology-specific fields and states onto these
types without redefining them.

| Type | Purpose | Lifecycle | Process-layer behavior | Notes |
|---|---|---|---|---|
| `prd` | Product requirement doc | draft → review → approved → delivered | No new fields; under cycle preset the default template emphasizes Shape Up's Appetite/Rabbit Holes/No-Gos. | Largest artifact; references stories and roadmap-item. Notion-fidelity editor. Pitch-shaped default template. |
| `story` | User story / dev-ready unit | drafted → refined → ready → in-flight → done | Sprint preset adds `sprint`, `points`. Cycle preset adds `cycle`, `hill_position`. Kanban preset adds `wip_age`. Phased preset adds `release`/`phase`. Horizon preset is orthogonal — no story-level field. | Handoff atom into `/design` and `/deliver`. The `/design`-on-story interaction is the platform's killer demo. |
| `epic` | Container for related stories | proposed → committed → in-flight → done | Sprint preset adds rollup `velocity_target`. Cycle preset can flip an epic into a "bet" (one cycle, fixed appetite). Phased preset assigns to a `release`. | Maps to tracker epic. |
| `roadmap-item` | Coarse-grained future bet | candidate → committed → shipped → dropped | Horizon preset (almost always on) adds `horizon: now\|next\|later` or a quarter (`q3-2026`). Cycle preset adds `appetite` (1w / 2w / 6w). Phased preset adds `target_release`. | Carries the prioritization weight rolled up from linked intake-items. |
| `intake-item` _(new)_ | Inbound feedback / request / signal | new → triaged → linked → rejected | No preset-specific fields. Source attribution (`source`, `source_url`, `customer_segment`) is universal. | Promotes-to / merges-with a `roadmap-item`. The Productboard-shaped surface. Was missing from the prior table; added because Intake funnel screen needs it as a real spec type. |
| `okr` _(deferred)_ | Objectives + KRs | active → closed | Out of v1 scope. | Deferred to v2; possibly a separate `strategy` domain. |

Process-layer fields are declared by the active preset, not by the spec
type. The spec-type registry must accept layered-field declarations
from preset config — see the Methodology layers section below for the
required shape.

## Methodology layers

Methodology in Hero PM is **layered presets**, not modes. Artifacts
(`prd`, `story`, `epic`, `roadmap-item`, `intake-item`) are universal.
Process layers overlay methodology-specific fields, lifecycle hints,
and dashboard rollups onto those artifacts. Switching presets changes
boards, cadences, and rollups — never the underlying spec types or
the on-disk shape.

Teams pick:

- **One roadmap layer** (almost always one: Horizon).
- **One delivery layer** (Continuous flow / Sprint-based / Cycle-based
  / Phased).
- **Optional milestone overlay** (release groupings, gated phases).

This is the answer to "everyone is a hybrid." Most teams run a horizon
roadmap layer *plus* a delivery cadence; some add a release/milestone
overlay. The preset config in `hero.json` (under the `pm:` block)
declares which presets are active.

### Supported presets (v1)

| Preset | Layer | Adds to story | Adds to roadmap-item | Dashboard impact |
|---|---|---|---|---|
| **Horizon roadmap** | Roadmap | — | `horizon: now\|next\|later` or quarter | Roadmap board defaults to a horizon grouping. |
| **Continuous flow (kanban)** | Delivery | optional `wip_age` | — | Story queue shows Icebox / Backlog / Current / Done bands; no sprint/cycle markers; WIP-aging visual cue. |
| **Sprint-based (scrum / scrumban)** | Delivery | `sprint`, `points` | — | Story queue shows a "fits this sprint" cut line; sprint picker in chrome; velocity rollup on epic. |
| **Cycle-based (Shape Up)** | Delivery | `cycle`, `hill_position` | `appetite` (1w / 2w / 6w) | Roadmap shows betting-table layout; Story detail shows hill chart; cooldown is a first-class state. |
| **Phased / milestone** | Delivery (or overlay) | `release`, `phase` | `target_release` | Roadmap groups by release; Story queue gains a phase filter; gates render between phases. |

### Preset switching

- Preset choice lives in `hero.json` under `pm.presets`, e.g.
  `{ "roadmap": "horizon", "delivery": "cycle", "overlay": null }`.
- Switching a preset is a config edit + dashboard reload. No data
  migration — preset-specific fields are written when their preset
  is active and ignored otherwise.
- Switching off a delivery preset leaves its fields on existing
  artifacts (no destructive cleanup); switching back picks them up
  again.

### Design implications

- The dashboard view registry (parent primitive #4) must accept
  preset-conditional view variants. Roadmap board renders
  differently under cycle preset (betting table) than under
  horizon preset (Now/Next/Later grid).
- The spec-type registry (primitive #2) must accept preset-declared
  optional fields without making them required.
- `/refine` and `/triage` skills must read the active preset to
  decide what fields to populate (e.g. `/refine` under sprint
  preset prompts for points; under cycle preset prompts for
  appetite).

## Guiding principles

Hero PM is shaped by five product-management principles. Every dashboard
view, command, and agent must earn at least one. Screens that don't map
cleanly to a principle should be cut.

| # | Principle | Earned by |
|---|---|---|
| 1 | **Decide what's worth building** — prioritize against opportunity, capacity, and strategy. | Roadmap board, Intake funnel, prioritization-framework views (RICE / value-vs-effort). |
| 2 | **Define it clearly** — capture the *what* precisely enough that engineering can execute without ambiguity. | PRD editor, Story detail (acceptance criteria, EARS), `/refine` skill. |
| 3 | **Make trade-offs visible** — what's being deferred, why, what changing course costs. | Roadmap board's deferred lane, Story detail's "Skip:" line in kickoff, `hero why` on roadmap-items, Intake funnel's "rejected with reason." |
| 4 | **Maintain alignment** — same page for eng, sales, leadership, customers about what's coming and why. | Roadmap presentation view, Cross-domain handoff stream, evidence counts on roadmap-items. |
| 5 | **Learn from what shipped** — validate the thing moved a needle; feed back into priorities. | Handoff stream's "shipped" rows linking back to roadmap-items, retro hooks on completed stories, deferred metric/experiment spec types (`hero-data-analytics` initiative). |

Principle #5 is partially deferred — full metric/experiment specs live
in the future `hero-data-analytics` domain. v1 of `hero-pm` earns #5
by linking shipped stories back to their parent roadmap-item with a
"shipped on" timestamp and an optional retrospective note.

## Silo-tearing patterns

This is the differentiator no existing PM tool has, because none of
them own engineering. PM artifacts are natively visible inside
engineering surfaces, and engineering artifacts are natively visible
inside PM surfaces — through the shared knowledge graph, not via
sync.

### Cross-domain hooks

- **`hero search` returns cross-domain results.** From a PM session,
  searching for "csv export" returns both the PM story and the
  engineering feature spec it handed off to. Active-domain results
  rank first; cross-domain results are clearly marked. Requires
  domain-scoped graph namespacing (primitive #6) to render the
  boundary.
- **`hero why feature-X` walks story → epic → roadmap-item.** From
  an engineering session, asking "why does this feature exist" walks
  the cross-domain edge back to the PM story, then up to the epic
  and roadmap-item that originated the work. The chain of
  decisions is contiguous across the domain boundary.
- **`/diagnose` can link a regression to its originating story.** A
  bug spec discovered during `/diagnose` can be linked to the
  story whose delivery introduced the regression. Closes the loop
  from "feature shipped" → "regression filed" → "the bet that
  produced both."
- **PM Handoff stream pulls live delivery status from engineering.**
  No separate sync. The Handoff stream's "delivery state" column
  reads from the engineering side of the same graph in the same
  session. When an engineering feature flips to `delivering`, the
  PM Handoff stream row reflects it on next render.

### The hand-off interaction

Clicking **"Hand off to /design"** on a Story detail is the platform
thesis in one click. The interaction must:

1. Open `/design` with the story's content as input.
2. Produce an engineering `feature` spec via `feature-delivery-lead`.
3. Write a cross-domain edge (`story → feature`, `kind: handoff`)
   into the knowledge graph.
4. Surface the new feature on the Story detail page within the same
   session — the "linked engineering feature" rail is no longer
   empty.
5. Add a row to the Cross-domain handoff stream.

The button is the single most important interaction in the product.
Mockups must make it visually prominent and the corresponding
"linked engineering feature" rail must be unambiguous when present.

### Cross-domain edge semantics

Resolved in primitive #6 (`domain-scoped-knowledge-graph`). The
`hero-pm` design pass must align with whatever edge kind that
primitive lands on (regular edge with cross-namespace endpoints, or
dedicated `cross-domain-handoff` kind). v1 mockups can assume the
edge is queryable and renderable; whether it carries a special
kind tag is a primitive-#6 concern.

## Design philosophy

The headline UX target for `hero-pm`:

> *Pivotal Tracker's flow-first philosophy + Linear's speed + Shape Up's
> pitches + Notion's doc fidelity — all consumable by engineering
> without leaving engineering's tools.*

Full design philosophy bullets and the per-tool steal/leave matrix
live in `research-brief.md` (sibling file). The eight-bullet philosophy
distilled there is authoritative; the mockup brief (`mockup-brief.md`)
operationalizes it into six killer screens.

## Workflows

1. **Intake** — capture inbound (customer feedback, internal asks,
   competitive signals); triage into a roadmap-item or reject with reason.
2. **Refinement** — turn a roadmap-item into a PRD and child stories;
   apply INVEST shape and explicit acceptance criteria on each story.
3. **Prioritization** — sequence roadmap-items against capacity,
   dependencies, and OKRs.
4. **Roadmap maintenance** — keep roadmap status fresh from delivery
   signal (in-flight, shipped, dropped).
5. **Handoff to engineering** — `/design` on a `story` produces an
   engineering `feature` spec; the knowledge graph records the handoff
   edge. This is the killer demo of the multi-domain platform.

## Agent pack

The full agent / skill / command pack design lives in the sibling
file `agent-pack-design.md`. It defines 27 agents, 32 skills, and
22 commands organized to match engineering's depth (engineering ships
34 agents / 45 skills / 27 commands), plus a natural-language routing
table and a contextual-button inventory for every PM artifact.

Methodology coaching (Continuous Discovery, Shape Up) ships as
**skills** loaded by `pm-delivery-lead` and authoring agents on
demand — not as standalone coach agents. Skills teach, agents act.

Tier summary:

| Tier | Count | P0 / P1 / P2 |
|---|---|---|
| Coordination | 2 | 2 / 0 / 0 |
| Strategic | 5 | 2 / 2 / 1 |
| Authoring | 5 | 3 / 2 / 0 |
| Triage / curation | 3 | 2 / 1 / 0 |
| Prioritization | 3 | 1 / 2 / 0 |
| Coordination-delivery | 3 | 1 / 2 / 0 |
| Review | 3 | 1 / 1 / 1 |
| Scrubbers | 3 | 0 / 2 / 1 |
| **Total** | **27** | **13 / 9 / 5** |

The P0 roster (the minimum-viable pack v1 must ship to earn its
place — see `agent-pack-design.md` §H):
`pm-delivery-lead`, `pm-investigator`, `product-strategist`,
`discovery-researcher`, `prd-author`, `story-writer`,
`roadmap-curator`, `intake-triager`, `duplicate-detector`,
`prioritization-strategist`, `handoff-coordinator`, `pm-reviewer`,
plus the supporting v1 skill set (~18 skills) and command set (12).

The single most important agent in the pack is `handoff-coordinator`
— it executes the `story → feature` cross-domain handoff and is the
brand interaction of the platform.

### Agents (summary)

See `agent-pack-design.md` §C for the full roster with role,
when-invoked, what-produced, skills, delegation graph, prompt sketch,
and prior-art attribution per agent.

### Skills (summary)

See `agent-pack-design.md` §D for the 32 skills organized across
Writing (9), Frameworks (5), Process/methodology (6), Curation (6),
Cross-domain (3), and Operational (3). Each entry documents when
invoked, core content, anti-patterns, and prior-art source.

### Commands (summary)

See `agent-pack-design.md` §E. 14 PM-specific new commands plus 8
reused/cross-domain commands. The natural-language routing table
(mirroring engineering's CLAUDE.md style) is in §F.

New PM-specific commands: `/refine`, `/triage`, `/roadmap`,
`/prioritize`, `/pitch`, `/prd`, `/handoff`, `/discover`,
`/metrics`, `/interview`, `/release-notes`, `/capacity`,
`/plan-cycle` / `/plan-sprint` / `/plan-iteration`, `/standup`,
`/scrub roadmap` / `/scrub intake` / `/scrub stories`.

Reused commands (accepting PM artifact types via the
domain-routing-and-agents primitive): `/design`, `/deliver`,
`/diagnose`, `/search`, `/why`, `/blocked`, `/note`, `/decide`,
`/review`.

## Integrations

**v1 reuses existing tracker integrations (Jira, Linear, GitHub) —
no new integrations.** Importing Jira epics as `epic` specs and Jira
stories as `story` specs is the v1 onboarding moment.

This is an explicit decision, not a punt: it lets PM ship on the same
integration surface engineering already has, and stress-tests whether
the `DomainIntegration` interface tolerates two domains sharing one
provider before we add roadmap-shaped providers (Productboard, Aha) in
a follow-up.

**Tracker-fronting is local-first.** In tracker-fronted mode the
working surface is identical to standalone — instant local writes,
async propagation to the tracker, no syncing spinners. Conflict
policy: Hero wins for content (PRD body, AC, story description),
tracker wins for org-state (assignee, sprint, workflow status). PM
spec-type frontmatter must distinguish content fields from org-state
fields so the integration layer can apply the right policy on each.
See
[tracker-fronting-and-local-first](../../knowledge/decisions/tracker-fronting-and-local-first.md).

## Dashboard layout

The PM dashboard uses an IDE-style layout with two work modes: **design**
(chat is the work) and **housekeeping** (artifact is the work). Mode-
switching is tab-switching. Chat doesn't compete with artifacts for
screen space.

| Surface | Role | Word type |
|---|---|---|
| **Left nav** (~220–240px) | Inventory of openables — Roadmap, Library, Intake, Queue, Stream, Tracker mirrors, **Chat** (prominent, bolt icon, near top). Saved views nested as parameterized singletons under their parent. | Navigation |
| **Center pane** | Tabbed openables (VS Code-style). Singleton tabs + per-item tabs + the Chat tab. | Content |
| **Artifact header** | Identity / state chips — status, owner, priority, sprint/cycle, points, tracker link, tags. | State |
| **Artifact body** | The thing itself + load-bearing inline elements (linked-engineering-feature card with the `Hand off to /design` button, hill chart inline for cycle preset, AC list, PRD sections). | Content |
| **Bottom strip** (~70–90px, anchored to artifact tabs only) | State-aware contextual buttons (verbs) + chat input + minimal chrome (model chip, send). Not present on the Chat tab. | Actions |
| **Right panel** (~360–400px, toggleable; default open on first run) | Single rolling chat with **sticky pinned ambient region** at the top (smarts about the active artifact) + chat scroll below (hero-code visual fidelity). Sticky region updates on artifact switch; chat thread is continuous. Empty ambient state renders nothing. | Smarts + history |

**The bedrock distinction**: bottom strip is **verbs** (what the user
does). Ambient region is **nouns / smarts** (what the agent surfaces
about the artifact). Header chips are **identity / state** (what the
artifact is). These three roles never blur.

## Dashboard views

The pack registers seven views via the dashboard view registry. The
default landing page is Roadmap. Detail-level layout, sample content,
interactions, and methodology-preset variations are specified in
`mockup-brief.md` and reflected in the mockups under `mockups/`.

| View | Tab kind | Default landing? | Earns principle | Primary influences |
|---|---|---|---|---|
| **Roadmap board** | singleton | yes | 1 (decide), 3 (trade-offs), 4 (align) | ProductPlan / Roadmunk drag UX, Shape Up betting table (cycle preset), Productboard evidence counts |
| **Story queue** | singleton | — | 1 (decide), 2 (define) | Pivotal Tracker single-list flow, Linear keyboard/density, Kanbanize WIP aging |
| **PRD editor** | per-item | — | 2 (define) | Notion markdown-with-slash, Shape Up pitch sections, Aha goal-context strip |
| **Story detail** (with `Hand off to /design` inline) | per-item | — | 2 (define), 4 (align) | Notion description, Shape Up hill chart inline, Height ambient AI in right-panel ambient |
| **Intake funnel** | singleton | — | 1 (decide), 3 (trade-offs) | Productboard inbox + highlight-to-link, Linear Triage view |
| **Cross-domain handoff stream** | singleton | — | 4 (align), 5 (learn) | Linear cycle timeline, Kanbanize cycle-time histogram, Hero-shaped delivery rail |
| **Chat** | singleton | — | 1 (decide), 2 (define) — the design-mode surface | hero-code chat tab — full provider/model picker, context fullness, attachments, mentions, slash, reasoning. Single rolling conversation viewed full-screen here; slim version visible in right panel during housekeeping. |

All artifact and workspace surfaces must adapt their rendering to the
active methodology preset (see Methodology layers above). Roadmap board
has the largest preset-driven variance (Now/Next/Later vs betting table
vs quarterly); Story queue gains/loses the cut line; Story detail
gains/loses the hill chart.

### Contextual buttons (bottom strip)

Per the locked UX, every artifact surfaces a state-aware set of 4-6
contextual buttons in the **bottom strip** anchored to the artifact's
center pane (~70-90px tall). Buttons fire commands (typically with an
`--inline-propose` flag so the output lands in the artifact pane as
proposed content with accept/edit/reject controls) and are the
primary way agents are invoked outside slash commands and direct
chat-input prompts. The button set varies by artifact state — a drafted
story shows different actions than a ready story.

The bottom strip also contains the chat input (continuing the same
single rolling conversation visible in the right panel and Chat tab)
and minimal chrome (model chip, send affordance).

The full per-artifact inventory (story, PRD, epic, roadmap-item,
intake-item, handoff-stream row) lives in `agent-pack-design.md` §G.
Highlights:

- **Story:** `Hand off to /design` *(brand button)*, `Refine`,
  `Draft AC`, `Find duplicates`, `Find similar stories`,
  `Show dependencies`, `Review`.
- **PRD:** `Approve`, `Suggest AC`, `Find related decisions`,
  `Summarize for standup`, `Refine section`, `Convert to pitch`.
- **Roadmap-item:** `Promote to active`, `Draft PRD`, `Write Pitch`
  (cycle preset), `Show evidence`, `Prioritize`, `Reject with reason`.
- **Intake-item:** `Link to existing`, `Promote to roadmap-item`,
  `Reject with reason`, `Find duplicates`, `Triage`.
- **Handoff-stream row:** `Open story`, `Open feature`, `Re-handoff`.

### Ambient region (sticky top of chat scroll)

Distinct from the bottom strip's actions, the right panel's sticky top
holds **smarts about the artifact** — not actions on it. Surfaced by
agents (proactively, not on user request), with accept/dismiss/customize
controls inline on each card:

- **Hard relationships** (when they exist): parent epic, linked
  engineering feature + delivery status, related decisions.
- **Dynamic agent suggestions**: similar-item detection ("Looks like
  Story #234 — merge?"), possible parents ("Might belong to: Epic
  'Auth Refresh'"), completeness diagnostics ("Missing: description,
  AC").
- **Activity rollup**: "3 events →" expandable.
- **Starter helpers** (only when the artifact is new/empty): template
  pickers, quick-draft offers.

Max 3-4 cards visible at once. Dismissed suggestions persist per-artifact
until state genuinely changes. Empty ambient state renders nothing —
the chat scroll starts at the top of the panel cleanly.

The agent that drives ambient suggestions on artifact open is
`pm-investigator` (see `agent-pack-design.md` §C). Specific suggestion
categories map to specific agents (`duplicate-detector` for similar-item
detection, `dependency-mapper` for parent-suggestion, etc).

## Unknowns for design pass

These must be resolved during `/design hero-pm` before delivery:

1. **OKRs — PM domain or separate `strategy` domain?** Defer
   recommendation; design decides whether `okr` lives here or in a
   future strategy pack.
2. **Cross-tracker handoff** — does the platform need to support a PM
   project on Jira handing off to an engineering project on Linear?
   Plausible, adds scope. Decide in/out for v1.
3. **Domain coexistence model** — should PM and engineering live in
   the same workspace (one `.hero/` dir, two active packs) or as
   separate projects sharing a knowledge graph? Affects whether the
   graph namespace tags from `domain-scoped-knowledge-graph` (item #6)
   are required for v1.
4. **Story acceptance criteria format** — free-text, Gherkin, or
   structured frontmatter? Engineering uses EARS today; PM stories
   need a compatible-enough shape that `/design` on a story produces a
   useful feature spec.
5. **Roadmap horizon model** — time-based (Q1/Q2/Q3) or ordered-bets
   (now/next/later)? Affects the Roadmap view, the roadmap-item
   lifecycle, and `roadmap-curator`'s prioritization signal.

## Boundaries

- **Not** designing OKR support in v1 — see unknown #1.
- **Not** adding new integration providers in v1 — Jira/Linear/GitHub
  reuse only.
- **Not** building a PM-only Hero binary — same `hero` binary, domain
  selected via `hero.json`.
- **Not** building cross-domain reporting (PM-eng combined dashboards)
  — that's a follow-up after both domains ship.
- **Not** modeling product analytics, experiment results, or metrics
  pipelines — those belong to a future `hero-data-analytics`
  initiative.

## Risks

- **PM-as-first-domain feels like incremental engineering.** Mitigated
  by making the handoff edge (story→feature) a hero demo, and by
  planning `hero-qa` (item #8 in parent) on a real cadence.
- **Spec-type registry surface area surprises.** If item #2 of the
  parent initiative underestimates the audit, PM design and delivery
  slip with it.
- **Tracker reuse hides design tension.** Reusing Jira for PM and
  engineering is right for v1 but masks integration-shape questions
  that the QA pack and later domains will surface.
