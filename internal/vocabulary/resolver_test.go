package vocabulary

import (
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func mustLoad(t *testing.T) map[string]*Vocabulary {
	t.Helper()
	vocabs, err := Load(CoreFS(), nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return vocabs
}

func TestResolve_EmptyConfigDefault(t *testing.T) {
	vocabs := mustLoad(t)
	v, err := Resolve(&config.Config{}, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "default" {
		t.Errorf("Resolve empty cfg = %q, want %q", v.Name, "default")
	}
}

func TestResolve_NilConfigDefault(t *testing.T) {
	vocabs := mustLoad(t)
	v, err := Resolve(nil, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "default" {
		t.Errorf("Resolve(nil) = %q, want default", v.Name)
	}
}

func TestResolve_ExplicitVocabularyWins(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{Vocabulary: "jira"}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "jira" {
		t.Errorf("Resolve explicit got %q, want jira", v.Name)
	}
}

func TestResolve_TrackerInfersJira(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{Tracker: &config.TrackerConfig{Type: "jira"}}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "jira" {
		t.Errorf("Resolve tracker=jira got %q, want jira", v.Name)
	}
}

func TestResolve_TrackerInfersLinear(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{Tracker: &config.TrackerConfig{Type: "linear"}}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "linear" {
		t.Errorf("Resolve tracker=linear got %q, want linear", v.Name)
	}
}

func TestResolve_DeliveryCycleInfersShapeUp(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{
		PM: &config.PMConfig{Presets: &config.PMPresets{Delivery: "cycle"}},
	}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "shape-up" {
		t.Errorf("Resolve delivery=cycle got %q, want shape-up", v.Name)
	}
}

func TestResolve_DeliverySprintInfersAgileScrum(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{
		PM: &config.PMConfig{Presets: &config.PMPresets{Delivery: "sprint"}},
	}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "agile-scrum" {
		t.Errorf("Resolve delivery=sprint got %q, want agile-scrum", v.Name)
	}
}

func TestResolve_DeliveryFlowInfersKanban(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{
		PM: &config.PMConfig{Presets: &config.PMPresets{Delivery: "flow"}},
	}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "kanban" {
		t.Errorf("Resolve delivery=flow got %q, want kanban", v.Name)
	}
}

func TestResolve_ExplicitBeatsTracker(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{
		Vocabulary: "default",
		Tracker:    &config.TrackerConfig{Type: "jira"},
	}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "default" {
		t.Errorf("Resolve explicit-over-tracker got %q, want default", v.Name)
	}
}

func TestResolve_OverridesApplied(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{
		Vocabulary: "default",
		VocabularyOverrides: map[string]string{
			"kinds.spec.feature":           "Backlog Item",
			"types.epic":                   "Bucket",
			"sections.acceptance_criteria": "Done When",
			"lifecycle.spec.in-flight":     "Cooking",
			"malformed-key":                "should be ignored",
		},
	}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := v.Display("spec", "feature"); got != "Backlog Item" {
		t.Errorf("override Display(spec,feature) = %q, want %q", got, "Backlog Item")
	}
	if got := v.DisplayType("epic"); got != "Bucket" {
		t.Errorf("override DisplayType(epic) = %q, want %q", got, "Bucket")
	}
	if got := v.DisplaySection("acceptance_criteria"); got != "Done When" {
		t.Errorf("override DisplaySection = %q, want %q", got, "Done When")
	}
	if got := v.Lifecycle["spec"]["in-flight"]; got != "Cooking" {
		t.Errorf("override Lifecycle = %q, want %q", got, "Cooking")
	}
}

func TestResolve_UnknownVocabularyFallsBackToDefault(t *testing.T) {
	vocabs := mustLoad(t)
	cfg := &config.Config{Vocabulary: "does-not-exist"}
	v, err := Resolve(cfg, vocabs)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if v.Name != "default" {
		t.Errorf("Resolve unknown name got %q, want default fallback", v.Name)
	}
}

func TestResolve_DoesNotMutateBase(t *testing.T) {
	vocabs := mustLoad(t)
	before := vocabs["default"].DisplayType("epic")
	cfg := &config.Config{
		Vocabulary: "default",
		VocabularyOverrides: map[string]string{
			"types.epic": "Mutant",
		},
	}
	if _, err := Resolve(cfg, vocabs); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := vocabs["default"].DisplayType("epic"); got != before {
		t.Errorf("Resolve mutated cached base vocab: was %q, now %q", before, got)
	}
}
