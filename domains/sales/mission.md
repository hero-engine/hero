---
title: Hero Sales — Sales Vertical Charter
type: mission
scope: vertical
vertical: sales
inherits: ../../.hero/mission.md
status: scaffold
locked_at: 2026-04-28
locked_by: chet-bellows
version: 0
note: |
  This is a scaffold mission, written alongside Hero Code's vertical
  charter to triangulate what belongs in core vs. what belongs in a
  vertical. Real Hero Sales content (agents, skills, commands) will be
  built out when the vertical is prioritized. This file exists today
  primarily as a guidance artifact for the core/vertical split.
---

## Mission

**Hero Sales is the sales vertical of Hero — the sidekick brain for
AI-driven sales workflows.**

It rides on [Core Hero](../../.hero/mission.md) and adds the spec
types, agents, skills, commands, and vocabulary that turn the core
engine into a complete deal-tracking + account-management +
discovery + qualification + close-and-debrief toolkit for sales
teams.

The core mission applies unchanged: the model in the rep's harness
starts cold; Hero captures everything that happens during the work
and injects it back automatically; sessions start smart, end smarter,
the floor rises for everyone — junior BDR and senior AE alike.

What this vertical adds: the **shape** of the work, in sales terms.
*Discover before you pitch*. *Qualify before you propose*.
*Decisions live with the deal, not in someone's head*. *Objection
patterns travel across reps so the team learns together*. *Win/loss
debriefs become institutional memory, not Slack threads that vanish*.

## What this vertical brings (planned)

| Layer | What's in it (planned) |
|---|---|
| **Spec types** | account, opportunity, deal, prospect, demo, proposal, objection-history, deal-debrief |
| **Agents** | sdr/bdr, account-executive, sales-engineer, customer-success, revenue-ops, deal-coach, objection-strategist, demo-architect |
| **Skills** | discovery patterns, qualification frameworks (BANT, MEDDIC, MEDDPICC), demo design, proposal writing, negotiation tactics, follow-up cadences, win/loss analysis, CRM patterns |
| **Commands** | /prospect, /qualify, /discover, /demo-prep, /propose, /negotiate, /close, /debrief, /forecast |
| **Lifecycle** | discover → qualify → demo → propose → negotiate → close → debrief |
| **Trackers** | Salesforce, HubSpot, Pipedrive — read/write integration |
| **Surfaces** | Slack briefings, email-prep summaries, CRM enrichment |

## How it specializes the core

The core mission test asks: *"Does this make the next agent session
start smarter than the last one ended?"* For Hero Sales, "next agent
session" means *the next sales session — picking up an account,
prepping a discovery call, writing a follow-up, planning a renewal,
debriefing a loss.* Every Hero Sales feature must answer: does a
sales session start with the right context loaded — current deal
state, recent customer conversations, what objections have surfaced
in similar deals, who else on the team has touched this account,
what playbooks are converting?

## Vocabulary additions (sales-specific)

These extend (never override) the core vocabulary.

- **deal** — the unit of sales work; a deal-spec is a sales-domain spec
- **account** — the customer entity; multiple deals belong to one account
- **discovery** — the qualification phase; analogous to engineering's
  *diagnose*
- **objection** — a recurring resistance pattern; tracked across deals
- **playbook** — codified motion that converted; analogous to
  engineering's *convention*
- **debrief** — post-deal retro; analogous to engineering's *retro*

## Anti-patterns specific to this vertical

In addition to core anti-patterns, Hero Sales must never become:

- **A CRM replacement.** Salesforce/HubSpot are the system of record.
  Hero rides alongside, capturing context the CRM doesn't (call notes,
  internal reasoning, what worked, what didn't).
- **A pipeline-management tool.** Forecasting and pipeline health live
  in the CRM and in revenue-ops tools. Hero's job is making each
  individual deal's session start smart.
- **A rep-replacement.** The rep runs the call. Hero feeds them the
  right context. (Direct mirror of Hero Code's "we don't write code.")

## Why this scaffold exists today

Two reasons:

1. **Triangulation.** With only one vertical (engineering), it's hard
   to tell what's *engineering-specific* from what's *core but happens
   to look engineering-shaped because it's the only example we have*.
   A second vertical mission, even as a scaffold, lets us look at any
   existing Hero artifact and ask: *"Does this serve sales too? Then
   it's core."*
2. **Strategic signal.** Hero Sales is the next vertical the user
   plans to tackle. Locking the manifesto at scaffold-time makes the
   eventual build easier — less drift between Hero Code's shape and
   Hero Sales's shape.

## Inheritance discipline

When [`.hero/mission.md`](../../.hero/mission.md) (the core charter)
changes, this vertical charter is reviewed for compatibility within
the same PR. Vertical charter cannot weaken or contradict the core;
it can only specialize.
