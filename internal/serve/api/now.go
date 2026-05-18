// Package api hosts cross-cutting HTTP endpoints that sit alongside the
// /api/* surface owned by the serve package but are scoped to a single
// home or shared concern. The Now home's SSE channel and per-section
// fragment endpoints live here so the page package (internal/serve/
// pages/now) doesn't need to depend on the serve.EventBus directly.
package api

import (
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
//   GET /api/now/quicklaunch     — fragment endpoint for the Quick launch section,
//                                  re-rendered on adapter connect / disconnect so
//                                  the empty-state notice flips without a reload
func (h *NowHandler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/now/events", h.handleEvents)
	mux.HandleFunc("/api/now/inbox", h.handleSection("inbox"))
	mux.HandleFunc("/api/now/plate", h.handleSection("plate"))
	mux.HandleFunc("/api/now/agents", h.handleSection("agents"))
	mux.HandleFunc("/api/now/changes", h.handleSection("changes"))
	mux.HandleFunc("/api/now/quicklaunch", h.handleQuickLaunch)
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
	// but we collapse close-together events per section name. "hero"
	// and "capability" are treated as virtual section names so the
	// same debounce + emit path covers the page-hero subhead refresh
	// and the adapter-availability fragment refresh.
	pending := map[string]*time.Timer{}
	emit := func(section string) {
		switch section {
		case "hero":
			// Plain-text subhead payload; client drops it into the
			// [data-page-hero-subhead] span's textContent.
			fmt.Fprintf(w, "event: hero\ndata: %s\n\n", escapeSSEData(now.SubheadText(h.deps)))
		default:
			fmt.Fprintf(w, "event: %s\ndata: \n\n", section)
		}
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
			for _, section := range sectionsForEventType(ev.Type) {
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

// handleQuickLaunch renders the Quick launch section as a standalone
// HTML fragment. Driven by the `event: capability` SSE channel — when
// an adapter connects or disconnects the client refetches this URL and
// swaps `#now-quicklaunch` in place, so the empty-state notice flips
// without a full page reload.
func (h *NowHandler) handleQuickLaunch(w http.ResponseWriter, r *http.Request) {
	body, err := now.QuickLaunchFragment(h.deps)
	if err != nil {
		http.Error(w, "render quicklaunch: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body)
}

// escapeSSEData renders s as the data: line(s) of an SSE frame.
// Newlines are split into separate data: lines per the SSE spec; the
// caller is responsible for the trailing blank line. Today's
// SubheadText output is single-line so this is defensive only.
func escapeSSEData(s string) string {
	if s == "" {
		return ""
	}
	// SSE allows multi-line data by prefixing each line with "data: ".
	// We collapse newlines to spaces — page-hero subheads are never
	// multi-line and this keeps the SSE frame format trivial.
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\n', '\r':
			out = append(out, ' ')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}

// sectionForEventType maps an upstream event type to the Now section
// that should refresh. Returns "" when the event doesn't map to a
// section (in which case it is silently dropped).
//
// Kept as the single-section helper for backwards compatibility with
// the existing TestSectionForEventType cases; sectionsForEventType
// extends it with the cross-cutting "hero" / "capability" channels.
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

// sectionsForEventType returns the full set of virtual section names
// an upstream event triggers. Most events fan out to a single section,
// but inbox/agent count changes also bump the page-hero subhead, and
// chat-adapter lifecycle events bump the Quick launch fragment.
func sectionsForEventType(t string) []string {
	primary := sectionForEventType(t)
	switch {
	case t == "":
		return nil
	case primary == "inbox":
		// Inbox count drives the subhead's "N needs your input" segment.
		return []string{primary, "hero"}
	case primary == "plate":
		return []string{primary}
	case stringHasPrefix(t, "session.") || stringHasPrefix(t, "agent."):
		// Live-session lifecycle changes bump both the Currently-
		// running card and the page-hero subhead's running-count.
		return []string{"agents", "hero"}
	case stringHasPrefix(t, "chat.adapter.") || t == "chat.connected" || t == "chat.disconnected":
		return []string{"capability"}
	case primary != "":
		return []string{primary}
	default:
		return nil
	}
}

// stringHasPrefix is a tiny local helper that avoids dragging the
// strings package import in for one call site.
func stringHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
