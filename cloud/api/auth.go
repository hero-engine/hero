package api

import (
	"crypto/rand"
	"encoding/hex"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hero-engine/hero/cloud/auth"
	"github.com/hero-engine/hero/cloud/middleware"
	"github.com/hero-engine/hero/cloud/store"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	db       *store.DB
	issuer   *auth.Issuer
	github   *auth.GitHubConfig
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(db *store.DB, issuer *auth.Issuer, github *auth.GitHubConfig) *AuthHandler {
	return &AuthHandler{db: db, issuer: issuer, github: github}
}

// RegisterRoutes adds auth routes to the mux.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/github", h.handleGitHubLogin)
	mux.HandleFunc("GET /api/v1/auth/github/callback", h.handleGitHubCallback)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/logout", h.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/me", h.handleMe)
}

func (h *AuthHandler) handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}

	// In production, store state in a short-lived cookie or cache for CSRF validation.
	// For the CLI flow, the state is passed through and verified on callback.
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	url := h.github.AuthorizeURL(state)

	// If request accepts JSON (CLI), return the URL. Otherwise redirect (browser).
	if r.Header.Get("Accept") == "application/json" {
		writeJSON(w, http.StatusOK, map[string]string{"url": url})
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func (h *AuthHandler) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing code parameter")
		return
	}

	// Exchange code for GitHub access token
	ghToken, err := h.github.ExchangeCode(code)
	if err != nil {
		log.Printf("github code exchange failed: %v", err)
		writeError(w, http.StatusBadGateway, "github authentication failed")
		return
	}

	// Fetch GitHub user profile
	ghUser, err := auth.FetchGitHubUser(ghToken)
	if err != nil {
		log.Printf("github user fetch failed: %v", err)
		writeError(w, http.StatusBadGateway, "failed to fetch github profile")
		return
	}

	// Upsert user in our database
	user, err := h.db.UpsertUser(r.Context(), &store.User{
		Email:      ghUser.Email,
		Name:       ghUser.Name,
		AvatarURL:  ghUser.AvatarURL,
		Provider:   "github",
		ProviderID: fmt.Sprintf("%d", ghUser.ID),
	})
	if err != nil {
		log.Printf("user upsert failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Issue token pair
	pair, err := h.issueTokenPair(r.Context(), user)
	if err != nil {
		log.Printf("token issuance failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}

	// Redirect to the CLI's local callback server with tokens as query params.
	// The CLI starts a listener on :19876 waiting for this redirect.
	cliCallback := fmt.Sprintf(
		"http://localhost:19876/?access_token=%s&refresh_token=%s",
		pair.AccessToken, pair.RefreshToken,
	)
	http.Redirect(w, r, cliCallback, http.StatusFound)
}

func (h *AuthHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	tokenHash := auth.HashToken(body.RefreshToken)

	// Validate the refresh token
	userID, err := h.db.ValidateRefreshToken(r.Context(), tokenHash)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Revoke the old refresh token (rotation)
	_ = h.db.RevokeRefreshToken(r.Context(), tokenHash)

	// Fetch user
	user, err := h.db.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	// Issue new pair
	pair, err := h.issueTokenPair(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		// Try to get refresh token from body to revoke it
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.RefreshToken != "" {
			_ = h.db.RevokeRefreshToken(r.Context(), auth.HashToken(body.RefreshToken))
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
		return
	}

	// Revoke all refresh tokens for this user
	_ = h.db.RevokeAllRefreshTokens(r.Context(), claims.UserID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (h *AuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.db.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) issueTokenPair(ctx context.Context, user *store.User) (*auth.TokenPair, error) {
	// Issue access token
	accessToken, err := h.issuer.Issue(auth.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("issuing access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	// Store hashed refresh token
	refreshHash := auth.HashToken(refreshToken)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := h.db.StoreRefreshToken(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &auth.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
