package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	gh "github.com/hero-engine/hero/cloud/github"
	"github.com/hero-engine/hero/cloud/store"
)

// GovernanceHandler handles GitHub App webhooks and governance config.
type GovernanceHandler struct {
	db     *store.DB
	app    *gh.App
	sseHub *SSEHub
}

// NewGovernanceHandler creates a governance handler.
func NewGovernanceHandler(db *store.DB, app *gh.App) *GovernanceHandler {
	return &GovernanceHandler{db: db, app: app}
}

// RegisterRoutes adds governance routes to the mux.
func (h *GovernanceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/github/webhook", h.handleWebhook)
}

// RegisterAuthenticatedRoutes adds governance routes that require auth.
// Routes are wrapped with withOrg, which validates membership and binds
// an RLS-scoped connection to r.Context().
func (h *GovernanceHandler) RegisterAuthenticatedRoutes(mux *http.ServeMux) {
	wrap := func(handler http.HandlerFunc) http.HandlerFunc { return withOrg(h.db, handler) }
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/governance", wrap(h.handleGetGovernance))
	mux.HandleFunc("PUT /api/v1/orgs/{org_id}/governance", wrap(h.handleUpdateGovernance))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/governance/stats", wrap(h.handleGovernanceStats))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/audit", wrap(h.handleAuditExport))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/audit/summary", wrap(h.handleAuditSummary))
}

func (h *GovernanceHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	event, err := gh.ParseWebhook(r, h.app.Config().WebhookSecret)
	if err != nil {
		log.Printf("webhook parse error: %v", err)
		http.Error(w, "invalid webhook", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	switch event.EventType {
	case "installation":
		h.handleInstallation(ctx, w, event)
	case "pull_request":
		h.handlePullRequest(ctx, w, event)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (h *GovernanceHandler) handleInstallation(ctx context.Context, w http.ResponseWriter, event *gh.WebhookEvent) {
	if event.Installation == nil {
		http.Error(w, "missing installation", http.StatusBadRequest)
		return
	}

	switch event.Action {
	case "created":
		inst := &store.Installation{
			InstallationID: event.Installation.ID,
			AccountLogin:   event.Installation.Account.Login,
			AccountType:    event.Installation.Account.Type,
			GovernanceMode: "advisory",
		}

		// Try to match to an existing org by account login
		org, err := h.db.GetOrgBySlug(ctx, event.Installation.Account.Login)
		if err == nil && org != nil {
			inst.OrgID = org.ID
		}

		if err := h.db.UpsertInstallation(ctx, inst); err != nil {
			log.Printf("upsert installation: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

	case "deleted":
		if err := h.db.DeleteInstallation(ctx, event.Installation.ID); err != nil {
			log.Printf("delete installation: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *GovernanceHandler) handlePullRequest(ctx context.Context, w http.ResponseWriter, event *gh.WebhookEvent) {
	switch event.Action {
	case "opened", "synchronize", "reopened":
		// proceed
	default:
		w.WriteHeader(http.StatusOK)
		return
	}

	if event.PullRequest == nil || event.Installation == nil || event.Repository == nil {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	inst, err := h.db.GetInstallationByGitHubID(ctx, event.Installation.ID)
	if err != nil {
		log.Printf("installation lookup failed: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	mode := gh.GovernanceMode(inst.GovernanceMode)
	if mode == gh.ModeDisabled {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Bind a session scoped to this installation's org so the RLS-protected
	// writes below (pr_checks, activity_events) pass WITH CHECK.
	if inst.OrgID != "" {
		boundCtx, release, err := h.db.WithOrg(ctx, inst.OrgID)
		if err != nil {
			log.Printf("acquire org session for webhook: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer release()
		ctx = boundCtx
	}

	result := gh.CheckPR(event.PullRequest, mode)

	// Compliance: fetch PR files and match against conventions
	h.runComplianceCheck(ctx, result, inst, event)

	// Record the check
	check := &store.PRCheck{
		InstallationID: inst.ID,
		RepoFullName:   event.Repository.FullName,
		PRNumber:       event.PullRequest.Number,
		HeadSHA:        event.PullRequest.Head.SHA,
		SpecSlugs:      result.SpecSlugs,
		HasSpec:        result.HasSpec,
		Conclusion:     result.Conclusion,
	}
	if err := h.db.RecordPRCheck(ctx, check); err != nil {
		log.Printf("recording PR check: %v", err)
	}

	// Audit: record PR check event
	if inst.OrgID != "" {
		_ = h.db.RecordAudit(ctx, store.AuditEntry{
			OrgID:     inst.OrgID,
			EventType: store.EventPRChecked,
			Payload: map[string]interface{}{
				"repo":       event.Repository.FullName,
				"pr_number":  event.PullRequest.Number,
				"head_sha":   event.PullRequest.Head.SHA,
				"has_spec":   result.HasSpec,
				"spec_slugs": result.SpecSlugs,
				"conclusion": result.Conclusion,
				"mode":       string(mode),
			},
		})

		// Audit: record convention matches
		for _, m := range result.ComplianceMatches {
			_ = h.db.RecordAudit(ctx, store.AuditEntry{
				OrgID:     inst.OrgID,
				EventType: store.EventConventionMatched,
				Payload: map[string]interface{}{
					"repo":          event.Repository.FullName,
					"pr_number":     event.PullRequest.Number,
					"convention":    m.Convention.Slug,
					"matched_files": m.MatchedFiles,
				},
			})
		}

		// Audit: record scope drift
		if result.ScopeDrift != nil && result.ScopeDrift.HasDrift {
			_ = h.db.RecordAudit(ctx, store.AuditEntry{
				OrgID:     inst.OrgID,
				EventType: store.EventScopeDrift,
				Payload: map[string]interface{}{
					"repo":        event.Repository.FullName,
					"pr_number":   event.PullRequest.Number,
					"spec_slug":   result.ScopeDrift.SpecSlug,
					"drift_files": result.ScopeDrift.DriftFiles,
				},
			})
		}

		// Broadcast SSE event for real-time dashboard updates
		if h.sseHub != nil {
			h.sseHub.Broadcast(inst.OrgID, SSEEvent{
				Type: "pr.checked",
				Payload: map[string]interface{}{
					"repo":       event.Repository.FullName,
					"pr_number":  event.PullRequest.Number,
					"has_spec":   result.HasSpec,
					"conclusion": result.Conclusion,
				},
				Timestamp: time.Now(),
			})
		}
	}

	// Create check run
	if err := gh.CreateCheckRun(h.app, inst.InstallationID, event.Repository.FullName, event.PullRequest.Head.SHA, result); err != nil {
		log.Printf("creating check run: %v", err)
	}

	// Advisory comment when no spec linked
	if mode == gh.ModeAdvisory && !result.HasSpec {
		comment := "**Hero Governance** (advisory mode)\n\n" + result.Summary
		if err := gh.PostComment(h.app, inst.InstallationID, event.Repository.FullName, event.PullRequest.Number, comment); err != nil {
			log.Printf("posting advisory comment: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// runComplianceCheck fetches PR files, matches conventions, detects scope drift,
// and augments the check result with compliance information.
func (h *GovernanceHandler) runComplianceCheck(ctx context.Context, result *gh.CheckResult, inst *store.Installation, event *gh.WebhookEvent) {
	if h.app == nil || inst.OrgID == "" {
		return
	}

	// Fetch PR files from GitHub
	prFiles, err := gh.GetPRFiles(h.app, inst.InstallationID, event.Repository.FullName, event.PullRequest.Number)
	if err != nil {
		log.Printf("fetching PR files for compliance: %v", err)
		return
	}

	filenames := make([]string, len(prFiles))
	for i, f := range prFiles {
		filenames[i] = f.Filename
	}

	// Load active conventions for the org
	storeConventions, err := h.db.GetActiveConventionsByOrg(ctx, inst.OrgID)
	if err != nil {
		log.Printf("loading conventions for compliance: %v", err)
		return
	}

	if len(storeConventions) > 0 {
		// Convert store conventions to github conventions for matching
		ghConventions := make([]gh.Convention, len(storeConventions))
		for i, c := range storeConventions {
			ghConventions[i] = gh.Convention{
				Slug:   c.Slug,
				Title:  c.Title,
				Scope:  c.Scope,
				Status: c.Status,
			}
		}

		result.ComplianceMatches = gh.MatchConventions(ghConventions, filenames)
	}

	// Scope drift: check if linked specs declare a scope
	if result.HasSpec {
		for _, slug := range result.SpecSlugs {
			spec, err := h.db.GetSpecBySlugForOrg(ctx, inst.OrgID, slug)
			if err != nil || spec == nil {
				continue
			}
			if len(spec.FilesTouched) > 0 {
				drift := gh.DetectScopeDrift(spec.FilesTouched, filenames)
				drift.SpecSlug = slug
				if drift.HasDrift {
					result.ScopeDrift = drift
					break // report first drift found
				}
			}
		}
	}

	gh.FormatComplianceSummary(result)
}

func (h *GovernanceHandler) handleGetGovernance(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("org_id")

	installations, err := h.db.GetInstallationByOrg(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetching installations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"installations": installations,
	})
}

func (h *GovernanceHandler) handleUpdateGovernance(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InstallationID int64  `json:"installation_id"`
		Mode           string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	switch body.Mode {
	case "advisory", "enforcement", "disabled":
		// valid
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid mode %q (use: advisory, enforcement, disabled)", body.Mode))
		return
	}

	if err := h.db.UpdateGovernanceMode(r.Context(), body.InstallationID, body.Mode); err != nil {
		writeError(w, http.StatusInternalServerError, "updating governance mode")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "mode": body.Mode})
}

func (h *GovernanceHandler) handleGovernanceStats(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("org_id")

	total, linked, unlinked, err := h.db.PRCheckStats(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetching stats")
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

func (h *GovernanceHandler) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("org_id")
	q := r.URL.Query()

	// Parse filters
	var eventTypes []string
	if types := q.Get("types"); types != "" {
		eventTypes = strings.Split(types, ",")
	}

	var since, until *time.Time
	if s := q.Get("since"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'since' format (use RFC3339)")
			return
		}
		since = &t
	}
	if u := q.Get("until"); u != "" {
		t, err := time.Parse(time.RFC3339, u)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'until' format (use RFC3339)")
			return
		}
		until = &t
	}

	limit := 100
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

	events, err := h.db.ListAuditEvents(r.Context(), orgID, eventTypes, since, until, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetching audit events")
		return
	}

	// CSV export if requested
	if q.Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=audit-export.csv")
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"id", "event_type", "org_id", "repo_id", "user_id", "payload", "created_at"})
		for _, e := range events {
			repoID := ""
			if e.RepoID != nil {
				repoID = *e.RepoID
			}
			userID := ""
			if e.UserID != nil {
				userID = *e.UserID
			}
			_ = cw.Write([]string{
				e.ID, e.EventType, e.OrgID, repoID, userID,
				string(e.Payload), e.CreatedAt.Format(time.RFC3339),
			})
		}
		cw.Flush()
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (h *GovernanceHandler) handleAuditSummary(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("org_id")

	var since *time.Time
	if s := r.URL.Query().Get("since"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'since' format (use RFC3339)")
			return
		}
		since = &t
	}

	summary, err := h.db.AuditSummary(r.Context(), orgID, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetching audit summary")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"summary": summary,
	})
}
