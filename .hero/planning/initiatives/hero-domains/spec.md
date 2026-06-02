---
title: Hero Domains — Platform Architecture for Non-Engineering Verticals
slug: hero-domains
type: initiative
status: planning
tags: [platform, domains, product-management, qa, roadmap, vertical]
created: 2026-04-25
relations:
  - target: hero-platform
    kind: related
horizon: next
size: giant
---

## Goal

Transform Hero from an engineering-specific tool into a domain-agnostic
platform where the core engine (specs, knowledge, agents, automations,
runner, dashboard) stays the same but the domain content (agents, skills,
commands, spec types, integrations, dashboard views) swaps based on the
user's function. Start with Product Management as the first non-engineering
domain because it produces spec-shaped artifacts, reuses existing trackers,
and closes the loop with engineering in a single session.

## Kickoff

Make Hero work for non-engineers — starting with PM, then QA. The core engine stays the same; each domain ships its own agents, commands, spec types, dashboard views, and integrations as a "pack."

**Status:** planning — **design locked 2026-05-17**. After multiple iterations, the architecture settled on a **PM-additive, no-migration** design: nine canonical work-tracking spec types using real industry names (`initiative`, `prd`, `epic`, `feature`, `bug`, `chore`, `intake`, `release`, `sprint`), with two independent adaptation layers (methodology profile for structural variation, vocabulary preset for display). Engineering's existing 137 features / 16 bugs / 14 initiatives stay exactly where they are — canonical type names match what they already declare. AC infrastructure unchanged. Meta / knowledge types (decision, convention, plan, reference, external, note, rule, tripwire, context) out of scope. Hero-code (Rust) peer-notified — zero conflicting committed work.

**Active sprint — "ship PM additively + unblock hero-code":**

| # | Item | Status |
|---|---|---|
| 0 | `unified-spec-type-model` decision spec — locked design | **designed** ✓ |
| 1 | `spec-type-registry` design — rescoped for nine-type model, no migration | **designed** ✓ |
| 2 | `inline-propose-output-mode` design | **designed** ✓ |
| 3 | PM content pack (5 spec-types + 12 agents + 19 skills + 10 commands) | **authored** ✓ |
| 4 | v1 vocabulary presets (6 files at `core/vocabularies/`) | **authored** ✓ |
| 5 | `internal/vocabulary/` Go package + loader | **delivered** ✓ |
| 6 | PM pack content alignment to final names (`roadmap-item` → `initiative`, `intake-item` → `intake`, etc.) | pending — content edits |
| 7 | Author `core/spec-types/` nine canonical type files | pending — content authoring |
| 8 | Author `core/methodologies/` five v1 methodology profiles (Scrum / Kanban / Shape Up / Waterfall / Scrumban) | pending — content authoring |
| 9 | Finish `domain-plugin-architecture` cutover (`domains/pm/` embed + ContentFS migration) | pending — ~80% built in WIP |
| 10 | `spec-type-registry` Go impl: markdown loader, kind support, schema 1.1 export to `.hero/cache/spec-types.json` | pending — Go work |
| 11 | `internal/methodology/` Go package (mirrors `internal/vocabulary/`) | pending — Go work |
| 12 | `internal/tasks/` Go package + `hero task` CLI (additive; AC infrastructure unchanged) | pending — Go work |
| 13 | Inline-propose Go side delivery (stdout shim + daemon proposal store + SSE + REST) | pending — Go work |
| 14 | Vocabulary + methodology-aware rendering spread across CLI / MCP / NEXT.md | pending — Go work |

After this sprint completes, hero-code (Rust) consumes three independent contracts: `.hero/cache/spec-types.json` (schema 1.1), `core/vocabularies/*.yaml`, `core/methodologies/*.yaml`. A follow-up advisory peer call lands when items 7-11 are done.

**Pick up at:** Item 6 (PM pack content alignment) or item 7 (author core/spec-types/) — start with whichever feels right for the session. Items 6-8 are content authoring and can run in parallel. Items 9-14 are Go implementation and require items 7-8 to complete first.

→ See `.hero/planning/features/pm-foundation-delivery/spec.md` for the sprint plan with kickoff prompts per work item (forthcoming — being authored alongside this update).

→ See `.hero/planning/features/unified-spec-type-model/spec.md` for the binding architectural decisions.

**Files:** .hero/planning/initiatives/hero-domains/spec.md, .hero/planning/features/unified-spec-type-model/spec.md (authoritative), .hero/planning/features/spec-type-registry/spec.md, .hero/planning/features/inline-propose-output-mode/spec.md, .hero/planning/features/hero-pm/spec.md, .hero/planning/features/domain-plugin-architecture/spec.md, domains/pm/, core/vocabularies/, internal/vocabulary/
**Skip:** Sales-first sequencing (deferred). New tracker integrations for PM v1 (reuse Jira/Linear/GitHub). Meta / knowledge types refactor (out of scope). Forced migration of existing engineering specs (no migration; aliasing unnecessary; canonical names match existing frontmatter). `internal/acceptance/` rename (AC infra stays). Custom user-defined spec types beyond `kind` sub-typing.

## Problem

Hero's core loop — design before you act, capture knowledge, coordinate
specialists, automate repetitive work — isn't unique to software
engineering. Product managers refine roadmap bets into PRDs and stories.
QA leads design test plans and triage defects. Designers shape product
specs. The workflow is universal; the vocabulary, artifacts, and
integrations are domain-specific.

Today Hero's agents, skills, commands, spec types, integrations, and
dashboard views are all hardcoded for engineering. A PM user would need to
rip out 33 agents and write new ones — and even then would hit the deeper
problem that the spec-type registry has no notion of `prd`, `story`, or
`epic`. The platform architecture should make new domains a configuration
choice, not a fork.

## Domain selection rationale

We evaluated eight candidate domains across four axes before committing to
Product Management as the first non-engineering vertical.

| Domain | Workflow clarity | Integration surface | Spec-shaped output | First-UX risk |
|---|---|---|---|---|
| Product Mgmt | High | Low (reuse existing tracker) | Very high | Low |
| QA | High | Medium-high (fragmented: TestRail, Xray, Zephyr, qTest, sheets) | Medium | Medium |
| Customer Support | High | Medium (Zendesk/Intercom/Front/HelpScout — pick one) | Low (tickets are transactional) | Medium |
| Design (UX) | High | Medium (Figma + tracker) | High (design specs, design system entries) | Low-medium |
| Data / Analytics | Medium | Medium (warehouse + BI) | High (metric specs, experiment specs) | Low-medium |
| Sales | Medium | High (CRM + quote + comp) | Low | High |
| Marketing | Low (too many sub-disciplines) | High | Low | High |
| Finance | Medium | High + regulated | Low | High |

### Recommendation

1. **PM first** — spec-shaped artifacts (PRDs, stories, epics, roadmap
   items), reuses existing trackers (Jira/Linear/GitHub), closes the loop
   with engineering in one session, and creates a network effect on
   existing engineering workflows.
2. **QA second** — symmetric downstream value, surfaces gaps PM didn't
   hit (test plans, defect lifecycles, regression suites). Less
   spec-shaped, more integration-fragmented.
3. **Design and Data/Analytics** — spin off as separate initiatives
   after PM ships. Both have spec-shaped output but different integration
   shapes that warrant their own design pass.
4. **Customer Support** — clean workflow but transactional artifacts;
   revisit after multi-domain primitives are battle-tested.
5. **Sales / Marketing / Finance** — deferred. See _Deferred domains_
   below.

### Tradeoff acknowledged

PM-first is the safest learning ground for the multi-domain primitives
but risks feeling like incremental engineering extension — "engineering
with different words." QA-first would more visibly prove "Hero is more
than an engineering tool" at the cost of higher integration scope and a
weaker spec-shape fit. The recommendation favors PM-first because we
need the primitives proven against the lowest-risk content domain
before we ask the platform to absorb fragmented integration shapes.

## Architecture

### Domain packs

A domain is a directory of agents/, commands/, skills/, spec types, an
integrations manifest, and a dashboard view manifest:

```
domains/
  engineering/           # what we ship today (default)
    agents/              # 33 engineering agents
    commands/            # 30+ engineering commands
    skills/              # 40+ engineering skills
    spec-types.json      # feature, bug, convention, decision
    views/               # dashboard view manifest
    integrations.json    # github, jira, linear
    AGENTS.md            # engineering routing table
  pm/
    agents/              # product-strategist, story-writer, ...
    commands/            # /refine, /triage, /roadmap
    skills/              # PM-specific skills
    spec-types.json      # prd, story, epic, roadmap-item
    views/               # roadmap, story queue, intake funnel, handoff
    integrations.json    # jira/linear/github (reused)
    AGENTS.md            # PM routing table
  qa/                    # second domain (future)
    ...
```

### Domain selection

```bash
hero init --domain pm              # new project with PM domain
hero init --domain engineering     # default, same as today
hero domain switch pm              # switch an existing project
hero domain list                   # list available domains
```

The domain is stored in `hero.json`:

```json
{
  "domain": "pm",
  "folder": ".hero"
}
```

### What's domain-agnostic (the core engine)

Everything in `internal/` stays the same:

- Spec parser, lifecycle, discovery (`internal/spec/`)
- Knowledge base, ingest, lint (`internal/index/`, CLI commands)
- Drift detection, impact analysis, coverage (`internal/drift/`, etc.)
- Runner, automations, job queue (`internal/runner/`, `internal/automations/`)
- Dashboard, team server, MCP protocol (`internal/serve/`)
- Session management, NEXT.md, handoff (`internal/sessions/`)
- Recap, pulse, velocity, cost (`internal/recap/`, `internal/pulse/`)
- Cross-repo, feed, active sessions

### What's domain-specific (the content layer)

| Component | Engineering | Product Management |
|---|---|---|
| Spec types | feature, bug, convention, decision | prd, story, epic, roadmap-item |
| Agents | feature-delivery-lead, debug-investigator, engineer | product-strategist, story-writer, roadmap-curator, intake-triager |
| Commands | /design, /deliver, /diagnose, /scrub | /refine, /triage, /roadmap (plus reused /design, /deliver, /diagnose) |
| Skills | go-stack, testing-and-validation | discovery-framing, INVEST-shaping, roadmap-grooming |
| Integrations | GitHub, Jira, Linear | Jira, Linear, GitHub (reused — no new integrations v1) |
| AGENTS.md | Engineering routing table | PM routing table |
| Scans | Code scan (languages, frameworks) | Roadmap-doc / tracker-epic scan |
| Dashboard views | Specs kanban, drift, CI status, velocity | Roadmap, Story queue, Intake funnel, Handoff stream |

### Integration interface

Each domain defines its integrations via a provider interface, same
pattern as the tracker integration:

```go
type DomainIntegration interface {
    Name() string
    Import(filter ImportFilter) ([]*spec.Spec, error)
    Sync(spec *spec.Spec) error
    Events() <-chan Event  // for automations
}
```

Engineering has GitHub/Jira/Linear. PM v1 reuses the same three. Future
domains (QA, Design) may add new providers — the interface must accept
roadmap-shaped (Productboard, Aha) and test-management-shaped (TestRail,
Xray) tools without forcing a tracker-shaped abstraction.

### UI customization

The dashboard (`hero serve`) adapts to the domain. Each pack registers
views via a `views/` manifest and declares a default landing page.

- Engineering: specs kanban, drift reports, CI status, velocity
- PM: Roadmap (default landing), Story queue, Intake funnel, Handoff stream

Shared infrastructure (jobs, automations, team server) stays the same;
only the page registry and routing differ per domain.

## Children

Work items 1–6 are platform primitives that must land before any
non-engineering domain content can ship. Item 7 (PM) is the first content
pack. Item 8 (QA) is the second domain, planned to surface gaps PM didn't
hit.

| # | Slug | Title | Kind | Priority |
|---|---|---|---|---|
| 1 | domain-plugin-architecture | Refactor engineering content into a domain pack | Platform primitive | P0 |
| 2 | spec-type-registry | Domain-declared spec types and lifecycles | Platform primitive | P0 |
| 3 | domain-routing-and-agents | Active-domain AGENTS.md and agent loader | Platform primitive | P0 |
| 4 | dashboard-view-registry | Pluggable dashboard pages per domain | Platform primitive | P0 |
| 4b | inline-propose-output-mode | Agents propose into artifact pane; accept / edit / reject UI | Platform primitive | P0 |
| 5 | scan-pluggability | Per-domain scanners | Platform primitive | P0 |
| 6 | domain-scoped-knowledge-graph | Namespace tags on graph nodes | Platform primitive | P0 |
| 7 | hero-pm | Product Management domain pack | Domain content pack | P0 |
| 8 | hero-qa | QA domain pack | Domain content pack | P1 |

### Deferred candidates (outside this initiative)

The following are plausible follow-on domain packs but are not children
of `hero-domains`. Each should be spun off as its own initiative after
PM ships and the primitives are proven. Ordering reflects current
sequencing instinct, not commitment.

- **hero-ops** _(new, 2026-05-16)_ — incident specs, runbook specs,
  post-mortem specs, on-call rotation context. Highest spec-shape fit
  of any unbuilt candidate (post-mortem is already spec-shaped;
  `incident-response` and `root-cause-classification` skills already
  ship in engineering). Integration target is well-defined (PagerDuty
  / Datadog / Sentry — pick one). Cleanest silo-tearing edge:
  incident → `/diagnose` → bug, closing the "feature shipped →
  incident raised → bug filed → lesson captured" loop. Strongest
  candidate for the *third* domain after PM + QA.
- **hero-customer-support** _(reframed, 2026-05-16)_ — was previously
  filed under "deferred forever" with "tickets don't map onto specs."
  Reframed as a serious candidate because the silo-tearing surface is
  the largest of any domain (CS ↔ PM intake ↔ engineering bug ↔ Ops
  incident), and agentic deflection + escalation routing is reshaping
  the category in real time. The artifact-shape concern is real (a
  ticket is not a spec) and may be the right place to introduce a
  second artifact archetype alongside specs — possibly a "case"
  shape that has spec-like persistence but ticket-like volume.
  Boldest "Hero replaces a category" story we have. The one we're
  most excited to design even though it's hardest.
- **hero-design** — UX/design specs, design system entries, Figma
  integration. Spec-shape fit is high; integration shape is novel.
  Tight PM↔design coupling (PRD ↔ design spec) makes this the
  smallest, cleanest of the deferred candidates.
- **hero-data-analytics** — metric specs, experiment specs, warehouse
  + BI integration. Spec-shape fit is high; integration shape is novel.
  Closes PM principle #5 ("Learn from what shipped") which today
  references this domain as a dependency.
- **hero-customer-success / revops** _(new, 2026-05-16)_ — distinct
  from raw Sales. Onboarding plays, retention specs, expansion plays,
  QBR briefs, account-health rollups. Spec-shape fit is real (a play
  is a spec); integration narrower than full CRM (Gainsight / Catalyst
  / native CRM modules). May be the right *first* revenue-side domain
  before tackling raw Sales.

### Deferred domains (revisit after multi-domain platform proven)

- **Sales** — high CRM/quote/comp integration surface, low spec-shaped
  output. Original CRO-brother rationale is still live and an existing
  `hero-sales` spec captures the design from before this initiative
  was reshaped. Re-engage once the platform has absorbed two content
  packs (PM, QA) and we know the `DomainIntegration` interface
  tolerates CRM shapes. May follow `hero-customer-success` rather
  than precede it.
- **Marketing** — too many sub-disciplines (content, growth, brand,
  events) to model as one domain. Likely a future cluster of domains.
- **Finance** — high integration surface plus regulatory constraints
  (SOX, audit trails) that warrant their own design pass.

## Sequencing

The order below is dictated by hard dependencies — every primitive after
#1 assumes the domain-pack layout exists, and #7 (PM) cannot ship until
the registry-shaped primitives (#2, #3, #4, #5) are in place.

1. **domain-plugin-architecture** — foundation. Pure refactor of existing
   engineering content into `domains/engineering/`, plus `hero init --domain`,
   `hero domain switch/list`, the `domain` field in `hero.json`, and
   `hero install` reading from the active pack. Zero new behavior.
2. **spec-type-registry** — blocking for PM. Today's spec types
   (`feature`, `bug`, `convention`, `decision`) are hardcoded across the
   spec parser, lint, status filters, and importers. Each domain pack
   must declare its spec types, lifecycle states, frontmatter schema, and
   accepting commands. Budget for a full audit of `internal/spec/`.
3. **domain-routing-and-agents** — without this, the model routes to
   engineering agents inside a PM project because the hardcoded
   `AGENTS.md` routing table is engineering-shaped. The active pack's
   routing table must be authoritative.
4. **dashboard-view-registry** — PM ships its own dashboard pages
   (Roadmap, Story queue, Intake funnel, Handoff stream). Today pages
   are fixed in the dashboard. Move to a config-driven registry plus
   per-domain `views/` manifest.
4b. **inline-propose-output-mode** — agent output-mode contract
   (`--inline-propose`) plus the view-layer accept / edit / reject
   widget. Required by the locked Hero PM UX pattern — every PM
   authoring agent (`story-writer`, `prd-author`, `roadmap-curator`,
   `prioritization-strategist`, etc.) proposes into the artifact
   pane rather than writing to disk; the user accepts, edits, or
   rejects each proposal in place. Depends on #3 (the agent loader
   that surfaces the new output mode) and #4 (the view registry
   that hosts the shared proposal widget). Slotted between #4 and #5
   because it attaches to both. Hero-wide primitive — engineering
   agents (e.g. `pr-reviewer`) inherit the capability.
5. **scan-pluggability** — PM onboarding needs domain-specific scanning
   (import roadmap docs, tracker epics) instead of code scanning.
   Generalize `hero scan` and reduce the existing engineering scan to a
   reference implementation under `domains/engineering/scan/`. Depends
   on #2 — domain scanners can only emit type-correct nodes once the
   spec-type registry exists.
6. **domain-scoped-knowledge-graph** — namespace tags on graph nodes
   so queries can filter or join by domain. P0 because the PM killer
   demo (PM `story` handed off to engineering `feature` via `/design`)
   requires PM and engineering content to coexist in one graph from
   day one — flat-namespace queries would silently mix domains. Adding
   tags later forces a re-index, so land it before PM ships.
7. **hero-pm** — first non-engineering domain. Depends on #1–#6 plus
   #4b (inline-propose).
   Validates the platform end-to-end and is the proving ground for the
   platform narrative. See the child spec for the full artifact-type
   table, workflow list, agent roster, and open questions.
8. **hero-qa** — second domain. Hard dependency on primitives #1–#6
   plus #4b.
   Soft sequencing behind #7: design can proceed in parallel with PM
   delivery, but `hero-qa` should not ship before `hero-pm` so PM
   lessons (especially around the spec-type registry and dashboard
   view registry) inform the QA pack's shape. Closes the downstream
   loop from engineering and is the test that the platform absorbs a
   meaningfully different content shape (test plans, defect lifecycles)
   without regressing PM.

## Cross-cutting risks

1. **Spec-type registry hardcoding is deeper than it looks.** Spec types
   are referenced from the parser, lint, status filters, importers, and
   the dashboard. Budget for a full audit during #2 rather than
   discovering the surface area mid-delivery.
2. **Knowledge graph queries need domain-aware filtering before PM
   ships.** Even in single-domain v1, every query path that touches the
   graph must tolerate a namespace tag — silently mixing domains in
   shared queries is worse than blocking on the work upfront.
3. **Dashboard needs a domain router, not just a page registry.** The
   dashboard must select the active pack and route to its default
   landing — not just expose a flat list of pages.
4. **`hero init --domain` and `hero domain switch` are dangerous on
   populated workspaces.** Switching domains must hide-not-delete and
   warn loudly. Treat the switch as a re-install of content with
   `.hero/` data preserved.
5. **PM-as-first-domain may feel like "engineering with different
   words."** The platform narrative needs QA on a real cadence — plan
   #8 with intent, not as an afterthought, so the multi-domain story
   holds up under scrutiny.
6. **Integration reuse is right for v1, trap for v2.** PM reusing
   Jira/Linear/GitHub is correct for the first content pack. The
   `DomainIntegration` interface must be shaped to accept roadmap tools
   (Productboard, Aha) and test-management tools (TestRail, Xray)
   without forcing a tracker-shaped abstraction onto them. Design the
   interface with this in mind during #1 even though no new providers
   ship until #8 or later.

Role and permission shapes (multi-user permissioning per domain) are
deferred to the `cloud-admin` initiative — not in scope here.

## Hero-wide principles that apply across primitives

- **Tracker fronting is local-first.** Trackers and Hero Cloud are
  backing stores, not front doors. All three operating modes
  (standalone, Cloud-backed, tracker-fronted) share one UX; local
  writes are instant, propagation is async, and the conflict policy
  is fixed (Hero wins on content, tracker wins on org-state). The
  `DomainIntegration` interface designed in #1 must support
  local-first write semantics as a first-class contract; the
  spec-type registry in #2 must distinguish content fields from
  org-state fields. See
  [tracker-fronting-and-local-first](../../knowledge/decisions/tracker-fronting-and-local-first.md).
