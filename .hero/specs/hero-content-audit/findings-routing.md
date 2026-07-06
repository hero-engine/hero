# Findings — surface pass 3d (ROUTING FILES)

Audited: `domains/engineering/AGENTS.md`, `domains/pm/AGENTS.md`,
`domains/sales/AGENTS.md`, `core/commands/hero.md`, and the 6 per-domain
README.md files (freshness only). All CLI claims verified against the
`hero` binary (`hero version dev`, source at `bc86ad9`) **and**
cross-checked against `internal/cli/*.go` registrations; MCP tool claims
verified against `internal/serve/mcp_tools_def.go`.

---

## How the shipped instruction body is assembled (context for every finding)

`internal/install/agents_md.go` is the renderer:

1. `loadPackAgentsMdBody` resolves the body through a fixed chain:
   explicit override → **`AGENTS.md` at the root of the install source FS**
   (which is `OverlayFS(domains/<domain>/, core/)` — see
   `internal/cli/install.go` and the comment at `agents_md.go:80-84`) →
   Go fallback `generateEngineeringAgentsMdBody` (kept content-identical
   to `domains/engineering/AGENTS.md` by
   `TestEngineeringPackBodyMatchesGoFallback`).
2. `splitPackAgentsMd` strips the pack file's leading `# H1`; the managed
   orchestrator re-emits it as an **H2** section title inside the managed
   region (`<!-- hero:managed-start -->` … `end`).
3. The body gets a per-harness dialect block appended
   (`renderActiveDialectBlock`), plus a Codex-only workflow section for
   `--target codex`.
4. The same managed body is written into **both** `AGENTS.md` and
   `CLAUDE.md` at the user's project root (`claude_md.go`), for all six
   targets. So every claim in these files lands in **every session** of
   that domain, on every harness.
5. Domains with no `AGENTS.md` fall back to the engineering body with a
   stderr warning (`agents_md.go:105-108`).

Embedded packs (`content.go`): `engineering`, `pm`, `sales` each embed
`AGENTS.md`; **`domains/chat/` has no `AGENTS.md`** — but chat is also not
an embedded/installable domain (`hero domain list` → engineering, sales,
pm), so the fallback path is currently unreachable for chat. Noted as S3.

Two assembly-level consequences that generate findings below:

- **Heading depth matters.** Because the pack H1 becomes an H2,
  engineering's `###` sections nest correctly under it; PM's and sales'
  `##` sections render as *siblings* of the pack title, escaping the
  section (see S3 structural finding).
- **Relative links die at install.** The body is transplanted verbatim to
  the user's project-root AGENTS.md, so relative links like
  `commands/qualify.md` or `../../.hero/knowledge/...` resolve against
  the user's repo root, where those paths don't exist.

Adjacent shipping observation: `installFlat` (`internal/install/content.go`)
filters only on the `.md` suffix with **no README exclusion**, and the
embeds include whole directories — so `domains/pm/agents/README.md`,
`domains/pm/commands/README.md`, `domains/sales/agents/README.md`, and
`domains/sales/commands/README.md` install into user harness dirs (e.g.
`.claude/agents/README.md`) as frontmatter-less pseudo-agents/commands.
(`internal/spectypes/loader.go:97` *does* exclude README — the content
installer doesn't.)

---

## Per-file claims tables

### domains/engineering/AGENTS.md (1,954 words)

| Claim | Status |
|---|---|
| Slash: /diagnose /design /deliver /drive /review /compose /convention /decide /discover /mock /docs /release /retro /note /scan /check /sprint /import (+/handoff in prose) | **all exist** in `domains/engineering/commands/` |
| CLI: `hero status`, `search`, `snapshot`, `sync import`, `sync pull`, `note`, `check`, `peer list/show/call` (`--mode=advisory|spec-out`, `--related-spec`, `--reason`), `handoff` (+`status`/`accept`), `admin repos add`, `spec verify`, `spec mock detect`, `search --list --type` | **all exist**, flags verified |
| MCP: `hero_search` (`compact: true`), `hero_read_spec`, `hero_list`, `hero_queue`, `hero_blocked`, `hero_why` | **all exist** (`mcp_tools_def.go`) |
| Agents named: feature-delivery-lead, debug-investigator (as examples) | exist |
| **Claimed & MISSING** | none — engineering is clean on existence |
| **Shipped & unlisted** | commands: `/blocked` `/capture` `/challenge` `/peer` `/prime` `/resume` `/roadmap-review` `/scrub` `/split` `/why` `/hero` (11 of 30). Agents: **33 of 35** unnamed (no roster). Skills: **all 52** unnamed. |

### domains/pm/AGENTS.md (2,247 words)

| Claim | Status |
|---|---|
| Slash (pm pack): /triage /refine /prioritize /handoff /prd /pitch /roadmap /discover /metrics /release-notes | exist |
| Slash (core overlay): /why /blocked /note /decide /retro | exist |
| Slash: **/interview, /capacity, /plan-cycle, /plan-sprint, /plan-iteration, /standup, /scrub roadmap, /scrub intake, /scrub specs, /diagnose, /search, /review** | **MISSING** — not in `domains/pm/commands/` nor `core/commands/` (a PM install = pm + core only) |
| Agents: intake-triager, product-strategist, discovery-researcher, story-writer, pm-reviewer, handoff-coordinator, roadmap-curator, pm-investigator, prd-author (+ engineering's `engineer`, cross-domain) | exist |
| Agents: **pitch-author, cycle-planner** | **MISSING** from `domains/pm/agents/` (README scopes them P1; AGENTS.md does not) |
| CLI: `hero status/search/sync pull/note/check/why/blocked/peer list/peer call/feed --since/next path/recap --since` | exist, flags verified |
| CLI: **`hero event <type> "..." --slug`** | **WRONG** — registered as `hero agent events` (`internal/cli/agent.go:28`, `event.go:21`); `hero event`/`hero events` at root: unknown command |
| CLI: **`hero active register` / `hero active list`** | **MISSING** — no such CLI command (grep source: no `activeCmd`); only MCP `hero_active` exists |
| CLI: **`hero queue --owner engineering`** | **MISSING flag** — `hero queue --help` has only `--format --horizon --limit --subproject`; `hero list` has no `--owner` either |
| CLI: **`hero new feature` / `hero new bug` / `hero new epic` / `hero new initiative`** | **WRONG syntax & type** — actual: `hero new <slug> --type <t>`; supported types: feature, bug, initiative, convention, decision, rule, external, context, note — **no `epic`** (`internal/cli/new.go:25-42`) |
| CLI: **`hero import` "import issues from tracker via tracker_mappings"** | **WRONG** — root `hero import` ingests a URL/file/dir into the knowledge base; tracker import is `hero sync import` |
| CLI: **`hero search … --kind=…`** | **MISSING flag** — no `--kind` on `hero search` |
| MCP: `hero_plan`, `hero_ask` | exist |
| Skills named: pm-preset-detection, next-md | exist |
| **Shipped & unlisted** | agents: duplicate-detector, pm-delivery-lead, prioritization-strategist (3 of 12). Skills: **17 of 19** unnamed. Commands: all 10 pm commands listed ✓. |

### domains/sales/AGENTS.md (1,233 words)

| Claim | Status |
|---|---|
| Slash: /qualify /strategize /forecast /pipeline /research /debrief /prospect; /note /check (core) | **all exist** |
| Agents Reference: all 5 (deal-strategist, qualification-analyst, forecast-analyst, competitive-intel, buyer-researcher) | **all exist** |
| Skills Reference: all 7 | **all exist** |
| `spec-types/deal.yaml` file | exists — but **never loaded**: `internal/spectypes/loader.go:94` reads only `*.md`, so `type: deal` is not a registered spec type |
| CLI: `hero status`, `hero queue`, `hero search`, `hero next` | exist (but see semantics findings) |
| CLI: **`hero read-spec <slug>`** | **MISSING** — unknown command; the real surface is MCP `hero_read_spec` |
| CLI: **`hero pulse --week`** | **WRONG** — registered as `hero sprint pulse` (`internal/cli/sprint.go:64`); root `hero pulse`: unknown command |
| CLI: **`hero forecast`** | **MISSING** — no such command anywhere in `internal/cli/` |
| CLI: **`hero search --match "stale"`** | **MISSING flag** — no `--match` on `hero search` |
| CLI: `hero search --type playbook|battlecard|knowledge` | flag exists but the **type values don't** — no such spec types are registered (core + sales registry); queries return nothing |
| Config: `.hero/hero.json` keys `crm`, `qualification`, `forecast`, `pipeline.{stale_days,hygiene_schedule}` | **do not exist** in `internal/config/config.go` (only `TeamConfig.StaleDays` elsewhere) |
| MCP: `hero_anchor` | exists |
| `.hero/mission.md` | exists (in a live workspace) |
| **Shipped & unlisted** | none — sales rosters are complete |

### core/commands/hero.md (487 words)

| Claim | Status |
|---|---|
| Slash: /design /diagnose /deliver /review /compose /convention /decide /discover /docs /release /retro /note /scan /check | all exist **in engineering installs**; `/design /diagnose /deliver /review /compose /release` are **MISSING in pm/sales installs**, which receive this very file via the core overlay |
| CLI: `hero status`, `hero dashboard`, `hero search`, `hero spec new`, `hero scan`, `hero check`, `hero note`, `hero do` | **all exist** |
| **Unlisted** | core commands `/drive /import /handoff /prime /resume /why /blocked /capture`; engineering routing additions `/mock /sprint /peer /scrub /split` — the meta/help command has drifted behind both rosters |

### Per-domain READMEs (freshness only)

| File | Result |
|---|---|
| domains/pm/agents/README.md | OK — all 12 agent files exist; linked design doc `.hero/planning/features/hero-pm/agent-pack-design.md` exists |
| domains/pm/commands/README.md | **stale**: "Reused" section claims `/search` (exists nowhere — no pack ships a search command) and `/deliver` (engineering-only; not in a PM install); refine row routes to `epic-framer` which doesn't exist and, unlike `pitch-author`/`metrics-analyst`, carries no "(P1)" scoping |
| domains/pm/skills/README.md | all 19 skill dirs exist; cosmetic: "Writing (5)" lists 6 |
| domains/sales/agents/README.md | OK — all links resolve |
| domains/sales/commands/README.md | OK — all links resolve |
| domains/sales/skills/README.md | OK — all links resolve |

---

## Findings

### [S1] PM routing table routes to 10 slash commands that don't exist in a PM install — domains/pm/AGENTS.md
- Surface/dimension: routing / 4 (freshness) + claims accuracy
- Evidence: routing rows for `/interview`, `/capacity`, `/plan-cycle`, `/plan-sprint`, `/plan-iteration`, `/standup`, `/scrub roadmap`, `/scrub intake`, `/scrub specs`, `/diagnose`, `/search`, `/review` (lines 36-52). A PM install is `OverlayFS(domains/pm/, core/)`; none of these exist in either (`domains/pm/commands/` has 10 commands; `core/commands/` has 17, none of these). `domains/pm/commands/README.md` itself says these "ship in v1.5+". A PM session told to "Run the command — don't just suggest it" will attempt a command that isn't installed, every time one of these intents fires.
- Fix shape: delete the dead rows or route them to real surfaces (e.g. "/search" → `hero search` CLI).

### [S1] PM CLI section teaches six wrong or nonexistent invocations — domains/pm/AGENTS.md
- Surface/dimension: routing / 4 (freshness) + slash-vs-CLI parity
- Evidence (each verified against binary + source):
  - `hero event decision_made "..." --slug ...` (lines 110-111) — actual command is `hero agent events <type> <msg>` (`internal/cli/agent.go:28` registers `eventCmd` under `agentCmd`; `event.go:21` `Use: "events"`). `hero event` → unknown command.
  - `hero active register <session-id> <slug>` / `hero active list` (lines 293-294, 298) — no `hero active` CLI exists; only MCP `hero_active`.
  - `hero queue --owner engineering` (line 156) — `hero queue` has no `--owner` flag; neither does `hero list`.
  - `hero new feature` / `hero new bug` / `hero new epic` / `hero new initiative` (lines 27-30, 69-72) — actual: `hero new <slug> --type <t>`; supported types exclude `epic` (`internal/cli/new.go:36,42`).
  - `hero import` described as tracker import via `tracker_mappings` (lines 178-180) — root `hero import` ingests URLs/files into the knowledge base; the tracker path is `hero sync import`.
  - `hero search --list --type feature … --kind=…` (lines 241-243) — no `--kind` flag on `hero search`.
- Fix shape: correct each invocation (`hero agent events`, MCP `hero_active`, drop `--owner`, `hero new <slug> --type`, `hero sync import`, drop `--kind`) or delete the claim.

### [S1] Sales CLI blocks name four nonexistent/wrong commands and fabricated search types — domains/sales/AGENTS.md
- Surface/dimension: routing / 4 (freshness) + slash-vs-CLI parity
- Evidence: `hero read-spec <slug>` (line 28, session-start step 2) — unknown command (real surface: MCP `hero_read_spec`). `hero pulse --week` (line 156) — actual: `hero sprint pulse --week` (`internal/cli/sprint.go:64`). `hero forecast` (line 157) — no such command in `internal/cli/`. `hero search --match "stale"` (line 172) — no `--match` flag. `hero search --type playbook|battlecard|knowledge` (lines 43-44, 166-167, 192-194) — `--type` exists but no `playbook`/`battlecard`/`knowledge` spec type is registered anywhere, so every such query silently returns nothing. Session-start is the first thing every sales session executes, so step 2 fails immediately.
- Fix shape: replace with real invocations (`hero_read_spec` MCP or `hero search`, `hero sprint pulse`, drop `hero forecast`/`--match`) and either register the knowledge types or drop the `--type playbook/battlecard` idiom.

### [S1] Sales "Domain Configuration" documents a hero.json schema that doesn't exist — domains/sales/AGENTS.md
- Surface/dimension: routing / 4 (freshness) + 3 (actionability)
- Evidence: lines 217-251 present a full JSON block (`"crm"`, `"qualification"`, `"forecast"`, `"pipeline": {"stale_days", "hygiene_schedule"}`) as what "Hero Sales reads … from `.hero/hero.json`". None of these keys exist in `internal/config/config.go` (no `crm`/CRM anywhere; `stale_days` exists only inside `TeamConfig`). An agent asked to configure Salesforce sync will write dead config and report success.
- Fix shape: delete the block or mark it as a design-target schema not yet read by the engine.

### [S1] Sales deal spec-type is defined in a format the engine never loads — domains/sales/AGENTS.md + domains/sales/spec-types/deal.yaml
- Surface/dimension: routing / 4 (freshness)
- Evidence: AGENTS.md lines 128-129: "All deals are tracked as specs following the schema in `spec-types/deal.yaml`". `internal/spectypes/loader.go:94` skips any file not ending `.md` — the sales pack's only spec-type file is `.yaml`, so the registry contains no `deal` type for sales installs (pm's `intake.md`/`prd.md` load fine, showing `.md` is the contract). Descriptions of `hero status` as "pipeline by stage with ARR totals" and `hero queue` as "deals needing attention" (lines 155, 171) are likewise unbacked — no deal/ARR/pipeline rendering exists in `internal/cli/status.go` or `dashboard.go`.
- Fix shape: convert `deal.yaml` to the `.md` spec-type format; rewrite the status/queue descriptions to what the generic commands actually print.

### [S1] PM "Project Structure" points installed users at hero-engine source paths — domains/pm/AGENTS.md
- Surface/dimension: routing / 6 (harness-agnosticism) + 4
- Evidence: lines 192-201 list `domains/pm/agents/`, `domains/pm/skills/`, `domains/pm/commands/`, `domains/pm/spec-types/`, `core/spec-types/`, `core/vocabularies/` as the project structure. After install these paths don't exist in the user's repo — content lands under `<harness>/agents|commands|skills/`. This is exactly the failure mode the renderer warns about (`agents_md.go:283-288`: paths "MUST match the actual on-disk layout — pointing the model at directories that don't exist sends it hunting through the workspace"). Engineering's equivalent section correctly uses `<harness>/…` placeholders. Same defect at line 239 ("in `core/vocabularies/*.yaml`") and line 237's relative link `../../.hero/knowledge/decisions/tracker-fronting-and-local-first.md` (resolves in-source, breaks from a user's project-root AGENTS.md).
- Fix shape: mirror engineering's `<harness>/` placeholder convention; replace the relative knowledge link with a plain `.hero/knowledge/decisions/…` path.

### [S1] PM AGENTS.md names two agents that don't ship — domains/pm/AGENTS.md
- Surface/dimension: routing / 4 (freshness)
- Evidence: lines 91-93: "Authoring agents (`prd-author`, `story-writer`, `pitch-author`) and the `cycle-planner` agent must load `pm-preset-detection`…". `domains/pm/agents/` contains neither `pitch-author.md` nor `cycle-planner.md`. The commands README scopes pitch-author as "(P1) → falls back to prd-author"; AGENTS.md presents both as present-tense obligations on agents that can't be invoked.
- Fix shape: reword to the shipping roster (`prd-author`, `story-writer`) or add the P1 scoping.

### [S2] core/commands/hero.md (the meta/help command) has drifted from both rosters — core/commands/hero.md
- Surface/dimension: command / 4 (freshness) + 7 (consistency)
- Evidence: (a) routes to `/design /diagnose /deliver /review /compose /release`, none of which exist in pm/sales installs that receive this file via the core overlay — the "which command should I run" helper answers with commands the workspace doesn't have; (b) omits `/drive /sprint /import /mock /handoff /prime /resume /why /blocked /capture /peer /scrub /split`, i.e. the engineering AGENTS.md routing table and this file disagree on ~13 workflows; (c) CLI list is fine (all 8 verified, including `hero do` and `hero dashboard`).
- Fix shape: regenerate the workflow list per-domain (or from the overlay at install time), and sync the routing bullet list with the AGENTS.md table.

### [S2] Sales relative links break at install — domains/sales/AGENTS.md
- Surface/dimension: routing / 4 (freshness)
- Evidence: Commands/Agents/Skills reference tables link `commands/qualify.md`, `agents/deal-strategist.md`, `skills/deal-qualification/SKILL.md`, `spec-types/deal.yaml` (lines 90-129). The managed body is transplanted verbatim into the user's project-root `AGENTS.md`/`CLAUDE.md`; installed content lives under `<harness>/…`, so every one of these ~20 links 404s for the file's actual audience. (They only resolve when reading the file inside hero-engine.)
- Fix shape: drop the links (keep names) or use `<harness>/…` placeholders like engineering does.

### [S2] Engineering AGENTS.md leaves a third of its command roster and all agents/skills unroutable — domains/engineering/AGENTS.md
- Surface/dimension: routing / 1 (earns place) + claims accuracy (invisible content)
- Evidence: 11 of 30 shipped commands appear nowhere in the file: `/blocked /capture /challenge /peer /prime /resume /roadmap-review /scrub /split /why /hero` (`/handoff` appears only inside the peering-disambiguation prose). 33 of 35 agents and all 52 skills are unnamed — sessions can't route to what isn't listed, and sales proves the pack can afford roster tables at 1,233 words. Notably `/peer` exists as a slash command while the routing table sends peer intents to raw CLI invocations only.
- Fix shape: add routing rows (or a compact roster line) for the 11 missing commands. Note: changes must land in **both** `domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody` (identity is test-enforced, `agents_md.go:374-378`).

### [S2] Claude-Code-specific machinery presented as the mechanism in a six-target file — domains/engineering/AGENTS.md
- Surface/dimension: routing / 6 (harness-agnosticism)
- Evidence: "Internal Lookups — Tool Routing" (lines 110-127) instructs use of `mcp__hero__hero_search` (double-underscore MCP naming is the Claude Code convention), the `Explore` agent ("context-protective"), and "Deferred-tool friction (the one-time `ToolSearch` schema load)" — `ToolSearch` and `Explore` exist only in Claude Code, yet the section is unscoped and ships identically to opencode/cursor/copilot/codex/generic. Contrast line 82, which scopes harness examples correctly ("e.g. `.claude/commands/` … for Claude").
- Fix shape: scope the section or express it in harness-neutral capability terms.

### [S2] PM AGENTS.md is the heaviest routing file and re-teaches three skills inline — domains/pm/AGENTS.md
- Surface/dimension: routing / 2 (token efficiency)
- Evidence (2,247 words, every PM session): "The handoff is an owner flip" (lines 142-169, ~230 words) restates `handoff-protocol` skill + `handoff.md` command content; "Methodology presets" (lines 79-102, ~180 words) restates `pm-preset-detection` (which line 101 already defers to); "Vocabulary-aware routing" second table (lines 67-77) re-derives four rows already implied by the main table; "Capture execution plans" (lines 301-316) duplicates core plan-capture guidance. Cutting the duplicated bodies and keeping pointers saves ~500 words without losing a routing decision.
- Fix shape: compress the four named sections to 2-3-line pointers at their skill/command owners.

### [S2] Engineering AGENTS.md carries three sections that belong to their command/skill owners — domains/engineering/AGENTS.md
- Surface/dimension: routing / 2 (token efficiency)
- Evidence (1,954 words, every engineering session): "Declaring Spec Relationships" (lines 84-108, ~150 words + 2 YAML blocks) duplicates `spec-format` skill (2,482 words, whose job this is); the mockup-routing paragraph (line 42, ~185 words of anti-confabulation prose) duplicates `mock.md` (780 words); the peering-disambiguation paragraph (line 44, ~160 words) duplicates `cross-repo-peering` skill (1,454 words). ~450 words of always-loaded context re-teaching content that loads on demand. (Same dual-edit constraint: Go fallback must be updated in lockstep.)
- Fix shape: reduce each to a one-line rule + pointer ("relationships are frontmatter-only — see spec-format skill").

### [S2] agents/ and commands/ READMEs install as pseudo-content — domains/pm/*, domains/sales/*
- Surface/dimension: routing-adjacent / 4 (freshness, shipping side effect)
- Evidence: `content.go` embeds whole directories (`domains/pm/agents` includes `README.md`); `installFlat` (`internal/install/content.go:44`) skips only non-`.md` entries and dirs — no README exclusion exists in `internal/install/`, and the Claude target installs agents/commands via `installFlat` (`target_claude.go:30-33`). PM/sales installs therefore materialize `README.md` as a frontmatter-less agent and command in the user's harness. (`internal/spectypes/loader.go:97` shows the codebase already knows to exclude READMEs elsewhere.)
- Fix shape: add a README exclusion to `installFlat` (code change — flagged for the fix pass).

### [S2] Sales "Key CLI Commands" block mixes slash commands into a bash block — domains/sales/AGENTS.md
- Surface/dimension: routing / slash-vs-CLI parity (task 2)
- Evidence: lines 153-173, a ```bash block titled "Key CLI Commands (Sales)" interleaves `/qualify acme-corp-enterprise`, `/strategize …`, `/research …`, `/debrief … --won` with real `hero` invocations. Both sibling packs open their CLI sections with "These are run in the terminal, not as slash commands" precisely to prevent this confusion; here the file itself commits it.
- Fix shape: split the block into CLI vs slash lists; adopt the shared disclaimer sentence.

### [S2] PM commands README routes "reused" work to commands that don't exist — domains/pm/commands/README.md
- Surface/dimension: command README / 4 (freshness)
- Evidence: "Reused (cross-domain / core)" lists `/search` (no search command exists in any pack — core, pm, engineering all lack one; the CLI `hero search` is the real surface) and `/deliver` ("picked up by engineering's engineer agent" — true cross-repo, but not available in the PM install this README documents). The `refine.md` row names `epic-framer` with no P1 scoping; no such agent exists.
- Fix shape: change `/search` → `hero search`, scope `/deliver` as engineering-side, mark `epic-framer` (P1) like its siblings.

### [S3] The three domain AGENTS.md files diverge structurally — and two break the installed heading hierarchy — domains/{engineering,pm,sales}/AGENTS.md
- Surface/dimension: routing / 7 (consistency)
- Evidence: engineering uses `###` sections; pm and sales use `##`. At install the pack H1 is demoted to an H2 section title inside the managed region (`splitPackAgentsMd` + orchestrator), so engineering's `###` sections nest under it while pm/sales `##` sections render as siblings of "Hero PM — …"/"Hero Sales — …", visually escaping Hero's managed section. Beyond headings: sales has full roster tables + a numbered session-start protocol + `---` dividers; pm has routing + heavy prose, partial rosters; engineering has routing + no rosters. The shared obligations (session title, "run don't suggest", CLI disclaimer, compaction survival) appear in three different shapes and section orders — divergence reads accidental, not domain-motivated.
- Fix shape: pick one skeleton and one heading depth (`###`); document it where pack authors will see it.

### [S3] PM AGENTS.md misplaces hero.json and duplicates a routing row — domains/pm/AGENTS.md
- Surface/dimension: routing / 4 + 7
- Evidence: line 210 lists "`hero.json` — Project configuration" at what reads as repo root; actual location is `.hero/hero.json` (`config.Load`: `filepath.Join(projectRoot, cfg.Folder, "hero.json")`; engineering and sales both say `.hero/hero.json`). PM agents/commands READMEs repeat the bare "hero.json". The routing table also has two rows targeting `/refine` (lines 25 and 45).
- Fix shape: `.hero/hero.json`; merge the duplicate row.

### [S3] Sales "hero next" described as writing the handoff — domains/sales/AGENTS.md
- Surface/dimension: routing / 3 (actionability)
- Evidence: lines 211-213: "run `hero next` to write a crisp handoff" — `hero next` *shows* the briefing; the agent writes the file at the path from `hero next path` (as PM's equivalent section correctly instructs).
- Fix shape: "write your briefing to the path from `hero next path`".

### [S3] domains/chat has no AGENTS.md — currently harmless, will misroute if chat becomes installable — domains/chat/
- Surface/dimension: routing / 4
- Evidence: `domains/chat/` contains only `commands/` (6 commands: ask-corpus, capture, discover, note, space, why). No AGENTS.md, no embed in `content.go`, not listed by `hero domain list` — so today the engineering-fallback path (`agents_md.go:105-108`) can't fire for it. If/when chat is embedded, installs would warn and ship the engineering routing table, routing chat sessions to `/diagnose//design//deliver//mock`, none of which chat ships.
- Fix shape: author a chat AGENTS.md before (or in the same change as) embedding the domain.

### [S3] PM skills README section count off by one — domains/pm/skills/README.md
- Surface/dimension: skill README / 7
- Evidence: "### Writing (5)" lists six skills (story-writing-invest, acceptance-criteria-ears, prd-structure, prd-anti-patterns, pitch-writing-shape-up, roadmap-framing).
- Fix shape: "(6)".

---

## Severity totals

| Severity | Count |
|---|---|
| S1 | 7 |
| S2 | 9 |
| S3 | 5 |
| **Total** | **21** |

Verified-clean highlights (no finding): engineering AGENTS.md's entire
slash + CLI + MCP claim set exists as written; sales rosters are 100%
complete (nothing shipped goes unlisted); core/hero.md's CLI list is fully
accurate; sales READMEs and pm agents README are link-clean;
`hero peer call --related-spec/--reason`, `hero spec mock detect`,
`hero admin repos add`, `hero next path`, `hero feed --since`,
`hero recap --since`, and MCP tools `hero_plan/hero_ask/hero_anchor/
hero_active/hero_pulse/hero_read_spec` all check out against source.
