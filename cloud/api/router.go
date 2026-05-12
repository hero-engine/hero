// Package api provides the HTTP router and handlers for Hero Cloud.
package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/cloud"
	"github.com/hero-engine/hero/cloud/auth"
	gh "github.com/hero-engine/hero/cloud/github"
	"github.com/hero-engine/hero/cloud/middleware"
	"github.com/hero-engine/hero/cloud/store"
)

// NewRouter builds the HTTP handler with all routes.
func NewRouter(db *store.DB, version string) http.Handler {
	mux := http.NewServeMux()

	// SSE hub for real-time events
	sseHub := NewSSEHub()

	// JWT issuer
	secret := os.Getenv("HERO_JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	issuer := auth.NewIssuer(secret, 1*time.Hour, 30*24*time.Hour)

	// GitHub OAuth config
	ghConfig := &auth.GitHubConfig{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
	}

	// Auth middleware
	authMiddleware := middleware.Auth(issuer)

	// GitHub App (optional — only if configured)
	var ghApp *gh.App
	if appIDStr := os.Getenv("GITHUB_APP_ID"); appIDStr != "" {
		appID, _ := strconv.ParseInt(appIDStr, 10, 64)
		keyPEM := os.Getenv("GITHUB_APP_PRIVATE_KEY")
		if keyPEM != "" {
			if key, err := gh.ParsePrivateKey([]byte(keyPEM)); err == nil {
				ghApp = gh.NewApp(&gh.AppConfig{
					AppID:         appID,
					PrivateKey:    key,
					WebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
				})
			}
		}
	}

	// Governance handler (GitHub App webhook + config)
	if ghApp != nil {
		govHandler := NewGovernanceHandler(db, ghApp)
		govHandler.RegisterRoutes(mux)
	}

	// Health / info (unauthenticated)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": version,
		})
	})

	mux.HandleFunc("GET /api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"api":     "hero-cloud",
			"version": "v1",
		})
	})

	// Auth routes (mostly unauthenticated)
	authHandler := NewAuthHandler(db, issuer, ghConfig)
	authHandler.RegisterRoutes(mux)

	// Authenticated routes
	authedMux := http.NewServeMux()

	orgHandler := NewOrgHandler(db)
	orgHandler.RegisterRoutes(authedMux)

	repoHandler := NewRepoHandler(db)
	repoHandler.RegisterRoutes(authedMux)

	specHandler := NewSpecHandler(db, sseHub)
	specHandler.RegisterRoutes(authedMux)

	// Knowledge-graph federation endpoints (phase 7).
	graphHandler := NewGraphHandler(db)
	graphHandler.RegisterRoutes(authedMux)

	dashHandler := NewDashboardHandler(db)
	dashHandler.RegisterRoutes(authedMux)

	// SSE endpoint (authenticated)
	authedMux.HandleFunc("GET /api/v1/orgs/{org_id}/events", handleSSE(sseHub, db))

	// Governance config (authenticated)
	if ghApp != nil {
		govHandler := NewGovernanceHandler(db, ghApp)
		govHandler.sseHub = sseHub
		govHandler.RegisterAuthenticatedRoutes(authedMux)
	}

	// Mount all authenticated routes under the auth middleware
	mux.Handle("GET /api/v1/orgs", authMiddleware(authedMux))
	mux.Handle("POST /api/v1/orgs", authMiddleware(authedMux))
	mux.Handle("GET /api/v1/orgs/", authMiddleware(authedMux))
	mux.Handle("POST /api/v1/orgs/", authMiddleware(authedMux))
	mux.Handle("DELETE /api/v1/orgs/", authMiddleware(authedMux))

	// Dashboard SPA (catch-all, after API routes)
	mux.Handle("/", serveSPA(cloud.WebAssets))

	return mux
}

func serveSPA(assets embed.FS) http.Handler {
	// Strip "web/" prefix so files are served from root
	sub, _ := fs.Sub(assets, "web")
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try serving the static file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		// Check if file exists in embedded FS
		f, err := sub.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			// SPA fallback: serve index.html for all non-file routes
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
