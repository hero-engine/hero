package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/serve/pages/agentspage"
)

// AgentsHandler owns the Agents SSE channel and per-section fragment
// endpoints. Construct with NewAgentsHandler and Mount() into a
// top-level mux.
type AgentsHandler struct {
	deps        agentspage.Deps
	sub         Subscriber
	debounceDur time.Duration

	mu      sync.Mutex
	pending map[string]*time.Timer
}

// NewAgentsHandler wires the Agents SSE channel to the active
// subscriber and dependency bundle. A nil sub disables live updates —
// the fragment endpoints still serve standalone HTML.
func NewAgentsHandler(deps agentspage.Deps, sub Subscriber) *AgentsHandler {
	return &AgentsHandler{
		deps:        deps,
		sub:         sub,
		debounceDur: 250 * time.Millisecond,
		pending:     make(map[string]*time.Timer),
	}
}

// Mount registers the Agents endpoints on the given mux:
//
//	GET /api/agents/events             — SSE channel multiplexing per-section refreshes
//	GET /api/agents/{section}          — fragment endpoint returning the section HTML
//
// The per-session SSE topic (`session.token` stream) and the per-item
// CRUD endpoints described in the spec land in a follow-up; v1 lights
// up the page-level multiplex stream and the four section fragments.
func (h *AgentsHandler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/agents/events", h.handleEvents)
	mux.HandleFunc("/api/agents/sessions", h.handleSection("sessions"))
	mux.HandleFunc("/api/agents/approvals", h.handleSection("approvals"))
	mux.HandleFunc("/api/agents/completed", h.handleSection("completed"))
	mux.HandleFunc("/api/agents/scheduled-preview", h.handleSection("scheduled-preview"))
}

// handleEvents streams Server-Sent Events to the Agents page. Filters
// the upstream event bus into per-section refresh signals, debouncing
// each signal so a storm of events under one section produces at most
// one refresh per debounce window.
func (h *AgentsHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	if h.sub == nil {
		<-r.Context().Done()
		return
	}

	id, ch := h.sub.Subscribe(64)
	defer h.sub.Unsubscribe(id)

	pending := map[string]*time.Timer{}
	emit := func(section string) {
		fmt.Fprintf(w, "event: %s\ndata: \n\n", section)
		flusher.Flush()
	}
	schedule := func(section string) {
		if t, ok := pending[section]; ok {
			t.Reset(h.debounceDur)
			return
		}
		t := time.AfterFunc(h.debounceDur, func() { emit(section) })
		pending[section] = t
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			for _, t := range pending {
				t.Stop()
			}
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if section := agentsSectionForEventType(ev.Type); section != "" {
				schedule(section)
			}
		}
	}
}

// handleSection returns an http.HandlerFunc that renders the named
// section as a standalone HTML fragment. Used by the client-side SSE
// subscriber to swap a section in place.
func (h *AgentsHandler) handleSection(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := agentspage.SectionFragment(h.deps, section)
		if err != nil {
			http.Error(w, "render section: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	}
}

// agentsSectionForEventType maps an upstream event type to the Agents
// section that should refresh. Returns "" when the event doesn't map.
func agentsSectionForEventType(t string) string {
	switch {
	case t == "":
		return ""
	case t == "proposal_emitted", t == "proposal_accepted",
		t == "proposal_edited", t == "proposal_rejected",
		t == "proposal_dismissed":
		return "approvals"
	case stringHasPrefix(t, "session."):
		return "sessions"
	case stringHasPrefix(t, "scheduled."), stringHasPrefix(t, "deferred."):
		return "scheduled-preview"
	case stringHasPrefix(t, "automation."):
		return "scheduled-preview"
	default:
		return ""
	}
}
