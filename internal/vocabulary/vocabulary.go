// Package vocabulary loads, resolves, and serves vocabulary preset
// files that map Hero's canonical (type, kind) spec model to
// methodology- or tracker-specific display names. Vocabularies live
// under core/vocabularies/<name>.yaml (shared across all domains) and
// optionally per-domain at domains/<domain>/vocabularies/<name>.yaml.
//
// This package is Phase A part 1 of the unified-spec-type-model spread
// plan. It is library-only — no CLI surface; consumers (registry export,
// CLI list, dashboard, NL routing) compose canonical type/kind data
// with a resolved *Vocabulary at render time.
package vocabulary

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// Vocabulary is a loaded vocabulary preset. It carries the parsed YAML
// content normalized into Go maps and slices. Nil-safe accessors are
// provided as methods.
type Vocabulary struct {
	Name        string
	DisplayName string
	Description string

	// AutoSelect declares conditions under which this vocabulary
	// becomes the inferred default. Used by Resolve when no explicit
	// vocabulary is set in hero.json. Each rule is a single-condition
	// map (e.g. {"tracker": "jira"} or {"delivery_preset": "cycle"});
	// any matching rule causes the vocabulary to be selected.
	AutoSelect []AutoSelectRule

	// Types maps a canonical type name (e.g. "spec", "epic") to the
	// vocabulary's display string for that type.
	Types map[string]string

	// Kinds maps a canonical type.kind pair (e.g. "spec.feature") to
	// the vocabulary's display string. Takes precedence over Types
	// when present.
	Kinds map[string]string

	// Sections maps canonical section identifiers (e.g.
	// "acceptance_criteria", "tasks", or scoped like "prd.problem") to
	// the rendered section heading.
	Sections map[string]string

	// NLTriggers are the natural-language phrase → canonical (type,
	// kind) mappings used by the natural-language router.
	NLTriggers []NLTrigger

	// Lifecycle maps canonical type.status pairs (e.g. "spec.in-flight")
	// to the display string for that lifecycle state.
	Lifecycle map[string]map[string]string

	// TrackerMappings maps tracker name → tracker issue type →
	// canonical (type, kind). Consumed by importers/exporters for
	// round-trip fidelity.
	TrackerMappings map[string]map[string]TrackerMapping
}

// AutoSelectRule is a single-condition match used during Resolve's
// auto-selection precedence step. The condition map is parsed from the
// YAML auto_select list; the recognized keys today are "tracker" and
// "delivery_preset".
type AutoSelectRule struct {
	Condition map[string]string
}

// NLTrigger maps a list of phrases (any of which matches) to a
// canonical (type, kind) ref. Phrases are matched case-insensitively
// against substrings of the input.
type NLTrigger struct {
	Phrases   []string
	Canonical CanonicalRef
}

// CanonicalRef is a (type, kind) pair. Kind may be empty when the
// trigger targets a type with no canonical kind (e.g. epic, prd).
type CanonicalRef struct {
	Type string
	Kind string
}

// TrackerMapping is the canonical (type, kind) a tracker issue type
// maps to under a given vocabulary.
type TrackerMapping struct {
	Type string
	Kind string
}

// rawVocabulary is the on-disk YAML shape. Decoded from each file and
// normalized into Vocabulary. Tolerant of missing fields.
type rawVocabulary struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
	Description string `yaml:"description"`

	AutoSelect []map[string]string `yaml:"auto_select"`

	Types map[string]string `yaml:"types"`
	Kinds map[string]string `yaml:"kinds"`

	Sections map[string]string `yaml:"sections"`

	NLTriggers []rawNLTrigger `yaml:"nl_triggers"`

	Lifecycle map[string]string `yaml:"lifecycle"`

	TrackerMappings map[string]map[string]rawTrackerMapping `yaml:"tracker_mappings"`
}

type rawNLTrigger struct {
	// Phrase is the source-of-truth key in the v1 vocabulary files; it
	// holds a list of phrases (the field name is singular for author
	// readability — "phrase: [story, user story]"). Phrases is
	// accepted as a synonym for forward compatibility.
	Phrase    []string         `yaml:"phrase"`
	Phrases   []string         `yaml:"phrases"`
	Canonical rawCanonicalRef  `yaml:"canonical"`
}

type rawCanonicalRef struct {
	Type string `yaml:"type"`
	Kind string `yaml:"kind"`
}

type rawTrackerMapping struct {
	Type string `yaml:"type"`
	Kind string `yaml:"kind"`
}

// Load reads vocabulary YAML files from coreFS (rooted at the
// core/vocabularies/ directory) and optionally a domainFS (rooted at a
// domain's vocabularies/ directory). Returns a map keyed by
// vocabulary name. Files with parse errors are skipped with a warning
// — Load returns an error only on I/O failure walking the root of
// coreFS. domainFS may be nil.
//
// When a domain vocabulary uses the same name as a core vocabulary,
// the domain version replaces the core one entirely (no per-key merge
// across files; per-key overrides are the user's job via
// vocabulary_overrides in hero.json).
func Load(coreFS fs.FS, domainFS fs.FS) (map[string]*Vocabulary, error) {
	out := make(map[string]*Vocabulary)

	if coreFS != nil {
		if err := loadInto(coreFS, ".", out); err != nil {
			return nil, fmt.Errorf("loading core vocabularies: %w", err)
		}
	}

	if domainFS != nil {
		// Domain failures are warnings, not errors — a domain pack
		// without a vocabularies/ subdir is the common case.
		if err := loadInto(domainFS, ".", out); err != nil {
			log.Printf("vocabulary: skipping domain vocabularies: %v", err)
		}
	}

	return out, nil
}

func loadInto(fsys fs.FS, root string, out map[string]*Vocabulary) error {
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
		v, err := loadFile(fsys, full)
		if err != nil {
			log.Printf("vocabulary: skipping %s: %v", full, err)
			continue
		}
		stem := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		if v.Name == "" {
			log.Printf("vocabulary: skipping %s: missing required 'name' field", full)
			continue
		}
		if v.Name != stem {
			log.Printf("vocabulary: skipping %s: name %q does not match filename stem %q", full, v.Name, stem)
			continue
		}
		if len(v.Types) == 0 && len(v.Kinds) == 0 {
			log.Printf("vocabulary: skipping %s: must declare at least one of types/kinds", full)
			continue
		}
		out[v.Name] = v
	}
	return nil
}

func loadFile(fsys fs.FS, p string) (*Vocabulary, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var raw rawVocabulary
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}
	return normalize(&raw), nil
}

func normalize(r *rawVocabulary) *Vocabulary {
	v := &Vocabulary{
		Name:            r.Name,
		DisplayName:     r.DisplayName,
		Description:     r.Description,
		Types:           cloneStringMap(r.Types),
		Kinds:           cloneStringMap(r.Kinds),
		Sections:        cloneStringMap(r.Sections),
		Lifecycle:       map[string]map[string]string{},
		TrackerMappings: map[string]map[string]TrackerMapping{},
	}
	for _, rule := range r.AutoSelect {
		if len(rule) == 0 {
			continue
		}
		v.AutoSelect = append(v.AutoSelect, AutoSelectRule{Condition: cloneStringMap(rule)})
	}
	for _, t := range r.NLTriggers {
		phrases := t.Phrase
		if len(phrases) == 0 {
			phrases = t.Phrases
		}
		if len(phrases) == 0 {
			continue
		}
		v.NLTriggers = append(v.NLTriggers, NLTrigger{
			Phrases:   append([]string(nil), phrases...),
			Canonical: CanonicalRef{Type: t.Canonical.Type, Kind: t.Canonical.Kind},
		})
	}
	// Lifecycle YAML is a flat map keyed by "<type>.<status>"; split
	// into nested form so callers can index per-type cleanly.
	for k, val := range r.Lifecycle {
		typ, status, ok := splitDot(k)
		if !ok {
			continue
		}
		if v.Lifecycle[typ] == nil {
			v.Lifecycle[typ] = map[string]string{}
		}
		v.Lifecycle[typ][status] = val
	}
	for tracker, mappings := range r.TrackerMappings {
		nested := map[string]TrackerMapping{}
		for issueType, m := range mappings {
			nested[issueType] = TrackerMapping{Type: m.Type, Kind: m.Kind}
		}
		v.TrackerMappings[tracker] = nested
	}
	return v
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func splitDot(s string) (string, string, bool) {
	i := strings.Index(s, ".")
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

// Display returns the display string for a (type, kind) pair. The
// resolution order is:
//  1. Kinds["<type>.<kind>"]  — the most specific mapping
//  2. Types["<type>"]         — falls back to type-level when kind has no override
//  3. kind                    — fall through to the canonical kind literal
//  4. type                    — fall through to the canonical type literal when kind is empty
func (v *Vocabulary) Display(typeName, kind string) string {
	if v == nil {
		if kind != "" {
			return kind
		}
		return typeName
	}
	if kind != "" {
		if s, ok := v.Kinds[typeName+"."+kind]; ok && s != "" {
			return s
		}
	}
	if s, ok := v.Types[typeName]; ok && s != "" {
		return s
	}
	if kind != "" {
		return kind
	}
	return typeName
}

// DisplayType returns the display string for a type alone (no kind).
// Falls through to the canonical type literal when no override exists.
func (v *Vocabulary) DisplayType(typeName string) string {
	if v == nil {
		return typeName
	}
	if s, ok := v.Types[typeName]; ok && s != "" {
		return s
	}
	return typeName
}

// DisplaySection returns the display heading for a canonical section
// identifier (e.g. "acceptance_criteria", "tasks"). Falls through to
// a title-cased rendering of the canonical name when no override
// exists.
func (v *Vocabulary) DisplaySection(canonical string) string {
	if v != nil {
		if s, ok := v.Sections[canonical]; ok && s != "" {
			return s
		}
	}
	return defaultSectionTitle(canonical)
}

func defaultSectionTitle(canonical string) string {
	if canonical == "" {
		return ""
	}
	parts := strings.Split(canonical, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// RecognizeNL takes a user phrase (e.g. "create a story") and returns
// the canonical (type, kind) pair if any NLTrigger matches. The
// returned float is a simple confidence score in [0,1] proportional to
// the trigger phrase's length relative to the input — longer matches
// score higher. Returns false when no trigger phrase appears in the
// input.
func (v *Vocabulary) RecognizeNL(phrase string) (CanonicalRef, float64, bool) {
	if v == nil || phrase == "" {
		return CanonicalRef{}, 0, false
	}
	lower := strings.ToLower(phrase)
	var best NLTrigger
	bestLen := 0
	for _, trig := range v.NLTriggers {
		for _, p := range trig.Phrases {
			lp := strings.ToLower(p)
			if lp == "" {
				continue
			}
			if !containsWord(lower, lp) {
				continue
			}
			if len(lp) > bestLen {
				bestLen = len(lp)
				best = trig
			}
		}
	}
	if bestLen == 0 {
		return CanonicalRef{}, 0, false
	}
	score := float64(bestLen) / float64(len(lower))
	if score > 1 {
		score = 1
	}
	return best.Canonical, score, true
}

// containsWord reports whether needle appears in haystack with word
// boundaries on either side (start, end, or non-alphanumeric). Avoids
// matching "story" inside "history".
func containsWord(haystack, needle string) bool {
	idx := 0
	for {
		i := strings.Index(haystack[idx:], needle)
		if i < 0 {
			return false
		}
		start := idx + i
		end := start + len(needle)
		left := start == 0 || !isWordChar(haystack[start-1])
		right := end == len(haystack) || !isWordChar(haystack[end])
		if left && right {
			return true
		}
		idx = start + 1
		if idx >= len(haystack) {
			return false
		}
	}
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_'
}

// MappedFromTracker returns the canonical (type, kind) for a tracker
// issue type under this vocabulary. Returns false when no mapping is
// declared. Tracker name matching is case-insensitive; issue-type
// matching is case-sensitive (Jira issue types are case-sensitive on
// the tracker side).
func (v *Vocabulary) MappedFromTracker(issueType, trackerName string) (CanonicalRef, bool) {
	if v == nil {
		return CanonicalRef{}, false
	}
	for name, mappings := range v.TrackerMappings {
		if !strings.EqualFold(name, trackerName) {
			continue
		}
		if m, ok := mappings[issueType]; ok {
			return CanonicalRef{Type: m.Type, Kind: m.Kind}, true
		}
	}
	return CanonicalRef{}, false
}
