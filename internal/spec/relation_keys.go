package spec

import "strings"

// relationShorthandKeys are the top-level frontmatter keys that parse into
// relation edges. Everything else in frontmatter is either a known field, a
// tracker-prefixed field, or ignored.
var relationShorthandKeys = []string{
	"parent", "child", "children", "child-of", "child_of", "initiative",
	"depends-on", "depends_on", "relates-to",
	"supersedes", "conflicts-with", "conflicts_with",
}

// relationKeyAliases maps the variants authors actually reach for onto the
// relation they meant. These are near misses the parser does not accept, so
// without a warning they vanish: a dropped child roster is exactly how an
// initiative auto-completed with three of its four children unbuilt.
var relationKeyAliases = map[string]string{
	"child-specs":   "child",
	"childspecs":    "child",
	"sub-specs":     "child",
	"subspecs":      "child",
	"sub-spec":      "child",
	"subspec":       "child",
	"sub-tasks":     "child",
	"subtasks":      "child",
	"parents":       "parent",
	"parent-spec":   "parent",
	"epic":          "parent",
	"depends":       "depends-on",
	"depend-on":     "depends-on",
	"depends-upon":  "depends-on",
	"dependencies":  "depends-on",
	"dependency":    "depends-on",
	"deps":          "depends-on",
	"blocked-by":    "depends-on",
	"blockedby":     "depends-on",
	"blocks":        "blocks",
	"blocking":      "blocks",
	"related":       "relates-to",
	"related-to":    "relates-to",
	"relates":       "relates-to",
	"relation":      "relates-to",
	"relationship":  "relates-to",
	"supersede":     "supersedes",
	"superseded-by": "supersedes",
	"replaces":      "supersedes",
	"conflicts":     "conflicts-with",
	"conflict-with": "conflicts-with",
}

// NearMissRelationKey reports the relation an unrecognized frontmatter key
// most likely meant, or "" when the key is not a plausible relation at all.
// It is deliberately conservative — a false warning on an author's own
// bookkeeping field is worse than a missed typo, because the whole point is
// that this warning gets read.
//
// Matching runs in three passes: a canonical key that the parser happens not
// to accept as a shorthand (`blocks:`), a curated alias table, then an
// edit-distance fallback for one-off misspellings.
func NearMissRelationKey(key string) string {
	norm := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(key)), "_", "-")
	if norm == "" {
		return ""
	}
	for _, k := range relationShorthandKeys {
		if strings.ReplaceAll(k, "_", "-") == norm {
			return "" // accepted by the parser — nothing to warn about
		}
	}
	if kind, ok := relationKeyAliases[norm]; ok {
		return kind
	}
	// Fuzzy fallback for a genuine typo (`chidl`, `dpends-on`). Short keys are
	// excluded: at four characters or fewer, distance 2 matches almost
	// anything and the warning turns to noise.
	if len(norm) < 5 {
		return ""
	}
	best, bestDist := "", 3
	for _, k := range relationShorthandKeys {
		cand := strings.ReplaceAll(k, "_", "-")
		if d := levenshtein(norm, cand); d < bestDist {
			best, bestDist = cand, d
		}
	}
	return best
}

// RelationKeyNearMiss is one unrecognized frontmatter key on a spec plus the
// relation it most likely meant.
type RelationKeyNearMiss struct {
	Key   string // the key as written in frontmatter
	Meant string // the relation kind the author likely intended
}

// NearMissRelationKeys returns the near-miss relation keys in a spec's
// frontmatter, in file order.
func NearMissRelationKeys(s *Spec) []RelationKeyNearMiss {
	if s == nil {
		return nil
	}
	var out []RelationKeyNearMiss
	for _, key := range s.UnknownKeys {
		if meant := NearMissRelationKey(key); meant != "" {
			out = append(out, RelationKeyNearMiss{Key: key, Meant: meant})
		}
	}
	return out
}
