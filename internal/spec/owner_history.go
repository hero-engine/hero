package spec

import (
	"fmt"
	"strings"
	"time"
)

// OwnerHistoryEntry is one segment of a spec's ownership timeline. From
// is when the owner took over; To is when they handed off (nil marks the
// currently-active entry). Extra holds any per-entry frontmatter keys the
// parser did not recognize, preserved verbatim so a round-trip through
// set-owner is forward-compatible (unknown fields survive).
type OwnerHistoryEntry struct {
	Owner string
	From  *time.Time
	To    *time.Time
	// Extra captures unknown per-entry fields (key → raw scalar value)
	// in first-seen order so re-serialization is stable. Known keys
	// (owner, from, to) never land here.
	Extra []ExtraField
}

// ExtraField is a forward-compat passthrough of an unrecognized
// per-entry key/value pair in an owner_history block.
type ExtraField struct {
	Key   string
	Value string
}

// SynthesizeHistory builds a single-entry owner_history from a spec's
// current top-level owner. The file mtime stands in for the (unknown)
// moment the owner took over — the same rule the workspace loader uses
// for read-time synthesis. The synthesized entry is active (To == nil).
//
// An empty currentOwner yields a nil history: there is nothing to
// synthesize from, and callers append the first real entry instead.
func SynthesizeHistory(currentOwner string, fileMTime time.Time) []OwnerHistoryEntry {
	if currentOwner == "" {
		return nil
	}
	from := fileMTime.UTC()
	return []OwnerHistoryEntry{
		{Owner: currentOwner, From: &from, To: nil},
	}
}

// AppendOwnerFlip closes the currently-active entry (the last one with a
// nil To) by setting its To to at, then appends a fresh active entry for
// newOwner with From == at and To == nil. The input slice is not
// mutated; a new slice is returned. When history is empty the result is
// a single active entry — the caller is responsible for synthesizing a
// prior entry first if one is wanted.
func AppendOwnerFlip(history []OwnerHistoryEntry, newOwner string, at time.Time) []OwnerHistoryEntry {
	at = at.UTC()
	out := make([]OwnerHistoryEntry, len(history))
	copy(out, history)

	// Close the last active entry (nil To). Scan from the end so the
	// most recent active entry is the one we close.
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].To == nil {
			closed := at
			out[i].To = &closed
			break
		}
	}

	out = append(out, OwnerHistoryEntry{Owner: newOwner, From: &at, To: nil})
	return out
}

// ActiveOwner returns the owner of the active entry (nil To), or "" when
// there is none. When multiple entries are open it returns the last.
func ActiveOwner(history []OwnerHistoryEntry) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].To == nil {
			return history[i].Owner
		}
	}
	return ""
}

// ---- round-trip serialization ----------------------------------------

const ownerHistoryTimeFormat = time.RFC3339

// ParseOwnerHistoryBlock parses the indented YAML list under an
// `owner_history:` key. lines are the SplitAfter("\n") slice of the full
// content; start is the first line after the key; end is the frontmatter
// close index. Returns the parsed entries and the index of the next line
// the outer parser should resume from.
//
// Recognized per-entry keys: owner, from, to. `to: null` (or an empty
// value) yields a nil To. Any other key is preserved in Extra for
// forward compat.
func ParseOwnerHistoryBlock(lines []string, start, end int) ([]OwnerHistoryEntry, int) {
	var out []OwnerHistoryEntry
	var current *OwnerHistoryEntry
	idx := start
	for ; idx < end; idx++ {
		raw := lines[idx]
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		// A top-level key (no indent) ends the block.
		if leadingSpaceCount(raw) == 0 {
			break
		}
		if strings.HasPrefix(trimmed, "- ") {
			// Flush prior entry, start a new one.
			if current != nil {
				out = append(out, *current)
			}
			current = &OwnerHistoryEntry{}
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			applyOwnerHistoryField(current, rest)
			continue
		}
		if current != nil {
			applyOwnerHistoryField(current, trimmed)
		}
	}
	if current != nil {
		out = append(out, *current)
	}
	return out, idx
}

// applyOwnerHistoryField parses "key: value" inside an owner_history
// entry and assigns it. owner/from/to are recognized; everything else is
// stashed in Extra (forward compat).
func applyOwnerHistoryField(e *OwnerHistoryEntry, line string) {
	k, v, ok := strings.Cut(line, ":")
	if !ok {
		return
	}
	k = strings.TrimSpace(k)
	v = strings.TrimSpace(strings.Trim(strings.TrimSpace(v), `"'`))
	switch k {
	case "owner":
		e.Owner = v
	case "from":
		e.From = parseOwnerHistoryTime(v)
	case "to":
		e.To = parseOwnerHistoryTime(v)
	default:
		e.Extra = append(e.Extra, ExtraField{Key: k, Value: v})
	}
}

// parseOwnerHistoryTime parses an RFC3339 timestamp. "null", "~", and
// the empty string all mean "no timestamp" (active entry / open bound).
func parseOwnerHistoryTime(v string) *time.Time {
	v = strings.TrimSpace(v)
	if v == "" || v == "null" || v == "~" {
		return nil
	}
	if t, err := time.Parse(ownerHistoryTimeFormat, v); err == nil {
		return &t
	}
	// Date-only fallback keeps malformed-but-present timestamps visible
	// rather than silently dropping them to nil (which would mis-mark an
	// entry as active).
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return &t
	}
	return nil
}

// RenderOwnerHistoryBlock renders an owner_history value as YAML
// frontmatter lines (without the trailing newline on the last line). The
// output is a `owner_history:` key followed by an indented list. Unknown
// per-entry fields (Extra) are re-emitted after the known keys, in their
// original order. An empty history renders nothing (returns "").
func RenderOwnerHistoryBlock(history []OwnerHistoryEntry) string {
	if len(history) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("owner_history:\n")
	for _, e := range history {
		fmt.Fprintf(&b, "  - owner: %s\n", e.Owner)
		fmt.Fprintf(&b, "    from: %s\n", renderOwnerHistoryTime(e.From))
		fmt.Fprintf(&b, "    to: %s\n", renderOwnerHistoryTime(e.To))
		for _, ex := range e.Extra {
			fmt.Fprintf(&b, "    %s: %s\n", ex.Key, ex.Value)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderOwnerHistoryTime renders a timestamp bound: nil → "null", set →
// RFC3339 (UTC, no fractional seconds).
func renderOwnerHistoryTime(t *time.Time) string {
	if t == nil {
		return "null"
	}
	return t.UTC().Format(ownerHistoryTimeFormat)
}
