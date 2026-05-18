package data

import "html/template"

// AutomationsInputs is the per-request input bundle for the
// automations tab. Rules is dependency-injected so this package never
// imports the (not-yet-landed) automations engine.
type AutomationsInputs struct {
	ProjectRoot string
	HeroDir     string

	// Rules returns the registered automation rules. Nil-safe.
	Rules func() []AutomationRow
}

// AutomationRow is one rule in the automations list.
type AutomationRow struct {
	ID         string
	Name       string
	Trigger    string // "tracker" | "webhook" | "schedule" | "file" | "feed"
	Filter     string
	Action     string
	Mode       string // "autopilot" | "gated"
	Enabled    bool
	LastFired  string
	RunsLast7d int
}

// Automations is the /agents/automations payload.
type Automations struct {
	Rows  []AutomationRow
	Total int
}

// LoadAutomations returns the automation-list payload.
func LoadAutomations(in AutomationsInputs) Automations {
	if in.Rules == nil {
		return Automations{Rows: []AutomationRow{}}
	}
	rows := in.Rules()
	return Automations{Rows: rows, Total: len(rows)}
}

// CompactAutomationsPreview projects an Automations snapshot to the
// compact-row list used in the sessions-view preview split.
func CompactAutomationsPreview(in Automations, limit int) []CompactRow {
	if limit <= 0 {
		limit = 3
	}
	out := make([]CompactRow, 0, limit)
	for i, r := range in.Rows {
		if i >= limit {
			break
		}
		sub := template.HTMLEscapeString(r.Trigger)
		if r.Filter != "" {
			sub += ": " + template.HTMLEscapeString(r.Filter)
		}
		if r.Mode != "" {
			sub += " · " + template.HTMLEscapeString(r.Mode)
		}
		when := r.LastFired
		whenSub := ""
		if r.RunsLast7d > 0 {
			whenSub = pluralRuns(r.RunsLast7d) + " / 7d"
		}
		out = append(out, CompactRow{
			Name:    r.Name,
			Sub:     template.HTML(sub),
			When:    when,
			WhenSub: whenSub,
		})
	}
	return out
}

func pluralRuns(n int) string {
	if n == 1 {
		return "1 run"
	}
	return itoa(n) + " runs"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
