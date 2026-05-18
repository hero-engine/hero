package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// relation is a tiny test-only helper describing a frontmatter relation.
type relation struct{ Target, Kind string }

// writeSpec creates a minimal spec.md under heroDir/planning/<type>s/<slug>/
// with the given frontmatter fields. Returns the file path so callers can
// set mtime via os.Chtimes for last-touched assertions.
func writeSpec(t *testing.T, heroDir, slug, typ, status, horizon string, relations []relation) string {
	t.Helper()
	dir := filepath.Join(heroDir, "planning", typ+"s", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	body := "---\n"
	body += "title: " + slug + "\n"
	body += "type: " + typ + "\n"
	body += "status: " + status + "\n"
	if horizon != "" {
		body += "horizon: " + horizon + "\n"
	}
	if len(relations) > 0 {
		body += "relations:\n"
		for _, r := range relations {
			body += "  - target: " + r.Target + "\n"
			body += "    kind: " + r.Kind + "\n"
		}
	}
	body += "---\n\nbody.\n"
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestLoadRoadmap_ColumnCapAtTen(t *testing.T) {
	dir := t.TempDir()
	// Write 15 feature specs in horizon=now.
	for i := 0; i < 15; i++ {
		writeSpec(t, dir, "feat-"+itoaTest(i), "feature", "planning", "now", nil)
	}

	rm := LoadRoadmap(RoadmapInputs{HeroDir: dir})
	if rm.Now.Count != 15 {
		t.Errorf("Now.Count = %d, want 15 (total before cap)", rm.Now.Count)
	}
	if len(rm.Now.Cards) != 10 {
		t.Errorf("len(Now.Cards) = %d, want 10 (default cap)", len(rm.Now.Cards))
	}
	if !rm.Now.Capped {
		t.Errorf("Now.Capped = false, want true")
	}
	if rm.Now.ShowAllHref == "" {
		t.Errorf("Now.ShowAllHref empty")
	}
}

func TestLoadRoadmap_ShowAllExpands(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 15; i++ {
		writeSpec(t, dir, "feat-"+itoaTest(i), "feature", "planning", "now", nil)
	}
	rm := LoadRoadmap(RoadmapInputs{HeroDir: dir, ShowAll: true})
	if len(rm.Now.Cards) != 15 {
		t.Errorf("len(Now.Cards) under ShowAll = %d, want 15", len(rm.Now.Cards))
	}
	if rm.Now.Capped {
		t.Errorf("Now.Capped = true under ShowAll, want false")
	}
}

func TestLoadRoadmap_PaginationAtFifty(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 75; i++ {
		writeSpec(t, dir, "feat-"+itoaTest(i), "feature", "planning", "now", nil)
	}
	rm := LoadRoadmap(RoadmapInputs{HeroDir: dir, ShowAll: true, Page: 1})
	if len(rm.Now.Cards) != pageSize {
		t.Errorf("page 1 cards = %d, want %d", len(rm.Now.Cards), pageSize)
	}
	if rm.Now.PageInfo == nil {
		t.Fatalf("PageInfo nil on paginated column")
	}
	if rm.Now.PageInfo.Pages != 2 {
		t.Errorf("Pages = %d, want 2", rm.Now.PageInfo.Pages)
	}

	rm2 := LoadRoadmap(RoadmapInputs{HeroDir: dir, ShowAll: true, Page: 2})
	if len(rm2.Now.Cards) != 25 {
		t.Errorf("page 2 cards = %d, want 25 (remainder)", len(rm2.Now.Cards))
	}
}

func TestLoadRoadmap_InitiativeChildDedupe(t *testing.T) {
	dir := t.TempDir()
	// Parent initiative with two children — children should NOT render
	// at top level.
	writeSpec(t, dir, "epic", "initiative", "planning", "now", nil)
	writeSpec(t, dir, "child-1", "feature", "planning", "now", []relation{
		{Target: "epic", Kind: "parent"},
	})
	writeSpec(t, dir, "child-2", "feature", "planning", "now", []relation{
		{Target: "epic", Kind: "parent"},
	})
	writeSpec(t, dir, "loner", "feature", "planning", "now", nil)

	rm := LoadRoadmap(RoadmapInputs{HeroDir: dir})
	// Expect: 1 initiative card + 1 loner = 2 top-level cards.
	if rm.Now.Count != 2 {
		t.Errorf("Now.Count = %d, want 2 (epic + loner, children deduped)", rm.Now.Count)
	}
	// Verify child slugs are NOT present at top level.
	for _, card := range rm.Now.Cards {
		if card.Slug == "child-1" || card.Slug == "child-2" {
			t.Errorf("child %q rendered at top level (expected deduped)", card.Slug)
		}
	}
}

func TestLoadRoadmap_TypeFilter(t *testing.T) {
	dir := t.TempDir()
	writeSpec(t, dir, "f-1", "feature", "planning", "now", nil)
	writeSpec(t, dir, "b-1", "bug", "planning", "now", nil)
	writeSpec(t, dir, "f-2", "feature", "planning", "now", nil)

	rm := LoadRoadmap(RoadmapInputs{HeroDir: dir, TypeFilter: "bug"})
	if rm.Now.Count != 1 {
		t.Errorf("with TypeFilter=bug: Now.Count = %d, want 1", rm.Now.Count)
	}
	for _, c := range rm.Now.Cards {
		if c.TypeKey != "bug" {
			t.Errorf("TypeFilter=bug produced card type %q", c.TypeKey)
		}
	}
}

func TestLoadRoadmap_AgeFilter(t *testing.T) {
	dir := t.TempDir()
	freshPath := writeSpec(t, dir, "fresh", "feature", "planning", "now", nil)
	stalePath := writeSpec(t, dir, "stale", "feature", "planning", "now", nil)

	// Backdate stale by 30 days.
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(stalePath, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	_ = freshPath

	rm := LoadRoadmap(RoadmapInputs{HeroDir: dir, AgeFilter: "active-7d"})
	if rm.Now.Count != 1 {
		t.Errorf("active-7d: Now.Count = %d, want 1 (only fresh)", rm.Now.Count)
	}
	for _, c := range rm.Now.Cards {
		if c.Slug != "fresh" {
			t.Errorf("active-7d kept %q, want only fresh", c.Slug)
		}
	}
}

func TestLoadRoadmap_SortByLastTouched(t *testing.T) {
	dir := t.TempDir()
	oldP := writeSpec(t, dir, "old-spec", "feature", "planning", "now", nil)
	midP := writeSpec(t, dir, "mid-spec", "feature", "planning", "now", nil)
	newP := writeSpec(t, dir, "new-spec", "feature", "planning", "now", nil)

	now := time.Now()
	_ = os.Chtimes(oldP, now.Add(-10*24*time.Hour), now.Add(-10*24*time.Hour))
	_ = os.Chtimes(midP, now.Add(-2*24*time.Hour), now.Add(-2*24*time.Hour))
	_ = os.Chtimes(newP, now, now)

	rm := LoadRoadmap(RoadmapInputs{HeroDir: dir})
	if len(rm.Now.Cards) < 3 {
		t.Fatalf("expected 3 cards, got %d", len(rm.Now.Cards))
	}
	if rm.Now.Cards[0].Slug != "new-spec" {
		t.Errorf("first card = %q, want new-spec (newest first)", rm.Now.Cards[0].Slug)
	}
}

// itoaTest is a tiny local int→string helper to keep this file's
// imports minimal.
func itoaTest(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
