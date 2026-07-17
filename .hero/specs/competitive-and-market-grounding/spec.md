---
title: "Competitive + Market Grounding — Retrieval-Only Competitive Analyst"
slug: competitive-and-market-grounding
type: feature
status: completed
domain: pm
priority: medium
size: small
created: 2026-07-17
tags: [pm, competitive, market-sizing, differentiation, wave-2]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-doctrine-and-skill-backfill
    kind: conflicts-with
completed_at: 2026-07-17T21:50:41Z
---

# Competitive + Market Grounding — Retrieval-Only Competitive Analyst

## Goal

Author the PM pack's competitive + market-grounding surface as **content only,
inside `domains/pm/`** — no Go, no engine changes. Three new artifacts plus one
sharpen:

1. A `competitive-analyst` agent whose defining doctrine is
   **retrieval-augmented, never model-memory**: it refuses to produce a
   teardown from training-data recollection and pulls live, dated, linkable
   sources for every competitive claim. It loads `pm-agent-doctrine` and the two
   child-#1 foundation skills (`competitive-research`, `feature-comparison-framing`)
   plus `evidence-synthesis`.
2. An `opportunity-assessment` skill — Cagan's 10-question opportunity
   assessment, run under single-challengeable-assumption discipline.
3. A `market-sizing` skill — TAM/SAM/SOM where every step rests on ONE statable,
   challengeable assumption, computed top-down **and** bottom-up with divergence
   flagged.
4. `product-strategist` sharpened to load the two new skills and to demand
   defensible sizing before committing a bet — **preserving** child #1's existing
   load-list additions.

"Done" is verifiable by the `## Validation` bash block: the files exist, contain
the named doctrine/framework sections, wire the correct load-lists, and the
AGENTS.md region is extended additively with no dangling references.

## Kickoff

Adds the `competitive-analyst` agent (retrieval-only, never model-memory) plus
`opportunity-assessment` (Cagan 10-Q) and `market-sizing` (TAM/SAM/SOM) skills,
and sharpens `product-strategist` to demand defensible sizing before a bet — all
content-only under `domains/pm/`.

**Status:** planning — spec just materialized from the initiative stub; no files
written yet.

**Pick up at:** author `domains/pm/agents/competitive-analyst.md` first (it's the
anchor and the retrieval doctrine is the whole point), modelling frontmatter on
`domains/pm/agents/experiment-designer.md`. Then the two skills, then the
`product-strategist` sharpen, then the additive AGENTS.md edits.

→ `.hero/planning/initiatives/pm-pack-completion/competitive-and-market-grounding/spec.md`

**Files:** `domains/pm/agents/experiment-designer.md` (frontmatter model), `domains/pm/skills/competitive-research/SKILL.md` (the retrieval rule the agent inherits), `domains/pm/agents/product-strategist.md:23-31` (load-list to extend), `domains/pm/AGENTS.md:62-65` (Wave-2 marker + prior children)
**Skip:** editing the canonical AGENTS.md routing table or prior children's Wave-2 routes — this child appends below them only. No Go/engine changes.

## Problem

The `pm-pack-audit-2026-07.md` refresh (Wave 2, "Defensible market sizing /
opportunity assessment" and "Live-augmented competitive teardown" rows) found
two related gaps that the external best-practice scan ranks among the pack's
highest-leverage differentiators:

- **Competitive analysis is designed as a role only** (`agent-pack-design.md`
  §C.2 `competitive-analyst` — P1) with no agent file on disk. The audit's
  central finding is that *corpus grounding is the line between praised and
  gimmicky*, and competitive analysis is the highest-stakes place for it: a
  model's training-data recollection of a rival's feature set is **stale by
  construction** and confidently wrong often enough to be dangerous — a
  fabricated "Competitor X already has this" can kill a good bet or greenlight a
  redundant one. The foundation skills already exist (`competitive-research`,
  `feature-comparison-framing`, both authored by child #1, both forward-authored
  *for this agent*) but nothing loads them.
- **Market sizing and opportunity assessment don't exist** (audit: "**no**" in
  the design). `product-strategist` frames bets in outcomes and tradeoffs but has
  no discipline for *how big* the opportunity is or *whether it's worth
  pursuing*, so a bet can be framed with a compelling outcome and an
  undefended market. The audit calls for **single-challengeable-assumption
  discipline** and **top-down↔bottom-up convergence** — the difference between a
  defensible size and a number pulled from the air.

This child depends on child #1 (`pm-doctrine-and-skill-backfill`): the
retrieval-not-memory constraint *is* the corpus-grounding doctrine applied to
competitive work, and `competitive-analyst` loads `pm-agent-doctrine` plus the
two skills child #1 shipped. It carries a reciprocal `conflicts-with` on child #1
because both edit `domains/pm/AGENTS.md` — child #1 owns the canonical table and
the Wave-2 marker region; this child appends strictly below the prior children's
routes.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL provide `domains/pm/agents/competitive-analyst.md`
  with valid agent frontmatter (`name: competitive-analyst`, `mode: subagent`,
  `webfetch: allow`) and a body whose defining doctrine is stated as
  **retrieval-augmented, never model-memory** — a memory-only teardown named as
  an explicit anti-pattern the agent refuses.
- **AC-2:** THE SYSTEM SHALL have `competitive-analyst.md` declare a Startup /
  load-list that loads exactly `pm-agent-doctrine`, `competitive-research`,
  `feature-comparison-framing`, and `evidence-synthesis` — all four of which
  exist on disk (no dangling skill reference).
- **AC-3:** THE SYSTEM SHALL provide `domains/pm/skills/opportunity-assessment/SKILL.md`
  containing the Cagan **10-question** opportunity assessment (problem / for-whom
  / how-big / alternatives / why-us / why-now / go-to-market / success-metric /
  critical-success-factors / go-no-go) and an explicit
  **single-challengeable-assumption** discipline section.
- **AC-4:** THE SYSTEM SHALL provide `domains/pm/skills/market-sizing/SKILL.md`
  containing a **TAM / SAM / SOM** section where each step is documented as
  resting on ONE statable, challengeable assumption, plus a **top-down↔bottom-up
  convergence** section that instructs flagging divergence rather than averaging
  it away.
- **AC-5:** THE SYSTEM SHALL update `domains/pm/agents/product-strategist.md` so
  its load-list gains `opportunity-assessment` and `market-sizing` AND its stance
  requires defensible sizing before committing a bet, WHILE preserving child #1's
  existing additions (`pm-agent-doctrine`, `outcomes-over-outputs`,
  `risk-surfacing` all remain in the load-list).
- **AC-6:** WHEN `domains/pm/AGENTS.md` is edited THE SYSTEM SHALL append the new
  routes **below** the `<!-- WAVE-2 ROUTES -->` marker region, after the prior
  children's Wave-2 subsections, leaving the canonical routing table and every
  prior child's routes byte-for-byte unchanged (additions-only diff above the new
  subsection).
- **AC-7:** THE SYSTEM SHALL add `competitive-analyst` to the AGENTS.md Agents
  Reference and both new skills (`opportunity-assessment`, `market-sizing`) to the
  Skills Reference, such that every agent/skill named in the new routes resolves
  to a file that exists (no dangling references anywhere in the pack).
- **AC-8:** IF any authored file references a skill or agent id THEN THE SYSTEM
  SHALL ensure that id resolves to an on-disk `domains/pm/` artifact — the
  `## Validation` block's dangling-reference check passes.

## Changes

Author only inside `domains/pm/` pack source (tripwire
`harness-changes-cover-all-targets` — do NOT hand-edit installed `.claude/` /
`.agents/` / `.codex/` copies; `hero install` regenerates them from pack source).

1. **New agent — `domains/pm/agents/competitive-analyst.md`.**
   Model frontmatter on `domains/pm/agents/experiment-designer.md` (`mode:
   subagent`, `temperature: 0.1`, `color: secondary`, `permission.edit: allow`,
   `task."*": deny`, `skill."*": allow`, `webfetch: allow`). Body:
   - Opening stance: senior competitive analyst; describes what competitors
     **actually ship** (observed behavior, not marketing claims); distinguishes
     must-match parity from optional differentiation from white space.
   - **The retrieval doctrine, stated as the agent's spine:** retrieval-augmented,
     **never model-memory**. Model recollection is a lead for *where to look*,
     never a source for *what is true*. Every competitive claim carries
     what / source (linkable, retrieved) / observed-date, or is marked an
     unverified lead. A memory-only teardown is named as an **anti-pattern the
     agent refuses** — "a teardown without live data is a teardown of last
     year's market."
   - `## Startup` load-list: `pm-agent-doctrine` (corpus-grounding /
     suggest-don't-decide), `competitive-research` (the retrieval rule + teardown
     method it inherits), `feature-comparison-framing` (the three-band matrix),
     `evidence-synthesis` (weighting sourced signal). No other skills.
   - `## When invoked`: `/discover` "what are competitors doing about X",
     competitive intake from sales, "should we match feature X" asks; produces
     competitive snapshots as notes in `.hero/knowledge/` and attached evidence
     on initiatives (per §C.2). Delegates to none.
   - `## Anti-patterns` and `## Default output` sections in the house style of
     the sibling agents (e.g. `experiment-designer.md`, `product-strategist.md`).

2. **New skill — `domains/pm/skills/opportunity-assessment/SKILL.md.**
   Match the SKILL.md shape of `domains/pm/skills/competitive-research/SKILL.md`
   (`---` frontmatter with `name`, `description`, `metadata.audience:
   product-strategist`, `metadata.purpose: framework-guidance`). Body:
   - `## What I do` / `## When to use me`.
   - The **Cagan 10-question** assessment as a numbered walkthrough: (1) problem /
     value proposition, (2) for whom / target market, (3) how big / market size,
     (4) alternatives / competitive landscape, (5) why us / our differentiator,
     (6) why now / market window, (7) go-to-market, (8) how we measure success /
     revenue, (9) critical success factors, (10) recommendation / go-no-go.
   - **Single-challengeable-assumption discipline:** each answer rests on ONE
     statable assumption a skeptic can attack; if you can't state it in one
     challengeable sentence, the answer is hand-waving. Cross-reference
     `market-sizing` for Q3 and `competitive-research` for Q4.
   - A copy-paste `## Opportunity Assessment` artifact template.

3. **New skill — `domains/pm/skills/market-sizing/SKILL.md`.**
   Same SKILL.md shape, `metadata.audience: product-strategist`. Body:
   - `## What I do` / `## When to use me`.
   - **TAM / SAM / SOM** section: define each tier, and require that every step
     down (TAM→SAM→SOM) is a documented, ONE-sentence challengeable assumption
     (e.g. "SAM = TAM × {% reachable by our GTM} — assumption: we can reach
     English-speaking SMBs on self-serve, ~X%"). No un-annotated multipliers.
   - **Top-down↔bottom-up convergence** section: compute the number both ways
     (top-down from market reports; bottom-up from unit economics × reachable
     accounts), then **flag divergence** as a signal to investigate rather than
     averaging the two into a false midpoint. Divergence > ~2–3× means an
     assumption is wrong — name which one.
   - A copy-paste `## Market Sizing` artifact template.

4. **Sharpen — `domains/pm/agents/product-strategist.md`.**
   - In `## Startup` (currently lines ~23–31), **add two bullets**:
     `opportunity-assessment` — the Cagan 10-Q gate before committing a bet; and
     `market-sizing` — defensible TAM/SAM/SOM with single-challengeable-assumption
     discipline. **Preserve every existing bullet**, especially child #1's
     additions: `pm-agent-doctrine`, `outcomes-over-outputs`, `risk-surfacing`
     (all three must remain).
   - Extend the stance/workflow: before the bet is committed (step 4 "Make
     tradeoffs visible" / the artifact in step 6), require a **defensible size** —
     a bet against an undefended or unsized market is theater the same way a bet
     without an outcome is. Add a short note that the strategist runs (or
     delegates to `competitive-analyst` for) the sizing + opportunity pass, and
     cite `market-sizing` / `opportunity-assessment`. Keep edits surgical and in
     the file's existing voice; do not restructure the numbered workflow.

5. **Extend — `domains/pm/AGENTS.md` (additions-only, below the marker).**
   - Append a new `#### Wave-2 competitive & market-grounding routes` subsection
     **below** the existing Wave-2 subsections (after the
     `story-detail-and-intake-scrubber-backing` block and before the "When
     routing, pass the user's original context…" paragraph), with rows:
     competitive teardown → invoke `competitive-analyst` directly (agent —
     retrieval-only teardown + feature matrix + positioning; no `/review`-style
     command ships in pm); opportunity assessment / market sizing surface **via
     `product-strategist`** (the strategist runs the Cagan 10-Q + TAM/SAM/SOM
     before committing a bet — no new command).
   - Add `competitive-analyst` to the **Agents Reference** list under a
     `- **PM Wave-2 competitive & market:**` bullet (retrieval-only competitive
     teardown; loads `competitive-research` + `feature-comparison-framing`).
   - Add both skills to the **Skills Reference** under a
     `- **Wave-2 competitive & market:**` bullet: `opportunity-assessment`
     (Cagan 10-Q, single-challengeable-assumption) and `market-sizing`
     (TAM/SAM/SOM, top-down↔bottom-up convergence).
   - Do **NOT** touch the canonical routing table (lines ~30–60) or any prior
     child's Wave-2 subsection.

## Validation

Run from repo root (`/Users/bwheeler/projects/hero-engine/repository/hero`):

```bash
set -e
PM=domains/pm
CA=$PM/agents/competitive-analyst.md
OA=$PM/skills/opportunity-assessment/SKILL.md
MS=$PM/skills/market-sizing/SKILL.md
PS=$PM/agents/product-strategist.md
AG=$PM/AGENTS.md

echo "== AC-1/AC-2: competitive-analyst exists, retrieval doctrine, load-list =="
test -f "$CA"
grep -qi "never .*model-memory\|not model-memory\|never model memory" "$CA"
grep -qiE "retrieval-augmented" "$CA"
for s in pm-agent-doctrine competitive-research feature-comparison-framing evidence-synthesis; do
  grep -q "$s" "$CA" || { echo "MISSING load-list skill in competitive-analyst: $s"; exit 1; }
done

echo "== AC-3: opportunity-assessment skill, Cagan 10-Q + single-assumption =="
test -f "$OA"
grep -qiE "10[- ]question|ten[- ]question" "$OA"
grep -qi "go.?no.?go" "$OA"
grep -qi "single.challengeable.assumption\|one .*challengeable" "$OA"

echo "== AC-4: market-sizing skill, TAM/SAM/SOM + convergence =="
test -f "$MS"
grep -q "TAM" "$MS"; grep -q "SAM" "$MS"; grep -q "SOM" "$MS"
grep -qiE "top-down.*bottom-up|bottom-up.*top-down" "$MS"
grep -qi "diverg" "$MS"

echo "== AC-5: product-strategist gains 2 skills, preserves child #1 loads =="
grep -q "opportunity-assessment" "$PS"
grep -q "market-sizing" "$PS"
for s in pm-agent-doctrine outcomes-over-outputs risk-surfacing; do
  grep -q "$s" "$PS" || { echo "REGRESSION: product-strategist lost child-#1 load: $s"; exit 1; }
done

echo "== AC-6/AC-7: AGENTS.md additive — new routes below marker, refs added =="
grep -q "WAVE-2 ROUTES" "$AG"
grep -q "competitive-analyst" "$AG"
grep -q "opportunity-assessment" "$AG"
grep -q "market-sizing" "$AG"
# new subsection must appear AFTER the last prior Wave-2 child block
awk '/story-detail-and-intake-scrubber-backing/{a=NR} /Wave-2 competitive & market-grounding routes/{b=NR} END{ if(b>a && a>0) exit 0; else {print "new routes not after prior children"; exit 1} }' "$AG"

echo "== AC-8: no dangling skill/agent references from the new files =="
# every skill id the new agent loads resolves to a dir with SKILL.md
for s in pm-agent-doctrine competitive-research feature-comparison-framing evidence-synthesis opportunity-assessment market-sizing; do
  test -f "$PM/skills/$s/SKILL.md" || { echo "DANGLING skill: $s"; exit 1; }
done
test -f "$PM/agents/competitive-analyst.md"

echo "ALL VALIDATION CHECKS PASSED"
```

Also expected to pass: `hero check` reports no new dangling-reference or
convention warnings in `domains/pm/`, and `hero install --dry-run` (or the
pack's install path) regenerates the harness copies without error.

## Boundaries

- **Content only, `domains/pm/` only.** No Go, no engine, no `internal/`, no CLI
  or MCP surface. No new slash command — opportunity/sizing surface *through*
  `product-strategist`; competitive teardown is a direct agent invocation.
- **Do NOT hand-edit installed harness copies** (`.claude/`, `.agents/`,
  `.codex/`). Author pack source; `hero install` regenerates (tripwire
  `harness-changes-cover-all-targets`).
- **Do NOT re-author the child-#1 skills.** `competitive-research` and
  `feature-comparison-framing` already exist and are owned by child #1; this
  child only *loads* them. Do not modify them.
- **Do NOT touch the canonical AGENTS.md routing table** or any prior child's
  Wave-2 subsection. Append only, below the marker, after prior children.
- **Do NOT undo child #1's `product-strategist` load-list additions.** The
  sharpen is additive to its Startup list and stance.
- **Not in scope:** the other `pm-pack-completion` children (experiment stage,
  adversarial critics, exec-narrative, discovery-framing, remaining
  roles/scrubbers). Live web-fetch *tooling* is not built here — the agent
  *declares* `webfetch: allow` and instructs retrieval; wiring a specific search
  provider is out of scope.
- **Not a positioning-canvas / GTM-tiering skill** — those are Wave-3 audit items
  (April Dunford canvas, launch tiering); this child ships only opportunity
  assessment + market sizing.

## Risks

- **AGENTS.md merge seam with child #1.** Both edit the file; the reciprocal
  `conflicts-with` marks them non-concurrent. If child #1 hasn't landed, the
  Wave-2 marker region and the two foundation skills won't exist — this child
  depends-on child #1 and must not start until it's `completed`.
- **Skill-shape drift.** The two new SKILL.md files must match the existing
  `metadata.audience` / `purpose` frontmatter convention so `hero check` and the
  pack's skill loader recognize them. Model on `competitive-research/SKILL.md`.
- **Over-authoring the agent.** Keep `competitive-analyst` scoped to teardown +
  matrix + positioning; delegation is "none." Resist adding metrics/experiment
  responsibilities that belong to sibling Wave-2 agents.

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|-----------|--------|------|
| AC-1 | `competitive-analyst.md` exists with valid frontmatter (`name`, `mode: subagent`, `webfetch: allow`) and retrieval-augmented-never-model-memory doctrine, memory-only teardown named as a refused anti-pattern | DONE | `domains/pm/agents/competitive-analyst.md` frontmatter lines 1-14; "The retrieval doctrine — your spine" section states "Retrieval-augmented, never model-memory" and "A memory-only teardown is an anti-pattern you refuse" / "a teardown without live data is a teardown of last year's market." Validation AC-1/AC-2 block PASSED. |
| AC-2 | Load-list loads exactly `pm-agent-doctrine`, `competitive-research`, `feature-comparison-framing`, `evidence-synthesis`, all on disk | DONE | `## Startup` lists the four and "No other skills." grep loop for all four PASSED; AC-8 confirms each resolves to a `SKILL.md`. |
| AC-3 | `opportunity-assessment/SKILL.md` with Cagan 10-question assessment (incl. go-no-go) + single-challengeable-assumption discipline | DONE | `domains/pm/skills/opportunity-assessment/SKILL.md` "The Cagan 10-question assessment" (Q1-Q10, Q10 = go/no-go) + "Single-challengeable-assumption discipline" section. grep for 10-question / go-no-go / single-challengeable PASSED. |
| AC-4 | `market-sizing/SKILL.md` with TAM/SAM/SOM (one challengeable assumption per step) + top-down↔bottom-up convergence flagging divergence | DONE | `domains/pm/skills/market-sizing/SKILL.md` "TAM / SAM / SOM — one challengeable assumption per step" + "Top-down ↔ bottom-up convergence" (flag divergence, do not average). grep for TAM/SAM/SOM, top-down↔bottom-up, diverg PASSED. |
| AC-5 | `product-strategist` gains `opportunity-assessment` + `market-sizing` and a defensible-sizing stance, preserving child #1 loads (`pm-agent-doctrine`, `outcomes-over-outputs`, `risk-surfacing`) | DONE | `domains/pm/agents/product-strategist.md` Startup gains 2 bullets; step 4 gains "Demand a defensible size before you commit the bet." Child-#1 preservation grep loop PASSED; git diff shows 0 deletions. |
| AC-6 | AGENTS.md new routes appended below `<!-- WAVE-2 ROUTES -->` after prior children; canonical table + prior children byte-for-byte unchanged | DONE | `domains/pm/AGENTS.md` new `#### Wave-2 competitive & market-grounding routes` after the `story-detail-and-intake-scrubber-backing` block. awk ordering check PASSED; `git diff --numstat` = 18 insertions / 0 deletions. |
| AC-7 | `competitive-analyst` added to Agents Reference; both skills to Skills Reference; every referenced agent/skill resolves | DONE | AGENTS.md "PM Wave-2 competitive & market:" bullets in both Agents Reference and Skills Reference. grep for all three refs PASSED. |
| AC-8 | Every skill/agent id referenced by the new files resolves to an on-disk `domains/pm/` artifact | DONE | Validation AC-8 loop over all 6 skill dirs + competitive-analyst.md PASSED; `hero check` surfaces no new pm dangling-reference warning. |

### Changes

| # | Changes item | Status | Note |
|---|--------------|--------|------|
| 1 | New agent `domains/pm/agents/competitive-analyst.md` (retrieval doctrine spine, Startup load-list, When invoked, Workflow, Anti-patterns, Default output) | DONE | Created, 75 lines, frontmatter modelled on `experiment-designer.md` (`mode: subagent`, `temperature: 0.1`, `color: secondary`, `webfetch: allow`, `task."*": deny`, `skill."*": allow`). |
| 2 | New skill `domains/pm/skills/opportunity-assessment/SKILL.md` (Cagan 10-Q, single-assumption discipline, copy-paste artifact) | DONE | Created; `metadata.audience: product-strategist`, `purpose: framework-guidance`; cross-refs `market-sizing` (Q3) + `competitive-research` (Q4). |
| 3 | New skill `domains/pm/skills/market-sizing/SKILL.md` (TAM/SAM/SOM one-assumption-per-step, top-down↔bottom-up convergence, copy-paste artifact) | DONE | Created; same frontmatter convention; divergence-not-averaging rule with ~2–3× threshold. |
| 4 | Sharpen `domains/pm/agents/product-strategist.md` (add 2 load-list skills + defensible-sizing stance, preserve child #1 loads) | DONE | Additive edit: 2 Startup bullets + step-4 paragraph. 0 deletions in git diff; child #1 loads intact. |
| 5 | Extend `domains/pm/AGENTS.md` additively below the marker (new Wave-2 subsection + Agents Reference + Skills Reference bullets) | DONE | 18 insertions / 0 deletions; canonical table and prior children untouched. |

### Exercise-the-feature check

- [x] Spec `## Validation` bash block run verbatim from repo root — **ALL VALIDATION CHECKS PASSED** (AC-1 through AC-8).
- [x] `git diff --numstat` confirms additions-only on AGENTS.md (18/0) and product-strategist (no deletions) — AC-5 preservation + AC-6 additive diff.
- [x] `hero check` runs clean for `domains/pm/` — no new dangling-reference or convention warning attributable to these files (sole failing check is pre-existing workspace-wide `status-truthfulness`, unrelated).
- [x] `hero install project <tmp> --domain pm --dry-run` completes exit 0 with no frontmatter/parse error — the pack loads the new source cleanly.
- [~] Live agent/skill invocation not exercised: the installer's current install-state selection copies only the canonical PM agent set, so Wave-2 agents/skills (including already-shipped `experiment-designer` / `competitive-research`) are not selected into the harness copy. This is identical to prior completed Wave-2 sibling work and orthogonal to this content-only spec (wiring the installer's Wave-2 selection is out of scope). Structural equivalence to shipped siblings + clean pack load is the available evidence.

### Excellence Bar self-check

- The retrieval doctrine is authored as the agent's *spine* (its own top-level section before Startup), with a hard refusal and the "teardown of last year's market" line — not a footnote. It inherits and reinforces `competitive-research`'s rule rather than restating it thinly.
- Both skills carry a worked discipline, a copy-paste artifact template, an anti-patterns section, and cross-references in the exact house style of `competitive-research`/`feature-comparison-framing` — no stubs; 82 and 81 lines respectively, in line with the shipped `competitive-research` (93 lines).
- The `product-strategist` sharpen is surgical (2 bullets + 1 paragraph, no workflow restructure) and states *why* an unsized bet is theater, tying back to the file's existing "bet without an outcome is theater" voice.
- AGENTS.md edits mirror the prior children's subsection prose shape (header + rationale paragraph + table) and are strictly additive, keeping the merge seam with child #1 clean.
