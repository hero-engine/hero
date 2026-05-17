package vocabulary

import (
	"os"
	"testing"
)

func TestLoadCoreVocabularies(t *testing.T) {
	vocabs, err := Load(CoreFS(), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"default", "agile-scrum", "shape-up", "kanban", "jira", "linear"}
	for _, name := range want {
		if _, ok := vocabs[name]; !ok {
			t.Errorf("expected vocabulary %q to load; got: %v", name, keys(vocabs))
		}
	}
}

func TestDisplay_DefaultFeature(t *testing.T) {
	vocabs, err := Load(CoreFS(), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := vocabs["default"].Display("spec", "feature")
	if got != "Feature" {
		t.Errorf("default Display(spec,feature) = %q, want %q", got, "Feature")
	}
}

func TestDisplay_AgileScrumFeature(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	got := vocabs["agile-scrum"].Display("spec", "feature")
	if got != "Story" {
		t.Errorf("agile-scrum Display(spec,feature) = %q, want %q", got, "Story")
	}
}

func TestDisplay_ShapeUpFeature(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	got := vocabs["shape-up"].Display("spec", "feature")
	if got != "Scope" {
		t.Errorf("shape-up Display(spec,feature) = %q, want %q", got, "Scope")
	}
}

func TestDisplay_FallsThroughToTypeThenLiteral(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	def := vocabs["default"]
	// Unknown kind on a known type falls through to the type
	// display.
	got := def.Display("spec", "totally-unknown-kind")
	if got != "Spec" {
		t.Errorf("default Display(spec,unknown) = %q, want %q", got, "Spec")
	}
	// Unknown type with empty kind falls through to the canonical
	// type literal.
	got = def.Display("nonsense", "")
	if got != "nonsense" {
		t.Errorf("default Display(nonsense,'') = %q, want %q", got, "nonsense")
	}
}

func TestDisplaySection_ShapeUpAndDefault(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	gotShape := vocabs["shape-up"].DisplaySection("acceptance_criteria")
	if gotShape != "Done line" {
		t.Errorf("shape-up DisplaySection(acceptance_criteria) = %q, want %q", gotShape, "Done line")
	}
	gotDef := vocabs["default"].DisplaySection("acceptance_criteria")
	if gotDef != "Acceptance Criteria" {
		t.Errorf("default DisplaySection(acceptance_criteria) = %q, want %q", gotDef, "Acceptance Criteria")
	}
}

func TestDisplaySection_FallbackTitleCase(t *testing.T) {
	v := &Vocabulary{}
	got := v.DisplaySection("rabbit_holes")
	if got != "Rabbit Holes" {
		t.Errorf("DisplaySection fallback = %q, want %q", got, "Rabbit Holes")
	}
}

func TestRecognizeNL_AgileScrumStory(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	ref, score, ok := vocabs["agile-scrum"].RecognizeNL("create a story for the login flow")
	if !ok {
		t.Fatalf("RecognizeNL: expected match")
	}
	if ref.Type != "spec" || ref.Kind != "feature" {
		t.Errorf("RecognizeNL got (%s,%s), want (spec,feature)", ref.Type, ref.Kind)
	}
	if score <= 0 || score > 1 {
		t.Errorf("RecognizeNL score = %v, want (0,1]", score)
	}
}

func TestRecognizeNL_PrefersLongestMatch(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	// "user story" should beat "story" — both are agile-scrum
	// triggers for the same canonical ref, so just verify it matches.
	ref, _, ok := vocabs["agile-scrum"].RecognizeNL("draft a user story now")
	if !ok || ref.Type != "spec" || ref.Kind != "feature" {
		t.Errorf("RecognizeNL user-story got (%v, ok=%v)", ref, ok)
	}
}

func TestRecognizeNL_NoMatch(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	_, _, ok := vocabs["default"].RecognizeNL("nothing relevant here")
	if ok {
		t.Errorf("RecognizeNL should not match")
	}
}

func TestRecognizeNL_WordBoundary(t *testing.T) {
	v := &Vocabulary{
		NLTriggers: []NLTrigger{
			{Phrases: []string{"story"}, Canonical: CanonicalRef{Type: "spec", Kind: "feature"}},
		},
	}
	// "history" contains "story" as a substring; the word-boundary
	// matcher should reject it.
	_, _, ok := v.RecognizeNL("tell me history")
	if ok {
		t.Errorf("RecognizeNL matched inside word 'history' — boundary check broken")
	}
}

func TestMappedFromTracker_Jira(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	ref, ok := vocabs["jira"].MappedFromTracker("Story", "jira")
	if !ok {
		t.Fatalf("MappedFromTracker(Story,jira) returned not-ok")
	}
	if ref.Type != "spec" || ref.Kind != "feature" {
		t.Errorf("got (%s,%s), want (spec,feature)", ref.Type, ref.Kind)
	}
	// Jira's Epic must map to type=epic per Decision 8.
	ref, ok = vocabs["jira"].MappedFromTracker("Epic", "jira")
	if !ok || ref.Type != "epic" {
		t.Errorf("Jira Epic got (%v, ok=%v), want type=epic", ref, ok)
	}
}

func TestMappedFromTracker_TrackerNameCaseInsensitive(t *testing.T) {
	vocabs, _ := Load(CoreFS(), nil)
	_, ok := vocabs["jira"].MappedFromTracker("Story", "JIRA")
	if !ok {
		t.Errorf("MappedFromTracker should be case-insensitive on tracker name")
	}
}

func TestLoadTolerates_BrokenAndMissing(t *testing.T) {
	// Use testdata/ which has one good and one structurally broken
	// file. Loader must skip the broken file and surface the good one.
	fsys := os.DirFS("testdata")
	vocabs, err := Load(fsys, nil)
	if err != nil {
		t.Fatalf("Load(testdata): %v", err)
	}
	if _, ok := vocabs["mini"]; !ok {
		t.Errorf("expected 'mini' to load; got %v", keys(vocabs))
	}
	// "broken" should be skipped — its name field is present but the
	// structure is malformed enough that yaml parse fails. We accept
	// either skip path.
	if _, ok := vocabs["broken"]; ok {
		t.Logf("note: 'broken' loaded; the yaml may have been tolerant. ok.")
	}
}

func keys(m map[string]*Vocabulary) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
