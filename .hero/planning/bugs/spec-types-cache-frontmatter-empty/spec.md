---
title: spec-types.json records emit frontmatter as null — loader never populates it
slug: spec-types-cache-frontmatter-empty
type: bug
status: delivering
severity: medium
priority: P1
created: 2026-05-17
tags: [spectypes, registry, cache, contract, loader]
relations:
  - target: hero-code-handover-pack
    kind: surfaced-by
  - target: pm-foundation-delivery
    kind: regression-of
---

# spec-types.json records emit frontmatter as null — loader never populates it

## Problem

Every record in `.hero/cache/spec-types.json` carries `"frontmatter": null`. The contract documents this field as the canonical frontmatter schema for the spec type — the field types, defaults, enums, format hints, descriptions — that drives lint and hero-code's type-aware form rendering. The JSON Schema (`docs/contracts/spec-types-v1.1.schema.json`) declares it `oneOf: [object, null]` to permit today's `null` reality without locking the contract, but the **populated** shape is what consumers actually need.

Verified live:

```bash
jq '.types[].frontmatter' .hero/cache/spec-types.json | sort -u
# Output: null  (only — every one of the 11 types)
```

Surfaced by the C4 agent (spec-types JSON Schema work item, `hero-code-handover-pack`) while validating the schema against the live cache. Hero-code's planned PM dashboard work (typed form rendering per spec type) can't proceed past stub-quality fields without this populated.

## Steps to Reproduce

```
rm -f .hero/cache/spec-types.json
go run ./cmd/hero status > /dev/null
jq '.types[0].frontmatter' .hero/cache/spec-types.json
# Expected: { "fields": [ {"name": "title", "type": "string", "required": true, ...}, ... ] }
# Actual:   null
```

Every canonical type file (e.g. `core/spec-types/feature.md`) declares the frontmatter schema in its YAML — the data exists at source; it just isn't loaded.

## Expected Behavior

`frontmatter` on each cached record should be populated from the corresponding markdown source file. For `feature`, that means at minimum: `title (string, required)`, `type (string, required, const "feature")`, `status (enum, required, values from lifecycle.states)`, `priority (enum)`, `severity (enum)`, `tags (list[string])`, `created (date)`. The exact shape lives in `internal/spectypes/registry.go::FrontmatterSchema` / `FieldDecl`.

## Root Cause

`internal/spectypes/loader.go::parseRecord` parses every other YAML block in the frontmatter (lifecycle, kind, owner, tasks_schema, sections, accepting_commands, default_agents, relations) but **does not** parse the `frontmatter:` block from the markdown source. The `FrontmatterSchema` Go type is declared and the export-side `jsonExportFrontmatterSchema` mirrors it, but the loader-side parser entry is missing.

Likely sequence: when B2 (spec-type registry Go implementation) landed, the FrontmatterSchema export struct was authored against the design but the loader was never wired to populate it from the source markdown. No test pinned `Record.Frontmatter != nil` for any core type, so the gap went unnoticed.

## Fix

Add `frontmatter` parsing to `internal/spectypes/loader.go::parseRecord`:

1. Define a `rawFrontmatterSchema` raw-YAML helper struct mirroring `FrontmatterSchema` field names with `yaml:` tags.
2. Decode the `frontmatter:` block from the markdown's YAML frontmatter (same `extractFrontmatter` path the other blocks use).
3. Convert raw decl → `FrontmatterSchema` via a new `convertFrontmatterSchema(raw rawFrontmatterSchema) *FrontmatterSchema` helper (mirrors the existing `convertField` pattern).
4. Authoritative source for the `frontmatter:` block in each spec-type file is the existing markdown — verify the eleven core + engineering type files all declare their frontmatter block; if any don't, author it from the field set their lint validator currently enforces.

## Acceptance Criteria

- THE SYSTEM SHALL parse the `frontmatter:` block from each `core/spec-types/*.md` and `domains/<active>/spec-types/*.md` file when building the registry.
- WHEN a record is exported to `.hero/cache/spec-types.json` AND its source file declares a `frontmatter:` block, THE SYSTEM SHALL emit a non-null `frontmatter` object on the record.
- THE SYSTEM SHALL populate `Frontmatter.Fields[]` with one entry per declared field, each carrying `name`, `type`, `required`, `default`, `values` (for enums), `format`, `classification`, `description` as declared.
- THE SYSTEM SHALL preserve `frontmatter: null` behavior only for type files that genuinely have no `frontmatter:` declaration (knowledge/meta types out of sprint scope).
- THE SYSTEM SHALL keep `jsonExportFrontmatterSchema`'s wire shape unchanged — only the *value* changes, not the schema.
- THE SYSTEM SHALL pass `docs/contracts/spec-types-v1.1.schema.json` validation against the regenerated cache (the schema already permits both shapes; populated shape must validate).

## Boundaries

- **Not** changing the `FrontmatterSchema` Go struct shape or the JSON Schema document. The export contract is fixed; only the data fill changes.
- **Not** authoring new core type files. Only populating frontmatter blocks where missing on the eleven existing types.
- **Not** changing the legacy lint validator (`internal/triage/`). Registry-driven lint is separate work.
- **Not** modifying knowledge/meta types (`decision`, `convention` excepted since they ship in `domains/engineering/spec-types/`).

## Validation

- Unit test in `internal/spectypes/loader_test.go`: load the core registry and assert `Lookup("feature").Frontmatter != nil` and `len(Frontmatter.Fields) > 0`. Cover all nine canonical work types plus engineering's two.
- Update `TestExportTo_WritesCacheFile` to assert at least one record in the exported cache has a non-null `frontmatter` block.
- Run the JSON Schema validation against the regenerated cache (the existing Python `jsonschema` check should still pass — the schema accepts both shapes).
- `go build ./...` and `go test ./...` clean.

## Changes

- `core/spec-types/feature.md` — authored `frontmatter:` block (3 required, 13 optional fields).
- `core/spec-types/bug.md` — authored `frontmatter:` block (4 required, 12 optional fields; severity required for bugs).
- `core/spec-types/chore.md` — authored `frontmatter:` block (3 required, 8 optional fields; chore lifecycle).
- `core/spec-types/epic.md` — authored `frontmatter:` block (3 required, 9 optional fields; kind=[theme, delivery, bet, milestone]).
- `core/spec-types/intake.md` — authored `frontmatter:` block (3 required, 7 optional fields; intake lifecycle).
- `core/spec-types/prd.md` — authored `frontmatter:` block (3 required, 9 optional fields; kind=[pitch, ten-section, lightweight]).
- `core/spec-types/release.md` — authored `frontmatter:` block (3 required, 6 optional fields; release lifecycle).
- `core/spec-types/sprint.md` — authored `frontmatter:` block (3 required, 6 optional fields; sprint lifecycle).
- `domains/engineering/spec-types/convention.md` — authored `frontmatter:` block (3 required, 6 optional fields; draft/active/superseded lifecycle).
- `domains/engineering/spec-types/decision.md` — authored `frontmatter:` block (3 required, 5 optional fields; proposed/accepted/superseded lifecycle).
- `internal/spectypes/loader_test.go` — added `TestLoad_FrontmatterSchema_PopulatedForCoreAndEngineering`, `TestLoad_FrontmatterFieldShape_FeatureStatus`; extended `TestExportTo_WritesCacheFile` to assert at least one record carries a populated frontmatter block.
- `.hero/cache/spec-types.json` — regenerated; 10 of 11 types now carry `frontmatter` blocks of 8-16 fields; `initiative` remains `null` (owned by parallel fix).

Loader-side parser was already in place — the gap was purely missing `frontmatter:` blocks in the source markdown.

## Kickoff

> Read `.hero/planning/bugs/spec-types-cache-frontmatter-empty/spec.md` (this file). Inspect `internal/spectypes/loader.go::parseRecord` (the existing per-block parsing pattern), `internal/spectypes/registry.go::FrontmatterSchema` (the target shape), `internal/spectypes/export.go::exportRecord` (the export wiring — confirm the `Frontmatter` field is already serialized), and one canonical source file like `core/spec-types/feature.md` (verify the `frontmatter:` block is present in YAML; if not, author it from the legacy lint validator's field set in `internal/triage/`). Implement the loader-side parser per the Fix section. Add unit tests per the Acceptance Criteria. Regenerate `.hero/cache/spec-types.json` and confirm `jq '.types[].frontmatter | length' .hero/cache/spec-types.json | sort -u` returns positive numbers (not `null`). Run `go build ./...` and `go test ./...` clean. Report what shipped, any type files that needed a frontmatter block authored, and any open questions under 400 words.
