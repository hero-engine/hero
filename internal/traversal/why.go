// Package traversal implements multi-hop graph queries that read
// across subgraphs — the showcase that justifies the v2 graph
// substrate over flat files.
//
// `hero why <target>` answers "where did this come from": resolves
// the target to a node and walks origin edges (belongs_to,
// satisfied_by, attempted_in, decided_in, supersedes, mentions) in
// reverse, oldest → newest, depth-bounded.
//
// Phase 3 of traversal-queries. Phase 1+2 (`hero blocked`) live in
// internal/cli/brief.go (single-hop today; multi-hop CTE arrives
// when the Feature→Feature depends_on graph grows enough to need it).
package traversal

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// Hop is one step in a "why" trace. Each hop carries its inbound edge
// type so the rendered output reads "← <edge> → <node>".
type Hop struct {
	Depth      int    // 0 for the target itself; 1+ for ancestors
	NodeType   string // Feature / Commit / Decision / ...
	NodeKey    string
	NodeTitle  string // falls back to key when no title is set
	EdgeType   string // edge that pointed *to* this node from the previous hop
	ValidFrom  string // for chronological ordering when needed
	Subproject string // monorepo subproject scope (extracted from the spec node's props.subproject)
}

// Trace is the result of one `hero why` resolution: the target node
// plus a list of upstream hops sorted depth-ascending. Empty Chains
// means the target had no upstream edges — render with an explicit
// "no origin found" line.
type Trace struct {
	Target Hop
	Chains []Hop
}

// originEdgeTypes are the edge types `hero why` walks in reverse
// (target ← origin). Order doesn't matter for correctness but
// determines tie-break ordering in the SQL output.
//
// belongs_to is the workhorse — Criterion → Feature, sub-feature →
// initiative, etc. The remainder cover spec history, decision flow,
// and AC verification.
var originEdgeTypes = []string{
	"belongs_to",
	"satisfied_by",
	"attempted_in",
	"decided_in",
	"supersedes",
	"mentions",
	"depends_on",
	"derived_from",
	"originated_in",
	"closes",
	"fixes",
}

// DefaultDepth is the recursion bound applied when the caller doesn't
// override. Matches the v2 federation depth-cap convention captured
// in traversal-queries spec.
const DefaultDepth = 4

// Why resolves target to a node (by exact key match across types) and
// returns the chain of upstream origin hops up to maxDepth. Returns
// (nil, ErrNotFound) when no node matches.
//
// Resolution is exact-key for now — slug matching handles features,
// initiatives, decisions, criteria. Path/SHA disambiguation arrives
// in a follow-up.
func Why(store *graph.Store, repoKey, target string, maxDepth int) (*Trace, error) {
	if maxDepth <= 0 {
		maxDepth = DefaultDepth
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("traversal: empty target")
	}
	rootHop, rootID, err := resolveTarget(store, repoKey, target)
	if err != nil {
		return nil, err
	}

	chains, err := walkOrigins(store, rootID, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("walking origins: %w", err)
	}

	return &Trace{
		Target: rootHop,
		Chains: chains,
	}, nil
}

// resolveTarget returns the target's hop view + numeric node id. Tries
// the repo-scoped row first, then falls back to any matching key
// (mission, person, etc. live globally without a Repo stamp).
func resolveTarget(store *graph.Store, repoKey, target string) (Hop, int64, error) {
	row := store.DB().QueryRow(
		`SELECT id, type, key,
		        COALESCE(json_extract(props, '$.title'), key) AS title,
		        valid_from,
		        COALESCE(json_extract(props, '$.subproject'), '') AS subproject
		   FROM nodes
		  WHERE key = ? AND valid_to IS NULL AND (repo = ? OR COALESCE(repo,'') = '')
		  ORDER BY (repo = ?) DESC, ingested_at DESC
		  LIMIT 1`,
		target, repoKey, repoKey,
	)
	var h Hop
	var id int64
	if err := row.Scan(&id, &h.NodeType, &h.NodeKey, &h.NodeTitle, &h.ValidFrom, &h.Subproject); err != nil {
		if err == sql.ErrNoRows {
			return h, 0, fmt.Errorf("no node with key %q in repo %s", target, repoKey)
		}
		return h, 0, err
	}
	return h, id, nil
}

// walkOrigins follows outgoing edges of any `originEdgeTypes` from
// the root, depth-bounded by maxDepth. The "why" intuition is that a
// node exists because of what *it* points to (its parent feature, the
// note it was derived from, the commit that satisfied it). Edges
// are oriented from owner→owner-of (Criterion --belongs_to--> Feature),
// so walking outward through those edges reveals the origin chain.
//
// SQLite's recursive CTE is well-behaved here because `nodes` and
// `edges` are read-only during the query and the chain CTE only emits
// each (id, depth) once via DISTINCT in the outer SELECT.
//
// Result rows are emitted depth-ascending so the renderer can show
// the chain oldest-first (closest to the target first).
func walkOrigins(store *graph.Store, rootID int64, maxDepth int) ([]Hop, error) {
	placeholders := make([]string, 0, len(originEdgeTypes))
	args := []any{rootID}
	for _, t := range originEdgeTypes {
		placeholders = append(placeholders, "?")
		args = append(args, t)
	}
	args = append(args, maxDepth)

	q := `WITH RECURSIVE chain(id, depth, edge_type) AS (
	    SELECT ?, 0, NULL
	  UNION ALL
	    SELECT e.to_id, c.depth + 1, e.type
	      FROM edges e
	      JOIN chain c ON c.id = e.from_id
	     WHERE e.valid_to IS NULL
	       AND e.type IN (` + strings.Join(placeholders, ",") + `)
	       AND c.depth < ?
	  )
	  SELECT DISTINCT chain.depth, chain.edge_type,
	         n.type, n.key,
	         COALESCE(json_extract(n.props, '$.title'), n.key) AS title,
	         n.valid_from,
	         COALESCE(json_extract(n.props, '$.subproject'), '') AS subproject
	    FROM chain
	    JOIN nodes n ON n.id = chain.id AND n.valid_to IS NULL
	   WHERE chain.depth > 0
	   ORDER BY chain.depth, n.type, n.key`

	rows, err := store.DB().Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Hop
	for rows.Next() {
		var h Hop
		if err := rows.Scan(&h.Depth, &h.EdgeType, &h.NodeType, &h.NodeKey, &h.NodeTitle, &h.ValidFrom, &h.Subproject); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Markdown renders the trace as a human-readable trace, oldest →
// newest. When activeScope is non-empty, in-scope hops are marked with
// a leading "*" and out-of-scope hops show their scope inline.
//
// Format (no active scope):
//
//	# <target title> `<key>` (<type>)
//
//	← _<edge>_ <hop2 title> `<hop2 key>` (<hop2 type>)
//
// Format (active scope = "engines/mlx"):
//
//	# <target title> `<key>` (<type>) [scope: engines/mlx] *
//
//	* ← _<edge>_ <hop2 title> `<hop2 key>` (<hop2 type>) [scope: engines/mlx]
//	  ← _<edge>_ <hop3 title> `<hop3 key>` (<hop3 type>) [scope: engines/cuda]
//
// When there are no origin hops, prints "_(no upstream origin found)_".
func (t *Trace) Markdown() string {
	return t.MarkdownScoped("")
}

// MarkdownScoped is the scope-aware variant of Markdown. Pass empty
// activeScope to disable scope highlighting (still renders scope tags
// when hops carry one).
func (t *Trace) MarkdownScoped(activeScope string) string {
	if t == nil {
		return ""
	}
	var b strings.Builder
	mark, scopeTag := scopeBits(t.Target.Subproject, activeScope)
	fmt.Fprintf(&b, "# %s%s `%s` (%s)%s\n\n",
		mark, t.Target.NodeTitle, t.Target.NodeKey, t.Target.NodeType, scopeTag)
	if len(t.Chains) == 0 {
		b.WriteString("_(no upstream origin found within depth limit)_\n")
		return b.String()
	}
	for _, h := range t.Chains {
		indent := strings.Repeat("  ", h.Depth-1)
		hopMark, hopScopeTag := scopeBits(h.Subproject, activeScope)
		fmt.Fprintf(&b, "%s%s← _%s_ %s `%s` (%s)%s\n",
			indent, hopMark, h.EdgeType, h.NodeTitle, h.NodeKey, h.NodeType, hopScopeTag)
	}
	return b.String()
}

// scopeBits returns the leading marker and trailing scope tag for a
// hop given the active scope. When the hop's scope matches the active
// scope, the marker is "* " (in-scope highlight). When the hop has any
// scope at all, the trailing tag is " [scope: <name>]". When the hop
// has no scope and there is an active scope, the trailing tag is
// " [scope: (root)]" so the absence is visible rather than ambiguous.
func scopeBits(hopScope, activeScope string) (mark, scopeTag string) {
	if hopScope != "" {
		scopeTag = fmt.Sprintf(" [scope: %s]", hopScope)
	} else if activeScope != "" {
		scopeTag = " [scope: (root)]"
	}
	if activeScope != "" && hopScope == activeScope {
		mark = "* "
	}
	return mark, scopeTag
}
