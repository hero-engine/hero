// Package tasks parses, persists, and queries the `## Tasks` sub-element
// on work-shaped specs. Mirrors `internal/acceptance/` in shape, lives
// strictly beside it; the AC infrastructure is unchanged.
//
// Canonical task line shape (per unified-spec-type-model Decision 3):
//
//	## Tasks
//
//	- [ ] T-1 Fix login redirect loop {kind: qa-blocker, assignee: chet, discovered_against: checkout-flow}
//	- [x] T-2 Migrate token storage to keychain {kind: chore, done: 2026-05-15T14:22:00Z}
//	- [/] T-3 Wire up retry-with-backoff {assignee: bwheeler, started: 2026-05-16T09:00:00Z}
//
// `- [ ]` = todo; `- [/]` = doing; `- [x]` = done. The trailing
// `{...}` block is YAML-flow-shorthand metadata; every field is
// optional. IDs are author-assigned (`T-<int>`) and monotonically
// increasing per spec.
package tasks

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Task is one parsed entry from a spec's `## Tasks` section.
type Task struct {
	// ID is the author-assigned identifier, e.g. "T-3".
	ID string
	// Text is the human-readable statement following the ID.
	Text string
	// Status is "todo" | "doing" | "done", derived from the checkbox.
	Status string

	// Inline metadata fields (optional; empty when omitted).
	Kind              string
	Assignee          string
	DiscoveredAgainst string
	Started           string // RFC3339 if parseable; raw string otherwise
	Done              string // RFC3339 if parseable; raw string otherwise
}

// StatusTodo / StatusDoing / StatusDone are the canonical status strings
// emitted by the parser and recognized by Record. Keep these as exported
// constants so callers don't sprinkle string literals.
const (
	StatusTodo  = "todo"
	StatusDoing = "doing"
	StatusDone  = "done"
)

// IsOpen reports whether the task still requires action.
func (t Task) IsOpen() bool {
	return t.Status != StatusDone
}

// taskLinePattern matches the leading checkbox + ID portion of a task
// line. Captured groups:
//
//	1: status marker (one of " ", "/", "x", "X")
//	2: numeric N from "T-N"
//
// Examples that match:
//
//	- [ ] T-1 …
//	- [x] T-2 …
//	- [/] T-3 …
//	* [X] T-10 …
var taskLinePattern = regexp.MustCompile(`^\s*[-*]\s+\[([ \/xX])\]\s+T-(\d+)\s*`)

// metaPattern matches an optional trailing `{...}` block of YAML-flow
// shorthand metadata at the end of a task line.
var metaPattern = regexp.MustCompile(`\{([^}]*)\}\s*$`)

// ParseTasks scans the body of a `## Tasks` section and returns the
// structured task entries it contains. Tasks are returned in source
// order; duplicate IDs keep the first occurrence (parser is
// deterministic — authoring tools should not create duplicates).
//
// Returns nil if the body has no task lines.
func ParseTasks(body string) []Task {
	if body == "" {
		return nil
	}
	var out []Task
	seen := map[string]bool{}
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimRight(raw, "\r")
		m := taskLinePattern.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		marker := line[m[2]:m[3]]
		idNum := line[m[4]:m[5]]
		rest := strings.TrimSpace(line[m[1]:])

		t := Task{
			ID:     "T-" + idNum,
			Status: markerToStatus(marker),
		}
		// Pull the metadata trailer off the end if present.
		if meta := metaPattern.FindStringSubmatchIndex(rest); meta != nil {
			body := rest[meta[2]:meta[3]]
			applyMeta(&t, body)
			rest = strings.TrimSpace(rest[:meta[0]])
		}
		t.Text = normalizeText(rest)

		// Done-state without an explicit `done:` field still implies
		// done; same for doing → started. Don't fabricate a timestamp
		// (we don't know it), just leave the field empty.
		if seen[t.ID] {
			continue
		}
		seen[t.ID] = true
		out = append(out, t)
	}
	return out
}

// FindSection returns the body of the `## Tasks` section from a parsed
// spec's Sections map (which lowercases its keys, matching how
// `internal/spec.Spec.parseSections` populates it). Returns the empty
// string when the section is missing.
//
// Accepts the map directly rather than the Spec type so callers don't
// pull in the spec package when they only need the section body.
func FindSection(sections map[string]string) string {
	if sections == nil {
		return ""
	}
	for name, body := range sections {
		if strings.ToLower(strings.TrimSpace(name)) == "tasks" {
			return body
		}
	}
	return ""
}

// NextID returns the next author-assignable T-N for a parsed task set.
// Walks the highest existing N and returns N+1; starts at 1 for an
// empty set.
func NextID(existing []Task) string {
	max := 0
	for _, t := range existing {
		if n, ok := numericSuffix(t.ID); ok && n > max {
			max = n
		}
	}
	return fmt.Sprintf("T-%d", max+1)
}

// SortByID returns a copy of tasks ordered by their numeric T-N
// suffix. Malformed IDs sort last.
func SortByID(tasks []Task) []Task {
	out := make([]Task, len(tasks))
	copy(out, tasks)
	sort.SliceStable(out, func(i, j int) bool {
		return numericKey(out[i].ID) < numericKey(out[j].ID)
	})
	return out
}

// markerToStatus maps the checkbox character to the canonical status.
func markerToStatus(marker string) string {
	switch marker {
	case "x", "X":
		return StatusDone
	case "/":
		return StatusDoing
	default:
		return StatusTodo
	}
}

// applyMeta parses the contents of a `{k: v, k: v, ...}` shorthand
// block and sets matching fields on the task. Unknown keys are
// silently ignored (forward compatibility — adding a new key shouldn't
// break older parsers).
func applyMeta(t *Task, body string) {
	for _, raw := range splitMeta(body) {
		key, val, ok := splitKV(raw)
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "kind":
			t.Kind = val
		case "assignee":
			t.Assignee = val
		case "discovered_against", "discovered-against":
			t.DiscoveredAgainst = val
		case "started":
			t.Started = val
		case "done":
			t.Done = val
		}
	}
}

// splitMeta breaks a `k: v, k: v` body into individual `k: v` chunks.
// Honors brace nesting (none expected at v1 but cheap to support) and
// trims surrounding whitespace.
func splitMeta(body string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	for _, r := range body {
		switch r {
		case '{':
			depth++
			cur.WriteRune(r)
		case '}':
			depth--
			cur.WriteRune(r)
		case ',':
			if depth == 0 {
				out = append(out, cur.String())
				cur.Reset()
				continue
			}
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// splitKV splits a single "k: v" entry into trimmed key and value.
// Returns ok=false when the entry has no colon.
func splitKV(s string) (string, string, bool) {
	idx := strings.Index(s, ":")
	if idx < 0 {
		return "", "", false
	}
	k := strings.TrimSpace(s[:idx])
	v := strings.TrimSpace(s[idx+1:])
	v = strings.Trim(v, `"'`)
	return k, v, k != ""
}

// normalizeText trims surrounding whitespace and collapses internal
// double spaces (a frequent markdown artifact).
func normalizeText(s string) string {
	s = strings.TrimSpace(s)
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

// numericSuffix extracts N from "T-N". Returns ok=false for malformed.
func numericSuffix(id string) (int, bool) {
	id = strings.TrimPrefix(id, "T-")
	n, err := strconv.Atoi(id)
	if err != nil {
		return 0, false
	}
	return n, true
}

// numericKey returns N from "T-N" so sorting is numeric, not lexical
// (T-10 must come after T-9). Returns a sentinel for malformed IDs.
func numericKey(id string) int {
	if n, ok := numericSuffix(id); ok {
		return n
	}
	return 1 << 30
}
