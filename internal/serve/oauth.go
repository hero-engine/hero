package serve

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuthConfig defines the OAuth provider configuration.
type OAuthConfig struct {
	Provider     string `json:"provider"`      // github or google
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"-"`
	Org          string `json:"org,omitempty"`           // GitHub org restriction
	HostedDomain string `json:"hosted_domain,omitempty"` // Google hosted domain restriction
}

// oauthStateEntry tracks a pending OAuth state parameter.
type oauthStateEntry struct {
	createdAt   time.Time
	redirectURI string
}

// OAuthHandler manages OAuth login and callback endpoints.
type OAuthHandler struct {
	cfg       *OAuthConfig
	jq        *JobQueue
	jwtSecret string
	states    sync.Map // map[string]*oauthStateEntry
}

// NewOAuthHandler creates a handler for OAuth login flows.
func NewOAuthHandler(cfg *OAuthConfig, jq *JobQueue, jwtSecret string) *OAuthHandler {
	h := &OAuthHandler{
		cfg:       cfg,
		jq:        jq,
		jwtSecret: jwtSecret,
	}
	// Clean expired states periodically.
	go h.cleanStates()
	return h
}

// generateState creates a random state parameter for CSRF protection.
func (h *OAuthHandler) generateState(redirectURI string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)
	h.states.Store(state, &oauthStateEntry{
		createdAt:   time.Now(),
		redirectURI: redirectURI,
	})
	return state, nil
}

// validateState checks and consumes a state parameter. Returns the
// redirect URI if valid.
func (h *OAuthHandler) validateState(state string) (string, bool) {
	v, ok := h.states.LoadAndDelete(state)
	if !ok {
		return "", false
	}
	entry := v.(*oauthStateEntry)
	if time.Since(entry.createdAt) > 10*time.Minute {
		return "", false
	}
	return entry.redirectURI, true
}

// cleanStates removes expired state entries every minute.
func (h *OAuthHandler) cleanStates() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.states.Range(func(key, value interface{}) bool {
			entry := value.(*oauthStateEntry)
			if time.Since(entry.createdAt) > 10*time.Minute {
				h.states.Delete(key)
			}
			return true
		})
	}
}

// HandleLogin redirects to the OAuth provider's consent screen.
func (h *OAuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		jsonError(w, "redirect_uri is required", http.StatusBadRequest)
		return
	}

	state, err := h.generateState(redirectURI)
	if err != nil {
		jsonError(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	var authURL string
	switch h.cfg.Provider {
	case "github":
		authURL = h.githubAuthURL(state, redirectURI)
	case "google":
		authURL = h.googleAuthURL(state, redirectURI)
	default:
		jsonError(w, "unsupported provider: "+h.cfg.Provider, http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleCallback processes the OAuth callback from the provider.
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		// Check for error from provider.
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			jsonError(w, fmt.Sprintf("OAuth error: %s — %s", errMsg, desc), http.StatusBadRequest)
			return
		}
		jsonError(w, "missing code or state parameter", http.StatusBadRequest)
		return
	}

	redirectURI, ok := h.validateState(state)
	if !ok {
		jsonError(w, "invalid or expired state parameter", http.StatusBadRequest)
		return
	}

	var (
		user *User
		err  error
	)
	switch h.cfg.Provider {
	case "github":
		user, err = h.githubCallback(code)
	case "google":
		user, err = h.googleCallback(code)
	default:
		jsonError(w, "unsupported provider", http.StatusBadRequest)
		return
	}
	if err != nil {
		jsonError(w, err.Error(), http.StatusForbidden)
		return
	}

	token, err := IssueJWT(user, h.jwtSecret, 30*24*time.Hour)
	if err != nil {
		jsonError(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	// Redirect to the CLI's localhost callback with the token.
	u, err := url.Parse(redirectURI)
	if err != nil {
		jsonError(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	q := u.Query()
	q.Set("token", token)
	q.Set("email", user.Email)
	q.Set("name", user.DisplayName)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// HandleConfig returns the OAuth configuration (without secrets).
func (h *OAuthHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"enabled":  true,
		"provider": h.cfg.Provider,
	})
}

// --- GitHub ---

func (h *OAuthHandler) githubAuthURL(state, redirectURI string) string {
	// The callback URL is the server's own callback endpoint. The
	// redirect_uri for the CLI is stored in the state entry.
	v := url.Values{}
	v.Set("client_id", h.cfg.ClientID)
	v.Set("scope", "read:user read:org")
	v.Set("state", state)
	return "https://github.com/login/oauth/authorize?" + v.Encode()
}

func (h *OAuthHandler) githubCallback(code string) (*User, error) {
	// Exchange code for access token.
	tokenResp, err := h.githubExchangeToken(code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Fetch user info.
	userInfo, err := h.githubFetchUser(tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Check org membership if restricted.
	if h.cfg.Org != "" {
		if err := h.githubCheckOrg(tokenResp, h.cfg.Org); err != nil {
			return nil, err
		}
	}

	oauthID := fmt.Sprintf("%v", userInfo["id"])
	email, _ := userInfo["email"].(string)
	name, _ := userInfo["name"].(string)
	if name == "" {
		name, _ = userInfo["login"].(string)
	}

	return h.jq.FindOrCreateOAuthUser("github", oauthID, email, name)
}

func (h *OAuthHandler) githubExchangeToken(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", h.cfg.ClientID)
	data.Set("client_secret", h.cfg.ClientSecret)
	data.Set("code", code)

	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}
	return result.AccessToken, nil
}

func (h *OAuthHandler) githubFetchUser(token string) (map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var user map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return user, nil
}

func (h *OAuthHandler) githubCheckOrg(token, org string) error {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/orgs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check org membership: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch orgs (status %d)", resp.StatusCode)
	}

	var orgs []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return fmt.Errorf("failed to decode orgs: %w", err)
	}

	for _, o := range orgs {
		login, _ := o["login"].(string)
		if strings.EqualFold(login, org) {
			return nil
		}
	}
	return fmt.Errorf("user is not a member of organization %q", org)
}

// --- Google ---

func (h *OAuthHandler) googleAuthURL(state, redirectURI string) string {
	v := url.Values{}
	v.Set("client_id", h.cfg.ClientID)
	v.Set("response_type", "code")
	v.Set("scope", "openid email profile")
	v.Set("state", state)
	// Use the server's own callback as the redirect_uri for Google.
	// The CLI redirect_uri is stored in the state entry.
	v.Set("redirect_uri", h.serverCallbackURL(redirectURI))
	if h.cfg.HostedDomain != "" {
		v.Set("hd", h.cfg.HostedDomain)
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + v.Encode()
}

// serverCallbackURL derives the server's callback URL from the CLI's
// redirect_uri. For Google OAuth we need a registered callback URL that
// points back at the server, not at localhost. The server's scheme+host
// is not known here, so we construct a relative path and let the caller
// configure the Google OAuth app with the right redirect URI.
//
// In practice the server operator registers
// https://<server>/auth/oauth/callback as the Google redirect URI. This
// helper is a no-op placeholder that returns a fixed path; the actual
// redirect_uri sent to Google must match what is registered in the
// Google Cloud Console. The CLI's localhost redirect_uri is stored in
// the state map and used after the callback completes.
func (h *OAuthHandler) serverCallbackURL(_ string) string {
	// This must match the redirect URI registered in Google Cloud Console.
	// The operator sets HERO_OAUTH_REDIRECT_URI if the default doesn't work.
	return oauthRedirectURI
}

// oauthRedirectURI is the server's OAuth callback URL. Set from
// HERO_OAUTH_REDIRECT_URI env or defaults to a relative path that works
// when the Google OAuth app is configured with the server's base URL.
var oauthRedirectURI = "/auth/oauth/callback"

func (h *OAuthHandler) googleCallback(code string) (*User, error) {
	token, err := h.googleExchangeToken(code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	userInfo, err := h.googleFetchUser(token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Check hosted domain restriction.
	if h.cfg.HostedDomain != "" {
		hd, _ := userInfo["hd"].(string)
		if !strings.EqualFold(hd, h.cfg.HostedDomain) {
			return nil, fmt.Errorf("user is not in hosted domain %q", h.cfg.HostedDomain)
		}
	}

	oauthID, _ := userInfo["sub"].(string)
	email, _ := userInfo["email"].(string)
	name, _ := userInfo["name"].(string)

	if oauthID == "" {
		return nil, fmt.Errorf("missing user ID from Google")
	}

	return h.jq.FindOrCreateOAuthUser("google", oauthID, email, name)
}

func (h *OAuthHandler) googleExchangeToken(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", h.cfg.ClientID)
	data.Set("client_secret", h.cfg.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", oauthRedirectURI)

	resp, err := http.DefaultClient.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}
	return result.AccessToken, nil
}

func (h *OAuthHandler) googleFetchUser(token string) (map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google userinfo returned %d: %s", resp.StatusCode, string(body))
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return info, nil
}

// RegisterOAuthAPI registers the OAuth endpoints on the mux.
func RegisterOAuthAPI(mux *http.ServeMux, jq *JobQueue, jwtSecret string, cfg *OAuthConfig) {
	h := NewOAuthHandler(cfg, jq, jwtSecret)

	mux.HandleFunc("/auth/oauth/login", h.HandleLogin)
	mux.HandleFunc("/auth/oauth/callback", h.HandleCallback)
	mux.HandleFunc("/auth/oauth/config", h.HandleConfig)
}
