package serve

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Event types
// ---------------------------------------------------------------------------

// EventType identifies what happened.
type EventType string

const (
	EventSpecCreated  EventType = "spec.created"
	EventSpecModified EventType = "spec.modified"
	EventSpecDeleted  EventType = "spec.deleted"
	EventIndexRebuilt EventType = "index.rebuilt"
	EventHealthCheck  EventType = "health.check"
)

// Event is a single item published to the bus.
type Event struct {
	Type      EventType `json:"type"`
	Project   string    `json:"project,omitempty"`
	Slug      string    `json:"slug,omitempty"`
	Path      string    `json:"path,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ---------------------------------------------------------------------------
// EventBus — in-memory pub/sub
// ---------------------------------------------------------------------------

// EventBus distributes events to multiple subscribers.
type EventBus struct {
	mu   sync.RWMutex
	subs map[uint64]chan Event
	next uint64
}

// NewEventBus creates an event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subs: make(map[uint64]chan Event),
	}
}

// Subscribe returns a channel that receives events and an ID used to unsubscribe.
// The buffer size controls how many events can be queued before the subscriber
// blocks the publisher (dropped on slow consumers).
func (eb *EventBus) Subscribe(bufSize int) (uint64, <-chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	id := eb.next
	eb.next++
	ch := make(chan Event, bufSize)
	eb.subs[id] = ch
	return id, ch
}

// Unsubscribe removes a subscriber and closes its channel.
func (eb *EventBus) Unsubscribe(id uint64) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch, ok := eb.subs[id]
	if ok {
		delete(eb.subs, id)
		close(ch)
	}
}

// Publish sends an event to all subscribers.
// Non-blocking: if a subscriber's buffer is full the event is dropped for that subscriber.
func (eb *EventBus) Publish(ev Event) {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}

	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, ch := range eb.subs {
		select {
		case ch <- ev:
		default:
			// subscriber too slow — drop event
		}
	}
}

// SubscriberCount returns the current number of subscribers.
func (eb *EventBus) SubscriberCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subs)
}

// ---------------------------------------------------------------------------
// SSE handler
// ---------------------------------------------------------------------------

// SSEHandler returns an http.HandlerFunc that streams events as Server-Sent Events.
// Supports ?project=slug to filter events to a single project.
func SSEHandler(bus *EventBus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		projectFilter := r.URL.Query().Get("project")

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		id, ch := bus.Subscribe(64)
		defer bus.Unsubscribe(id)

		// Send initial connected event
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				// Filter by project if requested
				if projectFilter != "" && ev.Project != "" && ev.Project != projectFilter {
					continue
				}
				fmt.Fprintf(w, "event: %s\ndata: {\"project\":%q,\"slug\":%q,\"path\":%q,\"message\":%q,\"timestamp\":%q}\n\n",
					ev.Type, ev.Project, ev.Slug, ev.Path, ev.Message, ev.Timestamp.Format(time.RFC3339))
				flusher.Flush()
			}
		}
	}
}
