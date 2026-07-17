# Delivery audit — exec-narrative-and-evidence-synthesis

**Audited:** `git diff HEAD -- domains/pm/` + untracked `domains/pm/skills/{prfaq-writing,exec-narrative}/SKILL.md`
**Verdict:** SHIP
**Surface:** noteworthy (1 benign note)

## Acceptance criteria
- [✓] AC1 prfaq-writing exists, PR/FAQ + dragons + reasoning-over-prose — `domains/pm/skills/prfaq-writing/SKILL.md:11-13` (press release + FAQ, reasoning-is-the-deliverable frame), `:37-44` (FAQ as load-bearing half, dragon defined). Valid frontmatter (name/description/metadata).
- [✓] AC2 exec-narrative all six 6-pager sections + "so what?" + prose-exposes-gaps — `SKILL.md:24-33` (Intro/Goals/Tenets/State of the Business/Lessons Learned/Strategic Priorities, load-bearing ordering), `:35-44` ("so what?" test with fail/survive table), `:11` (prose forces connective tissue bullets skip).
- [✓] AC3 discovery-researcher gains compare/verbatim/outlier; child #1's 6 loads intact — diff adds stance para (line 19), synthesize-step para (line 109), 3 anti-patterns (lines 137-139); cross-refs doctrine 3. All six startup loads grep-present; zero deletions in diff.
- [✓] AC4 evidence-synthesis extended additively; every prior H2 survives — diff is purely `+` (3 new H2 sections + 2 anti-patterns before `## Anti-patterns`); prior-content guard confirms all 8 pre-existing H2 headings present; no existing line removed or reworded.
- [✓] AC5 stakeholder-communicator gains prfaq-writing + exec-narrative; 7 loads intact — diff inserts 2 skills after `stakeholder-communication` load; all seven prior loads grep-present; zero deletions.
- [✓] AC6 AGENTS.md new subsection after marker + competitive routes; skills in Skills Reference; table + prior subsections untouched — new subsection at line 148 (marker line 62 → competitive line 132 → new 148 → "When routing" close 165); diff is additions only, no edits above marker or inside prior subsections; Skills-Reference bullet added at line 279.
- [✓] AC7 no dangling refs — both skill dirs exist on disk; `stakeholder-communication` (`SKILL.md:54,69`) and `stakeholder-communicator` forward-refs to `exec-narrative` now resolve; every new cross-ref names an existing skill.
- [✓] AC8 all authoring under domains/pm/; no Go, no installed harness copy — `git diff --name-only HEAD` touches only `domains/pm/{AGENTS.md,agents/discovery-researcher.md,agents/stakeholder-communicator.md,skills/evidence-synthesis/SKILL.md}` + 2 untracked new skills. No `.go`, `.claude/`, `.codex/`, `.agents/` touched (verified over tracked+untracked).

## Changes
- [✓] 1. New `prfaq-writing/SKILL.md` — 66 lines, press-release shape with quotes-as-placeholders, FAQ/dragons half, kill-switch framing, 6 anti-patterns, 5 cross-refs.
- [✓] 2. New `exec-narrative/SKILL.md` — 66 lines, six sections section-by-section, paragraph-level "so what?" table, 6-pager-vs-cut guidance, 6 anti-patterns, 5 cross-refs.
- [✓] 3. Sharpen discovery-researcher.md — 3 disciplines in stance + synthesize step + 3 anti-patterns; loads preserved.
- [✓] 4. Extend evidence-synthesis.md — 3 additive H2 sections + 2 anti-patterns; guard-clean.
- [✓] 5. Wire stakeholder-communicator.md — 2 skills added, 7 preserved.
- [✓] 6. Wire AGENTS.md — new Wave-2 subsection (2 routes) + Skills-Reference bullet; canonical table + prior subsections byte-unchanged.

## Open items
- None blocking. Ledger has no PARTIAL/SKIPPED/BLOCKED rows.

## Audit notes
- **Verbatim `## Validation` block does not print `ALL CHECKS PASSED` as-run in the live working tree** — it exits at the AC8 guard because three harness-maintained files (`.hero/NEXT.md`, `.hero/QUEUE.md`, `.hero/events.log`) are dirty from the running `/drive`. This exactly matches what the ledger disclosed: the block goes green only after those three are set aside (`git stash`). Confirmed independently that with the three harness-churn files excluded, no non-`domains/pm/` product file is modified and AC1–AC7 all pass under `set -e`. The churn is Hero projection/log output, not this delivery's product — so the SHIP holds — but the Validation block as written is not self-cleaning against harness churn. Benign; noted for honesty.
- **Substance is genuine, not thin.** `prfaq-writing` is real Amazon working-backwards: mock press release (headline/sub-head/problem/solution/quotes-as-placeholders/CTA) + anticipated FAQ split into customer + internal halves, "a dragon is any question whose honest answer threatens the bet," explicit reasoning-over-copy frame and don't-build kill-switch. `exec-narrative` is a real 6-pager: all six sections with load-bearing ordering rationale, a paragraph-level "so what?" fail/survive worked table, and prose-exposes-gaps as the format's whole thesis. Both cross-reference `pm-agent-doctrine` doctrine 3/doctrine 1 rather than restating.
- **Boundary discipline held.** `evidence-synthesis` and both agent edits are strictly additive (no `-` lines except pure insertions among existing bullets). AGENTS.md canonical routing table and every prior child's Wave-2 subsection are byte-unchanged; new content is strictly after the last prior child.
- **Minor imprecision (not a defect, out of scope to fix):** `stakeholder-communication/SKILL.md:54` still says the full "PR-FAQ / narrative mechanics ... live in `exec-narrative`," whereas PR-FAQ mechanics now live in the dedicated `prfaq-writing` skill. The pointer still resolves (exec-narrative exists and cross-refs prfaq-writing as its sibling), so no ref dangles; editing that skill was outside this child's scope.
