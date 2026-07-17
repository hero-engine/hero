---
title: "Discovery + Framing Coverage Skills"
slug: discovery-framing-coverage-skills
type: feature
status: completed
domain: pm
priority: low
size: small
created: 2026-07-17
tags: [pm, discovery, framing, coverage, wave-3]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
completed_at: 2026-07-17T22:25:24Z
---

# Discovery + Framing Coverage Skills

## Goal

Author the seven table-stakes discovery-and-framing skills the PM pack is
missing, each a real, specific `domains/pm/skills/<name>/SKILL.md` (~90–130
lines) matching the shipped skill shape (frontmatter `name`/`description`/
`metadata`, `## What I do`, `## When to use me`, framework mechanics with
concrete examples, `## Anti-patterns`, `## Cross-references`), and list all
seven in the `domains/pm/AGENTS.md` Skills Reference. These are **coverage
skills** — loadable on demand, named in each skill's `metadata.audience` for
their intended consumer agents, but requiring **no** routing-table rows, no
Wave-2 route blocks, and no changes to agent load-lists. Done = seven skill
files exist with their framework's signature sections, valid frontmatter, and
the Skills Reference names all seven, with the AGENTS.md routing table and
Wave-2 region byte-unchanged.

## Kickoff

Cold-start prompt to paste into a fresh session:

> Deliver `discovery-framing-coverage-skills` (child of the `pm-pack-completion`
> initiative). Spec:
> `.hero/planning/initiatives/pm-pack-completion/discovery-framing-coverage-skills/spec.md`.
> Author 7 PM coverage skills under `domains/pm/skills/<name>/SKILL.md`:
> `personas-and-journey-maps`, `jtbd-job-stories`, `positioning-canvas`,
> `story-mapping`, `hill-chart-reasoning`, `domain-glossary-maintenance`,
> `product-vision-writing`. Match the shipped skill shape exactly — study
> `domains/pm/skills/roadmap-framing/SKILL.md`,
> `domains/pm/skills/story-writing-invest/SKILL.md`, and
> `domains/pm/skills/opportunity-solution-trees-torres/SKILL.md` for
> frontmatter, section order, depth, and voice. Each skill is ~90–130 lines of
> real, specific guidance (framework mechanics, worked examples,
> anti-patterns) — NOT a stub. Then add all 7 to the `domains/pm/AGENTS.md`
> Skills Reference as a single new "Discovery & framing coverage (Wave-3)"
> bullet inserted immediately before the `Core (installed with every pack)`
> bullet. Content only — author nothing outside `domains/pm/`; do NOT touch
> the Natural Language Routing table or any Wave-2 route/skill block. Run the
> `## Validation` bash block; every check must pass.

## Problem

The [PM Pack Audit — 2026-07-17](../../../features/hero-pm/pm-pack-audit-2026-07.md)
Wave-3 row and §"table-stakes coverage" call out that these framing skills
"recur across *every* published PM pack" and Hero currently has **none of
them**. Personas & journey maps, JTBD job-stories, positioning, story mapping,
hill-chart reasoning, glossary maintenance, and one-page product vision are the
baseline vocabulary any PM assistant is expected to know. Their absence is a
coverage gap, not a wiring bug: nothing references these skills today (so there
are no dangling refs to fix), but a PM asking Hero to "write a job story" or
"map this to a journey" gets nothing.

Three of the seven were named in the locked `hero-pm` design's P1 skill catalog
(`agent-pack-design.md` §D lists `hill-chart-reasoning`,
`product-vision-writing`, `domain-glossary-maintenance`); the other four
(`personas-and-journey-maps`, `jtbd-job-stories`, `positioning-canvas`,
`story-mapping`) are table-stakes additions surfaced by the external-scan
refresh. All seven are pure content under `domains/pm/` — no Go, no schema, no
consumer-side change.

This child **depends on** `pm-doctrine-and-skill-backfill`: these skills assume
the doctrine spine (corpus-grounding, decision-gate discipline, compare-don't-
replace) exists so they can cross-reference it rather than restate it. Author
after that child lands.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL provide seven skill files at
  `domains/pm/skills/{personas-and-journey-maps,jtbd-job-stories,positioning-canvas,story-mapping,hill-chart-reasoning,domain-glossary-maintenance,product-vision-writing}/SKILL.md`,
  each present and non-empty.
- **AC-2:** THE SYSTEM SHALL give each of the seven SKILL.md files valid YAML
  frontmatter carrying `name:` (matching the directory slug), a one-sentence
  `description:`, and a `metadata:` block with `audience:` (naming the intended
  consumer agents) and `purpose:`.
- **AC-3:** THE SYSTEM SHALL make each skill body carry its framework's
  signature sections — at minimum `## What I do`, `## When to use me`,
  `## Anti-patterns`, and `## Cross-references` — with substantive
  framework-specific content (each file ≥ 90 and ≤ ~140 lines, no `TODO`/`TBD`/
  `stub` placeholder markers).
- **AC-4:** WHEN a reader opens `personas-and-journey-maps` THE SYSTEM SHALL
  distinguish evidence-based personas from demographic fiction and lay out
  journey-map structure (stages → actions → pains → opportunities).
- **AC-5:** WHEN a reader opens `jtbd-job-stories` THE SYSTEM SHALL teach the
  `When [situation] I want [motivation] so [outcome]` shape, favor
  context/motivation over persona, and explicitly contrast it with INVEST user
  stories (cross-referencing `story-writing-invest`).
- **AC-6:** WHEN a reader opens `positioning-canvas` THE SYSTEM SHALL present
  April Dunford's five components (competitive alternatives → unique attributes
  → value → target customers → market category).
- **AC-7:** WHEN a reader opens `story-mapping` THE SYSTEM SHALL present the
  Patton backbone of activities, the walking-skeleton first slice, and
  release-slicing.
- **AC-8:** WHEN a reader opens `hill-chart-reasoning` THE SYSTEM SHALL frame
  the hill chart as unknowns-remaining (not progress %) and contrast the uphill
  (figuring-out) and downhill (executing) phases.
- **AC-9:** WHEN a reader opens `domain-glossary-maintenance` THE SYSTEM SHALL
  describe upkeep of a shared PM/eng vocabulary living in `.hero/knowledge/`.
- **AC-10:** WHEN a reader opens `product-vision-writing` THE SYSTEM SHALL
  describe the one-page vision laddering strategy → roadmap, rooted at the OST
  outcome, cross-referencing `outcomes-over-outputs` and `roadmap-framing`.
- **AC-11:** THE SYSTEM SHALL add all seven skill slugs to the
  `domains/pm/AGENTS.md` Skills Reference (a single new bullet) via
  line-insertion, leaving the Natural Language Routing table and every Wave-2
  route/skill block byte-unchanged.
- **AC-12:** THE SYSTEM SHALL introduce no dangling cross-references — every
  `` `skill-name` `` cross-ref in the seven new files resolves either to a
  shipped `domains/pm/skills/<name>/` directory or to one of the seven authored
  here.

## Changes

Author only under `domains/pm/`. Match the shipped skills' voice: first-person
"What I do", concrete worked examples with real-looking numbers, a "passes /
fails" contrast where it fits, and a candid `## Anti-patterns` list. Study
`roadmap-framing`, `story-writing-invest`, and `opportunity-solution-trees-torres`
before writing; mirror their frontmatter and section order.

1. **`domains/pm/skills/personas-and-journey-maps/SKILL.md`**
   - `metadata.audience`: `discovery-researcher, product-strategist, story-writer`.
   - Evidence-based personas: each persona attribute traceable to research
     (interviews, tickets, analytics), not invented demographics; contrast a
     fiction persona ("Marketing Mary, 34, likes yoga") against an
     evidence-grounded one (goal / context / pain / evidence source).
   - Journey maps: the stage → action → thought/feeling → pain → opportunity
     grid; how opportunities feed an OST (cross-ref
     `opportunity-solution-trees-torres`); anti-patterns (demographic fiction,
     happy-path-only maps, maps with no evidence column, persona zoos).

2. **`domains/pm/skills/jtbd-job-stories/SKILL.md`**
   - `metadata.audience`: `story-writer, discovery-researcher, product-strategist`.
   - The job-story shape `When [situation], I want [motivation], so [outcome]`;
     why context/motivation beats persona ("As a user…"); worked good/bad
     examples; the explicit contrast table vs. INVEST user stories and when to
     use which. Cross-ref `story-writing-invest` (INVEST is the delivery-ready
     spec bar; job stories are the discovery-framing lens) and
     `opportunity-solution-trees-torres`. Anti-patterns: persona-smuggling,
     solution-in-the-situation, outcome that restates the motivation.

3. **`domains/pm/skills/positioning-canvas/SKILL.md`**
   - `metadata.audience`: `product-strategist, competitive-analyst`.
   - April Dunford's five components in order (competitive alternatives → unique
     attributes → value those attributes enable → target customers who care most
     → market category that frames it); a worked example running the five
     components end-to-end; the "position relative to real alternatives, not an
     abstract ideal" discipline. Cross-ref `competitive-research` and
     `feature-comparison-framing`. Anti-patterns: category-first positioning,
     feature-list-as-value, positioning against a strawman.

4. **`domains/pm/skills/story-mapping/SKILL.md`**
   - `metadata.audience`: `story-writer, pm-delivery-lead, roadmap-curator`.
   - Jeff Patton's map: the horizontal backbone of user activities, tasks
     hanging below, the walking-skeleton thinnest end-to-end slice, then
     release slices across the map. Worked example. Cross-ref
     `story-writing-invest` (map cells become INVEST stories) and
     `roadmap-framing`. Anti-patterns: mapping features instead of activities,
     no walking skeleton, slicing by layer.

5. **`domains/pm/skills/hill-chart-reasoning/SKILL.md`**
   - `metadata.audience`: `pm-delivery-lead, cycle-planner, roadmap-curator`.
   - Basecamp's hill chart: position = unknowns-remaining, NOT progress %;
     uphill (still figuring out the approach — unknowns dominate) vs. downhill
     (approach is known, now executing). How to read a stuck-uphill scope and
     what it signals; how it differs from a burndown. Cross-ref `cycle-planning`
     and `pitch-writing-shape-up`. Anti-patterns: hill position as %-done,
     everything-at-the-top optimism, no movement between check-ins.

6. **`domains/pm/skills/domain-glossary-maintenance/SKILL.md`**
   - `metadata.audience`: `pm-reviewer, convention-author, product-strategist`.
   - Maintaining a shared PM/eng vocabulary as a knowledge entry under
     `.hero/knowledge/`; term entry shape (term → definition → owner → aliases →
     "not to be confused with"); when to add/retire a term; how ambiguity in
     specs traces back to a missing glossary entry. Cross-ref `convention-writing`
     and `pm-preset-detection` (vocabulary presets rename artifacts; the glossary
     records team-specific terms the preset doesn't cover). Anti-patterns:
     glossary nobody reads, duplicate terms with drifted definitions,
     eng-only or PM-only glossaries that never reconcile.

7. **`domains/pm/skills/product-vision-writing/SKILL.md`**
   - `metadata.audience`: `product-strategist, roadmap-curator`.
   - The one-page product vision laddering strategy → roadmap: vision statement
     rooted at the OST outcome, then how it constrains initiatives/horizons
     below it; a one-page template (who / problem / for-whom / unlike / our-
     approach / how-we'll-know); the ladder vision → strategy → roadmap →
     initiatives. Cross-ref `outcomes-over-outputs`, `roadmap-framing`, and
     `opportunity-solution-trees-torres`. Anti-patterns: vision as a slogan,
     vision that's really a feature list, vision with no measurable "how we'll
     know."

8. **`domains/pm/AGENTS.md` — Skills Reference (line-insertion only)**
   - Insert a single new bullet immediately **before** the
     `- **Core (installed with every pack):**` bullet (currently the last bullet
     of the Skills Reference list, after the four `Wave-2 …` bullets):
     ```
     - **Discovery & framing coverage (Wave-3):** `personas-and-journey-maps` (evidence-based personas + journey maps), `jtbd-job-stories` (`When … I want … so …`; context over persona), `positioning-canvas` (Dunford five-component positioning), `story-mapping` (Patton backbone + walking skeleton), `hill-chart-reasoning` (unknowns-remaining, not %-done), `domain-glossary-maintenance` (shared PM/eng vocabulary in `.hero/knowledge/`), `product-vision-writing` (one-page vision laddering strategy → roadmap).
     ```
   - Do **not** edit the Natural Language Routing table (no new rows), the
     Wave-2 route subsections, or the `Wave-2 …` Skills Reference bullets. This
     child adds no command or agent surface.

## Validation

```bash
set -euo pipefail
cd /Users/bwheeler/projects/hero-engine/repository/hero

skills="personas-and-journey-maps jtbd-job-stories positioning-canvas story-mapping hill-chart-reasoning domain-glossary-maintenance product-vision-writing"

# AC-1/AC-3: all seven files exist, non-empty, in the 90–140 line band, no placeholder markers
for s in $skills; do
  f="domains/pm/skills/$s/SKILL.md"
  test -s "$f" || { echo "MISSING: $f"; exit 1; }
  n=$(wc -l < "$f")
  [ "$n" -ge 90 ] && [ "$n" -le 145 ] || { echo "LINE COUNT out of band ($n): $f"; exit 1; }
  ! grep -qiE '\b(TODO|TBD|FIXME|placeholder|stub)\b' "$f" || { echo "PLACEHOLDER MARKER in $f"; exit 1; }
  # AC-2: frontmatter name matches slug + has metadata/audience/purpose
  grep -qE "^name: $s\$" "$f" || { echo "name != slug in $f"; exit 1; }
  grep -qE '^description: .+' "$f" || { echo "missing description in $f"; exit 1; }
  grep -qE '^metadata:' "$f" || { echo "missing metadata in $f"; exit 1; }
  grep -qE '^\s+audience:' "$f" || { echo "missing metadata.audience in $f"; exit 1; }
  grep -qE '^\s+purpose:' "$f" || { echo "missing metadata.purpose in $f"; exit 1; }
  # AC-3: signature sections present
  for h in '## What I do' '## When to use me' '## Anti-patterns' '## Cross-references'; do
    grep -qF "$h" "$f" || { echo "missing section '$h' in $f"; exit 1; }
  done
done

# AC-4..AC-10: framework signature phrases per skill
grep -qiF 'journey' domains/pm/skills/personas-and-journey-maps/SKILL.md
grep -qiE 'When .*I want .*so' domains/pm/skills/jtbd-job-stories/SKILL.md
grep -qF 'story-writing-invest' domains/pm/skills/jtbd-job-stories/SKILL.md
grep -qiF 'alternative' domains/pm/skills/positioning-canvas/SKILL.md
grep -qiF 'walking skeleton' domains/pm/skills/story-mapping/SKILL.md
grep -qiF 'backbone' domains/pm/skills/story-mapping/SKILL.md
grep -qiF 'unknown' domains/pm/skills/hill-chart-reasoning/SKILL.md
grep -qiF 'uphill' domains/pm/skills/hill-chart-reasoning/SKILL.md
grep -qiF '.hero/knowledge' domains/pm/skills/domain-glossary-maintenance/SKILL.md
grep -qF 'outcomes-over-outputs' domains/pm/skills/product-vision-writing/SKILL.md
grep -qF 'roadmap-framing' domains/pm/skills/product-vision-writing/SKILL.md

# AC-11: Skills Reference lists all seven; routing table + Wave-2 region unchanged
for s in $skills; do
  grep -qF "\`$s\`" domains/pm/AGENTS.md || { echo "AGENTS.md Skills Reference missing $s"; exit 1; }
done
grep -qF 'Discovery & framing coverage (Wave-3)' domains/pm/AGENTS.md

# AC-12: no dangling skill cross-refs — every backticked skill-ref resolves to a shipped or new skill dir
for s in $skills; do
  for ref in $(grep -oE '`[a-z0-9-]+`' "domains/pm/skills/$s/SKILL.md" | tr -d '`' | sort -u); do
    if [ -d "domains/pm/skills/$ref" ]; then continue; fi
    case " $skills " in *" $ref "*) continue;; esac
    # ignore non-skill backticked tokens (frontmatter keys, file paths handled above); only flag hyphenated names that look like skill slugs and don't resolve
    case "$ref" in *-*) echo "POSSIBLE dangling skill-ref '$ref' in $s (verify by hand)";; esac
  done
done

echo "VALIDATION OK"
```

The final `POSSIBLE dangling` lines are advisory (a hyphenated backticked token
need not be a skill slug — e.g. a spec-type name); the delivering engineer
inspects each and confirms it is not a broken cross-ref, satisfying AC-12.

## Boundaries

- **Content only, `domains/pm/` only.** No Go, no schema, no `hero.json`, no
  consumer-side (hero-code / GTK) change. The tripwire
  "harness-changes-cover-all-targets" is satisfied by authoring exclusively in
  the pack source.
- **No routing surface.** This child adds **no** Natural Language Routing rows,
  **no** slash command, **no** agent, and **no** Wave-2 route block. It does
  not wire any skill into an agent's load-list — coverage skills are loadable on
  demand, and the intended consumers are recorded in each skill's
  `metadata.audience` (optionally cross-referenced in prose), which is
  sufficient to ship.
- **Do not touch the canonical Natural Language Routing table or any prior
  child's Wave-2 blocks** in `domains/pm/AGENTS.md`. The only AGENTS.md edit is
  the single Skills Reference bullet insertion.
- **Out of scope (other Wave-3 children):** `acceptance-criteria-gherkin`,
  launch/GTM tiering + `/launch`, `risk-surfacing`, `assumption-testing`,
  `discovery-interview-design`, `okr-design`, and the remaining deferred
  agents/scrubbers — those belong to `remaining-roles-scrubbers-and-launch`.
- **Depends on `pm-doctrine-and-skill-backfill`.** These skills cross-reference
  the doctrine spine rather than restating it; author after that child lands.
  Delivering before it risks dangling doctrine cross-refs.

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| AC-1 | Seven SKILL.md files exist, present + non-empty | DONE | All seven `domains/pm/skills/{personas-and-journey-maps,jtbd-job-stories,positioning-canvas,story-mapping,hill-chart-reasoning,domain-glossary-maintenance,product-vision-writing}/SKILL.md` created; `test -s` passes for each in the Validation block. |
| AC-2 | Valid YAML frontmatter — `name:` = slug, one-sentence `description:`, `metadata.audience` + `metadata.purpose` | DONE | Ruby `YAML.safe_load` parses all seven; `name` equals dir slug, `description` non-empty, `metadata.audience`/`purpose` present. Grep guards (`^name: $s$`, `^description:`, `^metadata:`, `audience:`, `purpose:`) pass. |
| AC-3 | Signature sections present, 90–140 lines, no placeholder markers | DONE | Line counts 90/95/92/92/90/90/90; all four required headings (`## What I do`, `## When to use me`, `## Anti-patterns`, `## Cross-references`) present in each; no TODO/TBD/FIXME/placeholder/stub markers. |
| AC-4 | `personas-and-journey-maps` distinguishes evidence-based personas from demographic fiction + journey grid (stages → actions → pains → opportunities) | DONE | "Marketing Mary" fiction vs. evidence-grounded persona contrast; five-column journey grid (Stage / Action / Thought-feeling / Pain / Opportunity); `journey` grep passes. |
| AC-5 | `jtbd-job-stories` teaches `When … I want … so …`, context over persona, contrasts INVEST (x-ref `story-writing-invest`) | DONE | Job-story shape + good/weak examples; "Why context/motivation beats persona"; vs-INVEST comparison table; `When .*I want .*so` and `story-writing-invest` greps pass. |
| AC-6 | `positioning-canvas` presents Dunford's five components in order | DONE | Ordered §1–5 (competitive alternatives → unique attributes → value → target customers → market category) with worked example; `alternative` grep passes. |
| AC-7 | `story-mapping` presents Patton backbone + walking skeleton + release slicing | DONE | Backbone/tasks structure, "The walking skeleton — the first slice", "Release slicing" horizontal cuts; `walking skeleton` + `backbone` greps pass. |
| AC-8 | `hill-chart-reasoning` frames hill as unknowns-remaining (not %) + uphill/downhill | DONE | "Position = unknowns remaining, NOT percent done" section; uphill (figuring-out) vs downhill (executing); `unknown` + `uphill` greps pass. |
| AC-9 | `domain-glossary-maintenance` describes upkeep of shared PM/eng vocabulary in `.hero/knowledge/` | DONE | "Where it lives" (`.hero/knowledge/domain-glossary.md`), term entry shape, add/retire cadence, PM/eng reconciliation; `.hero/knowledge` grep passes. |
| AC-10 | `product-vision-writing` describes one-page vision laddering strategy → roadmap, rooted at OST outcome, x-ref `outcomes-over-outputs` + `roadmap-framing` | DONE | Vision→strategy→roadmap→initiatives ladder, six-clause one-page template, "Rooted at the OST outcome"; `outcomes-over-outputs` + `roadmap-framing` greps pass. |
| AC-11 | All seven slugs added to AGENTS.md Skills Reference via single bullet; routing table + Wave-2 region byte-unchanged | DONE | `git diff --stat` = `1 insertion(+)`; single "Discovery & framing coverage (Wave-3)" bullet inserted before the Core bullet; no changes to routing table or Wave-2 lines. All seven `` `slug` `` greps + Wave-3 marker grep pass. |
| AC-12 | No dangling cross-references | DONE | 20 advisory `POSSIBLE dangling` tokens inspected by hand: 19 are backticked consumer-agent roles in "When to use me" (real agents under `domains/pm/agents/` + `core/agents/`, plus the spec-mandated `cycle-planner` role), 1 is `convention-writing` (real shipped skill at `core/skills/convention-writing/`). Zero broken skill cross-refs. All prose skill x-refs resolve to shipped `domains/pm/skills/` dirs or the seven authored here. |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | `domains/pm/skills/personas-and-journey-maps/SKILL.md` | DONE | Authored, 94 lines. Evidence-based vs fiction personas, proto vs validated, build-from-evidence process, five-column journey grid, OST hand-off. audience: `discovery-researcher, product-strategist, story-writer`. |
| 2 | `domains/pm/skills/jtbd-job-stories/SKILL.md` | DONE | Authored, 95 lines. `When … I want … so …` shape, context-over-persona, four forces, force-ranking, vs-INVEST table. audience: `story-writer, discovery-researcher, product-strategist`. |
| 3 | `domains/pm/skills/positioning-canvas/SKILL.md` | DONE | Authored, 92 lines. Dunford five components in order, worked example, category-as-lever, staleness triggers. audience: `product-strategist, competitive-analyst`. |
| 4 | `domains/pm/skills/story-mapping/SKILL.md` | DONE | Authored, 92 lines. Patton backbone + tasks, walking skeleton, release slicing, mapping-session process, worked example. audience: `story-writer, pm-delivery-lead, roadmap-curator`. |
| 5 | `domains/pm/skills/hill-chart-reasoning/SKILL.md` | DONE | Authored, 90 lines. Unknowns-remaining (not %), uphill/downhill, scopes-not-tasks, check-in ritual, vs-burndown. audience: `pm-delivery-lead, cycle-planner, roadmap-curator`. |
| 6 | `domains/pm/skills/domain-glossary-maintenance/SKILL.md` | DONE | Authored, 90 lines. `.hero/knowledge/` home, five-field term entry, add/retire, review cadence, PM/eng reconciliation. audience: `pm-reviewer, convention-author, product-strategist`. |
| 7 | `domains/pm/skills/product-vision-writing/SKILL.md` | DONE | Authored, 90 lines. Vision→strategy→roadmap ladder, OST root, six-clause one-page template, writing process, decision test. audience: `product-strategist, roadmap-curator`. |
| 8 | `domains/pm/AGENTS.md` — Skills Reference bullet insertion | DONE | Single "Discovery & framing coverage (Wave-3)" bullet inserted immediately before the Core bullet. Routing table and Wave-2 region byte-unchanged (`git diff` = 1 insertion). |

### Exercise-the-feature check

- [x] All seven frontmatter blocks parse as valid YAML (`ruby -ryaml YAML.safe_load`, UTF-8) — not just grep-matched.
- [x] Each `metadata.audience` value verified equal to the spec's mandated consumer agents for that skill.
- [x] Each skill body carries its framework's signature sections + framework-specific phrase (journey / `When…I want…so` / alternative / walking skeleton + backbone / unknown + uphill / `.hero/knowledge` / outcomes-over-outputs + roadmap-framing).
- [x] Skills are discoverable: all seven listed in the `domains/pm/AGENTS.md` Skills Reference.
- [x] Spec's full `## Validation` bash block run verbatim under bash from repo root → `VALIDATION OK` (only advisory `POSSIBLE dangling` lines, each hand-verified as a non-skill agent-role reference or a real shipped skill).
- Note: these are content-only, load-on-demand coverage skills with no runtime/CLI surface to drive beyond being loaded. The appropriate exercise is frontmatter-parses + sections-present + discoverable-in-Skills-Reference, all confirmed above. Per the spec's Boundaries, they are intentionally NOT wired into any agent load-list, so there is no agent-invocation path to exercise.

### Excellence Bar self-check

- Each skill mirrors the shipped skill voice (first-person "What I do", concrete worked examples with real-looking numbers, "passes/fails" contrasts, candid `## Anti-patterns`) rather than reciting a definition — matched against `roadmap-framing`, `story-writing-invest`, `opportunity-solution-trees-torres`, `outcomes-over-outputs`.
- Every skill carries real framework depth beyond the AC minimum: JTBD's four forces + force-ranking, Dunford's category-as-lever, the story-mapping session process, the hill-chart check-in ritual and scopes-not-tasks granularity, glossary review cadence, the vision shelf-life ladder — none are stubs.
- Cross-references are genuine and resolve (verified against `domains/pm/skills/` and `core/skills/`), so the skills compose with the existing corpus instead of dangling.
- Scope held exactly to `domains/pm/` content: no Go, no schema, no routing rows, no Wave-2 edits, no agent load-list wiring — the single AGENTS.md edit is the mandated one-bullet insertion.
- Would I show this to a senior PM/engineer who cares? Yes — each file is a usable, opinionated reference a PM assistant can act from, not a placeholder to fill in later.
