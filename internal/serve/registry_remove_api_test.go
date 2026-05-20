package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

// newRemoveTestServer builds a Server wired with a real registry on
// disk (under t.TempDir() so each test gets a fresh ~/.hero/projects.json
// equivalent). Returns the server and the slug of the registered
// project.
func newRemoveTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	heroDir, projectRoot := setupTestWorkspace(t)
	regPath := filepath.Join(t.TempDir(), "projects.json")
	srv := NewServer(ServerConfig{
		HeroDir:      heroDir,
		ProjectRoot:  projectRoot,
		Version:      "test",
		Port:         0,
		AutoWatch:    false,
		RegistryPath: regPath,
	})
	// Register the primary project explicitly so the on-disk projects.json
	// reflects it before any remove fires.
	if srv.registry == nil {
		t.Fatalf("registry should be loaded for this test")
	}
	slug, err := srv.registry.Add(projectRoot)
	if err != nil {
		t.Fatalf("register project: %v", err)
	}
	if err := srv.registry.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}
	return srv, slug
}

func TestAPI_RegistryRemove_UndoBeforeDeadline(t *testing.T) {
	srv, slug := newRemoveTestServer(t)
	// Shrink the pending queue's grace window so the test doesn't sleep
	// for 5s. We swap the after func to a controllable channel.
	timerCh := make(chan time.Time, 1)
	srv.pendingRemove.after = func(d time.Duration) <-chan time.Time { return timerCh }

	rr := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/"+slug+"/registry/remove", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if !srv.pendingRemove.Pending(slug) {
		t.Fatalf("pending entry should be present after enqueue")
	}

	// Undo before deadline elapses.
	rr2 := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr2, httptest.NewRequest(http.MethodPost, "/api/"+slug+"/registry/remove/undo", nil))
	if rr2.Code != http.StatusOK {
		t.Fatalf("undo status = %d, want 200; body=%s", rr2.Code, rr2.Body.String())
	}

	// Fire the timer late — onCommit must NOT run because the entry was
	// cancelled.
	timerCh <- time.Now()
	time.Sleep(50 * time.Millisecond)

	if srv.GetProject(slug) == nil {
		t.Errorf("project %q removed despite undo within window", slug)
	}
	if !srv.registry.HasProject(slug) {
		t.Errorf("registry %q dropped despite undo within window", slug)
	}
}

func TestAPI_RegistryRemove_CommitsAfterDeadline(t *testing.T) {
	srv, slug := newRemoveTestServer(t)
	timerCh := make(chan time.Time, 1)
	srv.pendingRemove.after = func(d time.Duration) <-chan time.Time { return timerCh }

	rr := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/"+slug+"/registry/remove", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want 200", rr.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if _, ok := resp["deadline"]; !ok {
		t.Errorf("response missing deadline field: %+v", resp)
	}

	// Elapse the timer — onCommit must run, removing the project from
	// the in-memory map AND the on-disk registry.
	timerCh <- time.Now()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if srv.GetProject(slug) == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if srv.GetProject(slug) != nil {
		t.Fatalf("project %q still present after timer elapsed", slug)
	}
	if srv.registry.HasProject(slug) {
		t.Errorf("registry %q still present after timer elapsed", slug)
	}
}

func TestAPI_RegistryRemoveUndo_IdempotentWhenNothingPending(t *testing.T) {
	srv, slug := newRemoveTestServer(t)
	rr := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/"+slug+"/registry/remove/undo", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("undo with no pending returned %d, want 200", rr.Code)
	}
}

func TestAPI_RegistryRemove_GETNotAllowed(t *testing.T) {
	srv, slug := newRemoveTestServer(t)
	rr := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/"+slug+"/registry/remove", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET on remove returned %d, want 405", rr.Code)
	}
}

func TestAPI_DaemonOps_StopRejectsOtherVerbs(t *testing.T) {
	srv, _ := newRemoveTestServer(t)
	rr := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/daemon/ops/re-scan", nil))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("re-scan on daemon-ops returned %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}
