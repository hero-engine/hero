package data

import "html/template"

// ScheduledInputs is the per-request input bundle for the scheduled
// tab. ScheduledTasks is dependency-injected so the data package never
// imports the cron tick loop directly.
type ScheduledInputs struct {
	ProjectRoot string
	HeroDir     string

	// Tasks returns the registered scheduled task descriptors. Nil-safe.
	Tasks func() []ScheduledRow
}

// ScheduledRow is one row in the scheduled list. Mirrors the v1 fields
// the spec calls out — name, cron expression, action command, mode,
// timing.
type ScheduledRow struct {
	ID         string
	Name       string
	CronExpr   string
	HumanCron  string
	Action     string
	Mode       string // "autopilot" | "gated"
	NextRun    string
	LastRun    string
	LastStatus string // "ok" | "warn" | "deferred" | "missed" | ""
	Deferred   bool
}

// Scheduled is the /agents/scheduled payload.
type Scheduled struct {
	Rows  []ScheduledRow
	Total int
}

// LoadScheduled returns the scheduled-list payload. Empty rows when no
// fetcher is wired in.
func LoadScheduled(in ScheduledInputs) Scheduled {
	if in.Tasks == nil {
		return Scheduled{Rows: []ScheduledRow{}}
	}
	rows := in.Tasks()
	return Scheduled{Rows: rows, Total: len(rows)}
}

// CompactScheduledPreview projects a Scheduled snapshot to the
// compact-row list used in the sessions-view preview split.
func CompactScheduledPreview(in Scheduled, limit int) []CompactRow {
	if limit <= 0 {
		limit = 3
	}
	out := make([]CompactRow, 0, limit)
	for i, r := range in.Rows {
		if i >= limit {
			break
		}
		out = append(out, CompactRow{
			Name:    r.Name,
			Sub:     template.HTML("<code>" + template.HTMLEscapeString(r.CronExpr) + "</code> · " + template.HTMLEscapeString(r.HumanCron)),
			When:    r.NextRun,
			WhenSub: r.LastRun,
		})
	}
	return out
}
