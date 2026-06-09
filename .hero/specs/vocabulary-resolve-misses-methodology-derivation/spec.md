---
title: vocabulary.Resolve doesn't fold methodology-derived auto-derivation
slug: vocabulary-resolve-misses-methodology-derivation
type: bug
status: completed
severity: medium
priority: P1
created: 2026-05-17
tags: [vocabulary, methodology, resolver, contract, api]
relations:
  - target: hero-code-handover-pack
    kind: surfaced-by
  - target: pm-foundation-delivery
    kind: regression-of
completed_at: 2026-06-09T18:17:12Z
---

# vocabulary.Resolve doesn't fold methodology-derived auto-derivation

## Problem

The vocabulary resolver function `vocabulary.Resolve(cfg, vocabs)` documents and implements a precedence chain — explicit `cfg.Vocabulary` → tracker-inferred → delivery-preset-inferred → `default` — but it does **not** call `methodology.DeriveVocabularyName(cfg, m)`. Methodology-derived auto-derivation (Decision 6 step 3 from `unified-spec-type-model`: "vocabulary auto-derives from methodology's `aligned_vocabulary` field") is missing from the bare function entirely.

Every in-tree caller works around this by wrapping `Resolve` and injecting the derived name onto a `cfg` copy first:

- `internal/cli/vocab.go::activeVocab`
- `internal/serve/vocab.go::activeVocab`
- `internal/install/dialect.go::renderActiveDialectBlock`

Three call sites, three nearly identical wrapper shims. A fourth consumer (hero-code, or any future external caller of an exported resolver) that picks up `vocabulary.Resolve` directly will get a subtly wrong answer for workspaces that declare `methodology: scrum` alone — they'll see the `default` vocabulary instead of `agile-scrum`.

This was surfaced by the C3 agent (active-dialect doc work item, `hero-code-handover-pack`) while reading the resolver code to document the effective precedence chain for hero-code.

## Steps to Reproduce

```go
cfg := config.Config{Methodology: "scrum"} // vocabulary unset
methodologies, _ := methodology.Load(methodology.CoreFS(), nil)
vocabs, _ := vocabulary.Load(vocabulary.CoreFS(), nil)

bare := vocabulary.Resolve(cfg, vocabs)
// Returns: "default"  — WRONG (should be agile-scrum per Decision 6)

m, _ := methodology.Resolve(cfg, methodologies)
derived := methodology.DeriveVocabularyName(cfg, m) // returns "agile-scrum"
cfg.Vocabulary = derived
correct := vocabulary.Resolve(cfg, vocabs)
// Returns: "agile-scrum" — RIGHT
```

## Expected Behavior

`vocabulary.Resolve` should return the same answer that the in-tree wrappers do. The bare function name is a contract — callers reasonably expect it to be the single source of truth.

## Root Cause

`internal/vocabulary/resolver.go::Resolve` was written before `internal/methodology/` existed (B3 of pm-foundation-delivery landed after the vocabulary package). When methodology resolution shipped, the methodology-derived precedence step was added to the wrappers at the CLI layer, not folded into the resolver itself. No test pinned the bare resolver against the documented end-to-end precedence chain, so the divergence went unnoticed.

## Fix

Two viable approaches; pick one:

### Option A — fold derivation into `Resolve` (recommended)

Change the resolver signature to `Resolve(cfg, vocabs, methodologies)` and apply the methodology-derived step internally. Bump the second precedence step from "tracker-inferred" to "methodology-derived → tracker-inferred → …" so the chain reads as documented in `docs/contracts/active-dialect.md` §3.

- Pro: one function = one contract. External consumers can rely on the name.
- Con: signature change; in-tree callers must pass `methodologies`. Three call sites.

### Option B — rename to make scope loud

Keep current behavior; rename to `ResolveBare` (and/or add a `ResolveWithMethodology(cfg, vocabs, methodologies)` sibling that does the full chain). Mark `ResolveBare` as "step 1 + 3 + 4 only — for testing or when methodologies aren't available."

- Pro: no signature surprise; existing tests stay green untouched.
- Con: two-function API surface; consumers must read docs to pick the right one.

**Recommend Option A** because the bug report is "the bare function is misleading" — renaming doesn't make it less misleading, just more explicit about its sin. Folding the chain into one function is the cleaner contract.

## Acceptance Criteria

- THE SYSTEM SHALL apply methodology-derived vocabulary auto-derivation inside `vocabulary.Resolve` (Option A) such that `Resolve(cfg{Methodology:"scrum"}, vocabs)` returns `"agile-scrum"` without caller wrapping.
- THE SYSTEM SHALL update the three in-tree wrapper call sites (`internal/cli/vocab.go`, `internal/serve/vocab.go`, `internal/install/dialect.go`) to remove their derivation shims — direct `Resolve` use only.
- THE SYSTEM SHALL keep the documented precedence chain in `docs/contracts/active-dialect.md` §3 unchanged — only the implementation moves.
- WHEN `cfg.Methodology` is set and `cfg.Vocabulary` is empty, THE SYSTEM SHALL return the methodology's `aligned_vocabulary` from `Resolve`.
- WHEN both `cfg.Methodology` and `cfg.Vocabulary` are set, THE SYSTEM SHALL prefer the explicit `cfg.Vocabulary` (precedence step 1 still wins).
- WHEN neither is set, THE SYSTEM SHALL fall through to tracker-inferred → default per current behavior.

## Boundaries

- **Not** modifying `methodology.Resolve` or `methodology.DeriveVocabularyName`. The bug lives entirely in the vocabulary side.
- **Not** changing `docs/contracts/active-dialect.md` content — the doc already describes the correct effective chain.
- **Not** changing vocabulary YAML files or methodology profile YAML files.

## Validation

- Unit test in `internal/vocabulary/resolver_test.go`: `Resolve` with `Methodology:"scrum"` alone returns `"agile-scrum"`. Add similar coverage for shape-up → shape-up vocab, waterfall → default, kanban → kanban.
- Unit test asserting explicit `cfg.Vocabulary` still beats methodology-derived.
- Verify the three call-site shims have been removed by greping for `DeriveVocabularyName` — should only appear in the resolver itself and its tests after the fix.
- `go build ./...` and `go test ./...` clean.

## Changes

- `internal/vocabulary/resolver.go` — `Resolve` / `pickName` now take `methodologies map[string]*methodology.Methodology`; methodology-derived step inserted between explicit and tracker-inferred; doc comment updated to the full 5-step chain.
- `internal/cli/vocab.go` — removed methodology-derivation shim; calls new `Resolve(cfg, vocabs, methodologies)` directly.
- `internal/serve/vocab.go` — same shim removal.
- `internal/install/dialect.go` — same shim removal.
- `internal/vocabulary/resolver_test.go` — added 9 tests covering methodology-derived precedence (scrum/shape-up/kanban/waterfall), explicit-beats-methodology, methodology-beats-tracker, nil-methodologies fallback, unknown-methodology fallthrough, and tracker fallback when neither set.

## Kickoff

> Read `.hero/planning/bugs/vocabulary-resolve-misses-methodology-derivation/spec.md` (this file), `internal/vocabulary/resolver.go`, `internal/methodology/resolver.go::DeriveVocabularyName`, and the three wrapper call sites: `internal/cli/vocab.go`, `internal/serve/vocab.go`, `internal/install/dialect.go`. Implement Option A: extend `vocabulary.Resolve` to take a `methodologies map[string]*methodology.Methodology` (or accept a pre-resolved `*methodology.Methodology` — pick whichever has lower blast radius) and fold the methodology-derived step into the precedence chain between explicit and tracker-inferred. Remove the three wrapper shims and route them through the new bare `Resolve`. Add unit tests per the Acceptance Criteria. Run `go build ./...` and `go test ./...` clean. Report what shipped, the chosen signature, and any open questions under 300 words.

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| AC-1 | `Resolve(cfg{Methodology:"scrum"}, vocabs, methodologies)` returns `"agile-scrum"` | DONE | resolver.go:70-88 applies methodology-derived step; `TestResolve_MethodologyScrumDerivesAgileScrum` passes. |
| AC-2 | Three wrapper call sites updated (shims removed) | DONE | cli/vocab.go, serve/vocab.go, install/dialect.go all call `vocabulary.Resolve(cfg, vocabs, methodologies)` directly. `grep DeriveVocabularyName` only appears in resolver.go and resolver_test.go. |
| AC-3 | Documented precedence chain in active-dialect.md §3 unchanged | DONE | No changes to docs/contracts/active-dialect.md. Implementation matches the documented 5-step chain. |
| AC-4 | methodology-derived step applies when cfg.Methodology set and cfg.Vocabulary empty | DONE | `TestResolve_MethodologyScrumDerivesAgileScrum`, `TestResolve_MethodologyShapeUpDerivesShapeUp`, `TestResolve_MethodologyKanbanDerivesKanban` pass. |
| AC-5 | Explicit cfg.Vocabulary still wins over methodology-derived | DONE | `TestResolve_ExplicitVocabularyBeatsMethodologyDerived` passes. |
| AC-6 | Falls through to tracker-inferred → default when neither set | DONE | `TestResolve_TrackerFallbackWhenNoMethodologyOrVocab`, `TestResolve_EmptyConfigDefault` pass. |
| Changes: resolver.go extended | DONE | `Resolve`/`pickName` take `methodologies` param; methodology-derived step at resolver.go:75-88. |
| Changes: 3 call sites updated | DONE | See AC-2. |
| Changes: resolver_test.go 9+ new tests | DONE | 20 tests total including all 9 methodology-derived cases specified plus `TestResolve_NilMethodologiesSkipsDerivation`, `TestResolve_UnknownMethodologyFallsThrough`, `TestResolve_MethodologyDerivedBeatsTracker`. All pass. |

### Exercise-the-feature check

- [x] Exercised: `go test ./internal/vocabulary/... -v` → 20 tests pass including all 9 methodology-derived precedence tests. `grep -r DeriveVocabularyName internal/cli internal/serve internal/install` → no results (shims gone).
