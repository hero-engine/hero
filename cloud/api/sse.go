package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hero-engine/hero/cloud/middleware"
	"github.com/hero-engine/hero/cloud/store"
)

// SSEHub manages Server-Sent Events connections per org.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[string]map[chan []byte]struct{} // orgID -> set of channels
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string]map[chan []byte]struct{}),
	}
}

// subscribe adds a client channel for the given org.
func (h *SSEHub) subscribe(orgID string) chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	if h.clients[orgID] == nil {
		h.clients[orgID] = make(map[chan []byte]struct{})
	}
	h.clients[orgID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// unsubscribe removes a client channel.
func (h *SSEHub) unsubscribe(orgID string, ch chan []byte) {
	h.mu.Lock()
	if chs, ok := h.clients[orgID]; ok {
		delete(chs, ch)
		if len(chs) == 0 {
			delete(h.clients, orgID)
		}
	}
	h.mu.Unlock()
	close(ch)
}

// Broadcast sends an event to all subscribers for an org.
func (h *SSEHub) Broadcast(orgID string, event SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("sse marshal error: %v", err)
		return
	}

	h.mu.RLock()
	chs := h.clients[orgID]
	h.mu.RUnlock()

	for ch := range chs {
		select {
		case ch <- data:
		default:
			// slow client, drop message
		}
	}
}

// SSEEvent is a server-sent event payload.
type SSEEvent struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// handleSSE serves the SSE stream for an org's activity feed.
func handleSSE(hub *SSEHub, db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.GetClaims(r.Context())
		if claims == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		orgID := r.PathValue("org_id")
		role, err := db.GetMemberRole(r.Context(), orgID, claims.UserID)
		if err != nil || role == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		ch := hub.subscribe(orgID)
		defer hub.unsubscribe(orgID, ch)

		// Send initial keepalive
		fmt.Fprintf(w, ": connected\n\n")
		flusher.Flush()

		// Keepalive ticker
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-ch:
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-ticker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	}
}
