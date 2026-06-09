---
audit_for: graph-conflict-detection
audited_at: 2026-06-09
auditor: ccd-delivery-audit
diff_base: working-tree vs HEAD (uncommitted changes, 6 files modified + 1 new)
---

# Delivery audit — graph-conflict-detection

**Audited:** working-tree diff against HEAD (changes not yet committed)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] `hero sync graph push` prints warning when it overwrites a different client's node — `internal/cli/sync_graph.go:159-165`: `len(resp.Conflicts) > 0` guard prints "Warning: N conflict(s) — your version won, but a teammate's version was overwritten:" with per-conflict detail and reconciliation hint. `savePushConflicts()` caches records to `.hero/push_conflicts.json`. `SyncConflict` struct at `internal/graph/sync.go:64-71` carries `NodeType`, `NodeKey`, `Reason`.
- [✓] `hero check conflicts <slug>` reports graph divergence — `internal/cli/conflicts.go:77-94`: opens `graph.Store`, calls `store.FindGraphConflicts(slug)`, formats multi-client divergence with per-version client/status/timestamp output. Pre-existing push-conflict cache path (`loadPushConflictsForSlug`) and FTS5 spec-conflict path (`idx.FindConflicts`) also remain intact. All three conflict sources combined in a single command.
- [✓] Re-running push with no changes after pull is a no-op (same client_id, same hash → no conflict) — Server-side behavior (separate cloud repo). Client surface is correct: `PushResponse.Conflicts` is `omitempty`; empty slice means warning block never prints and `savePushConflicts` is never called. `FindGraphConflicts` also requires `client_id IS NOT NULL AND client_id != ''` and `len(clients) < 2` guard, so a single-client re-push independently produces zero local results.
- [✓] Same-client re-push of changed content is NOT flagged — `internal/graph/conflicts.go:110-114`: `len(clients) < 2` guard; single-client history never emits a conflict result. `TestFindGraphConflicts_SingleClientNoConflict` inserts two rows for client "alice" and asserts `len(results) == 0`.

## Changes

- [✓] `internal/graph/conflicts.go` — Fixed the timestamp parsing bug: switched from scanning directly into `time.Time` (which fails with SQLite TEXT storage) to scanning into strings and calling `parseGraphTime()`. Added `graphTimeLayouts` var and `parseGraphTime()` helper. The pre-change version had the correct SQL and types but would have errored at runtime on any real database row; this change makes the function production-safe.
- [✓] `internal/graph/conflicts_test.go` — New file (did not exist in HEAD). Two behavioral tests: `TestFindGraphConflicts_MultipleClientsDetected` inserts alice + bob rows and asserts a conflict result is returned with the correct type, key, and ≥2 versions. `TestFindGraphConflicts_SingleClientNoConflict` inserts two alice rows and asserts zero results. Both run against a real in-memory SQLite store via `openTestStore`.
- [✓] `internal/graph/graph.go` — `schemaVersion` bumped from `"3"` to `"4"`. Migration v4 adds `ALTER TABLE nodes ADD COLUMN client_id TEXT NOT NULL DEFAULT ''` and `CREATE INDEX IF NOT EXISTS idx_nodes_client ON nodes(client_id)`. Follows the existing ordered, idempotent migration pattern. Comment explains DEFAULT '' safety for pre-federation rows.
- [✓] `internal/graph/domain_test.go` — Replaced four hardcoded `"3"` version assertions with `schemaVersion` constant. `TestSchemaV3FreshDB`, `TestSchemaV3Idempotent`, `TestRollbackV3` all continue to pass and will not require future edits on schema bumps.
- [✓] `internal/graph/graph_test.go` — One hardcoded `"3"` replaced with `schemaVersion` in `TestPartialMigrationRecovery`. Same rationale.
- [✓] `internal/cli/conflicts.go` — Wired `graph` import and `store.FindGraphConflicts(slug)` call inside `runConflicts`. Added `countDistinctClients` helper. The bitemporal-history block is the third conflict source in the command, correctly gated behind `gerr == nil` for graceful degradation when `graph.db` is absent.

## Open items

None.

## Audit notes

- SC-1 was delivered in a prior session (`sync_graph.go` changes are not in this diff — they exist in HEAD already). The current diff wires the final missing piece: `FindGraphConflicts` called from `runConflicts`. The ledger correctly attributes SC-1 to the prior session.
- `conflicts_test.go` was confirmed absent in HEAD via `git show HEAD:internal/graph/conflicts_test.go` (empty output). It is a net-new file in this change.
- The pre-change `conflicts.go` had the correct SQL and grouping logic but scanned timestamps as `time.Time` directly; the SQLite `mattn/go-sqlite3` driver stores timestamps as TEXT and does not auto-convert. That scan would have returned a column-type mismatch error on any real database. The `parseGraphTime` fix handles both the RFC3339 format written by `nowRFC3339()` and the Go default `time.String()` format the driver emits when a `time.Time` is passed as a bound parameter.
- SC-3 and SC-4 rely on server-side `PushGraphDelta` logic in the cloud repo (explicitly noted as out of scope in the spec). The client-side guards (`len(clients) < 2`, `client_id != ''`) provide independent defense against false positives and are verified by the new tests.
- Diff is tightly scoped to the spec's stated files plus the new `conflicts_test.go`. No scope drift observed.

**Verdict:** SHIP
