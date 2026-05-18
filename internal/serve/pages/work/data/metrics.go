package data

import (
	"html/template"
	"strconv"
)

// MetricsInputs is the per-request input bundle for the Work-home
// metric strip.
type MetricsInputs struct {
	ProjectRoot string
	HeroDir     string
	Counts      PageCounts
}

// LoadMetrics composes the three Work metric tabs. Today the tiles are
// real workspace counts where cheap (delivering / blocked) and quiet
// em-dash placeholders where the upstream subsystem (sprint planner,
// drift detector, CI ingest, throughput calc) does not yet feed the
// page. The strip never blanks the page.
func LoadMetrics(in MetricsInputs) Metrics {
	return Metrics{
		SprintTiles:     sprintTiles(in),
		ThroughputTiles: throughputTiles(),
		QualityTiles:    qualityTiles(),
	}
}

// sprintTiles is the "This sprint" pane. Without a sprint pipeline we
// surface the workspace state as a quiet sprintless variant.
func sprintTiles(in MetricsInputs) []MetricTile {
	return []MetricTile{
		{
			Value: template.HTML(strconv.Itoa(in.Counts.Delivering)),
			Label: "specs delivering",
			Footer: template.HTML(
				`<div class="metric-sub">no active sprint configured</div>`,
			),
		},
		{
			Value: template.HTML("—"),
			Label: "days remaining",
			Footer: template.HTML(
				`<div class="metric-sub">configure sprint in hero.json</div>`,
			),
		},
		{
			Value: template.HTML(strconv.Itoa(in.Counts.Blocked)),
			Label: "specs blocked",
			Accent: accentForBlocked(in.Counts.Blocked),
		},
		{
			Value: template.HTML(strconv.Itoa(in.Counts.Total)),
			Label: "specs total",
		},
	}
}

func throughputTiles() []MetricTile {
	return []MetricTile{
		{Value: template.HTML("—"), Label: "specs shipped this week"},
		{Value: template.HTML(`—<span class="unit">d</span>`), Label: "lead time"},
		{Value: template.HTML(`—<span class="unit">d</span>`), Label: "cycle time"},
		{Value: template.HTML(`—<span class="unit">%</span>`), Label: "flow efficiency"},
	}
}

func qualityTiles() []MetricTile {
	return []MetricTile{
		{Value: template.HTML("—"), Label: "drift detected"},
		{Value: template.HTML(`—<span class="unit">%</span>`), Label: "contract coverage avg"},
		{Value: template.HTML(`—<span class="unit">%</span>`), Label: "re-review rate"},
		{Value: template.HTML(`—<span class="unit">%</span>`), Label: "CI pass rate"},
	}
}

func accentForBlocked(n int) string {
	if n > 0 {
		return "warn"
	}
	return ""
}
