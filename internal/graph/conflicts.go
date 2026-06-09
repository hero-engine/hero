package graph

import (
	"fmt"
	"time"
)

// graphTimeLayouts are the formats the SQLite driver may use when
// storing time.Time values. RFC3339 is what nowRFC3339() writes;
// the Go default format appears when time.Time is passed as a bound
// parameter directly (the driver calls t.String() internally).
var graphTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05 -0700 MST",
	"2006-01-02 15:04:05",
}

// parseGraphTime parses a timestamp string stored in the graph database.
// It tries each known layout in order and returns the first success.
func parseGraphTime(s string) (time.Time, error) {
	for _, layout := range graphTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised graph timestamp format: %q", s)
}

// GraphConflictVersion is one version of a node from a specific client.
type GraphConflictVersion struct {
	ClientID  string
	Status    string
	ValidFrom time.Time
	Current   bool // true if this is the currently-active version
}

// GraphConflictResult groups versions of the same (type, key) from
// different clients, indicating concurrent edits that were not
// coordinated before pushing.
type GraphConflictResult struct {
	NodeType string
	NodeKey  string
	Versions []GraphConflictVersion
}

// FindGraphConflicts looks for nodes matching slug (exact key or key
// suffix) where the bitemporal history contains versions from more than
// one distinct client_id in the last 30 days. These represent concurrent
// pushes that triggered last-write-wins on the server.
func (s *Store) FindGraphConflicts(slug string) ([]GraphConflictResult, error) {
	rows, err := s.db.Query(`
		SELECT type, key, COALESCE(client_id, ''), valid_from, valid_to,
		       COALESCE(json_extract(props, '$.status'), '') AS status
		  FROM nodes
		 WHERE (key = ? OR key LIKE '%:' || ?)
		   AND valid_from > datetime('now', '-30 days')
		   AND client_id IS NOT NULL AND client_id != ''
		 ORDER BY type, key, valid_from DESC`,
		slug, slug,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type nodeKey struct{ typ, key string }
	type entry struct {
		clientID  string
		validFrom time.Time
		validTo   *time.Time
		status    string
	}
	grouped := map[nodeKey][]entry{}
	order := []nodeKey{}

	for rows.Next() {
		var typ, key, clientID, status string
		// SQLite stores timestamps as TEXT (RFC3339); scan into strings and
		// parse rather than relying on the driver to convert automatically.
		var validFromStr string
		var validToStr *string
		if err := rows.Scan(&typ, &key, &clientID, &validFromStr, &validToStr, &status); err != nil {
			return nil, err
		}
		validFrom, err := parseGraphTime(validFromStr)
		if err != nil {
			return nil, fmt.Errorf("parsing valid_from %q: %w", validFromStr, err)
		}
		var validTo *time.Time
		if validToStr != nil {
			t, err := parseGraphTime(*validToStr)
			if err != nil {
				return nil, fmt.Errorf("parsing valid_to %q: %w", *validToStr, err)
			}
			validTo = &t
		}
		k := nodeKey{typ, key}
		if _, seen := grouped[k]; !seen {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], entry{clientID, validFrom, validTo, status})
	}

	var results []GraphConflictResult
	for _, k := range order {
		entries := grouped[k]
		// Only flag if more than one distinct client_id appears.
		clients := map[string]bool{}
		for _, e := range entries {
			clients[e.clientID] = true
		}
		if len(clients) < 2 {
			continue
		}
		var versions []GraphConflictVersion
		for _, e := range entries {
			versions = append(versions, GraphConflictVersion{
				ClientID:  e.clientID,
				Status:    e.status,
				ValidFrom: e.validFrom,
				Current:   e.validTo == nil,
			})
		}
		results = append(results, GraphConflictResult{
			NodeType: k.typ,
			NodeKey:  k.key,
			Versions: versions,
		})
	}
	return results, nil
}
