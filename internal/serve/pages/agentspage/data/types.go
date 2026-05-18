// Package data composes the per-section payloads the Agents home
// renders. Each section has a small Load* function that takes a
// value-typed inputs struct and returns a single value-typed payload —
// stateless so SSE fragment endpoints can re-run a loader on demand.
//
// This package also exports the canonical SessionRow snapshot type that
// the Now home will consume (read-only) once its own data fetcher is
// rewired against the live ledger. SessionRow is the shape every page
// in the agents home agrees on for "one running or recent session".
package data

import (
	"html/template"
	"time"

	"github.com/hero-engine/hero/internal/serve/shell"
)

// MetricTile aliases shell.MetricTile so data fetchers can build tile
// payloads without dragging the shell type through every call site.
type MetricTile = shell.MetricTile

// ---- Live session ledger ------------------------------------------------

// SessionRow is the canonical snapshot of one agent session as exposed
// by the Agents home to itself and (read-only) to the Now home.
//
// Fields are deliberately stable across editions. Fields that don't
// apply to the active edition (e.g. CostUSD under `local` when cost
// isn't tracked) are left zero.
type SessionRow struct {
	ID           string
	Agent        string    // "claude-opus" | "claude-sonnet" | "engineer" | ...
	Spec         string    // spec slug currently being worked on (may be "")
	Command      string    // originating slash command + args, e.g. "/deliver per-feature-smoke-coverage"
	UserID       string    // submitter
	Branch       string    // git branch the session is on
	Model        string    // model identifier, e.g. "opus-4.7-1m"
	Status       string    // "live" | "awaiting_approval" | "paused" | "done" | "failed"
	StartedAt    time.Time
	LastActiveAt time.Time
	CostUSD      float64
	ToolCalls    int
	ProposalsPending int
}

// ---- Sessions section ---------------------------------------------------

// Sessions is the /agents (default sub-state) page payload.
type Sessions struct {
	// Live is the list of in-flight sessions rendered as separator-line
	// blocks. Each carries a Variant ("prominent" | "compact" | "amber")
	// that drives partial selection.
	Live []SessionBlock

	// Approvals is the flat awaiting-approval row list.
	Approvals []ApprovalRow

	// CompletedToday is the timeline row list of finished sessions.
	CompletedToday []CompletedRow

	// Counts power section headers + metric strip subheads.
	LiveCount      int
	ApprovalsCount int
	CompletedCount int

	// SchedulingPreview is the right-hand pair of compact lists in the
	// scheduled/automations preview split.
	NextScheduled    []CompactRow
	TopAutomations   []CompactRow
	ScheduledTotal   int
	AutomationTotal  int

	// Snapshot of metric strip tab data.
	Metric MetricStrip

	// HeroSub is the page-hero subhead bits the handler renders.
	LiveLabel       string
	TodayLabel      string
	SpendTodayLabel string
	PendingLabel    string
	SpendTodayPct   int     // for amber/red coloring decisions
	SpendTodayValue string  // pre-formatted "$4.27"
}

// SessionBlock is one rendered session in the live list.
type SessionBlock struct {
	Variant       string // "prominent" | "compact" | "amber"
	ID            string
	Agent         string
	AgentClass    string // "opus" | "sonnet" | "engineer" | "debug"
	Initials      string
	OnVerb        string // "delivering" | "diagnosing" | "paused on"
	Spec          string
	SpecHref      string
	StatusTag     string // "Live" | "Awaiting your approval"
	StatusClass   string // "live" | "amber"
	StartedAt     string
	Branch        string
	Cost          string
	Model         string
	ToolCalls     int
	PendingNotice string // "1 proposal pending" (amber meta) or ""
	Transcript    []TranscriptLine
	Proposal      *ProposalPreview // amber variant only
	Actions       []SessionAction
	Tools         []ToolPill
}

// TranscriptLine is one rendered line of the live transcript preview.
type TranscriptLine struct {
	HTML template.HTML
}

// ProposalPreview is the embedded amber proposal-preview panel used by
// the amber session-block variant.
type ProposalPreview struct {
	Files string // "2 files · <code>...</code> · +47 / −0"
	Hunks []DiffHunk
}

// DiffHunk is a single diff line in the proposal preview panel.
type DiffHunk struct {
	Class string // "diff-add" | "diff-rem" | "diff-ctx"
	Text  string
}

// SessionAction is one inline text-link verb on a session block.
type SessionAction struct {
	Label   string
	Href    string
	Variant string // "" | "primary" | "danger"
}

// ToolPill is one mono pill in the tool inventory strip.
type ToolPill struct {
	Name  string
	Count int
}

// ---- Approvals section --------------------------------------------------

// ApprovalRow is one row in the awaiting-approval flat list.
type ApprovalRow struct {
	Summary template.HTML
	Meta    template.HTML
	Actions []SessionAction
}

// ---- Completed timeline -------------------------------------------------

// CompletedRow is one row in the completed-today timeline.
type CompletedRow struct {
	Time     string
	IconKind string // "ok" | "warn" | "review" | ""
	HTML     template.HTML
	Duration string
	Cost     string
}

// ---- Scheduled / automations preview ------------------------------------

// CompactRow is one row in the scheduled/automations preview lists.
type CompactRow struct {
	Name    string
	Sub     template.HTML
	When    string
	WhenSub string
}

// ---- Metric strip -------------------------------------------------------

// MetricStrip is the agents-home metric strip. Three tabs: Right now /
// Today / Health (7d). Each carries 4 tiles.
type MetricStrip struct {
	Tabs []MetricTab
}

// MetricTab is one pane in the strip.
type MetricTab struct {
	Slug   string
	Label  string
	Active bool
	Tiles  []MetricTile
}

// ---- Empty data sentinel ------------------------------------------------

// Empty returns a zero-value Sessions ready to render the "no data"
// state without nil checks in templates.
func Empty() Sessions {
	return Sessions{
		Live:           []SessionBlock{},
		Approvals:      []ApprovalRow{},
		CompletedToday: []CompletedRow{},
		NextScheduled:  []CompactRow{},
		TopAutomations: []CompactRow{},
	}
}
