---
title: PM Foundation Delivery — Ship PM Pack Additively + Unblock hero-code
type: feature
status: planning
priority: P0
tags: [sprint, delivery, pm, foundation, registry, vocabulary, methodology]
created: 2026-05-17
relations:
  - target: hero-domains
    kind: parent
  - target: unified-spec-type-model
    kind: delivers
  - target: spec-type-registry
    kind: delivers
  - target: inline-propose-output-mode
    kind: delivers
  - target: hero-pm
    kind: delivers
horizon: now
smoke: deferred
---

## Kickoff

This is the **delivery sprint** for the work-tracking refactor designed across `unified-spec-type-model`, `spec-type-registry`, and `inline-propose-output-mode`. Goal: ship PM as an additive Hero domain pack, with engineering unchanged, and unblock hero-code (Rust dashboard) to start building PM and QA UI against stable contracts.

The sprint is composed of **nine work items** organized into three tracks (content authoring, Go implementation, content alignment). Items can be picked up in fresh sessions independently where dependencies allow.

**Sprint completes when:**
- All nine canonical work-tracking spec types are registered (`initiative`, `prd`, `epic`, `feature`, `bug`, `chore`, `intake`, `release`, `sprint`)
- Five methodology profiles are authored (Scrum, Kanban, Shape Up, Waterfall, Scrumban)
- `internal/vocabulary/`, `internal/methodology/`, `internal/tasks/` Go packages are delivered and tested
- `spec-type-registry` Go implementation exports schema 1.1 to `.hero/cache/spec-types.json`
- `hero task` CLI surface ships (additive, AC infrastructure unchanged)
- Inline-propose Go side ships
- PM pack content is aligned to final canonical names
- `domain-plugin-architecture` finishing touches are in
- hero-code consumes the three stable contracts (registry / vocabulary / methodology)

→ Each work item below has a paste-ready kickoff prompt for a fresh session.

**Files:** `.hero/planning/features/pm-foundation-delivery/spec.md`, `.hero/planning/features/unified-spec-type-model/spec.md` (authoritative design), `.hero/planning/features/spec-type-registry/spec.md`, `.hero/planning/features/inline-propose-output-mode/spec.md`, `.hero/planning/features/hero-pm/spec.md`, `.hero/planning/features/domain-plugin-architecture/spec.md`, `domains/pm/`, `core/vocabularies/`, `internal/vocabulary/`

**Skip:** Anything outside the work-tracking refactor scope (knowledge graph, meta types, dashboard UI implementation, new tracker providers, migration of any kind).

## Goal

Deliver the unified-spec-type-model architecture as **PM-additive code and content** in this repo. By end of sprint, a fresh `hero` workspace can:

- Declare `methodology: scrum` + `vocabulary: agile-scrum` in `hero.json`
- Author a `feature` artifact that renders as "Story" in CLI, dashboard, and agent output
- Add `## Tasks` to a feature spec and manage them via `hero task add | list | done | history`
- Have a PM agent propose AC inline on an artifact via `--inline-propose`
- Hand off a `feature` from PM to engineering via owner-flip on the same artifact (no separate spec)
- Render `release` and `sprint` time-box artifacts with methodology-specific behavior
- Continue to author `feature` / `bug` / `initiative` specs as engineering does today, unchanged

And hero-code (Rust dashboard), reading three stable contracts:
- `.hero/cache/spec-types.json` — nine canonical types + kinds + lifecycles
- `core/vocabularies/*.yaml` — display names + tracker mappings
- `core/methodologies/*.yaml` — lifecycle / time-box / estimation / rituals / rollups

Can begin authoring the PM and QA UI without further blocking on Hero (Go) side.

## Work items

### Track A — Content authoring (parallelizable; no Go work)

#### A1. Author `core/spec-types/` nine canonical type files

Author the nine canonical work-tracking type-record markdown files at `core/spec-types/`:
- `initiative.md`, `prd.md`, `epic.md`, `feature.md`, `bug.md`, `chore.md`, `intake.md`, `release.md`, `sprint.md`

Each file declares: frontmatter (YAML), lifecycle states + transitions + owner-flip annotations, `kind` block (canonical values per Decision 2 of `unified-spec-type-model`), `tasks_schema` (mirrors AC schema; section heading `Tasks`; item shape with id/text/status/kind/assignee/discovered_against/timestamps), `owner` declaration, sections (required + optional), accepting_commands, default_agents, relations.

Use `.hero/planning/features/spec-type-registry/spec.md` §6 (Go type record shape) and §13 (Core reference registration) as the authoritative templates. Use existing PM pack spec-type files (`domains/pm/spec-types/prd.md`, `domains/pm/spec-types/intake-item.md`) as shape references — but author **fresh** at the new canonical location with correct names (`intake` not `intake-item`).

**Sizing:** ~1 day content authoring.

**Kickoff prompt:**
> Read `.hero/planning/features/unified-spec-type-model/spec.md` (Decisions 1, 2, 7, 8) and `.hero/planning/features/spec-type-registry/spec.md` (§6 type record shape, §13 reference registration). Then author the nine canonical type-record markdown files at `core/spec-types/{initiative,prd,epic,feature,bug,chore,intake,release,sprint}.md`. Use `domains/pm/spec-types/prd.md` and `domains/pm/spec-types/intake-item.md` as shape references but author at canonical names — `intake` not `intake-item`. Each file declares frontmatter, lifecycle, kind block, tasks_schema, owner, sections, accepting_commands, default_agents, relations. Do not invent fields or types not in the design. Do not author methodology-specific lifecycle states — those live in methodology profiles (separate work item). Use the unified default lifecycle per type (e.g. `planning → refined → ready → delivering → in-review → completed` for feature/bug/chore). Report files written and any open questions, under 300 words.

#### A2. Author `core/methodologies/` five v1 methodology profiles

Author five methodology profile YAML files at `core/methodologies/`:
- `scrum.yaml`, `kanban.yaml`, `shape-up.yaml`, `waterfall.yaml`, `scrumban.yaml`

Each file declares: aligned vocabulary, lifecycle overrides per type (where methodology-specific), time_boxes (release / sprint requirements + duration + rituals), estimation field, in-flight tracking style, cadence rituals, rollups. Use `.hero/planning/features/unified-spec-type-model/spec.md` Decision 5 as the authoritative format (the scrum.yaml excerpt there is the template).

**Sizing:** ~1 day content authoring.

**Kickoff prompt:**
> Read `.hero/planning/features/unified-spec-type-model/spec.md` Decision 5 (Methodology profile system) and the methodology × time-box / lifecycle / estimation / rituals / rollup table in that section. Then author five methodology profile YAML files at `core/methodologies/{scrum,kanban,shape-up,waterfall,scrumban}.yaml`. Each file: name, display_name, description, aligned_vocabulary, lifecycle overrides per spec type (where methodology-specific), time_boxes (release + sprint requirements + duration + rituals), estimation (required_field per type), in_flight_tracking, cadence, rollups. Use the scrum.yaml excerpt in the design spec as the canonical template. Report files written, sample distinctive line per file, and any open questions, under 300 words.

#### A3. PM pack content alignment — rename references to final canonical names

Update PM pack content to use final canonical names:
- `roadmap-item` → `initiative` (in agent prompts, skill bodies, command descriptions, AGENTS.md routing, mission table)
- `intake-item` → `intake`
- `spec` (as a type literal) → `feature`
- Inside skills/agents where "story" is used as the methodology word, leave it — vocabulary preset handles rendering

Touch: `domains/pm/mission.md`, `domains/pm/AGENTS.md`, all `domains/pm/agents/*.md`, all `domains/pm/skills/*/SKILL.md`, all `domains/pm/commands/*.md`, `domains/pm/spec-types/README.md`, `domains/pm/agents/README.md`, `domains/pm/commands/README.md`, `domains/pm/skills/README.md`.

Rename `domains/pm/spec-types/intake-item.md` → `domains/pm/spec-types/intake.md`; remove `roadmap-item.md` from `domains/pm/spec-types/` (lives in `core/spec-types/` now via item A1); remove `story.md` (collapses into `core/spec-types/feature.md`).

Keep agent filenames where they're vocabulary-aware (`story-writer.md` stays — vocabulary renders the rest).

**Sizing:** ~half-day content alignment.

**Kickoff prompt:**
> Read `.hero/planning/features/unified-spec-type-model/spec.md` Decision 1 (the nine canonical type names) and `.hero/planning/features/pm-foundation-delivery/spec.md` work item A3. Then update PM pack content to use the final canonical names: `roadmap-item` → `initiative`, `intake-item` → `intake`, type literal `spec` → `feature`. Touch mission.md, AGENTS.md, all agents/*.md, all skills/*/SKILL.md, all commands/*.md, READMEs in spec-types/ agents/ skills/ commands/. Rename intake-item.md → intake.md. Remove roadmap-item.md, epic.md, story.md from domains/pm/spec-types/ (they're now in core/spec-types/). Keep prd.md in domains/pm/spec-types/ as PM-led; keep agent filenames (e.g. story-writer.md stays — vocabulary handles rendering). In prose, leave "story" alone where it describes the agile methodology concept; only update where it's a type literal. Report what you touched and any decisions you made, under 400 words.

### Track B — Go implementation (sequenced; depends on Track A items A1 and A2)

#### B1. Finish `domain-plugin-architecture` cutover

The Go work for primitive #1 (`domain-plugin-architecture`) is ~80% in the working tree. Finish:

- Add `domains/pm/` to the `embed.FS` declarations in `content.go`
- Decide on the ContentFS migration: cut `ContentFS()` over to read from `domains/engineering/`, OR leave the legacy fallback. Either is fine; document the choice.
- Verify `hero init --domain pm` works against the PM pack
- Verify `hero domain list` shows `engineering`, `sales`, `pm` (and `core` if appropriate)
- Verify `hero domain switch pm` reinstalls content from the PM pack

Read `.hero/planning/features/domain-plugin-architecture/spec.md` for the design. Look at `content.go`, `internal/cli/domain.go`, `internal/cli/init.go`, `internal/install/install.go` for the current state.

**Sizing:** ~1 day Go.

**Kickoff prompt:**
> Read `.hero/planning/features/domain-plugin-architecture/spec.md` and the current state of `content.go`, `internal/cli/domain.go`, `internal/cli/init.go`, `internal/install/install.go`. The work is ~80% done in the working tree. Finish: add `domains/pm/` to embed.FS declarations in content.go (alongside engineering, sales). Verify `hero init --domain pm` creates a PM workspace. Verify `hero domain list` shows engineering, sales, pm. Verify `hero domain switch pm` reinstalls content. Decide whether to cut `ContentFS()` over to read from `domains/engineering/` now or leave the legacy fallback (small decision; document it). Add tests for the embed.FS surface. Run `go build ./...` and `go test ./...` clean. Report what you delivered, the ContentFS decision, and any open issues, under 300 words.

#### B2. `spec-type-registry` Go implementation — loader, kind, schema 1.1 export

Implement the spec-type registry in Go:

- New `internal/spectypes/` package with `Record`, `Registry`, `Kind` types
- Loader reads markdown frontmatter from `core/spec-types/*.md` and `domains/<active>/spec-types/*.md`
- YAML-based JSON-Schema-Lite frontmatter schema parser
- `Lookup(name)`, `LookupByKind(name, kind)`, `WorkTypes()`, `KnowledgeTypes()`, `Kinds(typeName)`, `DefaultWorkType()` accessors
- Schema 1.1 JSON export to `.hero/cache/spec-types.json`
- `TestLintParity` test gating the registry vs legacy validator parity

Read `.hero/planning/features/spec-type-registry/spec.md` for the full design. Depends on Track A item A1 (`core/spec-types/` files exist).

**Sizing:** ~3-4 days Go.

**Kickoff prompt:**
> Read `.hero/planning/features/spec-type-registry/spec.md` end-to-end. Verify `core/spec-types/` is populated with the nine canonical type files (Track A item A1). Then implement the registry: new `internal/spectypes/` package with `Record` / `Registry` / `Kind` / `ChecklistSchema` / `OwnerSchema` types per §6 of the spec; markdown loader reading `core/spec-types/*.md` and `domains/<active>/spec-types/*.md` per §1; YAML-based JSON-Schema-Lite schema parser per §2; full accessor surface per §7; schema 1.1 JSON export to `.hero/cache/spec-types.json` per §Cross-language contract; `TestLintParity` test per §4 (no `TestACParity` — AC infrastructure is unchanged). Replace hardcoded type literal `switch` statements in `internal/spec/spec.go`, `internal/lint/`, `internal/cli/new.go`, `internal/cli/list.go`, etc., with registry lookups per §Changes. Keep legacy `Type*` constants as string aliases for the canonical names (no semantic change). Run `go build ./...` and `go test ./...` clean. Report the package shape, tests passing, any deviations from the design, and any open questions, under 500 words.

#### B3. `internal/methodology/` Go package

Mirror `internal/vocabulary/`'s shape: new package that loads methodology profile YAML files, resolves the active methodology via precedence chain, exposes lifecycle / time-box / estimation / rituals / rollups accessors.

Depends on Track A item A2 (`core/methodologies/` files exist).

**Sizing:** ~2-3 days Go.

**Kickoff prompt:**
> Read `internal/vocabulary/` (vocabulary.go, resolver.go) as the reference shape, plus `.hero/planning/features/unified-spec-type-model/spec.md` Decisions 5 and 6 for methodology profile semantics and precedence. Verify `core/methodologies/` is populated with the five v1 profile YAMLs (Track A item A2). Then create `internal/methodology/` package with `Methodology` struct (lifecycle overrides per type, time_boxes, estimation, in_flight_tracking, cadence, rollups, aligned_vocabulary); `Load(coreFS, domainFS) (map[string]*Methodology, error)` loader; `Resolve(cfg) (*Methodology, error)` resolver applying the Decision 6 precedence chain (explicit methodology > tracker-inferred > default; vocabulary auto-derives from aligned_vocabulary unless overridden). Add accessors for lifecycle queries (`Lifecycle(typeName) StateMachine`), time-box requirements (`TimeBoxRequired(level) bool`), estimation field (`EstimationField(typeName) (name, type)`). Add `internal/config/config.go` `Methodology string` and `MethodologyOverrides map[string]string` fields. Embed core/methodologies/ in content.go alongside core/vocabularies/. Write tests covering: load all five profiles; resolve under various config; lifecycle overrides apply correctly; vocab auto-derivation. Run `go build ./...` and `go test ./...` clean. Report package shape, test counts, deviations, open questions, under 400 words.

#### B4. `internal/tasks/` Go package + `hero task` CLI

Implement the tasks sub-element infrastructure (additive; AC stays untouched):

- New `internal/tasks/` package — task parser (reads `## Tasks` markdown checklists), graph node kind `Task`, edge kinds (`discovered_against`, `blocks`, `assigned_to`)
- New `internal/cli/task.go` — `hero task add | list | start | done | history` CLI commands, mirroring `hero ac`'s shape
- Tasks support: id, text, status (todo/doing/done), kind (optional), assignee, discovered_against, started, done

**Do not touch `internal/acceptance/`.** AC stays bit-for-bit.

**Sizing:** ~2-3 days Go.

**Kickoff prompt:**
> Read `internal/acceptance/` (vocabulary, structure, shape) and `internal/cli/ac.go` (CLI surface) as the reference for what to mirror. Read `.hero/planning/features/unified-spec-type-model/spec.md` Decision 3 and `.hero/planning/features/spec-type-registry/spec.md` §2 (tasks_schema). Then create `internal/tasks/` package as an ADDITIVE peer of `internal/acceptance/` — do not modify, rename, or refactor `internal/acceptance/`. Parser reads `## Tasks` markdown checklists per the canonical form (`- [ ]` / `- [/]` / `- [x]` with `{kind, assignee, discovered_against, started, done}` inline). Graph nodes for `Task`; edges `discovered_against` (Task → Spec), `blocks` (Task → Spec), `assigned_to` (Task → Person, optional). Implement `internal/cli/task.go` with `add`, `list`, `start`, `done`, `history`, `status` subcommands mirroring `hero ac` exactly. Tests cover: parse the canonical Tasks format; round-trip with the spec parser; CLI commands work; graph queries function. Run `go build ./...` and `go test ./...` clean. AC infrastructure must not be touched — confirm `git diff internal/acceptance/` is empty. Report what you built, test counts, and any open questions, under 400 words.

#### B5. Inline-propose Go side delivery

Implement the Go side of the inline-propose contract (already designed). New `hero agent run` shim that tails agent stdout for `HERO-PROPOSAL: ` prefixed NDJSON and POSTs to a new daemon endpoint; daemon proposal store; five SSE event types on `/api/events`; REST POSTs for accept / edit-accept / reject (and bulk variants).

Read `.hero/planning/features/inline-propose-output-mode/spec.md` for the full design including the proposal envelope schema, transport details, and the cross-language contract.

**Sizing:** ~1 week Go.

**Kickoff prompt:**
> Read `.hero/planning/features/inline-propose-output-mode/spec.md` end-to-end. The design is locked. Implement the Go side: (1) New `hero agent run` shim that tails agent stdout for `HERO-PROPOSAL: ` prefixed NDJSON lines and POSTs to `/api/{project}/sessions/{id}/proposals/ingest`; (2) Daemon proposal store (in-memory, transient session state per Decision 1 of the spec) with per-anchor replacement scoped to same agent (Decision 2); (3) Five new SSE event types on existing `/api/events` stream: `proposal_emitted`, `proposal_accepted`, `proposal_edited`, `proposal_rejected`, `proposal_dismissed`; (4) REST endpoints for accept / edit-accept / reject including bulk variants on batch_id; (5) Agent prompt addendum for `--inline-propose` mode (the agent emits structured proposals instead of writing to disk); (6) Command router thread-through for `--inline-propose` flag from slash commands. Wire to existing `hero serve` daemon on localhost:7437. Tests cover: round-trip a proposal from stdout to dashboard; accept / edit / reject lifecycle; per-anchor replacement; bulk operations; lifecycle log line generation. Mirror the envelope schema to `docs/contracts/inline-propose-v1.md` for hero-code consumption. Run `go build ./...` and `go test ./...` clean. Report what you built, the envelope schema actually shipped, deviations from the design, and any open questions, under 500 words.

#### B6. Vocabulary + methodology-aware rendering spread

Wire the vocabulary and methodology resolvers into Hero's user-visible surfaces:

- `hero list`, `hero active`, `hero blocked`, `hero queue`, `hero new`, `hero status` — render type/kind via active vocabulary
- MCP dashboard tools — emit display names per active vocabulary
- `NEXT.md` generators — use methodology-aware language
- Agent prompt scaffolds — load active vocabulary/methodology into prompt context
- Natural-language routing in AGENTS.md — recognize vocabulary-shaped terms ("create a story" → `hero new feature` under agile-scrum)

**Sizing:** ~3-5 days Go.

**Kickoff prompt:**
> Verify `internal/vocabulary/` and `internal/methodology/` packages are delivered (B3 and Phase A pt 1). Then wire vocabulary + methodology-aware rendering into Hero's user-visible surfaces. Touch: `internal/cli/list.go`, `internal/cli/active.go`, `internal/cli/blocked.go`, `internal/cli/queue.go`, `internal/cli/new.go`, `internal/cli/status.go`, `internal/cli/dashboard.go`, `internal/cli/report.go` — render type/kind via `vocab.Display(type, kind)`. `internal/serve/mcp_tools.go` — emit display names per active vocabulary. `internal/cli/next.go` (and the next-md skill rendering) — use methodology-aware language. Agent prompt scaffolds (search for prompt builders that inject AGENTS.md content) — load active vocabulary and methodology into the prompt context so agents speak the right dialect. Update `domains/pm/AGENTS.md` routing table and engineering's `domains/engineering/AGENTS.md` routing — recognize vocabulary-aware natural-language terms ("create a story" routes to `hero new feature` under agile-scrum). Tests: render-under-each-vocab golden tests; routing recognition under each vocab. Run `go build ./...` and `go test ./...` clean. Report files touched, test approach, and any open questions, under 400 words.

### Dependency graph

```
A1 (core/spec-types/)   ─┐
                         ├──→ B2 (registry Go) ──┐
A2 (core/methodologies/) ┘                       ├──→ B6 (rendering spread)
                                                 │
A3 (PM pack alignment)   ───────────────────────┘
                                                 │
B1 (#1 cutover)          ───────────────────────┤
B3 (internal/methodology/) ─────────────────────┤
B4 (internal/tasks/ + hero task CLI) ───────────┤
B5 (inline-propose Go) ─────────────────────────┘
```

A1, A2, A3, B1 can all run in parallel. B2 depends on A1. B3 depends on A2. B4 and B5 can run anytime after Track A. B6 depends on B2, B3, A3.

Net critical path: A1 → B2 → B6. Other items run in parallel branches.

## Acceptance Criteria

- THE SYSTEM SHALL register nine canonical work-tracking spec types from `core/spec-types/*.md` files: `initiative`, `prd`, `epic`, `feature`, `bug`, `chore`, `intake`, `release`, `sprint`.
- THE SYSTEM SHALL register engineering's `decision` and `convention` types from `domains/engineering/spec-types/*.md` files.
- THE SYSTEM SHALL load methodology profiles from `core/methodologies/*.yaml` and expose lifecycle / time-box / estimation accessors.
- THE SYSTEM SHALL load vocabulary presets from `core/vocabularies/*.yaml` (already delivered) and resolve via precedence chain.
- WHEN `methodology:` is set and `vocabulary:` is unset in `hero.json`, THE SYSTEM SHALL auto-derive the vocabulary from the methodology's `aligned_vocabulary` field.
- WHEN a spec lists `## Tasks` section as a checklist, THE SYSTEM SHALL parse each task into a structured `Task` record.
- THE SYSTEM SHALL provide `hero task add | list | start | done | history` CLI subcommands.
- THE SYSTEM SHALL NOT modify `internal/acceptance/` or the `hero ac` CLI surface during this sprint.
- THE SYSTEM SHALL export `.hero/cache/spec-types.json` at schema version 1.1 with the nine canonical types, their kind enums, lifecycles, frontmatter schemas, and folder mappings.
- THE SYSTEM SHALL support `--inline-propose` mode on agent invocations, emitting structured proposals via `hero agent run` shim to the daemon proposal store.
- WHEN a workspace sets `vocabulary: agile-scrum`, THE SYSTEM SHALL render a `feature` as "Story" in CLI list output, MCP dashboard, agent output, and NEXT.md.
- WHEN PM hands off an artifact to engineering via `/handoff`, THE SYSTEM SHALL flip the `owner` field on the same artifact (not create a separate spec).
- THE SYSTEM SHALL preserve all existing engineering specs (137 features, 16 bugs, 14 initiatives) unchanged through the registry implementation.
- THE SYSTEM SHALL provide the inline-propose envelope contract to hero-code at `docs/contracts/inline-propose-v1.md`.

## Boundaries

- **Not** modifying `internal/acceptance/` or the `hero ac` CLI surface. AC stays bit-for-bit.
- **Not** migrating any existing engineering specs. Folders unchanged; frontmatter unchanged.
- **Not** modifying meta / knowledge types: `decision`, `convention`, `plan`, `reference`, `external`, `note`, `rule`, `tripwire`, `context`. They work today; out of scope for this sprint.
- **Not** building the Rust dashboard UI. Hero-code consumes the contracts; that work runs in parallel in the hero-code repo.
- **Not** adding new tracker integration providers beyond Jira / Linear / GitHub.
- **Not** adding OKR support, multi-domain coexistence concurrency, or user-defined custom spec types beyond `kind` sub-typing.
- **Not** changing the on-disk markdown format beyond adding new folders for new types and the new `## Tasks` section.

## Risks

- **Track A items drift from the design specs.** Mitigation: each work item's kickoff prompt explicitly anchors to the relevant decision spec sections. Authoring agents should not invent fields or types not in the design.
- **Vocabulary + methodology two-layer composition confuses authors.** Mitigation: `hero status` should display both active layers prominently; documentation in `core/vocabularies/README.md` and `core/methodologies/README.md` (forthcoming) explains the orthogonal relationship.
- **`hero agent run` shim breaks existing agent invocation.** Mitigation: shim is opt-in via `--inline-propose` flag; write-to-disk mode remains the default for all non-interactive flows.
- **Hero-code consumes contract before it stabilizes.** Mitigation: schema 1.1 is the locked contract version; any future changes bump to 1.2 with additive-only semantics; `schema_version` field gates compatibility.
- **Scope creep into the knowledge / meta layer.** Mitigation: this spec's Boundaries section is binding; any temptation to "fix" `plan`, `reference`, etc. during this sprint is a follow-up, not in scope.

## Sprint completion checklist

- [x] A1: nine spec-type files at `core/spec-types/`
- [x] A2: five methodology profiles at `core/methodologies/`
- [x] A3: PM pack content aligned to final canonical names
- [x] B1: `domain-plugin-architecture` cutover finished
- [x] B2: `internal/spectypes/` Go package; schema 1.1 JSON export verified (wired into `cli.PersistentPreRun` so `.hero/cache/spec-types.json` refreshes on every command)
- [x] B3: `internal/methodology/` Go package; tests green
- [x] B4: `internal/tasks/` Go package + `hero task` CLI; AC infrastructure untouched
- [x] B5: Inline-propose Go side; envelope contract published to `docs/contracts/inline-propose-v1.md` (shipped contract is `1.0` semver; divergences from original design documented in the inline-propose spec)
- [x] B6: Vocabulary + methodology-aware rendering across CLI / MCP / NEXT.md / agent prompts
- [x] Hero-code peer call (advisory) handing over the three stabilized contracts
- [x] `hero status` shows active methodology + vocabulary
- [x] `go test ./...` clean
- [x] Engineering corpus (137 features, 16 bugs, 14 initiatives) unchanged (registry `TestLintParity` walks 187 specs against the legacy validator — zero drift)

When all checked, sprint is done; hero-code is unblocked; PM ships.
