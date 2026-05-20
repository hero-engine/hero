package data

import (
	"html/template"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/spec"
)

// WeekInputs is the per-request input bundle for the "This week"
// rolling-window metric tiles.
type WeekInputs struct {
	ProjectRoot string
	HeroDir     string
	// Now is the wall-clock anchor for the "this week" window.
	// Defaults to time.Now() when zero so the data layer stays
	// testable without injecting a clock through every caller.
	Now time.Time
}

// LoadWeek returns the four rolling-window tiles per
// hero-serve-dashboard-redesign step 9:
//
//   - Touched this week (unique specs with any event in window)
//   - Shipped this week (delivery_complete events in window)
//   - Started this week (delivery_start events in window)
//   - Stale (>14d) (planning/delivering specs not touched in 14d)
//
// Each tile carries an Href query-param filter so the spec list below
// can scope to that window — matches the existing ?type= / ?age=
// filter pattern Roadmap already uses.
func LoadWeek(in WeekInputs) []MetricTile {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	weekAgo := now.Add(-7 * 24 * time.Hour)

	touched, shipped, started := weekCountsFromFeed(in.HeroDir, weekAgo)
	stale := staleCount(in.HeroDir, now)

	return []MetricTile{
		{
			Value: template.HTML(strconv.Itoa(touched)),
			Label: "touched this week",
			Href:  "/work?age=touched-7d",
		},
		{
			Value: template.HTML(strconv.Itoa(shipped)),
			Label: "shipped this week",
			Href:  "/work?age=shipped-7d",
		},
		{
			Value: template.HTML(strconv.Itoa(started)),
			Label: "started this week",
			Href:  "/work?age=started-7d",
		},
		{
			Value:  template.HTML(strconv.Itoa(stale)),
			Label:  "stale (>14d)",
			Href:   "/work?age=stale-14d",
			Accent: accentForStale(stale),
		},
	}
}

// weekCountsFromFeed walks the events log once and tallies touched /
// shipped / started across the window. "Touched" counts unique spec
// slugs to avoid an active spec dominating the number.
func weekCountsFromFeed(heroDir string, since time.Time) (touched, shipped, started int) {
	events, err := feed.ReadEvents(filepath.Join(heroDir, "events.log"), feed.Filter{Since: since})
	if err != nil {
		return 0, 0, 0
	}
	seen := make(map[string]struct{})
	for _, e := range events {
		if e.Slug != "" {
			if _, ok := seen[e.Slug]; !ok {
				seen[e.Slug] = struct{}{}
				touched++
			}
		}
		switch e.Type {
		case "delivery_complete":
			shipped++
		case "delivery_start":
			started++
		}
	}
	return touched, shipped, started
}

// staleCount counts specs in planning/delivering/in-review whose
// last touch is older than 14 days. The graph DB would give a more
// authoritative answer but spec file mtime is good enough for the
// tile — operators who want exact provenance click through.
func staleCount(heroDir string, now time.Time) int {
	specs := loadSpecsBest(heroDir)
	cutoff := now.Add(-14 * 24 * time.Hour)
	stale := 0
	for _, s := range specs {
		switch s.Status {
		case spec.StatusPlanning, spec.StatusDelivering, spec.StatusInReview:
			if s.ModifiedAt.Before(cutoff) {
				stale++
			}
		}
	}
	return stale
}

func accentForStale(n int) string {
	if n > 0 {
		return "warn"
	}
	return ""
}
