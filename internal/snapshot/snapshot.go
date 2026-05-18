// Package snapshot renders a project-shape rollup — surfaces, stages,
// active initiatives, recent completions, next-up, risks — into
// .hero/SNAPSHOT.md and paired CLI / MCP / serve readers.
//
// The package is a peer of internal/projection/ and internal/handoff/:
// total-rewrite-on-event, deterministic, fast. The graph is the source
// of truth; the rendered markdown regenerates on every Stop hook.
//
// Surface modeling is inference-first: detect.go derives a candidate
// surface list from repo structure and package manifests; surfaces.go
// merges an optional .hero/surfaces.yaml override layer on top. The
// user authors nothing to get a working snapshot.
//
// Archives (.hero/snapshots/YYYY-MM-DD.md) are byte-for-byte the same
// rendered markdown plus an archive-provenance frontmatter block.
// They are strictly isolated from default discovery — see archive.go
// for the six containment invariants.
package snapshot

import (
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// ProjectorVersion records the rendering schema this package emits.
// Bump when the rendered shape changes in a way archive readers care
// about — the value is captured in archive frontmatter so future
// readers can detect mismatches.
const ProjectorVersion = 1

// Stage is one of six lifecycle states a surface can occupy. The
// derivation rules live in stage.go.
type Stage string

const (
	StageConcept     Stage = "concept"
	StageScaffolded  Stage = "scaffolded"
	StageBuilding    Stage = "building"
	StageShippingV1  Stage = "shipping-v1"
	StageShipped     Stage = "shipped"
	StageMaturing    Stage = "maturing"
	StageUnknown     Stage = "" // never rendered; sentinel
)

// AllStages returns the canonical stage progression for validation
// and rendering. Order is meaningful — earlier stages come first.
func AllStages() []Stage {
	return []Stage{
		StageConcept,
		StageScaffolded,
		StageBuilding,
		StageShippingV1,
		StageShipped,
		StageMaturing,
	}
}

// IsValidStage reports whether s is one of the canonical stages.
func IsValidStage(s Stage) bool {
	for _, valid := range AllStages() {
		if s == valid {
			return true
		}
	}
	return false
}

// Surface is one coherent shipping unit visible in the snapshot.
// Surfaces are inferred from repo structure (see detect.go) and
// optionally overridden by .hero/surfaces.yaml.
type Surface struct {
	ID             string   // canonical id, e.g. "core", "serve", "domains/engineering"
	Name           string   // human label, falls back to ID
	Intent         string   // freeform one-line description; empty when unset
	Paths          []string // path globs / dirs declared or inferred for this surface
	Stage          Stage    // derived or pinned
	StagePinned    bool     // true when the override layer pinned the stage
	Owner          string   // freeform owner string from override; empty when unset
	Confidence     float64  // 0.0–1.0 inference confidence; 1.0 = explicit
	Signals        []string // rationale strings — which detection signals fired
	ReleaseTargets []ReleaseTarget // declared release scopes (rare; usually empty)
	Source         string   // "inferred", "added", "override"
}

// ReleaseTarget is a named release scope declared on a surface or a
// spec. Resolution priority lives in release.go.
type ReleaseTarget struct {
	Name        string // e.g. "v1", "v1.0.0"
	Description string // optional, freeform
	ScopeTag    string // optional tag that gates membership
}

// SpecAssignment ties one spec to a surface, with the release target
// resolved through the priority chain.
type SpecAssignment struct {
	Spec          *spec.Spec
	SurfaceID     string // resolved surface id, or "(unassigned)" sentinel
	ReleaseTarget string // resolved release target name, or "" when none
	ReleaseSource string // "frontmatter", "initiative", "tracker", "git-tag", "" (none)
	Inferred      bool   // true when the surface was inferred (no frontmatter declaration)
}

// UnassignedSurfaceID is the sentinel id used for the (unassigned)
// row in the surfaces table. Specs with no resolvable surface land
// here.
const UnassignedSurfaceID = "(unassigned)"

// Initiative is the rollup view of one initiative spec, scoped to
// snapshot rendering.
type Initiative struct {
	Slug      string
	Title     string
	Status    spec.Status
	Surfaces  []string // sorted, deduped surface ids touched by child specs
	Total     int      // count of child specs
	Done      int      // count of completed child specs
	InFlight  []string // slugs of child specs currently delivering
	CompletedAt time.Time // when status flipped to completed; zero if still active
}

// Snapshot is the assembled project-shape view returned by Build().
// All fields are populated in one pass and Render() turns them into
// markdown / JSON / compact MCP form.
type Snapshot struct {
	ProjectName    string
	Mission        string // one-liner pulled from .hero/mission.md or frontmatter mission:
	GeneratedAt    time.Time
	GraphRev       string // optional content-hash of source nodes; empty when unavailable

	Surfaces       []Surface
	Assignments    []SpecAssignment
	Initiatives    []Initiative
	RecentlyDone   []RecentItem
	NextUp         []NextItem
	Blockers       []Blocker
	StaleInFlight  []StaleItem
	AgedBugs       []AgedBug
	UnassignedCount int

	// Health metadata for the footer.
	InferredCount       int
	OverrideAppliedCount int
	OverrideEditedAt    time.Time
	GenerationMillis    int64
	SourceNodes         int

	// HasReleaseSignal is true when at least one assignment resolved a
	// release target. When false, the surfaces table hides the "% to
	// next milestone" column for every row and a footnote is shown.
	HasReleaseSignal bool
}

// RecentItem is one entry under "Recently completed".
type RecentItem struct {
	SurfaceID string
	Slug      string
	Title     string
	CompletedAt time.Time
	Type      spec.Type
}

// NextItem is one entry under "Next up across surfaces".
type NextItem struct {
	SurfaceID string
	Slug      string
	Title     string
	Status    spec.Status
	Priority  string
	Horizon   spec.Horizon
}

// Blocker is one entry under "Open risks & blockers".
type Blocker struct {
	Slug         string
	Title        string
	WaitsOn      []string // slugs of unmet hard dependencies
}

// StaleItem is one in-flight spec with no recent commit activity.
type StaleItem struct {
	Slug         string
	Title        string
	StaleDays    int
}

// AgedBug is one bug aged past the configured threshold.
type AgedBug struct {
	Slug         string
	Title        string
	AgeDays      int
	Severity     string
}
