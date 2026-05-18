package chat

import (
	"sync"
	"testing"
)

type captureBus struct {
	mu     sync.Mutex
	events []BusEvent
}

func (c *captureBus) Publish(ev BusEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func (c *captureBus) snapshot() []BusEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]BusEvent, len(c.events))
	copy(out, c.events)
	return out
}

func TestStreamerPublishesWithPrefix(t *testing.T) {
	bus := &captureBus{}
	s := NewStreamer(bus)
	s.Publish("conv1", TokenEvent("hello"))
	s.Publish("conv1", DoneEvent(0.5, map[string]interface{}{"file": "x"}))

	events := bus.snapshot()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != "chat.token" {
		t.Errorf("type[0] = %q, want chat.token", events[0].Type)
	}
	if events[0].Topic != "chat.conv1" {
		t.Errorf("topic[0] = %q, want chat.conv1", events[0].Topic)
	}
	if events[0].Payload["conversation_id"] != "conv1" {
		t.Errorf("payload[0].conversation_id = %v", events[0].Payload["conversation_id"])
	}
	if events[0].Payload["text"] != "hello" {
		t.Errorf("payload[0].text = %v", events[0].Payload["text"])
	}
	if events[1].Type != "chat.done" {
		t.Errorf("type[1] = %q, want chat.done", events[1].Type)
	}
}

func TestStreamerNilBusNoop(t *testing.T) {
	s := NewStreamer(nil)
	// Must not panic.
	s.Publish("c", TokenEvent("x"))
}
