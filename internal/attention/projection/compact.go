package projection

import (
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
)

const (
	DefaultAwarenessLimit = 8
	MaxAwarenessLimit     = 20
	AwarenessSummaryBytes = 240
)

// Compact returns a metadata-only, bounded copy of an authoritative snapshot.
// It never mutates the supplied snapshot.
func Compact(snapshot attention.AttentionSnapshot, limit int) attention.AttentionSnapshot {
	rows := snapshot.Rows
	if len(rows) > limit {
		rows = rows[:limit]
	}
	compactRows := make([]attention.AttentionRow, len(rows))
	for i, row := range rows {
		row.Body = ""
		if row.SourceKind == "mail" {
			row.Summary = ""
		} else {
			row.Summary = truncateUTF8(row.Summary, AwarenessSummaryBytes)
		}
		compactRows[i] = row
	}
	snapshot.Rows = compactRows
	state := attention.AttentionStateCurrent
	if snapshot.Counts.Total == 0 {
		state = attention.AttentionStateEmpty
	}
	snapshot.Window = &attention.AttentionWindow{
		State:     state,
		Limit:     limit,
		Returned:  len(compactRows),
		Truncated: snapshot.Counts.Total > len(compactRows),
	}
	return snapshot
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end]
}
