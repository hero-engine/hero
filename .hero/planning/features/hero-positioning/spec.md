---
title: Hero Positioning — Narrative, ICP, Messaging, Comparison
type: feature
status: planning
priority: P0
tags: [marketing, positioning, messaging, narrative]
created: 2026-04-25
relations:
  - target: hero-marketing
    kind: parent
horizon: someday
smoke: deferred
---

## Goal

Lock the narrative every other marketing surface inherits — the
one-liner, the elevator pitch, the ICP, the alternatives we compare
against, the words we use and the words we avoid. Produce a positioning
doc that the landing page, docs, launch posts, and sales conversations
all draw from.

## Problem

Without a single source of positioning truth, every surface drifts.
The README calls Hero a "spec-driven AI engineering workflow." NEXT.md
calls it a "platform." Internal notes call it a "context broker."
A new visitor reading three of those gets a fuzzy answer to "what is
this?" — and a fuzzy answer kills installs.

Positioning isn't a paragraph; it's a small set of explicit decisions
that other artifacts can quote verbatim.

## Deliverables

### 1. Positioning doc (`.hero/marketing/positioning.md`)

A single markdown file with the following sections, each ≤ 200 words:

- **One-liner** — the 12-word version
- **Elevator pitch** — 60 seconds, ~150 words
- **What it is / what it isn't** — two parallel lists
- **Who it's for** — primary ICP, secondary ICP, anti-personas
- **Why now** — the trend that makes this inevitable
- **Why us** — credibility, not features
- **Top 3 jobs to be done** — what people hire Hero to do
- **Top 5 objections + responses** — what trial users push back on
- **Vocabulary** — preferred terms, banned terms
- **Comparison matrix** — Hero vs Cursor, Aider, Continue, raw CLAUDE.md

### 2. Tagline candidates (3–5)

Short candidates with rationale. Examples to test:
- "The spec layer for AI coding tools."
- "Design before you build. Diagnose before you fix."
- "Compounding context for AI-native engineering teams."
- "The workflow your AI coding tools are missing."
- "Specs, knowledge, and conventions — every session inherits."

### 3. Comparison page content

Long-form copy for a `/vs` section on the landing page or docs:
- vs Cursor / Claude Code / Copilot (we're additive, not competitive)
- vs Aider / Continue / Cline (we're orchestration, they're execution)
- vs raw CLAUDE.md / cursorrules (we're a corpus + workflow, not a file)
- vs internal wikis (we're machine-readable + agent-consumable)

### 4. Messaging house

A one-page table mapping audience → pain → message → proof:

| Audience | Pain | Message | Proof |
|---|---|---|---|
| Eng lead | Sessions lose context | Knowledge compounds | Auto-capture + 35 MCP tools |
| Solo dev | Cursor improvises bad code | Spec-driven workflow | /design before /deliver |
| CTO | Can't see across team | Team feed + dashboard | hero feed + serve --team |

### 5. Boilerplate

Reusable copy blocks for everywhere Hero gets described:

- 25-word boilerplate (X/Twitter bio, GitHub repo description)
- 50-word boilerplate (press kit, podcast intros)
- 150-word boilerplate (about page, partnership emails)

## Process

1. Mine the existing surfaces — README, NEXT.md, AGENTS.md, GETTING-STARTED.md,
   internal notes — for every framing already in use. List them all.
2. Cluster them into themes. Pick winners. Kill the rest.
3. Write candidate one-liners and elevator pitches. Test them
   informally — read aloud, send to 3–5 trusted developers, see what
   sticks.
4. Lock vocabulary: pick the words we use ("spec", "workspace", "domain")
   and the ones we avoid ("AI assistant", "copilot", "agentic platform").
5. Write the comparison page from real product knowledge — not vibes.
   Each row should cite a concrete capability or limitation.
6. Run the doc past brother (CRO) for non-engineering readability.

## Acceptance criteria

- Single `positioning.md` file exists and is the source of truth
- One-liner is locked and used verbatim on landing page + GitHub repo
  description + README hero section
- Three audience messages are mapped to three pieces of proof each
- Comparison page treats competitors fairly (no straw-man framings)
- Banned-term list exists and we don't ship copy that violates it

## Out of scope

- Visual brand (logo, typography, color) — separate effort
- Pricing — defer until team-platform is shipped
- Sales collateral (one-pagers, decks) — defer until we have a sales motion

## Open questions

- Do we lead with "spec-driven" (descriptive but jargon-y) or "the
  workflow your AI coding tools are missing" (clearer but vaguer)?
- How much do we lean on "compounding context" — it's true and unique
  but might be too abstract for a first-impression line.
- Do we mention domains (sales, etc.) on the landing page or wait until
  Hero Sales is real and demoable?
