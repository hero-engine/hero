---
title: "Repo-scoped graph nodes collide in the search projection and break hero scan"
slug: node-index-repo-identity-collision
type: bug
status: completed
domain: engineering
priority: critical
severity: high
root_cause_class: code
size: small
tags: [scan, graph, index, federation, repo-partition, sqlite]
created: 2026-08-17
relates-to: [graph-node-identity-repo-scoped, graph-unpartitioned-writers-duplicate-nodes]
delivery_method: manual
completed_at: 2026-08-18T04:56:58Z
---

# Repo-scoped graph nodes collide in the search projection and break `hero scan`

## Kickoff

Fixes `hero scan` when local and sibling projects legitimately own the same graph key.

**Status:** completed — full tests, local installation, live Boxy validation, cold audit, and Hero verification all pass.

**Pick up at:** no delivery work remains; use this record if repo-scoped graph identity changes again.

→ `.hero/specs/node-index-repo-identity-collision/spec.md`

**Files:** `internal/index/index.go`, `internal/index/project.go`, `internal/index/index_test.go`, `internal/retrieval/retrieval_test.go`, `internal/cli/scan_test.go`
**Skip:** deduplicating or dropping sibling nodes; both repo-partitioned records are valid.

## Summary

### Categorization

| Attribute | Assessment |
|---|---|
| **Criticality** | high — `hero scan` exits non-zero in a normal federated workspace, blocking a release-critical workflow |
| **Ease of Fix** | moderate — the projection and its existing-database schema must move together |
| **Caused by our codebase?** | Yes — the search projection retained the pre-federation identity contract |
| **Needs more research?** | No — the live rows, schema mismatch, and failing insert are directly observed |

### Background

Boxy imports registered sibling subgraphs so cross-project search and graph traversal can see related work. Boxy and Hydra each legitimately own generated knowledge entries such as `ContextDoc/architecture-overview`. The graph database correctly stores those identities as `(type, key, repo)`, but the derived search index still treats `(node_type, key)` as globally unique.

### Analysis

`hero scan` successfully writes the local knowledge entries and reconciles the local and sibling graph partitions. During the later full-text-search projection, it streams every live graph node into `node_index`. The second repo's copy of a shared key violates the stale two-column unique constraint and aborts the projection transaction.

### Root Cause

Graph schema v5 changed current-node identity from `(type, key)` to `(type, key, repo)`. `internal/index/index.go` was not updated with that change: `node_index` still declares `UNIQUE(node_type, key)`. `internal/index/project.go` compounds the mismatch by neither selecting `nodes.repo` nor inserting `node_index.repo`, even though that destination column and retrieval support already exist.

### Source

The defect is in the graph-to-search projection boundary: `internal/cli/scan.go` invokes `internal/index/project.go`, which writes into the stale `node_index` schema created by `internal/index/index.go`.

### Fix Direction

Make the derived index use the graph's canonical repo-scoped identity. Migrate existing `node_index` tables atomically to `UNIQUE(node_type, key, repo)`, preserving row IDs and their FTS pairing; then select and insert `repo` during every projection. Do not collapse, overwrite, or discard either project's node.

## Issue

Running the newly built Hero in `/Users/developer/projects/astroville/repository/boxy` produces six knowledge stubs, begins the heuristic code scan, and then fails:

```text
Error: projecting graph nodes: insert node_index ContextDoc/architecture-overview:
constraint failed: UNIQUE constraint failed: node_index.node_type, node_index.key (2067)
```

### Reproduction

1. Use a Hero workspace that ingests a sibling subgraph.
2. Give both repos a live node with the same type and key, such as `ContextDoc/architecture-overview`.
3. Run `hero scan` or call `ProjectGraphNodes` against that graph.
4. Observe the second `node_index` insert fail on `(node_type, key)`.

### Confirmed evidence

- Boxy's live graph contains two valid rows for `ContextDoc/architecture-overview`: repo `astroville/hydra` (node 402) and repo `boxy` (node 1673).
- Five additional generated keys have the same valid cross-repo shape: `project-overview`, `dev-workflow`, `multi-module-structure`, `project-conventions`, and `project-rules`.
- Boxy's `node_index` schema includes a `repo` column but declares `UNIQUE(node_type, key)`.
- The failing projection transaction rolls back; it does not leave a partially rebuilt FTS index.
- The related `graph-unpartitioned-writers-duplicate-nodes` bug is distinct: these rows are correctly partitioned and must coexist.

## Environment Details

- Hero: `v0.32.0-2-ge09f3ee-dirty`, built from this workspace.
- Grok Build: 1.0.4 triggered the workflow, but the failure is in the Hero CLI and reproduces independently of the harness.
- Project: Boxy with the Hydra sibling corpus registered and ingested.
- Storage: SQLite `graph.db` plus derived `index.db`/FTS5 projection.

## Root Cause Analysis

The graph and the search projection implement different identity models. Graph migration v5 deliberately permits two live rows sharing `(type, key)` when their `repo` values differ. That change is required for federation: otherwise ingesting a sibling's same-slug node tombstones the local record.

The search index predates that migration. Its table has a dormant `repo` column, but its unique constraint remains global and its projector leaves every projected repo blank. The first shared key inserts successfully. The second has the same `(node_type, key)`, so SQLite raises constraint 2067. This is a code regression caused by an incomplete propagation of the graph-v5 identity contract, not corrupt data and not an unexpected user workflow.

## Code Flow (End to End)

1. `internal/cli/scan.go` — a full scan writes the code/work/sibling subgraphs into one repo-partitioned graph store.
2. `internal/cli/scan.go` — the projection phase opens `index.db` and calls `ProjectGraphNodesContext`.
3. `internal/index/project.go` — the graph query selects all live nodes, including local and sibling rows, but omits `repo` from its result.
4. `internal/index/project.go` — the transactional rebuild deletes the prior FTS projection and inserts each node into `fts_nodes` and `node_index`.
5. `internal/index/index.go` — `node_index` rejects the second repo's shared key because its unique constraint covers only `(node_type, key)`.
6. `internal/index/project.go` — the transaction rolls back and returns the constraint error through `hero scan`.

## Key Files

### Search index

| File | Lines | Relevance |
|---|---:|---|
| `internal/index/index.go` | 118-280 | Creates/evolves `node_index`; needs an idempotent legacy-schema migration |
| `internal/index/project.go` | 21-106 | Reads graph nodes and writes the derived FTS metadata |
| `internal/index/index_test.go` | — | Natural home for existing-database schema migration coverage |

### User-visible scan and retrieval

| File | Lines | Relevance |
|---|---:|---|
| `internal/cli/scan.go` | 480-520 | Calls the failing projection after sibling ingest |
| `internal/cli/scan_test.go` | — | End-to-end regression seam for a full scan with sibling same-key nodes |
| `internal/retrieval/retrieval.go` | 330-380 | Already reads `node_index.repo`, proving the field is part of the intended search contract |
| `internal/retrieval/retrieval_test.go` | 410-470 | Existing graph-projection and BM25 coverage |

## Secondary Defects

The projector has populated `node_index.repo` with its default empty value since the index was introduced. That prevents retrieval results from identifying their owning repo even when projection does not collide. Carrying `repo` through the corrected projection fixes this defect at the same boundary.

## Goal

`hero scan` completes when local and sibling projects own the same graph type/key, and the derived search index preserves both records with their repo identity without damaging existing index databases or FTS row pairing.

## Changes

1. Update and migrate the `node_index` schema in `internal/index/index.go`.
   - Change uniqueness to `(node_type, key, repo)` for new databases.
   - Detect the legacy two-column constraint and rebuild the table transactionally and idempotently.
   - Preserve existing row IDs and metadata so `fts_nodes` joins remain valid before the next projection.
2. Correct graph projection in `internal/index/project.go`.
   - Select `nodes.repo` and insert it into `node_index`.
   - Keep the all-or-nothing projection transaction and make collision errors identify the repo.
3. Add migration, projection, retrieval, and scan regression tests in `internal/index/index_test.go`, `internal/retrieval/retrieval_test.go`, and `internal/cli/scan_test.go`.
   - Prove an existing pre-fix index opens and migrates without losing its FTS linkage.
   - Prove two live same-type/key nodes in different repos both project and remain searchable with distinct repo metadata.
   - Recreate the Boxy/Hydra scan shape and assert the scan completes.

## Acceptance Criteria

- **AC-1:** WHEN two live graph nodes share `(type, key)` but have different non-empty repos, THE SYSTEM SHALL project both nodes into `node_index` without a uniqueness error.
- **AC-2:** WHEN a graph node is projected, THE SYSTEM SHALL preserve its repo in `node_index.repo` and expose that repo through the existing retrieval result contract.
- **AC-3:** WHEN Hero opens an existing index whose `node_index` uses `UNIQUE(node_type, key)`, THE SYSTEM SHALL migrate it idempotently to repo-scoped uniqueness while preserving existing row IDs, metadata, and FTS joins.
- **AC-4:** IF graph projection fails for any row, THEN THE SYSTEM SHALL roll back the replacement projection and retain the previously committed search index.
- **AC-5:** WHEN a full scan ingests local and sibling knowledge nodes with the same type/key, THE SYSTEM SHALL complete the graph projection and code-scan workflow without discarding either repo's node.

## Boundaries

- Do not remove sibling-subgraph ingestion or make generated knowledge slugs globally unique.
- Do not merge or deduplicate correctly repo-partitioned graph nodes.
- Do not fold in the separate cleanup of truly unpartitioned `repo = ''` writers.
- Do not redesign search ranking or local-versus-peer preference beyond preserving the repo metadata already modeled by retrieval.

## Risks

- Rebuilding `node_index` incorrectly could desynchronize its row IDs from the contentless FTS table. The migration must preserve row IDs and test the join before and after reopen.
- A non-idempotent schema evolution could break every normal `index.Open`; exercise both legacy and already-migrated databases.
- Filtering to one repo would hide valid federation data. Tests must assert both rows survive.

## Validation

- Run focused tests for `internal/index`, `internal/retrieval`, and `internal/cli`.
- Run the uncached full Go suite.
- Build and install the resulting local Hero binary.
- Re-run `hero scan` in Boxy on the same graph containing Boxy/Hydra key collisions and require exit zero.
- Inspect Boxy's `node_index` to confirm both `ContextDoc/architecture-overview` rows exist with distinct repos.
- Run `git diff --check`, a cold delivery audit, and `hero spec verify node-index-repo-identity-collision`.

## Notes

The six knowledge entries written before the failure are normal local scan output. The constraint occurs later in a derived, transactional projection; there is no evidence of duplicate local identity or partial index replacement.

## Recap

Federation correctly permits two repos to own the same graph key, but the derived search index still enforces the obsolete global key and drops repo metadata. Aligning the index schema and projector with `(type, key, repo)` removes the scan crash while retaining both projects' valid nodes.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Same type/key in different repos both project | DONE | `internal/retrieval/retrieval_test.go:464` — projects the exact `ContextDoc/architecture-overview` collision for `astroville/hydra` and `boxy` and asserts both rows survive. |
| 2 | Projection preserves repo through retrieval | DONE | `internal/index/project.go:26`, `internal/retrieval/retrieval_test.go:503` — selects/inserts repo and verifies both retrieval results expose distinct repo metadata. |
| 3 | Legacy index migrates idempotently without losing rowids, metadata, or FTS joins | DONE | `internal/index/index.go:295`, `internal/index/index_test.go:51` — exercises `pragma_index_info`, atomic legacy rebuild, preserved row 41 metadata/FTS join, widened uniqueness, and already-migrated reopen. |
| 4 | Failed projection rolls back to the prior committed index | DONE | `internal/retrieval/retrieval_test.go:523` — a trigger rejects the Boxy insert; the prior Hydra metadata and FTS content remain intact. |
| 5 | Full code-scan projection completes with the Boxy/Hydra collision | DONE | `internal/cli/scan_test.go:398` — runs full heuristic `refreshCodeIndex` over the two repo-scoped ContextDocs and asserts two projected rows. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Update and migrate `node_index` schema | DONE | `internal/index/index.go:198`, `internal/index/index.go:295` — new databases use repo-scoped uniqueness; legacy databases rebuild under `BEGIN IMMEDIATE` while copying every rowid and metadata column. |
| 2 | Correct graph projection | DONE | `internal/index/project.go:26` — graph reads and metadata inserts carry repo; projection remains one transaction and errors name the failing repo. |
| 3 | Add migration, projection, retrieval, and scan regressions | DONE | `internal/index/index_test.go:51`, `internal/retrieval/retrieval_test.go:464`, `internal/retrieval/retrieval_test.go:523`, `internal/cli/scan_test.go:398` — all requested seams have direct coverage. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end to end with the newly installed binary: `hero scan` in Boxy completed in 703 ms and projected 1,657 nodes; SQLite inspection and `hero search "architecture overview"` showed both `astroville/hydra` and `boxy` copies. The production full-refresh regression also passes in `TestFullCodeRefreshProjectsSameContextKeyAcrossRepos`.

### Excellence Bar self-check

Yes — the fix is atomic, idempotent, repo-aware end to end, preserves legacy FTS row pairing, and includes regression coverage at schema, projection, retrieval, rollback, and full-refresh boundaries.
