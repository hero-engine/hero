---
title: Document vocabulary auto_select rule schema — fields, allowed values, and authoring guide
type: feature
status: planning
priority: P2
created: 2026-05-17
tags: [docs, contracts, vocabulary, auto-select, consumer]
relations:
  - target: hero-code-handover-pack
    kind: surfaced-by
  - target: pm-foundation-delivery
    kind: follow-up-from
---

# Document vocabulary auto_select rule schema — fields, allowed values, and authoring guide

## Context

The `core/vocabularies/*.yaml` schema includes an `auto_select:` block that lets a vocabulary preset declare conditions under which it should be inferred as the active vocabulary — primarily by tracker type and delivery preset. The resolver in `internal/vocabulary/resolver.go` reads this block and uses it during precedence step 3 (tracker-inferred) and step 4 (delivery-preset-inferred).

The block is **referenced** in `docs/contracts/active-dialect.md` §3 and in the `core/vocabularies/agile-scrum.yaml` example file (which shows a single `delivery_preset:` rule), but it is **not exhaustively documented**. A consumer authoring a new vocabulary preset has to crack the Go source to discover:

- The full set of supported rule keys (`tracker`, `delivery_preset`, anything else?)
- Allowed values per rule (which trackers are recognized? which delivery presets?)
- Match semantics (all-rules-must-match? any-rule-matches? rule ordering?)
- Conflict resolution (two vocabs both auto_select on the same condition — which wins?)
- Whether/how to declare "never auto-select" (default fallback behavior)

Surfaced by the C3 agent (active-dialect doc work item, `hero-code-handover-pack`).

## Goal

Ship `docs/contracts/vocabulary-auto-select.md` — a focused doc covering the auto_select rule shape, allowed values, match semantics, and a worked authoring example. Aimed at anyone authoring a new vocabulary preset (whether for a custom methodology, a tracker integration, or a domain-specific dialect).

Companion update: cross-link from `docs/contracts/active-dialect.md` §3 and `docs/contracts/README.md` so the doc is discoverable.

## Design

Doc structure:

### §1. What auto_select does
One paragraph: vocabulary auto_select rules fire during precedence step 3 (tracker-inferred) and step 4 (delivery-preset-inferred) of the resolver chain. They let a preset opt in to being chosen automatically based on workspace signals when no explicit `vocabulary:` is declared.

### §2. Rule shape
The complete YAML schema, authoritatively derived from `internal/vocabulary/vocabulary.go`'s `AutoSelectRule` (or equivalent) struct. List every supported field with type and meaning. Lock the allowed-value set per field — e.g. `tracker:` values are `jira | github | linear | none` (whatever the loader actually accepts).

### §3. Match semantics
- Multiple rules within one preset: documented combinator (AND vs OR — confirm from the resolver).
- Multiple presets auto_select on the same workspace: documented tie-break (first match wins by file load order? specificity-based? deterministic alphabetical?).
- The implicit "never auto-select" case: omit the `auto_select:` block entirely.

### §4. Worked authoring example
Walk through authoring a hypothetical `team-foo.yaml` preset that auto-selects when a workspace uses GitHub Issues with `delivery_preset: cycle`. Show the YAML, then `hero status` output for a workspace matching the rule.

### §5. Authoring checklist
Quick checks before merging a new preset:
- Auto_select rules don't collide with existing presets in obvious ways (run `hero status` in a representative workspace, confirm the expected preset wins).
- Display map covers every canonical type the preset is expected to render.
- `aligned_methodology:` reverse-pointer set if the preset has a default methodology pairing.

### §6. Pointers to source
- Loader / resolver code in `internal/vocabulary/`.
- Existing examples: `core/vocabularies/agile-scrum.yaml` (delivery_preset rule), `core/vocabularies/jira.yaml` if/when it gains an auto_select block (currently has none — note).

## Acceptance Criteria

- THE SYSTEM SHALL ship `docs/contracts/vocabulary-auto-select.md` with the six sections above.
- THE SYSTEM SHALL derive the rule shape and allowed-value sets authoritatively from `internal/vocabulary/`'s Go source — no documented field that doesn't exist in code; no code-supported field omitted from docs.
- THE SYSTEM SHALL cross-link from `docs/contracts/active-dialect.md` §3 ("for the full auto_select schema, see vocabulary-auto-select.md") and add the doc to the table in `docs/contracts/README.md`.
- THE SYSTEM SHALL include at least one worked authoring example with a complete YAML preset that would auto_select under stated conditions.
- WHEN match semantics depend on file load order or other implementation details, THE SYSTEM SHALL document the dependency explicitly rather than imply deterministic-by-magic.
- THE SYSTEM SHALL keep the doc under 250 lines.

## Boundaries

- **Not** changing any Go code or YAML files. Pure doc work.
- **Not** documenting methodology auto_select. The methodology resolver uses different precedence and doesn't read an auto_select block per profile today (precedence is hardcoded: Jira → scrum heuristic). If methodology auto_select grows YAML-driven rules later, that's a separate doc.
- **Not** authoring a new vocabulary preset. The worked example in §4 is illustrative only — no `core/vocabularies/team-foo.yaml` ships from this work.

## Validation

- Author the doc; verify every field name and allowed value against `grep`-confirmed Go source.
- Spot-check by handing the doc to a fresh reader (or fresh agent) and asking "author a vocabulary preset that auto-selects when the workspace uses Linear with delivery_preset 'continuous'." If they can do it from the doc alone, it's sufficient. If they have to read Go, it isn't.
- `go build ./...` and `go test ./...` unchanged (no code touched).

## Kickoff

> Read `.hero/planning/features/document-vocabulary-auto-select-schema/spec.md` (this file). Inspect `internal/vocabulary/vocabulary.go` and `internal/vocabulary/resolver.go` for the auto_select rule struct shape, allowed-value set, and match semantics — these are the authoritative source. Check existing presets: `core/vocabularies/agile-scrum.yaml`, `core/vocabularies/kanban.yaml`, `core/vocabularies/shape-up.yaml`, `core/vocabularies/default.yaml`, `core/vocabularies/jira.yaml`, `core/vocabularies/linear.yaml` for what auto_select blocks already exist. Author `docs/contracts/vocabulary-auto-select.md` per the six sections in this spec's Design. Update `docs/contracts/active-dialect.md` §3 with a cross-link and add a row to `docs/contracts/README.md`'s table. Keep the doc under 250 lines. Run `go build ./...` (should be unchanged). Report what shipped, the auto_select rule fields you documented, and any code-vs-prose discrepancies you spotted, under 250 words.
