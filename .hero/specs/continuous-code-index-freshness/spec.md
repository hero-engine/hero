---
title: "Continuous Code-Index Freshness"
slug: continuous-code-index-freshness
type: initiative
status: completed
domain: engineering
size: large
priority: high
horizon: now
autonomy: autonomous
created: 2026-07-27
tags: [code-index, codescan, graph, embeddings, git-hooks, retrieval, freshness]
child:
  - incremental-code-graph-refresh
  - embeddings-never-refresh-on-commit
  - hook-driven-index-freshness
completed_at: 2026-07-27T20:36:47Z
---

# Continuous Code-Index Freshness

## Vision

Hero's repository-managed lifecycle keeps every retrieval leg current. After
local commits and merges, the configured source tree, generated code knowledge,
current `Package`/`File`/`Symbol` graph nodes, lexical graph projection, and code
embeddings converge without a developer remembering to run `hero scan`.

## Goal

After local commits and merges, code symbols, lexical graph projection, and
code embeddings reflect the configured source tree without manual
`hero scan`. The implementation reuses the existing scanner, cache, graph,
embedding, hook, and health surfaces; hook failures remain best-effort and
never block git.

## Kickoff

Makes Hero's code graph, lexical projection, and configured embeddings refresh
automatically after commits and merges using the existing incremental scanner
and repository-managed git hooks.

**Status:** completed — all three children passed their delivery gates and the
initiative auto-completed after hook-driven freshness verified cleanly.

**Pick up at:** run the final local build and commit the complete initiative to
main.

→ `.hero/specs/continuous-code-index-freshness/spec.md`

**Files:** `internal/cli/scan.go`, `internal/codescan/scanner.go`,
`internal/codescan/graph_ingest.go`, `internal/embeddings/refresh.go`,
`internal/cli/next_hooks.go`
**Skip:** watcher daemons, harness session hooks, git-diff symbol mapping,
parallel scan configuration, search-time writes, and deep/LLM hook enrichment.

## Context

`hero scan` already has every expensive building block this initiative needs:

- `config.CodeScanConfig`, `resolveParser`, and scanner excludes define the
  configured source tree once.
- `.hero/knowledge/code/.checksums.json` and `codescan.ScanCache` let
  `Scanner.Scan` reparse changed files while returning a complete merged
  `Result`.
- `codescan.GenerateKnowledge`, `codescan.WriteGraph`, and
  `index.ProjectGraphNodes` produce the generated, graph, and lexical views.
- `embeddings.Refresh`, its storage, and its chunkers already own semantic
  extraction and persistence.
- `internal/cli/next_hooks.go` already installs and currency-checks one managed
  block in `pre-commit` and `post-merge`.
- `hero check` and `hero embeddings status` are the existing diagnostic
  surfaces.

The gap is orchestration and authoritative deletion handling, not parsing.
`codescan.WriteGraph` currently upserts the complete result but never retires
codescan-owned nodes absent from it. Embeddings are only refreshed by manual
commands. The managed hooks refresh project handoff state and part of the
lexical index, but not the complete code retrieval pipeline.

This is mission-critical context hygiene: a fresh agent cannot start smarter
when recent or deleted code is invisible to the corpus it receives.

## Workstreams and waves

| Wave | Child | Type | Priority | Depends on | Outcome |
|---|---|---|---|---|---|
| 1 | [`incremental-code-graph-refresh`](incremental-code-graph-refresh/spec.md) | enhancement | critical | — | One reusable quiet incremental scan path updates generated code knowledge, reconciles the local codescan graph, and projects current graph nodes into FTS. |
| 2 | [`embeddings-never-refresh-on-commit`](embeddings-never-refresh-on-commit/spec.md) | bug | high / severity high | Wave 1 | Embedding refresh becomes hash-first, transactional, deadline-safe, and authoritative-prune aware; the incremental path refreshes configured corpora including `code`. |
| 3 | [`hook-driven-index-freshness`](hook-driven-index-freshness/spec.md) | enhancement | medium | Waves 1 and 2 | Existing pre-commit/post-merge managed blocks activate the ordered pipeline, while check/status and end-to-end tests make drift visible. |

## Recommended delivery order

1. Deliver `incremental-code-graph-refresh`. Deleted symbols must disappear
   from authoritative `graph.db` before the embedding layer is allowed to
   treat code extraction as complete.
2. Deliver `embeddings-never-refresh-on-commit`. It depends on the first
   child's graph authority and extends the same incremental command only after
   graph reconciliation and lexical projection succeed.
3. Deliver `hook-driven-index-freshness`. Activation comes last so git hooks
   never invoke an expensive, destructive, or partially designed path.

No child is safely skippable. Combining the first two would leave one oversized
engine spec spanning scan orchestration, graph deletion, vector storage, and
CLI behavior; combining the second and third would mix engine safety with
activation and observability. Three medium children are the smallest coherent
shape.

## In-flight overlap watch

`incremental-code-graph-refresh` and
`embeddings-never-refresh-on-commit` both change `internal/cli/scan.go` and
scan integration fixtures: the first factors the code-refresh orchestration;
the second extends that shared command after graph projection to call the
new embedding engine contract. They MUST NOT be delivered concurrently.
Their reciprocal `conflicts-with` relations are the machine-readable form of
this seam.

There are no conflict edges involving `hook-driven-index-freshness`.
That child wholly owns `internal/cli/next_hooks.go`, hook currency tests,
`internal/cli/check.go`, embeddings status rendering, and hook end-to-end
tests. The first two children explicitly exclude those files.

## Approach

The source tree is authoritative; git diff is not. The pipeline inventories
and hashes the files selected by the existing `CodeScanConfig`, parser
selection, and excludes. The complete merged `codescan.Result` is the keep-set
for generated knowledge and codescan-owned graph nodes.

The activation pipeline is ordered:

1. refresh spec/knowledge lexical state with existing
   `hero index --if-stale -q`;
2. run the quiet, deadline-bounded incremental code command;
3. inside that command, scan, reconcile code graph nodes, project current
   graph nodes into FTS, then refresh configured embeddings including `code`;
4. run the existing NEXT/queue projection work appropriate to the hook.

Destructive reconciliation and pruning occur only after a complete,
authoritative extraction. A deadline, read error, missing source, parse
failure, or partial pass never retires graph nodes, prunes vector chunks, or
advances checksum/cache state. Successful no-change runs do no graph,
knowledge, FTS, or vector writes.

## Boundaries

- No watcher or daemon.
- No Claude/Codex or other harness session hook. Repository-managed git hooks
  cover `opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic`
  without target-specific behavior.
- No new parser, git-diff-to-symbol mapper, file-level embedding scope, or
  parallel code-scan configuration.
- No inline mutation from `hero search`, hybrid retrieval, MCP search, or
  other read paths.
- No deep-mode or LLM enrichment from hooks. Existing manual deep scan remains
  manual.
- Defer the broken event chunker, unexplained chunk-ID collision, general
  unbounded SQL-parameter cleanup beyond the batching required here, and the
  separate Completion Ledger governance defect.

## Shared risks and rollback

| Risk | Mitigation | Rollback implication |
|---|---|---|
| Hook latency or lock contention disrupts normal git use. | One deadline spans the quiet command; unchanged work exits before writes; hook calls remain `|| true`. Measure pre-commit and post-merge latency. | Remove the new command line from the managed template and regenerate hooks. Manual `hero scan` remains valid. |
| A partial scan is mistaken for authority and deletes valid graph/vector rows. | Completion and authority are explicit phase results. Retirement/pruning and scan-state commit are last-success actions, never timeout cleanup. | Bitemporal graph history remains recoverable; a full `hero scan` reconstructs current rows and embeddings. |
| Scanner cache advances while graph or projection is stale. | Persist `.checksums.json` and `ScanCache` only after the complete code refresh succeeds; retries remain idempotent. | Delete only the two generated cache artifacts to force a full parse, then run `hero scan`. |
| Deletion reconciliation crosses repository or writer boundaries. | Match `Repo`, `Package`, `File`, and `Symbol` by the local repo partition and `source.kind=codescan`; do not retire other writers or sibling repos. | Disable the incremental hook line and rebuild from a manual full scan; historical rows remain bitemporal. |
| Existing installed hooks do not contain the new pipeline. | Reuse byte-comparison currency detection; upgrades/install-hooks regenerate both managed blocks. | The old managed block remains functional but stale and `hero check` reports it until refreshed. |

## Validation

- Lint the initiative and all three child specs, then verify their
  parent/child, dependency, and reciprocal conflict relations.
- Require each child to pass its focused, affected-package, and full-suite
  checks plus the cold delivery audit and `hero spec verify` closing gate.
- Prove add, change, and delete behavior through both pre-commit and post-merge
  paths across the graph, FTS, vector, and hybrid retrieval surfaces without a
  post-change manual scan.
- Run managed-hook currency tests, `hero check`, and `git diff --check`.

## Acceptance Criteria

- **AC-1:** WHEN all three children are delivered THE SYSTEM SHALL refresh code graph nodes, lexical graph projection, and configured code embeddings after local commits and merges without a manual `hero scan`
- **AC-2:** IF any incremental extraction is partial, unavailable, errored, or deadline-truncated THEN THE SYSTEM SHALL preserve the prior graph, vector keep-set, and persisted scan state rather than treating the partial result as authoritative
- **AC-3:** THE SYSTEM SHALL use the same repository-managed git-hook behavior for `opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic` without a harness-specific session hook
- **AC-4:** THE SYSTEM SHALL keep manual `hero scan` behavior on the same scanner, knowledge, graph, projection, and embedding code paths used by automatic refresh

## Progress

- Initiative composed and all three child specs fully designed on 2026-07-27.
- All three children passed their delivery gates and the initiative
  auto-completed on 2026-07-27.
- Final compiled-binary rollout validation moved the workspace refresh lock
  from tracked `.hero/code-refresh.lock` to the existing ignored
  `.hero/cache/code-refresh.lock` seam. Focused lifecycle, affected-package,
  and serial uncached full-suite validation pass after the amendment.
