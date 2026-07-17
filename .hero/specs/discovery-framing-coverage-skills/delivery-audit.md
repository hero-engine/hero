# Delivery audit — discovery-framing-coverage-skills

**Audited:** `git diff HEAD` + 7 untracked `domains/pm/skills/*/SKILL.md` (working tree)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — seven SKILL.md files exist, non-empty, in the 90–145 line band. Validation `test -s` + `wc -l` guards pass for all seven.
- [✓] AC-2 — valid YAML frontmatter: `name:` == slug, one-sentence `description:`, `metadata.audience` + `metadata.purpose` present. Confirmed by reading all seven frontmatter blocks; grep guards pass.
- [✓] AC-3 — signature sections (`## What I do`, `## When to use me`, `## Anti-patterns`, `## Cross-references`) present in each; no TODO/TBD/FIXME/placeholder/stub markers. Line counts 90–95, real content throughout.
- [✓] AC-4 — `personas-and-journey-maps`: "Marketing Mary" demographic-fiction vs. evidence-grounded persona (goal/context/pain/evidence source) contrast at lines 24–38; five-column journey grid (Stage → Action → Thought/feeling → Pain → Opportunity) at lines 56–61.
- [✓] AC-5 — `jtbd-job-stories`: `When [situation], I want [motivation], so [outcome]` shape (line 23); context-over-persona section (lines 40–44); explicit vs-INVEST comparison table (lines 69–76) cross-referencing `story-writing-invest`.
- [✓] AC-6 — `positioning-canvas`: Dunford's five components in order — competitive alternatives → unique attributes → value → target customers → market category (lines 24–42), worked example (lines 44–54).
- [✓] AC-7 — `story-mapping`: Patton backbone (lines 22–34), "The walking skeleton — the first slice" (lines 36–41), release slicing with horizontal cuts (lines 43–50).
- [✓] AC-8 — `hill-chart-reasoning`: "Position = unknowns remaining, NOT percent done" (lines 36–40); uphill (figuring-out) vs. downhill (executing) framing (lines 21–34).
- [✓] AC-9 — `domain-glossary-maintenance`: "Where it lives" → `.hero/knowledge/domain-glossary.md` (lines 20–22); five-field term entry shape (lines 30–46); PM/eng reconciliation (lines 60–62).
- [✓] AC-10 — `product-vision-writing`: vision → strategy → roadmap → initiatives ladder (lines 20–36); "Rooted at the OST outcome" (lines 42–44); six-clause one-page template (lines 46–56); cross-refs `outcomes-over-outputs` + `roadmap-framing`.
- [✓] AC-11 — single "Discovery & framing coverage (Wave-3)" bullet inserted before the Core bullet in `domains/pm/AGENTS.md` (line 280). `git diff HEAD` = exactly 1 insertion. Natural Language Routing table and all four Wave-2 bullets byte-unchanged.
- [✓] AC-12 — no dangling skill cross-refs. Every backticked skill-ref in the seven files resolves to a shipped `domains/pm/skills/` dir (or `core/skills/convention-writing`) or to one of the seven authored here. All 20 advisory `POSSIBLE dangling` tokens are consumer-agent role names, all verified real (see note).

## Changes

- [✓] 1 `personas-and-journey-maps/SKILL.md` — 94 lines; evidence vs. fiction personas, proto/validated, build-from-evidence, journey grid, OST handoff.
- [✓] 2 `jtbd-job-stories/SKILL.md` — 95 lines; job-story shape, context-over-persona, four forces + force-ranking, vs-INVEST table.
- [✓] 3 `positioning-canvas/SKILL.md` — 92 lines; Dunford five components in order, worked example, category-as-lever, staleness triggers.
- [✓] 4 `story-mapping/SKILL.md` — 92 lines; backbone + tasks, walking skeleton, release slicing, session process, worked example.
- [✓] 5 `hill-chart-reasoning/SKILL.md` — 90 lines; unknowns-not-%, uphill/downhill, scopes-not-tasks, check-in ritual, vs-burndown.
- [✓] 6 `domain-glossary-maintenance/SKILL.md` — 90 lines; `.hero/knowledge/` home, five-field entry, add/retire, review cadence, PM/eng reconciliation.
- [✓] 7 `product-vision-writing/SKILL.md` — 90 lines; vision→strategy→roadmap ladder, OST root, six-clause template, writing process, decision test.
- [✓] 8 `domains/pm/AGENTS.md` Skills Reference — single bullet insertion, routing table + Wave-2 region byte-unchanged.

## Audit notes

- **Substance is real, not filler.** Spot-checked hard per the task: jtbd carries the `When…I want…so` shape, a good/weak example, the four forces (push/pull/anxiety/habit), and a proper vs-INVEST table. positioning-canvas runs Dunford's five components strictly in order with a worked example and the category-as-lever insight. story-mapping has the Patton backbone, the walking-skeleton first slice, and horizontal release slicing. hill-chart frames position as unknowns-remaining (not %-done) with the stuck-uphill alarm and a vs-burndown contrast. Each file mirrors the shipped skill voice (first-person "What I do", worked examples, candid anti-patterns) and adds depth beyond the AC floor. None reads as generic padding.
- **`cycle-planner` (hill-chart audience + prose) is not a shipped agent.** The pm agent roster has `cycle-planning` (a skill) but no `cycle-planner` agent. This traces directly to the spec, which mandates `metadata.audience: pm-delivery-lead, cycle-planner, roadmap-curator` for this skill (spec line 176). The engineer followed the spec verbatim; it is a spec-level naming choice, not a delivery defect, and `metadata.audience` is descriptive, not a resolvable cross-ref. Non-blocking. Every *other* advisory token (`discovery-researcher`, `product-strategist`, `story-writer`, `competitive-analyst`, `pm-delivery-lead`, `roadmap-curator`, `pm-reviewer`, `convention-author`) resolves to a real agent under `domains/pm/agents/` or `core/agents/`, and `convention-writing` resolves to `core/skills/convention-writing/`.
- **Scope held.** Only `domains/pm/` content authored: 7 new skill dirs + the one mandated AGENTS.md bullet. No Go, no schema, no `hero.json`, no other domain. Other working-tree changes (`.hero/NEXT.md`, `.hero/QUEUE.md`, `.hero/events.log`, `.hero/drive/*`, spec file→dir conversion) are Hero workspace/projection state, not authored deliverable content — expected and benign.
- **Validation block run verbatim → `VALIDATION OK`** with only advisory `POSSIBLE dangling` lines, each hand-verified above.
