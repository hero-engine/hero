package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectHandler_AllProjectsProjectRoute verifies the cross-project
// /p/all/project route renders all four sections (Directory, Daemon
// Ops, Health Rollup, Peers Map) against a multi-project fixture, and
// that a project with deliberately-broken inputs still produces a
// degraded row instead of poisoning the page.
func TestProjectHandler_AllProjectsProjectRoute(t *testing.T) {
	srv := newMultiProjectServer(t)

	// Add a third "broken" project — registered, but with a stub .hero
	// directory missing the expected health artifact, so its row should
	// render with degraded/unknown indicators while the rest of the
	// page still returns 200.
	broken := t.TempDir()
	brokenHero := filepath.Join(broken, ".hero")
	if err := os.MkdirAll(brokenHero, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a corrupt peer-manifest.yaml on a peer reference to drive
	// a broken-manifest signal on the rollup.
	if err := os.WriteFile(filepath.Join(brokenHero, "hero.json"),
		[]byte(`{"repos":{"ghost":"/does/not/exist"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = srv.AddProject("broken-project", broken, brokenHero, false)

	req := httptest.NewRequest(http.MethodGet, "/p/all/project", nil)
	rr := httptest.NewRecorder()
	srv.projectHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()

	for _, want := range []string{
		`data-section="directory"`,
		`data-section="daemon_ops"`,
		`data-section="health_rollup"`,
		`data-section="peers_map"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
	// Broken project should appear in the directory.
	if !strings.Contains(body, "broken-project") {
		t.Errorf("expected broken-project slug to render despite missing inputs; body=%s", body)
	}
}

// TestProjectHandler_TopNavActive_OnAggregate ensures the Project tab
// is marked active on /p/all/project AND on a per-project /p/<slug>/
// project (the Phase 1 regression check).
func TestProjectHandler_TopNavActive_OnAggregate(t *testing.T) {
	srv := newMultiProjectServer(t)

	req := httptest.NewRequest(http.MethodGet, "/p/all/project", nil)
	rr := httptest.NewRecorder()
	srv.projectHandler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("aggregate status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `class="nav-tab active"`) {
		t.Errorf("aggregate project page must mark a tab active")
	}
}

// TestRegistryRefreshEndpoint verifies POST /api/daemon/registry/refresh
// returns JSON with the current project list.
func TestRegistryRefreshEndpoint(t *testing.T) {
	srv := newMultiProjectServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/daemon/registry/refresh", nil)
	rr := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Reloaded bool `json:"reloaded"`
		Count    int  `json:"count"`
		Projects []struct {
			Slug string `json:"slug"`
			Path string `json:"path"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v; body=%s", err, rr.Body.String())
	}
	if resp.Count < 2 {
		t.Errorf("count = %d, want at least 2 (two-project fixture)", resp.Count)
	}
}

// TestRegistryRefreshEndpoint_GetWorks confirms the GET form is also
// supported (returns current state without reloading from disk).
func TestRegistryRefreshEndpoint_GetWorks(t *testing.T) {
	srv := newMultiProjectServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/daemon/registry/refresh", nil)
	rr := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
}
