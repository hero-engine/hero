// Package data composes the per-section payloads the Now home renders.
// Each section has a small Load* function that takes a value-typed
// inputs struct and returns a single value-typed payload. The fetchers
// are deliberately stateless so they can run inside SSE fragment
// endpoints without sharing state with the main page handler.
package data

import (
	"html/template"
	"time"

	agentsdata "github.com/hero-engine/hero/internal/serve/pages/agentspage/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

// MetricTile aliases shell.MetricTile so data fetchers can build tile
// payloads without dragging the shell type through every call site, and
// so the page handler can pass them straight into the shared
// tabbed-metric-strip fragment with no conversion.
type MetricTile = shell.MetricTile

// SessionRow re-exports the canonical live-session snapshot shape from
// the Agents home so the Now home consumes the same source of truth.
// Picked Option B (direct import) over extracting a neutral shared
// package — agentspage/data/sessions.go's row shape is already neutral
// (no agentspage-specific shaping baked into the struct) and lifting it
// into a new package would churn ~3 files without adding clarity.
//
// TODO: if a third consumer appears, promote SessionRow to a neutral
// internal/serve/sessions package and have agentspage re-export it.
type SessionRow = agentsdata.SessionRow

// TryPrompt is one "Try: …" suggestion under the quick-launch input.
type TryPrompt struct {
	Label string
	Href  string
}

// ---- Inbox ---------------------------------------------------------------

// Inbox is the Needs-your-input section payload.
type Inbox struct {
	Rows  []InboxRow
	Total int
}

// InboxRow is one row in the inbox list. Kind drives the colored dot
// (proposal / handoff / review / import). Actions render as right-
// aligned links.
type InboxRow struct {
	Kind     string // proposal | handoff | review | import
	Summary  template.HTML
	Meta     template.HTML
	Actions  []InboxAction
}

// InboxAction is one inline action link on an inbox row.
type InboxAction struct {
	Label   string
	Href    string
	Variant string // "" | "danger" | "muted"
}

// ProposalRow is the dependency-injected representation of one pending
// proposal envelope, kept independent of the propose.* package so this
// package stays leaf-free of cross-cutting deps.
type ProposalRow struct {
	ProposalID  string
	SessionID   string
	SpecSlug    string
	Agent       string
	AnchorValue string
	EmittedAt   time.Time
	BatchID     string
	// FilesTouched is best-effort — empty when the envelope doesn't
	// carry a per-file breakdown.
	FilesTouched []string
}

// ---- Plate ---------------------------------------------------------------

// Plate is the On-your-plate section payload.
type Plate struct {
	Primary   *PlateCard
	Secondary *PlateCard
	Total     int
}

// PlateCard is one card in the plate grid (primary or secondary).
type PlateCard struct {
	Slug         string
	Title        string
	Status       string // "delivering" | "review" | "planning" | "completed" | ...
	StatusLabel  string
	Description  string
	Criteria     ProgressBar
	Coverage     ProgressBar
	Meta         []PlateMeta
	Actions      []PlateAction
	IsSecondary  bool
}

// ProgressBar is one of the dual mini-progress lines on a plate card.
// Variant controls fill color: "" → hero blue, "success" → green.
type ProgressBar struct {
	Label   string
	Pct     int // 0-100
	Value   string
	Variant string
}

// PlateMeta is one inline meta chip below the progress bars.
type PlateMeta struct {
	Label string
	Live  bool // animated dot
}

// PlateAction is one action link.
type PlateAction struct {
	Label string
	Href  string
	Mono  bool
}

// ---- Agents --------------------------------------------------------------

// Agents is the Your-agents section payload.
type Agents struct {
	Running          *RunningAgent
	RunningCount     int
	LastActivePretty string
	Today            TodayAgents
}

// RunningAgent is the Currently-running card payload. Nil when no
// session is in flight.
type RunningAgent struct {
	Name        string
	Initials    string
	SpecSlug    string
	SpecHref    string
	SessionAge  string // e.g. "4m"
	Transcript  []TranscriptLine
	Cost        string // e.g. "$0.32"
	ToolCalls   int
	Tokens      string // e.g. "47k"
	OpenHref    string
	InterruptHref string
}

// TranscriptLine is one line in the soft-grey transcript preview.
type TranscriptLine struct {
	Role     string // assistant | tool
	HTML     template.HTML
}

// TodayAgents is the right-hand 2×2 stat grid + 3-row session list.
type TodayAgents struct {
	SessionsDone     int
	ProposalsPending int
	Spend            string // e.g. "$0.74"
	SpendSpark       []int  // up to 8 bars, height units 0-15
	Autonomy         string // e.g. "71%"
	AutonomySpark    []int
	Sessions         []TodaySession
}

// TodaySession is one row in the bottom session list.
type TodaySession struct {
	Spec     string
	Subtitle string // e.g. "kickoff" | "proposal pending"
	Duration string // e.g. "28m"
	Status   string // "ok" | "warn"
}

// ---- Changes -------------------------------------------------------------

// Changes is the Since-you-were-here section payload.
type Changes struct {
	Rows    []ChangeRow
	Limited bool // true when emit pipeline unavailable and we're showing commits-only
}

// ChangeRow is one entry in the timeline feed.
type ChangeRow struct {
	Time string // relative-time chip
	Kind string // commit | spec | knowledge | drift | convention
	HTML template.HTML
}

// ---- Metrics -------------------------------------------------------------

// Metrics is the methodology-aware tabbed metric strip payload. Each
// field is a 4-tile slice; the page handler routes the right one into
// the first tab based on methodology resolution.
type Metrics struct {
	// FirstTabTiles is the tile set for the methodology-aware first
	// tab (sprint / cycle / week).
	FirstTabTiles []MetricTile

	// MyWeekTiles is the tile set for the "My week" tab.
	MyWeekTiles []MetricTile

	// ROITiles is the tile set for the "Hero ROI" tab.
	ROITiles []MetricTile
}

