package data

import (
	"path/filepath"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

// readEventsBest reads the events.log under heroDir, returning newest
// first. Mirrors the Now-home helper of the same name. Filters: events
// older than `since` are dropped (zero `since` disables); `limit` caps
// the result (zero disables). All errors are swallowed to a nil slice
// so the Knowledge page degrades gracefully when the log is missing.
func readEventsBest(heroDir string, since time.Time, limit int) []feed.FeedEvent {
	if heroDir == "" {
		return nil
	}
	evts, err := feed.ReadEvents(filepath.Join(heroDir, "events.log"), feed.Filter{
		Since: since,
		Limit: limit,
	})
	if err != nil {
		return nil
	}
	return evts
}

// CountCorpusEventsLastWeek counts capture/decision/convention events
// newer than 7 days. Used by the "new this week" tile when the corpus
// filesystem walk doesn't surface enough recent files (events.log is
// the canonical source for new captures).
func CountCorpusEventsLastWeek(heroDir string) int {
	return countCorpusEventsSince(heroDir, 7*24*time.Hour)
}

// countCorpusEventsSince counts capture/decision/convention events newer
// than the given duration. Used by the "new this week" tile.
func countCorpusEventsSince(heroDir string, since time.Duration) int {
	if heroDir == "" {
		return 0
	}
	events := readEventsBest(heroDir, time.Now().Add(-since), 0)
	count := 0
	for _, e := range events {
		switch e.Type {
		case "capture.created", "decision.created", "convention.created",
			"learning.captured", "knowledge.created":
			count++
		}
	}
	return count
}
