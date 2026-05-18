// Package data composes the per-section payloads the Work home renders.
// Each section has a small Load* function that takes a value-typed
// inputs struct and returns a single value-typed payload. The fetchers
// are deliberately stateless so they can run inside SSE fragment
// endpoints without sharing state with the main page handler.
package data

import (
	"html/template"

	"github.com/hero-engine/hero/internal/serve/shell"
)

// MetricTile aliases shell.MetricTile so data fetchers can build tile
// payloads without dragging the shell type through every call site, and
// so the page handler can pass them straight into the shared
// tabbed-metric-strip fragment with no conversion.
type MetricTile = shell.MetricTile

// ---- Page hero counts ----------------------------------------------------

// PageCounts is the workspace-state summary rendered in the page hero
// subhead.
type PageCounts struct {
	Total      int
	Delivering int
	Blocked    int
	// SprintState renders the trailing phrase, e.g. "Sprint 17 ends Wed"
	// or "No active sprint".
	SprintState string
}

// ---- Roadmap (Horizons centerpiece) --------------------------------------

// Roadmap is the Horizons centerpiece payload — three columns of cards.
type Roadmap struct {
	Now   RoadmapColumn
	Next  RoadmapColumn
	Later RoadmapColumn
	// BlockedCount feeds the view-toolbar "Blocked (n)" badge so the
	// roadmap and blocked sections stay in sync without a second pass.
	BlockedCount int
}

// RoadmapColumn is one of the three horizon columns.
type RoadmapColumn struct {
	Label string // "Now" | "Next" | "Later"
	Count int
	Pulse bool // true on the Now column
	Cards []SpecCard
}

// SpecCard is one card in a roadmap column. Initiative cards differ
// only by Kind, TypeChip styling, and the optional Children list.
type SpecCard struct {
	Slug        string
	Title       string
	TypeKey     string // "feature" | "bug" | "initiative" | "decision"
	TypeLabel   string // "Feature" | "Bug" | "Initiative" | "Decision"
	StatusKey   string // "delivering" | "planning" | "review" | "blocked" | "done"
	StatusLabel string
	Owner       CardOwner
	Methodology string // e.g. "sprint" — empty when unset
	// Bars are rendered only when the spec is delivering or in-review.
	Bars []ProgressBar
	// Signals render right of the bars when present.
	Signals []SpecSignal
	// QuietNote replaces the bars on backlog/planning cards where
	// progress is not yet meaningful.
	QuietNote string
	// Initiative-only fields:
	IsInitiative bool
	Children     []ChildRow
}

// CardOwner is the avatar+name shown in the meta row. Unclaimed=true
// renders the dashed placeholder.
type CardOwner struct {
	Initials  string
	Name      string
	Unclaimed bool
}

// ProgressBar is one of the dual mini-progress lines on a card.
// Variant controls fill color: "" → hero blue, "success" → green.
type ProgressBar struct {
	Label   string // "Criteria" | "Contract"
	Pct     int    // 0-100
	Value   string // e.g. "10 / 14" or "88%"
	Variant string
}

// SpecSignal is one signal chip on a card. Kind drives the styling
// class (drift / drift-major / ci-pass / ci-fail / agent / "" for the
// neutral chip used for proposal counts and PR refs).
type SpecSignal struct {
	Kind  string
	Label string
	// Agent kind chips render a pulsing live-dot before the label.
	Live bool
}

// ChildRow is one entry in an initiative's mini progress list.
type ChildRow struct {
	StatusKey string
	Slug      string
	Progress  string // e.g. "12 / 12"
}

// ---- Blocked -------------------------------------------------------------

// Blocked is the bottom-section payload. Total reflects the unclamped
// count so the section header reads "N specs can't move" even when
// Rows is capped.
type Blocked struct {
	Rows  []BlockedRow
	Total int
}

// BlockedRow is one row in the blocked list.
type BlockedRow struct {
	Slug    string
	Reason  string // post-em-dash one-line reason
	Dot     string // "" (red) | "warn" (amber)
	Chips   []BlockedChip
	Age     string // e.g. "3d blocked"
	Actions []BlockedAction
}

// BlockedChip is the reason-chip + optional dependency phrase.
type BlockedChip struct {
	Label    string
	Variant  string // "" | "dep" | "decision"
	// AsLink renders the chip text as a mono link (used for the "depends
	// on <slug>" tail). When set, the chip falls back to plain inline
	// text rather than a styled box.
	AsLink bool
	Href   string
}

// BlockedAction is one inline action link on a blocked row.
type BlockedAction struct {
	Label string
	Href  string
	Muted bool
}

// ---- Recently shipped ---------------------------------------------------

// RecentlyShipped is the bottom timeline payload.
type RecentlyShipped struct {
	Rows []ShippedRow
}

// ShippedRow is one entry in the timeline.
type ShippedRow struct {
	Time  string // "2h", "1d"
	Slug  string
	Title string
	Actor string
	HTML  template.HTML // optional pre-composed body (overrides Title/Actor when set)
}

// ---- Metrics -------------------------------------------------------------

// Metrics is the Work-home metric strip payload. Three tile sets:
// This sprint (default), Throughput, Quality.
type Metrics struct {
	SprintTiles     []MetricTile
	ThroughputTiles []MetricTile
	QualityTiles    []MetricTile
}
