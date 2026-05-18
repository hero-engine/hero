---
title: Unified Spec-Type Model — Nine Real-Named Types, Methodology + Vocabulary Adaptation
slug: unified-spec-type-model
type: feature
status: designed
priority: P0
tags: [architecture, decision, types, vocabulary, methodology, registry]
created: 2026-05-17
updated: 2026-05-17
relations:
  - target: hero-domains
    kind: parent
  - target: hero-pm
    kind: unblocks
  - target: spec-type-registry
    kind: drives
  - target: inline-propose-output-mode
    kind: related
  - target: hero-code
    kind: cross-repo-consumer
horizon: now
smoke: deferred
---

## Kickoff

Lock the work-tracking foundation for Hero so PM ships as an additive domain pack and engineering keeps doing what it's doing. **Nine canonical types using names every tool already uses** (`initiative`, `prd`, `epic`, `feature`, `bug`, `chore`, `intake`, `release`, `sprint`). Sub-typing via `kind`. Two independent adaptation layers — methodology profile (lifecycle, time-box, estimation, rituals, rollups) and vocabulary preset (display names, tracker mappings). **No migration**: existing engineering specs and folders unchanged; the registry registers what's already there plus the new PM-led and time-box types. AC infrastructure untouched. Tasks ships additively with its own package. Cross-domain handoff is an owner flip on the same artifact, not a separate spec creation.

→ Drives `spec-type-registry`, the PM pack delivery, the new `core/methodologies/` system, and Phase A of the `hero-domains` initiative.

## Goal

Give Hero a coherent work-tracking foundation that:

1. Uses **real industry names** for the work-shaped artifacts (no `spec` abstraction layer; no invented vocabulary).
2. **Adapts** to the methodologies people actually use (Scrum / Kanban / Shape Up / Waterfall / Scrumban) through structural overlays, not by forcing one process on everyone.
3. **Renames cleanly** through a vocabulary preset layer so each workspace's UI / CLI / agent output speaks the team's dialect.
4. Stays **additive** — engineering's existing 137 features / 16 bugs / 14 initiatives and the .hero/planning folder layout are unchanged. No forced migration. No alias indirection.
5. Leaves the **knowledge / meta machinery** (decisions, conventions, references, externals, notes, rules, tripwires, context) and the **AC infrastructure** alone. They work; they're out of scope for this refactor.

## Context — why now

Five observations from the design conversation:

1. Engineers in the real world mostly work off PM-shape artifacts (story / feature / card). The "engineer creates a separate feature spec from a PM story" workflow is ceremony — same content, two artifacts.
2. Hero's existing engineering types (`feature`, `bug`, `initiative`, `decision`, `convention`) and the PM pack's proposed types (`prd`, `story`, `epic`, `roadmap-item`, `intake-item`) overlap conceptually at the unit-of-work and aspirational-bet layers.
3. Methodology variation isn't just renaming. Lifecycle states, time-box shapes, estimation fields, rituals, and rollups differ structurally between Scrum / Kanban / Shape Up / Waterfall. Vocabulary alone doesn't carry that load.
4. Hero had no first-class time-box artifact (the existing single `type: plan` was the only attempt, and it's fringe). Sprint / cycle / release rituals had skills and commands but no spec to land in. Real gap.
5. Earlier passes of this design over-reached into forced migration, package renames, and meta-type bulldozing. The right scope is *additive*: register what's already there, formalize the missing time-box, ship the adaptation layers, leave the rest alone.

## Decisions

### Decision 1 — Nine canonical types, using real industry names

Hero registers the following work-tracking types via the spec-type registry. Every name is one a real tool already uses; there is **no `spec` abstraction layer**.

| Type | Role | Engineering today | PM domain |
|---|---|---|---|
| `initiative` | Top-level aspirational strategic bet; anchors PRDs; multi-quarter or cycle-spanning | 14 specs in flight | new — PM-led usage extends "any aspirational bet" |
| `prd` | Heavy authoring doc; flushed-out version of an initiative; PM-led | — | yes |
| `epic` | Mid-tier grouping; coherent bucket of features that go together | — | yes |
| `feature` | THE unit of work — user-facing capability change | 137 specs in flight | renders as "Story" / "Card" / "Scope" per vocab |
| `bug` | Defect — diagnose-fix lifecycle | 16 specs in flight | shared with engineering |
| `chore` | Operational / maintenance / sub-task — simple do-it-done lifecycle | — | new shared type |
| `intake` | Inbound funnel; promotes to initiative, epic, or feature by scope | — | PM-led |
| `release` | Large time-box; the unit that gets shipped (quarterly / monthly / PI / project) | — | new |
| `sprint` | Iteration time-box; checkpoint inside a release (Sprint / Cycle / Iteration per vocab) | — | new |

**No abstraction. No alias indirection.** Engineering's existing `type: feature` / `type: bug` / `type: initiative` frontmatter IS the canonical type. PM authors `feature` artifacts that render as "Story" through the agile-scrum vocabulary preset — same data, different display.

### Decision 2 — `kind` sub-typing per type (canonical, semantic, methodology-neutral)

`kind` is an optional first-class field on every type. Values are semantic categories — not methodology-loaded names. The vocabulary preset (Decision 4) maps canonical kinds to display names.

Canonical `kind` values:

| Type | Canonical `kind` values |
|---|---|
| `initiative` | (no kind v1; uses `horizon: now / next / later` or quarter string instead) |
| `prd` | `pitch`, `ten-section`, `lightweight` |
| `epic` | `theme`, `delivery`, `bet`, `milestone` |
| `feature` | `new`, `refactor`, `perf`, `infra`, `security`, `ux` |
| `bug` | `regression`, `edge-case`, `security`, `data` |
| `chore` | (no kind v1) |
| `intake` | `customer`, `support`, `sales`, `internal`, `competitive` |
| `release` | (no kind v1; methodology defines shape) |
| `sprint` | (no kind v1; methodology defines shape) |

`kind` is optional and back-fillable. Existing engineering specs without `kind` set continue to work; new specs can specify; lint may eventually nudge for it but doesn't require it.

### Decision 3 — `tasks` as an additive sub-element (independent of AC infrastructure)

Tasks ships as a **new, additive** structured sub-element on work-shaped types. **It does not rename, replace, or refactor the existing AC infrastructure.**

| Layer | AC (existing, unchanged) | Tasks (new, additive) |
|---|---|---|
| Section | `## Acceptance Criteria` | `## Tasks` |
| Frontmatter | id, text, status, EARS pattern | id, text, status (`todo`/`doing`/`done`), kind, assignee, optional `discovered_against: <other-spec-slug>`, timestamps |
| CLI | `hero ac list / record / status / history` | `hero task add / list / start / done / history` (new commands) |
| Backing infra | `internal/acceptance/` (unchanged) | `internal/tasks/` (new package; can share primitives via interface but **not via rename**) |

**Semantics — bar vs next thing.**

- **AC** = the bar to pass. Flips green/red on evidence (test runs). Doesn't move because a human said so.
- **Task** = the next thing to do. Flips done when someone does it. Doesn't move because a test ran.
- A task may exist *because* an AC is failing; `discovered_against` captures the lineage.

**Highest-value use case: QA-blocker capture.** `hero task add <spec-slug> "fix login redirect loop" --kind qa-blocker --discovered-against checkout-flow` writes the blocker into the spec it pertains to while preserving where it was found. Today this is freeform prose that doesn't query; tasks make it cross-corpus queryable.

**Tasks frontmatter shape (canonical):**

```markdown
## Tasks

- [ ] T-1 Fix login redirect loop {kind: qa-blocker, assignee: chet, discovered_against: checkout-flow}
- [x] T-2 Migrate token storage to keychain {kind: chore, done: 2026-05-15T14:22:00Z}
- [/] T-3 Wire up retry-with-backoff {assignee: bwheeler, started: 2026-05-16T09:00:00Z}
```

`- [ ]` = todo; `- [/]` = doing; `- [x]` = done. Inline metadata is YAML-flow shorthand; parser tolerates omission. IDs author-assigned, monotonically increasing per spec.

**AC's contract is older and load-bearing.** Tasks earns its place beside AC; AC keeps its existing infrastructure, surfaces, and behavior bit-for-bit.

### Decision 4 — Vocabulary preset system (display layer)

Display names are a workspace-level preset, decoupled from data storage **and** from methodology profile. Vocabulary handles renaming and tracker-mapping only — never structural variation.

Vocabulary preset files live under `core/vocabularies/<name>.yaml` (shared across all domains). Each file maps canonical type/kind pairs to display names plus a small set of secondary mappings.

**v1 inventory** (already authored):

| File | Speaks |
|---|---|
| `default.yaml` | Hero's existing engineering-leaning vocab (Feature / Bug / Initiative) |
| `agile-scrum.yaml` | Story / Bug / Epic / Theme |
| `shape-up.yaml` | Scope / Bug / Pitch / Bet; AC section renamed "Done line" |
| `kanban.yaml` | Card / Card / Swimlane / Lane Group |
| `jira.yaml` | Jira-shaped + `tracker_mappings.jira` |
| `linear.yaml` | Linear-shaped + `tracker_mappings.linear` |

Vocabulary flows into every user-visible surface: dashboard labels, filter pills, kanban headers; natural-language routing ("create a story" recognized in agile-scrum vocab, routes to `hero new feature`); agent output ("Drafting a Story…" under agile-scrum vs "Drafting a Scope…" under shape-up); section heading renames; CLI output; templates.

### Decision 5 — Methodology profile system (structural layer)

Distinct from vocabulary. Methodology profiles declare **structural variation** on the foundation: lifecycle state machines, time-box requirements, required/optional fields, rituals, rollups, in-flight tracking.

Methodology profile files live under `core/methodologies/<name>.yaml`. Each profile declares per spec type and per time-box:

- Lifecycle states + transitions + gates
- Time-box requirements (release: required/optional/none; sprint: required/optional/none; durations)
- Estimation field (points / appetite / dates / none)
- In-flight tracking style (burndown / hill-chart / WIP-aging / gantt)
- Cadence rituals (daily standup / weekly sync / none)
- Rollup metrics (velocity / lead-time / hill-position / phase-progress)
- Aligned vocabulary default

**v1 inventory** (to author):

| File | Methodology |
|---|---|
| `scrum.yaml` | sprint required (2wk default); release optional (quarterly/PI); story points; burndown; daily standup; sprint review + retro |
| `kanban.yaml` | no sprint; release optional; no estimation typical; WIP limits per column; lead-time / cycle-time rollups |
| `shape-up.yaml` | sprint renders as Cycle (6wk default) + Cooldown (2wk); appetite (small/big) not points; hill chart; no daily standup; betting table pre-cycle |
| `waterfall.yaml` | release required (phase-gated); no sprint; end-date estimation; phase-gate reviews; gantt rollup |
| `scrumban.yaml` | hybrid; sprint optional; release optional; team configures |

**v1 sketch — scrum profile excerpt:**

```yaml
name: scrum
display_name: "Scrum"
aligned_vocabulary: agile-scrum

lifecycle:
  feature:
    states: [backlog, ready, in_progress, in_review, done]
    transitions:
      - {from: backlog, to: ready, gate: "AC present; estimated"}
      - {from: ready, to: in_progress, gate: "claimed; sprint-committed"}
      - {from: in_progress, to: in_review, gate: "implementation done"}
      - {from: in_review, to: done, gate: "AC passing; review approved"}
  bug:
    states: [reported, diagnosing, fixing, verifying, done]
  sprint:
    states: [planning, active, completed, retrospected]

time_boxes:
  - level: release
    artifact_type: release
    duration_typical: quarter
    required: false
  - level: iteration
    artifact_type: sprint
    duration_default: 2w
    required: true
    rituals:
      - {kind: planning, when: start}
      - {kind: standup, when: daily}
      - {kind: review, when: end}
      - {kind: retro, when: end}

estimation:
  feature:
    required_field: points
    scale: [1, 2, 3, 5, 8, 13, 21]

rollups:
  - {kind: velocity, over: last_3_sprints}
  - {kind: sprint_burndown, scope: current_sprint}
```

Shape-up, kanban, waterfall, scrumban profiles follow the same shape with different values. Methodology and vocabulary compose orthogonally — a workspace can run Scrum lifecycle with kanban vocabulary if they want; most teams pick aligned pairs.

### Decision 6 — Workspace composes both layers, with overrides

`hero.json` carries both preset choices and per-key overrides:

```json
{
  "methodology": "scrum",                    // structural profile
  "vocabulary": "agile-scrum",               // display preset
  "vocabulary_overrides": {                  // per-term tweaks
    "types.initiative": "Theme"
  },
  "methodology_overrides": {                 // rare; per-axis tweaks
    "time_boxes.iteration.duration_default": "3w"
  }
}
```

**Precedence chain:**

1. Explicit `methodology:` and `vocabulary:` settings (highest priority)
2. Tracker-inferred default (Jira integration → `jira` vocabulary; if no methodology set, falls through)
3. **Methodology-implied vocabulary** — if `methodology: scrum` is set and no vocabulary is, auto-derive `vocabulary: agile-scrum`
4. `default` for both

Overrides apply on top. Override key count guards prevent dialect drift (warn above 10).

### Decision 7 — Folder layout

Per-type folders under `.hero/planning/`. Existing folders preserved exactly; new ones added for the new types.

```
.hero/planning/
  features/      ← existing engineering features stay here
  bugs/          ← existing engineering bugs stay here
  initiatives/   ← existing engineering initiatives stay here
  chores/        ← new — for chore-typed work
  intake/        ← new — for intake artifacts
  prds/          ← new — for PRD docs
  epics/         ← new — for epic groupings
  releases/      ← new — for release time-boxes
  sprints/       ← new — for sprint time-boxes
```

The registry knows the folder mapping per type. Authors can override via `location:` in the spec-type registry record if needed.

### Decision 8 — No migration

**Existing engineering specs are unchanged.** The 137 features, 16 bugs, and 14 initiatives stay where they are with their existing frontmatter. The registry registers `feature`, `bug`, and `initiative` as canonical types — those specs already speak the canonical names.

**Authors can opt into kind decoration** (e.g. `kind: refactor` on a feature) at their own pace. Lint may eventually nudge for `kind` on new specs; never forces it on existing ones.

**Aliasing is not needed.** Earlier drafts considered aliasing `type: feature` → some abstract `type: spec`; this draft abandons that abstraction. The real names are the canonical names.

### Decision 9 — Cross-domain handoff = owner flip on the same artifact

When PM hands a feature off to engineering, it's the **same artifact** transferring ownership through its lifecycle. Not two specs linked by a cross-domain edge.

- PM authors a `feature` (or `initiative`/`epic`/`prd`). Owner is `pm` during refinement.
- When `status: ready`, PM flips `owner: engineering` (via `handoff-coordinator`).
- Engineering picks it up; engineer agent loads; companion artifacts like `plan.md`, `mockups/`, `retro.md` hang off the same spec folder.
- Through lifecycle: engineering owns it through `in_progress` → `in_review` → `done`.
- The Cross-domain Handoff Stream queries the bitemporal `owner_history` rather than a dedicated `kind: handoff` edge.

One artifact per unit of work. No translation. No duplicate content.

### Decision 10 — Scope boundaries

**In scope** — work-tracking foundation and methodology adaptation as defined above.

**Out of scope** — explicitly leave alone:

- **Meta / knowledge types**: `decision`, `convention`, `plan` (legacy single-instance), `reference`, `external`, `note`, `rule`, `tripwire`, `context`. These are how Hero gets smart at feeding context to the model — separate concern, separate refactor.
- **The knowledge graph mechanics**.
- **`internal/acceptance/` package** — AC infrastructure stays put.
- **`hero ac` CLI surface** — unchanged.
- **The legacy `type: plan` single instance** (`execution-plan/spec.md`) — leave as a one-off; new release planning uses `release`.
- **Forced migration** of any kind.

## Design implications

- **`spec-type-registry`** (Go) — registers nine canonical types via markdown spec-type files at `core/spec-types/<type>.md` (cross-domain) and `domains/<active>/spec-types/<type>.md` (domain-led). Exports schema 1.1 to `.hero/cache/spec-types.json` for hero-code consumption. `kind` field is first-class. No alias layer needed.
- **`internal/vocabulary/` package** (Go) — already built; loads `core/vocabularies/*.yaml`; resolves active vocabulary via precedence chain; exposes `Display(type, kind)` / `DisplayType(type)` / `DisplaySection(canonical)` / `RecognizeNL(phrase)` / `MappedFromTracker(issueType, tracker)`.
- **`internal/methodology/` package** (Go, new) — mirrors `internal/vocabulary/`'s shape. Loads `core/methodologies/*.yaml`; resolves active methodology; exposes lifecycle / time-box / estimation / rollup accessors.
- **`internal/tasks/` package** (Go, new) — additive task parsing, graph nodes, CLI surface. Does not touch `internal/acceptance/`.
- **PM pack** — uses canonical type names (`feature` not `story`; `initiative` not `roadmap-item`; `intake` not `intake-item`). Agents like `story-writer` keep their filenames but target `feature` artifacts; vocabulary preset renders "Story" in agile-scrum contexts. Handoff workflow stays as owner-flip per Decision 9.
- **Hero-code (Rust)** consumes `.hero/cache/spec-types.json` (registry export) + `core/vocabularies/*.yaml` (display) + `core/methodologies/*.yaml` (structural). Three independent reads; all forward-looking; no migration dependency.

## Changes

| File / path | What changes |
|---|---|
| `core/spec-types/` | New: nine canonical type-record markdown files (`initiative.md`, `prd.md`, `epic.md`, `feature.md`, `bug.md`, `chore.md`, `intake.md`, `release.md`, `sprint.md`). Each declares lifecycle, kinds, sections, frontmatter shape. |
| `core/vocabularies/` | Already authored. Six v1 files unchanged. |
| `core/methodologies/` | New: five v1 profile files (`scrum.yaml`, `kanban.yaml`, `shape-up.yaml`, `waterfall.yaml`, `scrumban.yaml`). |
| `domains/pm/spec-types/` | Rename `intake-item.md` → `intake.md`. Remove `story.md`, `roadmap-item.md`, `epic.md` from this directory (moved to or already in `core/spec-types/`). `prd.md` stays as PM-led with optional PM-domain decoration. |
| `domains/engineering/spec-types/` | New directory (currently empty). Author small files for `feature.md`, `bug.md`, `initiative.md`, `decision.md`, `convention.md` declaring engineering's lifecycle and frontmatter shape. The existing Go constants `TypeFeature`/`TypeBug`/etc. continue to resolve via the same registry. |
| `internal/spec/spec.go` | Replace hardcoded `switch` statements on type literals with registry lookups. Legacy `Type*` constants retained as canonical name string aliases (no semantic change). |
| `internal/lint/`, `internal/cli/`, `internal/serve/`, `internal/tracker/sprint.go` | Replace hardcoded type/kind literals with registry lookups. |
| `internal/vocabulary/` | Already built. |
| `internal/methodology/` | New package: loader + resolver + accessors. |
| `internal/tasks/` | New package: parser, graph nodes, CLI commands. |
| `internal/cli/task.go` | New file: `hero task add / list / start / done / history` commands. |
| `internal/cli/new.go` | Update to recognize all nine canonical types. |
| `internal/config/config.go` | Already has `Domain`, `Vocabulary`, `VocabularyOverrides`, `PM` fields. Add `Methodology` field (string). |
| `.hero/cache/spec-types.json` | Generated by registry export. Schema 1.1: nine types, kind enums, frontmatter schemas, lifecycles, folder mappings. |
| `domains/pm/AGENTS.md`, `domains/pm/mission.md` | Rename references: `roadmap-item` → `initiative`; `intake-item` → `intake`; `spec` type literal → `feature`. Routing table updated. |
| `domains/pm/agents/`, `domains/pm/skills/`, `domains/pm/commands/` | Targeted edits where old type names appear as literals. Agent filenames retained where vocabulary handles the rendering (e.g. `story-writer.md` keeps its name; targets `feature`). |
| `domains/pm/spec-types/README.md`, `agents/README.md`, `commands/README.md` | Type list updates. |

## Acceptance Criteria

- THE SYSTEM SHALL load nine canonical work-tracking spec types (`initiative`, `prd`, `epic`, `feature`, `bug`, `chore`, `intake`, `release`, `sprint`) from `core/spec-types/*.md` and any active domain's `spec-types/*.md` files at process start.
- THE SYSTEM SHALL expose a `kind` field per type via the registry, with the canonical values per type listed in Decision 2.
- WHEN a spec frontmatter declares a `kind:` value not in the type's canonical kind set, THE SYSTEM SHALL emit a structural lint error listing the canonical kinds for that type.
- THE SYSTEM SHALL preserve the `TrackerType` field on every spec unchanged through this refactor.
- THE SYSTEM SHALL load methodology profile files from `core/methodologies/*.yaml` at startup and expose a `Methodology` accessor with lifecycle, time-box, estimation, and rollup queries.
- THE SYSTEM SHALL resolve the active methodology using `methodology:` in `hero.json` with `default` fallback.
- WHEN `methodology: <name>` is set and `vocabulary:` is unset, THE SYSTEM SHALL auto-derive the vocabulary from the methodology's `aligned_vocabulary` field.
- WHERE `methodology_overrides` is set in `hero.json` THE SYSTEM SHALL merge those overrides on top of the base profile at load time.
- THE SYSTEM SHALL load vocabulary preset files from `core/vocabularies/*.yaml` and `domains/<active>/vocabularies/*.yaml` and expose a `Resolve(type, kind)` accessor.
- WHERE `vocabulary_overrides` is set in `hero.json` THE SYSTEM SHALL merge those overrides on top of the base vocabulary.
- WHEN a spec lists `## Tasks` as a checklist section, THE SYSTEM SHALL parse each task into a structured `Task` record with id, text, status, kind, assignee, and optional `discovered_against` fields.
- THE SYSTEM SHALL provide `hero task add | list | start | done | history` CLI subcommands.
- THE SYSTEM SHALL back tasks via the new `internal/tasks/` package without modifying `internal/acceptance/`.
- THE SYSTEM SHALL preserve the `hero ac` CLI surface, the `Criterion` graph node kind, and AC pass-rate rollups bit-for-bit through this refactor.
- WHEN PM marks an artifact `status: ready` and flips `owner: engineering`, THE SYSTEM SHALL record the ownership transition bitemporally on the spec node so the Cross-domain Handoff Stream can query it.
- THE SYSTEM SHALL NOT create a separate engineering spec when PM hands an artifact off to engineering.
- WHEN the active vocabulary is `agile-scrum`, THE SYSTEM SHALL render a `feature` as "Story" in every user-visible surface (CLI list, dashboard, agent output, NEXT.md, generated headings).
- WHEN a tracker importer encounters a Jira `Epic`, THE SYSTEM SHALL land it as `type: epic` in the workspace (not silently collapse to `initiative` or `feature`).
- THE SYSTEM SHALL emit `spec-types.json` with `schema_version: 1.1` including the canonical type list, kind enums, frontmatter schemas, folder mappings, and lifecycles.
- THE SYSTEM SHALL NOT modify the meta / knowledge types (`decision`, `convention`, `plan`, `reference`, `external`, `note`, `rule`, `tripwire`, `context`).

## Spread plan

| Phase | What | Sized |
|---|---|---|
| 0 | Edit spec-type-registry design + hero-domains kickoff to match this spec | half-day |
| A | Finish `domain-plugin-architecture` cutover (add `domains/pm/` embed, finish ContentFS migration) | 1 day |
| B | Author `core/spec-types/` nine canonical type files + `domains/engineering/spec-types/` reference records | 1 day content |
| C | Author `core/methodologies/` five v1 profile files | 1 day content |
| D | `spec-type-registry` Go impl — markdown loader, kind field, lifecycle parser, schema 1.1 JSON export | 3-4 days Go |
| E | `internal/methodology/` Go package — loader, precedence chain, accessor | 2-3 days Go |
| F | `internal/tasks/` Go package + `hero task` CLI surface | 2-3 days Go |
| G | Inline-propose Go side delivery (already designed) | ~1 week Go, parallel |
| H | PM pack content alignment — rename references, update routing, light edits | half-day content |
| I | Vocabulary + methodology-aware rendering spread (CLI / MCP / NEXT.md) | 3-5 days Go |
| J | Hero-code (Rust) consumes `.hero/cache/spec-types.json` + vocab + methodology — Rust side | parallel, separate repo |

Phases A-C are content authoring. Phases D-I are Go. Phase G runs parallel to D-F. Phase J is cross-repo (hero-code) and runs in parallel once Phase D exports schema 1.1.

Net: **~2-3 weeks of focused Go work** to reach the point where PM ships, engineering keeps doing what it's doing, and hero-code can build UI against stable contracts.

## Cross-repo coordination

Hero-code (Rust dashboard) was peer-notified 2026-05-17 (see Handoff Trail below). Their response: zero spec-type model committed in Rust today; no conflicting work; ready to consume the new contracts. The contracts they'll read after Phase D:

1. `.hero/cache/spec-types.json` (schema 1.1) — the nine types + kinds + lifecycles
2. `core/vocabularies/*.yaml` — display names + tracker mappings
3. `core/methodologies/*.yaml` — lifecycle / time-box / estimation / rollup configuration
4. The inline-propose envelope contract (separate spec, already locked)

Three independent reads; all forward-looking; no migration dependency.

A follow-up `hero peer call hero-code --mode=advisory --related-spec unified-spec-type-model` after Phase D will hand over the finalized schema 1.1 export and the methodology profile format.

## Boundaries

- **Not** designing OKR support — defer to a future `strategy` domain or `hero-pm` v2.
- **Not** designing multi-domain coexistence concurrency. One active domain per workspace.
- **Not** allowing user-defined custom spec types beyond the nine. Sub-categorization is `kind` only.
- **Not** implementing the dashboard UI surfaces. Hero-code owns Rust dashboard rendering.
- **Not** adding new tracker integration providers. Jira / Linear / GitHub only.
- **Not** changing the on-disk markdown format beyond folder names for new types and the new `## Tasks` section. Frontmatter stays YAML; sections stay markdown; AC parsing stays the same.
- **Not** changing the AC infrastructure (`internal/acceptance/`, `hero ac` CLI, AC frontmatter, AC graph backing) in any way.
- **Not** modifying the knowledge / meta types (`decision`, `convention`, `plan`, `reference`, `external`, `context`, `note`, `rule`, `tripwire`).
- **Not** migrating existing engineering specs. They stay as authored.

## Risks

- **Vocabulary + methodology settings becoming a junk drawer.** Keep override keys narrowly scoped; validate against fixed schemas; warn when override count exceeds 10; surface the active pair prominently in `hero status`.
- **Methodology profile adoption velocity.** Five v1 profiles is ambitious if each requires deep judgment. Mitigation: ship `scrum` and `kanban` first (highest expected adoption); the other three follow as opportunity allows; per-key overrides let users compensate for under-tuned profiles.
- **Hero-code coordination drift.** Mitigated — peer call already happened; zero conflicting Rust work committed. A follow-up call after Phase D delivers the finalized schema.
- **`feature` as canonical name + `feature` as kind value confusion.** `feature.kind=new` reads cleanly; `feature.kind=refactor` reads cleanly; the type/kind distinction is structurally obvious in code (the Go type is `Type Type` + `Kind string`). Lint can warn if a non-canonical kind appears.
- **Tasks adoption pressure on AC discipline.** A tempting but wrong pattern is to use tasks where AC should be used. Mitigation: agent prompts explicitly distinguish "this should be an AC" vs "this should be a task"; lint warns when a spec has many tasks and zero AC.
- **Owner-flip workflow without a people registry.** Engineering teams without a people registry see `owner: engineering` as a flat string. Fine for v1; bitemporal history is recorded. Future per-person handoff workflows depend on a people directory.

## Handoff Trail

- 2026-05-17T14:39:46Z — out → hero-code (peer_id: ad027c2f-7f74-4a09-bf1d-6515cc906074)
  mode: advisory
  originating_spec: unified-spec-type-model
  at_commit: 19b6918
  result_ref: 18b06118ed381d48e1cf68277f6af0a2
  reason: "Cross-repo coordination after architectural pivot. Hero (Go) just locked a binding decision spec that significantly reshapes the type system before PM and QA UI work kicks off in hero-code. Sharing the decision summary so Rust implementation aligns with the new model from day one."
  outcome: "Acknowledged. Zero spec-type model committed in Rust today. No conflicting work. Ready to consume new contracts post-Phase-D."

  Note: subsequent design iterations (2026-05-17 evening) significantly revised the model from what was shared in this call:
  - Dropped the `spec` abstraction; canonical names are real industry words (`feature`, `bug`, `chore`, `initiative`, etc.)
  - Dropped forced migration; aliasing not needed; existing specs unchanged
  - Added the methodology profile system as a peer of vocabulary preset
  - Added `release` and `sprint` as the two time-box types
  - Kept AC infrastructure entirely unchanged (no `internal/checklists/` rename)
  A follow-up advisory peer call is scheduled for post-Phase-D when the schema 1.1 export is final.
