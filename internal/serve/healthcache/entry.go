package healthcache

import "sync"

// healthEntry is the per-slug bookkeeping the cache holds for health
// results. The outer map lock guards entry creation/lookup; entry-level
// mu guards the pointer + the in-flight refresh wave.
type healthEntry struct {
	mu sync.Mutex

	// result is the last-known HealthResult for this slug. Nil until
	// the first successful refresh populates it.
	result *HealthResult

	// wave is the in-flight refresh wave. Concurrent RefreshHealth
	// callers for the same slug coalesce onto this wave's done channel
	// rather than spawning N parallel subprocesses.
	wave *refreshWave
}

// peerEntry mirrors healthEntry for peer probes. Peer probes are
// synchronous (no subprocess), so there's no refresh wave to track —
// concurrent ProbePeer callers each run the cheap stat themselves and
// the last writer wins. That's intentionally simpler than health: an
// `os.Stat` race is harmless, and we'd pay more in lock contention
// than we'd save in stats.
type peerEntry struct {
	mu     sync.Mutex
	result *PeerResult
}

// refreshWave coordinates concurrent RefreshHealth callers for a
// single slug. The first caller into a wave runs the subprocess; later
// callers block on done and read the same result.
type refreshWave struct {
	done   chan struct{}
	result HealthResult
	err    error
}
