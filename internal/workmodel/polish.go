package workmodel

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// Polish kinds (read-contract-v1). `rated_worse` is reserved: Hero has no
// rating source yet.
const (
	PolishWeakVerify       = "weak_verify"
	PolishOpenFollowups    = "open_followups"
	PolishBugAgainstRecent = "bug_against_recent"
)

// SuggestedSourceQueue is the only suggested source in v1; snapshot, note
// and pulse are reserved.
const SuggestedSourceQueue = "queue"

// ThinBacklog is Hero's threshold: fewer ready + in-progress items than
// this means the backlog is thin.
const ThinBacklog = 3

// PolishItem is recently delivered work with a loose end.
type PolishItem struct {
	Slug   string    `json:"slug"`
	Title  string    `json:"title"`
	Kind   string    `json:"kind"`
	Reason string    `json:"reason"`
	Next   *NextStep `json:"next"`
}

// SuggestedItem is a deterministic "what next" pick.
type SuggestedItem struct {
	Slug   *string   `json:"slug"`
	Title  string    `json:"title"`
	Reason string    `json:"reason"`
	Source string    `json:"source"`
	Next   *NextStep `json:"next"`
}

// Polish lists loose ends on recently_done items, in item path order and
// then kind order. Items must already carry Next (ApplyNext).
func Polish(items []Item, specs []*spec.Spec) []PolishItem {
	c := NewCorpus(specs)
	byslug := map[string]Item{}
	for _, it := range items {
		byslug[it.Slug] = it
	}
	out := []PolishItem{}
	for _, it := range items {
		if it.Lane != LaneRecentlyDone {
			continue
		}
		done := c.Lookup(it.Slug)
		if done == nil {
			continue
		}
		if it.Verify != nil && it.Verify.State != VerifyPassed {
			out = append(out, PolishItem{Slug: it.Slug, Title: it.Title, Kind: PolishWeakVerify,
				Reason: "verify is " + strings.ReplaceAll(it.Verify.State, "_", " "), Next: it.Next})
		}
		var bugs, followups []Item
		for _, other := range items {
			if other.Slug == it.Slug || other.Lane == LaneRecentlyDone || other.CompletedAt != nil {
				continue
			}
			os := c.Lookup(other.Slug)
			if os == nil || os.IsFinished() || !related(done, os) {
				continue
			}
			if other.Type == "bug" {
				bugs = append(bugs, other)
			} else if doneAt := completionTime(done); !os.CreatedAt.IsZero() && !doneAt.IsZero() && !os.CreatedAt.Before(dayStart(doneAt)) {
				followups = append(followups, other)
			}
		}
		if len(followups) > 0 {
			out = append(out, PolishItem{Slug: it.Slug, Title: it.Title, Kind: PolishOpenFollowups,
				Reason: fmt.Sprintf("%d follow-up(s) open: %s", len(followups), slugList(followups)), Next: followups[0].Next})
		}
		if len(bugs) > 0 {
			out = append(out, PolishItem{Slug: it.Slug, Title: it.Title, Kind: PolishBugAgainstRecent,
				Reason: fmt.Sprintf("open bug(s) against it: %s", slugList(bugs)), Next: bugs[0].Next})
		}
	}
	return out
}

// related reports a relation edge in either direction.
func related(a, b *spec.Spec) bool {
	for _, r := range a.Relations {
		if r.Target == b.Slug {
			return true
		}
	}
	for _, r := range b.Relations {
		if r.Target == a.Slug {
			return true
		}
	}
	return false
}

// completionTime is when a finished spec was done: completed_at, else its
// file time — the same fallback Lane uses to place it in recently_done.
func completionTime(s *spec.Spec) time.Time {
	if !s.CompletedAt.IsZero() {
		return s.CompletedAt
	}
	return s.ModifiedAt
}

// dayStart is midnight UTC of t's day: `created:` is usually a bare date,
// so a follow-up created the same day as the completion counts.
func dayStart(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func slugList(items []Item) string {
	slugs := make([]string, len(items))
	for i, it := range items {
		slugs[i] = it.Slug
	}
	return strings.Join(slugs, ", ")
}

var priorityRank = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}

func rank(p *string) int {
	if p == nil {
		return 4
	}
	if r, ok := priorityRank[*p]; ok {
		return r
	}
	return 4
}

// Suggested returns up to three queue picks (ready items by priority, then
// undesigned stubs) and, when the backlog is thin, a trailing Explore item.
// Items must already carry Next (ApplyNext).
func Suggested(items []Item) []SuggestedItem {
	var ready, stubs []Item
	active := 0
	for _, it := range items {
		switch {
		case it.Lane == LaneReady:
			ready = append(ready, it)
			active++
		case it.Lane == LaneInProgress:
			active++
		case it.Lane == LaneNone && it.CompletedAt == nil && it.Next != nil &&
			(it.Next.Action == ActionDesign || it.Next.Action == ActionDiagnose):
			stubs = append(stubs, it)
		}
	}
	byPriority := func(list []Item) {
		sort.SliceStable(list, func(i, j int) bool {
			if ri, rj := rank(list[i].Priority), rank(list[j].Priority); ri != rj {
				return ri < rj
			}
			return list[i].Path < list[j].Path
		})
	}
	byPriority(ready)
	byPriority(stubs)

	out := []SuggestedItem{}
	add := func(it Item, reason string) {
		slug := it.Slug
		out = append(out, SuggestedItem{Slug: &slug, Title: it.Title, Reason: reason, Source: SuggestedSourceQueue, Next: it.Next})
	}
	for _, it := range ready {
		if len(out) == 3 {
			break
		}
		verb := "deliver"
		if it.Next != nil {
			verb = strings.ToLower(it.Next.Label)
		}
		add(it, "ready to "+verb+priorityPhrase(it.Priority))
	}
	for _, it := range stubs {
		if len(out) == 3 {
			break
		}
		add(it, "needs "+it.Next.Label+priorityPhrase(it.Priority))
	}
	if active < ThinBacklog {
		out = append(out, SuggestedItem{
			Title:  "Explore what's next",
			Reason: fmt.Sprintf("the backlog is thin (%d ready or in progress)", active),
			Source: SuggestedSourceQueue,
			Next: &NextStep{Action: ActionDiscover, Label: "Explore", Command: "/discover",
				Phase: Phase{Label: "Discovery", State: PhaseReady}, Enabled: true, Extras: []Extra{}},
		})
	}
	return out
}

func priorityPhrase(p *string) string {
	if p == nil {
		return ""
	}
	return ", " + *p + " priority"
}
