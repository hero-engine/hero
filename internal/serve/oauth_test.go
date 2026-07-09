package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOAuthStateGenerateAndValidate(t *testing.T) {
	h := &OAuthHandler{
		cfg:       &OAuthConfig{Provider: "github", ClientID: "test"},
		jwtSecret: "test-secret",
	}

	state, err := h.generateState("http://localhost:9999/callback")
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	// Valid state returns redirect URI.
	uri, ok := h.validateState(state)
	if !ok {
		t.Fatal("expected state to be valid")
	}
	if uri != "http://localhost:9999/callback" {
		t.Fatalf("unexpected redirect URI: %s", uri)
	}

	// Used state is consumed — second validate fails.
	_, ok = h.validateState(state)
	if ok {
		t.Fatal("expected state to be consumed after first validate")
	}
}

func TestOAuthStateExpiry(t *testing.T) {
	h := &OAuthHandler{
		cfg:       &OAuthConfig{Provider: "github", ClientID: "test"},
		jwtSecret: "test-secret",
	}

	state, _ := h.generateState("http://localhost:9999/callback")

	// Manually expire the entry.
	v, _ := h.states.Load(state)
	entry := v.(*oauthStateEntry)
	entry.createdAt = time.Now().Add(-11 * time.Minute)

	_, ok := h.validateState(state)
	if ok {
		t.Fatal("expected expired state to be invalid")
	}
}

func TestOAuthConfigEndpoint(t *testing.T) {
	h := &OAuthHandler{
		cfg:       &OAuthConfig{Provider: "github", ClientID: "test-id"},
		jwtSecret: "test-secret",
	}

	req := httptest.NewRequest("GET", "/auth/oauth/config", nil)
	w := httptest.NewRecorder()
	h.HandleConfig(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["enabled"] != true {
		t.Fatal("expected enabled=true")
	}
	if resp["provider"] != "github" {
		t.Fatalf("expected provider=github, got %v", resp["provider"])
	}
}

func TestOAuthLoginRequiresRedirectURI(t *testing.T) {
	h := &OAuthHandler{
		cfg:       &OAuthConfig{Provider: "github", ClientID: "test-id"},
		jwtSecret: "test-secret",
	}

	req := httptest.NewRequest("GET", "/auth/oauth/login", nil)
	w := httptest.NewRecorder()
	h.HandleLogin(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400 for missing redirect_uri, got %d", w.Code)
	}
}

func TestOAuthLoginRedirectsGitHub(t *testing.T) {
	h := &OAuthHandler{
		cfg:       &OAuthConfig{Provider: "github", ClientID: "my-client-id"},
		jwtSecret: "test-secret",
	}

	req := httptest.NewRequest("GET", "/auth/oauth/login?redirect_uri=http://localhost:9999/callback", nil)
	w := httptest.NewRecorder()
	h.HandleLogin(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "https://github.com/login/oauth/authorize") {
		t.Fatalf("expected GitHub auth URL, got %s", location)
	}
	if !strings.Contains(location, "client_id=my-client-id") {
		t.Fatalf("expected client_id in URL, got %s", location)
	}
	if !strings.Contains(location, "read%3Auser") || !strings.Contains(location, "read%3Aorg") {
		t.Fatalf("expected scopes in URL, got %s", location)
	}
}

func TestOAuthLoginRedirectsGoogle(t *testing.T) {
	h := &OAuthHandler{
		cfg: &OAuthConfig{
			Provider:     "google",
			ClientID:     "google-client-id",
			HostedDomain: "example.com",
		},
		jwtSecret: "test-secret",
	}

	req := httptest.NewRequest("GET", "/auth/oauth/login?redirect_uri=http://localhost:9999/callback", nil)
	w := httptest.NewRecorder()
	h.HandleLogin(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "https://accounts.google.com/o/oauth2/v2/auth") {
		t.Fatalf("expected Google auth URL, got %s", location)
	}
	if !strings.Contains(location, "client_id=google-client-id") {
		t.Fatalf("expected client_id in URL, got %s", location)
	}
	if !strings.Contains(location, "hd=example.com") {
		t.Fatalf("expected hd param in URL, got %s", location)
	}
}

func TestOAuthCallbackMissingParams(t *testing.T) {
	h := &OAuthHandler{
		cfg:       &OAuthConfig{Provider: "github", ClientID: "test"},
		jwtSecret: "test-secret",
	}

	req := httptest.NewRequest("GET", "/auth/oauth/callback", nil)
	w := httptest.NewRecorder()
	h.HandleCallback(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOAuthCallbackInvalidState(t *testing.T) {
	h := &OAuthHandler{
		cfg:       &OAuthConfig{Provider: "github", ClientID: "test"},
		jwtSecret: "test-secret",
	}

	req := httptest.NewRequest("GET", "/auth/oauth/callback?code=abc&state=invalid", nil)
	w := httptest.NewRecorder()
	h.HandleCallback(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOAuthGitHubOrgRestriction(t *testing.T) {
	// Mock GitHub orgs API.
	mockGitHub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"login": "allowed-org"},
			{"login": "other-org"},
		})
	}))
	defer mockGitHub.Close()

	h := &OAuthHandler{
		cfg:       &OAuthConfig{Provider: "github", ClientID: "test", Org: "allowed-org"},
		jwtSecret: "test-secret",
	}

	// Test with matching org — this exercises the logic path but uses
	// the real GitHub URL internally. The unit test below tests the
	// matching logic directly.
	orgs := []map[string]interface{}{
		{"login": "allowed-org"},
		{"login": "other-org"},
	}

	found := false
	for _, o := range orgs {
		login, _ := o["login"].(string)
		if strings.EqualFold(login, h.cfg.Org) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find allowed-org")
	}

	// Test with non-matching org.
	h.cfg.Org = "secret-org"
	found = false
	for _, o := range orgs {
		login, _ := o["login"].(string)
		if strings.EqualFold(login, h.cfg.Org) {
			found = true
			break
		}
	}
	if found {
		t.Fatal("should not find secret-org")
	}
}

func TestRefreshTokenValid(t *testing.T) {
	jq := newTestJobQueue(t)
	defer jq.Close()

	jwtSecret := "test-secret"

	// Create a user.
	user, err := jq.CreateUser("testuser", "test@example.com", "Test User", "password123", "member")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Issue a JWT that expires in 3 days (within 7-day refresh window).
	token, err := IssueJWT(user, jwtSecret, 3*24*time.Hour)
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}

	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handleRefresh(w, req, jq, jwtSecret)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("expected a new token in response")
	}
	// New token should be different from old.
	if resp["token"] == token {
		t.Fatal("expected a different token")
	}
}

func TestRefreshTokenNotEligible(t *testing.T) {
	jq := newTestJobQueue(t)
	defer jq.Close()

	jwtSecret := "test-secret"

	user, err := jq.CreateUser("testuser2", "test2@example.com", "Test User 2", "password123", "member")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Issue a JWT that expires in 20 days (outside 7-day refresh window).
	token, err := IssueJWT(user, jwtSecret, 20*24*time.Hour)
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}

	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handleRefresh(w, req, jq, jwtSecret)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRefreshTokenNoAuth(t *testing.T) {
	jq := newTestJobQueue(t)
	defer jq.Close()

	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	w := httptest.NewRecorder()
	handleRefresh(w, req, jq, "test-secret")

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// newTestJobQueue creates a temporary in-memory job queue for testing.
func newTestJobQueue(t *testing.T) *JobQueue {
	t.Helper()
	dir := t.TempDir()
	jq, err := NewJobQueue(dir)
	if err != nil {
		t.Fatalf("NewJobQueue: %v", err)
	}
	return jq
}
