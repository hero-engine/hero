package graph

import "fmt"

// Stats summarizes the current state of the graph.
type Stats struct {
	NodesByType   map[string]int   `json:"nodes_by_type"`
	EdgesByType   map[string]int   `json:"edges_by_type"`
	NodesByScope  map[string]int   `json:"nodes_by_scope"`
	NodesByRepo   map[string]int   `json:"nodes_by_repo,omitempty"`
	NodesByUnit   map[string]int   `json:"nodes_by_unit,omitempty"`
	TotalNodes    int              `json:"total_nodes"`
	TotalEdges    int              `json:"total_edges"`
	HistoryRows   HistoryRowCounts `json:"history_rows"`
	SchemaVersion string           `json:"schema_version"`
	InstallID     string           `json:"install_id"`
}

// HistoryRowCounts captures append-only history depth.
type HistoryRowCounts struct {
	Nodes int `json:"nodes"`
	Edges int `json:"edges"`
}

// Stats computes counts over current rows plus the meta fields. It does
// not materialize node/edge contents, so it is safe on large graphs.
func (s *Store) Stats() (*Stats, error) {
	stats := &Stats{
		NodesByType:  map[string]int{},
		EdgesByType:  map[string]int{},
		NodesByScope: map[string]int{},
		NodesByRepo:  map[string]int{},
		NodesByUnit:  map[string]int{},
	}

	// Counts of current nodes by type.
	rows, err := s.db.Query(
		`SELECT type, COUNT(*) FROM nodes
		  WHERE valid_to IS NULL GROUP BY type`)
	if err != nil {
		return nil, fmt.Errorf("nodes by type: %w", err)
	}
	for rows.Next() {
		var t string
		var n int
		if err := rows.Scan(&t, &n); err != nil {
			rows.Close()
			return nil, err
		}
		stats.NodesByType[t] = n
		stats.TotalNodes += n
	}
	rows.Close()

	rows, err = s.db.Query(
		`SELECT scope, COUNT(*) FROM nodes
		  WHERE valid_to IS NULL GROUP BY scope`)
	if err != nil {
		return nil, fmt.Errorf("nodes by scope: %w", err)
	}
	for rows.Next() {
		var sc string
		var n int
		if err := rows.Scan(&sc, &n); err != nil {
			rows.Close()
			return nil, err
		}
		stats.NodesByScope[sc] = n
	}
	rows.Close()

	rows, err = s.db.Query(
		`SELECT repo, COUNT(*) FROM nodes
		  WHERE valid_to IS NULL AND repo != '' GROUP BY repo`)
	if err != nil {
		return nil, fmt.Errorf("nodes by repo: %w", err)
	}
	for rows.Next() {
		var r string
		var n int
		if err := rows.Scan(&r, &n); err != nil {
			rows.Close()
			return nil, err
		}
		stats.NodesByRepo[r] = n
	}
	rows.Close()

	rows, err = s.db.Query(
		`SELECT unit, COUNT(*) FROM nodes
		  WHERE valid_to IS NULL AND unit != '' GROUP BY unit`)
	if err != nil {
		return nil, fmt.Errorf("nodes by unit: %w", err)
	}
	for rows.Next() {
		var u string
		var n int
		if err := rows.Scan(&u, &n); err != nil {
			rows.Close()
			return nil, err
		}
		stats.NodesByUnit[u] = n
	}
	rows.Close()

	rows, err = s.db.Query(
		`SELECT type, COUNT(*) FROM edges
		  WHERE valid_to IS NULL GROUP BY type`)
	if err != nil {
		return nil, fmt.Errorf("edges by type: %w", err)
	}
	for rows.Next() {
		var t string
		var n int
		if err := rows.Scan(&t, &n); err != nil {
			rows.Close()
			return nil, err
		}
		stats.EdgesByType[t] = n
		stats.TotalEdges += n
	}
	rows.Close()

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&stats.HistoryRows.Nodes); err != nil {
		return nil, fmt.Errorf("history nodes: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&stats.HistoryRows.Edges); err != nil {
		return nil, fmt.Errorf("history edges: %w", err)
	}

	if err := s.db.QueryRow(
		`SELECT value FROM meta WHERE key = 'schema_version'`,
	).Scan(&stats.SchemaVersion); err != nil {
		return nil, fmt.Errorf("schema version: %w", err)
	}
	if err := s.db.QueryRow(
		`SELECT value FROM meta WHERE key = 'install_id'`,
	).Scan(&stats.InstallID); err != nil {
		return nil, fmt.Errorf("install id: %w", err)
	}

	return stats, nil
}
