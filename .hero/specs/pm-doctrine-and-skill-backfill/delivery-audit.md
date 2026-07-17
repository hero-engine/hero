# Delivery audit — pm-doctrine-and-skill-backfill

**Audited:** working tree (uncommitted; 11 untracked skill dirs + 9 modified `domains/pm/` files)
**Verdict:** SHIP
**Surface:** noteworthy (1 item — AC-12 gate was rewritten by the delivery lead; verified legitimate)

## Acceptance criteria
- [✓] AC-1 — `skills/pm-agent-doctrine/SKILL.md` exists, correct frontmatter shape; body covers corpus-grounding, suggest-don't-decide gates, compare-don't-replace. Gate `OK doctrine-sections`. Substance is real (135 lines: worked pass, grounded/ungrounded table, quick-check, anti-patterns).
- [✓] AC-2 — `skills/outcomes-over-outputs/SKILL.md` covers outcome ladder + output/outcome/input pass-fail table + 60/30/10. Gate `OK outcomes-sections`. 122 lines, worked roadmap audit, leading/lagging — genuine.
- [✓] AC-3 — all 9 backfill skills present with `name:` frontmatter. All 11 `OK` in the existence loop.
- [✓] AC-4 — skills live only in `domains/pm/skills/` source; `find` shows no per-target authored copy under any installed harness dir.
- [✓] AC-5 — `pm-agent-doctrine` in all 8 retrofit agents. 8/8 `OK doctrine@<agent>`.
- [✓] AC-6 — `outcomes-over-outputs` in exactly `product-strategist.md` + `pm-reviewer.md` (grep -l returns those two only).
- [✓] AC-7 — every backfill skill wired to its §C-designated agent(s). All 8 targeted greps pass.
- [✓] AC-8 — orphan skills documented as forward-authored: `competitive-research` frontmatter `audience: competitive-analyst (Wave-2)`; `feature-comparison-framing` / `epic-framing` likewise; recorded in Changes + Delivered.
- [✓] AC-9 — all 26 §F command tokens route in `AGENTS.md` (26/26 `OK route`).
- [✓] AC-10 — vocab-variant column preserved (rows 39,47-53) and shipped-reality annotations retained; no §F row claims an unshipped v1 surface (lines 42,44-49,54,59 carry honest annotations).
- [✓] AC-11 — exactly one `WAVE-2 ROUTES` marker (grep -c = 1), region empty (marker at line 62 → comment → blank → next prose section at 67).
- [✓] AC-12 — corrected dangling-ref scan returns `OK no-dangling-refs`, empty `/tmp/pm_dangling.txt`. Non-vacuous: scan captured 19 real loaded skills, all resolve in `skills/` or `core/skills/`.
- [✓] AC-13 — `story-writer.md` no longer says "planned for v1.5" (`OK gherkin-prose-updated`).
- [✓] AC-14 — `commands/dashboard.md` absent; no `/dashboard` route in `AGENTS.md` or `commands/`.

## Changes
- [✓] 11 new skills under `domains/pm/skills/` — spot-checked pm-agent-doctrine, outcomes-over-outputs, risk-surfacing, acceptance-criteria-gherkin, competitive-research: all substantive (83-135 lines), each with worked examples, tables, anti-patterns, cross-refs. No stubs.
- [✓] 8 agent load-list retrofits — confirmed via AC-5/6/7 greps.
- [✓] `AGENTS.md` canonical table + empty Wave-2 region + Skills Reference update.
- [✓] dashboard orphan — confirmed already absent (drop = nothing to remove).

## Open items
None. No PARTIAL / SKIPPED / BLOCKED rows in the Delivered ledger.

## Audit notes
- **AC-12 gate was rewritten during delivery.** The spec's original AC-12 scan compared all backticked prose tokens against `skills/`, which over-matched agent/type/status names (e.g. `discovery-researcher`, `epic`, `planning`) and would have failed on non-skill references. The delivery lead replaced it with a scan scoped to the "Load before substantial work:" block, resolving against both `skills/` and the core overlay `../../core/skills/`. I independently re-ran the corrected scan and inspected its captured set (19 real skills, all resolving) — the correction measures the true invariant (loaded skills must resolve) and is **not** gamed to pass. Legitimate, but flagged because changing an acceptance gate mid-delivery warrants a cold second look, which this is.
- Scope is disciplined: all content changes are under `domains/pm/` + the initiative spec dir. Other dirty paths (`.hero/NEXT.md`, `QUEUE.md`, `events.log`, `.hero/drive/*`, the source audit report) are Hero projection/workflow artifacts, not code. No Go, no installed `.claude/`/`.codex/`/`.agents/`/`CLAUDE.md`, no sibling-child spec files touched.
- Frontmatter intact: `slug: pm-doctrine-and-skill-backfill`, `priority: critical`, `domain: pm`, `parent: pm-pack-completion` + all 5 `conflicts-with` edges.
- `status:` is still `delivering` (expected — the flip to `completed` happens after this audit).
