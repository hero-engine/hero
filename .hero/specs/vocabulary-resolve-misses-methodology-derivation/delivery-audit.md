---
audit_date: 2026-06-09
auditor: delivery-audit-agent
spec: vocabulary-resolve-misses-methodology-derivation
verdict: SHIP
surface: clean
confidence: high
---

# Delivery Audit — vocabulary-resolve-misses-methodology-derivation

## Summary

6 of 6 ACs verified. 3 Changes items verified. Build and all 20+ tests pass. No shims remain in the three call sites. Surface is clean — SHIP.

## AC Verification

| # | Acceptance Criterion | Verdict | Evidence |
|---|---|---|---|
| AC-1 | `Resolve(cfg{Methodology:"scrum"}, vocabs, methodologies)` returns `"agile-scrum"` | PASS | `resolver.go:80-88` — pickName applies methodology-derived step; `TestResolve_MethodologyScrumDerivesAgileScrum` passes. |
| AC-2 | Three wrapper call sites updated (shims removed) | PASS | `grep -r DeriveVocabularyName internal/cli internal/serve internal/install` → exit 1, no output. All three files call `vocabulary.Resolve(cfg, vocabs, methodologies)` directly with no derivation shim. |
| AC-3 | `docs/contracts/active-dialect.md` §3 unchanged | PASS | File not touched; doc comment in `resolver.go` now documents the full 5-step chain in code, matching the contract. |
| AC-4 | Methodology-derived step fires when `cfg.Methodology` set and `cfg.Vocabulary` empty | PASS | `TestResolve_MethodologyScrumDerivesAgileScrum`, `TestResolve_MethodologyShapeUpDerivesShapeUp`, `TestResolve_MethodologyKanbanDerivesKanban` all pass. |
| AC-5 | Explicit `cfg.Vocabulary` still beats methodology-derived | PASS | `TestResolve_ExplicitVocabularyBeatsMethodologyDerived` passes. |
| AC-6 | Falls through to tracker-inferred → default when neither set | PASS | `TestResolve_TrackerFallbackWhenNoMethodologyOrVocab` and `TestResolve_EmptyConfigDefault` pass. |

## Changes Verification

### `internal/vocabulary/resolver.go`

`Resolve` and `pickName` both accept `methodologies map[string]*methodology.Methodology` as a third parameter. Step 2 of `pickName` (lines 75–88) applies `methodology.DeriveVocabularyName(cfg, m)` between explicit and tracker-inferred, skipping gracefully when `methodologies` is nil or the methodology key is not found. The function doc comment documents all five steps of the precedence chain explicitly.

### `internal/cli/vocab.go`

`activeVocab` loads methodologies via `loadMethodologiesCached()` and passes the result directly to `vocabulary.Resolve(cfg, vocabs, methodologies)`. No `DeriveVocabularyName` call present. Shim removed.

### `internal/serve/vocab.go`

`activeVocab` loads methodologies via `methodology.Load(methodology.CoreFS(), nil)` and passes the result to `vocabulary.Resolve(cfg, vocabs, methodologies)`. No `DeriveVocabularyName` call present. Shim removed.

### `internal/install/dialect.go`

`renderActiveDialectBlock` loads both `vocabs` and `methodologies`, then calls `vocabulary.Resolve(&cfg, vocabs, methodologies)`. No `DeriveVocabularyName` call present. Shim removed.

### `internal/vocabulary/resolver_test.go`

21 tests total (11 pre-existing, 10 new). New methodology-derived tests cover: scrum→agile-scrum, shape-up→shape-up, kanban→kanban, waterfall→default (falls through), explicit-beats-methodology, tracker-fallback-when-no-methodology, methodology-derived-beats-tracker, nil-methodologies-skips-derivation, unknown-methodology-falls-through, and DoesNotMutateBase (pre-existing but exercises the full clone path). All 21 pass.

## Build and Test Results

```
go build ./...       → exit 0 (clean)
go test ./internal/vocabulary/... -v → 21 PASS, 0 FAIL
```

`grep -r DeriveVocabularyName internal/cli internal/serve internal/install` → no output (exit 1), confirming all three shims are gone. `DeriveVocabularyName` exists only in `internal/methodology/resolver.go` (definition) and `internal/methodology/resolver_test.go` (tests of the methodology package itself) plus the single call site now inside `internal/vocabulary/resolver.go:82`.

## Observations

No issues found. The implementation is clean:

- `nil` methodologies is handled gracefully (step 2 skips; falls through to tracker/delivery/default). This is a good nil-safe design choice — callers in tight paths can pass `nil` without risk.
- The `vocabs[derived]` existence check at `resolver.go:83` prevents a derived name from resolving to a missing vocabulary — it falls through to the next chain step instead of an error. This is correct and matches the documented behavior.
- The waterfall case (`TestResolve_MethodologyWaterfallDerivesDefault`) tests that a methodology whose `aligned_vocabulary` is `"default"` returns the default vocab — the test confirms this resolves to `"default"`, which is present in `vocabs`.
- No spec-document updates are needed; the contract doc was correct before the fix.

## Verdict

**SHIP** — all 6 ACs pass, all 3 Changes items verified against code, build is clean, test suite is comprehensive and green. No shims remain. The contract is now fulfilled by the bare `vocabulary.Resolve` function as documented.
