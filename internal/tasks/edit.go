package tasks

import (
	"fmt"
	"os"
	"strings"
)

// EditOptions controls how Render formats inline metadata. Defaults
// are fine for every caller today; the struct exists so we can add
// per-call knobs (e.g. canonical key ordering, wrapping) without
// breaking the call site.
type EditOptions struct{}

// Render formats a slice of tasks as the canonical `## Tasks`
// checklist body (without the heading itself). Stable ordering: tasks
// are emitted in numeric T-N order so re-rendering yields a
// deterministic diff.
func Render(tasks []Task, _ EditOptions) string {
	sorted := SortByID(tasks)
	var b strings.Builder
	for _, t := range sorted {
		b.WriteString(renderLine(t))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// renderLine formats one task. Status maps to checkbox char; inline
// metadata is emitted in canonical key order so diffs stay tight.
func renderLine(t Task) string {
	box := " "
	switch t.Status {
	case StatusDoing:
		box = "/"
	case StatusDone:
		box = "x"
	}
	line := fmt.Sprintf("- [%s] %s %s", box, t.ID, t.Text)
	meta := renderMeta(t)
	if meta != "" {
		line += " " + meta
	}
	return line
}

// renderMeta emits the trailing `{...}` block in a stable key order.
// Returns "" when no metadata is present.
func renderMeta(t Task) string {
	type kv struct{ k, v string }
	var pairs []kv
	add := func(k, v string) {
		if v != "" {
			pairs = append(pairs, kv{k, v})
		}
	}
	add("kind", t.Kind)
	add("assignee", t.Assignee)
	add("discovered_against", t.DiscoveredAgainst)
	add("started", t.Started)
	add("done", t.Done)
	if len(pairs) == 0 {
		return ""
	}
	// Canonical order is the order add() runs above — that's already
	// what readers expect — so just emit. Sorting here would shuffle.
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = p.k + ": " + p.v
	}
	// Canonical key order matches the design spec's sample
	// (`kind, assignee, discovered_against, started, done`) — keep
	// it stable so diffs of unchanged tasks are empty.
	return "{" + strings.Join(parts, ", ") + "}"
}

// ApplyToFile reads path, replaces (or inserts) the `## Tasks`
// section's body with `body`, and writes the file back.
//
// Insertion rules:
//   - If `## Tasks` exists, its body (everything until the next
//     heading at the same or higher level, or EOF) is replaced.
//   - If `## Tasks` does not exist, a new section is appended at the
//     end of the file (with a leading blank line separator).
//
// `body` should be the rendered checklist with one trailing newline;
// ApplyToFile inserts the `## Tasks\n\n` heading prelude itself.
func ApplyToFile(path, body string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	updated := UpsertSection(string(data), "Tasks", body)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// UpsertSection returns content with the section "## <heading>"
// replaced (or appended). Exposed for testability and so the CLI can
// preview a write without touching disk.
func UpsertSection(content, heading, body string) string {
	body = strings.TrimRight(body, "\n") + "\n"
	hdr := "## " + heading

	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == hdr {
			start = i
			break
		}
	}
	if start < 0 {
		// Append. Ensure exactly one blank line between prior content
		// and the new section.
		trimmed := strings.TrimRight(content, "\n")
		var b strings.Builder
		b.WriteString(trimmed)
		if trimmed != "" {
			b.WriteString("\n\n")
		}
		b.WriteString(hdr)
		b.WriteString("\n\n")
		b.WriteString(body)
		return b.String()
	}
	// Find the end of this section: the next line that begins with
	// "## " (or shallower — i.e. a single "# " heading) marks the
	// next section. EOF if none.
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trim, "## ") || strings.HasPrefix(trim, "# ") {
			end = i
			break
		}
	}
	var b strings.Builder
	for _, l := range lines[:start] {
		b.WriteString(l)
		b.WriteString("\n")
	}
	b.WriteString(hdr)
	b.WriteString("\n\n")
	b.WriteString(body)
	// Preserve a blank line before the next section if there was
	// content following.
	if end < len(lines) {
		b.WriteString("\n")
		for i, l := range lines[end:] {
			b.WriteString(l)
			if i < len(lines[end:])-1 {
				b.WriteString("\n")
			}
		}
	}
	out := b.String()
	// Normalize: never emit triple-blank-line runs.
	for strings.Contains(out, "\n\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n\n", "\n\n\n")
	}
	return out
}

// AddTask appends a new task to the parsed list and returns the
// resulting slice. The new task's ID is assigned via NextID; its
// status defaults to "todo".
func AddTask(existing []Task, text string, opts AddOptions) (Task, []Task) {
	t := Task{
		ID:                NextID(existing),
		Text:              text,
		Status:            StatusTodo,
		Kind:              opts.Kind,
		Assignee:          opts.Assignee,
		DiscoveredAgainst: opts.DiscoveredAgainst,
	}
	return t, append(append([]Task{}, existing...), t)
}

// AddOptions carries the metadata bits a CLI add-command accepts.
type AddOptions struct {
	Kind              string
	Assignee          string
	DiscoveredAgainst string
}

// TransitionTo flips one task by ID to the target status, stamping
// `started` or `done` timestamps as appropriate. Returns the updated
// slice and an error when the ID is unknown.
//
// Status precedence:
//   - todo  → doing: stamps `started: <now>` (if unset)
//   - doing → done : stamps `done: <now>`    (if unset)
//   - todo  → done : stamps `done: <now>` and skips started
func TransitionTo(existing []Task, id, target string) ([]Task, error) {
	out := make([]Task, len(existing))
	copy(out, existing)
	for i := range out {
		if out[i].ID != id {
			continue
		}
		t := out[i]
		switch target {
		case StatusDoing:
			t.Status = StatusDoing
			if t.Started == "" {
				t.Started = nowRFC()
			}
		case StatusDone:
			t.Status = StatusDone
			if t.Done == "" {
				t.Done = nowRFC()
			}
		case StatusTodo:
			t.Status = StatusTodo
		default:
			return out, fmt.Errorf("tasks: unknown status %q (want todo|doing|done)", target)
		}
		out[i] = t
		return out, nil
	}
	return out, fmt.Errorf("tasks: no task with ID %q", id)
}
