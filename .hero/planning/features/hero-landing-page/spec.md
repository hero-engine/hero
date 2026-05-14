---
title: Hero Landing Page — Public Homepage with Install CTA
type: feature
status: planning
priority: P0
tags: [marketing, landing, web, conversion]
created: 2026-04-25
relations:
  - target: hero-marketing
    kind: parent
  - target: hero-positioning
    kind: depends-on
  - target: hero-distribution
    kind: depends-on
  - target: hero-demo-content
    kind: depends-on
horizon: someday
smoke: deferred
---

## Goal

A single-page site that converts a curious visitor — someone who clicked
through from a tweet, HN comment, or a friend's recommendation — into a
trial install in under three minutes. Optimize for clarity over polish.

## Problem

Today, the GitHub README is the public face of Hero. README is fine for
people who already know they want to install something, but it's a poor
first impression for a developer evaluating tools. No hero shot, no demo
GIF above the fold, no clear differentiation from Cursor/Aider, no
visible install path that doesn't require reading 800 lines.

A landing page lets us:
- Lead with the demo, not the changelog
- Compress 800 lines of README into 7 sections that answer the
  questions visitors actually have
- Provide a stable, shareable URL for launch posts
- Capture signal (visits, scroll depth, install clicks) we can't
  capture from GitHub

## Page structure

### Above the fold
- **Hero shot** — animated GIF or autoplay video of the core loop
  (`/design` → `/deliver` → captured knowledge), ≤ 8 seconds
- **One-liner** — exact copy from positioning doc
- **Sub-line** — 1–2 sentences elaborating
- **Primary CTA** — `brew install hero-engine/tap/hero` with a copy button
- **Secondary CTA** — "View on GitHub" → repo

### Section 1 — The core loop
- 4-step diagram (Discover → Design → Deliver, with Diagnose branching in)
- One sentence per step
- Tiny screenshot/GIF for each

### Section 2 — Why specs
- 3 cards: "Design before you build", "Knowledge compounds",
  "Tool-agnostic"
- Each card: one-line claim, one-paragraph proof, one screenshot

### Section 3 — Works with your tools
- Logo row: Claude Code, Cursor, OpenCode, Copilot, "+ any MCP client"
- One-line install per tool

### Section 4 — Live demo
- Asciinema embed or longer screencast
- Caption: "30 seconds — design a spec, deliver it, capture the learning"

### Section 5 — Comparison
- Compact table: Hero vs raw CLAUDE.md vs Cursor vs Aider
- 5 rows max — workflow, knowledge persistence, multi-agent, tool-agnostic,
  team mode
- Link to full comparison page

### Section 6 — Get started
- 3-step quickstart (install, init, first spec)
- Code blocks copy-paste-able
- "Read the docs" link to docs site

### Section 7 — Footer
- GitHub, Discord/Discussions, docs, blog, status (later)
- Company/license note (MIT, "by @chet-bellows")

### Optional banner
- Pre-launch: waitlist email capture
- Post-launch: "v1.0 is live — read the announcement"

## Tech stack

- **Static site** — no SSR needed. Astro, Eleventy, or just plain HTML +
  Tailwind. Pick by familiarity.
- **Hosting** — Cloudflare Pages or Vercel free tier. Custom domain
  (depends on hero-positioning open question).
- **Analytics** — Plausible or self-hosted PostHog (see hero-telemetry).
  No GA, no FB pixel.
- **No JS framework needed** — keep bundle under 100KB total.
- **Repo** — `web/landing/` inside this repo, not a separate repo, so
  the docs/landing/marketing copy stays version-controlled with the
  product.

## Performance budget

- LCP < 1.5s on 4G
- Total page weight < 500KB excluding the demo video
- Demo video lazy-loaded; GIF placeholder on first paint
- 100/100 Lighthouse on accessibility (WCAG AA, alt text, contrast)

## Acceptance criteria

- Lives at the chosen domain and resolves with HTTPS
- Above-the-fold renders without the demo asset on first paint
- All install commands have a one-click copy button
- Page is keyboard-navigable, screen-reader-friendly
- OG image, Twitter card, and favicon all present
- Source lives in this repo and can be edited by the same flow as docs
- Telemetry fires for: page view, install-CTA click, GitHub click,
  docs click, demo play

## Out of scope

- Pricing page (defer until paid product exists)
- Customer logos (defer until we have permission + meaningful logos)
- Blog index (separate concern — see hero-content-engine)
- Login / dashboard (cloud product, separate)

## Open questions

- Do we need a separate `/install` deep-link page for ad-hoc share, or
  is anchor-link to the install section enough?
- Use Hero itself to scaffold the landing page (`/mock`) — yes,
  but does the output ship or do we hand-author after?

## Decisions

- **Domain**: `heroengine.ai` (purchased 2026-05-14). `teamhero.cloud` (purchased 2026-04-26) is parked for possible future microsite use.
