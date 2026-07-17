# Delivery audit — competitive-and-market-grounding

**Audited:** working tree vs `HEAD` (new files untracked); `git diff HEAD -- domains/pm/`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] **AC-1** retrieval-augmented-never-model-memory doctrine, memory-only teardown refused — `domains/pm/agents/competitive-analyst.md:21-31`. Doctrine is the agent's first body section ("The retrieval doctrine — your spine"), states "Retrieval-augmented, never model-memory. This is the whole point of the role, not a footnote" (line 23), and line 31 names a memory-only teardown "an anti-pattern you refuse" with the "teardown of last year's market" refusal. Frontmatter `name: competitive-analyst`, `mode: subagent`, `webfetch: allow` (lines 2,4,13). Genuinely load-bearing, not a passing mention.
- [✓] **AC-2** load-list = exactly the four skills — `competitive-analyst.md:35-41` lists `pm-agent-doctrine`, `competitive-research`, `feature-comparison-framing`, `evidence-synthesis` + "No other skills." All four resolve on disk (verified).
- [✓] **AC-3** Cagan 10-Q + single-challengeable-assumption — `opportunity-assessment/SKILL.md:20-43`; Q1-Q10 numbered walkthrough, Q10 = go/no-go (line 33); dedicated "Single-challengeable-assumption discipline" section (lines 35-43).
- [✓] **AC-4** TAM/SAM/SOM + top-down↔bottom-up convergence — `market-sizing/SKILL.md:20-46`; one-challengeable-assumption-per-step rule (line 28), convergence section flags divergence and explicitly says "do not average" (line 42), ~2-3× threshold (line 46).
- [✓] **AC-5** product-strategist gains 2 skills + defensible-sizing stance, child #1 loads preserved — `git diff HEAD`: 4 insertions / 0 deletions. Adds `opportunity-assessment` + `market-sizing` (lines 31-32) and a "Demand a defensible size before you commit the bet" paragraph (line 81). `pm-agent-doctrine`, `outcomes-over-outputs`, `risk-surfacing` all still present (lines 24,25,27). Zero deletions of any child #1 line.
- [✓] **AC-6** AGENTS.md additive below marker — `git diff HEAD`: 18 insertions / 0 deletions. New `#### Wave-2 competitive & market-grounding routes` subsection appended after the `story-detail-and-intake-scrubber-backing` block (awk ordering check PASSED); canonical routing table and all prior children's subsections byte-unchanged (0 deletions).
- [✓] **AC-7** references added — Agents Reference gains "PM Wave-2 competitive & market" bullet for `competitive-analyst`; Skills Reference gains bullet for `opportunity-assessment` + `market-sizing`. All resolve.
- [✓] **AC-8** no dangling refs — all 6 load/named skills resolve to a `SKILL.md`; `competitive-analyst.md` exists. Extra cross-referenced ids in the new skills (`assumption-testing`, `metrics-design`, `metric-rca`, `outcomes-over-outputs`) also all resolve — zero dangling references anywhere in the pack.

## Changes

- [✓] 1. New agent `competitive-analyst.md` — 75 lines; frontmatter modelled on `experiment-designer.md`; retrieval-doctrine spine + Startup + When invoked + Workflow + Anti-patterns + Default output.
- [✓] 2. New skill `opportunity-assessment/SKILL.md` — 82 lines; Cagan 10-Q, single-assumption discipline, copy-paste artifact, anti-patterns, cross-refs. Correct `metadata.audience: product-strategist` / `purpose: framework-guidance`.
- [✓] 3. New skill `market-sizing/SKILL.md` — 81 lines; TAM/SAM/SOM one-assumption-per-step, top-down↔bottom-up convergence (divergence-not-averaging, ~2-3× threshold), copy-paste artifact. Same frontmatter convention.
- [✓] 4. Sharpen `product-strategist.md` — additive (4/0), 2 Startup bullets + step-4 paragraph, no workflow restructure.
- [✓] 5. Extend `AGENTS.md` — additive (18/0), new Wave-2 subsection + Agents/Skills Reference bullets.

## Validation

Spec's full `## Validation` bash block run verbatim from repo root: **ALL VALIDATION CHECKS PASSED** (AC-1 through AC-8).

## Substance judgment (skills thin vs. sufficient)

The two skills (~81-82 lines) are **sufficient, not thin**. Each carries a worked discipline with concrete challengeable-assumption examples (the 40%-of-mid-market and the SAM/SOM multiplier lines), a copy-paste artifact template, an anti-patterns section, and cross-references — in the same shape and length band as the shipped `competitive-research` (93 lines). The single-challengeable-assumption doctrine and the divergence-not-averaging rule are stated as enforceable rules, not gestured at.

## Audit notes

- Ledger's one `[~]` item ("live agent/skill invocation not exercised") carries a **concrete** reason: the installer's current Wave-2 selection copies only the canonical PM agent set, identical to prior shipped Wave-2 siblings; wiring installer Wave-2 selection is out of scope for this content-only spec. Structural equivalence + clean pack load is the appropriate evidence for a content-only delivery. Not a downgrade.
- Working tree also carries `.hero/NEXT.md`, `.hero/QUEUE.md`, `.hero/events.log`, `.hero/drive/*`, and the stub→directory move of the spec (`D competitive-and-market-grounding.md` + new `competitive-and-market-grounding/`). These are expected Hero session/handoff/drive-run bookkeeping from a `/drive` run — not deliverable scope drift. The deliverable itself is cleanly scoped to `domains/pm/` + the initiative spec dir.
