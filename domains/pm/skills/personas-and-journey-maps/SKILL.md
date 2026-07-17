---
name: personas-and-journey-maps
description: Evidence-based personas (every attribute traceable to research, not invented demographics) and the stage → action → thought/feeling → pain → opportunity journey grid that feeds an OST.
metadata:
  audience: discovery-researcher, product-strategist, story-writer
  purpose: framework-guidance
---

## What I do

Give agents two linked discovery artifacts and the discipline that keeps them honest: **personas** (who we're building for) and **journey maps** (what their experience actually is, end to end). Both fail the same way — by becoming fiction the team invents in a room instead of evidence the team gathers from users. My job is to make every persona attribute and every journey cell traceable to a source, and to route the pains a journey surfaces into the opportunity space of an OST.

## When to use me

- synthesizing interviews, tickets, or analytics into a shareable picture of who the user is (`discovery-researcher`)
- framing a bet where "who is this for and what's their experience today" isn't written down (`product-strategist`)
- writing stories and you keep saying "the user" without knowing which one (`story-writer`)
- a stakeholder invents "our persona would love this" in a meeting and you need to ask "based on what?"

## Evidence-based personas, not demographic fiction

A persona is a **research artifact**, not a marketing cartoon. The test: point at any attribute and name the evidence behind it. If you can't, delete it.

**Demographic fiction (fails):**

> **Marketing Mary**, 34, lives in Austin, drives a Subaru, does yoga, has 2.3 kids. She's "tech-savvy but time-poor."

Every clause is invented. Age, hobby, and car predict nothing about product behavior, and "tech-savvy but time-poor" is a horoscope — true of everyone, actionable for no one.

**Evidence-grounded (passes):**

> **The escalation-owner** — ops lead who fields manual-data-pull requests.
> - **Goal:** close a data request without pulling in an engineer.
> - **Context:** handles ~14 requests/week, usually mid-incident, from a shared inbox.
> - **Pain:** no self-serve export; every request becomes a ticket.
> - **Evidence:** 6/8 Oct interviews described this verbatim; 23 support tickets in 30 days; analytics show 0 self-serve exports.

The shape that carries signal is **goal / context / pain / evidence source** — behavior and situation, not biography. Two or three grounded personas beat a "persona zoo" of eight look-alikes nobody can tell apart.

### Proto-persona vs validated persona

You don't always have research yet. A **proto-persona** is an explicit hypothesis — the team's current best guess, *labeled as a guess*, with the evidence column reading "assumption, not yet validated." That's honest and useful for framing early discovery. It becomes a **validated persona** only when each attribute earns a real source. The failure isn't starting from a hypothesis; it's letting a proto-persona harden into "fact" without anyone ever checking it against a user.

### Building one from evidence

1. **Gather** raw signal — interview notes, support tickets, session recordings, analytics segments.
2. **Cluster** by *behavior*, not demographic — group people who act the same way toward the same goal.
3. **Name the goal** each cluster is pursuing; that goal, not an age bracket, is the persona's spine.
4. **Attach evidence** to every attribute; drop any attribute you can't source.
5. **Prune** to the 2–3 clusters that behave distinctly enough to change what you'd build.

## Journey maps — the grid

A journey map traces one persona through one goal, stage by stage, and exposes where the experience breaks. Use a five-column grid, one row per stage:

| Stage | Action | Thought / feeling | Pain | Opportunity |
|---|---|---|---|---|
| Discover need | Gets a data-pull request in shared inbox | "Again? I don't have time." | Interrupted mid-incident | Self-serve export the requester runs themselves |
| Find the tool | Searches docs for "export" | "Is this even possible?" | No discoverable path | Surface export in the request flow |
| Attempt | Files a ticket to engineering | "This will take days." | Hand-off latency | One-click export, no ticket |
| Wait / verify | Chases the ticket, re-checks | "Did it even work?" | No confirmation | Success/failure receipt |

- **Stage** — a phase of the experience in the user's terms, not your feature names.
- **Action** — what they actually do (observed, not assumed).
- **Thought / feeling** — the emotional read, in the user's voice; this is where friction is felt before it's rational.
- **Pain** — the concrete breakdown at that stage.
- **Opportunity** — the reframed pain as a user need (not a solution). These are the cells that matter.

The **evidence column** is non-negotiable in a real map: every pain cites its source the same way persona attributes do. A map with no evidence column is a storyboard of what the team imagines.

## How opportunities feed an OST

The Opportunity column is the bridge to discovery. Each opportunity is written user-shaped ("the requester can't tell if the export worked"), which is exactly the shape the Opportunity level of an Opportunity Solution Tree wants — see `opportunity-solution-trees-torres`. A journey map is one of the richest sources for emptying the opportunity space: it walks the whole experience and surfaces pains a feature-request list never would, because users report the loud pains and stay silent on the ambient ones the map exposes.

The handoff: harvest the Opportunity column → cluster with other research → hang under the outcome in the tree → generate solutions there, not in the map. The map finds opportunities; it does not pick solutions.

## Anti-patterns

- **Demographic fiction.** Age, hometown, hobbies, invented names with no behavioral evidence. Predicts nothing; delete it.
- **Persona zoos.** Eight personas that differ only cosmetically. If two personas would use the product identically, they're one persona. Aim for 2–3 that behave distinctly.
- **Happy-path-only maps.** Mapping only the flow where everything works. The pains — and therefore the opportunities — live in the error, wait, and recovery stages you skipped.
- **Maps with no evidence column.** A journey the team imagined is a storyboard, not research. Every pain cites a source or it's a guess.
- **Solutions in the Opportunity column.** "Add a notification center" is a solution. The opportunity is "the requester can't tell if the export worked." Keep the column user-shaped so the OST can diverge on solutions.
- **Personas that never get revisited.** A persona built once and frozen rots as the user base shifts. Re-ground it when new research lands.
- **The composite everyone.** A single persona that averages all users into a mush with contradictory goals. Real personas are specific enough to exclude someone.

## Cross-references

- `opportunity-solution-trees-torres` — the Opportunity column feeds the tree's opportunity level; personas anchor whose outcome it is.
- `evidence-synthesis` — clustering raw interviews/tickets into the attributes and pains these artifacts cite.
- `discovery-interview-design` — how to run the interviews that populate a persona and a journey.
- `jtbd-job-stories` — the job-story lens is the alternative to persona-first framing; use both deliberately, not interchangeably.
- `customer-segment-weighting` — when multiple personas compete, which segment's pain the bet should serve.
- Prior art: Alan Cooper (*The Inmates Are Running the Asylum*) on goal-directed personas; the journey-map grid as used across the discovery-research literature.
