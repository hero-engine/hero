package serve

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
)

// API provides HTTP handlers for the Hero daemon.
type API struct {
	server    *Server
	bus       *EventBus
	uiEnabled bool

	// proposals holds per-project in-memory inline-propose stores.
	// Lazily initialized on first use; transient per daemon process
	// (Decision 1 of the inline-propose contract).
	proposals *proposalStores
}

// NewAPI creates a new API instance backed by a multi-project server.
func NewAPI(server *Server, bus *EventBus, uiEnabled bool) *API {
	return &API{
		server:    server,
		bus:       bus,
		uiEnabled: uiEnabled,
		proposals: newProposalStores(),
	}
}

// Handler returns a configured http.Handler with all routes.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health endpoint (daemon-level, no project)
	mux.HandleFunc("/health", a.handleHealth)

	// Project listing
	mux.HandleFunc("/api/projects", a.handleProjects)

	// SSE events (all projects, filterable by ?project=)
	mux.HandleFunc("/api/events", SSEHandler(a.bus))

	// Project-namespaced endpoints: /api/{project}/...
	mux.HandleFunc("/api/", a.routeProject)

	// Embedded dashboard UI
	if a.uiEnabled {
		uiSub, err := fs.Sub(uiFS, "ui")
		if err == nil {
			fileServer := http.FileServer(http.FS(uiSub))
			mux.Handle("/ui/", http.StripPrefix("/ui/", fileServer))
			// Serve index.html at root for SPA
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" {
					http.NotFound(w, r)
					return
				}
				data, err := fs.ReadFile(uiFS, "ui/index.html")
				if err != nil {
					http.Error(w, "dashboard not available", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(data)
			})
		}
	}

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
	if path == "" || path == "projects" || path == "events" {
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
