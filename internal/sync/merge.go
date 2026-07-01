package sync

import "sort"

// SharedField names a field that both sides (local spec + tracker) may edit —
// the fields that need a 3-way merge. There are exactly three, matching the
// `shared` owner in hero-code's FieldOwnership.swift (title, body, tags).
//
// Each shared field is named twice: by its Swift/baseline key (Baseline.Base
// map key, and the Swift FieldOwnership name) and by its Go canonical field
// name (the key used in localFields / GetFields / UpdateFields on the Go side).
// The two vocabularies differ — body↔description, tags↔labels — so the mapping
// is explicit and lives here, next to the merge that depends on it.
type SharedField struct {
	// BaselineKey is the key under Baseline.Base and the Swift FieldOwnership
	// name (title / body / tags).
	BaselineKey string
	// Canonical is the Go-side canonical field name used by the tracker
	// adapters and the push/pull diff (title / description / labels).
	Canonical string
	// Kind is the field's shape, which selects the both-changed merge policy.
	Kind SharedKind
}

// SharedKind selects the both-changed auto-merge policy for a shared field.
type SharedKind int

const (
	// KindTitle is short scalar text: not hunk-mergeable, upstream wins with
	// the local title preserved in a marker.
	KindTitle SharedKind = iota
	// KindBody is long text: diff3-style line/hunk merge, upstream-first.
	KindBody
	// KindTags is a string set: set-union relative to base.
	KindTags
)

// SharedFields is the canonical shared-field set. Matches the `shared` owner in
// FieldOwnership.swift exactly: title, body, tags. priority/parent/owner/
// cycle_index/release are tracker-owned and status/size/depends_on/etc. are
// hero-owned — none of those appear here, so they keep today's behavior.
var SharedFields = []SharedField{
	{BaselineKey: "title", Canonical: "title", Kind: KindTitle},
	{BaselineKey: "body", Canonical: "description", Kind: KindBody},
	{BaselineKey: "tags", Canonical: "labels", Kind: KindTags},
}

// SharedByCanonical returns the SharedField for a Go canonical field name, or
// false if the field is not shared (tracker/hero/local-owned).
func SharedByCanonical(canonical string) (SharedField, bool) {
	for _, f := range SharedFields {
		if f.Canonical == canonical {
			return f, true
		}
	}
	return SharedField{}, false
}

// LocalEditMarkerPrefix opens the informational (non-blocking) marker that
// preserves a superseded local edit inline. It is intentionally an HTML comment
// so it renders invisibly in Markdown and never looks like a git conflict
// marker a human must resolve.
const LocalEditMarkerPrefix = "<!-- local edit"

// MergeTextResult is the outcome of a scalar/text 3-way merge (title, body).
type MergeTextResult struct {
	// Merged is the reconciled value to converge to. For title, this equals
	// Remote in the both-changed case (upstream wins). For body, this is the
	// diff3 merge.
	Merged string
	// PushLocal reports whether the merged value differs from Remote and so
	// must be written up to the tracker. When false, the tracker already holds
	// Merged and only the local spec is updated.
	PushLocal bool
	// LocalNote, when non-empty, is a preserved-local marker block the caller
	// appends to the body so a superseded local edit is recoverable. Only set
	// for the title policy (the local title is preserved in the body).
	LocalNote string
}

// MergeText resolves a scalar or long-text shared field via the 3-way truth
// table. `kind` selects the both-changed policy (KindTitle vs KindBody).
//
//	local == base   (only remote changed) → take remote  (never lose upstream)
//	remote == base  (only local changed)  → push local
//	local == remote                       → no-op (converged)
//	both changed                          → field-typed merge
func MergeText(kind SharedKind, base, local, remote string) MergeTextResult {
	switch {
	case local == remote:
		// Already converged — Merged is that value, nothing to push.
		return MergeTextResult{Merged: remote}
	case local == base:
		// Only remote changed → take remote into the spec; tracker already
		// holds it, so no push.
		return MergeTextResult{Merged: remote}
	case remote == base:
		// Only local changed → push local up.
		return MergeTextResult{Merged: local, PushLocal: true}
	default:
		// Both changed → field-typed auto-merge.
		if kind == KindBody {
			merged := MergeBody(base, local, remote)
			return MergeTextResult{Merged: merged, PushLocal: merged != remote}
		}
		// Title: upstream is truth. Keep remote; preserve the local title in a
		// marker block so it's recoverable. Remote already holds the value —
		// never push local over it (this is the drift-test guarantee).
		return MergeTextResult{
			Merged:    remote,
			PushLocal: false,
			LocalNote: titleLocalNote(local),
		}
	}
}

// titleLocalNote renders the preserved-local-title marker appended to the body.
func titleLocalNote(local string) string {
	return LocalEditMarkerPrefix + " (title, unmerged): " + local + " -->"
}

// MergeTagsResult is the outcome of a tag-set 3-way merge.
type MergeTagsResult struct {
	// Merged is the reconciled tag set (order-stable).
	Merged []string
	// PushLocal reports whether Merged differs from Remote (as a set) and so
	// must be written up to the tracker.
	PushLocal bool
}

// MergeTags resolves the tags shared field. Non-both-changed cases follow the
// same truth table as text; the both-changed case is a set-union relative to
// base: every add and every remove made by either side is applied, and no tag
// present on either side against a non-removing base is ever dropped. Order is
// stable (remote order first, then local additions in local order).
func MergeTags(base, local, remote []string) MergeTagsResult {
	switch {
	case tagsEqual(local, remote):
		return MergeTagsResult{Merged: dedupePreserveOrder(remote)}
	case tagsEqual(local, base):
		// Only remote changed → take remote.
		return MergeTagsResult{Merged: dedupePreserveOrder(remote)}
	case tagsEqual(remote, base):
		// Only local changed → push local.
		return MergeTagsResult{Merged: dedupePreserveOrder(local), PushLocal: true}
	default:
		merged := unionTags(base, local, remote)
		return MergeTagsResult{Merged: merged, PushLocal: !tagsEqual(merged, remote)}
	}
}

// unionTags computes the 3-way set-union for tags: start from base, apply the
// removals each side made (a tag in base but absent from a side is removed only
// if BOTH sides removed it — a removal on one side plus an add/keep on the
// other keeps the tag, honoring "never drop a tag"), then apply every addition
// from either side. Order is stable: base order (surviving), then remote-only
// additions, then local-only additions.
func unionTags(base, local, remote []string) []string {
	baseSet := toSet(base)
	localSet := toSet(local)
	remoteSet := toSet(remote)

	var out []string
	seen := map[string]bool{}
	add := func(tag string) {
		if tag == "" || seen[tag] {
			return
		}
		seen[tag] = true
		out = append(out, tag)
	}

	// Surviving base tags, in base order. A base tag drops only when BOTH sides
	// removed it; otherwise it survives (never drop a tag on a lone removal).
	for _, tag := range dedupePreserveOrder(base) {
		removedByLocal := !localSet[tag]
		removedByRemote := !remoteSet[tag]
		if removedByLocal && removedByRemote {
			continue
		}
		add(tag)
	}
	// Remote-only additions (not in base), in remote order.
	for _, tag := range dedupePreserveOrder(remote) {
		if !baseSet[tag] {
			add(tag)
		}
	}
	// Local-only additions (not in base), in local order.
	for _, tag := range dedupePreserveOrder(local) {
		if !baseSet[tag] {
			add(tag)
		}
	}
	return out
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		if s != "" {
			m[s] = true
		}
	}
	return m
}

func dedupePreserveOrder(ss []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// tagsEqual reports set-equality (order-insensitive) between two tag slices,
// matching tracker.Value's order-insensitive label comparison.
func tagsEqual(a, b []string) bool {
	as := dedupePreserveOrder(a)
	bs := dedupePreserveOrder(b)
	if len(as) != len(bs) {
		return false
	}
	x := append([]string(nil), as...)
	y := append([]string(nil), bs...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
