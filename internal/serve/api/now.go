// Package api hosts cross-cutting HTTP endpoints that sit alongside the
// /api/* surface owned by the serve package but are scoped to a single
// home or shared concern. The Now home's SSE channel and per-section
// fragment endpoints live here so the page package (internal/serve/
// pages/now) doesn't need to depend on the serve.EventBus directly.
package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/serve/pages/now"
)

// Subscriber is the minimal subset of an event bus the Now SSE channel
// needs: subscribe to a typed event channel and unsubscribe by id.
// Implemented by *serve.EventBus.
type Subscriber interface {
	Subscribe(bufSize int) (uint64, <-chan Event)
	Unsubscribe(id uint64)
}

// Event mirrors serve.Event for the bits the Now SSE channel reads.
// Kept here so this package stays free of cross-package imports back
// into serve.
type Event struct {
	Type string
}

// NowHandler owns the Now SSE channel and per-section fragment
// endpoints. Construct with NewNowHandler and Mount() into a top-level
// mux.
type NowHandler struct {
	deps        now.Deps
	sub         Subscriber
	debounceDur time.Duration

	mu      sync.Mutex
	pending map[string]*time.Timer // section → pending debounce timer
}

// NewNowHandler wires the Now SSE channel to the active subscriber and
// dependency bundle. A nil sub disables live updates — the fragment
// endpoints still serve standalone HTML.
func NewNowHandler(deps now.Deps, sub Subscriber) *NowHandler {
	return &NowHandler{
		deps:        deps,
		sub:         sub,
		debounceDur: 250 * time.Millisecond,
		pending:     make(map[string]*time.Timer),
	}
}

// Mount registers the Now endpoints on the given mux:
//   GET /api/now/events          — SSE channel multiplexing per-section refreshes
//   GET /api/now/{section}       — fragment endpoint returning the section HTML
func (h *NowHandler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/now/events", h.handleEvents)
	mux.HandleFunc("/api/now/inbox", h.handleSection("inbox"))
	mux.HandleFunc("/api/now/plate", h.handleSection("plate"))
	mux.HandleFunc("/api/now/agents", h.handleSection("agents"))
	mux.HandleFunc("/api/now/changes", h.handleSection("changes"))
}

// handleEvents streams Server-Sent Events to the Now page. Filters the
// upstream event bus into per-section refresh signals, debouncing each
// signal so a storm of events under one section produces at most one
// refresh per debounce window.
func (h *NowHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Initial connected frame so the client knows we're live.
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	if h.sub == nil {
		// No upstream bus — hold the connection open for the client's
		// retry loop and exit on context cancel.
		<-r.Context().Done()
		return
	}

	id, ch := h.sub.Subscribe(64)
	defer h.sub.Unsubscribe(id)

	// Per-connection debounce: we don't share with other Now clients,
	// but we collapse close-together events per section name.
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
			if section := sectionForEventType(ev.Type); section != "" {
				schedule(section)
			}
		}
	}
}

// handleSection returns an http.HandlerFunc that renders the named
// section as a standalone HTML fragment. Used by the client-side SSE
// subscriber to swap a section in place.
func (h *NowHandler) handleSection(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := now.SectionFragment(h.deps, section)
		if err != nil {
			http.Error(w, "render section: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	}
}

// sectionForEventType maps an upstream event type to the Now section
// that should refresh. Returns "" when the event doesn't map to a
// section (in which case it is silently dropped).
func sectionForEventType(t string) string {
	switch {
	case t == "":
		return ""
	case t == "proposal_emitted", t == "proposal_accepted",
		t == "proposal_edited", t == "proposal_rejected",
		t == "proposal_dismissed":
		return "inbox"
	case t == "spec.created", t == "spec.modified", t == "spec.deleted":
		return "plate"
	case t == "index.rebuilt":
		return "changes"
	case stringHasPrefix(t, "peer."):
		return "inbox"
	default:
		return ""
	}
}

// stringHasPrefix is a tiny local helper that avoids dragging the
// strings package import in for one call site.
func stringHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// _ keeps the context import live for the file regardless of build tags.
var _ = context.Background
