package tasks

import (
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// Record is the read-side projection of a Task graph node — just
// enough for command output and rollups. Distinct from the parsed
// Task: a Record is what's currently in the graph, which may lead or
// lag what's on disk depending on when the spec was scanned.
type Record struct {
	Key               string // <spec-slug>:T-N
	TaskID            string // T-N
	Text              string
	Status            string
	Parent            string // spec slug
	Kind              string
	Assignee          string
	DiscoveredAgainst string
	Started           string
	Done              string
}

// IsOpen reports whether this task is still actionable.
func (r Record) IsOpen() bool {
	return r.Status != StatusDone
}

// ListBySpec returns all current Task nodes belonging to slug, sorted
// by numeric T-N. Returns an empty slice when the spec has no tasks.
func ListBySpec(store *graph.Store, slug string) ([]Record, error) {
	if store == nil || slug == "" {
		return nil, nil
	}
	all, err := store.ListNodesByType("Task")
	if err != nil {
		return nil, err
	}
	prefix := slug + ":"
	var out []Record
	for _, n := range all {
		if !strings.HasPrefix(n.Key, prefix) {
			continue
		}
		out = append(out, recordFromNode(n))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return numericKey(out[i].TaskID) < numericKey(out[j].TaskID)
	})
	return out, nil
}

// OpenAcrossCorpus returns every Task whose status is not "done".
// Drives the kind of "what's still on my plate" rollups callers want
// from the task surface — analogous to FailingAcrossCorpus for ACs
// (and intentionally not 1:1 because tasks are open/closed, not
// pass/fail).
func OpenAcrossCorpus(store *graph.Store) ([]Record, error) {
	if store == nil {
		return nil, nil
	}
	all, err := store.ListNodesByType("Task")
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, n := range all {
		r := recordFromNode(n)
		if r.Status != StatusDone {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Parent != out[j].Parent {
			return out[i].Parent < out[j].Parent
		}
		return numericKey(out[i].TaskID) < numericKey(out[j].TaskID)
	})
	return out, nil
}

// SpecStatus rolls up Task statuses for one parent spec into counts
// for the `hero task status` overview.
type SpecStatus struct {
	Parent string // spec slug
	Total  int
	Todo   int
	Doing  int
	Done   int
	Other  int // any non-canonical status
}

// CompletionRate returns the fraction of tasks marked done. Returns 0
// for empty specs.
func (s SpecStatus) CompletionRate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Done) / float64(s.Total)
}

// StatusByFeature returns a per-spec rollup sorted by parent slug.
// When featureFilter is non-empty, only that spec's row is returned.
func StatusByFeature(store *graph.Store, featureFilter string) ([]SpecStatus, error) {
	if store == nil {
		return nil, nil
	}
	all, err := store.ListNodesByType("Task")
	if err != nil {
		return nil, err
	}
	byParent := map[string]*SpecStatus{}
	for _, n := range all {
		r := recordFromNode(n)
		if r.Parent == "" {
			continue
		}
		if featureFilter != "" && r.Parent != featureFilter {
			continue
		}
		s, ok := byParent[r.Parent]
		if !ok {
			s = &SpecStatus{Parent: r.Parent}
			byParent[r.Parent] = s
		}
		s.Total++
		switch r.Status {
		case StatusTodo:
			s.Todo++
		case StatusDoing:
			s.Doing++
		case StatusDone:
			s.Done++
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

// HistoryEntry is one bitemporal row for a Task: the time-range it
// was current is [ValidFrom, ValidTo); the current row has empty
// ValidTo.
type HistoryEntry struct {
	Status    string
	ValidFrom string
	ValidTo   string
}

// History returns every recorded row for a Task, oldest first. Each
// flip (todo → doing → done) produced one entry — the bitemporal
// record is preserved indefinitely.
func History(store *graph.Store, key string) ([]HistoryEntry, error) {
	if store == nil || key == "" {
		return nil, nil
	}
	rows, err := store.DB().Query(
		`SELECT json_extract(props, '$.status') AS status,
		        valid_from,
		        COALESCE(valid_to, '')
		   FROM nodes
		  WHERE type = 'Task' AND key = ?
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

// recordFromNode pulls the canonical fields off a Task graph node. All
// fields are optional except Key, which the graph guarantees.
func recordFromNode(n *graph.Node) Record {
	r := Record{Key: n.Key}
	if v, ok := n.Props["task_id"].(string); ok {
		r.TaskID = v
	}
	if v, ok := n.Props["text"].(string); ok {
		r.Text = v
	}
	if v, ok := n.Props["status"].(string); ok {
		r.Status = v
	}
	if v, ok := n.Props["parent"].(string); ok {
		r.Parent = v
	}
	if v, ok := n.Props["kind"].(string); ok {
		r.Kind = v
	}
	if v, ok := n.Props["assignee"].(string); ok {
		r.Assignee = v
	}
	if v, ok := n.Props["discovered_against"].(string); ok {
		r.DiscoveredAgainst = v
	}
	if v, ok := n.Props["started"].(string); ok {
		r.Started = v
	}
	if v, ok := n.Props["done"].(string); ok {
		r.Done = v
	}
	if r.TaskID == "" {
		if idx := strings.LastIndex(n.Key, ":"); idx >= 0 {
			r.TaskID = n.Key[idx+1:]
		}
	}
	if r.Parent == "" {
		if idx := strings.Index(n.Key, ":"); idx > 0 {
			r.Parent = n.Key[:idx]
		}
	}
	return r
}
