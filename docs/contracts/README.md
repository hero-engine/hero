# Hero Cross-Language Contracts

Hero exposes eight load-bearing contracts that any non-Go consumer — the
hero-code Rust dashboard, IDE plugins, peer Hero installations — reads
to render and reason about a Hero workspace without coupling to the Go
implementation. Four of them describe the **dialect** of a workspace
(which work-tracking types exist, what they're called, how their
lifecycles run), one describes the **inline-propose** wire protocol between
Hero agents and the local daemon, and two define credential-safe tracker
boundaries, while one defines a credential-safe code-host boundary. Together
they pin
the shapes that have to stay stable across language boundaries.

## At a glance

| Contract | Location | Schema version | Owner | Stability |
|---|---|---|---|---|
| Spec-type registry | `.hero/cache/spec-types.json` (regenerated) | `1.1` | Hero (Go) | Additive in `1.x`; breaking → `2.0` |
| Vocabularies | `core/vocabularies/*.yaml` (static) | preset v1 | Hero (Go) | Presets additive; rename = breaking |
| Vocabulary auto-select | `docs/contracts/vocabulary-auto-select.md` | preset v1 | Hero (Go) | Recognized keys additive; rename = breaking |
| Methodologies | `core/methodologies/*.yaml` (static) | profile v1 | Hero (Go) | Profile names stable; fields additive |
| Inline-propose wire | `docs/contracts/inline-propose-v1.md` | `1.0` | Hero (Go) | Additive in `1.x`; breaking → `2.0` |
| Tracker broker | `docs/contracts/tracker-broker-v1.md` | `tracker-broker/v1` | Hero (Go) | Additive in v1; breaking → v2 |
| Tracker evidence | `docs/contracts/tracker-evidence-v1.md` | `tracker-evidence/v1` | Hero (Go) | Additive in v1; breaking → v2 |
| Code-host broker | `docs/contracts/code-host-broker-v1.md` | `code-host-broker/v1` | Hero (Go) | Additive in v1; breaking → v2 |

## Spec-type registry

Defines the canonical set of work-tracking types a workspace knows
about — the nine core types (`initiative`, `epic`, `feature`,
`sprint`, `release`, `prd`, `bug`, `intake`, `chore`) plus the
engineering domain's `decision` and `convention`. Source documents
live under `core/spec-types/*.md` and `domains/<active>/spec-types/*.md`;
the registry is **regenerated on every `hero` command** by
`cli.PersistentPreRun` (see `internal/cli/root.go`) and emitted to
`.hero/cache/spec-types.json`. Loader and exporter:
`internal/spectypes/`. Consumers should treat the cache file as the
single source of truth, never the markdown sources directly. Key
invariant: within `1.x` the schema is **additive only** — new fields
may appear, never removed or renamed. A breaking change bumps to
`2.0`. The `schema_version` field on the top-level export advertises
the version a given workspace is emitting.

## Vocabularies

Vocabularies are **display dialects**: they map canonical spec types
(`feature`, `bug`, `epic`, …) and kinds onto user-facing labels
(e.g. "Story" vs. "Card" vs. "Pitch") and configure tracker-side
mappings. Six v1 presets ship in `core/vocabularies/`: `default`,
`agile-scrum`, `kanban`, `shape-up`, `jira`, `linear`. The loader
lives in `internal/vocabulary/`. Files are **static** YAML — there is
no generated cache; consumers read them straight from the repo, or
through the resolver if they want the active dialect for a workspace.
Key invariant: existing preset filenames and the canonical type names
they key off of are **stable** within v1 — renaming a preset is a
breaking change. Adding a new preset or a new display entry is
additive. See `docs/contracts/active-dialect.md` for how a workspace
selects its active vocabulary.

## Methodologies

Methodologies are the **structural layer** that sits underneath
vocabularies: lifecycle overrides per canonical type, time-box
requirements (release + sprint duration + required rituals),
estimation field, in-flight tracking shape, cadence, rollups, and an
`aligned_vocabulary` pointer that auto-derives the display dialect
when no explicit vocabulary is set. Five v1 profiles ship in
`core/methodologies/`: `scrum`, `kanban`, `scrumban`, `shape-up`,
`waterfall`. Loader: `internal/methodology/`. Like vocabularies these
are **static** YAML, read directly. Key invariants: profile names
(`scrum`, `kanban`, …) are stable identifiers consumers may key off
of; field additions to a profile are additive; removing or
semantically reinterpreting an existing field is breaking. The
`aligned_vocabulary` link is what lets a workspace declare just
`methodology: scrum` and have the dialect resolve to `agile-scrum`
automatically.

## Inline-propose wire contract

Defines how Hero agents emit structured proposals (anchor + content
+ position) that the daemon stores and the hero-code dashboard
renders for human review. Two surfaces:

- **Stdout protocol** — NDJSON lines on agent stdout prefixed by the
  literal token `HERO-PROPOSAL: `, captured by the daemon shim.
- **REST + SSE** — endpoints under
  `/api/{project}/sessions/{session_id}/proposals` for CRUD plus five
  SSE event types: `proposal_emitted`, `proposal_accepted`,
  `proposal_edited`, `proposal_rejected`, `proposal_dismissed`.

Implementation: `internal/propose/` (envelope + store) and
`internal/serve/proposals.go` (HTTP + events). Full wire shapes,
field semantics, and validation rules live in
`docs/contracts/inline-propose-v1.md` (the load-bearing spec).
Key invariant: additive-only within `1.x`. A breaking change bumps
to `2.0` and the daemon serves **both** schema versions during a
deprecation window — consumers select via the `schema_version` field
on each envelope. Fixture envelopes for client-side tests live in
`testdata/proposals/v1/`.

## Read order for new consumers

Each layer composes on the previous, so a fresh consumer should read
top-down:

1. **Spec-type registry** — what kinds of work exist and what each
   type's lifecycle, sections, and frontmatter look like.
2. **Vocabularies** — how to render those types in the user's chosen
   dialect.
3. **Methodologies** — how lifecycles, time-boxes, and rollups
   change under a chosen methodology, and how the active vocabulary
   is auto-derived if not pinned.
4. **Inline-propose** — only relevant for tools that surface
   agent-generated edits; orthogonal to the dialect layers but reuses
   the registry's type names in proposal envelopes.
5. **Tracker broker** — only relevant for clients that need broad tracker
   operations while Hero retains the configured credential.
6. **Tracker evidence** — explicit per-spec full-details loading with a safe
   tracked manifest and validated private snapshot.
7. **Code-host broker** — provider-neutral pull-request lifecycle operations
   for clients while Hero retains the selected code-host credential.

Then read `docs/contracts/active-dialect.md` for the resolver
precedence chain (explicit → methodology-derived → tracker-inferred
→ default) and a worked example.

## Companion docs

- [`active-dialect.md`](./active-dialect.md) — how `hero.json` plus
  the resolver pick an active vocabulary and methodology for a
  workspace, with a worked example.
- [`vocabulary-auto-select.md`](./vocabulary-auto-select.md) —
  authoring guide for a vocabulary preset's `auto_select:` block:
  recognized keys, allowed values, match semantics, and a worked
  example.
- [`spec-types-v1.1.schema.json`](./spec-types-v1.1.schema.json) —
  JSON Schema (draft 2020-12) validating
  `.hero/cache/spec-types.json`; usable with `quicktype` / `schemafy`
  to generate Rust `serde` types.
- [`inline-propose-v1.md`](./inline-propose-v1.md) — full wire spec
  for the proposal protocol.
- [`tracker-broker-v1.md`](./tracker-broker-v1.md) — credential-safe
  broad tracker operations, response envelope, effects, and consumer fixture.
- [`tracker-evidence-v1.md`](./tracker-evidence-v1.md) — explicit full-details
  loading, cache validation, private sidecar storage, and consumer fixture.
- [`code-host-broker-v1.md`](./code-host-broker-v1.md) — repository-qualified
  pull-request operations, effects, idempotency, reconciliation, bounds, and
  canonical cross-language fixture.
- [`../../examples/scrum-workspace/`](../../examples/scrum-workspace/)
  — runnable sample workspace declaring `methodology: scrum` +
  `vocabulary: agile-scrum`, with specs across lifecycle states.

## How to file issues

Contract drift bugs (cache shape disagrees with this doc, fixture
envelope rejected by validator, vocabulary preset renamed without a
deprecation window) should be filed through the cross-repo peering
mechanism rather than as a generic GitHub issue, so the owning Hero
installation picks them up directly:

```
hero peer call hero --mode=advisory \
  --reason="contract-drift: <short description>" \
  --related-spec=<your-spec-slug> \
  "<details, including the contract name and observed vs. expected shape>"
```
