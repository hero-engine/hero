package graph

import "time"

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
		var validFrom time.Time
		var validTo *time.Time
		if err := rows.Scan(&typ, &key, &clientID, &validFrom, &validTo, &status); err != nil {
			return nil, err
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
