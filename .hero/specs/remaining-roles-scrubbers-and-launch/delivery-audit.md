# Delivery audit — remaining-roles-scrubbers-and-launch

**Audited:** working tree vs `HEAD` (`git diff HEAD` + untracked NEW files under `domains/pm/`)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1 six agent files with valid frontmatter — all six exist under `domains/pm/agents/`; each has `name`, `description`, `mode: subagent`, `permission` block. Validation AC-1 loop green.
- [✓] AC-2 every agent loads `pm-agent-doctrine` + designed skills, zero dangling — every skill token resolves to a real dir (`domains/pm/skills/` has all 14 referenced skills; verified `ls`). No-dangling check prints "ok — all designed skill refs resolve".
- [✓] AC-3 both scrubbers report-only — `edit: deny` in `stale-roadmap-scrubber.md:15` and `ambiguous-story-scrubber.md:15`. (discovery-reviewer also `edit: deny` per critic doctrine.)
- [✓] AC-4 roadmap + stories concerns appended below `#11 APPEND POINT` — `scrub.md` diff adds both `### Concern: roadmap` and `### Concern: stories` strictly inside the append region, above `## Dispatch`. Line-number guard green (both > marker).
- [✓] AC-5 child #5 intake block byte-for-byte unchanged — intake-block hash matched `37b0415…` with **no WARN emitted** (hard assertion). Diff shows the intake concern block only as unchanged context; the sole above-block edit is the frontmatter `description:` line (in-region per Change #7).
- [✓] AC-6 `launch-gtm-tiering/SKILL.md` — real tier 1/2/3 rubric table (4 dimensions), reading guidance, five-phase checklist with per-tier collapse, coverage table, worked SSO example, anti-patterns, cross-refs.
- [✓] AC-7 `commands/launch.md` — detects tier via `launch-gtm-tiering`, emits tier-scoped five-phase checklist, routes to `stakeholder-communicator` with per-phase owners noted, passes `$ARGUMENTS`.
- [✓] AC-8 new child-#11 subsection below last Wave-2 subsection; canonical table + prior Wave-2 routes byte-unchanged — Wave-3 subsection at `AGENTS.md:165` (after exec-narrative subsection at 148). Diff is purely additive at line 162; no canonical-table (line 21+) or prior-subsection lines touched. Roster line edits (271/304) are AC-9 scope, not the routing table.
- [✓] AC-9 rosters updated — PM Wave-3 agents bullet (6 agents), Launch/GTM skills bullet (`launch-gtm-tiering`), `/launch` added to PM command list, `/scrub` roster line rewritten to shipped state.
- [✓] AC-10 confined to `domains/pm/` — AC-10 check green; no `.claude/`/`.agents/`/`.codex/` mirror edited.

## Changes
- [✓] 1 `epic-framer.md` — coherent-bet framing; loads `pm-agent-doctrine`, `epic-framing`, `story-writing-invest`, `dependency-mapping`; edit: allow. 1019 words, 6 sections.
- [✓] 2 `risk-curator.md` — scenario+indicator+response; loads doctrine + `risk-surfacing`, `assumption-testing`, `evidence-synthesis`; edit: allow. 907 words.
- [✓] 3 `portfolio-curator.md` — theme balance + capacity-vs-ambition; loads doctrine + `outcomes-over-outputs`, `roadmap-framing`, `prioritization-frameworks`. Does **NOT** load `capacity-planning` (explicit note at line 29 explains the §C.2 substitution). edit: allow.
- [✓] 4 `discovery-reviewer.md` — adversarial rigor critic; loads doctrine + `opportunity-solution-trees-torres`, `discovery-interview-design`, `assumption-testing`, `evidence-synthesis`; edit: deny (report-only). 899 words, 7 sections.
- [✓] 5 `stale-roadmap-scrubber.md` — report-only `/scrub roadmap`; loads doctrine + `roadmap-framing`, `outcome-drift`; edit: deny.
- [✓] 6 `ambiguous-story-scrubber.md` — report-only `/scrub stories`; loads doctrine + `story-writing-invest`, `acceptance-criteria-ears`; edit: deny.
- [✓] 7 `scrub.md` extended — two concerns appended in region only; frontmatter description → past tense; intake block byte-unchanged (hash match).
- [✓] 8 `launch-gtm-tiering/SKILL.md` — see AC-6.
- [✓] 9 `launch.md` — see AC-7.
- [✓] 10 AGENTS.md Wave-3 subsection — see AC-8.
- [✓] 11 three rosters updated — see AC-9.

## Open items
None. No PARTIAL / SKIPPED / BLOCKED rows in the ledger.

## Audit notes
- Full `## Validation` bash block re-run verbatim from repo root → **ALL VALIDATION CHECKS PASSED**; the AC-5 intake-block hash guard matched with no WARN (hard byte-for-byte assertion, not a warning).
- The 6 agents are genuinely distinct and substantive (721–1019 words, 6–7 sections, anti-patterns each) — no stubs, no dups.
- Working tree also shows `.hero/` projection churn (`NEXT.md`, `QUEUE.md`, `events.log`, `drive/`) and a spec relocation (flat `remaining-roles-scrubbers-and-launch.md` → `remaining-roles-scrubbers-and-launch/spec.md`). These are Hero workspace/handoff bookkeeping expected to travel with the commit — not authored pack content and not scope drift. All authored deliverable content is confined to `domains/pm/`.
- Minor (non-blocking): delegation targets named in agent bodies (`metrics-analyst`, `prioritization-strategist`) are agent references, outside AC-2's skill-resolution scope — not audited as dangling. No AC depends on them.
