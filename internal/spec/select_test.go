package spec

import (
	"testing"
	"time"
)

func makeSpec(slug, title string, t Type, st Status, opts ...func(*Spec)) *Spec {
	s := &Spec{
		Slug:       slug,
		Title:      title,
		Type:       t,
		Status:     st,
		Sections:   map[string]string{},
		ModifiedAt: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
		CreatedAt:  time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func withHorizon(h Horizon) func(*Spec) { return func(s *Spec) { s.Horizon = h } }
func withTags(tags ...string) func(*Spec) {
	return func(s *Spec) { s.Tags = tags }
}
func withPinned() func(*Spec)              { return func(s *Spec) { s.Pinned = true } }
func withModified(t time.Time) func(*Spec) { return func(s *Spec) { s.ModifiedAt = t } }
func withDependsOn(targets ...string) func(*Spec) {
	return func(s *Spec) {
		for _, t := range targets {
			s.Relations = append(s.Relations, Relation{Target: t, Kind: "depends-on"})
		}
	}
}
func withClaimedBy(user string) func(*Spec) { return func(s *Spec) { s.ClaimedBy = user } }

func slugs(specs []*Spec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Slug
	}
	return out
}

func TestSelectorReadyExcludesUnmetDeps(t *testing.T) {
	all := []*Spec{
		makeSpec("foo", "Foo", TypeFeature, StatusDelivering),
		makeSpec("bar", "Bar", TypeFeature, StatusPlanning,
			withDependsOn("foo")),
		makeSpec("baz", "Baz", TypeFeature, StatusPlanning,
			withDependsOn("foo", "bar")),
		makeSpec("qux", "Qux", TypeFeature, StatusPlanning),
	}

	got := Selector{
		Filter: Filter{Ready: true, ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)

	want := []string{"foo", "qux"}
	if diff := slugDiff(slugs(got), want); diff != "" {
		t.Errorf("ready set: %s", diff)
	}

	// Now mark foo completed — bar should become ready, baz still
	// blocked on bar.
	all[0].Status = StatusCompleted
	got = Selector{
		Filter: Filter{Ready: true, ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)
	want = []string{"bar", "qux"}
	if diff := slugDiff(slugs(got), want); diff != "" {
		t.Errorf("ready after foo completed: %s", diff)
	}
}

func TestSelectorReadyExcludesKnowledge(t *testing.T) {
	// Knowledge entries (notes, contexts, conventions, explainers, …)
	// carry no delivery lifecycle, so the queue must not surface them
	// in its actionable / kickoff-advisory output even when they're in
	// an open status. Work specs remain unaffected.
	all := []*Spec{
		makeSpec("ship-it", "Ship It", TypeFeature, StatusDelivering),
		makeSpec("plan-it", "Plan It", TypeBug, StatusPlanning),
		makeSpec("buddy-model-architecture", "Buddy Model", TypeNote, StatusActive),
		makeSpec("architecture-overview", "Arch Overview", TypeContext, StatusActive),
		makeSpec("explainer-entry", "Explainer", TypeExplainer, StatusActive),
		makeSpec("naming-convention", "Naming", TypeConvention, StatusActive),
	}

	got := Selector{
		Filter: Filter{Ready: true, ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)

	want := []string{"plan-it", "ship-it"}
	if diff := slugDiff(slugs(got), want); diff != "" {
		t.Errorf("ready set: %s", diff)
	}
}

func TestSelectorBlockedComplementsReady(t *testing.T) {
	all := []*Spec{
		makeSpec("foo", "Foo", TypeFeature, StatusDelivering),
		makeSpec("bar", "Bar", TypeFeature, StatusPlanning,
			withDependsOn("foo")),
	}

	blocked := Selector{
		Filter: Filter{Blocked: true, ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)

	if got := slugs(blocked); len(got) != 1 || got[0] != "bar" {
		t.Errorf("blocked = %v, want [bar]", got)
	}
}

func TestPrioritySortPinsFirst(t *testing.T) {
	all := []*Spec{
		makeSpec("a-delivering", "A", TypeFeature, StatusDelivering),
		makeSpec("z-planning-pinned", "Z", TypeFeature, StatusPlanning, withPinned()),
		makeSpec("b-planning", "B", TypeFeature, StatusPlanning),
	}

	got := Selector{
		Filter: Filter{ExcludeClosedDefault: true},
		Sort:   SortPriority,
	}.Apply(all)

	want := []string{"z-planning-pinned", "a-delivering", "b-planning"}
	if diff := slugDiff(slugs(got), want); diff != "" {
		t.Errorf("priority sort: %s", diff)
	}
}

func TestPrioritySortStatusBeatsHorizon(t *testing.T) {
	all := []*Spec{
		makeSpec("planning-now", "PN", TypeFeature, StatusPlanning, withHorizon(HorizonNow)),
		makeSpec("delivering-someday", "DS", TypeFeature, StatusDelivering, withHorizon(HorizonSomeday)),
	}
	got := Selector{
		Filter: Filter{ExcludeClosedDefault: true},
		Sort:   SortPriority,
	}.Apply(all)
	if got[0].Slug != "delivering-someday" {
		t.Errorf("delivering should rank ahead of planning regardless of horizon — got %s", got[0].Slug)
	}
}

func TestSortByPriorityMatchesSelector(t *testing.T) {
	all := []*Spec{
		makeSpec("planning-next", "PN", TypeFeature, StatusPlanning, withHorizon(HorizonNext)),
		makeSpec("planning-now", "P0", TypeFeature, StatusPlanning, withHorizon(HorizonNow)),
		makeSpec("planning-pinned", "PP", TypeFeature, StatusPlanning, withHorizon(HorizonParking), withPinned()),
	}
	direct := append([]*Spec(nil), all...)
	SortByPriority(direct)
	selected := Selector{
		Filter: Filter{ExcludeClosedDefault: true},
		Sort:   SortPriority,
	}.Apply(all)
	if diff := slugDiff(slugs(direct), slugs(selected)); diff != "" {
		t.Errorf("SortByPriority drifted from Selector: %s", diff)
	}
}

func TestExcludeClosedDefault(t *testing.T) {
	all := []*Spec{
		makeSpec("done", "Done", TypeFeature, StatusCompleted),
		makeSpec("doing", "Doing", TypeFeature, StatusDelivering),
	}
	got := Selector{
		Filter: Filter{ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)
	if len(got) != 1 || got[0].Slug != "doing" {
		t.Errorf("ExcludeClosedDefault should drop completed: got %v", slugs(got))
	}

	// Explicit Statuses override the default exclusion.
	got = Selector{
		Filter: Filter{Statuses: []Status{StatusCompleted}, ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)
	if len(got) != 1 || got[0].Slug != "done" {
		t.Errorf("explicit Statuses should override exclusion: got %v", slugs(got))
	}
}

func TestFilterByType(t *testing.T) {
	all := []*Spec{
		makeSpec("a", "A", TypeFeature, StatusPlanning),
		makeSpec("b", "B", TypeBug, StatusPlanning),
		makeSpec("c", "C", TypeConvention, StatusActive),
	}
	got := Selector{
		Filter: Filter{Types: []Type{TypeFeature, TypeBug}, ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)
	if want := []string{"a", "b"}; slugDiff(slugs(got), want) != "" {
		t.Errorf("types filter: got %v want %v", slugs(got), want)
	}
}

func TestFilterByTagsRequiresAll(t *testing.T) {
	all := []*Spec{
		makeSpec("a", "A", TypeFeature, StatusPlanning, withTags("export", "users")),
		makeSpec("b", "B", TypeFeature, StatusPlanning, withTags("export")),
		makeSpec("c", "C", TypeFeature, StatusPlanning, withTags("users")),
	}
	got := Selector{
		Filter: Filter{Tags: []string{"export", "users"}, ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)
	if want := []string{"a"}; slugDiff(slugs(got), want) != "" {
		t.Errorf("tags AND filter: got %v want %v", slugs(got), want)
	}
}

func TestFilterStaleDays(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -45)
	recent := now.AddDate(0, 0, -5)
	all := []*Spec{
		makeSpec("old", "Old", TypeFeature, StatusPlanning, withModified(old)),
		makeSpec("new", "New", TypeFeature, StatusPlanning, withModified(recent)),
	}
	got := Selector{
		Filter: Filter{StaleDays: 30, ExcludeClosedDefault: true},
		Sort:   SortAlpha,
		Now:    now,
	}.Apply(all)
	if len(got) != 1 || got[0].Slug != "old" {
		t.Errorf("stale 30: got %v want [old]", slugs(got))
	}
}

func TestFilterMineUser(t *testing.T) {
	all := []*Spec{
		makeSpec("a", "A", TypeFeature, StatusPlanning, withClaimedBy("alice")),
		makeSpec("b", "B", TypeFeature, StatusPlanning, withClaimedBy("bob")),
		makeSpec("c", "C", TypeFeature, StatusPlanning),
	}
	got := Selector{
		Filter: Filter{MineUser: "alice", ExcludeClosedDefault: true},
		Sort:   SortAlpha,
	}.Apply(all)
	if len(got) != 1 || got[0].Slug != "a" {
		t.Errorf("mine alice: got %v", slugs(got))
	}
}

func TestSelectorLimit(t *testing.T) {
	all := []*Spec{
		makeSpec("a", "A", TypeFeature, StatusPlanning),
		makeSpec("b", "B", TypeFeature, StatusPlanning),
		makeSpec("c", "C", TypeFeature, StatusPlanning),
	}
	got := Selector{
		Filter: Filter{ExcludeClosedDefault: true},
		Sort:   SortAlpha,
		Limit:  2,
	}.Apply(all)
	if len(got) != 2 {
		t.Errorf("limit 2: got %d", len(got))
	}
}

// slugDiff returns "" if got and want match in order, otherwise a
// human-readable description of the mismatch.
func slugDiff(got, want []string) string {
	if len(got) != len(want) {
		return "len mismatch: got " + sliceStr(got) + " want " + sliceStr(want)
	}
	for i := range got {
		if got[i] != want[i] {
			return "got " + sliceStr(got) + " want " + sliceStr(want)
		}
	}
	return ""
}

func sliceStr(xs []string) string {
	out := "["
	for i, x := range xs {
		if i > 0 {
			out += " "
		}
		out += x
	}
	return out + "]"
}
