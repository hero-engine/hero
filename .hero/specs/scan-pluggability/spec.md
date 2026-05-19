---
title: Scan Pluggability — Per-Domain `hero scan` Implementations
slug: scan-pluggability
type: feature
status: completed
priority: P0
tags: [platform, domains, scan, ingestion, refactor]
created: 2026-05-15
designed: 2026-05-19
relations:
  - target: hero-domains
    kind: parent
depends-on:
  - domain-plugin-architecture
  - spec-type-registry
horizon: next
smoke: deferred
---

## Kickoff

Make `hero scan` domain-aware. Today scan is a code scanner — it detects
languages, frameworks, test runners, generates engineering-shaped
knowledge stubs, and ingests symbols/packages/files into the graph. PM
needs a totally different scan: import a roadmap doc, parse tracker
epics, ingest OKRs. This spec generalizes `hero scan` to dispatch to
the active pack's scanner. Engineering's code scan becomes the
reference implementation under `domains/engineering/scan/`. PM-specific
scanners (roadmap-doc parser, tracker-epic ingester, OKR ingester)
ship in `hero-pm`, not here.

**Status:** designed — 2026-05-19. Locks (a) the **uniform graph-node
schema with a `domain` tag** for scan output (defer domain-typed
specializations to the registry's `kind` facet and the DSKG namespace);
(b) the **manifest + in-tree dispatcher** shape for scanner
declaration; (c) the **fixed sub-command set with per-pack opt-in**;
(d) the **wire-through `Config.Domain` at dispatch time** rule for
stamping; (e) that `internal/scan/` becomes the dispatch shell while
the code-specific detection logic moves into
`domains/engineering/scan/` as the reference impl.

**Pick up at:** `/deliver scan-pluggability`. Land the dispatch
interface + manifest loader first (single PR), then the engineering
reference relocation (golden-output parity test gates merge), then
wire `Config.Domain` through to `WriteGraph` so codescan ingest
stamps `engineering` per the DSKG contract.

→ `/deliver scan-pluggability`

**Files:** .hero/planning/features/scan-pluggability/spec.md,
.hero/planning/initiatives/hero-domains/spec.md,
.hero/planning/features/domain-scoped-knowledge-graph/spec.md,
internal/scan/, internal/codescan/, internal/cli/scan.go,
internal/cli/init.go, internal/index/, domains/engineering/scan/,
domains/engineering/scan-manifest.yaml.

**Skip:** Designing PM scanners (roadmap-doc parser, tracker-epic
ingester, OKR ingester) — those live in `hero-pm`. Cross-domain scan
in a single workspace (multi-active-domain is deferred per DSKG).
Third-party / out-of-tree scanners loaded from disk. Changing scan
triggering (CLI command shape, post-init hook). Changing the
knowledge-graph storage backend.

## Goal

Generalize `hero scan` so the active domain pack decides what scanning
means. Engineering's current code-scan stack (language detection,
framework detection, test-runner detection, knowledge-base entry
generation, code-symbol ingestion) becomes the reference
implementation under `domains/engineering/scan/`. The CLI `hero scan`
looks up the active pack via `Config.Domain`, dispatches to its
registered scanner, and forwards the scanner's outputs (graph
nodes/edges, generated knowledge entries, structured report) through
the existing shared infrastructure (graph store, FTS5 index, merge
planner, hooks).

A v1 PM scanner — when `hero-pm` ships it — implements the same
`Scanner` interface, produces the same uniform graph-node schema with
a `pm` domain tag, and writes the same shape of generated knowledge
entries. `hero scan` doesn't know which pack it's calling; the
dispatch layer hides the difference.

## Why now

PM onboarding is fundamentally different. Running language detection
on a PM workspace is wrong — the right scan is "import the roadmap doc
you already have, parse tracker epics into stories, ingest OKRs if
they exist." Until scan is pluggable, PM onboarding either runs the
wrong scan or skips scan entirely, and both are bad first impressions.

This spec depends on:

- **`domain-plugin-architecture`** for `domains/<name>/` to exist as a
  composable surface. Engineering content already moved under
  `domains/engineering/` (agents, commands, skills, spec-types,
  AGENTS.md). The scan reference impl is the next thing that goes
  there.
- **`spec-type-registry`** because PM scanners emit type-correct
  specs (`epic`, `intake`, etc.). Without the registry, those types
  don't exist as registered node kinds, lint won't accept them, and
  the importer can't write them.

This spec also paves the way for the **DSKG write-path stamping
contract** (`internal/codescan/graph_ingest.go` always stamps
`engineering`). Today codescan doesn't stamp domain at all. After
this spec lands, `Config.Domain` threads through the dispatch into
the scanner, and codescan stamps `engineering` regardless of active
domain (because code is intrinsically engineering — DSKG §Phase 2
stamping rules).

## Design

### 1. Scan output schema — uniform node/edge types with `domain` tag

**Decision: scan output emits the existing graph node and edge types
unchanged. The `domain` namespace tag (added by
`domain-scoped-knowledge-graph` as Phase 1's schema v3 migration) is
the only addition. Domain-specific semantics ride on the spec-type
registry's `kind` facet (e.g. `Spec.kind = "epic"` vs
`Spec.kind = "feature"`) and on existing typed nodes
(`Symbol`, `Package`, `File`, `Story`, etc. as the registry adds
them). Scanners do not invent new top-level node types.**

#### Rationale

The choice was between two shapes:

| Option | Description | Pros | Cons |
|---|---|---|---|
| **A. Uniform** | Every scanner emits the same node/edge type set; domain is a tag on each row. PM scanner emits `Spec` nodes with `kind: epic` and `domain: pm`; engineering scanner emits `Spec` with `kind: feature` and `domain: engineering`. | Simple graph queries — one `WHERE n.type = 'Spec' AND n.domain = ?` works for both packs. Read paths in the DSKG audit (every query path in primitive #6) already partition by `domain`; they need zero per-domain type knowledge. Adding a future domain (QA, Ops) means **zero changes** to the audit table. | Loses some semantic precision — "a PM story IS a kind of spec" is structurally honest but reads less naturally than `Story` as a top-level type. |
| **B. Domain-typed** | Each scanner emits domain-typed nodes (`Story`, `Epic`, `PRD`, `RoadmapItem` for PM; `Feature`, `File`, `Symbol`, `Package` for engineering). Domain tag remains but is redundant with the type. | Richer, more honest — query `WHERE type = 'Story'` returns only PM stories, no domain filter needed. Type-level traversal stays sharp. | Every read path in the DSKG audit must know about every domain's node-type set. The MCP tool registry, the dashboard widget registry, `hero why`'s `originEdgeTypes`, the FTS5 boost table — all become per-domain. Adding a new pack means editing every read site. |

**Uniform wins** because of three concrete forces from the parent
initiative:

1. **The DSKG audit table is the v1 contract for D2
   (cross-domain-graph-query skill).** That table enumerates 30+
   query paths and tags each with a v1 stance. Every stance assumes
   `WHERE n.domain = ?` is sufficient to scope. Domain-typed nodes
   would require rewriting every row in the audit, plus the audit
   would need a "supported node types per domain" matrix.
2. **The `spec-type-registry` already differentiates work artifacts
   via `kind`.** A PM `story` and an engineering `feature` are both
   `type: spec` in the registry under the unified-spec-type-model
   design — they differ by `kind`, not by node type. The graph
   already mirrors the registry's category split (Work vs Knowledge)
   in `IsWorkSpec`/`IsKnowledge`. Adding a `Story` node type for PM
   would re-introduce the abstraction the registry refactor
   explicitly removed.
3. **Engineering's existing node types are already a uniform set.**
   Code intelligence today emits `Repo`, `Package`, `File`, `Symbol`
   — none of these are "engineering-typed" in the sense of being a
   subtype of a generic. They are intrinsic content types. PM scan
   doesn't need its own analogues; it ingests `Spec`-shaped work
   artifacts (epics, intake items, roadmap items, stories). Those
   already exist as `Spec`-kind nodes in the registry.

The uniform rule has one explicit exception: **intrinsic content
types stay typed**. `Package`, `File`, `Symbol` (engineering's code
intelligence node set) are code-intrinsic and only ever stamp
`engineering` per DSKG. A future Ops pack adding incident scan output
might introduce a typed `Incident` node — that is a registry-level
decision and falls outside this spec's scope.

#### Concrete node-type emission contract per scanner

| Scanner | Emits node types | Emits edge kinds |
|---|---|---|
| Engineering reference (`domains/engineering/scan/`) | `Repo`, `Package`, `File`, `Symbol`, plus `Spec` (knowledge stubs: `context`, `convention`, `rule`), `Convention`, `Rule` | `belongs_to`, `defines`, `imports`, `mentions` |
| PM (future, in `hero-pm`) | `Spec` (with `kind` ∈ `{epic, story, roadmap-item, intake, prd}`), plus existing knowledge nodes if scanner produces them | `belongs_to`, `derived_from`, `realizes` (cross-domain when story→feature) |

Both write through `internal/graph/graph.go` with `domain` set per the
DSKG stamping rules (engineering scanner always `engineering` for
code, active-domain for any work-artifact stubs it generates; PM
scanner stamps `pm` for everything it writes).

#### What this means for the DSKG spec

DSKG §Phase 2's stamping rule for `internal/codescan/graph_ingest.go`
("**Always stamps `engineering`** — code is intrinsically
engineering") becomes a property of the **engineering scanner's
codescan invocation**, not a property of `internal/codescan/`
itself. Code intelligence stays under `internal/codescan/` (it's
shared infrastructure that any future code-aware pack might reuse);
the engineering scanner is the only thing that calls it today, and
it passes `domain: "engineering"` regardless of `Config.Domain`. The
DSKG contract holds end-to-end.

### 2. Dispatch surface — manifest + in-tree dispatcher map

**Decision: each pack declares its scanner via a
`scan-manifest.yaml` file at `domains/<name>/scan-manifest.yaml`.
The manifest is a small declarative descriptor; the actual scanner
implementation is a Go function registered through an in-tree
dispatcher map (`scan.Dispatch`) keyed on the manifest's
`scanner_id`. Out-of-tree plugins are explicitly out of scope —
scanners ship with their pack, in-tree, no plugin loader, no
dynamic library shape, no `plugin.Open`.**

#### Manifest shape

```yaml
# domains/engineering/scan-manifest.yaml
scanner_id: engineering-code-scan
display_name: Engineering Code Scan
subcommands:
  - id: scan
    description: Full project scan — stack detection + knowledge generation + code intelligence
    flags:
      - { name: code, type: bool, default: false, description: "code intelligence only" }
      - { name: dry-run, type: bool, default: false, description: "preview without writing" }
      - { name: force, type: bool, default: false, description: "overwrite customized entries" }
emits:
  node_types: [Repo, Package, File, Symbol, Spec, Convention, Rule]
  edge_kinds: [belongs_to, defines, imports, mentions]
config_keys:
  - code_scan.depth
  - code_scan.parser
```

```yaml
# domains/pm/scan-manifest.yaml (future, ships in hero-pm)
scanner_id: pm-roadmap-scan
display_name: PM Roadmap Scan
subcommands:
  - id: scan
    description: Import roadmap, parse tracker epics, ingest OKRs
    flags:
      - { name: roadmap, type: string, default: "", description: "path to roadmap doc" }
      - { name: dry-run, type: bool, default: false }
emits:
  node_types: [Spec]
  edge_kinds: [belongs_to, derived_from, realizes]
```

#### Why manifest + Go dispatcher (not pure-Go and not external plugin)

- **Manifest-only (no Go registration) is too weak.** The actual
  detection logic is real Go code — language detectors, file
  walkers, framework matchers. A pure-manifest scanner would force
  every detection rule into declarative form, which the existing
  engineering scan provably doesn't fit (it embeds heuristics like
  `detectFromPackageJSON`, `detectJVMTestFrameworks`).
- **Pure-Go (no manifest) hides the scanner's contract.** Without
  a declarative descriptor, the dispatch CLI can't enumerate
  subcommands or validate `--flags` without instantiating the
  scanner. Manifest gives `hero scan --help` something to read
  without loading every pack's scanner.
- **External plugins (`plugin.Open`, separate binaries) are
  premature.** Go's plugin support is fragile (version-locked, no
  Windows support), and the parent initiative explicitly says
  "scanners ship with their pack, in-tree." The dispatcher map
  rule preserves that.

#### The dispatcher map

```go
// internal/scan/dispatch.go (new)

package scan

import (
    "github.com/hero-engine/hero/internal/config"
    "github.com/hero-engine/hero/internal/graph"
)

// Scanner is the interface a domain pack's scanner implements.
type Scanner interface {
    // ID returns the scanner_id declared in the manifest.
    ID() string

    // Scan executes the named subcommand. opts carries flag values
    // and shared context (project root, hero dir, config, graph store).
    // The scanner returns a Report; the dispatch shell prints it and
    // exits.
    Scan(subcommand string, opts ScanOpts) (*Report, error)
}

// ScanOpts is the shared context passed to every scanner invocation.
type ScanOpts struct {
    ProjectRoot string
    HeroDir     string
    Config      config.Config // includes Config.Domain — scanner uses this to stamp
    Store       *graph.Store  // shared graph store; scanner does not own it
    Index       IndexHandle   // FTS5 index handle for projection
    Flags       map[string]any // parsed flag values (typed per manifest)
    DryRun      bool
    Force       bool
    Reporter    Reporter      // progress / log sink; respects --quiet
}

// Report is what a scanner returns. The dispatch shell prints it.
type Report struct {
    Summary       string            // human-readable summary block
    EntriesPlan   []GeneratedEntry  // knowledge entries to merge (engineering)
    GraphSummary  GraphIngestSummary // node / edge counts per step
    Warnings      []string
}

// Register adds a scanner to the dispatch map. Called from each pack's
// init() — engineering's init lives in domains/engineering/scan/init.go,
// PM's would live in domains/pm/scan/init.go.
func Register(scanner Scanner) { ... }

// Dispatch picks the scanner for the active domain and runs the
// requested subcommand. Returns ErrScannerNotFound when the active
// pack ships no scanner (a valid state — some packs may opt out
// entirely; the dispatch shell prints a friendly skip message).
func Dispatch(subcommand string, opts ScanOpts) (*Report, error) { ... }
```

The dispatcher is keyed on the **active pack's scanner_id**, resolved
at startup:

1. Read `Config.Domain` (defaults to `engineering`).
2. Read `domains/<domain>/scan-manifest.yaml` from the embedded FS
   (`embed.FS`, per the embed pattern already in use by spec-types
   and vocabulary loaders).
3. Look up `manifest.scanner_id` in the dispatcher map.
4. If found, return that scanner; if not, return `ErrScannerNotFound`.

Manifest absence (no `scan-manifest.yaml` in the pack) is a clean
"this pack has no scanner" signal — `hero scan` prints `<domain> pack
does not ship a scanner; nothing to do` and exits 0. This is what
non-PM, non-engineering packs (chat, sales) get for free until they
opt in.

#### Why not auto-load on subcommand alone?

A naive alternative: `hero scan code` runs the `engineering` scanner's
`code` subcommand regardless of active domain. Rejected because it
breaks the "active pack decides what scanning means" principle — a PM
workspace running `hero scan code` would invoke engineering's code
scan against PM content, which is exactly the wrong behavior. The
active-domain gate is non-negotiable.

### 3. Sub-commands — fixed set, per-pack opt-in

**Decision: the CLI declares a fixed canonical sub-command set.
Each pack's manifest lists which sub-commands its scanner
implements. Unimplemented sub-commands return a friendly
`<domain> pack does not implement 'scan <X>'` message and exit
non-zero.**

#### Canonical sub-command set (v1)

| Sub-command | Purpose | Engineering implements? | PM implements? (future) |
|---|---|---|---|
| `hero scan` (default) | Full pack-specific scan | yes | yes |
| `hero scan --code` (engineering legacy flag) | Code intelligence only | yes (alias for a future `hero scan code`) | n/a |
| `hero scan --dry-run` | Preview without writing | universal flag, dispatch enforces dry-run before calling Scan | universal |
| `hero scan --force` | Overwrite customized entries | universal flag, dispatch passes through | universal |

For v1 the only sub-command is the default (`hero scan`), with
per-flag opt-ins. The existing `--code` flag stays as a backwards-
compatible engineering-only shortcut; the dispatch shell rejects
`--code` for any non-engineering active pack with a clear error.

A future `hero scan <subcommand>` syntax (e.g. `hero scan import`
for the PM scanner's roadmap-import-only path) lands when a pack
actually needs it — manifest declares the subcommand, dispatch
routes it. Engineering only ships the default sub-command in v1.

#### Why fixed set instead of free-form per-pack subcommands?

Free-form subcommands would let each pack invent its own
(`hero scan tracker-import`, `hero scan okrs`), which fragments the
CLI surface across packs and makes `hero scan --help` per-domain. A
fixed set with declared opt-ins keeps the CLI grammar stable across
packs and matches how `hero install` / `hero domain` already work.

### 4. Cross-domain scan in a single workspace — out of v1, dispatch shape preserves it

Per the parent initiative and the DSKG spec, single-active-domain
workspaces are the v1 boundary. `hero scan` runs exactly one
scanner per invocation: the active pack's. There is no `--all-packs`
flag and no `hero scan pm` from an engineering-active workspace.

The dispatch shape is designed so multi-active-domain v2 can land
without refactoring:

- Each scanner is independently invokable through `Dispatch`.
- The graph store is shared; scanners don't own it.
- The `Reporter` interface is per-scan; nothing in the scanner
  contract assumes "this is the only scanner running today."

A v2 `hero scan --all-domains` flag would iterate the dispatcher
map's registered scanners, calling each in sequence with the same
opts. That is a one-line CLI addition once active-domain becomes
a list. We don't ship it now.

### 5. Stamping — wire `Config.Domain` through dispatch into `WriteGraph`

**Decision: `Config.Domain` threads through `ScanOpts` into every
graph-write call the scanner makes. Each scanner implementation
decides whether to honor it (the default for most writes) or
override it (intrinsic content — engineering's codescan always
stamps `engineering`).**

#### The wire-through path

```
hero scan
  → cli/scan.go: build ScanOpts{Config: cfg, ...}
  → scan.Dispatch("scan", opts)
  → engineering scanner's Scan() method receives opts
  → engineering scanner calls codescan.WriteGraph(result, store, domain string)
                                                          ^^^^^^^^^^^^^^
                                                          NEW PARAMETER
  → codescan.WriteGraph passes domain into every UpsertNode/UpsertEdge
    EXCEPT for intrinsic code nodes which hardcode "engineering"
```

The codescan `WriteGraph` signature changes from:

```go
func WriteGraph(result *Result, store *graph.Store) (*GraphWriteSummary, error)
```

to:

```go
func WriteGraph(result *Result, store *graph.Store, domain string) (*GraphWriteSummary, error)
```

But — and this is the critical part — codescan's `WriteGraph` body
**ignores the `domain` parameter for all `Repo`/`Package`/`File`/
`Symbol` writes**. It hardcodes `"engineering"`. The parameter
exists for forward-compatibility (a future code-aware pack that
isn't engineering — e.g. a hypothetical `data-analytics` pack
scanning SQL files — would pass its own domain through). For v1,
codescan ignoring the parameter and stamping `engineering` is the
DSKG-correct behavior.

For non-code writes inside the engineering scanner (knowledge
stubs, conventions, rules), the scanner passes
`opts.Config.Domain` through, which resolves to `engineering` in
engineering-active workspaces and `pm` if a hypothetical PM session
ran the engineering scan (a non-supported state — the dispatcher
gates this — but the wire stays correct).

The PM scanner, when `hero-pm` ships it, calls into a parallel
`pm` ingest path with `domain: opts.Config.Domain` and writes
`Spec`/`Convention`/etc. nodes stamped `pm`.

#### Why route through the scanner, not through the dispatcher

The dispatcher could stamp domain on every UpsertNode call by
wrapping the store, but that would break the intrinsic-content rule
— codescan needs the freedom to override the dispatch-provided
domain for code-intrinsic writes. Routing through the scanner's own
calls keeps that decision explicit and local to the scanner.

### 6. How `domains/engineering/scan/` differs from `internal/scan/` after the refactor

**Decision: `internal/scan/` becomes the dispatch shell + shared
scan infrastructure. The code-specific detection logic
(language detection, framework detection, marker scanning,
knowledge-entry generation, import-source parsing) moves into
`domains/engineering/scan/` as a Go package that registers itself
via `init()`. `internal/codescan/` stays put — it's shared code
intelligence infrastructure that the engineering scanner invokes.**

#### After the refactor

```
internal/scan/                      # the dispatch shell + shared infra
  dispatch.go                       # NEW — Scanner interface, Register, Dispatch
  manifest.go                       # NEW — parse scan-manifest.yaml
  opts.go                           # NEW — ScanOpts, Report types
  merge.go                          # MOVED FROM here — knowledge-entry merge planner (shared)
  merge_test.go
  reporter.go                       # NEW — progress sink interface
  entry.go                          # NEW — GeneratedEntry type (shared with engineering scanner)

internal/codescan/                  # unchanged location — shared code intelligence
  scanner.go
  graph_ingest.go                   # signature gains domain string parameter
  parse_*.go
  ...

domains/engineering/scan/           # NEW — engineering reference scanner
  init.go                           # init() registers the scanner via scan.Register
  scanner.go                        # implements scan.Scanner interface
  analyze.go                        # MOVED FROM internal/scan/scan.go (detectLanguages,
                                    #   detectFromMarkers, detectCI, detectMonorepo, detectDocFiles)
  generate.go                       # MOVED FROM internal/scan/generate.go
  enrich.go                         # MOVED FROM internal/scan/enrich.go
  enrich_test.go
  import.go                         # MOVED FROM internal/scan/import.go
  import_test.go
  modules.go                        # MOVED FROM internal/scan/modules.go
  modules_test.go
  scan_test.go                      # MOVED FROM internal/scan/scan_test.go

domains/engineering/scan-manifest.yaml  # NEW — declares scanner_id, subcommands, emits
```

`internal/scan/` keeps:

- The `Scanner` interface and `Register`/`Dispatch` machinery
- `MergeDecision`/`PlanMerge`/`ExecuteMerge` — the knowledge-entry
  merge planner, which is genuinely shared infrastructure
  (any pack writing knowledge entries needs it)
- `GeneratedEntry` — the shared type for knowledge entries
- The manifest loader
- The `Reporter` interface

`internal/scan/` loses:

- `Analyze`, `detectLanguages`, `detectFromMarkers`, `detectCI`,
  etc. → move to `domains/engineering/scan/analyze.go`
- `Generate`, `GenerateRichProjectOverview`, the linter/CI/test
  convention generators → move to `domains/engineering/scan/generate.go`
- `Enrich`, `DetectImportSources`, `ParseImportSource`,
  `ClassifyImportedSections`, `ImportToEntries` → move to
  `domains/engineering/scan/{enrich.go,import.go}`
- `DetectMultiModule`, `DetectFrameworkDetails` → move to
  `domains/engineering/scan/modules.go`

#### `internal/cli/scan.go` becomes a thin shell

```go
func runScan(cmd *cobra.Command, args []string) error {
    projectRoot := findProjectRoot()
    cfg, err := config.Load(projectRoot)
    ...
    store, err := graph.Open(heroDir)
    ...
    opts := scan.ScanOpts{
        ProjectRoot: projectRoot,
        HeroDir:     heroDir,
        Config:      cfg,
        Store:       store,
        Flags:       parseFlags(cmd, args),
        DryRun:      scanDryRun,
        Force:       scanForce,
        Reporter:    scan.StdoutReporter(),
    }
    report, err := scan.Dispatch("scan", opts)
    if err != nil { return err }
    return report.Print()
}
```

The bulk of the current `cli/scan.go` (the entry-merge loop, the
ingest summary printing, the sibling-subgraph ingest, the work-
subgraph ingest, the tracker pull, the team-server sync) is
**shared infrastructure** that every scanner needs. It moves into
`internal/scan/postwork.go` (sibling-subgraph, work-subgraph,
tracker pull, team-server sync are all pack-agnostic) and runs
**after** the active scanner's `Scan()` call returns. The scanner
returns only the pack-specific outputs (the result of `Analyze` for
engineering); the dispatch shell runs the cross-cutting graph
ingestion that follows.

This split keeps PM's future scanner small — it doesn't have to
re-implement sibling-subgraph ingest or tracker pull; it inherits
them from the dispatch shell.

### 7. Discovery of pack-shipped scanner config

**Decision: pack-shipped scanner config lives alongside the
scanner code at `domains/<name>/scan/`. Detector dictionaries
(language extensions, framework markers) stay inline in Go code as
they are today — they are not user-tunable. The manifest declares
which `config_keys` from `hero.json` the scanner consumes.**

#### Concrete layout (engineering)

- `domains/engineering/scan/analyze.go` — contains the inline
  `langDef` map (extensions → language name), the marker action
  table (`go.mod` → Go modules, `pyproject.toml` → Python, etc.).
  Same as today, just relocated.
- `domains/engineering/scan-manifest.yaml` — declares
  `config_keys: [code_scan.depth, code_scan.parser]` so the
  dispatch shell knows which `hero.json` keys are relevant to this
  scanner (used by `hero domain show <pack>` and future tooling).

There is no separate YAML data file for language definitions or
framework detectors. The detectors are Go code because they
embed callable actions (`func() { ... }` that populate the
result). YAMLifying them would require evaluating closures from
data, which is not worth the indirection.

#### Concrete layout (PM, when `hero-pm` ships)

- `domains/pm/scan/import_roadmap.go` — PM-specific roadmap-doc
  parser
- `domains/pm/scan/import_tracker.go` — tracker-epic ingester
- `domains/pm/scan/import_okrs.go` — OKR ingester
- `domains/pm/scan-manifest.yaml` — declares `config_keys:
  [pm.roadmap_path, pm.tracker_filter]`

PM's scanner config (paths, filters) lives in `hero.json` under a
`pm:` block declared by the PM pack. The scanner reads it through
`opts.Config`.

### 8. Sequencing — three delivery PRs

This spec lands in three PRs in order. Each PR is reviewable on its
own and ships green tests.

1. **PR 1 — Dispatch shell + manifest loader.** Adds the `Scanner`
   interface, `Register`/`Dispatch`, manifest YAML parser, and the
   `Report`/`ScanOpts` types under `internal/scan/`. Adds a no-op
   default scanner that the test harness uses to exercise dispatch.
   Adds `cli/scan.go` integration behind a feature flag that
   defaults off — current behavior unchanged. Adds the manifest
   schema and a golden parser test.
2. **PR 2 — Engineering reference relocation.** Moves the existing
   `internal/scan/` detection/generation/enrichment/import code
   into `domains/engineering/scan/`. Adds
   `domains/engineering/scan-manifest.yaml`. Wires the engineering
   scanner into `Register` via `init()`. Flips the `cli/scan.go`
   feature flag default to on so `hero scan` now dispatches.
   **Gates on a golden-output parity test** — runs a representative
   real project through both the pre-refactor and post-refactor
   scanner, diffs the generated entries + graph ingest summary, and
   fails on any divergence.
3. **PR 3 — Domain stamping wire-through.** Threads
   `opts.Config.Domain` through `codescan.WriteGraph` and the
   engineering scanner's knowledge-entry writes. Updates
   `internal/codescan/graph_ingest.go` to accept the domain
   parameter but hardcode `"engineering"` for intrinsic code
   nodes. Adds a stamping test asserting all code nodes carry
   `domain = "engineering"` regardless of `Config.Domain`. This
   PR is the DSKG §Phase 2 contribution for the codescan ingest
   path.

The three PRs are independently revertible. PR 1 lands no
behavior change. PR 2 is the risky one (relocation parity). PR 3
is a narrow API change.

### 9. What does NOT change

- `internal/index/` — the FTS5 spec index. Scanners write through
  the same projection path; no API change.
- `internal/codescan/` location and surface (other than the new
  `domain` parameter on `WriteGraph`). The code intelligence
  package stays put because it's invokable by any future
  code-aware scanner.
- `hero scan` CLI surface for engineering users — flags, output
  format, behavior identical. The parity test gates this.
- The graph schema. DSKG owns the `domain` column addition;
  this spec consumes it.
- `hero.json` shape. The existing `domain` field and
  `code_scan.*` block are all that's needed.

## Acceptance Criteria

- THE SYSTEM SHALL expose a `scan.Scanner` interface that domain
  pack scanners implement.
- THE SYSTEM SHALL load `domains/<active-domain>/scan-manifest.yaml`
  at process start, parse its `scanner_id`, `subcommands`,
  `emits`, and `config_keys` blocks, and fail process startup
  with a clear error pointing at the offending file and line on
  malformed YAML.
- THE SYSTEM SHALL expose a `scan.Dispatch(subcommand, opts)`
  function that looks up the active pack's scanner via the
  manifest's `scanner_id` and invokes its `Scan` method.
- WHEN the active pack ships no `scan-manifest.yaml` THE SYSTEM
  SHALL print `<domain> pack does not ship a scanner; nothing
  to do` and exit zero.
- WHEN `hero scan` is invoked with `Config.Domain = "engineering"`
  (the default) THE SYSTEM SHALL dispatch to the engineering
  reference scanner registered from `domains/engineering/scan/`.
- WHEN `hero scan` is invoked AND the active pack's scanner
  implements the requested subcommand THE SYSTEM SHALL execute
  the scanner with `ScanOpts` carrying `Config`, `Store`, project
  root, hero dir, parsed flags, dry-run / force, and a `Reporter`.
- WHEN `hero scan` is invoked AND the active pack's scanner does
  not implement the requested subcommand THE SYSTEM SHALL print
  `<domain> pack does not implement 'scan <subcommand>'` and exit
  non-zero.
- WHEN `hero scan --code` is invoked from a non-engineering active
  pack THE SYSTEM SHALL reject the flag with a clear
  engineering-only error and exit non-zero.
- WHEN the engineering reference scanner runs on a real project
  whose pre-refactor `hero scan` output was captured as a golden
  baseline THE SYSTEM SHALL produce identical generated entries
  AND an identical graph ingest summary (`TestScanReferenceParity`).
- THE SYSTEM SHALL pass `Config.Domain` from `ScanOpts` into the
  engineering scanner's knowledge-entry writes so that knowledge
  stubs and conventions emitted by the scanner carry the active
  domain tag in the graph.
- THE SYSTEM SHALL stamp every `Repo`/`Package`/`File`/`Symbol`
  node written by `codescan.WriteGraph` with `domain =
  "engineering"` regardless of the `domain` parameter passed in.
- WHEN `codescan.WriteGraph` is called with a non-engineering
  `domain` parameter THE SYSTEM SHALL still stamp every emitted
  node with `engineering` AND emit a single info-level reporter
  message naming the override.
- THE SYSTEM SHALL retain `hero scan`'s existing flags (`--dry-run`,
  `--force`, `--code`, `--no-hooks`) with their current semantics
  during the dispatch refactor (parity gated by
  `TestScanReferenceParity`).
- THE SYSTEM SHALL run the cross-cutting work-subgraph ingest
  (planning specs, mission, sessions, git, knowledge, NEXT,
  handoff, AC participation, tier-2 extraction, claude-memory,
  tracker, team-server) from the dispatch shell after the
  scanner's `Scan()` returns, not from inside the scanner.
- WHEN the engineering scanner runs in a workspace with sibling
  repos configured in `hero.json` THE SYSTEM SHALL ingest sibling
  specs through the dispatch shell's shared sibling-subgraph
  pass, unchanged from today.
- THE SYSTEM SHALL list all registered scanners via a new
  `hero domain show <name>` subcommand that includes the scanner's
  display name, scanner_id, and declared subcommands.
- THE SYSTEM SHALL emit the same FTS5 projection (`index.ProjectGraphNodes`)
  after dispatch completes, regardless of which pack's scanner ran.
- IF a manifest declares a `scanner_id` that has no matching
  `Register` call at startup THEN THE SYSTEM SHALL fail process
  startup with a clear error naming the missing scanner_id and the
  manifest file.

## Boundaries

- **Not** designing PM scanners. The roadmap-doc parser,
  tracker-epic ingester, and OKR ingester all live in `hero-pm`
  and use this spec's interface as their plug-in shape.
- **Not** changing the knowledge-graph schema. DSKG owns the
  `domain` column; this spec consumes it via the wire-through.
- **Not** changing scan triggering. The CLI command shape,
  post-init hook behavior, and `--code` flag stay as-is.
- **Not** introducing third-party / out-of-tree scanners.
  Scanners ship with their pack, in-tree, via `init()` registration.
  No `plugin.Open`, no dynamic library loading.
- **Not** introducing free-form per-pack subcommands. The
  sub-command set is canonical and packs opt in via manifest.
- **Not** shipping multi-active-domain scan. `hero scan` runs
  exactly one scanner per invocation in v1.
- **Not** changing `internal/index/` or `internal/codescan/`
  locations or surfaces beyond the `domain` parameter addition
  on `codescan.WriteGraph`.
- **Not** introducing a separate YAMLified detector dictionary
  for language extensions or framework markers. Detectors stay
  in Go.
- **Not** changing the merge-planner contract for knowledge
  entries (`PlanMerge`/`ExecuteMerge` stay in `internal/scan/`
  with the same shape; only the call site moves).
- **Not** introducing scanner-level cancellation. Existing scan
  has no cancellation today; the interface preserves that. A v2
  context-aware scan is a separate spec.

## Risks

1. **Engineering reference parity is the highest-risk piece.**
   Moving ~3,000 lines of detection logic from `internal/scan/`
   into `domains/engineering/scan/` must produce bit-identical
   output for engineering users on day one. The
   `TestScanReferenceParity` golden test must run against at least
   three real projects (Hero itself, a multi-module Java project,
   a Python project with a `pyproject.toml`) and gate merge on
   zero diff. Mitigation: PR 2 includes the golden harness and
   the captured baselines before any code moves; the relocation
   PR fails CI on any byte-level divergence in the merged entry
   set or the graph ingest summary.
2. **Dispatcher map registration timing.** Go `init()` functions
   run in import order, which is not deterministic across builds
   if not carefully wired. Mitigation: a single
   `cmd/hero/main.go` blank-import (`_ "github.com/hero-engine/hero/domains/engineering/scan"`)
   anchors registration; the dispatcher's `Dispatch` call asserts
   the scanner is registered and fails fast with a clear error if
   not. The same pattern is already used for spec-types and
   vocabulary loaders.
3. **The `Config.Domain` wire-through can be silently bypassed.**
   A scanner that forgets to pass `opts.Config.Domain` into its
   graph writes will stamp the wrong domain (or stamp empty). The
   DSKG spec's CI lint catches new `UpsertNode`/`UpsertEdge` calls
   without a `Domain` field. Mitigation: this spec's PR 3 adds
   tests asserting engineering's scanner stamps `engineering` on
   every emitted node; the lint covers future scanners.
4. **Pack-shipped scanners are user-visible early.** PM users run
   scan during onboarding; a flaky scanner is a bad first
   impression. This spec ships only the engineering reference,
   which is already battle-tested. PM's scanner ships under
   `hero-pm` and must include its own e2e test against a
   representative roadmap doc before merge.
5. **The manifest format will evolve.** The v1 manifest declares
   subcommands, emits, and config keys. Future packs may need
   pre-scan dependency declarations ("this scanner requires the
   tracker integration to be configured"), output-format
   variants, or progress-event taxonomies. Mitigation: the
   manifest YAML carries a `manifest_version: "1"` field; loader
   rejects unknown versions with a clear error so additions land
   as a v2 bump.
6. **`internal/codescan/` location stays even though it's only
   called by the engineering scanner today.** A reviewer might
   argue codescan belongs under `domains/engineering/scan/`.
   Decision: it stays at `internal/codescan/` because it is a
   reusable code-intelligence library that any future code-aware
   pack (data-analytics scanning SQL, qa scanning test files)
   would call into without needing engineering to be active.
   Risk: dead-code feel for non-engineering users. Mitigation:
   the package is unimported when no scanner calls it; the binary
   size impact is minimal (Go's linker strips unused packages
   at link time).
7. **Cross-cutting work-subgraph ingest moves out of
   `cli/scan.go` and into `internal/scan/postwork.go`.** This is
   a ~400-line shuffle and could be reverted into the dispatch
   shell vs the scanner with little semantic change. Mitigation:
   the split is justified by future-PM-scanner inheritance
   (don't make PM's scanner reimplement sibling-subgraph ingest)
   and the `TestScanReferenceParity` golden test asserts the
   final output is identical regardless of where the cross-
   cutting code lives.
8. **Existing callers of `scan.Analyze` and `scan.Generate` from
   non-CLI code paths.** `internal/cli/init.go` calls
   `scan.Analyze` directly during `hero init` to populate initial
   knowledge stubs. After the refactor, `scan.Analyze` lives
   under `domains/engineering/scan/`. Mitigation: `internal/cli/
   init.go` either dispatches through `scan.Dispatch` (preferred
   — the same shell engineering uses) or imports the engineering
   scanner package directly with a clear comment that init
   currently assumes engineering-default-domain. The latter is
   simpler for v1; document the assumption.
9. **MCP tool surface for scan.** `internal/serve/mcp_tools.go`
   exposes a scan-adjacent tool. Confirm whether it calls into
   `internal/scan/` or `internal/codescan/` directly; the
   relocation must keep its surface working. Mitigation: PR 2's
   golden test includes an MCP-driven scan call against a fixture
   workspace.

## Resolved open questions

1. **Output schema — shared or domain-typed.** Resolved: shared
   node/edge types with `domain` tag (and `kind` facet from the
   spec-type registry for work-spec sub-categorization). Intrinsic
   content types (`Package`, `File`, `Symbol`) stay typed. Domain-
   typed nodes were rejected because the DSKG audit table assumes
   `WHERE n.domain = ?` is sufficient scoping; per-domain node-
   type matrices would require rewriting that audit and every
   read site.
2. **Scanner manifest vs Go code.** Resolved: manifest +
   in-tree dispatcher map. Manifest is a small declarative
   descriptor (`scanner_id`, `subcommands`, `emits`, `config_keys`);
   the scanner itself is Go code that registers via `init()`
   into `internal/scan.Register`.
3. **Scan composition.** Resolved: single-active-pack v1. The
   dispatch interface preserves multi-pack v2 without refactoring
   (`Dispatch` runs one scanner per call; a future
   `--all-domains` flag iterates the map).
4. **Progress and cancellation.** Resolved: `ScanOpts.Reporter`
   gives every scanner a progress sink. Cancellation is not in v1
   — existing engineering scan has no cancellation today; we
   carry that forward and address it in a separate spec.
5. **Failure handling.** Resolved: scanners return errors from
   `Scan()`; the dispatch shell prints them and exits non-zero.
   Partial outputs (a knowledge entry written before the failure)
   stay on disk — this matches today's behavior. The graph store
   is bitemporal; partial graph writes are idempotent and
   re-runnable.
6. **Sub-commands per pack.** Resolved: fixed canonical sub-command
   set (`scan` default, with `--code`/`--dry-run`/`--force` flags
   universal). Per-pack opt-in via manifest. The existing `--code`
   engineering-legacy flag stays as a shortcut and is rejected
   for non-engineering active packs.
7. **Where pack-shipped scanner config lives.** Resolved: alongside
   the scanner code at `domains/<name>/scan/`. Detector
   dictionaries stay inline in Go (callable actions). Manifest
   declares which `hero.json` config keys the scanner reads.
8. **`internal/scan/` vs `domains/engineering/scan/`.** Resolved:
   `internal/scan/` becomes the dispatch shell + shared infra
   (merge planner, manifest loader, dispatch, reporter,
   post-scan work-subgraph ingest). Detection / generation /
   enrichment / import code moves to `domains/engineering/scan/`.
   `internal/codescan/` stays put.

## Touchpoints

- `internal/scan/dispatch.go` — NEW: `Scanner` interface, `Register`,
  `Dispatch`, dispatcher map
- `internal/scan/manifest.go` — NEW: scan-manifest.yaml parser
- `internal/scan/opts.go` — NEW: `ScanOpts`, `Report`,
  `GraphIngestSummary` types
- `internal/scan/reporter.go` — NEW: `Reporter` interface +
  stdout / quiet implementations
- `internal/scan/entry.go` — NEW: `GeneratedEntry` type (moved from
  current `internal/scan/scan.go`)
- `internal/scan/postwork.go` — NEW: cross-cutting work-subgraph
  ingest extracted from `cli/scan.go` (sibling subgraphs, mission,
  sessions, git, knowledge, NEXT, handoff, AC participation, tier-2
  extraction, claude-memory, tracker, team-server)
- `internal/scan/merge.go` — STAYS: merge planner (`MergeDecision`,
  `PlanMerge`, `ExecuteMerge`)
- `internal/scan/merge_test.go` — STAYS
- `internal/scan/scan.go` — DELETED (logic moves to engineering)
- `internal/scan/scan_test.go` — MOVED to
  `domains/engineering/scan/scan_test.go`
- `internal/scan/generate.go` — MOVED to
  `domains/engineering/scan/generate.go`
- `internal/scan/enrich.go` — MOVED to
  `domains/engineering/scan/enrich.go`
- `internal/scan/enrich_test.go` — MOVED
- `internal/scan/import.go` — MOVED to
  `domains/engineering/scan/import.go`
- `internal/scan/import_test.go` — MOVED
- `internal/scan/modules.go` — MOVED to
  `domains/engineering/scan/modules.go`
- `internal/scan/modules_test.go` — MOVED
- `domains/engineering/scan/init.go` — NEW: `init()` registers the
  engineering scanner via `scan.Register`
- `domains/engineering/scan/scanner.go` — NEW: implements
  `scan.Scanner` interface; orchestrates Analyze + Generate +
  knowledge merge + codescan ingest
- `domains/engineering/scan/analyze.go` — MOVED from
  `internal/scan/scan.go` (detection logic: `Analyze`,
  `detectLanguages`, `detectFromMarkers`, `detectCI`,
  `detectMonorepo`, `detectDocFiles`)
- `domains/engineering/scan-manifest.yaml` — NEW: declares
  `scanner_id: engineering-code-scan`, subcommands, emits, config
  keys
- `internal/cli/scan.go` — REWRITTEN: thin shell that builds
  `ScanOpts`, calls `scan.Dispatch`, prints `Report`
- `internal/cli/init.go` — UPDATED: `hero init`'s call to
  `scan.Analyze` becomes a direct import of the engineering
  scanner package (engineering-default-domain assumption,
  documented in code)
- `internal/codescan/graph_ingest.go` — UPDATED: `WriteGraph`
  signature gains `domain string` parameter; body still hardcodes
  `"engineering"` for all `Repo`/`Package`/`File`/`Symbol` writes
  per DSKG §Phase 2
- `internal/codescan/graph_ingest_test.go` — UPDATED: assert all
  code nodes stamped `engineering` regardless of input domain
- `internal/codescan/scanner.go` — UNCHANGED (still calls
  `scan.DetectMultiModule`; after relocation this call is updated
  to `engineering.DetectMultiModule` or moves up into the
  engineering scanner)
- `internal/serve/mcp_tools.go` — UPDATED: scan MCP tool routes
  through `scan.Dispatch` like the CLI does
- `cmd/hero/main.go` — UPDATED: blank-import
  `_ "github.com/hero-engine/hero/domains/engineering/scan"` to
  anchor registration order
- `content.go` (or wherever embed.FS for domains lives) — UPDATED:
  ensure `domains/<name>/scan-manifest.yaml` is included in the
  embedded filesystem
- `internal/scan/dispatch_test.go` — NEW: unit tests for
  manifest loading, scanner registration, dispatch routing, the
  no-scanner-shipped path
- `internal/scan/parity_test.go` — NEW: `TestScanReferenceParity`
  golden test runs three fixture projects through the dispatched
  engineering scanner and diffs against captured pre-refactor
  baselines
- `internal/cli/scan_report_test.go` — UPDATED: assertions on the
  report block format unchanged
- `internal/cli/scan_test.go` — UPDATED: integration test wires
  through the dispatch shell
- `internal/cli/domain.go` — UPDATED: `hero domain show <name>`
  subcommand reads `scan-manifest.yaml` and lists declared
  subcommands + emits + config_keys
- `.hero/planning/features/domain-scoped-knowledge-graph/spec.md`
  — cross-references this spec for the codescan stamping
  contract (no edit required; DSKG already names the engineering
  hardcode rule)
