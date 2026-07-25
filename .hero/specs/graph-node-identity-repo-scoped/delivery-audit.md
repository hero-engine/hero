# Delivery audit — graph-node-identity-repo-scoped

**Audited:** `git diff HEAD -- internal/` on `fix/graph-why-residual-peer-ingest` (uncommitted), plus untracked `internal/graph/node_identity_test.go`
**Verdict:** SHIP

_(Reached at round 3; see the Round 3 section. The verdict line is kept as a
bare token because `hero spec verify` Gate 2 parses it literally.)_
**Surface:** noteworthy

> **Rounds 2 and 3 appended at the bottom.** All six acceptance criteria are
> genuinely delivered and independently falsified. Every substantive defect
> found in rounds 1 and 2 is closed and verified. One residual: the ledger
> claims a test doc-comment was corrected; it was not.

Re-run independently: `go build ./...` (clean), `go vet ./...` (clean),
`go test ./...` (60 packages, 53 with tests, 0 failures). Falsification and
live-corpus claims re-executed from scratch; working tree restored byte-for-byte
(sha1 verified) afterwards.

## Acceptance criteria

- [~] **AC-1** Upsert under repo B leaves repo A's node live — holds when B is a
  *non-empty* repo (`internal/graph/node.go:231-236`, `TestSiblingRepoIngestDoesNotTombstoneLocal`).
  **Does not hold when B is the empty partition.** Reproduced: a live row under
  `hero-engine/hero` was tombstoned and replaced by a `repo=''` row. See Audit
  note 2.
- [✓] **AC-2** Same-repo re-upsert stays idempotent — `partitionUnchanged` path
  unchanged (`internal/graph/node.go:159-174`); `TestSameRepoUpsertStaysIdempotent`
  asserts both the no-op and the single-live-row-after-supersede case.
- [✓] **AC-3** Legacy `repo=''` row upgraded in place — `repoPredicate`'s
  `OR repo = ''` plus `repoOrder`'s exact-partition preference
  (`internal/graph/node.go:235,243`); `TestLegacyUnpartitionedRowIsUpgradedInPlace`
  asserts exactly one live row survives.
- [✓] **AC-4** Edge endpoints resolve within the intended partition —
  `internal/graph/alias.go:20,55-60` (`ResolveAlias`/`MakeAlias` take repo),
  `internal/graph/sync.go:206,211` (pulled edges resolve under `e.Repo`);
  `TestEdgeEndpointsResolveWithinPartition`.
- [✓] **AC-5** Existing `graph.db` migrates without duplicate live rows or
  orphaned edges — `internal/graph/graph.go:266-281`;
  `TestMigrationV5RepoScopesExistingDatabase` rewinds a real store to the v4
  index and reopens. Confirmed independently against the live 6,579-node
  `.hero/graph.db`: 0 duplicates, 0 orphaned live edges.
- [~] **AC-6** `hero why <local-slug>` still resolves after a sibling ingest —
  the *behavior* is genuinely fixed and independently confirmed on live data
  (`team-oauth` now has a live row under `hero-engine/hero`). But the test the
  ledger cites as its evidence, `TestWhySurvivesSiblingRepoIngest`
  (`internal/cli/why_resolution_test.go:325-374`), **passes with the fix
  reverted** — it does not pin the regression. See Audit note 1.

## Changes

- [✓] `internal/graph/graph.go` — migration v5, `schemaVersion` "5"
  (`:150-157`, `:266-281`). v1's `(type, key)` statement correctly left intact
  so each migration remains valid at its own point in the timeline. Migration is
  idempotent and safe on a partially-applied db: `DROP INDEX IF EXISTS` +
  `CREATE UNIQUE INDEX IF NOT EXISTS`, and `schema_version` is only written after
  both statements succeed, so an interrupted run replays cleanly.
- [✓] `internal/graph/node.go` — `repoPredicate`/`repoOrder`/`repoScope`
  (`:231-252`); scoped `UpsertNode` lookup (`:122-128`); repo argument on
  `GetNodeAt`/`GetNode`/`GetNodeID`/`InvalidateNode` (`:262`, `:281`, `:295`,
  `:316`). Argument ordering verified correct at all four call sites — the
  fragments are appended after the caller's leading args and the arg slices
  match. `GetNodeAt` correctly injects the predicate *before* `ORDER BY
  valid_from DESC LIMIT 1`, so bitemporal ordering is not displaced.
- [✓] `internal/graph/alias.go`, `internal/graph/sync.go` — partition-scoped
  endpoint resolution. (`sync.go` also carries unrelated gofmt reflow of a doc
  comment and the `SyncConflict` struct — cosmetic churn, no behavior change.)
- [✓] 10 caller packages threaded with their repoKey. All 23 sites reviewed
  individually against the repo their corresponding *write* uses; one residual
  mismatch found — see Audit note 5.
- [✓] `internal/attention/mail/promotion.go:286` — `cfg.PeerID` →
  `gitutil.RepoKey(s.root)`. `s.root` is the projectRoot
  (`internal/attention/mail/service.go:62,71-72`), so the derivation is correct.
- [✓] Tests — `internal/graph/node_identity_test.go` (new, 9 tests + 2 helpers),
  `internal/cli/why_resolution_test.go` (1 new), 3 fixtures corrected.

## Test-fixture scrutiny

All three changed fixtures were checked against git history for a bent-to-fit
regression. **None of the three is bent.**

- `internal/traversal/why_federation_test.go` — **honest rewrite.** The version
  at HEAD asserted `if _, _, err := resolveTarget(store, localRepo, slug); err
  == nil { t.Errorf(...) }` — i.e. it *required* a local query to fail after a
  sibling ingest. That is the team-oauth bug pinned as the contract, and the doc
  comment said so out loud ("tombstones the local partition's copy"). The
  rewrite's assertions strictly subsume the old test's genuine intent: it now
  asserts the local query resolves to `localID` with the local title, which
  implies the old "must not return the peer copy" property. The
  peer-resolves-its-own-copy assertion is retained unchanged. No assertion was
  lost.
- `internal/cli/verify_test.go` — **legitimate.** `seedGraphCriterion`'s
  hard-coded `"test-repo"` only ever worked because the Criterion lookup was
  unscoped; production writes Criterion under `gitutil.RepoKey(projectRoot)`
  (`internal/spec/graph_ingest.go`, `internal/acceptance/record.go`). The fixture
  was seeding a partition production never queries. Minor weakening:
  `getGraphCriterionStatus` now reads with `repo=""` (match-any), so it would no
  longer catch a writer landing in the wrong partition.
- `internal/attention/mail/triage_test.go` — **legitimate; follows a real
  production fix.** The test asserted `traversal.Why(store, "peer_b", ...)`
  because `writeMailProvenance` wrote the local spec corpus under `cfg.PeerID` (a
  UUID). Writing local specs into a peer-UUID partition is unambiguously wrong —
  no reader filters on it. The production change came first and the test followed
  it, which is the right order.

## Open items

- **AC-6 test does not falsify** — `TestWhySurvivesSiblingRepoIngest` — the
  ledger presents it as the AC-6 regression test; it passes on the pre-fix code.
  Assessment: **performative evidence**. The AC itself is satisfied (confirmed on
  live data), but the automated guard claimed for it does not exist.
- **AC-1 hole for empty-repo writers** — not disclosed in the ledger.
  Assessment: **concrete, reproduced**.

## Audit notes

1. **AC-6's regression test passes on the broken code.** I reverted the v5 index
   and the `UpsertNode` repo scoping and re-ran the full named set. The ledger's
   7 named tests all failed exactly as claimed — `TestSiblingRepoIngestDoesNotTombstoneLocal`,
   `TestInvalidateNodeIsPartitionScoped`, `TestGetNodeAtIsPartitionScoped`,
   `TestSchemaIndexIsRepoScoped`, `TestEdgeEndpointsResolveWithinPartition`,
   `TestMigrationV5RepoScopesExistingDatabase`, and traversal's
   `TestResolveTarget_FederatedPeerCopyDoesNotShadowLocal`. But
   `TestWhySurvivesSiblingRepoIngest` **PASSED** on the same reverted tree. The
   parent spec's read-side reconcile re-asserts the local node at query time, so
   the test's second `hero why` never exercises the identity bug. The ledger's
   falsification list names 7 tests and omits this one without comment, while the
   AC-6 row cites it as the evidence. Either the test needs to bypass the
   reconcile, or AC-6's evidence should be stated honestly as live-corpus
   confirmation only.

2. **An empty-repo writer can still tombstone a repo-stamped node — the original
   bug shape, one partition over.** `repoOrder("")` is
   `ORDER BY (repo <> '') DESC, repo ASC LIMIT 1` (`internal/graph/node.go:241`),
   so an unpartitioned writer *prefers* a repo-stamped row, sees
   `partitionUnchanged == false`, and invalidates-and-reinserts under `repo=''`.
   Reproduced directly: upsert `(Feature, alpha, "hero-engine/hero")` then
   `(Feature, alpha, "")` leaves a single live row under `repo=''` — the stamped
   row is tombstoned. Preferring `repo = ''` for an empty-repo writer would be the
   non-destructive choice and would still satisfy AC-3. This is reachable: 10
   production `UpsertNode` sites still write with no `Repo` —
   `internal/tracker/graph_ingest.go:54,104,131,166,267,281`,
   `internal/gitutil/graph_ingest.go:109,188`,
   `internal/acceptance/record.go:227`, `internal/extract/decisions.go:166` — and
   the live `.hero/graph.db` still holds 16 live nodes in the empty partition.

3. **Reachable hard error: `UNIQUE constraint failed: nodes.type, nodes.key,
   nodes.repo`.** Follows from note 2. Once a live `repo=''` row and a live
   stamped row coexist for one `(type, key)` — reachable with ≥2 repo partitions
   plus any empty-repo writer — an empty-repo upsert selects the stamped row,
   tombstones it, then fails to insert because the live `repo=''` row is still
   there. Reproduced. The transaction rolls back (`defer tx.Rollback()`), so
   there is no corruption, but `UpsertNode` returns a hard error and the ingest
   path fails. Pre-v5 this was impossible — only one live row could exist. No
   test covers it; the spec's Risks section anticipated "silent duplicate live
   rows" but not this failure.

4. **`GetNodeAt` can answer a repo-scoped query with an unpartitioned row.**
   By design it drops `repoOrder` to protect bitemporal ordering
   (`internal/graph/node.go:264-266`) — a defensible tradeoff, and the ledger
   discloses it. But the consequence is that a *newer* `repo=''` row outranks the
   exact-partition row on `valid_from DESC`. Reproduced:
   `GetNodeAt("Feature", "eps", t, "hero-engine/hero")` returned the `repo=""`
   row. `TestGetNodeAtIsPartitionScoped` does not cover this — despite a doc
   comment claiming it verifies "bitemporal ordering as its primary sort rather
   than partition preference", it makes a single call at `now` with no historical
   rows and asserts only that a *different non-empty* repo's row isn't returned.

5. **Residual read/write partition mismatch in `internal/acceptance/record.go`.**
   `resolveOrStubCommit` now looks up `Commit` with `repoKey` (`:209`) but the
   stub it writes (`:227`) still carries no `Repo`. Benign today only because
   `repoPredicate`'s `OR repo = ''` fallback catches it on the next read. This is
   the same defect class as the `cli/task.go` task stub the ledger says it fixed —
   two instances existed; one was fixed.

6. **`gofmt -l` claim is false.** The ledger states "gofmt -l on every touched
   file — clean." Six changed files are unformatted under this toolchain:
   `internal/acceptance/record.go`, `internal/extract/decisions.go`,
   `internal/mission/mission_test.go`, `internal/spec/graph_ingest.go`,
   `internal/spec/graph_ingest_test.go`, `internal/tasks/record_test.go`. All six
   were already unformatted at HEAD, so the delivery did not introduce it — but
   the claim as written is not true.

7. **v5 is the first non-additive schema change, and `checkSchemaMismatch` still
   treats "graph newer than binary" as tolerated** on the stated rationale that
   "migrations are additive, so an older binary can still read the extra columns"
   (`internal/graph/graph.go:358-362`). A v4 binary opening a v5 graph gets a
   warning and proceeds — then performs unscoped `(type, key)` upserts against a
   widened index, reintroducing the tombstoning against a database that now
   legitimately holds two partitions' rows. Warning-only is arguably the right
   call since it points at `hero doctor`, but the tolerance rationale no longer
   strictly holds and the ledger does not mention it.

8. **Ledger count inaccuracies.** "10 new in `node_identity_test.go`" — there are
   9 tests plus 2 helpers. "98 packages green" — the run reports 60 packages, 53
   with tests.

9. **Live-corpus claims all verified** (read-only, against a copy):
   `schema_version=5`; `idx_nodes_current ON nodes(type, key, repo)`; 6,579 live
   nodes across 4 partitions (`''`=16, `hero-engine/hero`=5,641,
   `hero-engine/hero-cloud`=31, `hero-engine/hero-code`=891); 0 duplicate live
   rows; 0 orphaned live edges; `team-oauth` now has 1 live row under
   `hero-engine/hero` with the cloud copies tombstoned — exactly as reported.
   Two caveats: the "0 duplicate live rows per `(type, key, repo)`" check is
   tautological (the unique index enforces it), and there are currently **0** live
   rows sharing a `(type, key)` across repos — so the coexistence path the fix
   exists to enable is not yet exercised by any real data.

10. **Commit coupling.** `graphRepoKey`, used by this delivery's
    `internal/cli/task.go` and `internal/cli/graph_edge.go`, is defined in
    `internal/cli/brief.go:767` — a *staged* change belonging to the parent spec's
    delivery. The two must be committed together or the build breaks.

---

# Round 2 — remediation re-audit

**Audited:** the five remediations only, against the same working tree.
Re-ran `go build ./...`, `go vet ./...`, `go test ./...` — all clean (115
packages per `go list ./...`). Each remediation was independently falsified by
reverting only that remediation. Working tree restored byte-for-byte (sha1
verified); `.hero/graph.db` never opened for write — all corpus queries ran
against a copy.

## Verdict on each remediation

| # | Remediation | Production behavior | Its test evidence |
|---|---|---|---|
| 1 | AC-6 test queries the store directly | correct | **genuinely falsifies** |
| 2 | `repoWriteScope` | correct | **genuinely falsifies** |
| 3 | `GetNodeAt` partition-over-recency ordering | **correct** (verified independently) | **does NOT falsify** |
| 4 | `checkSchemaMismatch` names the v5 risk | **dead in production** | passes on a path that cannot execute |
| 5 | Ledger accuracy | n/a | verified accurate |

## Final acceptance-criteria status (supersedes the round-1 list above)

All six now carry genuine, falsified evidence:

- **AC-1 ✓** — closed in both directions. Non-empty partition: `repoPredicate`
  + `TestSiblingRepoIngestDoesNotTombstoneLocal`. Empty partition:
  `repoWriteScope` + `TestUnpartitionedWriteDoesNotClobberStampedNode`. Both
  falsify.
- **AC-2 ✓** · **AC-3 ✓** · **AC-4 ✓** · **AC-5 ✓** — unchanged from round 1.
- **AC-6 ✓** — upgraded from `~`. The direct store assertion falsifies with the
  exact claimed message.

The remaining HOLD is **not** about the acceptance criteria. It is about two
remediation claims whose evidence does not hold up (findings 5 and 6) — both
were responses to round-1 audit notes, not to spec ACs, and both are small
fixes.

## Confirmed good

1. **AC-6 test now genuinely falsifies.** Reverting the v5 index and the upsert
   scoping makes `TestWhySurvivesSiblingRepoIngest` fail with exactly the claimed
   message: `local live rows = 0, want 1 — the sibling ingest tombstoned the local
   node`. The direct `SELECT count(*)` between the sibling upsert and
   `store.Close()` runs before any second `hero why` can reconcile, so the masking
   is genuinely removed, not relocated. **AC-6 is now honestly evidenced.**

2. **`repoWriteScope` closes the unpartitioned-side hole.** Swapping it back to
   `repoScope` fails both new tests —
   `TestUnpartitionedWriteDoesNotClobberStampedNode` (`live rows = map[:2], want 1
   still live`) and `TestUnpartitionedWriteWithBothRowsLiveDoesNotError`. The
   `repo == ""` branch returns `AND repo = '' LIMIT 1`, which cannot match a
   stamped row, so the tombstoning I reproduced in round 1 is structurally
   impossible. `LIMIT 1` without `ORDER BY` is safe here: the v5 unique index
   guarantees at most one live `(type, key, '')` row. **AC-1 is now fully
   delivered in both directions.**

3. **The v1→v2 backfill is not broken in the other direction.** `repoWriteScope`
   delegates to `repoScope` unchanged when `repo != ""`, so a stamped writer still
   matches and upgrades a legacy `repo = ''` row in place.
   `TestLegacyUnpartitionedRowIsUpgradedInPlace` still passes, and I confirmed the
   upgrade independently: an unpartitioned write followed by a stamped write leaves
   exactly one live row, under the stamped partition. **AC-3 intact.**

4. **Ledger accuracy verified.** `go list ./...` = 115 packages;
   `node_identity_test.go` contains 13 `func Test` declarations; the gofmt claim
   now correctly states that no newly-unformatted file was introduced and that all
   six flagged files were already unformatted at HEAD (I re-confirmed each against
   `git show HEAD:<file>` in round 1).

## New findings

5. **`TestGetNodeAtPrefersExactPartitionOverRecency` does not falsify — same
   defect class as the round-1 AC-6 test, second occurrence.** I reverted the
   `GetNodeAt` ordering to plain `ORDER BY valid_from DESC` and the test **passed**.
   Cause: the test builds its two rows with back-to-back `upsert(...)` calls, and
   `nowRFC3339()` is second-precision — both rows land on the identical
   `valid_from` (observed: `2026-07-25T19:09:20Z` on both). There is no recency gap
   for partition preference to win against, so the test cannot distinguish the two
   orderings. Its own comment — "Written second, so it is the newer row by
   valid_from" — is factually wrong.

   **The fix itself is correct.** With an explicit recency gap (stamped row at
   `2020-01-01`, legacy `repo=''` row at `2030-01-01`, queried at `2031-01-01`),
   the remediated `GetNodeAt` returns the stamped row. The same construction
   returned `repo=""` against the round-1 code, so remediation 3 genuinely works —
   it is simply unguarded. Fix: give the test explicit `ValidFrom` values instead
   of relying on wall-clock ordering.

6. **The `checkSchemaMismatch` v5 risk message cannot execute in a shipped
   binary.** The `risk` string is gated on `schemaLess(binarySchema, "5")`, and the
   only production caller is `internal/graph/graph.go:333`, which passes the
   compile-time constant `schemaVersion` — currently `"5"`. So `schemaLess("5","5")`
   is always false and `risk` is always empty in production. More fundamentally: a
   binary that predates schema 5 is running *pre-v5 code* and contains the old
   `checkSchemaMismatch`, so it can never print this message. The warning is
   written for a user whose binary structurally cannot emit it.
   `TestSchemaMismatchWarningNamesTheV5Risk` passes only because it calls the
   unexported function directly with `("4", "5")` — an argument pair production
   cannot produce.

   **On warning vs. hard error: warning is the right call**, and for the stated
   reasons — an older binary can read a newer graph, `graph.db` is regenerable, and
   erroring would strand every user on an older binary. But the mitigation cannot
   live here. If the v5-write hazard is worth guarding, the guard belongs on the
   *write* path in a v5+ binary, or is simply accepted and documented — this branch
   should be removed rather than left as apparently-live safety code.

7. **`repoWriteScope` creates permanently coexisting `''` and stamped live rows.**
   The coordinator asked directly, and the answer is yes. Sequence, reproduced:
   unpartitioned write → stamped write (backfill upgrades in place, 1 live row) →
   unpartitioned write again (now inserts rather than clobbering) → **both live**;
   re-running the stamped writer does *not* retire the `''` row. Neither writer
   ever retires the other, so the pair is stable.
   `ListNodesByType("Feature")` then returns both rows for one slug.

   This is a deliberate and net-positive trade — round-1 behavior was an
   oscillating clobber that tombstoned the stamped node (the actual bug); this is
   duplication visible only to unscoped consumers. Sizing: `ListNodesByType` has 7
   unscoped production callers (`internal/tasks/query.go` ×3 for `Task`,
   `internal/acceptance/query.go` ×4 for `Criterion`), and neither `Task` nor
   `Criterion` is among the 10 unpartitioned writers, so there is no impact today.
   The live corpus confirms it: **0** `(type, key)` pairs currently hold both a
   `''` and a stamped live row; the `''` partition holds only `Issue` (12) and
   `Person` (4). `Person` is the one live collision surface, since
   `internal/tasks/record.go:184` writes it stamped while
   `internal/tracker/graph_ingest.go` and `internal/gitutil/graph_ingest.go` write
   it unpartitioned. Latent, not active — worth a follow-up spec to stamp the
   remaining 10 writers, not a blocker.

8. **Minor — `TestUnpartitionedWriteWithBothRowsLiveDoesNotError` does not
   reproduce the failure its comment claims.** Under the `repoScope` revert it
   fails on the row-count assertion (`live rows = map[:3], want one per partition`),
   not on `UNIQUE constraint failed`. The precondition for the UNIQUE violation —
   a live `''` row *and* a live stamped row — cannot be established through
   `UpsertNode` alone once the read rule clobbers on the first unpartitioned write;
   my round-1 reproduction needed a direct `INSERT` or a three-partition sequence.
   The test is a valid guard for the correct end state; only its doc comment
   overstates what it reproduces.

## Round 2 open items

- `TestGetNodeAtPrefersExactPartitionOverRecency` — inert guard. **Fix before
  ship:** set explicit `ValidFrom` values so a real recency gap exists.
- `checkSchemaMismatch` v5 `risk` branch — unreachable in production. **Decide:**
  remove it, or move the guard to the write path.
- Unpartitioned/stamped duplicate live rows — **follow-up spec:** stamp `Repo` on
  the 10 remaining production `UpsertNode` sites.

---

# Round 3 — remediation re-audit

**Audited:** the four round-3 items only. `go build ./...`, `go vet ./...`,
`go test ./...` all clean. Each fix falsified by reverting only that fix.
Working tree restored byte-for-byte (sha1 verified); `.hero/graph.db` never
opened for write.

| # | Claim | Verdict |
|---|---|---|
| 1 | `TestGetNodeAtPrefersExactPartitionOverRecency` rewritten with explicit `ValidFrom` | **verified — now falsifies** |
| 2 | Inert schema-warning branch removed | **verified — fully removed** |
| 3 | Overstated comment on `TestUnpartitionedWriteWithBothRowsLiveDoesNotError` corrected | **NOT DONE — claim is false** |
| 4 | Follow-up spec for the duplicate-node consequence | **verified — faithful, not a fig leaf** |

## 1. GetNodeAt test — genuinely falsifies now

Reverting the `GetNodeAt` ordering to plain `ORDER BY valid_from DESC` makes it
fail with exactly the claimed message:
`GetNodeAt("hero-engine/hero").Repo = "" — a newer unpartitioned row answered a
scoped query`.

**The explicit `ValidFrom` does not change the code path under test.** It is the
same `UpsertNode` (which honours a caller-supplied `ValidFrom` and only defaults
it when empty) and the same `GetNodeAt`. `IngestedAt` still defaults to now, and
both rows are live (`valid_to IS NULL`), so both satisfy the bitemporal `WHERE`
at the 2026 query point and the `ORDER BY` is the only thing that decides
between them — which is precisely the behavior under test. Backdated
`valid_from` is also realistic for this store, since bitemporal ingest can
backdate. Sound test.

## 2. Inert branch — fully removed

The `risk` variable, its `%s` verb, its argument, and
`TestSchemaMismatchWarningNamesTheV5Risk` are all gone. Grep confirms no
remaining reference; the only surviving matches for "repo-scoped node identity"
are the two migration comments, which are unrelated. `go vet` passes, which
independently confirms the `Sprintf` verb/argument count still matches (3/3).
`node_identity_test.go` drops 13 → 12 tests, consistent with removing exactly
one. The replacement comment accurately records why the guard cannot live in
`checkSchemaMismatch` and where it would have to live instead. Pre-existing
`mismatch_test.go` cases still pass.

## 3. Comment correction — claimed but not made

The ledger states the comment now "pins the reachable half (the write stays in
its own partition and both rows survive)." **That wording appears nowhere in the
repository.** The original overstated text is intact, verbatim, in
`internal/graph/node_identity_test.go`:

- `:358` — "covers the hard failure the same audit reproduced"
- `:361` — "colliding with the live repo = '' row on the v5 unique index"
- `:372` — "this is the one that used to fail with UNIQUE constraint failed:
  nodes.type, key, repo"

Impact is cosmetic — the test is a valid guard for the correct end state, and no
behavior depends on the comment. But it is a false completion claim, and the
third consecutive round in which a self-reported remediation did not match the
tree. **Either make the edit or strike the bullet from the ledger; do not leave
the ledger asserting it.**

## 4. Follow-up spec — faithful

`.hero/planning/bugs/graph-unpartitioned-writers-duplicate-nodes/spec.md` is a
real spec, not a placeholder. I verified every factual claim in it independently:

- **All ten `file:line` entries and their node types are exactly right** —
  `tracker/graph_ingest.go:54` Sprint, `:104` Issue, `:131` Person, `:166` Issue,
  `:267` Issue, `:281` Person; `gitutil/graph_ingest.go:109` Person, `:188`
  Issue; `extract/decisions.go:166` Concept; `acceptance/record.go:227` Commit.
  This matches my own independent scan exactly.
- `tasks/record.go:175` is indeed `upsertPerson`, the stamped Person writer.
- It reproduces my audit findings accurately: 0 collisions in the live corpus,
  16 nodes in the `''` bucket, 7 unscoped `ListNodesByType` callers reading only
  `Task`/`Criterion`, and `Person` as the one genuine collision surface.
- **AC-1 carries the source-level guard** so a new unpartitioned writer cannot
  be added silently — the thing that makes this a fix rather than a one-time
  cleanup.
- Boundaries correctly exclude changing `repoWriteScope`, naming the duplication
  as the intended safe side of the trade rather than a defect in it.
- It also raises a question I had not: `Person`/`Repo` are `globalNodeTypes`, and
  writers currently *disagree* about whether they are repo-global or per-repo
  (AC-2). That is the actual root of the collision surface.

Minor wording note: AC-3 says the system shall not *return* both rows; the
stronger and more testable invariant is that both shall not *exist*.

## Round 3 open items

- Ledger bullet 3 asserts a comment correction that was not made. Correct the
  comment at `internal/graph/node_identity_test.go:358-372`, or strike the
  bullet.
