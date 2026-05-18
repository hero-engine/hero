---
title: Hero Team Experience — Complete Multi-Developer Workflow
slug: hero-team-experience
type: initiative
status: planning
tags: [team, coordination, server, dashboard, workflow]
created: 2026-04-25
relations:
  - target: hero-platform
    kind: parent
  - target: hero-team-server
    kind: depends-on
horizon: next
---

## Goal

Define and deliver the complete experience for a team of developers each
running Hero locally, connected to a shared team server, working together
on one or more codebases. Every solo feature that benefits from team
awareness gets wired into the team layer. The result: a team that sees
each other's work, avoids stepping on each other, and has automated
guardrails running continuously.

## The Three Layers

```
┌──────────────────────────────────────────────────────────────┐
│                    DEVELOPER MACHINES                         │
│                                                              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐       │
│  │ Alice   │  │  Bob    │  │ Carol   │  │  CI Box │       │
│  │ Cursor  │  │ Claude  │  │OpenCode │  │  hero   │       │
│  │ + hero  │  │  Code   │  │ + hero  │  │  run    │       │
│  │  mcp    │  │ + hero  │  │  mcp    │  │ --auto  │       │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘       │
│       │            │            │             │             │
│  Each machine has:                                          │
│  • .hero/ workspace (specs, knowledge, NEXT.md)             │
│  • Git repo (source of truth for code + specs)              │
│  • hero mcp (serves tools to AI session)                    │
│  • Local analysis (drift, impact, recap, coverage)          │
│  • Solo mode works fully without a server                   │
└───────┼────────────┼────────────┼─────────────┼─────────────┘
        │            │            │             │
        │  push events, jobs, sessions, usage   │
        │  pull status, approvals, team feed    │
        │            │            │             │
┌───────┴────────────┴────────────┴─────────────┴─────────────┐
│                    HERO TEAM SERVER                           │
│              (shared box, VM, or cloud)                       │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Job      │  │ Session  │  │ Auth +   │  │ Notifi-  │   │
│  │ Queue +  │  │ Registry │  │ Creds    │  │ cations  │   │
│  │ Workers  │  │ + Claims │  │ + Usage  │  │ Webhooks │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Auto-    │  │ Scheduled│  │ Team     │  │ HTTP     │   │
│  │ mations  │  │ Tasks    │  │ Feed     │  │ API      │   │
│  │ Engine   │  │ (cron)   │  │ (events) │  │          │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  DASHBOARD (web UI)                    │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │  │
│  │  │Overview │ │ Jobs    │ │  Team   │ │Pipeline │   │  │
│  │  │ & Feed  │ │ & Queue │ │Activity │ │& Health │   │  │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘   │  │
│  └───────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
        │
        │  (optional — same code, hosted)
        ▼
┌──────────────────────────────────────────────────────────────┐
│                    HERO CLOUD (future)                        │
│  Hero Team Server + SSO + billing + multi-org                │
└──────────────────────────────────────────────────────────────┘
```

## Data Flow

```
Developer's session                    Team Server
─────────────────                    ───────────

Alice starts /deliver auth-flow
  │
  ├─► POST /api/sessions              Session registered
  │   {user: alice, spec: auth-flow}   → dashboard shows "Alice: auth-flow"
  │                                    → Bob's CLI warns if he tries same spec
  │
  ├─► POST /api/jobs (if --team)       Job queued
  │   {command: deliver, args: auth}   → worker picks it up
  │   OR runs locally (solo fallback)  → OR Alice runs it herself
  │
  ├─► POST /api/events                Feed event stored
  │   {type: delivery_started}         → team feed shows activity
  │                                    → dashboard live updates via SSE
  │
  │   ... agent works ...
  │
  ├─► POST /api/events                Feed events accumulate
  │   {type: files_modified,           → team sees progress
  │    files: [auth.go, session.go]}
  │
  │   ... delivery complete ...
  │
  ├─► POST /api/events                 Notification fires
  │   {type: delivery_complete}         → Slack: "Alice delivered auth-flow"
  │
  ├─► Server runs drift check          Results on dashboard
  │   (automated post-delivery)         → if drift: Slack alert
  │
  ├─► Server runs coverage check       Results on dashboard
  │   (automated post-delivery)         → gaps flagged
  │
  ├─► Server runs contract check       Results on dashboard
  │   (automated post-delivery)         → regressions flagged
  │
  ├─► Usage recorded                   Per-user tracking
  │   {user: alice, cost: $3.40}        → budget enforcement
  │                                     → usage dashboard
  │
  └─► DELETE /api/sessions             Session unregistered
      (on disconnect)                   → dashboard updates
```

## Solo Features → Team-Aware Mapping

### Coordination (currently local, should be server-enforced)

| Feature | Solo behavior | Team behavior |
|---|---|---|
| `hero claim` | Writes to local events.log | POST to server; server prevents duplicate claims across machines |
| `hero active` | Local session file | Server aggregates all sessions; CLI shows team-wide view |
| `hero feed` | Local JSONL file | Server is the central store; all clients write to it |
| NEXT.md | Single file or per-user | Server aggregates team overview; each dev has their own |

### Analysis (runs locally, results surface on server)

| Feature | Solo behavior | Team behavior |
|---|---|---|
| `hero drift` | CLI output | Workers run post-delivery; results on dashboard + notifications |
| `hero coverage` | CLI output | Workers run post-delivery; gaps on dashboard |
| `hero contract check` | CLI output | Workers run post-delivery; regressions on dashboard + alert |
| `hero suggestions` | CLI output | Server runs weekly; results on dashboard |
| `hero check --reconcile` | Manual | Server runs hourly; status drift never accumulates |
| `hero pulse` | CLI output | Dashboard renders team-wide pulse with all project data |
| `hero recap` | Local git history | Dashboard shows team-aggregated recap |
| `hero impact` | Stays local | Too fast and file-dependent to centralize |

### Execution (local or routed through server)

| Feature | Solo behavior | Team behavior |
|---|---|---|
| `hero run` | Executes locally | Submits to team queue; worker executes; results tracked |
| `hero run --all` | Executes locally | Submits batch to team queue; parallel workers |
| Automations | Local `hero serve` | Server runs automation engine; all triggers centralized |
| Approval gates | N/A | Server parks jobs; notifications; approve/reject via CLI or dashboard |

### Knowledge (git-managed, team-visible)

| Feature | Solo behavior | Team behavior |
|---|---|---|
| `hero ingest` | Writes to local .hero/ | Same — git distributes it; server could offer a web ingest UI |
| `hero ask` | Local index query | Same — index is per-repo; cross-repo search goes through server |
| Knowledge flywheel | Agent-driven | Server runs pattern detection across all captured knowledge |
| Auto-capture | Agent writes to .hero/ | Same — git distributes; server surfaces knowledge stats |

## Children

| Slug | Title | Priority | Status |
|---|---|---|---|
| hero-team-server | Job queue, workers, API, auth | P0 | **Delivered** |
| team-oauth | GitHub/Google SSO | P1 | Planning |
| team-notifications | Webhook alerts for job events | P1 | Planning |
| team-connect | CLI registration + job routing | P1 | Planning |
| team-claims | Server-enforced claims | P1 | New |
| team-feed-central | Centralized event feed | P1 | New |
| team-post-delivery | Auto-run drift/coverage/contract after jobs | P1 | New |
| team-scheduled-tasks | Hourly reconcile, weekly suggestions | P2 | New |
| team-dashboard-wiring | Dashboard consumes all team data | P2 | New |
| hero-dashboard-v2 | Full visual dashboard overhaul | P2 | Planning |

## Sequencing

```
Phase 1 (DONE):    hero-team-server — job queue, workers, API, auth
                        │
Phase 2 (next):    ┌────┼────────────────┐
                   │    │                │
              team-connect  team-oauth  team-notifications
                   │
Phase 3:      ┌────┼────────────┐
              │    │            │
         team-claims  team-feed  team-post-delivery
              │         │              │
Phase 4:      └────┬────┘              │
                   │                   │
         team-scheduled-tasks  team-dashboard-wiring
                        │
Phase 5:        hero-dashboard-v2
```

Phase 2 items are independent — build in parallel.
Phase 3 requires team-connect (needs server registration).
Phase 4 builds on Phase 3 data.
Phase 5 is the visual layer on top of everything.

## The Team Experience (end state)

**Morning:**
Alice opens her IDE. Hero's MCP server starts, registers with the team
server, loads her NEXT.md. She sees: "Bob delivered 3 bug fixes overnight
(automated). Carol is mid-delivery on payment-flow. 2 jobs awaiting your
approval."

**Working:**
She runs `/deliver auth-flow`. Hero checks the team server — nobody else
has claimed it. The claim is registered server-side. Bob's session sees
"Alice is delivering auth-flow" if he runs `hero team status`.

**Automation:**
A new Jira bug comes in. The automation engine diagnoses it headlessly,
produces a spec with root cause and fix plan. Slack gets a notification:
"New diagnosis ready: PROJ-456 — null pointer in session handler."

**Review:**
Alice reviews the diagnosed spec. She `/challenge`s it with "I think it's
actually a race condition." The investigator re-examines. Updated spec
lands. She approves the delivery job. The worker picks it up.

**End of day:**
Hero's dashboard shows: 5 specs delivered today, 2 by humans, 3 by
automation. 0 drift warnings. 1 contract regression flagged (auth-flow
criterion 3 — test needs updating). Team spend: $12.40. Knowledge base
grew by 3 entries.

**What nobody had to do:**
- Check Jira for new bugs (automation did it)
- Run reconcile (server did it hourly)
- Check for test coverage gaps (post-delivery automation did it)
- Ask "what happened overnight?" (recap + dashboard)
- Worry about conflicting claims (server enforced it)
