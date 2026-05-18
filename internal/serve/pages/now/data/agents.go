package data

import (
	"fmt"
	"strings"
	"time"
)

// AgentsInputs is the per-request input bundle for the Your-agents
// section. Live session tracking is handled by the chat / runner
// subsystems; this loader pulls the Currently-running card from
// LiveSessions when the ledger is wired and falls back to event-log
// synthesis for the Today list.
type AgentsInputs struct {
	ProjectRoot string
	HeroDir     string
	Edition     string

	// LiveSessions returns the canonical live-session snapshot. Nil is
	// safe — the loader renders the existing "no agent running" empty
	// state in that case. Shape matches the Agents home's ledger so
	// both pages surface the same source of truth.
	LiveSessions func() []SessionRow
}

// LoadAgents composes the agents section payload. Returns an Agents
// with a nil Running pointer when no session is active — the partial
// renders an empty state in that case. Edition-gated: under `local`
// the loader omits team-presence fields entirely.
func LoadAgents(in AgentsInputs) Agents {
	out := Agents{
		Today: TodayAgents{
			SpendSpark:    []int{},
			AutonomySpark: []int{},
			Sessions:      []TodaySession{},
		},
	}

	// In solo mode there is no live session store yet. Surface today's
	// recent delivery/diagnose events as the session-history list, and
	// leave Running == nil so the partial renders the empty state.
	events := readEventsBest(in.HeroDir, time.Now().Add(-24*time.Hour), 200)

	doneCount := 0
	for _, e := range events {
		switch e.Type {
		case "delivery_complete":
			doneCount++
			if len(out.Today.Sessions) < 3 {
				out.Today.Sessions = append(out.Today.Sessions, TodaySession{
					Spec:     e.Slug,
					Subtitle: "delivered",
					Duration: prettyAge(e.Timestamp),
					Status:   "ok",
				})
			}
		case "spec_updated", "spec_created":
			if len(out.Today.Sessions) < 3 {
				out.Today.Sessions = append(out.Today.Sessions, TodaySession{
					Spec:     e.Slug,
					Subtitle: shortenEventType(e.Type),
					Duration: prettyAge(e.Timestamp),
					Status:   "ok",
				})
			}
		}
	}
	out.Today.SessionsDone = doneCount
	out.Today.Spend = "—"
	out.Today.Autonomy = "—"

	if len(events) > 0 {
		out.LastActivePretty = prettyAgeChip(events[0].Timestamp)
	}

	// Live ledger — populates the Currently-running card from the same
	// snapshot the Agents home consumes. The primary card uses the
	// first row; RunningCount reflects the total so the page-hero
	// subhead reads "N agents running" accurately.
	if in.LiveSessions != nil {
		live := in.LiveSessions()
		running := filterRunning(live)
		out.RunningCount = len(running)
		if len(running) > 0 {
			out.Running = buildRunningAgent(running[0])
			// Treat the most recent live-session activity as a stronger
			// signal than the event-log scrape for "last active".
			if out.LastActivePretty == "" || !running[0].LastActiveAt.IsZero() {
				out.LastActivePretty = prettyAgeChip(latestActivity(running[0]))
			}
		}
	}
	return out
}

// filterRunning keeps only sessions in a state worth surfacing on the
// Currently-running card. "live" / "running" / "awaiting_approval" /
// "paused" all qualify; "done" / "failed" / "" are dropped so the
// empty state renders when nothing is in flight.
func filterRunning(in []SessionRow) []SessionRow {
	out := make([]SessionRow, 0, len(in))
	for _, s := range in {
		switch s.Status {
		case "live", "running", "awaiting_approval", "paused":
			out = append(out, s)
		}
	}
	return out
}

// buildRunningAgent projects a SessionRow into the Now-shaped
// RunningAgent card. Field assumptions match the existing template
// (templates/agents.html) — cost, tool-calls, tokens, spec link.
func buildRunningAgent(row SessionRow) *RunningAgent {
	name := row.Agent
	if name == "" {
		name = "agent"
	}
	cost := "$0.00"
	if row.CostUSD > 0 {
		cost = fmt.Sprintf("$%.2f", row.CostUSD)
	}
	specHref := "#"
	if row.Spec != "" {
		specHref = "/work/spec/" + row.Spec
	}
	openHref := "#"
	interruptHref := "#"
	if row.ID != "" {
		openHref = "/agents/session/" + row.ID
		interruptHref = "/agents/session/" + row.ID + "?action=interrupt"
	}
	return &RunningAgent{
		Name:          name,
		Initials:      runningInitials(name),
		SpecSlug:      row.Spec,
		SpecHref:      specHref,
		SessionAge:    prettyAge(row.StartedAt),
		Transcript:    nil, // Transcript streaming is owned by the Agents home; Now renders a static card.
		Cost:          cost,
		ToolCalls:     row.ToolCalls,
		Tokens:        "—",
		OpenHref:      openHref,
		InterruptHref: interruptHref,
	}
}

// runningInitials returns the two-letter avatar initials for an agent
// label. Mirrors the Agents home's agentInitials helper so both pages
// render the same chip.
func runningInitials(agent string) string {
	a := agent
	if a == "" {
		return "??"
	}
	if i := strings.IndexByte(a, '-'); i > 0 && i+2 <= len(a) {
		return string(a[0]) + string(a[i+1])
	}
	if len(a) >= 2 {
		return a[:2]
	}
	return a
}

// latestActivity returns the freshest timestamp on a session, falling
// back to StartedAt when LastActiveAt is zero.
func latestActivity(row SessionRow) time.Time {
	if !row.LastActiveAt.IsZero() {
		return row.LastActiveAt
	}
	return row.StartedAt
}

// shortenEventType maps spec lifecycle events to a one-word session-row
// subtitle suitable for the Today list.
func shortenEventType(t string) string {
	switch t {
	case "spec_created":
		return "created"
	case "spec_updated":
		return "updated"
	default:
		return t
	}
}

// prettyAgeChip is the short form used in the page-hero subhead ("Fri
// 4:12pm" style). Falls back to relative time when the timestamp is
// recent enough.
func prettyAgeChip(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	if d := time.Since(t); d < 24*time.Hour {
		return prettyAge(t)
	}
	return fmt.Sprintf("%s %s", t.Format("Mon"), t.Format("3:04pm"))
}
