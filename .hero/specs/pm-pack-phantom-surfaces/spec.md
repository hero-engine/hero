---
title: "PM Pack Phantom Surfaces — fix every reference to a surface that doesn't exist"
slug: pm-pack-phantom-surfaces
type: bug
status: completed
priority: P1
size: medium
domain: engineering
created: 2026-07-06
tags: [pm-pack, content-audit, phantom-surfaces, routing, handoff]
relations:
  - { target: content-remediation, kind: parent }
  - { target: hero-content-audit, kind: related }
delivery_method: manual
completed_at: 2026-07-06T20:58:30Z
---

# PM Pack Phantom Surfaces — fix every reference to a surface that doesn't exist

## Context

The hero-content-audit (base SHA `bc86ad9`) found that the PM pack systematically
references surfaces that don't exist or are wrong. Evidence lives in
`.hero/specs/hero-content-audit/`:

- **findings-routing.md** — `domains/pm/AGENTS.md` routes 10+ slash commands that
  don't install with pm+core (`/interview`, `/capacity`, `/plan-cycle`,
  `/plan-sprint`, `/plan-iteration`, `/standup`, `/scrub roadmap|intake|specs`,
  `/diagnose`, `/search`, `/review`); teaches six wrong CLI invocations
  (`hero event`, `hero active register|list`, `hero queue --owner`, `hero new
  feature|bug|epic|initiative`, `hero import` as tracker import, `hero search
  --kind`); names two ghost agents (`pitch-author`, `cycle-planner`); points
  installed users at hero-engine source paths in Project Structure; misplaces
  `hero.json`; and carries a duplicate `/refine` routing row.
- **findings-agents.md** — `pm-delivery-lead`'s `permission.task` allowlist and
  specialist table route to 8 agents that don't ship (`pitch-author`,
  `epic-framer`, `dependency-mapper`, `risk-curator`, `metrics-analyst`,
  `cycle-planner`, `capacity-planner`, `stakeholder-communicator`);
  `handoff-coordinator`'s mechanics are built on surfaces that don't exist
  (`.hero/planning/specs/` path, a false "updating `owner` causes the history
  append" mechanism, `hero event` shell syntax, `hero queue --owner --status`,
  non-engine statuses); PM lifecycle vocabulary (`drafted`/`refined`/`ready`/
  `handed-off`/`shipped`/`candidate`/`committed`/...) doesn't match engine
  statuses — systemic across story-writer, prd-author, pm-delivery-lead,
  pm-reviewer, roadmap-curator, handoff-coordinator; and pm agents reference
  engineering-only skills/commands (`kickoff-prompt`, `/review`, `/scrub`,
  `/deliver`, `/interview`) that a `--domain pm` install doesn't ship.
- **findings-skills.md** — 11 phantom skill refs and 9 phantom agent refs across
  pm skills; `handoff-protocol`'s dead CLI; `intake-classification`'s
  `hero search --themes`; `acceptance-criteria-ears` asserting the phantom
  `acceptance-criteria-gherkin` exists; unified-type-model told inconsistently
  (`handoff-protocol`, `dependency-mapping`, `cross-domain-graph-query`).
- **findings-commands.md** — F2, F3, F7, F18 (pm `/handoff`'s
  `planning/specs/` path), F19 (README reused-commands errors), F20
  (release-notes' self-retracting v1.5 skill), F25 (pm commands leading with
  ghost agents), F26 (pm `/discover`'s `planning/roadmap/` path).

**Current-state deltas since the audit, verified against the tree and binary at
HEAD (`3aaad62`):**

- Commit `177e8a1` (content-dedup-resync) made `core/` the single master for the
  34 core↔engineering duplicates and added `content_parity_test.go`. All paths
  in Changes below were re-verified to exist today.
- `hero new` no longer exists at the CLI root at all — scaffolding is
  `hero spec new <slug> --type <t>` (alias `hero design <slug>`). Supported
  types: feature (default), bug, initiative, intake, convention, decision, rule,
  external, context, note — **no `epic`** (`internal/cli/new.go:136-162`,
  registered under `spec` at `internal/cli/spec.go:51`).
- The events CLI is `hero agent events <type> <message> [--slug]` with valid
  types `spec_created, spec_updated, files_modified, decision_made,
  blocker_hit, delivery_complete` — note there is **no `handoff` event type**,
  so `hero event handoff ...` is doubly phantom. MCP `hero_event` and
  `hero_active` exist (`internal/serve/mcp_tools_def.go:465,478`).
- `hero spec set-owner <slug> <owner>` is the real owner-flip surface: it flips
  `owner:`, appends the `owner_history` row, and rewrites spec.md atomically.
  A raw frontmatter edit records **no** history row.
- `hero queue` flags: `--format --horizon --limit --subproject` only.
  `hero list` filters: `--status --type --tag --mine --ready --blocked
  --horizon --stale --pinned` — no `--owner` anywhere.
- Engine work-spec statuses (`internal/spec/spec.go:40-90`): `planning`,
  `in-review`, `delivering`, `completed`, `regressed`; intake: `triaged`,
  `promoted`, `rejected`, `merged`; shared: `superseded`. The `handed_off` /
  `awaiting_peer` / `handed_back` family is **cross-repo peering** semantics,
  not the pm→engineering owner flip.
- `hero search` FTS5 flags: `--type --status --tag --list ...` — no `--kind`,
  no `--themes`. `--tag` is the real theme filter.
- `kickoff-prompt` still lives only in `domains/engineering/skills/` — a
  `--domain pm` install (OverlayFS(pm, core)) does not receive it, yet
  `prd-author` and `pm-delivery-lead` require loading it.

## Goal

Every surface named anywhere in `domains/pm/` (AGENTS.md, 12 agents, 11
commands + README, 19 skills + README, spec-types) exists in a `--domain pm`
install or is explicitly scoped as "(P1, ships v1.5)" / engineering-side. Every
CLI invocation in the pack runs as written against the current binary. The
handoff mechanics describe the real owner-flip surface (`hero spec set-owner`).
PM lifecycle vocabulary maps onto engine statuses in exactly one skill that the
agents cite. A cold PM session can follow any instruction in the pack without
hitting an unknown command, a nonexistent agent, or a dead path.

## Kickoff

Fixes everything in the PM pack that references surfaces that don't exist —
dead slash routes, six wrong CLI invocations, ghost agents in allowlists,
broken owner-flip mechanics, and non-engine lifecycle statuses.

**Status:** planning — spec authored from hero-content-audit findings; all
paths and CLI claims re-verified at `3aaad62`; no edits yet.

**Pick up at:** Change 1 (lifecycle mapping in `pm-preset-detection`) and
Change 2 (promote `kickoff-prompt` to core), then sweep `domains/pm/AGENTS.md`
(Changes 3–5).

→ `/deliver pm-pack-phantom-surfaces`

**Files:** domains/pm/AGENTS.md, domains/pm/agents/pm-delivery-lead.md,
domains/pm/agents/handoff-coordinator.md,
domains/pm/skills/pm-preset-detection/SKILL.md

## Approach

Four principles govern every edit:

1. **Prefer real surfaces over deletion.** Where the capability exists under a
   different name, repoint rather than remove: `hero event` → `hero agent
   events` (CLI) or `hero_event` (MCP); `hero active` → `hero_active` (MCP);
   owner flip → `hero spec set-owner`; `hero import` → `hero sync import`;
   `hero new <type>` → `hero spec new <slug> --type <t>`; queue-by-owner →
   `hero list --status <s>` plus reading `owner:` from the spec; `/interview` →
   `/discover --interview <count>` (a real mode of pm's own `/discover`);
   `/scrub roadmap` → `/roadmap` (reconcile); `/search` → `hero search`.
2. **Ghost agents and skills get "(P1, ships v1.5)" scoping or removal.**
   Allowlists (`permission.task`) are executable config, not documentation —
   they must contain **only** shipped agents. Prose and tables may keep a
   scoped forward reference where the pm READMEs already frame v1.5, but every
   such row also names the shipped v1 fallback.
3. **Lifecycle vocabulary is defined once.** A "PM lifecycle → engine statuses"
   section is added to `domains/pm/skills/pm-preset-detection/SKILL.md` (the
   skill the authoring agents and pm-delivery-lead are already required to
   load; it is the pack's "how PM process maps onto the engine" home). All six
   agents that enumerate statuses cite it instead of restating divergent sets.
   Proposed canonical mapping (delivery may tune wording, not the status set):
   `drafting`/`drafted`/`shaping`/`refining` → `planning`; `refined`/`ready`
   (reviewed, handoff-eligible) → `in-review`; engineering claimed →
   `delivering`; `shipped` → `completed`; `dropped` → `superseded` (work
   specs) / `rejected` (intake); initiative `candidate` → `planning`,
   `committed` → `delivering`, `shipped` → `completed`. The section explicitly
   warns that `handed_off`/`handed_back` are cross-repo peering statuses, not
   the pm→engineering owner flip (which is an `owner:` change, not a status).
4. **Engineering-only references get repointed or scoped.** `kickoff-prompt` is
   promoted to `core/skills/` (single master; the engineering install is
   unchanged because the overlay falls through, and `hero queue` renders
   `## Kickoff` on every domain, so the skill is domain-universal).
   `/deliver`, `/design`, `/diagnose`, `/review`, `/scrub` references that
   describe the engineering side of the boundary stay, but gain "engineering
   pack" scoping; references that tell a *pm session* to run them are repointed
   to pm-shipped surfaces (pm-investigator, pm-reviewer, `/triage`, `/roadmap`,
   `/refine`).

Out of scope by design: rewriting content for token efficiency, restructuring
heading depth, and any Go code changes (no CLI shims are added — the pack is
corrected to the surfaces that exist).

## Changes

1. **Add the canonical lifecycle mapping —
   `domains/pm/skills/pm-preset-detection/SKILL.md`**
   - Append a `## PM lifecycle vocabulary → engine statuses` section carrying
     the mapping table from Approach §3, the "handed_off is cross-repo, not
     the owner flip" warning, and one sentence: "This table is the single
     source for status vocabulary; agents cite it, they don't restate it."
   - Fix frontmatter `metadata.audience` (line 6): remove ghosts
     `epic-framer`, `cycle-planner`, `capacity-planner`, `pitch-author`; keep
     `prd-author, story-writer, roadmap-curator, pm-delivery-lead`.
   - Rewrite body ghost refs: line 50 (`capacity-planner` or `cycle-planner`
     needs the math model) and line 131 (`epic-framer` reads the preset) to
     name `pm-delivery-lead` as the v1 owner with "(P1 planners ship v1.5)".

2. **Promote `kickoff-prompt` to core —
   `git mv domains/engineering/skills/kickoff-prompt core/skills/kickoff-prompt`**
   - Single master in core; engineering installs are unchanged (overlay falls
     through), pm/sales installs now receive the skill `prd-author` and
     `pm-delivery-lead` are told to load.
   - In the moved SKILL.md, scope the engineering command names in "When to
     write or update me" (`/design`, `/deliver`, `/diagnose`) with one
     parenthetical: "(engineering pack; in pm, `prd-author` and `story-writer`
     write the kickoff as part of authoring)".
   - `content_parity_test.go` is satisfied (no same-named pair remains).

3. **Fix the routing table — `domains/pm/AGENTS.md` (Natural Language Routing,
   lines 21-52)**
   - Repoint real capabilities: `/interview` row (line 36) →
     `/discover --interview <count>`; `/scrub roadmap` (41) → `/roadmap`
     (reconcile mode); `/scrub intake` (42) → `/triage` (duplicate clustering
     is intake-triager + duplicate-detector); `/scrub specs` (43) → `/refine`;
     `/diagnose` (44) → "invoke `pm-investigator` directly (agent, no command
     shim)"; `/search` (46) → `hero search <query>` (CLI); `/review` (51) →
     "invoke `pm-reviewer` directly".
   - Delete rows with no v1 surface: `/capacity` (38), `/plan-cycle` /
     `/plan-sprint` / `/plan-iteration` (39), `/standup` (40) — or keep a
     single collapsed row marked "(P1, ships v1.5 — no v1 surface)".
   - Merge the duplicate `/refine` rows (lines 24 and 45) into one.
   - Replace the four `hero new <type>` rows (27-30): feature →
     `hero spec new <slug> --type feature` (alias `hero design <slug>`); bug →
     `--type bug`; initiative → `--type initiative`; epic → "no CLI scaffolder
     (`hero spec new` has no epic type) — hand-author
     `.hero/planning/epics/<slug>/spec.md` per `core/spec-types/epic.md`".
     Apply the same four corrections to the vocabulary-aware table (69-72).

4. **Fix CLI and compaction mechanics — `domains/pm/AGENTS.md`**
   - "Log significant events" (109-112): replace `hero event <type> "..."
     --slug` with `hero agent events <type> "..." --slug` and swap the invalid
     `handoff` type for `spec_updated`; mention `hero_event` (MCP) as the
     tool-call equivalent.
   - "The handoff is an owner flip" (149-159): step 1 gate becomes
     `status: in-review` per the Change-1 mapping; step 2 becomes "flip via
     `hero spec set-owner <slug> engineering` — this appends the
     `owner_history` row; a raw frontmatter edit records no history"; step 3
     drops `hero queue --owner engineering` — verification is reading the spec
     back (`hero_read_spec` MCP or the file) plus `hero list --status
     in-review`; step 4 keeps `/deliver` but scopes it "(engineering pack)".
   - CLI Commands (178-180): `hero import` → `hero sync import` (root
     `hero import` ingests URLs/files into the knowledge base — say so).
   - Important Rules (241-243): drop `--kind=…` (no such flag); keep
     `hero search --list --type feature|prd|epic|initiative|intake`.
   - "Survive context compaction" (293-298): replace `hero active register` /
     `hero active list` with the `hero_active` MCP tool (register/list are its
     actions), phrased harness-neutrally; keep `hero recap --since 1h`.

5. **Fix structure, config path, and dead link — `domains/pm/AGENTS.md`**
   - Project Structure (192-201): replace `domains/pm/*` and `core/*` source
     paths with engineering's placeholder convention — `<harness>/agents/`,
     `<harness>/skills/`, `<harness>/commands/` (see
     `domains/engineering/AGENTS.md:72-84` for the model, including the
     "hero install writes these" sentence); keep the `.hero/planning/*` rows.
   - Line 210: `hero.json` → `.hero/hero.json`.
   - Line 237: replace the relative link
     `../../.hero/knowledge/decisions/tracker-fronting-and-local-first.md`
     with the plain workspace path
     `.hero/knowledge/decisions/tracker-fronting-and-local-first.md`; line
     239's `core/vocabularies/*.yaml` mention → "the active vocabulary
     preset" (no source path).
   - Lines 91-92 (Methodology presets): reword "Authoring agents (`prd-author`,
     `story-writer`, `pitch-author`) and the `cycle-planner` agent" to the
     shipped roster: "Authoring agents (`prd-author`, `story-writer`) must
     load `pm-preset-detection` (P1 agents `pitch-author` and `cycle-planner`
     join in v1.5)".

6. **Trim pm-delivery-lead to shipped surfaces —
   `domains/pm/agents/pm-delivery-lead.md`**
   - Frontmatter `permission.task` (lines 22-29): delete all 8 ghost `allow`
     entries (`pitch-author`, `epic-framer`, `dependency-mapper`,
     `risk-curator`, `metrics-analyst`, `cycle-planner`, `capacity-planner`,
     `stakeholder-communicator`). The allowlist ends at the shipped 11
     subagents.
   - Specialist table (82-95): rewrite the 8 ghost rows to their v1 owners —
     pitch → `prd-author` (pitch template) "(pitch-author P1)"; epic framing →
     `pm-delivery-lead` direct "(epic-framer P1)"; capacity + cycle/sprint
     planning → `pm-delivery-lead` + `sprint-planning`/`cycle-planning` skills
     "(planners P1)"; dependencies → `pm-delivery-lead` + `dependency-mapping`
     skill; risks → `prd-author` (Risks section); metrics →
     `pm-delivery-lead` + `metrics-design` skill "(metrics-analyst P1)";
     stakeholder translation → `pm-delivery-lead` "(stakeholder-communicator
     P1)".
   - Line 117 and line 142: replace the freeform status chains with engine
     statuses and a cite: "per the lifecycle table in `pm-preset-detection`".
   - Lines 118 and 144: `hero event ...` → `hero agent events ...` (valid
     types only) or `hero_event` MCP.
   - Line 136: "status `ready → delivering`" → "status `in-review →
     delivering`".

7. **Rewrite handoff-coordinator mechanics on real surfaces —
   `domains/pm/agents/handoff-coordinator.md`**
   - Line 53: `.hero/planning/specs/<slug>/spec.md` → "resolve the slug
     (`hero_read_spec` MCP or `hero search --list`); pm specs live under
     `.hero/planning/{features,bugs,epics,prds,intake}/<slug>/spec.md`".
   - Lines 56-58: pre-flight gate `status: ready` (not `drafted`/`refined`) →
     `status: in-review` (not `planning`), citing the Change-1 mapping.
   - Lines 80-92 (step 2): replace the "updating `owner` causes the history
     append" mechanism with `hero spec set-owner <slug> engineering` —
     appends the `owner_history` row atomically; the read-back verification
     stays and now actually passes.
   - Line 104: `hero event handoff "..." --slug` → `hero agent events
     spec_updated "owner flipped pm → engineering" --slug <slug>` (or
     `hero_event` MCP); line 107's prose reference likewise.
   - Lines 118-124 (step 4): drop `hero queue --owner engineering --status
     ready`; verify pickup by re-reading the spec — `owner: engineering` on
     disk, then `status: delivering` after engineering claims (`hero list
     --status delivering` as the sweep). Scope `/deliver` "(engineering
     pack)".
   - Line 135: `hero queue --owner pm` → "the spec shows `owner: pm` again
     with a `handed_back_reason:`".
   - Line 148 (Produces): `ready → delivering` → `in-review → delivering`.

8. **Same-surface fixes in the pm `/handoff` command —
   `domains/pm/commands/handoff.md`**
   - Pre-flight #1: `status: ready` / `drafted` / `refined` → engine statuses
     (`in-review` gate; `planning` isn't shippable), citing the mapping.
   - Workflow step 1 (line 28) and Output (line 52):
     `.hero/planning/specs/<slug>/spec.md` → slug-resolved real planning dirs
     as in Change 7.
   - Step 2: flip via `hero spec set-owner <slug> engineering`.
   - Step 4 (line 31): `hero event handoff ...` → `hero agent events
     spec_updated ... --slug <spec-slug>`.
   - Step 5 (line 32) and Failure modes (line 48): drop `hero queue --owner
     engineering --status ready`; verify via spec read-back / `hero list
     --status delivering`; scope `/deliver` "(engineering pack)"; `ready →
     delivering` → `in-review → delivering`.

9. **Kill the dead CLI in handoff-protocol —
   `domains/pm/skills/handoff-protocol/SKILL.md`**
   - Verification steps (lines 160-170): step 3's `hero event handoff ...` →
     `hero agent events spec_updated ...`; step 4's `hero queue --owner
     engineering --status ready` → spec read-back + `hero list --status
     in-review`; step 5's `ready → delivering` → `in-review → delivering`.
   - Hand-back path (line 183): `hero queue --owner pm` → "the spec's
     `owner:` is `pm` again (read it back)".
   - Add the owner-flip surface once, where the flip is described:
     `hero spec set-owner <slug> engineering`.
   - Scope the three `/deliver` references (lines 52, 170-174, 209) as
     "(engineering pack)". Status vocabulary cites the Change-1 mapping.

10. **Normalize the type model where it contradicts the engine —
    `domains/pm/skills/cross-domain-graph-query/SKILL.md` and
    `domains/pm/skills/dependency-mapping/SKILL.md`**
    - cross-domain-graph-query lines 30 and 90: the engine has no `spec` type
      — registered types are `feature`, `bug`, `initiative`, etc. Rewrite "the
      *type* is the same (`spec`)" to "the *artifact* is the same (`type:
      feature`) — the `owner:` field, not the type, marks the domain", keeping
      the (correct) "no separate `kind: handoff` edge" claim.
    - cross-domain-graph-query lines 98-100: `hero_why feature:X` →
      `hero_why <slug>` (the tool takes a slug, not a `type:` prefix).
    - cross-domain-graph-query line 6: remove ghost `dependency-mapper` from
      `metadata.audience`.
    - dependency-mapping lines 86-88: rewrite the `story:enable-saml` /
      `blocked-by feature:saml-provider` example tree in slug + frontmatter
      terms (`enable-saml [in-review] depends-on saml-provider [delivering]`);
      line 6 audience: drop `dependency-mapper`, `cycle-planner`; lines 18 and
      97: `cycle-planner`/`capacity-planner` → `pm-delivery-lead` "(planners
      P1)".

11. **Sweep the remaining agent files —**
    - `domains/pm/agents/story-writer.md` §7 (lines 145-154): replace the
      seven-state list with engine statuses + cite the mapping; line 150
      `/deliver` scoped "(engineering pack)"; line 92 `/diagnose` →
      "engineering's `/diagnose` (engineering pack)". Line 137's Gherkin
      scoping is already correct — leave it.
    - `domains/pm/agents/prd-author.md` §8 (lines 147-153): same treatment
      (`drafting`/`refining` → `planning`, `ready` → `in-review`,
      `handed-off` → children's `owner:` flipped, `shipped` → `completed`) +
      cite; line 92 `risk-curator` → "(P1; v1: prd-author owns Risks)"; line
      162 `metrics-analyst` → "(P1; v1: `metrics-design` skill via
      pm-delivery-lead)". Lines 30/143 `kickoff-prompt` need no edit after
      Change 2.
    - `domains/pm/agents/pm-reviewer.md`: line 32 `/review` → "review requests
      routed per the AGENTS.md table (no `/review` command ships with pm)";
      lines 33-34 `status: ready` / `candidate → committed` → engine statuses
      + cite.
    - `domains/pm/agents/roadmap-curator.md`: line 26 `/scrub roadmap` →
      `/roadmap` (reconcile); lines 44-45 `hero event decision_made` →
      `hero agent events decision_made`; status transitions (36, 50) keep the
      roadmap vocabulary but add one cite line to the mapping (`committed` ↔
      `delivering`, `shipped` ↔ `completed`).
    - `domains/pm/agents/intake-triager.md`: line 67 `/scrub intake` → "a
      `/triage` sweep finding".
    - `domains/pm/agents/discovery-researcher.md`: line 30 `/interview` →
      `/discover --interview`.
    - `domains/pm/agents/product-strategist.md`: line 61 `metrics-analyst` →
      "(P1; v1: `metrics-design` skill)".
    - `domains/pm/agents/prioritization-strategist.md`: line 26
      `cycle-planner` → `pm-delivery-lead` cycle planning "(P1)"; line 58
      ghost list → "`discovery-researcher`, or pm-delivery-lead loading
      `metrics-design` / `sprint-planning` (P1 specialists ship v1.5)".

12. **Sweep the remaining command files —**
    - `domains/pm/commands/README.md`: refine row (line 14) — mark
      `epic-framer` "(P1, ships v1.5)" like its siblings; Reused section —
      `/search` → "`hero search` (CLI; no pack ships a `/search` command)";
      `/deliver` → scope "engineering-pack command — runs on the engineering
      side after the owner flip, not in a pm install".
    - `domains/pm/commands/metrics.md` (line 4), `pitch.md` (line 4),
      `refine.md` (line 20): invert to lead with the v1 owner ("Route to
      `pm-delivery-lead` loading `metrics-design`…"; "Route to `prd-author`
      with the pitch template…"; epic → `pm-delivery-lead` direct), footnoting
      "(`metrics-analyst`/`pitch-author`/`epic-framer` take over in v1.5)".
    - `domains/pm/commands/release-notes.md` (line 4): state only the v1
      behavior — `pm-delivery-lead` drafting from the template in this command
      file; delete the `stakeholder-communication` skill load and its
      self-retracting parenthetical; footnote "(stakeholder-communicator +
      skill ship v1.5)".
    - `domains/pm/commands/discover.md` (line 20):
      `.hero/planning/roadmap/<slug>/research/` →
      `.hero/planning/initiatives/<slug>/research/`.
    - `domains/pm/commands/triage.md`: line 19 "invoke `pm-investigator` first
      via `/diagnose`" → "invoke `pm-investigator` directly (agent — no
      command shim ships with pm)"; line 33 example "refiled via /diagnose" →
      "refiled to engineering as a bug"; line 38 `hero event decision_made` →
      `hero agent events decision_made`.
    - `domains/pm/commands/prioritize.md` (line 30): `hero event
      decision_made` → `hero agent events decision_made`.

13. **Sweep the remaining skill files (phantom refs → P1-scope or repoint) —**
    - `intake-classification/SKILL.md`: line 34 `hero search --themes` →
      `hero search --list --tag <theme>`; line 27 `portfolio-curator` →
      `roadmap-curator`; line 29 `domain-glossary-maintenance` → "(P1, ships
      v1.5)"; line 87 `/scrub intake` → "a `/triage` sweep".
    - `acceptance-criteria-ears/SKILL.md`: line 178 — "(
      `acceptance-criteria-gherkin` exists as an alternate AC skill…)" →
      "(`acceptance-criteria-gherkin` is planned for v1.5 — until it ships,
      stay in EARS)", matching story-writer.md:137.
    - `pitch-writing-shape-up/SKILL.md`: lines 279-280 `shape-up-cadence`,
      `hill-chart-reasoning` → "(P1, ships v1.5)"; line 6 audience drop
      `pitch-author`; lines 18/264 `pitch-author` → "(P1; v1: `prd-author` in
      pitch shape)".
    - `cycle-planning/SKILL.md`: lines 182-183 same two phantom skills →
      "(P1)"; line 6 audience drop `cycle-planner`, `pitch-author`; line 65
      `pitch-author` → "(P1; v1: prd-author)".
    - `roadmap-framing/SKILL.md`: lines 198/203 `outcomes-over-outputs`,
      `risk-surfacing` → drop the skill-style refs (line 45's book citation
      stays); line 105 `stale-roadmap-scrubber` → `roadmap-curator`.
    - `sprint-planning/SKILL.md`: lines 162/165 `iteration-planning`,
      `capacity-planning` → "(P1, ships v1.5)"; line 6 audience →
      `pm-delivery-lead`; line 35 `capacity-planner` → "a capacity model".
    - `continuous-discovery-cadence/SKILL.md` (155-156) and
      `opportunity-solution-trees-torres/SKILL.md` (185-186):
      `discovery-interview-design`, `assumption-testing` → "(P1, ships
      v1.5)".
    - `prd-structure/SKILL.md`: line 167 `stakeholder-communication` → "(P1,
      ships v1.5)"; lines 6/18 `pitch-author` → drop from audience / "(P1)".
    - `metrics-design/SKILL.md` line 6: audience `metrics-analyst` →
      `pm-delivery-lead` (keep `prd-author`, `roadmap-curator`).
    - `evidence-synthesis/SKILL.md`: line 6 audience drop
      `competitive-analyst`, `metrics-analyst`; lines 20-21 → "(P1)" scoping.
    - `story-writing-invest/SKILL.md`: line 6 audience drop `epic-framer`;
      line 132 "Promote to an epic with the `epic-framer`" → "Promote to an
      epic (v1: `pm-delivery-lead` frames it; `epic-framer` ships v1.5)".
    - `duplicate-detection/SKILL.md`: line 22 `/scrub intake` → "a `/triage`
      dedup sweep".

14. **Scope the pm spec-type docs — `domains/pm/spec-types/intake.md`**
    - Line 116: "route to `/diagnose`" → "route to engineering's `/diagnose`
      (engineering pack)"; line 154: `/scrub intake` → "a `/triage` sweep
      finding". (`prd.md:153-154` already carries correct "(P1)" scoping —
      no edit.)

## Boundaries

- **Sales pack** (`domains/sales/`) phantom CLI, fake hero.json schema,
  unloadable `deal.yaml` — sibling spec `sales-pack-reality-sync`.
- **Routing-file completeness and structure** — engineering AGENTS.md's 11
  unrouted commands, `core/commands/hero.md` drift, the parity table living
  only in installed CLAUDE.md, heading-depth divergence across the three
  AGENTS.md files — sibling spec `routing-file-completeness`. This spec only
  *removes falsehoods* from pm's AGENTS.md; it does not add missing rosters.
- **Token efficiency** — pm AGENTS.md re-teaching three skills inline
  (~500 words), skill-body duplication (pitch/cycle, prd-structure triplets,
  OST/cadence) — sibling spec `token-efficiency-pass`. Where this spec rewrites
  a sentence it may get shorter, but no compression pass is in scope.
- **Core commands delegating to engineering-only agents** (findings-commands
  F4 — `core/commands/{check,decide,discover,retro,drive,hero}.md`) affects pm
  installs but is core content, not pm-pack content — out of scope here.
- **Go code changes** — no `hero event` CLI shim, no `--owner` flag on
  `hero queue`/`hero list`, no README exclusion in `installFlat`, no `epic`
  type in `hero spec new`. The pack is corrected to today's engine.
- **Chat pack** (dead content, F9) — untouched.
- The only file touched outside `domains/pm/` and `core/skills/` is the
  `git mv` of `kickoff-prompt` (Change 2); no engineering-pack content is
  otherwise edited.

## Risks

- **The lifecycle mapping is a judgment call.** Collapsing `refined`/`ready`
  into `in-review` changes the documented handoff gate. The mapping table is
  the deliberate single place to argue about it — if delivery finds a better
  fit (e.g. `ready` staying `planning` + a review flag), change the table, not
  the citing agents.
- **`hero spec set-owner` owner roles** are `pm | engineering | qa | devops |
  design | docs` — consistent with the pack's flip today, but verify the
  command against a real pm workspace during delivery (it synthesizes history
  from mtime on first flip; the read-back step in the rewritten docs covers
  this).
- **Moving `kickoff-prompt` to core** ships it to sales/chat installs too.
  Content is domain-neutral after the Change-2 scoping edit, but
  `content_parity_test.go` and any install-snapshot tests must pass; if a test
  pins the engineering skill count, update the fixture, not the move.
- **Line-number drift.** Anchors above were verified at `3aaad62`; if pm files
  move under a concurrent sibling, match on the quoted strings, not the line
  numbers.
- **P1 scoping can rot in the other direction** — when the v1.5 agents ship,
  every "(P1, ships v1.5)" marker becomes stale. Acceptable: the markers are
  greppable (`rg "P1, ships v1.5" domains/pm`), and un-scoping is mechanical.

## Acceptance Criteria

- THE SYSTEM SHALL contain no routing row in domains/pm/AGENTS.md that targets a slash command absent from the pm+core merged command set.
- THE SYSTEM SHALL invoke only CLI commands and flags that exist in the current `hero` binary in every file under domains/pm/ (verified per invocation against `hero <cmd> --help`).
- WHEN a pm file logs an event THE SYSTEM SHALL use `hero agent events <type>` with a type from the valid set (spec_created, spec_updated, files_modified, decision_made, blocker_hit, delivery_complete) or the `hero_event` MCP tool.
- THE SYSTEM SHALL describe the pm→engineering owner flip exclusively via `hero spec set-owner <slug> engineering` wherever domains/pm/ content states the flip mechanism.
- THE SYSTEM SHALL list only shipped agents in pm-delivery-lead's `permission.task` allowlist.
- WHEN a pm file names an unshipped agent or skill THE SYSTEM SHALL scope it with a "(P1, ships v1.5)"-style marker and name the shipped v1 fallback.
- THE SYSTEM SHALL define the PM-lifecycle-to-engine-status mapping in exactly one pm skill (pm-preset-detection), and every domains/pm/ agent that enumerates lifecycle statuses SHALL cite it.
- THE SYSTEM SHALL reference pm spec paths only under .hero/planning/{features,bugs,epics,initiatives,prds,intake}/ (no planning/specs/, no planning/roadmap/) in domains/pm/ content.
- THE SYSTEM SHALL use `<harness>/` placeholders in pm AGENTS.md Project Structure and locate configuration at `.hero/hero.json`.
- IF a pm file references an engineering-only command (/deliver, /design, /diagnose, /review, /scrub) THEN THE SYSTEM SHALL either scope the reference as engineering-side or repoint it to a pm-shipped surface.
- THE SYSTEM SHALL resolve the `kickoff-prompt` skill in a `--domain pm` install (skill present at core/skills/kickoff-prompt/SKILL.md, engineering copy removed).
- THE SYSTEM SHALL pass `go test ./...` including content_parity_test.go after all edits.

## Validation

1. **Grep gates (all must return empty over `domains/pm/`):**
   - `rg -n "hero event |hero active|queue --owner|planning/specs|planning/roadmap|hero new |--kind=|--themes|hero import\`" domains/pm` — dead CLI and dead paths gone (the one legitimate `hero sync import` mention must be spelled with `sync`).
   - `rg -n "pitch-author|epic-framer|dependency-mapper|risk-curator|metrics-analyst|cycle-planner|capacity-planner|stakeholder-communicator|competitive-analyst|portfolio-curator|stale-roadmap-scrubber" domains/pm | rg -v "P1|v1.5"` — every ghost-agent mention is scoped or gone; separately confirm `domains/pm/agents/pm-delivery-lead.md` frontmatter has zero ghost entries.
   - `rg -n "shape-up-cadence|hill-chart-reasoning|outcomes-over-outputs|risk-surfacing|stakeholder-communication|discovery-interview-design|assumption-testing|iteration-planning|capacity-planning|domain-glossary-maintenance|acceptance-criteria-gherkin" domains/pm | rg -v "P1|v1.5"` — phantom skills scoped or gone.
   - `rg -n "status: ready|drafted|handed-off|→ refined" domains/pm/agents domains/pm/commands domains/pm/skills` — no non-engine statuses outside the mapping table.
2. **CLI spot-checks:** run each invocation that survives in the pack verbatim
   (with a placeholder slug) against the binary: `hero agent events
   decision_made "x" --slug y`, `hero spec new y --type feature`, `hero spec
   set-owner y engineering`, `hero sync import`, `hero search --list --tag x`,
   `hero list --status in-review` — none may print "unknown command" or
   "unknown flag".
3. **Install smoke:** `hero install project <tmpdir> --target claude --domain
   pm`, then verify every slash command, agent, and skill named in the
   installed AGENTS.md/CLAUDE.md managed body exists in the installed tree —
   including `kickoff-prompt` under the harness skills dir.
4. **Tests:** `go test ./...` — content parity, install, and drift tests green.
5. **Cold read:** open `domains/pm/agents/handoff-coordinator.md` and execute
   its workflow steps 1-4 mentally against a scratch pm workspace spec — every
   step names a runnable surface and the read-back verification can succeed.

## Completion Ledger

Delivered as a content-remediation sweep of `domains/pm/` (plus the `core/`
promotion of `kickoff-prompt`). No Go code changed. Base: `40a3b85`. All CLI
surfaces re-verified against a freshly-built binary (line numbers had drifted
from `3aaad62`, so edits matched on quoted strings per the Risks section).

### Acceptance Criteria

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | No AGENTS.md routing row targets an absent slash command | DONE | `domains/pm/AGENTS.md` routing + vocabulary tables rewritten; `/interview`→`/discover --interview`, `/scrub *`→`/roadmap`/`/triage`/`/refine`, `/diagnose`/`/review`→direct agent, `/search`→`hero search`, `/capacity`/`/plan-*`/`/standup` collapsed to a scoped v1.5 row. Untouched targets (`/why`,`/blocked`,`/note`,`/decide`,`/retro`) confirmed present in `core/commands/`. |
| 2 | Only existing CLI commands/flags in every `domains/pm/` file | DONE | Grep gate 1a empty; CLI spot-checks pass against fresh binary: `hero agent events`, `hero spec new/set-owner/design`, `hero sync import`, `hero search --tag`, `hero list --status` all exit 0; `--kind`/`--themes`/`--owner` confirmed absent. |
| 3 | Events use `hero agent events <valid-type>` or `hero_event` MCP | DONE | All `hero event ...` → `hero agent events ...`; invalid `handoff` type → `spec_updated` (AGENTS.md, pm-delivery-lead, handoff-coordinator, handoff command, handoff-protocol, roadmap-curator, triage, prioritize, release-notes). |
| 4 | Owner flip described exclusively via `hero spec set-owner` | DONE | AGENTS.md handoff steps, handoff-coordinator §2, handoff command, handoff-protocol all flip via `hero spec set-owner <slug> engineering` (atomic history append); "raw frontmatter edit records no history" warning added. |
| 5 | pm-delivery-lead `permission.task` allowlist lists only shipped agents | DONE | 8 ghost `allow` entries deleted; allowlist is the 11 shipped subagents (`git diff` confirms). |
| 6 | Unshipped agent/skill scoped with P1 marker + named v1 fallback | DONE | Grep gates 1b/1c empty (excl. sanctioned Marty Cagan citation, roadmap-framing:45). Specialist tables, audience lists, prose across 20+ files scoped; ghosts dropped from `metadata.audience`. Also fixed `prd-anti-patterns` (omitted from the spec's Change list). |
| 7 | Lifecycle mapping in exactly one skill; citing agents cite it | DONE | `## PM lifecycle vocabulary → engine statuses` added to `pm-preset-detection`; pm-delivery-lead, handoff-coordinator, handoff command/protocol, story-writer, prd-author, pm-reviewer, roadmap-curator cite "per the lifecycle table in `pm-preset-detection`". |
| 8 | pm spec paths only under `.hero/planning/{...}` | DONE | `planning/specs/` → slug-resolved `.hero/planning/{features,bugs,epics,prds,intake}/` (handoff-coordinator, handoff command); `planning/roadmap/` → `planning/initiatives/` (discover command). Grep gate 1a empty. |
| 9 | `<harness>/` placeholders + `.hero/hero.json` in AGENTS.md structure | DONE | Project Structure rewritten to mirror `domains/engineering/AGENTS.md` (`<harness>/commands\|agents\|skills` + install sentence); `hero.json` → `.hero/hero.json`; dead relative decision link flattened. |
| 10 | Engineering-only command refs scoped or repointed | DONE | `/deliver`/`/design`/`/diagnose`/`/review`/`/scrub` scoped "(engineering pack)" or repointed to pm surfaces across agents, commands, skills, spec-types. |
| 11 | `kickoff-prompt` resolves in a `--domain pm` install | DONE | `git mv` engineering→`core/skills/kickoff-prompt` + engineering-pack scope note. Install smoke: skill lands at `~/pm-smoke2/.claude/skills/kickoff-prompt/SKILL.md` (was absent pre-move). |
| 12 | `go test ./...` passes including `content_parity_test.go` | DONE | `go test ./...` → 86 packages ok, 0 FAIL, exit 0. `TestDomainPacks_NoUnannotatedCoreShadows` (parity) PASS for engineering/sales/pm. |

### Changes

| # | Change | Status | Evidence |
|---|---|---|---|
| 1 | Lifecycle mapping + audience/body ghost fixes in `pm-preset-detection` | DONE | Section appended; audience trimmed to shipped 4; body ghosts → pm-delivery-lead "(P1 … v1.5)". |
| 2 | Promote `kickoff-prompt` to `core/skills/` + scope engineering cmd names | DONE | `git mv` + scope note; parity test green; install-verified. |
| 3 | Fix AGENTS.md routing + vocabulary tables | DONE | Both tables rewritten; `hero new <type>` → `hero spec new`; dup `/refine` merged. |
| 4 | Fix AGENTS.md CLI + compaction mechanics | DONE | events, handoff steps, `hero sync import`, `--kind`→`--tag`, `hero_active` MCP. |
| 5 | Fix AGENTS.md structure/config/dead-link + presets roster | DONE | `<harness>/` structure, `.hero/hero.json`, flattened link, dropped `core/vocabularies` path, roster reworded. |
| 6 | Trim pm-delivery-lead to shipped surfaces | DONE | allowlist ghost-free; 8 specialist rows repointed; status chain + events fixed. |
| 7 | Rewrite handoff-coordinator mechanics on real surfaces | DONE | slug-resolve paths, `set-owner` flip, `spec_updated` event, read-back verify, no `queue --owner`. |
| 8 | Same-surface fixes in pm `/handoff` command | DONE | in-review gate, real paths, `set-owner`, `spec_updated`, read-back verify. |
| 9 | Kill dead CLI in handoff-protocol | DONE | events + verify + hand-back on real surfaces; `set-owner` added; `/deliver` scoped. |
| 10 | Normalize type model (cross-domain-graph-query, dependency-mapping) | DONE | `spec` type → `type: feature`; `hero_why <slug>`; example tree in slug/status terms; ghosts dropped. |
| 11 | Sweep remaining agent files (8) | DONE | story-writer, prd-author, pm-reviewer, roadmap-curator, intake-triager, discovery-researcher, product-strategist, prioritization-strategist. |
| 12 | Sweep remaining command files (8) | DONE | README, metrics, pitch, refine, release-notes, discover, triage, prioritize. |
| 13 | Sweep remaining skill files (13 + `prd-anti-patterns`) | DONE | phantom skills/agents scoped or repointed; `--themes`→`--list --tag`; `/scrub intake`→`/triage`. |
| 14 | Scope pm spec-type docs (`intake.md`) | DONE | `/diagnose` → engineering-pack; `/scrub intake` → `/triage` sweep. |

### Exercise-the-feature check

The "feature" is a corrected content pack; exercised by:
- **Install smoke** — `hero install project ~/pm-smoke2 --target claude --domain pm` (fresh binary): 34 skills install, `kickoff-prompt` present, installed `AGENTS.md`/`CLAUDE.md` managed bodies grep-clean of every dead surface, lifecycle section present.
- **CLI spot-checks** — every surviving invocation run verbatim against the binary; none printed "unknown command"/"unknown flag".
- **Cold read** — walked `handoff-coordinator.md` steps 1–5; each names a runnable surface and the read-back verification can succeed.

### Excellence Bar self-check

- Went beyond the letter of the spec where the ACs required it: fixed `prd-anti-patterns` (an unscoped `pitch-author` + `prd-author-scrubber` the Change list omitted) and tightened two sub-engineer rows that were still phantom-skill-shaped (capacity row → real `sprint-planning`/`cycle-planning`; handoff-coordinator delegation prose → `in-review`).
- Left the one spec-sanctioned residual intact (Marty Cagan book citation, roadmap-framing:45) rather than over-scrubbing.
