---
title: Canonical Web Domain — heroengine.ai
type: decision
status: accepted
created: 2026-05-14
tags: [marketing, domain, branding, infrastructure]
relations:
  - target: hero-marketing
    kind: resolves
  - target: hero-landing-page
    kind: informs
  - target: hero-docs-site
    kind: informs
  - target: hero-distribution
    kind: informs
  - target: hero-telemetry
    kind: informs
  - target: hero-community
    kind: informs
---

# Canonical Web Domain — heroengine.ai

## Decision

`heroengine.ai` is the canonical public domain for the Hero product.

- Root: `heroengine.ai` — landing page, install script, privacy policy, RSS
- Docs: `docs.heroengine.ai`
- Telemetry collector: `collector.heroengine.ai`
- Other subdomains as needed (`status.`, `blog.`, etc.) live under
  the same root

The product is called **Hero** in user-facing copy everywhere — the CLI,
docs prose, marketing. "Hero Engine" is not a brand surface; it is a
URL/org-name artifact only (same shape as `nextjs.org` for `next.js`).

`teamhero.cloud` (purchased 2026-04-26) is **parked**, not retired —
held for possible future microsite use. Not redirected, not abandoned.

## Context

Four planning specs had drifted to different assumptions:

- `hero-landing-page` and `hero-docs-site` named `teamhero.cloud`
  (already owned)
- `hero-distribution`, `hero-telemetry`, `hero-community` aspirationally
  used `hero.dev` (never owned)
- `hero-marketing` initiative had an explicit open question:
  *"do we have hero.dev, gohero.io, hero.so, etc.? Need to lock one."*

The GitHub org is `hero-engine`. The `.ai` TLD signals AI-tooling
category cleanly. `heroengine.ai` aligns the org, install URL, and
docs domain with a single coherent shape and resolves the open
question without forcing a rename of any existing surface.

## Alternatives considered

- **`hero.dev`** — clean, but not owned; would have required acquisition
  on the secondary market at unknown cost
- **`teamhero.cloud`** — already owned, but `.cloud` reads as
  enterprise/SaaS and the "team" prefix dates the brand to the
  pre-community phase
- **`gohero.io` / `hero.so`** — speculative; no specific advantage over
  `heroengine.ai`

## Hosting (related decision)

Both the landing page and the docs site deploy to **Cloudflare Pages**
connected to the private GitHub repo. Free, unlimited bandwidth,
native PR previews, single dashboard with the Cloudflare DNS fronting
the domain. This supersedes the earlier "GitHub Pages from `gh-pages`
branch" plan in `hero-docs-site`.

## Consequences

- Specs updated to point at `heroengine.ai` (see relations)
- `mkdocs.yml` `site_url` set to `https://docs.heroengine.ai`
- Cloudflare DNS for `heroengine.ai` is the source of truth for all
  Hero-product DNS
- Email forwarding for `heroengine.ai` runs on Cloudflare Email
  Routing (free); paid mailbox added when send-as capability is needed
- `teamhero.cloud` continues to renew until a use is decided or
  dropped at a later renewal

## Status of dependent specs

- `hero-landing-page` — domain field updated
- `hero-docs-site` — domain + hosting updated, IA tree uses
  `docs.heroengine.ai`
- `hero-distribution` — install URL + RSS URL updated; open question
  on `install.sh` URL resolved
- `hero-telemetry` — collector URL, privacy policy URL, docs URL
  updated; open question on `collector.hero.dev` resolved
- `hero-community` — Hall of Fame URL updated
- `hero-marketing` — open questions on domain and hosting closed
