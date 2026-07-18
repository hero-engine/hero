---
title: "Story Queue Backing — Capacity + Cycle Planning Agents, Skills, Commands"
slug: story-queue-planning-backing
type: feature
status: completed
domain: pm
priority: high
size: medium
created: 2026-07-17
tags: [pm, planning, story-queue, wave-1]
relations:
  - target: pm-pack-completion
    kind: parent
  - target: pm-doctrine-and-skill-backfill
    kind: depends-on
  - target: pm-foundation-delivery
    kind: depends-on
completed_at: 2026-07-18T00:03:48Z
---

# Story Queue Backing — Capacity + Cycle Planning Agents, Skills, Commands

## Goal

Back the Story Queue hero-code view — the **only** live PM surface with **zero**
backing agents today — by authoring, content-only under `domains/pm/`, the two
planning agents it needs plus their supporting skills and command shims:

- **`capacity-planner`** — reconciles committed work against team capacity
  (velocity under sprint, appetite under cycle, WIP under kanban, release scope
  under phased); powers the Story Queue **velocity cut-line**.
- **`cycle-planner`** — one preset-adaptive agent that plans the next iteration
  under the active preset (sprint / cycle / iteration); powers the Story Queue
  **cycle-fit marker**.
- Three net-new skills — **`capacity-planning`**, **`iteration-planning`**,
  **`shape-up-cadence`** — that the two agents load and that **un-dangle** the
  three forward-references the shipped `sprint-planning` and `cycle-planning`
  skills already carry as "(P1, ships v1.5)".
- Four command shims — **`/capacity`** → `capacity-planner`, and
  **`/plan-cycle` / `/plan-sprint` / `/plan-iteration`** → `cycle-planner` (each
  the preset-specific entry point into the one preset-adaptive agent).
- The `domains/pm/AGENTS.md` reconciliation: a new Wave route subsection
  (appended below the marker, after every prior child's block) that **supersedes**
  the canonical table's `/capacity` + `/plan-*` "no v1 surface (ships v1.5)"
  annotations, plus additive entries in the Agents / Skills / Commands references.

"Done" = the two agent files, three skill files, and four command files exist
with valid frontmatter and the mandated content; both agents load
`pm-agent-doctrine` plus their real, resolvable skills; the three new skills
resolve on disk (closing the two shipped forward-refs); the AGENTS.md edits are
additions-only with the canonical routing table and every prior child's Wave
block byte-unchanged; and the `## Validation` bash block passes with `VALIDATION
OK`.

## Kickoff

Cold-start prompt to paste into a fresh session:

> Deliver `story-queue-planning-backing` (final child of the `pm-pack-completion`
> initiative). Spec:
> `.hero/planning/initiatives/pm-pack-completion/story-queue-planning-backing/spec.md`.
> Author two PM planning agents, three planning skills, and four command shims —
> all content-only under `domains/pm/`, no Go, no schema, no consumer-side change:
> agents `domains/pm/agents/{capacity-planner,cycle-planner}.md`; skills
> `domains/pm/skills/{capacity-planning,iteration-planning,shape-up-cadence}/SKILL.md`;
> commands `domains/pm/commands/{capacity,plan-cycle,plan-sprint,plan-iteration}.md`.
> Match the shipped shapes exactly — study `domains/pm/agents/stakeholder-communicator.md`
> for the agent frontmatter + Startup/When-invoked/Workflow/Anti-patterns/Default-output
> shape, `domains/pm/skills/sprint-planning/SKILL.md` and
> `domains/pm/skills/cycle-planning/SKILL.md` for the skill shape and voice, and
> `domains/pm/commands/metrics.md` for the command-as-router shape. `capacity-planner`
> loads `pm-agent-doctrine` + `capacity-planning` + `sprint-planning` +
> `cycle-planning` + `pm-preset-detection`; `cycle-planner` loads `pm-agent-doctrine`
> + `sprint-planning` + `cycle-planning` + `iteration-planning` + `shape-up-cadence`
> + `capacity-planning` + `pm-preset-detection`. Then append one new Wave route
> subsection to `domains/pm/AGENTS.md` **below** the `<!-- WAVE-2 ROUTES -->`
> marker and **after** every prior child's block (before the "When routing, pass…"
> paragraph), and add the two agents / three skills / four commands to the
> Agents / Skills / Commands references — additions-only; do NOT edit the canonical
> Natural Language Routing table or any prior child's Wave block. Run the
> `## Validation` bash block; every check must pass (`VALIDATION OK`).

## Problem

hero-code embeds the Hero engine and serves PM agents/commands/skills live per
active domain. The [PM Pack Audit — 2026-07-17](../../../features/hero-pm/pm-pack-audit-2026-07.md)
Surface-coverage matrix records the Story Queue view as the single worst gap:

> | Story Queue | capacity-planner, cycle-planner | ❌ **both missing** |

Its **velocity cut-line** and **cycle-fit marker** are drawn in the mockups with
nothing behind them. Both agents were named in the locked `hero-pm` design
(`agent-pack-design.md` §C.5 `capacity-planner`, §C.6 `cycle-planner` — the
latter explicitly "intentionally one agent across the three presets"), and both
were deferred out of the v1 minimum-viable pack. This child authors them.

The three supporting skills are already **referenced but undesigned** in two
directions:

1. The shipped `sprint-planning` and `cycle-planning` skills each carry
   Cross-references to `capacity-planning`, `iteration-planning`, and
   `shape-up-cadence` annotated "(P1, ships v1.5)" — live forward-references to
   files that do not yet exist. Authoring the three here **un-dangles** them.
2. `agent-pack-design.md` §D.3/§D.4 specifies all three as design skills
   (`capacity-planning`, `iteration-planning`, `shape-up-cadence`), and the
   shipped `shape-up-cadence` consumers cross-reference `hill-chart-reasoning`
   (authored by child #10, now on disk — the cross-ref resolves).

### Dependency context — both prerequisites are satisfiable in practice

- **`pm-doctrine-and-skill-backfill`** (child #1) established the
  `pm-agent-doctrine` spine (corpus-grounding, decision-gate discipline,
  compare-don't-replace) and the AGENTS.md Wave-region marker + append
  convention. Both new agents load `pm-agent-doctrine`; the routing edits append
  below the marker per that convention.
- **`pm-foundation-delivery`** delivers the canonical `feature` spec type
  rendered as **"Story"** via the `agile-scrum` vocabulary plus the
  `scrum`/`kanban`/`shape-up` methodology profiles that make sprint/cycle/
  iteration presets meaningful. **This prerequisite is VERIFIED PRESENT and
  engine-loadable on disk** — `core/spec-types/feature.md`,
  `core/methodologies/*.yaml`, `core/vocabularies/*.yaml`, and
  `.hero/cache/spec-types.json` all exist, and `hero spec new --type feature`
  works — even though `pm-foundation-delivery`'s own `status:` flag is stale at
  `planning`. **This child authors NO spec types.** It operates over the existing
  `feature` type rendered as "Story"; the planning agents populate the
  preset-conditional Story fields (`points`/`sprint` under scrum, `appetite`/
  `cycle`/`hill_position` under shape-up) that `pm-preset-detection` and the
  registered profiles already define. No new type, vocabulary, or methodology is
  created here.

Everything in scope is pure content under `domains/pm/` — no Go, no schema, no
consumer-side (hero-code / GTK) change.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL provide two agent files —
  `domains/pm/agents/capacity-planner.md` and
  `domains/pm/agents/cycle-planner.md` — each present, non-empty, and carrying
  valid YAML frontmatter with `name:` equal to the file slug, a one-sentence
  `description:`, `mode: subagent`, and a `permission:` block.
- **AC-2:** WHEN a reader opens `capacity-planner` THE SYSTEM SHALL declare a
  Startup load-list naming `pm-agent-doctrine`, `capacity-planning`,
  `sprint-planning`, `cycle-planning`, and `pm-preset-detection`, and describe
  reconciling committed work against capacity across all four preset lenses
  (velocity / appetite / WIP / release), producing the Story Queue **velocity
  cut-line**.
- **AC-3:** WHEN a reader opens `cycle-planner` THE SYSTEM SHALL declare a
  Startup load-list naming `pm-agent-doctrine`, `sprint-planning`,
  `cycle-planning`, `iteration-planning`, `shape-up-cadence`, `capacity-planning`,
  and `pm-preset-detection`, present itself as **one preset-adaptive agent**
  (sprint / cycle / iteration — not three agents) that reads the active preset
  via `pm-preset-detection`, recommends (never auto-commits) the next iteration,
  and powers the Story Queue **cycle-fit marker**.
- **AC-4:** THE SYSTEM SHALL provide three skill files —
  `domains/pm/skills/{capacity-planning,iteration-planning,shape-up-cadence}/SKILL.md`
  — each present, non-empty, with valid YAML frontmatter carrying `name:` equal
  to the directory slug, a one-sentence `description:`, and a `metadata:` block
  with `audience:` and `purpose:`, and a body carrying at minimum the sections
  `## What I do`, `## When to use me`, `## Anti-patterns`, and
  `## Cross-references` (each file ≥ 90 and ≤ ~200 lines, no `TODO`/`TBD`/
  `FIXME`/`placeholder`/`stub` markers).
- **AC-5:** WHEN a reader opens `capacity-planning` THE SYSTEM SHALL give
  per-preset capacity math (velocity distribution under sprint, appetite under
  cycle, WIP + aging under kanban, release scope under phased), the discipline of
  **honest velocity** (distribution not point estimate; commit vs forecast), WIP
  limits, and how the **cut-line** is drawn on the Story Queue.
- **AC-6:** WHEN a reader opens `iteration-planning` THE SYSTEM SHALL give the
  generic iteration shape for **kanban and phased** presets — WIP limits as a
  tool not a wall, rolling commitment, and phase-gate semantics — explicitly
  distinct from fixed-sprint and Shape-Up-cycle planning.
- **AC-7:** WHEN a reader opens `shape-up-cadence` THE SYSTEM SHALL give the
  operational **6-week build + cooldown** rhythm, **betting-table timing**, and
  **hill-chart update** cadence, and cross-reference `hill-chart-reasoning` for
  the deeper unknowns-remaining reading (that skill is shipped on disk).
- **AC-8:** THE SYSTEM SHALL provide four command files —
  `domains/pm/commands/capacity.md` routing to `capacity-planner`, and
  `domains/pm/commands/{plan-cycle,plan-sprint,plan-iteration}.md` each routing to
  `cycle-planner` — where each `plan-*` file names itself the **preset-specific
  entry point** into the one preset-adaptive agent; every command file carries a
  `description:` frontmatter line and a `$ARGUMENTS` pass-through.
- **AC-9:** THE SYSTEM SHALL append a single new Wave route subsection to
  `domains/pm/AGENTS.md` **below** the `<!-- WAVE-2 ROUTES -->` marker and
  **after** every prior child's Wave block (immediately before the "When routing,
  pass the user's original context…" paragraph), containing routes for
  `/capacity`, `/plan-cycle`, `/plan-sprint`, `/plan-iteration`,
  `capacity-planner`, and `cycle-planner`, and stating that these routes
  **supersede** the canonical table's `/capacity` + `/plan-*` "no v1 surface
  (ships v1.5)" annotations (the same supersede pattern children #7/#8 used).
- **AC-10:** THE SYSTEM SHALL add the two new agents (`capacity-planner`,
  `cycle-planner`) to the AGENTS.md **Agents Reference** as a new bullet, the
  three new skills (`capacity-planning`, `iteration-planning`, `shape-up-cadence`)
  to the **Skills Reference** as a new bullet, and the four new commands
  (`/capacity`, `/plan-cycle`, `/plan-sprint`, `/plan-iteration`) to the PM
  **Commands** roster bullet — all via line-insertion.
- **AC-11:** THE SYSTEM SHALL leave the canonical Natural Language Routing table
  (including the `/capacity` and `/plan-*` "no v1 surface (ships v1.5)" rows) and
  every prior child's Wave route subsection byte-unchanged — the only AGENTS.md
  edits are the one new Wave subsection and the three reference-bullet insertions.
- **AC-12:** THE SYSTEM SHALL introduce no dangling references: every skill named
  in either agent's load-list resolves to a `domains/pm/skills/<name>/` directory
  (shipped or authored here); the three skills authored here resolve on disk,
  closing the `(P1, ships v1.5)` forward-refs in `sprint-planning` and
  `cycle-planning`; and every `` `skill-name` `` cross-reference inside the three
  new skills resolves to a shipped or newly-authored skill directory.
- **AC-13:** THE SYSTEM SHALL confine all changes to `domains/pm/` pack source —
  no Go, no schema, no `hero.json`, no consumer-side (hero-code / GTK) change,
  and no new spec type / vocabulary / methodology — satisfying the
  `harness-changes-cover-all-targets` tripwire by authoring exclusively in the
  pack.

## Changes

Author only under `domains/pm/`. Match the shipped shapes exactly: study
`domains/pm/agents/stakeholder-communicator.md` (agent frontmatter + `## Startup`
load-list + `## When invoked` + numbered `## Workflow` + `## Anti-patterns` +
`## Default output`), `domains/pm/skills/sprint-planning/SKILL.md` and
`domains/pm/skills/cycle-planning/SKILL.md` (skill frontmatter + voice + worked
numeric examples + `## Anti-patterns` + `## Cross-references`), and
`domains/pm/commands/metrics.md` (thin command-as-router).

### 1. `domains/pm/agents/capacity-planner.md`

- Frontmatter: `name: capacity-planner`; one-sentence `description:` naming the
  Story Queue velocity cut-line; `mode: subagent`; `temperature: 0.1`;
  `color: secondary`; `permission:` with `edit: allow`, `task: {"*": deny}`,
  `skill: {"*": allow}`, `webfetch: deny` (a planner reads local state; it does
  not fetch the web).
- `You are a capacity planner.` opener.
- `## Startup` load-list (all five, in this order): `pm-agent-doctrine` (decision
  gates: a capacity read is a *recommendation*, never an auto-commit; corpus-
  grounding — cite the actual velocity history, not a vibe), `capacity-planning`
  (the per-preset math + the cut-line), `sprint-planning` (velocity-distribution
  reading under scrum), `cycle-planning` (appetite-as-constraint under shape-up),
  `pm-preset-detection` (read `hero.json` `pm.presets` to pick which lens applies).
- `## When invoked`: `/capacity`; the Story Queue **velocity cut-line** render;
  "can we fit X this cycle" / "are we over capacity"; called by
  `prioritization-strategist` for effort inputs and by `cycle-planner` for the
  capacity read.
- `## Workflow`: (1) detect the active preset via `pm-preset-detection`; (2) pull
  the honest capacity signal for that preset — velocity **distribution** (sprint,
  never a single number), appetite budget (cycle), WIP + aging (kanban), release
  scope (phased); (3) walk the prioritized Story Queue, sum against capacity, and
  place the **cut-line** (what's in / what's below the line and why); (4) surface
  over-capacity as an explicit warning with the specific overcommit, never a
  silent stretch; (5) honor doctrine on the way out — the cut-line is a proposal
  the team accepts, and every number traces to real history.
- `## Anti-patterns`: velocity as a point estimate; sandbag-then-overcommit;
  auto-committing the cut-line; appetite treated as an estimate to validate;
  hiding the overcommit instead of naming it.
- `## Default output`: preset detected; capacity signal (with distribution/appetite/
  WIP shown); the cut-line placement + rationale; any over-capacity warning; a
  one-line log.

### 2. `domains/pm/agents/cycle-planner.md`

- Frontmatter: `name: cycle-planner`; one-sentence `description:` stating it is
  **one preset-adaptive agent** (sprint / cycle / iteration) powering the Story
  Queue cycle-fit marker; `mode: subagent`; `temperature: 0.1`; `color: secondary`;
  `permission:` with `edit: allow`, `task:` allowing `capacity-planner` +
  `prioritization-strategist` (it delegates the capacity read + the ranked queue)
  and denying the rest, `skill: {"*": allow}`, `webfetch: deny`.
- `You are a cycle / sprint / iteration planner.` opener, immediately stating it
  is intentionally **one agent** whose behavior switches on the active preset
  (per `agent-pack-design.md` §C.6: splitting into three would create three
  near-identical files).
- `## Startup` load-list (all seven, in this order): `pm-agent-doctrine`,
  `sprint-planning`, `cycle-planning`, `iteration-planning`, `shape-up-cadence`,
  `capacity-planning`, `pm-preset-detection`.
- `## When invoked`: `/plan-sprint`, `/plan-cycle`, `/plan-iteration` (the three
  preset-specific entry points); "plan next cycle/sprint/iteration"; the Story
  Queue **cycle-fit marker** render.
- `## Workflow`: (1) read the active preset via `pm-preset-detection` and select
  the model — sprint (velocity + commit/stretch), cycle (betting table + appetite
  + cooldown, cadence from `shape-up-cadence`), iteration (kanban/phased rolling
  commit + WIP, from `iteration-planning`); (2) delegate the capacity read to
  `capacity-planner` and the ranked queue to `prioritization-strategist`; (3)
  produce a **recommended commit** — populate the right preset-conditional Story
  fields (`points`/`sprint`; `appetite`/`cycle`/`hill_position`; `phase`) and mark
  cycle-fit per Story; (4) surface what gets **cut** and why; (5) never
  auto-commit — recommend, the team decides (decision gate from doctrine).
- `## Anti-patterns`: one-model-for-all-presets (ignoring the active preset);
  estimating inside cycles; skipping cooldown to "catch up"; auto-committing the
  plan; a betting-table decision made outside the betting table; a cycle-fit
  marker asserted without the capacity read.
- `## Default output`: preset detected + model chosen; recommended commit list
  with per-Story cycle-fit; what's cut + why; the fields populated; a one-line log.

### 3. `domains/pm/skills/capacity-planning/SKILL.md`

- Frontmatter: `name: capacity-planning`; `description:` (per-preset capacity math
  + honest velocity + WIP + the cut-line); `metadata.audience: capacity-planner,
  cycle-planner, prioritization-strategist, portfolio-curator`;
  `metadata.purpose: process-guidance`.
- Body (~120–170 lines): `## What I do`; `## When to use me`; **per-preset math** —
  sprint (velocity **distribution**, median-not-mean, commit vs forecast), cycle
  (appetite math — budget not estimate), kanban (WIP limits + cycle-time/aging),
  phased (release scope + gate capacity); **honest velocity** (show the
  distribution, uncertainty bands, the difference between a *commit* and a
  *forecast*); **WIP limits** as a tool; **the cut-line** — how it's drawn on the
  Story Queue (sort by priority, walk summing against capacity, promote
  dependencies, cut-line non-negotiable for promotion); a worked numeric example;
  `## Anti-patterns` (sandbag-then-overcommit; last sprint's velocity as next
  sprint's commit with no bands; a headline velocity number that erases the
  distribution; WIP as a wall); `## Cross-references` (`sprint-planning`,
  `cycle-planning`, `iteration-planning`, `pm-preset-detection`, and the
  `feature`/`initiative` preset-conditional fields).

### 4. `domains/pm/skills/iteration-planning/SKILL.md`

- Frontmatter: `name: iteration-planning`; `description:` (generic iteration for
  kanban / phased — WIP, rolling commit, phase gates); `metadata.audience:
  cycle-planner, capacity-planner`; `metadata.purpose: process-guidance`.
- Body (~110–160 lines): `## What I do` (the generic iteration shape used under
  **kanban and phased** presets, explicitly not fixed-sprint and not Shape-Up-
  cycle); `## When to use me`; **WIP limits as a tool, not a wall** (what a limit
  is for, how to set one, what a breach signals); **rolling commitment** (flow /
  pull vs a timeboxed batch commit; how the Story Queue behaves without a fixed
  boundary); **phase-gate semantics** for the phased preset (a gate is an explicit
  entry/exit criterion, not a date); a worked example; how it populates the
  `phase` Story field; `## Anti-patterns` (WIP limit ignored under pressure;
  phased treated as mini-waterfall with date-gates; "rolling commit" as an excuse
  for no commitment at all); `## Cross-references` (`sprint-planning`,
  `cycle-planning`, `capacity-planning`, `pm-preset-detection`).

### 5. `domains/pm/skills/shape-up-cadence/SKILL.md`

- Frontmatter: `name: shape-up-cadence`; `description:` (the operational 6-week +
  cooldown rhythm, betting-table timing, hill-chart update cadence);
  `metadata.audience: cycle-planner, pm-delivery-lead`; `metadata.purpose:
  process-guidance`.
- Body (~110–160 lines): `## What I do` (the recurring **operational rhythm** of
  the cycle preset — distinct from `cycle-planning`, which is the planning
  mechanics; this skill is the cadence that repeats); `## When to use me`;
  **the 6-week build + 2-week cooldown** rhythm and variations (4+1, 5+1),
  cooldown as non-negotiable; **betting-table timing** — once per cycle near the
  end of cooldown, not continuous; **hill-chart update cadence** — at least twice
  per cycle, at standup / weekly check-ins, movement is the data; how it drives
  the cycle-fit marker's timing; `## Anti-patterns` (ad-hoc cycle starts; betting
  outside the betting table; cooldown used to extend the previous cycle;
  hill charts updated as progress bars or never updated); `## Cross-references`
  — **must include `hill-chart-reasoning`** (child #10, shipped — the deeper
  unknowns-remaining reading), plus `cycle-planning`, `pitch-writing-shape-up`,
  `capacity-planning`.

### 6. `domains/pm/commands/capacity.md`

- Thin router (model `metrics.md`): `description:` frontmatter one-liner; body
  routes to `capacity-planner`; states the required argument (a cycle/sprint
  context or the Story Queue to reconcile — ask if absent, don't infer); describes
  what lands (the cut-line + capacity read, a proposal not an auto-commit);
  `Request: $ARGUMENTS` at the end.

### 7. `domains/pm/commands/plan-cycle.md`, `plan-sprint.md`, `plan-iteration.md`

- Three thin routers, each routing to **`cycle-planner`**. Each names itself the
  **preset-specific entry point** into the one preset-adaptive agent:
  `plan-cycle` → shape-up cycle model; `plan-sprint` → scrum sprint model;
  `plan-iteration` → generic kanban/phased iteration model. Each notes that the
  agent still reads the active preset via `pm-preset-detection` (the command is a
  hint, the preset is authoritative), states what lands (a **recommended** commit,
  never auto-committed), and ends with `Request: $ARGUMENTS`.

### 8. `domains/pm/AGENTS.md` — additions only

- **New Wave route subsection** — insert immediately **before** the line
  `When routing, pass the user's original context as arguments to the` (currently
  the paragraph right after the Wave-3 launch routes table), so it lands **after**
  every prior child's Wave block. Header + framing paragraph + a routing table
  with the five routes. Model the framing on child #8's block, explicitly stating
  the routes **supersede** the canonical table's `/capacity` + `/plan-*` "no v1
  surface (ships v1.5)" annotations. Suggested content:

  ```
  #### Wave-1 Story Queue planning routes (story-queue-planning-backing)

  These are net-new routes appended by `story-queue-planning-backing` (the final
  pm-pack-completion child): the two agents that back the live Story Queue view —
  the **velocity cut-line** (`capacity-planner`) and the **cycle-fit marker**
  (`cycle-planner`, one preset-adaptive agent) — plus the four planning command
  shims. Both agents are **capacity/planning** agents: they recommend, they never
  auto-commit a plan (decision gate). This subsection **supersedes** the canonical
  table's `/capacity` and `/plan-cycle` / `/plan-sprint` / `/plan-iteration`
  "no v1 surface (ships v1.5)" annotations (the pattern children #7 `/metrics` and
  #8 `/standup` used).

  | User intent | Vocabulary-variant phrasing | Command (shipped surface) |
  |---|---|---|
  | Capacity, "can we fit X this cycle", velocity room, appetite room, WIP headroom, "are we over capacity" (un-deferred) | | `/capacity` → `capacity-planner` — reconciles committed work vs capacity under the active preset (velocity / appetite / WIP / release) and places the Story Queue cut-line; recommends, never auto-commits; supersedes the canonical "no v1 surface (ships v1.5)" row |
  | Plan next cycle, "what should we bet on", betting table, appetite (un-deferred) | "plan the scope cycle" | `/plan-cycle` → `cycle-planner` — the **shape-up cycle** entry point into the one preset-adaptive planner (betting table + appetite + cooldown cadence); the agent still reads the active preset via `pm-preset-detection` |
  | Plan next sprint, "what should we commit to", velocity commit, cut-line (un-deferred) | | `/plan-sprint` → `cycle-planner` — the **scrum sprint** entry point into the same preset-adaptive planner (velocity + commit/stretch) |
  | Plan next iteration, kanban pull, phased release plan, "what's next in the queue" (un-deferred) | "plan the phase" | `/plan-iteration` → `cycle-planner` — the **generic kanban/phased** entry point into the same preset-adaptive planner (WIP + rolling commit + phase gates) |
  ```

- **Agents Reference** — insert a new bullet immediately **before** the
  `- **Core (installed with every pack):**` bullet of the Agents Reference list:

  ```
  - **PM Wave-1 Story Queue planning:** `capacity-planner` (reconciles committed work vs capacity under the active preset — velocity/appetite/WIP/release; backs the Story Queue velocity cut-line; recommends, never auto-commits; loads `capacity-planning` + `sprint-planning` + `cycle-planning`), `cycle-planner` (**one preset-adaptive** planner — sprint/cycle/iteration; backs the Story Queue cycle-fit marker and the `/plan-cycle` / `/plan-sprint` / `/plan-iteration` shims; delegates the capacity read to `capacity-planner` and the ranked queue to `prioritization-strategist`). See the Wave-1 Story Queue planning routes above.
  ```

- **Skills Reference** — insert a new bullet immediately **before** the
  `- **Core (installed with every pack):**` bullet of the Skills Reference list:

  ```
  - **Story Queue planning (Wave-1):** `capacity-planning` (per-preset capacity math + honest velocity distribution + WIP limits + the Story Queue cut-line — un-dangles the `(P1, ships v1.5)` forward-ref in `sprint-planning`/`cycle-planning`), `iteration-planning` (generic kanban/phased iteration — WIP as a tool, rolling commit, phase gates), `shape-up-cadence` (the operational 6-week + cooldown rhythm, betting-table timing, hill-chart update cadence; cross-refs `hill-chart-reasoning`). All three loaded by `capacity-planner` and/or `cycle-planner`.
  ```

- **Commands roster** — in the existing `- **PM:**` Commands Reference bullet,
  extend the command list to include `/capacity`, `/plan-cycle`, `/plan-sprint`,
  and `/plan-iteration` (keep the sentence's existing phrasing; add the four
  commands to the enumerated set and, if a trailing clause fits the style, note
  they route to the Story Queue planning agents per the Wave-1 routes above).

- Do **not** edit the canonical Natural Language Routing table (rows 30–61,
  including the `/capacity` and `/plan-*` "no v1 surface (ships v1.5)" rows) or
  any prior child's Wave route subsection.

## Boundaries

- **Content only, `domains/pm/` only.** No Go, no schema, no `hero.json`, no
  consumer-side (hero-code / GTK) change. The `harness-changes-cover-all-targets`
  tripwire is satisfied by authoring exclusively in the pack source.
- **No new spec type / vocabulary / methodology.** The `feature`-rendered-as-
  "Story" type, the `scrum`/`kanban`/`shape-up` profiles, and the `agile-scrum`
  vocabulary are the `pm-foundation-delivery` prerequisite and are already on
  disk. This child consumes them; it authors none of them.
- **One `cycle-planner`, not three.** Per `agent-pack-design.md` §C.6 the sprint/
  cycle/iteration behavior is one preset-adaptive agent; do not split it into
  three agents. The three `/plan-*` commands are entry-point shims into the one
  agent.
- **Additions-only on AGENTS.md.** The only edits are the one new Wave subsection
  and the three reference-bullet insertions (Agents / Skills / Commands). Do not
  touch the canonical routing table or any prior child's Wave block.
- **Agents recommend, they never auto-commit.** Both planners surface a proposal
  the team accepts; neither flips Story fields autonomously without the human
  decision gate. This is doctrine (`pm-agent-doctrine`), not a nicety.
- **Out of scope (other children / waves):** the `dependency-mapper`,
  `stakeholder-communicator`, `pitch-author`, and `duplicate-intake-scrubber`
  agents; the `/standup`, `/interview`, `/experiment`, `/launch`, `/scrub`
  surfaces; and any prioritization-framework work — those belong to other
  pm-pack-completion children (several already shipped).
- **Depends on `pm-doctrine-and-skill-backfill`** (the `pm-agent-doctrine` spine
  + AGENTS.md marker convention both agents rely on) and, nominally, on
  `pm-foundation-delivery` (the Story type + presets — verified present on disk;
  proceed even though its status flag is stale).

## Risks

- **Stale prerequisite flag.** `pm-foundation-delivery` reads `status: planning`
  though its artifacts are on disk. A dependency-gate check may warn about
  delivering against an unfinished prerequisite. Mitigation: the Problem section
  records the on-disk verification (`core/spec-types/feature.md`,
  `core/methodologies/*.yaml`, `core/vocabularies/*.yaml`,
  `.hero/cache/spec-types.json`, `hero spec new --type feature` works); proceed
  and note the verification in the Completion Ledger.
- **Forward-ref annotations drift.** The shipped `sprint-planning`/`cycle-planning`
  Cross-references label the three new skills "(P1, ships v1.5)". Authoring the
  skills makes those labels stale-but-harmless (the refs now resolve). Leaving
  them is acceptable (they still point at a real skill); optionally the delivering
  engineer may drop the "(P1, ships v1.5)" qualifier — but that edits shipped
  files beyond this child's core scope, so treat it as optional and out of the
  required diff.
- **AGENTS.md insertion precision.** The Wave subsection must land after every
  prior child's block and before the "When routing…" paragraph; the reference
  bullets must land before their respective `Core` bullets. A misplaced insertion
  risks reordering. Mitigation: the Validation block asserts both the new content
  and the byte-stability of the canonical rows + prior child headers.

## Validation

```bash
set -euo pipefail
cd /Users/bwheeler/projects/hero-engine/repository/hero

agents="capacity-planner cycle-planner"
newskills="capacity-planning iteration-planning shape-up-cadence"
cmds="capacity plan-cycle plan-sprint plan-iteration"

# AC-1: both agent files exist, non-empty, valid frontmatter (name=slug, description, mode, permission)
for a in $agents; do
  f="domains/pm/agents/$a.md"
  test -s "$f" || { echo "MISSING agent: $f"; exit 1; }
  grep -qE "^name: $a\$" "$f" || { echo "name != slug in $f"; exit 1; }
  grep -qE '^description: .+' "$f" || { echo "missing description in $f"; exit 1; }
  grep -qE '^mode: subagent\b' "$f" || { echo "missing mode: subagent in $f"; exit 1; }
  grep -qE '^permission:' "$f" || { echo "missing permission block in $f"; exit 1; }
done

# AC-2: capacity-planner load-list + surface
cp="domains/pm/agents/capacity-planner.md"
for s in pm-agent-doctrine capacity-planning sprint-planning cycle-planning pm-preset-detection; do
  grep -qF "$s" "$cp" || { echo "capacity-planner missing load: $s"; exit 1; }
done
grep -qiE 'cut-?line' "$cp" || { echo "capacity-planner missing cut-line"; exit 1; }

# AC-3: cycle-planner load-list + preset-adaptive framing + cycle-fit
yp="domains/pm/agents/cycle-planner.md"
for s in pm-agent-doctrine sprint-planning cycle-planning iteration-planning shape-up-cadence capacity-planning pm-preset-detection; do
  grep -qF "$s" "$yp" || { echo "cycle-planner missing load: $s"; exit 1; }
done
grep -qiE 'preset-adaptive|one agent|one .*planner' "$yp" || { echo "cycle-planner missing preset-adaptive framing"; exit 1; }
grep -qiE 'cycle-?fit' "$yp" || { echo "cycle-planner missing cycle-fit marker"; exit 1; }

# AC-4: three new skills exist, non-empty, frontmatter + sections + line band + no placeholders
for s in $newskills; do
  f="domains/pm/skills/$s/SKILL.md"
  test -s "$f" || { echo "MISSING skill: $f"; exit 1; }
  n=$(wc -l < "$f")
  [ "$n" -ge 90 ] && [ "$n" -le 205 ] || { echo "LINE COUNT out of band ($n): $f"; exit 1; }
  ! grep -qiE '\b(TODO|TBD|FIXME|placeholder|stub)\b' "$f" || { echo "PLACEHOLDER MARKER in $f"; exit 1; }
  grep -qE "^name: $s\$" "$f" || { echo "name != slug in $f"; exit 1; }
  grep -qE '^description: .+' "$f" || { echo "missing description in $f"; exit 1; }
  grep -qE '^metadata:' "$f" || { echo "missing metadata in $f"; exit 1; }
  grep -qE '^\s+audience:' "$f" || { echo "missing metadata.audience in $f"; exit 1; }
  grep -qE '^\s+purpose:' "$f" || { echo "missing metadata.purpose in $f"; exit 1; }
  for h in '## What I do' '## When to use me' '## Anti-patterns' '## Cross-references'; do
    grep -qF "$h" "$f" || { echo "missing section '$h' in $f"; exit 1; }
  done
done

# AC-5/6/7: framework signature phrases per new skill
cap="domains/pm/skills/capacity-planning/SKILL.md"
grep -qiF 'distribution' "$cap" || { echo "capacity-planning missing velocity distribution"; exit 1; }
grep -qiF 'appetite' "$cap" || { echo "capacity-planning missing appetite (cycle)"; exit 1; }
grep -qiF 'WIP' "$cap" || { echo "capacity-planning missing WIP"; exit 1; }
grep -qiE 'cut-?line' "$cap" || { echo "capacity-planning missing cut-line"; exit 1; }

itr="domains/pm/skills/iteration-planning/SKILL.md"
grep -qiF 'kanban' "$itr" || { echo "iteration-planning missing kanban"; exit 1; }
grep -qiF 'WIP' "$itr" || { echo "iteration-planning missing WIP"; exit 1; }
grep -qiF 'phase' "$itr" || { echo "iteration-planning missing phase gates"; exit 1; }
grep -qiF 'rolling' "$itr" || { echo "iteration-planning missing rolling commit"; exit 1; }

suc="domains/pm/skills/shape-up-cadence/SKILL.md"
grep -qiF 'cooldown' "$suc" || { echo "shape-up-cadence missing cooldown"; exit 1; }
grep -qiF 'betting table' "$suc" || { echo "shape-up-cadence missing betting table"; exit 1; }
grep -qiF 'hill' "$suc" || { echo "shape-up-cadence missing hill-chart cadence"; exit 1; }
grep -qF 'hill-chart-reasoning' "$suc" || { echo "shape-up-cadence missing hill-chart-reasoning x-ref"; exit 1; }

# AC-8: four command files exist, route correctly, carry description + $ARGUMENTS
test -s domains/pm/commands/capacity.md || { echo "MISSING /capacity"; exit 1; }
grep -qF 'capacity-planner' domains/pm/commands/capacity.md || { echo "/capacity not routed to capacity-planner"; exit 1; }
for c in plan-cycle plan-sprint plan-iteration; do
  f="domains/pm/commands/$c.md"
  test -s "$f" || { echo "MISSING /$c"; exit 1; }
  grep -qF 'cycle-planner' "$f" || { echo "/$c not routed to cycle-planner"; exit 1; }
done
for c in $cmds; do
  f="domains/pm/commands/$c.md"
  grep -qE '^description: .+' "$f" || { echo "/$c missing description frontmatter"; exit 1; }
  grep -qF '$ARGUMENTS' "$f" || { echo "/$c missing \$ARGUMENTS pass-through"; exit 1; }
done

AG="domains/pm/AGENTS.md"

# AC-9: new Wave subsection present with all five routes + supersede statement, placed before the "When routing" paragraph
grep -qF 'Wave-1 Story Queue planning routes (story-queue-planning-backing)' "$AG" || { echo "AGENTS.md missing new Wave subsection header"; exit 1; }
awk '/Wave-1 Story Queue planning routes \(story-queue-planning-backing\)/{f=1} /^When routing, pass the user/{if(f)print "AFTER_MARKER_OK"; exit}' "$AG" | grep -q AFTER_MARKER_OK || { echo "Wave subsection not before the 'When routing' paragraph"; exit 1; }
for r in '/capacity' '/plan-cycle' '/plan-sprint' '/plan-iteration'; do
  grep -qF "$r" "$AG" || { echo "AGENTS.md missing route $r"; exit 1; }
done
grep -qiE 'supersede' "$AG" || { echo "AGENTS.md missing supersede language"; exit 1; }

# AC-10: references gain the new agents / skills / commands
grep -qF 'PM Wave-1 Story Queue planning:' "$AG" || { echo "Agents Reference bullet missing"; exit 1; }
grep -qF 'Story Queue planning (Wave-1):' "$AG" || { echo "Skills Reference bullet missing"; exit 1; }
for s in $newskills; do grep -qF "\`$s\`" "$AG" || { echo "Skills Reference missing $s"; exit 1; }; done
for a in $agents; do grep -qF "\`$a\`" "$AG" || { echo "Agents Reference missing $a"; exit 1; }; done

# AC-11: canonical routing table rows + prior child Wave headers byte-unchanged (still present verbatim)
grep -qF 'P1, no v1 surface' "$AG" || { echo "canonical /capacity + /plan-* v1.5 rows altered/missing"; exit 1; }
for h in 'Wave-2 adversarial critic routes' 'Wave-2 experiment & metrics routes' 'Wave-2 PRD Editor & comms routes' 'Wave-1 backing routes (story-detail-and-intake-scrubber-backing)' 'Wave-2 competitive & market-grounding routes' 'Wave-2 exec narrative & working-backwards routes' 'Wave-3 remaining roles, scrubbers & launch routes (remaining-roles-scrubbers-and-launch)'; do
  grep -qF "$h" "$AG" || { echo "prior child Wave header lost: $h"; exit 1; }
done

# AC-12: no dangling refs — every skill named in either agent's load-list resolves on disk
for s in pm-agent-doctrine capacity-planning sprint-planning cycle-planning pm-preset-detection iteration-planning shape-up-cadence; do
  test -d "domains/pm/skills/$s" || { echo "DANGLING agent load-ref: $s (no domains/pm/skills/$s/)"; exit 1; }
done
# the three new skills close the shipped forward-refs
for s in $newskills; do test -d "domains/pm/skills/$s" || { echo "forward-ref not closed: $s"; exit 1; }; done
# hill-chart-reasoning (child #10) resolves for the shape-up-cadence cross-ref
test -d domains/pm/skills/hill-chart-reasoning || { echo "hill-chart-reasoning x-ref dangling"; exit 1; }
# every backticked skill x-ref inside the three new skills resolves to a shipped or newly-authored skill dir (advisory for non-skill tokens)
for s in $newskills; do
  for ref in $(grep -oE '`[a-z0-9-]+`' "domains/pm/skills/$s/SKILL.md" | tr -d '`' | sort -u); do
    if [ -d "domains/pm/skills/$ref" ]; then continue; fi
    case " $newskills " in *" $ref "*) continue;; esac
    case "$ref" in *-*) echo "POSSIBLE dangling skill-ref '$ref' in $s (verify by hand)";; esac
  done
done

# AC-13: changes confined to domains/pm/ (no Go / schema / hero.json / consumer touched by this child)
echo "NOTE: confirm 'git diff --name-only' shows only domains/pm/ + this spec folder."

echo "VALIDATION OK"
```

The trailing `POSSIBLE dangling` lines are advisory: a hyphenated backticked
token need not be a skill slug (it may be an agent role, a spec-type name, or a
field). The delivering engineer inspects each and confirms it is not a broken
cross-reference, satisfying AC-12.

## Completion Ledger

Content-only delivery under `domains/pm/` pack source: two planning agents, three
planning skills, four command shims, and additions-only reconciliation of
`domains/pm/AGENTS.md`. No Go, no schema, no `hero.json`, no consumer-side
change. Stack: Hero PM domain-pack markdown (agents / skills / commands +
AGENTS.md routing). The spec's full `## Validation` bash block was run verbatim
from repo root under bash (the login shell is zsh, which does not word-split
unquoted `$vars`; the block opens with `set -euo pipefail` and is written for
bash) and printed `VALIDATION OK`. The six trailing `POSSIBLE dangling`
advisories are all **agent** roles (`capacity-planner`, `cycle-planner`,
`pm-delivery-lead`) backticked in skill bodies — each resolves to a real
`domains/pm/agents/<name>.md` on disk, not a broken skill cross-reference, which
is exactly the advisory case the Validation note describes (AC-12 satisfied).

**Prerequisite verification (Risks §):** `pm-foundation-delivery` reads
`status: planning` but its artifacts are on disk and engine-loadable; this child
authors no spec type / vocabulary / methodology and only consumes the existing
`feature`-rendered-as-"Story" type and the sprint/cycle/iteration presets.
`pm-doctrine-and-skill-backfill`'s `pm-agent-doctrine` spine and the AGENTS.md
`<!-- WAVE-2 ROUTES -->` marker both resolve; the new Wave subsection appends
below the marker and after every prior child's block.

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Two agent files present, non-empty, valid frontmatter (name=slug, description, mode: subagent, permission) | DONE | `domains/pm/agents/capacity-planner.md`, `cycle-planner.md` — AC-1 grep block passed |
| 2 | capacity-planner Startup names all 5 skills + describes reconciling capacity across 4 lenses → velocity cut-line | DONE | `domains/pm/agents/capacity-planner.md` `## Startup` + `## Workflow`; AC-2 grep passed (`cut-line` present) |
| 3 | cycle-planner Startup names all 7 skills + one preset-adaptive agent + reads preset + recommends + cycle-fit marker | DONE | `domains/pm/agents/cycle-planner.md` opener + `## Startup` + `## Workflow`; AC-3 grep passed (preset-adaptive + cycle-fit) |
| 4 | Three skill files present, frontmatter + metadata + 4 mandatory sections + 90–205 lines + no placeholder markers | DONE | `capacity-planning` (90), `iteration-planning` (91), `shape-up-cadence` (90); AC-4 grep+line-band block passed |
| 5 | capacity-planning: per-preset math + honest velocity distribution + WIP + cut-line | DONE | `domains/pm/skills/capacity-planning/SKILL.md`; AC-5 signature-phrase greps passed |
| 6 | iteration-planning: kanban/phased generic iteration — WIP as tool, rolling commit, phase gates | DONE | `domains/pm/skills/iteration-planning/SKILL.md`; AC-6 greps passed (kanban/WIP/phase/rolling) |
| 7 | shape-up-cadence: 6-week + cooldown, betting-table timing, hill-chart cadence, cross-refs hill-chart-reasoning | DONE | `domains/pm/skills/shape-up-cadence/SKILL.md`; AC-7 greps passed (cooldown/betting table/hill/hill-chart-reasoning) |
| 8 | Four command files route correctly, each with description + $ARGUMENTS; plan-* name preset-specific entry point | DONE | `commands/capacity.md` → capacity-planner; `plan-cycle`/`plan-sprint`/`plan-iteration` → cycle-planner; AC-8 greps passed |
| 9 | New Wave subsection below WAVE-2 marker, after prior blocks, before "When routing" para, 5 routes + supersede | DONE | `domains/pm/AGENTS.md` "Wave-1 Story Queue planning routes"; AC-9 awk placement + route + supersede greps passed |
| 10 | Two agents → Agents Reference bullet, three skills → Skills Reference bullet, four commands → PM Commands roster (line-insertion) | DONE | AGENTS.md Agents/Skills Reference bullets + PM Commands roster edited; AC-10 greps passed |
| 11 | Canonical routing table rows + every prior child Wave subsection byte-unchanged | DONE | AC-11 verbatim greps passed; `git diff` shows only the AC-10 Commands-roster in-place edit + pure additions — no canonical-table/prior-Wave-header lines touched |
| 12 | No dangling refs: every agent load-list skill resolves; three new skills close forward-refs; skill x-refs resolve | DONE | AC-12 dir-existence loops passed; 6 advisory tokens are agents on disk, not skill slugs — verified by hand |
| 13 | All changes confined to `domains/pm/` — no Go/schema/hero.json/consumer/new type-vocab-methodology | DONE | `git status --porcelain domains/pm/` shows only the 10 target paths; no changes outside `domains/pm/` |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | Author `domains/pm/agents/capacity-planner.md` | DONE | Frontmatter (name/description/mode/temp 0.1/color secondary/permission edit-allow, task deny-all, skill allow, webfetch deny); opener; 5-skill Startup; When invoked; 5-step Workflow; Anti-patterns; Default output |
| 2 | Author `domains/pm/agents/cycle-planner.md` | DONE | One-preset-adaptive framing (§C.6); permission task allows capacity-planner + prioritization-strategist, denies rest; 7-skill Startup; 3 preset entry points; delegating Workflow; Anti-patterns; Default output |
| 3 | Author `domains/pm/skills/capacity-planning/SKILL.md` | DONE | Per-preset math (sprint distribution / cycle appetite / kanban WIP+aging / phased release+gate); honest velocity; WIP-as-tool; the cut-line + worked passes; Anti-patterns; Cross-references |
| 4 | Author `domains/pm/skills/iteration-planning/SKILL.md` | DONE | Generic kanban/phased iteration; WIP as tool-not-wall; rolling commitment + worked kanban pass; phase-gate semantics + worked phased pass; Story-field population; Anti-patterns; Cross-references |
| 5 | Author `domains/pm/skills/shape-up-cadence/SKILL.md` | DONE | 6+2 rhythm + variations; cooldown non-negotiable; betting-table timing; hill-update cadence + cycle-fit-marker timing; worked cadence; interlock diagram; Anti-patterns; Cross-references incl. hill-chart-reasoning |
| 6 | Author `domains/pm/commands/capacity.md` | DONE | Thin router → capacity-planner; required-arg; what-lands (cut-line, proposal not auto-commit); `Request: $ARGUMENTS` |
| 7 | Author `plan-cycle.md`, `plan-sprint.md`, `plan-iteration.md` → cycle-planner | DONE | Three shims, each names its preset-specific entry point (shape-up / scrum / kanban-phased), notes preset is authoritative via pm-preset-detection, recommends-never-auto-commits, ends `Request: $ARGUMENTS` |
| 8 | `domains/pm/AGENTS.md` — additions only (new Wave subsection + Agents/Skills Reference bullets + Commands roster) | DONE | Wave subsection inserted before "When routing" para; Agents Reference + Skills Reference bullets before their Core bullets; PM Commands roster extended with the 4 commands; canonical table + prior Wave blocks untouched |

### Exercise-the-feature check

- [x] Validation exercised end-to-end: the spec's full `## Validation` bash block run verbatim from repo root under bash → `VALIDATION OK` (all AC-1…AC-13 gates), with only advisory agent-role notes.
- [x] AGENTS.md diff confirmed additions-only: `git diff` shows one intended in-place edit (the AC-10 PM Commands roster bullet) plus pure insertions; canonical routing-table rows and all seven prior-child Wave headers verified present verbatim.
- [x] Scope confinement confirmed: `git status --porcelain domains/pm/` shows exactly the 2 agents + 3 skill dirs + 4 commands + AGENTS.md — nothing outside `domains/pm/`.
- [ ] Not exercisable in this environment: live hero-code rendering of the Story Queue velocity cut-line / cycle-fit marker, and live agent invocation via `/capacity` / `/plan-*` — these require the hero-code consumer + a configured PM workspace, which are out of this content-only child's scope. The agents/skills/commands are authored to the shipped shapes and resolve on disk; runtime binding is the consumer's responsibility (Boundaries §).

### Excellence Bar self-check

Yes — a senior engineer who cares about this codebase would be proud to ship this. The two agents and three skills match the shipped voice, worked-example density, and doctrine discipline of `sprint-planning` / `cycle-planning` / `stakeholder-communicator` rather than being thin scaffolds: every skill carries per-preset numeric worked passes, a real Anti-patterns section, and resolvable cross-references; both agents are decision-gated (recommend, never auto-commit) per `pm-agent-doctrine`, and `cycle-planner` is correctly the single preset-adaptive agent (§C.6) with delegating permissions rather than three near-duplicate files. The three new skills close the `(P1, ships v1.5)` forward-refs the shipped `sprint-planning`/`cycle-planning` carried, so the pack has no dangling planning refs left. The AGENTS.md reconciliation is genuinely additions-only against the canonical table and every prior child's Wave block, following the exact supersede pattern children #7/#8 established — the final pm-pack-completion child lands without disturbing the ten before it.
