package tracker

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ValueKind tags the shape of a tracker field Value. The set is
// intentionally small (string / int / array-of-strings / null) — the
// minimum needed to cover the field shapes the vocabulary declares
// today (title, description, points, priority, labels). Adapters
// convert these to provider-native representations. See the spec's
// "Field value type contract" and Risks → "Provider field-type
// mismatches": more kinds get added as concrete failures emerge.
type ValueKind int

const (
	// ValueNull is an explicit absence (clears the tracker field).
	ValueNull ValueKind = iota
	// ValueString is a scalar string (title, description, priority).
	ValueString
	// ValueInt is an integer scalar (points / estimate).
	ValueInt
	// ValueStrings is an array of strings (labels).
	ValueStrings
)

// Value is a tagged union for a single tracker field value. Only the
// member matching Kind is meaningful. Construct via the StringValue /
// IntValue / StringsValue / NullValue helpers so the tag and payload
// stay consistent.
type Value struct {
	Kind    ValueKind
	Str     string
	Int     int
	Strings []string
}

// StringValue builds a string-kinded Value.
func StringValue(s string) Value { return Value{Kind: ValueString, Str: s} }

// IntValue builds an int-kinded Value.
func IntValue(n int) Value { return Value{Kind: ValueInt, Int: n} }

// StringsValue builds an array-of-strings Value.
func StringsValue(ss []string) Value { return Value{Kind: ValueStrings, Strings: ss} }

// NullValue builds a null Value (clears the field on the tracker).
func NullValue() Value { return Value{Kind: ValueNull} }

// ParseScalar converts a raw frontmatter scalar string into a Value.
// The hint biases the parse toward a kind:
//
//	"int"     — parse as integer; falls back to string on parse failure
//	"strings" — split a comma/space list into an array of strings
//	otherwise — string
//
// An empty raw string with a non-strings hint yields NullValue so a
// cleared local field clears the tracker field. ParseScalar never
// errors — an unparseable int degrades to a string so callers don't
// have to special-case malformed frontmatter.
func ParseScalar(raw, hint string) Value {
	raw = strings.TrimSpace(raw)
	switch hint {
	case "int":
		if raw == "" {
			return NullValue()
		}
		if n, err := strconv.Atoi(raw); err == nil {
			return IntValue(n)
		}
		return StringValue(raw)
	case "strings":
		return StringsValue(splitList(raw))
	default:
		if raw == "" {
			return NullValue()
		}
		return StringValue(raw)
	}
}

// splitList parses a YAML-ish inline list ("[a, b]") or a comma list
// ("a, b") into a slice of trimmed, non-empty strings.
func splitList(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.Trim(p, `"'`))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Equal reports whether two Values are equal. Array equality is
// order-insensitive (labels are a set, not a sequence) so a reorder on
// either side does not register as a diff — preserving idempotency.
func (v Value) Equal(other Value) bool {
	if v.Kind != other.Kind {
		return false
	}
	switch v.Kind {
	case ValueNull:
		return true
	case ValueString:
		return v.Str == other.Str
	case ValueInt:
		return v.Int == other.Int
	case ValueStrings:
		if len(v.Strings) != len(other.Strings) {
			return false
		}
		a := append([]string(nil), v.Strings...)
		b := append([]string(nil), other.Strings...)
		sort.Strings(a)
		sort.Strings(b)
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	return false
}

// String renders a Value for human-readable diff output.
func (v Value) String() string {
	switch v.Kind {
	case ValueNull:
		return "<null>"
	case ValueString:
		return v.Str
	case ValueInt:
		return strconv.Itoa(v.Int)
	case ValueStrings:
		return "[" + strings.Join(v.Strings, ", ") + "]"
	}
	return fmt.Sprintf("<unknown:%d>", v.Kind)
}

// JSON returns the Value in a form suitable for json.Marshal:
// string → string, int → int, strings → []string, null → nil.
func (v Value) JSON() interface{} {
	switch v.Kind {
	case ValueString:
		return v.Str
	case ValueInt:
		return v.Int
	case ValueStrings:
		if v.Strings == nil {
			return []string{}
		}
		return v.Strings
	default:
		return nil
	}
}
