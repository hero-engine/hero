---
title: Hero Telemetry — Opt-In Usage Analytics + Feedback Channel
slug: hero-telemetry
type: feature
status: planning
priority: P1
tags: [marketing, telemetry, analytics, privacy, feedback]
created: 2026-04-25
relations:
  - target: hero-marketing
    kind: parent
  - target: hero-launch-playbook
    kind: related
horizon: someday
smoke: deferred
---

## Goal

Know who's actually using Hero — where they install, what commands they
run, where they get stuck, what features they ignore — without
violating user trust. Build a privacy-respecting, opt-in telemetry
system that we can ship at launch and use to drive product decisions
for years.

## Problem

Today we know nothing about real usage. GitHub stars are vanity. Homebrew
analytics are crude. Without telemetry:
- We don't know which commands are dead weight
- We don't know where the funnel leaks (install → init → first spec)
- We can't measure if changes ship improvements
- We can't quantify launch impact beyond "the chart went up"

The trade-off is privacy. Developer tools that ship surveillance burn
trust permanently. The play is: opt-in, transparent, minimal, anonymous.

## Principles

1. **Opt-in, not opt-out.** No data leaves the user's machine until
   they say yes.
2. **First-run prompt.** Clear language, easy to decline, easy to
   change later.
3. **Transparent.** A `hero telemetry --show` command prints exactly
   what would be sent.
4. **Anonymous.** No paths, no project names, no spec content, no API
   keys, no IPs (we do not log IPs at the receiver).
5. **Sampled, not surveilled.** Aggregate counts, not per-event traces.
6. **Self-hostable.** The schema and endpoint are public; teams can
   point at their own collector via env var.
7. **Easy to leave.** `hero telemetry --disable` and we forget.

## What we collect

| Event | Fields |
|---|---|
| `install` | os, arch, install_method (brew/script/source), hero_version |
| `init` | os, arch, hero_version, domain (engineering/sales) |
| `command` | os, hero_version, command_name, exit_code, duration_bucket |
| `mcp_tool` | os, hero_version, tool_name |
| `runner_job` | provider, model, success, duration_bucket, turn_count_bucket |
| `update` | from_version, to_version |
| `uninstall` | hero_version, days_since_install |

What we explicitly **don't** collect:
- File paths, project names, repo URLs, branch names
- Spec content, knowledge content, code content
- API keys, tokens, env var values
- IP addresses at the receiver (proxy strips them)
- Anything that could identify a person, project, or company

Each event carries a stable, anonymous installation ID (UUID generated
at first run, stored at `~/.hero/install-id`). Resets if the user runs
`hero telemetry --reset`.

## Architecture

```
┌──────────────┐
│  hero CLI    │
│  hero serve  │  ─── HTTPS POST ──▶  collector.heroengine.ai
└──────────────┘                      (proxy, strips IPs)
                                              │
                                              ▼
                                       PostHog (self-hosted)
                                       or Plausible Events
                                              │
                                              ▼
                                       Internal dashboard
```

- Events queue locally, batch-flush every 10 minutes or on graceful
  shutdown
- Network failures silent — we never block the CLI on telemetry
- Total daily traffic: ≤ 5 KB per active install
- Collector strips IPs, applies rate limits, and writes to the analytics
  backend

## First-run prompt

```
Hero collects anonymous usage data to help improve the tool.
We never see your code, specs, file paths, or who you are.
See exactly what's sent: hero telemetry --show

Allow anonymous telemetry? [Y/n]
```

The prompt:
- Defaults to Y in interactive mode
- Defaults to N in non-interactive mode (CI, scripts) — explicit opt-in only
- Honored permanently in `~/.hero/config.yml`
- Honored across upgrades (no re-prompting)

## Commands

```bash
hero telemetry              # show current status
hero telemetry --enable     # opt in
hero telemetry --disable    # opt out, stop sending
hero telemetry --show       # print recent events that would be sent
hero telemetry --reset      # rotate install ID
hero telemetry --endpoint <url>   # point at a custom collector
```

Environment variables:
- `HERO_TELEMETRY=0` — hard opt-out, overrides config
- `HERO_TELEMETRY_ENDPOINT=<url>` — custom collector

## Backend choice

**Recommend PostHog (self-hosted) for v1.** Open source, supports
funnels, retention, cohorts. Self-host on a $20 VPS or Hetzner box.

Alternative: Plausible (simpler, friendlier privacy story) — but
limited to page views + custom events; harder to do funnel analysis.

We can switch later — the CLI emits a stable schema, the collector is
just a transport.

## Feedback channel

Telemetry tells us what's used; feedback tells us why. Two complementary
channels:

1. **In-CLI feedback** — `hero feedback "<message>"` posts an anonymous
   note (with optional opt-in email reply-to). Same opt-in as telemetry.
2. **Survey nudges** — at session count milestones (10, 50, 100), the
   CLI shows a one-time "What's working / what's not?" prompt with a
   docs link to a Tally or Typeform short survey.

## Acceptance criteria

- Opt-in prompt fires once on first run; never again unless reset
- `hero telemetry --show` prints actual queued events, no surprises
- Collector strips IPs and rate-limits aggressively
- Privacy policy page exists at heroengine.ai/privacy and is linked from
  the prompt
- Telemetry is documented in detail at docs.heroengine.ai/telemetry
- Total network traffic ≤ 5 KB/day per active install
- A backend dashboard answers: weekly active installs, command
  popularity, install→init→first-spec funnel, retention by week
- Independent reviewer (a privacy-savvy developer) signs off on the
  collected events

## Out of scope

- Per-user identification, even pseudonymous
- Crash reporting / Sentry-style error capture (separate, also opt-in)
- A/B experimentation framework
- Per-team analytics for self-hosted Hero (defer to team-server work)
- Selling or sharing telemetry data with anyone, ever (anti-goal)

## Open questions

- Is there a way to publish telemetry results publicly (e.g. monthly
  numbers post) to build trust + feed content engine?
- Do we delay first-run prompt until after the first command (less
  intrusive) or hit it at install (clearer informed consent)?
- ~~Domain decision blocks `collector.hero.dev` URL~~ — resolved: `collector.heroengine.ai`
- Hosting: own VPS vs PostHog Cloud paid plan? Lean self-host for
  privacy story.
