package api

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"

	"github.com/hero-engine/hero/cloud/middleware"
	"github.com/hero-engine/hero/cloud/store"
)

// OrgHandler handles organization endpoints.
type OrgHandler struct {
	db *store.DB
}

// NewOrgHandler creates a new org handler.
func NewOrgHandler(db *store.DB) *OrgHandler {
	return &OrgHandler{db: db}
}

// RegisterRoutes adds org routes to the mux. These require authentication.
func (h *OrgHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/orgs", h.handleListOrgs)
	mux.HandleFunc("POST /api/v1/orgs", h.handleCreateOrg)
	mux.HandleFunc("GET /api/v1/orgs/{org_id}", h.handleGetOrg)
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/members", h.handleListMembers)
	mux.HandleFunc("POST /api/v1/orgs/{org_id}/members", h.handleAddMember)
	mux.HandleFunc("DELETE /api/v1/orgs/{org_id}/members/{user_id}", h.handleRemoveMember)
}

func (h *OrgHandler) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	orgs, err := h.db.ListUserOrgs(r.Context(), claims.UserID)
	if err != nil {
		log.Printf("list orgs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list orgs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"orgs":  orgs,
		"count": len(orgs),
	})
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

func (h *OrgHandler) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" || body.Slug == "" {
		writeError(w, http.StatusBadRequest, "name and slug are required")
		return
	}
	if !slugPattern.MatchString(body.Slug) {
		writeError(w, http.StatusBadRequest, "slug must be 3-40 lowercase alphanumeric characters or hyphens")
		return
	}

	// Check slug uniqueness
	existing, err := h.db.GetOrgBySlug(r.Context(), body.Slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check slug")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "slug already taken")
		return
	}

	org, err := h.db.CreateOrg(r.Context(), body.Name, body.Slug, claims.UserID)
	if err != nil {
		log.Printf("create org: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create org")
		return
	}

	writeJSON(w, http.StatusCreated, org)
}

func (h *OrgHandler) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	orgID := r.PathValue("org_id")
	org, err := h.db.GetOrgByID(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get org")
		return
	}
	if org == nil {
		writeError(w, http.StatusNotFound, "org not found")
		return
	}

	// Check membership
	role, err := h.db.GetMemberRole(r.Context(), orgID, claims.UserID)
	if err != nil || role == "" {
		writeError(w, http.StatusForbidden, "not a member of this org")
		return
	}

	writeJSON(w, http.StatusOK, org)
}

func (h *OrgHandler) handleListMembers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	orgID := r.PathValue("org_id")

	// Check membership
	role, err := h.db.GetMemberRole(r.Context(), orgID, claims.UserID)
	if err != nil || role == "" {
		writeError(w, http.StatusForbidden, "not a member of this org")
		return
	}

	members, err := h.db.ListOrgMembers(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list members")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"members": members,
		"count":   len(members),
	})
}

func (h *OrgHandler) handleAddMember(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	orgID := r.PathValue("org_id")

	// Only owner/admin can add members
	role, err := h.db.GetMemberRole(r.Context(), orgID, claims.UserID)
	if err != nil || (role != "owner" && role != "admin") {
		writeError(w, http.StatusForbidden, "only owners and admins can add members")
		return
	}

	var body struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if body.Role == "" {
		body.Role = "member"
	}
	// Validate role
	switch body.Role {
	case "admin", "member", "viewer":
		// ok
	case "owner":
		if role != "owner" {
			writeError(w, http.StatusForbidden, "only owners can add other owners")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "role must be owner, admin, member, or viewer")
		return
	}

	if err := h.db.AddOrgMember(r.Context(), orgID, body.UserID, body.Role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"status": "added",
		"org_id": orgID,
		"user_id": body.UserID,
		"role":   body.Role,
	})
}

func (h *OrgHandler) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	orgID := r.PathValue("org_id")
	targetUserID := r.PathValue("user_id")

	// Can remove yourself, or owner/admin can remove others
	if targetUserID != claims.UserID {
		role, err := h.db.GetMemberRole(r.Context(), orgID, claims.UserID)
		if err != nil || (role != "owner" && role != "admin") {
			writeError(w, http.StatusForbidden, "only owners and admins can remove members")
			return
		}
	}

	if err := h.db.RemoveOrgMember(r.Context(), orgID, targetUserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "removed",
		"org_id":  orgID,
		"user_id": targetUserID,
	})
}
