package spec

// canonicalOwners is the locked-in owner enum from the spec-type-registry
// (per hero-pm-handoff-via-owner-history). An owner flip may only target
// one of these values; anything else is refused by IsValidOwner.
var canonicalOwners = map[string]bool{
	"pm":          true,
	"engineering": true,
	"qa":          true,
	"devops":      true,
	"design":      true,
	"docs":        true,
}

// CanonicalOwners returns the canonical owner enum values. Order is not
// guaranteed; callers that need a stable list should sort.
func CanonicalOwners() []string {
	out := make([]string, 0, len(canonicalOwners))
	for o := range canonicalOwners {
		out = append(out, o)
	}
	return out
}

// IsValidOwner reports whether s is a member of the canonical owner
// enum. The check is exact (case-sensitive) — owners are lowercase
// identifiers, not display strings.
func IsValidOwner(s string) bool {
	return canonicalOwners[s]
}
