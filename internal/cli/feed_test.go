package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

func TestFeedCmdShowsRecentEvents(t *testing.T) {
	env := newTestEnv(t)
	writeFeedEvents(t, env)

	out, err := runCmd("feed")
	if err != nil {
		t.Fatalf("feed returned error: %v", err)
	}

	if !strings.Contains(out, "[decision") || !strings.Contains(out, "agent/beta") {
		t.Fatalf("expected decision event in output, got:\n%s", out)
	}
	if !strings.Contains(out, "restore-hero-feed-cli") {
		t.Fatalf("expected slug in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Restored feed reader") {
		t.Fatalf("expected newest event message in output, got:\n%s", out)
	}
}

func TestFeedCmdFiltersByTypeAndSlug(t *testing.T) {
	env := newTestEnv(t)
	writeFeedEvents(t, env)

	out, err := runCmd("feed", "--type", "decision_made", "--slug", "restore-hero-feed-cli")
	if err != nil {
		t.Fatalf("feed returned error: %v", err)
	}

	if !strings.Contains(out, "Restored feed reader") {
		t.Fatalf("expected matching decision event, got:\n%s", out)
	}
	if strings.Contains(out, "Unrelated blocker") {
		t.Fatalf("unexpected event leaked through filters:\n%s", out)
	}
}

func TestFeedCmdJSON(t *testing.T) {
	env := newTestEnv(t)
	writeFeedEvents(t, env)

	out, err := runCmd("feed", "--type", "decision_made", "--format", "json")
	if err != nil {
		t.Fatalf("feed returned error: %v", err)
	}

	var events []feed.FeedEvent
	if err := json.Unmarshal([]byte(out), &events); err != nil {
		t.Fatalf("expected JSON events, got %v:\n%s", err, out)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d: %#v", len(events), events)
	}
	if events[0].Message != "Restored feed reader" {
		t.Fatalf("unexpected event: %#v", events[0])
	}
}

func writeFeedEvents(t *testing.T, env *testEnv) {
	t.Helper()

	logPath := filepath.Join(env.heroDir, "events.log")
	events := []feed.FeedEvent{
		{
			Timestamp: time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
			Type:      "blocker_hit",
			Agent:     "agent/alpha",
			Slug:      "other-work",
			Message:   "Unrelated blocker",
		},
		{
			Timestamp: time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC),
			Type:      "decision_made",
			Agent:     "agent/beta",
			Slug:      "restore-hero-feed-cli",
			Message:   "Restored feed reader",
		},
	}
	for _, evt := range events {
		if err := feed.AppendEvent(logPath, evt); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
	}
}
