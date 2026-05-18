---
title: hero.local.json merge doesn't forward vocabulary / methodology fields
slug: hero-local-merge-missing-dialect-fields
type: bug
status: delivering
severity: medium
priority: P1
created: 2026-05-17
tags: [config, hero-local, vocabulary, methodology, dialect, merge]
relations:
  - target: hero-code-handover-pack
    kind: surfaced-by
  - target: pm-foundation-delivery
    kind: regression-of
---

# hero.local.json merge doesn't forward vocabulary / methodology fields

## Problem

`hero.local.json` lets an individual developer override workspace config without committing personal preferences to the project's tracked `hero.json`. The merge function `config.MergeLocal` is the canonical mechanism for layering local overrides on top of the project config. B3 (`internal/methodology/`) and the existing `internal/vocabulary/` package both expose four config fields — `Vocabulary`, `VocabularyOverrides`, `Methodology`, `MethodologyOverrides` — but `MergeLocal` does **not** forward any of them from `hero.local.json` into the active config.

Symptom: a developer who wants to render their personal copy of the workspace as Shape Up (`methodology: shape-up`) while the team's `hero.json` declares Scrum cannot do so via `hero.local.json`. The override is silently dropped at config-load time.

Surfaced by the C3 agent (active-dialect doc work item, `hero-code-handover-pack`) while documenting the read path for active dialect. The doc honestly noted this as a "planned extension point" — but the C-track sprint's discoverability work shouldn't have to paper over a missing config wire. Bug.

## Steps to Reproduce

```bash
# Team workspace declares scrum + agile-scrum:
cat hero.json | jq '.methodology, .vocabulary'
# "scrum"
# "agile-scrum"

# Developer overrides locally to shape-up:
echo '{"methodology": "shape-up", "vocabulary": "shape-up"}' > hero.local.json

hero status
# Expected: "Vocabulary: shape-up · Methodology: shape-up"
# Actual:   "Vocabulary: agile-scrum · Methodology: scrum"   (local overrides ignored)
```

## Expected Behavior

`hero.local.json` overrides for `vocabulary`, `vocabulary_overrides`, `methodology`, and `methodology_overrides` should take precedence over the tracked `hero.json` values exactly as they do for `domain`, `tracker`, and other already-merged fields.

## Root Cause

`internal/config/config.go::MergeLocal` (or equivalent merge function — name may differ) was authored before the dialect fields existed. When B3 added `Methodology` / `MethodologyOverrides` and the vocabulary-aware rendering of B6 went live, the merge function was not extended. There's no per-field forwarding for the four new keys, so a `hero.local.json` setting them is parsed (into a separate `Config` struct) and then discarded.

## Fix

Extend `MergeLocal` (or the equivalent merger) to forward all four dialect fields from the local layer:

- `local.Vocabulary` non-empty → overwrites `base.Vocabulary`.
- `local.VocabularyOverrides` non-empty → merges entry-by-entry into `base.VocabularyOverrides` (local entries replace base entries on key collision).
- `local.Methodology` non-empty → overwrites `base.Methodology`.
- `local.MethodologyOverrides` non-empty → merges entry-by-entry into `base.MethodologyOverrides`.

The merge semantics mirror how the function already handles the `Tracker` block and other nested config — copy the pattern, don't invent a new one.

## Acceptance Criteria

- THE SYSTEM SHALL forward `vocabulary` from `hero.local.json` into the active config when present.
- THE SYSTEM SHALL forward `methodology` from `hero.local.json` into the active config when present.
- THE SYSTEM SHALL merge `vocabulary_overrides` map entries from `hero.local.json` into the base map (local entries win on collision).
- THE SYSTEM SHALL merge `methodology_overrides` map entries from `hero.local.json` into the base map (local entries win on collision).
- WHEN `hero.local.json` is absent OR doesn't declare these fields, THE SYSTEM SHALL preserve existing behavior — no behavior change for any current workspace.
- THE SYSTEM SHALL update `docs/contracts/active-dialect.md` §2 to drop the "planned extension point" note once the merge ships, replacing it with normative documentation of `hero.local.json` override behavior.

## Boundaries

- **Not** changing how `hero.local.json` itself is discovered or loaded. The bug is in the merge step, not the load step.
- **Not** changing the resolver chain in `internal/vocabulary/` or `internal/methodology/`. They consume the already-merged config; this fix lives one layer up.
- **Not** introducing a new local-override file format. Use what's already documented.

## Validation

- Unit test in `internal/config/config_test.go`: load a base config with `methodology: scrum, vocabulary: agile-scrum`, merge a local config with `methodology: shape-up`, assert the result has `Methodology == "shape-up"` and `Vocabulary == "agile-scrum"` (only methodology overridden, vocab untouched).
- Unit test: merge with both `methodology` and `vocabulary` in local — both override.
- Unit test: `vocabulary_overrides` map merge with key collision — local key wins, non-colliding base keys preserved.
- Integration test (CLI): set up a temp workspace with the team `hero.json` declaring scrum, then `hero.local.json` declaring shape-up. Run `hero status` and assert the active-dialect header reads `shape-up`.
- `go build ./...` and `go test ./...` clean.

## Kickoff

> Read `.hero/planning/bugs/hero-local-merge-missing-dialect-fields/spec.md` (this file), `internal/config/config.go::MergeLocal` (or whatever the local-merge function is named there — `grep -n "MergeLocal\|LoadLocal\|merge" internal/config/config.go`), and one neighboring already-merged nested block (e.g. `Tracker`) for the merge pattern to copy. Extend the merger to forward the four dialect fields per the Fix section. Add unit tests per the Acceptance Criteria. Update `docs/contracts/active-dialect.md` §2 to make the `hero.local.json` override behavior normative (replacing the "planned extension point" note). Run `go build ./...` and `go test ./...` clean. Report what shipped, the exact functions touched, and any open questions under 300 words.

## Changes

- `internal/config/config.go` — extend `MergeLocal` to forward the four dialect fields from local: `Vocabulary` and `Methodology` as scalar local-wins overrides; `VocabularyOverrides` and `MethodologyOverrides` as entry-by-entry map merges (local entries replace base on key collision; non-colliding base keys preserved).
- `internal/config/config_test.go` — add seven dialect-merge tests (`TestMergeLocal_MethodologyOverridesScalar`, `TestMergeLocal_VocabularyOverridesScalar`, `TestMergeLocal_BothDialectScalars`, `TestMergeLocal_VocabularyOverridesMapMerge`, `TestMergeLocal_MethodologyOverridesMapMerge`, `TestMergeLocal_VocabularyOverridesIntoNilBase`, `TestMergeLocal_EmptyLocalLeavesDialectUntouched`) plus a `Load`-path test (`TestLoad_AppliesLocalDialectOverride`).
- `internal/cli/vocab_test.go` — add `TestDialectLine_LocalOverrideEndToEnd` integration test: write team `hero.json` (scrum + agile-scrum), drop `hero.local.json` declaring shape-up, run `config.Load`, assert `dialectLine` reports shape-up with no team-dialect leakage.
- `docs/contracts/active-dialect.md` — replace §2 "planned extension point" hedge with normative override semantics (scalar local-wins, map entry-by-entry merge with local-wins-on-collision), referencing `internal/config/config.go::MergeLocal`.
