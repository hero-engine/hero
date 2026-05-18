---
title: Hero Launch Playbook — Show HN, Reddit, X, Podcast Outreach
slug: hero-launch-playbook
type: feature
status: planning
priority: P0
tags: [marketing, launch, distribution, growth]
created: 2026-04-25
relations:
  - target: hero-marketing
    kind: parent
  - target: hero-positioning
    kind: depends-on
  - target: hero-landing-page
    kind: depends-on
  - target: hero-distribution
    kind: depends-on
  - target: hero-demo-content
    kind: depends-on
horizon: someday
smoke: deferred
---

## Goal

Produce a coordinated, time-boxed launch — landing page, install paths,
docs, and demo all live, all linked, with a sequence of public posts
across the channels engineering leads actually read. The goal is not
"go viral" — it's to put Hero in front of 10,000 of the right people in
a 7-day window and convert ~1% into trials.

## Problem

A launch without a playbook is one tweet that dies in 4 hours. A launch
playbook turns the surface area we built (positioning, landing, docs,
distribution, demo) into a sequence of moments that compound:
each post links to the next, each comment thread feeds the next post,
each metric tells us where to push.

## Channels

| Channel | Audience | Format | Slot |
|---|---|---|---|
| Show HN | Engineering leads, AI tinkerers | Title + URL + comment | Day 1 morning ET |
| /r/programming | Developers (broad) | Title + URL + brief comment | Day 1 afternoon |
| /r/LocalLLaMA, /r/ClaudeAI | AI-native devs | Title + URL + tailored intro | Day 2 |
| Twitter/X thread | Followers + algo | 6–10 tweet thread | Day 1 morning |
| Bluesky thread | Same as X, slightly different audience | Same thread | Day 1 morning |
| LinkedIn post | Eng leads, CTOs | Long-form post | Day 2 |
| Hacker Newsletter / TLDR / Bytes | Newsletters | Pitch email + asset pack | Pre-launch (Day -7) |
| Indie Hackers / Lobsters | Builder-adjacent | Title + URL | Day 3 |
| Podcasts | Eng leadership | Outreach for spots | Pre + ongoing |
| Product Hunt | Builder + PM crowd | Full PH listing | Day 7 (extends the cycle) |

## Pre-launch (T-14 to T-1)

1. **Asset pack ready** — landing page, docs site, install paths,
   demo GIFs, screencast, OG images, comparison page (all P0 children
   of this initiative complete)
2. **Pre-launch list** — collect 50–100 emails from people who said
   "tell me when it's live" via README, Twitter DMs, network
3. **Newsletter pitches** — email Hacker Newsletter, TLDR, Bytes,
   Console.dev with the asset pack 7 days before launch
4. **Friendly preview** — share the landing + docs with 5–10 trusted
   developers a few days early; capture a quote or two
5. **Show HN draft** — write title + comment in advance; A/B test the
   title with 3–5 friends
6. **X/Bluesky thread draft** — write the thread, screenshots ready,
   schedule for morning launch
7. **HN account warmth** — make sure the account posting Show HN has
   karma and history (not a fresh account — they get auto-flagged)
8. **Tier-1 supporters** — line up 5–10 people who'll upvote/comment
   in the first hour without it looking coordinated

## Launch day (T-0)

| Time (ET) | Action |
|---|---|
| 7:00 | Pre-launch email goes out to waitlist |
| 7:30 | Show HN posted |
| 7:45 | X/Bluesky thread published |
| 8:00 | Personal post to LinkedIn |
| 9:00 | Post to /r/programming |
| 10:00 | Reply to Show HN top comments |
| 12:00 | Post to /r/ClaudeAI |
| 14:00 | Reply to all open threads |
| 17:00 | Day-1 metrics snapshot, decide next-day adjustments |

The author stays on a single laptop, in a quiet room, replying to
comments in real time. No multitasking.

## Day 2–7

- Daily comment triage on every active thread
- One follow-up post per day with a different angle:
  - Day 2: behind-the-scenes ("how Hero builds itself")
  - Day 3: technical deep-dive ("the spec parser")
  - Day 4: case study ("X engineer's first week with Hero")
  - Day 5: comparison ("Hero vs raw CLAUDE.md")
  - Day 6: roadmap ("what we're building next")
  - Day 7: Product Hunt launch
- Track every install (telemetry), every star, every PR, every issue
- Publish a Day-7 retro on the blog — what worked, what didn't, what's
  next

## Content artifacts (deliverables)

- `marketing/launch/show-hn.md` — title + comment + comment-thread
  prepared replies
- `marketing/launch/twitter-thread.md` — thread copy + image refs
- `marketing/launch/linkedin.md` — long-form post
- `marketing/launch/reddit-{subreddit}.md` — per-subreddit copy
- `marketing/launch/newsletter-pitches.md` — outreach copy + asset pack
  link
- `marketing/launch/podcast-pitch.md` — generic + per-podcast pitch
- `marketing/launch/checklist.md` — full T-14 → T+7 timeline
- `marketing/launch/retro-template.md` — Day-7 retro template

All committed to this repo so the launch is reproducible (and so we
have a prior to copy from on v1.1, v2, etc.).

## Acceptance criteria

- Every channel has a draft committed before T-7
- Launch checklist is followed end-to-end and the retro is published by
  T+10
- Metrics from launch week feed into hero-telemetry dashboards
- A second-launch playbook (template) exists by T+14, derived from the
  retro

## Anti-goals

- Don't fake reviews. Don't astroturf comments. Don't pay for upvotes.
  HN and Reddit will catch it and the founder reputation cost is huge.
- Don't oversell. If it breaks for someone in the first hour, fix or
  acknowledge — don't deflect.
- Don't launch before the team-platform story is dialed in. NEXT.md is
  explicit. A premature launch wastes the cycle.

## Open questions

- Pick the launch date — coordinate with team-platform readiness
- Decide whether to do a paid PH "Featured" slot or rely on organic
- Solo-launch vs co-launch with a friendly partner (e.g. an LLM API
  vendor, a complementary tool)
- How aggressive on podcast outreach — 5 pitches or 30?
