---
title: Hero PM — Product Management Domain Pack
slug: hero-pm
type: feature
status: designed
priority: P0
tags: [platform, domains, product-management, roadmap, content-pack]
created: 2026-05-15
designed: 2026-05-19
updated: 2026-08-22
relations:
  - target: hero-domains
    kind: parent
  - target: pm-foundation-delivery
    kind: follows
  - target: pm-platform-unblock
    kind: follows
  - target: dual-mode-pm-qa-capability-packs
    kind: related
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

> **Status: design complete (2026-05-19). Delivery blocked on platform primitives.**
> The strategic frame, layout grammar, agent/skill/command pack, view manifest,
> spec-type declarations, routing table, scanner contract, cross-domain edge
> contract, and tracker integration model are all locked. The four sibling
> documents (`research-brief.md`, `mockup-brief.md`, `agent-pack-design.md`,
> `handoff-to-hero-code.md`) and the nine mockups under `mockups/` are the
> source of truth for their respective concerns; this spec references them
> rather than re-deriving their content. Delivery moves to the `hero-code`
> repository once the parent `pm-platform-unblock` sprint's work items W6
> (`domain-scoped-knowledge-graph` Go delivery), W7 (`hero-code-handover-pack`),
> and W8–W11 (any remaining contract reconciliation) land.

## Composition amendment (2026-08-22)

The full PM pack is dual-mode. It can be enabled as a bounded extension
of an engineering workspace or selected as the primary pack in a
dedicated PM workspace. Engineering retains lightweight planning
essentials for ordinary feature work; the full PM pack adds discovery,
roadmaps, portfolio and capacity work, metrics, specialist artifacts,
agents, and PM views. These are activation roles for one PM package, not
separate `pm-lite` and `pm-full` content forks. The composition and
collision contract is owned by `dual-mode-pm-qa-capability-packs`.

## Public-pack delivery boundary (2026-08-22)

The open-source, model-facing PM content is delivered and verified separately
by `pm-public-pack`. This broad spec remains open for the proprietary Hero Code
dashboard and application work described below; those surfaces are not part of
the public repository or its future Apache-2.0 grant.

## Kickoff

First non-engineering Hero domain pack. PM-shaped spec types (PRD, story, epic, roadmap-item, intake-item) with methodology-preset overlays, PM agents and workflows, seven dashboard views (Roadmap default + Story queue + PRD editor + Intake funnel + Story detail + Handoff stream + Chat). IDE-style layout: left nav, tabbed center pane, bottom strip (artifact-state actions + chat input), toggleable right panel (chat with sticky ambient smarts at top). Reuses existing tracker integrations (Jira/Linear/GitHub). Killer demo: a Jira epic becomes a Hero story, `/design` on the story produces an engineering `feature` spec, and the handoff edge surfaces in the Handoff stream live.

**Status:** designed — 2026-05-19. Open questions resolved (see *Resolved open questions* below). Spec-type declarations locked against the registry (5 PM types). Agent / skill / command pack locked (27 / 32 / 22 with a 13 / 9 / 5 P-tier split). Seven dashboard views locked with slot assignments. Routing table locked. Scanner contract locked. Cross-domain edge contract locked against the DSKG `handoff` / `derived_from` / `realizes` kinds. Tracker integration locked to existing adapters with content-vs-org-state conflict policy per `tracker-fronting-and-local-first` decision.

**Pick up at:** `/deliver hero-pm` in the **`hero-code`** repository. This spec is canonical here; delivery happens there because the dashboards/widgets are Rust. The `handoff-to-hero-code.md` sibling is the implementation guide. Within `hero` the only outstanding work is the PM-pack content under `domains/pm/` — agent prompts, skill bodies, command shims, and the four still-stub spec-type files (`story.md`, `epic.md`, `roadmap-item.md` + AGENTS routing additions).

→ `/deliver hero-pm` *(runs in `hero-code`)*

**Files:** `.hero/planning/features/hero-pm/spec.md`, `.hero/planning/features/hero-pm/research-brief.md`, `.hero/planning/features/hero-pm/mockup-brief.md`, `.hero/planning/features/hero-pm/agent-pack-design.md`, `.hero/planning/features/hero-pm/handoff-to-hero-code.md`, `.hero/planning/features/hero-pm/mockups/*.html`, `.hero/planning/initiatives/hero-domains/spec.md`, `.hero/planning/features/pm-platform-unblock/spec.md`, `.hero/planning/features/domain-scoped-knowledge-graph/spec.md`, `domains/pm/` *(existing scaffold; canonical content lands at delivery)*.

**Skip:** New tracker integrations in v1 (reuse Jira/Linear/GitHub only). OKRs as a PM spec type (deferred to v2 / a future `strategy` domain). Methodology as a "mode" (not a mode — a layered preset). Forced point estimation. Cross-tracker handoff (Jira-PM → Linear-eng). Arbitrary multi-primary-domain unions; PM extension composition is governed by `dual-mode-pm-qa-capability-packs`. Roadmap-shaped providers (Productboard, Aha). PM-only Hero binary. Cross-domain reporting (combined PM/eng dashboards). Product analytics, experiment results, metrics pipelines.

## Goal

Ship the first non-engineering Hero domain pack: Product Management. Provides PM-shaped spec types (PRD, story, epic, roadmap-item, intake-item), PM agents and workflows, seven PM-specific dashboard views, and onboarding that imports an existing roadmap or tracker into a queryable spec corpus. Reuses existing tracker integrations (Jira / Linear / GitHub) — no new integrations in v1. Success means a PM can take a Jira epic into Hero, refine it into stories with EARS-shaped acceptance criteria, hand a story off to engineering as a `feature` spec via one click, and see the cross-domain `handoff` edge appear in the Handoff stream view within the same session, with engineering-side delivery state visible from the PM side of the workspace.

## Why now

The parent `hero-domains` initiative validates the multi-domain platform end-to-end through the PM pack. The platform primitives (#1–#6 in the parent, plus #4b inline-propose) are either delivered or designed. Per `pm-platform-unblock`, the remaining gaps are:

- W6 `domain-scoped-knowledge-graph` Go delivery (schema v3 → write stamping → read filtering → frontmatter `domain:` field).
- W7 `hero-code-handover-pack` artifacts (fixtures, contracts index, active-dialect doc, schema, scrum workspace).
- Plus the existing in-flight delivery on `spec-type-registry`, `inline-propose-output-mode`, `domain-routing-and-agents`, and `scan-pluggability`.

This spec produces the canonical design hero-code consumes. After delivery, the **killer demo** — Jira epic → Hero story → `/design` → engineering feature → `handoff` edge → live cross-domain status in the Handoff stream — runs end to end and earns the platform's multi-domain claim.

## Design

This spec **incorporates by reference** the four sibling design documents. Each section below names the load-bearing decisions and points at the sibling that owns the detail.

### 1. Strategic frame (sibling: `spec.md` self-references + `research-brief.md`)

The five PM principles, the silo-tearing thesis, the "layered presets not modes" stance, and the "flow over ceremony" UX worldview are locked. The full bullet list of the eight-bullet design philosophy lives in `research-brief.md` §"Consolidated design philosophy" and is binding on every dashboard and agent decision below.

| Concern | Authoritative location |
|---|---|
| Five PM principles + which view earns each | This file, *Guiding principles* |
| Silo-tearing patterns + cross-domain hooks | This file, *Silo-tearing patterns* |
| Layered methodology presets (Horizon + Continuous flow / Sprint / Cycle / Phased + Milestone overlay) | This file, *Methodology layers* |
| Per-competitor steal/leave matrix (Pivotal, Linear, Jira, Shape Up, Productboard, Aha, ProductPlan, Notion, Trello, Kanbanize, Height) | `research-brief.md` |
| Eight-bullet design philosophy | `research-brief.md` §"Consolidated design philosophy" |
| Anti-pattern list (10 things to explicitly not do) | `research-brief.md` §"Anti-pattern list" |

### 2. Layout grammar (sibling: `handoff-to-hero-code.md` §"Locked design decisions")

IDE-style two-mode layout is **locked**:

- **Left nav (~220–240px):** workspace + domain pill + view inventory + Chat entry (bolt icon, prominent) + saved views as parameterized singletons + active sessions + settings.
- **Center pane:** VS Code-style tabs — singleton tabs (Roadmap, Story Queue, Intake, Handoff stream, Search, Chat) + per-item tabs (Story detail, PRD editor, Epic detail, Roadmap-item detail, Intake-item detail).
- **Artifact body:** header chips (identity / state) + body content + load-bearing inline elements (linked-engineering-feature card with the *Hand off to /design* button; hill chart inline under cycle preset; AC list; PRD sections).
- **Bottom strip (~70–90px, artifact tabs only, NOT on Chat tab):** state-aware verb buttons (4–6) + chat input + minimal chrome (model chip, send).
- **Right panel (~360–400px, toggleable, default open):** sticky pinned ambient region (smarts about the active artifact, max 3–4 cards, empty state renders nothing) + chat scroll below (single rolling conversation, hero-code visual fidelity).

The bedrock distinction — bottom strip = **verbs**, ambient = **nouns / smarts**, header chips = **identity / state**, body = **content**, chat scroll = **time-ordered** — is the rule for every future surface decision. Detail (widths, paddings, breakpoints, animation behavior, single-rolling-chat semantics, sticky-vs-scroll boundaries, dismiss persistence) lives in `handoff-to-hero-code.md`.

The nine mockups under `mockups/` are the visual source of truth. When implementation diverges, update the mockups — do not quietly drift.

### 3. Brand (sibling: `handoff-to-hero-code.md` §"Brand")

Hero brand is the **lightning bolt** logo + the **light-blue palette**:

- `--hero-blue-300: #9bc1e6` — logo fill, light accents
- `--hero-blue-500: #6cb6ff` — mid, dark-mode accent
- `--hero-blue-700: #2a6cb5` — primary on light backgrounds
- `--hero-ink: #14181e` — dark chrome

**Purple is reserved** as the cross-domain accent — visible only on the handoff edge (Story detail's linked engineering feature card and Handoff stream per-row domain bars). Do not use Linear's indigo (#5e6ad2), Linear's purple (#7c3aed), or an "H" letterform anywhere in PM surfaces. Source assets: `web/docs/src/assets/logo.svg` and `web/docs/src/stylesheets/brand.css`.

### 4. Spec-types declaration (locked)

The pack declares **five** spec types via the spec-type registry (`spec-type-registry`, status: `delivering`). Each type's frontmatter, lifecycle, and per-preset overlays are below. **`okr` is explicitly deferred** to v2 / a future `strategy` domain (resolves open question #1). PM and engineering share the canonical work-tracking types (`feature`, `bug`, `chore`, `epic`, `intake`, `release`, `sprint`, `prd`, `initiative`) per `unified-spec-type-model` / `pm-foundation-delivery`; this pack contributes the PM-shaped types and the PM-shaped lifecycle / kind overlays on the shared ones.

#### 4.1 Canonical PM type table

| Type | Status today | Domain | Lifecycle | Owner-flip semantics | Vocabulary mapping (default → preset renders) | Per-preset frontmatter |
|---|---|---|---|---|---|---|
| `prd` | Authored at `domains/pm/spec-types/prd.md` | pm (shared core type) | `draft → review → approved → delivered` | None (PRD is PM-owned end-to-end) | "PRD" → cycle: "Pitch Doc" / shape-up vocabulary | `kind: pitch \| ten-section \| lightweight` (registry-declared). Cycle preset: lints empty `Appetite` / `Rabbit Holes`. Sprint preset: surfaces `Target sprint` in metadata row. |
| `story` *(to author at `domains/pm/spec-types/story.md`)* | **Stub — author at delivery** | pm | `drafted → refined → ready → in-flight → done` | `owner: pm → engineering` at `/handoff` time (per `pm-foundation-delivery` owner-flip model). Same artifact; engineering's `feature-delivery-lead` takes it from here. | "Story" → cycle: "Scope" / sprint: "Story" / kanban: "Card" | Sprint: `sprint`, `points`. Cycle: `cycle`, `hill_position`. Kanban: `wip_age` (computed). Phased: `release`, `phase`. Horizon: no story-level field. |
| `epic` *(shared core type per registry; PM overlay at `domains/pm/spec-types/epic.md`)* | **Author overlay at delivery** | shared (core); PM provides preset overlays | `proposed → committed → in-flight → done` | None at the epic level; child stories flip individually. | "Epic" → cycle: "Bet" / phased: "Release group" | Sprint: rollup `velocity_target`. Cycle: optional `is_bet: true` (one cycle, fixed appetite). Phased: `target_release`. |
| `roadmap-item` *(to author at `domains/pm/spec-types/roadmap-item.md`)* | **Stub — author at delivery** | pm | `candidate → committed → shipped → dropped` | None (roadmap-items roll up child outcomes; child stories carry delivery state). | "Roadmap-item" → cycle: "Pitch" / horizon: "Bet" | Horizon (almost always on): `horizon: now \| next \| later` or quarter. Cycle: `appetite: 1w \| 2w \| 6w`. Phased: `target_release`. Carries `evidence_count` + customer-segment rollup computed from linked intake-items. |
| `intake-item` *(authored at `domains/pm/spec-types/intake.md` as `intake`)* | Authored as `intake` per registry canonical naming | pm | `new → triaged → linked → rejected` | None; intake-items promote-to or merge-with a roadmap-item via graph edges (`promotes_to`, `merges_with`). | "Intake" → productboard vocabulary: "Insight" / generic: "Feedback" | No preset-specific frontmatter. Source attribution (`source`, `source_url`, `source_id`, `customer_segment`, `signal_strength`) is universal. |

**Naming reconciliation:** `pm-foundation-delivery` consolidates to **canonical industry names** at `core/spec-types/`; this pack uses `intake` (not `intake-item`) and `feature` (not `story`) **at the registry layer**. The vocabulary preset renders the user-facing term per active methodology. Where this spec says `story` it refers to the **PM authoring shape and lifecycle** of the `feature` type with `owner: pm`. Where it says `intake-item` it refers to the `intake` type as authored by PM. The pack's `domains/pm/spec-types/story.md` overlay declares the PM-shaped lifecycle, AC requirement, and preset frontmatter on top of the shared `feature` core record.

#### 4.2 Methodology layer overlay

Process-layer fields are declared by the **active preset config** in `hero.json`'s `pm.presets` block (e.g. `{ "roadmap": "horizon", "delivery": "cycle", "overlay": null }`), read at runtime by the `pm-preset-detection` skill, and applied as additive optional fields by the spec-type registry. Switching presets is a config edit + dashboard reload; no data migration. Preset-specific fields are written when their preset is active and ignored otherwise.

| Preset | Layer | Adds to story | Adds to roadmap-item | Dashboard impact |
|---|---|---|---|---|
| Horizon roadmap | Roadmap | — | `horizon: now \| next \| later` or quarter | Roadmap board defaults to horizon grouping. |
| Continuous flow (kanban) | Delivery | optional `wip_age` (computed) | — | Story queue: Icebox/Backlog/Current/Done bands; WIP aging; no cut line. |
| Sprint (scrum / scrumban) | Delivery | `sprint`, `points` | — | Story queue: velocity cut line; sprint picker in chrome; epic rollup `velocity_target`. |
| Cycle (Shape Up) | Delivery | `cycle`, `hill_position` | `appetite` (1w / 2w / 6w) | Roadmap betting-table view; Story detail hill chart; cooldown band on Story queue. |
| Phased / milestone | Delivery (or overlay) | `release`, `phase` | `target_release` | Roadmap groups by release; Story queue phase filter; gates between phases. |

The spec-type registry (`spec-type-registry`) must accept these preset-conditional optional-field declarations from the methodology profile. This is locked by `pm-foundation-delivery` §Track A and the `methodology` package contract; no further design work is required here.

#### 4.3 Lifecycle / AC requirements per type

- `prd`: cycle preset lints empty `Appetite`, `Rabbit Holes`, `No-Gos` sections. Other presets allow but do not require them.
- `story`: AC is **required** before transition `refined → ready`. EARS format default, Gherkin alternate (resolves open question #4). The pre-handoff gate (`/handoff`) enforces non-empty AC.
- `epic`: rollup AC computed from child stories; explicit AC optional.
- `roadmap-item`: no AC requirement; outcome statement + evidence count drive readiness.
- `intake-item`: no AC; triage state machine drives lifecycle.

### 5. Agent / skill / command pack manifest (locked)

The full design with role / when-invoked / produces / skills / delegations / prompt sketch / prior-art per item lives in `agent-pack-design.md`. This section names the canonical roster and tier counts so the pack manifest is unambiguous.

#### 5.1 Agent roster — 27 agents, 13 / 9 / 5 (P0 / P1 / P2)

Already-scaffolded P0 agents at `domains/pm/agents/`:
`pm-delivery-lead`, `pm-investigator`, `discovery-researcher`, `duplicate-detector`, `handoff-coordinator`, `intake-triager`, `pm-reviewer`, `prd-author`, `prioritization-strategist`, `product-strategist`, `roadmap-curator`, `story-writer` — present as scaffold files; canonical prompt bodies land at delivery per `agent-pack-design.md` §C.

| Tier | Count | P0 | P1 | P2 |
|---|---|---|---|---|
| Coordination | 2 | `pm-delivery-lead`, `pm-investigator` | — | — |
| Strategic | 5 | `product-strategist`, `discovery-researcher` | `competitive-analyst`, `metrics-analyst` | `portfolio-curator` |
| Authoring | 5 | `prd-author`, `story-writer`, `roadmap-curator` | `pitch-author`, `epic-framer` | — |
| Triage / curation | 3 | `intake-triager`, `duplicate-detector` | `dependency-mapper` | — |
| Prioritization | 3 | `prioritization-strategist` | `capacity-planner`, `risk-curator` | — |
| Coordination-delivery | 3 | `handoff-coordinator` | `stakeholder-communicator`, `cycle-planner` | — |
| Review | 3 | `pm-reviewer` | `roadmap-reviewer` | `discovery-reviewer` |
| Scrubbers | 3 | — | `stale-roadmap-scrubber`, `duplicate-intake-scrubber` | `ambiguous-story-scrubber` |
| **Total** | **27** | **13** | **9** | **5** |

**Methodology coaching ships as skills, not agents** (Continuous Discovery, Shape Up, hill-chart reasoning) — loaded on demand by `pm-delivery-lead` and authoring agents. Confirmed in `agent-pack-design.md` §C.8.

**The brand-interaction agent is `handoff-coordinator`** — it executes the cross-domain `story → feature` handoff, writes the `handoff` edge, and verifies the Story detail's linked-engineering-feature rail. It is the single most important agent in the pack and the platform thesis in one delegation.

#### 5.2 Skill library — 32 skills, ~18 in v1

Already-scaffolded skills at `domains/pm/skills/` (19 directories present): `acceptance-criteria-ears`, `continuous-discovery-cadence`, `cross-domain-graph-query`, `cycle-planning`, `dependency-mapping`, `duplicate-detection`, `evidence-synthesis`, `handoff-protocol`, `intake-classification`, `metrics-design`, `opportunity-solution-trees-torres`, `pitch-writing-shape-up`, `pm-preset-detection`, `prd-anti-patterns`, `prd-structure`, `prioritization-frameworks`, `roadmap-framing`, `sprint-planning`, `story-writing-invest`.

Remaining 13 skills (P1 / P2) per `agent-pack-design.md` §D: `acceptance-criteria-gherkin`, `release-notes-writing`, `stakeholder-communication`, `okr-design` (P2 — deferred per open question #1), `discovery-interview-design`, `shape-up-cadence`, `hill-chart-reasoning`, `iteration-planning`, `capacity-planning`, `risk-surfacing`, `domain-glossary-maintenance`, `product-vision-writing`, `assumption-testing`.

Canonical v1 set (~18 skills) is enumerated in `agent-pack-design.md` §H "Minimum viable PM pack". This pack ships those at delivery; P1 / P2 ride along as catalog entries with empty SKILL.md placeholders that fill in over v1.5 / v2.

#### 5.3 Command set — 22 commands

Already-scaffolded commands at `domains/pm/commands/` (11): `discover`, `handoff`, `metrics`, `pitch`, `prd`, `prioritize`, `refine`, `release-notes`, `roadmap`, `triage`, plus `dashboard.md` (not in §E). Remaining 11 to ship: `interview`, `capacity`, `plan-cycle`, `plan-sprint`, `plan-iteration`, `standup`, `scrub` *(with `roadmap` / `intake` / `stories` concerns)*. Plus 8 shared / reused commands from engineering that the PM pack adapts to accept PM artifact types: `/design`, `/deliver`, `/diagnose`, `/search`, `/why`, `/blocked`, `/note`, `/decide`, `/review`, `/retro`. The `/discover` collision with engineering is resolved by domain routing (`domain-routing-and-agents`) — only the active domain's binding is live; no alias is shipped.

The minimum-viable v1 command set (12) per `agent-pack-design.md` §H: `/refine`, `/triage`, `/roadmap`, `/prioritize`, `/prd`, `/pitch`, `/handoff`, `/discover`, `/metrics`, `/release-notes`, plus reused `/design` (PM artifact types), `/why` (cross-domain), `/search`, `/note`.

### 6. Dashboard view manifest (locked)

The pack registers **seven views** via the dashboard view registry (`dashboard-view-registry`, status: `completed`, absorbed by `hero-surface-shell`). Each view is described in full in `mockup-brief.md` and rendered in `mockups/`. Slot assignments follow the locked grammar.

| # | View slug | Tab kind | Default landing | Earns principle(s) | Layout slots | Mockup file | Sibling brief section |
|---|---|---|---|---|---|---|---|
| 1 | `pm.roadmap` | singleton | **yes** | 1 (decide), 3 (trade-offs), 4 (align) | Left nav: highlighted. Center: Roadmap board (Now/Next/Later default; view-toggle for Quarters / RICE / Value-vs-Effort / Betting table under cycle preset). Bottom strip: none on workspace tabs. Right panel: selection-driven roadmap-item detail + evidence rollup. | `02-roadmap-board.html` | Screen 2 |
| 2 | `pm.story-queue` | singleton | — | 1 (decide), 2 (define) | Left nav: highlighted. Center: single-list flow queue with Icebox/Backlog/Current/Done bands; velocity cut line (sprint preset) or cycle-fit marker (cycle preset); WIP aging cue. Bottom strip: none on workspace tabs. Right panel: row-selection preview. | `03-story-queue.html` | Screen 3 |
| 3 | `pm.prd-editor` | per-item | — | 2 (define) | Left nav: PRDs section + per-PRD tree. Center: goal-context strip + Notion-fidelity body (slash-commands, not block editor — per handoff doc); pitch-shaped default template. Bottom strip: PRD verbs (Approve / Suggest AC / Find related decisions / Summarize for standup / Refine section / Convert to pitch) + chat input. Right panel: review state + backlinks + ambient AI. | `04-prd-editor.html` | Screen 4 |
| 4 | `pm.story-detail` *(includes the brand button)* | per-item | — | 2 (define), 4 (align) | Left nav: nested under Story queue. Center: breadcrumb strip + title + status pills + **primary action bar with `Hand off to /design` brand button** + description (Notion editor) + EARS AC list + hill chart (cycle preset only) + linked-engineering-feature card (purple-accent border, prominently inline). Bottom strip: story verbs (state-aware — drafted vs ready vs in-flight) + chat input. Right panel: ambient suggestions (similar stories, parent epic, completeness diagnostics) + activity. | `01-story-detail.html`, `08-inline-proposal.html` | Screen 1 |
| 5 | `pm.intake-funnel` | singleton | — | 1 (decide), 3 (trade-offs) | Left nav: Intake + unread badge + yellow accent strip on content pane top (Linear Triage influence). Center: split — left list (~40%) of intake-items with source icons + right detail pane (~60%) with quote + highlight-to-link floating action bar + triage action bar (Link / Promote / Reject). Bottom strip: intake verbs on selected item. Right panel: ambient AI cluster suggestions. | `05-intake-funnel.html` | Screen 5 |
| 6 | `pm.handoff-stream` | singleton | — | 4 (align), 5 (learn) | Left nav: Handoff stream. Center: header strip cycle-time histogram + timeline list grouped by week — each row split into PM-side / handoff arrow (purple) / engineering-side with live delivery state pulled across the domain boundary. Bottom strip: per-row actions. Right panel: handoff-edge metadata + open story / feature / re-handoff. | `06-handoff-stream.html` | Screen 6 |
| 7 | `pm.chat` | singleton | — | 1 (decide), 2 (define) — the design-mode surface | Left nav: Chat entry (bolt icon, near top). Center: full-bleed hero-code chat fidelity — provider/model picker, context fullness, attachments, mentions, slash, reasoning, single rolling conversation. **No bottom strip on this tab.** Right panel: not applicable in design mode. | `07-chat-tab.html` | n/a (handoff doc) |

**Saved views are parameterized singletons** — e.g. "Mobile bugs" nested under Intake, "Q3 bets" nested under Roadmap. Same view registry slot, parameterized via URL query. Rendered example: mockup `05-intake-funnel.html` shows two open Intake tabs (master + "Mobile bugs" saved view).

### 7. Natural-language routing table (locked, PM-domain-scoped)

Under `domain-routing-and-agents`, the active domain's `AGENTS.md` is authoritative for natural-language intent mapping. The PM table lives at `domains/pm/AGENTS.md` (already scaffolded; canonical version reconciled at delivery) and is reproduced in `agent-pack-design.md` §F. Engineering's `feature-delivery-lead`, `engineer`, etc. are **not findable** in a PM-only project. Reused verbs (`/design`, `/deliver`, `/diagnose`, `/search`, `/why`, `/blocked`, `/note`, `/decide`, `/review`, `/retro`) route to PM agents under the PM domain. The 25-row routing table is in `agent-pack-design.md` §F; binding rows:

- "Hand off, send to engineering, ready for dev" → `/handoff` → `handoff-coordinator` *(the platform-thesis row)*.
- "New feedback, customer ask, support escalation" → `/triage` → `intake-triager`.
- "Refine, tighten, make this ready" → `/refine` → `pm-delivery-lead` → `story-writer` or `prd-author`.
- "Bug in a customer ask, this signal is confusing" → `/diagnose` → `pm-investigator` (PM-flavored; not engineering's `debug-investigator`).

### 8. Scanner contract (locked, PM-side)

Under `scan-pluggability` (status: `planning`), the PM pack registers a scanner at `domains/pm/scan/` that imports:

| Input source | Imports as | Emits into graph |
|---|---|---|
| Tracker epics (Jira / Linear / GitHub) — read via the **existing tracker adapters** (no new providers in v1) | `epic` specs with `owner: pm`, `kind:` mapped from tracker issue type, frontmatter `tracker_id`, `tracker_url` | `Epic` nodes; `belongs_to` edges to roadmap-items if matched by tag/label; `mentions` edges to linked feedback |
| Tracker stories/tickets (Jira / Linear / GitHub) | `story` specs with `owner: pm`, AC extracted from issue body if EARS-shaped; preset-conditional `sprint`/`points`/`cycle` from tracker fields | `Feature` nodes (canonical type) with domain=pm; child-of edges to epics |
| Roadmap docs (markdown files under a configured roots glob, e.g. `roadmap/`, `product/roadmap/`) | `roadmap-item` specs (one per heading match + outcome paragraph) | `RoadmapItem` nodes with horizon/quarter parsed from heading |
| OKR docs *(out of v1 scope; defer per open question #1)* | n/a | n/a |
| Customer feedback files / exports (`.hero/intake/raw/*.md` or similar) | `intake` specs with source attribution preserved | `Intake` nodes; source-segment-weighted; `promotes_to` / `merges_with` edges stay null until triaged |

**Conservative assumptions for the scan-pluggability contract** (since that sibling design has not landed):

- Scanner interface: a Go interface `domain.Scanner { Scan(ctx, root, cfg) (Report, error) }` implementing the dispatch shape sketched in `scan-pluggability` §"Scope outline" item 1.
- Output schema: **shared node/edge schema across domains** (registry types stamped with `domain` from `domain-scoped-knowledge-graph`). Domain-typed nodes are not used; node `type` stays canonical (`Feature`, `Epic`, `RoadmapItem`, `Intake`) and the `domain` column partitions.
- Dispatch: `hero scan` reads `hero.json`'s active domain and invokes that pack's scanner via the `scan-pluggability` dispatch surface. Cross-domain scans (one workspace, two domains) are deferred to v2.

If `scan-pluggability` lands with a different decision on output schema, this section needs a one-line update — but the listed input sources and emitted node types are correct regardless.

### 9. Cross-domain edge contract (locked from PM's side)

The PM pack consumes the cross-domain edge contract designed in `domain-scoped-knowledge-graph` (status: `designed`). PM is the **write side** for the `handoff` edge and the **read side** for `derived_from` and `realizes`. The contract:

| Edge kind | Direction | Written by | Read by (PM side) |
|---|---|---|---|
| `handoff` | PM `Story` (Feature with `domain=pm`) → engineering `Feature` (`domain=engineering`) | `handoff-coordinator` agent on `/handoff` (Story detail's *Hand off to /design* button) | Story detail's "Linked engineering feature" card; Handoff stream view; Roadmap-item evidence rollup ("X of Y stories handed off") |
| `derived_from` | engineering `Feature` (`domain=engineering`) → PM `Story` (`domain=pm`) | engineering's `feature-delivery-lead` when `/design` is invoked from a handoff packet (the reverse pointer) | Cross-domain `hero why` walks; Roadmap-item-to-shipped traceability |
| `realizes` | engineering `Feature` → PM `PRD` | engineering's `feature-delivery-lead` when a feature directly realizes a PRD without a story-level handoff | PRD's "delivered by" rollup; cross-domain release-notes generation |

The boundary is **computed** (`from.domain != to.domain`), not declared via a special edge kind — per DSKG's locked decision. The edge row's own `domain` column is the partition tag (set to the from-node's domain) for fast filtering; the boundary itself is a property of the endpoints. PM does **not** introduce a new `cross-domain-handoff` kind — the existing `handoff` kind crosses domains by virtue of its endpoints.

**Write-time invariants** consumed by PM:

1. `handoff-coordinator` calls `UpsertEdge` with explicit `Domain="pm"` (the from-node's domain) and asserts the to-node exists (engineering side returned by `/design`).
2. PM never writes engineering-side nodes directly; it always routes through `/design` invocation so engineering owns the to-node.
3. Re-handoff (engineering rejected the spec or it was abandoned) writes a new `handoff` edge with timestamp; the original edge stays for history (per the `handoff-protocol` skill in `agent-pack-design.md` §D.5).

**Read-time stances** (per the DSKG audit table consumed by `cross-domain-graph-query`):

- Story detail's linked-feature card: **boundary-aware (single-edge read)** — queries the `handoff` edge by kind, resolves the target across domains, renders the engineering domain tag in the card.
- Handoff stream widget: **boundary-aware (always)** — queries all `handoff` edges regardless of active domain and JOINs both endpoint domains for the label.
- Roadmap board: **boundary-aware** — child-feature delivery state pulls cross-domain.
- Story queue, Intake funnel, PRD editor: **filtered to `pm`** — engineering content does not appear unless `--all-domains`.

### 10. Tracker integration (locked)

v1 **reuses the existing Jira / Linear / GitHub adapters**. No new tracker providers in v1. This is an explicit decision (not a punt) per `pm-foundation-delivery` and the parent initiative: PM ships on the same integration surface engineering already uses, stress-testing whether one `DomainIntegration` interface tolerates two domains sharing one provider before adding roadmap-shaped providers (Productboard, Aha) in a v2 follow-up.

**PM-specific behavior is in the import mapping, not in new connectors:**

- Jira epic → `epic` spec with `owner: pm` and tracker-prefixed frontmatter (`jira_status`, `jira_priority`, `jira_assignee`).
- Jira story → `story` spec; AC extracted from description if EARS-shaped; sprint/points/cycle populated from Jira fields per active preset.
- Linear project → `epic` or `roadmap-item` based on import-time hint (configurable in `hero.json`'s `pm.tracker_mapping` block).
- Linear issue → `story` spec; cycle parsed from Linear's cycle field if cycle preset is active.
- GitHub issue with `epic` label → `epic`; otherwise → `story`. Milestones → `release` (when phased preset is on).

**Tracker-fronting and local-first** is the operating model (per `tracker-fronting-and-local-first` decision):

- Local writes are instant; propagation to the tracker is async; no syncing spinners in any UI surface.
- Conflict policy: **Hero wins for content** (PRD body, AC, story description), **tracker wins for org-state** (assignee, sprint, workflow status). The spec-type registry's content-vs-org-state field declaration (per `pm-foundation-delivery` Track A) drives which conflict rule applies to which field.
- Cross-tracker handoff (Jira-PM → Linear-eng) is **deferred** to v2 (resolves open question #2). v1 assumes PM and engineering share one tracker for any given workspace.

### 11. Domain coexistence (locked v1 stance)

PM and engineering live in **the same workspace** (one `.hero/` dir, two active packs is the future; v1 ships **single-active-domain** workspaces — switching is a `hero install`/re-install per `domain-routing-and-agents`). The graph namespace tags from `domain-scoped-knowledge-graph` make cross-domain edges queryable even in single-active mode (engineering content from a prior session of the same repo is filtered out by default; `--all-domains` widens the view). This resolves open question #3 with a deliberate v1 simplification.

The implication: a Hero project that does **both** PM and engineering switches active domain via re-install (or, post-v2, a `hero domain switch` command). The killer demo runs cross-domain in single-active-PM mode because the engineering content already exists in the same `.hero/graph.db` from prior engineering sessions, and the handoff write goes through the existing engineering `/design` command which stamps `domain=engineering` on the resulting feature node.

### 12. AC format (locked)

Stories default to **EARS** with **Gherkin as an alternate format** when the team prefers it. The `acceptance-criteria-ears` skill is the primary; `acceptance-criteria-gherkin` is the alternate (catalog entry — content fills in over v1). The pre-handoff gate requires non-empty AC in either format; the engineering `/design` accepts both and the engineering side's `feature-delivery-lead` re-shapes into engineering's preferred EARS as part of the handoff packet. Resolves open question #4.

### 13. Roadmap horizon model (locked)

Horizon is **ordered-bets-first** (now / next / later) as the default rendering, with **quarter-based** (`q3-2026` etc.) as a parallel-supported value in the same `horizon:` field. Switching between renders is a Roadmap board view toggle, not a config change. The `roadmap-curator` agent reads both forms. Resolves open question #5.

## Acceptance Criteria

The killer demo runs end to end. These criteria collectively prove the multi-domain platform claim:

- WHEN a user runs `hero sync import` in a `pm`-active workspace configured against a Jira project AND a Jira epic exists THE SYSTEM SHALL create an `epic` spec at `.hero/planning/epics/{slug}/spec.md` with `domain: pm`, `owner: pm`, `jira_status:` / `jira_priority:` / `jira_assignee:` frontmatter populated, and an `Epic` graph node stamped `domain=pm`.
- WHEN a Jira story is imported THE SYSTEM SHALL create a `story` spec (canonical `feature` type with `owner: pm`) under the configured `feature` location, with AC extracted into the `## Acceptance Criteria` section if the description is EARS-shaped, and a `Feature` graph node stamped `domain=pm` linked to its parent `Epic` via a `belongs_to` edge.
- WHEN a PM user clicks **Hand off to /design** on a Story detail view AND the story is in `ready` state AND AC is non-empty THE SYSTEM SHALL invoke `handoff-coordinator`, which SHALL call engineering's `/design` with the story content + handoff packet, write a `handoff` edge from the PM `Feature` node to the resulting engineering `Feature` node with `Domain="pm"` and `kind=handoff`, and surface the engineering feature on the Story detail's "Linked engineering feature" card within the same session.
- IF the story is not in `ready` state OR AC is empty WHEN the user clicks **Hand off to /design** THEN THE SYSTEM SHALL refuse the handoff and surface a refinement prompt (route to `/refine`).
- WHEN the Handoff stream view renders THE SYSTEM SHALL query all `handoff`-kind edges regardless of active domain AND render each row with the PM story (left), the purple cross-domain arrow (center), the engineering feature with live delivery state (right) pulled via `cross-domain-graph-query`.
- WHEN an engineering session updates the engineering feature's status to `delivering` THE SYSTEM SHALL reflect the new state on the PM-side Handoff stream row and Story detail linked-feature card on next render (within-session reactivity is a hero-code dashboard concern; this spec requires correctness on render, not push-based update).
- WHILE a PM session is active AND the user runs `hero why <story-slug>` THE SYSTEM SHALL walk the cross-domain edge to the engineering feature and render the boundary as `← _handoff (cross-domain pm → engineering)_`, continuing the walk into engineering's `derived_from` and `decided_in` edges per the DSKG `originEdgeTypes` set.
- WHEN a PM user creates a new intake-item via the Intake funnel's quick-create flow THE SYSTEM SHALL run `duplicate-detector` at write time AND surface near-duplicate candidates inline before commit.
- WHEN `pm-reviewer` reviews a story AND the AC list is empty THE SYSTEM SHALL block the story's transition to `ready` and write specific findings to the story spec's `## Review` section.
- WHEN a PM session opens the Story detail view of a story without a `handoff` edge THE SYSTEM SHALL render the "Linked engineering feature" card in its empty state with the dashed-border placeholder and the prominent **Hand off to /design** call-to-action.
- WHEN a Story-detail bottom-strip button fires a command with `--inline-propose` (e.g. **Draft AC**) THE SYSTEM SHALL render the proposed content in the artifact body with a dotted-border treatment, a "proposed by `story-writer`" badge, and accept / edit / reject affordances inline, per the `inline-propose-output-mode` v1.0 contract.
- WHEN the active methodology preset is `cycle` AND a PRD is opened in the PRD editor THE SYSTEM SHALL surface a lint warning if the `Appetite`, `Rabbit Holes`, or `No-Gos` sections are empty.
- WHEN `pm.presets.delivery: "sprint"` is set in `hero.json` AND a story has `points: 5` AND the active sprint's velocity boundary is 18 points AND prior stories in the queue total 14 points THE SYSTEM SHALL render the velocity cut line on the Story queue immediately below this story.
- WHEN the PM pack scanner runs in a workspace configured with `roadmap_doc_roots: ["roadmap/"]` THE SYSTEM SHALL parse markdown headings under those roots into `roadmap-item` specs with `horizon:` populated from heading prefixes (`## Now`, `## Next`, `## Later`) or quarter tags (`q3-2026`).
- WHEN `hero search "csv export"` runs in a PM-active workspace without `--all-domains` AND a matching story exists at `domain=pm` AND a matching engineering feature exists at `domain=engineering` THE SYSTEM SHALL return the PM story at full score and the engineering feature at 0.5× score with a `[domain: engineering]` snippet tag, per the DSKG retrieval audit.
- WHEN a PM user runs `/scrub roadmap` THE SYSTEM SHALL invoke `stale-roadmap-scrubber` (P1) and produce a report listing roadmap-items unchanged for >N weeks (where N is configurable; default 12), with recommended actions (archive / drop with reason / refresh) presented to the user — never auto-applied.

## Boundaries

- **Not** designing OKR support in v1 (resolves open question #1) — `okr-design` skill stays in the catalog as P2; agent does not ship until v2 / future `strategy` domain.
- **Not** supporting cross-tracker handoff (Jira-PM → Linear-eng) in v1 (resolves open question #2). PM and engineering share one tracker per workspace; cross-tracker is a v2 concern.
- **Not** designing multi-active-domain workspaces (resolves open question #3 with a v1 single-active stance). `hero domain switch` remains a re-install.
- **Not** adding new tracker / roadmap integration providers (Productboard, Aha) — v2 work after PM ships on the existing adapter surface.
- **Not** building a PM-only Hero binary — same `hero` binary, domain selected via `hero.json`.
- **Not** building cross-domain reporting / combined PM-eng dashboards — separate spec after PM ships.
- **Not** modeling product analytics, experiment results, metrics pipelines — those belong to the future `hero-data-analytics` initiative. Principle #5 ("learn from what shipped") is partially deferred to v2; v1 earns it via shipped-story → roadmap-item linkage + optional retro note.
- **Not** authoring the agent prompts in this spec — prompt bodies land at delivery in `domains/pm/agents/*.md`. The roster, role, when-invoked, and prior-art per agent are locked in `agent-pack-design.md` §C.
- **Not** redesigning the locked UX layout — the mockups under `mockups/` and the locked-decisions section in `handoff-to-hero-code.md` are authoritative.
- **Not** introducing new artifact types beyond `prd`, `story` (canonical `feature` with `owner: pm`), `epic`, `roadmap-item`, `intake-item` (canonical `intake`).
- **Not** redefining the canonical work-tracking types shipped by `pm-foundation-delivery` — this pack overlays preset-conditional fields and PM lifecycle, nothing more.

## Risks

1. **`domain-scoped-knowledge-graph` read-path delivery slips.** The killer demo's cross-domain handoff visibility depends on the DSKG read-path audit landing. Mitigation: DSKG is designed (2026-05-19); `pm-platform-unblock` W6 owns delivery; the audit table in DSKG §"Query-shape audit" is the v1 contract for the `cross-domain-graph-query` skill — if delivery slips a query path, the skill surfaces the gap.
2. **`scan-pluggability` decision on output schema differs from this spec's assumption.** This spec assumes a shared node/edge schema across domains, partitioned by `domain` column. If the scan design lands with domain-typed nodes instead, §8 needs adjustment (one-line change). Mitigation: surface this assumption to the `scan-pluggability` design pass in `pm-platform-unblock` W2; reconcile in the next iteration.
3. **`spec-type-registry` doesn't expose preset-conditional optional fields.** The methodology overlay (§4.2) requires the registry to accept additive optional fields declared by the active methodology profile. Mitigation: `pm-foundation-delivery` Track A delivers the registry with this hook per the registry's §6 type-record shape; verify at delivery.
4. **Tracker import mapping is harder than estimated.** Mapping Jira/Linear/GitHub fields onto PM types under variable preset configs is fiddly; AC extraction from free-text descriptions has unknown precision. Mitigation: v1 ships best-effort with explicit fallback to "story without AC, blocked at `refined → ready`"; PM users refine via `/refine` before handoff. The pre-handoff gate catches AC-less stories.
5. **`handoff-coordinator` becomes a single point of failure for the brand demo.** If the agent's prompt regresses or the cross-domain edge write fails silently, the demo breaks invisibly. Mitigation: agent ships with an explicit verification step (re-read the edge after write; assert Story detail rail updated); `pm-platform-unblock` W6 includes a smoke test for the round-trip; `handoff-protocol` skill carries the verification contract.
6. **Tracker-fronting conflict policy not enforced uniformly across all integration adapters.** Hero-wins-on-content / tracker-wins-on-org-state must be implemented in three adapters. Mitigation: the policy is owned by the `tracker-fronting-and-local-first` decision and `pm-foundation-delivery`'s integration code; the spec-type registry declares which fields are content vs org-state; the adapter implementation is a single shared layer reading those flags.
7. **Methodology preset switching causes invisible field drift.** Switching from sprint to cycle leaves `points` on existing stories; switching back may or may not be expected behavior. Mitigation: §4.2 documents the v1 behavior (preset-specific fields are written when active, ignored otherwise, never destructively cleared); the methodology profile's transition rules govern any future destructive cleanup.
8. **Hero-code dashboard implementation diverges from mockups.** Implementation happens in a separate repo; visual drift risk is real. Mitigation: `handoff-to-hero-code.md` explicitly forbids quiet UX divergence — diverge by updating the mockup, not by shipping. The mockups are the source of truth; PR review on the hero-code side enforces this.
9. **PM-first as a domain choice feels like incremental engineering UX.** Mitigated by making the handoff edge (`story → feature`) the hero demo, and by planning `hero-qa` (item #8 in the parent initiative) on a real cadence so the multi-domain platform claim has a second proof point.
10. **Agent / skill / command pack content cold-start.** 27 / 32 / 22 is a lot of files; ~30% are stubs even after delivery. Mitigation: v1 ships the minimum-viable set per `agent-pack-design.md` §H (13 P0 agents + ~18 skills + 12 commands); P1 / P2 ride in catalog as placeholders with empty SKILL.md/agents files and fill in over v1.5 / v2.

## Resolved open questions

1. **OKRs — PM domain or separate `strategy` domain?** *Resolved.* OKRs are **deferred to v2** and likely belong to a future `strategy` domain pack. v1 of `hero-pm` ships without `okr` as a spec type; the `okr-design` skill stays in the catalog (P2) as a placeholder.
2. **Cross-tracker handoff (Jira-PM → Linear-eng).** *Resolved.* **Out of v1 scope.** v1 assumes PM and engineering share one tracker per workspace. Cross-tracker handoff is a v2 concern after the existing tracker adapters prove the cross-domain pattern.
3. **Domain coexistence model.** *Resolved.* **Single-active-domain workspaces in v1**; PM and engineering live in the same `.hero/` directory but only one domain is active at a time per `domain-routing-and-agents`. The DSKG namespace tags make cross-domain queries work in single-active mode (engineering content from prior sessions is queryable cross-domain). `hero domain switch` remains a re-install; per-query domain swap is v2.
4. **Story AC format.** *Resolved.* **EARS default, Gherkin alternate.** The `acceptance-criteria-ears` skill is primary; `acceptance-criteria-gherkin` is the alternate. The pre-handoff gate accepts either format; engineering's `/design` re-shapes into engineering's preferred EARS as part of the handoff packet.
5. **Roadmap horizon model.** *Resolved.* **Ordered-bets-first (now / next / later) is the default**; quarter-based (`q3-2026`) is a parallel-supported value in the same `horizon:` field. Rendering is a Roadmap board view-toggle.

## New open questions (raised during design pass)

These do not block delivery but should be tracked:

1. **`scan-pluggability` output schema decision.** This spec assumes shared node/edge schema partitioned by `domain` column. If the `scan-pluggability` design lands with domain-typed nodes, §8 needs adjustment. Resolution lives in `pm-platform-unblock` W2.
2. **Engineering-side `derived_from` / `realizes` edges — who writes them?** This spec asserts engineering's `feature-delivery-lead` writes the reverse pointer. The agent's prompt update lives outside this pack (in `domains/engineering/agents/feature-delivery-lead.md`). Verify at delivery that the engineering agent is updated.
3. **Cycle-time histogram data source on the Handoff stream view.** The mockup shows p50 / p90 / p99 percentiles; the data source is implied to be the graph's `handoff` edges grouped by from→to delta against the engineering feature's `completed` event. Confirm with hero-code at implementation time that the Rust dashboard can compute this without a new dedicated table.
4. **PRD editor's slash-commands-not-block-editor stance** (per `handoff-to-hero-code.md`) tension with `mockup-brief.md` Screen 4 wording ("block-based editing + slash commands"). Mockup wording is reconciled in the handoff doc: the editor renders as markdown with slash-command affordances, not as a true block editor. Verify mockup `04-prd-editor.html` matches the handoff doc stance; update the mockup brief copy at next iteration if it lingers.
5. **Saved-view persistence model.** Parameterized singletons under Intake / Roadmap save their URL params somewhere — workspace config? User profile? Per-session? Not specified; defer to dashboard-view-registry's resolution under the absorbed `hero-surface-shell` work.

## Touchpoints

Two repos. Spec and pack content live here (`hero`); dashboards and widgets are implemented in `hero-code`.

### `hero` repo (this repo) — pack content + spec authoring

**Spec types** (`domains/pm/spec-types/`):
- `prd.md` *(authored)* — verify against §4.1, surface preset-conditional lint warnings per §4.3.
- `intake.md` *(authored — canonical `intake` per registry)* — verify source-attribution fields, no preset-specific frontmatter.
- `story.md` *(to author)* — PM-shaped lifecycle (`drafted → refined → ready → in-flight → done`), AC requirement at `refined → ready`, preset-conditional `sprint`/`points`/`cycle`/`hill_position`/`phase` fields, `owner: pm` default, owner-flip semantics at `/handoff`.
- `epic.md` *(to author — PM overlay on shared core type)* — preset-conditional rollup `velocity_target` (sprint) / `is_bet: true` (cycle) / `target_release` (phased).
- `roadmap-item.md` *(to author)* — `horizon:` (ordered-bets-first OR quarter), per-preset `appetite` / `target_release`, `evidence_count` rollup from linked intake-items.

**Agents** (`domains/pm/agents/`):
- 12 P0 agent files *(scaffolded)* — replace scaffolds with canonical prompts at delivery per `agent-pack-design.md` §C. The 13th P0 (the count includes scaffold + canonical) accounts for the prompt-body authoring pass.
- 9 P1 agent files *(to scaffold + author at v1.5)* — `competitive-analyst`, `metrics-analyst`, `pitch-author`, `epic-framer`, `dependency-mapper`, `capacity-planner`, `risk-curator`, `stakeholder-communicator`, `cycle-planner`, `roadmap-reviewer`, `stale-roadmap-scrubber`, `duplicate-intake-scrubber`.
- 5 P2 agent files *(catalog placeholders at v1)* — `portfolio-curator`, `discovery-reviewer`, `ambiguous-story-scrubber`.

**Skills** (`domains/pm/skills/<name>/SKILL.md`):
- 18 v1 skill bodies *(scaffolded as directories)* — author content per `agent-pack-design.md` §D.
- 14 catalog-placeholder skills — empty SKILL.md scaffolds at delivery for v1.5 / v2.

**Commands** (`domains/pm/commands/`):
- 11 v1 command files *(scaffolded — verify count)*; 11 P1 / shared commands to scaffold (`interview`, `capacity`, `plan-cycle` / `plan-sprint` / `plan-iteration`, `standup`, `scrub` *(three concerns)*, plus reuse adapters for `/design`, `/deliver`, `/diagnose`, `/search`, `/why`, `/blocked`, `/note`, `/decide`, `/review`, `/retro`).

**Routing + project context** (`domains/pm/`):
- `AGENTS.md` *(scaffolded)* — replace placeholder routing rows with the canonical 25-row table from `agent-pack-design.md` §F.
- `mission.md` *(authored)* — PM-domain mission statement (keep as-is).
- `README.md` files under `agents/` / `commands/` / `skills/` / `spec-types/` — inventory + cross-link.

**Methodology profiles** (per `pm-foundation-delivery` Track A — confirm not regressed):
- `core/methodologies/scrum.yaml`, `kanban.yaml`, `shape-up.yaml`, `waterfall.yaml`, `scrumban.yaml` — declare PM preset overlay fields per §4.2.

**Vocabulary presets** (per `pm-foundation-delivery` Track A):
- `core/vocabularies/agile-scrum.yaml`, `shape-up.yaml`, `kanban-flow.yaml`, etc. — render PM type display names per §4.1's vocabulary mapping column.

**This spec** (`.hero/planning/features/hero-pm/spec.md`) — canonical design; this file.

### `hero-code` repo (consumed by) — dashboard + widget implementation

Per `handoff-to-hero-code.md`, dashboards and widgets are Rust and live in `hero-code`. Touchpoints there (not authoritative here; cross-referenced for traceability):

- Seven view registrations under hero-code's view registry implementation, slot-mapped per §6.
- Story detail's *Hand off to /design* button and linked-engineering-feature card (the brand interaction).
- Handoff stream widget with the live cross-domain delivery state column.
- Inline-propose rendering on every artifact pane (per `inline-propose-output-mode` v1.0 contract).
- Roadmap board's view-toggle (Now/Next/Later / Quarters / RICE / Value-vs-Effort / Betting table).
- PRD editor's markdown-with-slash-commands surface (NOT a block editor).
- Intake funnel's highlight-to-link floating action bar.
- Chat tab with full hero-code chrome (provider/model picker, context fullness, attachments, mentions, slash, reasoning).
- Cmd-K command palette spanning PM + engineering specs + actions.

### Cross-cutting (touched by primitive deliveries already in flight)

These primitives are designed/delivering; PM does not own them but consumes their contracts. Tracked here for delivery readiness:

- `domain-scoped-knowledge-graph` schema v3 migration + cross-domain edge contract — owned by DSKG.
- `spec-type-registry` schema 1.1 export + preset-conditional optional fields hook — owned by `pm-foundation-delivery` / `spec-type-registry`.
- `inline-propose-output-mode` v1.0 envelope + SSE event names — shipped per the delivery snapshot in that spec.
- `domain-routing-and-agents` active-pack `AGENTS.md` loader — owned by that spec.
- `scan-pluggability` dispatch surface — owned by that spec.
- Tracker adapter content-vs-org-state field declaration — owned by `tracker-fronting-and-local-first` decision + `pm-foundation-delivery` integration code.
