---
title: Roadmap Shape — Detect, Surface, and Resolve Spec-Shape Drift
slug: roadmap-shape
type: initiative
domain: engineering
status: completed
size: large
priority: P1
tags: [roadmap, sizing, composition, ambient-context, dogfood]
created: 2026-06-01
relations:
  - target: roadmap-review
    kind: child
  - target: roadmap-review-ambient-surfacing
    kind: child
  - target: size-drift-actionable-output
    kind: child
  - target: multi-spec-design-routing
    kind: child
  - target: spec-size-and-promotion-nudge
    kind: derived-from
---

## Goal

Make Hero genuinely good at *roadmap shape* — the meta-discipline of
detecting when the in-flight body of work is the wrong size, the wrong
grouping, or the wrong granularity, then surfacing that drift to the
user at the moment it's actionable and resolving it cleanly through
existing primitives (`/compose`, `/split`, size bumps). Across four
small, scoped children: a `/roadmap-review` command + agent + skill that
performs the detection pass on demand, ambient surfacing so users hear
about drift without remembering to ask, sharper actionable output on
`hero size --check`, and a routing nudge that catches the "I'm
`/design`-ing four related things" anti-pattern before it lands four
orphan specs.

## Kickoff

Initiative that makes Hero detect and resolve roadmap-shape drift
(oversized specs, related-but-orphaned features, unfinished
composition) instead of leaving it to the user to notice.

**Status:** ready — four child stubs scaffolded, ready for individual `/design` passes.

**Pick up at:** run `/design roadmap-review` first — it locks the
command name and Lenses scaffolding the other three reference. Then
`/design size-drift-actionable-output` and `/design multi-spec-design-routing`
in parallel. `/design roadmap-review-ambient-surfacing` ships last.

→ `.hero/planning/initiatives/roadmap-shape/spec.md`

**Files:** `.hero/planning/features/roadmap-review/spec.md`, `.hero/planning/features/roadmap-review-ambient-surfacing/spec.md`, `.hero/planning/features/size-drift-actionable-output/spec.md`, `.hero/planning/features/multi-spec-design-routing/spec.md`
**Skip:** designing all four in parallel — #2 depends on #1's locked command/agent names.

## Problem

Shipping `spec-size-and-promotion-nudge` proved the size mechanic works
in isolation: `hero size`, `hero estimate`, drift detection, and tracker
sync all do their jobs. But dogfooding it surfaced two structural gaps
that the size mechanic alone can't close:

1. **Detection without ambient surfacing is just noise generation.**
   `hero size --check` and `hero_warnings` will dutifully report drift,
   but only if the user remembers to run them. Users don't. The signal
   has to come *to* the user — in the handoff briefing, in pulse, in
   the delivery lead's pre-flight — at moments where action is
   natural. Without that, the field becomes the latest in a long line
   of lying frontmatter.

2. **Related specs go orphaned, not grouped.** When the user runs
   `/design foo`, `/design bar`, `/design baz` in close succession on
   topically related work, Hero scaffolds three flat features with no
   parent. The user discovers the missing initiative only later, when
   trying to reason about the work as a whole. The right moment to
   catch this is at the second or third `/design`, not after — and
   nothing in the current workflow does.

The initiative closes both gaps with four narrowly-scoped children
plus a sizing-only first cut of a roadmap-review capability that can
grow lenses (horizons, releases, sprint-shape) later without re-architecting.

## Approach

Four children, value-first sequencing, Lenses extension model.

**The four children.**

1. **`roadmap-review`** (medium) — the main capability. A `/roadmap-review`
   slash command + `roadmap-reviewer` agent + roadmap-review skill that
   performs a one-shot detection pass over the current planning corpus,
   classifies findings (oversized specs, related-but-orphaned specs,
   unfinished compositions), and emits a triage list. Sizing-only lens
   in v1 — horizons, releases, and sprint-shape are scaffolded
   placeholders in the skill's Lenses section but explicitly unimplemented.

2. **`roadmap-review-ambient-surfacing`** (small) — wire roadmap-review
   findings into three existing surfaces (NEXT.md projection, hero_pulse
   / hero_kickoff, delivery-lead pre-flight) so users hear about drift
   at natural moments. Surface message stays lens-agnostic ("size drift"
   not "sizing-lens drift") so the format extends cleanly as new lenses
   come online.

3. **`size-drift-actionable-output`** (trivial) — small fix to
   `hero size --check`: inline the next-step command on each drift row
   ("declared `medium`, computed `large` → run `hero size foo large`"),
   and dedupe the duplicate error from the `hero_warnings` path.

4. **`multi-spec-design-routing`** (small) — when the user fires
   `/design` against work that's clearly part of a larger group (heuristic:
   2+ related specs surfaced in the current session), nudge them toward
   `/compose` instead. Routing nudge only — no new heuristics, no
   `/compose` UX redesign.

**Value-first sequencing.** Ship the main capability first (#1), then
the supporting trivial/small work in parallel (#3, #4), then the
surfacing layer last (#2) so it benefits from a stable command name
and stable CLI shape.

**Lenses extension model.** The `roadmap-review` skill exposes a
`## Lenses` section listing the analytical perspectives the reviewer
applies: sizing today, horizons / releases / sprint-shape as named
placeholders for future iterations. The agent and command are
deliberately structured so adding a new lens is "extend a list,"
not "redesign the surface." The ambient surfacing format is also
lens-agnostic — we say "drift" or "shape concern," not the specific
lens name, so #2 doesn't have to revise messaging every time #1
grows a lens.

**One sentence on `hero check` vs `/roadmap-review`.** `hero check`
covers *workspace hygiene* (stale specs, missing fields, convention
drift). `/roadmap-review` covers *roadmap shape* (oversized work,
orphaned groupings, unfinished composition). They're complementary;
#1's spec should include the one-line distinction so the difference
is unambiguous in docs and agent prompts.

## Children

| Slug | Size | Goal | Status | Depends on |
|---|---|---|---|---|
| [`roadmap-review`](../../features/roadmap-review/spec.md) | medium | `/roadmap-review` command + `roadmap-reviewer` agent + skill; sizing-only lens in v1, Lenses scaffolding for future | planning | — |
| [`roadmap-review-ambient-surfacing`](../../features/roadmap-review-ambient-surfacing/spec.md) | small | Wire roadmap-review findings into NEXT.md, hero_pulse / hero_kickoff, delivery-lead pre-flight | planning | `roadmap-review` (names), soft on `size-drift-actionable-output` (CLI shape) |
| [`size-drift-actionable-output`](../../features/size-drift-actionable-output/spec.md) | trivial | Inline next-step command on `hero size --check` rows; dedupe `hero_warnings` duplicate error | planning | — |
| [`multi-spec-design-routing`](../../features/multi-spec-design-routing/spec.md) | small | Routing nudge from `/design` × N → `/compose` when 2+ related specs surface in a session | planning | — |

## Cross-cutting concerns

- **Command name `/roadmap-review` is load-bearing.** It appears in the
  command file, the agent prompt, the skill name, NEXT.md projection
  text, hero_pulse output, delivery-lead pre-flight, and #4's routing
  guidance. Lock it in #1 before #2 / #4 land. If name churn happens
  later, every cross-cutter rewrites.
- **Lenses extension model.** #2's surface message must not name a
  specific lens ("size drift," not "sizing-lens drift"). Keep the
  format extensible without committing to a taxonomy today.
- **Shared phrasing for "multiple related specs."** #1's triage logic
  and #4's `/design`→`/compose` nudge both detect the same condition
  ("2+ specs that look topically grouped"). Capture the canonical
  sentence once — either in the `spec-sizing` skill or as a new shared
  note — and quote it from both surfaces. Don't re-invent wording.
- **`hero check` vs `/roadmap-review` distinction.** One sentence in
  #1's spec, one sentence in the README / docs. Don't let the two
  capabilities collide in user mental models.

## Shared risks

- **Nudge fatigue (#2).** The biggest risk to the whole initiative. A
  user who gets pinged about roadmap shape on every session pickup
  will mute the channel — and once they mute it, the detection work
  in #1 is wasted. Mitigations: ship #2 last so we can tune the noise
  threshold against real #1 usage; make the surface count-only by
  default ("3 specs have shape concerns") and require a click-through
  for detail; have #2 surface only when findings *changed* since the
  last surface.
- **Name churn.** If `/roadmap-review` doesn't stick (users call it
  `/triage`, `/shape`, `/review`), every cross-cutter rewrites. Mitigation:
  validate the name in #1 against real user phrasing before #2 / #4
  reference it; consider stamping aliases.
- **Lens scaffolding rot.** Placeholders for horizons / releases /
  sprint-shape lenses with no owner or timeline become dead weight in
  the skill file. Either name the follow-up initiative that will
  implement them, or drop the scaffolding entirely in v1.
- **Overlap with `hero check`.** Risk that `/roadmap-review` feels
  redundant if the boundary isn't sharp. Mitigation: the one-sentence
  distinction baked into #1 (check = hygiene; roadmap-review = shape).

## Recommended delivery order

**Phase 1 — `roadmap-review`** (#1). Locks command name, agent name,
skill structure, Lenses scaffolding. Nothing else lands until this is
stable.

**Phase 2 — `size-drift-actionable-output` and `multi-spec-design-routing`**
(#3 and #4), in parallel. Both are small/trivial, both independent of
each other and of #1. #3 stabilizes the `hero size --check` row format
before #2 references it. #4 borrows the "multiple related specs"
phrasing established in #1.

**Phase 3 — `roadmap-review-ambient-surfacing`** (#2). Ships last so it
benefits from a stable command/agent name (#1) and a stable CLI shape
(#3). This is also where nudge fatigue gets tuned, so shipping it after
the rest gives a real corpus of drift to calibrate against.

## Out of scope

- Horizons, releases, and sprint-shape lenses beyond named scaffolding
  in #1's skill. v1 ships sizing only; other lenses are explicitly
  future work.
- Spec composition (`/compose`) UX redesign. #4 nudges *toward*
  `/compose` from `/design`; it does not redesign `/compose` itself.
- Ambient surfacing on surfaces beyond the three named in #2 (NEXT.md,
  hero_pulse / hero_kickoff, delivery-lead pre-flight). `/prime`,
  `/resume`, `hero status`, and the status bar are out for v1.
- New check categories in #3. Just the inline next-step formatting and
  the duplicate-error fix — no schema changes to `--check` output.
- Tracker-side roadmap-shape surfacing (e.g., labeling oversized Jira
  epics). Lens-level work; not v1.

## Boundaries

The initiative is deliberately narrow: detection + surfacing +
clean-resolution paths for *one* lens (sizing), wired into existing
primitives. It does not introduce new spec types, new tracker
integrations, new sequencing primitives, or a new "roadmap" data
model. If a child grows beyond its declared size, split the child;
do not let the initiative itself grow.

## Validation

- All four children scaffolded as `planning`-status stubs at the
  paths in the Children table, each with frontmatter linking back
  to this initiative.
- After delivery of all four, a fresh session running `/roadmap-review`
  on a corpus with known drift surfaces the drift; the same drift
  surfaces ambiently in NEXT.md and hero_pulse without explicit prompt;
  `hero size --check` rows include the inline next-step command;
  and a sequence of related `/design` calls produces a routing nudge.
- Nudge fatigue measured in a one-week dogfood window before declaring
  #2 done.
