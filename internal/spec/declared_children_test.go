package spec

import (
	"strings"
	"testing"
	"time"
)

func parseInit(t *testing.T, content string) *Spec {
	t.Helper()
	s, err := Parse(content, "/project/.hero/planning/initiatives/gov/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return s
}

func childTargets(s *Spec) []string {
	var out []string
	for _, r := range s.Relations {
		if r.Kind == "child" {
			out = append(out, r.Target)
		}
	}
	return out
}

// TestParsePluralChildrenInline covers AC-1 for the inline flow form. This is
// the exact shape that silently dropped: `children:` matched no relation case,
// fell through to default, and the initiative ended up with zero child edges.
func TestParsePluralChildrenInline(t *testing.T) {
	s := parseInit(t, `---
title: Governance
type: initiative
status: planning
slug: gov
children: [blast-radius-tiers, financial-action-gate, earned-autonomy, governance-gate]
---
# Governance
`)
	got := childTargets(s)
	want := []string{"blast-radius-tiers", "financial-action-gate", "earned-autonomy", "governance-gate"}
	if len(got) != len(want) {
		t.Fatalf("child relations = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("child relations = %v, want %v", got, want)
		}
	}
}

// TestParsePluralChildrenBlock covers AC-1 for the block form, and asserts the
// plural key normalizes to the same `child` kind as the singular spelling —
// downstream consumers must not have to know which the author wrote.
func TestParsePluralChildrenBlock(t *testing.T) {
	plural := parseInit(t, `---
title: Governance
type: initiative
status: planning
slug: gov
children:
  - alpha
  - bravo
---
# Governance
`)
	singular := parseInit(t, `---
title: Governance
type: initiative
status: planning
slug: gov
child:
  - alpha
  - bravo
---
# Governance
`)
	if len(plural.Relations) != len(singular.Relations) {
		t.Fatalf("children: produced %v, child: produced %v", plural.Relations, singular.Relations)
	}
	for i := range plural.Relations {
		if plural.Relations[i] != singular.Relations[i] {
			t.Fatalf("relation %d: children: %+v, child: %+v", i, plural.Relations[i], singular.Relations[i])
		}
	}
}

// TestParseChildOfIsAParentPointer pins the direction of the `child-of:` /
// `child_of:` aliases. "child-of: X" means X is my parent, which is how every
// consumer in the tree reads the kind — normalizing it to `child` would invert
// the edge and break parent discovery in `hero spec verify`.
func TestParseChildOfIsAParentPointer(t *testing.T) {
	for _, key := range []string{"child-of", "child_of"} {
		s := parseInit(t, `---
title: Leaf
type: bug
status: planning
slug: leaf
`+key+`: gov
---
# Leaf
`)
		if len(s.Relations) != 1 {
			t.Fatalf("%s: relations = %+v, want exactly one", key, s.Relations)
		}
		if s.Relations[0].Kind != "parent" || s.Relations[0].Target != "gov" {
			t.Errorf("%s: relation = %+v, want {gov parent}", key, s.Relations[0])
		}
	}
}

// TestDeclaredChildrenUnionsBothSources is the AC-4 fixture at the unit level:
// one function returns the union of the two rosters that used to be read
// independently, and a child named in both counts once.
func TestDeclaredChildrenUnionsBothSources(t *testing.T) {
	s := parseInit(t, `---
title: Governance
type: initiative
status: planning
slug: gov
children: [alpha, bravo]
---
# Governance

## Child Specs & Sequence

1. **[bravo](bravo/spec.md)** — declared in both places
2. **[charlie](charlie/spec.md)** — table only
3. [external](../../features/other/spec.md) — not a child link
`)
	got := DeclaredChildren(s)
	want := []string{"alpha", "bravo", "charlie"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("DeclaredChildren = %v, want %v (frontmatter first, de-duped, external ignored)", got, want)
	}
}

// TestDeclaredChildrenNormalizesPathTargets confirms a path-form relation
// target collapses to its slug, so it can be matched against on-disk specs.
func TestDeclaredChildrenNormalizesPathTargets(t *testing.T) {
	s := &Spec{Slug: "gov", Type: TypeInitiative, Relations: []Relation{
		{Kind: "child", Target: "../../features/alpha/spec.md"},
		{Kind: "child", Target: "gov"}, // self-reference is dropped
		{Kind: "depends-on", Target: "unrelated"},
	}}
	got := DeclaredChildren(s)
	if len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("DeclaredChildren = %v, want [alpha]", got)
	}
}

func mkLeaf(slug, parent string, status Status) *Spec {
	return &Spec{Slug: slug, Type: TypeBug, Status: status,
		Relations: []Relation{{Kind: "parent", Target: parent}}}
}

// TestInitiativeBlockedByUnscaffoldedChild is the regression for the reported
// failure: four children declared via `children:`, one delivered on disk. The
// initiative must not complete at 1-of-4.
func TestInitiativeBlockedByUnscaffoldedChild(t *testing.T) {
	init := parseInit(t, `---
title: Governance
type: initiative
status: planning
slug: gov
children: [blast-radius-tiers, financial-action-gate, earned-autonomy, governance-gate]
---
# Governance
`)
	all := []*Spec{init, mkLeaf("blast-radius-tiers", "gov", StatusCompleted)}
	if InitiativeReadyToComplete(init, all) {
		t.Fatal("initiative completed at 1-of-4 declared children — the roster gate is starved")
	}

	// Scaffolding a second leaf must not unblock it either (2-of-4).
	all = append(all, mkLeaf("financial-action-gate", "gov", StatusCompleted))
	if InitiativeReadyToComplete(init, all) {
		t.Fatal("initiative completed at 2-of-4 declared children")
	}

	// A declared child that exists but is not finished still blocks.
	all = append(all,
		mkLeaf("earned-autonomy", "gov", StatusCompleted),
		mkLeaf("governance-gate", "gov", StatusDelivering))
	if InitiativeReadyToComplete(init, all) {
		t.Fatal("initiative completed with a materialized but unfinished child")
	}
}

// TestInitiativeCompletesWhenFullRosterDelivered covers AC-3.
func TestInitiativeCompletesWhenFullRosterDelivered(t *testing.T) {
	init := parseInit(t, `---
title: Governance
type: initiative
status: planning
slug: gov
children: [alpha, bravo]
---
# Governance

## Child Specs & Sequence

1. **[charlie](charlie/spec.md)** — table-declared child
`)
	all := []*Spec{init,
		mkLeaf("alpha", "gov", StatusCompleted),
		mkLeaf("bravo", "gov", StatusCompleted),
	}
	if InitiativeReadyToComplete(init, all) {
		t.Fatal("table-declared charlie is unbuilt — must still block")
	}
	all = append(all, mkLeaf("charlie", "gov", StatusCompleted))
	if !InitiativeReadyToComplete(init, all) {
		t.Fatal("every declared child is completed — initiative should be ready")
	}
	// Fires only while the initiative is open; already-completed returns false
	// so the caller can't complete-and-archive it twice.
	init.Status = StatusCompleted
	if InitiativeReadyToComplete(init, all) {
		t.Fatal("an already-completed initiative must not re-complete")
	}
}

// TestInitiativeSupersededChildCountsAsFinished pins the documented escape
// hatch: an exploratory child the operator no longer intends to build is
// dropped by marking it superseded, not by leaving the initiative stuck.
func TestInitiativeSupersededChildCountsAsFinished(t *testing.T) {
	init := parseInit(t, `---
title: Governance
type: initiative
status: planning
slug: gov
children: [alpha, bravo]
---
# Governance
`)
	all := []*Spec{init,
		mkLeaf("alpha", "gov", StatusCompleted),
		mkLeaf("bravo", "gov", StatusSuperseded),
	}
	if !InitiativeReadyToComplete(init, all) {
		t.Fatal("a superseded declared child should count as finished")
	}
}

// TestInitiativeSingularChildBehaviorUnchanged is the AC-7 regression guard:
// the pre-existing singular block-style `child:` roster still completes
// exactly when it did before.
func TestInitiativeSingularChildBehaviorUnchanged(t *testing.T) {
	init := parseInit(t, `---
title: Governance
type: initiative
status: planning
slug: gov
child:
  - alpha
  - bravo
---
# Governance
`)
	partial := []*Spec{init, mkLeaf("alpha", "gov", StatusCompleted), mkLeaf("bravo", "gov", StatusPlanning)}
	if InitiativeReadyToComplete(init, partial) {
		t.Fatal("singular child: roster with an unfinished child must block")
	}
	full := []*Spec{init, mkLeaf("alpha", "gov", StatusCompleted), mkLeaf("bravo", "gov", StatusCompleted)}
	if !InitiativeReadyToComplete(init, full) {
		t.Fatal("singular child: roster fully completed must complete")
	}
}

// TestInitiativeChildCountGateStillGuards confirms the secondary gate survives
// the roster rework: an initiative that declares nothing still needs at least
// one materialized, completed child before it can auto-complete.
func TestInitiativeChildCountGateStillGuards(t *testing.T) {
	init := &Spec{Slug: "gov", Type: TypeInitiative, Status: StatusPlanning}
	if InitiativeReadyToComplete(init, []*Spec{init}) {
		t.Fatal("an initiative with no children at all must not auto-complete")
	}
	withOpenChild := []*Spec{init, mkLeaf("alpha", "gov", StatusDelivering)}
	if InitiativeReadyToComplete(init, withOpenChild) {
		t.Fatal("an undeclared but on-disk unfinished child must still block")
	}
}

// TestDeclaredChildrenIgnoresChildOfKind pins the direction of the `child-of`
// *kind* in the roster. "child-of: X" means X is this spec's parent, so an
// initiative that is itself a child must not list its own parent among its
// children — with the roster shared, that would also make drive's `done`
// unreachable, not merely over-block a completion.
func TestDeclaredChildrenIgnoresChildOfKind(t *testing.T) {
	sub := &Spec{Slug: "sub-init", Type: TypeInitiative, Status: StatusPlanning, Relations: []Relation{
		{Kind: "child-of", Target: "umbrella"},
		{Kind: "child", Target: "alpha"},
	}}
	got := DeclaredChildren(sub)
	if len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("DeclaredChildren = %v, want [alpha] (child-of names the parent, not a child)", got)
	}

	// And the gate must not wait on the umbrella initiative above it.
	all := []*Spec{sub, mkLeaf("alpha", "sub-init", StatusCompleted)}
	if !InitiativeReadyToComplete(sub, all) {
		t.Fatal("a sub-initiative must not be blocked by its own parent appearing in the roster")
	}
}

// TestChildTableSlugsFallbackIsDeterministic covers the sorted section-key
// fallback: with two `child*` sections the roster must be the same on every
// run, since a completion gate cannot depend on map iteration order.
func TestChildTableSlugsFallbackIsDeterministic(t *testing.T) {
	init := &Spec{Slug: "gov", Type: TypeInitiative, Sections: map[string]string{
		"child specs":   "1. **[alpha](alpha/spec.md)**\n",
		"child roadmap": "1. **[bravo](bravo/spec.md)**\n",
	}}
	first := DeclaredChildren(init)
	for i := 0; i < 50; i++ {
		if got := DeclaredChildren(init); strings.Join(got, ",") != strings.Join(first, ",") {
			t.Fatalf("roster varies across calls: %v then %v", first, got)
		}
	}
	if len(first) != 1 || first[0] != "bravo" {
		t.Fatalf("DeclaredChildren = %v, want [bravo] (first section by sorted key)", first)
	}
}
