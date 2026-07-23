package projection

import (
	"sort"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
)

var groupRank = map[string]int{"mail": 0, "focus": 1, "suggestion": 2}

func orderRows(rows []attention.AttentionRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if groupRank[a.Group] != groupRank[b.Group] {
			return groupRank[a.Group] < groupRank[b.Group]
		}
		at, aok := parseActivity(a.ActivityAt)
		bt, bok := parseActivity(b.ActivityAt)
		if aok != bok {
			return aok // missing activity sorts last
		}
		if aok && !at.Equal(bt) {
			if a.Group == "mail" {
				return at.Before(bt)
			}
			return at.After(bt)
		}
		return a.SourceID < b.SourceID
	})
}

func parseActivity(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	return t, err == nil
}
