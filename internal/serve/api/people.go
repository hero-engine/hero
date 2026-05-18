package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/serve/pages/people"
)

// PeopleHandler owns the People & ROI SSE channel and per-section
// fragment endpoints. Construct with NewPeopleHandler and Mount() into
// a top-level mux. Mirrors NowHandler's shape so server.go wires it the
// same way.
type PeopleHandler struct {
	deps        people.Deps
	sub         Subscriber
	debounceDur time.Duration

	mu      sync.Mutex
	pending map[string]*time.Timer
}

// NewPeopleHandler wires the People SSE channel to the active
// subscriber and dependency bundle. A nil sub disables live updates —
// the fragment endpoints still serve standalone HTML.
func NewPeopleHandler(deps people.Deps, sub Subscriber) *PeopleHandler {
	return &PeopleHandler{
		deps:        deps,
		sub:         sub,
		debounceDur: 500 * time.Millisecond, // spec: ≤1 feed fragment / 500ms / subscriber
		pending:     make(map[string]*time.Timer),
	}
}

// Mount registers the People endpoints on the given mux:
//   GET /api/people/events           — SSE channel multiplexing per-section refreshes
//   GET /api/people/{section}        — fragment endpoint returning the section HTML
func (h *PeopleHandler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/people/events", h.handleEvents)
	mux.HandleFunc("/api/people/pulse", h.handleSection("pulse"))
	mux.HandleFunc("/api/people/overview", h.handleSection("overview"))
}

func (h *PeopleHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
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
			if section := peopleSectionForEventType(ev.Type); section != "" {
				schedule(section)
			}
		}
	}
}

func (h *PeopleHandler) handleSection(section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := people.SectionFragment(h.deps, section)
		if err != nil {
			http.Error(w, "render section: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	}
}

// peopleSectionForEventType maps an upstream event type to the People
// section that should refresh. The pulse view consumes the activity
// feed; ROI views refresh on a slow timer (handled by client JS) so
// only feed-relevant events are mapped here.
func peopleSectionForEventType(t string) string {
	switch {
	case t == "":
		return ""
	case t == "spec.created", t == "spec.modified", t == "spec.deleted":
		return "pulse"
	case t == "proposal_emitted", t == "proposal_accepted",
		t == "proposal_edited", t == "proposal_rejected",
		t == "proposal_dismissed":
		return "pulse"
	case stringHasPrefix(t, "peer."):
		return "pulse"
	default:
		return ""
	}
}
