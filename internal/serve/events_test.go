package serve

import (
	"testing"
	"time"
)

func TestEventBus_PubSub(t *testing.T) {
	bus := NewEventBus()

	id, ch := bus.Subscribe(10)
	defer bus.Unsubscribe(id)

	bus.Publish(Event{
		Type:    EventSpecCreated,
		Slug:    "my-feature",
		Path:    "/project/.hero/specs/my-feature/spec.md",
		Message: "spec created",
	})

	select {
	case ev := <-ch:
		if ev.Type != EventSpecCreated {
			t.Errorf("type = %q, want %q", ev.Type, EventSpecCreated)
		}
		if ev.Slug != "my-feature" {
			t.Errorf("slug = %q, want my-feature", ev.Slug)
		}
		if ev.Timestamp.IsZero() {
			t.Error("timestamp should be set automatically")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()

	id1, ch1 := bus.Subscribe(10)
	id2, ch2 := bus.Subscribe(10)
	defer bus.Unsubscribe(id1)
	defer bus.Unsubscribe(id2)

	bus.Publish(Event{Type: EventSpecModified, Slug: "test"})

	// Both should receive the event
	select {
	case <-ch1:
	case <-time.After(time.Second):
		t.Fatal("subscriber 1 timed out")
	}
	select {
	case <-ch2:
	case <-time.After(time.Second):
		t.Fatal("subscriber 2 timed out")
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus()

	id, ch := bus.Subscribe(10)
	if bus.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe(id)
	if bus.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}

	// Channel should be closed
	_, open := <-ch
	if open {
		t.Error("channel should be closed after unsubscribe")
	}
}

func TestEventBus_SlowSubscriberDropsEvents(t *testing.T) {
	bus := NewEventBus()

	// Buffer of 1 — second event should be dropped
	id, ch := bus.Subscribe(1)
	defer bus.Unsubscribe(id)

	bus.Publish(Event{Type: EventSpecCreated, Slug: "first"})
	bus.Publish(Event{Type: EventSpecModified, Slug: "second"})

	ev := <-ch
	if ev.Slug != "first" {
		t.Errorf("expected first event, got %q", ev.Slug)
	}

	// Second event was dropped because buffer was full
	select {
	case ev := <-ch:
		t.Errorf("expected no more events, got %q", ev.Slug)
	default:
		// Expected — buffer was full, event was dropped
	}
}

func TestEventBus_NoSubscribers(t *testing.T) {
	bus := NewEventBus()

	// Should not panic
	bus.Publish(Event{Type: EventIndexRebuilt, Message: "ok"})
}

func TestEventBus_TimestampAutoSet(t *testing.T) {
	bus := NewEventBus()
	id, ch := bus.Subscribe(10)
	defer bus.Unsubscribe(id)

	before := time.Now()
	bus.Publish(Event{Type: EventSpecCreated})
	after := time.Now()

	ev := <-ch
	if ev.Timestamp.Before(before) || ev.Timestamp.After(after) {
		t.Errorf("timestamp %v not between %v and %v", ev.Timestamp, before, after)
	}
}

func TestEventBus_ExplicitTimestampPreserved(t *testing.T) {
	bus := NewEventBus()
	id, ch := bus.Subscribe(10)
	defer bus.Unsubscribe(id)

	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	bus.Publish(Event{Type: EventSpecCreated, Timestamp: ts})

	ev := <-ch
	if !ev.Timestamp.Equal(ts) {
		t.Errorf("timestamp = %v, want %v", ev.Timestamp, ts)
	}
}

func TestEventBus_SubscriberCount(t *testing.T) {
	bus := NewEventBus()
	if bus.SubscriberCount() != 0 {
		t.Fatalf("expected 0, got %d", bus.SubscriberCount())
	}

	id1, _ := bus.Subscribe(1)
	id2, _ := bus.Subscribe(1)
	if bus.SubscriberCount() != 2 {
		t.Fatalf("expected 2, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe(id1)
	if bus.SubscriberCount() != 1 {
		t.Fatalf("expected 1, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe(id2)
	if bus.SubscriberCount() != 0 {
		t.Fatalf("expected 0, got %d", bus.SubscriberCount())
	}
}
