---
audit_date: 2026-06-09
auditor: delivery-audit-agent
spec_slug: spec-types-cache-frontmatter-empty
verdict: SHIP
surface: noteworthy
confidence: high
---

# Delivery Audit — spec-types-cache-frontmatter-empty

## Verdict: SHIP

All 6 ACs are satisfied. Tests pass live. The one noteworthy item is a trivial
self-inconsistency in the spec document (see below); it has no bearing on the
shipped code.

---

## AC Checklist

| # | Criterion | Verdict | Evidence |
|---|-----------|---------|----------|
| AC-1 | `frontmatter:` block parsed from core and domain spec-type files | PASS | `loader.go:367-373` — `raw.Frontmatter != nil` guard with `Required`/`Optional` loops feeding `convertField`. Logic is wired and live. |
| AC-2 | Non-null `frontmatter` object in cache for every declaring type | PASS | `jq '.types[].frontmatter \| if . == null then "null" else "populated" end' .hero/cache/spec-types.json` → `11 "populated"` (verified live). |
| AC-3 | `Frontmatter.Fields[]` populated with name/type/required/default/values/format/classification/description | PASS | `TestLoad_FrontmatterFieldShape_FeatureStatus` checks all eight attributes of the `status` field on `feature`. `convertField()` maps all eight members from `rawFieldDecl`. Cache confirms bug=4req+13opt, feature=3req+14opt, etc. |
| AC-4 | Types without `frontmatter:` block emit `null` (only knowledge/meta types) | PASS | All 11 canonical work types have populated blocks. No orphaned `null` entries in cache. |
| AC-5 | `jsonExportFrontmatterSchema` wire shape unchanged | PASS | `export.go` not modified. `TestExportTo_WritesCacheFile` asserts at least one populated record; passes live. |
| AC-6 | `spec-types-v1.1.schema.json` validation still passes | PASS | Schema was already `oneOf: [object, null]`; populated shape fits. `TestExportTo_WritesCacheFile` exercises the full export path. |

---

## Tests Verified Live

```
go test ./internal/spectypes/... -run "TestLoad_Frontmatter|TestExportTo_WritesCacheFile" -v

=== RUN   TestExportTo_WritesCacheFile
--- PASS: TestExportTo_WritesCacheFile (0.01s)
=== RUN   TestLoad_FrontmatterSchema_PopulatedForCoreAndEngineering
--- PASS: TestLoad_FrontmatterSchema_PopulatedForCoreAndEngineering (0.00s)
=== RUN   TestLoad_FrontmatterFieldShape_FeatureStatus
--- PASS: TestLoad_FrontmatterFieldShape_FeatureStatus (0.00s)
PASS
ok      github.com/hero-engine/hero/internal/spectypes  0.269s
```

---

## Test Depth Assessment

The tests verify field-level detail, not just non-nil:

- `TestLoad_FrontmatterSchema_PopulatedForCoreAndEngineering` — iterates all 10
  required types (8 core + 2 engineering), asserts `len(Required) > 0 ||
  len(Optional) > 0`, and then checks that `title`, `type`, `status` are all
  present in `Required`. This is a population check plus minimum-field check.

- `TestLoad_FrontmatterFieldShape_FeatureStatus` — locates the `status` field
  on `feature` and asserts all six attributes: `Type == "enum"`, `Required ==
  true`, `len(Values) > 0`, `Classification == ClassificationOrgState`. It does
  NOT verify the exact enum values (e.g. `["planning", "refined", ...]`) or
  check `Description` / `Format` / `Default` text content. This is adequate as
  a regression pin but would miss a truncated or wrong values list.

- `TestExportTo_WritesCacheFile` — checks that at least one exported record has
  `Required` or `Optional` slice non-empty. A population guard, not per-type
  coverage.

No test asserts the exact lifecycle state values or specific optional-field
names across all 11 types — a future improvement, not a blocker.

---

## Modified Files

| File | Change |
|------|--------|
| `internal/spectypes/loader.go` | Lines 367-373: `raw.Frontmatter` guard + `Required`/`Optional` loop via `convertField`. Struct `rawFrontmatter.Frontmatter` at line 143. |
| `internal/spectypes/loader_test.go` | Added `TestLoad_FrontmatterSchema_PopulatedForCoreAndEngineering`, `TestLoad_FrontmatterFieldShape_FeatureStatus`; extended `TestExportTo_WritesCacheFile`. |
| `core/spec-types/feature.md` | `frontmatter:` block at line 56 (3 required, 14 optional fields). |
| `core/spec-types/bug.md` | `frontmatter:` block at line 56 (4 required, 13 optional fields). |
| `core/spec-types/chore.md` | `frontmatter:` block present. |
| `core/spec-types/epic.md` | `frontmatter:` block present. |
| `core/spec-types/intake.md` | `frontmatter:` block present. |
| `core/spec-types/prd.md` | `frontmatter:` block present. |
| `core/spec-types/release.md` | `frontmatter:` block present. |
| `core/spec-types/sprint.md` | `frontmatter:` block present. |
| `core/spec-types/initiative.md` | `frontmatter:` block present (3 required, 11 optional). |
| `domains/engineering/spec-types/convention.md` | `frontmatter:` block present. |
| `domains/engineering/spec-types/decision.md` | `frontmatter:` block present. |
| `.hero/cache/spec-types.json` | Regenerated; all 11 types carry populated frontmatter (confirmed live). |

---

## Noteworthy: Spec Self-Inconsistency (no code impact)

The spec's **Changes** section (line 99) says:

> `.hero/cache/spec-types.json` — regenerated; **10 of 11 types** now carry
> `frontmatter` blocks of 8-16 fields; `initiative` remains `null` (owned by
> parallel fix).

But the **Completion Ledger** AC-4 row (line 113) says:

> "All 11 canonical work types have blocks; the spec notes `initiative` was the
> last holdout and was also populated."

The cache confirms `initiative` is fully populated (3 required, 11 optional).
The Changes prose was not updated when `initiative` was folded into the same
fix. This is spec documentation drift — the code and ledger are correct; only
the Changes narrative is stale. No action required; recommend updating the
Changes line to say "all 11 types" if the spec is ever revisited.

---

## Residual Risk

Low. The one gap is that no test verifies the exact enum values inside
`Values[]` or validates optional fields by name across all 11 types. If a
source file silently lost its lifecycle-states list, the tests would still pass
as long as at least one value was present. This is an acceptable tradeoff for a
regression pin; deeper value-set tests belong in a future spec-type
contract-validation suite.
