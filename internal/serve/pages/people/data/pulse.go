package data

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

// PulseInputs is the per-request input bundle for the team-pulse view.
type PulseInputs struct {
	ProjectRoot string
	HeroDir     string
	Edition     string
	UserName    string
}

// LoadPulse composes the team-pulse payload. Best-effort everywhere —
// missing event log produces empty rows, never an error.
func LoadPulse(in PulseInputs) Pulse {
	feedRows := loadFeedRows(in.HeroDir, 12)

	if strings.EqualFold(in.Edition, "local") {
		return Pulse{
			RightNow:     template.HTML(`<span>Solo workspace — no team to roll up. Activity feed below shows your own events.</span>`),
			EmptyInLocal: true,
			Feed:         feedRows,
		}
	}

	cards := []PresenceCard{}
	if in.UserName != "" {
		cards = append(cards, PresenceCard{
			Initials:   initials(in.UserName),
			Name:       in.UserName,
			ActiveSpec: "—",
			SessionAge: "—",
		})
	}

	return Pulse{
		RightNow: template.HTML(fmt.Sprintf(
			`<strong>%d humans active</strong> · <strong>0 agent sessions running</strong> · <strong>0 awaiting your approval</strong>`,
			len(cards),
		)),
		Cards: cards,
		Feed:  feedRows,
	}
}

// loadFeedRows reads .hero/events.log under heroDir and returns the
// newest `limit` rows as render-ready FeedRow values. Empty list when
// the log is missing or unreadable.
func loadFeedRows(heroDir string, limit int) []FeedRow {
	if heroDir == "" {
		return nil
	}
	evts, err := feed.ReadEvents(filepath.Join(heroDir, "events.log"), feed.Filter{Limit: limit})
	if err != nil || len(evts) == 0 {
		return nil
	}
	rows := make([]FeedRow, 0, len(evts))
	for _, e := range evts {
		actor := e.Agent
		if actor == "" {
			actor = "system"
		}
		msg := e.Message
		if msg == "" {
			msg = e.Type
		}
		rows = append(rows, FeedRow{
			Time:  relTime(e.Timestamp),
			Actor: actor,
			HTML: template.HTML(fmt.Sprintf(
				`<strong>%s</strong> <span class="people-feed-actor">%s</span> %s`,
				template.HTMLEscapeString(e.Type),
				template.HTMLEscapeString(actor),
				template.HTMLEscapeString(msg),
			)),
		})
	}
	return rows
}

func relTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func initials(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	fields := strings.Fields(name)
	out := ""
	for _, f := range fields {
		if len(f) == 0 {
			continue
		}
		out += strings.ToUpper(f)[:1]
		if len(out) >= 2 {
			break
		}
	}
	return out
}
