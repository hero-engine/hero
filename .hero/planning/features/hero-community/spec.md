---
title: Hero Community — Discord/Discussions, Contributor Guide, Issue Templates
type: feature
status: planning
priority: P1
tags: [marketing, community, support, contributors]
created: 2026-04-25
relations:
  - target: hero-marketing
    kind: parent
horizon: someday
smoke: deferred
---

## Goal

Open a single, obvious place where Hero users hang out, ask questions,
and contribute back. Make it easy to file a useful bug report, propose
a feature, or land a first PR. Build the early-adopter base into a
support flywheel before paid plans exist.

## Problem

There's nowhere to send "hey, I tried Hero and have questions". GitHub
issues are too formal for "is this the right command?" and too slow for
real-time. Without a community surface:
- Early users churn quietly when stuck
- Questions get repeatedly DMed to the founder
- We don't see what people are confused by until they tweet about it
- Contributors don't have a clear path from "this would be cool" to
  merged PR

A community surface costs a few hours per week to maintain and pays
back in support cost reduction, product feedback, and word-of-mouth.

## Surface choice

**Recommend GitHub Discussions for v1; revisit Discord at 200+ active users.**

| | Discord | GitHub Discussions |
|---|---|---|
| Setup cost | Medium (server, bots, mods) | Low (already have GH repo) |
| Engagement ceiling | High (live chat) | Medium (async forum) |
| Search / discoverability | Poor (locked behind login) | Good (indexed) |
| Moderation surface | High | Lower |
| Time-to-first-response expectation | Minutes | Hours |
| Where contributors already are | Maybe | Yes |
| Cost when active | Real time investment | Reasonable |

GitHub Discussions has lower ceiling but better fit for the launch
phase. Move to Discord (or add it alongside) when we have the volume
to justify and a moderator.

## Setup

### GitHub Discussions

Categories:
- **Announcements** (maintainer-only) — releases, breaking changes
- **Q&A** — user questions; mark answered
- **Show & tell** — what people built with Hero
- **Ideas** — feature requests, before they become issues
- **Sales domain** — once Hero Sales is live (separate so engineering
  threads aren't drowned)
- **Polls** — short, occasional decisions ("should we ship X?")

### Issue templates

`.github/ISSUE_TEMPLATE/`:
- `bug_report.yml` — version, install path, repro, expected/actual,
  logs (`hero --version`, `hero check` output prefilled)
- `feature_request.yml` — problem, proposed solution, alternatives
- `docs_issue.yml` — page, what's wrong, suggestion
- `config.yml` — disable blank issues, link to Discussions for questions

### PR template

`.github/PULL_REQUEST_TEMPLATE.md`:
- Linked spec / issue
- What changed
- How tested
- Risk assessment
- Checklist (tests, docs updated, knowledge captured)

### Contributor guide (`CONTRIBUTING.md`)

- Setup: clone, `make build`, `make test`
- Architecture orientation: pointer to repo layout in README
- Where to start: "good first issue" label, suggested entry points
- How to design — link to Hero's own design docs (we use specs!)
- How to submit — PR flow, review expectations, response SLA
- Code of conduct (CoC) — link to a standard one (Contributor Covenant)

### Code of Conduct

`CODE_OF_CONDUCT.md` — Contributor Covenant 2.1, lightly customized.

### Maintainer commitments

- Triage new issues + discussions within 48 hours on weekdays
- Label and link to specs
- Respond to PRs within a week (review or "next steps")
- Quarterly maintainer post acknowledging contributors

## Onboarding hooks

- After successful install, `hero` prints a one-liner: "Questions or
  feedback? github.com/hero-engine/hero/discussions"
- Landing page footer: Community link
- Docs site: footer + a "Need help?" callout in `/quickstart`
- Error messages that mention community where appropriate
  ("If this is unexpected, please file at <link>")

## Recognition

- README "Contributors" section auto-updated by all-contributors bot
- Every release notes contributors by GitHub handle
- A Hall of Fame page on docs.hero.dev for first 50 contributors
  (one-time, retired after that)

## Acceptance criteria

- GitHub Discussions enabled with category structure
- Issue + PR templates committed and rendering correctly
- `CONTRIBUTING.md` and `CODE_OF_CONDUCT.md` committed
- "Community" link in landing footer + docs footer + README
- Triage SLA documented and publicly visible (in CONTRIBUTING.md)
- One pinned "Welcome — start here" Discussions thread
- Hero CLI prints community link on first run

## Out of scope (for v1)

- Discord server (add later)
- Slack-style real-time chat
- Paid support tier
- Office hours / community calls (defer to month 3+ if traction
  warrants)
- Bug bounty program

## Open questions

- Pin a "introduce yourself" thread or skip the cliché?
- All-contributors bot or hand-curated CONTRIBUTORS.md?
- How aggressive on enforcing CoC violations — written rubric or
  case-by-case?
- Do we open source the marketing repo / landing page or keep that
  closed?
