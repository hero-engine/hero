---
title: "Graph node identity is (type, key) — a sibling-repo ingest tombstones the local node"
slug: graph-node-identity-repo-scoped
type: bug
status: completed
domain: engineering
priority: high
severity: high
root_cause_class: design
tags: [graph, substrate, federation, repoKey, identity, schema, why, reliability]
created: 2026-07-25
depends-on: [graph-why-resolution-and-peer-spec-indexing]
delivery_method: manual
completed_at: 2026-07-25T19:24:06Z
---

# Graph node identity is (type, key) — a sibling-repo ingest tombstones the local node

## Kickoff

Paste into a fresh session to start delivery:

> Deliver `graph-node-identity-repo-scoped`. Graph node identity is
> `(type, key)` with no repo, enforced by
> `CREATE UNIQUE INDEX idx_nodes_current ON nodes(type, key) WHERE valid_to IS
> NULL` (`internal/graph/graph.go:179`). So when a sibling repo's corpus is
> ingested and carries the same slug, `UpsertNode`
> (`internal/graph/node.go:105-113`) matches the LOCAL row, sees a differing
> partition, and invalidates-and-reinserts it under the sibling's repoKey —
> permanently tombstoning the local node until something re-asserts it. Make
> node identity repo-scoped, and resolve the resulting ambiguity at every
> unscoped accessor (`GetNodeID`, `GetNode`, `GetNodeAt`, `InvalidateNode`,
> and edge resolution in `graph/sync.go`). Start by reading **Key Files**,
> then work the Acceptance Criteria in order. Close with the cold delivery
> audit and `hero spec verify`.

## Summary

The graph substrate can hold only **one live node per `(type, key)`**, across
every repo partition — the unique index enforces it. Repo is a *column*, not
part of identity. Federation, however, ingests sibling repos' corpora into the
same store under their own repoKeys, and slugs are not globally unique.

When a sibling ingest carries a slug the local repo also owns, `UpsertNode`
finds the local row by `(type, key)`, sees `existingRepo != n.Repo`, and takes
the invalidate-and-reinsert branch. The local partition's node is tombstoned
and replaced by the sibling's. Readers that filter on the local repoKey — which
is all of them, correctly — then find nothing.

## Issue

Observed as the `team-oauth` case in
`graph-why-resolution-and-peer-spec-indexing`, where direct DB inspection showed
every local row tombstoned and the only live row under a peer partition:

```
$ sqlite3 .hero/graph.db "SELECT repo, valid_to IS NULL AS live, count(*) \
    FROM nodes WHERE key='team-oauth' GROUP BY repo, live;"
  hero-engine/hero        | 0(tombstoned) | 15
  hero-engine/hero-cloud  | 0(tombstoned) | 14
  hero-engine/hero-cloud  | 1(live)       |  1
```

That spec hypothesized a cross-repo scan-path bug and scoped the fix as "audit
`scan.go:491-530`." A cold delivery audit of its delivery established the real
mechanism is schema-level identity, and the work was deferred here rather than
rushed into an unrelated delivery. See that spec's **Deferred Scope**.

## Root Cause

`internal/graph/graph.go:179`:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_current
    ON nodes(type, key) WHERE valid_to IS NULL
```

`internal/graph/node.go:105-113` — the current-row lookup that decides whether
an upsert is a no-op, an update, or an insert:

```sql
SELECT id, content_hash, props, repo, unit, domain
  FROM nodes
 WHERE type = ? AND key = ? AND valid_to IS NULL
```

No repo predicate. `node.go:151` then computes `partitionUnchanged` including
`existingRepo == n.Repo`; a mismatch falls through to the invalidate branch at
`:168`, which also invalidates every edge hanging off the prior node.

## Key Files

- `internal/graph/graph.go:179` — the unique index that makes this a schema
  invariant; changing it is the migration.
- `internal/graph/node.go:74-205` — `UpsertNode`: the current-row lookup,
  the `partitionUnchanged` check, and the invalidate-and-reinsert branch.
- `internal/graph/node.go:244` `GetNodeID`, `:220` `GetNode`, `:208`
  `GetNodeAt`, `:259` `InvalidateNode` — unscoped accessors that become
  ambiguous once two live rows can share `(type, key)`.
- `internal/graph/sync.go:201-206` — edge resolution by `(type, key)`; would
  bind an edge to an arbitrary repo's node.
- `internal/traversal/why.go:108-130` — `resolveTarget`, already repo-scoped
  with `ORDER BY (repo = ?) DESC`; the model the accessors should follow.
- `internal/graph/alias.go:18`, `internal/digest/digest.go:782`,
  `internal/sessions/graph_ingest.go:96`, `internal/tasks/record.go:165,176`,
  `internal/acceptance/record.go:209`, `internal/extract/decisions.go:83` —
  `GetNodeID` callers needing a repo decision.

## Goal

A sibling repo's ingest must never tombstone or re-key another repo's node.
Two repos owning the same slug must each keep a live node in their own
partition, and every accessor must resolve to the intended partition rather
than whichever row happens to come back first.

## Suggested Fix Approach

1. **Repo-scope the identity index.** `idx_nodes_current` becomes
   `(type, key, repo)`. Needs a migration path for existing `graph.db` files —
   note that `graph.db` is a regenerable cache, so rebuild-on-migrate is a
   legitimate option and probably simpler than a data migration.
2. **Repo-scope the upsert lookup**, preserving the v1→v2 backfill: match
   `repo = ?`, fall back to a legacy `repo = ''` row, and never match a
   *different* non-empty repo. `ORDER BY (repo = ?) DESC LIMIT 1`.
3. **Resolve accessor ambiguity.** Decide per accessor whether to take a repo
   argument or to prefer-local like `resolveTarget`. Prefer an explicit repo
   parameter where the caller knows it; the compiler then finds every site.
4. **Edge resolution** must bind within a partition, or explicitly across one
   when the edge is genuinely federated (the boundary-handoff case
   `traversal` already models).

## Acceptance Criteria

- **AC-1:** WHEN a node is upserted under repo B with a `(type, key)` that
  already has a live node under repo A THE SYSTEM SHALL leave repo A's node
  live and create a separate live node for repo B.
- **AC-2:** WHEN a node is upserted twice under the same repo with unchanged
  content THE SYSTEM SHALL remain idempotent (no new history rows) — the
  existing `UpsertNode` idempotency contract is preserved.
- **AC-3:** WHEN a legacy row carries an empty `repo` and is upserted by a
  writer that now stamps a repoKey THE SYSTEM SHALL upgrade that row in place
  rather than leaving a duplicate live row (the v1→v2 backfill, preserved).
- **AC-4:** WHEN an edge is written between two nodes THE SYSTEM SHALL resolve
  each endpoint within the intended repo partition, not an arbitrary one.
- **AC-5:** THE SYSTEM SHALL migrate an existing `graph.db` to the repo-scoped
  identity without leaving duplicate live rows or orphaned edges.
- **AC-6:** WHEN a sibling repo's corpus is ingested THE SYSTEM SHALL leave
  `hero why <local-slug>` resolving the local spec — the `team-oauth`
  regression, asserted end-to-end.

## Boundaries

- **In scope:** node identity, the unique index and its migration, the upsert
  lookup, the unscoped accessors, and edge endpoint resolution.
- **Out of scope:** the read-side reconcile and repoKey derivation (delivered
  in `graph-why-resolution-and-peer-spec-indexing`); the wider "one
  self-healing store" direction; changing what federation ingests.

## Changes

- `internal/graph/graph.go` — migration v5: `idx_nodes_current` moves from
  `(type, key)` to `(type, key, repo)`; `schemaVersion` bumped to "5".
- `internal/graph/node.go` — `repoPredicate` / `repoOrder` / `repoScope`
  helpers; `UpsertNode`'s current-row lookup repo-scoped; `GetNodeID`,
  `GetNode`, `GetNodeAt`, `InvalidateNode` take a repo argument.
- `internal/graph/alias.go`, `internal/graph/sync.go` — alias and pulled-edge
  endpoints resolve within a partition.
- Callers threaded with their repoKey: `internal/sessions/graph_ingest.go`,
  `internal/tasks/record.go`, `internal/acceptance/record.go`,
  `internal/spec/graph_ingest.go`, `internal/extract/decisions.go`,
  `internal/gitutil/graph_ingest.go`, `internal/digest/digest.go`,
  `internal/handoff/handoff.go`, `internal/cli/task.go`,
  `internal/cli/graph_edge.go`.
- `internal/attention/mail/promotion.go` — `writeMailProvenance` keyed spec
  nodes by `cfg.PeerID` instead of `gitutil.RepoKey`; corrected.
- Tests: `internal/graph/node_identity_test.go` (new),
  `internal/cli/why_resolution_test.go`,
  `internal/traversal/why_federation_test.go`,
  `internal/cli/verify_test.go`, `internal/attention/mail/triage_test.go`.

## Risks

- **Blast radius.** Identity is the substrate's most load-bearing invariant;
  ~8 call sites bind to it. Mitigation: take an explicit repo parameter so the
  compiler enumerates the sites rather than relying on review.
- **Silent duplicate live rows.** Relaxing the unique index before every
  accessor is repo-aware could turn a loud tombstone into a quiet wrong-node
  read. Mitigation: land the accessor changes in the same change as the index,
  and add an invariant test that no two live rows share `(type, key, repo)`.
- **Migration on a live `graph.db`.** Mitigation: `graph.db` is regenerable —
  prefer rebuild-on-migrate over an in-place data migration.

## Completion Ledger

**Task as executed.** Scaffolded as a deferred follow-up while delivering
`graph-why-resolution-and-peer-spec-indexing`, then delivered in the same
session on the user's explicit decision to implement it now rather than
accept the descope.

**Validation performed.**
- `go build ./...`, `go vet ./...` — clean. `go test ./...` — all 115 packages
  green, 0 failures.
- `gofmt -l`: this change introduced no newly-unformatted file. Six touched
  files are unformatted, and all six were already unformatted at HEAD
  (`internal/acceptance/record.go`, `internal/extract/decisions.go`,
  `internal/spec/graph_ingest.go`, and three `_test.go` files) — a
  pre-existing repo-wide condition, verified file-by-file against `HEAD`.
- **Pre-fix falsification:** reverting the v5 index and the upsert's repo
  scoping fails 7 tests — `TestSiblingRepoIngestDoesNotTombstoneLocal`,
  `TestInvalidateNodeIsPartitionScoped`, `TestGetNodeAtIsPartitionScoped`,
  `TestSchemaIndexIsRepoScoped`, `TestEdgeEndpointsResolveWithinPartition`,
  `TestMigrationV5RepoScopesExistingDatabase`, and traversal's
  `TestResolveTarget_FederatedPeerCopyDoesNotShadowLocal`.
- **Live end-to-end on this repo's real 6.5k-node `graph.db`** — see Exercise.

**Design note — why `repo == ""` still matches any partition.** The lookup
rule is: a non-empty repo matches that partition or an unpartitioned
(`repo = ''`) row, preferring its own, and *never* another repo's; an empty
repo matches anything, exactly as before v5. That makes adoption incremental —
a caller that cannot know its partition keeps pre-v5 semantics instead of
silently losing its nodes — and it preserves the v1→v2 backfill, where a
writer that now stamps a repoKey must upgrade a legacy `repo = ''` row in
place rather than leave a duplicate live row (AC-3).

**Two pre-existing defects surfaced by this change and fixed with it:**
1. `writeMailProvenance` (`internal/attention/mail/promotion.go`) wrote the
   whole local spec corpus under `cfg.PeerID` — a UUID — instead of
   `gitutil.RepoKey`, so those nodes sat in a partition no reader queries.
   Before v5 this was masked because the mis-keyed write simply clobbered the
   correctly-keyed one; after v5 the two would coexist as duplicate live rows,
   so fixing the derivation was required for coherence, not optional.
   `internal/attention/mail/triage_test.go` asserted the buggy partition and
   was corrected.
2. `resolveSpecGraphID` (`internal/cli/task.go`) upserted its parent-spec stub
   with no `Repo` at all, landing it in the unpartitioned bucket.

**A test that encoded the bug as the contract.**
`TestResolveTarget_FederatedPeerCopyDoesNotShadowLocal` was written during the
parent spec's earlier landing to characterize this behavior — its own doc
comment said the invariant "tombstones the local partition's copy," and it
asserted that a local query then *fails*. That is the team-oauth bug pinned as
if it were intended. Rewritten to assert what the comment actually wished for:
after a sibling ingest the local node is still live and still what a local
query resolves to.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Upsert under repo B leaves repo A's node live and creates a separate live node for B | DONE | `internal/graph/node.go` — repo-scoped current-row lookup in `UpsertNode`; `InvalidateNode` scoped via subquery so it can't cross partitions. Tests: `TestSiblingRepoIngestDoesNotTombstoneLocal`, `TestInvalidateNodeIsPartitionScoped` |
| 2 | Same-repo re-upsert of unchanged content stays idempotent | DONE | Partition check unchanged for same-repo writes. Test: `TestSameRepoUpsertStaysIdempotent` (no-op on unchanged, single live row after a supersede) |
| 3 | A legacy `repo = ''` row is upgraded in place, not duplicated | DONE | `repoPredicate`'s `OR repo = ''` fallback with `repoOrder` preferring the exact partition. Test: `TestLegacyUnpartitionedRowIsUpgradedInPlace` |
| 4 | Edge endpoints resolve within the intended partition | DONE | `alias.go` (`ResolveAlias`/`MakeAlias` take repo) and `sync.go` (pulled edges resolve under `e.Repo`). Test: `TestEdgeEndpointsResolveWithinPartition` |
| 5 | Existing `graph.db` migrates with no duplicate live rows or orphaned edges | DONE | Migration v5 drops and recreates the index; widening a unique index cannot fail on existing data. Test: `TestMigrationV5RepoScopesExistingDatabase` rewinds a real store to the v4 shape and reopens. Also verified on this repo's actual 6.5k-node graph — see Exercise |
| 6 | `hero why <local-slug>` still resolves after a sibling repo ingests the same slug | DONE | Test: `TestWhySurvivesSiblingRepoIngest` drives the real command and also asserts it does not resolve the *sibling's* node. Confirmed live against the reported `team-oauth` case — see Exercise |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `internal/graph/graph.go` — migration v5, schemaVersion "5" | DONE | v1's index statement left intact so each migration stays valid at its own point in the timeline |
| 2 | `internal/graph/node.go` — scoping helpers + repo-scoped upsert and accessors | DONE | `repoPredicate`/`repoOrder`/`repoScope`; `GetNodeAt` uses the predicate only, so bitemporal ordering isn't displaced by partition preference |
| 3 | `alias.go`, `sync.go` — partition-scoped endpoint resolution | DONE | |
| 4 | 10 caller packages threaded with their repoKey | DONE | Signature change chosen deliberately so the compiler enumerated every site rather than relying on review |
| 5 | `promotion.go` — repoKey derivation corrected from `cfg.PeerID` | DONE | See preamble; required for coherence under v5 |
| 6 | Tests | DONE | 13 in `node_identity_test.go` (new file), 1 in `why_resolution_test.go`, 3 fixtures corrected, ~12 signature-follow updates |

### Exercise-the-feature check

- [x] Exercised against this repo's **real** `graph.db` (6,579 live nodes
  across 4 partitions) with a binary built from this tree:
  - Migration applied cleanly: `idx_nodes_current` is now
    `ON nodes(type, key, repo)`; **0** duplicate live rows per
    `(type, key, repo)`; **0** orphaned live edges.
  - The reported `team-oauth` case is fixed: it previously had *every*
    `hero-engine/hero` row tombstoned and its only live row under
    `hero-engine/hero-cloud`. It now has a **live node in
    `hero-engine/hero`**, with the cloud copies tombstoned.
  - `hero why team-oauth` — the exact command from the bug report — resolves:
    `# "Team OAuth — GitHub/Google SSO for Team Server Authentication"
    \`team-oauth\` (Feature)`.
- [x] `hero why graph-why-resolution-and-peer-spec-indexing` resolves,
  confirming no regression to the read path the parent spec delivered.

**Post-audit corrections (round 2).** A cold audit returned HOLD with two
blockers and several accuracy findings. All are fixed:

1. **My own AC-6 regression test passed on the broken code.** `hero why`
   reconciles the spec subgraph from disk before resolving, so the second
   command run re-asserted the local node and succeeded even with identity
   unscoped — the test never exercised the bug it was named for.
   `TestWhySurvivesSiblingRepoIngest` now asserts directly against the store,
   between the sibling ingest and any reconcile, and fails pre-fix with
   `local live rows = 0, want 1`.
2. **The original bug was still reachable from the unpartitioned side.**
   Scoping the READ rule was not enough: an upsert with an empty `Repo`
   preferred a repo-stamped row, saw a partition mismatch, and tombstoned it.
   Ten production writers still upsert without stamping a Repo, so this was
   live. Introduced `repoWriteScope`, deliberately stricter than the read
   rule — an unpartitioned write matches only `repo = ''`. Tests:
   `TestUnpartitionedWriteDoesNotClobberStampedNode`,
   `TestUnpartitionedWriteWithBothRowsLiveDoesNotError` (the latter covers a
   hard `UNIQUE constraint failed` the read rule could produce — impossible
   pre-v5). Both reproduce the audit's findings against the read-rule version.
3. **`GetNodeAt` could answer a scoped query with a newer unpartitioned row.**
   Now orders by exact partition first, then bitemporal recency.
4. **v5 is the first non-additive migration**, so a pre-v5 binary writing to a
   v5 graph reintroduces tombstoning.
5. Two inaccurate ledger claims (gofmt, counts) corrected above.

**Post-audit corrections (round 3).** The re-audit confirmed rounds 1–2 closed
and all six ACs sound, and found two of the *remediations* had unsound
evidence. Both fixed:

- **I repeated the round-1 mistake on #3.**
  `TestGetNodeAtPrefersExactPartitionOverRecency` passed with its own fix
  reverted: both rows came from back-to-back `upsert()` calls and
  `nowRFC3339()` is second-precision, so they shared an identical
  `valid_from` — there was no recency gap for the partition preference to beat,
  and my comment claiming one was simply wrong. Rewritten with explicit
  `ValidFrom` values a decade apart (via a new `mustUpsert` helper); it now
  fails on the reverted code with `GetNodeAt(...).Repo = "" — a newer
  unpartitioned row answered a scoped query`. The fix was always correct; the
  evidence for it was not.
- **#4 was inert and has been removed.** The v5 risk paragraph was gated on
  `schemaLess(binarySchema, "5")`, but `binarySchema` is always the
  compile-time constant `schemaVersion` — so in any shipped binary the branch
  is dead. Worse, the binary that would actually cause the damage is running
  its *own older copy* of `checkSchemaMismatch` and could never print it. The
  test passed only by calling the unexported function with an argument pair
  production cannot produce. Dead mitigation code that reads as a safeguard is
  worse than none, so the branch and its test are gone, replaced by a comment
  recording why the guard cannot live there. The warning-vs-error call stands.
- Overstated comment on `TestUnpartitionedWriteWithBothRowsLiveDoesNotError`
  corrected: the UNIQUE collision needs state `UpsertNode` alone cannot
  produce, so the test pins the reachable half (the write stays in its own
  partition and both rows survive).

  *This edit silently failed the first time and I reported it as done anyway* —
  the round-3 audit caught it. The source text had been reformatted (`''` had
  become typographic quotes), so an exact-match `str.replace` matched nothing
  and returned the file unchanged. Three consecutive rounds carried a
  self-reported remediation that did not match the tree; the common cause was
  patching by unchecked string replacement. Re-applied with an editor that
  fails loudly on a miss, and verified in the file.

### Excellence Bar self-check

Yes — with the caveat that it took two audit rounds to get there, and the
first round shipped a regression test that passed on the broken code. That is
the failure this project keeps hitting, and the reason the cold audit is worth
its cost. The change was scoped by what the compiler could prove rather than what
review could catch: taking a repo argument on four accessors forced all 23
call sites to be examined individually, which is how the two latent
mis-partitioned writers (`writeMailProvenance`, the task stub) were found at
all. Every acceptance criterion was falsified against the pre-fix code, the
migration was tested by rewinding a real store to the old schema rather than
by inspection, and the fix was confirmed on the actual reported data. The one
test that encoded the bug as the contract was rewritten rather than deleted,
with the reason recorded.
