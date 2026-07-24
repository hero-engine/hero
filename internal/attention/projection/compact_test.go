package projection

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
)

func TestCompactBoundsRowsAndRemovesBodiesWithoutMutatingSource(t *testing.T) {
	rows := make([]attention.AttentionRow, 22)
	for i := range rows {
		kind := "focus"
		summary := strings.Repeat("é", 121)
		if i == 0 {
			kind = "mail"
			summary = "secret mail body"
		}
		rows[i] = attention.AttentionRow{
			ID: kind + ":item", SourceKind: kind,
			Summary: summary, Body: "body must not escape",
		}
	}
	source := attention.AttentionSnapshot{
		SchemaVersion: 1, Revision: "revision",
		Counts: attention.AttentionCounts{Mail: 1, Focus: 21, Total: 22},
		Rows:   rows,
	}

	got := Compact(source, DefaultAwarenessLimit)

	if len(got.Rows) != DefaultAwarenessLimit {
		t.Fatalf("rows = %d, want %d", len(got.Rows), DefaultAwarenessLimit)
	}
	if got.Window == nil || got.Window.State != attention.AttentionStateCurrent ||
		got.Window.Limit != DefaultAwarenessLimit || got.Window.Returned != DefaultAwarenessLimit ||
		!got.Window.Truncated {
		t.Fatalf("window = %#v", got.Window)
	}
	if got.Counts.Total != 22 || got.Revision != "revision" {
		t.Fatalf("authority metadata changed: %#v", got)
	}
	if got.Rows[0].Body != "" || got.Rows[0].Summary != "" {
		t.Fatalf("mail content escaped compact view: %#v", got.Rows[0])
	}
	if len(got.Rows[1].Summary) > AwarenessSummaryBytes || !utf8.ValidString(got.Rows[1].Summary) {
		t.Fatalf("summary is not a valid bounded excerpt: %q", got.Rows[1].Summary)
	}
	if source.Rows[0].Body != "body must not escape" || source.Rows[0].Summary != "secret mail body" {
		t.Fatalf("source snapshot was mutated: %#v", source.Rows[0])
	}
}

func TestCompactMarksAuthoritativeEmptySnapshot(t *testing.T) {
	got := Compact(attention.AttentionSnapshot{
		SchemaVersion: 1,
		Counts:        attention.AttentionCounts{},
		Rows:          []attention.AttentionRow{},
	}, 1)
	if got.Window == nil || got.Window.State != attention.AttentionStateEmpty ||
		got.Window.Returned != 0 || got.Window.Truncated {
		t.Fatalf("window = %#v", got.Window)
	}
}
