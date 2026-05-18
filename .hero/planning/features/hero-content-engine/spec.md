---
title: Hero Content Engine — Ongoing Blog, Dev Posts, Case Studies
slug: hero-content-engine
type: feature
status: planning
priority: P1
tags: [marketing, content, blog, seo, growth]
created: 2026-04-25
relations:
  - target: hero-marketing
    kind: parent
  - target: hero-positioning
    kind: depends-on
  - target: hero-docs-site
    kind: depends-on
horizon: someday
smoke: deferred
---

## Goal

Stand up a sustainable content engine that ships one substantive piece
per week post-launch, builds organic search traffic over 6–12 months,
and gives the community something to share when there isn't a launch
moment. Use Hero itself to make content production cheap.

## Problem

Launches are one-shot moments; content is the ongoing surface that
brings people in for years. Without a content engine:
- We rely on launches and word of mouth, both of which decay fast
- New features ship without a public narrative
- SEO traffic from "AI coding workflow", "spec-driven AI", "claude code
  context management" etc. all goes to competitors
- We lose the chance to show technical credibility (deep dives,
  retrospectives, opinions)

A weekly cadence is enough to compound. Daily is too much for a small
team; monthly fades into noise.

## Pillars

Pick 3–4 content pillars and round-robin them. Avoids decision fatigue
and ensures topical balance.

1. **Dispatches** — short ops posts: shipped this week, broken this week,
   community PRs, what we learned. ~500 words. Builds in-the-room feel.
2. **Deep dives** — long-form technical pieces. "How the spec parser
   handles cross-repo references." "Why we chose BM25 over embeddings
   for `hero ask`." 1,500–3,000 words.
3. **Field reports** — case studies from real users (or our own
   dogfooding). "Migrating a 50-spec backlog from Jira to Hero in a
   day." "How CRO @brother runs his pipeline on Hero Sales." 1,000 words.
4. **Opinions** — point-of-view pieces. "Specs are the missing
   abstraction in AI coding." "Why we don't believe in 'agentic'
   automation without humans." 800–1,500 words.

## Workflow

Use Hero to produce content:

1. `/design` a content spec — outline, audience, key points, evidence
   needed
2. `/deliver` the post — drafts the markdown using the project's
   knowledge base for citations
3. `/review` with `pr-reviewer` and a `content-reviewer` agent (new) for
   tone + technical accuracy
4. Publish to docs site `/blog/` + cross-post to dev.to + send via newsletter
5. `/capture` learnings — what got engagement, what didn't, save the
   pattern

The content pipeline becomes a public dogfood: every post is itself
evidence that Hero works.

## Distribution

| Channel | Cadence | Format |
|---|---|---|
| docs.heroengine.ai/blog | Weekly | Canonical |
| dev.to | Weekly (canonical link to docs) | Mirrored |
| Hacker News | Selective (deep dives + opinions) | Submit |
| X / Bluesky | Every post | Thread or single |
| LinkedIn | Selective (field reports + opinions) | Long-form |
| Newsletter | Monthly digest | Curated |
| RSS | Every post | Auto |

## Newsletter

- Hosted on Buttondown or self-hosted via Listmonk
- Monthly digest: top 4 posts + one "behind the scenes" + one community
  highlight
- Sign-up form on landing page (footer) and docs site
- Opt-in only, no purchased lists, double opt-in

## SEO

- Each post targets one primary keyword and 2–3 secondary
- Maintain a topic map in `marketing/content/topics.md` so we don't
  thrash
- Internal linking discipline: every post links to ≥ 2 other posts and
  ≥ 1 docs reference
- Schema.org Article markup auto-applied via mkdocs plugin
- 6-month review of which posts drive traffic; double down + retire dead
  weight

## Editorial calendar

Single source of truth in `.hero/planning/content/calendar.md` —
generated and updated via Hero specs (each post is its own spec under
`.hero/planning/content/<slug>/`).

3 months of topics planned ahead at any time; 6 weeks of drafts queued.

## Acceptance criteria

- One post per week for 12 weeks post-launch with no gaps
- Editorial calendar in the repo, 3 months ahead
- Newsletter live with ≥ 200 subscribers by month 3
- ≥ 5 posts ranked top-10 on their target keywords by month 6
- A community contributor lands a guest post by month 6
- The content pipeline itself is documented as a Hero workflow recipe
  (proof of dogfooding)

## Out of scope

- Paid content amplification — defer until we know what converts
- Video / podcast production — separate effort, defer
- Foreign-language translation
- Sponsored posts on third-party blogs

## Open questions

- Newsletter platform — Buttondown vs Listmonk vs Substack. Lean
  Buttondown (clean, indie-friendly, owns own data).
- Comments on blog posts — yes (giscus → GitHub Discussions) or no
  (less moderation surface area)?
- First post topic — "Introducing Hero" launch piece, or a deep dive
  that earns the launch?
