---
title: Spec Type Registry — Nine-Type Registry With Kind, Tasks, Owner, and Methodology Profile Hook
slug: spec-type-registry
type: feature
status: completed
priority: P0
tags: [platform, domains, spec-types, registry, refactor]
created: 2026-05-15
updated: 2026-05-17
relations:
  - target: hero-domains
    kind: parent
  - target: unified-spec-type-model
    kind: implements
  - target: hero-pm
    kind: unblocks
  - target: hero-qa
    kind: unblocks
  - target: tracker-fronting-and-local-first
    kind: implements
depends-on:
  - domain-plugin-architecture
  - unified-spec-type-model
horizon: next
smoke: deferred
completed_at: 2026-05-19T14:11:08Z
---

## Goal

Replace hardcoded spec types with a **single registry** read at process
start. The registry declares the **nine canonical work-tracking types**
locked in by `unified-spec-type-model` — using **real industry names with
no abstraction layer**:

| Tier | Types |
|---|---|
| Aspirational / strategic | `initiative` |
| Heavy authoring | `prd` |
| Mid-tier grouping | `epic` |
| Unit-of-work | `feature`, `bug`, `chore` |
| Inbound funnel | `intake` |
| Time-box | `release`, `sprint` |

Each carries a `kind` block (where applicable), a `tasks_schema`, an
`owner` field declaration, and a `location:` folder template. PM and
engineering share the same registry — they author the same `feature`,
`bug`, and `chore` artifacts. Five types are cross-domain at
`core/spec-types/`; engineering's `decision` and `convention` remain at
`domains/engineering/spec-types/` (engineering-led but
cross-domain-visible). Engineering's existing 137 features / 16 bugs /
14 initiatives are unchanged — they already speak the canonical type
names.

**Aliasing is not required.** Earlier drafts considered aliasing
`type: feature` → some abstract `type: spec`; this design abandons that
abstraction. The real names are the canonical names.

The parser, lint, status filters, importers, dashboard, and command
router all read from this single registry instead of enumerated literals.
Two **independent adaptation layers** sit beside the registry:

- **Vocabulary preset** (under `core/vocabularies/`, loaded by
  `internal/vocabulary/`) — display names + tracker mappings.
- **Methodology profile** (under `core/methodologies/`, loaded by
  `internal/methodology/`) — lifecycle state machines, time-box
  requirements, estimation field, rituals, rollups.

The registry exports its own data; vocabulary and methodology each have
their own read paths. Three orthogonal concerns, three loaders.

The registry surfaces a **stable, language-neutral schema export** that
the hero-code Rust dashboard consumes. The export bumps to schema version
`1.1` to carry `kind`, `tasks_schema`, and `owner` declarations alongside
the v1 shape.

**`internal/acceptance/` is unchanged.** The AC infrastructure stays
exactly as it is. Tasks ship via a new additive `internal/tasks/` package
that does not rename, replace, or refactor AC. AC's contract with the rest
of Hero is older and load-bearing.

## Audit findings

Grepped `internal/` for spec-type string literals (`"feature"`, `"bug"`,
`"convention"`, `"decision"`, `"initiative"`) and `Type*` constant
references. **103 hardcoded literal references across 25 files plus
folder path constants.** The audit from the prior pass remains valid —
the type literals are still hardcoded; the change introduced by
`unified-spec-type-model` is *what they get replaced by*. Each touchpoint
below now resolves to a registry lookup keyed on the unified six-type
set, with `kind` and `tasks` parsed from the same record.

### 1. Type/status constants and helpers (canonical source)

- `internal/spec/spec.go:15-26` — declares `TypeFeature`, `TypeBug`,
  `TypeConvention`, `TypeDecision`, `TypeInitiative`, `TypeRule`,
  `TypeExternal`, `TypeContext`, `TypeNote`, `TypeTripwire`. Under the
  unified model, the four engineering work-type constants become
  deprecated aliases that resolve to `(type: spec, kind: feature|bug)` or
  `type: epic | roadmap-item` as the migration script (Decision 7) maps
  them. `TypeConvention` and `TypeDecision` survive as-is (they remain
  registry types). The remaining knowledge constants (`TypeRule`, etc.)
  are not part of the unification and stay unchanged.
- `internal/spec/spec.go:31-67` — status enum constants stay structurally
  but are now keyed off the registry's per-type `lifecycle.states` map.
- `internal/spec/spec.go:705-734` `typeFromPath` — folder→type fallback.
  Replaced by registry `LookupFolder()` over the per-type `location:`
  declaration.
- `internal/spec/spec.go:736-762` `statusFromPath` — folder→status
  fallback. Same.
- `internal/spec/spec.go:1027-1038` `IsWorkSpec`, `IsKnowledge` —
  hardcoded membership tests. Replaced by registry `record.Category`.
- `internal/spec/graph_ingest.go:168-176` — switch on `TypeFeature/Bug/
  Convention/Decision` for graph node-kind emission. All work specs now
  emit as kind `Spec` with a sub-kind facet from the `kind:` field.

### 2. Lint / structural validation

- `internal/triage/structural.go:13-23` `validTypes` map — enumerates
  engineering types today. Becomes `registry.Lookup(s.Type) != nil`.
- `internal/triage/structural.go:25-45` — three status enum maps. Become
  per-type lookups on `record.Lifecycle.States`.
- `internal/triage/structural.go:67-110` `ValidateStructure` — switch on
  type to pick the right status set. Becomes a single registry-keyed
  lookup. **New:** also validates the spec's `kind:` value against the
  registry's per-type `kinds.values` list, and validates each `## Tasks`
  checklist entry against the `tasks_schema`.

### 3. CLI: scaffolding, listing, importing

- `internal/cli/new.go:42` — `--type` flag default+description. Add
  `--kind` flag; both read enums from registry.
- `internal/cli/new.go:130-160, 159-281, 485-609, 750-769` — six
  `switch specType` blocks. Collapse to one table-driven lookup keyed
  on `(type, kind)`.
- `internal/cli/templates.go:71` — type iteration replaced by
  `registry.All()`.
- `internal/cli/sync_import.go:127, 305-345, 503-506, 833-836, 1243` —
  Jira/Linear/GitHub issue-type → spec-type mapping. **Decision 8 shift:**
  the registry no longer carries `import_aliases` directly. Importers
  consult the **active vocabulary's `tracker_mappings` block** (loaded
  separately by `internal/vocabulary/`). The registry exposes the
  canonical six-type set the vocabulary maps onto; the alias
  table itself lives in the vocabulary file.
- `internal/cli/sprint.go:144` — sprint generator hardcodes
  `specType = "feature"`. Replaced by
  `registry.DefaultWorkType().Name` which under the unified model
  resolves to `spec`. The default `kind` for created specs is the
  registry's per-type default kind (typically `feature` for `spec`).
- `internal/cli/do.go:47-159` — natural-language routing table. Keep
  routing vocabulary; the routing layer composes registry types and the
  active vocabulary's display terms.
- `internal/cli/ac.go:114` — `--feature` flag. Out of scope for this
  spec; touched by the `internal/checklists/` rename (Decision 3 in the
  parent spec).
- `internal/cli/dashboard.go:92-93` — knowledge type sections become
  `registry.KnowledgeTypes()`. New: a tasks-by-kind summary alongside
  the AC pass-rate summary.
- `internal/cli/report.go:259-266` — knowledge type color/order map
  becomes a registry lookup.
- `internal/cli/init.go:135-137` — init banner. New folder layout per
  Decision 6 in the parent spec: `planning/specs/`, `planning/epics/`,
  `planning/roadmap/`, `planning/prds/`, `planning/intake/`,
  `planning/decisions/`, `planning/conventions/`. Banner iterates the
  registry's per-type `location:` template.

### 4. Tracker importer mapping

- `internal/tracker/sprint.go:21, 651-661, 887` — `mapJiraType` switch
  on `bug/epic/story/feature request`. Becomes vocabulary-driven (the
  active vocabulary's `tracker_mappings.jira` block returns
  `{type, kind}` pairs). Critically, Jira `Epic` → `type: epic` (not
  `type: feature` and not `type: initiative`) — this fixes the
  long-standing audit finding.
- `internal/peering/handoff.go:130, 312-316` — cross-repo handoff target
  type. Defaults to the receiving pack's default work type (`spec`).

### 5. Knowledge / context / retrieval surfaces

Six locations in `internal/context/format.go` plus
`internal/context/truncate.go`, `internal/refs/refs.go`,
`internal/retrieval/retrieval.go`, `internal/feed/feed.go`,
`internal/impact/impact.go`, `internal/extract/decisions.go`,
`internal/cost/calibration.go`, `internal/scan/generate.go`,
`internal/scan/import.go`, `internal/serve/mcp_tools.go`,
`internal/config/config.go`. Each switches from a literal-driven
membership test to a `registry.KnowledgeTypes()` / `WorkTypes()` /
`Lookup()` call. No structural change from the prior audit; the targets
are simply the unified six-type set.

### 6. Folder paths (separate from type strings)

Hardcoded `planning/{features,bugs,initiatives}/` and
`knowledge/{conventions,decisions,…}/` references collapse to
`record.Location` lookups. The folder layout itself changes per
Decision 6 in the parent spec — `features/` and `bugs/` merge into
`specs/`, `initiatives/` splits between `epics/` and `roadmap/`,
`decisions/` and `conventions/` move from `knowledge/` to `planning/`
(active) and `specs/` (archived).

### 7. Embed.FS and content layout

- `content.go:16-36` — three embed.FS variables. Under the unified model
  the loader reads from **two** sources composed at startup:
  - `core/spec-types/` — the shared five (`spec`, `epic`,
    `roadmap-item`, `prd`, `intake-item`).
  - `domains/<active-domain>/spec-types/` — the active domain's
    extensions (engineering ships `decision.md` and `convention.md`
    here; PM contributes nothing new beyond what core declares since
    the previously-PM-specific types are now in core).
- A new `coreContent` embed.FS variable joins the existing three.

### 8. Markdown content (commands, agents, skills)

Only one match in committed content
(`domains/pm/agents/story-writer.md:152`). Slash commands and skills
reference types semantically — the registry refactor touches Go code,
not content packs. The PM-pack revision (`hero-pm`, in parallel)
collapses `story.md` into the core `spec.md` declaration and renames
the writer agent.

**Risk surfaced by audit (still real).** The `Type*` constants
(`TypeFeature` etc.) are imported by 14+ packages as compile-time
values. Removing them outright is a breaking refactor. The design
keeps them as **legacy aliases** that resolve to the canonical
registry entries at init time, so existing call sites compile until
they're migrated one package at a time.

## Design

### 1. Registry mechanism — markdown frontmatter is the source of truth

**Decision: markdown files in `core/spec-types/<type>.md` (for the
shared five) and `domains/<name>/spec-types/<type>.md` (for
domain-led types like engineering's `decision.md` and
`convention.md`) are the canonical declaration. Go code reads them
via a typed loader; Go constants for the legacy four engineering
types become legacy aliases.**

Rationale (unchanged from prior pass):

- Markdown-with-frontmatter is the format Hero already uses for every
  other declaration (skills, agents, commands, conventions). The
  registry should not introduce a new file shape.
- The author-facing prose section of each file documents the type for
  humans; the frontmatter is consumed by the loader. One file, two
  audiences.
- Bundling: `embed.FS` per scope (`coreContent`, `engineeringContent`,
  `pmContent`, `salesContent`) — the loader composes core + active
  domain at startup.
- Reload: not a v1 requirement. The registry loads once at process
  start.

**Composition.** The loader reads `core/spec-types/*.md` first; then
overlays `domains/<active>/spec-types/*.md`. A domain may **extend**
core (add `decision`, `convention`) but may not **redefine** a core
type — collision is a startup error. This is what lets engineering's
`decision` and `convention` ship as engineering-led types without PM
needing to know about them, while keeping the shared five truly
shared.

**Rejected (still): pure-Go registry, JSON manifest as authoring
format.** Same reasoning as the prior pass.

### 2. Frontmatter schema language — JSON-Schema-Lite encoded in YAML, extended for kind, tasks, and owner

**Decision: the small fixed-shape schema language stays. It is extended
to include `kind:`, `tasks_schema:`, and an `owner:` declaration.**

The base schema language (supported `type` values: `string`, `int`,
`bool`, `date`, `duration`, `enum`, `list[T]`, `ref(<type>)`; per-field
`classification: content | org-state`) is unchanged.

Three additions land in v1.1 of the schema:

**(a) `kind:` block per type.** Optional first-class enum on each type's registry record. Canonical kinds per type:

| Type | `kind:` values |
|---|---|
| `initiative` | (none in v1; uses `horizon: now / next / later` or quarter string) |
| `prd` | `pitch`, `ten-section`, `lightweight` |
| `epic` | `theme`, `delivery`, `bet`, `milestone` |
| `feature` | `new`, `refactor`, `perf`, `infra`, `security`, `ux` |
| `bug` | `regression`, `edge-case`, `security`, `data` |
| `chore` | (none in v1) |
| `intake` | `customer`, `support`, `sales`, `internal`, `competitive` |
| `release` | (none in v1; methodology profile defines shape) |
| `sprint` | (none in v1; methodology profile defines shape) |
| `decision`, `convention` | (none in v1; use `tags:` for sub-categorization) |

Example:

```yaml
# core/spec-types/feature.md (frontmatter excerpt)
kind:
  values: [new, refactor, perf, infra, security, ux]
  default: new
  required: false
  description: "Sub-category for feature work; methodology-neutral."
```

CLI surfaces (`--kind` filter on `hero list`, `hero new --kind=<value>`,
dashboard kind chips) all read this block. A spec frontmatter declaring
`kind:` not in `values` produces a structural lint error. `kind` is
optional and back-fillable; existing engineering specs without `kind` set
continue to work unchanged.

**(b) `tasks_schema:` block per type.** Mirrors `ac_schema:` exactly:

```yaml
tasks_schema:
  required: false              # whether ## Tasks must appear at all
  section_heading: "Tasks"     # vocabulary may override at display time
  item_shape:
    id:                  { type: string, required: true, format: "T-<int>" }
    text:                { type: string, required: true }
    status:              { type: enum, values: [todo, doing, done], default: todo }
    kind:                { type: enum, values: [feature, bug, chore, refactor, qa-blocker, perf, infra, security, ux], required: false }
    assignee:            { type: string, required: false }
    discovered_against:  { type: ref(spec), required: false }
    started:             { type: date, required: false }
    done:                { type: date, required: false }
  history: bitemporal          # status flips tracked in graph
```

The parser reads `## Tasks` as a markdown checklist (per the canonical
form in `unified-spec-type-model` Decision 3 — `- [ ]`/`- [/]`/`- [x]`
with inline `{kind, assignee, discovered_against, started, done}`
metadata). Tasks are backed by a new **additive `internal/tasks/`
package**.

**Critical: AC infrastructure is unchanged.** `internal/acceptance/`
keeps its current shape, surfaces, and behavior bit-for-bit. There is no
rename to `internal/checklists/`, no parameterization-via-NodeKind that
touches AC, and no `TestACParity` gate. Tasks ships its own package
beside AC; the two may share bitemporal-row primitives via an interface
layer but the AC package stays put. AC's contract with the rest of Hero
is older and load-bearing; tasks earns its place beside AC, not by
refactoring it.

**(c) `owner:` field declaration.** First-class registry-level field
type:

```yaml
owner:
  type: enum
  values: [pm, engineering, qa, devops, design, docs]
  default: <per-domain>        # engineering domain defaults to "engineering"
  classification: org-state
  description: "Active owner of this artifact. Distinct from claimed_by."
  lifecycle_trigger:
    - { transition: "ready → delivering", action: "flip owner to engineering" }
```

`owner` is distinct from `claimed_by` (session-presence marker, set by
`hero claim`). `owner` is the persistent organizational-state field —
who *owns* this artifact at this moment, regardless of who has a
session open against it. The bitemporal history of `owner` flips is
recorded in the graph; the Cross-domain Handoff stream queries
`owner_history` rather than a dedicated handoff edge (per Decision 9
in the parent spec).

The registry declares the owner field shape and the lifecycle trigger
on `ready → delivering`; the **actual flip workflow** (pre-flight
checks, agent loading, plan.md authoring) lives in the
`handoff-coordinator` agent, not in the registry. The registry's job
is to declare that the field exists, what values it accepts, and which
lifecycle transition fires the flip.

**Export.** The loader compiles all of the above into an in-memory
record and writes a **JSON manifest** (`.hero/cache/spec-types.json`)
at schema version `1.1`. Hero-code reads it.

### 3. Coexistence — no migration; existing specs unchanged

**Decision: registry rework is purely additive. Existing engineering
specs and folders stay exactly as they are.** The canonical type names
(`feature`, `bug`, `initiative`) match what the 167 existing engineering
specs already declare. The registry registers what's there plus the new
PM-led and time-box types.

Sequencing (this is the binding order for the implementation PR
sequence):

1. **Author registry records.** Land the nine canonical type files at
   `core/spec-types/{initiative,prd,epic,feature,bug,chore,intake,release,sprint}.md`.
   Author `domains/engineering/spec-types/{decision,convention}.md` for
   engineering-led knowledge types (these already exist in the corpus;
   the registry just adds explicit records).
2. **Land registry loader + JSON export under schema 1.1.** Loader reads
   markdown frontmatter from `core/spec-types/*.md` and active
   `domains/<x>/spec-types/*.md`. Registry-driven validators coexist with
   legacy Go constants via `TestLintParity` (see §4) until call-site
   migration completes.
3. **Migrate call sites incrementally.** `internal/spec/`, `internal/lint/`,
   `internal/cli/`, `internal/serve/`, tracker importers — each PR
   replaces a hardcoded `switch` on type literals with a registry lookup.
   Order driven by audit findings; lowest-risk surfaces first.
4. **Legacy `Type*` constants stay as canonical name string aliases.**
   `TypeFeature = "feature"`, `TypeBug = "bug"`, `TypeInitiative =
   "initiative"`, etc. They remain valid Go constants pointing at the
   canonical type names; no semantic change; no deprecation needed since
   they ARE the canonical names. New code prefers registry lookups for
   forward compatibility but legacy uses keep working.
5. **No corpus rewrite. No folder rename.** The 137 features stay in
   `.hero/planning/features/`; the 16 bugs in `.hero/planning/bugs/`; the
   14 initiatives in `.hero/planning/initiatives/`. The registry's
   `location:` field on each type record maps to the existing folder
   paths. New types (`chore`, `intake`, `release`, `sprint`, `prd`,
   `epic`) get new folders under `.hero/planning/`.

This entire spec runs in one or two PRs. There is no migration script.
There is no AC infrastructure rename. There is no forced rewrite.

### 4. Parity test — one, focused on lint behavior

One parity test guards the rework:

- **`TestLintParity`**. Walks every spec in `.hero/` and `.hero-cloud/`;
  runs `ValidateStructure` against both a frozen snapshot of the legacy
  hardcoded validator and the registry-driven validator; fails on any
  divergence. Since the canonical type names match the existing
  frontmatter (`feature`, `bug`, `initiative`), parity should hold
  trivially — the test catches regressions where the registry-driven
  validator diverges from established behavior for any reason.

Must be green before the loader merges.

No `TestACParity` is needed because `internal/acceptance/` is not
touched by this rework. The AC infrastructure stays put.

### 5. Lifecycle transitions — declared, advisory, **not** enforced

Unchanged from the prior pass. Lifecycle states live in the registry
record's `lifecycle:` block; transitions are documented and consulted
by the dashboard and `hero check`. The registry does not reject
invalid jumps at write time.

**One addition under the unified model.** The `lifecycle:` block now
declares which transitions trigger an `owner:` flip:

```yaml
lifecycle:
  states: [planning, refined, ready, delivering, in-review, completed]
  initial: planning
  terminal: [completed]
  transitions:
    - { from: refined, to: ready,      gate: "pm pass with AC" }
    - { from: ready,   to: delivering, gate: "engineering pickup", owner_flip: { to: engineering } }
```

The `owner_flip:` annotation is the registry's contract that the
`handoff-coordinator` agent (and any future automation) reads to know
when to flip ownership.

### 6. Type record (Go shape)

```go
package spectypes

type Record struct {
    Name              string            // "initiative", "prd", "epic", "feature", "bug", "chore", "intake", "release", "sprint", "decision", "convention"
    Title             string            // canonical display name (vocabulary may override at render time)
    Domain            string            // "core" for the shared nine; "engineering" for decision/convention
    Category          Category          // Work | Knowledge
    Location          LocationTemplate  // ".hero/planning/features/{slug}/spec.md" — existing folders preserved
    Bucket            string            // "features" | "bugs" | "initiatives" | "chores" | "intake" | "prds" | "epics" | "releases" | "sprints" | "decisions" | "conventions"
    Lifecycle         Lifecycle         // states + transitions + owner_flip annotations
    Frontmatter       FrontmatterSchema
    Kind              KindSchema        // canonical kind enum for this type (Decision 2 of parent spec)
    Owner             OwnerSchema       // org-state owner field declaration
    Sections          SectionSpec       // required / optional section headings
    ACSchema          ChecklistSchema   // acceptance criteria schema (existing)
    TasksSchema       ChecklistSchema   // tasks schema (new — Decision 3 parity)
    AcceptingCommands []string          // ["/design", "/deliver", "/diagnose"]
    DefaultAgents     map[string]string // {authoring: "spec-writer", review: "pm-reviewer", handoff: "handoff-coordinator"}
    Relations         []RelationDecl    // outgoing relation declarations
}

type KindSchema struct {
    Values   []string
    Default  string
    Required bool
    Notes    map[string]string // optional per-value description
}

type ChecklistSchema struct {
    Required       bool
    SectionHeading string
    ItemShape      map[string]FieldDecl
    History        HistoryMode // Bitemporal | None
}

type OwnerSchema struct {
    Values           []string          // ["pm", "engineering", "qa", "devops", "design", "docs"]
    Default          string            // active-domain default
    Classification   Classification    // OrgState
    LifecycleTrigger []OwnerFlipTrigger
}
```

The `Kind` block surfaces the canonical kind enum; the registry
exposes a `Kinds(typeName)` accessor for CLI filter validation.

### 7. Registry interface

```go
package spectypes

type Registry interface {
    All() []*Record
    Lookup(name string) (*Record, bool)
    LookupFolder(folder string) (*Record, bool)
    LookupByKind(name, kind string) (*Record, bool) // unified-model addition
    Kinds(typeName string) []string                 // unified-model addition
    WorkTypes() []*Record
    KnowledgeTypes() []*Record
    AcceptingCommand(cmd string) []*Record
    DefaultWorkType() *Record                       // "spec"
    JSONSchema() []byte                             // language-neutral export (schema 1.1)
}
```

One registry per process. Built at startup by reading
`core/spec-types/` and `domains/<active>/spec-types/` from the
bundled `embed.FS`. The active domain comes from `hero.json` `domain`
field (default `engineering`).

The registry does **not** carry vocabulary data. A sibling resolver
(`internal/vocabulary/`) loads display names independently;
consumers needing a human-readable label compose registry data with
vocabulary at render time.

### 8. Parser refactor

`internal/spec/spec.go`:

- `typeFromPath`, `statusFromPath` consult the registry's
  `LookupFolder` instead of hardcoded prefixes.
- `IsWorkSpec`/`IsKnowledge` consult `record.Category`.
- New `kind` field on `Spec` struct; parsed from frontmatter and
  validated against `record.Kind.Values`.
- New `Tasks []Task` field on `Spec` struct; parsed from the
  `## Tasks` markdown section using the canonical form from
  `unified-spec-type-model` Decision 3. Task IDs (`T-N`) preserved
  across edits for edge stability.
- New `Owner string` field; parsed from frontmatter; validated
  against `record.Owner.Values`.
- `Type` and `Status` stay as `string` aliases.

### 9. Lint refactor

`internal/triage/structural.go`:

- `validTypes` becomes `registry.Lookup(s.Type) != nil`.
- Status maps become per-type lookups on `record.Lifecycle.States`.
- New: kind validation against `record.Kind.Values`.
- New: tasks schema validation against `record.TasksSchema.ItemShape`.
- Error messages enumerate the registered types and kinds at runtime.

### 10. CLI refactor

- `internal/cli/new.go` — six switches collapse to one table-driven
  scaffolder keyed on `(type, kind)`. Adds `--kind` flag.
- `internal/cli/list.go`, `internal/cli/active.go`,
  `internal/cli/blocked.go`, `internal/cli/queue.go` — all accept
  `--kind=<value>` and validate against the registry's per-type kind
  set.
- `internal/cli/templates.go` — iterates `registry.All()`.
- `internal/cli/sync_import.go` — drops the in-file `mapIssueType`;
  consults the active vocabulary's `tracker_mappings` block (not the
  registry — Decision 8 shift).
- `internal/cli/sprint.go:144` — replaces `"feature"` literal with
  `registry.DefaultWorkType().Name` (= `"spec"`); seeds the
  default kind from `record.Kind.Default`.
- `internal/cli/do.go` — keyword routing reads canonical type names
  from the registry and display terms from the active vocabulary.
- `internal/cli/dashboard.go`, `internal/cli/report.go` —
  knowledge type lists come from `registry.KnowledgeTypes()`;
  vocabulary applied at render time.

### 11. Importer refactor — tracker_mappings flow through vocabulary, not registry

Under Decision 8 of the parent spec, the registry **does not** carry
tracker_mappings directly. The active vocabulary file
(`core/vocabularies/<name>.yaml`) declares the
`tracker_mappings.jira/linear/github` blocks. Importers consult
`internal/vocabulary/`'s resolver:

```go
mapping, ok := vocab.ResolveTrackerMapping("jira", jiraIssueType)
// mapping = {Type: "spec", Kind: "feature"} for Jira "Story" under jira vocabulary
```

The registry validates that the `mapping.Type` is a real registered
type and `mapping.Kind` (when present) is in `record.Kind.Values`.
Invalid mappings in a vocabulary file are a startup error pointing at
the offending vocabulary file and the problematic Jira issue type.

This simplifies the registry's surface area meaningfully — the prior
design's per-type `import_aliases` block disappears; the vocabulary
is the single source of truth for tracker term ↔ Hero (type, kind)
correspondence.

The existing `TrackerType` field on `internal/spec/spec.go:140`
remains unchanged. `TrackerType` preserves the *tracker-side* original
type string for round-trip fidelity; the new `kind` field is the
*Hero-side* canonical categorization. Both coexist on every imported
spec.

### 12. Dashboard / serve refactor

- `internal/serve/mcp_tools.go:2622-2678` reads counts per registry
  type; surfaces a new tasks-by-status widget on each spec detail
  view alongside the existing AC widget.
- Kanban swimlanes iterate `record.Lifecycle.States`; the default
  spec view groups by `kind` chips (per Decision 11 of the parent
  spec — hero-code does the rendering; this spec ships the data).

### 13. Core reference registration

Ship core spec-type files under `core/spec-types/` (the nine canonical
work-tracking types) and `domains/engineering/spec-types/` (the
engineering-led knowledge types):

```
core/spec-types/
  initiative.md       # no kind v1; uses horizon: now/next/later or quarter
  prd.md              # kind: [pitch, ten-section, lightweight]; default ten-section
  epic.md             # kind: [theme, delivery, bet, milestone]; default theme
  feature.md          # kind: [new, refactor, perf, infra, security, ux]; default new
  bug.md              # kind: [regression, edge-case, security, data]; default regression
  chore.md            # no kind v1; simple do-it-done lifecycle
  intake.md           # kind: [customer, support, sales, internal, competitive]; default customer
  release.md          # no kind v1; methodology profile defines shape
  sprint.md           # no kind v1; methodology profile defines shape

domains/engineering/spec-types/
  decision.md         # engineering-led knowledge; no kind v1
  convention.md       # engineering-led knowledge; no kind v1
```

Each file carries the full schema declaration (frontmatter, lifecycle,
kind block, tasks_schema, owner declaration, sections, accepting
commands, default agents, relations). The canonical type names
**already match what engineering's existing 167 specs declare** — no
rename, no migration, no alias indirection.

PM's existing `domains/pm/spec-types/{intake-item.md, prd.md}` files
are renamed to use canonical names (`intake.md` stays in PM as a
PM-led decoration; `prd.md` is the same name and stays). PM's
previous `story.md`, `roadmap-item.md`, and `epic.md` files are
removed — those types live in `core/spec-types/` as shared types.

Lifecycles per type are declared in the registry file but **structural
variation across methodologies** (Scrum's `backlog → in_progress →
done` vs Shape Up's `unshaped → shaped → bet → building → cooldown →
shipped`) lives in **methodology profiles** (see §14), not in
spec-type files. The registry declares a *neutral default lifecycle*
per type; methodology profiles override.

Meta / knowledge types beyond `decision` and `convention` (`rule`,
`external`, `context`, `note`, `tripwire`, `plan`) are **out of scope**
for this registry rework. They exist in the corpus today and continue
to work via their current code paths.

### 14. Methodology profile system (peer of vocabulary)

The registry declares the **structural foundation** — what types exist,
what kinds they have, what fields they carry, what default lifecycles
they declare. **Methodology profiles** (loaded by `internal/methodology/`,
not by the registry) overlay structural variation on top:

- Lifecycle state machine per type (overrides registry default)
- Time-box requirements (which of `release` / `sprint` are required /
  optional / none for this methodology)
- Required estimation field per type (points / appetite / dates / none)
- In-flight tracking style (burndown / hill-chart / WIP-aging / gantt)
- Cadence rituals (daily standup / weekly sync / none)
- Rollup metric definitions
- Aligned vocabulary default

Methodology profile files live at `core/methodologies/<name>.yaml`.
v1 ships five profiles: `scrum.yaml`, `kanban.yaml`, `shape-up.yaml`,
`waterfall.yaml`, `scrumban.yaml`. The full profile format is
specified in `unified-spec-type-model` Decision 5; the registry's
contract with methodology profiles is:

1. **The registry's `lifecycle:` block on each type is a default.**
   Methodology profiles supply per-type lifecycle overrides via
   `methodology.lifecycle.<type>.states` and `.transitions`.
2. **Time-box artifact types (`release`, `sprint`) are always
   registered.** Their *required-ness* in a given workspace is
   declared by the active methodology profile, not the registry.
3. **The registry does not load methodology profiles.** They are a
   separate concern with their own loader and resolver.

Hero-code consumes registry data and methodology data via separate
paths — `.hero/cache/spec-types.json` (registry) and `.hero/cache/
methodology.json` (resolved methodology profile). The registry export
is methodology-neutral; the methodology export is structural overlay.

## Cross-language contract

This section is the surface hero-code (Rust dashboard) reads. Schema
version bumps to `1.1` to carry `kind`, `tasks_schema`, and `owner`.

### File path and format

- **Path:** `.hero/cache/spec-types.json` (per-workspace, regenerated
  by `hero` on every CLI invocation that loads the registry).
- **Format:** JSON. Schema version field at the top level.

### Schema (v1.1)

```json
{
  "schema_version": "1.1",
  "active_domain": "engineering",
  "generated_at": "2026-05-17T12:00:00Z",
  "types": [
    {
      "name": "feature",
      "title": "Feature",
      "domain": "core",
      "category": "work",
      "location": ".hero/planning/features/{slug}/spec.md",
      "bucket": "features",
      "lifecycle": {
        "states": ["planning", "refined", "ready", "delivering", "in-review", "completed"],
        "initial": "planning",
        "terminal": ["completed"],
        "transitions": [
          {"from": "refined", "to": "ready", "gate": "pm pass with AC"},
          {"from": "ready", "to": "delivering", "gate": "engineering pickup", "owner_flip": {"to": "engineering"}}
        ]
      },
      "kind": {
        "values": ["new", "refactor", "perf", "infra", "security", "ux"],
        "default": "new",
        "required": false
      },
      "owner": {
        "values": ["pm", "engineering", "qa", "devops", "design", "docs"],
        "default": "engineering",
        "classification": "org-state"
      },
      "frontmatter": {
        "required": [
          {"name": "title", "type": "string", "classification": "content"},
          {"name": "type", "type": "enum", "values": ["feature"], "classification": "content"},
          {"name": "status", "type": "enum", "values": ["planning", "refined", "ready", "delivering", "in-review", "completed"], "classification": "org-state"}
        ],
        "optional": [
          {"name": "kind", "type": "enum", "values": ["new", "refactor", "perf", "infra", "security", "ux"], "classification": "content"},
          {"name": "owner", "type": "enum", "values": ["pm", "engineering", "qa", "devops", "design", "docs"], "classification": "org-state"},
          {"name": "tracker_id", "type": "string", "classification": "org-state"}
        ]
      },
      "sections": {
        "required": ["Goal", "Acceptance Criteria"],
        "optional": ["Tasks", "Boundaries", "Risks"]
      },
      "ac_schema": { "...": "existing AC schema shape" },
      "tasks_schema": {
        "required": false,
        "section_heading": "Tasks",
        "item_shape": {
          "id":                 {"type": "string", "required": true, "format": "T-<int>"},
          "text":               {"type": "string", "required": true},
          "status":             {"type": "enum", "values": ["todo", "doing", "done"], "default": "todo"},
          "kind":               {"type": "enum", "values": ["feature", "bug", "chore", "refactor", "qa-blocker", "perf", "infra", "security", "ux"], "required": false},
          "assignee":           {"type": "string", "required": false},
          "discovered_against": {"type": "ref(feature)", "required": false},
          "started":            {"type": "date", "required": false},
          "done":               {"type": "date", "required": false}
        },
        "history": "bitemporal"
      },
      "accepting_commands": ["/refine", "/design", "/deliver", "/diagnose", "/handoff"],
      "default_agents": {
        "authoring": "story-writer",
        "review": "pm-reviewer",
        "delivery": "engineer"
      },
      "relations": [
        {"kind": "parent", "target_type": "epic", "cardinality": "zero-or-one"},
        {"kind": "parent", "target_type": "roadmap-item", "cardinality": "zero-or-one"}
      ]
    }
  ]
}
```

Note the deliberate absence of any `display:` block. **Display names
are not in the registry export.** Hero-code reads the active
vocabulary independently (per Decision 5 of the parent spec —
"the Rust dashboard reads vocabulary on workspace open, not from
the registry cache"). The registry's job is methodology-neutral
storage shape; vocabulary's job is human-readable rendering.

### Stability contract

- **Additive changes** (new optional fields, new kinds, new agents)
  are minor-version bumps. Consumers must ignore unknown keys.
- **Breaking changes** (removed fields, type changes, enum-value
  removals) bump `schema_version` to `2.0` and require a new export
  path (`spec-types-v2.json` shipped alongside v1.1 for one release).
- Hero-code's Rust types live in a shared `hero-spec-types` crate
  that derives from this schema; the crate's serde compatibility
  test is the enforcement mechanism.

### What hero-code does with this

- Renders artifact pane frontmatter forms (required fields → form
  inputs; classification → which fields are read-only because the
  tracker owns them).
- Powers the type filter pills in spec lists; renders kind chips as
  a second-level filter under each type.
- Drives kanban column ordering per type lifecycle.
- Surfaces the tasks widget on every spec detail view, alongside the
  AC widget, both backed by the same bitemporal store.
- Validates user edits client-side against the kind enum, owner
  enum, and tasks schema before posting back.
- Reads vocabulary separately (workspace open hook) and renders
  every label through it.

## Changes

**Sequencing reminder.** The registry rework lands in lockstep with
the migration script from `unified-spec-type-model` Decision 7. The
registry cannot load the corpus until `hero migrate spec-types
--apply` has rewritten frontmatter and renamed folders. Plan delivery
PRs accordingly:

1. Registry loader + JSON export under schema 1.1 (this spec).
2. Core spec-type files authored (this spec).
3. Migration script lands (`unified-spec-type-model` delegated work).
4. `hero migrate spec-types --apply` runs against `.hero/`.
5. Registry loads against migrated corpus.
6. Parity tests (`TestLintParity`, `TestACParity`) confirm green.
7. Call-site migration package by package.
8. Legacy `Type*` constants removed in a final cleanup PR.

### Additive new files

- `internal/spectypes/registry.go` — `Registry` interface, `Record`,
  `KindSchema`, `ChecklistSchema`, `OwnerSchema`, `FieldDecl`,
  `Lifecycle`, `LocationTemplate`, `Category`, `Classification` types.
- `internal/spectypes/loader.go` — reads core + active-domain
  `spec-types/*.md`, parses frontmatter, builds in-memory records,
  validates referential integrity (every `transition.from/to` in
  `states`, every `kind` value distinct, every `ref(<type>)`
  resolves), enforces core ∩ domain collision rule.
- `internal/spectypes/export.go` — emits
  `.hero/cache/spec-types.json` at schema version 1.1 including
  `kind`, `tasks_schema`, and `owner` blocks.
- `internal/spectypes/parity_test.go` — `TestLintParity` per §4.
- `internal/spectypes/loader_test.go` — golden tests on the seven
  registered types.
- `core/spec-types/initiative.md`, `prd.md`, `epic.md`, `feature.md`,
  `bug.md`, `chore.md`, `intake.md`, `release.md`, `sprint.md` — the
  nine canonical work-tracking type declarations.
- `core/spec-types/README.md` — author-facing overview of the core
  type-set.
- `domains/engineering/spec-types/decision.md`, `convention.md` —
  engineering-led knowledge type declarations.
- `domains/engineering/spec-types/README.md`.
- `core/methodologies/scrum.yaml`, `kanban.yaml`, `shape-up.yaml`,
  `waterfall.yaml`, `scrumban.yaml` — v1 methodology profiles. (Note:
  these are loaded by `internal/methodology/` — a peer package — not
  by the registry.)

### Modified existing files

- `content.go` — add `coreContent` embed.FS for `core/spec-types/`
  and `core/vocabularies/`; extend `engineeringContent` to include
  `domains/engineering/spec-types/`; add a generic
  `DomainSpecTypesFS(domain)` accessor; add a parallel
  `CoreSpecTypesFS()` accessor.
- `internal/spec/spec.go` — deprecate `Type*` constants to aliases
  resolving via the registry; add `Kind`, `Tasks []Task`, `Owner`
  fields to `Spec`; rewrite `typeFromPath`, `statusFromPath`,
  `IsWorkSpec`, `IsKnowledge` to consult the registry. Replace
  `graph_ingest.go:168-176` switch with registry category lookup;
  add a sub-kind facet on emitted spec nodes.
- `internal/triage/structural.go` — gut and rewrite against the
  registry; validates kind enum and tasks schema in addition to
  type/status.
- `internal/tasks/` (new package) — parser for `## Tasks` markdown
  checklist; graph nodes for `Task`; CLI handlers. **Does not touch
  `internal/acceptance/`.** AC infrastructure stays bit-for-bit
  unchanged.
- `internal/cli/new.go` — collapse switches to one registry-driven
  scaffolder; add `--kind` flag.
- `internal/cli/templates.go` — iterate `registry.All()`.
- `internal/cli/sync_import.go` — drop `mapIssueType`; consult
  vocabulary's `tracker_mappings`. Validate vocabulary mappings
  against registry on startup.
- `internal/cli/sprint.go` — replace `"feature"` literal with
  `registry.DefaultWorkType()` (= `feature`).
- `internal/cli/do.go` — augment keyword table from registry +
  active vocabulary.
- `internal/cli/list.go`, `active.go`, `blocked.go`, `queue.go` —
  accept `--kind=<value>`; validate against registry.
- `internal/cli/dashboard.go`, `internal/cli/report.go`,
  `internal/cli/init.go` — iterate registry instead of literals;
  init banner enumerates the unified folder layout.
- `internal/cli/task.go` (new) — mirrors `internal/cli/ac.go`:
  `add`, `list`, `start`, `done`, `history`, `status` subcommands.
- `internal/serve/mcp_tools.go` — iterate registry types for surface
  counts; iterate `record.Lifecycle.States` for kanban swimlanes;
  expose tasks widget MCP tool.
- `internal/tracker/sprint.go` — `mapJiraType` reads from
  vocabulary's `tracker_mappings.jira` block (not from registry).
- `internal/peering/handoff.go` — handoff target type defaults to
  `registry.DefaultWorkType()`.
- `internal/context/format.go`, `internal/context/truncate.go`,
  `internal/refs/refs.go`, `internal/retrieval/retrieval.go`,
  `internal/feed/feed.go`, `internal/impact/impact.go`,
  `internal/scan/generate.go`, `internal/scan/import.go`,
  `internal/cost/calibration.go`, `internal/config/config.go` —
  consult `registry.KnowledgeTypes()`/`WorkTypes()` instead of
  enumerating literals.

### Out of this spec (deferred or owned elsewhere)

- The vocabulary preset system (`internal/vocabulary/`, the six v1
  vocabulary files, `--vocabulary` CLI plumbing) lives in
  `unified-spec-type-model`'s spread plan, not here. This spec
  consumes vocabulary via the resolver interface but does not author
  it.
- The `hero migrate spec-types` script is delegated to
  `migration-engineer` under `unified-spec-type-model`. This spec
  declares the migration is a prerequisite and the migrated corpus
  is what the registry loads against.
- The `handoff-coordinator` agent rewrite (owner-flip workflow,
  pre-flight checks, hand-back path) lives in the PM-pack revision
  (`hero-pm`). This spec declares only the registry-level `owner:`
  field and the `lifecycle.transitions[].owner_flip` annotation
  that the agent reads.
- Per-spec inline custom types — boundaries section forbids.
- Removing the `Type*` Go constants — follow-up cleanup PR after
  every call site has migrated.

## Acceptance Criteria

- THE SYSTEM SHALL load all spec-type declarations from
  `core/spec-types/*.md` and `domains/<active-domain>/spec-types/*.md`
  at process start, composing core types with active-domain types.
  Active domain comes from `hero.json` `domain` field (default
  `engineering`).
- THE SYSTEM SHALL fail process startup with a clear error if a
  domain pack attempts to redefine a core type name.
- THE SYSTEM SHALL expose a `Registry` interface (per §7) with
  lookups by name, folder, command, kind, and category.
- THE SYSTEM SHALL load exactly seven canonical types: five shared
  work types from core (`spec`, `epic`, `roadmap-item`, `prd`,
  `intake-item`) and two engineering-led knowledge types
  (`decision`, `convention`), plus the five unchanged
  knowledge-corpus types (`rule`, `external`, `context`, `note`,
  `tripwire`).
- THE SYSTEM SHALL expose a `kind` block per work-type record with
  canonical values per the parent spec's Decision 2 (e.g.
  `spec.kind ∈ {feature, bug, chore, refactor, perf, infra,
  security, ux}`).
- WHEN a spec frontmatter declares a `kind:` value not in the type's
  canonical kind set, THE SYSTEM SHALL emit a structural lint error
  naming the invalid kind and listing canonical kinds for that type.
- THE SYSTEM SHALL expose a `tasks_schema` block per work-type
  record matching the canonical shape (id, text, status, kind,
  assignee, discovered_against, started, done; bitemporal history).
- THE SYSTEM SHALL parse `## Tasks` checklist sections per the
  canonical form (`- [ ]` / `- [/]` / `- [x]` with inline
  `{kind, assignee, discovered_against, started, done}` metadata)
  and validate each item against the registered tasks_schema.
- THE SYSTEM SHALL expose an `owner` field declaration per work-type
  record with canonical values (`pm`, `engineering`, `qa`, `devops`,
  `design`, `docs`).
- THE SYSTEM SHALL declare via the lifecycle `owner_flip`
  annotation that the `ready → delivering` transition triggers an
  owner flip to `engineering` (or the registered destination owner).
- THE SYSTEM SHALL write `.hero/cache/spec-types.json` at
  `schema_version: 1.1` including kind, tasks_schema, and owner
  blocks per type, on every CLI invocation that loads the registry.
- THE SYSTEM SHALL NOT include a `display:` block or any vocabulary
  resolution in the JSON export. Display names are the vocabulary
  layer's responsibility.
- WHEN the registry-driven validator runs after `hero migrate
  spec-types --apply` has rewritten the corpus, THE SYSTEM SHALL
  produce identical lint output to the pre-migration validator
  baseline across every spec under `.hero/` and `.hero-cloud/`
  (`TestLintParity`).
- WHEN the `internal/acceptance/` package is renamed to
  `internal/checklists/` and parameterized for AC and tasks, THE
  SYSTEM SHALL produce bit-for-bit identical AC behavior across
  every AC CLI operation (`TestACParity`). Failure blocks the
  rename.
- WHEN a spec frontmatter declares `type:` that is not in the
  registry, THE SYSTEM SHALL emit a structural lint error naming
  the invalid type and listing all registered types from the
  composed core+domain registry.
- WHEN a spec frontmatter declares `status:` that is not in its
  type's declared lifecycle, THE SYSTEM SHALL emit a structural
  lint error naming the invalid status and listing the declared
  states.
- WHEN an importer reads a tracker issue type, THE SYSTEM SHALL
  consult the active vocabulary's `tracker_mappings` block (not the
  registry) and land a spec at the mapped `(type, kind)` pair.
- THE SYSTEM SHALL validate every vocabulary `tracker_mappings`
  entry against the registry at startup, failing with a clear error
  if a mapping references an unknown type or invalid kind.
- WHEN `hero new --type spec --kind bug` is run, THE SYSTEM SHALL
  scaffold the spec at the registry-declared location
  (`.hero/planning/specs/{slug}/spec.md`) using the registry-declared
  sections.
- WHERE a frontmatter field is `classification: org-state`, THE
  SYSTEM SHALL expose that classification through the JSON export
  so the integration layer can apply the tracker-fronting conflict
  policy.
- WHEN a status transition occurs that is not declared in the type's
  lifecycle, THE SYSTEM SHALL emit a lint warning (not error)
  naming the unexpected transition.
- IF the registry fails to load (invalid YAML, broken ref, missing
  core or domain), THEN THE SYSTEM SHALL fail process startup with a
  clear error pointing at the offending file and line.
- THE SYSTEM SHALL retain `TypeFeature`/`TypeBug`/`TypeConvention`/
  `TypeDecision`/`TypeInitiative` as deprecated Go aliases until all
  internal call sites are migrated; final removal is a follow-up
  cleanup PR.

## Boundaries

- **Not** authoring the vocabulary preset system. That's
  `unified-spec-type-model`'s spread plan (`internal/vocabulary/`,
  six v1 vocabulary files, override merging). This spec consumes
  vocabulary via a resolver interface and validates the resolver's
  `tracker_mappings` at startup.
- **Not** writing the `hero migrate spec-types` script. Delegated
  to `migration-engineer` under `unified-spec-type-model`. This
  spec declares the script is a sequencing prerequisite.
- **Not** implementing the handoff workflow. The registry declares
  `owner` and the lifecycle `owner_flip` annotation; the agent
  workflow (pre-flight checks, plan.md authoring, hand-back path)
  lives in `handoff-coordinator` under the PM-pack revision.
- **Not** changing the on-disk spec format beyond what the parent
  spec (`unified-spec-type-model`) already locked: folder renames,
  the new `## Tasks` section, the `kind:` and `owner:` frontmatter
  fields. Markdown structure otherwise unchanged.
- **Not** introducing per-spec inline custom types. Types are a
  domain-pack concern.
- **Not** addressing graph node namespacing — that's
  `domain-scoped-knowledge-graph` (primitive #6).
- **Not** enforcing lifecycle transitions — they are declared and
  advisory (see §5).
- **Not** implementing runtime registry reload. Loads once at
  startup.
- **Not** supporting multiple active domains in one workspace v1.
- **Not** designing the dashboard form-rendering logic. Hero-code's
  surface.
- **Not** removing `Type*` Go constants in the same PR. Final
  cleanup PR after migration.

## Risks

- **Sequencing dependency on the migration script.** The registry
  cannot load until the corpus is migrated. If the migration script
  slips or breaks, the registry sits unused. Mitigation: the
  migration is the very next delegated piece of work after this
  spec lands; pair the loader PR with a dry-run of the migration on
  Hero's own `.hero/` and gate merge on a clean dry-run plus a
  successful apply on a copy.
- **TestLintParity across a migrated corpus is subtler than across
  an unmigrated one.** The legacy validator snapshot must run
  against the *unmigrated* frontmatter, and the registry-driven
  validator against the *migrated* frontmatter, and they must
  agree on the structural verdict. Mitigation: bake the mapping
  function (legacy `feature` → registry `spec, kind: feature`,
  etc.) into the test harness as a single canonical translator; the
  test asserts agreement on the verdict, not on raw type strings.
- **TestACParity is non-negotiable.** If the
  `internal/acceptance/` → `internal/checklists/` rename even
  slightly perturbs AC behavior, downstream consumers (dashboard
  pass-rate rollups, drift detection, the spec-corpus history view)
  break silently. Mitigation: write `TestACParity` *first*, before
  touching the package; it must be green on every commit of the
  rename PR.
- **Vocabulary validation at registry startup couples two
  subsystems.** If vocabulary loads first and contains a
  tracker_mapping referencing an unregistered type, startup fails.
  Mitigation: load vocabulary lazily — vocabulary validation runs
  only when an importer first asks for a mapping, with a clear
  error pointing at the offending vocabulary file. Registry startup
  does not depend on vocabulary integrity for non-import workflows.
- **`Type*` alias removal is its own delivery hazard.** Fourteen
  packages import the constants. The migration PR sequence treats
  removal as a final cleanup, but if deferred indefinitely the
  codebase carries permanent dead code. Mitigation: add a
  follow-up spec stub when this lands; tie its delivery to a
  release boundary.
- **Audit surface area still real.** 103 literal references across
  25 files. Embedded paths in comments, test fixtures, and `.hero/`
  content reference the old layout. Budget for a few more call
  sites surfacing mid-delivery.
- **Cross-language schema lock-in.** Once hero-code consumes
  `spec-types.json` at schema 1.1, the shape is contract. The
  `kind`, `tasks_schema`, and `owner` blocks are new this release;
  any tweak to their shape post-merge requires a 1.2 bump or a 2.0
  break. Mitigation: review the 1.1 schema with hero-code via the
  pending `hero peer call` before merging the loader; lock the
  shape in that round-trip.
- **`feature` canonical-name collision.** `feature` is both a `kind`
  value (on `spec`) and a legacy `Type` name during the alias
  period. Risk: confusion in code reading and grep results.
  Mitigation: the deprecated aliases are clearly marked; the
  registry's Go type for kind is `Kind string` not reusing `Type`;
  lint enforces that `type: feature` (legacy) does not appear in
  new specs after migration.
- **Lifecycle advisory-only is a softer guarantee than some callers
  may assume.** Authoring agents that "set status to ready" need a
  way to opt into stricter checks. Provide a
  `registry.IsValidTransition()` helper so opt-in is one call away.

## Kickoff

`/deliver spec-type-registry` — but **not before** the migration
script (delegated under `unified-spec-type-model`) is ready to
run alongside. First implementation step is the loader + JSON
export + parity tests, with engineering's reference registration
and core's shared five spec-type files authored together. Do not
modify any existing call site until both `TestLintParity` and
`TestACParity` are green. Then migrate `internal/triage/
structural.go` first (centralizes the existing behavior), then
the importer (highest external surface — and the one that
consumes the vocabulary's `tracker_mappings`), then CLI
scaffolding, then dashboard and knowledge surfaces.

The vocabulary system (loader, resolver, six v1 files,
override merging) lands in parallel under the parent spec's
spread plan. The registry validates vocabulary `tracker_mappings`
against itself at startup but otherwise treats vocabulary as an
opaque external service.

**Files:** .hero/planning/features/spec-type-registry/spec.md,
.hero/planning/features/unified-spec-type-model/spec.md,
.hero/planning/features/hero-pm/spec.md,
.hero/planning/initiatives/hero-domains/spec.md,
core/ (target for new spec-types/),
domains/pm/spec-types/ (collapsing into core),
domains/engineering/ (target for new spec-types/),
internal/spec/spec.go, internal/triage/structural.go,
internal/cli/new.go, internal/cli/sync_import.go,
internal/tracker/sprint.go, internal/serve/mcp_tools.go,
internal/acceptance/ (renaming to internal/checklists/),
content.go.

**Skip:** Authoring the vocabulary preset system (parent spec's
spread plan). Writing the migration script (delegated to
`migration-engineer`). Rewriting `handoff-coordinator` (PM-pack
revision). Removing `Type*` Go constants (follow-up cleanup PR).
Runtime registry reload. Multi-domain coexistence in one
workspace. Per-spec inline custom types.
