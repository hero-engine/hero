---
title: Hero Cloud — Team Visibility and Cross-Repo Knowledge Platform
type: initiative
status: in-progress
tags: [cloud, saas, platform]
created: 2026-04-12
child:
  - cloud-sync
  - cloud-dashboard
  - cloud-mcp
  - cloud-auth
  - cloud-api
  - cloud-billing
  - cloud-notifications
  - cloud-admin
horizon: next
---

## Goal

Build the Hero Cloud platform — a SaaS layer that adds multi-user visibility,
cross-repo knowledge sharing, and team coordination on top of the free open-source
Hero CLI. The CLI remains the authoring tool; the cloud is the coordination layer.

## Business Model

| Tier | Price | What You Get |
|---|---|---|
| **Open Source CLI** | Free forever | Full spec authoring, local MCP, local indexing, all agent commands |
| **Hero Cloud (Team)** | $15-30/seat/month | Cloud dashboard, cross-repo search, team activity feed, spec sync, cloud MCP, async agents |
| **Enterprise** | $50-100/seat/month | SSO/SAML, audit log, compliance controls, SLA, dedicated support, self-hosted option, custom agent marketplace |

### Key Principle

The local MCP server is free forever. Cloud MCP (cross-repo, cross-team knowledge
federation) is the paid differentiator. The CLI never phones home, never requires
a login, never degrades without cloud.

## Capabilities

The cloud layer delivers five user-visible capabilities on top of the local CLI.
Each maps to one or more of the delivery phases below.

### 1. Async Agent Execution

Developer approves a spec locally, pushes it to the cloud, and a sandboxed cloud
agent picks up the work and opens a PR. Progress is visible via the dashboard or
`hero status --remote`. Supports the "kick off 5 specs and go to lunch" workflow
and the batch diagnose/deliver pipelines that are impractical on a developer's
laptop.

### 2. Cross-Repo Knowledge Store (via Cloud MCP)

Conventions, ADRs, and code patterns searchable across every repo in an org.
- "How did team X solve caching?"
- "3 repos use auth pattern A, 1 repo diverges"
- When scanning a new project, leverage patterns learned from similar projects
  ("this looks like a Django project — here are the conventions that worked for
  12 other Django projects in your org")
- Shared error-pattern catalog across projects

### 3. Spec Analytics

Metrics grounded in spec completion, not story points.
- Spec quality scores over time
- Delivery success rate by spec type, author, complexity
- Time from spec to merged PR
- Rework rate and reasons
- Team velocity as a function of spec throughput

### 4. Team Dashboard and Activity Feed

Web UI showing cross-repo spec status, in-flight work, who's claimed what,
knowledge search, and recent activity.

### 5. Agent Marketplace (future)

Custom agents (e.g. `compliance-reviewer`, `security-auditor`) shareable across
teams, plus industry-specific agent packs (fintech compliance, healthcare HIPAA,
etc.). Enterprise-tier feature.

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                   Hero Cloud                         │
│                                                     │
│  ┌─────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │ Cloud   │  │ Cloud    │  │ Cloud MCP         │  │
│  │ API     │  │ Dashboard│  │ (cross-repo       │  │
│  │ (REST)  │  │ (Web UI) │  │  knowledge)       │  │
│  └────┬────┘  └────┬─────┘  └────────┬──────────┘  │
│       │            │                  │             │
│  ┌────┴────────────┴──────────────────┴──────────┐  │
│  │              Cloud Data Layer                  │  │
│  │  (Postgres, per-org isolation, event stream)   │  │
│  └────────────────────┬──────────────────────────┘  │
│                       │                             │
└───────────────────────┼─────────────────────────────┘
                        │ HTTPS (push/pull sync)
        ┌───────────────┼───────────────┐
        │               │               │
   ┌────┴────┐    ┌─────┴────┐   ┌──────┴─────┐
   │ hero CLI│    │ hero CLI │   │ hero CLI   │
   │ (repo A)│    │ (repo B) │   │ (repo C)   │
   │ local   │    │ local    │   │ local      │
   │ MCP     │    │ MCP      │   │ MCP        │
   └─────────┘    └──────────┘   └────────────┘
```

### Architectural Principles

- **Local-first** — everything works offline; the cloud enhances, never gates.
- **Git as source of truth** — specs and knowledge live in the repo. The cloud
  syncs and indexes; it doesn't own.
- **CLI-driven sync** — the CLI pushes and pulls; the cloud never modifies local
  state without explicit sync.
- **Per-org isolation from day one** — specs may contain sensitive architectural
  detail; tenant boundaries are non-negotiable.
- **Self-hostable** — enterprises can run the full stack in their own cloud.

## Delivery Phases

### ✅ Phase 1: Foundation (cloud-auth, cloud-api) — completed

Cloud service stood up with authentication, org/team management, and a REST API
the CLI can push/pull specs to. Specs in `.hero/specs/cloud-auth` and
`.hero/specs/cloud-api`.

### ✅ Phase 2: Sync (cloud-sync) — completed

`hero sync --cloud` pushes spec metadata to the cloud and pulls aggregated views
back down. Spec in `.hero/specs/cloud-sync`.

### Phase 3: Dashboard (cloud-dashboard, cloud-dashboard-ui)

Web UI showing cross-repo spec status, team velocity, in-flight work, and
knowledge search. Delivers Capability 4.

### Phase 4: Cloud MCP (cloud-mcp)

The flagship paid feature — an MCP server that federates knowledge across repos
and teams. An AI tool connected to Cloud MCP can search specs, conventions, and
decisions from any repo the user has access to. Delivers Capability 2.

### Phase 5: Async Agents (async-agents — spec TBD)

Cloud workers that execute approved specs in sandboxed environments and open
PRs. Integrates with the batch diagnose/deliver pipelines. Delivers Capability 1.

### Phase 6: Spec Analytics (spec-analytics — spec TBD)

Metrics surface on the dashboard: quality scores, rework rates, time-to-merge,
velocity. Delivers Capability 3.

### Phase 7: Monetization (cloud-billing, cloud-admin, cloud-notifications)

Stripe integration, seat management, org admin controls, activity notifications.

### Phase 8: Enterprise (cloud-enterprise — spec TBD)

SSO/SAML, audit logging, compliance controls, SLA guarantees, self-hosted
deployment support.

### Phase 9: Agent Marketplace (agent-marketplace — spec TBD)

Shareable custom agents and industry-specific agent packs. Enterprise-tier.
Delivers Capability 5.

## Non-Goals

- The CLI will never require a cloud account
- The CLI will never phone home or degrade without cloud
- We will not build a web-based spec editor — the CLI and AI tools are the editors
- No real-time collaboration (specs are git-committed, not live docs)
- The cloud never mutates local spec state without an explicit CLI sync

## Risks

- **Adoption chicken-and-egg**: Cloud is only valuable with enough CLI users. Launch cloud after CLI has traction.
- **Scope creep into PM tool**: Stay focused on spec-driven workflow, not project management.
- **Security**: Specs may contain sensitive architectural details. Per-org data isolation is critical from day one.
- **Async agent cost**: Cloud execution costs money per spec. Pricing must cover sandbox compute without pricing out teams.
