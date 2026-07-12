# Delivery audit — next-context-carry-forward-drift

**Audited:** `git diff 9ea4737 -- internal/projection/projection.go internal/projection/projection_test.go`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1 (gate passes on correctly-committed repo) — projection now deterministic from committed fields; `.hero/NEXT.md` regenerated in working tree. Real CI-green acceptance is post-push (AC-7), not observable on disk.
- [✓] AC-2 (`## Context to carry forward` orders on committed fields only) — `projection.go:310-312` now `priority ASC, created DESC, key ASC`; no `ingested_at` in this query.
- [✓] AC-3 (`## Next` tie-breaks on committed fields only) — `projection.go:202-204` same ordering; `ingested_at` removed from ORDER BY (still SELECTed at line 198 but only scanned into an unused local, never affects output/order).
- [✓] AC-4 (byte-identical across session-touched vs clean graph) — `TestNextMD_CarryForward_DeterministicAcrossIngestOrder` and `TestNextMD_Next_TieBreakDeterministic` project the same committed nodes from a clustered-`ingested_at` graph and a reverse-inserted graph with bumped `ingested_at`, asserting the section is byte-identical. Both PASS fresh (`-count=1`).
- [✓] AC-5 (stale/hook-less commit still fails gate — detection preserved) — preserved by construction: the byte-exact gate is untouched; the change only reorders correctly-projected content, it does not fuzzy-match. A stale managed region still diverges on stable content. No new test added (spec's Change 4 was verify-only).
- [✓] AC-6 (carry-forward stays populated — handoff magic) — regenerated `.hero/NEXT.md` shows 5 real Decisions/Initiatives in `## Context to carry forward` and a real P0 pick (`core-vertical-layering`) in `## Next`. Tests also assert the section is non-empty and contains pinned keys.
- [✓] AC-7 (`go vet/build/test` pass; Test workflow green) — `go build ./cmd/hero` OK; ledger reports `go test ./...` 86 pkgs 0 FAIL. Two new tests re-verified PASS independently. CI-green-on-main is the only piece not verifiable from disk.

## Changes
- [✓] Change 1 — `contextToCarry` ORDER BY → `priority ASC, created DESC, key ASC` — `projection.go:310-312`, matches spec verbatim.
- [✓] Change 2 — `openFeaturesByPriority` ORDER BY → same committed-derivable keys — `projection.go:202-204`, matches spec verbatim.
- [✓] Change 3 — `attemptsForSession` left as-is (`ORDER BY a.ingested_at`, `projection.go:276`) — spec designated this out of scope (session-filtered, empty in CI path). Confirmed untouched.
- [✓] Change 4 — gate purpose preserved, no code change — confirmed the gate/hook were not modified.
- [✓] Tests — two determinism tests + a `section()` helper added to `projection_test.go`; real assertions on section text, ordering, and non-emptiness.

## Open items
None. No PARTIAL / SKIPPED / BLOCKED rows.

## Audit notes
- **Total-order guarantee is airtight for `## Next`, in-practice for `## Context to carry forward`.** The `## Next` query filters `type = 'Feature'`, and the nodes unique index is `(type, key) WHERE valid_to IS NULL` (`graph.go:145-146`), so `key ASC` is a fully unique final tie-break there — provably total. `contextToCarry` filters `type IN ('Decision','Initiative')` but the ORDER BY does **not** include `type`; the unique constraint is on `(type, key)`, so `key` alone is only unique *within* a type. A Decision and an Initiative sharing the same slug **and** the same `priority` **and** the same `created` would tie on all three sort keys and fall back to SQLite rowid — the exact nondeterminism this fix targets. In practice spec slugs are unique across the workspace tree, so this collision does not occur and CI will stay green; the determinism guarantee for this section rests on that external invariant rather than on the table's own constraint. Adding `type` as a final tie-break would make it total by construction. Low probability, not a ship blocker — the spec itself chose `key ASC` as the final tie-break — but worth knowing since determinism is the whole point of this fix.
- **Tests are genuine regression guards; byte-identical sub-assertion is belt-and-suspenders.** For the specific seedings chosen, an `ingested_at DESC` sort coincidentally reproduces the same row order across the two stores, so the byte-identical assertion alone would not have failed the old code. The explicit ordering assertions (`priority/created/key` positions) are the real teeth — they definitively fail under the old `ingested_at` ordering. Net: the tests do prove the fix; the perturbation (distinct bumped `ingested_at` in a flipping order) is real, and the ordering asserts pin the committed-derivable result.
- **Scope is tight.** Working tree changes beyond the two code files are the expected projection outputs (`.hero/NEXT.md`, `.hero/SNAPSHOT.md`, `.hero/next/chet-bellows.md`) and the new spec directory — data, not code. No unrelated code touched; no regression to the prior `next-drift-gate-unwinnable` fix (gate, hook, `-I'^updated: '` all unchanged).
