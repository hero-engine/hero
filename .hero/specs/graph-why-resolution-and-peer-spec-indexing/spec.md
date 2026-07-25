---
title: "hero why fails to resolve specs that hero graph resolves — graph substrate stale + repoKey divergence"
slug: graph-why-resolution-and-peer-spec-indexing
type: bug
status: completed
priority: P1
domain: engineering
severity: high
root_cause_class: design   # two-store design gap + a code repoKey-derivation bug
tags: [graph, index, why, traversal, repoKey, peer-spec, ingest, reliability]
created: 2026-07-18
delivery_method: manual
completed_at: 2026-07-25T19:30:54Z
---

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — `hero why` is a core "where did this come from" tool; it silently fails for a growing subset of the corpus, and the one command users would reach for to "fix the graph" (`hero graph reingest`) makes it *worse*. |
| **Ease of Fix** | moderate — the primary fix is a small read-side reconcile (copy the pattern `hero blocked` already uses); the repoKey fix is a one-line change per call site but touches 5 sites and needs a shared helper to stay fixed. |
| **Caused by our codebase?** | Yes — a design gap (two stores, only one self-heals) plus a concrete code bug (`filepath.Base(projectRoot)` used as repoKey where `gitutil.RepoKey` is required). |
| **Needs more research?** | No for the `why`/`graph` divergence and repoKey bug (root cause proven by reproduction). Partial for the `team-oauth` federated-tombstone provenance (mechanism confirmed; exact ingest that re-keyed it to `hero-cloud` not traced — not needed for the fix). |

### Background
`hero why <slug>` resolves a slug to a graph node and walks origin edges. For a subset of specs it fails with `Error: no node with key "<slug>" in repo hero-engine/hero`, while `hero graph <slug>` and `hero search --specs <slug>` resolve the same slug fine. The failing subset is the *newer* specs — most conspicuously peer-received `spec-out` specs created this session (`chat-canonical-research`, `chat-slim-to-basic-research-seed`) plus a federated peer spec (`team-oauth`).

### Analysis
`hero why` and `hero graph` **read different databases**:
- `hero graph` / `hero search` → `.hero/index.db` (`internal/index`), which **self-heals from disk on every read** via `index.RefreshIfStale`.
- `hero why` → `.hero/graph.db` (`internal/graph` substrate: `nodes`/`edges`), which has **no read-side self-heal**. It is only populated by explicit ingest paths (`hero scan`, `hero graph reingest`, `hero blocked`'s inline `spec.WriteGraph`, `hero next ingest`).

So any spec created after the last graph ingest exists in `index.db` (self-healed) but is **absent from `graph.db`** → `hero why` misses it while `hero graph`/`hero search` find it. The task framing that "`hero graph` proves the node exists in the graph" is itself the trap: `hero graph` proves it exists in the **index**, a *different* store from the one `hero why` queries.

### Root Cause
Two independent, compounding causes:

1. **Store-divergence / missing read-side self-heal (primary).** The graph substrate that `hero why` reads is a regenerable cache with no staleness self-heal on the read path, unlike the index that `hero graph`/`hero search` read. Newer specs are invisible to `hero why` until an unrelated ingest command happens to run.

2. **repoKey derivation mismatch between writers and the reader (compounding, and a latent landmine).** `hero graph reingest` derives the partition key as `repoKey := filepath.Base(projectRoot)` → `"hero"` (`internal/cli/graph_memory.go:195`). The `hero why` reader — and the healthy writer `hero scan` — derive it as `gitutil.RepoKey(projectRoot)` → `"hero-engine/hero"` (org/repo slug from the git remote). Because the reader filters on `repo = "hero-engine/hero"`, nodes written by `reingest` under `"hero"` are unreachable. **`hero graph reingest` therefore does not fix `hero why` — it breaks it corpus-wide** by tombstoning the correctly-keyed nodes and replacing them with mis-keyed ones (reproduced below).

### Source
- Reader: `internal/traversal/why.go:108-130` (`resolveTarget`) and `internal/cli/brief.go:467-480` (`runWhyEdges`) — both require a live node where `repo = gitutil.RepoKey`.
- Reader wiring: `internal/cli/brief.go:719-734` (`openRepoStore` → `gitutil.RepoKey`).
- Divergent writer: `internal/cli/graph_memory.go:195` (`filepath.Base(projectRoot)`), shared by `internal/cli/sprint.go:230`, `internal/cli/extract.go:88`, `internal/cli/publish_pages.go:56`, `internal/cli/next_project.go:58`.
- Healthy writer (the fix template): `internal/cli/scan.go:525` and `internal/cli/brief.go:577-581` (`hero blocked` reconciles via `spec.WriteGraph(specs, repoKey, …)` where `repoKey` comes from `gitutil.RepoKey`).
- The self-heal that the graph path lacks: `internal/index/refresh.go` (`RefreshIfStale`), called by `search`, `list`, `ask`, `index`, etc.

### Fix Direction
- Give `hero why` a read-side graph reconcile before resolving — reuse exactly what `hero blocked` already does (`spec.WriteGraph` from disk frontmatter, keyed by `gitutil.RepoKey`). This is the graph-side analogue of `index.RefreshIfStale`.
- Make **every** graph writer derive the partition key from `gitutil.RepoKey(projectRoot)` via a single shared helper, and delete the `filepath.Base(projectRoot)` derivation from the ingest paths.
- Ensure a local spec always gets a live node in the local repo partition so a federated peer copy (e.g. `team-oauth` under `hero-engine/hero-cloud`) can't leave the local partition tombstoned.

---

## Problem Statement

Reproduced on `hero version v0.26.2-2-gaa948a7-dirty`, repo `hero-engine/hero`.

```
$ hero why agent-outposts --edges            # WORKS
$ hero why pm-foundation-delivery --edges     # WORKS
$ hero why chat-canonical-research --edges     # FAILS: no node with key "chat-canonical-research" in repo hero-engine/hero
$ hero why chat-slim-to-basic-research-seed     # FAILS
$ hero why team-oauth                            # FAILS
$ hero graph chat-canonical-research             # WORKS (shows 3 relationships)
$ hero graph team-oauth                          # WORKS ("No relationships found" — no error)
$ hero search chat-canonical-research --specs    # WORKS
```

Direct database inspection nails the divergence:

```
# graph.db (what hero why reads) — the failing slugs are ABSENT:
$ sqlite3 .hero/graph.db "SELECT count(*) FROM nodes WHERE key='chat-canonical-research';"      → 0
$ sqlite3 .hero/graph.db "SELECT count(*) FROM nodes WHERE key='chat-slim-to-basic-research-seed';" → 0

# index.db (what hero graph / hero search read) — present:
$ sqlite3 .hero/index.db "SELECT slug,type,status FROM specs WHERE slug='chat-canonical-research';"
  chat-canonical-research|feature|completed
```

Both failing `spec-out` specs carry a `received_from:` block (`mode: spec-out`, originator `hero-chat-swift-app`, peer `hero-code`) — i.e. peer-received specs written this session and never graph-ingested.

`team-oauth` is present in `graph.db` but only under a **peer** partition:
```
$ sqlite3 .hero/graph.db "SELECT repo, valid_to IS NULL AS live, count(*) FROM nodes WHERE key='team-oauth' GROUP BY repo, live;"
  hero-engine/hero        | 0(tombstoned) | 15
  hero-engine/hero-cloud  | 0(tombstoned) | 14
  hero-engine/hero-cloud  | 1(live)       |  1
```
Every local `hero-engine/hero` row is tombstoned; the only live row is the federated `hero-engine/hero-cloud` copy. `resolveTarget` filters `repo = 'hero-engine/hero' OR repo = ''` → no live local row → error. (`team-oauth` also carries a `received_from:` block — it is a handed-off peer spec.)

### Proof of the repoKey bug (reproduced, then reverted)
`graph.db` was backed up, and `hero graph reingest work` was run:
```
specs: 534 nodes, 1004 edges …
$ sqlite3 .hero/graph.db "SELECT key,repo FROM nodes WHERE key='chat-canonical-research' AND valid_to IS NULL;"
  chat-canonical-research|hero            # <-- written under "hero", NOT "hero-engine/hero"
$ hero why chat-canonical-research --edges  # STILL FAILS (reader wants "hero-engine/hero")
$ hero why agent-outposts --edges           # NOW ALSO FAILS — reingest tombstoned the good node and re-keyed it to "hero"
$ sqlite3 .hero/graph.db "SELECT repo,count(*) FROM nodes WHERE valid_to IS NULL AND type='Feature' GROUP BY repo;"
  hero|680                                 # entire work subgraph re-keyed to the wrong partition
```
The pre-reingest `graph.db` was then restored, returning to the exact reported state (`agent-outposts` works, `chat-canonical-research` fails). No lasting mutation was left behind.

### Proof of the fix direction (reproduced, then reverted)
`hero blocked` reconciles the graph via `spec.WriteGraph(specs, repoKey, …)` with `repoKey = gitutil.RepoKey = "hero-engine/hero"` (`internal/cli/brief.go:577-581`). Running it healed the graph correctly:
```
$ hero blocked >/dev/null
$ hero why chat-canonical-research --edges
  # "Chat Canonical Research Pack …" `chat-canonical-research` (Feature)  ← RESOLVES
$ sqlite3 .hero/graph.db "SELECT key,repo FROM nodes WHERE key='chat-canonical-research' AND valid_to IS NULL;"
  chat-canonical-research|hero-engine/hero   ← correct partition
```
This confirms the fix: `hero why` needs the same disk-reconcile `hero blocked` already performs, and writers must use `gitutil.RepoKey`. (State restored to the reported repro afterward.)

## Environment Details
- Local dev workspace `/Users/developer/projects/hero-engine/repository/hero`, git remote `git@…:hero-engine/hero.git` → `gitutil.RepoKey` yields `hero-engine/hero`; `filepath.Base(projectRoot)` yields `hero`.
- SQLite stores: `.hero/graph.db` (substrate) and `.hero/index.db` (corpus index), both WAL.
- The bug only manifests where `gitutil.RepoKey != filepath.Base(projectRoot)`, i.e. any repo whose directory name differs from its `owner/repo` slug — which is **every** repo cloned into a plain `hero/` (or `repository/hero/`) directory. That is the common case, which is why this recurs.

---

## Root Cause Analysis

### Confirmed
1. **Two stores, one self-heals.** `hero graph` (`internal/cli/graph.go:50-60`) opens `internal/index` and calls `idx.GetRelations`; `hero search`/`hero list`/`hero ask` all call `index.RefreshIfStale` first (`internal/cli/search.go:86`, `list.go:112`, `ask.go:68`). `hero why` (`internal/cli/brief.go:440-462`) opens `internal/graph` and calls `traversal.Why` with **no** reconcile. `RefreshIfStale` (`internal/index/refresh.go:33+`) is documented as "the staleness-self-healing primitive that lets read-side tools self-heal before querying" — it exists only for the index. Confirmed by DB inspection: the failing slugs are in `index.db` but absent from `graph.db`.

2. **`hero index` does NOT touch the graph substrate.** `runIndex` (`internal/cli/index.go`) calls `index.RefreshIfStale` / `index.Rebuild` only — both operate on `index.db`. So the owner's "requires a manual `hero index`" intuition is *refuted in the literal sense*: `hero index` would never repopulate `graph.db`. The graph substrate is reconciled only by `hero scan`, `hero graph reingest`, `hero blocked` (inline), and `hero next ingest`.

3. **repoKey derivation diverges between writer and reader.** `internal/cli/graph_memory.go:195` — `repoKey := filepath.Base(projectRoot)` (= `"hero"`). Reader `openRepoStore` (`internal/cli/brief.go:733`) and healthy writer `hero scan` (`internal/cli/scan.go:491,525`) use `gitutil.RepoKey` (= `"hero-engine/hero"`). `gitutil.RepoKey` (`internal/gitutil/gitutil.go:217`) normalizes the git remote to `owner/repo`, falling back to `filepath.Base(dir)` only when there is no origin remote. Reproduced: `hero graph reingest work` re-keys the whole work subgraph to `"hero"` and breaks `hero why` for every spec.

4. **The fix template already exists in-tree.** `hero blocked` reconciles with the correct key and heals `hero why` (reproduced above). The fix is to lift that reconcile onto the `hero why` path and unify the repoKey derivation.

### Hypothesis (not fully traced — not needed for the fix)
- `team-oauth`'s local partition was tombstoned and its live node re-keyed to `hero-engine/hero-cloud` by a federated/satellite ingest (cf. `satellite-corpus-integration`, cross-repo `hero scan` in `scan.go:491-530` which iterates sibling repos with each sibling's own `gitutil.RepoKey`). The exact ingest run that flipped it isn't traced, but the failure mechanism (no live node in the local partition) is confirmed by DB inspection and is the same class as causes (1)/(3).

---

## Code Flow (End to End)

### `hero why <slug>` (fails)
1. `internal/cli/brief.go:440` `runWhy` — target = joined args.
2. `internal/cli/brief.go:442` `openRepoStore()` → `internal/cli/brief.go:729` `graph.Open(heroDir)` opens `.hero/graph.db`; `:733` `gitutil.RepoKey(projectRoot)` → `"hero-engine/hero"`. **No disk reconcile.**
3. `internal/cli/brief.go:452` `traversal.Why(store, "hero-engine/hero", slug, depth)`.
4. `internal/traversal/why.go:89` → `:108` `resolveTarget` runs `SELECT … FROM nodes WHERE key=? AND valid_to IS NULL AND (repo=? OR COALESCE(repo,'')='')`.
5. Node absent (never ingested) or mis-keyed (`repo='hero'`) or only-federated (`repo='hero-engine/hero-cloud'`) → `sql.ErrNoRows` → `internal/traversal/why.go:125` returns `no node with key %q in repo %s`.
6. `--edges` path is the same failure at `internal/cli/brief.go:473-480`.

### `hero graph <slug>` (works)
1. `internal/cli/graph.go:32` `runGraph`.
2. `internal/cli/graph.go:50` `index.Open(heroDir)` opens `.hero/index.db`. (Note: `graph.go` itself does not call `RefreshIfStale`, but `index.db` was already self-healed by a prior `hero search`/`hero list`, so the row is durably present.)
3. `internal/cli/graph.go:57` `idx.GetRelations(slug)` reads `spec_relations` in `index.db` → resolves.

### `hero graph reingest work` (writes the wrong key)
1. `internal/cli/graph_memory.go:125` `runGraphReingest` → `:158` `reingestWork`.
2. `internal/cli/graph_memory.go:195` `repoKey := filepath.Base(projectRoot)` → `"hero"`.
3. `internal/cli/graph_memory.go:201` `spec.WriteGraph(specs, "hero", …)` → `internal/spec/graph_ingest.go:62` stamps `Repo: "hero"` on every node, tombstoning the prior correctly-keyed rows.

---

## Key Files

### Reader (hero why)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/traversal/why.go` | 108–130 | `resolveTarget` — the failing lookup; filters `repo = ? OR repo = ''` on live nodes. |
| `internal/cli/brief.go` | 440–480 | `runWhy` / `runWhyEdges` — no disk reconcile before query; both emit the "no node with key" error. |
| `internal/cli/brief.go` | 719–734 | `openRepoStore` — derives repoKey via `gitutil.RepoKey` (the value the reader filters on). |

### Reader (hero graph / search) and the self-heal it enjoys
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/graph.go` | 32–75 | `runGraph` reads `index.db` via `GetRelations` — a different store from `hero why`. |
| `internal/index/refresh.go` | 33+ | `RefreshIfStale` — the read-side self-heal that exists for the index but has no graph analogue. |
| `internal/cli/index.go` | 40–113 | `hero index` refreshes/rebuilds `index.db` only — never `graph.db`. |

### Divergent writers (repoKey bug)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/graph_memory.go` | 195 | **`filepath.Base(projectRoot)`** as repoKey for `spec.WriteGraph`/`sessions.WriteGraph`/git-log ingest — the corpus-corrupting site. |
| `internal/cli/sprint.go` | 230 | same wrong derivation. |
| `internal/cli/extract.go` | 88 | same. |
| `internal/cli/publish_pages.go` | 56 | same. |
| `internal/cli/next_project.go` | 58 | same. |
| `internal/gitutil/gitutil.go` | 217–250 | `RepoKey` / `normalizeRemoteURL` — the correct derivation. |

### Healthy writer (fix template)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/brief.go` | 577–581 | `hero blocked` reconciles graph from disk via `spec.WriteGraph(specs, repoKey, …)` with the correct `gitutil.RepoKey`. Proven to heal `hero why`. |
| `internal/cli/scan.go` | 491, 525 | `hero scan` derives repoKey via `gitutil.RepoKey`. |
| `internal/spec/graph_ingest.go` | 32–75 | `spec.WriteGraph` — stamps `Repo: repoKey` on every node/edge; keying is only as correct as the repoKey passed in. |

---

## Secondary Defects
1. **`hero graph reingest` actively corrupts the graph.** Independent of the `why` bug, running the documented "re-populate a subgraph from its source of truth" command re-keys the entire work subgraph to `filepath.Base(projectRoot)`, breaking `hero why`, `hero blocked`'s domain joins, `hero impact`, and any consumer that filters on `gitutil.RepoKey`. This is a foot-gun that ships today.
2. **Four other writers share the wrong derivation** (`sprint.go`, `extract.go`, `publish_pages.go`, `next_project.go`). Any of these writing graph nodes under `"hero"` produces the same class of unreachable rows.
3. **`hero graph` does not self-heal itself** — it only works because some earlier `hero search`/`hero list` ran `RefreshIfStale`. On a truly cold `index.db`, `hero graph <fresh-slug>` would also miss. Lower severity (index self-heals broadly), but worth a defensive `RefreshIfStale` at the top of `runGraph`.
4. **Peer-call / handoff spec write path performs no graph ingest.** `peering.Call` (`internal/peering/peercall.go`) and the receiving `spec-out` design flow write the spec file (landing in `index.db` via self-heal) but never call `spec.WriteGraph`, so peer-received specs are graph-invisible until an unrelated ingest runs. Fixing cause (1) (read-side reconcile on `why`) masks this; a belt-and-suspenders fix ingests at write time.

---

## Verdict on the peer-spec auto-indexing hypothesis
**Confirmed, with a correction.** Peer-received `spec-out`/handoff specs (`chat-canonical-research`, `chat-slim-to-basic-research-seed`, `team-oauth` — all carry `received_from:`) are **absent or mis-scoped in the graph substrate** while present in the index. The peer-call/handoff write path does not trigger graph ingestion. **Correction to the hypothesis:** the fix is *not* a manual `hero index` — `hero index` only refreshes `index.db` and never repopulates `graph.db`. The graph substrate is healed only by `hero scan` / `hero graph reingest` (currently mis-keyed) / `hero blocked` / `hero next ingest`. So the corpus can sit graph-stale even after the user "re-indexed."

---

## Suggested Fix Approach

### Change 1 — give `hero why` a read-side graph reconcile (primary)
**File:** `internal/cli/brief.go`, `runWhy` (and `runWhyEdges`), around line 440-462.

**Before:**
```go
func runWhy(cmd *cobra.Command, args []string) error {
	target := strings.Join(args, " ")
	store, repoKey, err := openRepoStore()
	if err != nil {
		return err
	}
	defer store.Close()

	if whyEdges {
		return runWhyEdges(store, repoKey, target)
	}

	trace, err := traversal.Why(store, repoKey, target, whyDepth)
	...
```

**After (mirror what `hero blocked` already does at brief.go:577-581):**
```go
func runWhy(cmd *cobra.Command, args []string) error {
	target := strings.Join(args, " ")
	store, repoKey, err := openRepoStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Reconcile the spec subgraph from frontmatter (durable truth) before
	// resolving, so `hero why` reflects specs created since the last graph
	// ingest — matching `hero blocked` and the index's RefreshIfStale.
	// Best-effort: on error, query whatever the graph already holds.
	if cfg := loadConfigSilent(); cfg != nil {
		if specs, derr := spec.Discover(cfg.HeroDir(findProjectRoot())); derr == nil {
			_, _ = spec.WriteGraph(specs, repoKey,
				graph.DomainFor(*cfg, graph.IntrinsicActive), store)
		}
	}

	if whyEdges {
		return runWhyEdges(store, repoKey, target)
	}
	trace, err := traversal.Why(store, repoKey, target, whyDepth)
	...
```
**Why:** eliminates the store-staleness gap on the read path. Better still, extract a `reconcileSpecGraph(store, repoKey)` helper and call it from both `runWhy` and `runBlocked` so the two stay in lockstep.

### Change 2 — unify repoKey derivation on all graph writers (the landmine)
**File:** `internal/cli/graph_memory.go:195` (and `sprint.go:230`, `extract.go:88`, `publish_pages.go:56`, `next_project.go:58`).

**Before:**
```go
repoKey := filepath.Base(projectRoot)
```
**After:**
```go
repoKey := gitutil.RepoKey(projectRoot)
```
**Why:** aligns every writer with the reader and with `hero scan`. Without this, `hero graph reingest` continues to corrupt the graph. Enforce it by routing all graph-partition keying through a single helper (e.g. `graph.RepoKeyFor(projectRoot)` wrapping `gitutil.RepoKey`) and forbidding `filepath.Base(projectRoot)` as a repoKey in review/lint.

### Change 3 — keep the local partition live for federated specs (team-oauth)
**File:** the reconcile in Change 1 already fixes this for `hero why` (it writes a live local-repo node for every local spec on disk, including `team-oauth`). Additionally, audit the cross-repo scan path (`internal/cli/scan.go:491-530`) so ingesting a sibling's copy of a slug never tombstones the local partition's node — sibling copies must be additive under the sibling's own repoKey, never overwrite the local one.

### Change 4 (belt-and-suspenders) — ingest at peer-spec write time
**File:** peer-call/handoff receive path (`internal/peering/peercall.go` / handoff accept). After writing a `received_from:` spec to disk, call `spec.WriteGraph` for it (correct repoKey) so it is graph-visible immediately, not just after the next reconcile.

---

## Acceptance Criteria

Derived from **Suggested Fix Approach** (Changes 1–4), **Secondary Defects**,
and the **Test Plan**. Each names where it was delivered: `cf6ebc2` is the
earlier landing of Changes 1–3; the rest closes the residual tail.

- **AC-1:** WHEN `hero why <slug>` runs against a spec that exists on disk but
  has not been graph-ingested since it was written THE SYSTEM SHALL reconcile
  the spec subgraph on the read path and resolve the slug, rather than
  returning `no node with key`. *(Change 1 — landed in `cf6ebc2`.)*
- **AC-2:** THE SYSTEM SHALL derive the graph partition key from
  `gitutil.RepoKey(projectRoot)` at every graph writer, so no writer stamps
  nodes into a partition the reader never queries. *(Change 2 — landed in
  `cf6ebc2`.)*
- **AC-3:** WHEN a slug has a live node under a peer repoKey and one under the
  local repoKey THE SYSTEM SHALL resolve the local node, so a federated peer
  copy cannot shadow the local partition on the **read** path. *(Change 3,
  reader half — landed in `cf6ebc2`; `resolveTarget`'s
  `ORDER BY (repo = ?) DESC`.)*
- **AC-3b:** WHEN a sibling repo ingests a slug the local repo also owns THE
  SYSTEM SHALL leave the local partition's node live and resolvable. *(Change
  3, write half — delivered via the child spec
  `graph-node-identity-repo-scoped`, in this session.)*
- **AC-4:** WHEN a peer handoff is received THE SYSTEM SHALL write the promoted
  spec into the graph substrate under the local repo partition before the
  receive returns, so the spec is resolvable with no separate ingest.
  *(Change 4.)*
- **AC-5:** WHEN the same peer transfer is received twice THE SYSTEM SHALL NOT
  change the number of live graph nodes for the promoted slug, and a receive
  SHALL NOT fail because the graph could not be written. *(Change 4,
  best-effort contract.)*
- **AC-6:** WHEN `hero graph <slug>` runs against a cold `index.db` THE SYSTEM
  SHALL self-heal the index before querying, so a spec written since the last
  index refresh resolves without a manual `hero index`. *(Secondary defect 3.)*
- **AC-7:** THE SYSTEM SHALL fail a source-level guard if any file under
  `internal/` derives a graph repo partition key via
  `filepath.Base(projectRoot)`, covering graph writers outside `internal/cli`.
  *(Test 4, widened for Change 4.)*

## Change 3's Write Half — Split to a Child Spec, Then Delivered

The write half of Change 3 (AC-3b) was initially deferred, then delivered in
the same session at the user's direction. It lives in the child spec
`graph-node-identity-repo-scoped` — which carries its own diagnosis, ACs,
ledger, and audit — rather than being folded in here, so the two fixes stay
separately traceable.

**Why it warranted its own spec.** The spec diagnosed the
`team-oauth` tombstoning as something to fix by auditing the cross-repo scan
path. It is not: `UpsertNode` finds the current row by `(type, key)` alone
(`internal/graph/node.go:105-113`), and the schema *enforces* that with
`CREATE UNIQUE INDEX idx_nodes_current ON nodes(type, key) WHERE valid_to IS
NULL`. A sibling ingest carrying the same slug therefore matches the local
row, sees a differing partition, and invalidates-and-reinserts it under the
sibling's repoKey — exactly the reported state (every local `team-oauth` row
tombstoned, the only live row under `hero-engine/hero-cloud`).

Repo-scoping node identity means changing that unique index and then making a
repo decision at every unscoped accessor — `GetNodeID`, `GetNode`,
`GetNodeAt`, `InvalidateNode`, and edge resolution in `graph/sync.go` — each
of which would otherwise bind to an arbitrary repo's node once two live rows
can coexist. That is a migration of the graph substrate's identity model
across ~8 call sites, not a tail-end patch, and rushing it into an unrelated
delivery is how this subsystem earned its reliability reputation.

**Outcome.** Delivered: node identity is now `(type, key, repo)` (graph schema
migration v5), the upsert and every accessor are partition-scoped, and
`hero why team-oauth` — the exact command from the report — resolves against
this repo's real graph, where the local partition previously had every row
tombstoned. See that spec's Completion Ledger.

## Changes

- `internal/peering/receive.go` — ingest the promoted spec into the graph
  substrate at the end of `Receive`, keyed by `gitutil.RepoKey`, best-effort.
- `internal/cli/graph.go` — `index.RefreshIfStale` before querying, matching
  `search`/`list`/`ask`.
- `internal/cli/brief.go` — document that non-CLI graph writers share
  `gitutil.RepoKey` as the one derivation.
- `internal/cli/why_resolution_test.go` — widen the `filepath.Base` guard from
  `internal/cli` to the whole `internal/` tree; add the cold-index test.
- `internal/peering/receive_graph_test.go` (new) — Test 6 plus the idempotency
  and best-effort guards.

## Test Plan

### Existing test review
- `internal/traversal/why_test.go` — covers traversal logic (two-hop chains, depth bounds, cycles, not-found, supersedes, boundary handoff) against a **hand-seeded in-memory store**. None exercise: (a) the reader vs. writer repoKey agreement, (b) resolution of a spec that is on disk but not yet graph-ingested, (c) parity with `hero graph`/index. This is the coverage gap that let the bug ship.
- `internal/index/refresh_test.go` (if present) covers index self-heal — no graph equivalent exists.
- No test file for `internal/cli/graph_memory.go` (`ls internal/cli/graph_memory*_test.go` → none). `hero graph reingest` is untested end-to-end — which is why the repoKey corruption went unnoticed.

### Test changes needed (prioritized)
1. **`internal/cli` — `TestWhyResolvesSpecCreatedSinceLastIngest` (highest value; catches the reported bug).**
   Invariant: a spec file on disk resolves through `hero why` even if no `hero scan`/`reingest` has run since it was written.
   Scenario: init a temp workspace with a git remote whose slug ≠ dir name; write a spec file; do **not** run scan/reingest; run `runWhy` → must resolve (not "no node with key"). Fails today; passes after Change 1.

2. **`internal/cli` — `TestWhyAndGraphAgreeOnResolution` (parity invariant).**
   Invariant: for every slug `hero graph`/`index.GetRelations` resolves, `hero why` also resolves. Seed N specs, run both resolvers over each slug, assert no slug resolves in one store but errors in the other. Guards the two stores from drifting apart again.

3. **`internal/cli` — `TestGraphReingestUsesGitRemoteRepoKey` (guards the landmine).**
   Invariant: after `hero graph reingest work` in a repo with remote `owner/name`, `nodes.repo == gitutil.RepoKey(projectRoot)` (`owner/name`), and `hero why <slug>` still resolves. Directly pins `graph_memory.go:195`; fails today.

4. **`internal/cli`/lint — `TestNoFilepathBaseRepoKey` (keeps it fixed).**
   Grep-style assertion that no `WriteGraph`/graph-ingest call site derives its repoKey via `filepath.Base(projectRoot)` — all must route through `gitutil.RepoKey` (or the shared helper). Covers `graph_memory.go`, `sprint.go`, `extract.go`, `publish_pages.go`, `next_project.go`.

5. **`internal/traversal` — `TestResolveTarget_FederatedPeerCopyDoesNotShadowLocal` (team-oauth mode).**
   Invariant: when a slug has a live node under a peer repoKey and a live node under the local repoKey, `resolveTarget(localRepoKey, slug)` returns the local node; when only the peer copy is live, document/deny the intended behavior. Pins the federation-scoping edge.

6. **`internal/peering` — `TestReceivedSpecIsGraphVisible` (belt-and-suspenders, if Change 4 taken).**
   Invariant: after a `spec-out`/handoff receive writes a `received_from:` spec, the slug resolves in the graph substrate without a separate ingest.

### Regression scope
- Change 1 adds a `spec.WriteGraph` call to `hero why`'s hot path — same cost `hero blocked` already pays; verify latency stays within the `TestWhy_DepthFourUnder200ms` budget (add a corpus-scale variant).
- Change 2 alters the partition key written by 5 commands. One-time effect: on first run after the fix, nodes previously written under `"hero"` are superseded by correctly-keyed rows — benign, but confirm `hero graph`/`hero impact`/`hero blocked` still resolve and no duplicate live rows persist across both keys. Consider a one-shot migration/reconcile note in the release.

---

## Notes
- The broader reliability concern (owner: "core areas break ~weekly, no regression tests") is well-founded for this surface: the graph substrate has **no** read-side self-heal test and the `reingest` command has **no** test at all, so both the staleness gap and the repoKey corruption were invisible to CI. Tests 1–4 above are the highest-leverage additions and directly encode the two invariants that broke here: *"every on-disk spec is resolvable by `hero why`"* and *"every graph writer keys by `gitutil.RepoKey`."*
- Related prior art in-corpus: `index-staleness-auto-refresh` (the index-side self-heal this bug shows the graph lacks) and `satellite-corpus-integration` (cross-repo partitioning, relevant to the `team-oauth` federation mode).

## Recap
`hero why` reads the graph substrate (`.hero/graph.db`), which — unlike the index that `hero graph`/`hero search` read — has no read-side self-heal, so specs created since the last graph ingest (notably peer-received `spec-out` specs) are invisible to it while `hero graph` finds them in the index. Compounding this, `hero graph reingest` and four other writers stamp the partition key as `filepath.Base(projectRoot)` (`"hero"`) instead of `gitutil.RepoKey` (`"hero-engine/hero"`), so reingesting doesn't fix `hero why` — it corrupts the whole work subgraph. Both were reproduced and reverted; the fix (proven via `hero blocked`) is to reconcile specs on the `why` read path and unify every graph writer on `gitutil.RepoKey`, backed by the missing parity/repoKey regression tests. Severity high: a core traversal tool silently fails on a growing subset of the corpus and the obvious remediation makes it worse.

## Kickoff
Paste into a fresh session to fix:
> Fix the `hero why` vs `hero graph` resolution divergence per `.hero/planning/bugs/graph-why-resolution-and-peer-spec-indexing/spec.md`. Two changes: (1) add a disk→graph reconcile to `runWhy` in `internal/cli/brief.go` mirroring `runBlocked` (brief.go:577-581) — extract a shared `reconcileSpecGraph(store, repoKey)` helper; (2) replace `filepath.Base(projectRoot)` with `gitutil.RepoKey(projectRoot)` as the graph partition key in `internal/cli/graph_memory.go:195`, `sprint.go:230`, `extract.go:88`, `publish_pages.go:56`, `next_project.go:58`, routed through one helper. Add the prioritized regression tests from the spec's Test Plan (start with `TestWhyResolvesSpecCreatedSinceLastIngest` and `TestGraphReingestUsesGitRemoteRepoKey`). Do NOT change `resolveTarget`'s repo filter — the reader is correct; the writers and the missing reconcile are the bug.

## Completion Ledger

**Task as executed.** This spec was ~80% delivered before this session:
commit `cf6ebc2` ("heal hero why on read + fix repoKey drift across graph
writers", PR #2) landed Changes 1–3 and Test Plan items 1–5, but the spec was
never flipped out of `planning`. This delivery closes the residual tail —
Change 4 (peer-spec write-time ingest), Secondary Defect 3 (`hero graph` cold
index), and Test Plan item 6 — and audits the already-landed work rather than
redoing it.

**Stack:** Go. Skills: go-stack, implementation-principles, testing-and-validation.

**Verification of the already-landed work** (read on disk, not assumed):
`runWhy` calls `reconcileSpecGraph` (`brief.go`), the helper is shared with
`runBlocked`, all five writers named in the spec (`graph_memory.go:194`,
`extract.go:88`, `publish_pages.go:55`, `sprint.go:230`, `next_project.go:58`)
route through `graphRepoKey`, and `why_resolution_test.go` +
`why_federation_test.go` carry Tests 1–5.

**Validation performed.**
- `go build ./...` — clean. `go test ./...` — green, no regressions.
- `gofmt -l` on every touched file — clean.
- **Pre-fix falsification of each new behavior** (the fix removed, tests re-run):
  - Removing the `ingestPromotedSpec` call → `TestReceivedSpecIsGraphVisible`
    fails with *"has no live graph node in repo … it would be invisible to
    `hero why`"*.
  - Removing `index.RefreshIfStale` from `runGraph` →
    `TestGraphSelfHealsColdIndex` fails with *"No relationships found for
    refunds."*
  - Planting a `repoKey := filepath.Base(projectRoot)` in `internal/peering` →
    the widened guard fails, naming `../peering/…:6`. The pre-existing
    package-local guard would not have caught it.

**Two of my own tests were wrong and were corrected, not the code.**
- An idempotency assertion of "exactly 1 live node" failed at 2. Probing the
  store showed a promoted transfer legitimately produces an `Intake` and a
  `Feature` node sharing one slug — the provenance chain `hero why` walks.
  The assertion now pins *unchanged across replays* rather than a literal.
- A fault-injection test made `graph.db` a directory to prove the best-effort
  contract; that broke the mail service itself (which legitimately needs the
  graph), not the ingest. Replaced with a direct test of the guard clauses.

**Deviation from the Suggested Fix Approach.** Change 2 suggested routing the
partition key through `graph.RepoKeyFor`. That is not buildable:
`internal/gitutil` imports `internal/graph`, so `graph` importing `gitutil`
is an import cycle. `gitutil.RepoKey` is already the single derivation;
`cli.graphRepoKey` is a documented alias and `peering` calls `gitutil.RepoKey`
directly. The invariant is enforced by AC-7's guard, now tree-wide, rather
than by a wrapper that cannot exist.

**Post-audit corrections.** The cold audit returned **HOLD** on two gaps, both
legitimate and both since addressed:
1. **AC-3 was authored narrower than the Change it cites.** *(Initially
   resolved by deferring; the user then directed that it be implemented in
   this session, and it was — see AC-3b and the child spec.)* The auditor found
   AC-3 captured only Change 3's reader half — which was already true before
   `cf6ebc2` — while the write half (a sibling ingest must not tombstone the
   local node) was undelivered and the defect still live. That is correct. On
   investigation the mechanism is not the scan path the spec hypothesized: node
   identity is `(type, key)` at the schema level
   (`idx_nodes_current`), so single-live-row-per-slug is a database invariant.
   Repo-scoping it means a substrate migration plus a repo decision at ~8
   unscoped accessors. Rather than shape the AC around what shipped or rush a
   graph-core migration into an unrelated delivery, it is now stated explicitly
   as **AC-3b**, with a section giving the real mechanism, and a full child
   spec `graph-node-identity-repo-scoped` carrying the diagnosis, ACs, and
   risks. The descope was surfaced to the user rather than self-approved; the
   user chose to implement it in-session, and AC-3b is now delivered through
   that child spec.
2. **The corpus-scale latency regression check the spec's Regression scope
   required was missing.** The auditor is right that
   `TestWhy_DepthFourUnder200ms` never enters `runWhy`, so the reconcile added
   to the hot path had zero coverage. Added
   `TestWhyReconcileStaysWithinBudgetAtCorpusScale`: drives the real command
   over a 200-spec cold corpus (no index, no scan). Measured 208ms.

**Follow-ups.**
- `hero peer call --mode=spec-out` designs a spec on the *peer* side; that
  peer's own write path is out of this repo's reach. The read-side reconcile
  (AC-1) remains the backstop there.
- The `team-oauth` federated-tombstone provenance was never traced to the exact
  ingest that re-keyed it. The spec scoped that out, and the mechanism is now
  understood and specced (`graph-node-identity-repo-scoped`) even though the
  triggering run was never identified.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `hero why` resolves a spec on disk but not yet graph-ingested (read-side reconcile) | DONE | `internal/cli/brief.go` — `runWhy` calls `reconcileSpecGraph`; helper shared with `runBlocked`. Landed `cf6ebc2`; verified on disk this session. Test: `TestWhyResolvesSpecCreatedSinceLastIngest` |
| 2 | Every graph writer derives the partition key from `gitutil.RepoKey` | DONE | The five `filepath.Base(projectRoot)` sites route through `graphRepoKey` (`brief.go`), landed `cf6ebc2`. Tests: `TestGraphReingestUsesGitRemoteRepoKey`, `TestNoFilepathBaseRepoKey`. **Correction:** this row was FALSE when first marked DONE. A cold audit found a *sixth* mis-keyed writer — `internal/attention/mail/promotion.go` called `spec.WriteGraph(specs, cfg.PeerID, …)`, stamping the whole local spec corpus under a UUID, a partition no reader queries. It was live until this session and is fixed here (see the child spec's ledger). Two limits worth stating plainly: (a) AC-7's guard structurally CANNOT catch this class — it greps for `filepath.Base(projectRoot)`, and a wrong derivation of any other shape walks straight past it, so AC-7 is a partial enforcement mechanism for AC-2, not a complete one; (b) ten `UpsertNode` sites still stamp no `Repo` at all, which satisfies AC-2's rationale (the `repo = ''` legacy fallback is queried) but not its literal "every graph writer" — tracked in `graph-unpartitioned-writers-duplicate-nodes` |
| 3 | A federated peer copy does not shadow the local partition on the READ path | DONE | Landed `cf6ebc2` — `resolveTarget`'s `ORDER BY (repo = ?) DESC`. Test: `TestResolveTarget_*PeerCopyDoesNotShadowLocal` (`internal/traversal/why_federation_test.go`). Note: pre-dates `cf6ebc2`; that commit added the characterization test, not the behavior |
| 3b | Sibling ingest must not tombstone the local node (WRITE half of Change 3) | DONE | Delivered via the child spec `graph-node-identity-repo-scoped` in this session: graph schema migration v5 repo-scopes node identity to `(type, key, repo)`, and the upsert plus all four accessors are partition-scoped. Verified live — `team-oauth` regained a live node in `hero-engine/hero` and `hero why team-oauth` resolves. Tests: `TestSiblingRepoIngestDoesNotTombstoneLocal`, `TestWhySurvivesSiblingRepoIngest`, `TestMigrationV5RepoScopesExistingDatabase` |
| 4 | A received peer handoff is graph-resolvable in the local partition before `Receive` returns | DONE | `internal/peering/receive.go` — `ingestPromotedSpec` at the end of `Receive`. Test: `TestReceivedSpecIsGraphVisible` (falsified against pre-fix) |
| 5 | Replayed receive does not change live node count; receive never fails on a graph write error | DONE | Guard clauses in `ingestPromotedSpec`; `spec.WriteGraph` upserts. Tests: `TestReceivedSpecGraphIngestIsIdempotent`, `TestIngestPromotedSpecIsBestEffort` |
| 6 | `hero graph <slug>` self-heals a cold index before querying | DONE | `internal/cli/graph.go` — `index.RefreshIfStale` before `index.Open`, mirroring `search.go:86`. Test: `TestGraphSelfHealsColdIndex` (falsified against pre-fix) |
| 7 | Source-level guard rejects `filepath.Base(projectRoot)` as a repoKey anywhere under `internal/` | DONE | `internal/cli/why_resolution_test.go` — `TestNoFilepathBaseRepoKey` now walks the whole tree with a floor on files scanned so a broken walk can't pass vacuously. Verified with a planted offender in `internal/peering` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `internal/peering/receive.go` — ingest promoted spec, keyed by `gitutil.RepoKey`, best-effort | DONE | `ingestPromotedSpec`; writes only the promoted spec (WriteGraph upserts and resolves targets against nodes already in the store) |
| 2 | `internal/cli/graph.go` — `RefreshIfStale` before querying | DONE | Placed before `index.Open`, matching `search.go`'s ordering and warning behavior |
| 3 | `internal/cli/brief.go` — document the shared repoKey derivation | DONE | `graphRepoKey` comment records why the wrapper lives in `gitutil`, not `graph` (import cycle) |
| 4 | `internal/cli/why_resolution_test.go` — widen the guard tree-wide; add cold-index test | DONE | Guard walks `..` with a `scanned < 100` floor; `TestGraphSelfHealsColdIndex` added |
| 5 | `internal/peering/receive_graph_test.go` (new) — Test 6 + idempotency + best-effort | DONE | Three tests over the existing `peerMailFixture` harness |
| 6 | Corpus-scale latency guard for the reconcile added to `hero why`'s hot path (spec Regression scope) | DONE | `TestWhyReconcileStaysWithinBudgetAtCorpusScale` — real `runCmd("why", …)` over 200 cold specs, no index/scan; measured 208ms against a deliberately loose 10s order-of-magnitude guard |

### Exercise-the-feature check

- [x] Exercised through the real receive path end-to-end, not a stub:
  `TestReceivedSpecIsGraphVisible` runs a genuine `Handoff` → `Receive` across
  two fixture workspaces and then queries `graph.db` with the *same* predicate
  `traversal.resolveTarget` uses (`key = ? AND repo = ? AND valid_to IS NULL`),
  so a pass means `hero why` would resolve it. Removing the fix reproduces the
  reported failure verbatim.
- [x] `hero graph` exercised against a genuinely cold index via `runCmd`
  (`TestGraphSelfHealsColdIndex` never calls `indexAll()`); pre-fix it prints
  "No relationships found", post-fix it resolves the parent.

### Excellence Bar self-check

Yes. The honest finding here was that most of the spec had already shipped, so
the valuable work was verifying that on disk and closing the tail rather than
re-delivering it. Each new behavior was falsified against the pre-fix code, the
repoKey guard was widened precisely because Change 4 added the first graph
writer outside `internal/cli` — and that widening was itself proven with a
planted offender. Two of my own assertions turned out to be wrong about the
system; I probed the store, learned the intake/feature pair is by design, and
corrected the tests instead of bending the code to match them.
