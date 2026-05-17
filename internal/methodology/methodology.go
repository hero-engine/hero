// Package methodology loads, resolves, and serves methodology profile
// files that declare the structural layer of Hero's unified spec-type
// model: lifecycle state machines per spec type, time-box
// requirements, estimation fields, in-flight tracking style, cadence
// rituals, and rollups. Methodology profiles live under
// core/methodologies/<name>.yaml (shared across all domains) and
// optionally per-domain at domains/<domain>/methodologies/<name>.yaml.
//
// This package is the structural counterpart to internal/vocabulary
// (display layer). The two compose orthogonally — a workspace can run
// Scrum lifecycle with kanban vocabulary, though most teams pick
// aligned pairs. See `.hero/planning/features/unified-spec-type-model/spec.md`
// Decisions 5 and 6 for the design.
//
// Library-only: no CLI surface. Consumers (registry export, CLI list,
// NEXT.md, agent prompts) call Resolve with a config and read accessors.
package methodology

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// Methodology is a loaded methodology profile. It carries the parsed
// YAML content normalized into Go structs. Nil-safe accessors are
// provided as methods.
type Methodology struct {
	Name        string
	DisplayName string
	Description string

	// AlignedVocabulary names the vocabulary preset that pairs with
	// this methodology by default. When a workspace sets `methodology:`
	// in hero.json without an explicit `vocabulary:`, the resolver
	// auto-derives the vocabulary from this field.
	AlignedVocabulary string

	// Lifecycle maps a canonical spec type name (e.g. "feature",
	// "bug", "sprint", "release") to its state machine under this
	// methodology. Types omitted here fall back to the registry's
	// default lifecycle.
	Lifecycle map[string]StateMachine

	// TimeBoxes declares the time-box artifacts this methodology
	// requires or expects (release, iteration, cooldown, ...). Each
	// entry carries the artifact type, required flag, default
	// duration, and rituals.
	TimeBoxes []TimeBox

	// Estimation maps a canonical spec type name to its estimation
	// rule (required field, scale, optionality).
	Estimation map[string]Estimation

	// InFlightTracking names the in-flight visualization style for
	// this methodology (e.g. "burndown", "hill_chart", "wip_aging",
	// "gantt", "mixed").
	InFlightTracking string

	// Cadence captures the recurring rhythm of the methodology
	// (daily standup, weekly sync, ritual list).
	Cadence Cadence

	// Rollups declares the rollup metrics this methodology surfaces
	// (velocity, lead-time, hill-position, phase-progress, ...).
	Rollups []Rollup
}

// StateMachine is the lifecycle state machine for a single spec type
// under a methodology. States is the ordered list of valid states;
// Transitions captures the legal transitions with their gates.
type StateMachine struct {
	States      []string
	Transitions []Transition
}

// Transition is a legal lifecycle transition with a human-readable
// gate description (e.g. "AC present; estimated").
type Transition struct {
	From string
	To   string
	Gate string
}

// TimeBox is a time-box artifact declared by a methodology.
type TimeBox struct {
	// Level names the role this time-box plays (e.g. "release",
	// "iteration", "cooldown"). Multiple time-boxes per methodology
	// are allowed; level disambiguates them.
	Level string
	// ArtifactType is the canonical spec type that materializes this
	// time-box (e.g. "release", "sprint").
	ArtifactType string
	// DurationDefault is the recommended duration (e.g. "2w", "6w").
	// May be empty for time-boxes whose duration is project-scoped.
	DurationDefault string
	// DurationTypical is an informal typical-cadence label
	// (e.g. "quarter", "project_scoped", "none"). Either or both of
	// DurationDefault and DurationTypical may be set.
	DurationTypical string
	// Required reports whether this methodology requires the
	// time-box artifact to be present.
	Required bool
	// Rituals lists the rituals associated with this time-box.
	Rituals []Ritual
}

// Ritual is a recurring meeting or event tied to a time-box or to the
// overall cadence.
type Ritual struct {
	Kind string
	When string
}

// Estimation declares the estimation requirement for a spec type
// under a methodology.
type Estimation struct {
	// RequiredField names the field used for estimation
	// (e.g. "points", "appetite", "end_date", "none").
	RequiredField string
	// Scale enumerates the allowed values when applicable
	// (e.g. [1,2,3,5,8,13,21] for fibonacci points, [small, big] for
	// appetite). Values are kept as strings for cross-shape uniformity.
	Scale []string
	// Optional reports whether the estimation field is encouraged but
	// not strictly required (e.g. scrumban's points).
	Optional bool
}

// Cadence captures the recurring rhythm of a methodology.
type Cadence struct {
	DailyStandup bool
	WeeklySync   bool
	Rituals      []Ritual
}

// Rollup is a single rollup metric declared by a methodology.
type Rollup struct {
	// Kind names the metric (e.g. "velocity", "lead_time",
	// "hill_position", "phase_progress").
	Kind string
	// Over is the time window the metric spans
	// (e.g. "last_3_sprints", "last_30_days"). May be empty when the
	// metric is scope-based instead.
	Over string
	// Scope is the spatial scope the metric covers
	// (e.g. "current_sprint", "current_cycle", "current_board",
	// "current_release"). May be empty when the metric is window-based.
	Scope string
}

// rawMethodology is the on-disk YAML shape. Decoded from each file and
// normalized into Methodology. Tolerant of missing fields.
type rawMethodology struct {
	Name              string `yaml:"name"`
	DisplayName       string `yaml:"display_name"`
	Description       string `yaml:"description"`
	AlignedVocabulary string `yaml:"aligned_vocabulary"`

	Lifecycle map[string]rawStateMachine `yaml:"lifecycle"`

	TimeBoxes []rawTimeBox `yaml:"time_boxes"`

	Estimation map[string]rawEstimation `yaml:"estimation"`

	InFlightTracking string `yaml:"in_flight_tracking"`

	Cadence rawCadence `yaml:"cadence"`

	Rollups []rawRollup `yaml:"rollups"`
}

type rawStateMachine struct {
	States      []string        `yaml:"states"`
	Transitions []rawTransition `yaml:"transitions"`
}

type rawTransition struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
	Gate string `yaml:"gate"`
}

type rawTimeBox struct {
	Level           string       `yaml:"level"`
	ArtifactType    string       `yaml:"artifact_type"`
	DurationDefault string       `yaml:"duration_default"`
	DurationTypical string       `yaml:"duration_typical"`
	Required        bool         `yaml:"required"`
	Rituals         []rawRitual  `yaml:"rituals"`
}

type rawRitual struct {
	Kind string `yaml:"kind"`
	When string `yaml:"when"`
}

// rawEstimation tolerates a mixed-type scale: integer points
// (1, 2, 3...) and string appetites ("small", "big") coexist across
// profiles. yaml.Node parses both and is normalized to []string.
type rawEstimation struct {
	RequiredField string    `yaml:"required_field"`
	Scale         yaml.Node `yaml:"scale"`
	Optional      bool      `yaml:"optional"`
}

type rawCadence struct {
	DailyStandup bool        `yaml:"daily_standup"`
	WeeklySync   bool        `yaml:"weekly_sync"`
	Rituals      []rawRitual `yaml:"rituals"`
}

type rawRollup struct {
	Kind  string `yaml:"kind"`
	Over  string `yaml:"over"`
	Scope string `yaml:"scope"`
}

// Load reads methodology YAML files from coreFS (rooted at the
// core/methodologies/ directory) and optionally a domainFS (rooted at a
// domain's methodologies/ directory). Returns a map keyed by
// methodology name. Files with parse errors are skipped with a warning
// — Load returns an error only on I/O failure walking the root of
// coreFS. domainFS may be nil.
//
// When a domain methodology uses the same name as a core methodology,
// the domain version replaces the core one entirely (no per-key merge
// across files; per-key overrides are the user's job via
// methodology_overrides in hero.json).
func Load(coreFS fs.FS, domainFS fs.FS) (map[string]*Methodology, error) {
	out := make(map[string]*Methodology)

	if coreFS != nil {
		if err := loadInto(coreFS, ".", out); err != nil {
			return nil, fmt.Errorf("loading core methodologies: %w", err)
		}
	}

	if domainFS != nil {
		// Domain failures are warnings, not errors — a domain pack
		// without a methodologies/ subdir is the common case.
		if err := loadInto(domainFS, ".", out); err != nil {
			log.Printf("methodology: skipping domain methodologies: %v", err)
		}
	}

	return out, nil
}

func loadInto(fsys fs.FS, root string, out map[string]*Methodology) error {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		full := path.Join(root, name)
		m, err := loadFile(fsys, full)
		if err != nil {
			log.Printf("methodology: skipping %s: %v", full, err)
			continue
		}
		stem := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		if m.Name == "" {
			log.Printf("methodology: skipping %s: missing required 'name' field", full)
			continue
		}
		if m.Name != stem {
			log.Printf("methodology: skipping %s: name %q does not match filename stem %q", full, m.Name, stem)
			continue
		}
		out[m.Name] = m
	}
	return nil
}

func loadFile(fsys fs.FS, p string) (*Methodology, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var raw rawMethodology
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}
	return normalize(&raw), nil
}

func normalize(r *rawMethodology) *Methodology {
	m := &Methodology{
		Name:              r.Name,
		DisplayName:       r.DisplayName,
		Description:       r.Description,
		AlignedVocabulary: r.AlignedVocabulary,
		Lifecycle:         map[string]StateMachine{},
		Estimation:        map[string]Estimation{},
		InFlightTracking:  r.InFlightTracking,
	}
	for typeName, sm := range r.Lifecycle {
		out := StateMachine{
			States: append([]string(nil), sm.States...),
		}
		for _, t := range sm.Transitions {
			out.Transitions = append(out.Transitions, Transition{From: t.From, To: t.To, Gate: t.Gate})
		}
		m.Lifecycle[typeName] = out
	}
	for _, tb := range r.TimeBoxes {
		out := TimeBox{
			Level:           tb.Level,
			ArtifactType:    tb.ArtifactType,
			DurationDefault: tb.DurationDefault,
			DurationTypical: tb.DurationTypical,
			Required:        tb.Required,
		}
		for _, ritual := range tb.Rituals {
			out.Rituals = append(out.Rituals, Ritual{Kind: ritual.Kind, When: ritual.When})
		}
		m.TimeBoxes = append(m.TimeBoxes, out)
	}
	for typeName, e := range r.Estimation {
		m.Estimation[typeName] = Estimation{
			RequiredField: e.RequiredField,
			Scale:         scaleAsStrings(e.Scale),
			Optional:      e.Optional,
		}
	}
	m.Cadence = Cadence{
		DailyStandup: r.Cadence.DailyStandup,
		WeeklySync:   r.Cadence.WeeklySync,
	}
	for _, ritual := range r.Cadence.Rituals {
		m.Cadence.Rituals = append(m.Cadence.Rituals, Ritual{Kind: ritual.Kind, When: ritual.When})
	}
	for _, rr := range r.Rollups {
		m.Rollups = append(m.Rollups, Rollup{Kind: rr.Kind, Over: rr.Over, Scope: rr.Scope})
	}
	return m
}

// scaleAsStrings turns a yaml.Node holding a sequence of mixed ints
// and strings into a uniform []string. Returns nil for empty / scalar
// nodes.
func scaleAsStrings(n yaml.Node) []string {
	if n.Kind != yaml.SequenceNode {
		return nil
	}
	out := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		out = append(out, item.Value)
	}
	return out
}

// Lifecycle returns the state machine for a given canonical spec type
// under this methodology. Returns the zero value when no override is
// declared — callers should fall back to the registry's default
// lifecycle in that case.
func (m *Methodology) LifecycleFor(typeName string) StateMachine {
	if m == nil {
		return StateMachine{}
	}
	return m.Lifecycle[typeName]
}

// TimeBoxRequired reports whether the methodology requires a time-box
// artifact at the given level (e.g. "release", "iteration"). Returns
// false when the methodology declares no such time-box.
func (m *Methodology) TimeBoxRequired(level string) bool {
	if m == nil {
		return false
	}
	for _, tb := range m.TimeBoxes {
		if tb.Level == level {
			return tb.Required
		}
	}
	return false
}

// TimeBoxFor returns the time-box declaration for the given level
// (e.g. "release", "iteration", "cooldown"). The boolean is false when
// no such level is declared.
func (m *Methodology) TimeBoxFor(level string) (TimeBox, bool) {
	if m == nil {
		return TimeBox{}, false
	}
	for _, tb := range m.TimeBoxes {
		if tb.Level == level {
			return tb, true
		}
	}
	return TimeBox{}, false
}

// EstimationField returns the estimation field name and a coarse type
// label for a given canonical spec type under this methodology. The
// type label is one of "points", "appetite", "date", "none", or the
// raw field name when not recognized. Returns ("", "none") when the
// methodology declares no estimation for that type.
func (m *Methodology) EstimationField(typeName string) (name string, kind string) {
	if m == nil {
		return "", "none"
	}
	e, ok := m.Estimation[typeName]
	if !ok {
		return "", "none"
	}
	field := e.RequiredField
	switch field {
	case "", "none":
		return field, "none"
	case "points":
		return field, "points"
	case "appetite":
		return field, "appetite"
	case "end_date", "start_date":
		return field, "date"
	default:
		return field, field
	}
}
