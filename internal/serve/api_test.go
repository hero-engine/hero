package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupAPITestEnv(t *testing.T) *API {
	t.Helper()
	heroDir, projectRoot := setupTestWorkspace(t) // reuses the helper from mcp_test.go
	bus := NewEventBus()
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        0,
		AutoWatch:   false,
	})
	srv.bus = bus
	return srv.api
}

func TestAPI_Health(t *testing.T) {
	api := setupAPITestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestAPI_Health_MethodNotAllowed(t *testing.T) {
	api := setupAPITestEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestAPI_Projects(t *testing.T) {
	api := setupAPITestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)
	count := body["count"].(float64)
	if count < 1 {
		t.Errorf("count = %v, want >= 1", count)
	}
}

func setupAPIWithSlug(t *testing.T) (*API, string) {
	t.Helper()
	heroDir, projectRoot := setupTestWorkspace(t)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        0,
		AutoWatch:   false,
	})
	slugs := srv.Projects()
	if len(slugs) == 0 {
		t.Fatal("no projects")
	}
	return srv.api, slugs[0]
}

func TestAPI_Status(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/status", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)

	total := body["total"].(float64)
	if total < 1 {
		t.Errorf("total = %v, want >= 1", total)
	}
	if body["project"] != slug {
		t.Errorf("project = %v, want %s", body["project"], slug)
	}
}

func TestAPI_Specs_List(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/specs", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)

	count := body["count"].(float64)
	if count < 1 {
		t.Errorf("count = %v, want >= 1", count)
	}
}

func TestAPI_Specs_ListFiltered(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/specs?type=feature", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)

	specs := body["specs"].([]interface{})
	for _, s := range specs {
		sp := s.(map[string]interface{})
		if sp["type"] != "feature" {
			t.Errorf("expected type=feature, got %v", sp["type"])
		}
	}
}

func TestAPI_Specs_BySlug(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/specs/auth-login", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)

	if body["slug"] != "auth-login" {
		t.Errorf("slug = %v, want auth-login", body["slug"])
	}
	content := body["content"].(string)
	if !strings.Contains(content, "Auth Login") {
		t.Errorf("content should contain 'Auth Login'")
	}
}

func TestAPI_Specs_BySlug_NotFound(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/specs/nonexistent", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestAPI_Search(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/search?q=login", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)

	count := body["count"].(float64)
	if count < 1 {
		t.Errorf("count = %v, want >= 1", count)
	}
}

func TestAPI_Search_MissingQuery(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/search", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestAPI_Context(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/context?files=src/auth/login.go", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

func TestAPI_Context_MissingFiles(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/context", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestAPI_Check(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/check", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)

	if _, ok := body["stats"]; !ok {
		t.Error("expected stats in response")
	}
	if _, ok := body["stale_days"]; !ok {
		t.Error("expected stale_days in response")
	}
	if body["project"] != slug {
		t.Errorf("project = %v, want %s", body["project"], slug)
	}
}

func TestAPI_Knowledge(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/knowledge", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)

	if _, ok := body["entries"]; !ok {
		t.Error("expected entries in response")
	}
}

func TestAPI_Knowledge_TypeFilter(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/knowledge?type=convention", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestAPI_CORS(t *testing.T) {
	api := setupAPITestEnv(t)
	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header")
	}
}

func TestAPI_ProjectNotFound(t *testing.T) {
	api := setupAPITestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent-project/status", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestAPI_Inventory(t *testing.T) {
	api, slug := setupAPIWithSlug(t)
	req := httptest.NewRequest(http.MethodGet, "/api/"+slug+"/inventory", nil)
	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)

	if _, ok := body["bugs"]; !ok {
		t.Error("expected 'bugs' key in response")
	}
	if _, ok := body["count"]; !ok {
		t.Error("expected 'count' key in response")
	}
}

// The v1 dashboard at `/` has been replaced by the shell router (see
// internal/serve/shell). The API handler no longer serves HTML at `/`;
// it returns 404 there. Shell behavior is covered by
// internal/serve/shell/shell_test.go.
func TestAPI_RootNotHandledByAPI(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        0,
		AutoWatch:   false,
		UIEnabled:   true,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	srv.api.Handler().ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Fatalf("expected non-200 from API handler at /, got %d (shell composes /)", rr.Code)
	}
}
