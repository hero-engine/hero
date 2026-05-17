package methodology

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCoreMethodologies(t *testing.T) {
	got, err := Load(CoreFS(), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"scrum", "kanban", "shape-up", "waterfall", "scrumban"}
	for _, name := range want {
		if _, ok := got[name]; !ok {
			t.Errorf("expected methodology %q to load; got: %v", name, keys(got))
		}
	}
	if len(got) != len(want) {
		t.Errorf("loaded %d methodologies, want exactly %d; got: %v", len(got), len(want), keys(got))
	}
}

func TestLoad_ScrumShape(t *testing.T) {
	methodologies, err := Load(CoreFS(), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	scrum := methodologies["scrum"]
	if scrum == nil {
		t.Fatal("scrum not loaded")
	}
	if scrum.AlignedVocabulary != "agile-scrum" {
		t.Errorf("scrum.AlignedVocabulary = %q, want %q", scrum.AlignedVocabulary, "agile-scrum")
	}
	if scrum.InFlightTracking != "burndown" {
		t.Errorf("scrum.InFlightTracking = %q, want %q", scrum.InFlightTracking, "burndown")
	}
	if !scrum.Cadence.DailyStandup {
		t.Error("scrum.Cadence.DailyStandup should be true")
	}
	sm := scrum.LifecycleFor("feature")
	if len(sm.States) == 0 {
		t.Fatal("scrum feature lifecycle is empty")
	}
	if sm.States[0] != "backlog" || sm.States[len(sm.States)-1] != "done" {
		t.Errorf("scrum feature lifecycle states unexpected: %v", sm.States)
	}
}

func TestLoad_ShapeUpEstimationAppetite(t *testing.T) {
	methodologies, _ := Load(CoreFS(), nil)
	su := methodologies["shape-up"]
	if su == nil {
		t.Fatal("shape-up not loaded")
	}
	field, kind := su.EstimationField("feature")
	if field != "appetite" {
		t.Errorf("shape-up feature estimation field = %q, want %q", field, "appetite")
	}
	if kind != "appetite" {
		t.Errorf("shape-up feature estimation kind = %q, want %q", kind, "appetite")
	}
	// shape-up declares a scale [small, big]
	e := su.Estimation["feature"]
	if len(e.Scale) != 2 {
		t.Errorf("shape-up feature scale len = %d, want 2 (%v)", len(e.Scale), e.Scale)
	}
}

func TestLoad_ScrumEstimationPointsScale(t *testing.T) {
	methodologies, _ := Load(CoreFS(), nil)
	scrum := methodologies["scrum"]
	field, kind := scrum.EstimationField("feature")
	if field != "points" || kind != "points" {
		t.Errorf("scrum feature estimation = (%q, %q), want (points, points)", field, kind)
	}
	e := scrum.Estimation["feature"]
	// Fibonacci 1, 2, 3, 5, 8, 13, 21
	if len(e.Scale) != 7 {
		t.Errorf("scrum feature scale len = %d, want 7 (%v)", len(e.Scale), e.Scale)
	}
	if e.Scale[0] != "1" || e.Scale[6] != "21" {
		t.Errorf("scrum feature scale bounds unexpected: %v", e.Scale)
	}
}

func TestTimeBoxRequired(t *testing.T) {
	methodologies, _ := Load(CoreFS(), nil)
	if !methodologies["scrum"].TimeBoxRequired("iteration") {
		t.Error("scrum should require iteration time-box")
	}
	if methodologies["scrum"].TimeBoxRequired("release") {
		t.Error("scrum should NOT require release time-box (it's optional)")
	}
	if methodologies["kanban"].TimeBoxRequired("iteration") {
		t.Error("kanban should NOT require iteration time-box")
	}
	if !methodologies["waterfall"].TimeBoxRequired("release") {
		t.Error("waterfall should require release time-box")
	}
	if !methodologies["shape-up"].TimeBoxRequired("release") {
		t.Error("shape-up should require release time-box (the 6w cycle)")
	}
}

func TestTimeBoxFor_DurationDefault(t *testing.T) {
	methodologies, _ := Load(CoreFS(), nil)
	tb, ok := methodologies["scrum"].TimeBoxFor("iteration")
	if !ok {
		t.Fatal("scrum iteration time-box missing")
	}
	if tb.DurationDefault != "2w" {
		t.Errorf("scrum iteration duration_default = %q, want %q", tb.DurationDefault, "2w")
	}
	tb, ok = methodologies["shape-up"].TimeBoxFor("release")
	if !ok {
		t.Fatal("shape-up release time-box missing")
	}
	if tb.DurationDefault != "6w" {
		t.Errorf("shape-up release duration_default = %q, want %q", tb.DurationDefault, "6w")
	}
}

func TestLoad_KanbanNoEstimation(t *testing.T) {
	methodologies, _ := Load(CoreFS(), nil)
	field, kind := methodologies["kanban"].EstimationField("feature")
	if field != "none" || kind != "none" {
		t.Errorf("kanban feature estimation = (%q, %q), want (none, none)", field, kind)
	}
}

func TestLoad_WaterfallDateEstimation(t *testing.T) {
	methodologies, _ := Load(CoreFS(), nil)
	field, kind := methodologies["waterfall"].EstimationField("release")
	if field != "end_date" || kind != "date" {
		t.Errorf("waterfall release estimation = (%q, %q), want (end_date, date)", field, kind)
	}
}

func TestLoad_ScrumbanInheritsScrumLifecycle(t *testing.T) {
	methodologies, _ := Load(CoreFS(), nil)
	sb := methodologies["scrumban"]
	if sb == nil {
		t.Fatal("scrumban not loaded")
	}
	if sb.AlignedVocabulary != "agile-scrum" {
		t.Errorf("scrumban.AlignedVocabulary = %q, want agile-scrum", sb.AlignedVocabulary)
	}
	if sb.InFlightTracking != "mixed" {
		t.Errorf("scrumban.InFlightTracking = %q, want mixed", sb.InFlightTracking)
	}
	tb, ok := sb.TimeBoxFor("iteration")
	if !ok {
		t.Fatal("scrumban iteration time-box missing")
	}
	if tb.Required {
		t.Error("scrumban iteration should be optional (required=false)")
	}
}

func TestLoad_DomainFSNilOk(t *testing.T) {
	got, err := Load(CoreFS(), nil)
	if err != nil {
		t.Fatalf("Load(coreFS, nil): %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected non-empty methodologies")
	}
}

func TestLoad_BrokenFileSkipped(t *testing.T) {
	dir := t.TempDir()
	// Good file.
	good := `name: tiny
display_name: Tiny
aligned_vocabulary: default
in_flight_tracking: none
`
	if err := os.WriteFile(filepath.Join(dir, "tiny.yaml"), []byte(good), 0o644); err != nil {
		t.Fatalf("write tiny: %v", err)
	}
	// Broken: name does not match filename stem.
	broken := `name: wrong
`
	if err := os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte(broken), 0o644); err != nil {
		t.Fatalf("write broken: %v", err)
	}
	got, err := Load(os.DirFS(dir), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := got["tiny"]; !ok {
		t.Errorf("tiny should load: %v", keys(got))
	}
	if _, ok := got["broken"]; ok {
		t.Errorf("broken should not load: %v", keys(got))
	}
}

func keys(m map[string]*Methodology) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
