package data

import (
	"sort"
)

// BlockedInputs is the per-request input bundle for the Blocked
// section.
type BlockedInputs struct {
	ProjectRoot string
	HeroDir     string
}

// LoadBlocked composes the Blocked section by scanning specs for the
// rough "can't move" heuristic (status == regressed or status string
// contains "block"). Returns an empty payload when nothing matches —
// the bottom section then collapses entirely.
func LoadBlocked(in BlockedInputs) Blocked {
	specs := loadSpecsBest(in.HeroDir)
	var rows []BlockedRow
	for _, s := range specs {
		if !isBlocked(s) {
			continue
		}
		rows = append(rows, BlockedRow{
			Slug:   s.Slug,
			Reason: "—",
			Dot:    "",
			Chips: []BlockedChip{
				{Label: "blocked"},
			},
			Actions: []BlockedAction{
				{Label: "Open", Href: "/work/spec/" + s.Slug},
				{Label: "Reassign", Href: "#", Muted: true},
			},
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Slug < rows[j].Slug })
	return Blocked{Rows: rows, Total: len(rows)}
}
