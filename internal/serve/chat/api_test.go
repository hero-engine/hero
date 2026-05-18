package chat

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestAPI(t *testing.T) (*API, *captureBus, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	bus := &captureBus{}
	api := NewAPI(NewRegistry(), store, NewStreamer(bus), root)
	return api, bus, root
}

func TestCapabilityEndpoint(t *testing.T) {
	api, _, _ := newTestAPI(t)
	mux := http.NewServeMux()
	api.Mount(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/chat/capability", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var cap Capability
	if err := json.Unmarshal(w.Body.Bytes(), &cap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(cap.Adapters) != 0 {
		t.Errorf("expected 0 adapters, got %d", len(cap.Adapters))
	}
	if cap.Interactive != "" {
		t.Errorf("expected empty interactive, got %q", cap.Interactive)
	}
}

func TestTurnNoAdapterEmitsError(t *testing.T) {
	api, bus, _ := newTestAPI(t)
	mux := http.NewServeMux()
	api.Mount(mux)

	body, _ := json.Marshal(turnRequest{Prompt: "tell me about Y", PageScope: "global"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/turn", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var resp turnResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ConversationID == "" {
		t.Fatal("expected conversation_id")
	}
	if !strings.HasPrefix(resp.SSETopic, "chat.") {
		t.Errorf("sse_topic = %q", resp.SSETopic)
	}

	// dispatch is async; poll briefly for the error event.
	if !waitForEvent(bus, "chat.error", 2*time.Second) {
		t.Fatal("expected chat.error event for no-adapter")
	}
	events := bus.snapshot()
	var foundNoAdapter bool
	for _, ev := range events {
		if ev.Type == "chat.error" {
			if code, _ := ev.Payload["code"].(string); code == "no_adapter" {
				foundNoAdapter = true
			}
		}
	}
	if !foundNoAdapter {
		t.Errorf("expected error.code=no_adapter, events=%+v", events)
	}
}

func TestTurnAskSlashDispatchesInline(t *testing.T) {
	api, bus, _ := newTestAPI(t)
	mux := http.NewServeMux()
	api.Mount(mux)

	body, _ := json.Marshal(turnRequest{Prompt: "/ask what specs do we have", PageScope: "global"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/turn", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}

	// /ask is runner-free and runs synchronously inside handleTurn.
	events := bus.snapshot()
	var sawDone bool
	for _, ev := range events {
		if ev.Type == "chat.done" {
			sawDone = true
		}
	}
	if !sawDone {
		t.Errorf("expected chat.done, events=%+v", events)
	}
}

func TestHistoryRoundTrip(t *testing.T) {
	api, _, _ := newTestAPI(t)
	mux := http.NewServeMux()
	api.Mount(mux)

	// Post a /ask turn so a conversation gets created + persisted.
	body, _ := json.Marshal(turnRequest{Prompt: "/ask hi", PageScope: "global"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/turn", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Fetch history.
	req2 := httptest.NewRequest(http.MethodGet, "/api/chat/history?scope=global&limit=10", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("history status = %d", w2.Code)
	}
	var resp historyResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if resp.Scope != "global" {
		t.Errorf("scope = %q", resp.Scope)
	}
	if len(resp.Turns) < 1 {
		t.Errorf("expected at least 1 turn, got %d", len(resp.Turns))
	}
	if resp.Turns[0].Role != "user" {
		t.Errorf("first turn role = %q, want user", resp.Turns[0].Role)
	}
}

func TestPreferenceEndpoint(t *testing.T) {
	api, _, _ := newTestAPI(t)
	mux := http.NewServeMux()
	api.Mount(mux)

	body, _ := json.Marshal(preferenceRequest{InteractiveAdapter: "hero-code"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/preference", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}

func waitForEvent(bus *captureBus, evType string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, ev := range bus.snapshot() {
			if ev.Type == evType {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
