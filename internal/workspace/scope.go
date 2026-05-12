package workspace

import (
	"path/filepath"
	"sort"
	"strings"
)

// RootScope is the scope identifier used when a path is at the workspace
// root or under no declared subproject.
const RootScope = ""

// Scope returns the effective scope identifier for the workspace.
//
// Resolution order:
//  1. If the workspace was located via a satellite marker that carries an
//     explicit scope, return that scope.
//  2. Otherwise, compute the longest declared subproject prefix of CWD
//     relative to Root, using the provided declared paths.
//  3. If neither matches, return RootScope.
//
// declaredPaths is the list of subproject paths from
// .hero/subprojects.json (relative to Root). Order does not matter; the
// function chooses the longest match.
func (w *Workspace) Scope(declaredPaths []string) string {
	if w == nil {
		return RootScope
	}
	if w.IsSatellite && w.MarkerScope != "" {
		return w.MarkerScope
	}
	return MatchScope(w.Root, w.CWD, declaredPaths)
}

// MatchScope returns the longest declared subproject path that is a
// directory-prefix of cwd-relative-to-root. If nothing matches, returns
// RootScope.
//
// All inputs are normalised: declared paths and the relative cwd are
// converted to forward-slash form for comparison so the result is a
// stable scope identifier across OSes.
func MatchScope(rootAbs, cwdAbs string, declaredPaths []string) string {
	rel, err := filepath.Rel(rootAbs, cwdAbs)
	if err != nil {
		return RootScope
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || rel == "" {
		return RootScope
	}
	if strings.HasPrefix(rel, "../") || rel == ".." {
		// cwd is above root — this should not happen if Locate succeeded.
		return RootScope
	}

	// Sort declared paths longest-first so the first match wins.
	sorted := make([]string, len(declaredPaths))
	copy(sorted, declaredPaths)
	sort.SliceStable(sorted, func(i, j int) bool {
		return len(sorted[i]) > len(sorted[j])
	})

	for _, p := range sorted {
		norm := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(p)), "/")
		if norm == "" || norm == "." {
			continue
		}
		if rel == norm || strings.HasPrefix(rel, norm+"/") {
			return norm
		}
	}
	return RootScope
}
