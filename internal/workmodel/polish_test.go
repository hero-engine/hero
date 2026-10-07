package workmodel

import (
	"strings"
	"testing"
)

func polishFixture(t *testing.T) ([]Item, []PolishItem, []SuggestedItem) {
	t.Helper()
	c := newCorpus(t)
	seed(c)
	// recent (completed 2026-10-01, no audit → weak verify) gets an open bug
	// and a follow-up created after it shipped, plus an older related spec.
	c.write("planning/bugs/recentbug", fm("recentbug", "bug", "planning", "relations:\n  - target: recent\n    kind: related\n"))
	c.write("planning/features/followup", "---\ntitle: Followup\nslug: followup\ntype: feature\nstatus: planning\ncreated: 2026-10-01\nrelations:\n  - target: recent\n    kind: related\n---\n# F\n")
	c.write("planning/features/older", "---\ntitle: Older\nslug: older\ntype: feature\nstatus: planning\ncreated: 2026-08-01\nrelations:\n  - target: recent\n    kind: related\n---\n# O\n")
	specs, _ := c.build()
	items := Build(specs, Options{Now: testNow, Root: c.root})
	ApplyNext(items, specs)
	return items, Polish(items, specs), Suggested(items)
}

// polish-and-suggested AC-1: kinds on recently_done items only.
func TestPolishKinds(t *testing.T) {
	_, polish, _ := polishFixture(t)
	kinds := map[string]PolishItem{}
	for _, p := range polish {
		if p.Slug != "recent" {
			t.Errorf("polish on non-recent item %s (%s)", p.Slug, p.Kind)
		}
		kinds[p.Kind] = p
	}
	if p, ok := kinds[PolishWeakVerify]; !ok || p.Next == nil || p.Next.Command != "/verify recent" {
		t.Errorf("weak_verify = %+v", p)
	}
	if p, ok := kinds[PolishBugAgainstRecent]; !ok || !strings.Contains(p.Reason, "recentbug") || p.Next == nil || p.Next.Command != "/diagnose recentbug" {
		t.Errorf("bug_against_recent = %+v", p)
	}
	if p, ok := kinds[PolishOpenFollowups]; !ok || !strings.Contains(p.Reason, "followup") || strings.Contains(p.Reason, "older") {
		t.Errorf("open_followups = %+v (only specs created on/after completion count)", p)
	}
	if _, ok := kinds["rated_worse"]; ok {
		t.Error("rated_worse is reserved in v1")
	}
}

// polish-and-suggested AC-2: ready picks by priority, then stubs, at most 3,
// and Explore last only when the backlog is thin.
func TestSuggested(t *testing.T) {
	items, _, sugg := polishFixture(t)
	if len(sugg) == 0 || sugg[0].Slug == nil || *sugg[0].Slug != "ready" || sugg[0].Source != SuggestedSourceQueue {
		t.Fatalf("first pick should be the ready item: %+v", sugg)
	}
	if sugg[0].Reason != "ready to deliver, high priority" {
		t.Errorf("reason should follow the next step: %q", sugg[0].Reason)
	}
	picks := 0
	for _, s := range sugg {
		if s.Slug != nil {
			picks++
			if s.Next == nil || !strings.HasPrefix(s.Next.Command, "/") {
				t.Errorf("pick %s without a slash next step", *s.Slug)
			}
		}
	}
	if picks > 3 {
		t.Errorf("%d picks, want at most 3", picks)
	}
	active := 0
	for _, it := range items {
		if it.Lane == LaneReady || it.Lane == LaneInProgress {
			active++
		}
	}
	last := sugg[len(sugg)-1]
	explore := last.Slug == nil && last.Next != nil && last.Next.Command == "/discover"
	if (active < ThinBacklog) != explore {
		t.Errorf("active=%d explore=%v, want explore only when thin", active, explore)
	}

	// Not thin: no Explore item.
	busy := append([]Item{}, items...)
	for i := 0; i < ThinBacklog; i++ {
		busy = append(busy, Item{Slug: "busy" + string(rune('a'+i)), Lane: LaneInProgress})
	}
	for _, s := range Suggested(busy) {
		if s.Slug == nil {
			t.Error("Explore suggested while the backlog is not thin")
		}
	}
}
