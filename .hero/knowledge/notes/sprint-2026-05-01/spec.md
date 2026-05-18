---
title: Sprint Plan — 2026-05-01 → 2026-05-14
slug: sprint-2026-05-01
type: note
status: active
tags: [sprint, planning, delivery]
created: 2026-05-01
---

# Sprint Plan — 2026-05-01 → 2026-05-14

## Sprint Goal

**Close the load-bearing in-flight specs that are currently blocking the largest fan-out of downstream work, and stage the next mission-fit feature for the sprint after.**

Translation: stop starting, start finishing. The corpus has 11 specs in `delivering`, and the dependency graph shows that just two of those (`master-ingest-restore`, `traversal-queries`) sit upstream of 7 other specs that cannot move until they close. Closing the right four specs unblocks more work than starting any new one.

## State of the Workspace

| Bucket | Count | Note |
|---|---|---|
| Delivering | 11 | Genuinely too many; sprint should reduce, not expand |
| Planning | 55 | Healthy backlog |
| Completed | 80 | Good track record |
| Knowledge | 15 | Active corpus |
| Unclaimed | 47 | Includes most planning specs — single-dev project, claiming is informal |

Recent velocity (git log, last 14 days): ~25 commits, 6 phased feature deliveries closed, 3 bugs fixed, several new specs scaffolded. Realistic close-out capacity for a 2-week sprint at this pace: **3–5 features**, weighted toward the smaller / already-mostly-done end.

## Dependency Graph (in-flight + critical planning)

```
master-ingest-restore (delivering) ────┬──► traversal-queries (delivering) ──► e2e-traversal (delivering)
                                       │                                  └──► next-as-projection (planning)
                                       └──► unified-retrieval-layer (delivering)
                                       └──► next-as-projection (planning)

project-charter (planning, unclaimed) ─┬──► core-vertical-layering (planning)
                                       └──► e2e-area-suites (planning)

hero-positioning (planning, unclaimed) ─► hero-docs-site, hero-demo-content,
                                          hero-landing-page, hero-launch-playbook,
                                          hero-content-engine

hero-runner (planning, unclaimed) ─────┬──► hero-automations (planning)
                                       └──► hero-team-server (planning) ──► hero-dashboard-v2

agent-outposts (planning, NEW)         (no in-flight blockers; horizon: next)
```

**Highest fan-out target**: `master-ingest-restore` — closing it unblocks 4 specs immediately (3 in delivering, 1 in planning) and is itself prerequisite to closing two others currently in delivering.

## Selected Specs (in delivery order)

### 1. `master-ingest-restore` — close this week
- **Status**: delivering
- **Why first**: blocks 4 other specs; sits at the top of the in-flight dependency graph
- **Mission-fit**: highest — `hero scan` returning to its V2 promise is foundational corpus plumbing
- **Verify before close**: acceptance criteria, drift check, smoke run against a real repo

### 2. `traversal-queries` — close once #1 is in
- **Status**: delivering
- **Why second**: depends on #1; itself blocks `e2e-traversal` and `next-as-projection`
- **Mission-fit**: high — `hero why` and `hero blocked` are how the corpus surfaces causation, which is the mechanism by which "next session starts smarter"
- **Verify before close**: drift check, both queries return correct answers on a seeded graph

### 3. `spec-status-integrity` — close in parallel where possible
- **Status**: delivering
- **Why third**: independent of #1 and #2; can be closed in parallel; small scope (graph-verified delivery claims)
- **Mission-fit**: high — without this, the corpus contains specs that *say* they're done but aren't, and every downstream surface (recap, feed, why) lies
- **Verify before close**: spec audit catches `delivering`-but-not-merged drift

### 4. `tripwire-system` — close to lock in the work we just designed
- **Status**: delivering
- **Why fourth**: just designed; phases 1–4 in spec map cleanly to a close-out sequence; reinforces the "design → deliver → close in same window" cadence
- **Mission-fit**: high — directly solves model drift, which is the mechanism by which agents lose mission alignment mid-session
- **Verify before close**: integration with `/decide`, `/deliver`, `/diagnose`, `/design` skills; `hero_anchor` returns full mission + tripwires

### 5. (Stretch) `e2e-validation` OR `per-feature-smoke-coverage` — close one of two
- **Status**: both delivering
- **Why fifth, conditional**: only attempt if 1–4 land cleanly with time to spare. These are smaller-scope continuous-validation features. Pick whichever is closer to done — likely `per-feature-smoke-coverage` since Phase 5 (CI integration) already shipped per recent commits.

### 6. (Stretch) Promote `project-charter` from planning → delivering
- **Status**: planning, unclaimed
- **Why sixth, conditional**: doesn't need to *close* this sprint, but if 1–5 land we should *start* it. It blocks two important specs (`core-vertical-layering`, `e2e-area-suites`) and is mission-critical (mission auto-injection is a peer of tripwires).
- **Action**: read existing spec, refine if needed, claim, begin Phase 1.

## Explicitly Out of Sprint

- **`agent-outposts`** (just designed, horizon `next`): hold for next sprint. Spec is fresh; let it settle. No in-flight blockers.
- **`graph-memory`, `graph-memory-7c-live-test`, `graph-memory-federation`**: large unclaimed initiatives; need scoping/sequencing before they're sprintable. Candidate for `/compose` next sprint.
- **All `cloud-*` specs** (admin, billing, dashboard, mcp, notifications): different horizon (platform/ops), separate initiative, no immediate blocker pressure.
- **All `hero-marketing` specs** (positioning, docs-site, landing-page, launch-playbook, content-engine, demo-content, distribution): blocked behind `hero-positioning`, which is itself unclaimed. Marketing should be its own initiative-sprint when product is closer to launch-ready, not interleaved now.
- **`hero-runner`, `hero-team-server`, `hero-automations`, `hero-dashboard-v2`**: these form a coherent platform/team-server bundle that should be planned as one initiative, not picked off individually.
- **`beaver-db`, `hero-domains`, `hero-platform-vision`, `hero-inference-stack`**: speculative architecture initiatives; need scoping before delivery work.

## Capacity Allocation

Single-developer project. All work assigned to **chet-bellows**.

| Slot | Spec | Days estimate | Type |
|---|---|---|---|
| 1 | master-ingest-restore | 3–4 | close |
| 2 | traversal-queries | 2–3 | close |
| 3 | spec-status-integrity | 1–2 | close |
| 4 | tripwire-system | 2–3 | close |
| 5 (stretch) | per-feature-smoke-coverage **or** e2e-validation | 1–2 | close |
| 6 (stretch) | project-charter | 1–2 | start |
| Buffer | unplanned bugs, drift, context | 2–3 days (~20%) | reserved |

Total committed: ~8–12 working days for the four primary closes. 14-day sprint gives ~10 working days. Tight but achievable given recent velocity.

## Risks and Open Questions

1. **"Delivering" status may overstate progress.** Some specs marked `delivering` may not actually be near completion. Before starting, run `hero drift <slug>` against each of the four primary targets to verify they are actually close-able. If any is far from done, drop it from this sprint and replace with a smaller close.

2. **`master-ingest-restore` is the keystone.** If it slips, traversal-queries slips behind it, and three other specs stay blocked. Treat it as the priority-one item and protect time for it. If it shows signs of going long, consider `/split` to land a partial close.

3. **Tripwire system was designed today.** "Designed" and "delivering" are close cousins but not the same. Verify the existing delivery work matches the spec's Phase 1 scope before claiming Phase 1 is closeable; if not, re-anchor the spec's phasing to match reality.

4. **Single-dev sequencing is unforgiving.** No parallel pipelines means a stuck spec halts the sprint. Daily check-in: at end of each day, is the current spec on track to close in its allotted days? If not, swap or split.

5. **Velocity baseline includes a lot of small fixes.** The recent 25 commits in 2 weeks include 3 bug fixes, several refactors, and 6 phased deliveries — but several were small. Closing 4 large in-flight features is a heavier lift than the headline number suggests.

## Definition of Done for the Sprint

- 4 specs moved from `delivering` to `completed` with `hero spec complete` (the four primary closes).
- `hero blocked` shows fewer items at end-of-sprint than at start (the unblock-fan-out actually realized).
- `hero status` shows ≤8 in `delivering` at sprint end (down from 11), pointing the workspace back toward sustainable WIP.
- One stretch spec either closed (5) or started (6) if time permits.
- Sprint retro captured as a `/retro` on each closed spec.
