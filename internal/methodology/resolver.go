package methodology

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hero-engine/hero/internal/config"
)

// DefaultName is the fall-through methodology name when no explicit
// methodology is set and no inference succeeds.
const DefaultName = "kanban"

// OverrideWarnThreshold is the per-Decision-5 risk-mitigation count
// above which Resolve emits a warning log about workspace-methodology
// drift.
const OverrideWarnThreshold = 10

// Resolve returns the active methodology profile for a workspace. It
// applies the precedence chain from Decision 6 of
// unified-spec-type-model:
//
//  1. Explicit cfg.Methodology (highest priority).
//  2. Tracker-inferred — a configured tracker's type matches the
//     methodology name (jira → scrum is the only common default;
//     callers may override).
//  3. Delivery-preset-inferred — cfg.PM.Presets.Delivery maps to a
//     methodology ("sprint" → scrum, "cycle" → shape-up,
//     "flow"/"continuous" → kanban).
//  4. DefaultName (kanban — least-opinionated baseline).
//
// On top of the chosen base methodology, Resolve applies any
// cfg.MethodologyOverrides per-key. The returned methodology is a
// deep copy of the chosen base with overrides merged in; callers may
// keep it for the lifetime of the process.
//
// Resolve returns an error only when no candidate methodology exists in
// methodologies (including the default fallback).
func Resolve(cfg *config.Config, methodologies map[string]*Methodology) (*Methodology, error) {
	if len(methodologies) == 0 {
		return nil, fmt.Errorf("no methodologies loaded")
	}

	name := pickName(cfg, methodologies)
	base, ok := methodologies[name]
	if !ok {
		// Picked name does not exist — fall back to default.
		base, ok = methodologies[DefaultName]
		if !ok {
			return nil, fmt.Errorf("methodology %q not loaded and no %q fallback present", name, DefaultName)
		}
	}

	merged := cloneMethodology(base)
	if cfg != nil && len(cfg.MethodologyOverrides) > 0 {
		applyOverrides(merged, cfg.MethodologyOverrides)
		if len(cfg.MethodologyOverrides) > OverrideWarnThreshold {
			log.Printf("methodology: workspace declares %d methodology_overrides keys (>%d) — consider authoring a domain methodology instead",
				len(cfg.MethodologyOverrides), OverrideWarnThreshold)
		}
	}
	return merged, nil
}

// DeriveVocabularyName returns the vocabulary preset name that pairs
// with the active methodology when the workspace has not set an
// explicit `vocabulary:`. Implements Decision 6 step 3 ("methodology-
// implied vocabulary"). Returns "" when no auto-derivation applies,
// leaving the vocabulary resolver to fall back to its own chain.
func DeriveVocabularyName(cfg *config.Config, m *Methodology) string {
	if cfg != nil && cfg.Vocabulary != "" {
		return ""
	}
	if m == nil {
		return ""
	}
	return m.AlignedVocabulary
}

func pickName(cfg *config.Config, methodologies map[string]*Methodology) string {
	if cfg != nil && cfg.Methodology != "" {
		return cfg.Methodology
	}

	// 2. Tracker-inferred. Jira workspaces default to scrum unless the
	// workspace says otherwise; other trackers don't carry a strong
	// signal and fall through to delivery preset / default.
	if cfg != nil && cfg.Tracker != nil && cfg.Tracker.Type != "" && cfg.Tracker.Type != "none" {
		if strings.EqualFold(cfg.Tracker.Type, "jira") {
			if _, ok := methodologies["scrum"]; ok {
				return "scrum"
			}
		}
	}

	// 3. Delivery-preset-inferred.
	if cfg != nil && cfg.PM != nil && cfg.PM.Presets != nil && cfg.PM.Presets.Delivery != "" {
		switch strings.ToLower(cfg.PM.Presets.Delivery) {
		case "sprint":
			if _, ok := methodologies["scrum"]; ok {
				return "scrum"
			}
		case "cycle":
			if _, ok := methodologies["shape-up"]; ok {
				return "shape-up"
			}
		case "flow", "continuous":
			if _, ok := methodologies["kanban"]; ok {
				return "kanban"
			}
		}
	}

	return DefaultName
}

func cloneMethodology(m *Methodology) *Methodology {
	if m == nil {
		return nil
	}
	out := &Methodology{
		Name:              m.Name,
		DisplayName:       m.DisplayName,
		Description:       m.Description,
		AlignedVocabulary: m.AlignedVocabulary,
		Lifecycle:         map[string]StateMachine{},
		Estimation:        map[string]Estimation{},
		InFlightTracking:  m.InFlightTracking,
	}
	for typeName, sm := range m.Lifecycle {
		copySm := StateMachine{
			States:      append([]string(nil), sm.States...),
			Transitions: append([]Transition(nil), sm.Transitions...),
		}
		out.Lifecycle[typeName] = copySm
	}
	for _, tb := range m.TimeBoxes {
		copyTb := TimeBox{
			Level:           tb.Level,
			ArtifactType:    tb.ArtifactType,
			DurationDefault: tb.DurationDefault,
			DurationTypical: tb.DurationTypical,
			Required:        tb.Required,
			Rituals:         append([]Ritual(nil), tb.Rituals...),
		}
		out.TimeBoxes = append(out.TimeBoxes, copyTb)
	}
	for typeName, e := range m.Estimation {
		out.Estimation[typeName] = Estimation{
			RequiredField: e.RequiredField,
			Scale:         append([]string(nil), e.Scale...),
			Optional:      e.Optional,
		}
	}
	out.Cadence = Cadence{
		DailyStandup: m.Cadence.DailyStandup,
		WeeklySync:   m.Cadence.WeeklySync,
		Rituals:      append([]Ritual(nil), m.Cadence.Rituals...),
	}
	out.Rollups = append([]Rollup(nil), m.Rollups...)
	return out
}

// applyOverrides merges per-key overrides onto a methodology in place.
// Recognized key shapes (all dot-delimited, lowercase prefix):
//
//	time_boxes.<level>.duration_default   -> TimeBox.DurationDefault
//	time_boxes.<level>.required            -> TimeBox.Required (parsed as bool)
//	estimation.<type>.required_field       -> Estimation[type].RequiredField
//	estimation.<type>.optional             -> Estimation[type].Optional (bool)
//	in_flight_tracking                     -> InFlightTracking
//	cadence.daily_standup                  -> Cadence.DailyStandup (bool)
//	cadence.weekly_sync                    -> Cadence.WeeklySync (bool)
//
// Unrecognized keys are logged and ignored.
func applyOverrides(m *Methodology, overrides map[string]string) {
	for key, val := range overrides {
		switch {
		case key == "in_flight_tracking":
			m.InFlightTracking = val
		case key == "cadence.daily_standup":
			m.Cadence.DailyStandup = parseBool(val)
		case key == "cadence.weekly_sync":
			m.Cadence.WeeklySync = parseBool(val)
		case strings.HasPrefix(key, "time_boxes."):
			rest := strings.TrimPrefix(key, "time_boxes.")
			level, field, ok := splitDot(rest)
			if !ok {
				log.Printf("methodology: ignoring malformed time_boxes override key %q (need time_boxes.<level>.<field>)", key)
				continue
			}
			idx := findTimeBox(m, level)
			if idx < 0 {
				log.Printf("methodology: ignoring time_boxes override for unknown level %q (key %q)", level, key)
				continue
			}
			switch field {
			case "duration_default":
				m.TimeBoxes[idx].DurationDefault = val
			case "duration_typical":
				m.TimeBoxes[idx].DurationTypical = val
			case "required":
				m.TimeBoxes[idx].Required = parseBool(val)
			default:
				log.Printf("methodology: ignoring unrecognized time_boxes field %q (key %q)", field, key)
			}
		case strings.HasPrefix(key, "estimation."):
			rest := strings.TrimPrefix(key, "estimation.")
			typeName, field, ok := splitDot(rest)
			if !ok {
				log.Printf("methodology: ignoring malformed estimation override key %q (need estimation.<type>.<field>)", key)
				continue
			}
			e := m.Estimation[typeName]
			switch field {
			case "required_field":
				e.RequiredField = val
			case "optional":
				e.Optional = parseBool(val)
			default:
				log.Printf("methodology: ignoring unrecognized estimation field %q (key %q)", field, key)
				continue
			}
			if m.Estimation == nil {
				m.Estimation = map[string]Estimation{}
			}
			m.Estimation[typeName] = e
		default:
			log.Printf("methodology: ignoring unrecognized override key %q", key)
		}
	}
}

func findTimeBox(m *Methodology, level string) int {
	for i, tb := range m.TimeBoxes {
		if tb.Level == level {
			return i
		}
	}
	return -1
}

func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}

func splitDot(s string) (string, string, bool) {
	i := strings.Index(s, ".")
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}
