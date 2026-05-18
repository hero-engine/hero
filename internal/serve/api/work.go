// Work SSE channel + per-section fragment endpoints. Mirrors the Now
// handler shape: a single /api/work/events stream multiplexes
// per-section refresh signals, debounced per section so a storm of
// upstream events under one section produces at most one refresh per
// debounce window. Fragment endpoints serve standalone HTML so the
// client-side SSE subscriber can swap a section in place.
//
// The Subscriber + Event types are defined alongside the Now handler
// (see now.go) and shared across this file — both handlers consume the
// same upstream event bus.
package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/serve/pages/work"
)

// WorkHandler owns the Work SSE channel and per-section fragment
// endpoints. Construct with NewWorkHandler and Mount() into a top-level
// mux.
type WorkHandler struct {
	deps        work.Deps
	sub         Subscriber
	debounceDur time.Duration

	mu      sync.Mutex
	pending map[string]*time.Timer // section → pending debounce timer
}

// NewWorkHandler wires the Work SSE channel to the active subscriber
// and dependency bundle. A nil sub disables live updates — the fragment
// endpoints still serve standalone HTML.
func NewWorkHandler(deps work.Deps, sub Subscriber) *WorkHandler {
	return &WorkHandler{
		deps:        deps,
		sub:         sub,
		debounceDur: 250 * time.Millisecond,
		pending:     make(map[string]*time.Timer),
	}
}

// Mount registers the Work endpoints on the given mux:
//
//	GET /api/work/events         — SSE channel multiplexing per-section refreshes
//	GET /api/work/{section}      — fragment endpoint returning the section HTML
func (h *WorkHandler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/work/events", h.handleEvents)
	mux.HandleFunc("/api/work/roadmap", h.handleSection("roadmap"))
	mux.HandleFunc("/api/work/blocked", h.handleSection("blocked"))
	mux.HandleFunc("/api/work/shipped", h.handleSection("shipped"))
	mux.HandleFunc("/api/work/toolbar", h.handleSection("toolbar"))
}

// handleEvents streams Server-Sent Events to the Work page. Filters the
// upstream event bus into per-section refresh signals, debouncing each
// signal so a storm of events under one section produces at most one
// refresh per debounce window.
func (h *WorkHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
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
			if section := sectionForWorkEventType(ev.Type); section != "" {
				schedule(section)
			}
		}
	}
}

// handleSection returns an http.HandlerFunc that renders the named
// section as a standalone HTML fragment. Used by the client-side SSE
// subscriber to swap a section in place. Per v5 Fix 4, the toolbar
// section honors `?view=<slug>` so a fragment refresh on a non-root
// Work sub-route doesn't snap the active tab back to Horizons.
func (h *WorkHandler) handleSection(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view := r.URL.Query().Get("view")
		body, err := work.SectionFragment(h.deps, section, view)
		if err != nil {
			http.Error(w, "render section: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	}
}

// sectionForWorkEventType maps an upstream event type to the Work
// section that should refresh. Returns "" when the event doesn't map
// to a section (in which case it is silently dropped).
//
// Mapping rationale:
//   - spec.* (created/modified/deleted): a card may have appeared, the
//     count badge changed, and the roadmap layout shifted. Refresh
//     the roadmap and the view toolbar's blocked badge.
//   - index.rebuilt: shipped specs may have appeared. Refresh shipped.
//   - drift.* + ci.*: signals on a card changed. Refresh roadmap.
func sectionForWorkEventType(t string) string {
	switch {
	case t == "":
		return ""
	case t == "spec.created", t == "spec.modified", t == "spec.deleted":
		return "roadmap"
	case t == "index.rebuilt":
		return "shipped"
	case stringHasPrefix(t, "drift."), stringHasPrefix(t, "ci."):
		return "roadmap"
	case t == "delivery_complete":
		return "shipped"
	default:
		return ""
	}
}
