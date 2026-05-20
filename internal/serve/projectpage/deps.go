// Package projectpage hosts the per-project Project page — a read-only
// dashboard at GET /p/<slug>/project (plus /project as a single-project
// fallback) composed of eight stacked sections: Identity, Health,
// Stack, Registry, Peers, Trackers, Knowledge, Config.
//
// Phase 1 of the hero-serve-project-section initiative. Read-only:
// no live `hero check` invocation, no peer probes, no ops dispatch.
// Section loaders read whatever is on disk and degrade to "no X" /
// "as of: never" empty states when their data source is missing.
//
// Per the spec, this package replaced the older surface that lived at
// /project. That surface (project-shape rollup) moved to
// internal/serve/pages/rollup and now mounts at /rollup.
package projectpage

import (
	"time"

	"github.com/hero-engine/hero/internal/serve/projectpage/data"
)

// Deps is the per-request bundle a Project page render needs. It is
// rebuilt per request from the URL slug — never captured at startup —
// so adding or removing a project at runtime is reflected on the next
// page load.
//
// Phase 1 ships read-only with no Phase 2-5 hooks declared yet (health
// cache, peer cache, ops runner). Those land on this struct as nil-
// tolerant fields in their respective phases without touching the
// section-loader signatures.
type Deps struct {
	// ProjectRoot is the absolute path to the project being rendered.
	// Empty disables every file-system-backed loader (each section
	// degrades to its empty state).
	ProjectRoot string

	// HeroDir is the absolute path to .hero/ inside ProjectRoot.
	// Knowledge counts, spec discovery, and the peer manifest reader
	// all root here.
	HeroDir string

	// Slug is the URL slug under which this project is served. Used
	// for the localStorage collapse-state key on the rendered page and
	// to look up the registry entry. Empty in the single-project
	// fallback when no slug is in the URL — handler synthesizes the
	// project root's basename in that case.
	Slug string

	// RegistryEntry is the project's entry in the global registry
	// (~/.hero/projects.json), or nil when the project isn't
	// registered. Drives the Registry Membership section. Nil renders
	// the "not registered" empty state.
	RegistryEntry *RegistryEntry

	// IsDefaultProject reports whether this project is set as the
	// default project for the daemon. Surfaced in the Registry
	// section. Phase 1 always renders false; the wiring point is here
	// for a future default-project mechanism.
	IsDefaultProject bool

	// OpsRunner is the runner probe surfaced to the Operations section
	// at page-render time. Nil-tolerant: when nil the Operations section
	// renders an "unavailable" empty state (defensive; never actually
	// nil once Phase 3 is wired). The concrete *opsrunner.Runner
	// satisfies the data.OpsLookup interface, so this field is typed at
	// that boundary to keep deps.go free of opsrunner imports.
	OpsRunner data.OpsLookup
}

// RegistryEntry is the minimal registry shape the project page reads.
// Kept here (rather than importing internal/serve.ProjectEntry
// directly) so the projectpage tests can construct fixtures without
// dragging the whole serve package into the unit-test boundary.
//
// Server-side wiring adapts internal/serve.ProjectEntry into this
// shape via a helper in server.go.
type RegistryEntry struct {
	// Path is the registered absolute path.
	Path string
	// RegisteredAt is the registration timestamp from projects.json.
	RegisteredAt time.Time
}
