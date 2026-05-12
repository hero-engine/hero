---
title: Unified Search — Merge Federation Graph and On-Disk Spec Index
type: feature
status: delivering
priority: P1
tags: [search, federation, graph-memory, cross-repo, fts5]
created: 2026-04-27
relations:
  - target: graph-memory-federation
    kind: child
  - target: graph-memory-7c-live-test
    kind: sibling
horizon: now
smoke: deferred
---

## Implementation status

**Phase 1 (default graph search covers federation data):** delivered
in the phase 7c federation work — the repo filter on graph search was
removed so pulled teammate data is always visible.

**Phase 3 (sibling-repo scan into local graph.db):** delivered. `hero scan`
now iterates `hero.json`'s `repos` block, derives each accessible
sibling's git remote-origin key via `gitutil.RepoKey`, and ingests its
specs into the local `graph.db`. Search results show the origin repo
in `[brackets]` when it differs from the local one.

**Phase 2 (FTS5 fallback):** intentionally skipped — Phase 3 covers the
same gap by always ingesting sibling specs at scan time, so the
fallback isn't needed.

Limitation: the local graph.db has a UNIQUE (type, key) constraint, so
two repos with the same spec slug will overwrite each other. In practice
slugs are repo-scoped per workspace; lifting this to (type, repo, key)
is deferred to a future v2 if it becomes a real-world problem.

## Problem

There are currently two disconnected search surfaces in `hero search`:

| Path | Triggered by | Data source | Federation-aware? |
|---|---|---|---|
| Graph search | default (no flags) | `graph.db` — local + pulled federation data | ✅ yes |
| FTS5 spec search | `--cross-repo`, filters, `--specs` | On-disk spec files from `repos` in `hero.json` | ❌ no |

The gap bites in two directions:

1. **Federation data is invisible to `--cross-repo`.** A teammate's
   specs, notes, and features land in `graph.db` after a pull, but
   `hero search --cross-repo` never queries `graph.db` — it only
   walks on-disk directories listed in `hero.json`'s `repos` block.
   A developer who doesn't have a related repo cloned locally gets
   zero cross-repo results even if they've pulled federation data
   that covers it.

2. **On-disk `repos` entries are invisible to default graph search.**
   The FTS5 index for a configured sibling repo is rich (full spec
   markdown, tags, status) but completely bypassed when using the
   default graph-aware search path. Users have to know to pass
   `--cross-repo` to access it, and even then it's a separate corpus
   with different result formatting.

3. **`repos` in `hero.json` implies intent to search across those
   repos**, but the search subsystem doesn't honour that intent
   automatically — developers have to opt in on every query with a
   flag.

## Goal

A single `hero search <query>` that returns the best results across:
- The local graph (own work, pulled teammate data)
- Configured sibling repos (whether accessed via federation pull or
  on-disk FTS5 index)

No flags required. Federation-pulled data and locally-cloned repo
data surface together, ranked by relevance.

## Design

### Phase 1 — Graph search covers `repos` entries automatically

When executing the default graph search, also query graph.db for
nodes whose `repo` key matches any repo configured in `hero.json`'s
`repos` block (by remote-origin key, not directory alias). Since
after a `hero sync graph pull` all federation data is already in
graph.db regardless of `repo` tag, the current repo-filter removal
(phase 7c fix) already handles this for pulled data.

**Remaining gap:** repos listed in `hero.json` that have never been
pulled (local-only teams, no server) won't appear in graph.db.

### Phase 2 — FTS5 fallback for repos not in graph.db

After graph search runs, check which configured `repos` have zero
nodes in graph.db (i.e., haven't been pulled). For those repos,
fall back to their on-disk FTS5 spec index if the path is accessible.
Merge results, dedup by slug.

This gives:
- Pulled repos → graph.db (federation data, richer node types)
- Unpulled repos → FTS5 (spec-only, but still surfaced)

### Phase 3 — Scan configured repos into graph.db on `hero scan`

When `hero scan` runs, also scan any `repos` entries with accessible
on-disk paths, ingesting their specs and notes into the local
`graph.db` with their correct remote-origin repo key. This makes
graph search the single authoritative surface and removes the need
for the FTS5 fallback path.

This is the end state: graph.db is the union of local work + pulled
federation data + locally accessible sibling repos.

## What "repo key" means across the seam

The bridge between the two surfaces is the repo key:
- `graph.db` nodes carry `repo = <remote-origin-slug>` (e.g.
  `chet-bellows/hero`), set at push/pull time using `gitutil.RepoKey`.
- `hero.json`'s `repos` block maps *aliases* to *paths*
  (`"sibling-a": "../some-sibling"`).

To join them, `hero scan` and the search layer must resolve each
`repos` alias to its remote-origin key (via `gitutil.RepoKey` on the
path). The alias is a display label; the remote-origin key is the
stable identity used in graph.db.

**Migration:** existing graph.db rows pushed before `gitutil.RepoKey`
was introduced carry directory-name repo keys (e.g.
`alice-workspace`). A one-time migration or re-scan is needed to
re-tag them, or a compatibility lookup that checks both forms.

## Success criteria

- `hero search "some-cross-repo-term"` returns results from
  federation-pulled teammate data with no flags
- `hero search "some-cross-repo-term"` returns results from an
  on-disk sibling repo listed in `repos`, even without a pull
- `hero search --cross-repo` continues to work but is no longer the
  *only* way to get cross-repo results
- Search results include a `repo` label so users know which workspace
  a result came from
- `hero scan` ingests sibling repo specs into `graph.db` with their
  correct remote-origin key

## Out of scope

- Ranking/scoring across surfaces (phase 1 treats all results equally
  by recency; relevance ranking is a separate workstream)
- Deduplication when the same spec exists in both graph.db and FTS5
  index (phase 2 dedup is slug-level; content-level dedup is future)
- Cloud-side cross-repo search (handled by `hero impact --cross-repo`
  via the federation API — different use case)

## Files to touch

| File | Change |
|---|---|
| `internal/cli/brief.go` | Phase 2: after graph search, query FTS5 for repos missing from graph.db |
| `internal/cli/search.go` | Route `--cross-repo` through graph-aware path first |
| `internal/graph/ingest.go` or new `internal/graph/scan_repos.go` | Phase 3: scan `repos` entries into graph.db |
| `internal/gitutil/gitutil.go` | Already has `RepoKey` — expose a batch form for multiple paths |
| `internal/config/config.go` | Expose resolved repo keys alongside repo paths |

## References

- `internal/cli/brief.go:runGraphSearch` — current graph search
- `internal/cli/search.go:runSearch` — FTS5 path and routing logic
- `internal/cli/sync_graph.go` — uses `gitutil.RepoKey` for federation
- `internal/gitutil/gitutil.go:RepoKey` — remote-origin key derivation
- `.hero/hero.json` — `repos` block with sibling repo aliases
- [graph-memory-federation/spec.md](../graph-memory-federation/spec.md)
