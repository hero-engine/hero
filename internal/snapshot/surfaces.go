package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SurfacesOverride is the parsed form of an optional
// .hero/surfaces.yaml file. None of the fields are required; an empty
// file is valid.
type SurfacesOverride struct {
	Version   int                       `yaml:"version" json:"version"`
	Renames   []RenameOverride          `yaml:"renames" json:"renames,omitempty"`
	Ignore    []IgnoreOverride          `yaml:"ignore" json:"ignore,omitempty"`
	Additions []SurfaceAddition         `yaml:"additions" json:"additions,omitempty"`
	Overrides []SurfaceFieldOverride    `yaml:"overrides" json:"overrides,omitempty"`

	// Path is the absolute path the override was loaded from, retained
	// for diagnostics. Zero when synthesized in tests.
	Path string `yaml:"-" json:"-"`
}

type RenameOverride struct {
	From string `yaml:"from" json:"from"`
	To   string `yaml:"to" json:"to"`
}

type IgnoreOverride struct {
	ID string `yaml:"id" json:"id"`
}

type SurfaceAddition struct {
	ID             string                          `yaml:"id" json:"id"`
	Name           string                          `yaml:"name" json:"name,omitempty"`
	Intent         string                          `yaml:"intent" json:"intent,omitempty"`
	Paths          []string                        `yaml:"paths" json:"paths,omitempty"`
	Stage          string                          `yaml:"stage" json:"stage,omitempty"`
	Owner          string                          `yaml:"owner" json:"owner,omitempty"`
	ReleaseTargets map[string]ReleaseTargetEntry   `yaml:"release_targets" json:"release_targets,omitempty"`
}

type ReleaseTargetEntry struct {
	Description string `yaml:"description" json:"description,omitempty"`
	ScopeTag    string `yaml:"scope_tag" json:"scope_tag,omitempty"`
}

type SurfaceFieldOverride struct {
	ID     string   `yaml:"id" json:"id"`
	Stage  string   `yaml:"stage" json:"stage,omitempty"`
	Owner  string   `yaml:"owner" json:"owner,omitempty"`
	Intent string   `yaml:"intent" json:"intent,omitempty"`
	Paths  []string `yaml:"paths" json:"paths,omitempty"`
}

// LoadOverride parses .hero/surfaces.yaml when present. Absence is
// not an error — a missing file produces a zero-valued override.
func LoadOverride(heroDir string) (SurfacesOverride, error) {
	path := filepath.Join(heroDir, "surfaces.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SurfacesOverride{}, nil
		}
		return SurfacesOverride{}, err
	}
	o, err := parseOverride(string(data))
	if err != nil {
		return SurfacesOverride{}, fmt.Errorf("%s: %w", path, err)
	}
	o.Path = path
	return o, nil
}

// parseOverride is a minimal YAML-ish parser tuned for the override
// schema. It is deliberately tolerant: keys we don't know are
// silently ignored (forward-compatible per the spec's risks).
//
// Schema is shallow:
//   version: 1
//   renames: [ { from: x, to: y } ]
//   ignore:  [ { id: x } ]
//   additions: [ { id, name, intent, paths: [...], stage, owner, release_targets: { v1: { description, scope_tag } } } ]
//   overrides: [ { id, stage, owner, intent, paths: [...] } ]
//
// We avoid pulling in a YAML dependency to keep the snapshot package
// leaf — config.go is the closest peer and parses similarly.
func parseOverride(src string) (SurfacesOverride, error) {
	var out SurfacesOverride
	lines := strings.Split(src, "\n")
	i := 0
	for i < len(lines) {
		raw := lines[i]
		line := strings.TrimRight(raw, " \t\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}
		// Only top-level keys land here (no leading indent).
		if leadingSpace(line) != 0 {
			i++
			continue
		}
		// version: N
		if strings.HasPrefix(trimmed, "version:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "version:"))
			var v int
			_, _ = fmt.Sscanf(val, "%d", &v)
			out.Version = v
			i++
			continue
		}
		key := strings.TrimSuffix(trimmed, ":")
		switch key {
		case "renames":
			items, advance, err := parseListOfMaps(lines, i+1)
			if err != nil {
				return out, fmt.Errorf("renames: %w", err)
			}
			for _, m := range items {
				out.Renames = append(out.Renames, RenameOverride{From: m["from"], To: m["to"]})
			}
			i += advance + 1
		case "ignore":
			items, advance, err := parseListOfMaps(lines, i+1)
			if err != nil {
				return out, fmt.Errorf("ignore: %w", err)
			}
			for _, m := range items {
				out.Ignore = append(out.Ignore, IgnoreOverride{ID: m["id"]})
			}
			i += advance + 1
		case "additions":
			items, advance, err := parseListOfMaps(lines, i+1)
			if err != nil {
				return out, fmt.Errorf("additions: %w", err)
			}
			for _, m := range items {
				add := SurfaceAddition{
					ID:     m["id"],
					Name:   m["name"],
					Intent: m["intent"],
					Stage:  m["stage"],
					Owner:  m["owner"],
				}
				if p := m["paths"]; p != "" {
					add.Paths = parseInlineList(p)
				}
				out.Additions = append(out.Additions, add)
			}
			i += advance + 1
		case "overrides":
			items, advance, err := parseListOfMaps(lines, i+1)
			if err != nil {
				return out, fmt.Errorf("overrides: %w", err)
			}
			for _, m := range items {
				ov := SurfaceFieldOverride{
					ID:     m["id"],
					Stage:  m["stage"],
					Owner:  m["owner"],
					Intent: m["intent"],
				}
				if p := m["paths"]; p != "" {
					ov.Paths = parseInlineList(p)
				}
				out.Overrides = append(out.Overrides, ov)
			}
			i += advance + 1
		default:
			i++
		}
	}
	return out, nil
}

func leadingSpace(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 2
			continue
		}
		break
	}
	return n
}

// parseListOfMaps reads a YAML list-of-maps block starting at lines[start]
// and returns the list plus the number of lines consumed.
//
// Supported shape:
//   - id: foo
//     stage: building
//     paths:
//       - a/
//       - b/
//   - id: bar
//
// The inner "paths:" list is flattened to a comma-separated string
// stored under the "paths" key for upstream parsing.
func parseListOfMaps(lines []string, start int) ([]map[string]string, int, error) {
	var out []map[string]string
	var cur map[string]string
	var pendingKey string
	var pendingList []string

	consumed := 0
	baseIndent := -1

	flush := func() {
		if pendingKey != "" && cur != nil {
			cur[pendingKey] = strings.Join(pendingList, ",")
		}
		pendingKey = ""
		pendingList = nil
	}

	for i := start; i < len(lines); i++ {
		raw := lines[i]
		stripped := strings.TrimRight(raw, " \t\r")
		if strings.TrimSpace(stripped) == "" {
			consumed++
			continue
		}
		indent := leadingSpace(stripped)
		if indent == 0 {
			// Next top-level key — stop.
			break
		}
		if baseIndent < 0 {
			baseIndent = indent
		}
		content := strings.TrimSpace(stripped)

		// New list item: "- key: value" at baseIndent
		if strings.HasPrefix(content, "- ") && indent == baseIndent {
			flush()
			if cur != nil {
				out = append(out, cur)
			}
			cur = map[string]string{}
			rest := strings.TrimPrefix(content, "- ")
			if k, v, ok := splitKV(rest); ok {
				cur[k] = v
			}
			consumed++
			continue
		}

		// Continuation of the current item.
		if cur == nil {
			consumed++
			continue
		}

		// "paths:" or another sub-list start: a key with no value.
		if k, v, ok := splitKV(content); ok {
			flush()
			if v == "" {
				// Could be a list start; collect following "- foo" lines.
				pendingKey = k
				pendingList = nil
			} else {
				cur[k] = v
			}
			consumed++
			continue
		}

		// Sub-list item: "- foo"
		if strings.HasPrefix(content, "- ") {
			pendingList = append(pendingList, strings.TrimSpace(strings.TrimPrefix(content, "- ")))
			consumed++
			continue
		}
		consumed++
	}
	flush()
	if cur != nil {
		out = append(out, cur)
	}
	return out, consumed, nil
}

func splitKV(s string) (string, string, bool) {
	idx := strings.Index(s, ":")
	if idx < 0 {
		return "", "", false
	}
	k := strings.TrimSpace(s[:idx])
	v := strings.TrimSpace(s[idx+1:])
	v = strings.Trim(v, "\"'")
	return k, v, true
}

func parseInlineList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Merge applies the override layer to the inferred candidates and
// returns the final Surface list. Order: rename → ignore → field
// overrides → additions. Inference-only surfaces are marked
// Source="inferred"; overridden surfaces become "override"; added
// surfaces become "added". Confidence stays from the inferred entry
// when applicable; explicit additions get 1.0.
func Merge(detected []CandidateSurface, override SurfacesOverride) []Surface {
	// 1. Apply renames first so subsequent ignores/overrides can refer
	//    to the renamed id.
	renamed := make([]CandidateSurface, 0, len(detected))
	for _, c := range detected {
		for _, r := range override.Renames {
			if r.From == c.ID && r.To != "" {
				c.ID = r.To
				c.Signals = append(c.Signals, "renamed via override")
				break
			}
		}
		renamed = append(renamed, c)
	}

	// 2. Apply ignores.
	ignoreSet := map[string]bool{}
	for _, ig := range override.Ignore {
		if ig.ID != "" {
			ignoreSet[ig.ID] = true
		}
	}
	filtered := make([]Surface, 0, len(renamed))
	for _, c := range renamed {
		if ignoreSet[c.ID] {
			continue
		}
		filtered = append(filtered, Surface{
			ID:         c.ID,
			Name:       c.Name,
			Paths:      append([]string{}, c.Paths...),
			Signals:    append([]string{}, c.Signals...),
			Confidence: c.Confidence,
			Source:     "inferred",
		})
	}

	// 3. Apply per-field overrides.
	byID := map[string]int{}
	for i, s := range filtered {
		byID[s.ID] = i
	}
	for _, ov := range override.Overrides {
		idx, ok := byID[ov.ID]
		if !ok {
			// Override targets an unknown surface — skip silently to
			// stay forward-compatible. (hero check surfaces a warning.)
			continue
		}
		s := filtered[idx]
		if ov.Stage != "" && IsValidStage(Stage(ov.Stage)) {
			s.Stage = Stage(ov.Stage)
			s.StagePinned = true
		}
		if ov.Owner != "" {
			s.Owner = ov.Owner
		}
		if ov.Intent != "" {
			s.Intent = ov.Intent
		}
		if len(ov.Paths) > 0 {
			s.Paths = append([]string{}, ov.Paths...)
		}
		s.Source = "override"
		filtered[idx] = s
	}

	// 4. Append additions (lines wholly authored by the user).
	for _, add := range override.Additions {
		if add.ID == "" {
			continue
		}
		if _, exists := byID[add.ID]; exists {
			// Don't double-add an inferred id; treat as overrides instead.
			continue
		}
		name := add.Name
		if name == "" {
			name = titleCase(add.ID)
		}
		s := Surface{
			ID:         add.ID,
			Name:       name,
			Intent:     add.Intent,
			Paths:      append([]string{}, add.Paths...),
			Owner:      add.Owner,
			Confidence: 1.0,
			Signals:    []string{"declared via additions"},
			Source:     "added",
		}
		if add.Stage != "" && IsValidStage(Stage(add.Stage)) {
			s.Stage = Stage(add.Stage)
			s.StagePinned = true
		}
		for name, rt := range add.ReleaseTargets {
			s.ReleaseTargets = append(s.ReleaseTargets, ReleaseTarget{
				Name:        name,
				Description: rt.Description,
				ScopeTag:    rt.ScopeTag,
			})
		}
		filtered = append(filtered, s)
		byID[add.ID] = len(filtered) - 1
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})
	return filtered
}
