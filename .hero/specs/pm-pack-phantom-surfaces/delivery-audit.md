# Delivery audit — pm-pack-phantom-surfaces

**Audited:** `git diff 40a3b85` (working tree, uncommitted) over `domains/pm/`, `core/skills/`, `domains/engineering/`
**Verdict:** SHIP
**Surface:** noteworthy

Cold audit by a fresh auditor. Verified the Completion Ledger's DONE claims against the actual diff, the four grep gates from §Validation, and a freshly-built `hero` binary. No Go code changed; scope is content-only.

## Acceptance criteria

- [✓] AC1 — No AGENTS.md routing row targets an absent slash command — routing + vocabulary tables rewritten; every surviving target resolves: `/discover /roadmap /triage /refine /prd /pitch /metrics /prioritize /handoff /release-notes` ship in `domains/pm/commands/`; `/why /blocked /note /decide /retro /discover` ship in `core/commands/`. `/diagnose`/`/review` repointed to "invoke agent directly"; epic scaffolding correctly points at hand-authoring per `core/spec-types/epic.md` (file exists).
- [✓] AC2 — Only existing CLI commands/flags in every `domains/pm/` file — grep gate 1a empty; binary spot-checks pass: `hero agent events`, `hero spec new`, `hero spec set-owner`, `hero sync import`, `hero search --list --tag`, `hero list --status` all exist. Negatives confirmed: `hero new` (root) → "unknown command"; `hero spec new --type epic` → "unknown spec type"; `--kind`/`--themes`/`--owner` absent from search/list/queue.
- [✓] AC3 — Events use `hero agent events <valid-type>` or `hero_event` MCP — binary help confirms valid type set (spec_created, spec_updated, files_modified, decision_made, blocker_hit, delivery_complete); all `hero event ...` repointed; invalid `handoff` type → `spec_updated` / `delivery_complete`. No `hero event ` token survives (gate 1a empty).
- [✓] AC4 — Owner flip described exclusively via `hero spec set-owner` — handoff-coordinator:84, handoff command, handoff-protocol, AGENTS.md all flip via `hero spec set-owner <slug> engineering`; "raw frontmatter edit records no history" warning present (handoff-coordinator:87). Binary confirms the command exists and appends `owner_history`.
- [✓] AC5 — pm-delivery-lead `permission.task` allowlist lists only shipped agents — frontmatter block (lines 9–21) is `"*": deny` + exactly the 11 shipped subagents; zero ghost entries. Verified against executable config, not prose.
- [✓] AC6 — Unshipped agent/skill scoped with P1 marker + named v1 fallback — grep gates 1b/1c empty except the sanctioned Marty Cagan citation (roadmap-framing:45). Random sample (metrics-analyst, epic-framer, pitch-author) each carry "(P1)"/"v1.5" AND name a real v1 owner (pm-delivery-lead, `metrics-design` skill, prd-author, story-writer).
- [✓] AC7 — Lifecycle mapping in exactly one skill; citing agents cite it — `## PM lifecycle vocabulary → engine statuses` table added to `pm-preset-detection` (SKILL.md:37+, "single source" sentence + `handed_off` cross-repo warning). Eight citers found: pm-delivery-lead, handoff-coordinator, handoff command, pm-reviewer, prd-author, roadmap-curator (+ handoff-protocol). See Audit note 1 on `roadmap-framing`.
- [✓] AC8 — pm spec paths only under `.hero/planning/{...}` — gate 1a empty for `planning/specs`/`planning/roadmap`; handoff-coordinator:53 and handoff command use slug-resolved `.hero/planning/{features,bugs,epics,prds,intake}/`; discover command → `planning/initiatives/`.
- [✓] AC9 — `<harness>/` placeholders + `.hero/hero.json` in AGENTS.md structure — Project Structure rewritten to `<harness>/commands|agents|skills`; `hero.json` → `.hero/hero.json`; dead relative decision link flattened.
- [✓] AC10 — Engineering-only command refs scoped or repointed — `/deliver /design /diagnose /review /scrub` either "(engineering pack)"-scoped or repointed to pm surfaces across agents/commands/skills/spec-types.
- [✓] AC11 — `kickoff-prompt` resolves in `--domain pm` install — `git mv` confirmed: `core/skills/kickoff-prompt/SKILL.md` present, `domains/engineering/skills/kickoff-prompt/` gone (RM in `git status`). Engineering-pack scope note added to the moved SKILL.md (lines 36–37). pm agents that load it (prd-author:30, pm-delivery-lead:36) now resolve via the overlay.
- [✓] AC12 — `go test ./...` passes including `content_parity_test.go` — log shows 86 packages ok, `GO TEST EXIT=0`, 0 FAIL. `internal/install` (parity test home), `internal/drift` both green.

## Changes

- [✓] 1 — Lifecycle mapping + audience/body ghost fixes in `pm-preset-detection` — section present; audience trimmed; body ghosts repointed to pm-delivery-lead with P1 scope.
- [✓] 2 — Promote `kickoff-prompt` to `core/skills/` + scope engineering cmd names — `git mv` confirmed; scope note present; parity test green.
- [✓] 3 — Fix AGENTS.md routing + vocabulary tables — both tables rewritten; `hero new <type>` → `hero spec new`; epic → hand-author; dup `/refine` merged.
- [✓] 4 — Fix AGENTS.md CLI + compaction mechanics — events, handoff steps, `hero sync import`, `--kind`→`--tag`, `hero_active` MCP.
- [✓] 5 — Fix AGENTS.md structure/config/dead-link + presets roster — `<harness>/` structure, `.hero/hero.json`, flattened link.
- [✓] 6 — Trim pm-delivery-lead to shipped surfaces — allowlist ghost-free; specialist rows repointed; status chain + events fixed.
- [✓] 7 — Rewrite handoff-coordinator mechanics on real surfaces — slug-resolve paths, `set-owner` flip, `spec_updated` event, read-back verify, no `queue --owner`.
- [✓] 8 — Same-surface fixes in pm `/handoff` command — in-review gate, real paths, `set-owner`, `spec_updated`.
- [✓] 9 — Kill dead CLI in handoff-protocol — events + verify + hand-back on real surfaces; `set-owner` added; `/deliver` scoped.
- [✓] 10 — Normalize type model (cross-domain-graph-query, dependency-mapping) — `spec` type → `type: feature`; `hero_why <slug>`; ghosts dropped.
- [✓] 11 — Sweep remaining agent files (8) — all 8 touched in diff.
- [✓] 12 — Sweep remaining command files (8) — release-notes self-retracting skill removed, leads with v1 owner + template.
- [✓] 13 — Sweep remaining skill files (13 + `prd-anti-patterns`) — `acceptance-criteria-gherkin` now "planned for v1.5"; `--themes`→`--list --tag`; `/scrub intake`→`/triage`.
- [✓] 14 — Scope pm spec-type docs (`intake.md`) — `/diagnose` → engineering-pack; `/scrub intake` → `/triage`.

## Open items

None from the ledger — every row is DONE with a concrete reason, and every DONE was corroborated against the diff/binary.

## Audit notes

1. **Residual non-engine status vocabulary in `roadmap-framing/SKILL.md:96,103`** (not a blocker). Lines 96 ("child story is in `ready` or `in-flight`") and 103 ("child stories are all `drafted` or `refined` (not `ready`)") use PM lifecycle terms as child-story status values, outside the mapping table and without citing `pm-preset-detection`. This file was in the Change list (Change 13) only for its phantom-skill refs, which were fixed; the lifecycle-vocabulary content in this file was not scoped for edit. AC#7's letter binds "agents," and roadmap-framing is a skill, so this is not an AC violation — but it is a genuine surviving instance of the exact vocabulary the initiative is trying to converge, and the spec's own gate 1d surfaces it. Worth a one-line cite in a follow-up, not a re-engineer. The gate-1d hits at story-writer:149, prd.md:26, prd-anti-patterns:21, roadmap-framing:95, cross-domain-graph-query:123, handoff-coordinator:76 are all prose ("externally-drafted PRD", "not-yet-handed-off") and correctly not status declarations.

2. **Ledger accurately self-reported two beyond-spec fixes** — `prd-anti-patterns/SKILL.md` (omitted from the Change list) and two sub-engineer rows tightened to real skills. Both confirmed present in the diff; not scope drift, they serve the ACs.

3. **Scope is clean.** Diff touches only `domains/pm/` (40 files), the `core/skills/kickoff-prompt` promotion, and the engineering→core `kickoff-prompt` move. Zero Go files. No sibling-spec scope bled in: no sales-pack edits, no routing-file roster additions, no token-efficiency compaction. The `.hero/NEXT.md`, `.hero/SNAPSHOT.md`, `.hero/next/chet-bellows.md` modifications are handoff/snapshot projection files, not pack content.

4. **Sanctioned residual intact** — Marty Cagan "outcomes-over-outputs" book citation at roadmap-framing:45 left as prose (not a skill ref), as the spec directed.
