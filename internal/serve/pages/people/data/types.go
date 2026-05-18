// Package data composes the per-section payloads the People & ROI home
// renders. Each section has a small Load* function returning a value
// struct so SSE fragment endpoints can re-render any section without
// sharing state with the main handler.
package data

import (
	"html/template"
	"time"

	"github.com/hero-engine/hero/internal/serve/shell"
)

// MetricTile aliases shell.MetricTile so data fetchers can build tiles
// without importing the shell type through every call site.
type MetricTile = shell.MetricTile

// ---- Pulse (default landing) --------------------------------------------

// Pulse is the team-pulse section payload — presence pill, active card
// grid, and recent feed rows.
type Pulse struct {
	RightNow      template.HTML
	Cards         []PresenceCard
	Feed          []FeedRow
	EmptyInLocal  bool
}

// PresenceCard is one card in the presence + claims grid.
type PresenceCard struct {
	Initials    string
	Name        string
	ActiveSpec  string
	SessionAge  string
	AgentBadge  bool
	AwaitsYou   bool
}

// FeedRow is one row of the recent activity feed.
type FeedRow struct {
	Time   string
	Actor  string
	HTML   template.HTML
}

// ---- ROI Overview (the headline) ----------------------------------------

// ROIOverview is the canonical Money/Throughput/Quality computation
// passed to the overview template. Each tile is pre-rendered using
// metrics-package output.
type ROIOverview struct {
	WindowLabel       string
	SubheadHTML       template.HTML
	MoneyTiles        []MetricTile
	ThroughputTiles   []MetricTile
	QualityTiles      []MetricTile
	TimeSpent         TimeSpent
	Savings           []SavingsRow
	TrendChips        []TrendChip
	Contributors      []ContributorRow
	WhatChanged       []WhatChangedRow
	MethodologyText   template.HTML
}

// TimeSpent is the stacked-bar "how time was spent" breakdown. Three
// segments sum to 100.
type TimeSpent struct {
	AgentAutonomousPct int
	AgentReviewPct     int
	HumanPct           int
	Caption            template.HTML
}

// SavingsRow is one row in the Where-the-savings-came-from list.
type SavingsRow struct {
	Color      string
	Label      string
	Pct        int
	Hours      string
	Dollars    string
}

// TrendChip is one chip in the 12-week-trend metric switcher.
type TrendChip struct {
	Slug  string
	Label string
	Active bool
}

// ContributorRow is one row in the Top-contributors table.
type ContributorRow struct {
	Initials      string
	AvatarClass   string // e.g. "av-h1" | "av-a1"
	Name          string
	BadgeAgent    bool
	SpecsTouched  int
	AutonomyPct   int
	HoursSaved    string
	HoursBarPct   int // 0-100
	DollarsSaved  string
}

// WhatChangedRow is one row in the What-changed-this-period stack.
type WhatChangedRow struct {
	Variant string // "good" | "warn"
	HTML    template.HTML
	When    string
}

// ---- Other sub-views (minimal placeholders for v1) ----------------------

// Generic is a fallback payload used by sub-views whose v1 surface is a
// title + empty-state body. The deeper layouts land in follow-up specs.
type Generic struct {
	Title   string
	Body    template.HTML
	When    time.Time
}
