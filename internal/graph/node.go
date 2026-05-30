package graph

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// Node is a logical entity in the graph. (Type, Key) is the stable
// logical identity; ID is the internal row id and is not stable across
// re-ingests on different machines.
type Node struct {
	ID          int64          `json:"id"`
	Type        string         `json:"type"`
	Key         string         `json:"key"`
	Props       map[string]any `json:"props"`
	Scope       Scope          `json:"scope"`
	Repo        string         `json:"repo,omitempty"`   // partition key — repo this node belongs to
	Unit        string         `json:"unit,omitempty"`   // partition key — business unit / product line
	Domain      string         `json:"domain,omitempty"` // namespace tag — empty = global (Mission, Person, etc.)
	ContentHash string         `json:"content_hash,omitempty"`
	Source      map[string]any `json:"source"`
	ValidFrom   string         `json:"valid_from"`
	ValidTo     string         `json:"valid_to,omitempty"`
	IngestedAt  string         `json:"ingested_at"`
}

// ErrNotFound is returned when a lookup matches no current row.
var ErrNotFound = errors.New("graph: not found")

// ErrDomainRequired is returned by UpsertNode when Domain is empty AND
// the node type is not in globalNodeTypes. Catches ingest paths that
// forgot to stamp the namespace partition.
var ErrDomainRequired = errors.New("graph: domain required (node type not in global allow-list)")

// ErrDomainMutation is returned by UpsertNode when an existing
// (type, key) row has a different domain than the incoming write.
// First write wins; relocating a node across domains is a v2 retag
// concern, not an ingest concern.
var ErrDomainMutation = errors.New("graph: cannot change a node's domain via upsert")

// globalNodeTypes is the allow-list of node types whose Domain may be
// "" (workspace-wide / federation-wide rather than per-domain). Adding
// a type here is a deliberate decision — most node types belong to
// exactly one domain.
var globalNodeTypes = map[string]struct{}{
	"Mission": {}, // workspace-wide brief; no domain
	"Person":  {}, // people are shared across domains in v1
	"Org":     {}, // federation partition, not domain partition
	"Repo":    {}, // federation partition, not domain partition
	"Unit":    {}, // federation partition, not domain partition
}

// IsGlobalNodeType reports whether the type is exempt from the domain
// invariant — i.e. Domain == "" is permitted on upsert.
func IsGlobalNodeType(typ string) bool {
	_, ok := globalNodeTypes[typ]
	return ok
}

// UpsertNode inserts or updates the current row for (type, key).
//
// Behavior:
//   - If no current row exists, a new one is inserted.
//   - If a current row exists with the same content_hash and props,
//     it is left untouched (idempotent — safe to call repeatedly).
//   - Otherwise the existing current row is invalidated (valid_to set
//     to now) and a new current row is inserted with the new values.
//
// The caller supplies n.Type, n.Key, n.Props, n.Scope, n.Source, and
// optionally n.ContentHash. valid_from / ingested_at default to now
// if empty. ID and valid_to are managed by the store.
func (s *Store) UpsertNode(n *Node) (int64, error) {
	if n.Type == "" || n.Key == "" {
		return 0, fmt.Errorf("graph: node requires type and key")
	}
	if n.Domain == "" && !IsGlobalNodeType(n.Type) {
		return 0, fmt.Errorf("%w: type=%q key=%q", ErrDomainRequired, n.Type, n.Key)
	}
	if n.Scope == "" {
		n.Scope = ScopeTeam
	}
	if n.ValidFrom == "" {
		n.ValidFrom = nowRFC3339()
	}
	if n.IngestedAt == "" {
		n.IngestedAt = nowRFC3339()
	}

	propsJSON, err := jsonOrEmpty(n.Props, "{}")
	if err != nil {
		return 0, fmt.Errorf("marshalling props: %w", err)
	}
	sourceJSON, err := jsonOrEmpty(n.Source, "{}")
	if err != nil {
		return 0, fmt.Errorf("marshalling source: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Look up the current row, if any.
	var (
		existingID     int64
		existingHash   sql.NullString
		existingProps  string
		existingRepo   string
		existingUnit   string
		existingDomain string
	)
	err = tx.QueryRow(
		`SELECT id, content_hash, props, repo, unit, domain
		   FROM nodes
		  WHERE type = ? AND key = ? AND valid_to IS NULL`,
		n.Type, n.Key,
	).Scan(&existingID, &existingHash, &existingProps, &existingRepo, &existingUnit, &existingDomain)

	switch {
	case err == sql.ErrNoRows:
		// fall through — insert new
	case err != nil:
		return 0, fmt.Errorf("selecting current: %w", err)
	default:
		// First write wins on domain: relocating a node across domains
		// is a v2 retag concern, not an upsert concern. Catch the trap
		// at the write site rather than silently flipping the tag.
		//
		// Exception: global node types (Person, Repo, etc.) may have been
		// created with a domain before they were added to the global
		// allow-list. Allow the upsert to correct them to domain="".
		if existingDomain != n.Domain {
			if IsGlobalNodeType(n.Type) && n.Domain == "" {
				// Self-healing: let the global node type shed its stale
				// domain tag. The invalidate-and-reinsert below handles it.
			} else {
				return 0, fmt.Errorf("%w: type=%q key=%q existing=%q new=%q",
					ErrDomainMutation, n.Type, n.Key, existingDomain, n.Domain)
			}
		}
		// Partition columns must match for an upsert to be a no-op,
		// even if content is otherwise unchanged. This makes the v1→v2
		// backfill idempotent: existing rows with empty repo/unit get
		// invalidated and replaced when the ingest path now stamps
		// them. Domain is included in the partition check for
		// consistency, though ErrDomainMutation above means it always
		// matches when we get here.
		partitionUnchanged := existingRepo == n.Repo &&
			existingUnit == n.Unit &&
			existingDomain == n.Domain

		if partitionUnchanged && n.ContentHash != "" && existingHash.Valid && existingHash.String == n.ContentHash {
			if err := tx.Commit(); err != nil {
				return 0, fmt.Errorf("commit (no-op): %w", err)
			}
			return existingID, nil
		}
		if partitionUnchanged && n.ContentHash == "" && !existingHash.Valid && propsEqual(existingProps, propsJSON) {
			if err := tx.Commit(); err != nil {
				return 0, fmt.Errorf("commit (no-op): %w", err)
			}
			return existingID, nil
		}
		// Otherwise invalidate the prior row.
		if _, err := tx.Exec(
			`UPDATE nodes SET valid_to = ? WHERE id = ?`,
			n.IngestedAt, existingID,
		); err != nil {
			return 0, fmt.Errorf("invalidating prior node: %w", err)
		}
		// Also invalidate any edges hanging off the prior version.
		// (Edges are re-asserted by the next ingest pass.)
		if _, err := tx.Exec(
			`UPDATE edges SET valid_to = ?
			  WHERE valid_to IS NULL AND (from_id = ? OR to_id = ?)`,
			n.IngestedAt, existingID, existingID,
		); err != nil {
			return 0, fmt.Errorf("invalidating prior edges: %w", err)
		}
	}

	res, err := tx.Exec(
		`INSERT INTO nodes
		 (type, key, props, scope, repo, unit, domain, content_hash, source, valid_from, valid_to, ingested_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?)`,
		n.Type, n.Key, propsJSON, string(n.Scope),
		n.Repo, n.Unit, n.Domain,
		nullableString(n.ContentHash), sourceJSON,
		n.ValidFrom, n.IngestedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting node: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	n.ID = id
	return id, nil
}

// GetNodeAt returns the row that was current for (type, key) at the
// given RFC3339 timestamp — the bitemporal "what was true at time t"
// query. A row is considered current at t when valid_from ≤ t and
// (valid_to IS NULL OR valid_to > t).
//
// Returns ErrNotFound when no row matched (e.g. asking before the
// node ever existed). Useful for verifying that a status flip was
// recorded with bitemporal correctness.
func (s *Store) GetNodeAt(typ, key, at string) (*Node, error) {
	row := s.db.QueryRow(
		`SELECT id, type, key, props, scope, repo, unit, domain, content_hash, source,
		        valid_from, valid_to, ingested_at
		   FROM nodes
		  WHERE type = ? AND key = ?
		    AND valid_from <= ?
		    AND (valid_to IS NULL OR valid_to > ?)
		  ORDER BY valid_from DESC
		  LIMIT 1`,
		typ, key, at, at,
	)
	return scanNode(row)
}

// GetNode returns the current row for (type, key), or ErrNotFound.
func (s *Store) GetNode(typ, key string) (*Node, error) {
	row := s.db.QueryRow(
		`SELECT id, type, key, props, scope, repo, unit, domain, content_hash, source,
		        valid_from, valid_to, ingested_at
		   FROM nodes
		  WHERE type = ? AND key = ? AND valid_to IS NULL`,
		typ, key,
	)
	return scanNode(row)
}

// GetNodeID returns the current node's row id for (type, key).
func (s *Store) GetNodeID(typ, key string) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM nodes
		  WHERE type = ? AND key = ? AND valid_to IS NULL`,
		typ, key,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	return id, err
}

// InvalidateNode marks the current row for (type, key) as no longer
// valid. Returns ErrNotFound if no current row exists.
func (s *Store) InvalidateNode(typ, key string) error {
	now := nowRFC3339()
	res, err := s.db.Exec(
		`UPDATE nodes SET valid_to = ?
		  WHERE type = ? AND key = ? AND valid_to IS NULL`,
		now, typ, key,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListNodesByType returns all current nodes of the given type, sorted
// by key. Pass empty string for typ to return all current nodes.
func (s *Store) ListNodesByType(typ string) ([]*Node, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if typ == "" {
		rows, err = s.db.Query(
			`SELECT id, type, key, props, scope, repo, unit, domain, content_hash, source,
			        valid_from, valid_to, ingested_at
			   FROM nodes
			  WHERE valid_to IS NULL
			  ORDER BY type, key`,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, type, key, props, scope, repo, unit, domain, content_hash, source,
			        valid_from, valid_to, ingested_at
			   FROM nodes
			  WHERE type = ? AND valid_to IS NULL
			  ORDER BY key`,
			typ,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ScanNode reads a row in the nodes column order into a *Node.
// Exported so callers that need to filter rows beyond what the
// built-in helpers offer (e.g. handoff's repo-scoped singleton and
// reflection queries) can run a custom SELECT against Store.DB()
// and reuse the canonical column shape and decode logic.
//
// Column order required: id, type, key, props, scope, repo, unit,
// domain, content_hash, source, valid_from, valid_to, ingested_at.
func ScanNode(r interface {
	Scan(...any) error
}) (*Node, error) {
	return scanNode(r)
}

// scanNode reads a row in the nodes column order into a *Node.
func scanNode(r interface {
	Scan(...any) error
}) (*Node, error) {
	var (
		n            Node
		propsJSON    string
		sourceJSON   string
		contentHash  sql.NullString
		validToNS    sql.NullString
		scopeStr     string
	)
	err := r.Scan(
		&n.ID, &n.Type, &n.Key, &propsJSON, &scopeStr,
		&n.Repo, &n.Unit, &n.Domain,
		&contentHash, &sourceJSON,
		&n.ValidFrom, &validToNS, &n.IngestedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	n.Scope = Scope(scopeStr)
	if contentHash.Valid {
		n.ContentHash = contentHash.String
	}
	if validToNS.Valid {
		n.ValidTo = validToNS.String
	}
	if err := json.Unmarshal([]byte(propsJSON), &n.Props); err != nil {
		return nil, fmt.Errorf("decoding props: %w", err)
	}
	if err := json.Unmarshal([]byte(sourceJSON), &n.Source); err != nil {
		return nil, fmt.Errorf("decoding source: %w", err)
	}
	return &n, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// propsEqual compares two JSON-encoded prop blobs by structural equality.
func propsEqual(a, b string) bool {
	var ax, bx any
	if err := json.Unmarshal([]byte(a), &ax); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bx); err != nil {
		return false
	}
	aj, err := json.Marshal(ax)
	if err != nil {
		return false
	}
	bj, err := json.Marshal(bx)
	if err != nil {
		return false
	}
	return string(aj) == string(bj)
}
