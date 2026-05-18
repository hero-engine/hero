// Package data composes the per-section payloads the Knowledge home
// renders. Mirrors the now/data layout: one Load* function per section
// taking a value-typed inputs struct and returning a value-typed
// payload. All fetchers are stateless and best-effort — they degrade
// to empty results when the underlying data source is unavailable so
// the page never blanks.
package data

import (
	"html/template"
	"time"

	"github.com/hero-engine/hero/internal/serve/shell"
)

// MetricTile aliases shell.MetricTile so data fetchers can build tile
// payloads without dragging the shell type through every call site.
type MetricTile = shell.MetricTile

// ---- Corpus (the tiny orientation strip + entry index) ------------------

// Corpus is the orientation payload used by the metric strip and the
// browse card grid. The four numbers map to the mockup's quiet 4-tile
// strip (corpus entries, stale flags, reuse rate, new this week).
type Corpus struct {
	TotalEntries  int
	StaleFlags    int
	ReuseRatePct  int
	NewThisWeek   int
	Entries       []CorpusEntry
}

// CorpusEntry is one knowledge entry summarized for the browse grid.
type CorpusEntry struct {
	Kind        string // convention | decision | learning | note | rule | pattern
	Slug        string
	Title       string
	Description string
	Domain      string
	UpdatedAt   time.Time
	UpdatedPretty string
}

// ---- Why / Provenance ----------------------------------------------------

// Why is the provenance-chain payload. When traversal data isn't
// available the Available flag is false and the template falls through
// to the empty-state notice. The fields below mirror the mockup's
// detail-pane scaffolding so the template stays simple.
type Why struct {
	Available     bool
	Target        string
	TargetPretty  string
	ChainCrumbs   []string
	DetailType    string // decision | spec | convention | learning
	DetailStatus  string
	DetailSlug    string
	DetailDate    string
	DetailTitle   string
	Sections      []WhySection
	CitedBy       []string
	Cites         []string
	MadeBy        string
	MadeByDate    string
}

// WhySection is one heading + paragraphs block in the detail pane.
type WhySection struct {
	Heading string
	Body    template.HTML
}

// ---- Summary (plain-English narrator paragraph) -------------------------

// Summary is the plain-English narrator block. Available=false renders
// an empty-state hint.
type Summary struct {
	Available    bool
	Paragraph    template.HTML
	Author       string
	RegenPretty  string
}

// ---- Neighbors (You might also explore) ---------------------------------

// Neighbors is the suggested-neighbors payload.
type Neighbors struct {
	Rows []NeighborRow
}

// NeighborRow is one row in the neighbors list.
type NeighborRow struct {
	Kind        string // decision | convention | spec | learning
	Slug        string
	Description string
}

// ---- Staleness ---------------------------------------------------------

// Staleness is the worth-re-checking payload. Total drives the sub-nav
// badge. Available=false renders the empty-state notice.
type Staleness struct {
	Available bool
	Total     int
	Headline  template.HTML
	Detail    template.HTML
	Actions   []StaleAction
}

// StaleAction is one inline action on the staleness block.
type StaleAction struct {
	Label   string
	Href    string
	Variant string // "" | "muted"
}
