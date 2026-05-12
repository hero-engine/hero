package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// Edge is a typed relationship between two nodes. (FromID, Type, ToID)
// is the logical identity of an edge — only one current row exists for
// any such triple at a time.
type Edge struct {
	ID         int64          `json:"id"`
	FromID     int64          `json:"from_id"`
	ToID       int64          `json:"to_id"`
	Type       string         `json:"type"`
	Props      map[string]any `json:"props"`
	Scope      Scope          `json:"scope"`
	Repo       string         `json:"repo,omitempty"` // partition key — set when both endpoints are repo-bounded
	Unit       string         `json:"unit,omitempty"` // partition key — set when edge is unit-scope
	Source     map[string]any `json:"source"`
	ValidFrom  string         `json:"valid_from"`
	ValidTo    string         `json:"valid_to,omitempty"`
	IngestedAt string         `json:"ingested_at"`
}

// UpsertEdge inserts or refreshes the (from, type, to) edge.
//
// Idempotent: if a current row exists with equal props, it is left
// untouched. If props differ, the prior row is invalidated and a new
// current row is inserted.
func (s *Store) UpsertEdge(e *Edge) (int64, error) {
	if e.FromID == 0 || e.ToID == 0 || e.Type == "" {
		return 0, fmt.Errorf("graph: edge requires from_id, to_id, type")
	}
	if e.Scope == "" {
		e.Scope = ScopeTeam
	}
	if e.ValidFrom == "" {
		e.ValidFrom = nowRFC3339()
	}
	if e.IngestedAt == "" {
		e.IngestedAt = nowRFC3339()
	}

	propsJSON, err := jsonOrEmpty(e.Props, "{}")
	if err != nil {
		return 0, fmt.Errorf("marshalling props: %w", err)
	}
	sourceJSON, err := jsonOrEmpty(e.Source, "{}")
	if err != nil {
		return 0, fmt.Errorf("marshalling source: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var (
		existingID    int64
		existingProps string
		existingRepo  string
		existingUnit  string
	)
	err = tx.QueryRow(
		`SELECT id, props, repo, unit FROM edges
		  WHERE from_id = ? AND type = ? AND to_id = ? AND valid_to IS NULL`,
		e.FromID, e.Type, e.ToID,
	).Scan(&existingID, &existingProps, &existingRepo, &existingUnit)

	switch {
	case err == sql.ErrNoRows:
		// fall through
	case err != nil:
		return 0, fmt.Errorf("selecting current edge: %w", err)
	default:
		partitionUnchanged := existingRepo == e.Repo && existingUnit == e.Unit
		if partitionUnchanged && propsEqual(existingProps, propsJSON) {
			if err := tx.Commit(); err != nil {
				return 0, fmt.Errorf("commit (no-op): %w", err)
			}
			return existingID, nil
		}
		if _, err := tx.Exec(
			`UPDATE edges SET valid_to = ? WHERE id = ?`,
			e.IngestedAt, existingID,
		); err != nil {
			return 0, fmt.Errorf("invalidating prior edge: %w", err)
		}
	}

	res, err := tx.Exec(
		`INSERT INTO edges
		 (from_id, to_id, type, props, scope, repo, unit, source, valid_from, valid_to, ingested_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?)`,
		e.FromID, e.ToID, e.Type, propsJSON, string(e.Scope),
		e.Repo, e.Unit,
		sourceJSON, e.ValidFrom, e.IngestedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting edge: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	e.ID = id
	return id, nil
}

// EdgesFrom returns all current edges originating at fromID, optionally
// filtered to a single type. Pass empty string for typ to return all.
func (s *Store) EdgesFrom(fromID int64, typ string) ([]*Edge, error) {
	return s.queryEdges(
		`from_id = ?`+typeFilter(typ),
		fromID, typ,
	)
}

// EdgesTo returns all current edges targeting toID, optionally filtered
// to a single type.
func (s *Store) EdgesTo(toID int64, typ string) ([]*Edge, error) {
	return s.queryEdges(
		`to_id = ?`+typeFilter(typ),
		toID, typ,
	)
}

func typeFilter(typ string) string {
	if typ == "" {
		return ""
	}
	return ` AND type = ?`
}

func (s *Store) queryEdges(where string, idArg int64, typ string) ([]*Edge, error) {
	q := `SELECT id, from_id, to_id, type, props, scope, repo, unit, source,
	             valid_from, valid_to, ingested_at
	        FROM edges
	       WHERE valid_to IS NULL AND ` + where +
		` ORDER BY id`

	var (
		rows *sql.Rows
		err  error
	)
	if typ == "" {
		rows, err = s.db.Query(q, idArg)
	} else {
		rows, err = s.db.Query(q, idArg, typ)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanEdge(r interface{ Scan(...any) error }) (*Edge, error) {
	var (
		e          Edge
		propsJSON  string
		sourceJSON string
		validToNS  sql.NullString
		scopeStr   string
	)
	err := r.Scan(
		&e.ID, &e.FromID, &e.ToID, &e.Type, &propsJSON, &scopeStr,
		&e.Repo, &e.Unit,
		&sourceJSON, &e.ValidFrom, &validToNS, &e.IngestedAt,
	)
	if err != nil {
		return nil, err
	}
	e.Scope = Scope(scopeStr)
	if validToNS.Valid {
		e.ValidTo = validToNS.String
	}
	if err := json.Unmarshal([]byte(propsJSON), &e.Props); err != nil {
		return nil, fmt.Errorf("decoding props: %w", err)
	}
	if err := json.Unmarshal([]byte(sourceJSON), &e.Source); err != nil {
		return nil, fmt.Errorf("decoding source: %w", err)
	}
	return &e, nil
}
