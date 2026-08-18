# Delivery audit — node-index-repo-identity-collision

**Audited:** `git diff -- internal/index/index.go internal/index/index_test.go internal/index/project.go internal/retrieval/retrieval_test.go internal/cli/scan_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Two live nodes with the same type/key and different repos both project — `internal/index/project.go:26`, `internal/index/project.go:59`, and `TestProjectionPreservesSameKeyAcrossRepos` at `internal/retrieval/retrieval_test.go:464` carry repo through the insert and assert both `ContextDoc/architecture-overview` rows survive.
- [✓] Projection preserves repo through retrieval — `internal/index/project.go:26`, `internal/index/project.go:59`, and `internal/retrieval/retrieval_test.go:503` verify the two retrieved results expose `astroville/hydra` and `boxy` repo metadata.
- [✓] Legacy indexes migrate idempotently without losing row IDs, metadata, or FTS joins — `internal/index/index.go:198`, `internal/index/index.go:270`, and `internal/index/index.go:295` implement repo-scoped creation and an atomic legacy rebuild; `TestOpenMigratesLegacyNodeIndexIdentity` at `internal/index/index_test.go:51` verifies the widened constraint, preserved row 41 metadata/FTS join, and no-op reopen.
- [✓] Failed projection retains the prior committed index — the replacement remains within one transaction at `internal/index/project.go:40`; `TestProjectionFailureRetainsCommittedIndex` at `internal/retrieval/retrieval_test.go:523` forces a repo-specific insert failure and asserts the prior metadata and FTS body remain.
- [✓] Full code-scan projection completes with the Boxy/Hydra collision — `TestFullCodeRefreshProjectsSameContextKeyAcrossRepos` at `internal/cli/scan_test.go:398` runs the full heuristic refresh and asserts both repo rows; supplied exercise evidence records a successful installed-binary Boxy scan projecting 1,657 nodes and live search/SQLite results for both repos.

## Changes

- [✓] Update and migrate the `node_index` schema — `internal/index/index.go:198` defines repo-scoped uniqueness for new indexes; `internal/index/index.go:295` detects the legacy constraint, rebuilds under `BEGIN IMMEDIATE`, copies every metadata field with explicit row IDs, recreates indexes, and rechecks under the write lock.
- [✓] Correct graph projection — `internal/index/project.go:26` selects repo, `internal/index/project.go:59` inserts it, and `internal/index/project.go:100` includes the failing repo in insert errors while preserving the existing transaction.
- [✓] Add migration, projection, retrieval, rollback, and scan regressions — `internal/index/index_test.go:51`, `internal/retrieval/retrieval_test.go:464`, `internal/retrieval/retrieval_test.go:523`, and `internal/cli/scan_test.go:398` directly assert each requested seam.

## Audit notes

- None.
