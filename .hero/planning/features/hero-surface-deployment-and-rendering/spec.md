---
title: Hero Surface — Deployment Layers and Rendering Model
slug: hero-surface-deployment-and-rendering
type: decision
status: accepted
tags: [serve, surface, deployment, edition, rendering, decision]
created: 2026-05-17
relations:
  - target: hero-surface-architecture
    kind: parent
  - target: hero-serve-scope-decision
    kind: supersedes
  - target: hero-community-edition
    kind: relates-to
  - target: hero-cloud-split
    kind: relates-to
horizon: now
---

## Context

Three intertwined questions have been blocking surface work:

1. **Where does `hero serve` end and `hero-cloud` begin?**
   ([hero-serve-scope-decision](../hero-serve-scope-decision/spec.md))
2. **What renders the dashboard?** Today: vanilla JS + JSON API.
   `hero-dashboard-v2` proposes Go templates + SSE. PM/QA packs assume
   rich client-side editors with ambient AI panels.
   ([dashboard-view-registry](../dashboard-view-registry/spec.md) §"Unknown #1")
3. **What edition runs where?** Community / Cloud / Enterprise.
   ([hero-community-edition](../hero-community-edition/spec.md))

These can't be answered separately. The deployment shape determines
what features must render; the rendering model determines what's
practical to ship across editions; the edition determines what code
needs to be conditionally absent at build or runtime.

This spec settles all three with one architecture.

## Options considered

### Question 1 — Where does `hero serve` end?

**Option A: Same binary, gated features (chosen).** `hero serve` ships
every surface element. Layer-specific features (team feed, OAuth,
RBAC, audit export) compile in based on edition tag and activate based
on configured server target. A solo user runs `hero serve` and gets
the local layer. A team points `hero serve` at a shared server target
and gets team features. Cloud and Enterprise editions add features at
compile time.

**Option B: Two binaries (`hero` + `hero-cloud`).** As `hero-cloud-split`
laid out. Local binary covers solo; cloud binary covers team and up.

**Trade-off:** Option B is cleaner on paper but forces users to learn
two surfaces and forces us to maintain two sets of dashboard pages.
The PM/QA experience demands continuity — a PM working solo and a PM
working on a team server should see the same views, just with team
data populated.

**Decision: Option A.** One binary. Edition flag controls feature
compilation. Server target controls feature activation. The
`hero-cloud-split` work is preserved as a *package boundary inside
the repo*, not a separate binary.

### Question 2 — Rendering model

**Option A: Pure Go templates + SSE (the v2 spec proposal).** Cheap,
fast, no build step. Excellent for lists, status, feeds. Bad for
editors with rich text, drag-and-drop, inline AI panels.

**Option B: SPA (React or similar).** Best for editors. Heavy. Build
step. Conflicts with existing dashboard pattern.

**Option C: Templates as chassis, web components as islands (chosen).**
Go templates render the shell, navigation, and most list/grid views
via SSE-driven HTML. Where editors need richer behavior (PRD writer,
test plan editor, knowledge writer, agent session transcript with
live tools), a small set of vanilla web components hydrate islands.
No top-level build step; islands can be authored as `.js` files
served directly. Preact via CDN is acceptable for island heavy lifts
if hand-rolled web components get unwieldy, but the default is
hand-rolled.

**Trade-off:** This is the seam. Templates handle 80% (lists, cards,
status, feeds). Islands handle the 20% that needs interactivity. The
risk is islands proliferating into a de facto SPA. Guard: every
island must be justifiable by an editor or live-transcript need; lists
and grids must use templates.

**Decision: Option C.** Templates + islands. No React. No build step
at the top level.

### Question 3 — Editions

**Option A: Build-time tags (chosen for "is this code present?").**
`//go:build !ce` style tags strip Enterprise-only code from the
Community Edition build. Used for proprietary IP that legally must
not ship in CE.

**Option B: Runtime `HERO_EDITION` flag (chosen for "is this feature
active?").** A single runtime env var selects `local | team | cloud |
enterprise`. Controls UI feature visibility, API endpoint
registration, and policy enforcement. Lets a single Cloud binary
serve Community Edition users by setting `HERO_EDITION=ce`.

**Decision: Both.** Build tags strip code that legally must not ship.
Runtime flag toggles features available in the shipped binary. The
default install (`hero serve` from a developer machine) is
`HERO_EDITION=local`, with no auth, no team features visible.

## Decision

### The 4-layer feature matrix

| Surface element | `local` | `team` | `cloud` | `enterprise` |
|---|---|---|---|---|
| Now / Work / Knowledge homes | ✓ | ✓ | ✓ | ✓ |
| Agents — your own sessions | ✓ | ✓ | ✓ | ✓ |
| Agents — team sessions | — | LAN | SSO org-wide | SSO + RBAC |
| Proposals queue | local | server-mediated | + cross-org review | + policy gates |
| Scheduled agents / automations | local cron | shared queue | + cross-repo | + custom triggers + audit |
| People & Pulse home | self-only | LAN team | org-wide | + retention/RBAC |
| Credential broker | — | per-user keys | shared org keys + budgets | + customer-held keys |
| Hero ROI metrics | personal | team rollup | org rollup | + SOC2 export |
| Federation (peer repos) | local peers | LAN peers | hosted hub | private hub |
| Auth model | none (localhost) | shared token / OAuth | SSO (Google / GH / SAML) | SSO + SAML + audit |
| Audit retention | local logs | 30 days | configurable | configurable + signed chain |

### Edition flag mechanics

```
HERO_EDITION=local       # default; no auth; localhost-only
HERO_EDITION=team        # team server; OAuth or shared token
HERO_EDITION=cloud       # hosted; full features; SSO
HERO_EDITION=enterprise  # on-prem; customer keys; audit
HERO_EDITION=ce          # community edition (cloud binary, capped)
```

Build tags strip Enterprise-only code from non-Enterprise builds:

```go
//go:build enterprise
// signed-audit-chain.go
```

Build tags strip Cloud-and-Enterprise code from CE/local builds where
the code is proprietary IP that must not ship to non-paying customers.

### Rendering model

Hero serve is a **web app companion to the CLI** — opened in a
browser tab to deep-dive into projects. It is not a desktop
workspace. The chrome is deliberately thin; the page is the
artifact.

- **Fixed chrome (slim top nav only ~56px):** Hero brand mark
  (bolt SVG), top-nav text-link tabs (`Now`, `Work`, `Knowledge`,
  `Agents`, `People`) with hero-blue underline on active, ⌘K
  search pill, avatar / workspace state. Go template rendered
  server-side, served on every route.
- **Each home is a page** at `/now`, `/work`, `/knowledge`,
  `/agents`, `/people` (and per-item routes like
  `/work/spec/<slug>`). No VS Code-style tab strip. No fixed
  left rail, right rail, or bottom bar.
- **Page layout:** scrolling content, max-width ~1200px centered
  with ~32px horizontal padding. Sections stack vertically with
  generous whitespace (~48–56px between sections). Actions live
  inline with the content they affect.
- **Optional sub-nav row** of text-link tabs below the top nav
  for in-home navigation (used by Knowledge / Agents / People).
  Same idiom as top nav, smaller.
- **Page templates:** Go templates per page. Lists, grids, feeds,
  cards, tile rows, charts (inline SVG) render server-side. SSE
  delivers fragment updates on data change. No client-side
  reactivity beyond fragment swapping.
- **Islands** (small hand-rolled vanilla web components) for
  things templates can't do: the ⌘K command bar overlay, the
  agent-session live transcript, a diff viewer, an automation
  rule builder, a knowledge writer, a chat input with streaming.
  Each island is its own `.js` file served directly; no bundler
  at the top level.
- **State management:** server is the source of truth. Islands
  hold only ephemeral UI state. Persistence goes through
  `/api/*` calls.
- **No React, Vue, Svelte, or top-level build step.** Preact via
  CDN is permitted only for island-internal complexity if
  hand-rolled components become unwieldy — requires spec
  justification.

**Brand DNA shared with PM/QA desktop mocks** — bolt logo,
hero-blue palette, Inter typography, status-chip aesthetics —
but **chrome is web, not desktop**. The Now mock
([hero-now-home/mockups/01-now-default.html](../hero-now-home/mockups/01-now-default.html))
is the visual source of truth for this surface.

### Server target configuration

Solo users run `hero serve` with no extra config. Team users add a
single config block in `hero.json`:

```json
{
  "serve": {
    "target": "https://hero.team.internal",
    "auth": "oauth-google"
  }
}
```

When a server target is configured, `hero serve` becomes a local
**broker** to the team server: it still hosts the local MCP and file
watcher, but delegates job queue, claims, presence, automations, and
the People home to the remote server. Local users always see their
local Now / Work / Knowledge homes; team features overlay when
authenticated.

## Consequences

**Easier:**
- One surface to design, document, and teach.
- Solo → team upgrade is a config change, not a product migration.
- PM and QA packs work identically at every layer.
- View registry has a single rendering target (templates + islands).
- ROI metrics travel from personal to team to org without rewrite.

**Harder:**
- Edition discipline: every new feature must specify its layer
  visibility. Enforce via a field in the view registry registration.
- Islands hygiene: tempting to keep growing them. Need a rule: lists
  go in templates, editors and live transcripts can be islands.
- Build complexity: build tags multiply the build matrix
  (local / team / cloud / enterprise / ce). Acceptable; CI must build
  all five.

**Becomes possible:**
- The "Hero hours saved" rollup as a top-level surface manageable
  across layers.
- Cross-domain navigation (PM story → eng feature → QA test plan)
  inside a single shell.
- A solo developer's Now home and a team's People home built from the
  same templates.

## Open questions deferred to child specs

- Specific island inventory — owned by per-home specs.
- View-registry record schema — owned by `hero-surface-shell`.
- Per-view CE disposition — declared on each view in its home spec.
- Server target wire protocol — owned by the (now-folded)
  `hero-team-server` body of work, retargeted under
  `hero-surface-shell`.
