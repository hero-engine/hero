# Hero Content Audit — Report

Spec: [spec.md](spec.md) · Base SHA: `bc86ad9` · Date: 2026-07-05
Inputs: [inventory.md](inventory.md) · [rubric.md](rubric.md) ·
[dup-map.txt](dup-map.txt) · per-surface findings:
[agents](findings-agents.md) · [commands](findings-commands.md) ·
[skills](findings-skills.md) · [routing](findings-routing.md)

## Executive summary

All 227 shipped content files were audited against the 7-dimension rubric.
The per-surface passes produced **120 raw findings (31 S1 · 54 S2 · 35 S3)**;
after cross-pass deduplication they collapse into **five systemic themes**.
(Counts are post-cold-audit: one S1 was withdrawn as false after independent
re-verification — see delivery-audit.md.)
The content is *not* uniformly bad — engineering's routing claims are 100%
real, sales' rosters are complete, most command descriptions trigger well,
and the stack micro-skills earn their place. The damage concentrates in two
places: **the core↔engineering duplicate set has silently forked** (with the
stale copy winning for someone in every case), and **the pm/sales packs ship
instructions verified against nothing** (phantom commands, fictional CLI
flags, config schemas the engine never reads).

Every S1 was verified against the `hero` binary **and** the Go source before
being filed. Zero edits were made to `core/` or `domains/`.

## Coverage

| Pass | Files | Read | Clean | Flagged | Doc-only |
|---|---|---|---|---|---|
| Agents | 58 | 58 | 14 | 42 | 2 |
| Commands | 72 | 72 | 31 | 39 | 2 |
| Skills | 94 | 94 | 25 | 67 | 2 |
| Routing (AGENTS.md ×3) | 3 | 3 | 0 | 3 | — |
| **Total** | **227** | **227** | **70** | **151** | **6** |

(Clean/flagged splits are taken from the per-file verdict tables in the
findings files. "Flagged" includes files whose only flag is membership in
the dedup/drift matrix.)

(The 6 domain READMEs are counted inside their surface rows as doc-only;
`core/commands/hero.md` is audited both as a command and as a routing
surface. Per-file verdicts are in each findings file's coverage table.)

## The five themes

### T1 — The core↔engineering duplicate set has forked, and the stale copy always wins for someone

34 same-named files exist in both `core/` and `domains/engineering/`
(13 skills, 17 commands, 4 agents). Engine precedence is
`OverlayFS(domain, core)` — domain wins — so:

- **20 byte-identical copies** (8 skills, 9 commands, 3 agents) are pure
  dead weight in the engineering pack; deleting them changes nothing.
- **14 forked pairs** are all classified **accidental drift** (git evidence:
  fork origin `92c94aa`; every divergence is one-sided, none is domain
  specialization beyond ~1 paragraph). Because each copy is live for a
  different audience, drift ships stale guidance in *both* directions:
  - **Engineering installs (the default)** get `spec-format` and
    `context-injection` copies that never received the supersede-genealogy
    update (`d7fd9e9`) — they still teach the deprecated
    `status: superseded` hand-edit and never see superseded-spec handling.
  - **pm/sales installs** get core copies frozen at v0.8.0: a `session-primer`
    with nonexistent `hero status --delivering/--claimed` flags, an
    `import.md` instructing `hero import --preset/--jql` (a live bug —
    those flags belong to `hero sync import`), and `agent-reliability` /
    `next-md` / `next-handoff-emit` missing two generations of universal
    rules (including "never hand-edit `.local.md`; the checkpoint wipes it").

There is no sync mechanism and no CI check. Full matrix:
[findings-skills.md §(b)](findings-skills.md), commands table in
[findings-commands.md §D](findings-commands.md).

### T2 — Phantom surfaces: content routes to things that don't exist

The pm pack is the worst offender; sales second; and six *core* commands
dangle on non-engineering installs.

- **PM**: the AGENTS.md routing table routes to **10+ slash commands that
  don't install** (`/interview`, `/capacity`, `/plan-*`, `/standup`,
  `/scrub`, `/diagnose`, `/search`, `/review` — its own README says v1.5+);
  the CLI section teaches **six wrong invocations** (`hero event`,
  `hero active`, `hero queue --owner`, `hero new epic`, `hero import`-as-
  tracker-import, `hero search --kind`); `pm-delivery-lead` allowlists and
  routes to **8 ghost agents**; skills carry **11 phantom skill refs and
  9 phantom agent refs**; and the pack's lifecycle vocabulary
  (`drafted/refined/ready/…`) isn't the engine's status set.
- **Sales**: session-start step 2 fails immediately (`hero read-spec`);
  `hero forecast`, `hero pulse --week`, `--match` don't exist; the entire
  "Domain Configuration" hero.json schema (`crm`/`qualification`/`forecast`/
  `pipeline`) has **no counterpart in config.go**; `spec-types/deal.yaml`
  is never loaded (loader reads `.md` only); `hero search --type
  playbook|battlecard` filters on unregistered types and silently returns
  nothing — the agent concludes no intel exists.
- **Core-on-pm/sales**: `check`/`decide`/`discover`/`retro`/`drive`/`hero`
  delegate to engineering-only agents and skills that a pm/sales install
  doesn't ship; `/hero`'s workflow list routes to 6 phantoms there too.
- **Cross-pack**: `hero event` is instructed in ~10 files across pm agents,
  commands, and AGENTS.md — the real surfaces are `hero agent events` (CLI)
  or the `hero_event` MCP tool.

### T3 — Process-gate contradictions inside the delivery doctrine

Three shipped files contradict the four-gate `hero spec verify` doctrine
that `deliver.md` and the engineering AGENTS.md declare non-negotiable:

- `platform-delivery-lead` is a **stale fork of feature-delivery-lead**
  frozen at an older process: no Completion Ledger, no cold audit, and step
  13 instructs hand-editing `status: completed` and moving the spec.
- `sprint.md` execute mode instructs `hero spec complete` directly,
  bypassing verify.
- `handoff-coordinator`'s owner-flip mechanics are wrong end-to-end (false
  `owner_history` mechanism, wrong paths, nonexistent statuses and flags) —
  its own verification step fails every time.

### T4 — Install-reality mismatches: content written in this repo, for this repo

- **Dogfood path leakage**: PM's "Project Structure" ships `domains/pm/…`
  and `core/vocabularies/…` source paths into user repos (the exact failure
  `agents_md.go:283-288` warns about); engineering skills reference
  `internal/sizing/ambient.go`, `CROSS-REPO-PEERING.md`, sibling spec slugs;
  the `drive` skill's arming step needs `scripts/drive/stop-hook.sh`, which
  ships nowhere.
- **Links die at install**: sales AGENTS.md's ~20 relative links and the
  `skills/next-md.md`-style cross-refs 404 in the installed layout.
- **Claude-only machinery unscoped**: engineering AGENTS.md's "Internal
  Lookups" section (`mcp__hero__*` naming, `Explore` agent, `ToolSearch`),
  command files invoking "Task agents"/"general-purpose subagent", and the
  Stop-hook-dependent NEXT.md machinery presented as universal — hooks
  install only for claude + codex; cursor/copilot/generic get none.
- **Fossils**: `compatibility: opencode` on 80 skills has zero engine
  consumers and is wrong for 5 of 6 targets; `role:` frontmatter is dead in
  17 agents; `domains:` is inert everywhere it's set.
- **The slash/CLI parity table exists only in this repo's installed
  CLAUDE.md managed region** — no pack or Go source generates it, so the
  next `hero install` silently drops it (and it's missing three "Both" rows).
- **Engine bug found in passing**: `installFlat` has no README exclusion, so
  pm/sales `agents/README.md` + `commands/README.md` install as
  frontmatter-less pseudo-agents/commands.
- **Dead pack**: `domains/chat/` (6 commands) is not embedded in
  `content.go` at all — unreachable content; if wired as-is it would inherit
  the engineering routing table it can't satisfy.

### T5 — Token waste and invisible content

- ~9,900 words of duplicated skill source (T1) plus systematic restating:
  `deliver.md` (2,673 w) can shed ~1,100–1,300 words that its skills own;
  `feature-delivery-lead` (2,685 w, corpus max) ~40%; `peer.md` ~400 w;
  the architectural stance is triplicated across three agents that all
  *also* load the skill that owns it; the Completion Ledger contract lives
  in `engineer.md` but is consumed by three surfaces; pm/sales skills
  retell each other's frameworks (betting table ×2, pitch bars ×3,
  question banks ×2, thresholds ×3 with **numeric drift** — staleness is
  7 days in one skill, 10 in another).
- **Invisible content**: engineering AGENTS.md never mentions 11 of 30
  commands, 33 of 35 agents, or any of its 52 skills — sessions can't route
  to what isn't listed. `/hero`, the meta-router, is stale on both rosters.
- **Roster overlap** (agents): `feature`↔`platform-delivery-lead` (merge),
  `deadcode`↔`legacy-scrubber` (merge or sharpen), `dedup`↔`type-scrubber`
  (cut the overlap), the five ~145-word "engineer + load skill X" specialist
  stubs (cut or differentiate), `prime`↔`resume` commands both claiming the
  session-start slot with no disambiguation.

## Top S1 findings (deduplicated, ranked by severity × blast radius)

| # | Finding | Blast radius | Source |
|---|---|---|---|
| 1 | `convention-writing` (core+eng, identical) points at `.hero/conventions/` — engine truth is `.hero/knowledge/conventions/` | all domains, all 6 targets | skills |
| 2 | Engineering installs teach deprecated supersede + never see superseded-spec handling (`spec-format`, `context-injection` drift) | every default install | skills |
| 3 | core `import.md` instructs `hero import --preset/--jql` — wrong command entirely | every pm/sales install | commands |
| 4 | PM AGENTS.md: 10+ phantom slash routes + 6 wrong CLI invocations, under "run the command — don't just suggest it" | every PM session | routing |
| 5 | Sales AGENTS.md: fictional session-start, CLI, config schema, and spec-type | every sales session | routing |
| 6 | `platform-delivery-lead` bypasses all four delivery gates (stale fork) | platform-scoped `/deliver` | agents |
| 7 | Six core commands delegate to engineering-only agents/skills | every pm/sales install | commands |
| 8 | `pm-delivery-lead` + 4 sibling agents route to 8 ghost agents | PM delivery flows | agents |
| 9 | `handoff-coordinator` mechanics are wrong end-to-end | PM↔eng handoffs | agents |
| 10 | `hero event` CLI instructed in ~10 pm files — command doesn't exist | PM event logging | agents/commands/routing |
| 11 | core `session-primer` uses nonexistent flags (eng twin already fixed) | pm/sales session start | agents |
| 12 | `sprint.md` execute mode bypasses `hero spec verify` | sprint delivery | commands |
| 13 | `drive` skill arming step unexecutable outside this repo | any `/drive` user | skills |
| 14 | PM Project Structure ships source-repo paths to user repos | every PM install | routing |

Full lists (31 S1, 54 S2, 35 S3 with evidence and fix shapes) live in the
four findings files. One further S1 (executive-report's
`hero sprint status --week`) was withdrawn during the cold audit — the flag
exists; the recipe is correct.

## What came back clean (worth saying)

Engineering AGENTS.md's entire slash/CLI/MCP claim set verified real.
Sales rosters 100% complete. 70/72 command descriptions well-formed and
well-triggered. Stack micro-skills (87–167 w) all earn their place. ~60
CLI invocations verified correct across the corpus (lists recorded in each
findings file so future passes don't re-check).

## Proposed follow-up specs

| # | Slug (proposed) | Scope (one line) | Type | Size |
|---|---|---|---|---|
| 1 | `content-dedup-resync` | Delete the 20 identical eng copies; backport the 14 forks to single masters (universal→core, eng-delta stays); add CI parity check for same-named core/domain files | bug | medium |
| 2 | `pm-pack-phantom-surfaces` | Fix PM AGENTS.md routing+CLI, ghost agents/skills (scope as P1 or delete), lifecycle vocabulary → engine statuses, handoff-coordinator mechanics | bug | medium |
| 3 | `sales-pack-reality-sync` | Fix sales AGENTS.md CLI/session-start, delete-or-implement the config schema, convert `deal.yaml` → `.md` spec type, fix `--type` search idioms + threshold drift | bug | medium |
| 4 | `core-commands-domain-neutral` | Fix core `import.md` (live bug), give the six dangling core commands domain-neutral fallbacks, regenerate `/hero` router per-domain, resolve `prime`↔`resume` | bug | medium |
| 5 | `delivery-gate-consistency` | Merge `platform-delivery-lead` into feature (thin delta), fix `sprint.md` verify bypass, extract Completion Ledger contract to a skill | enhancement | medium |
| 6 | `harness-agnosticism-sweep` | Scope Claude-only machinery, Stop-hook caveats for hookless targets, move parity table into pack source (+Go fallback), strip `compatibility:`/`role:` fossils, de-dogfood shipped paths | enhancement | medium |
| 7 | `routing-file-completeness` | Add engineering AGENTS.md rows/rosters for unlisted content, align the three pack skeletons + heading depth, fix install-dead links | enhancement | small |
| 8 | `token-efficiency-pass` | The named cuts: deliver.md, feature-delivery-lead, peer.md, diagnose batch protocol, architecture-stance triplication, pm/sales skill dedup | enhancement | medium |
| 9 | `installflat-readme-exclusion` | Engine fix: exclude READMEs in `installFlat` (loader already does) | bug | trivial |
| 10 | `chat-pack-decision` | Decide: wire the chat domain (embed + AGENTS.md) or move it out of `domains/` | decision | small |

Sequencing note: #1 first — it removes ~34 files' worth of double-edit
hazard, so every later fix lands once. #9 is a one-liner an engineer can
take anytime. #2/#3 are independent. #6 overlaps the
`harness-changes-cover-all-targets` tripwire and the in-flight
`agent-safety-conventions` work — check before starting. Structural
questions (where content *lives*) stay with `core-vertical-layering`;
this report hands it T1 as evidence.
