package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

func writeEvents(t *testing.T, heroDir string, events []feed.FeedEvent) {
	t.Helper()
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logPath := filepath.Join(heroDir, "events.log")
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create log: %v", err)
	}
	defer f.Close()
	for _, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		f.Write(b)
		f.Write([]byte("\n"))
	}
}

func TestLoadActivity_EmptyWindow(t *testing.T) {
	dir := t.TempDir()
	got := LoadActivity(ActivityInputs{HeroDir: dir, Window: WindowWeek})
	if !got.Empty {
		t.Errorf("expected Empty=true for missing log")
	}
	if len(got.Filters) != 4 {
		t.Errorf("expected 4 window filters, got %d", len(got.Filters))
	}
	// One filter must be Active.
	activeCount := 0
	for _, f := range got.Filters {
		if f.Active {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("expected exactly 1 active filter, got %d", activeCount)
	}
}

func TestLoadActivity_PopulatedFeed_ClassifiesKinds(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// Log is append-only oldest-first; ReadEvents reverses to newest-first.
	writeEvents(t, dir, []feed.FeedEvent{
		{Timestamp: now.Add(-4 * time.Hour), Type: "peer.call.completed", Slug: "alpha", Agent: "engineer"},
		{Timestamp: now.Add(-3 * time.Hour), Type: "knowledge.captured", Slug: "note1", Agent: "user"},
		{Timestamp: now.Add(-2 * time.Hour), Type: "decision_made", Slug: "alpha", Agent: "user"},
		{Timestamp: now.Add(-1 * time.Hour), Type: "delivery_complete", Slug: "alpha", Agent: "engineer"},
	})
	got := LoadActivity(ActivityInputs{HeroDir: dir, Window: WindowWeek})
	if got.Empty {
		t.Fatalf("expected non-empty feed")
	}
	wantKinds := []string{"spec", "decision", "knowledge", "handoff"}
	if len(got.Entries) != 4 {
		t.Fatalf("expected 4 entries, got %d: %+v", len(got.Entries), got.Entries)
	}
	for i, want := range wantKinds {
		if got.Entries[i].Kind != want {
			t.Errorf("entry %d kind = %q, want %q", i, got.Entries[i].Kind, want)
		}
	}
}

func TestLoadActivity_TodayWindowFiltersOlder(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	writeEvents(t, dir, []feed.FeedEvent{
		// older than today (appended first)
		{Timestamp: now.Add(-48 * time.Hour), Type: "delivery_complete", Slug: "old", Agent: "engineer"},
		// today
		{Timestamp: now.Add(-1 * time.Hour), Type: "delivery_complete", Slug: "fresh", Agent: "engineer"},
	})
	got := LoadActivity(ActivityInputs{HeroDir: dir, Window: WindowToday})
	if len(got.Entries) != 1 {
		t.Fatalf("expected 1 entry in today's window, got %d", len(got.Entries))
	}
	if !strings.Contains(string(got.Entries[0].Title), "fresh") {
		t.Errorf("expected fresh in today entry, got %q", got.Entries[0].Title)
	}
}

func TestLoadActivity_GroupsCommitsOnSameSpec(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// Three consecutive commits on the same spec — grouping needs
	// the entries to share kind+slug. The feed builds Link from the
	// slug, so we drive grouping through real spec-bearing events:
	// three delivery_start events on the same spec, listed newest-
	// first.
	writeEvents(t, dir, []feed.FeedEvent{
		{Timestamp: now.Add(-3 * time.Hour), Type: "delivery_start", Slug: "alpha", Agent: "engineer"},
		{Timestamp: now.Add(-2 * time.Hour), Type: "delivery_start", Slug: "alpha", Agent: "engineer"},
		{Timestamp: now.Add(-1 * time.Hour), Type: "delivery_start", Slug: "alpha", Agent: "engineer"},
	})
	got := LoadActivity(ActivityInputs{HeroDir: dir, Window: WindowWeek})
	if len(got.Entries) != 1 {
		t.Fatalf("expected 1 grouped entry, got %d: %+v", len(got.Entries), got.Entries)
	}
	if got.Entries[0].GroupCount != 3 {
		t.Errorf("GroupCount = %d, want 3", got.Entries[0].GroupCount)
	}
}

func TestLoadActivity_AggregateMergesAcrossProjects(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	now := time.Now()
	writeEvents(t, dir1, []feed.FeedEvent{
		{Timestamp: now.Add(-1 * time.Hour), Type: "delivery_complete", Slug: "a-spec", Agent: "engineer"},
	})
	writeEvents(t, dir2, []feed.FeedEvent{
		// 30 minutes ago is newer than 1 hour ago → beta first.
		{Timestamp: now.Add(-30 * time.Minute), Type: "delivery_complete", Slug: "b-spec", Agent: "engineer"},
	})
	got := LoadActivity(ActivityInputs{
		Aggregate: []ActivityProject{
			{Slug: "alpha", HeroDir: dir1},
			{Slug: "beta", HeroDir: dir2},
		},
		Window: WindowWeek,
	})
	if !got.Aggregate {
		t.Errorf("expected Aggregate=true")
	}
	if len(got.Entries) != 2 {
		t.Fatalf("expected 2 merged entries, got %d", len(got.Entries))
	}
	// Newest first → beta should come first.
	if got.Entries[0].Project != "beta" {
		t.Errorf("expected beta first (newest), got %q", got.Entries[0].Project)
	}
	if got.Entries[1].Project != "alpha" {
		t.Errorf("expected alpha second, got %q", got.Entries[1].Project)
	}
	// Aggregate links must be /p/all/-prefixed.
	for _, e := range got.Entries {
		if !strings.HasPrefix(e.Link, "/p/all/") {
			t.Errorf("expected /p/all/ prefix in aggregate link, got %q", e.Link)
		}
	}
}

func TestWindowFromString(t *testing.T) {
	cases := map[string]Window{
		"":      WindowWeek,
		"today": WindowToday,
		"week":  WindowWeek,
		"WEEK":  WindowWeek,
		"month": WindowMonth,
		"all":   WindowAll,
		"junk":  WindowWeek,
	}
	for in, want := range cases {
		if got := WindowFromString(in); got != want {
			t.Errorf("WindowFromString(%q) = %q, want %q", in, got, want)
		}
	}
}
