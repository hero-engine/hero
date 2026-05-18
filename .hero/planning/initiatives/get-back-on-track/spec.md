---
title: Get Back on Track — Mission-First V2 Recovery
slug: get-back-on-track
type: initiative
status: planning
priority: P0
tags: [recovery, mission, principles, dogfood, foundational]
created: 2026-04-28
relations:
  - target: project-charter
    kind: child
  - target: acceptance-criteria-graph
    kind: child
  - target: e2e-area-suites
    kind: child
  - target: master-ingest-restore
    kind: child
  - target: traversal-queries
    kind: child
  - target: spec-status-integrity
    kind: child
  - target: spec-prioritization
    kind: child
  - target: core-vertical-layering
    kind: child
  - target: per-feature-smoke-coverage
    kind: child
  - target: recovery-strategy-conversation
    kind: derived-from
  - target: v2-delivery-audit-2026-04-28
    kind: derived-from
  - target: e2e-validation
    kind: supersedes-scope
horizon: now
---

## Mission alignment

Hero is a context-engineering system for AI coding tools. Its job is
to make sure the AI working in your editor has the right project
knowledge in its window at the right moment — including the stuff
nobody told it. The corpus that compounds across sessions, people, and
time is the substance; the spec-driven workflow is the surface; v2's
ambition is to make that corpus a team-shared institutional memory
layer.

This initiative serves that mission directly: every child feature
either makes the corpus richer, makes injection automatic, or prevents
the system from drifting away from the mission again.

## Experience principles (locked here for the initiative)

1. **It just works.** Zero ceremony. No command sequences. The system
   does the right thing without being told.
2. **Natural language is the interface; tools are the escape hatch.**
   Default user describes intent in their words; the system routes.
3. **Sessions start omniscient.** No re-priming. Resume feels like the
   previous session never ended.
4. **Sessions end making everyone smarter.** Zero ephemera; the corpus
   always grows.
5. **Two audiences, one product: the model and the human.** Magic for
   the model and the less-experienced human; tools for the
   practitioner. Practitioner surface never drowns the magic.

These principles are how each child spec will be evaluated and how
work will be sequenced.

## Goal

Recover from the post-v2 drift by making the mission and principles
load-bearing artifacts (not prose), restoring the broken corpus →
injection loop, and shipping the v2 traversal-query showcase that
justified the graph substrate. Treat the recovery itself as the
dogfood proof that Hero serves its own mission.

## Premise

The audits in [v2-delivery-audit-2026-04-28](../../knowledge/notes/v2-delivery-audit-2026-04-28/spec.md)
established the gap pattern: foundations ship; the surface that
delivers the mission doesn't. Every drift mode the audit catalogued
violates one or more of the principles above. The recovery has to
address both layers at once: fix the broken surface *and* install the
prevention layer that catches the next drift before it lands.

## Children — six features

| Slug | What it delivers | Sequence |
|---|---|---|
| [project-charter](../../features/project-charter/spec.md) | Mission + principles as a graph-backed, auto-injected, validation-enforcing artifact. The prevention layer. | **1** |
| [acceptance-criteria-graph](../../features/acceptance-criteria-graph/spec.md) | ACs become `Criterion` nodes; status flips through commit/test signals; injected into deliver/relevant/why/blocked. Ends frontmatter-status drift. | **2** |
| [spec-status-integrity](../../features/spec-status-integrity/spec.md) | A spec can't claim `status: completed` without graph verification. Lying becomes structurally expensive. | **3** |
| [master-ingest-restore](../../features/master-ingest-restore/spec.md) | `hero scan` returns to its v2 promise: code + planning + notes + raw + git + tracker + sync + memory + Tier-2, all in one verb. | **4** |
| [traversal-queries](../../features/traversal-queries/spec.md) | Build `hero why` and `hero blocked` for real. The v2 substrate finally pays its complexity tax. | **5** |
| [e2e-area-suites](../../features/e2e-area-suites/spec.md) | Split monolithic `e2e_smoke.sh` into 8 area suites; each gets its own ACs (graph-backed via #2). The repeatable proof the recovery is real. | **6 (continuous)** |

## Sequencing rationale

**Why charter first.** Every later spec, every later commit gets
checked against mission + principles. Without the artifact existing
first, the recovery itself can drift. Three days of `project-charter`
prevents three weeks of repeated drift.

**Why AC-graph + status-integrity second and third.** They're the
process fix that prevents recurrence. If we ship the corpus repairs
first without these, we get four months of confidence followed by
silent regression. AC-graph also unlocks a clean substrate for the
e2e suites.

**Why master-ingest before traversal.** Traversal queries that read
from a thin graph aren't a showcase — they're a disappointment. The
corpus must be rich before the queries that read it light up.

**Why e2e suites last (and continuous).** They're the verification
loop, not the deliverable. Each area suite hardens as the upstream
features land. They run continuously after that.

## Explicitly out of scope (deferred)

These were on the v2 roadmap but are mission-adjacent or speculative.
They wait until the recovery delivers:

- Cloud monetization scaffolding: `team-oauth`, `cloud-billing`,
  Stripe, NATS, `launch-readiness`
- Dashboard-data wiring (the shells are fine until the corpus is real)
- Automation triggers (Jira/webhook/cron) — engine loads YAML; firing
  it is mission-adjacent until #4 is done
- Marketing / content / distribution: `hero-marketing`,
  `hero-content-engine`, `hero-launch-playbook`, `hero-positioning`,
  `hero-sales`, `hero-community`, `hero-distribution`,
  `hero-landing-page`. Recovery first; launch later.
- The four flagship-but-unbuilt features: `cross-spec-awareness`,
  `institutional-memory`, `architectural-drift-detection`,
  `cross-org-intelligence`. Each gets re-evaluated for re-scoping
  after the substrate works. Some may merge with `traversal-queries`;
  others may defer to v2.1.

## Definition of done (initiative-level)

The initiative is complete when:

- `.hero/mission.md` exists, is locked, and is auto-injected into every
  agent context bundle
- Every spec authored after charter ships requires mission/principles
  fields and is rejected by `hero check` if missing
- Every existing spec's `status: completed` claim is graph-verified
  (or downgraded to honest `partial` / `planning`)
- `hero scan` populates the full graph in one run — code, planning,
  notes, raw, git, tracker, memory, Tier-2 — without the user
  remembering separate verbs
- `hero why <thing>` and `hero blocked` exist and return real
  multi-hop traversal results
- All 8 e2e area suites run green against `go-task/task` and the hero
  repo itself
- A single fresh session, on a fresh clone, picks up the mission +
  current state + open ACs + recent activity automatically — no
  command sequence required

## How this initiative is measured

Mission delivery, not feature count. Three metrics that matter:

- **Cold-start fidelity:** time from `hero resume` to a session that
  knows what's open, what's blocked, what changed since last time.
  Goal: < 3 seconds. Today: depends on how stale NEXT.md is.
- **Corpus completeness on `hero scan`:** ratio of
  promised-source-types to actually-ingested. Today: ~60%. Goal: 100%.
- **Spec-status truthfulness:** ratio of specs whose `status:
  completed` claim matches graph-verified AC pass rate. Today:
  unknown but at least 2 known liars. Goal: 100%.

## Why this is the right move now

Per the audits, the v2 work was the wrong shape: heavy investment in
team-platform business model, light investment in the mission itself.
Reversing that — making the mission the load-bearing artifact, fixing
the corpus → injection loop, then *building the queries that make the
graph worth having* — is the smallest set of moves that recovers v2's
promise.

The recovery is also the demo. If Hero can prevent its own drift from
this point forward, that's the case study every customer needs.
