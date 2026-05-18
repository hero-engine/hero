package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hero-engine/hero/internal/serve/session"
)

// API ties the chat subsystem to HTTP. Construct one per process via
// NewAPI and call Mount to register handlers on a mux.
type API struct {
	registry  *Registry
	store     *Store
	streamer  *Streamer
	workspace string

	// installLink is surfaced in no-adapter chat.error events so the
	// UI can render an "Install hero-code →" CTA. Default points at
	// the public install page; an empty value omits the link.
	installLink string
}

// NewAPI constructs a chat API surface.
//
// workspace is the absolute path to the user's project root (the same
// path serve uses to find .hero/). Slashes that need filesystem
// access resolve heroDir from req.Context.Workspace, which the API
// populates from this value when the request omits it.
func NewAPI(registry *Registry, store *Store, streamer *Streamer, workspace string) *API {
	return &API{
		registry:    registry,
		store:       store,
		streamer:    streamer,
		workspace:   workspace,
		installLink: "https://heroengine.ai/install/hero-code",
	}
}

// Mount registers chat HTTP handlers under /api/chat/ on the given
// mux. Endpoint set documented in api-contract.md.
func (a *API) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/chat/capability", a.handleCapability)
	mux.HandleFunc("/api/chat/turn", a.handleTurn)
	mux.HandleFunc("/api/chat/history", a.handleHistory)
	mux.HandleFunc("/api/chat/preference", a.handlePreference)
	mux.HandleFunc("/api/chat/clear", a.handleClear)
}

func (a *API) handleCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pref := ""
	if a.store != nil {
		if p, err := a.store.Preference(session.UserID(r)); err == nil {
			pref = p
		}
	}
	cap := Resolve(a.registry, pref)
	writeJSON(w, http.StatusOK, cap)
}

type turnRequest struct {
	Prompt         string          `json:"prompt"`
	ConversationID string          `json:"conversation_id"`
	PageScope      string          `json:"page_scope"`
	Context        DispatchContext `json:"context"`
}

type turnResponse struct {
	ConversationID string `json:"conversation_id"`
	SSETopic       string `json:"sse_topic"`
}

func (a *API) handleTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req turnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		http.Error(w, "prompt required", http.StatusBadRequest)
		return
	}
	userID := session.UserID(r)
	scope := req.PageScope
	if scope == "" {
		scope = "global"
	}

	// Resolve conversation (create if missing).
	convID := req.ConversationID
	if convID == "" && a.store != nil {
		id, err := a.store.NewConversation(userID, scope)
		if err != nil {
			http.Error(w, "create conversation: "+err.Error(), http.StatusInternalServerError)
			return
		}
		convID = id
	} else if convID == "" {
		// Fallback id when no store is configured (test mode).
		id, err := newID()
		if err != nil {
			http.Error(w, "mint id: "+err.Error(), http.StatusInternalServerError)
			return
		}
		convID = id
	}

	// Persist the user turn before dispatching so history reflects
	// the prompt even if the adapter never responds.
	if a.store != nil {
		if err := a.store.AppendMessage(convID, "user", prompt); err != nil {
			// Non-fatal: log via header so tests can detect.
			w.Header().Set("X-Hero-Chat-Persist-Error", err.Error())
		}
	}

	// Populate workspace from server config if the request did not.
	ctx := req.Context
	if ctx.Workspace == "" {
		ctx.Workspace = a.workspace
	}

	slash, _ := ParseSlash(prompt)
	disp := DispatchRequest{
		Kind:           KindInteractive,
		ConversationID: convID,
		Prompt:         prompt,
		Context:        ctx,
		Slash:          slash,
	}
	if a.store != nil {
		history, _ := a.store.History(convID, MaxHistoryTurns)
		disp.History = history
	}

	// Resolve user preference now while we still have the request;
	// dispatch runs in a goroutine that no longer sees the HTTP
	// context.
	pref := ""
	if a.store != nil {
		pref, _ = a.store.Preference(userID)
	}

	// Respond immediately with the conversation id + sse topic so
	// the UI can subscribe before events finish streaming. Then
	// dispatch in the background.
	resp := turnResponse{
		ConversationID: convID,
		SSETopic:       "chat." + convID,
	}

	// For runner-free slashes we dispatch synchronously before
	// responding so the events have published by the time the UI
	// looks at the topic. For adapter-dispatched turns we go async
	// because the adapter may stream for many seconds.
	runSync := slash != nil && isRunnerFree(slash.Name)
	if runSync {
		a.dispatch(r.Context(), disp, pref)
		writeJSON(w, http.StatusOK, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
	go a.dispatch(context.Background(), disp, pref)
}

func isRunnerFree(name string) bool {
	s, ok := Lookup(name)
	return ok && s.RunnerFree
}

// dispatch routes a turn to a runner-free slash handler or to the
// resolved interactive adapter, republishing emitted events on the
// bus. pref is the caller's preferred adapter type (looked up
// upstream from the user record).
func (a *API) dispatch(ctx context.Context, req DispatchRequest, pref string) {
	if req.Slash != nil {
		if s, ok := Lookup(req.Slash.Name); ok && s.RunnerFree {
			a.runSlash(ctx, s, req)
			return
		}
		if !exists(req.Slash.Name) {
			a.streamer.Publish(req.ConversationID, ErrorEvent("slash_unknown", "unknown slash /"+req.Slash.Name, ""))
			a.streamer.Publish(req.ConversationID, DoneEvent(0, nil))
			return
		}
	}
	a.dispatchToAdapter(ctx, req, pref)
}

func exists(name string) bool {
	_, ok := Lookup(name)
	return ok
}

func (a *API) runSlash(ctx context.Context, s Slash, req DispatchRequest) {
	out := make(chan Event, 16)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range out {
			a.streamer.Publish(req.ConversationID, ev)
			if ev.Type == EvDone {
				a.persistAssistant(req.ConversationID, ev)
			}
		}
	}()
	err := s.Handler(ctx, req, out)
	close(out)
	<-done
	if err != nil {
		// Handler returned non-nil after emitting events — publish a
		// trailing error so subscribers see the failure.
		a.streamer.Publish(req.ConversationID, ErrorEvent("slash_failed", err.Error(), ""))
	}
}

func (a *API) dispatchToAdapter(ctx context.Context, req DispatchRequest, pref string) {
	cap := Resolve(a.registry, pref)
	if cap.Interactive == "" {
		a.streamer.Publish(req.ConversationID, ErrorEvent(
			"no_adapter",
			"No Hero adapter is connected. Install hero-code to run agent workflows.",
			a.installLink,
		))
		a.streamer.Publish(req.ConversationID, DoneEvent(0, nil))
		return
	}
	adapter := a.registry.Get(cap.Interactive)
	if adapter == nil {
		a.streamer.Publish(req.ConversationID, ErrorEvent("no_adapter", "adapter vanished mid-dispatch", a.installLink))
		a.streamer.Publish(req.ConversationID, DoneEvent(0, nil))
		return
	}
	stream, err := adapter.Stream(ctx, req)
	if err != nil {
		a.streamer.Publish(req.ConversationID, ErrorEvent("adapter_error", err.Error(), ""))
		a.streamer.Publish(req.ConversationID, DoneEvent(0, nil))
		return
	}
	for ev := range stream {
		a.streamer.Publish(req.ConversationID, ev)
		if ev.Type == EvDone {
			a.persistAssistant(req.ConversationID, ev)
		}
	}
}

// persistAssistant records a synthetic "assistant" turn summary in
// the store. The summary today is just the outcome JSON or "(empty)";
// future versions can stitch streamed tokens together as they arrive.
func (a *API) persistAssistant(convID string, done Event) {
	if a.store == nil {
		return
	}
	var body string
	if outcome, ok := done.Payload["outcome"].(map[string]interface{}); ok {
		if raw, err := json.Marshal(outcome); err == nil {
			body = string(raw)
		}
	}
	if body == "" {
		body = "(done)"
	}
	_ = a.store.AppendMessage(convID, "assistant", body)
}

type historyResponse struct {
	Scope string        `json:"scope"`
	Turns []HistoryTurn `json:"turns"`
}

func (a *API) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "global"
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}
	turns := []HistoryTurn{}
	if a.store != nil {
		t, err := a.store.HistoryByScope(session.UserID(r), scope, limit)
		if err != nil {
			http.Error(w, "history: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if t != nil {
			turns = t
		}
	}
	writeJSON(w, http.StatusOK, historyResponse{Scope: scope, Turns: turns})
}

type preferenceRequest struct {
	InteractiveAdapter string `json:"interactive_adapter"`
}

func (a *API) handlePreference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req preferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if a.store == nil {
		http.Error(w, "store unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := a.store.SetPreference(session.UserID(r), strings.TrimSpace(req.InteractiveAdapter)); err != nil {
		http.Error(w, "set preference: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type clearRequest struct {
	ConversationID string `json:"conversation_id"`
}

func (a *API) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req clearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if a.store == nil {
		http.Error(w, "store unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := a.store.Clear(req.ConversationID); err != nil {
		http.Error(w, "clear: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(w, `{"error":"encode failed"}`)
	}
}
