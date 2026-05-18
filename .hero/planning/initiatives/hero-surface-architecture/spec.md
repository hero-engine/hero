---
title: Hero Surface Architecture — One Surface, Every Layer, Every Role
slug: hero-surface-architecture
type: initiative
status: planning
tags: [serve, dashboard, surface, ui, layers, automation, roles, pm, qa, engineering]
created: 2026-05-17
relations:
  - target: hero-platform
    kind: parent
  - target: hero-team-experience
    kind: relates-to
  - target: hero-pm
    kind: relates-to
  - target: hero-qa
    kind: relates-to
horizon: now
---

## Vision

`hero serve` is not "the daemon with a dashboard." It is **the Hero
operating surface** — the always-on home for every human and every agent
working in and around a codebase. Same way `git` is plumbing and GitHub
is the surface, `hero` is plumbing and `hero serve` is the surface.

This initiative pins down the **one surface** that adapts to **four
deployment layers** and **N role packs** without forking the product. It
resolves the open scope question between `hero` and `hero-cloud`,
consolidates the in-flight dashboard / team-server / automation specs
under one architecture, and adds the surfaces that have been undersold
or missing entirely (Agents & Automation, Hero ROI, Now/cold-start).

When this initiative is complete:

- Every Hero user — solo to enterprise — opens the same surface, with
  layer-appropriate features and role-appropriate views.
- New role packs (PM, QA, future Sales / Ops / Support) register against
  a single shell, not a separate product.
- Scheduled agents and automations are a first-class top-level home,
  not a hidden team-mode feature.
- A manager can see Hero's ROI — velocity, autonomy ratio, time saved —
  without leaving the surface.
- The "scope decision" between `hero` and `hero-cloud` is closed: same
  binary, gated features, configurable server target.

## Why now

Three forces converging:

1. The existing dashboard ([hero-dashboard-ui](../../../specs/hero-dashboard-ui/spec.md))
   is a static workspace browser, not an operating surface. It opens to
   project stats; it should open to "here's your context."
2. Hero serve is a **web app companion to the CLI** — you open it in a
   browser tab to deep-dive into what's going on across your projects
   beyond what one CLI command can show. This is a different product
   shape from the PM and QA domain packs ([hero-pm](../../features/hero-pm/spec.md),
   [hero-qa](../../features/hero-qa/spec.md)), which are desktop apps for
   end users. Hero serve borrows the brand DNA (bolt, hero-blue, type)
   but the chrome belongs in a web browser: top nav, scrolling content,
   no fixed body grid.
3. The scope question between `hero` and `hero-cloud` has been an
   ongoing drag. This initiative resolves it by structure: **one
   surface, layer-gated features.**

## Architecture

### The five top-level homes

The surface is organized around five homes. Each home is a page (or
small set of pages) reached from a top nav. Hero serve is a web app
in a browser tab — not a desktop workspace — so the chrome is
deliberately thin.

```
┌──────────────────────────────────────────────────────────────────────┐
│  ⚡ Hero │ Now  Work  Knowledge  Agents  People │  ⌘K  · avatar      │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ─── max-width ~1200px centered, scrolling content ───              │
│                                                                      │
│   ┌─ Page hero ────────────────────────────────────────────────┐    │
│   │  Now                                                        │    │
│   │  Sunday May 17 · 2 need your input · 1 agent running        │    │
│   └────────────────────────────────────────────────────────────┘    │
│                                                                      │
│   ─── Tabbed metric strip (text-link tabs, swap tile content) ──    │
│   This sprint · My week · Hero ROI                                   │
│   [tile] [tile] [tile] [tile]                                        │
│                                                                      │
│   ─── Section: Needs your input ─────────────────────────────       │
│   ...inline content...                                               │
│                                                                      │
│   ─── Section: Quick launch ─────────────────────────────────       │
│   ...big chat input...                                               │
│                                                                      │
│   ...more sections, top-down by day-to-day flow...                   │
│                                                                      │
│   ─── Quiet text-link footer at end of scroll ───                    │
└──────────────────────────────────────────────────────────────────────┘
```

Each home has its own page (`/now`, `/work`, …) with sections
appropriate to that home's purpose. Some homes carry an optional
**sub-nav row** of text-link tabs just below the top nav for in-home
navigation (Knowledge/Agents/People all use this). There is no fixed
left rail, no fixed right rail, no fixed bottom bar, no VS Code tab
strip. Actions live inline with the content they affect.

The signature mockups in
[hero-now-home/mockups/](../../features/hero-now-home/mockups/01-now-default.html),
[hero-work-home/mockups/](../../features/hero-work-home/mockups/01-work-roadmap.html),
[hero-knowledge-home/mockups/](../../features/hero-knowledge-home/mockups/01-knowledge-why.html),
[hero-agents-home/mockups/](../../features/hero-agents-home/mockups/01-agents-sessions.html),
and
[hero-people-and-roi-home/mockups/](../../features/hero-people-and-roi-home/mockups/01-roi-overview.html)
are the visual source of truth.

| Home | Purpose | What you find there |
|---|---|---|
| **Now** | Personal cold-start — "here's where you are." Default landing. | Active spec, your claims, unread handoffs from peers, what changed since you last looked, agent in-flight + recent proposals, quick-launch chat |
| **Work** | The spec + delivery surface (today's dashboard, evolved) | Roadmap / kanban / graph of specs, drift, contract coverage, blocked queue, CI + commit stream, sprint planner, sprint health |
| **Knowledge** | The corpus — conventions, decisions, learnings, search | Browse + filter by domain, recently captured, contradictions / staleness, unified search across specs+knowledge+commits+code, "why does this exist?" graph traversal |
| **Agents** | The under-loved differentiator: agents and automation | Live sessions, proposals queue, scheduled agents, automations (trigger→action), agent health, credential broker (team+) |
| **People** | Team home + Hero ROI | Presence + claims, activity feed (humans + agents), handoff stream, velocity / cycle time, autonomy ratio, individual productivity (opt-in), Hero hours saved |

### The four deployment layers

Same binary. Same surface. Features gated by **edition** and **server
target**. The split is not what you can see, but what reaches across
people and machines.

| Surface element | Local (solo) | Team Server (free / CE) | Cloud (paid) | Enterprise |
|---|---|---|---|---|
| Now, Work, Knowledge homes | ✓ | ✓ | ✓ | ✓ |
| Agents — your sessions | ✓ | ✓ | ✓ | ✓ |
| Agents — team sessions | — | ✓ (LAN) | ✓ (SSO) | ✓ (SSO + RBAC) |
| Scheduled agents / automations | ✓ local | ✓ shared | ✓ shared + cross-repo | ✓ + custom triggers |
| Proposals queue | local | server-mediated | + cross-org review | + policy gates |
| People & Pulse | n/a | LAN team | org-wide | + RBAC + audit retention |
| Credential broker | — | basic | usage + budgets | + customer-held keys |
| Hero ROI metrics | personal | team | org rollup | + SOC2 export |
| Federation / cross-repo | local peers | LAN peers | hosted hub | private hub |

**Same surface, more reach.** A solo user upgrading to team server gets
new things visible on the same homes — not a new product to learn.

The decision spec
[hero-surface-deployment-and-rendering](../../features/hero-surface-deployment-and-rendering/spec.md)
formalizes this matrix and resolves the rendering question
(Go templates + SSE for chrome and lists; islands of richer
client-side JS only where editors demand it; no React).

## Specs (children)

Ordered roughly in build order. Foundational specs first.

| Spec | Type | Status | Purpose |
|---|---|---|---|
| [hero-surface-deployment-and-rendering](../../features/hero-surface-deployment-and-rendering/spec.md) | decision | planning | Closes scope decision; defines 4-layer feature matrix and rendering model |
| [hero-surface-shell](../../features/hero-surface-shell/spec.md) | feature | planning | The chrome: slim top nav, page routing, optional sub-nav per home, shared page-hero / tabbed-metric-strip / chat-input fragments; the engineering pack is the reference registration |
| [hero-chat-and-model](../../features/hero-chat-and-model/spec.md) | feature | planning | Chat dispatch via the Hero adapter abstraction. hero-code is the canonical (required) adapter; in-IDE adapters (Claude Code first) are an optional enhancement. Hero serve runs no inference. ⌘K command bar; slash palette. Consumed by every home. |
| [hero-now-home](../../features/hero-now-home/spec.md) | feature | planning | Personal cold-start home (the default landing surface) |
| [hero-work-home](../../features/hero-work-home/spec.md) | feature | planning | Spec + delivery surface; evolves today's dashboard |
| [hero-knowledge-home](../../features/hero-knowledge-home/spec.md) | feature | planning | Corpus browsing, unified search, decision/convention exploration |
| [hero-agents-home](../../features/hero-agents-home/spec.md) | feature | planning | Sessions, proposals queue, scheduled agents, automations |
| [hero-people-and-roi-home](../../features/hero-people-and-roi-home/spec.md) | feature | planning | Team presence, activity feed, Hero ROI metrics, individual productivity |

### Mockups (per child spec)

Each child spec carries its own `mockups/` directory of numbered HTML
files. Mockups for hero serve use the **web-app grammar** locked by
the Now home mock: slim top nav as the only fixed chrome, scrolling
content (~1200px centered), sections with breathing room, optional
sub-nav row of text-link tabs per home, tabbed metric strip near the
top where useful, inline actions, no fixed bottom strip or right
rail. Brand DNA shared with the PM/QA desktop mocks (bolt logo,
hero-blue palette, Inter typography, status chip aesthetics) — but
the layout is a different product shape.

Each mockup is self-contained HTML, opens in any browser without a
build step.

## Dependencies and sequencing

```
hero-surface-deployment-and-rendering   (decision; unblocks everything)
        │
        ▼
hero-surface-shell                       (foundation; carries the layout)
        │
        ▼
hero-chat-and-model                      (chat plumbing every home consumes)
        │
        ├─► hero-now-home                (lights up the cold-start)
        ├─► hero-work-home               (supersedes hero-dashboard-v2 work pages)
        ├─► hero-knowledge-home          (supersedes scattered knowledge UI)
        ├─► hero-agents-home             (consumes hero-automations + scheduled-tasks MCP)
        └─► hero-people-and-roi-home     (team-layer features visible in CE+)
```

**Hero serve runs no inference.** It is a dispatcher. Every chat
invocation routes to a **Hero adapter** — a process that listens
for dispatches, runs the agent loop, and streams events back.

The canonical adapter is **hero-code**, the sibling runner. It
always works, handles all dispatch kinds (interactive + headless),
and is required for scheduled agents and automations. Solo users
install it locally; teams run a shared endpoint. *"Requires
hero-code"* is an honest baseline.

An optional second adapter type is an **in-IDE Hero adapter** — a
plugin/skill inside the user's IDE (Claude Code, Cursor, Codex)
that picks up dispatches and runs them in the IDE's own agent
loop. Claude Code is the realistic v1 target via its skills /
hooks system. Feasibility for Cursor and Codex is TBD; the
dispatch protocol enables them when their plumbing allows.

If no adapter is connected, `/ask`, `/note`, and `/scheduled` still
work (they execute inside hero serve), and the chat shows a clear
"Install hero-code →" CTA.

See [hero-chat-and-model](../../features/hero-chat-and-model/spec.md)
for the adapter abstraction, dispatch protocol, ⌘K command bar,
slash palette, deferred-run policies, and cost reporting.

**Merge-with-hero-code is a deferred decision.** This architecture
works whether hero serve and hero-code stay separate (current
plan) or hero-code is later embedded as a subprocess of hero serve
(one install, two processes). The dispatcher boundary is the same
either way.

External dependencies:

- [dashboard-view-registry](../../features/dashboard-view-registry/spec.md)
  becomes a child of `hero-surface-shell` (the registry IS the shell's
  pluggability primitive)
- [hero-automations](../../features/hero-automations/spec.md) is
  consumed by `hero-agents-home`
- [unified-search](../../features/unified-search/spec.md) and
  [traversal-queries](../../features/traversal-queries/spec.md) are
  consumed by `hero-knowledge-home`
- [hero-pm](../../features/hero-pm/spec.md) and
  [hero-qa](../../features/hero-qa/spec.md) consume the shell as
  additional packs — they are siblings to engineering, not children

## What this supersedes / consolidates

These specs are not deleted — they are explicitly absorbed or marked
`superseded` with a pointer to this initiative:

- [hero-serve-scope-decision](../../features/hero-serve-scope-decision/spec.md) →
  resolved by `hero-surface-deployment-and-rendering`
- [hero-dashboard-v2](../../features/hero-dashboard-v2/spec.md) →
  split across `hero-now-home`, `hero-work-home`, `hero-agents-home`,
  `hero-people-and-roi-home`
- [hero-team-server](../../features/hero-team-server/spec.md) →
  becomes the **server target** under
  `hero-surface-deployment-and-rendering`; team-coordination features
  remain owned by their existing specs but render under
  `hero-people-and-roi-home`
- [hero-community-edition](../../features/hero-community-edition/spec.md) →
  one column in the deployment-layers matrix; its open questions
  resolve there

## Open questions to resolve in the decision spec

1. **Edition flag mechanics** — compile-time tag vs. runtime
   `HERO_EDITION` env var
2. **Same-binary vs `hero-cloud` repo** — does the cloud edition build
   from `hero` source tree with edition flag, or remain a separate
   binary? (`hero-community-edition` proposes same source, edition
   flag; `hero-cloud-split` already moved code)
3. **Rendering model** — Go templates + SSE + vanilla JS (current
   shipped pattern) vs. richer islands for editors (PM PRD, QA test
   plan) vs. wholesale SPA
4. **View registry mechanics** — server-rendered view declarations vs.
   client-side route table; how packs deliver their views
5. **CE dashboard scope** — which views ship in Community Edition,
   declared per-view rather than per-home

## Risks

- **Scope creep:** five homes is ambitious. Order matters. Now and
  Work cover the existing dashboard's job; Agents is the new
  differentiator; Knowledge and People can land iteratively.
- **Grammar conflation with PM/QA.** Tempting to copy the PM/QA
  desktop layout because the mocks share visual DNA. Don't. PM/QA
  is a different product (end-user desktop app); hero serve is a
  web companion to the CLI. The Now mock is the source of truth for
  hero serve grammar — share tokens (colors, typography, chip
  styling), not chrome (no fixed left rail, no VS Code tabs, no
  fixed bottom bar, no fixed right rail).
- **Rendering model split:** if we ship Go templates for some pages
  and rich JS islands for others without a clear seam, we'll build
  two front-ends. Decision spec picks the seam: templates + small
  vanilla web component islands; no React; no top-level build step.
- **Routing across homes.** Each home is its own page (`/now`,
  `/work/…`). Per-item URLs (`/work/spec/<slug>`) need to deep-link
  cleanly. URL shape and back-button behavior is the design
  question to settle in shell delivery.

## Progress

- 2026-05-17 — Initiative drafted from session brainstorm. Brainstorm
  notes captured in this session; the 5-homes / 4-layers / one-surface
  framing is the load-bearing idea.
- Child specs and mockups to be drafted next.
