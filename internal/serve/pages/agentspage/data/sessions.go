package data

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// SessionsInputs is the per-request input bundle for the Sessions view.
// LiveSessions is dependency-injected so this package never has to
// import the team-mode JobQueue directly; the server wires whatever
// concrete snapshot source it has.
type SessionsInputs struct {
	ProjectRoot string
	HeroDir     string
	Edition     string
	UserName    string

	// LiveSessions returns the current snapshot of live sessions. Nil
	// is safe; the loader renders an empty list in that case.
	LiveSessions func() []SessionRow
}

// LoadSessions composes the /agents sessions-view payload. Returns
// Empty() shaped output when no sources are wired in — the templates
// render an empty state without nil checks.
func LoadSessions(in SessionsInputs) Sessions {
	out := Empty()

	live := callLive(in.LiveSessions)
	out.LiveCount = len(live)

	// Build the live-session blocks. The first prominent block is
	// reserved for the longest-running non-amber session; everything
	// awaiting approval renders amber; remaining blocks compact.
	prominentTaken := false
	for _, row := range live {
		variant := "compact"
		switch row.Status {
		case "awaiting_approval", "paused":
			variant = "amber"
		default:
			if !prominentTaken {
				variant = "prominent"
				prominentTaken = true
			}
		}
		out.Live = append(out.Live, buildBlock(row, variant))
	}

	// Approval rows and completed-today are wired up when richer data
	// sources land; v1 surfaces the live ledger plus empty lists so the
	// page renders cleanly.
	out.ApprovalsCount = 0
	out.CompletedCount = 0

	// Scheduled / automations preview totals reported as 0 until the
	// automations engine + scheduled_runs table land per the spec's
	// build order. The lists themselves remain empty.
	out.ScheduledTotal = 0
	out.AutomationTotal = 0

	// Hero subhead — composes the four-number subhead phrase.
	out.LiveLabel = fmt.Sprintf("%d live", out.LiveCount)
	out.TodayLabel = fmt.Sprintf("%d today", out.CompletedCount)
	out.SpendTodayValue = "$0.00"
	out.SpendTodayLabel = "$0.00 spent today"
	out.PendingLabel = fmt.Sprintf("%d awaiting your approval", out.ApprovalsCount)

	// Metric strip — populate from snapshot. Calls a single builder so
	// the strip stays consistent across the page and the SSE fragment.
	out.Metric = buildMetricStrip(live, out.SpendTodayValue, out.SpendTodayPct, out.ApprovalsCount)
	return out
}

// callLive invokes the injected snapshot function, nil-safe.
func callLive(fn func() []SessionRow) []SessionRow {
	if fn == nil {
		return nil
	}
	return fn()
}

// buildBlock projects a raw SessionRow into the render-ready block.
func buildBlock(row SessionRow, variant string) SessionBlock {
	agentClass := agentClassFor(row.Agent)
	initials := agentInitials(row.Agent)
	statusTag, statusClass := statusFor(row.Status)
	onVerb := onVerbFor(row.Command, variant)

	cost := "$0.00"
	if row.CostUSD > 0 {
		cost = fmt.Sprintf("$%.2f", row.CostUSD)
	}

	startedAt := prettyAge(row.StartedAt)

	block := SessionBlock{
		Variant:     variant,
		ID:          row.ID,
		Agent:       row.Agent,
		AgentClass:  agentClass,
		Initials:    initials,
		OnVerb:      onVerb,
		Spec:        row.Spec,
		SpecHref:    specHref(row.Spec),
		StatusTag:   statusTag,
		StatusClass: statusClass,
		StartedAt:   startedAt,
		Branch:      row.Branch,
		Cost:        cost,
		Model:       row.Model,
		ToolCalls:   row.ToolCalls,
	}

	if row.ProposalsPending > 0 {
		noun := "proposal"
		if row.ProposalsPending != 1 {
			noun = "proposals"
		}
		block.PendingNotice = fmt.Sprintf("%d %s pending", row.ProposalsPending, noun)
	}

	switch variant {
	case "amber":
		// Use a stable placeholder until the propose store joins the
		// per-session ledger; the spec calls out that the page renders
		// empty-state cleanly when no diff snippet is available yet.
		if row.ProposalsPending > 0 {
			block.Proposal = &ProposalPreview{
				Files: fmt.Sprintf("%d proposal pending — open the proposal queue to review", row.ProposalsPending),
			}
		}
		block.Actions = []SessionAction{
			{Label: "Review proposal", Href: proposalsHref(row.ID), Variant: "primary"},
			{Label: "Resume", Href: sessionHref(row.ID)},
			{Label: "Open transcript", Href: sessionHref(row.ID)},
			{Label: "Stop", Href: "#", Variant: "danger"},
		}
	case "compact":
		block.Actions = []SessionAction{
			{Label: "Open transcript", Href: sessionHref(row.ID), Variant: "primary"},
			{Label: "Interrupt", Href: "#", Variant: "danger"},
		}
		block.Transcript = []TranscriptLine{
			{HTML: template.HTML(`<span class="role">tool</span><span class="tool">…</span> live transcript will stream here via SSE`)},
		}
	default: // prominent
		actions := []SessionAction{
			{Label: "Open transcript", Href: sessionHref(row.ID), Variant: "primary"},
		}
		if row.Spec != "" {
			actions = append(actions, SessionAction{Label: "Open spec", Href: specHref(row.Spec)})
		}
		if row.ProposalsPending > 0 {
			actions = append(actions,
				SessionAction{Label: fmt.Sprintf("Approve all (%d)", row.ProposalsPending), Href: "#", Variant: "primary"},
				SessionAction{Label: "Reject all", Href: "#"})
		}
		actions = append(actions, SessionAction{Label: "Interrupt", Href: "#", Variant: "danger"})
		block.Actions = actions
		block.Transcript = []TranscriptLine{
			{HTML: template.HTML(`<span class="role assistant">assistant</span>Live transcript will stream into this panel via SSE once the per-session token topic lands.`)},
		}
	}
	return block
}

// statusFor maps an internal status string to the (label, css class)
// pair used by the session-status-tag pill.
func statusFor(s string) (string, string) {
	switch s {
	case "awaiting_approval", "paused":
		return "Awaiting your approval", "amber"
	case "live", "running", "":
		return "Live", "live"
	case "done":
		return "Done", "live"
	case "failed":
		return "Failed", "amber"
	default:
		return strings.ToUpper(s), "live"
	}
}

// onVerbFor pulls a one-word "delivering/diagnosing/paused on" verb
// from the command string. Falls back to "working on".
func onVerbFor(command, variant string) string {
	if variant == "amber" {
		return "paused on"
	}
	c := strings.ToLower(command)
	switch {
	case strings.Contains(c, "/diagnose"), strings.Contains(c, "diagnose "):
		return "diagnosing"
	case strings.Contains(c, "/deliver"), strings.Contains(c, "deliver "):
		return "delivering"
	case strings.Contains(c, "/design"), strings.Contains(c, "design "):
		return "designing"
	case strings.Contains(c, "/review"), strings.Contains(c, "review "):
		return "reviewing"
	default:
		return "working on"
	}
}

// agentClassFor returns the CSS modifier class for the agent avatar
// gradient. Unknown agents render with the default blue gradient (no
// class).
func agentClassFor(agent string) string {
	a := strings.ToLower(agent)
	switch {
	case strings.Contains(a, "opus"):
		return "opus"
	case strings.Contains(a, "sonnet"):
		return "sonnet"
	case strings.Contains(a, "engineer"):
		return "engineer"
	case strings.Contains(a, "debug"):
		return "debug"
	default:
		return ""
	}
}

// agentInitials returns the lowercase two-letter initials shown in the
// circular avatar.
func agentInitials(agent string) string {
	a := strings.ToLower(strings.TrimSpace(agent))
	if a == "" {
		return "??"
	}
	if i := strings.Index(a, "-"); i > 0 && i+2 <= len(a) {
		return string(a[0]) + string(a[i+1])
	}
	if len(a) >= 2 {
		return a[:2]
	}
	return a
}

// specHref returns the canonical /work/spec/<slug> URL for a spec slug.
// Empty input returns "#".
func specHref(slug string) string {
	if slug == "" {
		return "#"
	}
	return "/work/spec/" + slug
}

// sessionHref returns the canonical /agents/session/<id> URL.
func sessionHref(id string) string {
	if id == "" {
		return "#"
	}
	return "/agents/session/" + id
}

// proposalsHref returns the deep link to the session-scoped proposal
// queue. Empty input falls back to the top-level proposals page.
func proposalsHref(sessionID string) string {
	if sessionID == "" {
		return "/agents/proposals"
	}
	return "/agents/proposals?session=" + sessionID
}

// prettyAge returns a Linear-style relative time string like "12m" or
// "2h" or "3d". Zero time returns "just now".
func prettyAge(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// buildMetricStrip assembles the three-tab metric strip from the
// snapshot data. The strip is fully server-rendered: each pane's tile
// set is included in the DOM and the inline metric-tabs.js toggles the
// `.active` class on click.
func buildMetricStrip(live []SessionRow, spendTodayValue string, spendPct int, awaiting int) MetricStrip {
	nowTab := MetricTab{
		Slug:   "now",
		Label:  "Right now",
		Active: true,
		Tiles: []MetricTile{
			{Value: template.HTML(fmt.Sprintf(`<span class="live-dot" aria-hidden="true"></span>%d`, len(live))), Label: "live sessions", Footer: template.HTML(liveAgentsSummary(live))},
			{Value: template.HTML(fmt.Sprintf(`<span class="live-dot amber" aria-hidden="true"></span>%d`, awaiting)), Label: "awaiting your approval", Accent: "warn"},
			{Value: template.HTML("0"), Label: "queue depth", Footer: template.HTML("no jobs queued")},
			{Value: template.HTML(spendTodayValue), Label: "spend today", Footer: spendBarFooter(spendPct)},
		},
	}
	todayTab := MetricTab{
		Slug:  "today",
		Label: "Today",
		Tiles: []MetricTile{
			{Value: template.HTML("—"), Label: "sessions completed"},
			{Value: template.HTML("—"), Label: "autonomy today"},
			{Value: template.HTML("—"), Label: "top tool"},
			{Value: template.HTML("—"), Label: "total cost today"},
		},
	}
	healthTab := MetricTab{
		Slug:  "health",
		Label: "Health (7d)",
		Tiles: []MetricTile{
			{Value: template.HTML("—"), Label: "interrupt rate"},
			{Value: template.HTML("—"), Label: "approval rate"},
			{Value: template.HTML("—"), Label: "failure rate"},
			{Value: template.HTML("—"), Label: "cost per merged proposal"},
		},
	}
	return MetricStrip{Tabs: []MetricTab{nowTab, todayTab, healthTab}}
}

// liveAgentsSummary collapses the live snapshot into the short sub
// shown below the first tile, e.g. "opus · sonnet · engineer".
func liveAgentsSummary(live []SessionRow) string {
	if len(live) == 0 {
		return "no sessions running"
	}
	parts := make([]string, 0, len(live))
	for _, s := range live {
		parts = append(parts, shortAgentLabel(s.Agent))
	}
	return strings.Join(parts, " · ")
}

func shortAgentLabel(agent string) string {
	a := strings.ToLower(agent)
	switch {
	case strings.Contains(a, "opus"):
		return "opus"
	case strings.Contains(a, "sonnet"):
		return "sonnet"
	case strings.Contains(a, "engineer"):
		return "engineer"
	case strings.Contains(a, "debug"):
		return "debug"
	default:
		if agent == "" {
			return "agent"
		}
		return agent
	}
}

// spendBarFooter renders the thin progress bar shown under the spend
// tile. Returns an empty string if percentage is zero.
func spendBarFooter(pct int) template.HTML {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return template.HTML(fmt.Sprintf(
		`<div class="metric-progress"><div class="metric-progress-fill" style="width:%d%%"></div></div>`, pct))
}
