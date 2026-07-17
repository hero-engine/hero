# Delivery audit — prd-editor-comms-backing

**Audited:** working tree vs `HEAD` (new files untracked, inspected on disk)
**Verdict:** SHIP
**Surface:** noteworthy (1 scope note)

## Acceptance criteria
- [✓] AC-1 `pitch-author.md` Shape Up specialist — `domains/pm/agents/pitch-author.md:21-30` Startup loads `pm-agent-doctrine`, `pitch-writing-shape-up`, `prd-structure` (+4 shipped ids); `:73-80` enforcement gate refuses `status: review` with empty Appetite or No-Gos.
- [✓] AC-2 `stakeholder-communicator.md` — `:21-30` Startup loads `pm-agent-doctrine`, `outcomes-over-outputs`, `stakeholder-communication`, `release-notes-writing` (+3); `:47-55` produces exec/customer/internal cuts.
- [✓] AC-3 `stakeholder-communication/SKILL.md` — four-audience table `:27-32`; "so what" pressure-test `:36-48`; working-backwards `:50-54` names PR-FAQ/6-pager and defers the format to `exec-narrative` rather than duplicating it.
- [✓] AC-4 `release-notes-writing/SKILL.md` — Customer-facing shape `:26-35`, Internal update shape `:38-46`.
- [✓] AC-5 `/standup` → `stakeholder-communicator` loading `stakeholder-communication` + `cross-domain-graph-query`; composes from intra-cycle graph changes (`standup.md:4,14-21`).
- [✓] AC-6 `/interview` → `discovery-researcher` loading `discovery-interview-design` (`interview.md:4`); both ids present on disk.
- [✓] AC-7 `/pitch` repointed — `pitch.md:4` now `Route to `pitch-author`.`; v1.5 parenthetical removed; grep for `v1.5` empty.
- [✓] AC-8 `/release-notes` repointed — `release-notes.md:4` now `Route to `stakeholder-communicator`, loading `release-notes-writing`.`; "ship v1.5" removed.
- [✓] AC-9 Four routes appended — `AGENTS.md:97` "#### Wave-2 PRD Editor & comms routes" sits strictly after #7 (line 82) and below the marker (line 62). No removed lines in the canonical table or #6/#7 bodies; the only two `-` lines are the additive Reference-roster replacements (Commands PM list, Skills Writing list), explicitly permitted.
- [✓] AC-10 Registrations — Skills Writing line adds both skills (`:217`); new Agents "PM Wave-2 PRD Editor & comms" bullet (`:209`); Commands PM list adds `/interview` + `/standup` (`:199`).
- [✓] AC-11 `prd-author.md` untouched — `git diff HEAD -- domains/pm/agents/prd-author.md` empty.
- [✓] AC-12 No dangling refs — every Startup id resolves under `domains/pm/skills/` (or core: `spec-format`, `kickoff-prompt`); `discovery-researcher` present. `shape-up-cadence` absent from pitch-author's load list; `exec-narrative` appears only as prose cross-ref, never in a Startup block.

## Changes
- [✓] 1 `agents/pitch-author.md` — frontmatter matches shipped shape (`mode: subagent`, `temperature: 0.1`, permission block); enforcement gate; writes `.hero/planning/prds/<slug>/spec.md` with `prd_template: pitch`.
- [✓] 2 `agents/stakeholder-communicator.md` — audience-cut agent; doctrine 1+3 posture; distinct from pitch-author and prd-author.
- [✓] 3 `skills/stakeholder-communication/SKILL.md` — real, specific; four-cut table, so-what test, PR-FAQ cross-ref, anti-patterns.
- [✓] 4 `skills/release-notes-writing/SKILL.md` — customer + internal shapes, shipped-status-from-graph, anti-patterns.
- [✓] 5 `commands/standup.md` — thin router, ends `Request: $ARGUMENTS`.
- [✓] 6 `commands/interview.md` — thin router, ends `Request: $ARGUMENTS`.
- [✓] 7 `commands/pitch.md` repointed to `pitch-author`.
- [✓] 8 `commands/release-notes.md` repointed to `stakeholder-communicator`.
- [✓] 9 `AGENTS.md` Wave-2 subsection appended after #6/#7.
- [✓] 10 `AGENTS.md` Reference sections registered additively.
- [✓] 11 `prd-author.md` untouched.

## Open items
None. All rows carry real evidence.

## Audit notes
- Full `## Validation` bash block ran verbatim from repo root → `ALL CHECKS PASSED` (exit 0).
- **Substance (not thin):** `pitch-author` (Shape Up, appetite/rabbit-holes/no-gos enforcement gate) and `stakeholder-communicator` (audience-cut, so-what discipline) are genuinely distinct agents, both loading `pm-agent-doctrine`, neither a near-duplicate of `prd-author`. Both new skills are full-length with concrete tables/examples, not stubs.
- **Scope note (why noteworthy):** working tree also contains changes outside `domains/pm/` + the initiative spec dir — `.hero/drive/pm-pack-completion.json`, `.hero/drive/trust/`, and the refreshed projected handoff files (`.hero/NEXT.md`, `.hero/QUEUE.md`, `.hero/events.log`). These are expected `/drive`-run orchestration + handoff machinery, not delivery scope drift into product source. The initiative spec itself moved from a flat file (`prd-editor-comms-backing.md`, now `D`) to a directory (`prd-editor-comms-backing/`), which is the named spec-dir change. No unexpected source files touched.
