# Delivery audit — story-queue-planning-backing

**Audited:** working tree (new files untracked) + `git diff HEAD -- domains/pm/AGENTS.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1 Two agent files, valid frontmatter (name=slug, description, mode: subagent, permission) — `domains/pm/agents/capacity-planner.md`, `cycle-planner.md`; validation grep block passed
- [✓] AC-2 capacity-planner Startup names all 5 skills + cut-line surface — `capacity-planner.md:21-29` load-list, `:17,49-51` cut-line + 4 preset lenses
- [✓] AC-3 cycle-planner Startup names all 7 skills + one-preset-adaptive framing + cycle-fit — `cycle-planner.md:19` ("intentionally **one agent**, not three", §C.6), `:23-33` load-list, `:21,43` cycle-fit marker
- [✓] AC-4 Three skills, frontmatter + metadata + 4 sections + 90–205 lines, no placeholder markers — capacity-planning (90), iteration-planning (91), shape-up-cadence (90); validation passed
- [✓] AC-5 capacity-planning: per-preset math + honest velocity distribution + WIP + cut-line — `capacity-planning/SKILL.md:24-49` (4 preset models w/ worked passes), `:51-58` honest velocity, `:63-73` cut-line
- [✓] AC-6 iteration-planning: kanban/phased generic iteration — WIP-as-tool, rolling commit, phase gates — `iteration-planning/SKILL.md:31-37,39-51,53-63` (each with worked pass)
- [✓] AC-7 shape-up-cadence: 6-week+cooldown, betting-table timing, hill-chart cadence, hill-chart-reasoning x-ref — `shape-up-cadence/SKILL.md:21-32,34-53,56-64,87`
- [✓] AC-8 Four command files route correctly + description + `$ARGUMENTS`; plan-* name preset-specific entry point — `commands/capacity.md`→capacity-planner; `plan-{cycle,sprint,iteration}.md`→cycle-planner; validation passed
- [✓] AC-9 New Wave subsection below WAVE-2 marker, after prior blocks (line 190 > Wave-3 at 165), before "When routing" para (209), 5 routes + supersede statement — `AGENTS.md:190-205`
- [✓] AC-10 Agents/Skills Reference bullets + PM Commands roster extended — `AGENTS.md:308` (agents), Skills Reference bullet, `:293` (commands roster line-edit)
- [✓] AC-11 Canonical routing table + all 7 prior child Wave headers byte-unchanged — only removed line is the AC-10-permitted in-place PM Commands roster edit; validation confirmed "P1, no v1 surface" canonical row + all 7 prior headers present
- [✓] AC-12 No dangling refs — all 7 agent load-refs resolve; 3 new skills close the shipped `(P1, ships v1.5)` forward-refs (targets now on disk); all skill x-refs resolve; 6 advisory tokens (capacity-planner, cycle-planner, pm-delivery-lead) verified real agents, not skill slugs
- [✓] AC-13 Changes confined to `domains/pm/` — no Go/schema/hero.json/consumer/new type-vocab-methodology

## Changes
- [✓] capacity-planner.md — velocity/WIP/cut-line reconciler; permission task deny-all, webfetch deny; 5-skill Startup; 5-step delegating-free workflow
- [✓] cycle-planner.md — one preset-adaptive planner; permission task allows capacity-planner + prioritization-strategist, denies rest; 7-skill Startup; delegates capacity read + ranked queue
- [✓] capacity-planning/SKILL.md — 4 preset math models, honest-velocity distribution discipline, WIP-as-tool, cut-line drawing w/ worked sprint+kanban passes
- [✓] iteration-planning/SKILL.md — kanban/phased-only model, WIP-as-tool, rolling commit, phase-gate semantics, worked kanban+phased passes, Story-field population
- [✓] shape-up-cadence/SKILL.md — 6+2 rhythm + variations, cooldown-non-negotiable, betting-table timing, hill-update cadence, interlock diagram
- [✓] capacity.md / plan-cycle.md / plan-sprint.md / plan-iteration.md — thin routers, preset-authoritative note, recommend-never-auto-commit, `$ARGUMENTS`
- [✓] AGENTS.md — additions-only: new Wave subsection + Agents/Skills Reference bullets + Commands roster line-edit

## Open items
None.

## Audit notes
- Validation block ran verbatim from repo root → `VALIDATION OK`. The 6 `POSSIBLE dangling` advisories are all agent role names backticked in skill bodies (`capacity-planner`, `cycle-planner`, `pm-delivery-lead`) — each resolves to a real `domains/pm/agents/<name>.md`, not a broken skill x-ref. Confirmed on disk.
- Both agents are genuinely distinct and substantive (not stubs): capacity-planner = capacity/cut-line reconciliation with a deny-all task block; cycle-planner = one preset-adaptive planner that delegates the capacity read to capacity-planner. Load-lists and permission blocks match the spec's Changes items exactly.
- All three skills are real framework skills, not thin scaffolds — each carries per-preset math, worked numeric passes, a real Anti-patterns section, and resolvable cross-references. They sit at the floor of the line band (90–91 lines) but are dense, not padded.
- Forward-refs closed in the load-bearing sense: the shipped `sprint-planning`/`cycle-planning` skills reference the three new skills; those targets now exist on disk so the refs resolve. The shipped files still carry the cosmetic "(P1, ships v1.5)" qualifier text — the spec's Risks section explicitly sanctioned leaving that qualifier as optional/out-of-scope, so it is not a defect.
- Scope: only tracked source file changed is `domains/pm/AGENTS.md`; the 9 new artifacts are all under `domains/pm/`. The other working-tree changes (`.hero/NEXT.md`, `QUEUE.md`, `events.log`, the flat spec `.md` replaced by the spec folder, `.hero/drive/`) are Hero workspace/projected-handoff bookkeeping, not pack source — expected and permitted.
