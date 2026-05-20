package data

import "testing"

type stubLookup struct {
	inFlight map[string]string // verb -> jobID
}

func (s *stubLookup) Lookup(_, verb string) (string, bool) {
	id, ok := s.inFlight[verb]
	return id, ok
}

func TestLoadOperations_NilLookup(t *testing.T) {
	out := LoadOperations(OperationsInputs{Slug: "p", Available: true})
	if len(out.Verbs) != 7 {
		t.Fatalf("verbs len = %d, want 7", len(out.Verbs))
	}
	for _, v := range out.Verbs {
		if v.InFlight {
			t.Errorf("verb %q unexpectedly in-flight with nil Lookup", v.Verb)
		}
		if v.Label == "" {
			t.Errorf("verb %q missing label", v.Verb)
		}
	}
	if !out.Available {
		t.Error("Available should propagate from inputs")
	}
}

func TestLoadOperations_InFlight(t *testing.T) {
	out := LoadOperations(OperationsInputs{
		Slug:      "p",
		Available: true,
		Lookup:    &stubLookup{inFlight: map[string]string{"re-scan": "job-abc"}},
	})
	var found bool
	for _, v := range out.Verbs {
		if v.Verb == "re-scan" {
			if !v.InFlight {
				t.Error("re-scan should be marked in flight")
			}
			if v.ActiveJobID != "job-abc" {
				t.Errorf("ActiveJobID = %q, want job-abc", v.ActiveJobID)
			}
			found = true
		} else if v.InFlight {
			t.Errorf("verb %q unexpectedly in flight", v.Verb)
		}
	}
	if !found {
		t.Fatal("re-scan verb missing from output")
	}
}

func TestLoadOperations_VerbOrder(t *testing.T) {
	out := LoadOperations(OperationsInputs{Slug: "p", Available: true})
	want := []string{"re-scan", "re-index", "run-check", "refresh-queue", "capture-knowledge", "snapshot", "export"}
	if len(out.Verbs) != len(want) {
		t.Fatalf("len = %d, want %d", len(out.Verbs), len(want))
	}
	for i, v := range out.Verbs {
		if v.Verb != want[i] {
			t.Errorf("verb[%d] = %q, want %q", i, v.Verb, want[i])
		}
	}
}
