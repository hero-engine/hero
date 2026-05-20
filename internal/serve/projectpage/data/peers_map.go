package data

import (
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// PeersMapInputs is the per-request bundle for the Cross-Project Peers
// Map. Projects is the same registry snapshot the directory + health
// rollup consume — the loader walks each project's hero.json Repos and
// emits one row per (source-project, peer-alias) pair.
type PeersMapInputs struct {
	Projects []DirectoryProject
}

// PeersMap is what the partial renders.
type PeersMap struct {
	Rows []PeersMapRow
}

// PeersMapRow is one cross-project peering relationship.
//
// PeerProject is the slug of the peer when it can be matched to a
// project in the registry (by peer_id, falling back to path). Empty
// when the peer is configured but not registered with the daemon —
// useful for spotting "you peered with hero-cloud but it's not
// registered here" mistakes.
type PeersMapRow struct {
	SourceProject   string
	PeerAlias       string
	PeerPath        string
	PeerProject     string // resolved registry slug, "" when unresolved
	Reachable       bool
	ManifestExists  bool
	LastCallAt      time.Time
	LastCallPretty  string
}

// LoadPeersMap fans out each project's configured Repos into a flat
// list of cross-project peering rows.
func LoadPeersMap(in PeersMapInputs) PeersMap {
	if len(in.Projects) == 0 {
		return PeersMap{}
	}

	// Build a path → slug index so peers configured by absolute path
	// can resolve to a registered project. peer_id resolution lives in
	// each project's peer-manifest.yaml; the alias→path shape in
	// hero.json is the source of truth Phase 2 reads.
	slugByPath := map[string]string{}
	for _, p := range in.Projects {
		if p.ProjectRoot == "" {
			continue
		}
		slugByPath[p.ProjectRoot] = p.Slug
	}

	rows := make([]PeersMapRow, 0)
	for _, p := range in.Projects {
		if p.ProjectRoot == "" {
			continue
		}
		cfg, err := config.Load(p.ProjectRoot)
		if err != nil || len(cfg.Repos) == 0 {
			continue
		}
		peers := LoadPeers(PeersInputs{ProjectRoot: p.ProjectRoot, HeroDir: p.HeroDir})
		peersByAlias := make(map[string]PeerRow, len(peers.Rows))
		for _, pr := range peers.Rows {
			peersByAlias[pr.Alias] = pr
		}
		aliases := make([]string, 0, len(cfg.Repos))
		for a := range cfg.Repos {
			aliases = append(aliases, a)
		}
		sort.Strings(aliases)
		for _, alias := range aliases {
			path := cfg.Repos[alias]
			row := PeersMapRow{
				SourceProject: p.Slug,
				PeerAlias:     alias,
				PeerPath:      path,
			}
			if pr, ok := peersByAlias[alias]; ok {
				row.Reachable = pr.Reachable
				row.ManifestExists = pr.ManifestExists
				row.LastCallAt = pr.ScannedAt
				if pr.HasScan {
					row.LastCallPretty = pr.ScannedAtPretty
				}
			}
			if slug, ok := slugByPath[path]; ok {
				row.PeerProject = slug
			}
			rows = append(rows, row)
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].SourceProject != rows[j].SourceProject {
			return rows[i].SourceProject < rows[j].SourceProject
		}
		return rows[i].PeerAlias < rows[j].PeerAlias
	})
	return PeersMap{Rows: rows}
}
