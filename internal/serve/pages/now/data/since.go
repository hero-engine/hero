package data

import (
	"fmt"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/spec"
)

// SinceInputs is the per-request input bundle for the
// "Since-you-last-looked" callout. LastLookedAt is the timestamp we
// diff against — read from the hero_last_looked cookie or, if absent,
// 24 hours ago.
type SinceInputs struct {
	HeroDir      string
	LastLookedAt time.Time
}

// SinceCallout is the section payload. Empty when nothing changed
// since LastLookedAt — the template renders nothing in that case.
type SinceCallout struct {
	Headline     string
	Summary      string
	SpecsTouched int
	Decisions    int
	Handoffs     int
	Show         bool
	Since        time.Time
}

// LoadSince computes a short "since you last looked" diff. We count
// spec status transitions, decisions captured, and peer handoffs in
// the window between LastLookedAt and now. Counts of zero collapse to
// Show=false so the callout stays out of the way.
func LoadSince(in SinceInputs) SinceCallout {
	if in.HeroDir == "" {
		return SinceCallout{}
	}
	since := in.LastLookedAt
	if since.IsZero() {
		since = time.Now().Add(-24 * time.Hour)
	}
	specs := countMovedSince(in.HeroDir, since)
	decisions := countEventsSince(in.HeroDir, since, []string{"decision_made"})
	handoffs := countEventsSince(in.HeroDir, since, []string{
		"peer.handoff.received", "peer.handoff.accepted", "peer.handoff.sent",
	})
	if specs == 0 && decisions == 0 && handoffs == 0 {
		return SinceCallout{Since: since}
	}
	parts := make([]string, 0, 3)
	if specs > 0 {
		parts = append(parts, fmt.Sprintf("%d spec%s moved", specs, plural(specs)))
	}
	if decisions > 0 {
		parts = append(parts, fmt.Sprintf("%d decision%s captured", decisions, plural(decisions)))
	}
	if handoffs > 0 {
		parts = append(parts, fmt.Sprintf("%d peer handoff%s", handoffs, plural(handoffs)))
	}
	return SinceCallout{
		Headline:     "Since you last looked",
		Summary:      strings.Join(parts, " · "),
		SpecsTouched: specs,
		Decisions:    decisions,
		Handoffs:     handoffs,
		Since:        since,
		Show:         true,
	}
}

func countMovedSince(heroDir string, since time.Time) int {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, s := range specs {
		if s == nil {
			continue
		}
		if s.ModifiedAt.After(since) {
			count++
		}
	}
	return count
}

func countEventsSince(heroDir string, since time.Time, types []string) int {
	evts, err := feed.ReadEvents(heroDirEventsLog(heroDir), feed.Filter{Since: since})
	if err != nil {
		return 0
	}
	want := map[string]bool{}
	for _, t := range types {
		want[t] = true
	}
	count := 0
	for _, e := range evts {
		if want[e.Type] {
			count++
		}
	}
	return count
}

func heroDirEventsLog(heroDir string) string {
	if heroDir == "" {
		return ""
	}
	if strings.HasSuffix(heroDir, "/events.log") {
		return heroDir
	}
	if strings.HasSuffix(heroDir, "/") {
		return heroDir + "events.log"
	}
	return heroDir + "/events.log"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
