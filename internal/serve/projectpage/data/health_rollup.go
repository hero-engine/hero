// health_rollup.go — Cross-project Health Rollup loader for the
// aggregate /p/all/project page.
//
// Color rule (canonical — evolve here, not in templates):
//
//   Green  = zero health items across all projects.
//   Yellow = ONLY stale-spec or missing-kickoff items (no drift /
//            no broken peer manifests / no orphan files).
//   Red    = drift detected OR broken peer manifest OR orphan files
//            present anywhere in the registered set.
//
// The same colour rule applies per-project for the Project Directory
// rollup column (via healthRollupColor). Both the aggregate counts and
// the per-project colours come from the same on-disk cached artifact
// Phase 1's LoadHealth produces — Phase 2 does NOT recompute health.
//
// When a project's .hero/cache/health.json is missing, the project
// contributes nothing to the rollup counts and its directory colour is
// "unknown" (green-ish absence of evidence, not a positive all-clear).
package data

import "strings"

// HealthRollupInputs is the per-request bundle for the Cross-Project
// Health Rollup section. Projects mirrors DirectoryProject — same
// per-project pointers, same fan-out shape.
type HealthRollupInputs struct {
	Projects []DirectoryProject
}

// HealthRollup is what the partial renders. AllClear=true means zero
// items across the entire registered set (green colour).
type HealthRollup struct {
	AllClear        bool
	OverallColor    string // "green" | "yellow" | "red" | "unknown"
	StaleSpecs      int
	MissingKickoffs int
	BrokenManifests int
	OrphanFiles     int
	Drift           int
	// PerProject lists the per-project breakdown rows, sorted by slug,
	// so the partial can render a drill-down without re-doing the
	// fan-out itself.
	PerProject []HealthRollupProject
}

// HealthRollupProject is one project's contribution to the rollup, used
// by the partial's drill-down expansion.
type HealthRollupProject struct {
	Slug            string
	Color           string
	StaleSpecs      int
	MissingKickoffs int
	BrokenManifests int
	OrphanFiles     int
	Drift           int
	HasArtifact     bool
}

// LoadHealthRollup fans out Phase 1's LoadHealth across every project
// and folds the categorised row counts into one aggregate result.
//
// Recognised row Name conventions (best-effort — Phase 5 owns the
// canonical set; Phase 2 sniffs the category from the name + status
// fields produced by `hero check`):
//
//   "stale" / "stale-spec"     → StaleSpecs
//   "missing-kickoff" / "kickoff" → MissingKickoffs
//   "broken-manifest" / "peer-manifest" → BrokenManifests
//   "orphan" / "orphan-files"  → OrphanFiles
//   "drift"                    → Drift
//
// Anything else with a non-pass status falls into the drift bucket so
// it still drives the colour rule.
func LoadHealthRollup(in HealthRollupInputs) HealthRollup {
	out := HealthRollup{
		PerProject: make([]HealthRollupProject, 0, len(in.Projects)),
	}
	for _, p := range in.Projects {
		proj := HealthRollupProject{Slug: p.Slug, Color: "unknown"}
		if p.HeroDir != "" {
			h := LoadHealth(HealthInputs{HeroDir: p.HeroDir})
			proj.HasArtifact = h.HasArtifact
			peers := LoadPeers(PeersInputs{ProjectRoot: p.ProjectRoot, HeroDir: p.HeroDir})
			for _, row := range h.Rows {
				if row.Status == "pass" || row.Status == "" || row.Status == "info" {
					continue
				}
				bucket := categorizeHealthRow(row.Name)
				switch bucket {
				case "stale":
					proj.StaleSpecs++
				case "kickoff":
					proj.MissingKickoffs++
				case "manifest":
					proj.BrokenManifests++
				case "orphan":
					proj.OrphanFiles++
				default:
					proj.Drift++
				}
			}
			// A missing-but-configured peer manifest is broken regardless
			// of what the cached health artifact had to say.
			for _, peer := range peers.Rows {
				if peer.Path != "" && !peer.ManifestExists {
					proj.BrokenManifests++
				}
			}
			proj.Color = healthRollupColor(h, peers) // uses per-project view, keeps directory + rollup aligned
		}
		out.StaleSpecs += proj.StaleSpecs
		out.MissingKickoffs += proj.MissingKickoffs
		out.BrokenManifests += proj.BrokenManifests
		out.OrphanFiles += proj.OrphanFiles
		out.Drift += proj.Drift
		out.PerProject = append(out.PerProject, proj)
	}
	out.OverallColor = aggregateColor(out)
	out.AllClear = out.OverallColor == "green"
	return out
}

// categorizeHealthRow maps a free-form row name (as produced by
// hero check) to one of the rollup buckets. Match is substring +
// case-insensitive so we tolerate the cosmetic variation hero check
// already ships ("stale specs" vs "stale-specs" vs "Stale: specs").
func categorizeHealthRow(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "stale"):
		return "stale"
	case strings.Contains(n, "kickoff"):
		return "kickoff"
	case strings.Contains(n, "manifest") || strings.Contains(n, "peer manifest"):
		return "manifest"
	case strings.Contains(n, "orphan"):
		return "orphan"
	case strings.Contains(n, "drift"):
		return "drift"
	}
	return ""
}

// healthRollupColor applies the colour rule to one project's health
// data + peer state. Exposed inside the package so directory.go and
// health_rollup.go agree by construction.
func healthRollupColor(h Health, peers Peers) string {
	if !h.HasArtifact {
		// No on-disk artifact and no broken manifests is treated as
		// "unknown" — the per-project page still shows "as of: never"
		// and the rollup must not pretend to know.
		if hasBrokenManifest(peers) {
			return "red"
		}
		return "unknown"
	}
	var stale, kickoff, manifest, orphan, drift int
	for _, r := range h.Rows {
		if r.Status == "pass" || r.Status == "" || r.Status == "info" {
			continue
		}
		switch categorizeHealthRow(r.Name) {
		case "stale":
			stale++
		case "kickoff":
			kickoff++
		case "manifest":
			manifest++
		case "orphan":
			orphan++
		case "drift":
			drift++
		default:
			drift++
		}
	}
	if hasBrokenManifest(peers) {
		manifest++
	}
	if drift > 0 || manifest > 0 || orphan > 0 {
		return "red"
	}
	if stale > 0 || kickoff > 0 {
		return "yellow"
	}
	return "green"
}

// aggregateColor folds the across-projects totals into one colour using
// the same rule as the per-project view.
func aggregateColor(r HealthRollup) string {
	if r.Drift > 0 || r.BrokenManifests > 0 || r.OrphanFiles > 0 {
		return "red"
	}
	if r.StaleSpecs > 0 || r.MissingKickoffs > 0 {
		return "yellow"
	}
	return "green"
}

// hasBrokenManifest reports whether any configured peer is missing its
// on-disk manifest. A configured peer with no manifest is a broken
// configuration — it triggers the red colour by rule.
func hasBrokenManifest(p Peers) bool {
	for _, row := range p.Rows {
		if row.Path != "" && !row.ManifestExists {
			return true
		}
	}
	return false
}
