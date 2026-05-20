package opsrunner

import (
	"sort"
	"testing"
)

func TestVerbs_AllowlistCompleteness(t *testing.T) {
	want := []string{
		"re-scan",
		"re-index",
		"run-check",
		"refresh-queue",
		"capture-knowledge",
		"snapshot",
		"export",
	}
	if len(Verbs) != len(want) {
		t.Fatalf("Verbs len = %d, want %d", len(Verbs), len(want))
	}
	for _, v := range want {
		args, ok := Verbs[v]
		if !ok {
			t.Errorf("missing verb %q", v)
			continue
		}
		if len(args) == 0 {
			t.Errorf("verb %q has empty args", v)
		}
	}
}

func TestVerbs_NoSurpriseEntries(t *testing.T) {
	// Belt to the suspenders: AllVerbs() and Verbs must agree on
	// membership. Anyone adding to one and not the other gets caught.
	allMap := make(map[string]bool, len(Verbs))
	for k := range Verbs {
		allMap[k] = true
	}
	for _, v := range AllVerbs() {
		if !allMap[v] {
			t.Errorf("AllVerbs includes %q not in Verbs", v)
		}
		delete(allMap, v)
	}
	if len(allMap) != 0 {
		leftover := make([]string, 0, len(allMap))
		for k := range allMap {
			leftover = append(leftover, k)
		}
		sort.Strings(leftover)
		t.Errorf("Verbs has entries not in AllVerbs: %v", leftover)
	}
}

func TestIsAllowed(t *testing.T) {
	if !IsAllowed("re-scan") {
		t.Error("expected re-scan to be allowed")
	}
	if IsAllowed("rm-rf") {
		t.Error("expected rm-rf to be denied")
	}
}

func TestVerbLabel_FallsBackToVerb(t *testing.T) {
	if got := VerbLabel("re-scan"); got == "" || got == "re-scan" {
		t.Errorf("expected human label for re-scan, got %q", got)
	}
	if got := VerbLabel("unknown-verb"); got != "unknown-verb" {
		t.Errorf("expected unknown verb to fall through to itself, got %q", got)
	}
}
