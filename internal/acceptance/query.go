package acceptance

import (
	"sort"
	"strconv"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// Criterion is the read-side projection of a Criterion graph node —
// just enough for command output. Phase-3 injection paths consume
// this; Phase 5 query verbs will too.
type Criterion struct {
	Key       string // <spec-slug>:AC-N
	ACID      string // AC-N
	Statement string
	Status    string
	Parent    string // spec slug
}

// IsOpen reports whether the AC is still open from a delivery
// perspective: anything that isn't currently passing.
func (c Criterion) IsOpen() bool {
	return c.Status != "passing"
}

// ListBySpec returns all current Criterion nodes belonging to the
// given spec slug, sorted by AC numeric suffix (AC-1, AC-2, AC-10, …).
//
// Returns an empty slice (not an error) when the spec has no
// Criterion nodes — e.g. specs that haven't been scanned with the
// AC-graph parser yet.
func ListBySpec(store *graph.Store, slug string) ([]Criterion, error) {
	if store == nil || slug == "" {
		return nil, nil
	}
	all, err := store.ListNodesByType("Criterion")
	if err != nil {
		return nil, err
	}
	prefix := slug + ":"
	var out []Criterion
	for _, n := range all {
		if !strings.HasPrefix(n.Key, prefix) {
			continue
		}
		out = append(out, criterionFromNode(n))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return acNumericKey(out[i].ACID) < acNumericKey(out[j].ACID)
	})
	return out, nil
}

// FailingAcrossCorpus returns every Criterion whose status is
// "failing" or "regressed". Drives the AC injection block in
// `hero blocked`.
func FailingAcrossCorpus(store *graph.Store) ([]Criterion, error) {
	if store == nil {
		return nil, nil
	}
	all, err := store.ListNodesByType("Criterion")
	if err != nil {
		return nil, err
	}
	var out []Criterion
	for _, n := range all {
		c := criterionFromNode(n)
		if c.Status == "failing" || c.Status == "regressed" {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Parent != out[j].Parent {
			return out[i].Parent < out[j].Parent
		}
		return acNumericKey(out[i].ACID) < acNumericKey(out[j].ACID)
	})
	return out, nil
}

// SpecStatus rolls up Criterion statuses for one parent spec into
// counts for the `hero ac status` overview.
type SpecStatus struct {
	Parent    string // spec slug
	Total     int
	Passing   int
	Failing   int
	Regressed int
	Proposed  int
	Other     int // unknown / retired / placeholder
}

// PassRate returns the fraction of ACs in this spec that are
// currently passing. Returns 0 for empty specs (callers should
// avoid emitting a percentage for those).
func (s SpecStatus) PassRate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Passing) / float64(s.Total)
}

// StatusByFeature returns a per-spec rollup of current Criterion
// statuses, sorted by parent slug. Drives `hero ac status` and the
// AC pass-rate column in `hero check`.
//
// When featureFilter is non-empty, only that spec's row is returned.
func StatusByFeature(store *graph.Store, featureFilter string) ([]SpecStatus, error) {
	if store == nil {
		return nil, nil
	}
	all, err := store.ListNodesByType("Criterion")
	if err != nil {
		return nil, err
	}
	byParent := map[string]*SpecStatus{}
	for _, n := range all {
		c := criterionFromNode(n)
		if c.Parent == "" {
			continue
		}
		if featureFilter != "" && c.Parent != featureFilter {
			continue
		}
		s, ok := byParent[c.Parent]
		if !ok {
			s = &SpecStatus{Parent: c.Parent}
			byParent[c.Parent] = s
		}
		s.Total++
		switch c.Status {
		case "passing":
			s.Passing++
		case "failing":
			s.Failing++
		case "regressed":
			s.Regressed++
		case "proposed":
			s.Proposed++
		default:
			s.Other++
		}
	}
	out := make([]SpecStatus, 0, len(byParent))
	for _, s := range byParent {
		out = append(out, *s)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Parent < out[j].Parent
	})
	return out, nil
}

// HistoryEntry is one bitemporal status row for a Criterion. The
// time-range it was current is [ValidFrom, ValidTo); the current
// row has empty ValidTo.
type HistoryEntry struct {
	Status    string
	ValidFrom string
	ValidTo   string // empty for the current row
}

// History returns every recorded row for a Criterion, oldest first.
// Each status flip produced one entry — the bitemporal record is
// preserved indefinitely and shows the full timeline of how this
// AC's verdict has moved.
//
// Returns an empty slice when the AC has no rows (typo, or never
// ingested).
func History(store *graph.Store, key string) ([]HistoryEntry, error) {
	if store == nil || key == "" {
		return nil, nil
	}
	rows, err := store.DB().Query(
		`SELECT json_extract(props, '$.status') AS status,
		        valid_from,
		        COALESCE(valid_to, '')
		   FROM nodes
		  WHERE type = 'Criterion' AND key = ?
		  ORDER BY valid_from ASC`,
		key,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HistoryEntry
	for rows.Next() {
		var status, from, to string
		if err := rows.Scan(&status, &from, &to); err != nil {
			return nil, err
		}
		out = append(out, HistoryEntry{Status: status, ValidFrom: from, ValidTo: to})
	}
	return out, rows.Err()
}

// ChangedSince returns every Criterion whose current row's valid_from
// is at or after the given RFC3339 timestamp. Useful for "since last
// session" diffs in `hero next` / `hero resume`.
//
// Empty `since` returns all current Criterion nodes (caller can decide
// what "all" means).
func ChangedSince(store *graph.Store, since string) ([]Criterion, error) {
	if store == nil {
		return nil, nil
	}
	all, err := store.ListNodesByType("Criterion")
	if err != nil {
		return nil, err
	}
	var out []Criterion
	for _, n := range all {
		if since != "" && n.ValidFrom < since {
			continue
		}
		out = append(out, criterionFromNode(n))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Parent != out[j].Parent {
			return out[i].Parent < out[j].Parent
		}
		return acNumericKey(out[i].ACID) < acNumericKey(out[j].ACID)
	})
	return out, nil
}

func criterionFromNode(n *graph.Node) Criterion {
	c := Criterion{Key: n.Key}
	if v, ok := n.Props["ac_id"].(string); ok {
		c.ACID = v
	}
	if v, ok := n.Props["statement"].(string); ok {
		c.Statement = v
	}
	if v, ok := n.Props["status"].(string); ok {
		c.Status = v
	}
	if v, ok := n.Props["parent"].(string); ok {
		c.Parent = v
	}
	if c.ACID == "" {
		// Fall back to deriving from the key so legacy rows still
		// sort correctly.
		if idx := strings.LastIndex(n.Key, ":"); idx >= 0 {
			c.ACID = n.Key[idx+1:]
		}
	}
	if c.Parent == "" {
		if idx := strings.Index(n.Key, ":"); idx > 0 {
			c.Parent = n.Key[:idx]
		}
	}
	return c
}

// acNumericKey returns the integer N from "AC-N" so sorting is
// numeric, not lexical (AC-10 must come after AC-9). Returns a large
// sentinel for malformed IDs so they sort last.
func acNumericKey(id string) int {
	id = strings.TrimPrefix(id, "AC-")
	n, err := strconv.Atoi(id)
	if err != nil {
		return 1 << 30
	}
	return n
}
