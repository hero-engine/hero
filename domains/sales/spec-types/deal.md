---
title: Deal
type: deal
domain: sales
category: work
bucket: deals
location: .hero/planning/deals/{slug}/spec.md
lifecycle:
  states: [prospect, qualifying, demo, proposal, negotiation, won, lost]
  initial: prospect
  terminal: [won, lost]
  transitions:
    - { from: prospect, to: qualifying, gate: "Initial response from a contact; at least one substantive conversation; a solvable problem confirmed" }
    - { from: qualifying, to: demo, gate: "MEDDPICC >= 40; pain confirmed/quantified; Economic Buyer identified by name and title; timeline established" }
    - { from: qualifying, to: lost, gate: "Qualify out — MEDDPICC < 25 after 3 qualification conversations" }
    - { from: demo, to: proposal, gate: "Demo/POC completed with positive feedback; EB engaged; decision criteria and process documented; MEDDPICC >= 55" }
    - { from: demo, to: lost, gate: "Evaluation failed or deal disqualified" }
    - { from: proposal, to: negotiation, gate: "Proposal submitted and acknowledged; budget confirmed; verbal intent from EB or champion; legal/procurement started" }
    - { from: proposal, to: lost, gate: "Proposal rejected or deal disqualified" }
    - { from: negotiation, to: won, gate: "Signed agreement / MSA / Order Form; PO received if required; payment terms agreed" }
    - { from: negotiation, to: lost, gate: "Deal closed lost" }
sections:
  optional: [Qualification, Deal Strategy, Research, Competitive Situation, Debrief]
accepting_commands: [/qualify, /strategize, /forecast, /pipeline, /research, /debrief]
default_agents:
  authoring: deal-strategist
  qualification: qualification-analyst
  forecast: forecast-analyst
frontmatter:
  required:
    - { name: title, type: string, required: true, classification: content, description: "Human-readable deal name (e.g. \"Acme Corp — Enterprise Platform Deal\")." }
    - { name: type, type: enum, required: true, values: [deal], default: deal, classification: content, description: "Spec type discriminator; always 'deal'." }
    - { name: status, type: enum, required: true, values: [prospect, qualifying, demo, proposal, negotiation, won, lost], default: prospect, classification: org-state, description: "Current deal stage (lifecycle position); maps to CRM stage." }
    - { name: company, type: string, required: true, classification: content, description: "Company name (matches CRM account name)." }
  optional:
    - { name: owner, type: string, classification: org-state, description: "Rep email or username who owns this deal." }
    - { name: arr, type: integer, classification: content, description: "Annual Recurring Revenue of this deal in USD (whole number, e.g. 120000)." }
    - { name: close_date, type: date, format: "YYYY-MM-DD", classification: org-state, description: "Targeted close date (synced from CRM)." }
    - { name: stage, type: string, classification: org-state, description: "CRM stage label (display only; status is the canonical field)." }
    - { name: meddpicc_score, type: integer, classification: content, description: "MEDDPICC qualification score 0-100 (0-39 high risk, 40-59 moderate, 60-79 solid, 80-100 well-qualified). Computed by /qualify." }
    - { name: probability, type: integer, classification: content, description: "Win probability 0-100 for the weighted forecast. Stage defaults: prospect 10, qualifying 20, demo 40, proposal 60, negotiation 80, won 100, lost 0." }
    - { name: qualification_framework, type: string, default: meddpicc, classification: content, description: "Framework used to qualify this deal (meddpicc, bant, spin, custom)." }
    - { name: crm_id, type: string, classification: org-state, description: "CRM opportunity ID (e.g. Salesforce Opportunity ID)." }
    - { name: crm_type, type: enum, values: [salesforce, hubspot, pipedrive, manual], classification: org-state, description: "Which CRM this deal lives in." }
    - { name: priority, type: enum, values: [P0, P1, P2, P3], classification: org-state, description: "Deal priority (P0 = must close this quarter, P3 = low urgency)." }
    - { name: tags, type: "list[string]", classification: content, description: "Free-form tags (e.g. [enterprise, competitive, expansion])." }
    - { name: relations, type: "list[relation]", classification: content, description: "Links to related specs (e.g. parent campaign, informing playbook)." }
---

# Deal spec-type

A **deal** is THE unit of work in the Hero Sales domain — an active sales
opportunity being worked toward close. It is the sales analog of engineering's
`feature`: every opportunity is tracked as a spec, and the same graph,
search, and lifecycle machinery applies.

`deal` is registered as a `category: work` type on sales installs
(`hero spec new`, `hero list --type deal`, `hero search --type deal` all
work). It does not collide with any core type, so the sales domain overlay
loads it cleanly on top of the core spec types.

## When to use

- An active opportunity with an identifiable company and a path to close.
- A prospect to research and qualify — a prospect is a `deal` at
  `status: prospect` (the lifecycle's initial state), not a separate type.

## When NOT to use

- Repeatable sales motions, competitive battlecards, buyer personas, and
  win/loss retros are **knowledge**, not deals. They live under
  `.hero/knowledge/{playbooks,battlecards,prospects,personas}/` as plain
  markdown with descriptive titles ("Playbook: …", "Battlecard — Hero vs. …")
  and carry no work-ish `type:` frontmatter, so they stay out of work-spec
  discovery (`hero list`). They are retrieved from the knowledge corpus —
  `hero search --knowledge` / `hero ask` (or by browsing the directory) — not
  the default `hero search`, which covers work specs only.

## Lifecycle

States: `prospect → qualifying → demo → proposal → negotiation → won`
(terminal), with `lost` reachable as the other terminal state. Exit-criteria
gates are owned by the `pipeline-management` skill; the transitions here mirror
its stage table:

- `prospect → qualifying` — initial contact response; ≥1 substantive
  conversation; a solvable problem confirmed.
- `qualifying → demo` — MEDDPICC ≥ 40; pain confirmed/quantified; Economic
  Buyer identified by name and title; timeline established.
- `qualifying → lost` — qualify out: MEDDPICC < 25 after 3 qualification
  conversations.
- `demo → proposal` — demo/POC completed with positive feedback; EB engaged;
  decision criteria and process documented; MEDDPICC ≥ 55.
- `proposal → negotiation` — proposal submitted and acknowledged; budget
  confirmed; verbal intent from EB or champion; legal/procurement started.
- `negotiation → won` — signed agreement / MSA / Order Form; PO received if
  required; payment terms agreed.
- `demo/proposal/negotiation → lost` — evaluation failed, proposal rejected,
  or the deal is otherwise closed lost.

Stage probability defaults (weighted forecast): `prospect 10, qualifying 20,
demo 40, proposal 60, negotiation 80, won 100, lost 0`. Per-deal `probability`
overrides the stage default when set.

## Sections

Optional: `Qualification`, `Deal Strategy`, `Research`, `Competitive
Situation`, `Debrief`. Deals accrue these as they progress — a fresh prospect
may have none; a deal in negotiation typically carries Qualification and Deal
Strategy.

## Accepting Commands

- `/qualify` — score the deal against its qualification framework.
- `/strategize` — build or update the deal strategy.
- `/forecast` — roll the deal into the weighted forecast.
- `/pipeline` — hygiene and stage review.
- `/research` — buyer / account / competitive research.
- `/debrief` — win/loss retro on a closed deal.

## Default Agents

- authoring: `deal-strategist`
- qualification: `qualification-analyst`
- forecast: `forecast-analyst`

## Example Frontmatter

```yaml
---
title: Acme Corp — Enterprise Platform Deal
type: deal
status: qualifying
company: Acme Corp
owner: jane.smith@company.com
arr: 120000
close_date: 2026-09-30
stage: Qualifying
meddpicc_score: 42
probability: 25
qualification_framework: meddpicc
crm_id: 006Dn000003abcDEFG
crm_type: salesforce
priority: P1
tags: [enterprise, competitive, new-logo]
relations:
  - target: q3-enterprise-campaign
    kind: part-of
---
```
