package data

import (
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// DirectoryInputs is the per-request input bundle for the cross-project
// Project Directory section. Projects is the snapshot of registered
// projects the aggregate handler hands in. An empty Projects slice
// renders the "no projects registered" empty state.
type DirectoryInputs struct {
	Projects []DirectoryProject
}

// DirectoryProject is one project's worth of input. The aggregate
// handler builds this slice from Server.registry + Server.projects so
// the loader stays pure (no filesystem walking by registry).
type DirectoryProject struct {
	Slug        string
	ProjectRoot string
	HeroDir     string
	RegisteredAt time.Time
}

// Directory is the cross-project rollup table.
type Directory struct {
	Rows []DirectoryRow
}

// DirectoryRow is one row in the Project Directory. Health is a colour
// string ("green"/"yellow"/"red") computed by the health_rollup loader's
// per-project rule; we recompute it here from the same per-project
// Health loader so the rollup never disagrees with the directory.
//
// Degraded marks a project whose per-project loaders raised an error or
// found a broken on-disk artifact (e.g. missing .hero/ or corrupt peer
// manifest). The template renders a degraded indicator in that case.
type DirectoryRow struct {
	Slug             string
	Name             string
	Path             string
	LastTouchedAt    time.Time
	LastTouchedPretty string
	Health           string // "green" | "yellow" | "red" | "unknown"
	SpecCount        int
	PeerCount        int
	Tracker          string // "github" | "jira" | "linear" | "" (none/unconfigured)
	Degraded         bool
	DegradedReason   string
}

// LoadDirectory builds the rollup directory rows. The loader is
// failure-tolerant: a single project's loader hiccup yields a row with
// Degraded=true rather than dropping the project from the page.
func LoadDirectory(in DirectoryInputs) Directory {
	if len(in.Projects) == 0 {
		return Directory{}
	}
	rows := make([]DirectoryRow, 0, len(in.Projects))
	for _, p := range in.Projects {
		rows = append(rows, buildDirectoryRow(p))
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Slug < rows[j].Slug })
	return Directory{Rows: rows}
}

// buildDirectoryRow is split out so the aggregate handler can call it
// per-project inside a recover() block. Failures fold into a degraded
// row instead of poisoning the page.
func buildDirectoryRow(p DirectoryProject) (row DirectoryRow) {
	row = DirectoryRow{
		Slug: p.Slug,
		Name: p.Slug,
		Path: p.ProjectRoot,
	}
	defer func() {
		if r := recover(); r != nil {
			row.Degraded = true
			if row.DegradedReason == "" {
				row.DegradedReason = "loader panic"
			}
		}
	}()

	if p.ProjectRoot == "" || p.HeroDir == "" {
		row.Degraded = true
		row.DegradedReason = "missing project path"
		row.Health = "unknown"
		return row
	}

	// Identity loader fans out git + spec.Discover. Wrap any subtle
	// failure into a degraded row.
	id := LoadIdentity(IdentityInputs{
		ProjectRoot: p.ProjectRoot, HeroDir: p.HeroDir, Slug: p.Slug,
	})
	if id.Name != "" {
		row.Name = id.Name
	}
	row.LastTouchedAt = id.LastTouchedAt
	if !id.LastTouchedAt.IsZero() {
		row.LastTouchedPretty = id.LastTouchedAt.Format("2006-01-02 15:04")
	}
	row.SpecCount = id.SpecCount
	row.PeerCount = id.PeerCount

	// Tracker — read hero.json. config.Load returns ErrNotExist
	// silently as a zero config; missing tracker block is the empty
	// string by design.
	if cfg, cerr := config.Load(p.ProjectRoot); cerr == nil {
		if cfg.Tracker != nil && cfg.Tracker.Type != "" && !strings.EqualFold(cfg.Tracker.Type, "none") {
			row.Tracker = cfg.Tracker.Type
		}
	} else {
		row.Degraded = true
		row.DegradedReason = "config.Load: " + cerr.Error()
	}

	// Health colour — runs the per-project health loader and folds it
	// into the rollup colour rule.
	health := LoadHealth(HealthInputs{HeroDir: p.HeroDir})
	peers := LoadPeers(PeersInputs{ProjectRoot: p.ProjectRoot, HeroDir: p.HeroDir})
	row.Health = healthRollupColor(health, peers)
	if row.Health == "" {
		row.Health = "unknown"
	}
	return row
}
