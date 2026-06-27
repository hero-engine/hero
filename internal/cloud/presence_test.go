package cloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPresenceReporterRegisterAndUnregister(t *testing.T) {
	var registered atomic.Bool
	var unregistered atomic.Bool
	var mu sync.Mutex
	var capturedBody registerRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/orgs/org-1/sessions":
			registered.Store(true)
			mu.Lock()
			json.NewDecoder(r.Body).Decode(&capturedBody)
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "sess-123"})
		case r.Method == "DELETE" && r.URL.Path == "/api/v1/orgs/org-1/sessions/sess-123":
			unregistered.Store(true)
			w.WriteHeader(http.StatusOK)
		case r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg := Config{
		CloudURL: srv.URL,
		Token:    "test-token",
		OrgID:    "org-1",
		RepoID:   "repo-1",
	}
	reporter := NewPresenceReporter(cfg, NewAuthenticatedClient("test-token"))
	reporter.Start("auth-flow", "deliver")
	time.Sleep(100 * time.Millisecond)
	reporter.Stop()

	if !registered.Load() {
		t.Error("expected register call")
	}
	if !unregistered.Load() {
		t.Error("expected unregister call")
	}

	mu.Lock()
	defer mu.Unlock()
	if capturedBody.RepoID != "repo-1" {
		t.Errorf("expected repo_id repo-1, got %s", capturedBody.RepoID)
	}
	if capturedBody.SpecSlug != "auth-flow" {
		t.Errorf("expected spec_slug auth-flow, got %s", capturedBody.SpecSlug)
	}
	if capturedBody.Command != "deliver" {
		t.Errorf("expected command deliver, got %s", capturedBody.Command)
	}
}

func TestPresenceReporterHeartbeat(t *testing.T) {
	var heartbeatCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "sess-456"})
		case r.Method == "PUT":
			heartbeatCount.Add(1)
			w.WriteHeader(http.StatusOK)
		case r.Method == "DELETE":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	cfg := Config{
		CloudURL: srv.URL,
		Token:    "test-token",
		OrgID:    "org-1",
		RepoID:   "repo-1",
	}
	reporter := NewPresenceReporter(cfg, NewAuthenticatedClient("test-token"))

	// Override heartbeat interval for testing isn't possible with const,
	// so we just verify at least one heartbeat fires within 35s.
	reporter.Start("", "serve")
	time.Sleep(32 * time.Second)
	reporter.Stop()

	count := heartbeatCount.Load()
	if count < 1 {
		t.Errorf("expected at least 1 heartbeat, got %d", count)
	}
}

func TestPresenceReporterBearerToken(t *testing.T) {
	var capturedAuth string
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		capturedAuth = r.Header.Get("Authorization")
		mu.Unlock()
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "sess-789"})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	cfg := Config{
		CloudURL: srv.URL,
		Token:    "my-secret",
		OrgID:    "org-1",
		RepoID:   "repo-1",
	}
	reporter := NewPresenceReporter(cfg, NewAuthenticatedClient("my-secret"))
	reporter.Start("", "serve")
	time.Sleep(100 * time.Millisecond)
	reporter.Stop()

	mu.Lock()
	got := capturedAuth
	mu.Unlock()

	if got != "Bearer my-secret" {
		t.Errorf("expected Bearer my-secret, got %q", got)
	}
}

func TestPresenceReporterRegisterFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := Config{
		CloudURL: srv.URL,
		Token:    "test",
		OrgID:    "org-1",
		RepoID:   "repo-1",
	}
	reporter := NewPresenceReporter(cfg, NewAuthenticatedClient("test"))
	// Should not panic on registration failure.
	reporter.Start("", "serve")
	time.Sleep(100 * time.Millisecond)
	reporter.Stop()
}
