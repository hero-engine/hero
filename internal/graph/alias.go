package graph

import "fmt"

// MaxAliasDepth bounds alias-chain traversal. Bad data can produce
// loops; this guarantees ResolveAlias returns even when the chain is
// pathological. Five hops is generous — real chains rarely exceed two
// (rename + cross-repo unification).
const MaxAliasDepth = 5

// ResolveAlias follows `alias_of` edges from (typ, key) to the
// canonical node and returns its id. If the input is already canonical
// (no outgoing alias_of edge), its own id is returned. repo scopes the
// starting node to a partition (see repoPredicate); alias_of edges are then
// followed by row id, which is already partition-unambiguous. Unknown nodes
// produce ErrNotFound. Cycles or chains exceeding MaxAliasDepth return
// the last seen id with no error — the partial answer is still useful
// and the caller can decide whether to flag the chain.
func (s *Store) ResolveAlias(typ, key, repo string) (int64, error) {
	id, err := s.GetNodeID(typ, key, repo)
	if err != nil {
		return 0, err
	}
	visited := map[int64]bool{id: true}

	for hop := 0; hop < MaxAliasDepth; hop++ {
		var nextID int64
		err := s.db.QueryRow(
			`SELECT to_id FROM edges
			  WHERE from_id = ? AND type = 'alias_of' AND valid_to IS NULL
			  LIMIT 1`,
			id,
		).Scan(&nextID)
		if err != nil {
			// No outgoing alias_of edge → id is canonical.
			return id, nil
		}
		if visited[nextID] {
			// Cycle — return the current id as best-effort canonical.
			return id, nil
		}
		visited[nextID] = true
		id = nextID
	}
	// Hit depth cap — return whatever we have.
	return id, nil
}

// MakeAlias declares (fromType, fromKey) as an alias of (toType, toKey).
// Both nodes must already exist in the given repo partition. Idempotent — re-declaring the same
// alias is a no-op. Updating to point at a different canonical
// invalidates the prior alias_of edge.
func (s *Store) MakeAlias(fromType, fromKey, toType, toKey, repo string) error {
	fromID, err := s.GetNodeID(fromType, fromKey, repo)
	if err != nil {
		return fmt.Errorf("alias source %s/%s: %w", fromType, fromKey, err)
	}
	toID, err := s.GetNodeID(toType, toKey, repo)
	if err != nil {
		return fmt.Errorf("alias target %s/%s: %w", toType, toKey, err)
	}
	if fromID == toID {
		return fmt.Errorf("alias source and target are the same node")
	}
	_, err = s.UpsertEdge(&Edge{
		FromID: fromID, ToID: toID, Type: "alias_of",
		Source: map[string]any{"kind": "alias-declaration"},
	})
	return err
}
