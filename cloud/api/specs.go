package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/cloud/store"
)

// SpecHandler handles spec and sync endpoints.
type SpecHandler struct {
	db     *store.DB
	sseHub *SSEHub
}

// NewSpecHandler creates a new spec handler.
func NewSpecHandler(db *store.DB, sseHub *SSEHub) *SpecHandler {
	return &SpecHandler{db: db, sseHub: sseHub}
}

// RegisterRoutes adds spec routes to the mux. All routes are wrapped
// with withOrg so handlers receive an RLS-bound r.Context().
func (h *SpecHandler) RegisterRoutes(mux *http.ServeMux) {
	wrap := func(handler http.HandlerFunc) http.HandlerFunc { return withOrg(h.db, handler) }
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/repos/{repo_id}/specs", wrap(h.handleListSpecs))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/repos/{repo_id}/specs/{slug}", wrap(h.handleGetSpec))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/repos/{repo_id}/subprojects", wrap(h.handleListSubprojects))
	mux.HandleFunc("POST /api/v1/orgs/{org_id}/repos/{repo_id}/sync", wrap(h.handleSync))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/search", wrap(h.handleSearch))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/activity", wrap(h.handleActivity))
}

// handleListSubprojects returns the distinct subproject scopes for a repo.
// Used by the dashboard's filter dropdown.
func (h *SpecHandler) handleListSubprojects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")
	repoID := r.PathValue("repo_id")
	repo, err := h.db.GetRepoByID(ctx, orgID, repoID)
	if err != nil || repo == nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}
	subs, err := h.db.ListRepoSubprojects(ctx, repoID)
	if err != nil {
		log.Printf("list subprojects: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list subprojects")
		return
	}
	if subs == nil {
		subs = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"subprojects": subs,
		"count":       len(subs),
	})
}

func (h *SpecHandler) handleListSpecs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")
	repoID := r.PathValue("repo_id")
	repo, err := h.db.GetRepoByID(ctx, orgID, repoID)
	if err != nil || repo == nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}

	specType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	subproject := r.URL.Query().Get("subproject")

	specs, err := h.db.ListRepoSpecs(ctx, repoID, specType, status, subproject)
	if err != nil {
		log.Printf("list specs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list specs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"specs": specs,
		"count": len(specs),
	})
}

func (h *SpecHandler) handleGetSpec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")
	repoID := r.PathValue("repo_id")
	repo, err := h.db.GetRepoByID(ctx, orgID, repoID)
	if err != nil || repo == nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}

	slug := r.PathValue("slug")
	spec, err := h.db.GetSpec(ctx, repoID, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get spec")
		return
	}
	if spec == nil {
		writeError(w, http.StatusNotFound, "spec not found")
		return
	}

	writeJSON(w, http.StatusOK, spec)
}

// SyncRequest is the payload for the sync endpoint.
type SyncRequest struct {
	Specs []store.Spec `json:"specs"`
}

func (h *SpecHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")
	userID := UserIDFromContext(ctx)
	repoID := r.PathValue("repo_id")
	repo, err := h.db.GetRepoByID(ctx, orgID, repoID)
	if err != nil || repo == nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}

	var body SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var synced int
	for _, s := range body.Specs {
		s.RepoID = repoID
		if _, err := h.db.UpsertSpec(ctx, &s); err != nil {
			log.Printf("sync spec %s: %v", s.Slug, err)
			continue
		}
		synced++

		_ = h.db.RecordAudit(ctx, store.AuditEntry{
			OrgID:     orgID,
			RepoID:    &repoID,
			UserID:    &userID,
			EventType: store.EventSpecSynced,
			Payload: map[string]interface{}{
				"slug":   s.Slug,
				"type":   s.Type,
				"status": s.Status,
			},
		})
	}

	_ = h.db.TouchRepoSyncTime(ctx, repoID)

	payload, _ := json.Marshal(map[string]interface{}{
		"synced_count": synced,
		"total_count":  len(body.Specs),
	})
	_ = h.db.RecordEvent(ctx, orgID, &repoID, &userID, "sync", payload)

	// Broadcast SSE event
	if h.sseHub != nil {
		h.sseHub.Broadcast(orgID, SSEEvent{
			Type: "sync",
			Payload: map[string]interface{}{
				"synced_count": synced,
				"total_count":  len(body.Specs),
				"repo_id":      repoID,
				"user":         userID,
			},
			Timestamp: time.Now(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"synced": synced,
		"total":  len(body.Specs),
	})
}

func (h *SpecHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")

	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required")
		return
	}

	specs, err := h.db.SearchSpecs(ctx, orgID, q, 50)
	if err != nil {
		log.Printf("search specs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to search")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": specs,
		"count":   len(specs),
		"query":   q,
	})
}

func (h *SpecHandler) handleActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")

	q := r.URL.Query()

	var eventTypes []string
	if types := q.Get("types"); types != "" {
		eventTypes = strings.Split(types, ",")
	}

	var since, until *time.Time
	if s := q.Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = &t
		}
	}
	if u := q.Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			until = &t
		}
	}

	limit := 50
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	// Use filtered query if any filters present, otherwise basic query
	var events []store.ActivityEvent
	var err error
	if len(eventTypes) > 0 || since != nil || until != nil {
		events, err = h.db.ListActivityFiltered(ctx, orgID, eventTypes, since, until, limit, offset)
	} else {
		events, err = h.db.ListActivity(ctx, orgID, limit, offset)
	}
	if err != nil {
		log.Printf("list activity: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list activity")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}
