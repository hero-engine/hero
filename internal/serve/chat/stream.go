package chat

import "time"

// EventBus is the minimum interface the chat streamer needs from the
// serve event bus. Defining it here (rather than importing the parent
// package) avoids a circular import: the parent serve package consumes
// chat for its API handlers, so chat must not depend on it directly.
//
// serve.EventBus satisfies this interface structurally via a tiny
// adapter (busAdapter in api_chat.go).
type EventBus interface {
	Publish(BusEvent)
}

// BusEvent mirrors the subset of serve.Event the chat layer
// populates. The serve-side adapter copies fields across.
type BusEvent struct {
	Type      string                 // bus event type, e.g. "chat.token"
	Topic     string                 // per-conversation topic, "chat." + conversation_id
	Payload   map[string]interface{} // raw payload (includes conversation_id)
	Timestamp time.Time
}

// Streamer republishes adapter / slash events onto the bus, namespaced
// per-conversation.
type Streamer struct {
	bus EventBus
}

// NewStreamer constructs a Streamer over the given bus. bus may be
// nil for tests that only assert handler behavior; Publish becomes a
// no-op in that case.
func NewStreamer(bus EventBus) *Streamer {
	return &Streamer{bus: bus}
}

// Publish republishes ev on the bus, prefixing the event type with
// "chat." and injecting conversation_id into the payload so consumers
// can filter without re-parsing the topic.
//
// Subscribers can filter incoming events by checking either the bus
// event Type (a "chat.<sub>" string) or the topic field, both of
// which carry the conversation id.
func (s *Streamer) Publish(conversationID string, ev Event) {
	if s == nil || s.bus == nil {
		return
	}
	payload := map[string]interface{}{
		"conversation_id": conversationID,
	}
	for k, v := range ev.Payload {
		payload[k] = v
	}
	s.bus.Publish(BusEvent{
		Type:      "chat." + ev.Type,
		Topic:     "chat." + conversationID,
		Payload:   payload,
		Timestamp: time.Now(),
	})
}
