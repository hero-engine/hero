package vocabulary

import (
	"fmt"
	"log"
	"strings"

	"github.com/hero-engine/hero/internal/config"
)

// DefaultName is the fall-through vocabulary name when no explicit
// vocabulary is set and no inference succeeds.
const DefaultName = "default"

// OverrideWarnThreshold is the per-Decision-5 risk-mitigation count
// above which Resolve emits a warning log about workspace-vocabulary
// drift.
const OverrideWarnThreshold = 10

// Resolve returns the active vocabulary for a workspace. It applies
// the precedence chain from Decision 5 of unified-spec-type-model:
//
//  1. Explicit cfg.Vocabulary (highest priority).
//  2. Tracker-inferred — a configured tracker whose type matches a
//     vocabulary's auto_select { tracker: <name> } rule.
//  3. Methodology-preset-inferred — cfg.PM.Presets.Delivery matches a
//     vocabulary's auto_select { delivery_preset: <value> } rule.
//  4. DefaultName.
//
// On top of the chosen base vocabulary, Resolve applies any
// cfg.VocabularyOverrides per-key. The returned vocabulary is a deep
// copy of the chosen base with overrides merged in; callers may keep
// it for the lifetime of the process.
//
// Resolve returns an error only when no candidate vocabulary exists in
// vocabs (including the default fallback).
func Resolve(cfg *config.Config, vocabs map[string]*Vocabulary) (*Vocabulary, error) {
	if len(vocabs) == 0 {
		return nil, fmt.Errorf("no vocabularies loaded")
	}

	name := pickName(cfg, vocabs)
	base, ok := vocabs[name]
	if !ok {
		// Picked name does not exist — fall back to default.
		base, ok = vocabs[DefaultName]
		if !ok {
			return nil, fmt.Errorf("vocabulary %q not loaded and no %q fallback present", name, DefaultName)
		}
	}

	merged := cloneVocabulary(base)
	if cfg != nil && len(cfg.VocabularyOverrides) > 0 {
		applyOverrides(merged, cfg.VocabularyOverrides)
		if len(cfg.VocabularyOverrides) > OverrideWarnThreshold {
			log.Printf("vocabulary: workspace declares %d vocabulary_overrides keys (>%d) — consider authoring a domain vocabulary instead",
				len(cfg.VocabularyOverrides), OverrideWarnThreshold)
		}
	}
	return merged, nil
}

func pickName(cfg *config.Config, vocabs map[string]*Vocabulary) string {
	if cfg != nil && cfg.Vocabulary != "" {
		return cfg.Vocabulary
	}

	// 2. Tracker-inferred.
	if cfg != nil && cfg.Tracker != nil && cfg.Tracker.Type != "" && cfg.Tracker.Type != "none" {
		if name := findAutoSelect(vocabs, "tracker", cfg.Tracker.Type); name != "" {
			return name
		}
	}

	// 3. Methodology-preset-inferred. We map the configured delivery
	// preset onto the vocabulary's declared auto_select rule. Hero-pm
	// uses "sprint" / "cycle" / "continuous" / "flow"; vocabularies
	// declare the matching value verbatim.
	if cfg != nil && cfg.PM != nil && cfg.PM.Presets != nil && cfg.PM.Presets.Delivery != "" {
		if name := findAutoSelect(vocabs, "delivery_preset", cfg.PM.Presets.Delivery); name != "" {
			return name
		}
	}

	return DefaultName
}

func findAutoSelect(vocabs map[string]*Vocabulary, key, value string) string {
	for name, v := range vocabs {
		if v == nil {
			continue
		}
		for _, rule := range v.AutoSelect {
			if matchedValue, ok := rule.Condition[key]; ok && strings.EqualFold(matchedValue, value) {
				return name
			}
		}
	}
	return ""
}

func cloneVocabulary(v *Vocabulary) *Vocabulary {
	if v == nil {
		return nil
	}
	out := &Vocabulary{
		Name:            v.Name,
		DisplayName:     v.DisplayName,
		Description:     v.Description,
		Types:           cloneStringMap(v.Types),
		Kinds:           cloneStringMap(v.Kinds),
		Sections:        cloneStringMap(v.Sections),
		Lifecycle:       map[string]map[string]string{},
		TrackerMappings: map[string]map[string]TrackerMapping{},
	}
	for _, rule := range v.AutoSelect {
		out.AutoSelect = append(out.AutoSelect, AutoSelectRule{Condition: cloneStringMap(rule.Condition)})
	}
	for _, t := range v.NLTriggers {
		out.NLTriggers = append(out.NLTriggers, NLTrigger{
			Phrases:   append([]string(nil), t.Phrases...),
			Canonical: t.Canonical,
		})
	}
	for typ, m := range v.Lifecycle {
		out.Lifecycle[typ] = cloneStringMap(m)
	}
	for tracker, mappings := range v.TrackerMappings {
		nested := make(map[string]TrackerMapping, len(mappings))
		for k, m := range mappings {
			nested[k] = m
		}
		out.TrackerMappings[tracker] = nested
	}
	return out
}

// applyOverrides merges per-key overrides onto a vocabulary in place.
// Recognized key shapes (all dot-delimited, lowercase prefix):
//
//	types.<type>                  -> Types[<type>]
//	kinds.<type>.<kind>            -> Kinds["<type>.<kind>"]
//	sections.<canonical>           -> Sections[<canonical>]
//	lifecycle.<type>.<status>      -> Lifecycle[<type>][<status>]
//
// Unrecognized keys are logged and ignored.
func applyOverrides(v *Vocabulary, overrides map[string]string) {
	for key, val := range overrides {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) < 2 {
			log.Printf("vocabulary: ignoring malformed override key %q", key)
			continue
		}
		head, rest := parts[0], parts[1]
		switch head {
		case "types":
			if v.Types == nil {
				v.Types = map[string]string{}
			}
			v.Types[rest] = val
		case "kinds":
			// rest is "<type>.<kind>" — store as a single dotted key.
			if !strings.Contains(rest, ".") {
				log.Printf("vocabulary: ignoring malformed kinds override key %q (need kinds.<type>.<kind>)", key)
				continue
			}
			if v.Kinds == nil {
				v.Kinds = map[string]string{}
			}
			v.Kinds[rest] = val
		case "sections":
			if v.Sections == nil {
				v.Sections = map[string]string{}
			}
			v.Sections[rest] = val
		case "lifecycle":
			typ, status, ok := splitDot(rest)
			if !ok {
				log.Printf("vocabulary: ignoring malformed lifecycle override key %q (need lifecycle.<type>.<status>)", key)
				continue
			}
			if v.Lifecycle == nil {
				v.Lifecycle = map[string]map[string]string{}
			}
			if v.Lifecycle[typ] == nil {
				v.Lifecycle[typ] = map[string]string{}
			}
			v.Lifecycle[typ][status] = val
		default:
			log.Printf("vocabulary: ignoring unrecognized override key prefix %q (got %q)", head, key)
		}
	}
}
