// Package healthcache holds the in-process per-project cache of
// `hero check` results and peer reachability probes that backs the
// /p/<slug>/project Health and Peers sections.
//
// Phase 5 of the hero-serve-project-section initiative.
//
// Design notes:
//
//   - In-process only. The cache repopulates on first request after a
//     daemon restart. Persistent / team-shared caches are deferred.
//   - Health refreshes dispatch through the existing opsrunner so the
//     subprocess gets the same SSE streaming + dedup machinery that
//     backs the Operations buttons.
//   - Peer probes are a single cheap reachability check
//     (`os.Stat(.hero/peer-manifest.yaml)` today) — synchronous,
//     no subprocess involved.
//   - Concurrent refresh callers coalesce onto a single in-flight
//     subprocess so a refresh stampede across N tabs only runs one
//     `hero check`.
package healthcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// OpsDispatcher is the subset of *opsrunner.Runner that the cache
// uses. Declared here so healthcache does not import the opsrunner
// package and the server can wire it without a circular dependency.
type OpsDispatcher interface {
	// Start spawns the named verb for slug+projectRoot, returning the
	// job id and a started flag. When started is false a job was
	// already in flight for the same slug+verb — Wait still works on
	// the returned id.
	Start(ctx context.Context, slug, projectRoot, verb string) (jobID string, started bool, err error)

	// Wait blocks until the job exits or ctx is cancelled.
	Wait(ctx context.Context, slug, jobID string) (exitCode int, err error)
}

// HealthRow mirrors one entry in the on-disk health.json schema.
// Status is "pass" | "warn" | "fail".
type HealthRow struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HealthResult is the cached health snapshot for a single project.
//
// Captured is the on-disk capture timestamp (what the JSON file
// recorded — the moment `hero check` finished). Timestamp is when the
// cache stored it (used for TTL).
type HealthResult struct {
	Slug      string
	Captured  time.Time
	Rows      []HealthRow
	FromDisk  bool
	Timestamp time.Time
	TTL       time.Duration
}

// PeerResult is the cached peer reachability snapshot for a single
// slug+alias pair.
type PeerResult struct {
	Slug      string
	Alias     string
	Reachable bool
	LastOK    time.Time
	LastError string
	Timestamp time.Time
	TTL       time.Duration
}

// PeerLookup is the read interface the projectpage data loader uses.
// Defined as an interface so the loader can take a *Cache without an
// import cycle.
type PeerLookup interface {
	Peer(slug, alias string) (PeerResult, bool)
}

// HealthLookup is the read interface the projectpage data loader uses
// for health. Mirrors PeerLookup's shape so the loader signatures stay
// symmetrical.
type HealthLookup interface {
	Health(slug string) (HealthResult, bool)
}

// Cache is the per-daemon health+peer cache. Safe for concurrent use.
type Cache struct {
	ttl  time.Duration
	ops  OpsDispatcher
	clk  func() time.Time
	peer peerProber

	mu      sync.Mutex
	health  map[string]*healthEntry
	peerMap map[string]*peerEntry
}

// peerProber is the function the cache calls to perform a single peer
// reachability check. Default implementation lives in peers.go;
// exposed as an interface so tests can inject a deterministic stub.
type peerProber func(ctx context.Context, slug, alias, peerPath string) (PeerResult, error)

// Options configures a new Cache. All fields are optional.
type Options struct {
	// Ops is the opsrunner dispatcher used to refresh health. Nil-tolerant
	// — RefreshHealth returns an error when ops is nil, but Health reads
	// still work (they just always miss the cache until something else
	// populates the on-disk artifact).
	Ops OpsDispatcher

	// Now is the clock used for cache timestamps + TTL math. Zero
	// defaults to time.Now.
	Now func() time.Time

	// PeerProber overrides the default peer reachability check. Zero
	// defaults to the manifest-stat prober.
	PeerProber peerProber
}

// New constructs a Cache with the given TTL.
func New(ttl time.Duration, opts Options) *Cache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	clk := opts.Now
	if clk == nil {
		clk = time.Now
	}
	prober := opts.PeerProber
	if prober == nil {
		prober = manifestPeerProber
	}
	return &Cache{
		ttl:     ttl,
		ops:     opts.Ops,
		clk:     clk,
		peer:    prober,
		health:  make(map[string]*healthEntry),
		peerMap: make(map[string]*peerEntry),
	}
}

// TTL returns the configured TTL. Useful for the template to render
// "stale" markers.
func (c *Cache) TTL() time.Duration { return c.ttl }

// Health returns the cached health result for slug. The bool reports
// whether a cached entry exists at all (regardless of staleness).
// Callers compare the result's Timestamp+TTL against the current time
// to decide whether to render a "stale" marker.
func (c *Cache) Health(slug string) (HealthResult, bool) {
	if c == nil || slug == "" {
		return HealthResult{}, false
	}
	c.mu.Lock()
	e, ok := c.health[slug]
	c.mu.Unlock()
	if !ok {
		return HealthResult{}, false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.result == nil {
		return HealthResult{}, false
	}
	return *e.result, true
}

// Peer returns the cached peer reachability for slug+alias.
func (c *Cache) Peer(slug, alias string) (PeerResult, bool) {
	if c == nil || slug == "" || alias == "" {
		return PeerResult{}, false
	}
	key := peerKey(slug, alias)
	c.mu.Lock()
	e, ok := c.peerMap[key]
	c.mu.Unlock()
	if !ok {
		return PeerResult{}, false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.result == nil {
		return PeerResult{}, false
	}
	return *e.result, true
}

// RefreshHealth dispatches `hero check --json` through the opsrunner
// for slug+projectRoot, waits for it to finish, reads the on-disk
// health.json artifact, stores it in the cache, and returns the new
// result.
//
// Concurrent callers for the same slug coalesce onto a single
// subprocess — the second caller blocks on the first's completion
// and receives the same HealthResult.
func (c *Cache) RefreshHealth(ctx context.Context, slug, projectRoot string) (HealthResult, error) {
	if c == nil {
		return HealthResult{}, errors.New("healthcache: nil cache")
	}
	if c.ops == nil {
		return HealthResult{}, errors.New("healthcache: no ops dispatcher configured")
	}
	if slug == "" {
		return HealthResult{}, errors.New("healthcache: empty slug")
	}
	if projectRoot == "" {
		return HealthResult{}, errors.New("healthcache: empty projectRoot")
	}

	c.mu.Lock()
	e, ok := c.health[slug]
	if !ok {
		e = &healthEntry{}
		c.health[slug] = e
	}
	c.mu.Unlock()

	// Per-entry refresh-coalescing. The first caller into a fresh
	// "wave" runs the subprocess; concurrent callers wait on the wave's
	// done channel and read its result. After the wave completes, the
	// wave pointer is reset so the next caller starts a fresh refresh
	// rather than reusing the stale wave's result indefinitely (TTL
	// freshness is the cache's job — not the wave's).
	e.mu.Lock()
	if e.wave != nil {
		w := e.wave
		e.mu.Unlock()
		<-w.done
		return w.result, w.err
	}
	w := &refreshWave{done: make(chan struct{})}
	e.wave = w
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.wave = nil
		e.mu.Unlock()
		close(w.done)
	}()

	jobID, _, err := c.ops.Start(ctx, slug, projectRoot, "run-check-json")
	if err != nil {
		w.err = fmt.Errorf("dispatch run-check-json: %w", err)
		return HealthResult{}, w.err
	}
	exit, err := c.ops.Wait(ctx, slug, jobID)
	if err != nil {
		w.err = fmt.Errorf("wait run-check-json: %w", err)
		return HealthResult{}, w.err
	}
	if exit != 0 {
		// Non-zero exit doesn't necessarily mean the JSON file wasn't
		// written — `hero check` exits non-zero whenever issues are
		// found, but still writes the JSON. Read it anyway, but
		// surface the exit code in the error if the file is missing.
	}

	heroDir := filepath.Join(projectRoot, ".hero")
	result, readErr := readHealthFromDisk(slug, heroDir, c.ttl, c.clk())
	if readErr != nil {
		if exit != 0 {
			w.err = fmt.Errorf("run-check-json exit %d and no health.json on disk: %w", exit, readErr)
		} else {
			w.err = fmt.Errorf("read health.json: %w", readErr)
		}
		return HealthResult{}, w.err
	}

	c.storeHealth(slug, result)
	w.result = result
	return result, nil
}

// ProbePeer performs a single reachability check for slug+alias, writes
// the result into the cache, and returns it. Probes are cheap
// (`os.Stat` on the peer's `.hero/peer-manifest.yaml` by default) so
// this method is synchronous.
//
// peerPath is the peer's project-root path on disk; passed by the
// caller (typically the API handler) after resolving the alias against
// the project's hero.json. Empty peerPath marks the peer unreachable.
func (c *Cache) ProbePeer(ctx context.Context, slug, alias, peerPath string) (PeerResult, error) {
	if c == nil {
		return PeerResult{}, errors.New("healthcache: nil cache")
	}
	if slug == "" || alias == "" {
		return PeerResult{}, errors.New("healthcache: empty slug or alias")
	}
	result, err := c.peer(ctx, slug, alias, peerPath)
	if err != nil {
		return PeerResult{}, err
	}
	result.Slug = slug
	result.Alias = alias
	result.Timestamp = c.clk()
	result.TTL = c.ttl
	c.storePeer(slug, alias, result)
	return result, nil
}

// RefreshFromDisk re-reads .hero/cache/health.json for slug under
// projectRoot and stores it in the cache without dispatching a
// subprocess. Used by the /api/{slug}/health/refresh background
// goroutine after the opsrunner-managed `hero check --json` subprocess
// exits — the runner has already produced the on-disk artifact, so we
// just need to pull it into the in-memory layer.
func (c *Cache) RefreshFromDisk(slug, projectRoot string) (HealthResult, error) {
	if c == nil {
		return HealthResult{}, errors.New("healthcache: nil cache")
	}
	if slug == "" || projectRoot == "" {
		return HealthResult{}, errors.New("healthcache: empty slug or projectRoot")
	}
	heroDir := filepath.Join(projectRoot, ".hero")
	result, err := readHealthFromDisk(slug, heroDir, c.ttl, c.clk())
	if err != nil {
		return HealthResult{}, err
	}
	c.storeHealth(slug, result)
	return result, nil
}

// InvalidateHealth drops the cached entry for slug. Used by tests; the
// API surface doesn't expose explicit invalidation today — RefreshHealth
// overwrites unconditionally.
func (c *Cache) InvalidateHealth(slug string) {
	if c == nil || slug == "" {
		return
	}
	c.mu.Lock()
	delete(c.health, slug)
	c.mu.Unlock()
}

// storeHealth installs result under slug in the cache.
func (c *Cache) storeHealth(slug string, result HealthResult) {
	c.mu.Lock()
	e, ok := c.health[slug]
	if !ok {
		e = &healthEntry{}
		c.health[slug] = e
	}
	c.mu.Unlock()
	e.mu.Lock()
	r := result
	e.result = &r
	e.mu.Unlock()
}

// storePeer installs result under slug+alias in the cache.
func (c *Cache) storePeer(slug, alias string, result PeerResult) {
	key := peerKey(slug, alias)
	c.mu.Lock()
	e, ok := c.peerMap[key]
	if !ok {
		e = &peerEntry{}
		c.peerMap[key] = e
	}
	c.mu.Unlock()
	e.mu.Lock()
	r := result
	e.result = &r
	e.mu.Unlock()
}

// peerKey is the cache-map key for a peer entry. Exposed for tests.
func peerKey(slug, alias string) string { return slug + ":" + alias }

// readHealthFromDisk parses .hero/cache/health.json for slug and
// returns a HealthResult populated with the on-disk captured-at + rows.
func readHealthFromDisk(slug, heroDir string, ttl time.Duration, now time.Time) (HealthResult, error) {
	path := filepath.Join(heroDir, "cache", "health.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return HealthResult{}, err
	}
	var doc struct {
		CapturedAt time.Time   `json:"captured_at"`
		Rows       []HealthRow `json:"rows"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return HealthResult{}, fmt.Errorf("parse health.json: %w", err)
	}
	return HealthResult{
		Slug:      slug,
		Captured:  doc.CapturedAt,
		Rows:      doc.Rows,
		FromDisk:  true,
		Timestamp: now,
		TTL:       ttl,
	}, nil
}

// manifestPeerProber is the default peer reachability check: a single
// os.Stat of the peer's peer-manifest.yaml. Cheap, no network, and
// matches what Phase 1's loader already considers "reachable".
func manifestPeerProber(_ context.Context, _ /* slug */, _ /* alias */, peerPath string) (PeerResult, error) {
	if peerPath == "" {
		return PeerResult{Reachable: false, LastError: "no peer path configured"}, nil
	}
	manifest := filepath.Join(peerPath, ".hero", "peer-manifest.yaml")
	if _, err := os.Stat(manifest); err != nil {
		if os.IsNotExist(err) {
			return PeerResult{Reachable: false, LastError: "peer manifest missing"}, nil
		}
		return PeerResult{Reachable: false, LastError: err.Error()}, nil
	}
	return PeerResult{Reachable: true, LastOK: time.Now().UTC()}, nil
}
