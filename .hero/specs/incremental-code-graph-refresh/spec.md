---
title: "Incremental Code Graph Refresh"
slug: incremental-code-graph-refresh
type: enhancement
status: completed
domain: engineering
size: large
priority: critical
horizon: now
created: 2026-07-27
parent: continuous-code-index-freshness
conflicts-with:
  - embeddings-never-refresh-on-commit
tags: [codescan, graph, incremental, fts, cache, code-index]
delivery_method: manual
completed_at: 2026-07-27T19:37:18Z
---

# Incremental Code Graph Refresh

## Context

`internal/cli/scan.go:runCodeScan` already orchestrates the canonical code
scan: load `.checksums.json` and `ScanCache`, select the configured parser,
call `Scanner.Scan`, detect hot files, write generated knowledge, ingest graph
nodes, project graph nodes to FTS, and refresh embeddings. The hook path needs
the code portion of that flow without duplicating it or running the broader
stack/project scan.

`Scanner.Scan` already performs the key incremental optimization. It hashes
the files admitted by `CodeScanConfig`, reparses only changed files, carries
unchanged `FileInfo`, config variables, and endpoints from `ScanCache`, and
re-aggregates a complete merged `Result`. That result—not `git diff`—can be
used as an authoritative keep-set.

The missing correctness behavior is deletion. `codescan.WriteGraph` upserts
current `Repo`, `Package`, `File`, and `Symbol` nodes but does not retire
codescan-owned rows absent from the complete result. Deleted symbols therefore
remain current in `graph.db`, survive `ProjectGraphNodes`, and cannot be
pruned safely from code embeddings.

## Goal

Provide one reusable quiet incremental code refresh primitive and CLI mode,
factored from `runCodeScan`, that cheaply no-ops when configured source content
is unchanged and, when it changes, updates generated code knowledge,
reconciles the local codescan graph including deletions, and projects current
graph nodes into FTS. Cache/checksum state advances only after the complete
refresh succeeds.

## Kickoff

Factors `runCodeScan` into a reusable incremental code-refresh path that turns
the existing complete `Scanner.Scan` result into authoritative generated,
graph, and lexical state.

**Status:** completed — implementation, full tests, compiled-CLI exercise,
cold audit, and `hero spec verify` all passed.

**Pick up at:** deliver the dependent embedding-refresh child, which can now
reuse the post-structure seam and authoritative code graph.

→ `/deliver embeddings-never-refresh-on-commit --autopilot`

**Files:** `internal/cli/scan.go`, `internal/codescan/scanner.go`,
`internal/codescan/scancache.go`, `internal/codescan/graph_ingest.go`,
`internal/graph/transaction.go`
**Skip:** git diff mapping, a second scan config, new parsers, deep enrichment,
hook wiring, embeddings changes, and harness-specific hooks.

## Problem

Automatic freshness cannot safely call `runCodeScan` as written:

- it mixes reusable work with human progress/tour output;
- its graph and projection failures are best-effort, yet generated knowledge
  persists checksums/cache before those downstream failures are known;
- it has no structured changed/no-change/retired stats;
- it cannot be bounded by a hook deadline;
- its graph writer is additive for deletions.

Adding a second scanner or interpreting `git diff` would create competing
source-selection rules and break rename/delete correctness. The smallest safe
change is to factor the existing orchestration around the existing complete
result.

## Approach

### One reusable orchestration result

Add a package-private refresh primitive in `internal/cli/scan.go` (or a
tightly scoped sibling `code_refresh.go` if tests make separation clearer)
used by both manual `runCodeScan` and the new incremental CLI mode. It accepts
the loaded `config.Config`, project root, Hero directory, quiet flag, and
deadline/context; it returns structured stats:

- files inventoried, reparsed, unchanged, added, and deleted;
- packages/files/symbols current and retired;
- graph nodes projected;
- elapsed time and phase reached;
- `Changed` and `Complete` booleans.

Do not add configuration. Resolve the parser through the same
`resolveParser(cfg.CodeScan.Parser)` path, create the same
`codescan.NewScannerWithMode`, and use `cfg.CodeScan.ShouldExclude`.

### Cheap no-change gate

Load `codescan.LoadChecksums` and `codescan.LoadScanCache`, then call the
existing incremental scanner. Compare the complete result checksums with the
previous checksum map, including added and deleted paths. When equal and the
cache is usable, return structured no-change stats before hot-file detection,
generated writes, graph ingest, or FTS projection. Preserve a downstream
post-structure seam: child 2 still checks configured embedding coverage on a
structurally unchanged run so an earlier model/DB failure can catch up.

The scan still inventories and hashes configured source files; it does not
parse unchanged files and performs no database or generated-file writes.
Missing, corrupt, or version-stale cache degrades to the existing full-parse
path in manual mode and is not a no-change result. Hook mode skips a
full-bootstrap parse when the cache pair is unusable and leaves freshness
reported as unavailable until an explicit `hero scan` rebuilds it.

Today `Scanner.Scan` silently skips walk, read, and parse errors. Add explicit
inventory completeness to the result: every skipped read/parse/walk error is
recorded and makes `Complete=false`. Such a result may report diagnostics but
cannot drive deletion retirement, projection, or persisted scan-state
advancement. Only a successfully traversed configured source tree is an
authoritative keep-set.

Bring endpoint-only `.proto`, `.graphql`, and `.gql` files through the same
checksum/cache accounting before extraction. Today they are read ahead of the
checksum path, so changing one can falsely satisfy checksum-map equality. This
corrects the existing inventory; it does not add a parser or diff mapper.

### Authoritative graph reconciliation

Extend `internal/codescan/graph_ingest.go:WriteGraph` around its existing
upsert logic and run the full codescan upsert/reconcile as one graph
transaction:

1. derive the existing repo key with `repoKeyFor(result.ProjectRoot)`;
2. build keep-sets for every `Package`, `File`, and `Symbol` key in the
   complete result;
3. upsert those nodes and their existing `belongs_to`, `defines`, and
   `imports` edges through the current graph APIs;
4. only after every upsert succeeds, query current nodes whose `repo` is the
   local repo key, whose type is `Package`, `File`, or `Symbol`, and whose
   JSON `source.kind` is `codescan`;
5. retire absent nodes and their current incident edges using the existing
   bitemporal invalidation semantics;
6. report retired counts in `GraphWriteSummary`.

The `Repo` node is updated but not retired by a source-tree-empty result.
Rows from sibling repo partitions, legacy rows not stamped
`source.kind=codescan`, and nodes owned by other writers are not candidates.
An empty configured source tree is authoritative only after the walk
completed successfully.

Do not implement direct SQL that forks node identity rules. Add the narrow
transaction-scoped graph-store entry point needed for `WriteGraph` to reuse
current upsert/invalidate semantics. Error or cancellation rolls back the
whole codescan mutation, including edge invalidations.

Add focused tests in `internal/codescan/graph_ingest_test.go` for deleted
symbol, file, and package retirement; current-edge retirement; empty-tree
authority; repo partition isolation; source-kind isolation; and idempotent
retry.

### Safe phase ordering

For a changed, complete result:

1. scan and build the complete result;
2. check the deadline before mutations;
3. call the reconciliating `codescan.WriteGraph`;
4. call the existing `index.ProjectGraphNodes`, whose clear-and-rebuild runs
   in one transaction;
5. write generated code knowledge through the existing
   `codescan.GenerateKnowledge` implementation;
6. after child 2 lands, refresh configured embeddings from the now-current
   graph and generated knowledge;
7. commit checksum/cache state only after every required phase succeeds.

Factor scan-state persistence out of the tail of `GenerateKnowledge` without
duplicating its package/index/prune behavior: keep the existing public helper
as a compatibility wrapper for callers that want generation+state in one
call, and let the coordinator call the same generation implementation followed
by the shared state-commit helper. If knowledge generation or a later phase
fails, the next run repeats the scan and graph work; `WriteGraph` and
`ProjectGraphNodes` are idempotent.

Harden the existing `.checksums.json` + `ScanCache` pair against interruption:
write both through temp-file-plus-rename, stamp a shared generation/checksum
manifest in the cache, and make the loaders reject a split generation. Keep
both current artifact paths and `BuildScanCache`; do not introduce a parallel
cache. This prevents a new checksum map from carrying forward older parsed
products after a crash between the two writes.

Stamp the resolved parser identity in the versioned `ScanCache`. If `auto`
resolves differently in a hook's PATH than in the prior manual scan, reject
the cache and rebuild/skip according to mode instead of carrying parse
products across parser backends.

The child-2 delivery fills the embedding seam before final state commit.
Until that child lands, manual full scan retains its existing embedding call.

### Deadline and quiet semantics

Add an incremental CLI mode to the existing `hero scan --code` surface rather
than a parallel top-level command. The final command shape is:

`hero scan --code --incremental --deadline <duration> -q`

The exact default deadline is a CLI constant, not configuration. Acquire one
fail-fast workspace refresh lock before loading the source snapshot so
pre-commit, post-merge, manual scan, and serve-triggered work cannot interleave
cache generations. A busy lock is a structured skip in quiet mode.

Propagate one context through file walking, graph SQL, FTS projection, and
downstream embedding integration, using backward-compatible wrappers where
public methods currently lack context. Check remaining budget before each
mutation phase. Codescan graph reconciliation is one transaction: it either
commits before the deadline or rolls back. Do not implement a timer goroutine
that returns while uncancellable writes continue. Add context-aware wrappers
to every reused SQLite/filesystem phase required for the aggregate hook
deadline; a soft caller-side timer is not an acceptable implementation of the
CLI contract.

Quiet mode writes nothing to stdout/stderr and returns structured failure to
the caller while the Cobra command normalizes the hook invocation to exit
zero. Non-quiet manual mode returns errors normally and renders the same
human scan summary/tour it does today. Child 3 adds `|| true` as a second
best-effort guard in git hooks.

Deep/LLM enrichment is never run by `--incremental`; normal structure scanning
is used even if manual `code_scan.depth` is `deep`. This is a mode behavior,
not a second config tree. A normal manual `hero scan --code` continues to
honor deep mode.

## Changes

1. `internal/cli/scan.go`, `internal/cli/scan_test.go`,
   `internal/cli/helpers_test.go` — factor the shared coordinator; add the
   incremental/deadline/quiet CLI contract, fail-fast lock, structured phase
   stats, manual rendering parity, and CLI/integration coverage.
2. `internal/codescan/scanner.go`, `internal/codescan/types.go`,
   `internal/codescan/scancache.go`, `internal/codescan/codescan_test.go` — add
   context-aware authoritative inventory, changed-file accounting,
   endpoint-only checksums, parser-stamped paired cache generations, and
   incomplete/split-state tests.
3. `internal/codescan/graph_ingest.go`,
   `internal/codescan/graph_ingest_test.go`, `internal/graph/node.go`,
   `internal/graph/edge.go`, `internal/graph/transaction.go` — reconcile
   codescan-owned deletions and incident edges atomically through the canonical
   graph upsert semantics; include imports in package identity.
4. `internal/index/project.go` — retain the existing lexical projector while
   propagating cancellation through its transactional clear-and-rebuild.
5. `internal/codescan/generate.go` — separate generated knowledge writes from
   the final context-aware scan-state commit so graph/projection/generation
   failures cannot advance the cache pair.

## Boundaries

- No changes to `internal/embeddings/**`,
  `internal/cli/next_hooks.go`, `internal/cli/check.go`, or embeddings status;
  those belong to children 2 and 3.
- No hook installation or hook-template edits.
- No watcher, daemon, background goroutine, or post-commit hook.
- No parser implementation, file-to-symbol diff mapping, or parallel
  `CodeScanConfig`.
- No deep or LLM enrichment from the incremental mode.
- No sibling-repo graph reconciliation.
- No broad graph-store refactor. Add only the retirement behavior needed for
  codescan-owned current rows and incident edges in one transaction.

## Risks

| Risk | Mitigation / rollback |
|---|---|
| Reconciliation retires another writer's or repo's nodes. | Triple-scope by repo, type, and `source.kind`; isolation tests are mandatory. Roll back activation and run manual `hero scan`; bitemporal history retains prior rows. |
| Deadline lands after some graph upserts. | One codescan transaction rolls back all node/edge mutations; retirement and scan-state commit require a complete result. |
| Scanner silently skips a file and absence looks like deletion. | Make skipped walk/read/parse errors explicit and prohibit retirement from `Complete=false`. |
| Advancing checksums hides a failed graph projection or carries an old cache. | Run graph and FTS before `GenerateKnowledge`; generation-stamp and temp-rename the existing checksum/cache pair and reject mismatch. |
| Concurrent hook/manual runs overwrite scan generations. | One fail-fast local refresh lock serializes the coordinator; quiet contenders skip and remain visibly stale. |
| Full source hashing is too slow for commits. | Measure separately from parsing; no-change path avoids parse and all writes. If the bound is still unacceptable, disable the hook line—not source-of-truth correctness. |
| Manual scan output or deep behavior regresses during factoring. | Preserve `runCodeScan` as a rendering wrapper and add parity tests before child 3 activates the new mode. |

## Validation

- Focused: `go test ./internal/codescan ./internal/cli ./internal/index`
- Full: `go test ./...`
- Run the incremental command twice on an unchanged fixture and assert the
  second run reports no changes and leaves generated files, `graph.db`, and
  FTS rows untouched.
- Add, change, and delete exported symbols in a temp repo; assert the complete
  result, generated knowledge, current graph rows, incident edges, and FTS
  projection converge each time.
- Hold graph/index locks and force a deadline; assert no cache/checksum
  advancement and no deletion retirement.
- Inject walk/read/parse failure and assert the result is non-authoritative and
  cannot retire a node; interrupt between cache artifact writes and assert the
  pair is rejected on reload.
- Run `git diff --check`.

## Acceptance Criteria

- **AC-1:** WHEN `hero scan --code --incremental` inventories an unchanged configured source tree with a usable `ScanCache` THE SYSTEM SHALL parse zero unchanged files, perform no generated/graph/FTS writes, and return structured no-change stats
- **AC-2:** THE SYSTEM SHALL select source files, parser mode, and excludes through the existing `CodeScanConfig`, `resolveParser`, and `Scanner.Scan` paths without a second scan configuration or git-diff mapper
- **AC-3:** WHEN a configured source file is added or changed THE SYSTEM SHALL use the complete merged `Result` to update generated code knowledge, current codescan graph nodes, and `ProjectGraphNodes` lexical projection
- **AC-4:** WHEN a package, file, or exported symbol is deleted from a successfully scanned configured source tree THE SYSTEM SHALL retire its current codescan-owned graph node and incident edges and remove it from lexical graph projection
- **AC-5:** IF a current code node belongs to another repo partition or its `source.kind` is not `codescan` THEN THE SYSTEM SHALL NOT retire that node during local code reconciliation
- **AC-6:** WHEN graph reconciliation and lexical projection complete successfully THE SYSTEM SHALL persist the complete result's `.checksums.json` and `ScanCache` state through the existing `GenerateKnowledge` path
- **AC-7:** IF scanning, graph reconciliation, projection, knowledge generation, or the deadline fails THEN THE SYSTEM SHALL NOT advance checksum/cache state or perform deletion retirement from a partial result
- **AC-8:** WHEN the incremental mode runs with `-q` THE SYSTEM SHALL suppress stdout/stderr, honor its deadline across scan phases, and normalize hook-facing failure to a non-blocking exit
- **AC-9:** WHEN normal `hero scan --code` runs THE SYSTEM SHALL use the factored orchestration while preserving its existing human output, deep/manual behavior, and generated artifacts
- **AC-10:** WHERE incremental hook mode is enabled THE SYSTEM SHALL perform structural scanning only and SHALL NOT invoke deep or LLM enrichment
- **AC-11:** IF source walking, reading, or parsing skips any configured file THEN THE SYSTEM SHALL mark the snapshot incomplete and SHALL NOT use absence from that snapshot to retire graph nodes or advance persisted scan state
- **AC-12:** WHEN codescan graph reconciliation mutates current nodes and edges THE SYSTEM SHALL commit the complete upsert-and-retire set atomically or roll it back
- **AC-13:** IF `.checksums.json` and `ScanCache` do not carry the same completed generation THEN THE SYSTEM SHALL reject the pair rather than reuse stale parsed products
- **AC-14:** IF another code-index refresh owns the workspace refresh lock THEN THE SYSTEM SHALL skip the quiet incremental run without mutation and preserve truthful stale coverage
- **AC-15:** WHEN an endpoint-only configured source file changes THE SYSTEM SHALL include its checksum in changed-source accounting and SHALL NOT take the unchanged no-op path
- **AC-16:** IF the resolved parser identity differs from the parser recorded in `ScanCache` THEN THE SYSTEM SHALL reject cached parse products rather than mix parser generations
- **AC-17:** WHEN a package's imports change THE SYSTEM SHALL invalidate and rebuild its current package and `imports` edge projection even if its other package fields are unchanged
- **AC-18:** WHEN structural source is unchanged but a configured embedding corpus remains missing, mismatched, or previously unavailable THE SYSTEM SHALL preserve the post-structure seam for child 2 to retry embeddings without rewriting graph or generated code knowledge

## Post-Delivery Amendment

Final compiled-binary rollout validation found that the refresh lock was
created at `.hero/code-refresh.lock`, which made a clean repository appear
dirty after hook execution. The lock now uses the existing ignored
`.hero/cache/code-refresh.lock` seam without changing fail-fast flock
semantics. `TestIncrementalCodeRefreshSkipsUnusableCacheAndBusyLock` proves the
cache lock exists, the old workspace-root lock does not, and contention still
skips without mutation.

The amended focused lifecycle tests, affected packages, and serial uncached
full repository suite pass.

## Completion Ledger

Implemented the Go incremental refresh by reusing the existing config, parser,
scanner, generated-knowledge, graph, and lexical projection paths. Validation:
focused packages, `go test ./...`, compiled-CLI unchanged/add/delete exercise,
and `git diff --check` all pass. No harness or embedding implementation files
were touched.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Unchanged incremental scan parses zero and performs no structural writes | DONE | `internal/cli/scan.go:403`, `internal/cli/scan_test.go:398` — structured post-structure return; unchanged artifact hashes and reparsed=0 verified |
| 2 | Reuse CodeScanConfig, parser resolution, scanner, and excludes | DONE | `internal/cli/scan.go:403`, `internal/cli/scan_test.go:661` — coordinator passes the existing config and resolved parser directly to `NewScannerWithMode`; configured exclude test passes |
| 3 | Added/changed sources converge generated, graph, and lexical state | DONE | `internal/cli/scan_test.go:398` — add/change exercise asserts generated symbol plus current graph and `node_index`/`fts_nodes` projection |
| 4 | Deleted package/file/symbol nodes and incident edges retire and leave FTS | DONE | `internal/codescan/graph_ingest_test.go:174`, `internal/cli/scan_test.go:398` — retirement counts/edges and zero graph/index/FTS rows verified |
| 5 | Repo and source-kind isolation | DONE | `internal/codescan/graph_ingest_test.go:229` — sibling codescan and local non-codescan nodes survive empty local reconciliation |
| 6 | Persist checksum/cache state only after graph, FTS, and generation | DONE | `internal/cli/scan.go:403`, `internal/codescan/generate.go:24` — state commit is the final coordinator phase, separate from knowledge generation |
| 7 | Failures and partial snapshots do not advance state | DONE | `internal/cli/scan_test.go:574`, `internal/codescan/codescan_test.go:615` — deadline/lock failure preserves checksums and graph; incomplete scan cannot generate or commit state |
| 8 | Quiet incremental mode is silent, deadline-bound, and non-blocking | DONE | `internal/cli/scan_test.go:574`, `internal/cli/scan_test.go:644` — deadline and byte-silent normalized-exit tests pass |
| 9 | Manual scan uses factored orchestration and preserves output/artifacts | DONE | `internal/cli/scan.go:279`, existing `TestScanBasic`/`TestScanExplainsUncustomizedUpdates`, and compiled manual CLI exercise pass |
| 10 | Incremental mode remains structural-only | DONE | `internal/cli/scan.go:403` — incremental branch omits work/sibling/deep/embedding phases; compiled incremental exercise confirms structural output only |
| 11 | Walk/read/parse skips make the snapshot incomplete and non-authoritative | DONE | `internal/codescan/scanner.go:158`, `internal/codescan/graph_ingest.go:41`, `internal/codescan/graph_ingest_test.go:330` — explicit diagnostics plus generation/state/graph rejection tested; direct incomplete graph input leaves current rows and history unchanged |
| 12 | Graph upsert-and-retire commits atomically or rolls back | DONE | `internal/graph/transaction.go:19`, `internal/codescan/graph_ingest_test.go:303` — injected reconcile failure rolls back prior upserts |
| 13 | Split checksum/cache generations are rejected | DONE | `internal/codescan/scancache.go:117`, `internal/codescan/codescan_test.go:647` — manifest mismatch is unusable |
| 14 | Busy refresh lock skips without mutation | DONE | `internal/cli/scan.go:563`, `internal/cli/scan_test.go:535` — fail-fast flock contention returns a structured skip before state load, and the lock lives under ignored `.hero/cache/` rather than dirtying the workspace root |
| 15 | Endpoint-only sources participate in change accounting | DONE | `internal/codescan/codescan_test.go:563` — proto deletion and GraphQL addition produce add/delete accounting and reparsing |
| 16 | Parser identity mismatch rejects cached products | DONE | `internal/codescan/scancache.go:117`, `internal/codescan/codescan_test.go:647` — heuristic cache rejected for treesitter |
| 17 | Import-only package changes rebuild package/imports projection | DONE | `internal/codescan/graph_ingest.go:332`, `internal/codescan/graph_ingest_test.go:258` — package ID changes and imports edge retargets |
| 18 | Structurally unchanged path preserves child-2 post-structure seam | DONE | `internal/cli/scan.go:389`, `internal/cli/scan_test.go:398` — `PostStructureReady` is true without graph/generated rewrites |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Factor CLI coordinator and flags | DONE | `internal/cli/scan.go`, `internal/cli/scan_test.go`, `internal/cli/helpers_test.go` — incremental/deadline/quiet, ignored cache-local lock, stats, manual wrapper, and end-to-end tests |
| 2 | Add scanner accounting and paired cache state | DONE | `internal/codescan/scanner.go`, `internal/codescan/types.go`, `internal/codescan/scancache.go`, `internal/codescan/codescan_test.go` — context, completeness, endpoint inventory, manifests, parser identity |
| 3 | Add transactional graph reconciliation | DONE | `internal/codescan/graph_ingest.go`, `internal/codescan/graph_ingest_test.go`, `internal/graph/node.go`, `internal/graph/edge.go`, `internal/graph/transaction.go` — canonical transactional upsert plus scoped retirement |
| 4 | Keep and context-enable lexical projection | DONE | `internal/index/project.go` — existing projector now uses context-aware graph reads and index transaction operations; deletion verified through CLI integration test |
| 5 | Separate generation from final state commit | DONE | `internal/codescan/generate.go` — context-aware generation/pruning precedes shared state commit |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: compiled `hero`, ran manual bootstrap and `hero scan --code --incremental --deadline 2s -q` unchanged/add/delete in an isolated repo; unchanged output was empty with identical artifact hashes, add appeared in generated/graph/index state, and delete left zero current code nodes or lexical rows.

### Excellence Bar self-check

Honest answer to "would a senior engineer who cares about this codebase be proud
to ship this?" — yes; source authority, partition isolation, transactional
rollback, deadline behavior, state-last persistence, and real CLI convergence
are all implemented and exercised without introducing a second scanner or
touching later-child/harness surfaces.
