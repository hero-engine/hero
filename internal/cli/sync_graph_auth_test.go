package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
)

// refreshHandler is the shared /api/v1/auth/refresh stub: hands back a
// fresh access token so the transport can retry.
func refreshHandler(newToken string, calls *int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*calls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  newToken,
			"refresh_token": "rotated-refresh",
			"expires_in":    3600,
		})
	}
}

// TestAuthTransport_RefreshesOn401AndRetries is the core AC #6 guard:
// a stale token yields 401, the transport refreshes once and replays the
// request with the new token, and the call succeeds.
func TestAuthTransport_RefreshesOn401AndRetries(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate saveCredentials from real ~/.hero

	var refreshCalls, protectedCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/refresh", refreshHandler("new-token", &refreshCalls))
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		protectedCalls++
		if r.Header.Get("Authorization") == "Bearer new-token" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	tr := newAuthTransport(&cloudCredentials{
		AccessToken: "stale-token", RefreshToken: "r", CloudURL: ts.URL,
	})
	resp, err := (&http.Client{Transport: tr}).Get(ts.URL + "/protected")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 after refresh", resp.StatusCode)
	}
	if refreshCalls != 1 {
		t.Errorf("refreshCalls = %d, want exactly 1", refreshCalls)
	}
	if protectedCalls != 2 {
		t.Errorf("protectedCalls = %d, want 2 (initial 401 + retry)", protectedCalls)
	}
	if tok := tr.currentToken(); tok != "new-token" {
		t.Errorf("token not rotated in transport: %q", tok)
	}
}

// TestAuthTransport_ReplaysPostBodyAfterRefresh ensures a POST (the push
// path) replays its body intact on the post-refresh retry — GetBody must
// rebuild the payload, not send an empty body.
func TestAuthTransport_ReplaysPostBodyAfterRefresh(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var refreshCalls int
	var gotBodies []string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/refresh", refreshHandler("new-token", &refreshCalls))
	mux.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBodies = append(gotBodies, string(b))
		if r.Header.Get("Authorization") == "Bearer new-token" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	tr := newAuthTransport(&cloudCredentials{
		AccessToken: "stale", RefreshToken: "r", CloudURL: ts.URL,
	})
	resp, err := (&http.Client{Transport: tr}).Post(
		ts.URL+"/push", "application/json", strings.NewReader(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if len(gotBodies) != 2 {
		t.Fatalf("server saw %d bodies, want 2", len(gotBodies))
	}
	for i, b := range gotBodies {
		if b != `{"hello":"world"}` {
			t.Errorf("attempt %d body = %q, want intact payload", i, b)
		}
	}
}

// TestAuthTransport_NoRefreshTokenSurfaces401 confirms the loop guard:
// without a refresh token there's nothing to refresh, so the 401 is
// surfaced after exactly one request (no infinite retry).
func TestAuthTransport_NoRefreshTokenSurfaces401(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	tr := newAuthTransport(&cloudCredentials{
		AccessToken: "stale", RefreshToken: "", CloudURL: ts.URL,
	})
	resp, err := (&http.Client{Transport: tr}).Get(ts.URL + "/x")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (refresh impossible)", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("server calls = %d, want 1 (no retry without refresh token)", calls)
	}
}

// TestLoadRefreshedCredentials_ProactiveRefreshWhenExpired covers the
// proactive half of AC #6: an already-expired token at rest is refreshed
// before any sync request is made.
func TestLoadRefreshedCredentials_ProactiveRefreshWhenExpired(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var refreshCalls int
	ts := httptest.NewServer(refreshHandler("fresh-token", &refreshCalls))
	defer ts.Close()

	if err := saveCredentials(&cloudCredentials{
		AccessToken:  "expired-token",
		RefreshToken: "r",
		CloudURL:     ts.URL,
		ExpiresAt:    time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	creds, err := loadRefreshedCredentials()
	if err != nil {
		t.Fatalf("loadRefreshedCredentials: %v", err)
	}
	if creds == nil || creds.AccessToken != "fresh-token" {
		t.Errorf("expected proactive refresh to fresh-token, got %+v", creds)
	}
	if refreshCalls != 1 {
		t.Errorf("refreshCalls = %d, want 1", refreshCalls)
	}
}

// TestLoadRefreshedCredentials_KeepsValidToken confirms a non-expired
// token is returned untouched (no needless refresh call).
func TestLoadRefreshedCredentials_KeepsValidToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var refreshCalls int
	ts := httptest.NewServer(refreshHandler("should-not-be-used", &refreshCalls))
	defer ts.Close()

	if err := saveCredentials(&cloudCredentials{
		AccessToken:  "good-token",
		RefreshToken: "r",
		CloudURL:     ts.URL,
		ExpiresAt:    time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	creds, err := loadRefreshedCredentials()
	if err != nil {
		t.Fatalf("loadRefreshedCredentials: %v", err)
	}
	if creds.AccessToken != "good-token" {
		t.Errorf("valid token should be kept, got %q", creds.AccessToken)
	}
	if refreshCalls != 0 {
		t.Errorf("refreshCalls = %d, want 0 (token still valid)", refreshCalls)
	}
}

// TestRunOpportunisticTeamSync_SkipsWhenUnreachable is the AC #5 guard:
// an unreachable server during scan-time sync degrades to a skipped step
// with a reason, never an error that would break the scan.
func TestRunOpportunisticTeamSync_SkipsWhenUnreachable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Stand a server up then close it so requests are refused.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := ts.URL
	ts.Close()

	if err := saveCredentials(&cloudCredentials{
		AccessToken: "t", RefreshToken: "r", CloudURL: url,
		ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	store, err := graph.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	defer store.Close()

	cfg := config.Config{Cloud: &config.CloudConfig{OrgID: "org1"}}
	res, err := runOpportunisticTeamSync(cfg, "repo1", store)
	if err != nil {
		t.Fatalf("opportunistic sync must not return an error, got: %v", err)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true on unreachable server, got %+v", res)
	}
	if !strings.Contains(res.Reason, "failed") {
		t.Errorf("reason should mention the failure, got %q", res.Reason)
	}
}

// TestAugmentAuthError adds the re-login hint only for 401-bearing errors.
func TestAugmentAuthError(t *testing.T) {
	hinted := augmentAuthError(fmt.Errorf("push: server 401: bad token"))
	if !strings.Contains(hinted.Error(), "hero login") {
		t.Errorf("401 error should get re-login hint, got: %v", hinted)
	}
	plain := augmentAuthError(fmt.Errorf("push: server 500: boom"))
	if strings.Contains(plain.Error(), "hero login") {
		t.Errorf("non-401 error must not get login hint, got: %v", plain)
	}
	if augmentAuthError(nil) != nil {
		t.Error("nil error should pass through as nil")
	}
}
