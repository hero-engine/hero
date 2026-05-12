---
title: Hero Domains — Platform Architecture for Non-Engineering Verticals
type: initiative
status: planning
tags: [platform, domains, sales, marketing, research, vertical]
created: 2026-04-25
relations:
  - target: hero-platform
    kind: related
horizon: someday
---

## Goal

Transform Hero from an engineering-specific tool into a domain-agnostic
platform where the core engine (specs, knowledge, agents, automations,
runner, dashboard) stays the same but the domain content (agents, skills,
commands, integrations) swaps based on the user's function. Start with
sales as the first non-engineering domain.

## Problem

Hero's core loop — design before you act, capture knowledge, coordinate
specialists, automate repetitive work — isn't unique to software
engineering. Sales teams design deal strategies, capture competitive intel,
coordinate specialists (SE, legal, exec sponsor), and automate pipeline
hygiene. Research teams design studies, capture literature reviews,
coordinate collaborators, and automate paper tracking. The workflow is
universal; the vocabulary and integrations are domain-specific.

Today Hero's agents, skills, and commands are hardcoded for engineering.
A sales user would need to rip out 33 agents and write new ones. The
platform architecture should make this a configuration choice, not a
fork.

## Architecture

### Domain packs

A domain is a directory of agents/, commands/, skills/, and an optional
integrations/ folder:

```
domains/
  engineering/           # what we ship today (default)
    agents/              # 33 engineering agents
    commands/            # 30+ engineering commands
    skills/              # 40+ engineering skills
    integrations.json    # github, jira, linear
    AGENTS.md            # engineering routing table
  sales/
    agents/              # sales-specific agents
    commands/            # sales-specific commands
    skills/              # sales-specific skills
    integrations.json    # salesforce, hubspot, gong
    AGENTS.md            # sales routing table
  research/
    agents/              # research agents (future)
    ...
```

### Domain selection

```bash
hero init --domain sales           # new project with sales domain
hero init --domain engineering     # default, same as today
hero domain switch sales           # switch an existing project
hero domain list                   # list available domains
```

The domain is stored in `hero.json`:

```json
{
  "domain": "sales",
  "folder": ".hero"
}
```

### What's domain-agnostic (the core engine)

Everything in `internal/` stays the same:

- Spec parser, lifecycle, discovery (`internal/spec/`)
- Knowledge base, ingest, lint (`internal/index/`, CLI commands)
- Drift detection, impact analysis, coverage (`internal/drift/`, etc.)
- Runner, automations, job queue (`internal/runner/`, `internal/automations/`)
- Dashboard, team server, MCP protocol (`internal/serve/`)
- Session management, NEXT.md, handoff (`internal/sessions/`)
- Recap, pulse, velocity, cost (`internal/recap/`, `internal/pulse/`)
- Cross-repo, feed, active sessions

### What's domain-specific (the content layer)

| Component | Engineering | Sales |
|---|---|---|
| Spec types | feature, bug, convention, decision | deal, campaign, objection, playbook |
| Agents | feature-delivery-lead, debug-investigator, engineer | deal-strategist, pipeline-analyst, proposal-writer |
| Commands | /design, /deliver, /diagnose, /scrub | /qualify, /forecast, /propose, /win |
| Skills | go-stack, testing-and-validation | discovery-call, competitive-analysis |
| Integrations | GitHub, Jira, Linear | Salesforce, HubSpot, Gong, LinkedIn |
| AGENTS.md | Engineering routing table | Sales routing table |
| Scans | Code scan (languages, frameworks) | CRM scan (pipeline, stages, fields) |

### Integration interface

Each domain defines its integrations via a provider interface, same
pattern as the tracker integration:

```go
type DomainIntegration interface {
    Name() string
    Import(filter ImportFilter) ([]*spec.Spec, error)
    Sync(spec *spec.Spec) error
    Events() <-chan Event  // for automations
}
```

Engineering has GitHub/Jira/Linear. Sales has Salesforce/HubSpot. Each
is a Go package under `internal/integrations/<name>/`.

### UI customization

The dashboard (`hero serve`) adapts to the domain:
- Engineering: specs kanban, drift reports, CI status, velocity
- Sales: pipeline view, forecast chart, deal stage board, win/loss analysis

The dashboard reads the domain from `hero.json` and renders
domain-appropriate pages. Shared infrastructure (jobs, automations, team)
stays the same.

## Children

| Slug | Title | Priority |
|---|---|---|
| domain-plugin-architecture | Refactor content into domain packs | P0 |
| hero-sales | Sales domain — agents, skills, commands, Salesforce integration | P0 |

## Sequencing

1. **domain-plugin-architecture** — refactor the existing engineering
   content into `domains/engineering/`, add domain selection to `hero init`
   and `hero.json`, update `hero install` to copy from the active domain.
   Small refactor, no new features.
2. **hero-sales** — write the sales agents, commands, skills, and
   Salesforce integration. This is the content work that needs domain
   expertise (the CRO brother).
