package data

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/spec"
)

// ShippedInputs is the per-request input bundle for the Recently
// shipped timeline.
type ShippedInputs struct {
	ProjectRoot string
	HeroDir     string
}

// LoadRecentlyShipped composes the timeline from .hero/events.log
// delivery_complete events in the last 7 days, falling back to
// completed-status specs (sorted by modified time) when the log is
// unavailable. Returns an empty payload when nothing matches.
func LoadRecentlyShipped(in ShippedInputs) RecentlyShipped {
	if rows := shippedFromEvents(in.HeroDir); len(rows) > 0 {
		return RecentlyShipped{Rows: rows}
	}
	return RecentlyShipped{Rows: shippedFromSpecs(in.HeroDir)}
}

func shippedFromEvents(heroDir string) []ShippedRow {
	if heroDir == "" {
		return nil
	}
	since := time.Now().Add(-7 * 24 * time.Hour)
	evts, err := feed.ReadEvents(filepath.Join(heroDir, "events.log"), feed.Filter{
		Since: since,
		Limit: 200,
	})
	if err != nil {
		return nil
	}
	var rows []ShippedRow
	for _, e := range evts {
		if e.Type != "delivery_complete" {
			continue
		}
		rows = append(rows, ShippedRow{
			Time:  prettyAge(e.Timestamp),
			Slug:  e.Slug,
			Title: e.Message,
			Actor: e.Agent,
		})
	}
	if len(rows) > 6 {
		rows = rows[:6]
	}
	return rows
}

func shippedFromSpecs(heroDir string) []ShippedRow {
	specs := loadSpecsBest(heroDir)
	var completed []*spec.Spec
	for _, s := range specs {
		if s.Status == spec.StatusCompleted {
			completed = append(completed, s)
		}
	}
	// Prefer the frontmatter-stamped completion time for both sort
	// order and the relative-age chip. Fall back to ModifiedAt (file
	// mtime) for legacy specs that pre-date the stamp.
	sort.SliceStable(completed, func(i, j int) bool {
		return shippedCompletionTime(completed[i]).After(shippedCompletionTime(completed[j]))
	})
	if len(completed) > 6 {
		completed = completed[:6]
	}
	rows := make([]ShippedRow, 0, len(completed))
	for _, s := range completed {
		rows = append(rows, ShippedRow{
			Time:  prettyAgeSince(shippedCompletionTime(s)),
			Slug:  s.Slug,
			Title: fallbackTitle(s),
		})
	}
	return rows
}

// shippedCompletionTime returns the canonical completion timestamp for
// a completed spec: the frontmatter-stamped `completed_at:` when
// present, otherwise the file modification time. Centralized so the
// fallback rule stays in lockstep between sort order and rendered age.
func shippedCompletionTime(s *spec.Spec) time.Time {
	if s == nil {
		return time.Time{}
	}
	if !s.CompletedAt.IsZero() {
		return s.CompletedAt
	}
	return s.ModifiedAt
}

// prettyAge mirrors the Now-home idiom for the events.feed input shape.
func prettyAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return prettyAgeSince(t)
}

// prettyAgeSince returns "2h", "1d", etc. for the relative-time chip.
func prettyAgeSince(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
