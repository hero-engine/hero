// Package api — Knowledge home SSE channel and per-section fragment
// endpoints, mirroring the Now-home wire-up in this same package. The
// page package (internal/serve/pages/knowledge) supplies the section-
// fragment renderer; this file owns the multiplexing onto the event
// bus and the HTTP plumbing.
package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/serve/pages/knowledge"
)

// KnowledgeHandler owns the Knowledge SSE channel and per-section
// fragment endpoints. Construct with NewKnowledgeHandler and Mount()
// into a top-level mux.
type KnowledgeHandler struct {
	deps        knowledge.Deps
	sub         Subscriber
	debounceDur time.Duration

	mu      sync.Mutex
	pending map[string]*time.Timer // section → pending debounce timer
}

// NewKnowledgeHandler wires the Knowledge SSE channel to the active
// subscriber and dependency bundle. A nil sub disables live updates —
// the fragment endpoints still serve standalone HTML.
func NewKnowledgeHandler(deps knowledge.Deps, sub Subscriber) *KnowledgeHandler {
	return &KnowledgeHandler{
		deps:        deps,
		sub:         sub,
		debounceDur: 250 * time.Millisecond,
		pending:     make(map[string]*time.Timer),
	}
}

// Mount registers the Knowledge endpoints on the given mux:
//   GET /api/knowledge/events              — SSE channel multiplexing per-section refreshes
//   GET /api/knowledge/{section}           — fragment endpoint returning the section HTML
func (h *KnowledgeHandler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/knowledge/events", h.handleEvents)
	mux.HandleFunc("/api/knowledge/browse", h.handleSection("browse"))
	mux.HandleFunc("/api/knowledge/provenance", h.handleSection("provenance"))
	mux.HandleFunc("/api/knowledge/summary", h.handleSection("summary"))
	mux.HandleFunc("/api/knowledge/neighbors", h.handleSection("neighbors"))
	mux.HandleFunc("/api/knowledge/staleness", h.handleSection("staleness"))
}

// handleEvents streams Server-Sent Events to the Knowledge page.
// Filters the upstream event bus into per-section refresh signals,
// debouncing each signal so a storm of events under one section
// produces at most one refresh per debounce window.
func (h *KnowledgeHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
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
		t := time.AfterFunc(h.debounceDur, func() {
			emit(section)
		})
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
			if section := knowledgeSectionForEventType(ev.Type); section != "" {
				schedule(section)
			}
		}
	}
}

// handleSection returns an http.HandlerFunc that renders the named
// section as a standalone HTML fragment. Used by the client-side SSE
// subscriber to swap a section in place.
func (h *KnowledgeHandler) handleSection(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := knowledge.SectionFragment(h.deps, section)
		if err != nil {
			http.Error(w, "render section: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	}
}

// knowledgeSectionForEventType maps an upstream event type to the
// Knowledge section that should refresh. Returns "" when the event
// doesn't map to a section (in which case it is silently dropped).
func knowledgeSectionForEventType(t string) string {
	switch {
	case t == "":
		return ""
	case t == "capture.created", t == "knowledge.created",
		t == "decision.created", t == "convention.created",
		t == "learning.captured":
		return "browse"
	case t == "contradictions.updated", t == "staleness.updated":
		return "staleness"
	case t == "index.rebuilt":
		return "browse"
	default:
		return ""
	}
}
