package data

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/serve/metrics"
)

// OverviewInputs is the per-request input bundle for the ROI Overview.
type OverviewInputs struct {
	HeroDir      string
	Edition      string
	WindowDays   int     // typically 28 — "Last 4 weeks"
	APISpendUSD  float64 // adapter-reported chat.cost spend; 0 disables Net/ROI tiles
	Coefficients metrics.Coefficients
}

// LoadOverview computes the canonical Money/Throughput/Quality view.
// Empty inputs render with em-dashes and zero counts — the page must
// render for any caller (including the kitchen-sink dev surface).
func LoadOverview(in OverviewInputs) ROIOverview {
	days := in.WindowDays
	if days <= 0 {
		days = 28
	}
	window := metrics.Last(time.Duration(days) * 24 * time.Hour)
	counts := metrics.LoadCounts(in.HeroDir, window)
	k := in.Coefficients
	if k == (metrics.Coefficients{}) {
		k = metrics.DefaultCoefficients()
	}

	hours := metrics.HoursSaved(counts, k)
	dollars := metrics.DollarsSaved(hours, k)
	net := metrics.NetValue(hours, in.APISpendUSD, k)
	roi := metrics.ROIMultiple(net, in.APISpendUSD)

	return ROIOverview{
		WindowLabel: fmt.Sprintf("Last %d days", days),
		SubheadHTML: buildSubhead(days, counts.SpecsDelivered, net, roi),
		MoneyTiles: []MetricTile{
			tile("Hours saved", metrics.FormatHours(hours), "vs unassisted baseline"),
			tile("Dollars saved", metrics.FormatDollars(dollars), fmt.Sprintf("≈%.0fh × $%.0f/hr loaded", hours, k.HourlyCost)),
			netValueTile(dollars, in.APISpendUSD, net),
			tile("ROI multiple", metrics.FormatROIMultiple(roi), "net value ÷ API spend"),
		},
		ThroughputTiles: []MetricTile{
			tile("Specs delivered", fmt.Sprintf("%d", counts.SpecsDelivered), "in window"),
			tile("Autonomy ratio 7d", "—", "without-edit merges ÷ all merges"),
			tile("Cycle time", "—", "median: claimed → completed"),
			tile("Time-to-spec", "—", "median: imported → designed"),
		},
		QualityTiles: []MetricTile{
			tile("Spec coverage 7d", "—", "merged commits linked to a spec"),
			tile("Re-review rate", "—", "completed → reopened"),
			tile("Knowledge reuse", "—", "agent sessions injecting ≥1 entry"),
			tile("Drift catches", "—", "spec/code mismatches surfaced"),
		},
		TimeSpent: TimeSpent{
			AgentAutonomousPct: 0,
			AgentReviewPct:     0,
			HumanPct:           100,
			Caption:            template.HTML("Agent autonomy ratio not yet computed — the proposal-lifecycle log lands in a sibling spec."),
		},
		Savings: []SavingsRow{},
		TrendChips: []TrendChip{
			{Slug: "net", Label: "Net value", Active: true},
			{Slug: "hours", Label: "Hours saved"},
			{Slug: "autonomy", Label: "Autonomy ratio"},
			{Slug: "specs", Label: "Specs delivered"},
		},
		Contributors: []ContributorRow{},
		WhatChanged:  []WhatChangedRow{},
		MethodologyText: template.HTML(fmt.Sprintf(
			`Numbers use configurable coefficients (<strong>loaded hourly cost: $%.0f</strong> · no-edit save: %.1fh · with-edit save: %.1fh · …). <a href="#">View methodology →</a> <a href="#">Tune coefficients →</a>`,
			k.HourlyCost, k.NoEdit, k.WithEdit,
		)),
	}
}

func tile(label, value, sub string) MetricTile {
	return MetricTile{
		Value:  template.HTML(template.HTMLEscapeString(value)),
		Label:  label,
		Footer: template.HTML(`<div class="metric-sub">` + template.HTMLEscapeString(sub) + `</div>`),
	}
}

// netValueTile renders the Net-value tile with its inline segmented bar
// legend. Falls back to a dash when the API spend is zero (the Net
// value still equals Dollars saved in that case per the spec).
func netValueTile(dollarsSaved, apiSpend, net float64) MetricTile {
	var sub string
	if apiSpend > 0 {
		sub = fmt.Sprintf("%s saved − %s Hero API spend", metrics.FormatDollars(dollarsSaved), metrics.FormatDollars(apiSpend))
	} else {
		sub = "adapter cost reporting not configured"
	}
	value := metrics.FormatDollars(net)

	var savedPct, spendPct float64
	if dollarsSaved+apiSpend > 0 {
		savedPct = (dollarsSaved / (dollarsSaved + apiSpend)) * 100
		spendPct = 100 - savedPct
	} else {
		savedPct = 100
	}
	footer := fmt.Sprintf(
		`<div class="metric-sub">%s</div>`+
			`<div class="metric-segbar" aria-hidden="true">`+
			`<span class="seg-saved" style="width:%.1f%%;"></span>`+
			`<span class="seg-spend" style="width:%.1f%%;"></span>`+
			`</div>`,
		template.HTMLEscapeString(sub), savedPct, spendPct,
	)
	_ = sub // already escaped into footer above
	return MetricTile{
		Value:  template.HTML(template.HTMLEscapeString(value)),
		Label:  "Net value",
		Footer: template.HTML(footer),
	}
}

func buildSubhead(days, specsDelivered int, net, roi float64) template.HTML {
	parts := []string{
		fmt.Sprintf("Last %d days", days),
		fmt.Sprintf("%d specs delivered", specsDelivered),
	}
	if net != 0 {
		parts = append(parts, fmt.Sprintf("%s net value", metrics.FormatDollars(net)))
	}
	if roi != 0 {
		parts = append(parts, metrics.FormatROIMultiple(roi)+" ROI")
	}
	return template.HTML(strings.Join(parts, " <span class=\"dot-sep\">·</span> "))
}
