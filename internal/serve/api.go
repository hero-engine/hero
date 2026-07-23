package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/serve/healthcache"
	"github.com/hero-engine/hero/internal/serve/opsrunner"
	"github.com/hero-engine/hero/internal/spec"
)

// API provides HTTP handlers for the Hero daemon.
//
// The API owns /health and /api/* (projects, events, project-scoped
// endpoints). The web-app shell — top nav, page routing, home content,
// /-redirect — lives in internal/serve/shell and is composed alongside
// this handler in Server.Run.
type API struct {
	server *Server
	bus    *EventBus

	// proposals holds per-project in-memory inline-propose stores.
	// Lazily initialized on first use; transient per daemon process
	// (Decision 1 of the inline-propose contract).
	proposals *proposalStores

	// opsRunner spawns + tracks the Operations-section subprocesses for
	// every project. Nil-tolerant: when nil, /api/{slug}/ops/* returns 503.
	// Set via Server.NewServer during construction; shared across all
	// per-project + aggregate handlers.
	opsRunner *opsrunner.Runner

	// healthCache backs the /api/{slug}/health, /health/refresh, and
	// /peers/{alias}/probe endpoints. Phase 5 of
	// hero-serve-project-section. Nil-tolerant — endpoints return 503
	// when unset.
	healthCache *healthcache.Cache
}

// NewAPI creates a new API instance backed by a multi-project server.
func NewAPI(server *Server, bus *EventBus) *API {
	return &API{
		server:    server,
		bus:       bus,
		proposals: newProposalStores(),
	}
}

// SetOpsRunner wires the ops runner into the API. Called by
// Server.NewServer after the runner is constructed.
func (a *API) SetOpsRunner(r *opsrunner.Runner) { a.opsRunner = r }

// OpsRunner returns the wired ops runner, or nil if not configured.
// Used by Server to inject the same runner into projectpage.Deps.
func (a *API) OpsRunner() *opsrunner.Runner { return a.opsRunner }

// SetHealthCache wires the in-process health/peer cache into the API.
// Called by Server.NewServer after the cache is constructed.
func (a *API) SetHealthCache(c *healthcache.Cache) { a.healthCache = c }

// Handler returns a configured http.Handler with the API routes. The
// shell router is layered on top of this handler in Server.Run.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health endpoint (daemon-level, no project)
	mux.HandleFunc("/health", a.handleHealth)

	// Daemon-level status — pid, port, uptime, version, project list.
	// Distinct from /health (lightweight liveness) and from the
	// per-project /api/{slug}/status.
	mux.HandleFunc("/api/status", a.handleDaemonStatus)

	// Re-read ~/.hero/projects.json and surface the refreshed list as
	// JSON. POST-only; used by the /p/all/project "Refresh registry"
	// button. Phase 2 of hero-serve-project-section.
	mux.HandleFunc("/api/daemon/registry/refresh", a.handleRegistryRefresh)

	// Daemon-scoped ops dispatch (Phase 4 of hero-serve-project-section).
	// /api/daemon/ops/stop launches `hero serve stop` via the opsrunner.
	// Mounted before the generic /api/ catch-all so it wins routing.
	mux.HandleFunc("/api/daemon/ops/", a.handleDaemonOps)

	// Project listing
	mux.HandleFunc("/api/projects", a.handleProjects)

	// SSE events (all projects, filterable by ?project=)
	mux.HandleFunc("/api/events", SSEHandler(a.bus))

	// Project-namespaced endpoints: /api/{project}/...
	mux.HandleFunc("/api/", a.routeProject)

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// Project router — dispatches /api/{project}/... to the right handlers
// ---------------------------------------------------------------------------

func (a *API) routeProject(w http.ResponseWriter, r *http.Request) {
	// Parse: /api/{project}/{endpoint}[/{extra}]
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	if path == "" || path == "projects" || path == "events" || path == "status" {
		// Already handled by specific routes
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.SplitN(path, "/", 2)
	slug := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	// Look up the project
	pc := a.server.GetProject(slug)
	if pc == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %q not found", slug))
		return
	}

	// Route to the right handler
	endpoint := rest
	extra := ""
	if idx := strings.IndexByte(rest, '/'); idx >= 0 {
		endpoint = rest[:idx]
		extra = rest[idx+1:]
	}

	switch endpoint {
	case "status":
		a.handleStatus(w, r, pc)
	case "specs":
		if extra != "" {
			a.handleSpecBySlug(w, r, pc, extra)
		} else {
			a.handleSpecs(w, r, pc)
		}
	case "search":
		a.handleSearch(w, r, pc)
	case "context":
		a.handleContext(w, r, pc)
	case "check":
		a.handleCheck(w, r, pc)
	case "knowledge":
		a.handleKnowledge(w, r, pc)
	case "inventory":
		a.handleInventory(w, r, pc)
	case "sessions":
		// /api/{project}/sessions/{session_id}/proposals[/...]
		sid, rest := splitFirst(extra, "/")
		section, rest := splitFirst(rest, "/")
		if section != "proposals" {
			writeError(w, http.StatusNotFound, fmt.Sprintf("unknown sessions sub-endpoint: %s", section))
			return
		}
		a.routeProposals(w, r, pc, sid, rest)
	case "ops":
		// /api/{slug}/ops/{verb}                      — POST: start
		// /api/{slug}/ops/{job_id}/stream             — GET:  SSE
		a.routeOps(w, r, pc, extra)
	case "registry":
		// /api/{slug}/registry/remove              — POST: enqueue
		// /api/{slug}/registry/remove/undo         — POST: cancel
		// Phase 4 of hero-serve-project-section.
		a.routeRegistry(w, r, pc, extra)
	case "health":
		// /api/{slug}/health           — GET: cached health snapshot
		// /api/{slug}/health/refresh   — POST: kick a refresh
		// Phase 5 of hero-serve-project-section.
		a.routeHealthCache(w, r, pc, extra)
	case "peers":
		// /api/{slug}/peers/{alias}/probe — POST: refresh peer reachability
		// Phase 5 of hero-serve-project-section.
		a.routePeerProbe(w, r, pc, extra)
	default:
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown endpoint: %s", endpoint))
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"version":  a.server.version,
		"projects": a.server.ProjectCount(),
	})
}

// handleDaemonStatus returns the daemon-level status: pid, port,
// uptime, version, and the list of served projects. Used by the CLI
// `hero serve status` command and by the bind-collision probe.
func (a *API) handleDaemonStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	started := a.server.StartedAt()
	var uptime int64
	if !started.IsZero() {
		uptime = int64(time.Since(started).Seconds())
	}

	projects := make([]DaemonStatusPC, 0)
	a.server.mu.RLock()
	for slug, pc := range a.server.projects {
		projects = append(projects, DaemonStatusPC{
			Slug: slug,
			Name: slug,
			Path: pc.Path,
		})
	}
	a.server.mu.RUnlock()

	resp := DaemonStatusResponse{
		Running:       true,
		PID:           os.Getpid(),
		Port:          a.server.Port(),
		StartedAt:     started,
		UptimeSeconds: uptime,
		Version:       a.server.Version(),
		ProjectCount:  len(projects),
		Projects:      projects,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleRegistryRefresh re-reads ~/.hero/projects.json (or the
// configured registry path) and returns the refreshed list as JSON.
//
// Behaviour:
//   - POST: reload registry from disk, ensure server.projects reflects
//     newly-added entries, return the current list.
//   - GET:  return the current list without re-reading disk (cheap
//     "show me the current state" probe).
//
// The endpoint is best-effort — a reload error is logged and the
// current in-memory list is still returned (degraded operation
// surface). Phase 2 of hero-serve-project-section.
func (a *API) handleRegistryRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	type entryView struct {
		Slug         string    `json:"slug"`
		Path         string    `json:"path"`
		RegisteredAt time.Time `json:"registered_at,omitempty"`
	}

	resp := struct {
		Reloaded bool        `json:"reloaded"`
		Count    int         `json:"count"`
		Projects []entryView `json:"projects"`
		Error    string      `json:"error,omitempty"`
	}{}

	if r.Method == http.MethodPost {
		if a.server.registry != nil {
			reloaded, rerr := LoadRegistryFrom(a.server.registry.FilePath())
			if rerr != nil {
				resp.Error = rerr.Error()
			} else {
				a.server.mu.Lock()
				a.server.registry = reloaded
				a.server.mu.Unlock()
				a.server.loadRegistryProjects()
				resp.Reloaded = true
			}
		}
	}

	a.server.mu.RLock()
	for slug, pc := range a.server.projects {
		if pc == nil {
			continue
		}
		view := entryView{Slug: slug, Path: pc.Path}
		if a.server.registry != nil {
			if entry := a.server.registry.Get(slug); entry != nil {
				view.RegisteredAt = entry.Registered
			}
		}
		resp.Projects = append(resp.Projects, view)
	}
	a.server.mu.RUnlock()
	resp.Count = len(resp.Projects)
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	type projectInfo struct {
		Slug string `json:"slug"`
		Path string `json:"path"`
	}

	var projects []projectInfo
	a.server.mu.RLock()
	for slug, pc := range a.server.projects {
		projects = append(projects, projectInfo{
			Slug: slug,
			Path: pc.Path,
		})
	}
	a.server.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"projects": projects,
		"count":    len(projects),
	})
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	specs, err := spec.Discover(pc.HeroDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("discovering specs: %v", err))
		return
	}

	type statusResponse struct {
		Project    string        `json:"project"`
		Delivering int           `json:"delivering"`
		InReview   int           `json:"in_review"`
		Planning   int           `json:"planning"`
		Completed  int           `json:"completed"`
		Knowledge  int           `json:"knowledge"`
		Total      int           `json:"total"`
		Specs      []specSummary `json:"specs"`
	}

	var resp statusResponse
	resp.Project = pc.Slug
	for _, sp := range specs {
		resp.Total++
		summary := specSummary{
			Slug:      sp.Slug,
			Title:     sp.Title,
			Type:      string(sp.Type),
			Status:    string(sp.Status),
			ClaimedBy: sp.ClaimedBy,
			Tags:      sp.Tags,
			TrackerID: sp.TrackerID,
		}
		if !sp.CreatedAt.IsZero() {
			summary.Created = sp.CreatedAt.Format("2006-01-02")
		}
		if sp.IsKnowledge() {
			resp.Knowledge++
		} else {
			switch sp.Status {
			case spec.StatusDelivering:
				resp.Delivering++
			case spec.StatusInReview:
				resp.InReview++
			case spec.StatusPlanning:
				resp.Planning++
			case spec.StatusCompleted:
				resp.Completed++
			}
		}
		resp.Specs = append(resp.Specs, summary)
	}

	writeJSON(w, http.StatusOK, resp)
}

type specSummary struct {
	Slug      string   `json:"slug"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Status    string   `json:"status"`
	ClaimedBy string   `json:"claimed_by,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	TrackerID string   `json:"tracker_id,omitempty"`
	Priority  string   `json:"priority,omitempty"`
	Created   string   `json:"created,omitempty"`
}

type specDetail struct {
	specSummary
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

func (a *API) handleSpecs(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// List with filters
	specType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	tag := r.URL.Query().Get("tag")

	idx, err := index.Open(pc.HeroDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("opening index: %v", err))
		return
	}
	defer idx.Close()

	var results []index.SearchResult
	if specType == "" && status == "" && tag == "" {
		results, err = idx.AllSpecs()
	} else {
		results, err = idx.ListFiltered(specType, status, tag, "")
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing specs: %v", err))
		return
	}

	summaries := make([]specSummary, 0, len(results))
	for _, r := range results {
		s := specSummary{
			Slug:      r.Slug,
			Title:     r.Title,
			Type:      string(r.Type),
			Status:    string(r.Status),
			ClaimedBy: r.ClaimedBy,
		}
		if r.Tags != "" {
			s.Tags = splitTags(r.Tags)
		}
		summaries = append(summaries, s)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"specs": summaries,
		"count": len(summaries),
	})
}

func (a *API) handleSpecBySlug(w http.ResponseWriter, r *http.Request, pc *ProjectContext, slug string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idx, err := index.Open(pc.HeroDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("opening index: %v", err))
		return
	}
	defer idx.Close()

	all, err := idx.AllSpecs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing specs: %v", err))
		return
	}

	var found *index.SearchResult
	for _, sr := range all {
		if sr.Slug == slug {
			found = &sr
			break
		}
	}

	if found == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("spec %q not found", slug))
		return
	}

	content, err := os.ReadFile(found.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("reading spec: %v", err))
		return
	}

	// Re-parse the spec to get full metadata (tracker_id, priority, etc.)
	sp, parseErr := spec.ParseFile(found.Path)

	detail := specDetail{
		specSummary: specSummary{
			Slug:      found.Slug,
			Title:     found.Title,
			Type:      string(found.Type),
			Status:    string(found.Status),
			ClaimedBy: found.ClaimedBy,
		},
		Path:    found.Path,
		Content: string(content),
	}

	// Enrich from parsed spec if available
	if parseErr == nil && sp != nil {
		detail.Tags = sp.Tags
		detail.TrackerID = sp.TrackerID
		if !sp.CreatedAt.IsZero() {
			detail.Created = sp.CreatedAt.Format("2006-01-02")
		}
	} else if found.Tags != "" {
		detail.Tags = splitTags(found.Tags)
	}

	writeJSON(w, http.StatusOK, detail)
}

func (a *API) handleSearch(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required")
		return
	}

	specType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")

	idx, err := index.Open(pc.HeroDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("opening index: %v", err))
		return
	}
	defer idx.Close()

	var results []index.SearchResult
	if specType != "" || status != "" {
		results, err = idx.SearchFiltered(query, specType, status, "", "")
	} else {
		results, err = idx.Search(query)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("searching: %v", err))
		return
	}

	summaries := make([]specSummary, 0, len(results))
	for _, sr := range results {
		s := specSummary{
			Slug:      sr.Slug,
			Title:     sr.Title,
			Type:      string(sr.Type),
			Status:    string(sr.Status),
			ClaimedBy: sr.ClaimedBy,
		}
		if sr.Tags != "" {
			s.Tags = splitTags(sr.Tags)
		}
		summaries = append(summaries, s)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": summaries,
		"count":   len(summaries),
	})
}

func (a *API) handleContext(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	filesParam := r.URL.Query().Get("files")
	if filesParam == "" {
		writeError(w, http.StatusBadRequest, "files parameter is required")
		return
	}

	filePaths := strings.Split(filesParam, ",")
	for i := range filePaths {
		filePaths[i] = strings.TrimSpace(filePaths[i])
	}

	idx, err := index.Open(pc.HeroDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("opening index: %v", err))
		return
	}
	defer idx.Close()

	ctx, err := idx.BuildContext(filePaths)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("building context: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, ctx)
}

func (a *API) handleCheck(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	cfg, err := config.Load(pc.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("loading config: %v", err))
		return
	}

	idx, err := index.Open(pc.HeroDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("opening index: %v", err))
		return
	}
	defer idx.Close()

	stats, err := idx.GetStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("getting stats: %v", err))
		return
	}

	staleDays := 14
	if daysStr := r.URL.Query().Get("stale_days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			staleDays = d
		}
	} else if cfg.Team != nil && cfg.Team.StaleDays > 0 {
		staleDays = cfg.Team.StaleDays
	}

	stale, _ := idx.CheckStale(staleDays)
	unclaimed, _ := idx.CheckUnclaimed()

	type checkResponse struct {
		Project   string               `json:"project"`
		Stats     index.Stats          `json:"stats"`
		StaleDays int                  `json:"stale_days"`
		Stale     []index.SearchResult `json:"stale"`
		Unclaimed []index.SearchResult `json:"unclaimed"`
		Issues    int                  `json:"issues"`
	}

	resp := checkResponse{
		Project:   pc.Slug,
		Stats:     stats,
		StaleDays: staleDays,
		Stale:     stale,
		Unclaimed: unclaimed,
		Issues:    len(stale) + len(unclaimed),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleKnowledge(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	typeFilter := r.URL.Query().Get("type")

	specs, err := spec.Discover(pc.HeroDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("discovering specs: %v", err))
		return
	}

	var entries []specSummary
	for _, sp := range specs {
		if !sp.IsKnowledge() {
			continue
		}
		if typeFilter != "" && string(sp.Type) != typeFilter {
			continue
		}
		entries = append(entries, specSummary{
			Slug:   sp.Slug,
			Title:  sp.Title,
			Type:   string(sp.Type),
			Status: string(sp.Status),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

func (a *API) handleInventory(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	specs, err := spec.Discover(pc.HeroDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("discovering specs: %v", err))
		return
	}

	type bugEntry struct {
		Slug      string   `json:"slug"`
		Title     string   `json:"title"`
		Status    string   `json:"status"`
		Priority  string   `json:"priority,omitempty"`
		TrackerID string   `json:"tracker_id,omitempty"`
		ClaimedBy string   `json:"claimed_by,omitempty"`
		Tags      []string `json:"tags,omitempty"`
		Created   string   `json:"created,omitempty"`
	}

	var bugs []bugEntry
	for _, sp := range specs {
		if sp.Type != spec.TypeBug {
			continue
		}
		b := bugEntry{
			Slug:      sp.Slug,
			Title:     sp.Title,
			Status:    string(sp.Status),
			TrackerID: sp.TrackerID,
			ClaimedBy: sp.ClaimedBy,
			Tags:      sp.Tags,
		}
		if !sp.CreatedAt.IsZero() {
			b.Created = sp.CreatedAt.Format(time.RFC3339)
		}
		// Extract priority from frontmatter sections or tags
		for _, tag := range sp.Tags {
			switch strings.ToLower(tag) {
			case "critical", "blocker", "high", "medium", "low":
				b.Priority = tag
			}
		}
		bugs = append(bugs, b)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"bugs":  bugs,
		"count": len(bugs),
	})
}

// splitFirst splits s on the first occurrence of sep, returning the
// head and the remainder (without the separator). If sep is not
// found, head is s and rest is empty.
func splitFirst(s, sep string) (head, rest string) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):]
	}
	return s, ""
}

// routeOps dispatches the /api/{slug}/ops/... namespace.
//
//	POST /api/{slug}/ops/{verb}                  — start a job for verb
//	GET  /api/{slug}/ops/{job_id}/stream         — SSE progress stream
//
// Anything else under /ops/ returns 404. Allowlist enforcement happens
// inside opsrunner.Start (the API layer just trusts the runner — same
// 400 surface).
func (a *API) routeOps(w http.ResponseWriter, r *http.Request, pc *ProjectContext, extra string) {
	if a.opsRunner == nil {
		writeError(w, http.StatusServiceUnavailable, "ops runner not configured")
		return
	}
	head, rest := splitFirst(extra, "/")
	if head == "" {
		writeError(w, http.StatusNotFound, "ops endpoint missing verb or job id")
		return
	}
	switch r.Method {
	case http.MethodPost:
		if rest != "" {
			writeError(w, http.StatusNotFound, "POST to /ops/{verb} only")
			return
		}
		a.handleOpsStart(w, r, pc, head)
	case http.MethodGet:
		if rest != "stream" {
			writeError(w, http.StatusNotFound, "GET requires /ops/{job_id}/stream")
			return
		}
		a.handleOpsStream(w, r, pc, head)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) handleOpsStart(w http.ResponseWriter, r *http.Request, pc *ProjectContext, verb string) {
	if !opsrunner.IsAllowed(verb) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("verb %q not in allowlist", verb))
		return
	}
	jobID, started, err := a.opsRunner.Start(r.Context(), pc.Slug, pc.Path, verb)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := map[string]interface{}{"job_id": jobID}
	if !started {
		resp["already_running"] = true
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleOpsStream(w http.ResponseWriter, r *http.Request, pc *ProjectContext, jobID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	if err := a.opsRunner.Stream(r.Context(), pc.Slug, jobID, w); err != nil {
		// Stream sets no body on its own when the job is missing — write
		// a JSON envelope here. We have not yet emitted any SSE bytes,
		// so changing status is still valid.
		http.Error(w, err.Error(), http.StatusNotFound)
	}
}

// pendingRemoveGraceWindow is the 5-second undo window for registry
// removals. Phase 4 of hero-serve-project-section — UX-tested at 5s in
// the parent spec.
const pendingRemoveGraceWindow = 5 * time.Second

// routeRegistry dispatches /api/{slug}/registry/... endpoints. Today
// this is just remove + remove/undo (Phase 4); future destructive
// operations on the registry slot for a project land here.
func (a *API) routeRegistry(w http.ResponseWriter, r *http.Request, pc *ProjectContext, extra string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	switch extra {
	case "remove":
		a.handleRegistryRemove(w, r, pc)
	case "remove/undo":
		a.handleRegistryRemoveUndo(w, r, pc)
	default:
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown registry endpoint: %s", extra))
	}
}

// handleRegistryRemove enqueues a pending-remove for pc.Slug, returning
// the absolute deadline so the client can render its undo countdown.
// onCommit invokes Server.RemoveProject (which both removes the project
// from the in-memory map and persists the registry to disk).
//
// Re-posting before the deadline elapses resets the timer — the existing
// entry is cancelled and a fresh one is enqueued. That keeps the API
// idempotent under double-clicks.
func (a *API) handleRegistryRemove(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if a.server == nil || a.server.pendingRemove == nil {
		writeError(w, http.StatusServiceUnavailable, "pending-remove queue not configured")
		return
	}
	slug := pc.Slug
	deadline := a.server.pendingRemove.Enqueue(slug, pendingRemoveGraceWindow, func() error {
		if err := a.server.RemoveProject(slug); err != nil {
			fmt.Fprintf(os.Stderr, "hero serve: pending remove %s: %v\n", slug, err)
			return err
		}
		return nil
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"slug":     slug,
		"deadline": deadline.UTC().Format(time.RFC3339Nano),
	})
}

// handleRegistryRemoveUndo cancels any pending-remove for pc.Slug.
// Always returns 200 — idempotent by design so a double-click on Undo
// (or an Undo after the window has already elapsed) doesn't surface a
// confusing error to the user.
func (a *API) handleRegistryRemoveUndo(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	cancelled := false
	if a.server != nil && a.server.pendingRemove != nil {
		cancelled = a.server.pendingRemove.Cancel(pc.Slug)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"slug":      pc.Slug,
		"cancelled": cancelled,
	})
}

// handleDaemonOps dispatches /api/daemon/ops/<verb> through the same
// opsrunner that backs the per-project ops endpoints, using the
// daemon-scoped slug. Today the only supported verb is `stop` (Phase 4
// of hero-serve-project-section); other verbs are rejected.
//
// The SSE stream for a daemon op uses the same per-project URL shape
// — /api/_daemon/ops/{job_id}/stream — which is reachable via the
// generic project-router because we treat the special slug as just
// another project (it has no ProjectContext, so the router must accept
// it explicitly).
func (a *API) handleDaemonOps(w http.ResponseWriter, r *http.Request) {
	if a.opsRunner == nil {
		writeError(w, http.StatusServiceUnavailable, "ops runner not configured")
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/api/daemon/ops/")
	head, rest := splitFirst(tail, "/")
	if head == "" {
		writeError(w, http.StatusNotFound, "daemon ops endpoint missing verb or job id")
		return
	}
	switch r.Method {
	case http.MethodPost:
		if rest != "" {
			writeError(w, http.StatusNotFound, "POST to /api/daemon/ops/{verb} only")
			return
		}
		if head != "stop" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("verb %q not allowed on daemon ops endpoint", head))
			return
		}
		if !opsrunner.IsAllowed(head) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("verb %q not in allowlist", head))
			return
		}
		jobID, started, err := a.opsRunner.Start(r.Context(), opsrunner.DaemonScopedSlug, "", head)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		resp := map[string]interface{}{"job_id": jobID}
		if !started {
			resp["already_running"] = true
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodGet:
		if rest != "stream" {
			writeError(w, http.StatusNotFound, "GET requires /api/daemon/ops/{job_id}/stream")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		if _, ok := w.(http.Flusher); !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		if err := a.opsRunner.Stream(r.Context(), opsrunner.DaemonScopedSlug, head, w); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// healthSnapshotResponse is the JSON shape returned by
// GET /api/{slug}/health. Mirrors the on-disk schema with the addition
// of age_seconds + stale, computed at request time.
type healthSnapshotResponse struct {
	Slug        string                 `json:"slug"`
	CapturedAt  *time.Time             `json:"captured_at,omitempty"`
	Rows        []healthcache.HealthRow `json:"rows"`
	FromDisk    bool                   `json:"from_disk"`
	AgeSeconds  *int64                 `json:"age_seconds,omitempty"`
	Stale       bool                   `json:"stale"`
	TTLSeconds  int64                  `json:"ttl_seconds"`
}

// routeHealthCache dispatches /api/{slug}/health and
// /api/{slug}/health/refresh.
func (a *API) routeHealthCache(w http.ResponseWriter, r *http.Request, pc *ProjectContext, extra string) {
	if a.healthCache == nil {
		writeError(w, http.StatusServiceUnavailable, "health cache not configured")
		return
	}
	switch extra {
	case "":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		a.handleHealthGet(w, r, pc)
	case "refresh":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		a.handleHealthRefresh(w, r, pc)
	default:
		writeError(w, http.StatusNotFound, "unknown health endpoint")
	}
}

func (a *API) handleHealthGet(w http.ResponseWriter, _ *http.Request, pc *ProjectContext) {
	resp := healthSnapshotResponse{
		Slug:       pc.Slug,
		Rows:       []healthcache.HealthRow{},
		TTLSeconds: int64(a.healthCache.TTL().Seconds()),
	}
	if cached, ok := a.healthCache.Health(pc.Slug); ok {
		resp.Rows = cached.Rows
		resp.FromDisk = cached.FromDisk
		if !cached.Captured.IsZero() {
			t := cached.Captured
			resp.CapturedAt = &t
		}
		if !cached.Timestamp.IsZero() {
			age := int64(time.Since(cached.Timestamp).Seconds())
			resp.AgeSeconds = &age
			if cached.TTL > 0 && time.Since(cached.Timestamp) > cached.TTL {
				resp.Stale = true
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleHealthRefresh dispatches a `hero check --json` subprocess
// through the opsrunner and returns the job id immediately. The client
// subscribes to /api/{slug}/ops/{job_id}/stream for progress; once it
// receives the exit event it should re-fetch GET /api/{slug}/health to
// pick up the new cached result.
//
// The cache update itself happens in a background goroutine that
// blocks on the runner's Wait and then reads the on-disk artifact.
// This decouples the HTTP response from the (potentially slow)
// subprocess and matches the existing /ops/ UX.
func (a *API) handleHealthRefresh(w http.ResponseWriter, r *http.Request, pc *ProjectContext) {
	if a.opsRunner == nil {
		writeError(w, http.StatusServiceUnavailable, "ops runner not configured")
		return
	}
	jobID, _, err := a.opsRunner.Start(r.Context(), pc.Slug, pc.Path, "run-check-json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Update the cache in the background once the job completes. We
	// intentionally drop the cache.RefreshHealth path here because the
	// runner already started the subprocess — re-dispatching would
	// just dedupe to the same job. Instead, wait on the existing job
	// and pull the artifact off disk ourselves.
	go func(slug, projectRoot, id string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if _, werr := a.opsRunner.Wait(ctx, slug, id); werr != nil {
			fmt.Fprintf(os.Stderr, "hero serve: health refresh wait %s: %v\n", slug, werr)
			return
		}
		if _, rerr := a.healthCache.RefreshFromDisk(slug, projectRoot); rerr != nil {
			fmt.Fprintf(os.Stderr, "hero serve: health refresh read %s: %v\n", slug, rerr)
		}
	}(pc.Slug, pc.Path, jobID)
	writeJSON(w, http.StatusAccepted, map[string]string{
		"job_id":      jobID,
		"stream_url":  fmt.Sprintf("/api/%s/ops/%s/stream", pc.Slug, jobID),
		"refresh_url": fmt.Sprintf("/api/%s/health", pc.Slug),
	})
}

// routePeerProbe dispatches /api/{slug}/peers/{alias}/probe.
func (a *API) routePeerProbe(w http.ResponseWriter, r *http.Request, pc *ProjectContext, extra string) {
	if a.healthCache == nil {
		writeError(w, http.StatusServiceUnavailable, "health cache not configured")
		return
	}
	alias, rest := splitFirst(extra, "/")
	if alias == "" || rest != "probe" {
		writeError(w, http.StatusNotFound, "expected /peers/{alias}/probe")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Resolve the peer alias against the project's hero.json. Empty
	// path means the peer isn't configured — pass through anyway so
	// the prober records an "unreachable" entry; the client sees a
	// useful row update either way.
	var peerPath string
	if cfg, err := config.Load(pc.Path); err == nil {
		if p, ok := cfg.Repos[alias]; ok {
			peerPath = p
		}
	}
	result, err := a.healthCache.ProbePeer(r.Context(), pc.Slug, alias, peerPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type peerProbeResponse struct {
		Slug      string    `json:"slug"`
		Alias     string    `json:"alias"`
		Reachable bool      `json:"reachable"`
		LastOK    time.Time `json:"last_ok,omitempty"`
		LastError string    `json:"last_error,omitempty"`
		Timestamp time.Time `json:"timestamp"`
	}
	writeJSON(w, http.StatusOK, peerProbeResponse{
		Slug:      result.Slug,
		Alias:     result.Alias,
		Reachable: result.Reachable,
		LastOK:    result.LastOK,
		LastError: result.LastError,
		Timestamp: result.Timestamp,
	})
}

// splitTags splits a comma-separated tags string into a slice,
// trimming whitespace from each tag.
func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}
