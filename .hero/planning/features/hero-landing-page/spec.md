---
title: Hero Landing Page — Public Homepage with Install CTA
slug: hero-landing-page
type: feature
status: delivering
priority: P0
tags: [marketing, landing, web, conversion]
created: 2026-04-25
updated: 2026-05-15
relations:
  - target: hero-positioning
    kind: depends-on
  - target: hero-distribution
    kind: depends-on
  - target: hero-demo-content
    kind: depends-on
horizon: now
superseded_by: hero-landing-message-refresh
# superseded_reason: Replaced stale v0.9 landing scope with the v0.34 memory-first public message and proof refresh.
---

## Goal

A single-page site at **heroengine.ai** that gives Hero a real-feeling public
face and converts a curious visitor — someone who clicked through from a
tweet, HN comment, or a friend's recommendation — into a working install
(brew on macOS, scoop or PowerShell on Windows, curl script on Linux) in
under three minutes. Ship v1 with what we already have today; iterate from
there.

## Kickoff

Phase 1 has been delivered at the file level (2026-05-15). The landing
page source lives in `web/landing/site/index.html`, with `wrangler.toml`,
`favicon.svg`, `og-image.svg`, `robots.txt`, and `README.md` alongside.
A gated GitHub Actions workflow at `.github/workflows/landing.yml`
mirrors the docs deploy pattern. Bundle is ~47KB (well under the 100KB
budget). Deploy is still pending — to go live at `heroengine.ai`:

1. Add Cloudflare secrets to the repo: `CLOUDFLARE_API_TOKEN`,
   `CLOUDFLARE_ACCOUNT_ID`.
2. Configure the `heroengine.ai` DNS / custom-domain mapping in the
   Cloudflare dashboard for the `hero-landing` Worker.
3. Either run `cd web/landing && wrangler deploy` once manually, or set
   repo variable `DEPLOY_LANDING=true` and push to `main`.
4. After deploy, verify the live URL against the acceptance criteria
   (HTTPS at `heroengine.ai`, LCP < 1.5s on 4G, all install commands
   copy, keyboard-navigable, OG card renders). Then bump the spec
   status from `delivering` to `completed` and run
   `hero spec complete hero-landing-page`.

Phase 2 work (animated demo, comparison table, real telemetry provider,
launch banner) is deferred and tracked in the `## Phasing` section
below. A telemetry hook (`window.heroTrack`) is wired as a no-op and
ready for a Plausible or PostHog provider.

## Problem

Today the GitHub README is the public face of Hero. The docs site
(`web/docs/src/`, served from Cloudflare via `web/docs/wrangler.toml`) tells people
*how* to use Hero — but a visitor arriving cold at heroengine.ai has
nowhere to land that says *what Hero is* in one paragraph and funnels them
into installing. We need that page now, not later.

A landing page lets us:
- Communicate the sidekick-brain idea in one paragraph
- Hand visitors an install command and a docs link, side by side
- Tease what's coming so the project feels alive
- Provide a stable, shareable URL for launch posts
- Capture signal (visits, install clicks) GitHub can't give us

## Scope discipline

Ship v1 with whatever we have today. The original spec proposed seven
sections including a comparison table and a hero-shot animated demo —
both are deferred. v1 is intentionally small so it can ship this week.

## Page structure (v1)

### Above the fold
- **One-liner** — from positioning doc: Hero is the sidekick brain for
  AI-augmented engineering — it captures everything during agent sessions
  and injects it back so every session starts smarter
- **Sub-line** — 1–2 sentences elaborating
- **Hero shot** — clean static screenshot of `hero` in action (CLI output
  or a `hero status` snapshot). No animated GIF or video required for v1.
- **Primary CTA** — a tabbed install block with three platforms, each
  with a copy button. Mirror the commands shown in
  `web/docs/src/getting-started/installation.md`:
  - **macOS** — `brew install hero-engine/tap/hero`
  - **Windows** — `scoop bucket add hero-engine https://github.com/hero-engine/scoop-bucket && scoop install hero` is the primary command. A PowerShell `irm ... | iex` line is offered as a secondary, smaller line below.
  - **Linux** — `curl -fsSL https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.sh | sh`
  Default tab can be detected from the visitor's user-agent; otherwise
  default to macOS.
- **Secondary CTA** — "Read the docs" → docs site
- **Tertiary CTA** — "View on GitHub" → repo

### Section 1 — The core loop
- 4-step diagram (Discover → Design → Deliver, with Diagnose branching in)
- One sentence per step
- Tiny screenshot per step if we have one; otherwise text-only is fine

### Section 2 — You don't memorize commands
- One-paragraph claim drawn from `web/docs/src/what-is-hero.md` ("Hero installs
  as slash commands … but you don't have to memorize them — just say
  what you want.")
- A small example block — five lines, one per mapping (taken verbatim
  from `web/docs/src/what-is-hero.md`):
  - `"fix the login timeout bug"` → `/diagnose`
  - `"add CSV export to the reports page"` → `/design`
  - `"implement the auth spec"` → `/deliver`
  - `"review my PR"` → `/review`
  - `"what's blocking the billing migration?"` → asks the corpus
- Keep it short — this section is one paragraph plus the five-line
  example block, nothing more.

### Section 3 — Why specs
- 4 cards: "Design before you build", "Knowledge compounds",
  "Tool-agnostic", and **"Ask why anything exists"**
- Each card: one-line claim, one-paragraph proof
- The fourth card carries the `hero why` / `hero blocked` callout —
  promoted from a sentence to its own card because tracing decision
  history and unblocking work are differentiators worth their own
  space. Copy points: `hero why <thing>` traces the chain of decisions
  and specs that led to anything existing; `hero blocked` lists open
  work that can't move forward and on what dependency. Both are real
  CLI commands today, not roadmap items.

### Section 4 — Works with your tools
- Logo row: Claude Code, Cursor, OpenCode, Codex, Copilot, "+ any MCP
  client". All five named tools have real installer paths in
  `internal/install/` (`target_<name>.go`); list reflects what
  `hero install project . --target <tool>` actually supports today.
- One-line install per tool

### Section 5 — What's coming soon (teaser)
- A single short section, three cards
- Clearly labeled "What we're building next" — not "available today"
- Three themes (drawn from in-flight planning specs):
  - **Cloud and team mode** — shared knowledge across a team (covers
    `cloud-admin`, `cloud-billing`, `hero-team-server`)
  - **Agent outposts** — let agents operate scoped external systems
    with audit-by-construction (`agent-outposts`)
  - **Tripwire system** — forbidden-option guardrails so model sessions
    can't wander into known dead ends (`tripwire-system`)
- Unified search / retrieval is intentionally moved out of this teaser
  because `hero search` and cross-spec retrieval already ship today —
  reference that capability in Section 3's "Ask why anything exists"
  card rather than promising it as future work.
- No dates. No vaporware. No signup form, no email capture.

### Section 6 — Get started
- 3-step quickstart (install, init, first spec)
- Code blocks copy-paste-able
- "Read the docs" link to docs site

### Section 7 — Footer
- GitHub, Discord/Discussions, docs
- MIT license note, "by @chet-bellows"

## Deferred to phase 2

- Animated GIF or autoplay video of the core loop
- Live demo asciinema / screencast embed (was Section 4 in v0)
- Comparison table vs Cursor / Aider / raw CLAUDE.md (was Section 5)
- Waitlist / launch banner
- Customer logos / social proof
- Telemetry beyond the basic events listed below

## Tech stack

- **Framework**: plain HTML + Tailwind. Chosen to match the docs site's
  minimalism and to keep the page hand-editable by anyone reading the
  source — no build-step ceremony to learn.
- **Hosting**: Cloudflare Pages, mirroring `web/docs/src/`. Create
  `web/landing/wrangler.toml` modeled on the project-root `wrangler.toml`
  (`name = "hero-landing"`, `compatibility_date`, `[assets] directory =
  "./site"`).
- **Repo layout**: `web/landing/` inside this repo so copy stays
  version-controlled with the product.
- **Analytics**: Plausible or self-hosted PostHog (see hero-telemetry).
  No GA, no FB pixel.
- **Bundle**: under 100KB total excluding the screenshot.

## Content sources

Canonical copy lives in the docs and README — the implementer should
quote and adapt rather than write fresh:

- `web/docs/src/what-is-hero.md` — one-liner ("Hero gives your project a memory
  that AI coding tools can use"), sub-line, and the natural-language
  routing example block (see "You Don't Memorize Commands" section).
- `web/docs/src/why-hero.md` — value-prop copy (the problem precisely, what
  Hero adds, when it's a fit / when it isn't). Useful for the "Why
  specs" cards and the core-loop explanations.
- `web/docs/src/getting-started/installation.md` — exact install commands per
  platform; mirror these verbatim in the install tabs so they stay in
  sync with what we publish.
- `README.md` (repo root) — fallback for anything not yet in docs.

## Performance budget

- LCP < 1.5s on 4G
- Total page weight < 500KB
- 100/100 Lighthouse on accessibility (WCAG AA, alt text, contrast)

## Acceptance criteria

- THE SYSTEM SHALL be served at `heroengine.ai` over HTTPS
- WHEN a visitor loads `heroengine.ai` THE SYSTEM SHALL render
  above-the-fold content within 1.5s on 4G without requiring the
  screenshot to be loaded first
- THE SYSTEM SHALL show install commands for macOS (brew), Windows
  (scoop) and Linux (curl install script) without requiring navigation
  away from the page
- WHEN a visitor scrolls past the hero section THE SYSTEM SHALL surface
  natural-language routing (e.g. `"fix the login bug"` → `/diagnose`)
  within the first two sections below the fold
- WHEN a visitor clicks any install command THE SYSTEM SHALL copy the
  command to the clipboard
- THE SYSTEM SHALL be fully keyboard-navigable
- THE SYSTEM SHALL be screen-reader-friendly with alt text on all images
  and semantic landmarks
- THE SYSTEM SHALL provide an OG image, Twitter card, and favicon
- THE SYSTEM SHALL include a "What's coming soon" section that contains
  no specific dates and no signup or email-capture form
- WHERE telemetry is enabled THE SYSTEM SHALL fire events for: page view,
  install-CTA click, docs-link click, GitHub-link click
- THE SYSTEM SHALL be deployable via the same Cloudflare workflow as
  `web/docs/src/`, with its source living under `web/landing/` in this repo

## Out of scope

- Pricing page (defer until paid product exists)
- Customer logos (defer until we have permission + meaningful logos)
- Blog index (separate concern — see hero-content-engine)
- Login / dashboard (cloud product, separate)
- CMS, blog engine, auth
- Feature flags or fallback flows — this is a static marketing page

## Open questions

- *(resolved)* Do we need a separate `/install` deep-link page? **No** —
  an anchor link to the install section is enough for v1.
- *(resolved)* Use Hero itself to scaffold via `/mock`? **Yes** — ship the
  `/mock` output as the v1 starting point and hand-edit copy after.
- *(resolved)* Windows install default? **Scoop** is primary; PowerShell
  `irm | iex` is a secondary line.
- *(resolved)* Third coming-soon card? **Tripwire system** (more visceral
  than monorepo support; can revisit if monorepo lands first).
- *(resolved)* NL-routing example count? **Five** — use the full set
  from `web/docs/src/what-is-hero.md`.
- *(resolved)* Codex in harness row? **Yes** — installer is real
  (`internal/install/target_codex.go`).
- *(resolved)* `hero why` placement? **Own card** in Section 3 (Why
  specs), not a one-line callout in Section 1.

## Decisions

- **Domain**: `heroengine.ai` (purchased 2026-05-14). `teamhero.cloud`
  (purchased 2026-04-26) is parked for possible future microsite use.
- **Stack**: plain HTML + Tailwind, Cloudflare Pages, mirror the `web/docs/src/`
  deploy pattern. Locked to remove the original "Astro vs Eleventy vs
  plain HTML" ambiguity.
- **v1 scope**: ship what we have this week — no demo video, no
  comparison table.

## Phasing

- **Phase 1 (this week)**: ship at heroengine.ai. Hero section with
  static screenshot, positioning copy, three-platform install tabs
  (brew / scoop / curl), docs link, core-loop section, the
  natural-language-routing section, four why-specs cards (including
  the `hero why` / `hero blocked` card), works-with-your-tools row
  (Claude Code, Cursor, OpenCode, Codex, Copilot, + any MCP client),
  coming-soon teaser, get-started block, footer. Cloudflare Pages
  deploy from `web/landing/`.
- **Phase 2 (after)**: animated demo / asciinema, comparison table,
  refined telemetry, optional launch banner.

## Changes

Phase 1 (2026-05-15) — files created:

- `web/landing/site/index.html` — production landing page (ported from
  `.hero/mocks/hero-landing-page/index.html`, asset paths fixed to
  in-directory `favicon.svg` / `og-image.svg`, telemetry hook added
  via `window.heroTrack` + `data-track` attributes, UA-based default
  install tab)
- `web/landing/site/favicon.svg` — copied from `web/docs/src/assets/favicon.svg`
- `web/landing/site/og-image.svg` — 1200x630 OG card with bolt + wordmark
- `web/landing/site/robots.txt` — allow all, sitemap pointer
- `web/landing/wrangler.toml` — `name = "hero-landing"`, mirrors root
  `wrangler.toml` pattern
- `web/landing/README.md` — preview / deploy instructions
- `.github/workflows/landing.yml` — gated on `vars.DEPLOY_LANDING ==
  'true'`, mirrors `.github/workflows/docs.yml` structure, uses
  `cloudflare/wrangler-action@v3`
