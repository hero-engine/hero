package data

import (
	"fmt"
	"time"
)

// AgentsInputs is the per-request input bundle for the Your-agents
// section. Live session tracking is handled by the chat / runner
// subsystems; today this loader synthesizes the Currently-running card
// from event-log data when no live session store is wired in.
type AgentsInputs struct {
	ProjectRoot string
	HeroDir     string
	Edition     string
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

	// RunningCount is intentionally 0 until a live session store is
	// wired — the page-hero subhead reflects that with "no agent
	// running".
	out.RunningCount = 0
	return out
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
