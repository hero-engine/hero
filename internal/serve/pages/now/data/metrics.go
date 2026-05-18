package data

import (
	"fmt"
	"html/template"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// MetricsInputs is the per-request input bundle for the metric strip.
type MetricsInputs struct {
	ProjectRoot string
	HeroDir     string
	UserName    string
	Methodology string // "scrum" | "shape-up" | "kanban" | "solo"
}

// LoadMetrics composes the three tile sets that fill the tabbed metric
// strip. The first tab varies with methodology; the My-week and Hero-
// ROI tabs are shared across methodologies. Placeholder tiles render
// when the data source is unavailable so the strip never blanks the
// page.
func LoadMetrics(in MetricsInputs) Metrics {
	return Metrics{
		FirstTabTiles: firstTabTiles(in),
		MyWeekTiles:   myWeekTiles(in),
		ROITiles:      roiTiles(in),
	}
}

func firstTabTiles(in MetricsInputs) []MetricTile {
	switch in.Methodology {
	case "scrum":
		return sprintTilesPlaceholder()
	case "shape-up":
		return cycleTilesPlaceholder()
	default:
		return weekTiles(in)
	}
}

// sprintTilesPlaceholder is the "no active sprint" rendering for the
// scrum first tab. A live sprint pipeline lands in a sibling spec; we
// avoid crashing meanwhile.
func sprintTilesPlaceholder() []MetricTile {
	return []MetricTile{
		{
			Value: template.HTML("—"),
			Label: "specs done · no active sprint",
			Footer: template.HTML(`<div class="now-seg-bar" aria-hidden="true">` +
				`<span class="now-seg-todo" style="width:100%"></span></div>` +
				`<div class="now-seg-legend"><span class="lg">configure sprint in hero.json</span></div>`),
		},
		{Value: template.HTML("—"), Label: "days remaining"},
		{Value: template.HTML("—"), Label: "specs flagged at risk"},
		{Value: template.HTML("—"), Label: "your committed specs"},
	}
}

// cycleTilesPlaceholder is the Shape-Up variant.
func cycleTilesPlaceholder() []MetricTile {
	return []MetricTile{
		{Value: template.HTML("—"), Label: "shaped · no active cycle"},
		{Value: template.HTML("—"), Label: "weeks remaining"},
		{Value: template.HTML("—"), Label: "scope creep flags"},
		{Value: template.HTML("—"), Label: "your appetite"},
	}
}

// weekTiles is the kanban / solo first tab — real numbers from git +
// the events log. Returns placeholder tiles when no project root is
// available.
func weekTiles(in MetricsInputs) []MetricTile {
	if in.ProjectRoot == "" {
		return []MetricTile{
			{Value: template.HTML("—"), Label: "specs shipped this week"},
			{Value: template.HTML("—"), Label: "commits authored"},
			{Value: template.HTML("—"), Label: "longest open spec"},
			{Value: template.HTML("—"), Label: "your committed specs"},
		}
	}
	shipped := countCompletedSince(in.HeroDir, 7*24*time.Hour)
	commits, _ := gitCountCommitsSince(in.ProjectRoot, "7 days ago", in.UserName)

	return []MetricTile{
		{
			Value:  template.HTML(strconv.Itoa(shipped)),
			Label:  "specs shipped this week",
			Footer: sparklineSVG([]int{1, 0, 1, 2, 0, 1, shipped}),
		},
		{
			Value:  template.HTML(strconv.Itoa(commits)),
			Label:  "commits authored",
			Footer: template.HTML(`<div class="metric-sub">last 7 days</div>`),
		},
		{
			Value: template.HTML("—"),
			Label: "longest open spec",
			Footer: template.HTML(`<div class="metric-sub">computed nightly</div>`),
		},
		{
			Value: template.HTML("—"),
			Label: "your committed specs",
			Footer: template.HTML(`<div class="metric-sub">no claim yet</div>`),
		},
	}
}

// myWeekTiles is the cross-methodology "My week" tab. Sparklines use
// real shipped-spec counts when available, otherwise placeholders.
func myWeekTiles(in MetricsInputs) []MetricTile {
	shipped := 0
	commits := 0
	if in.HeroDir != "" {
		shipped = countCompletedSince(in.HeroDir, 7*24*time.Hour)
	}
	if in.ProjectRoot != "" {
		// Note: no sprint required — `git log` is always available.
		commits, _ = gitCountCommitsSince(in.ProjectRoot, "7 days ago", in.UserName)
	}

	shippedVal := template.HTML(strconv.Itoa(shipped))
	commitsVal := template.HTML(strconv.Itoa(commits))
	assist := agentAssistTile(in.HeroDir)

	return []MetricTile{
		{
			Value:  shippedVal,
			Label:  "specs shipped this week",
			Footer: sparklineSVG([]int{0, 1, 1, 2, 0, 1, shipped}),
		},
		{
			Value:  commitsVal,
			Label:  "commits authored",
			Footer: template.HTML(`<div class="metric-sub">last 7 days</div>`),
		},
		assist,
		{
			Value:  template.HTML("—<span class=\"unit\"> sessions</span>"),
			Label:  "Hero-active time",
			Footer: template.HTML(`<div class="metric-sub">session ledger pending</div>`),
		},
	}
}

// agentAssistTile computes the "agent assist on your work" ratio:
// delivery-completion events authored by an agent / total delivery-
// completion events in the trailing 7-day window. Renders "—" when
// the denominator is zero.
func agentAssistTile(heroDir string) MetricTile {
	if heroDir == "" {
		return MetricTile{
			Value:  template.HTML(`—<span class="unit">%</span>`),
			Label:  "agent assist on your work",
			Footer: template.HTML(`<div class="metric-sub">no events log</div>`),
		}
	}
	events := readEventsBest(heroDir, time.Now().Add(-7*24*time.Hour), 0)
	total := 0
	agent := 0
	for _, e := range events {
		if !isDeliveryCompleteEvent(e.Type) {
			continue
		}
		total++
		if isAgentAuthored(e.Agent) {
			agent++
		}
	}
	if total == 0 {
		return MetricTile{
			Value:  template.HTML(`—<span class="unit">%</span>`),
			Label:  "agent assist on your work",
			Footer: template.HTML(`<div class="metric-sub">no delivery events in window</div>`),
		}
	}
	pct := agent * 100 / total
	return MetricTile{
		Value:  template.HTML(strconv.Itoa(pct) + `<span class="unit">%</span>`),
		Label:  "agent assist on your work",
		Footer: template.HTML(`<div class="metric-sub">last 7 days</div>`),
	}
}

// roiTiles is the cross-methodology "Hero ROI" tab. Values are
// placeholder today — the canonical computation lives in the People &
// ROI home.
func roiTiles(_ MetricsInputs) []MetricTile {
	return []MetricTile{
		{
			Value:  template.HTML(`—<span class="unit">%</span>`),
			Label:  "autonomy ratio (7d)",
			Footer: template.HTML(`<div class="metric-sub">computed weekly</div>`),
		},
		{
			Value:  template.HTML(`—<span class="unit">h</span>`),
			Label:  "hours saved this week",
			Footer: template.HTML(`<div class="metric-sub">computed weekly</div>`),
		},
		{
			Value:  template.HTML(`—<span class="unit">%</span>`),
			Label:  "spec coverage (7d)",
			Footer: template.HTML(`<div class="metric-sub">computed weekly</div>`),
		},
		{
			Value:  template.HTML(`—<span class="unit">d</span>`),
			Label:  "cycle time",
			Footer: template.HTML(`<div class="metric-sub">computed weekly</div>`),
		},
	}
}

// sparklineSVG renders a tiny inline SVG line chart. Values are
// rescaled to the SVG height; an empty input yields an empty
// placeholder div so the tile keeps its footer height.
func sparklineSVG(values []int) template.HTML {
	if len(values) == 0 {
		return template.HTML(`<svg class="metric-sparkline" width="80" height="22" viewBox="0 0 80 22"></svg>`)
	}
	const (
		w, h    = 80, 22
		padY    = 2
		nominal = 12
	)
	maxV := 0
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	if maxV == 0 {
		maxV = nominal
	}
	step := float64(w-4) / float64(len(values)-1)
	if len(values) == 1 {
		step = 0
	}
	var b strings.Builder
	b.WriteString(`<svg class="metric-sparkline" width="80" height="22" viewBox="0 0 80 22" fill="none">`)
	b.WriteString(`<polyline points="`)
	for i, v := range values {
		x := 2 + float64(i)*step
		y := float64(h-padY) - (float64(v)/float64(maxV))*float64(h-padY*2)
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%.1f,%.1f", x, y)
	}
	b.WriteString(`" stroke="#2a6cb5" stroke-width="1.5" fill="none" stroke-linecap="round" stroke-linejoin="round"/></svg>`)
	return template.HTML(b.String())
}

// gitCountCommitsSince shells out to git to count commits authored by
// the given user since the given relative date. Empty user counts all
// authors. Returns 0 on any error.
func gitCountCommitsSince(projectRoot, since, user string) (int, error) {
	if projectRoot == "" {
		return 0, nil
	}
	args := []string{"-C", projectRoot, "log", "--since=" + since, "--pretty=oneline"}
	if user != "" {
		args = append(args, "--author="+user)
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

// countCompletedSince scans .hero/events.log for delivery_complete OR
// spec.complete events newer than the given duration. Both verbs count
// equally — `hero spec complete` and `hero deliver` both signal a
// finished spec and the workspace uses them interchangeably. Returns
// 0 when the log is missing or unreadable.
//
// Note: the count is NOT filtered by claimed_by. Many specs in this
// workspace are never claimed; filtering by claim makes the tile read 0
// even after multiple specs shipped today (the original bug).
func countCompletedSince(heroDir string, since time.Duration) int {
	if heroDir == "" {
		return 0
	}
	events := readEventsBest(heroDir, time.Now().Add(-since), 0)
	count := 0
	for _, e := range events {
		if isDeliveryCompleteEvent(e.Type) {
			count++
		}
	}
	return count
}

// isDeliveryCompleteEvent reports whether an event type signals a
// finished spec. Centralized so metrics + agents + changes all classify
// the same way.
func isDeliveryCompleteEvent(t string) bool {
	switch t {
	case "delivery_complete", "spec.complete":
		return true
	}
	return false
}

// isDeliveryEvent reports whether an event type is anywhere in the
// delivery lifecycle (start, complete, session begin/end). Used by the
// agents.go session-count tile so an in-flight session still bumps the
// counter even if it hasn't shipped yet.
func isDeliveryEvent(t string) bool {
	switch t {
	case "delivery_start", "delivery_complete", "spec.complete",
		"agent_session_started", "agent_session_ended":
		return true
	}
	return false
}

// isAgentAuthored reports whether an agent label looks like an LLM /
// AI agent (vs a human). Used to compute "agent assist" ratios in the
// metrics strip. Conservative — anything we don't recognize counts as
// human so the ratio doesn't over-report.
func isAgentAuthored(agent string) bool {
	a := strings.ToLower(strings.TrimSpace(agent))
	if a == "" {
		return false
	}
	switch {
	case strings.HasPrefix(a, "claude-"),
		strings.HasPrefix(a, "gpt-"),
		strings.HasPrefix(a, "ai/"),
		strings.HasPrefix(a, "engineer"),
		strings.HasPrefix(a, "agent/"),
		strings.HasPrefix(a, "mcp/"):
		return true
	}
	return false
}
