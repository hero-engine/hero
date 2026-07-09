package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupSessionTestMux(t *testing.T) (*http.ServeMux, *JobQueue) {
	t.Helper()
	heroDir, _ := setupTestWorkspace(t)
	jq, err := NewJobQueue(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { jq.Close() })

	mux := http.NewServeMux()
	RegisterJobsAPI(mux, jq, nil)
	return mux, jq
}

func TestSessionsAPI_RegisterListUnregister(t *testing.T) {
	mux, _ := setupSessionTestMux(t)

	// 1. Register a session
	body := `{"id":"sess-1","user_id":"alice","agent":"claude","spec_slug":"csv-export","command":"deliver"}`
	req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /api/sessions: status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var regResp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &regResp)
	if regResp["status"] != "registered" || regResp["id"] != "sess-1" {
		t.Errorf("unexpected register response: %v", regResp)
	}

	// 2. List sessions — should contain the one we just registered
	req = httptest.NewRequest("GET", "/api/sessions", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/sessions: status = %d, want 200", rr.Code)
	}
	var sessions []map[string]string
	json.Unmarshal(rr.Body.Bytes(), &sessions)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0]["id"] != "sess-1" || sessions[0]["user_id"] != "alice" {
		t.Errorf("unexpected session: %v", sessions[0])
	}
	if sessions[0]["command"] != "deliver" || sessions[0]["spec_slug"] != "csv-export" {
		t.Errorf("unexpected session fields: %v", sessions[0])
	}

	// 3. Unregister the session
	req = httptest.NewRequest("DELETE", "/api/sessions/sess-1", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("DELETE /api/sessions/sess-1: status = %d, want 200", rr.Code)
	}

	// 4. List again — should be empty
	req = httptest.NewRequest("GET", "/api/sessions", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/sessions: status = %d, want 200", rr.Code)
	}
	var empty []map[string]string
	json.Unmarshal(rr.Body.Bytes(), &empty)
	if len(empty) != 0 {
		t.Errorf("expected 0 sessions after unregister, got %d", len(empty))
	}
}

func TestSessionsAPI_RegisterMissingID(t *testing.T) {
	mux, _ := setupSessionTestMux(t)

	body := `{"user_id":"alice"}`
	req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("POST without id: status = %d, want 400", rr.Code)
	}
}

func TestSessionsAPI_UserFromHeader(t *testing.T) {
	mux, _ := setupSessionTestMux(t)

	// Register without user_id in body — should fall back to X-Hero-User header
	body := `{"id":"sess-hdr"}`
	req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hero-User", "bob")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rr.Code)
	}

	// List and verify user_id came from header
	req = httptest.NewRequest("GET", "/api/sessions", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	var sessions []map[string]string
	json.Unmarshal(rr.Body.Bytes(), &sessions)
	if len(sessions) != 1 || sessions[0]["user_id"] != "bob" {
		t.Errorf("expected user_id=bob from header, got %v", sessions)
	}
}

func TestSessionsAPI_MethodNotAllowed(t *testing.T) {
	mux, _ := setupSessionTestMux(t)

	req := httptest.NewRequest("PUT", "/api/sessions", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT /api/sessions: status = %d, want 405", rr.Code)
	}
}
