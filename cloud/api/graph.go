package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/hero-engine/hero/cloud/store"
)

// GraphHandler hosts the team-server side of the federation sync
// protocol — push deltas from clients, serve incremental pulls.
type GraphHandler struct {
	db *store.DB
}

func NewGraphHandler(db *store.DB) *GraphHandler {
	return &GraphHandler{db: db}
}

// RegisterRoutes adds the graph endpoints to the mux. The wire format
// matches internal/graph/sync.go on the client side.
//
//   POST /api/v1/orgs/{org_id}/graph/push?repo=<repo>
//   GET  /api/v1/orgs/{org_id}/graph/pull?repo=<repo>&since=<ts>&limit=<n>
//
// Both require org membership; the JWT's UserID is matched against
// the org's member list.
func (h *GraphHandler) RegisterRoutes(mux *http.ServeMux) {
	wrap := func(handler http.HandlerFunc) http.HandlerFunc { return withOrg(h.db, handler) }
	mux.HandleFunc("POST /api/v1/orgs/{org_id}/graph/push", wrap(h.handlePush))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/graph/pull", wrap(h.handlePull))
	mux.HandleFunc("GET /api/v1/orgs/{org_id}/graph/impact", wrap(h.handleImpact))
}

// pushRequest mirrors the client-side graph.PushRequest. We don't
// import internal/graph here to avoid cycles; the on-the-wire shape
// is documented in the federation spec.
type pushRequest struct {
	ClientID string         `json:"client_id"`
	Since    string         `json:"since"`
	Nodes    []wireNode     `json:"nodes"`
	Edges    []wireEdge     `json:"edges"`
}

type wireNode struct {
	Type        string                 `json:"type"`
	Key         string                 `json:"key"`
	Props       map[string]any         `json:"props"`
	Scope       string                 `json:"scope"`
	Repo        string                 `json:"repo,omitempty"`
	Unit        string                 `json:"unit,omitempty"`
	ContentHash string                 `json:"content_hash,omitempty"`
	Source      map[string]any         `json:"source"`
	ValidFrom   string                 `json:"valid_from"`
	ValidTo     string                 `json:"valid_to,omitempty"`
	IngestedAt  string                 `json:"ingested_at"`
}

type wireEdge struct {
	FromID     int64                  `json:"from_id"`
	ToID       int64                  `json:"to_id"`
	Type       string                 `json:"type"`
	Props      map[string]any         `json:"props"`
	Scope      string                 `json:"scope"`
	Repo       string                 `json:"repo,omitempty"`
	Unit       string                 `json:"unit,omitempty"`
	Source     map[string]any         `json:"source"`
	ValidFrom  string                 `json:"valid_from"`
	ValidTo    string                 `json:"valid_to,omitempty"`
	IngestedAt string                 `json:"ingested_at"`
}

type pushResponse struct {
	Accepted   int                  `json:"accepted"`
	Conflicts  []store.GraphConflict `json:"conflicts,omitempty"`
	ServerTime string               `json:"server_time"`
}

type pullResponse struct {
	Nodes      []wireNode `json:"nodes"`
	Edges      []wireEdge `json:"edges"`
	NextCursor string     `json:"next_cursor"`
	ServerTime string     `json:"server_time"`
}

func (h *GraphHandler) handlePush(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")
	userID := UserIDFromContext(ctx)
	repo := r.URL.Query().Get("repo")

	var req pushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid push body: "+err.Error())
		return
	}
	// Bind client_id to the authenticated user, not whatever the client sends.
	// Two workspaces by the same user push as the same identity (no false
	// conflicts). Two users push as different identities (real conflicts).
	clientID := userID
	_ = req.ClientID // ignored — kept on wire for backward compat

	storeNodes := make([]store.GraphNode, 0, len(req.Nodes))
	for _, n := range req.Nodes {
		propsJSON, _ := json.Marshal(n.Props)
		sourceJSON, _ := json.Marshal(n.Source)
		nodeRepo := n.Repo
		if nodeRepo == "" {
			nodeRepo = repo
		}
		gn := store.GraphNode{
			Repo:     nodeRepo,
			Unit:     n.Unit,
			Type:     n.Type,
			Key:      n.Key,
			Props:    propsJSON,
			Scope:    n.Scope,
			Hash:     n.ContentHash,
			Source:   sourceJSON,
			ClientID: clientID,
		}
		storeNodes = append(storeNodes, gn)
	}

	storeEdges := make([]store.GraphEdge, 0, len(req.Edges))
	for _, e := range req.Edges {
		// Endpoints are smuggled in props as _from_type/_from_key/etc
		// (set by client-side hydration in internal/graph/sync.go).
		fromType := stringFromProps(e.Props, "_from_type")
		fromKey := stringFromProps(e.Props, "_from_key")
		toType := stringFromProps(e.Props, "_to_type")
		toKey := stringFromProps(e.Props, "_to_key")
		if fromType == "" || fromKey == "" || toType == "" || toKey == "" {
			continue
		}
		// Strip routing keys from props before persistence.
		clean := make(map[string]any, len(e.Props))
		for k, v := range e.Props {
			if k == "_from_type" || k == "_from_key" || k == "_to_type" || k == "_to_key" {
				continue
			}
			clean[k] = v
		}
		propsJSON, _ := json.Marshal(clean)
		sourceJSON, _ := json.Marshal(e.Source)
		edgeRepo := e.Repo
		if edgeRepo == "" {
			edgeRepo = repo
		}
		ge := store.GraphEdge{
			Repo:     edgeRepo,
			Unit:     e.Unit,
			FromType: fromType,
			FromKey:  fromKey,
			ToType:   toType,
			ToKey:    toKey,
			Type:     e.Type,
			Props:    propsJSON,
			Scope:    e.Scope,
			Source:   sourceJSON,
			ClientID: clientID,
		}
		storeEdges = append(storeEdges, ge)
	}

	accepted, conflicts, serverTime, err := h.db.PushGraphDelta(ctx, orgID, storeNodes, storeEdges)
	if err != nil {
		log.Printf("push graph: %v", err)
		writeError(w, http.StatusInternalServerError, "push failed")
		return
	}

	writeJSON(w, http.StatusOK, pushResponse{
		Accepted:   accepted,
		Conflicts:  conflicts,
		ServerTime: serverTime.Format(time.RFC3339Nano),
	})
}

func (h *GraphHandler) handlePull(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")
	repo := r.URL.Query().Get("repo")
	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339Nano, sinceStr)
		if err != nil {
			t, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid since cursor: "+err.Error())
				return
			}
		}
		since = t
	}

	// Pass empty repo so pull returns all org data regardless of which repo
	// pushed it — federation requires all teammates' nodes to be visible.
	// The repo param identifies the caller for cursor tracking only.
	_ = repo
	nodes, edges, maxTime, err := h.db.PullGraphDelta(ctx, orgID, "", since, 0)
	if err != nil {
		log.Printf("pull graph: %v", err)
		writeError(w, http.StatusInternalServerError, "pull failed")
		return
	}

	wireNodes := make([]wireNode, 0, len(nodes))
	for _, n := range nodes {
		var props map[string]any
		if len(n.Props) > 0 {
			_ = json.Unmarshal(n.Props, &props)
		}
		var src map[string]any
		if len(n.Source) > 0 {
			_ = json.Unmarshal(n.Source, &src)
		}
		wn := wireNode{
			Type:        n.Type,
			Key:         n.Key,
			Props:       props,
			Scope:       n.Scope,
			Repo:        n.Repo,
			Unit:        n.Unit,
			ContentHash: n.Hash,
			Source:      src,
		}
		wireNodes = append(wireNodes, wn)
	}

	wireEdges := make([]wireEdge, 0, len(edges))
	for _, e := range edges {
		var props map[string]any
		if len(e.Props) > 0 {
			_ = json.Unmarshal(e.Props, &props)
		}
		if props == nil {
			props = map[string]any{}
		}
		// Re-hydrate routing keys for the client.
		props["_from_type"] = e.FromType
		props["_from_key"] = e.FromKey
		props["_to_type"] = e.ToType
		props["_to_key"] = e.ToKey

		var src map[string]any
		if len(e.Source) > 0 {
			_ = json.Unmarshal(e.Source, &src)
		}
		we := wireEdge{
			Type:   e.Type,
			Props:  props,
			Scope:  e.Scope,
			Repo:   e.Repo,
			Unit:   e.Unit,
			Source: src,
		}
		wireEdges = append(wireEdges, we)
	}

	cursor := maxTime.UTC().Format(time.RFC3339Nano)
	if maxTime.IsZero() && sinceStr != "" {
		cursor = sinceStr
	}

	writeJSON(w, http.StatusOK, pullResponse{
		Nodes:      wireNodes,
		Edges:      wireEdges,
		NextCursor: cursor,
		ServerTime: time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// handleImpact serves cross-repo blast-radius queries. Given a
// (type, key) target, returns every Package / File / etc. across the
// org that has an incoming edge to it (imports / references /
// depends_on / touches), grouped by source repo. Phase 8 of the
// federation roadmap.
//
//   GET /api/v1/orgs/{org_id}/graph/impact?type=Symbol&key=auth.NewToken
//
// type defaults to Symbol; key is required.
func (h *GraphHandler) handleImpact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.PathValue("org_id")
	targetType := r.URL.Query().Get("type")
	if targetType == "" {
		targetType = "Symbol"
	}
	targetKey := r.URL.Query().Get("key")
	if targetKey == "" {
		writeError(w, http.StatusBadRequest, "key parameter is required")
		return
	}

	callers, err := h.db.ImpactCrossRepo(ctx, orgID, targetType, targetKey)
	if err != nil {
		log.Printf("impact query: %v", err)
		writeError(w, http.StatusInternalServerError, "impact query failed")
		return
	}

	// Group by repo for readable output. Order preserved by query.
	byRepo := map[string][]store.ImpactCaller{}
	var repos []string
	for _, c := range callers {
		if _, seen := byRepo[c.FromRepo]; !seen {
			repos = append(repos, c.FromRepo)
		}
		byRepo[c.FromRepo] = append(byRepo[c.FromRepo], c)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"target_type": targetType,
		"target_key":  targetKey,
		"total":       len(callers),
		"repos":       repos,
		"by_repo":     byRepo,
	})
}

func stringFromProps(p map[string]any, key string) string {
	v, ok := p[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
