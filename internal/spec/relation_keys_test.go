package spec

import (
	"testing"
	"time"
)

// TestNearMissRelationKey covers AC-6's classifier: a key that reads like a
// relation is named with what it probably meant, and a key that doesn't is
// left alone. The negative half matters most — a warning that fires on an
// author's own bookkeeping fields stops being read.
func TestNearMissRelationKey(t *testing.T) {
	hits := map[string]string{
		"subspecs":      "child",
		"child_specs":   "child",
		"sub-tasks":     "child",
		"parents":       "parent",
		"epic":          "parent",
		"depends":       "depends-on",
		"dependencies":  "depends-on",
		"blocked_by":    "depends-on",
		"blocks":        "blocks",
		"related":       "relates-to",
		"relates":       "relates-to",
		"superseded_by": "supersedes",
		"chidl":         "child", // plain typo, caught by edit distance
		"depends-onn":   "depends-on",
	}
	for key, want := range hits {
		if got := NearMissRelationKey(key); got != want {
			t.Errorf("NearMissRelationKey(%q) = %q, want %q", key, got, want)
		}
	}

	// Accepted keys are not near misses — they form edges already.
	for _, key := range []string{"parent", "child", "children", "child-of", "child_of",
		"depends-on", "depends_on", "initiative", "supersedes", "relates-to", "conflicts_with"} {
		if got := NearMissRelationKey(key); got != "" {
			t.Errorf("NearMissRelationKey(%q) = %q, want \"\" (key is accepted)", key, got)
		}
	}

	// Real frontmatter fields and unrelated keys must stay quiet.
	for _, key := range []string{"title", "type", "status", "slug", "tags", "size", "domain",
		"priority", "severity", "created", "surface", "owner", "owner_history", "autonomy",
		"release_target", "delivery_method", "tracker_id", "component", "area", "team",
		"scope", "triggers", "smoke", "synthesized_from", "received_from", "source"} {
		if got := NearMissRelationKey(key); got != "" {
			t.Errorf("NearMissRelationKey(%q) = %q, want \"\" (not a relation)", key, got)
		}
	}
}

// TestUnknownKeysRecordedNotDropped proves the parser stops discarding
// unrecognized keys without a trace, and that known + tracker-prefixed keys
// don't pollute the list.
func TestUnknownKeysRecordedNotDropped(t *testing.T) {
	s, err := Parse(`---
title: Governance
type: initiative
status: planning
slug: gov
jira_status: In Progress
subspecs: [alpha, bravo]
bespoke_field: whatever
---
# Governance
`, "/project/.hero/planning/initiatives/gov/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{"subspecs", "bespoke_field"}
	if len(s.UnknownKeys) != len(want) {
		t.Fatalf("UnknownKeys = %v, want %v", s.UnknownKeys, want)
	}
	for i := range want {
		if s.UnknownKeys[i] != want[i] {
			t.Fatalf("UnknownKeys = %v, want %v", s.UnknownKeys, want)
		}
	}

	misses := NearMissRelationKeys(s)
	if len(misses) != 1 || misses[0].Key != "subspecs" || misses[0].Meant != "child" {
		t.Fatalf("NearMissRelationKeys = %+v, want one subspecs→child hit", misses)
	}
}

// TestNearMissNotReportedForAcceptedPluralChildren closes the loop between the
// alias and the warning: now that `children:` parses, it must not also be
// reported as a near miss.
func TestNearMissNotReportedForAcceptedPluralChildren(t *testing.T) {
	s, err := Parse(`---
title: Governance
type: initiative
status: planning
slug: gov
children: [alpha, bravo]
---
# Governance
`, "/project/.hero/planning/initiatives/gov/spec.md", time.Now())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(s.UnknownKeys) != 0 {
		t.Errorf("UnknownKeys = %v, want empty (children: is accepted)", s.UnknownKeys)
	}
	if misses := NearMissRelationKeys(s); len(misses) != 0 {
		t.Errorf("NearMissRelationKeys = %+v, want none", misses)
	}
}
