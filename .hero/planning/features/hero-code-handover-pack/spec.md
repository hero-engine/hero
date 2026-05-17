---
title: Hero-Code Handover Pack — Make the PM Foundation Consumable
type: feature
status: planning
priority: P0
tags: [sprint, delivery, pm, contracts, hero-code, handoff]
created: 2026-05-17
relations:
  - target: pm-foundation-delivery
    kind: parent
  - target: hero-code
    kind: cross-repo-consumer
horizon: now
smoke: deferred
---

## Kickoff

The PM Foundation Delivery sprint shipped the four cross-language contracts (`spec-types.json`, `vocabularies/*.yaml`, `methodologies/*.yaml`, `inline-propose-v1.md`) but left a handful of consumer-affordance gaps. This sprint closes them so hero-code can implement the first PM dashboard against real fixtures, a single discovery index, and a documented read path for active workspace dialect.

**Sprint completes when:**
- `testdata/proposals/v1/` carries a fixture envelope per anchor variant + batch + replacement scenarios — hero-code's Rust widget tests consume them
- `docs/contracts/README.md` indexes the four contracts with location, schema version, owner, and stability promise
- `docs/contracts/active-dialect.md` documents the resolver precedence chain and the on-disk read path from `hero.json` to display map
- `docs/contracts/spec-types-v1.1.schema.json` validates `.hero/cache/spec-types.json` and can generate Rust types via `serde`
- `examples/scrum-workspace/` ships a working hero.json + 4 specs across lifecycle states for hero-code to develop against
- Hero-code peer call (advisory) hands over the handover pack with pointers to all five artifacts

## Goal

Lower the activation energy for hero-code (and any future client) to consume Hero's stable contracts. The PM Foundation sprint produced the contracts; this sprint produces the **discoverability, validation, and development affordances** that make those contracts actually usable.

## Work items

### C1. Inline-propose test fixtures

Ship `testdata/proposals/v1/*.json` — one envelope per anchor variant plus batch and replacement scenarios. Required by the inline-propose-output-mode spec ("ship the fixture JSON in `testdata/proposals/`") and called out as a consumer obligation in `docs/contracts/inline-propose-v1.md`.

Fixtures to ship:
- `single-ac-append.json` — canonical case (section + append)
- `frontmatter-field-replace.json` — frontmatter target, YAML content
- `section-replace.json` — full section body replacement
- `heading-before.json` — heading anchor with before position
- `list-item-after.json` — list_item kind with after position
- `free-position.json` — anchor unknown, hint provided
- `batch-multi-ac.json` — four envelopes sharing one batch_id
- `replacement-scenario.json` — two envelopes, same agent + anchor (second replaces first)
- `README.md` — what each fixture exercises, how hero-code's tests consume them

Each fixture validates against the envelope schema in `docs/contracts/inline-propose-v1.md`.

### C2. Contract index README

New `docs/contracts/README.md` indexing the four contracts. Table shape:

| Contract | Location | Schema version | Owner | Stability |

Brief prose explaining how the contracts compose (registry → vocabulary → methodology → inline-propose), what's regenerated vs. static, and where to file consumer-side issues.

### C3. Active dialect read path

New `docs/contracts/active-dialect.md` documenting how a consumer reads the active vocabulary and methodology for a workspace. Covers:
- `hero.json` fields (`vocabulary`, `methodology`, `vocabulary_overrides`, `methodology_overrides`, `tracker`, `domain`)
- Resolver precedence chain (explicit > methodology-derived > tracker-inferred > default)
- The `aligned_vocabulary` auto-derivation from methodology
- How to map a canonical type + kind to a display name (the `display:` map on each vocabulary YAML)
- A worked example: workspace declares `methodology: scrum` only → derives `vocabulary: agile-scrum` → renders `feature` as "Story"

### C4. JSON Schema for spec-types.json

New `docs/contracts/spec-types-v1.1.schema.json` — a JSON Schema (draft 2020-12) describing the `.hero/cache/spec-types.json` export shape. Hero-code can:
- Validate the cache file at load time
- Generate Rust `serde` types from the schema via `quicktype` or `schemafy`

Schema mirrors `internal/spectypes/export.go`'s `jsonExport` / `jsonExportRecord` structs.

### C5. Sample scrum workspace

New `examples/scrum-workspace/` — a working Hero workspace declaring `methodology: scrum` + `vocabulary: agile-scrum`, with:
- `hero.json` — minimum config to exercise the dialect
- `.hero/planning/features/` — three specs across lifecycle states (`planning`, `delivering`, `completed`)
- `.hero/planning/bugs/` — one bug in `planning`
- `README.md` — how to use it: clone the example into a new workspace or `cd` in and run `hero status` / `hero list`

Lets hero-code spin up a real workspace immediately rather than authoring synthetic test data.

### C6. Handoff to hero-code

Once C1–C5 land, fire a `hero peer call hero-code --mode=advisory` referencing this spec, with explicit pointers to each artifact and a short integration checklist. The advisory call lands a `Handoff Trail` entry on this spec automatically.

## Acceptance Criteria

- THE SYSTEM SHALL ship `testdata/proposals/v1/{single-ac-append,frontmatter-field-replace,section-replace,heading-before,list-item-after,free-position,batch-multi-ac,replacement-scenario}.json` plus a `README.md`.
- THE SYSTEM SHALL ship `docs/contracts/README.md` indexing the four contracts (spec-types, vocabularies, methodologies, inline-propose) with location, version, owner, stability.
- THE SYSTEM SHALL ship `docs/contracts/active-dialect.md` documenting the resolver precedence chain and the worked example for `methodology: scrum`.
- THE SYSTEM SHALL ship `docs/contracts/spec-types-v1.1.schema.json` validating against a fresh `.hero/cache/spec-types.json`.
- THE SYSTEM SHALL ship `examples/scrum-workspace/` with `hero.json` declaring scrum + agile-scrum and at least four specs across lifecycle states.
- WHEN C1–C5 land, THE SYSTEM SHALL fire a hero-code advisory peer call referencing this spec.

## Boundaries

- **Not** generating Rust code in this repo. hero-code generates its own types from the JSON Schema.
- **Not** writing integration tests for hero-code. Fixtures are static JSON; hero-code's test suite is theirs.
- **Not** documenting the Go-side internals. The contracts and the read path documents are consumer-facing only.
- **Not** changing any of the four contracts. This sprint is additive discoverability work.

## Sprint completion checklist

- [x] C1: inline-propose test fixtures shipped + README — 8 envelopes + array fixtures + README at `testdata/proposals/v1/`
- [x] C2: docs/contracts/README.md index — discovery table + per-contract sections + read order
- [x] C3: docs/contracts/active-dialect.md — resolver precedence + worked example for `methodology: scrum`
- [x] C4: docs/contracts/spec-types-v1.1.schema.json validates fresh cache — Python `jsonschema` (Draft 2020-12) confirms cache validates with 0 errors
- [x] C5: examples/scrum-workspace/ runnable — `hero status` from inside it renders "Vocabulary: agile-scrum · Methodology: scrum", features render as Story
- [x] C6: hero-code advisory peer call fired
- [x] `go build ./...` and `go test ./...` clean — verified after merge

## Open contract gaps surfaced during the sprint

These are followups, not blockers — recorded here so hero-code (and we) can address them in their own specs:

1. **`vocabulary.Resolve` doesn't fold in methodology-derived auto-derivation.** Every in-tree caller (`internal/cli/vocab.go::activeVocab`, `internal/serve/vocab.go`, `internal/install/dialect.go`) wraps it and injects `DeriveVocabularyName` first. The contract in `active-dialect.md` documents the effective precedence chain consumers see, but the bare function name is misleading. Either fold the derivation into `Resolve` or rename to `ResolveBare` / `ResolveWithMethodology`.

2. **`frontmatter` field on `spec-types.json` records is always `null` today.** The loader doesn't yet populate it from the markdown source. Schema declares it `oneOf: [object, null]` so consumers handle both today's reality and tomorrow's populated shape without a contract bump. Worth filing as a registry-loader bug.

3. **Some envelope semantics are prose-only.** `anchor.value` carries different shapes per `anchor.kind` (frontmatter field name vs section slug vs list-item id vs free-form hint). hero-code may want a typed sum-type with `value` shape per variant. Worth considering a 1.1 additive `anchor.value_shape` discriminator if downstream typing pain materializes.

4. **`initiative` required-sections disagrees with prose.** `core/spec-types/initiative.md` YAML says `required: [Goal]`, prose says `[Bet, Evidence, Tradeoffs]`. Trivial fix.

5. **Methodology `auto_select` schema not exhaustively documented.** The `vocabulary.yaml` auto_select rule shape is referenced but only `delivery_preset` is shown by example. A consumer authoring a new preset will have to crack the Go struct.

6. **`hero.local.json` merge semantics** don't yet forward `vocabulary` / `methodology` fields. Documented in `active-dialect.md` as a planned extension point; the Go code doesn't yet honor it.
