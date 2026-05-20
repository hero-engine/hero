package data

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// PeersInputs is the per-request input bundle for the Peers section.
// ProjectRoot empty disables the loader and returns an empty Peers
// (the section renders "no peers registered").
type PeersInputs struct {
	ProjectRoot string
	HeroDir     string
}

// Peers is what the partial renders. Empty Rows means no peers
// configured.
type Peers struct {
	Rows []PeerRow
}

// PeerRow describes one configured sibling repo. Reachability and
// LastSuccessAt come from peer_meta cached on disk — Phase 1 never
// probes live. PeerID empty + ReachableUnknown=true means the peer is
// configured but never been scanned by `hero index`.
type PeerRow struct {
	Alias            string
	Path             string
	PeerID           string
	ScannedAt        time.Time
	ScannedAtPretty  string
	HasScan          bool
	Reachable        bool
	ManifestExists   bool
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
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Alias < rows[j].Alias })
	return Peers{Rows: rows}
}

