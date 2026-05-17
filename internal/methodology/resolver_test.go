package methodology

import (
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func mustLoad(t *testing.T) map[string]*Methodology {
	t.Helper()
	m, err := Load(CoreFS(), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return m
}

func TestResolve_EmptyConfigDefault(t *testing.T) {
	m, err := Resolve(&config.Config{}, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != DefaultName {
		t.Errorf("Resolve empty cfg = %q, want %q", m.Name, DefaultName)
	}
}

func TestResolve_NilConfigDefault(t *testing.T) {
	m, err := Resolve(nil, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != DefaultName {
		t.Errorf("Resolve(nil) = %q, want %q", m.Name, DefaultName)
	}
}

func TestResolve_ExplicitWins(t *testing.T) {
	cfg := &config.Config{Methodology: "shape-up"}
	m, err := Resolve(cfg, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != "shape-up" {
		t.Errorf("Resolve explicit got %q, want shape-up", m.Name)
	}
}

func TestResolve_TrackerJiraInfersScrum(t *testing.T) {
	cfg := &config.Config{Tracker: &config.TrackerConfig{Type: "jira"}}
	m, err := Resolve(cfg, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != "scrum" {
		t.Errorf("Resolve tracker=jira got %q, want scrum", m.Name)
	}
}

func TestResolve_DeliveryPresetSprintInfersScrum(t *testing.T) {
	cfg := &config.Config{
		PM: &config.PMConfig{Presets: &config.PMPresets{Delivery: "sprint"}},
	}
	m, err := Resolve(cfg, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != "scrum" {
		t.Errorf("Resolve delivery=sprint got %q, want scrum", m.Name)
	}
}

func TestResolve_DeliveryPresetCycleInfersShapeUp(t *testing.T) {
	cfg := &config.Config{
		PM: &config.PMConfig{Presets: &config.PMPresets{Delivery: "cycle"}},
	}
	m, err := Resolve(cfg, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != "shape-up" {
		t.Errorf("Resolve delivery=cycle got %q, want shape-up", m.Name)
	}
}

func TestResolve_DeliveryPresetFlowInfersKanban(t *testing.T) {
	cfg := &config.Config{
		PM: &config.PMConfig{Presets: &config.PMPresets{Delivery: "flow"}},
	}
	m, err := Resolve(cfg, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != "kanban" {
		t.Errorf("Resolve delivery=flow got %q, want kanban", m.Name)
	}
}

func TestResolve_ExplicitBeatsTracker(t *testing.T) {
	cfg := &config.Config{
		Methodology: "kanban",
		Tracker:     &config.TrackerConfig{Type: "jira"},
	}
	m, err := Resolve(cfg, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != "kanban" {
		t.Errorf("Resolve explicit-over-tracker got %q, want kanban", m.Name)
	}
}

func TestResolve_UnknownNameFallsBackToDefault(t *testing.T) {
	cfg := &config.Config{Methodology: "does-not-exist"}
	m, err := Resolve(cfg, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.Name != DefaultName {
		t.Errorf("Resolve unknown name got %q, want %q fallback", m.Name, DefaultName)
	}
}

func TestResolve_DoesNotMutateBase(t *testing.T) {
	methodologies := mustLoad(t)
	before := methodologies["scrum"].InFlightTracking
	cfg := &config.Config{
		Methodology: "scrum",
		MethodologyOverrides: map[string]string{
			"in_flight_tracking": "wip_aging",
		},
	}
	if _, err := Resolve(cfg, methodologies); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := methodologies["scrum"].InFlightTracking; got != before {
		t.Errorf("Resolve mutated cached base methodology: was %q, now %q", before, got)
	}
}

func TestResolve_OverridesApplied(t *testing.T) {
	cfg := &config.Config{
		Methodology: "scrum",
		MethodologyOverrides: map[string]string{
			"in_flight_tracking":                  "hill_chart",
			"time_boxes.iteration.duration_default": "3w",
			"time_boxes.iteration.required":         "false",
			"estimation.feature.required_field":     "appetite",
			"cadence.daily_standup":                 "false",
			"malformed-key":                         "ignored",
		},
	}
	m, err := Resolve(cfg, mustLoad(t))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if m.InFlightTracking != "hill_chart" {
		t.Errorf("override in_flight_tracking = %q, want hill_chart", m.InFlightTracking)
	}
	tb, ok := m.TimeBoxFor("iteration")
	if !ok {
		t.Fatal("iteration time-box missing post-override")
	}
	if tb.DurationDefault != "3w" {
		t.Errorf("override duration_default = %q, want 3w", tb.DurationDefault)
	}
	if tb.Required {
		t.Error("override iteration.required should be false")
	}
	if e := m.Estimation["feature"]; e.RequiredField != "appetite" {
		t.Errorf("override estimation.feature.required_field = %q, want appetite", e.RequiredField)
	}
	if m.Cadence.DailyStandup {
		t.Error("override cadence.daily_standup should be false")
	}
}

func TestDeriveVocabularyName_AlignedDefault(t *testing.T) {
	methodologies := mustLoad(t)
	cfg := &config.Config{Methodology: "scrum"}
	m, err := Resolve(cfg, methodologies)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if name := DeriveVocabularyName(cfg, m); name != "agile-scrum" {
		t.Errorf("DeriveVocabularyName(scrum) = %q, want agile-scrum", name)
	}
}

func TestDeriveVocabularyName_ExplicitVocabularyWins(t *testing.T) {
	methodologies := mustLoad(t)
	cfg := &config.Config{Methodology: "scrum", Vocabulary: "jira"}
	m, _ := Resolve(cfg, methodologies)
	// Explicit vocabulary in cfg short-circuits derivation.
	if name := DeriveVocabularyName(cfg, m); name != "" {
		t.Errorf("DeriveVocabularyName with explicit vocab = %q, want \"\" (no derivation)", name)
	}
}

func TestDeriveVocabularyName_ShapeUpAlignment(t *testing.T) {
	methodologies := mustLoad(t)
	cfg := &config.Config{Methodology: "shape-up"}
	m, _ := Resolve(cfg, methodologies)
	if name := DeriveVocabularyName(cfg, m); name != "shape-up" {
		t.Errorf("DeriveVocabularyName(shape-up) = %q, want shape-up", name)
	}
}

func TestDeriveVocabularyName_WaterfallDefault(t *testing.T) {
	methodologies := mustLoad(t)
	cfg := &config.Config{Methodology: "waterfall"}
	m, _ := Resolve(cfg, methodologies)
	// waterfall.yaml declares aligned_vocabulary: default
	if name := DeriveVocabularyName(cfg, m); name != "default" {
		t.Errorf("DeriveVocabularyName(waterfall) = %q, want default", name)
	}
}

func TestResolve_LifecycleOverridesPresentPerType(t *testing.T) {
	methodologies := mustLoad(t)
	shapeUp := methodologies["shape-up"]
	scrum := methodologies["scrum"]
	// shape-up's feature lifecycle starts at "unshaped"; scrum's at "backlog".
	if got := shapeUp.LifecycleFor("feature").States; len(got) == 0 || got[0] != "unshaped" {
		t.Errorf("shape-up feature lifecycle start = %v, want first=unshaped", got)
	}
	if got := scrum.LifecycleFor("feature").States; len(got) == 0 || got[0] != "backlog" {
		t.Errorf("scrum feature lifecycle start = %v, want first=backlog", got)
	}
	// kanban has no sprint lifecycle (flow-based).
	if got := methodologies["kanban"].LifecycleFor("sprint").States; len(got) != 0 {
		t.Errorf("kanban should have no sprint lifecycle override, got %v", got)
	}
}

func TestResolve_EmptyMap(t *testing.T) {
	_, err := Resolve(&config.Config{}, map[string]*Methodology{})
	if err == nil {
		t.Error("Resolve with empty map should return an error")
	}
}
