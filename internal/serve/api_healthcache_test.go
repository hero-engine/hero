package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/serve/healthcache"
)

// fakeDispatcher stubs the opsrunner for healthcache integration tests.
// Writes the on-disk health.json when Start is called so the cache's
// downstream read-from-disk path produces a meaningful result.
type fakeDispatcher struct {
	startCalls int
	projectRoot string
	rows       []healthcache.HealthRow
}

func (f *fakeDispatcher) Start(_ context.Context, slug, projectRoot, _ string) (string, bool, error) {
	f.startCalls++
	f.projectRoot = projectRoot
	cacheDir := filepath.Join(projectRoot, ".hero", "cache")
	_ = os.MkdirAll(cacheDir, 0o755)
	doc := map[string]interface{}{
		"captured_at": time.Now().UTC(),
		"rows":        f.rows,
	}
	body, _ := json.Marshal(doc)
	_ = os.WriteFile(filepath.Join(cacheDir, "health.json"), body, 0o644)
	return "job-" + slug, true, nil
}

func (f *fakeDispatcher) Wait(_ context.Context, _, _ string) (int, error) { return 0, nil }

func TestAPI_HealthGet_EmptyCache(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/health", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["slug"] != slug {
		t.Errorf("slug = %v want %s", body["slug"], slug)
	}
	if rows, _ := body["rows"].([]interface{}); len(rows) != 0 {
		t.Errorf("expected empty rows on cold cache, got %v", rows)
	}
	if body["from_disk"] != false {
		t.Errorf("from_disk = %v want false", body["from_disk"])
	}
}

func TestAPI_HealthGet_PopulatedCache(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	// Bypass the API surface and populate the cache directly via the
	// dispatcher integration. RefreshHealth uses the dispatcher to
	// write the on-disk artifact + populate the cache atomically.
	disp := &fakeDispatcher{rows: []healthcache.HealthRow{
		{Name: "stale-specs", Status: "pass", Message: "none"},
	}}
	// Swap the cache's dispatcher and refresh manually so we don't have to
	// drive a subprocess. RefreshHealth fills in-process cache state.
	cache := healthcache.New(time.Minute, healthcache.Options{Ops: disp})
	api.healthCache = cache
	pc := api.server.GetProject(slug)
	if pc == nil {
		t.Fatal("no project")
	}
	if _, err := cache.RefreshHealth(context.Background(), slug, pc.Path); err != nil {
		t.Fatalf("RefreshHealth: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/health", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	rows, _ := body["rows"].([]interface{})
	if len(rows) != 1 {
		t.Errorf("rows = %v want 1", rows)
	}
	if body["from_disk"] != true {
		t.Errorf("expected from_disk true after refresh; got %v", body["from_disk"])
	}
	if age, ok := body["age_seconds"].(float64); !ok || age < 0 {
		t.Errorf("expected age_seconds >= 0, got %v", body["age_seconds"])
	}
}

func TestAPI_PeerProbe(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	// The default prober just stat's .hero/peer-manifest.yaml on the
	// peer path. We don't need the path to actually exist — the
	// response just needs to round-trip and update the cache.
	req := httptest.NewRequest(http.MethodPost, "/api/"+slug+"/peers/ghost/probe", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["alias"] != "ghost" {
		t.Errorf("alias = %v want ghost", body["alias"])
	}
	if body["reachable"] != false {
		t.Errorf("expected unreachable for unconfigured peer; got %v", body["reachable"])
	}
	// Cache entry must now exist.
	if got, ok := api.healthCache.Peer(slug, "ghost"); !ok {
		t.Error("expected cache entry after probe")
	} else if got.Reachable {
		t.Error("expected unreachable cache entry")
	}
}

func TestAPI_PeerProbe_WrongMethod(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/peers/ghost/probe", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d want 405", rr.Code)
	}
}
