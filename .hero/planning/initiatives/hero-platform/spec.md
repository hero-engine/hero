---
title: Hero Platform — Headless Execution, Team Automation, and Shared Visibility
slug: hero-platform
type: initiative
status: planning
tags: [platform, runner, automations, team, cloud]
created: 2026-04-23
relations:
  - target: hero-cloud
    kind: related
child:
  - always-on-runtime
horizon: next
size: giant
---

## Goal

Transform Hero from a single-session CLI into a platform that runs agent
work headlessly, triggers automations from external events, coordinates
team members across sessions, and surfaces everything through a visual
dashboard. Same binary, new modes.

## Problem

Today, all Hero agent work happens inside a chat session (Claude Code,
Cursor, OpenCode). This means:

- Work dies if the session dies — no "fire and forget"
- Automations can't run unattended — someone must sit in a chat and type
- Team members can't see what others' agents are doing
- There's no approval workflow — an agent either runs or doesn't
- Always-on triggers (watch Jira, respond to webhooks) need a persistent
  process that no laptop provides

Hero has all the pieces (specs, tools, context, conventions) but no runner
to drive them without a chat UI, no event system to trigger them
automatically, and no shared state to coordinate across a team.

## Architecture

```
┌─────────────────────────────────────┐
│  Hero Cloud (optional, hosted)      │
│  - Managed runner pool              │
│  - Webhook ingress                  │
│  - Team dashboard (hosted)          │
│  - SSO + org management             │
└──────────────┬──────────────────────┘
               │ sync / API
┌──────────────┴──────────────────────┐
│  hero serve --team (self-hostable)  │
│  - Job queue + runner workers       │
│  - Automation engine (triggers)     │
│  - Approval gate system             │
│  - Dashboard web UI                 │
│  - Team state (who's doing what)    │
└──────────────┬──────────────────────┘
               │ events / feed
┌──────────────┴──────────────────────┐
│  Hero Sessions (laptops, IDEs)      │
│  - Claude Code / Cursor / OpenCode  │
│  - Interactive work + reviews       │
│  - Post events to team feed         │
│  - Receive approval requests        │
└─────────────────────────────────────┘
```

One binary (`hero`), four modes:
1. `hero mcp` — MCP server for AI tool sessions (exists)
2. `hero serve` — HTTP daemon + dashboard (exists, extended)
3. `hero <cmd>` — CLI commands for humans (exists)
4. `hero run` — headless agent execution (NEW)

## Children

| Slug | Title | Priority | Effort |
|---|---|---|---|
| hero-runner | Headless Agent Execution | P0 | L |
| hero-automations | Event-Driven Automation Engine | P1 | L |
| hero-team-server | Team Job Queue + Approval Gates | P1 | L |
| hero-dashboard-v2 | Visual Dashboard for Jobs, Automations, Team State | P1 | L |

## Sequencing

1. **hero-runner** first — it's the execution engine everything else depends on.
   Automations and team server both use runner to execute jobs.
2. **hero-automations** + **hero-team-server** can be built in parallel after
   runner ships — automations is the event→job layer, team-server is the
   coordination + queue layer.
3. **hero-dashboard-v2** layers on top of team-server — it's the visual face
   of the job queue, automation status, and team state.

## Relation to Hero Cloud

Hero Cloud is hero-team-server hosted by us. The cloud specs
(cloud-auth, cloud-api, cloud-sync, etc.) add SSO, billing, and
multi-org — but the core runtime is the same Go binary. Teams that
can't or won't use cloud self-host `hero serve --team` on their own
infrastructure.
