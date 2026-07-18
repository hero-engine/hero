---
title: "PM Pack Completion — Author the Deferred Roster and Reframe Around Critics + Corpus-Grounding"
slug: pm-pack-completion
type: initiative
status: planning
domain: pm
size: x-large
horizon: now
created: 2026-07-17
tags: [domains, product-management, pm, content-pack]
relations:
  - target: hero-domains
    kind: parent
  - target: hero-pm
    kind: related
  - target: pm-foundation-delivery
    kind: related
child:
  - pm-doctrine-and-skill-backfill
  - story-queue-planning-backing
  - prd-editor-comms-backing
  - story-detail-and-intake-scrubber-backing
  - adversarial-critics-bundle
  - experiment-stage-and-metric-rca
  - competitive-and-market-grounding
  - exec-narrative-and-evidence-synthesis
  - discovery-framing-coverage-skills
  - remaining-roles-scrubbers-and-launch
---

# PM Pack Completion

## Summary

The PM content pack shipped as its design's deliberate v1 minimum-viable set
(12 agents / 10 commands / 19 skills). Now that hero-code embeds the Hero
engine and serves the pack live per active domain, running as PM it reports
the pack as "very light." The
[PM Pack Audit — 2026-07-17](../../features/hero-pm/pm-pack-audit-2026-07.md)
reconciled designed-vs-shipped, mapped the audit against hero-code's real
surfaces, and scanned the external PM best-practice corpus. It found three
gaps: ~half the designed roster (all P1/P2) was never authored; several
*shipped* hero-code views draw buttons with no backing agent; and the design
predates both hero-code's real surfaces and the external signal about where
the differentiated value actually lives.

This initiative completes and refreshes the pack. "Complete + refresh" means:
author the deferred roster, un-dangle every shipped view, and — the load-bearing
part — **reframe the pack around critics over generators and corpus-grounding
over free-association.**

### The differentiation thesis (critics over generators + corpus-grounding)

The external scan is consistent and well-sourced: PM *generators* are
commoditized and actively distrusted (Notion AI PRDs ~70% right / ~30%
generic-or-hallucinated; fabricated interview quotes; PRDs that "read like
someone who read a lot of PRDs but never shipped"). The failure mode is
confident, evidence-free output that looks like analysis and then propagates
into a roadmap and gets cited in a QBR. The underserved, high-leverage
capabilities are the *keep-it-honest* ones: evidence-linked synthesis,
roadmap-drift detection, anti-gaming prioritization critique, adversarial
experiment readouts, metric-movement RCA. Every trusted feature grounds in
the team's own corpus; every criticized one free-associates.

Our original design mapped generators→commands, mechanics→skills, and
reviewers→agents — but framed the reviewers as *passive quality gates*. The
refresh **elevates the critics into first-class adversarial agents** and makes
**corpus-grounding + decision-gate discipline a pack-wide doctrine skill**, not
an afterthought. This is the reframe that turns the pack from "engineering with
different words" into something a PM would switch tools for.

## Goals

1. **Clear hero-code's "very light" signal.** Every shipped hero-code PM view
   has a backing agent; no drawn button is dead. The designed P1/P2 roster is
   authored to parity with (or beyond) the engineering pack's shape.
2. **Surface the differentiator authored, not just planned.** The adversarial
   critics, corpus-grounding doctrine, experiment stage, and metric RCA exist
   as real agents/skills/commands — the capabilities the external scan ranks
   highest.
3. **Anchor the pack on a doctrine spine.** `pm-agent-doctrine` and
   `outcomes-over-outputs` exist and are loaded by every authoring/critic agent,
   so corpus-grounding and decision-gate discipline are enforced pack-wide.

## Success criteria

- hero-code, running as PM, no longer reports the pack as "light" — the
  surface-coverage matrix in the audit shows zero "❌ missing" backing agents.
- The six referenced-but-undesigned skills exist as files; `outcomes-over-outputs`
  (loaded by 6 agents) resolves.
- The `domains/pm/AGENTS.md` scaffold is replaced by the full canonical routing
  table; the `dashboard.md` orphan is reconciled.
- The differentiator agents (`prioritization-challenger`, `experiment-designer`,
  `experiment-readout-reviewer`, `metrics-analyst`, drift-critic
  `roadmap-reviewer`, sharpened `pm-critic`) are authored and routed.

## Wave / Sprint plan

This is the "why" ordering the `/drive` judge's child selection must stay in
sync with. Sprints group children that ship together; the Wave label ties each
back to the audit's wave plan.

| Sprint | Wave (audit) | Theme | Children | Why this sprint |
|---|---|---|---|---|
| **Sprint 1** | Wave 0 + Wave 1 | Unblock the consumer | 1, 2, 3, 4, 5 | Wiring fixes (skill backfill, AGENTS.md, orphan reconcile) + author the deferred items that back *live* hero-code surfaces whose buttons are currently dead. Highest urgency — the surface already exists in the consumer. |
| **Sprint 2** | Wave 2 | Differentiation | 6, 7, 8, 9 | The critics/rigor-forcers the external scan ranks highest and that are mostly net-new (not in the original design). This is what makes the pack worth switching to rather than parity. |
| **Sprint 3** | Wave 3 | Coverage fill | 10, 11 | Round out table-stakes framing skills + the remaining designed P1/P2 roles, scrubbers, and launch/GTM. Deferrable polish. |

## Children

Sequenced. Hard ordering lives in `depends-on`; `priority:` is the judge's
tiebreak among dependency-ready children; `conflicts-with` marks same-file /
overlap seams (see Cross-cutting concerns). Every child is a stub at
`.hero/planning/initiatives/pm-pack-completion/<slug>.md`, materialized to a
full spec by `/design` when it enters delivery.

| # | Slug | Priority | Sprint | Depends-on | Conflicts-with | Size |
|---|---|---|---|---|---|---|
| 1 | pm-doctrine-and-skill-backfill | **critical** | 1 | — | 4, 6, 7, 8, 9 | large |
| 3 | story-queue-planning-backing | high | 1 | 1, `pm-foundation-delivery` | — | medium |
| 4 | prd-editor-comms-backing | high | 1 | 1 | 1 | medium |
| 5 | story-detail-and-intake-scrubber-backing | high | 1 | 1 | 11 | small |
| 6 | adversarial-critics-bundle | high | 2 | 1 | 1, 7 | large |
| 7 | experiment-stage-and-metric-rca | high | 2 | 1 | 1, 6 | medium |
| 8 | competitive-and-market-grounding | medium | 2 | 1 | 1 | small |
| 9 | exec-narrative-and-evidence-synthesis | medium | 2 | 1 | 1 | small |
| 10 | discovery-framing-coverage-skills | low | 3 | 1 | — | small |
| 11 | remaining-roles-scrubbers-and-launch | low | 3 | 1, 5 | 5 | medium |

### What each child delivers (one line; the stub carries the full deliverable list)

1. **pm-doctrine-and-skill-backfill** — root anchor. Doctrine spine
   (`pm-agent-doctrine`, `outcomes-over-outputs`) + the referenced-but-undesigned
   backfill skills; retrofit the spine into the 8 shipped agents; replace the
   `domains/pm/AGENTS.md` scaffold with the full canonical 25-row routing table;
   reconcile the `dashboard.md` orphan.
   *(Child #2 `pm-spec-type-completion` was dropped 2026-07-17 — see Cross-cutting.
   The child numbers below are stable labels; #2 is intentionally vacant.)*
3. **story-queue-planning-backing** — `capacity-planner`, `cycle-planner` +
   planning skills + cadence commands. Backs the Story Queue view (zero backing
   agents today).
4. **prd-editor-comms-backing** — `pitch-author` (split from prd-author),
   `stakeholder-communicator` + comms skills + `/standup`, `/interview`.
   Un-dangles shipped `/pitch` and `/release-notes`.
5. **story-detail-and-intake-scrubber-backing** — `dependency-mapper`,
   `duplicate-intake-scrubber` + the intake `scrub` concern.
6. **adversarial-critics-bundle** — THE differentiation thesis. Drift-critic
   `roadmap-reviewer` + `outcome-drift`; `prioritization-challenger` +
   `evidence-forcing`; sharpen `pm-reviewer` into `pm-critic`;
   `experiment-readout-reviewer`. Kept as one bundle.
7. **experiment-stage-and-metric-rca** — `experiment-designer` +
   `experiment-design` + `/experiment`; `metrics-analyst` + `metric-rca`
   (un-dangles shipped `/metrics`). A whole stage absent from the design.
8. **competitive-and-market-grounding** — retrieval-only `competitive-analyst`;
   sharpen `product-strategist`; `opportunity-assessment`, `market-sizing`.
9. **exec-narrative-and-evidence-synthesis** — `prfaq-writing`, `exec-narrative`;
   sharpen `discovery-researcher`; extend `evidence-synthesis`.
10. **discovery-framing-coverage-skills** — personas/journey-maps, JTBD
    job-stories, positioning canvas, story mapping, hill-chart, glossary,
    product-vision skills.
11. **remaining-roles-scrubbers-and-launch** — `epic-framer`, `risk-curator`,
    `portfolio-curator`, `discovery-reviewer`, `stale-roadmap-scrubber`,
    `ambiguous-story-scrubber`; extend `/scrub` (roadmap + stories concerns);
    `launch-gtm-tiering` + `/launch`.

## Cross-cutting concerns & shared risks

### The `domains/pm/AGENTS.md` routing-table hotspot (seams 1↔4, 1↔6, 1↔7, 1↔8, 1↔9)

`domains/pm/AGENTS.md` is the single most contended file in this initiative.
**Child #1 owns it**: it replaces the current scaffold with the full canonical
25-row routing table (`agent-pack-design.md` §F) and retrofits the doctrine +
outcomes load-lines into the 8 shipped agents. Children **#4, #6, #7, #8, and
#9** each introduce net-new Wave-2 agents whose routes must be *registered* in
that same table — so each shares a same-file seam with #1.

**Mitigation:** #1's canonical table is authoritative. Every downstream child
that adds an agent appends its route only inside a clearly marked
`<!-- wave-2 additions -->` region and never rewrites #1's canonical rows. The
reciprocal `conflicts-with` edges between #1 and each of {4, 6, 7, 8, 9} keep
the `/drive` judge from selecting any of them concurrently with #1 (the edge is
carried on both children because the judge honors outbound edges only —
whichever child is being selected must itself carry the edge).

### Critic ↔ experiment coupling (seam 6↔7)

Child #6 authors `experiment-readout-reviewer`, which reviews the experiment
brief that Child #7 defines (`experiment-designer` + `experiment-design`). They
share the experiment-artifact contract: delivering them concurrently risks the
readout-reviewer asserting against a brief shape #7 hasn't finalized. The
reciprocal `conflicts-with` (6↔7) serializes them. Whether this should harden
from the current soft coupling into a `depends-on` (7 before 6) is **open
question (c)** below.

### Intake scrubber ↔ launch roles same-file seam (seam 5↔11)

Child #5 scaffolds the intake `scrub` concern and Child #11 extends the *same*
`domains/pm/commands/scrub.md` with roadmap + stories concerns
(`stale-roadmap-scrubber`, `ambiguous-story-scrubber`). Both edit one file —
a genuine same-file seam. #11 also `depends-on` #5 so #5's scrub scaffold lands
first; the reciprocal `conflicts-with` (5↔11) additionally protects against
concurrent edits if that ordering ever slips.

### Pack-wide doctrine (delivered by #1, depended on by all)

Corpus-grounding contract, decision-gate discipline, and compare-don't-replace
synthesis are the three doctrines every authoring/critic agent must obey. They
land in `pm-agent-doctrine` (Child #1) and are why every other child carries a
`depends-on: pm-doctrine-and-skill-backfill` edge — an agent authored before the
doctrine exists would ship without its grounding contract.

## Open questions (flag for Child #1 design — do not resolve here)

- **(a) Spec-type overlay ownership — RESOLVED 2026-07-17, Child #2 dropped.**
  `story` and `roadmap-item` are **not spec types** in this architecture: the
  locked `unified-spec-type-model` collapsed `story` into the canonical `feature`
  type (rendered "Story" via the `agile-scrum` vocabulary + `scrum` methodology
  `points`/lifecycle) and `roadmap-item` into `initiative`. `epic` is a canonical
  type already authored at `core/spec-types/epic.md`. All 9 canonical types,
  5 methodology profiles, and 6 vocabularies are on disk **and loaded by the
  engine** (verified: `internal/{spectypes,vocabulary,methodology,tasks}` Go
  packages present, `.hero/cache/spec-types.json` exported, `hero task` +
  `hero spec new --type feature` work). There is nothing to author — the layer is
  a **delivered dependency** owned by `pm-foundation-delivery`. Child #2
  (`pm-spec-type-completion`) is therefore **dropped**; #3 now depends on
  `pm-foundation-delivery` directly. Residual: `pm-foundation-delivery` is still
  stamped `status: planning` despite being functionally delivered — a stale-status
  fix for that spec's owner, not pack-content work.
- **(b) `dashboard.md` orphan.** Verified **not on disk** at
  `domains/pm/commands/dashboard.md` as of 2026-07-17 (the audit assumed it was
  present). Child #1 should verify against the live tree, then document-or-drop
  accordingly rather than assuming an orphan to reconcile.
- **(c) #6 ↔ #7 coupling.** Decide whether the experiment-brief format must land
  strictly before the readout-reviewer — i.e. whether the current soft 6↔7
  coupling should harden into a `depends-on: experiment-stage-and-metric-rca` on
  Child #6.

## Boundaries (out of scope)

- **hero-code consumer wiring** — `PMManifest.swift` stale-ref repoint, GTK PM
  spec-type vocabulary (`gtk-m4-pm-surfaces`). These are consumer-side; hand off
  to hero-code with the corrected roster, don't build them here.
- **`okr-design` / okr-author** — belongs to a future `strategy` domain, not PM.
- **Wholesale prompt rewrite of the 12 shipped P0 agents** — this initiative
  *retrofits* the doctrine load-lines into them (Child #1) but does not rewrite
  their prompts wholesale.

## Recommended delivery order

1. **Design Child #1 first** — it is the sole `critical` anchor and every other
   child depends on it (doctrine spine + AGENTS.md canonical table). Nothing
   downstream should start authoring agents until the doctrine and routing table
   exist.
2. Then Sprint 1's remaining live-surface backers (#3, #4, #5), then Sprint 2's
   differentiators (#6, #7, #8, #9 — honoring the 6↔7 seam), then Sprint 3
   coverage (#10, #11 — honoring the 5↔11 seam). #3 depends on both #1 and the
   delivered `pm-foundation-delivery` type/methodology layer (no new types).

## Progress

Just composed (2026-07-17). Child #2 (`pm-spec-type-completion`) dropped same day
after verifying the spec-type/methodology/vocabulary layer is already delivered
(see open question (a)). No children materialized or in delivery yet.
Pick up at: design Child #1 (`pm-doctrine-and-skill-backfill`) — the sole
`critical` anchor everything else depends on.
