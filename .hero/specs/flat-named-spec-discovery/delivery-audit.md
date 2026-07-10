# Delivery audit — flat-named-spec-discovery

**Audited:** working tree at HEAD (cold read of on-disk code + tests, not the ledger)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1 flat `<slug>.md` with explicit work type discovered — `internal/spec/spec.go:1204` (flat branch) + `isDiscoverableFlatSpec` `spec.go:1248`; `TestDiscoverFlatNamedSpec` (`spec_test.go:636`) asserts f-01 (feature), f-02 (bug), and slug-less `loose-feature` are discovered with correct types.
- [✓] AC-2 untyped artifacts + knowledge entries excluded — explicit-type gate + `nonWorkFlatTypes` map `spec.go:1229`; `TestDiscoverIgnoresNonSpecFlatFiles` (`spec_test.go:685`) plants an untyped audit, `next/alice.md`, `type: decision`, `type: convention`, `mission`, `retro` and asserts `len(discovered)==1` (only the initiative).
- [✓] AC-3 ResolveOrHint returns the spec, not the "not designed yet" hint — `resolve_hint.go:26-30` exact-match loop runs at step 1, before initiative-child detection at step 3 (`:40-54`). Once discovery loads the flat child (AC-1) it is present in `specs` and resolves at step 1. `TestResolveOrHint` "exact match" case (`resolve_hint_test.go:43`) confirms step 1 returns the spec with empty hint. See Audit notes — the exact misfire scenario is not directly reproduced by a test.
- [✓] AC-4 flat-file archive moves only the file, siblings untouched — `complete.go:363` (`flat` detection), `:369-375` (slug from parsed spec), `:414-422` (skip `moveSiblingArtifacts`/`removeEmptyParents`); `TestMoveToSpecs_FlatFile` (`complete_test.go:560`) asserts child lands at `specs/f-15-buffer-pool/spec.md`, initiative + sibling remain in planning/, child gone from source, and NOT mis-archived under the initiative slug.
- [✓] AC-5 build + spec/cli tests pass — `go build ./...` exit 0; `go test ./internal/spec/... ./internal/cli/...` both `ok`.

## Changes
- [✓] `internal/spec/spec.go`: flat-file discovery branch + `isDiscoverableFlatSpec` helper — present (`spec.go:1204`, `:1248`), plus `frontmatterType` parser (`:1263`) and `nonWorkFlatTypes` gate (`:1229`).
- [✓] `internal/cli/complete.go`: `moveToSpecs` flat-file archive branch — present (`complete.go:358-422`).
- [✓] Tests for discovery, exclusion, and flat-file archive — all three present and non-vacuous.

## Audit notes
- **AC-3 has no direct regression guard for the misfire.** AC-3's whole point is that a slug which is *both* a discovered flat child *and* listed in an initiative's children table must resolve to the spec rather than emit the "hasn't been designed yet" hint. The code is correct — exact match (step 1) precedes initiative-child detection (step 3), and the ordering is documented at `resolve_hint.go:14-23`. But no test plants that co-occurrence (spec present AND initiative lists it). The existing "exact match" test proves step 1 in isolation; the "unmaterialized child" / "initiative-child beats fuzzy" tests only cover the case where the child is NOT a discovered spec. If someone reordered steps 1 and 3, every current test would still pass while the original bug regressed. Behavior is delivered and correct; the defense-in-depth test is missing.
- **Ledger test-name typo.** Ledger row AC-4 names `TestMoveToSpecsFlatFile`; the actual test is `TestMoveToSpecs_FlatFile` (`complete_test.go:560`). Cosmetic — the test exists and covers the claim.
- No scope drift: changes are confined to the two spec-named files plus their test files.
