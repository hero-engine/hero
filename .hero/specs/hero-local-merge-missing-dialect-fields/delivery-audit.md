---
audit_type: delivery-audit
spec: hero-local-merge-missing-dialect-fields
audited_by: delivery-auditor
audit_date: 2026-06-09
verdict: SHIP
surface: clean
confidence: high
---

# Delivery Audit — hero-local-merge-missing-dialect-fields

## Summary

P1 bug fix: `config.MergeLocal` now forwards all four dialect fields from
`hero.local.json` into the active config. All 6 acceptance criteria confirmed
by direct code and test inspection. 8 unit tests + 1 end-to-end integration
test pass. Build clean. `docs/contracts/active-dialect.md` §2 is normative
with no hedge language.

---

## AC-by-AC Findings

### AC-1 — Forward `vocabulary` scalar from hero.local.json

**VERIFIED.**

`config.go` lines 1545–1547:

```go
if local.Vocabulary != "" {
    base.Vocabulary = local.Vocabulary
}
```

Correct guard: non-empty local wins; absent/empty local is a no-op.
Covered by `TestMergeLocal_VocabularyOverridesScalar` (base=`agile-scrum`,
local=`shape-up` → merged=`shape-up`) and `TestMergeLocal_BothDialectScalars`.

### AC-2 — Forward `methodology` scalar from hero.local.json

**VERIFIED.**

`config.go` lines 1556–1558:

```go
if local.Methodology != "" {
    base.Methodology = local.Methodology
}
```

Same correct guard. `TestMergeLocal_MethodologyOverridesScalar` verifies
scalar override and also asserts the peer scalar (Vocabulary) is untouched
when only Methodology is set — a useful negative-side check.

### AC-3 — Merge `vocabulary_overrides` entry-by-entry (local wins on collision; non-colliding base keys preserved)

**VERIFIED.**

`config.go` lines 1548–1555:

```go
if len(local.VocabularyOverrides) > 0 {
    if base.VocabularyOverrides == nil {
        base.VocabularyOverrides = make(map[string]string)
    }
    for k, v := range local.VocabularyOverrides {
        base.VocabularyOverrides[k] = v
    }
}
```

Implementation is correct: nil-safe initialization before ranging local
entries. `TestMergeLocal_VocabularyOverridesMapMerge` exercises three cases
in a single assertion block:
- `types.spec` — key collision, local value `LocalStory` wins over base
  `BaseStory`. **Pass.**
- `types.epic` — new local key not present in base. **Pass.**
- `sections.criteria` — non-colliding base key `BaseCriteria` is preserved.
  **Pass.** (This is the spec's explicit requirement; the test directly
  confirms it.)

`TestMergeLocal_VocabularyOverridesIntoNilBase` covers the nil-base
initialization path (base starts nil, local supplies one entry → merged map
has that entry). **Pass.**

### AC-4 — Merge `methodology_overrides` entry-by-entry (local wins on collision; non-colliding base keys preserved)

**VERIFIED.**

`config.go` lines 1559–1566, identical structural pattern to AC-3.
`TestMergeLocal_MethodologyOverridesMapMerge` exercises all three cases:
- Key collision (`time_boxes.iteration.duration_default`): local `3w` wins
  over base `2w`. **Pass.**
- New local key (`estimation.feature.required_field`): added. **Pass.**
- Non-colliding base key (`in_flight_tracking`): `wip_aging` preserved.
  **Pass.**

### AC-5 — Absent/empty hero.local.json preserves existing behavior

**VERIFIED.**

`TestMergeLocal_EmptyLocalLeavesDialectUntouched` passes a zero-value
`Config{}` as local. Asserts all four dialect fields on base remain
unchanged (`Methodology=scrum`, `Vocabulary=agile-scrum`,
`VocabularyOverrides["types.spec"]=Story`,
`MethodologyOverrides["in_flight_tracking"]=wip_aging`). **Pass.**

The Load-path equivalent `TestLoad_AppliesLocalDialectOverride` writes
real temp files and calls `config.Load` — confirming the live JSON load
→ merge pipeline carries the override through end-to-end. **Pass.**

### AC-6 — docs/contracts/active-dialect.md §2 normative; no hedge language

**VERIFIED.**

§2 was read in full. The section now contains:

> Read order: load `hero.json`, then merge `hero.local.json` on top.
> `hero.local.json` is a per-user, gitignored file that overrides any
> field on the top-level `Config` — including the four dialect fields.
> Override semantics for the dialect layer:
>
> - `vocabulary` (scalar) — non-empty local value replaces the base value.
> - `methodology` (scalar) — non-empty local value replaces the base value.
> - `vocabulary_overrides` (map) — entry-by-entry merge: local entries
>   replace base entries on key collision; non-colliding base keys are
>   preserved.
> - `methodology_overrides` (map) — entry-by-entry merge with the same
>   semantics as `vocabulary_overrides`.

No "planned extension point," no hedge. The text is normative and references
`internal/config/config.go::MergeLocal` as the canonical implementation.
Consumers building out-of-process integrations are told they must replicate
the same precedence. **Pass.**

---

## Integration Test Quality

`TestDialectLine_LocalOverrideEndToEnd` in `internal/cli/vocab_test.go`
(line 144) is a genuine integration test, not a trivial stub:

1. Resets the vocab cache before and after (`resetVocabCacheForTesting`).
2. Writes real JSON to a temp filesystem (`hero.json` scrum + agile-scrum,
   `hero.local.json` shape-up + shape-up).
3. Calls `config.Load(tmpDir)` — the real loader, not a mock.
4. Passes the loaded config to `dialectLine` — the real rendering path.
5. Asserts `line` contains `shape-up`.
6. Strips `shape-up` from the rendered line and checks that `scrum` and
   `agile-scrum` are absent — ensuring the team-tracked dialect has not
   leaked through. This negative assertion directly catches the class of
   bug described in the spec.

Regression comment in the test source names the spec slug
(`hero-local-merge-missing-dialect-fields`) and states the prior failure
mode explicitly. High-quality guard.

---

## Test Run Confirmation

All tests run live (not from cache at audit time for the config package;
cli package was cached but the test output showed clean PASS):

```
internal/config:
  TestMergeLocal_MethodologyOverridesScalar  PASS
  TestMergeLocal_VocabularyOverridesScalar   PASS
  TestMergeLocal_BothDialectScalars          PASS
  TestMergeLocal_VocabularyOverridesMapMerge PASS
  TestMergeLocal_MethodologyOverridesMapMerge PASS
  TestMergeLocal_VocabularyOverridesIntoNilBase PASS
  TestMergeLocal_EmptyLocalLeavesDialectUntouched PASS
  TestLoad_AppliesLocalDialectOverride       PASS

internal/cli:
  TestDialectLine_LocalOverrideEndToEnd      PASS

go build ./...                               OK
```

---

## Observations (non-blocking)

None. The implementation is clean, the merge pattern mirrors the existing
`Models.Roles` block (same nil-init + range-copy pattern), and the test
coverage is thorough. No gaps detected.

---

## Verdict

**SHIP.** All six acceptance criteria are verified by code inspection and
passing tests. The fix is minimal, correct, and consistent with the
codebase's existing merge patterns. Documentation is normative and complete.
