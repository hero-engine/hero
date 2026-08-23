---
title: Hero Demo Content — GIFs, Screencast, Asciinema, Social Cards
slug: hero-demo-content
type: feature
status: planning
priority: P0
tags: [marketing, demo, content, video, social]
created: 2026-04-25
relations:
  - target: hero-positioning
    kind: depends-on
horizon: someday
smoke: deferred
superseded_by: hero-continuity-proof-demo
# superseded_reason: Replaced broad demo-asset scope with a bounded cross-tool continuity proof.
---

## Goal

Produce the visual artifacts a launch run needs — animated GIF for the
hero shot, a 60–90 second screencast for blog posts and the docs, an
asciinema cast for terminal-friendly embedding, and Open Graph / Twitter
card images for every shareable surface. Reuse `hero demo record` where
possible.

## Problem

You can describe spec-driven workflow in 1,000 words or you can show
it in 30 seconds. Today we have neither — no GIFs, no screencast, no
social card. Every share of a Hero link in a tweet or HN thread renders
as a blank rectangle, which kills click-through. Every blog draft stalls
because there's nothing to embed.

## Deliverables

### 1. Hero shot GIF (or autoplay video)
- 6–10 seconds, < 2MB
- Loops cleanly
- Shows the core loop in compressed form: typing `/design …`, spec
  appearing, `/deliver`, code shipping, knowledge captured
- Lives on the landing page above the fold

### 2. Screencast (60–90 seconds)
- One real example, one take, no marketing voiceover
- Demonstrates: install → init → /design → /deliver → /capture
- Hosted on YouTube + self-hosted MP4 fallback
- Embeddable in docs and blog
- Subtitles included (auto-generated, hand-edited)

### 3. Asciinema cast
- Terminal-only version of the screencast for the README and docs
- Same flow, no GUI scenes
- Self-hosted .cast file (asciinema-player JS lib in landing/docs)
- Loadable as a fallback when video can't autoplay

### 4. Per-feature GIFs
- Short loops (4–6s each) for the major features:
  - `/design` producing a spec
  - `/deliver` with context injection
  - `/diagnose` finding root cause
  - `hero recap`
  - `hero pulse`
  - `hero serve` dashboard (when v2 ships)
- Used inline in docs and feature pages

### 5. Open Graph + Twitter card images
- One default OG image (1200×630) for every page that doesn't override
- Per-page generated OG images for:
  - Landing
  - Quickstart
  - Each major workflow (/design, /deliver, /diagnose)
  - Each blog post
- Generation: a small Go or Node script that takes a title + subtitle
  and renders to PNG with the brand template. Runs in CI.

### 6. Brand assets folder
- `marketing/assets/` in this repo (or sibling repo)
- Contains: logo SVG (light + dark), wordmark, OG template, favicon set,
  social avatar, screenshot frame template (laptop bezel, terminal chrome)
- Documented usage rules in `marketing/BRAND.md`

## Production approach

- **Use `hero demo record`** for the screencast and per-feature GIFs.
  The infrastructure already exists — Playwright-driven runs against
  acceptance criteria with video capture. We dogfood the product to
  produce its own marketing.
- **Manual cuts** for the asciinema cast — `asciinema rec`, edit if
  needed with `asciinema-trim` or just re-record cleanly.
- **OG generator** — keep simple. A single template, parameterized.
  Don't build a Figma plugin.
- **Subtitles** — Whisper for first pass, hand-correct. Save to `.vtt`
  alongside the video.

## File budget

- Hero GIF: < 2MB
- Per-feature GIFs: < 1MB each
- Screencast MP4: < 10MB at 720p (host on YouTube primarily)
- Asciinema cast: < 100KB
- OG images: < 200KB each, PNG with WebP fallback

## Acceptance criteria

- Hero shot GIF lives on the landing page and renders in < 1s
- Screencast has captions and is embedded in docs `/quickstart`
- Asciinema cast is in the README + docs as a fallback
- Every landing page section has at least one supporting visual
- OG image renders correctly when a Hero URL is shared on Twitter,
  LinkedIn, Slack, Discord, and Bluesky
- Brand assets folder exists with documented usage
- Generation pipeline (OG images, GIFs from `hero demo record`) runs
  from a single Make target

## Out of scope

- Animated brand intro / outro — keep it raw and informal
- Multi-language subtitles
- Photography of humans / customer footage — defer until we have
  customers willing to be on camera
- Long-form YouTube channel — that's hero-content-engine territory

## Open questions

- Style: terminal-only (legible, niche) or include the dashboard
  (broader appeal but adds visual noise)? Lean terminal-led with one
  dashboard cameo.
- Light or dark mode in screencasts? Dark reads better on most embeds.
- Music in the screencast or silence? Silence — keeps file small and
  doesn't trigger copyright weirdness.
- Use the founder's voice for narration, no narration, or text overlays
  only? Lean text overlays for v1.
