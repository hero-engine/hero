---
title: "Sales Pack Reality Sync — make every sales-pack claim match the engine"
slug: sales-pack-reality-sync
type: bug
status: planning
priority: P1
size: medium
domain: engineering
created: 2026-07-06
tags: [content-audit, sales-pack, docs-accuracy, spec-types]
relations:
  - { target: content-remediation, kind: parent }
  - { target: hero-content-audit, kind: related }
---

# Sales Pack Reality Sync — make every sales-pack claim match the engine

## Context

The hero-content-audit (`.hero/specs/hero-content-audit/findings-{routing,agents,skills,commands}.md`) found that the sales pack — `domains/sales/` — teaches a workflow the engine doesn't implement. The very first thing every sales session executes (Session Start step 2 in `domains/sales/AGENTS.md`) invokes `hero read-spec`, which doesn't exist. From there the fiction compounds:

- **Phantom CLI**: `hero read-spec`, `hero pulse --week`, `hero forecast`, `hero search --match`, `hero note --type`, `hero research` (findings-routing S1, findings-agents S1, findings-commands F5).
- **Phantom config**: a full "Domain Configuration" `hero.json` block (`crm`, `qualification`, `forecast`, `pipeline.{stale_days,hygiene_schedule}`) with no counterpart in `internal/config/config.go` — re-verified today: the only `stale_days` in config is `TeamConfig.StaleDays` (config.go:1014). `forecast-analyst` and `qualification-analyst` read these keys at runtime and find nothing.
- **Unloaded spec type**: `spec-types/deal.yaml` is never loaded — `internal/spectypes/loader.go:94` reads only `*.md` — so `type: deal` is unregistered and every `hero search --type deal/playbook/battlecard/prospect/knowledge/retro` idiom in the pack silently returns nothing (F21). Save locations (`.hero/knowledge/battlecards/`) and lookup instructions (`--type battlecard`) disagree.
- **Discovery hazard**: battlecards authored per the `competitive-intel` template carry `type: battlecard`, which is *not* in `nonWorkFlatTypes` (`internal/spec/spec.go:1166-1179`) — since `Discover` walks the whole hero dir, these knowledge files get slurped into work-spec discovery as flat specs.
- **Numeric drift**: staleness/risk thresholds duplicated across `pipeline-management`, `forecast-methodology`, `deal-qualification` with conflicting values (Negotiation staleness 7 vs 10+ days; MEDDPICC qualify-out "<25 after 2+" vs "after 3" conversations).
- **Broken-at-install links**: ~20 relative links in AGENTS.md reference tables that 404 once the body is transplanted to a user's project root; a "Key CLI Commands" bash block that interleaves slash commands; "run `hero next` to write a crisp handoff" (it *shows* the briefing; the agent writes to the path from `hero next path`).
- **Roster lies**: skill descriptions/`metadata.audience` "Loaded by" claims contradict the agents' actual Required-skills sections.

**Current-state facts verified 2026-07-06** (post-`177e8a1` content-dedup-resync — that commit touched only core↔engineering duplicates; every sales-pack defect above is still live, re-read today):

- The weekly narrative command is **`hero sprint status --week`** — `pulseCmd` registers under `sprint` with `Use: "status"` (`internal/cli/sprint.go:59-64`, `--week` confirmed in live `--help`). There is no `hero sprint pulse` and no root `hero pulse`.
- `hero note` has only `--from`. `hero search` has `--type/--status/--tag/--list/...` but no `--match`; `--status` is a single value (no comma list). `hero list` has `--type strings`, `--status strings` (comma lists OK), and `--stale int`.
- `hero_read_spec`, `hero_anchor`, `hero_pulse` MCP tools exist.
- `parseRecord` (`internal/spectypes/loader.go:223`) requires top-level `type:` and `category: work|knowledge`; `Load("sales")` overlays `domains/sales/spec-types/*.md` on core and hard-errors on core-type collisions. `core/spec-types/feature.md` is the working reference shape. Caution: `domains/pm/spec-types/intake.md`/`prd.md` lack top-level `type:`/`category:` and do **not** satisfy `parseRecord` — follow the core files, not the pm ones.

## Goal

Every claim in `domains/sales/` matches the engine as it exists today: every cited CLI invocation runs without unknown-command/unknown-flag errors, `type: deal` is a registered spec type on sales installs, no documented `hero.json` key is unread by `internal/config/config.go`, battlecard/playbook save and lookup locations agree, staleness/risk thresholds have exactly one numeric owner (`pipeline-management`), and the installed AGENTS.md body contains no links that 404 in a user workspace. No engine (Go) changes; content only.

## Kickoff

Fixes every sales-pack claim that doesn't match the engine — phantom CLI commands, a `hero.json` schema nothing reads, a `deal.yaml` spec type the loader never loads, and battlecard/playbook lookups that silently return nothing.

**Status:** planning — spec authored from hero-content-audit findings; every path and CLI claim re-verified post-`177e8a1`; no edits yet.

**Pick up at:** Change 1 — convert `domains/sales/spec-types/deal.yaml` → `deal.md` in the core spec-type shape, then sweep AGENTS.md (Changes 3–9).

→ `.hero/planning/initiatives/content-remediation/sales-pack-reality-sync/spec.md`

**Files:** `domains/sales/AGENTS.md`, `domains/sales/spec-types/deal.yaml`, `core/spec-types/feature.md`, `internal/spectypes/loader.go:223`
**Skip:** don't copy `domains/pm/spec-types/*.md` frontmatter shape — it fails `parseRecord`; don't add Go config keys — content-only.

## Approach

**Replace fiction with verified invocations.** Every replacement below was run against the installed binary today. Where a capability genuinely doesn't exist as CLI (`read-spec`, weekly pulse via MCP), point at the real surface using the pack's existing convention of bare MCP tool names (`hero_anchor` precedent already in this file).

**Deal spec type: convert, don't drop.** The pack's core promise ("all deals are tracked as specs") is one file-format away from true. Convert `deal.yaml` to `deal.md` following the shape `parseRecord` actually consumes (see `core/spec-types/feature.md`): top-level `type:` + `category:` required, plus lifecycle with referential integrity (every transition state declared). `deal` doesn't collide with any core type, so the domain-overlay load is safe. This makes `hero search --type deal` / `hero list --type deal` real on sales installs.

**Playbooks/battlecards/prospects: plain-text lookup, no new type registrations.** Three options existed: (a) register `playbook`/`battlecard`/`prospect` as spec types, (b) plain-text queries against knowledge files, (c) leave broken. Chose (b): these are knowledge entries, not work items with lifecycles — registering them as work-category types would pull them into `hero list`/queue semantics they don't want, and a knowledge-category registration still wouldn't make the FTS `--type` filter index `.hero/knowledge/` files. Plain-text search works today with zero engine change, provided the discriminating word appears in the document — so battlecard/playbook templates get retitled to include it ("Battlecard — Hero vs. X", "Playbook: ..."). "Prospect" stops being a fake type: a prospect is a `type: deal` spec at `status: prospect` (the deal lifecycle's initial state), which matches where `buyer-researcher` already writes the file.

**Discovery hazard closed in the template.** `type: battlecard` on a flat knowledge `.md` makes it a discoverable work spec (`isDiscoverableFlatSpec` requires only an explicit type not in `nonWorkFlatTypes`). Fix at the source: the battlecard template drops the `type:` line entirely (no explicit type ⇒ not discoverable). Same rule for buyer-researcher's knowledge writes.

**Config block: delete, not design-target.** Justification: (1) the keys are read by nothing — an agent told to "configure Salesforce sync" writes dead JSON and reports success, the worst failure mode the audit identified; (2) marking it "future schema" still *invites* writing dead config and puts a maintenance IOU in installed content where no engine change will ever automatically update it; (3) the operative defaults (framework = MEDDPICC, weighted methodology, stage weights) already have real homes — the `deal` spec type's per-deal `qualification_framework`/`probability` fields and the `forecast-methodology`/`deal-qualification` skills. If CRM sync ships later, its docs land with the code. The section is replaced by three honest lines stating exactly that.

**Thresholds: single owner.** `pipeline-management` (the skill that owns stage definitions and hygiene) becomes the sole home of numeric staleness/risk thresholds. Conflicts resolve to its values: Negotiation stale at 7 days (its stale table), qualify-out review at MEDDPICC < 25 after **3** conversations. Other skills keep the *concept* with a one-line cross-reference, no numbers.

**Scope discipline.** The "What this skill covers" boilerplate ×7 is noted but *not* trimmed here — only the deal-qualification bullet that is factually wrong (phantom BANT coverage) is touched. Trimming is `token-efficiency-pass`'s job.

## Changes

1. **Convert `domains/sales/spec-types/deal.yaml` → `domains/sales/spec-types/deal.md`** (create the `.md`, delete the `.yaml`). Follow `core/spec-types/feature.md`: frontmatter with `title: Deal`, `type: deal`, `domain: sales`, `category: work`, `bucket: deals`, `location: .hero/planning/deals/{slug}/spec.md`; `lifecycle:` states `[prospect, qualifying, demo, proposal, negotiation, won, lost]`, initial `prospect`, terminal `[won, lost]`, transitions with gates lifted from `pipeline-management`'s exit criteria (e.g. `qualifying → demo`: "MEDDPICC ≥ 40; EB identified by name/title"); `frontmatter:` required `[title, type, status, company]`, optional `[owner, arr, close_date, stage, meddpicc_score, probability, qualification_framework, crm_id, crm_type, priority, tags, relations]` (carry descriptions over from deal.yaml); `sections:` optional `[Qualification, Deal Strategy, Research, Competitive Situation, Debrief]`; `accepting_commands: [/qualify, /strategize, /forecast, /pipeline, /research, /debrief]`; `default_agents:` authoring `deal-strategist`, qualification `qualification-analyst`, forecast `forecast-analyst`. Drop deal.yaml's 6-value `type` enum (`prospect/playbook/battlecard/campaign/retro` are not registered types — see Approach); align the probability stage-default names to the canonical statuses (`prospect: 10, qualifying: 20, demo: 40, proposal: 60, negotiation: 80, won: 100, lost: 0` — the yaml's `Prospecting/Discovery/Evaluation` labels match no status). Body: When-to-use / Lifecycle / Sections prose per the core files.
2. **`domains/sales/spec-types/README.md`** — repoint both `deal.yaml` links and the closing "See deal.yaml…" line to `deal.md`; note statuses are registry-backed lifecycle states.
3. **`domains/sales/AGENTS.md` Session Start** — step 2: replace `hero read-spec <slug>` with the `hero_read_spec` MCP tool (bare-name convention, matching this file's `hero_anchor` usage), with "or open the spec path returned by `hero search`" as the fallback. Step 5: replace `hero search --type playbook|battlecard "..."` with plain-text queries (`hero search "playbook <segment or motion>"`, `hero search "battlecard <competitor>"`).
4. **AGENTS.md "Key CLI Commands (Sales)"** (lines ~151-173) — split the single bash block into a real-CLI block and a slash-command list, opening with the sibling packs' disclaimer ("These are run in the terminal, not as slash commands"). Replace: `hero pulse --week` → `hero sprint status --week`; `hero forecast` → route to `/forecast`; `hero search --match "stale"` → `hero list --type deal --stale 14`; the two `--type playbook/battlecard` lines → plain-text queries. Rewrite the `hero status` ("pipeline by stage with ARR totals") and `hero queue` ("deals needing attention") annotations to what the generic commands actually print (workspace/spec state; ranked ready-to-work specs).
5. **AGENTS.md "Auto-Capture" review block** (lines ~190-195) — replace the three `--type knowledge/playbook/battlecard` queries with plain-text searches plus `hero list --type deal` for deal inventory.
6. **AGENTS.md "Surviving Context Compaction" step 4** — replace "run `hero next` to write a crisp handoff" with: write your briefing to the path from `hero next path` (`hero next` only *shows* the briefing).
7. **AGENTS.md "Domain Configuration"** — delete the entire JSON block (lines ~217-251); replace with 3-4 lines: no sales-specific `hero.json` keys are read by the engine today; qualification framework is per-deal frontmatter (`qualification_framework`, default `meddpicc`); forecast methodology and stage weights live in the `forecast-methodology` skill and the `deal` spec type's stage defaults.
8. **AGENTS.md "Deal Spec Structure"** — point at `spec-types/deal.md`; keep the example frontmatter (it is already consistent with the converted type).
9. **AGENTS.md reference tables** — strip all ~20 relative links in the Commands/Agents/Skills tables and the spec-type link (keep names as code spans); the body is transplanted verbatim to the user's project root where these paths don't exist.
10. **`domains/sales/agents/deal-strategist.md`** — step 3 (lines ~58-61): playbook searches → plain text. Step 8 (lines ~203-205): drop `--type playbook` from `hero note` (only `--from` exists); graduated patterns are written directly to `.hero/knowledge/playbooks/<slug>.md` titled "Playbook: …", no `type:` frontmatter.
11. **`domains/sales/agents/forecast-analyst.md`** — step 1 (line 32): `hero search --type deal --status "…"` → `hero list --type deal --status prospect,qualifying,demo,proposal,negotiation` (`hero list --status` accepts comma lists; `hero search --status` does not). Step 2 (lines 47-52): drop the `hero.json` `forecast.*` read; methodology defaults to weighted per `forecast-methodology`, weights from per-deal `probability` else the `deal` spec type's stage defaults. Step 6 (line 156): write forecasts to `.hero/reports/forecasts/<period>.md` (reports is an established output dir — cf. `hero supersede --scan`; `planning/forecasts/` is a non-canonical planning dir).
12. **`domains/sales/agents/qualification-analyst.md`** (line 38) — replace "Read `qualification.framework` from `.hero/hero.json`" with: read `qualification_framework` from the deal spec frontmatter; default `meddpicc`.
13. **`domains/sales/agents/buyer-researcher.md`** — grounding block (lines 34-38): keep `hero search "<company>"` and `--type deal`; `--type knowledge` → plain text. "Writing findings to disk" (lines 218-228): a new prospect is a `type: deal` spec at `status: prospect` (same path it already names); knowledge entries go under `.hero/knowledge/prospects/`/`personas/` with **no** work-ish `type:` frontmatter (discovery hazard, `internal/spec/spec.go:1160-1196`).
14. **`domains/sales/agents/competitive-intel.md`** — battlecard template (lines ~36-45): retitle to `Battlecard — Hero vs. [Competitor]` (makes plain-text lookup by the word "battlecard" reliable) and **remove the `type: battlecard` line** (keeps the file out of flat-spec discovery); keep `competitor:`, `updated:`, `win_rate:` and the `.hero/knowledge/battlecards/` location. Update any lookup phrasing to plain-text search.
15. **Sales skills — wrong claims only:**
    - `objection-handling/SKILL.md`: line ~236 drop `--type knowledge` from `hero note`; description + `metadata.audience` → `deal-strategist` (the only agent that lists it in Required skills).
    - `deal-qualification/SKILL.md`: delete the "BANT as an alternative for SMB deals" bullet (line 12 — no BANT content in the body; the BANT table lives in `qualification-analyst.md`); line ~251 qualify-out numeric → "< 25 after 3 qualification conversations — canonical thresholds: `pipeline-management`".
    - `pipeline-management/SKILL.md`: add one line declaring it the single owner of staleness/risk numeric thresholds; `metadata.audience` → `forecast-analyst` (+ the `/pipeline` command; drop `deal-strategist`, which doesn't load it).
    - `forecast-methodology/SKILL.md`: lines 137-138 staleness numerics → defer to `pipeline-management`'s stale-deal table (which sets Negotiation at 7 days, not "10+"); line 216 `hero search --type retro` → plain-text (`hero search "debrief <competitor or segment>"`); line 231 bare `forecast.md` → "the `/forecast` command".
    - `discovery-questioning/SKILL.md`: description + `metadata.audience` → `buyer-researcher` (neither `deal-strategist` nor `qualification-analyst` lists it).
    - `competitive-positioning/SKILL.md`: line 20 "See `competitive-intel.md`" → "the full battlecard template is carried by the `competitive-intel` agent" (no bare filename).
    - Leave the "What this skill covers" boilerplate ×7 in place — trimming belongs to `token-efficiency-pass`.
16. **Sales commands:**
    - `research.md` lines 9-12: `hero research …` ×5 → `/research …`; lines 17-18: `--type battlecard/knowledge` → plain text.
    - `strategize.md` lines 29-30: `--type playbook/battlecard` → plain text; line 76: replace the `/review <slug>` suggestion (engineering-only command) with "Run `/qualify <slug>` to re-score the deal against the new plan".
    - `prospect.md` line 19: `hero search --type prospect` → `hero list --type deal --status prospect` (or plain-text company search); lines 26-27: `--type knowledge "ICP"` → plain text.
    - `forecast.md` line 17: comma-list search → `hero list --type deal --status qualifying,demo,proposal,negotiation`; lines 24-25: "methodology/weights from `hero.json`" → from the `forecast-methodology` skill and deal frontmatter/spec-type defaults.
    - `pipeline.md` line 9: comma-list search → `hero list --type deal --status prospect,qualifying,demo,proposal,negotiation`.

## Boundaries

- **PM pack** phantom surfaces (`hero event`, `hero new`, `hero active`, `hero queue --owner`, phantom slash routes, phantom agents/skills) → sibling **`pm-pack-phantom-surfaces`**.
- **Routing-file structure** — unlisted rosters, `/hero` router drift, slash/CLI parity table provenance, heading-depth/skeleton unification across the three AGENTS.md files, harness-agnosticism scoping → sibling **`routing-file-completeness`**.
- **Token trims** — "What this skill covers" ×7 removal, buyer-researcher's ~700-word research catalog extraction, deal-qualification's duplicated question banks / champion test → sibling **`token-efficiency-pass`**. This spec touches boilerplate only where a claim is factually wrong.
- **No engine changes.** No `hero forecast`/`hero event` CLI shims, no `crm/qualification/forecast/pipeline` config keys, no `installFlat` README exclusion, no FTS changes. If a future spec adds CRM sync, its schema docs land with that code.
- Chat pack, core/engineering content (already resynced by `177e8a1`), and `.hero/mission.md` untouched.

## Risks

- **Loader blast radius.** `spectypes.Load("sales")` runs from `exportSpecTypesCache` on nearly every command in a sales workspace; a malformed `deal.md` (missing `type:`/`category:`, lifecycle referential-integrity failure) breaks all of them. Mitigation: follow `core/spec-types/feature.md` exactly and add/extend a `LoadFromFS` test asserting the sales overlay registers `deal` (see Validation). Do **not** imitate the pm spec-type files.
- **Registered lifecycle vs engine status semantics.** Deal statuses (`prospect`…`lost`) are not the generic work statuses, so `hero queue` readiness and `hero spec verify` won't treat deals like features. The rewritten `hero status`/`hero queue` descriptions must stay generic to avoid re-introducing overclaim.
- **Plain-text lookup recall** depends on the discriminating word being in the document — mitigated by the "Battlecard — …"/"Playbook: …" retitles; existing user battlecards created under the old template won't match until touched (acceptable: net-new guidance).
- **Behavioral change for existing deal files**: any user file already carrying `type: battlecard`/`prospect` remains a phantom work spec until edited; the fix prevents new pollution but doesn't migrate old files.
- Line numbers cited above drift trivially; anchor edits on the quoted text, not the numbers.

## Acceptance Criteria

- THE SYSTEM SHALL contain no occurrence of `hero read-spec`, `hero pulse`, `hero forecast`, `hero research`, `hero search --match`, `hero note --type`, or `hero search --type playbook|battlecard|prospect|knowledge|retro` anywhere under `domains/sales/`.
- WHEN `spectypes.Load("sales")` (or `LoadFromFS` with the sales overlay) runs THE SYSTEM SHALL register a `deal` spec type with lifecycle `prospect → qualifying → demo → proposal → negotiation → won|lost` and no load error.
- WHEN each `hero …` invocation cited in `domains/sales/` is executed against the current binary THE SYSTEM SHALL exit without an unknown-command or unknown-flag error.
- THE SYSTEM SHALL document no `.hero/hero.json` key in `domains/sales/` that `internal/config/config.go` does not read.
- THE SYSTEM SHALL state sales staleness and risk numeric thresholds in exactly one file, `domains/sales/skills/pipeline-management/SKILL.md`, with siblings cross-referencing it without numbers.
- WHEN a battlecard or playbook is saved per the pack's save instructions and then looked up per the pack's lookup instructions THE SYSTEM SHALL return the saved entry.
- IF a battlecard is authored per the updated `competitive-intel` template THEN THE SYSTEM SHALL NOT surface it in work-spec discovery (`hero list`).
- WHEN `hero install project <tmp> --domain sales` renders the managed AGENTS.md body THE SYSTEM SHALL emit no relative link that resolves only inside the hero-engine repo.
- THE SYSTEM SHALL have every sales skill's description/`metadata.audience` "Loaded by" claim match an agent's Required-skills list (or the loading command) exactly.

## Validation

1. **Phantom sweep**: `rg -n "hero read-spec|hero pulse|hero forecast|hero research|--match|hero note .*--type|--type (playbook|battlecard|prospect|knowledge|retro)" domains/sales/` → zero hits.
2. **Loader**: `go test ./internal/spectypes/...`; add (or extend) a test calling `LoadFromFS(coreFS, salesSpecTypesFS, "sales")` asserting `deal` registers with the 7-state lifecycle. Then smoke: in a temp workspace with `"domain": "sales"`, run `hero status` and confirm no spec-types cache error.
3. **CLI reality check**: script every `hero` invocation quoted in `domains/sales/` through `hero <cmd> --help` / a dry run in the temp workspace — all exit 0 without "unknown command"/"unknown flag" (in particular `hero sprint status --week`, `hero list --type deal --status prospect,qualifying,demo,proposal,negotiation`, `hero list --type deal --stale 14`).
4. **Save/lookup agreement**: in the temp sales workspace, create a battlecard per the updated template; `hero search "battlecard acme"` returns it; `hero list` does **not** list it. Create a deal spec at `status: prospect`; `hero list --type deal --status prospect` returns it.
5. **Install render**: `hero install project <tmpdir> --domain sales --target claude`; grep the emitted AGENTS.md/CLAUDE.md managed region for `](commands/`, `](agents/`, `](skills/`, `](spec-types/` → zero hits; confirm the CLI/slash blocks are separated and the config JSON block is gone.
6. **Roster check**: for each sales skill, diff its description/`metadata.audience` against the Required-skills sections of the five agents (+ `/pipeline`, `/forecast` command loads) — no contradictions.
7. **Repo gates**: `go test ./...` (content parity + docs drift tests must stay green; sales files have no core twins, so the parity gate is unaffected by design — verify).
