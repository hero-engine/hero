package data

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// CachedPeer is the minimum shape the Peers loader reads from the
// shared peer cache. Mirrors healthcache.PeerResult so the cache's
// concrete type satisfies this interface without a circular import.
// Phase 5 of hero-serve-project-section.
type CachedPeer struct {
	Reachable bool
	LastOK    time.Time
	LastError string
	Timestamp time.Time
	TTL       time.Duration
}

// PeerLookup is the read interface for the in-process peer cache.
// Nil-tolerant: when Inputs.Cache is nil, the loader falls back to the
// Phase 1 manifest-stat reachability heuristic.
type PeerLookup interface {
	Peer(slug, alias string) (CachedPeer, bool)
}

// PeersInputs is the per-request input bundle for the Peers section.
// ProjectRoot empty disables the loader and returns an empty Peers
// (the section renders "no peers registered").
type PeersInputs struct {
	ProjectRoot string
	HeroDir     string
	Slug        string
	Cache       PeerLookup
}

// Peers is what the partial renders. Empty Rows means no peers
// configured.
type Peers struct {
	Rows []PeerRow

	// ProbeAvailable is true when Cache was wired — the template uses
	// this to decide whether to render the per-row "Probe" button.
	ProbeAvailable bool

	// Slug plumbed through so the per-row buttons can build URLs.
	Slug string
}

// PeerRow describes one configured sibling repo. Reachability and
// LastSuccessAt come from peer_meta cached on disk — Phase 1 never
// probes live. PeerID empty + ReachableUnknown=true means the peer is
// configured but never been scanned by `hero index`.
type PeerRow struct {
	Alias           string
	Path            string
	PeerID          string
	ScannedAt       time.Time
	ScannedAtPretty string
	HasScan         bool
	Reachable       bool
	ManifestExists  bool

	// Phase 5: live cache state for this peer.
	HasCache    bool
	CacheStale  bool
	CacheAt     time.Time
	CacheAgo    string
	LastOK      time.Time
	LastOKPretty string
	LastError   string
}

// LoadPeers reads the local hero.json for configured sibling repos
// (the Repos map) and reports cached reachability metadata. Never
// probes the network.
func LoadPeers(in PeersInputs) Peers {
	if in.ProjectRoot == "" {
		return Peers{}
	}
	cfg, err := config.Load(in.ProjectRoot)
	if err != nil {
		return Peers{}
	}
	if len(cfg.Repos) == 0 {
		return Peers{}
	}
	rows := make([]PeerRow, 0, len(cfg.Repos))
	for alias, path := range cfg.Repos {
		row := PeerRow{Alias: alias, Path: path}
		if meta, ok := cfg.RepoMeta[alias]; ok {
			row.PeerID = meta.PeerID
			if meta.ScannedAt != "" {
				if t, perr := time.Parse(time.RFC3339, meta.ScannedAt); perr == nil {
					row.ScannedAt = t
					row.ScannedAtPretty = t.Format("2006-01-02 15:04")
					row.HasScan = true
				}
			}
		}
		// "Reachable" is a cached judgement: if the peer's manifest is
		// readable on disk we consider it reachable. We never network-
		// probe in Phase 1.
		manifestPath := filepath.Join(path, ".hero", "peer-manifest.yaml")
		if _, statErr := os.Stat(manifestPath); statErr == nil {
			row.ManifestExists = true
			row.Reachable = true
		}
		// Phase 5: layer the in-memory cache result (if any) on top of
		// the manifest-stat heuristic. A live probe result overrides
		// the heuristic — it's the authoritative signal.
		if in.Cache != nil && in.Slug != "" {
			if cached, ok := in.Cache.Peer(in.Slug, alias); ok {
				row.HasCache = true
				row.Reachable = cached.Reachable
				row.CacheAt = cached.Timestamp
				row.CacheAgo = relativeAgo(cached.Timestamp, time.Now())
				row.LastOK = cached.LastOK
				if !cached.LastOK.IsZero() {
					row.LastOKPretty = cached.LastOK.Format("2006-01-02 15:04")
				}
				row.LastError = cached.LastError
				if cached.TTL > 0 && !cached.Timestamp.IsZero() {
					if time.Since(cached.Timestamp) > cached.TTL {
						row.CacheStale = true
					}
				}
			}
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Alias < rows[j].Alias })
	return Peers{Rows: rows, ProbeAvailable: in.Cache != nil && in.Slug != "", Slug: in.Slug}
}

