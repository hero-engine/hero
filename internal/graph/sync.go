package graph

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SyncClient pushes local-graph deltas to and pulls server-graph
// deltas from a hero team server. The wire format is documented in
// .hero/planning/features/graph-memory-federation/spec.md:
//
//   POST /api/v1/graph/push?repo=<repo>
//     body: PushRequest
//     resp: PushResponse
//
//   GET  /api/v1/graph/pull?repo=<repo>&since=<cursor>&include=team,unit
//     resp: PullResponse
//
// All bodies are JSON. Auth is handled by the caller injecting an
// HTTP roundtripper that adds the right credentials.
type SyncClient struct {
	ServerURL string
	HTTP      *http.Client
	Repo      string
	Org       string
}

// NewSyncClient creates a client with sensible defaults. The default
// HTTP client uses a 30s timeout which is enough for a few thousand
// nodes per push.
func NewSyncClient(serverURL, repo, org string) *SyncClient {
	return &SyncClient{
		ServerURL: serverURL,
		Repo:      repo,
		Org:       org,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

// PushRequest is the wire payload for /api/v1/graph/push.
type PushRequest struct {
	ClientID string `json:"client_id"` // install_id from meta table
	Since    string `json:"since"`     // last_push_at; "" for full first push
	Nodes    []Node `json:"nodes"`
	Edges    []Edge `json:"edges"`
}

// PushResponse is what the server returns.
type PushResponse struct {
	Accepted   int            `json:"accepted"`
	Conflicts  []SyncConflict `json:"conflicts,omitempty"`
	ServerTime string         `json:"server_time"`
}

// SyncConflict describes a (type, key) where the local row diverged
// from the server's current row. Resolution strategy is decided by
// the caller — Phase 6a's conflict UI is the natural surface.
type SyncConflict struct {
	NodeType  string `json:"node_type,omitempty"` // empty for edge conflicts
	NodeKey   string `json:"node_key,omitempty"`
	EdgeFrom  string `json:"edge_from,omitempty"`
	EdgeTo    string `json:"edge_to,omitempty"`
	EdgeType  string `json:"edge_type,omitempty"`
	Reason    string `json:"reason"`
}

// PullResponse is the server's reply to /pull.
type PullResponse struct {
	Nodes      []Node `json:"nodes"`
	Edges      []Edge `json:"edges"`
	NextCursor string `json:"next_cursor"`
	ServerTime string `json:"server_time"`
}

// PendingPush returns nodes and edges that have been ingested locally
// since the last successful push to this server. Both team-scope and
// unit-scope rows are included; local-scope rows are never sent.
func (s *Store) PendingPush(serverURL string) ([]Node, []Edge, string, error) {
	since, err := s.lastPushAt(serverURL)
	if err != nil {
		return nil, nil, "", err
	}

	nodes, err := s.nodesSince(since)
	if err != nil {
		return nil, nil, since, fmt.Errorf("nodes since: %w", err)
	}
	edges, err := s.edgesSince(since)
	if err != nil {
		return nil, nil, since, fmt.Errorf("edges since: %w", err)
	}
	return nodes, edges, since, nil
}

// MarkPushed records that all rows up through serverTime were
// successfully accepted by this server. Subsequent PendingPush calls
// will use this as their lower bound.
func (s *Store) MarkPushed(serverURL, serverTime, org string) error {
	if serverTime == "" {
		serverTime = nowRFC3339()
	}
	_, err := s.db.Exec(
		`INSERT INTO sync_state(server_url, last_push_at, org)
		 VALUES (?, ?, ?)
		 ON CONFLICT(server_url) DO UPDATE SET
		   last_push_at = excluded.last_push_at,
		   org          = excluded.org`,
		serverURL, serverTime, org,
	)
	return err
}

// MarkPulled records the cursor returned by the server so the next
// pull picks up where we left off.
func (s *Store) MarkPulled(serverURL, cursor, org string) error {
	_, err := s.db.Exec(
		`INSERT INTO sync_state(server_url, last_pull_at, last_pull_cursor, org)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(server_url) DO UPDATE SET
		   last_pull_at     = excluded.last_pull_at,
		   last_pull_cursor = excluded.last_pull_cursor,
		   org              = excluded.org`,
		serverURL, nowRFC3339(), cursor, org,
	)
	return err
}

// LastPullCursor returns the last cursor saved for the given server
// (empty for never-pulled).
func (s *Store) LastPullCursor(serverURL string) (string, error) {
	var cursor sql.NullString
	err := s.db.QueryRow(
		`SELECT last_pull_cursor FROM sync_state WHERE server_url = ?`,
		serverURL,
	).Scan(&cursor)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return cursor.String, nil
}

// LastPullAt returns the RFC3339 timestamp of the last successful pull
// for the given server, or empty string if the server hasn't been
// pulled from yet.
func (s *Store) LastPullAt(serverURL string) (string, error) {
	var at sql.NullString
	err := s.db.QueryRow(
		`SELECT last_pull_at FROM sync_state WHERE server_url = ?`,
		serverURL,
	).Scan(&at)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return at.String, nil
}

// ApplyPull merges a pull response into the local graph. Each remote
// node/edge is upserted; the bitemporal model handles the conflict
// case naturally — divergent updates produce two valid rows and the
// caller can later surface them via the conflict UI.
//
// Returns the number of nodes and edges actually applied (after
// idempotency filtering) plus any rows that were skipped because
// their target node id couldn't be resolved (cross-batch edges).
func (s *Store) ApplyPull(nodes []Node, edges []Edge) (nodesApplied, edgesApplied, edgesDeferred int, err error) {
	for i := range nodes {
		n := nodes[i]
		// Server-supplied IDs aren't meaningful locally; the (type, key)
		// is the logical identity.
		n.ID = 0
		if _, err := s.UpsertNode(&n); err != nil {
			return nodesApplied, edgesApplied, edgesDeferred, fmt.Errorf("apply node %s/%s: %w", n.Type, n.Key, err)
		}
		nodesApplied++
	}

	// Resolve edges by (type, key) of their endpoints. The pull
	// response includes endpoint identifiers in the edge props so
	// resolution doesn't depend on shared row ids.
	for i := range edges {
		e := edges[i]
		fromKey := stringProp(e.Props, "_from_key")
		fromType := stringProp(e.Props, "_from_type")
		toKey := stringProp(e.Props, "_to_key")
		toType := stringProp(e.Props, "_to_type")

		fromID, err := s.GetNodeID(fromType, fromKey)
		if err != nil {
			edgesDeferred++
			continue
		}
		toID, err := s.GetNodeID(toType, toKey)
		if err != nil {
			edgesDeferred++
			continue
		}
		// Strip the routing keys before inserting so they don't
		// pollute on-disk props.
		clean := cloneProps(e.Props)
		delete(clean, "_from_key")
		delete(clean, "_from_type")
		delete(clean, "_to_key")
		delete(clean, "_to_type")
		e.Props = clean
		e.FromID = fromID
		e.ToID = toID
		e.ID = 0

		if _, err := s.UpsertEdge(&e); err != nil {
			return nodesApplied, edgesApplied, edgesDeferred, fmt.Errorf("apply edge: %w", err)
		}
		edgesApplied++
	}
	return nodesApplied, edgesApplied, edgesDeferred, nil
}

// Push runs PendingPush then sends the result to the server. On
// success it calls MarkPushed so the next push only sends new rows.
func (s *Store) Push(c *SyncClient) (*PushResponse, error) {
	nodes, edges, since, err := s.PendingPush(c.ServerURL)
	if err != nil {
		return nil, err
	}

	// Hydrate edges with endpoint type+key so the server can resolve
	// references even though it doesn't share row ids.
	hydrated := make([]Edge, 0, len(edges))
	for _, e := range edges {
		from, err := s.nodeByID(e.FromID)
		if err != nil {
			continue
		}
		to, err := s.nodeByID(e.ToID)
		if err != nil {
			continue
		}
		props := cloneProps(e.Props)
		props["_from_type"] = from.Type
		props["_from_key"] = from.Key
		props["_to_type"] = to.Type
		props["_to_key"] = to.Key
		e.Props = props
		hydrated = append(hydrated, e)
	}

	installID, err := s.installID()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(PushRequest{
		ClientID: installID,
		Since:    since,
		Nodes:    nodes,
		Edges:    hydrated,
	})
	if err != nil {
		return nil, err
	}

	endpoint := c.ServerURL + "/api/v1/graph/push?repo=" + url.QueryEscape(c.Repo)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("push request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server %d: %s", resp.StatusCode, string(respBytes))
	}
	var pr PushResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decoding push response: %w", err)
	}
	if err := s.MarkPushed(c.ServerURL, pr.ServerTime, c.Org); err != nil {
		return &pr, fmt.Errorf("mark pushed: %w", err)
	}
	return &pr, nil
}

// Pull fetches deltas from the server since the last pull cursor and
// applies them locally.
func (s *Store) Pull(c *SyncClient) (*PullResponse, int, int, int, error) {
	cursor, err := s.LastPullCursor(c.ServerURL)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	q := url.Values{}
	q.Set("repo", c.Repo)
	if cursor != "" {
		q.Set("since", cursor)
	}
	q.Set("include", "team,unit")

	endpoint := c.ServerURL + "/api/v1/graph/pull?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("pull request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, 0, 0, 0, fmt.Errorf("server %d: %s", resp.StatusCode, string(respBytes))
	}
	var pr PullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("decoding pull response: %w", err)
	}

	nodesApplied, edgesApplied, edgesDeferred, err := s.ApplyPull(pr.Nodes, pr.Edges)
	if err != nil {
		return &pr, nodesApplied, edgesApplied, edgesDeferred, err
	}
	if err := s.MarkPulled(c.ServerURL, pr.NextCursor, c.Org); err != nil {
		return &pr, nodesApplied, edgesApplied, edgesDeferred, fmt.Errorf("mark pulled: %w", err)
	}
	return &pr, nodesApplied, edgesApplied, edgesDeferred, nil
}

// --- helpers ---------------------------------------------------------------

func (s *Store) lastPushAt(serverURL string) (string, error) {
	var t sql.NullString
	err := s.db.QueryRow(
		`SELECT last_push_at FROM sync_state WHERE server_url = ?`,
		serverURL,
	).Scan(&t)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return t.String, nil
}

func (s *Store) installID() (string, error) {
	var id string
	err := s.db.QueryRow(
		`SELECT value FROM meta WHERE key = 'install_id'`,
	).Scan(&id)
	return id, err
}

// nodesSince returns nodes ingested at or after `since` whose scope
// is push-eligible (team, unit, public — never local).
func (s *Store) nodesSince(since string) ([]Node, error) {
	args := []any{}
	q := `SELECT id, type, key, props, scope, repo, unit, content_hash, source,
	             valid_from, valid_to, ingested_at
	        FROM nodes
	       WHERE scope IN ('team','unit','public')`
	if since != "" {
		q += ` AND ingested_at > ?`
		args = append(args, since)
	}
	q += ` ORDER BY ingested_at`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *n)
	}
	return out, nil
}

func (s *Store) edgesSince(since string) ([]Edge, error) {
	args := []any{}
	q := `SELECT id, from_id, to_id, type, props, scope, repo, unit, source,
	             valid_from, valid_to, ingested_at
	        FROM edges
	       WHERE scope IN ('team','unit','public')`
	if since != "" {
		q += ` AND ingested_at > ?`
		args = append(args, since)
	}
	q += ` ORDER BY ingested_at`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, nil
}

func (s *Store) nodeByID(id int64) (*Node, error) {
	row := s.db.QueryRow(
		`SELECT id, type, key, props, scope, repo, unit, content_hash, source,
		        valid_from, valid_to, ingested_at
		   FROM nodes WHERE id = ?`,
		id,
	)
	return scanNode(row)
}

func cloneProps(p map[string]any) map[string]any {
	out := make(map[string]any, len(p)+4)
	for k, v := range p {
		out[k] = v
	}
	return out
}

func stringProp(p map[string]any, key string) string {
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
