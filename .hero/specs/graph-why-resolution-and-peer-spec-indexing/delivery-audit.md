# Delivery audit — graph-why-resolution-and-peer-spec-indexing

**Audited:** `git diff HEAD -- internal/` on `fix/graph-why-residual-peer-ingest`, plus pre-landed commit `cf6ebc2` (PR #2) and the child spec `graph-node-identity-repo-scoped`
**Verdict:** SHIP

_(Reached at round 3. Verdict kept as a bare token because `hero spec verify`
Gate 2 parses this line literally.)_

**Surface:** noteworthy

> **Round history.** Round 1 returned HOLD on two gaps (an AC authored narrower
> than its Change, and a missing latency regression check). Round 2 returned
> HOLD on a third: the descope's sign-off escalation was inoperative. All three
> are now closed and independently re-verified. The descope itself no longer
> exists — AC-3b was implemented, not deferred.

## Round-3 verification summary

Every acceptance criterion is now behaviorally satisfied and independently
evidenced. The one remaining defect is a ledger-accuracy gap on AC-2's row
(below), not a missing behavior. It does not block, but it should be corrected.

## (a) Is parent AC-3b legitimately DONE? YES

AC-3b as worded: *"WHEN a sibling repo ingests a slug the local repo also owns
THE SYSTEM SHALL leave the local partition's node live and resolvable."*
The citation is not doing work the code doesn't — verified four independent ways:

1. **Schema.** `internal/graph/graph.go:277-282` — migration `version: "5"` drops
   and recreates `idx_nodes_current` as `ON nodes(type, key, repo) WHERE valid_to
   IS NULL`. `schemaVersion = "5"` at `:159`. The v1 statement is deliberately
   left intact so each migration stays valid at its own point in the timeline —
   correct migration hygiene.
2. **Write path.** `UpsertNode`'s current-row lookup (`node.go:122-128`) is now
   scoped through `repoWriteScope(n.Repo)`, which restricts an unpartitioned
   write to `repo = ''` so it can never match a stamped row. All four accessors
   take a repo argument: `GetNodeAt:286`, `GetNode:309`, `GetNodeID:323`,
   `InvalidateNode:344`.
3. **Falsified by the auditor.** I removed the partition scope from the upsert
   lookup and re-ran: `TestSiblingRepoIngestDoesNotTombstoneLocal` fails with
   `live rows by repo = map[hero-engine/hero-cloud:2]` — the local partition
   vanishes, reproducing the exact team-oauth tombstone. `TestInvalidateNodeIsPartitionScoped`
   fails with `live rows = map[]`. Restored; `node.go` byte-intact.
4. **Live corpus.** I copied `.hero/graph.db` to scratch and queried the copy
   (never the original): `team-oauth` now shows `hero-engine/hero | live | 1`,
   where the original report recorded zero live local rows. The copy's
   `idx_nodes_current` is `(type, key, repo)`. **Original `graph.db` md5
   `24e340452a93fcdee7fb7f9c42737474` unchanged before and after.**

`TestWhySurvivesSiblingRepoIngest` (cited in the row) exists at
`internal/cli/why_resolution_test.go:338`.

## (b) Is the parent ledger honest end-to-end? ONE REAL GAP

**Rows 1, 3, 4, 5, 6, 7 re-verified and still hold.** `brief.go` (+3) and
`graph.go` (+9) are byte-identical to what I accepted in round 2;
`receive.go` and `receive_graph_test.go` are unchanged. `go build ./...` clean,
`go test ./...` green with zero failures.

**Row 2 claims more than the tree supports.** AC-2 reads *"THE SYSTEM SHALL
derive the graph partition key from `gitutil.RepoKey(projectRoot)` at every
graph writer, so no writer stamps nodes into a partition the reader never
queries."* The row's note says only: *"All five sites route through
`graphRepoKey`. Landed `cf6ebc2`."*

That is not what happened. A **sixth** mis-keyed writer existed the whole time:
`internal/attention/mail/promotion.go:281` called
`spec.WriteGraph(specs, s.cfg.PeerID, domain, store)` — stamping spec nodes
under a **UUID PeerID**, unambiguously "a partition the reader never queries."
It was corrected only in this session (now `repoKey := gitutil.RepoKey(s.root)`).
So AC-2 was **false** when it was marked DONE in rounds 1 and 2, and the row
still attributes its satisfaction entirely to `cf6ebc2`'s five sites.

The fix *is* recorded — in the **child** spec's ledger (Changes row 5,
"`promotion.go` — repoKey derivation corrected from `cfg.PeerID`"). The parent
spec mentions `promotion.go` **nowhere**. A reader of the parent alone would
believe AC-2 closed at `cf6ebc2`.

I share the blame: my round-1 acceptance of AC-2 rested on a grep for
`filepath.Base(projectRoot)`. **AC-7's guard structurally cannot catch this
class** — a wrong derivation that isn't `filepath.Base` slips straight past it.
That is a real limitation of using AC-7 as AC-2's enforcement mechanism, and
worth recording.

**Also unmentioned in the parent:** the newly filed
`graph-unpartitioned-writers-duplicate-nodes` documents **ten** production
`UpsertNode` sites that stamp no `Repo` at all. Under AC-2's literal "every
graph writer" that is an open gap; under its rationale clause it is fine, since
`repo = ''` *is* queried via the legacy fallback. Either way the parent ledger
should say so rather than leave AC-2 reading as unqualified.

**Recommended correction (non-blocking):** amend row 2's note to record the
sixth writer, its session, and the ten unpartitioned writers now tracked in the
follow-up spec.

## (c) Does "## Change 3's Write Half" describe what happened? YES, substantially

The header reads "Split to a Child Spec, Then Delivered"; the opening sentence
states it "was initially deferred, then delivered in the same session at the
user's direction"; the **Outcome** paragraph records the v5 migration and the
live `team-oauth` verification. The deferral, the reversal, and the outcome are
all present and correctly sequenced.

Two cosmetic blemishes: the middle paragraph retains its present-tense argument
*for* deferring ("rushing it into an unrelated delivery is how this subsystem
earned its reliability reputation"), which now reads oddly against a decision
that went the other way; and "~8 call sites" understates the actual threading.
The header and Outcome correct both. Not worth blocking.

## (d) Anything the child's changes invalidated? YES — and it was handled well

`internal/traversal/why_federation_test.go` previously pinned the **opposite**
contract: it asserted that when only a peer copy is live, a local query *fails*.
That assertion encoded the team-oauth bug as if it were the intended behavior —
which is exactly what my round-1 audit flagged about AC-3. The child rewrote it
to assert the local node survives a sibling ingest, and its doc comment says so
explicitly: *"This test previously characterized the OPPOSITE... That was the
team-oauth bug, pinned as if it were the contract."* Correcting a
characterization test **and recording why** is the right handling.

Parent AC-3's row still cites that test; AC-3 (read-path local preference) still
holds and the test still asserts it. No stale claim.

Parent spec line 333 describes `UpsertNode` matching by `(type, key)` alone in
the present tense, but it sits inside the historical "why it warranted its own
spec" block and is immediately followed by the corrected Outcome. Acceptable.

## Acceptance criteria

- [✓] **AC-1** read-side reconcile — `brief.go:796` `reconcileSpecGraph`, called from `runWhy` (`:467`) and `runBlocked` (`:599`). Unchanged since round 2.
- [~] **AC-2** every writer keys by `gitutil.RepoKey` — **behaviorally true now**, but the row's note is inaccurate about scope and provenance (see (b)).
- [✓] **AC-3** read-path local preference — honest about pre-dating `cf6ebc2`; test corrected by the child.
- [✓] **AC-3b** sibling ingest does not tombstone the local node — schema v5, scoped upsert and accessors, falsified by the auditor, confirmed on the live corpus.
- [✓] **AC-4** `ingestPromotedSpec` at `receive.go:87` — falsified in round 1, unchanged since.
- [~] **AC-5** idempotency ✓ falsified; best-effort guaranteed structurally (void return), test covers guard clauses only — disclosed.
- [✓] **AC-6** `graph.go:50` `RefreshIfStale` before `index.Open` — falsified in round 1, unchanged since.
- [✓] **AC-7** tree-wide `filepath.Base` guard — proven in round 1 with a planted offender. Note its blind spot in (b).

## Changes

Parent Changes rows 1–6 all verified. Row 6 (corpus-scale latency guard)
accepted in round 2 with a note that the 10s budget is loose against measured
56–213ms — a "doesn't hang" guard rather than a regression budget. Tightening to
~2s would restore signal; still not worth blocking.

## Evidence re-run by the auditor

- `go build ./...` clean; `go test ./...` green, zero failures.
- `go test ./internal/graph/... ./internal/peering/... ./internal/traversal/...` — all ok.
- AC-3b falsified by removing the upsert partition scope; both identity tests
  failed with the tombstone signature; `node.go` restored and re-verified.
- Live `graph.db` inspected **via a scratch copy only**; original md5 unchanged.
- Working tree left exactly as found; no probe files remain.
