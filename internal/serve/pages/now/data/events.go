package data

import (
	"path/filepath"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

// readEventsBest reads the events.log under heroDir, returning newest
// first. Filters: events older than `since` are dropped (zero `since`
// disables); `limit` caps the result (zero disables). All errors are
// swallowed to a nil slice — the Now page degrades gracefully when the
// log is missing.
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
