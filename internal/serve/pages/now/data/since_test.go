package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

func TestLoadSince_EmptyHeroDir(t *testing.T) {
	got := LoadSince(SinceInputs{})
	if got.Show {
		t.Errorf("Show should be false for empty HeroDir, got %+v", got)
	}
}

func TestLoadSince_ZeroState_NoEvents(t *testing.T) {
	dir := t.TempDir()
	// Empty events log + no specs → Show=false.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got := LoadSince(SinceInputs{HeroDir: dir, LastLookedAt: time.Now().Add(-1 * time.Hour)})
	if got.Show {
		t.Errorf("expected Show=false on empty workspace, got %+v", got)
	}
}

func TestLoadSince_PopulatedCounts(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	lookback := now.Add(-6 * time.Hour)

	// One decision + two peer-handoff events in the window.
	writeEvents(t, dir, []feed.FeedEvent{
		{Timestamp: now.Add(-5 * time.Hour), Type: "decision_made", Slug: "alpha", Agent: "user"},
		{Timestamp: now.Add(-4 * time.Hour), Type: "peer.handoff.received", Slug: "beta", Agent: "user"},
		{Timestamp: now.Add(-3 * time.Hour), Type: "peer.handoff.accepted", Slug: "beta", Agent: "user"},
		// An older event that should NOT count.
		{Timestamp: now.Add(-48 * time.Hour), Type: "decision_made", Slug: "stale", Agent: "user"},
	})
	// Plus one freshly-touched spec on disk (its mtime defaults to now).
	specDir := filepath.Join(dir, "planning", "features", "fresh-spec")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	specPath := filepath.Join(specDir, "spec.md")
	body := "---\ntitle: fresh-spec\ntype: feature\nstatus: planning\n---\n\nbody.\n"
	if err := os.WriteFile(specPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	got := LoadSince(SinceInputs{HeroDir: dir, LastLookedAt: lookback})
	if !got.Show {
		t.Fatalf("expected Show=true when events present, got %+v", got)
	}
	if got.Decisions != 1 {
		t.Errorf("Decisions = %d, want 1", got.Decisions)
	}
	if got.Handoffs != 2 {
		t.Errorf("Handoffs = %d, want 2", got.Handoffs)
	}
	if got.SpecsTouched < 1 {
		t.Errorf("SpecsTouched = %d, want >= 1", got.SpecsTouched)
	}
	if got.Headline == "" {
		t.Errorf("Headline should be populated when Show=true")
	}
	if got.Summary == "" {
		t.Errorf("Summary should be populated when Show=true")
	}
}

func TestLoadSince_DefaultsTo24hWhenZero(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// One decision 6 hours ago — must count under the default 24h window.
	writeEvents(t, dir, []feed.FeedEvent{
		{Timestamp: now.Add(-6 * time.Hour), Type: "decision_made", Slug: "alpha", Agent: "user"},
	})
	got := LoadSince(SinceInputs{HeroDir: dir}) // zero LastLookedAt → 24h default
	if !got.Show {
		t.Fatalf("expected Show=true under default 24h window, got %+v", got)
	}
	if got.Decisions != 1 {
		t.Errorf("Decisions = %d, want 1", got.Decisions)
	}
	// And Since should be ~24h ago (within a small skew).
	if got.Since.IsZero() {
		t.Errorf("Since should be populated when LastLookedAt is zero")
	}
}
