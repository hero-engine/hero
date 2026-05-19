package graph

import (
	"fmt"
)

// DomainStats summarizes graph rows grouped by their `domain` namespace
// column. Returned by `hero domain verify` and the rollback dry-run.
type DomainStats struct {
	NodesByDomain     map[string]int            `json:"nodes_by_domain"`
	EdgesByDomain     map[string]int            `json:"edges_by_domain"`
	NodesByTypeDomain map[string]map[string]int `json:"nodes_by_type_domain"`
	EdgesByTypeDomain map[string]map[string]int `json:"edges_by_type_domain"`
	CrossDomainEdges  []CrossDomainEdgeGroup    `json:"cross_domain_edges"`
}

// CrossDomainEdgeGroup counts current edges whose endpoint domains differ.
type CrossDomainEdgeGroup struct {
	FromDomain string `json:"from_domain"`
	ToDomain   string `json:"to_domain"`
	Kind       string `json:"kind"`
	Count      int    `json:"count"`
}

// DomainStats returns row counts grouped by the `domain` column. Only
// callable against a v3+ schema; an older schema returns a clear error
// because the column does not exist.
func (s *Store) DomainStats() (*DomainStats, error) {
	stats := &DomainStats{
		NodesByDomain:     map[string]int{},
		EdgesByDomain:     map[string]int{},
		NodesByTypeDomain: map[string]map[string]int{},
		EdgesByTypeDomain: map[string]map[string]int{},
	}

	rows, err := s.db.Query(
		`SELECT domain, COUNT(*) FROM nodes
		  WHERE valid_to IS NULL GROUP BY domain`)
	if err != nil {
		return nil, fmt.Errorf("nodes by domain: %w", err)
	}
	for rows.Next() {
		var d string
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			rows.Close()
			return nil, err
		}
		stats.NodesByDomain[d] = n
	}
	rows.Close()

	rows, err = s.db.Query(
		`SELECT domain, COUNT(*) FROM edges
		  WHERE valid_to IS NULL GROUP BY domain`)
	if err != nil {
		return nil, fmt.Errorf("edges by domain: %w", err)
	}
	for rows.Next() {
		var d string
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			rows.Close()
			return nil, err
		}
		stats.EdgesByDomain[d] = n
	}
	rows.Close()

	rows, err = s.db.Query(
		`SELECT type, domain, COUNT(*) FROM nodes
		  WHERE valid_to IS NULL GROUP BY type, domain`)
	if err != nil {
		return nil, fmt.Errorf("nodes by type+domain: %w", err)
	}
	for rows.Next() {
		var t, d string
		var n int
		if err := rows.Scan(&t, &d, &n); err != nil {
			rows.Close()
			return nil, err
		}
		if stats.NodesByTypeDomain[t] == nil {
			stats.NodesByTypeDomain[t] = map[string]int{}
		}
		stats.NodesByTypeDomain[t][d] = n
	}
	rows.Close()

	rows, err = s.db.Query(
		`SELECT type, domain, COUNT(*) FROM edges
		  WHERE valid_to IS NULL GROUP BY type, domain`)
	if err != nil {
		return nil, fmt.Errorf("edges by type+domain: %w", err)
	}
	for rows.Next() {
		var t, d string
		var n int
		if err := rows.Scan(&t, &d, &n); err != nil {
			rows.Close()
			return nil, err
		}
		if stats.EdgesByTypeDomain[t] == nil {
			stats.EdgesByTypeDomain[t] = map[string]int{}
		}
		stats.EdgesByTypeDomain[t][d] = n
	}
	rows.Close()

	rows, err = s.db.Query(
		`SELECT f.domain, t.domain, e.type, COUNT(*)
		   FROM edges e
		   JOIN nodes f ON f.id = e.from_id
		   JOIN nodes t ON t.id = e.to_id
		  WHERE e.valid_to IS NULL
		    AND f.valid_to IS NULL
		    AND t.valid_to IS NULL
		    AND f.domain != t.domain
		  GROUP BY f.domain, t.domain, e.type`)
	if err != nil {
		return nil, fmt.Errorf("cross-domain edges: %w", err)
	}
	for rows.Next() {
		var g CrossDomainEdgeGroup
		if err := rows.Scan(&g.FromDomain, &g.ToDomain, &g.Kind, &g.Count); err != nil {
			rows.Close()
			return nil, err
		}
		stats.CrossDomainEdges = append(stats.CrossDomainEdges, g)
	}
	rows.Close()

	return stats, nil
}

// NonEngineeringRowCount reports how many current rows carry a domain
// other than "engineering". Used by the v3 rollback dry-run to surface
// data loss: rolling back drops the column, so any non-engineering
// tag is discarded.
func (s *Store) NonEngineeringRowCount() (nodes int, edges int, err error) {
	if err = s.db.QueryRow(
		`SELECT COUNT(*) FROM nodes
		  WHERE valid_to IS NULL AND domain != 'engineering'`,
	).Scan(&nodes); err != nil {
		return 0, 0, fmt.Errorf("non-engineering nodes: %w", err)
	}
	if err = s.db.QueryRow(
		`SELECT COUNT(*) FROM edges
		  WHERE valid_to IS NULL AND domain != 'engineering'`,
	).Scan(&edges); err != nil {
		return 0, 0, fmt.Errorf("non-engineering edges: %w", err)
	}
	return nodes, edges, nil
}

// HandoffEdge is one row in the cross-domain handoff stream view.
// Carries both endpoint domains so the dashboard widget can render
// the boundary label without an extra round-trip.
type HandoffEdge struct {
	EdgeID     int64  `json:"edge_id"`
	Kind       string `json:"kind"`
	FromType   string `json:"from_type"`
	FromKey    string `json:"from_key"`
	FromTitle  string `json:"from_title"`
	FromDomain string `json:"from_domain"`
	ToType     string `json:"to_type"`
	ToKey      string `json:"to_key"`
	ToTitle    string `json:"to_title"`
	ToDomain   string `json:"to_domain"`
	ValidFrom  string `json:"valid_from"`
}

// HandoffStream returns every current `handoff`-kind edge regardless
// of the workspace's active domain. The DSKG spec's audit table marks
// the Handoff stream dashboard widget as boundary-aware (always) —
// this is the data contract hero-code consumes for the brand-demo
// widget.
//
// onlyCrossDomain narrows to edges whose endpoints differ in domain.
// Pass false to include same-domain handoffs (rare but valid — a PM
// story handed off to another PM agent, for example).
func (s *Store) HandoffStream(onlyCrossDomain bool) ([]HandoffEdge, error) {
	q := `SELECT e.id, e.type,
	             f.type, f.key,
	             COALESCE(json_extract(f.props, '$.title'), f.key),
	             f.domain,
	             t.type, t.key,
	             COALESCE(json_extract(t.props, '$.title'), t.key),
	             t.domain,
	             e.valid_from
	        FROM edges e
	        JOIN nodes f ON f.id = e.from_id AND f.valid_to IS NULL
	        JOIN nodes t ON t.id = e.to_id   AND t.valid_to IS NULL
	       WHERE e.valid_to IS NULL
	         AND e.type = 'handoff'`
	if onlyCrossDomain {
		q += ` AND f.domain != t.domain`
	}
	q += ` ORDER BY e.valid_from DESC`

	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("handoff stream: %w", err)
	}
	defer rows.Close()
	var out []HandoffEdge
	for rows.Next() {
		var h HandoffEdge
		if err := rows.Scan(
			&h.EdgeID, &h.Kind,
			&h.FromType, &h.FromKey, &h.FromTitle, &h.FromDomain,
			&h.ToType, &h.ToKey, &h.ToTitle, &h.ToDomain,
			&h.ValidFrom,
		); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// CrossDomainUnusualKindWarning is one entry in the
// cross_domain_unusual_kind warning surfaced by `hero warnings`. The
// DSKG design treats edges with mismatched endpoint domains as valid
// when the kind is in crossDomainAllowedKinds; otherwise they get
// flagged so the contract evolves visibly rather than silently.
type CrossDomainUnusualKindWarning struct {
	EdgeID     int64  `json:"edge_id"`
	Kind       string `json:"kind"`
	FromKey    string `json:"from_key"`
	FromType   string `json:"from_type"`
	FromDomain string `json:"from_domain"`
	ToKey      string `json:"to_key"`
	ToType     string `json:"to_type"`
	ToDomain   string `json:"to_domain"`
}

// CrossDomainUnusualKindWarnings scans current edges, joins both
// endpoints, and returns any cross-domain edges whose kind is not in
// crossDomainAllowedKinds. Computed on demand — there is no separate
// warnings table; the warning is a property of live graph state.
func (s *Store) CrossDomainUnusualKindWarnings() ([]CrossDomainUnusualKindWarning, error) {
	rows, err := s.db.Query(
		`SELECT e.id, e.type, f.key, f.type, f.domain, t.key, t.type, t.domain
		   FROM edges e
		   JOIN nodes f ON f.id = e.from_id AND f.valid_to IS NULL
		   JOIN nodes t ON t.id = e.to_id   AND t.valid_to IS NULL
		  WHERE e.valid_to IS NULL
		    AND f.domain != t.domain
		    AND f.domain != '' AND t.domain != ''`)
	if err != nil {
		return nil, fmt.Errorf("cross-domain warnings: %w", err)
	}
	defer rows.Close()
	var out []CrossDomainUnusualKindWarning
	for rows.Next() {
		var w CrossDomainUnusualKindWarning
		if err := rows.Scan(&w.EdgeID, &w.Kind, &w.FromKey, &w.FromType, &w.FromDomain, &w.ToKey, &w.ToType, &w.ToDomain); err != nil {
			return nil, err
		}
		if IsCrossDomainAllowedKind(w.Kind) {
			continue
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// RollbackV3 reverts the schema-v3 migration: drops the `domain`
// indexes and columns and resets `meta.schema_version` to "2". SQLite
// 3.35+ supports `ALTER TABLE ... DROP COLUMN` without rewriting the
// table; the drop is non-data-corrupting (other columns preserved).
// The only side effect is that any post-migration writes that stamped
// a non-engineering domain lose that tag — call NonEngineeringRowCount
// first if you want to surface that to the operator.
//
// Safe to call against a v3 db. Against an older or already-rolled-back
// db, the DROP statements fail and the function returns the error;
// callers should check `meta.schema_version` before invoking.
func (s *Store) RollbackV3() error {
	stmts := []string{
		`DROP INDEX IF EXISTS idx_nodes_domain`,
		`DROP INDEX IF EXISTS idx_edges_domain`,
		`ALTER TABLE nodes DROP COLUMN domain`,
		`ALTER TABLE edges DROP COLUMN domain`,
		`UPDATE meta SET value = '2' WHERE key = 'schema_version'`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("rollback v3: %s: %w", stmt, err)
		}
	}
	return nil
}
