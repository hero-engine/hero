---
title: Hero Marketing — Positioning, Distribution, and Launch
slug: hero-marketing
type: initiative
status: planning
tags: [marketing, launch, distribution, docs, growth]
created: 2026-04-25
relations:
  - target: hero-platform
    kind: related
  - target: hero-domains
    kind: related
horizon: someday
size: giant
---

## Goal

Get Hero in front of the people who'd benefit most — engineering leads
and AI-native developers — and convert curiosity into installs, installs
into active workspaces, and active workspaces into teams. Build the
positioning, surfaces, and content engine to do this repeatably.

## Problem

Hero is technically capable but commercially invisible:

- No public landing page. The repo README is the de facto homepage.
- No hosted docs site. `mkdocs.yml` exists but isn't published.
- Homebrew install path is a personal tap (`hero-engine/tap/hero`),
  not a polished release.
- No demo media — nothing to show in a tweet, blog, or HN post.
- No launch narrative. No Show HN draft, no comparison page, no
  positioning doc the team can rally around.
- No analytics. We don't know who installs, who returns, who churns.
- No community surface. No Discord, no GitHub Discussions, no
  contributor guide.

The team-platform work (runner, dashboard, team server) is the
"why pay" story. The marketing initiative is the "why try" story.
Both need to exist before we open the floodgates.

## Audience

**Primary ICP — AI-native engineering lead at a 5-50 person team**

- Already using Cursor, Claude Code, OpenCode, or Copilot
- Frustrated that AI sessions lose context, repeat past mistakes, and
  can't see across the team
- Has tried writing CLAUDE.md / cursorrules / AGENTS.md by hand and
  found it doesn't scale
- Owns velocity, quality, and onboarding outcomes for their team
- Buys tools without procurement gauntlets ($0–$500/mo decisions)

**Secondary — solo AI-power-user developer**

- Builds side projects with AI tools, ships fast
- Wants structure without bureaucracy
- Will install via Homebrew, try it on a real project, tweet if it works

**Adjacent — CRO / RevOps lead (Hero Sales)**

- Out of scope for v1 launch but informs the "platform, not just engineering"
  story that prevents the wedge from feeling too narrow

## Positioning

> Hero is the spec layer for AI coding tools. Design before you build,
> diagnose before you fix. Your specs, knowledge, and conventions become
> context every future AI session inherits.

Three pillars to lead with:

1. **Spec-driven** — every change starts as a written spec; agents work
   against it instead of improvising
2. **Knowledge that compounds** — conventions, decisions, and learnings
   captured as you go, fed into every future session
3. **Tool-agnostic** — runs alongside Cursor, Claude Code, OpenCode,
   Copilot, or any MCP-capable client; not a new IDE

Anti-positioning (what Hero is not):

- Not a Cursor/Aider/Continue replacement — it sits on top of them
- Not an LLM provider — bring your own model
- Not a pure CLI tool — it's a workflow + corpus + dashboard
- Not enterprise-first — single-developer install path is a first-class citizen

## Surfaces

| Surface | Purpose | Owner |
|---|---|---|
| Landing page (heroengine.ai) | Convert visitors → installs | hero-landing-page |
| Docs site | Self-serve onboarding + reference | hero-docs-site |
| GitHub repo | Source, issues, releases, social card | hero-repo-polish (folded into landing) |
| Homebrew tap | Install path for macOS/Linux | hero-distribution |
| Install script (curl \| sh) | One-line install for everyone else | hero-distribution |
| Demo media | GIFs, screencast, asciinema, OG images | hero-demo-content |
| Launch posts | Show HN, Reddit, Twitter/X, dev.to | hero-launch-playbook |
| Blog / changelog | Ongoing content + version notes | hero-content-engine |
| Community (Discord or GitHub Discussions) | Support + signal | hero-community |
| Telemetry (opt-in) | Measure adoption + retention | hero-telemetry |

## Children

| Slug | Title | Priority | Effort |
|---|---|---|---|
| hero-positioning | Narrative, ICP, messaging, comparison | P0 | M |
| hero-landing-page | Public homepage with install CTA | P0 | M |
| hero-docs-site | Hosted docs at docs.heroengine.ai | P0 | M |
| hero-distribution | Homebrew formula, install.sh, GitHub releases | P0 | M |
| hero-demo-content | GIFs, screencast, social cards | P0 | S |
| hero-launch-playbook | Show HN, Reddit, X, podcast outreach | P0 | S |
| hero-content-engine | Ongoing blog + dev.to + case studies | P1 | M |
| hero-community | Discord / Discussions + contributor guide | P1 | S |
| hero-telemetry | Opt-in usage analytics + feedback channel | P1 | M |

## Sequencing

**Wave 1 — foundation (must precede anything public)**
1. **hero-positioning** — locks the narrative everything else inherits
2. **hero-distribution** — install path has to actually work before we
   point people at it

**Wave 2 — surfaces (can run in parallel after positioning lands)**
3. **hero-landing-page** — depends on positioning copy
4. **hero-docs-site** — depends on positioning + existing mkdocs
5. **hero-demo-content** — depends on positioning (what to demo)

**Wave 3 — go to market**
6. **hero-launch-playbook** — depends on all of wave 2 being live
7. **hero-telemetry** — should ship with launch so we measure from day 1
8. **hero-community** — open before launch so early adopters have a home

**Wave 4 — sustain**
9. **hero-content-engine** — ongoing after launch

## Constraints

- Don't ship the launch wave until the team-platform story (hero-runner,
  hero-team-server, hero-dashboard-v2) is dialed in. NEXT.md is explicit:
  "Don't ship until team story is dialed in."
- Solo-install path (Homebrew + a single project) must work before
  anything else — that's how 95% of trial users will arrive.
- Avoid enterprise-y polish that signals "this isn't for you" to solo
  developers (no "Request a Demo" CTA, no logo wall before we have logos).

## Success criteria

- A new visitor goes from landing page → installed → first spec written
  in under 10 minutes without reading code
- Show HN / Product Hunt / Reddit launch generates ≥ 1,000 unique
  landing-page visits in week 1
- ≥ 100 Homebrew installs in week 1
- ≥ 25 active workspaces (defined as a `.hero/` folder with ≥ 1 spec
  delivered, telemetry reporting back) by end of month 1
- Community channel has ≥ 50 members and one non-team contributor PR
  by end of month 1

## Open questions

- ~~Domain: do we have hero.dev, gohero.io, hero.so, etc.? Need to lock one
  before landing page work starts.~~ — resolved 2026-05-14: `heroengine.ai`
  (see `.hero/knowledge/decisions/domain-name.md`). `teamhero.cloud` parked
  for possible microsite use.
- ~~Hosting: GitHub Pages (free, simple) vs Vercel/Netlify (better DX) vs
  self-hosted on Cloudflare Pages.~~ — resolved: Cloudflare Pages for both
  landing and docs (unlimited bandwidth, native PR previews, one dashboard
  with our DNS).
- Telemetry vendor: PostHog (open-source, self-hostable) vs Plausible
  (privacy-first, simple) vs roll our own via the existing daemon.
- Community surface: Discord (best engagement, more work) vs GitHub
  Discussions (zero-friction, lower energy).
- Brand: do we invest in a logo / typeface / illustration system now
  or ship plain-but-clear?
