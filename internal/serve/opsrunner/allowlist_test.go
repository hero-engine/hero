package opsrunner

import (
	"testing"
)

func TestVerbs_AllowlistCompleteness(t *testing.T) {
	want := []string{
		"re-scan",
		"re-index",
		"run-check",
		"run-check-json",
		"refresh-queue",
		"capture-knowledge",
		"snapshot",
		"export",
		"stop",
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
	// Per-project AllVerbs() lists the operations surfaced on every
	// per-project Operations card; daemon-scoped verbs (currently just
	// `stop`) live in Verbs but are deliberately omitted from AllVerbs.
	// "stop" is daemon-scoped; "run-check-json" is the internal verb
	// the healthcache dispatches and is intentionally not surfaced as
	// an Operations-card button.
	daemonScoped := map[string]bool{"stop": true, "run-check-json": true}
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
	for k := range allMap {
		if !daemonScoped[k] {
			t.Errorf("Verbs has non-daemon-scoped entry %q not in AllVerbs", k)
		}
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
