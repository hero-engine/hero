---
audit_type: delivery
spec: unified-retrieval-layer
audited: 2026-06-09
auditor: cold-audit-agent
verdict: SHIP
---

# Delivery Audit — unified-retrieval-layer

## What was audited

Cold audit against spec at `.hero/planning/features/unified-retrieval-layer/spec.md`.
Key files examined:
- `internal/retrieval/retrieval.go` (800 lines)
- `internal/retrieval/retrieval_test.go` (1435 lines)

Test run: `go test ./internal/retrieval/... -v`

---

## AC Verification

### AC-1 — Graph nodes projected; type boosts prevent commit drowning

**Status: DONE — confirmed.**

`index.ProjectGraphNodes(graphDB)` populates `fts_nodes` + `node_index` at scan+rebuild time. `TestBM25RegressionD9997ea` seeds 1 Feature and 50 Commits matching "task", projects via `ProjectGraphNodes`, and asserts Feature ranks first with every Commit score strictly below it. Test passes. `TestProjectionPopulatesFTSNodes` verifies row counts in both tables (3 projected = 3 in `fts_nodes` and `node_index`). Both pass.

### AC-2 — BM25 ranking via fts_nodes (not per-type SQL windowing)

**Status: DONE — confirmed.**

`retrieveViaNodeIndex` issues a single `fts_nodes MATCH ?` query ordered by `fts_nodes.rank` (FTS5 BM25), then re-sorts by `-bm25Rank × typeBoost(nodeType)`. The boost is a multiplicative score factor, not a hard cutoff or window. `TestBM25RankingViaNodeIndex` seeds a Feature and a Commit, projects, retrieves on "task", and asserts Feature at rank 1 with `source=graph`. Passes.

### AC-3 — Facets exist in schema but not exposed in Query API

**Status: PARTIAL — confirmed by design.**

`node_index` schema stores `node_type`, `path`, and `repo`. The `Query` struct exposes only `Text`, `Types` (FTS5 spec-corpus path), `Filters` (FTS5 path), `Limit`, `SemanticOK`, and `IncludeSuperseded`. There is no `Facets` or `NodeTypes` field for the graph-node path. The spec says explicitly: *"Next step when a caller needs them."* This is a deliberate deferral, not a gap. The facets are present in storage and can be exposed without schema migration when the first caller needs them.

### AC-4 — Single retrieval interface; no raw SQL in callers

**Status: DONE — confirmed.**

`Retrieve(q Query) ([]Result, error)` is the sole external entry point. All five caller files verified:
- `internal/cli/search.go` — uses `retrieval.New` + `retrieval.Query`
- `internal/cli/ask.go` — uses `retrieval.New` + `retrieval.Query`
- `internal/cli/relevant.go` — uses `retrieval.New` + `retrieval.Query`
- `internal/serve/mcp_tools.go` — uses `retrieval.New` + `retrieval.Query`
- `internal/serve/chat/slash_ask.go` — uses `retrieval.New` + `retrieval.Query`

No caller references `fts_nodes` or `node_index` directly. Raw SQL for these tables is fully contained inside `internal/retrieval/` and `internal/index/`.

### AC-5 — BM25 zero-match fallback to graph node-key LIKE matching

**Status: DONE — confirmed.**

`Retrieve` routing: `retrieveViaNodeIndex` → on zero results → `retrieveViaGraph` (LIKE match on key/title/body/subject) → on zero results → `retrieveViaFTS`. `TestBM25FallbackToGraphLIKE` inserts a Feature into the graph without projecting it (fts_nodes empty), retrieves on "xyzfoo", and asserts non-empty results with `source=graph`. Passes.

---

## Test Suite Summary

**Total tests run: 23 — all PASS. Build: clean (`go build ./...` exits 0).**

Three spec-mandated tests confirmed passing:

| Test | AC | Result |
|---|---|---|
| `TestBM25RegressionD9997ea` | AC-1, AC-2 | PASS |
| `TestBM25RankingViaNodeIndex` | AC-2 | PASS |
| `TestBM25FallbackToGraphLIKE` | AC-5 | PASS |

Additional tests passing (all 23):
- `TestRoutingFiltersGoFTS`, `TestRoutingTypesGoFTS`, `TestRoutingGraphFirst`, `TestRoutingFallbackToFTS` — routing table correct
- `TestTypeBoostValues`, `TestTypeBoostIsMultiplier` — boost math verified as multiplicative (ratio 10:1 Feature:Commit)
- `TestProjectionPopulatesFTSNodes` — projection row-count integrity
- `TestFuseRRF_*` (6 tests) — RRF fusion correctness, limit, empty inputs
- `TestRetrieveHybrid_*` (4 tests) — Phase C hybrid path + supersede overlay
- `TestSupersededDeweightRanksAfterPeers`, `TestIncludeSupersededSkipsDeweight` — post-spec additions
- `TestLoadSupersededOverlay_*` (2 tests) — SQL helper correctness

---

## Phase Scope Check

Phase A (unified facade) and Phase B (enriched FTS5 + BM25) are fully shipped. Phase C (vector/hybrid) has code present and tested — the `retrieveHybrid` path with RRF fusion is implemented and covered by four tests, though model-dependent tests skip gracefully when no embedded model is installed. This is additive scope beyond what Phases A+B required; it does not break any AC.

---

## AC-3 PARTIAL Assessment

AC-3 is correctly classified PARTIAL, not a hold condition. The spec's own wording defers Query-API exposure to "Next step when a caller needs them." No caller currently needs faceted filtering on the graph-node path (that use case routes through the FTS5 spec-corpus path which already supports `status`/`tag`/`type`/`since` filters via `q.Filters`). The infrastructure (schema columns, join table) is present. This is an intentional horizon deferral, not a missing implementation.

---

## Risk Register

| Risk | Severity | Notes |
|---|---|---|
| `fts_nodes` empty until `ProjectGraphNodes` runs | Low | AC-5 fallback covers this; test confirms it |
| Phase C hybrid tests skip without embedded model | Low | Graceful skip is correct behavior for optional component |
| AC-3 facets deferred | None | By design; schema is in place |

No blockers found.

---

**Verdict:** SHIP
