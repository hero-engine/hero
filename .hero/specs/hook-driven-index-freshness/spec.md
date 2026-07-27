---
title: "Hook-Driven Index Freshness"
slug: hook-driven-index-freshness
type: enhancement
status: completed
domain: engineering
size: medium
priority: medium
horizon: now
created: 2026-07-27
parent: continuous-code-index-freshness
depends-on:
  - incremental-code-graph-refresh
  - embeddings-never-refresh-on-commit
tags: [git-hooks, freshness, check, embeddings-status, integration-tests]
delivery_method: manual
completed_at: 2026-07-27T20:36:46Z
---

# Hook-Driven Index Freshness

## Context

`internal/cli/next_hooks.go` already installs one marker-delimited managed
block into `pre-commit` and `post-merge`. It preserves user hook content,
regenerates both hooks during install/upgrade, byte-compares the installed
pre-commit block for currency, and keeps hook operations best-effort.

The current pre-commit block runs `hero next checkpoint -q`,
`hero index --if-stale -q`, and `hero queue write -q`, then stages projected
handoff files. Post-merge runs only the NEXT checkpoint. Neither path refreshes
the code graph or vectors.

This child activates the safe primitives from the first two children and owns
all hook/check/status/end-to-end integration. It does not change scanner,
graph, embedding engine, or storage semantics.

## Goal

Wire the existing repository-managed pre-commit and post-merge blocks to one
ordered, quiet, aggregate-deadline index pipeline; keep both hooks
best-effort; report actual missing/changed source coverage through existing
check/status surfaces; and prove add/change/delete convergence through graph,
lexical, vector, and hybrid retrieval without manual `hero scan`.

## Kickoff

Activates the completed code-graph and embedding primitives in Hero's existing
pre-commit/post-merge managed block and makes freshness observable and
end-to-end tested.

**Status:** completed — implementation, audit corrections, focused/affected
tests, compiled lifecycle exercise, clean cold re-audit, and delivery
verification all pass.

**Pick up at:** the parent `continuous-code-index-freshness` initiative is
complete; validate the combined initiative in the final local build.

→ `.hero/specs/hook-driven-index-freshness/spec.md`

**Files:** `internal/cli/next_hooks.go`, `internal/cli/next_hooks_test.go`,
`internal/cli/check.go`, `internal/cli/check_test.go`,
`internal/cli/check_json_test.go`, `internal/cli/embeddings.go`,
`internal/cli/embeddings_test.go`,
`internal/cli/index_freshness_integration_test.go`
**Skip:** post-commit hooks, harness session hooks, a watcher, search-time
refresh, scanner/graph rewrites, and embedding-engine changes.

## Problem

The engine fixes do nothing automatically until repository lifecycle events
invoke them. Activation must preserve git reliability and avoid a false
success chain: a code refresh that fails or times out cannot be followed by a
code embedding prune against stale graph authority.

Freshness reporting also cannot use HEAD time alone. A single newly embedded
chunk can make `MAX(embedded_at)` look current while many configured sources
are missing or hash-mismatched; a quiet repository can make an old timestamp
look stale when coverage is complete.

## Approach

### Reuse the existing managed block

Modify only `hookScript(kind)` and its current install/currency tests. Do not
add a hook installer or target-specific hook surface. The repository-managed
block covers all six Hero harness targets because it is a git lifecycle
surface, not a harness lifecycle surface.

Use this logical order in both hooks:

1. `hero index --if-stale -q || true` for spec/knowledge lexical state;
2. the shared quiet incremental code command from children 1 and 2, with one
   aggregate deadline; internally it performs code scan/reconciliation,
   `ProjectGraphNodes`, then configured embeddings including `code`;
3. existing `hero next checkpoint -q || true`;
4. pre-commit only: `hero queue write -q || true`, then the existing per-path
   staging loop.

Post-merge performs steps 1–3 but does not stage or write the queue, preserving
current post-merge semantics. There is no post-commit hook.

The coordinator must keep internal phase truth even though its quiet Cobra
surface and the shell block are non-blocking. One aggregate deadline prevents
separate phase budgets from accumulating. If the child-1 refresh lock is busy,
the hook skips quickly and leaves actual coverage stale for `hero check` to
report.

### Hook currency

Keep `preCommitHookStale`, `refreshHooksIfPresent`, `writeHookFile`, and
`installNextHooksQuiet` as the authority. Changing `hookScript` naturally makes
existing managed blocks compare stale; `hero upgrade` or
`hero next install-hooks` regenerates both pre-commit and post-merge.

Extend tests to prove:

- exact ordering in both blocks;
- every new command carries quiet/deadline and `|| true`;
- post-merge contains refresh but no queue/staging;
- pre-commit retains current staging paths and marker preservation;
- user content outside markers survives refresh;
- installed old blocks report stale and regenerated blocks report current.

Do not add an independent post-merge currency registry.

### Actual code-index coverage

Add a `code-index-freshness` health row in `internal/cli/check.go`. Reuse the
configured source inventory/checksum comparison exposed by child 1:

- changed path: current checksum differs from persisted complete generation;
- missing path: current configured file has no persisted checksum/cache entry;
- deleted path: persisted entry no longer appears in a complete current
  inventory;
- partial/unavailable: inventory cannot prove completeness.

Report `pass`, `warn`, or `unavailable`-equivalent warning detail without
parsing symbols or mutating state. HEAD timestamp is not freshness evidence.
When code scanning is disabled, report an intentional skipped/pass state and
do not demand code-vector freshness.

### Actual embedding coverage

Add an `embeddings-freshness` row using child 2's extraction/hash APIs without
loading vectors or calling `Embed`:

- missing chunk ID;
- stored hash differs from current extracted text hash;
- orphaned chunk ID only when the current extraction is authoritative;
- unavailable/partial corpus;
- configured corpus skipped/disabled.

Render per-corpus counts so one fresh chunk cannot hide a stale remainder.
Checks are read-only and do not refresh inline. An extraction that cannot
prove completeness reports unavailable/partial, never current.

### Embeddings status observability

Extend `hero embeddings status` in `internal/cli/embeddings.go` to render the
storage stats from child 2 with stable corpus ordering. For every corpus show
chunk count and newest `embedded_at`; JSON, if supported on this surface,
receives stable fields rather than human text parsing.

Timestamp answers “when was anything last embedded?” It is not labeled
current/stale; `hero check` owns source-coverage freshness.

### End-to-end git lifecycle

Add temp-repository integration coverage under `internal/cli` using the current
hook installer and the test-built Hero binary. Seed a small parseable source
tree and embedding model fixture, install hooks, and cover:

- pre-commit add, change, and delete of an exported symbol;
- post-merge add, change, and delete delivered from another branch;
- graph lookup has only current symbols;
- `node_index`/FTS lexical lookup includes additions/changes and excludes
  deletions;
- vector and hybrid retrieval include additions/changes and exclude deleted
  symbol chunks;
- no test or hook setup runs `hero scan` manually after initial bootstrap.

Also exercise a forced timeout/lock. Git succeeds, prior authoritative state
survives, and `hero check` reports real remaining coverage.

## Changes

1. Update `internal/cli/next_hooks.go`.
   - Reorder the shared managed block and insert the existing incremental
     command in both hook kinds.
   - Preserve markers, install/upgrade flow, user content, queue/staging
     differences, and best-effort shell behavior.
2. Extend `internal/cli/next_hooks_test.go`.
   - Assert exact command order, deadline/quiet flags, post-merge behavior,
     stale/current comparison, upgrade refresh, idempotence, and user opt-out.
3. Update `internal/cli/check.go`, `internal/cli/check_test.go`, and
   `internal/cli/check_json_test.go`.
   - Add code-index and embeddings coverage rows using child APIs.
   - Report changed/missing/deleted or partial/unavailable coverage, never
     HEAD-time guesses.
   - Keep the check read-only.
4. Update `internal/cli/embeddings.go` and
   `internal/cli/embeddings_test.go`.
   - Render newest timestamp and count per corpus from storage stats.
   - Use deterministic ordering and preserve existing enabled/model/scope
     output.
5. Add end-to-end hook tests in a dedicated
   `internal/cli/index_freshness_integration_test.go` (or the nearest existing
   git-hook integration fixture).
   - Cover symbol add/change/delete through both commit and merge paths.
   - Assert graph, FTS/lexical, vector, and hybrid outcomes without a manual
     post-change scan.
   - Cover timeout/lock non-blocking behavior and health visibility.
6. Update CLI docs/help only where the existing scan/embeddings/hook commands
   are documented.
   - Describe automatic commit/merge freshness, deadline/best-effort
     behavior, status timestamps, and manual recovery command.

## Boundaries

- No changes to `internal/codescan/**`, `internal/embeddings/refresh.go`,
  `internal/embeddings/storage.go`, or graph/index engine behavior.
- No post-commit hook.
- No watcher, daemon, scheduled job, or harness session hook.
- No target-specific files for opencode, cursor, claude, copilot, codex, or
  generic.
- No git-diff symbol mapper and no new parser/configuration.
- No refresh from search/retrieval read paths.
- No deep/LLM enrichment in hooks.
- No event chunker, chunk-ID collision, SQL pruning redesign, or
  ledger-governance work.

## Risks

| Risk | Mitigation / rollback |
|---|---|
| Hooks add noticeable latency. | One aggregate deadline, unchanged fast path, fail-fast refresh lock, measured real-repo budgets, and `|| true`. Roll back the managed-block command line; manual commands remain. |
| Quiet normalization hides internal failure and embeddings run after stale graph. | One coordinator owns phase order and passes explicit outcomes; embeddings do not run after graph/FTS failure. |
| Existing hooks remain old after release. | Current byte-comparison reports staleness; install/upgrade regenerates both blocks. No silent in-place mutation outside those paths. |
| Freshness checks become another expensive scan. | Reuse source inventory/hashes and embedding extraction hashes; never parse unchanged source or compute vectors. Bound output and measure check latency. |
| E2E tests depend on global Hero/model state. | Use temp repos, isolated Hero dirs, a deterministic test model/fixture, and the test-built executable; do not inspect the developer's live `.hero`. |
| Post-merge changes projected files without staging. | Preserve existing post-merge no-stage/no-queue contract; the next commit carries projections through the current pre-commit loop. |

## Validation

- Focused: `go test ./internal/cli -run 'Hook|IndexFreshness|EmbeddingsStatus|Check'`
- Affected packages: `go test ./internal/cli ./internal/codescan ./internal/embeddings ./internal/index ./internal/retrieval`
- Full: `go test ./...`
- Inspect generated pre-commit/post-merge blocks and run hook currency tests.
- Run add/change/delete commit and merge fixtures and verify all four retrieval
  surfaces without post-change `hero scan`.
- Run `git diff --check`.

## Acceptance Criteria

- **AC-1:** WHEN the managed pre-commit block runs THE SYSTEM SHALL execute spec/knowledge lexical refresh, the aggregate-deadline incremental code graph/FTS/embedding command, NEXT checkpoint, queue write, and existing staging in that order
- **AC-2:** WHEN the managed post-merge block runs THE SYSTEM SHALL execute spec/knowledge lexical refresh, the same aggregate-deadline incremental code graph/FTS/embedding command, and NEXT checkpoint in that order without queue write or staging
- **AC-3:** IF any refresh phase errors, times out, cannot acquire the refresh lock, or lacks a model/source THEN THE SYSTEM SHALL preserve git commit/merge success, suppress hook noise, and leave incomplete coverage visible to `hero check`
- **AC-4:** THE SYSTEM SHALL activate freshness only through the existing repository-managed pre-commit/post-merge installer for all six harness targets and SHALL NOT install a post-commit or harness session hook
- **AC-5:** WHEN the managed block template changes THE SYSTEM SHALL report an installed old block as stale and SHALL report both regenerated hooks current after existing upgrade/install-hooks refresh
- **AC-6:** WHEN `hero check` inspects code-index freshness THE SYSTEM SHALL report changed, missing, and deleted configured source coverage from actual inventory/checksum state rather than HEAD time
- **AC-7:** WHEN `hero check` inspects configured embeddings THE SYSTEM SHALL report per-corpus missing and hash-mismatched chunks, authoritative orphans, and partial/unavailable extraction without computing vectors or mutating the index
- **AC-8:** WHEN `hero embeddings status` runs THE SYSTEM SHALL show newest `embedded_at` and chunk count for every corpus in stable order without presenting timestamp alone as proof of freshness
- **AC-9:** WHEN an exported symbol is added, changed, or deleted and the pre-commit hook runs THE SYSTEM SHALL converge current graph lookup, FTS/lexical retrieval, vector retrieval, and hybrid retrieval without a post-change manual `hero scan`
- **AC-10:** WHEN an exported symbol is added, changed, or deleted by a merged branch and the post-merge hook runs THE SYSTEM SHALL converge current graph lookup, FTS/lexical retrieval, vector retrieval, and hybrid retrieval without a post-change manual `hero scan`

## Implementation Notes

The scanner package does not expose a checksum-inventory-only operation.
Within this child's boundary, `hero check` therefore calls the existing
`Scanner.ScanContext` with the persisted complete checksum/cache pair.
Unchanged files carry forward without symbol parsing (`reparsed=0`, covered by
test); changed and newly discovered files are parsed while their checksums are
collected. The check remains read-only and never writes scan state or computes
vectors. Eliminating parsing for changed/new files would require a new
inventory seam in `internal/codescan/**`, which this spec explicitly excludes.

Final compiled-binary rollout validation also found that the child-1 refresh
lock created `.hero/code-refresh.lock`, leaving a real repository dirty after
the hook ran. The lock now uses the existing ignored
`.hero/cache/code-refresh.lock` seam. Focused coverage proves the ignored lock
exists, the old root lock does not, and contention remains non-blocking; the
compiled hook lifecycle, affected packages, and serial uncached full suite all
pass with the amendment.

## Completion Ledger

Implemented the harness-neutral repository hook activation on Go/Cobra using
the existing incremental scanner and embedding coordinator. Loaded
`agent-reliability`, `context-injection`, and `completion-ledger`. Validation
included exact hook-template tests, human/JSON health tests, stable status
tests, a test-built CLI git lifecycle exercise, affected packages, and the
full repository suite.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Pre-commit runs lexical, aggregate code refresh, checkpoint, queue, staging in order | DONE | `internal/cli/next_hooks.go:317`, `internal/cli/next_hooks_test.go:175` — exact quiet/best-effort command order asserted |
| 2 | Post-merge runs lexical, aggregate code refresh, checkpoint only | DONE | `internal/cli/next_hooks.go:317`, `internal/cli/next_hooks_test.go:175` — exact order asserted with no queue or staging |
| 3 | Refresh failures remain silent/non-blocking and health-visible | DONE | `internal/cli/next_hooks.go:317`, `internal/cli/next_hooks_test.go:222`, `internal/cli/index_freshness_integration_test.go:17`, `internal/cli/scan_test.go:535` — every managed Hero command redirects stdout/stderr before `|| true`; a failing stub stays silent, a held lock remains health-visible, and the lock uses ignored `.hero/cache/` without dirtying the repository |
| 4 | Existing repository installer only; no post-commit/harness hook | DONE | `internal/cli/next_hooks.go:82` — only existing pre-commit/post-merge writer changed; boundary diff contains no target or harness files |
| 5 | Old block stale; regenerated hooks current | DONE | `internal/cli/next_hooks_test.go:351`, `internal/cli/next_hooks_test.go:385` — stale detector plus existing refresh path regenerates both hooks with the current command |
| 6 | Check reports actual changed/missing/deleted source coverage | DONE | `internal/cli/check.go:736`, `internal/cli/check_test.go:440` — checksum/cache comparison reports all three and reuses unchanged cached parse products (`reparsed=0`); changed/new parsing limitation is documented above |
| 7 | Check reports per-corpus embedding gaps read-only | DONE | `internal/cli/check.go:797`, `internal/cli/check_test.go:486`, `internal/cli/check_test.go:565` — missing, mismatch, authoritative orphan, unavailable, globally disabled, and code-scan-disabled/nil code-corpus skip covered while other corpora remain checked; JSON rows and row-count non-mutation asserted |
| 8 | Embeddings status shows stable per-corpus count/newest | DONE | `internal/cli/embeddings.go:123`, `internal/cli/embeddings_test.go:237` — all five corpora render canonically; empty is `count=0 newest=never` |
| 9 | Pre-commit add/change/delete converges graph/FTS/vector/hybrid | DONE | `internal/cli/index_freshness_integration_test.go:17` — real installed hook and test-built binary exercised all three transitions without post-bootstrap manual scan |
| 10 | Post-merge add/change/delete converges graph/FTS/vector/hybrid | DONE | `internal/cli/index_freshness_integration_test.go:17` — no-verify branch commits followed by real merges exercised all three transitions and four retrieval surfaces |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Update `internal/cli/next_hooks.go` managed blocks | DONE | Both hooks share lexical → aggregate incremental scan → checkpoint; pre-commit alone writes queue/stages, and all managed Hero commands redirect both output streams before best-effort normalization |
| 2 | Extend hook template/currency tests | DONE | `internal/cli/next_hooks_test.go` asserts exact order/redirection, real error silence with successful git commit, post-merge exclusions, stale/current regeneration, and existing preservation tests remain green |
| 3 | Add human/JSON code and embedding freshness rows | DONE | `internal/cli/check.go`, `check_test.go`, `check_json_test.go` implement categorized, read-only coverage with actual hashes/inventory and intentionally omit code-vector demand when code scanning is disabled/nil; inventory-only engine limitation is documented above |
| 4 | Extend embeddings status observability | DONE | `internal/cli/embeddings.go`, `embeddings_test.go` render stable corpus counts and newest timestamps without freshness claims |
| 5 | Add git lifecycle end-to-end coverage | DONE | `internal/cli/index_freshness_integration_test.go` builds Hero, installs hooks, covers commit/merge add/change/delete plus held-lock behavior |
| 6 | Update existing CLI help | DONE | Hook install help documents automatic best-effort deadline and the cache-rebuilding `hero scan --code` recovery command, with focused regression; embeddings status help distinguishes timestamps from freshness |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `go test ./internal/cli -run TestManagedHooksConvergeCodeIndexesAcrossCommitAndMerge -count=1 -v` built the CLI, installed generated hooks, committed/merged add-change-delete transitions, asserted graph/FTS/vector/hybrid identity, and proved a held lock leaves git successful with `hero check` warning. `TestManagedPreCommitSuppressesFailingHeroOutputAndAllowsCommit` additionally ran the installed block against a stdout/stderr-emitting failing Hero stub and observed a successful, noise-free commit.

### Excellence Bar self-check

- [x] Yes — the change reuses the existing managed hook and aggregate refresh
  seams, stays inside the declared source-code boundaries, exposes source truth
  rather than timestamps, and documents the scanner's missing
  inventory-only seam instead of hiding it.
