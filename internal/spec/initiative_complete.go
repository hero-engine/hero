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
// The logic is unchanged from the original inline gate in
// autoCompleteParentIfReady: a roster gate (every declared block-style
// `child:` entry must resolve to a completed spec) and a child-count gate
// (at least one materialized spec must declare this initiative as its parent,
// and all such specs must be completed).
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

	statusBySlug := make(map[string]Status, len(allSpecs))
	for _, s := range allSpecs {
		statusBySlug[s.Slug] = s.Status
	}

	// Roster gate: if the initiative declares children (block-style `child:`
	// lists parse to child relations), every declared child must resolve to a
	// materialized, completed spec — otherwise delivering a single child would
	// wrongly complete an initiative whose other children are unbuilt stubs.
	declaredCount := 0
	declaredComplete := true
	for _, r := range parent.Relations {
		if r.Kind != "child" && r.Kind != "child-of" {
			continue
		}
		declaredCount++
		if statusBySlug[normalizeRelTarget(r.Target)] != StatusCompleted {
			declaredComplete = false
		}
	}
	if declaredCount > 0 && !declaredComplete {
		return false
	}

	// Child-count gate: at least one materialized spec must declare this
	// initiative as its parent, and all such children must be completed.
	allDone := true
	childCount := 0
	for _, s := range allSpecs {
		for _, r := range s.Relations {
			if (r.Kind == "parent" || r.Kind == "child-of") &&
				normalizeRelTarget(r.Target) == parent.Slug {
				childCount++
				if s.Status != StatusCompleted {
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
