---
title: Hero Sales — AI-Powered Sales Workflow for Revenue Teams
slug: hero-sales
type: feature
status: planning
priority: P0
tags: [sales, domain, crm, salesforce, pipeline, forecast]
created: 2026-04-25
relations:
  - target: hero-domains
    kind: parent
  - target: domain-plugin-architecture
    kind: depends-on
horizon: someday
smoke: deferred
---

## Goal

Build a sales domain pack that gives revenue teams the same structured,
AI-powered workflow Hero gives engineering teams. Design before you pitch,
diagnose before you lose. Every deal starts as a spec, gets strategized,
then gets executed with full context.

## Problem

Sales teams drown in CRM busywork, lose institutional knowledge when reps
leave, repeat discovery mistakes across similar deals, and forecast by
gut feel instead of pattern analysis. AI tools exist for email writing
and call transcription, but nothing provides the structured workflow:
plan your approach → execute with context → capture what you learn →
every future deal benefits.

A CRO needs visibility across the pipeline, pattern detection across
deal histories, and automated pipeline hygiene — the same things Hero
gives engineering leads, translated to revenue.

## Design

### The sales core loop

```
/discover  →  /qualify  →  /strategize  →  /execute
                              ↗
              /diagnose  →
```

1. **Discover** prospects and opportunities — research companies, find pain points
2. **Qualify** a deal — score fit, identify stakeholders, assess timeline
3. **Strategize** a deal — produce a deal plan (the "spec") with approach, stakeholders, objections, and win criteria
4. **Diagnose** a stalled or lost deal — investigate why, find the real blocker, produce a recovery plan
5. **Execute** against a deal plan — run the plays, track progress, close

### Spec types (deal types)

| Type | Equivalent to | Purpose |
|---|---|---|
| `deal` | feature | An active opportunity being worked |
| `prospect` | bug (incoming) | A lead or prospect to research/qualify |
| `playbook` | convention | Repeatable sales motions (e.g., "enterprise land", "competitive displacement") |
| `battlecard` | decision | Competitive positioning against a specific competitor |
| `campaign` | initiative | Multi-deal coordinated effort (territory push, product launch) |
| `retro` | note | Win/loss analysis — what worked, what didn't |

### Agents (15 sales specialists)

| Agent | Role |
|---|---|
| `deal-strategist` | The delivery lead equivalent — coordinates specialists, produces deal plans |
| `pipeline-analyst` | Analyzes pipeline health, flags at-risk deals, forecasts revenue |
| `discovery-specialist` | Runs pre-call research — company, contacts, pain points, tech stack |
| `qualification-analyst` | Scores deals on MEDDPICC/BANT/custom frameworks |
| `competitive-analyst` | Researches competitors, maintains battlecards, suggests positioning |
| `proposal-writer` | Drafts proposals, SOWs, and business cases from deal specs |
| `objection-handler` | Surfaces relevant objection handling from playbooks and past deals |
| `executive-briefer` | Prepares exec sponsor briefings and board-level summaries |
| `account-researcher` | Deep-dives a company — financials, org chart, news, tech stack |
| `forecast-analyst` | Weighted pipeline forecast, commit vs. best-case vs. upside |
| `territory-planner` | Account prioritization, whitespace analysis, territory mapping |
| `onboarding-coach` | Ramp new reps by surfacing relevant playbooks, past deals, and conventions |
| `deal-reviewer` | Review a deal strategy for gaps — the architecture-reviewer of sales |
| `win-loss-analyst` | Post-close analysis — patterns across wins and losses |
| `crm-hygienist` | Pipeline cleanup — stale deals, missing fields, duplicate contacts |

### Commands (12 sales commands)

| Command | What it does |
|---|---|
| `/discover` | Research a company or market segment — produce a prospect brief |
| `/qualify` | Score and qualify a deal using the configured framework |
| `/strategize` | Produce a deal plan — approach, stakeholders, objections, timeline |
| `/diagnose` | Investigate a stalled deal — find the real blocker |
| `/execute` | Work a deal plan — track progress, next actions |
| `/propose` | Draft a proposal or SOW from a deal spec |
| `/forecast` | Pipeline forecast — weighted, by stage, by rep |
| `/review` | Get a second opinion on a deal strategy |
| `/retro` | Win/loss analysis on a closed deal |
| `/battlecard` | Create or update competitive positioning |
| `/campaign` | Plan a multi-deal campaign (territory push, product launch) |
| `/coach` | Surface relevant playbooks and past deals for a situation |

### Skills (20+ sales skills)

**Frameworks:** `meddpicc`, `bant`, `challenger-sale`, `sandler`, `value-selling`, `gap-selling`

**Workflows:** `discovery-call-prep`, `executive-alignment`, `multi-threading`, `negotiation`, `procurement-navigation`, `legal-review-prep`

**Analysis:** `pipeline-hygiene`, `forecast-methodology`, `territory-mapping`, `account-planning`, `competitive-positioning`, `win-loss-patterns`

**Content:** `proposal-structure`, `business-case`, `roi-calculator`, `customer-story-matching`

### Knowledge base (sales-specific)

| Knowledge type | What it holds |
|---|---|
| `playbooks/` | Repeatable sales motions — "How we sell to enterprise healthcare" |
| `battlecards/` | Competitive positioning — one per competitor |
| `objections/` | Common objections with proven responses |
| `personas/` | Buyer personas — CTO, CFO, VP Eng, etc. |
| `deal-history/` | Completed deal specs — the institutional memory |
| `context/` | Market context, pricing, product capabilities |

### Integrations

#### Salesforce (primary)

```go
type SalesforceIntegration struct {
    // OAuth2 connection to Salesforce REST API
}

func (s *Salesforce) Import(filter ImportFilter) ([]*spec.Spec, error)
// Imports opportunities as deal specs
// Maps: Opportunity → deal, Lead → prospect
// Pulls: stage, amount, close date, contacts, activities

func (s *Salesforce) Sync(spec *spec.Spec) error
// Updates opportunity fields, posts chatter updates
// Syncs: stage changes, next steps, deal notes

func (s *Salesforce) Events() <-chan Event
// Polls for: new opportunities, stage changes, close date changes
// Triggers automations: auto-qualify, auto-research
```

#### HubSpot (secondary)

Same interface, HubSpot Deals API.

#### Gong/Chorus (call intelligence)

Read-only integration — pull call transcripts and key moments as raw
knowledge for ingest. Agent summarizes calls and updates deal specs.

#### LinkedIn Sales Navigator (enrichment)

Read-only — enrich prospect/contact data with LinkedIn profiles,
company info, mutual connections.

### Dashboard (sales-specific pages)

| Page | What it shows |
|---|---|
| Pipeline | Deal board by stage (kanban), weighted forecast, stage conversion rates |
| Forecast | Commit vs. best-case vs. upside, by rep and by team |
| Deals | Individual deal status, strategy health, next actions due |
| Battlecards | Competitive landscape, win rates by competitor |
| Coaching | Rep performance, deal velocity, playbook adoption |
| Activity | Team activity feed — calls, meetings, proposals, deal updates |

### Automations (sales-specific)

```yaml
# .hero/automations/auto-qualify-new-opps.yaml
name: Auto-qualify new opportunities
trigger:
  type: crm
  event: opportunity_created
  filter:
    stage: Prospecting
action:
  command: qualify
  args: "{{opportunity_id}}"
  mode: autopilot
  budget: 1.00
```

```yaml
# .hero/automations/stale-deal-alert.yaml
name: Flag stale deals
trigger:
  type: schedule
  cron: "0 8 * * 1"  # Monday 8am
action:
  command: diagnose
  args: "--stale-days 14"
```

```yaml
# .hero/automations/close-date-pushed.yaml
name: Diagnose pushed close dates
trigger:
  type: crm
  event: opportunity_updated
  filter:
    field_changed: close_date
    direction: pushed
action:
  command: diagnose
  args: "{{opportunity_id}} close date pushed — investigate"
  budget: 1.00
approval:
  required: false
```

### hero.json for sales

```json
{
  "domain": "sales",
  "folder": ".hero",
  "crm": {
    "type": "salesforce",
    "instance": "your-org.my.salesforce.com",
    "token_env": "SALESFORCE_TOKEN",
    "auto_sync": true
  },
  "qualification": {
    "framework": "meddpicc",
    "min_score": 60
  },
  "forecast": {
    "methodology": "weighted",
    "stages": {
      "Prospecting": 10,
      "Discovery": 20,
      "Evaluation": 40,
      "Proposal": 60,
      "Negotiation": 80,
      "Closed Won": 100,
      "Closed Lost": 0
    }
  },
  "pipeline": {
    "stale_days": 14,
    "hygiene_schedule": "weekly"
  }
}
```

### CLI examples

```bash
# Pipeline overview
hero status                              # pipeline by stage
hero pulse --week                        # weekly pipeline narrative
hero forecast                            # weighted forecast

# Work a deal
hero search "Acme Corp"                  # find the deal
/strategize Acme Corp enterprise deal    # produce a deal plan
/execute .hero/planning/deals/acme-corp/spec.md  # run the plays
/diagnose acme-corp                      # deal stalled — investigate

# Knowledge
hero ingest call-transcript-acme.md "Acme discovery call"
/battlecard vs. Competitor X
/retro .hero/specs/deals/lost-bigco/spec.md  # why we lost

# Automation
hero run qualify --all --type prospect   # bulk qualify new leads
hero run diagnose --all --match "stale"  # investigate stale deals
hero suggestions                         # deals needing attention

# Research
/discover "Series B fintech companies in payments"
hero impact "pricing change"             # which deals are affected?
```

## Changes

- `domains/sales/agents/` — 15 agent definitions (markdown)
- `domains/sales/commands/` — 12 command definitions (markdown)
- `domains/sales/skills/` — 20+ skill definitions (markdown)
- `domains/sales/AGENTS.md` — sales routing table
- `internal/integrations/salesforce/` — Salesforce REST API integration
- `internal/integrations/hubspot/` — HubSpot API integration (stretch)
- `internal/serve/dashboard_sales.go` — sales-specific dashboard pages
- `internal/config/config.go` — CRM config, qualification config, forecast config
- `internal/forecast/` — weighted pipeline forecast engine

## Acceptance Criteria

- WHEN `hero init --domain sales` runs THE SYSTEM SHALL create a workspace with sales agents, commands, skills, and a sales AGENTS.md routing table
- WHEN `/qualify <deal>` runs THE SYSTEM SHALL score the deal using the configured framework (MEDDPICC by default) and write findings to the deal spec
- WHEN `/strategize <deal>` runs THE SYSTEM SHALL produce a deal plan with approach, stakeholder map, objection handling, and win criteria
- WHEN `/forecast` runs THE SYSTEM SHALL produce a weighted pipeline forecast grouped by stage, rep, and time period
- WHEN a Salesforce integration is configured THE SYSTEM SHALL import opportunities as deal specs and sync status bidirectionally
- WHEN `hero run qualify --all --type prospect` runs THE SYSTEM SHALL bulk-qualify all unqualified prospects using the configured framework
- WHEN the pipeline dashboard loads THE SYSTEM SHALL display a kanban board of deals by stage with forecast totals
- WHEN a deal is won or lost THE SYSTEM SHALL prompt for a retro and capture learnings in the knowledge base
- THE SYSTEM SHALL reuse the core Hero engine (specs, knowledge, runner, automations, MCP) without modification

## Boundaries

- Does **not** replace the CRM — Hero Sales is the AI strategy layer on top of it
- Does **not** store sensitive customer data in specs — references CRM records by ID
- Does **not** send emails or make calls — surfaces what to say and do, the rep acts
- Does **not** require Salesforce — works standalone with manual deal spec creation
- Does **not** modify the core Hero engine — sales is a domain pack, not a fork
