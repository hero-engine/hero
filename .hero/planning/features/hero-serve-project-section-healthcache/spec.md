---
title: hero serve Project Section — Phase 5 Health Cache and Peer Probes
slug: hero-serve-project-section-healthcache
type: feature
status: planning
priority: P2
tags: [hero-serve, dashboard, project, cache, peers, performance]
created: 2026-05-19
relations:
  - target: hero-serve-project-section
    kind: parent
  - target: hero-serve-project-section-mvp
    kind: depends-on
horizon: now
---

# `hero serve` Project Section — Phase 5 Health Cache and Peer Probes

## Context

This is **Phase 5 of 5** for the
[`hero-serve-project-section`](../../initiatives/hero-serve-project-section/spec.md)
initiative. See the parent for the full design rationale and boundaries.

Phase 1 ships static cached-output rendering for Health and Peers
(reads whatever exists on disk; shows "as of: never" otherwise).
Phase 5 replaces that with a live, per-project TTL cache plus on-
demand peer probes — the page renders cached values with an explicit
"as of" timestamp and a "Refresh now" affordance that re-runs
`hero check` server-side or probes a single peer.

This is intentionally last in the sequence because it is a swap-in
optimization rather than a new surface. The cache mounts behind the
section partials Phase 1 already shipped; the templates change
minimally.

## Kickoff

Live health + peer caches. Replaces Phase 1's "read whatever's on
disk" with a per-project TTL cache and explicit refresh affordances.

**Status:** planning — Phase 5 of 5; gated on Phase 1 having landed
the `projectpage` package and the Health/Peers section partials.

**Pick up at:** scaffold `internal/serve/healthcache/` with a per-
project TTL cache keyed by `slug` for `hero check` output and by
`slug+peer-alias` for peer reachability. Wire `GET /api/{slug}/health`,
`POST /api/{slug}/health/refresh`, and
`POST /api/{slug}/peers/{alias}/probe`.

→ `.hero/planning/features/hero-serve-project-section-healthcache/spec.md`

**Files:** `internal/serve/projectpage/data/health.go`,
`internal/serve/projectpage/data/peers.go`,
`internal/serve/projectpage/deps.go`, `internal/serve/api.go:51-132`,
`internal/serve/shell/static/js/project.js`

**Skip:** persistent cache across daemon restarts (in-process is
fine); team-shared cache (deferred to `hero-team-server`); a
real graph viz for peers (parent Boundary).

## Goal

Health and Peers sections on `/p/<slug>/project` render cached values
with a visible "as of" timestamp. Clicking "Refresh now" on Health
re-runs `hero check` server-side and streams progress; clicking
"Probe" on a peer row refreshes that single peer's reachability. The
default TTL is 5 minutes, configurable via `hero.json` under
`serve.health_ttl`. The cache lives in-process and repopulates on
demand after a daemon restart.

## Acceptance Criteria

### Health cache

- WHEN a user loads `/p/<slug>/project` THE SYSTEM SHALL render the
  Health section from the cached `hero check` result for that slug if
  one exists within the TTL window, including a visible "as of"
  timestamp.
- IF no cached result exists OR the cached result is older than the
  TTL THEN THE SYSTEM SHALL render the Health section with the last-
  cached values (if any) plus a "stale" marker, and NOT block the page
  on a live run.
- WHEN a user clicks "Refresh now" on the Health section THE SYSTEM
  SHALL POST `/api/{slug}/health/refresh`, invalidate the cache, run
  `hero check` server-side, stream progress without a full page
  reload, and write the result into the cache on completion.
- THE SYSTEM SHALL read the TTL from `hero.json` under
  `serve.health_ttl`, defaulting to 5 minutes when the field is
  absent.
- WHEN concurrent reads request the same slug's health while the
  cache is being populated THE SYSTEM SHALL coalesce them into a
  single in-flight `hero check` rather than spawning N parallel runs.
- THE SYSTEM SHALL expose `GET /api/{slug}/health` returning the
  cached result plus age in seconds.

### Peer probe cache

- WHEN the Peers section loads THE SYSTEM SHALL render each peer's
  cached reachability and last successful call timestamp, never
  blocking the page on a live probe.
- IF a peer's reachability cache is stale OR missing THEN THE SYSTEM
  SHALL show a "probe" affordance on that row.
- WHEN a user clicks "Probe" on a peer row THE SYSTEM SHALL POST
  `/api/{slug}/peers/{alias}/probe`, perform a single reachability
  probe, write the result to the cache, and return the updated row
  state to the page.
- THE SYSTEM SHALL key the peer reachability cache by
  `slug+peer-alias` so the same peer across different parent projects
  has independent cache entries.

### Aggregate page integration

- WHEN `/p/all/project` loads THE SYSTEM SHALL read each project's
  health roll-up from the same cache (no fan-out re-runs of
  `hero check`).

## Approach

### Cache shape

`internal/serve/healthcache/` exposes:

```go
type Cache struct { ... }

type HealthResult struct {
    Slug      string
    Result    *check.Result // existing hero check struct
    Timestamp time.Time
    TTL       time.Duration
}

type PeerResult struct {
    Slug, Alias string
    Reachable   bool
    LastOK      time.Time
    LastError   string
    Timestamp   time.Time
    TTL         time.Duration
}

func New(ttl time.Duration) *Cache
func (c *Cache) Health(slug string) (HealthResult, bool)
func (c *Cache) RefreshHealth(ctx context.Context, slug, projectRoot string) (HealthResult, error)
func (c *Cache) Peer(slug, alias string) (PeerResult, bool)
func (c *Cache) ProbePeer(ctx context.Context, slug, alias string) (PeerResult, error)
```

Internally:

- `map[string]*entry` keyed by `slug` for health, `slug+":"+alias`
  for peers.
- `sync.Mutex` for map access; per-entry `sync.Once` for
  refresh-coalescing.
- `RefreshHealth` invokes the in-process `check` runner the CLI uses
  (`internal/check/*`) rather than shelling out, so the cache update
  is fast and deterministic.

### SSE for `refresh`

`POST /api/{slug}/health/refresh` is dispatched through Phase 3's
`opsrunner` for `run-check` so the user gets the same streaming
progress UI as a manual ops button. On completion, the runner writes
the result into the health cache via a completion callback. This
keeps a single source-of-truth for "check is running" status across
the page.

### Config

`hero.json` gains an optional `serve.health_ttl` field. Parse it in
`internal/config/config.go` as a `time.Duration` (accept `"5m"`,
`"1h"`, etc.). Default to `5 * time.Minute` when absent. Document in
`docs/cli/serve.md`.

## Changes

1. **Create `internal/serve/healthcache/` package**
   - `cache.go` — `Cache` struct, `New`, `Health`, `Peer`,
     `RefreshHealth`, `ProbePeer` per the shape above.
   - `entry.go` — internal entry type with TTL bookkeeping and
     refresh-coalescing.
   - Unit tests for TTL behavior, invalidation, concurrent reads,
     coalesced refresh, and per-peer keying.

2. **Wire `Cache` into `projectpage.Deps`**
   - `internal/serve/projectpage/deps.go` — fill in the
     `HealthCache` field that Phase 1 stubbed.
   - `internal/serve/server.go` — construct one `*healthcache.Cache`
     at startup; inject into per-request `Deps`.

3. **Update Phase 1 loaders to read from the cache**
   - `internal/serve/projectpage/data/health.go` — read from
     `Deps.HealthCache.Health(slug)`; mark stale when older than
     TTL; render "as of" timestamp from the cache entry.
   - `internal/serve/projectpage/data/peers.go` — read peer
     reachability from `Deps.HealthCache.Peer(slug, alias)`; mark
     stale or "never probed" appropriately.

4. **API endpoints in `internal/serve/api.go`**
   - `GET /api/{slug}/health` — cached health result + age in
     seconds.
   - `POST /api/{slug}/health/refresh` — dispatch through
     `opsrunner` for `run-check`; on completion, write to cache.
   - `POST /api/{slug}/peers/{alias}/probe` — call
     `Cache.ProbePeer`, return updated row state.

5. **Client behavior in `project.js`**
   - "Refresh now" button on the Health section opens an SSE stream
     and updates the section partial when the final event arrives.
   - "Probe" button on each peer row POSTs and re-renders the row
     from the JSON response.

6. **Config**
   - `internal/config/config.go` — parse `serve.health_ttl` as a
     `time.Duration`. Default `5m`.
   - Update `docs/cli/serve.md` (or equivalent) with the new field.

7. **Aggregate page**
   - Phase 2's `data/health_rollup.go` already reads cached health;
     no change needed beyond confirming it reads via the cache and
     not direct loader output.

8. **Tests**
   - Unit tests for `Cache.Health`/`Peer` with TTL elapsed/within;
     refresh-coalescing under concurrent reads.
   - Integration test: `POST /api/{slug}/health/refresh` streams
     SSE, cache populates on exit, subsequent `GET
     /api/{slug}/health` returns the fresh value with low age.
   - Integration test: `POST /api/{slug}/peers/{alias}/probe`
     returns updated row; cache entry exists with the new
     timestamp.
   - Config test: missing `serve.health_ttl` defaults to 5m;
     `"15m"` parses correctly.

## Boundaries

- **Persistent cache across daemon restarts** is out of scope.
  In-process cache repopulates on first request.
- **A team-shared cache** is out of scope; deferred to
  `hero-team-server`.
- **A real graph visualization** for the peers map is out of scope
  (parent Boundary).
- **Backfill / warm-up on daemon startup** is out of scope. Cold
  start means first request to each slug fills the cache.
- **Pre-emptive cache refresh on a background timer** is out of
  scope. Refresh is user-driven via the "Refresh now" / "Probe"
  affordances.

## Risks

- **Refresh stampede**. Multiple users / tabs hitting refresh on the
  same slug at once spawns N parallel `hero check` runs.
  Mitigation: per-entry `sync.Once` coalescing in `Cache.RefreshHealth`;
  concurrent-reads unit test.
- **`hero check` running too long for the SSE proxy timeout**.
  Phase 3 already documents the keepalive comment pattern;
  refresh uses the same runner so it inherits the behavior.
- **Cache memory growth on registries with hundreds of projects**.
  Entries are small (one `check.Result` per slug); even 500 projects
  is well under 50MB. Mitigation: documented; revisit only if
  observed.
- **Peer probe interferes with peer's daemon under load**. Each
  probe is a single HTTP call to the peer's status endpoint;
  cached so repeated views don't multiply. Mitigation: probe is
  user-driven, not automatic; documented.

## Validation

- All `healthcache` unit tests pass: TTL, invalidation, concurrent
  reads, refresh-coalescing.
- Integration test: cold start → load page → `health.html`
  renders "as of: never"; click "Refresh now"; SSE streams; on
  completion `GET /api/{slug}/health` returns the fresh value.
- Integration test: cold start → load page → click Probe on one
  peer; row updates with fresh timestamp; cache entry exists.
- Manual: open `/p/<slug>/project`; Health section shows cached
  result with an age; click Refresh now; progress streams; section
  re-renders.
- Manual: edit `hero.json` to `serve.health_ttl = "30s"`; reload
  daemon; observe that Health section marks stale after 30s.
- Performance: page load with cached health and peer state
  completes in <200ms on a project with 100+ specs and 5 peers
  (parent spec's target).

## Changes (files touched on completion)

- `internal/serve/healthcache/cache.go` (new)
- `internal/serve/healthcache/entry.go` (new)
- `internal/serve/healthcache/*_test.go` (new)
- `internal/serve/projectpage/deps.go` (fill `HealthCache` field)
- `internal/serve/projectpage/data/health.go` (read from cache)
- `internal/serve/projectpage/data/peers.go` (read from cache)
- `internal/serve/api.go` (`/api/{slug}/health`,
  `/health/refresh`, `/peers/{alias}/probe`)
- `internal/serve/server.go` (cache construction + injection)
- `internal/serve/shell/static/js/project.js` (refresh + probe
  client wiring)
- `internal/config/config.go` (`serve.health_ttl` field)
- `docs/cli/serve.md` (document new config field)
