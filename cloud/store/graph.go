package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GraphNode is the server-side current state of a knowledge graph node.
type GraphNode struct {
	OrgID      string          `json:"org_id"`
	Repo       string          `json:"repo,omitempty"`
	Unit       string          `json:"unit,omitempty"`
	Type       string          `json:"type"`
	Key        string          `json:"key"`
	Props      json.RawMessage `json:"props"`
	Scope      string          `json:"scope"`
	Hash       string          `json:"hash,omitempty"`
	Source     json.RawMessage `json:"source"`
	ClientID   string          `json:"client_id,omitempty"`
	ServerTime time.Time       `json:"server_time"`
}

// GraphEdge is the server-side current state of a knowledge graph edge.
type GraphEdge struct {
	OrgID      string          `json:"org_id"`
	Repo       string          `json:"repo,omitempty"`
	Unit       string          `json:"unit,omitempty"`
	FromType   string          `json:"from_type"`
	FromKey    string          `json:"from_key"`
	ToType     string          `json:"to_type"`
	ToKey      string          `json:"to_key"`
	Type       string          `json:"type"`
	Props      json.RawMessage `json:"props"`
	Scope      string          `json:"scope"`
	Source     json.RawMessage `json:"source"`
	ClientID   string          `json:"client_id,omitempty"`
	ServerTime time.Time       `json:"server_time"`
}

// GraphConflict reports a node whose upsert overwrote a different client's version.
type GraphConflict struct {
	NodeType string `json:"node_type,omitempty"`
	NodeKey  string `json:"node_key,omitempty"`
	Reason   string `json:"reason"`
}

// PushGraphDelta upserts a batch of nodes and edges. Each entity type is
// handled in one read + one write + one history-append, regardless of batch
// size. Conflict detection is done by comparing the incoming client_id
// against the currently stored client_id for any row that changes.
func (db *DB) PushGraphDelta(ctx context.Context, orgID string, nodes []GraphNode, edges []GraphEdge) (accepted int, conflicts []GraphConflict, serverTime time.Time, err error) {
	now := time.Now().UTC()

	na, nc, err := pushNodes(ctx, db, orgID, nodes, now)
	if err != nil {
		return 0, nil, now, fmt.Errorf("push nodes: %w", err)
	}
	accepted += na
	conflicts = append(conflicts, nc...)

	ea, err := pushEdges(ctx, db, orgID, edges, now)
	if err != nil {
		return accepted, conflicts, now, fmt.Errorf("push edges: %w", err)
	}
	accepted += ea

	return accepted, conflicts, now, nil
}

func pushNodes(ctx context.Context, db *DB, orgID string, nodes []GraphNode, now time.Time) (int, []GraphConflict, error) {
	if len(nodes) == 0 {
		return 0, nil, nil
	}

	// Deduplicate — keep last occurrence per (type, key).
	nodes = dedupeNodes(nodes)

	// Build parallel arrays for UNNEST.
	n := len(nodes)
	repos := make([]string, n)
	units := make([]string, n)
	types := make([]string, n)
	keys := make([]string, n)
	props := make([][]byte, n)
	scopes := make([]string, n)
	hashes := make([]string, n)
	sources := make([][]byte, n)
	clientIDs := make([]string, n)
	for i, nd := range nodes {
		repos[i] = nd.Repo
		units[i] = nd.Unit
		types[i] = nd.Type
		keys[i] = nd.Key
		p := []byte(nd.Props)
		if len(p) == 0 {
			p = []byte("{}")
		}
		props[i] = p
		scopes[i] = nd.Scope
		hashes[i] = nd.Hash
		s := []byte(nd.Source)
		if len(s) == 0 {
			s = []byte("{}")
		}
		sources[i] = s
		clientIDs[i] = nd.ClientID
	}

	tx, err := db.Conn(ctx).Begin(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 1: read existing state for conflict detection.
	type prior struct{ hash, clientID string }
	existing := make(map[string]prior, n)
	rows, err := tx.Query(ctx,
		`SELECT gn.type, gn.key, gn.hash, gn.client_id
		   FROM graph_nodes gn
		   JOIN UNNEST($2::text[], $3::text[]) AS w(type, key)
		     ON gn.type = w.type AND gn.key = w.key
		  WHERE gn.org_id = $1`,
		orgID, types, keys,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("select existing nodes: %w", err)
	}
	for rows.Next() {
		var t, k, h, c string
		if err := rows.Scan(&t, &k, &h, &c); err != nil {
			rows.Close()
			return 0, nil, fmt.Errorf("scan existing: %w", err)
		}
		existing[t+"\x00"+k] = prior{h, c}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("existing nodes iter: %w", err)
	}

	// Classify: skip idempotent rows, collect conflicts, build history lists.
	var conflicts []GraphConflict
	histTypes := make([]string, 0, n)
	histKeys := make([]string, 0, n)
	histProps := make([][]byte, 0, n)
	histHashes := make([]string, 0, n)
	histClientIDs := make([]string, 0, n)

	accepted := 0
	for i, nd := range nodes {
		k := nd.Type + "\x00" + nd.Key
		if ex, found := existing[k]; found {
			if ex.hash != "" && ex.hash == nd.Hash {
				// Idempotent: same hash, nothing to do. Remove from insert arrays.
				repos[i] = "\x00skip"
				continue
			}
			if ex.clientID != "" && ex.clientID != nd.ClientID {
				conflicts = append(conflicts, GraphConflict{
					NodeType: nd.Type,
					NodeKey:  nd.Key,
					Reason:   fmt.Sprintf("overwrote version from client %s", ex.clientID),
				})
			}
		}
		histTypes = append(histTypes, nd.Type)
		histKeys = append(histKeys, nd.Key)
		histProps = append(histProps, props[i])
		histHashes = append(histHashes, nd.Hash)
		histClientIDs = append(histClientIDs, nd.ClientID)
		accepted++
	}

	// Filter out skipped rows from the insert arrays.
	fi := 0
	for i := range nodes {
		if repos[i] == "\x00skip" {
			continue
		}
		repos[fi] = repos[i]
		units[fi] = units[i]
		types[fi] = types[i]
		keys[fi] = keys[i]
		props[fi] = props[i]
		scopes[fi] = scopes[i]
		hashes[fi] = hashes[i]
		sources[fi] = sources[i]
		clientIDs[fi] = clientIDs[i]
		fi++
	}
	repos = repos[:fi]
	units = units[:fi]
	types = types[:fi]
	keys = keys[:fi]
	props = props[:fi]
	scopes = scopes[:fi]
	hashes = hashes[:fi]
	sources = sources[:fi]
	clientIDs = clientIDs[:fi]

	if len(types) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, conflicts, fmt.Errorf("commit: %w", err)
		}
		return accepted, conflicts, nil
	}

	// Step 2: upsert current state — one statement.
	if _, err := tx.Exec(ctx,
		`INSERT INTO graph_nodes
		   (org_id, repo, unit, type, key, props, scope, hash, source, client_id, server_time)
		 SELECT $1,
		        unnest($2::text[]), unnest($3::text[]),
		        unnest($4::text[]), unnest($5::text[]),
		        unnest($6::jsonb[]), unnest($7::text[]),
		        unnest($8::text[]), unnest($9::jsonb[]),
		        unnest($10::text[]), $11
		 ON CONFLICT (org_id, type, key) DO UPDATE
		   SET repo        = excluded.repo,
		       unit        = excluded.unit,
		       props       = excluded.props,
		       scope       = excluded.scope,
		       hash        = excluded.hash,
		       source      = excluded.source,
		       client_id   = excluded.client_id,
		       server_time = excluded.server_time`,
		orgID, repos, units, types, keys, props, scopes, hashes, sources, clientIDs, now,
	); err != nil {
		return 0, conflicts, fmt.Errorf("upsert nodes: %w", err)
	}

	// Step 3: append history for changed rows — one statement.
	if len(histTypes) > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO graph_node_history (org_id, type, key, props, hash, client_id, changed_at)
			 SELECT $1,
			        unnest($2::text[]), unnest($3::text[]),
			        unnest($4::jsonb[]), unnest($5::text[]),
			        unnest($6::text[]), $7`,
			orgID, histTypes, histKeys, histProps, histHashes, histClientIDs, now,
		); err != nil {
			return 0, conflicts, fmt.Errorf("history nodes: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, conflicts, fmt.Errorf("commit: %w", err)
	}
	return accepted, conflicts, nil
}

func pushEdges(ctx context.Context, db *DB, orgID string, edges []GraphEdge, now time.Time) (int, error) {
	if len(edges) == 0 {
		return 0, nil
	}

	edges = dedupeEdges(edges)
	n := len(edges)

	repos := make([]string, n)
	units := make([]string, n)
	fts := make([]string, n)
	fks := make([]string, n)
	tts := make([]string, n)
	tks := make([]string, n)
	typs := make([]string, n)
	props := make([][]byte, n)
	scopes := make([]string, n)
	sources := make([][]byte, n)
	clientIDs := make([]string, n)

	for i, e := range edges {
		repos[i] = e.Repo
		units[i] = e.Unit
		fts[i] = e.FromType
		fks[i] = e.FromKey
		tts[i] = e.ToType
		tks[i] = e.ToKey
		typs[i] = e.Type
		p := []byte(e.Props)
		if len(p) == 0 {
			p = []byte("{}")
		}
		props[i] = p
		scopes[i] = e.Scope
		s := []byte(e.Source)
		if len(s) == 0 {
			s = []byte("{}")
		}
		sources[i] = s
		clientIDs[i] = e.ClientID
	}

	tx, err := db.Conn(ctx).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 1: read existing props for idempotency check.
	type priorEdge struct{ props []byte }
	existingEdges := make(map[string]priorEdge, n)
	rows, err := tx.Query(ctx,
		`SELECT ge.from_type, ge.from_key, ge.to_type, ge.to_key, ge.type, ge.props
		   FROM graph_edges ge
		   JOIN UNNEST($2::text[],$3::text[],$4::text[],$5::text[],$6::text[])
		        AS w(from_type,from_key,to_type,to_key,type)
		     ON ge.from_type=w.from_type AND ge.from_key=w.from_key
		    AND ge.to_type=w.to_type AND ge.to_key=w.to_key
		    AND ge.type=w.type
		  WHERE ge.org_id = $1`,
		orgID, fts, fks, tts, tks, typs,
	)
	if err != nil {
		return 0, fmt.Errorf("select existing edges: %w", err)
	}
	for rows.Next() {
		var ft, fk, tt, tk, typ string
		var p []byte
		if err := rows.Scan(&ft, &fk, &tt, &tk, &typ, &p); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan existing edge: %w", err)
		}
		existingEdges[ft+"\x00"+fk+"\x00"+tt+"\x00"+tk+"\x00"+typ] = priorEdge{p}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("existing edges iter: %w", err)
	}

	// Mark idempotent rows for skip.
	accepted := 0
	for i, e := range edges {
		k := e.FromType + "\x00" + e.FromKey + "\x00" + e.ToType + "\x00" + e.ToKey + "\x00" + e.Type
		if ex, found := existingEdges[k]; found && jsonEqual(ex.props, props[i]) {
			repos[i] = "\x00skip"
			continue
		}
		accepted++
	}

	fi := 0
	for i := range edges {
		if repos[i] == "\x00skip" {
			continue
		}
		repos[fi] = repos[i]
		units[fi] = units[i]
		fts[fi] = fts[i]
		fks[fi] = fks[i]
		tts[fi] = tts[i]
		tks[fi] = tks[i]
		typs[fi] = typs[i]
		props[fi] = props[i]
		scopes[fi] = scopes[i]
		sources[fi] = sources[i]
		clientIDs[fi] = clientIDs[i]
		fi++
	}
	repos = repos[:fi]
	units = units[:fi]
	fts = fts[:fi]
	fks = fks[:fi]
	tts = tts[:fi]
	tks = tks[:fi]
	typs = typs[:fi]
	props = props[:fi]
	scopes = scopes[:fi]
	sources = sources[:fi]
	clientIDs = clientIDs[:fi]

	if len(fts) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit: %w", err)
		}
		return accepted, nil
	}

	// Step 2: upsert — one statement.
	if _, err := tx.Exec(ctx,
		`INSERT INTO graph_edges
		   (org_id, repo, unit, from_type, from_key, to_type, to_key, type,
		    props, scope, source, client_id, server_time)
		 SELECT $1,
		        unnest($2::text[]), unnest($3::text[]),
		        unnest($4::text[]), unnest($5::text[]),
		        unnest($6::text[]), unnest($7::text[]),
		        unnest($8::text[]), unnest($9::jsonb[]),
		        unnest($10::text[]), unnest($11::jsonb[]),
		        unnest($12::text[]), $13
		 ON CONFLICT (org_id, from_type, from_key, type, to_type, to_key) DO UPDATE
		   SET repo        = excluded.repo,
		       unit        = excluded.unit,
		       props       = excluded.props,
		       scope       = excluded.scope,
		       source      = excluded.source,
		       client_id   = excluded.client_id,
		       server_time = excluded.server_time`,
		orgID, repos, units, fts, fks, tts, tks, typs,
		props, scopes, sources, clientIDs, now,
	); err != nil {
		return 0, fmt.Errorf("upsert edges: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return accepted, nil
}

// PullGraphDelta returns nodes and edges with server_time > since for the org.
// All rows in graph_nodes/graph_edges are current state — no valid_to filter.
func (db *DB) PullGraphDelta(ctx context.Context, orgID, _ string, since time.Time, limit int) ([]GraphNode, []GraphEdge, time.Time, error) {
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}
	maxTime := since

	rows, err := db.Conn(ctx).Query(ctx,
		`SELECT org_id, repo, unit, type, key, props, scope, hash, source, client_id, server_time
		   FROM graph_nodes
		  WHERE org_id = $1 AND server_time > $2
		  ORDER BY server_time
		  LIMIT $3`,
		orgID, since, limit,
	)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []GraphNode
	for rows.Next() {
		var nd GraphNode
		if err := rows.Scan(
			&nd.OrgID, &nd.Repo, &nd.Unit, &nd.Type, &nd.Key,
			&nd.Props, &nd.Scope, &nd.Hash, &nd.Source, &nd.ClientID, &nd.ServerTime,
		); err != nil {
			return nil, nil, time.Time{}, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, nd)
		if nd.ServerTime.After(maxTime) {
			maxTime = nd.ServerTime
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("nodes iter: %w", err)
	}

	erows, err := db.Conn(ctx).Query(ctx,
		`SELECT org_id, repo, unit, from_type, from_key, to_type, to_key,
		        type, props, scope, source, client_id, server_time
		   FROM graph_edges
		  WHERE org_id = $1 AND server_time > $2
		  ORDER BY server_time
		  LIMIT $3`,
		orgID, since, limit,
	)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("query edges: %w", err)
	}
	defer erows.Close()

	var edges []GraphEdge
	for erows.Next() {
		var e GraphEdge
		if err := erows.Scan(
			&e.OrgID, &e.Repo, &e.Unit, &e.FromType, &e.FromKey, &e.ToType, &e.ToKey,
			&e.Type, &e.Props, &e.Scope, &e.Source, &e.ClientID, &e.ServerTime,
		); err != nil {
			return nil, nil, time.Time{}, fmt.Errorf("scan edge: %w", err)
		}
		edges = append(edges, e)
		if e.ServerTime.After(maxTime) {
			maxTime = e.ServerTime
		}
	}
	if err := erows.Err(); err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("edges iter: %w", err)
	}

	return nodes, edges, maxTime, nil
}

// ImpactCaller describes a node that has an incoming dependency edge to a target.
type ImpactCaller struct {
	FromRepo  string          `json:"from_repo"`
	FromType  string          `json:"from_type"`
	FromKey   string          `json:"from_key"`
	EdgeType  string          `json:"edge_type"`
	EdgeProps json.RawMessage `json:"edge_props,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ImpactCrossRepo finds all edges into the target (type, key) across the org.
// Tries exact key match first, then suffix fallback.
func (db *DB) ImpactCrossRepo(ctx context.Context, orgID, targetType, targetKey string) ([]ImpactCaller, error) {
	rows, err := db.Conn(ctx).Query(ctx,
		`SELECT e.repo, e.from_type, e.from_key, e.type, e.props, e.server_time
		   FROM graph_edges e
		  WHERE e.org_id = $1
		    AND e.to_type = $2 AND e.to_key = $3
		    AND e.type IN ('imports', 'references', 'depends_on', 'touches')
		  ORDER BY e.type, e.server_time DESC
		  LIMIT 500`,
		orgID, targetType, targetKey,
	)
	if err != nil {
		return nil, fmt.Errorf("query exact: %w", err)
	}
	out, err := scanImpactCallers(rows)
	if err != nil {
		return nil, err
	}
	if len(out) > 0 {
		return out, nil
	}

	// Suffix fallback: caller passes bare name without repo prefix.
	rows2, err := db.Conn(ctx).Query(ctx,
		`SELECT e.repo, e.from_type, e.from_key, e.type, e.props, e.server_time
		   FROM graph_edges e
		  WHERE e.org_id = $1
		    AND e.to_type = $2 AND e.to_key LIKE $3
		    AND e.type IN ('imports', 'references', 'depends_on', 'touches')
		  ORDER BY e.type, e.server_time DESC
		  LIMIT 500`,
		orgID, targetType, "%:"+targetKey,
	)
	if err != nil {
		return nil, fmt.Errorf("query suffix: %w", err)
	}
	return scanImpactCallers(rows2)
}

func scanImpactCallers(rows interface {
	Next() bool
	Scan(...any) error
	Close()
}) ([]ImpactCaller, error) {
	defer rows.Close()
	var out []ImpactCaller
	for rows.Next() {
		var c ImpactCaller
		if err := rows.Scan(&c.FromRepo, &c.FromType, &c.FromKey, &c.EdgeType, &c.EdgeProps, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func dedupeNodes(nodes []GraphNode) []GraphNode {
	seen := make(map[string]int, len(nodes))
	out := nodes[:0:0]
	for _, n := range nodes {
		k := n.Type + "\x00" + n.Key
		if idx, exists := seen[k]; exists {
			out[idx] = n
		} else {
			seen[k] = len(out)
			out = append(out, n)
		}
	}
	return out
}

func dedupeEdges(edges []GraphEdge) []GraphEdge {
	seen := make(map[string]int, len(edges))
	out := edges[:0:0]
	for _, e := range edges {
		k := e.FromType + "\x00" + e.FromKey + "\x00" + e.ToType + "\x00" + e.ToKey + "\x00" + e.Type
		if idx, exists := seen[k]; exists {
			out[idx] = e
		} else {
			seen[k] = len(out)
			out = append(out, e)
		}
	}
	return out
}

// jsonEqual compares two JSONB blobs structurally (whitespace/key-order insensitive).
func jsonEqual(a, b []byte) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		return false
	}
	aj, _ := json.Marshal(av)
	bj, _ := json.Marshal(bv)
	return string(aj) == string(bj)
}
