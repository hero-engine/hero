package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/hero-engine/hero/cloud/middleware"
	"github.com/hero-engine/hero/cloud/store"
)

// RepoHandler handles repo endpoints.
type RepoHandler struct {
	db *store.DB
}

// NewRepoHandler creates a new repo handler.
func NewRepoHandler(db *store.DB) *RepoHandler {
	return &RepoHandler{db: db}
}

// RegisterRoutes adds repo routes to the mux.
func (h *RepoHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/repos", h.handleListRepos)
	mux.HandleFunc("POST /api/v1/orgs/{org_id}/repos", h.handleCreateRepo)
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/repos/{repo_id}", h.handleGetRepo)
}

func (h *RepoHandler) requireOrgMembership(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return "", "", false
	}

	orgID := r.PathValue("org_id")
	role, err := h.db.GetMemberRole(r.Context(), orgID, claims.UserID)
	if err != nil || role == "" {
		writeError(w, http.StatusForbidden, "not a member of this org")
		return "", "", false
	}
	return orgID, claims.UserID, true
}

func (h *RepoHandler) handleListRepos(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	repos, err := h.db.ListOrgRepos(r.Context(), orgID)
	if err != nil {
		log.Printf("list repos: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list repos")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"repos": repos,
		"count": len(repos),
	})
}

func (h *RepoHandler) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	var body struct {
		Name    string `json:"name"`
		PushURL string `json:"push_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	repo, err := h.db.CreateRepo(r.Context(), orgID, body.Name, body.PushURL)
	if err != nil {
		log.Printf("create repo: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create repo")
		return
	}

	writeJSON(w, http.StatusCreated, repo)
}

func (h *RepoHandler) handleGetRepo(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	repoID := r.PathValue("repo_id")
	repo, err := h.db.GetRepoByID(r.Context(), orgID, repoID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get repo")
		return
	}
	if repo == nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}

	writeJSON(w, http.StatusOK, repo)
}
