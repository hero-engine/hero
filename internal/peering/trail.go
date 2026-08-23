package peering

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/peering"
)

// TrailSectionHeader is the canonical markdown header for the
// `## Handoff Trail` section in a spec file.
const TrailSectionHeader = "## Handoff Trail"

// trailHeaderLine is the literal line we look for / write.
const trailHeaderLine = TrailSectionHeader

// ReadTrail extracts and parses every TrailEntry from the
// `## Handoff Trail` section of a spec file. Returns (nil, nil) when
// the file has no such section.
//
// Recognized entry format (must round-trip cleanly with WriteTrail):
//
//   - 2026-05-15T14:00:00Z — out → app (peer_id: 9c1c2f3e-...)
//     mode: async-drop
//     originating_spec: order-failure-error-display
//     peer_spec: app/error-envelope-mismatch
//     peer_status: planning
//     at_commit: 3176736
//     result_ref: commit 4427cec
//     reason: "Symptom is in the client, root cause is the API response shape."
//
// The bullet header line is parsed for timestamp, direction, peer
// alias display, and peer_id. The indented YAML-style key:value lines
// that follow flesh out the entry until either a blank line OR
// another top-level `## ` header OR another `- ` bullet at column 0.
func ReadTrail(specPath string) ([]peering.TrailEntry, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseTrail(string(data)), nil
}

// ParseTrail extracts trail entries from a raw spec body. Exported
// for tests and for callers that already have the body in memory.
func ParseTrail(content string) []peering.TrailEntry {
	lines := strings.Split(content, "\n")
	startIdx := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == trailHeaderLine {
			startIdx = i + 1
			break
		}
	}
	if startIdx < 0 {
		return nil
	}

	// Find end: next "## " header at column 0 (any depth same level
	// or higher), or EOF.
	endIdx := len(lines)
	for i := startIdx; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") && i != startIdx-1 {
			endIdx = i
			break
		}
	}

	var entries []peering.TrailEntry
	var current *peering.TrailEntry
	flush := func() {
		if current != nil {
			entries = append(entries, *current)
			current = nil
		}
	}

	for i := startIdx; i < endIdx; i++ {
		raw := lines[i]
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		// Bullet header: "- <timestamp> — <direction> [→|←] <alias> (peer_id: <uuid>)"
		// Tolerant of "->" / "<-" and " - " separators.
		if strings.HasPrefix(raw, "- ") {
			flush()
			head := strings.TrimPrefix(raw, "- ")
			current = parseTrailBulletHeader(head)
			continue
		}
		// Continuation key:value line.
		if current == nil {
			continue
		}
		k, v, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = unquoteScalar(strings.TrimSpace(v))
		switch k {
		case "mode":
			current.Mode = peering.TrailMode(v)
		case "originating_spec":
			current.OriginatingSpec = v
		case "peer_spec":
			current.PeerSpec = v
		case "peer_status":
			current.PeerStatus = v
		case "at_commit":
			current.AtCommit = v
		case "result_ref":
			current.ResultRef = v
		case "reason":
			current.Reason = v
		case "transport":
			current.Transport = v
		case "message_id":
			current.MessageID = v
		case "thread_id":
			current.ThreadID = v
		case "peer_id":
			// Fallback if the header line didn't carry the id.
			if current.PeerID == "" {
				current.PeerID = v
			}
		}
	}
	flush()
	return entries
}

// unquoteScalar strips a single matched pair of surrounding quotes
// from a YAML scalar value. Lone or unmatched quotes are preserved.
func unquoteScalar(v string) string {
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

// parseTrailBulletHeader parses a single bullet header into the
// timestamp/direction/peer fields. The remainder of the entry is
// filled in by subsequent key:value lines.
//
// Accepted shapes (best-effort, very tolerant):
//
//	2026-05-15T14:00:00Z — out → app (peer_id: 9c1c2f3e-...)
//	2026-05-15T16:23:00Z - in <- app (peer_id: 9c1c2f3e-...)
func parseTrailBulletHeader(head string) *peering.TrailEntry {
	out := &peering.TrailEntry{}
	// Split off an optional "(peer_id: ...)" suffix.
	if idx := strings.LastIndex(head, "(peer_id:"); idx >= 0 {
		tail := head[idx:]
		head = strings.TrimSpace(head[:idx])
		// tail looks like "(peer_id: 9c1c... )"
		tail = strings.TrimPrefix(tail, "(peer_id:")
		tail = strings.TrimSuffix(strings.TrimSpace(tail), ")")
		out.PeerID = strings.TrimSpace(tail)
	}
	// Timestamp: the first whitespace-bounded token.
	parts := strings.Fields(head)
	if len(parts) == 0 {
		return out
	}
	if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
		out.At = t
	}
	// Direction: "out" or "in" appears somewhere in the line.
	rest := strings.ToLower(strings.Join(parts[1:], " "))
	switch {
	case strings.Contains(rest, "out "), strings.HasPrefix(rest, "out"), strings.Contains(rest, " out"):
		out.Direction = peering.DirectionOut
	case strings.Contains(rest, "in "), strings.HasPrefix(rest, "in"), strings.Contains(rest, " in"):
		out.Direction = peering.DirectionIn
	}
	// Peer alias display: last token after "→" / "->" / "←" / "<-".
	for _, sep := range []string{"→", "->", "←", "<-"} {
		if idx := strings.Index(head, sep); idx >= 0 {
			alias := strings.TrimSpace(head[idx+len(sep):])
			out.PeerAliasDisplay = alias
			break
		}
	}
	return out
}

// AppendTrail writes a TrailEntry to the spec file's
// `## Handoff Trail` section, creating the section if absent.
// Entries are kept in chronological order; entries with equal
// timestamps preserve insertion order.
func AppendTrail(specPath string, entry peering.TrailEntry) error {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	updated := AppendTrailToContent(string(data), entry)
	return os.WriteFile(specPath, []byte(updated), 0o644)
}

// AppendTrailToContent inserts the entry into the in-memory content
// and returns the updated content. Pure function — exposed for tests.
func AppendTrailToContent(content string, entry peering.TrailEntry) string {
	entries := ParseTrail(content)
	entries = append(entries, entry)
	// Stable sort by At (entries with zero At go last).
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].At.IsZero() {
			return false
		}
		if entries[j].At.IsZero() {
			return true
		}
		return entries[i].At.Before(entries[j].At)
	})

	rendered := RenderTrailSection(entries)
	return replaceOrAppendSection(content, trailHeaderLine, rendered)
}

// RenderTrailSection produces the full `## Handoff Trail` markdown
// section (header included) for a list of entries. Always ends with
// a trailing newline.
func RenderTrailSection(entries []peering.TrailEntry) string {
	var b strings.Builder
	b.WriteString(trailHeaderLine)
	b.WriteString("\n\n")
	for i, e := range entries {
		if i > 0 {
			b.WriteString("\n")
		}
		renderTrailEntry(&b, e)
	}
	b.WriteString("\n")
	return b.String()
}

func renderTrailEntry(b *strings.Builder, e peering.TrailEntry) {
	ts := e.At.UTC().Format(time.RFC3339)
	arrow := "→"
	dir := "out"
	if e.Direction == peering.DirectionIn {
		arrow = "←"
		dir = "in"
	}
	alias := e.PeerAliasDisplay
	if alias == "" {
		alias = "(unknown)"
	}
	fmt.Fprintf(b, "- %s — %s %s %s (peer_id: %s)\n", ts, dir, arrow, alias, e.PeerID)
	if e.Mode != "" {
		fmt.Fprintf(b, "  mode: %s\n", e.Mode)
	}
	if e.OriginatingSpec != "" {
		fmt.Fprintf(b, "  originating_spec: %s\n", e.OriginatingSpec)
	}
	if e.PeerSpec != "" {
		fmt.Fprintf(b, "  peer_spec: %s\n", e.PeerSpec)
	}
	if e.PeerStatus != "" {
		fmt.Fprintf(b, "  peer_status: %s\n", e.PeerStatus)
	}
	if e.AtCommit != "" {
		fmt.Fprintf(b, "  at_commit: %s\n", e.AtCommit)
	}
	if e.ResultRef != "" {
		fmt.Fprintf(b, "  result_ref: %s\n", e.ResultRef)
	}
	if e.Reason != "" {
		fmt.Fprintf(b, "  reason: %q\n", e.Reason)
	}
	if e.Transport != "" {
		fmt.Fprintf(b, "  transport: %s\n", e.Transport)
	}
	if e.MessageID != "" {
		fmt.Fprintf(b, "  message_id: %s\n", e.MessageID)
	}
	if e.ThreadID != "" {
		fmt.Fprintf(b, "  thread_id: %s\n", e.ThreadID)
	}
}

// replaceOrAppendSection rewrites (or appends) a top-level `## ` block
// in markdown. If header is absent, the rendered block is appended
// with a separating blank line. If present, the existing block (from
// the header to the next `## ` or EOF) is replaced verbatim.
func replaceOrAppendSection(content, header, rendered string) string {
	lines := strings.Split(content, "\n")
	startIdx := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == header {
			startIdx = i
			break
		}
	}
	if startIdx < 0 {
		// Append. Ensure exactly one blank line separator.
		trimmed := strings.TrimRight(content, "\n")
		return trimmed + "\n\n" + rendered
	}
	endIdx := len(lines)
	for j := startIdx + 1; j < len(lines); j++ {
		if strings.HasPrefix(lines[j], "## ") {
			endIdx = j
			break
		}
	}
	before := strings.Join(lines[:startIdx], "\n")
	after := strings.Join(lines[endIdx:], "\n")
	out := before
	if before != "" && !strings.HasSuffix(before, "\n") {
		out += "\n"
	}
	out += rendered
	if !strings.HasSuffix(rendered, "\n") {
		out += "\n"
	}
	if after != "" {
		out += after
	}
	return out
}
