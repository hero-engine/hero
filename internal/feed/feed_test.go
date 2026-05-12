package feed

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempLog(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "events.log")
}

func TestAppendAndRead(t *testing.T) {
	logPath := tempLog(t)

	e1 := FeedEvent{
		Timestamp: time.Date(2026, 4, 22, 14, 0, 0, 0, time.UTC),
		Type:      "spec_created",
		Agent:     "opencode/claude",
		Slug:      "csv-export",
		Message:   "Created spec",
	}
	e2 := FeedEvent{
		Timestamp: time.Date(2026, 4, 22, 15, 0, 0, 0, time.UTC),
		Type:      "decision_made",
		Agent:     "cursor/claude",
		Slug:      "csv-export",
		Message:   "Chose streaming",
	}

	if err := AppendEvent(logPath, e1); err != nil {
		t.Fatal(err)
	}
	if err := AppendEvent(logPath, e2); err != nil {
		t.Fatal(err)
	}

	events, err := ReadEvents(logPath, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	// Should be newest-first
	if events[0].Type != "decision_made" {
		t.Errorf("expected newest first, got %s", events[0].Type)
	}
}

func TestFilterBySince(t *testing.T) {
	logPath := tempLog(t)

	old := FeedEvent{
		Timestamp: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC),
		Type:      "spec_created",
		Agent:     "test",
		Message:   "Old event",
	}
	recent := FeedEvent{
		Timestamp: time.Now().UTC(),
		Type:      "decision_made",
		Agent:     "test",
		Message:   "Recent event",
	}

	AppendEvent(logPath, old)
	AppendEvent(logPath, recent)

	events, err := ReadEvents(logPath, Filter{Since: time.Now().UTC().Add(-1 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Message != "Recent event" {
		t.Errorf("expected recent event, got %s", events[0].Message)
	}
}

func TestFilterByType(t *testing.T) {
	logPath := tempLog(t)

	AppendEvent(logPath, FeedEvent{Timestamp: time.Now().UTC(), Type: "spec_created", Agent: "a", Message: "1"})
	AppendEvent(logPath, FeedEvent{Timestamp: time.Now().UTC(), Type: "decision_made", Agent: "a", Message: "2"})
	AppendEvent(logPath, FeedEvent{Timestamp: time.Now().UTC(), Type: "spec_created", Agent: "a", Message: "3"})

	events, err := ReadEvents(logPath, Filter{Type: "decision_made"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1, got %d", len(events))
	}
}

func TestFilterBySlug(t *testing.T) {
	logPath := tempLog(t)

	AppendEvent(logPath, FeedEvent{Timestamp: time.Now().UTC(), Type: "spec_created", Agent: "a", Slug: "csv", Message: "1"})
	AppendEvent(logPath, FeedEvent{Timestamp: time.Now().UTC(), Type: "spec_created", Agent: "a", Slug: "auth", Message: "2"})

	events, err := ReadEvents(logPath, Filter{Slug: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1, got %d", len(events))
	}
}

func TestLimit(t *testing.T) {
	logPath := tempLog(t)

	for i := 0; i < 10; i++ {
		AppendEvent(logPath, FeedEvent{Timestamp: time.Now().UTC(), Type: "spec_created", Agent: "a", Message: "x"})
	}

	events, err := ReadEvents(logPath, Filter{Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3, got %d", len(events))
	}
}

func TestBackwardCompatOldFormat(t *testing.T) {
	logPath := tempLog(t)

	// Write old-format claim event directly
	oldLine := `{"event":"claim","slug":"csv-export","agent":"human/chet-bellows","at":"2026-04-22T14:00:00Z"}` + "\n"
	os.WriteFile(logPath, []byte(oldLine), 0o644)

	events, err := ReadEvents(logPath, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1, got %d", len(events))
	}
	if events[0].Type != "spec_updated" {
		t.Errorf("expected spec_updated, got %s", events[0].Type)
	}
	if events[0].Slug != "csv-export" {
		t.Errorf("expected csv-export, got %s", events[0].Slug)
	}
}

func TestIsValidType(t *testing.T) {
	if !IsValidType("spec_created") {
		t.Error("spec_created should be valid")
	}
	if IsValidType("invalid_type") {
		t.Error("invalid_type should not be valid")
	}
}

func TestEmptyLog(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "nonexistent.log")
	events, err := ReadEvents(logPath, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0, got %d", len(events))
	}
}

func TestParseSince(t *testing.T) {
	// Duration
	ts, err := ParseSince("1h")
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(ts) > 2*time.Hour {
		t.Error("expected ~1h ago")
	}

	// RFC 3339
	ts, err = ParseSince("2026-04-22T14:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if ts.Year() != 2026 {
		t.Errorf("expected 2026, got %d", ts.Year())
	}

	// Date only
	ts, err = ParseSince("2026-04-22")
	if err != nil {
		t.Fatal(err)
	}
	if ts.Day() != 22 {
		t.Errorf("expected day 22, got %d", ts.Day())
	}

	// Invalid
	_, err = ParseSince("garbage")
	if err == nil {
		t.Error("expected error for garbage input")
	}
}
