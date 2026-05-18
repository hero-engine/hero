---
title: Hero v2 — System Design Initiative
slug: hero-v2-system-design
type: initiative
status: completed
tags: [v2, initiative, architecture]
created: 2026-04-12
relations:
  - target: spec-types-lifecycle
    kind: child
  - target: conventions-and-decisions
    kind: child
  - target: context-injection
    kind: child
  - target: team-coordination
    kind: child
  - target: v2-agents-and-skills
    kind: child
  - target: v2-commands
    kind: child
  - target: index-extensions
    kind: child
  - target: hero-serve-daemon
    kind: child
  - target: bulk-issue-import
    kind: child
  - target: sprint-planner
    kind: child
horizon: now
---

## Goal

Hero v2 transforms Hero from a solo productivity tool into the institutional memory layer for an entire engineering team. Every AI agent session benefits from collective project knowledge: conventions, past architectural decisions, bug root causes, and in-flight work.

## Delivered

All v2 features have been implemented. See child specs for details:

| Spec | What it delivered |
|---|---|
| `spec-types-lifecycle` | 9 spec types, full lifecycle state machine |
| `conventions-and-decisions` | Convention and decision specs, `/convention`, `/decide` commands |
| `context-injection` | `hero context`, convention scope matching, delivery lead integration |
| `team-coordination` | Claiming, conflict detection, `hero check`, spec review workflow |
| `v2-agents-and-skills` | 4 new agents, 7 new skills |
| `v2-commands` | `/compose`, `/convention`, `/decide`, `/retro`, `/check` |
| `index-extensions` | Relations, tags, type/status/date filters, `hero graph` |
| `hero-serve-daemon` | MCP server, HTTP API, file watcher, event stream |
| `bulk-issue-import` | GitHub, Jira, Linear import |
| `sprint-planner` | Agent-driven sprint planning from local backlog |

## Key Architectural Decisions

- Claiming is advisory (git-committed frontmatter) — no lock server, no distributed state
- No LLM calls from the hero binary — Hero is a corpus manager and context generator; AI work stays in the agent tool
- Convention enforcement is advisory — agents are instructed to follow conventions; the binary does not block builds
- FTS5 with porter stemming is sufficient at project scale — semantic search deferred
- Tracker and wiki sync: config shape defined in v2, implementations delivered in v2 (GitHub) and v0.2 (Jira deep, Confluence)
