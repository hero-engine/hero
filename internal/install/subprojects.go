package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SubprojectsFile is the relative path under .hero/ where the team-shared
// subprojects manifest lives.
const SubprojectsFile = "subprojects.json"

// Subproject is a declared canonical subproject within a Hero workspace.
type Subproject struct {
	// Path is the path relative to the workspace root, in forward-slash form.
	Path string `json:"path"`
	// Scope is the canonical scope identifier (typically equal to Path,
	// but allowed to differ — e.g. an app folder might use a shorter
	// scope name than its filesystem path).
	Scope string `json:"scope"`
	// Description is an optional human-readable note shown in install
	// walkthroughs and `hero list`.
	Description string `json:"description,omitempty"`
}

// SubprojectsManifest is the on-disk shape of .hero/subprojects.json.
//
// This file is committed to git. It is the team-shared source of truth
// for which folders are valid subprojects of the workspace.
type SubprojectsManifest struct {
	// Subprojects lists folders that ARE subprojects.
	Subprojects []Subproject `json:"subprojects"`
	// Excluded lists folders the install detector should never offer as
	// subprojects (e.g. vendored deps with build files). Paths are
	// relative to the workspace root, in forward-slash form.
	Excluded []string `json:"excluded,omitempty"`
}

// LoadSubprojects reads the manifest from heroDir. Returns an empty
// manifest (not an error) if the file does not exist — the absence of
// the file is the natural "no subprojects declared yet" state.
func LoadSubprojects(heroDir string) (*SubprojectsManifest, error) {
	path := filepath.Join(heroDir, SubprojectsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SubprojectsManifest{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m SubprojectsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &m, nil
}

// SaveSubprojects writes the manifest to heroDir, creating .hero/ if
// needed. Subprojects and Excluded are sorted by path for stable diffs.
func SaveSubprojects(heroDir string, m *SubprojectsManifest) error {
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", heroDir, err)
	}

	out := &SubprojectsManifest{
		Subprojects: append([]Subproject(nil), m.Subprojects...),
		Excluded:    append([]string(nil), m.Excluded...),
	}
	sort.Slice(out.Subprojects, func(i, j int) bool {
		return out.Subprojects[i].Path < out.Subprojects[j].Path
	})
	sort.Strings(out.Excluded)

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(heroDir, SubprojectsFile), data, 0o644)
}

// DeclaredPaths returns the list of subproject paths in the manifest.
// Convenience for callers that only need the paths (e.g. scope matching).
func (m *SubprojectsManifest) DeclaredPaths() []string {
	if m == nil {
		return nil
	}
	out := make([]string, 0, len(m.Subprojects))
	for _, sp := range m.Subprojects {
		out = append(out, sp.Path)
	}
	return out
}

// IsExcluded reports whether the given path (relative to root, forward
// slashes) is in the excluded list.
func (m *SubprojectsManifest) IsExcluded(relPath string) bool {
	if m == nil {
		return false
	}
	norm := normalizeRelPath(relPath)
	for _, e := range m.Excluded {
		if normalizeRelPath(e) == norm {
			return true
		}
	}
	return false
}

// IsDeclared reports whether the given path is in the subprojects list.
func (m *SubprojectsManifest) IsDeclared(relPath string) bool {
	if m == nil {
		return false
	}
	norm := normalizeRelPath(relPath)
	for _, sp := range m.Subprojects {
		if normalizeRelPath(sp.Path) == norm {
			return true
		}
	}
	return false
}

// AddSubproject inserts or updates a subproject entry in the manifest.
func (m *SubprojectsManifest) AddSubproject(sp Subproject) {
	sp.Path = normalizeRelPath(sp.Path)
	if sp.Scope == "" {
		sp.Scope = sp.Path
	}
	for i, existing := range m.Subprojects {
		if normalizeRelPath(existing.Path) == sp.Path {
			m.Subprojects[i] = sp
			return
		}
	}
	m.Subprojects = append(m.Subprojects, sp)
}

// AddExcluded inserts a path into the excluded list (idempotent).
func (m *SubprojectsManifest) AddExcluded(relPath string) {
	norm := normalizeRelPath(relPath)
	for _, e := range m.Excluded {
		if normalizeRelPath(e) == norm {
			return
		}
	}
	m.Excluded = append(m.Excluded, norm)
}

// RemoveSubproject removes a subproject by path. Returns true if removed.
func (m *SubprojectsManifest) RemoveSubproject(relPath string) bool {
	norm := normalizeRelPath(relPath)
	for i, sp := range m.Subprojects {
		if normalizeRelPath(sp.Path) == norm {
			m.Subprojects = append(m.Subprojects[:i], m.Subprojects[i+1:]...)
			return true
		}
	}
	return false
}

// normalizeRelPath produces a canonical forward-slash, slash-trimmed,
// cleaned form for a path-relative-to-root.
func normalizeRelPath(p string) string {
	p = strings.TrimSpace(p)
	p = filepath.ToSlash(p)
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(p))
}
