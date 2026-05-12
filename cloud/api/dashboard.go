package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/hero-engine/hero/cloud/store"
)

// DashboardHandler handles dashboard-specific endpoints.
type DashboardHandler struct {
	db *store.DB
}

// NewDashboardHandler creates a dashboard handler.
func NewDashboardHandler(db *store.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// RegisterRoutes adds dashboard routes to the mux. All routes are wrapped
// with withOrg, which validates org membership and binds an RLS-scoped
// connection to the request context. Handlers can use r.Context() for
// every db call without further auth setup.
func (h *DashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	wrap := func(handler http.HandlerFunc) http.HandlerFunc { return withOrg(h.db, handler) }

	mux.HandleFunc("GET /api/v1/orgs/{org_id}/specs", wrap(h.handleListOrgSpecs))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/specs/pipeline", wrap(h.handleSpecPipeline))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/conventions", wrap(h.handleListConventions))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/analytics/overview", wrap(h.handleAnalyticsOverview))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/analytics/velocity", wrap(h.handleAnalyticsVelocity))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/analytics/heatmap", wrap(h.handleAnalyticsHeatmap))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/repos/{repo_id}/governance/stats", wrap(h.handleRepoGovernanceStats))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/knowledge/search", wrap(h.handleKnowledgeSearch))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/specs/{slug}/conflicts", wrap(h.handleSpecConflicts))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/conflicts", wrap(h.handleAllConflicts))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/specs/sequence", wrap(h.handleSpecSequence))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/patterns", wrap(h.handlePatterns))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/conventions/suggestions", wrap(h.handleConventionSuggestions))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/insights", wrap(h.handleInsights))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/intelligence/status", wrap(h.handleIntelligenceStatus))
	mux.HandleFunc("POST /api/v1/orgs/{org_id}/intelligence/opt-in", wrap(h.handleIntelligenceOptIn))
}

// requireOrgMembership is a thin shim — auth and RLS session binding are
// done by the withOrg wrapper at route registration time. Handlers just
// need orgID from the URL.
func (h *DashboardHandler) requireOrgMembership(w http.ResponseWriter, r *http.Request) (string, bool) {
	return r.PathValue("org_id"), true
}

// handleListOrgSpecs returns specs across all repos in an org.
func (h *DashboardHandler) handleListOrgSpecs(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	specType := q.Get("type")
	status := q.Get("status")
	repoID := q.Get("repo")
	query := q.Get("q")
	subproject := q.Get("subproject")
	sort := q.Get("sort")

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

	specs, total, err := h.db.ListOrgSpecs(r.Context(), orgID, specType, status, repoID, query, subproject, sort, limit, offset)
	if err != nil {
		log.Printf("list org specs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list specs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"specs":  specs,
		"count":  len(specs),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// handleSpecPipeline returns spec counts by status.
func (h *DashboardHandler) handleSpecPipeline(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	counts, err := h.db.SpecStatusCounts(r.Context(), orgID)
	if err != nil {
		log.Printf("spec pipeline: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get pipeline")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pipeline": counts,
	})
}

// handleListConventions returns conventions for an org.
func (h *DashboardHandler) handleListConventions(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	repoID := q.Get("repo")
	query := q.Get("q")

	conventions, err := h.db.ListOrgConventions(r.Context(), orgID, repoID, query)
	if err != nil {
		log.Printf("list conventions: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list conventions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"conventions": conventions,
		"count":       len(conventions),
	})
}

func (h *DashboardHandler) parseTimeRange(r *http.Request) (time.Time, time.Time) {
	q := r.URL.Query()
	now := time.Now().UTC()
	since := now.AddDate(0, 0, -30) // default 30 days
	until := now

	if s := q.Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}
	if u := q.Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			until = t
		}
	}
	return since, until
}

// handleAnalyticsOverview returns aggregated KPI metrics.
func (h *DashboardHandler) handleAnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	since, until := h.parseTimeRange(r)

	overview, err := h.db.AnalyticsOverview(r.Context(), orgID, since, until)
	if err != nil {
		log.Printf("analytics overview: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to compute analytics")
		return
	}

	writeJSON(w, http.StatusOK, overview)
}

// handleAnalyticsVelocity returns time-bucketed delivery counts.
func (h *DashboardHandler) handleAnalyticsVelocity(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	since, until := h.parseTimeRange(r)
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "week"
	}

	buckets, err := h.db.DeliveryVelocity(r.Context(), orgID, since, until, interval)
	if err != nil {
		log.Printf("analytics velocity: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to compute velocity")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"buckets":  buckets,
		"interval": interval,
	})
}

// handleAnalyticsHeatmap returns daily activity counts.
func (h *DashboardHandler) handleAnalyticsHeatmap(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	now := time.Now().UTC()
	since := now.AddDate(-1, 0, 0) // default 1 year
	until := now

	if s := q.Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}
	if u := q.Get("until"); u != "" {
		if t, err := time.Parse(time.RFC3339, u); err == nil {
			until = t
		}
	}

	days, err := h.db.ActivityHeatmap(r.Context(), orgID, since, until)
	if err != nil {
		log.Printf("analytics heatmap: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to compute heatmap")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"days": days,
	})
}

// handleRepoGovernanceStats returns per-repo PR check stats.
func (h *DashboardHandler) handleRepoGovernanceStats(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	repoID := r.PathValue("repo_id")
	// Get repo to find its name
	repo, err := h.db.GetRepoByID(r.Context(), orgID, repoID)
	if err != nil || repo == nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}

	total, linked, unlinked, err := h.db.PRCheckStatsForRepo(r.Context(), orgID, repo.Name)
	if err != nil {
		log.Printf("repo governance stats: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	linkRate := 0.0
	if total > 0 {
		linkRate = float64(linked) / float64(total) * 100
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_prs":     total,
		"linked_prs":    linked,
		"unlinked_prs":  unlinked,
		"link_rate_pct": linkRate,
	})
}

// handleKnowledgeSearch searches across conventions and specs by keyword.
func (h *DashboardHandler) handleKnowledgeSearch(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "q parameter required")
		return
	}

	// Search conventions
	conventions, err := h.db.ListOrgConventions(r.Context(), orgID, "", query)
	if err != nil {
		log.Printf("knowledge search conventions: %v", err)
	}

	// Search specs
	specs, err := h.db.SearchSpecs(r.Context(), orgID, query, 20)
	if err != nil {
		log.Printf("knowledge search specs: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"conventions": conventions,
		"specs":       specs,
		"query":       query,
	})
}

// handleSpecConflicts returns specs that share files with a given spec.
func (h *DashboardHandler) handleSpecConflicts(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "slug is required")
		return
	}

	conflicts, err := h.db.FindConflicts(r.Context(), orgID, slug)
	if err != nil {
		log.Printf("finding conflicts for %s: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "failed to find conflicts")
		return
	}
	if conflicts == nil {
		conflicts = []store.SpecConflict{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"slug":      slug,
		"conflicts": conflicts,
	})
}

// handleAllConflicts returns all conflicting spec pairs in an org.
func (h *DashboardHandler) handleAllConflicts(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	conflicts, err := h.db.FindAllConflicts(r.Context(), orgID)
	if err != nil {
		log.Printf("finding all conflicts: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to find conflicts")
		return
	}
	if conflicts == nil {
		conflicts = []store.SpecConflict{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"conflicts": conflicts,
	})
}

// handleSpecSequence returns a suggested delivery order for in-flight specs.
func (h *DashboardHandler) handleSpecSequence(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	items, err := h.db.SuggestSequence(r.Context(), orgID)
	if err != nil {
		log.Printf("suggesting sequence: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to suggest sequence")
		return
	}
	if items == nil {
		items = []store.SequenceItem{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sequence": items,
	})
}

// handlePatterns returns mined patterns from institutional memory.
func (h *DashboardHandler) handlePatterns(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	patterns, err := h.db.MinePatterns(r.Context(), orgID)
	if err != nil {
		log.Printf("mining patterns: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to mine patterns")
		return
	}
	if patterns == nil {
		patterns = []store.Pattern{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"patterns": patterns,
	})
}

// handleConventionSuggestions returns auto-generated convention suggestions.
func (h *DashboardHandler) handleConventionSuggestions(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	suggestions, err := h.db.SuggestConventions(r.Context(), orgID)
	if err != nil {
		log.Printf("suggesting conventions: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to suggest conventions")
		return
	}
	if suggestions == nil {
		suggestions = []store.ConventionSuggestion{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"suggestions": suggestions,
	})
}

// handleInsights returns cross-org intelligence recommendations.
func (h *DashboardHandler) handleInsights(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	insights, err := h.db.GetInsights(r.Context(), orgID)
	if err != nil {
		log.Printf("getting insights: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get insights")
		return
	}
	if insights == nil {
		insights = []store.Insight{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"insights": insights,
	})
}

// handleIntelligenceStatus returns whether the org has opted into cross-org intelligence.
func (h *DashboardHandler) handleIntelligenceStatus(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	optedIn, _ := h.db.GetIntelligenceOptIn(r.Context(), orgID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"opted_in": optedIn,
	})
}

// handleIntelligenceOptIn toggles cross-org intelligence opt-in.
func (h *DashboardHandler) handleIntelligenceOptIn(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.requireOrgMembership(w, r)
	if !ok {
		return
	}

	var body struct {
		OptIn bool `json:"opt_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.db.SetIntelligenceOptIn(r.Context(), orgID, body.OptIn); err != nil {
		log.Printf("setting opt-in: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update opt-in")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"opted_in": body.OptIn,
	})
}
