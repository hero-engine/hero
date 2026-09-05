package spec

// InitiativeReadyToComplete reports whether an initiative should be
// auto-completed because every one of its children is already a completed
// spec. It is the single completion decision shared by two call sites:
//
//   - the in-process side-effect in `hero spec verify` (autoCompleteParentIfReady),
//     which fires when the last not-yet-completed child is verified, and
//   - the standalone reconcile re-check in `hero check --reconcile`, which
//     recovers an initiative whose children are all already completed and
//     archived (no child verified in the current process).
//
// Keeping the gate in one place guarantees the two paths can never diverge.
// Two gates run: a roster gate (every child the initiative declares — see
// DeclaredChildren — must resolve to a materialized, finished spec) and a
// child-count gate (at least one materialized spec must declare this
// initiative as its parent, and all such specs must be completed).
//
// It returns false for a nil parent, a non-initiative, or an already-completed
// initiative — so callers can pass any candidate without pre-filtering.
func InitiativeReadyToComplete(parent *Spec, allSpecs []*Spec) bool {
	if parent == nil || parent.Type != TypeInitiative {
		return false
	}
	if parent.Status == StatusCompleted {
		return false
	}

	specBySlug := make(map[string]*Spec, len(allSpecs))
	for _, s := range allSpecs {
		specBySlug[s.Slug] = s
	}

	// Roster gate: every child the initiative declares must resolve to a
	// materialized spec that is finished — otherwise delivering a single child
	// would wrongly complete an initiative whose other children are unbuilt
	// stubs. The roster comes from DeclaredChildren, the same union of
	// frontmatter relations and `## Child Specs & Sequence` table links that
	// drive's child-set builder consumes, so this gate and `hero goal --check`
	// cannot disagree about which children remain.
	//
	// A declared child with no spec on disk blocks: not-yet-scaffolded is the
	// starved case this gate exists for, and a governance initiative whose
	// unbuilt child is a safety invariant must never read as delivered. To
	// intentionally drop a child the operator removes it from the declaration
	// or marks it `superseded`, which counts as finished.
	for _, slug := range DeclaredChildren(parent) {
		child, ok := specBySlug[slug]
		if !ok || !childFinished(child) {
			return false
		}
	}

	// Child-count gate: at least one materialized spec must declare this
	// initiative as its parent, and all such children must be finished.
	allDone := true
	childCount := 0
	for _, s := range allSpecs {
		for _, r := range s.Relations {
			if (r.Kind == "parent" || r.Kind == "child-of") &&
				normalizeRelTarget(r.Target) == parent.Slug {
				childCount++
				if !childFinished(s) {
					allDone = false
				}
				break
			}
		}
	}
	if childCount == 0 || !allDone {
		return false
	}

	return true
}

// childFinished reports whether a child's status means the initiative has
// nothing left to wait on. Superseded counts alongside the type's own
// terminal status: it is the documented way to drop a child the initiative
// no longer intends to build, and both gates have to agree on that or the
// escape hatch doesn't work.
func childFinished(child *Spec) bool {
	return child.Status == StatusSuperseded || child.Status == terminalStatusForType(child.Type)
}

// terminalStatusForType returns the status that marks a spec of the given
// type as finished. Feature, bug, and initiative children follow the
// planning → delivering → completed work lifecycle and terminate at
// StatusCompleted. Decision specs don't follow that lifecycle at all — this
// workspace's own convention (and `hero spec verify`'s refusal to run
// delivery gates against a decision) terminates them at StatusAccepted
// instead. The remaining knowledge types have no completion lifecycle either
// and are finished once filed, at StatusActive — mirroring the same mapping
// statusFromPath already uses when inferring a spec's status from its
// archival location.
func terminalStatusForType(t Type) Status {
	switch t {
	case TypeDecision:
		return StatusAccepted
	case TypeConvention, TypeRule, TypeExternal, TypeContext, TypeNote, TypeTripwire, TypeExplainer:
		return StatusActive
	default:
		return StatusCompleted
	}
}
