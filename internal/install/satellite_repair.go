package install

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/workspace"
)

// DriftKind classifies a satellite-side drift finding.
type DriftKind string

const (
	DriftMissingFolder    DriftKind = "missing_folder"
	DriftBrokenSymlink    DriftKind = "broken_symlink"
	DriftMissingSymlink   DriftKind = "missing_symlink"
	DriftMissingMarker    DriftKind = "missing_marker"
	DriftDeclaredNotLocal DriftKind = "declared_not_local"
	DriftLocalNotDeclared DriftKind = "local_not_declared"
	DriftNewTargetAtRoot  DriftKind = "new_target_at_root"
)

// DriftFinding is one issue detected during satellite reconciliation.
type DriftFinding struct {
	Kind    DriftKind
	Path    string // satellite path, relative to root, forward-slash
	Target  Target // the target involved, when applicable
	Message string
}

// Repair runs reconciliation on the workspace's satellite state.
// In dry-run mode it only reports findings.
type RepairOptions struct {
	HeroDir string
	RootDir string
	Version string
	DryRun  bool
	Force   bool
}

// RepairResult summarizes a repair pass.
type RepairResult struct {
	Findings []DriftFinding
	Repaired []string // human-readable description of each repair action
}

// Repair walks satellites.local.json and reconciles each entry against
// reality and against subprojects.json. It does:
//
//  1. Drops manifest entries whose folder no longer exists.
//  2. Repairs broken or missing symlinks per target.
//  3. Re-writes the satellite marker if missing.
//  4. Surfaces declared-but-not-materialized subprojects (DryRun: report;
//     non-DryRun: caller decides — Repair itself does not create new
//     satellites without explicit input).
//  5. Surfaces local-but-not-declared satellites (informational).
//  6. Surfaces new harness targets installed at root since each
//     satellite was last touched.
func Repair(opts RepairOptions) (*RepairResult, error) {
	result := &RepairResult{}

	local, err := LoadSatellitesLocal(opts.HeroDir)
	if err != nil {
		return nil, err
	}
	subs, err := LoadSubprojects(opts.HeroDir)
	if err != nil {
		return nil, err
	}
	rootTargets := DetectInstalledTargets(opts.RootDir)
	rootTargetSet := make(map[Target]bool, len(rootTargets))
	for _, t := range rootTargets {
		rootTargetSet[t] = true
	}

	// 1. Walk existing local entries.
	keep := make([]SatelliteEntry, 0, len(local.Satellites))
	for _, e := range local.Satellites {
		satAbs := filepath.Join(opts.RootDir, filepath.FromSlash(e.Path))
		info, err := os.Stat(satAbs)
		if err != nil || !info.IsDir() {
			result.Findings = append(result.Findings, DriftFinding{
				Kind:    DriftMissingFolder,
				Path:    e.Path,
				Message: fmt.Sprintf("satellite folder missing on disk: %s", e.Path),
			})
			if !opts.DryRun {
				result.Repaired = append(result.Repaired, fmt.Sprintf("dropped missing satellite %s from manifest", e.Path))
			}
			continue
		}

		// 2. Marker present?
		markerPath := filepath.Join(satAbs, workspace.SatelliteMarker)
		if _, err := os.Stat(markerPath); err != nil {
			result.Findings = append(result.Findings, DriftFinding{
				Kind:    DriftMissingMarker,
				Path:    e.Path,
				Message: fmt.Sprintf("satellite marker missing: %s", markerPath),
			})
			if !opts.DryRun {
				scope := scopeForPath(subs, e.Path)
				if err := workspace.WriteMarker(satAbs, opts.RootDir, scope, opts.Version); err != nil {
					return result, fmt.Errorf("rewrite marker %s: %w", markerPath, err)
				}
				result.Repaired = append(result.Repaired, fmt.Sprintf("rewrote marker for %s", e.Path))
			}
		}

		// 3. Symlinks per target.
		entryTargets := stringsToTargets(e.Targets)
		for _, t := range entryTargets {
			layout := LayoutFor(t)
			if layout == nil {
				continue
			}
			rootBase := filepath.Join(opts.RootDir, layout.SubDir)
			satBase := filepath.Join(satAbs, layout.SubDir)
			for _, sub := range SymlinkedDirs {
				rootSub := filepath.Join(rootBase, sub)
				if info, err := os.Stat(rootSub); err != nil || !info.IsDir() {
					continue
				}
				satSub := filepath.Join(satBase, sub)
				broken := false
				missing := false
				if linkTarget, err := os.Readlink(satSub); err == nil {
					resolved := filepath.Clean(filepath.Join(filepath.Dir(satSub), linkTarget))
					if resolved != filepath.Clean(rootSub) {
						broken = true
					}
				} else if _, lerr := os.Lstat(satSub); lerr != nil {
					missing = true
				}
				if broken {
					result.Findings = append(result.Findings, DriftFinding{
						Kind:    DriftBrokenSymlink,
						Path:    e.Path,
						Target:  t,
						Message: fmt.Sprintf("symlink points elsewhere: %s", satSub),
					})
				}
				if missing {
					result.Findings = append(result.Findings, DriftFinding{
						Kind:    DriftMissingSymlink,
						Path:    e.Path,
						Target:  t,
						Message: fmt.Sprintf("symlink missing: %s", satSub),
					})
				}
				if (broken || missing) && !opts.DryRun {
					if err := os.MkdirAll(filepath.Dir(satSub), 0o755); err != nil {
						return result, err
					}
					rel, err := filepath.Rel(filepath.Dir(satSub), rootSub)
					if err != nil {
						return result, err
					}
					if err := writeSymlink(satSub, rel, true); err != nil {
						return result, fmt.Errorf("repair symlink %s: %w", satSub, err)
					}
					result.Repaired = append(result.Repaired, fmt.Sprintf("repaired symlink %s", satSub))
				}
			}
		}

		// 6. New target at root since this satellite was created?
		for _, t := range rootTargets {
			if !slices.Contains(entryTargets, t) {
				layout := LayoutFor(t)
				if layout == nil {
					continue
				}
				result.Findings = append(result.Findings, DriftFinding{
					Kind:    DriftNewTargetAtRoot,
					Path:    e.Path,
					Target:  t,
					Message: fmt.Sprintf("target %s installed at root but not in satellite %s", t, e.Path),
				})
			}
		}

		keep = append(keep, e)
	}

	// 4. Declared but not materialized.
	for _, sp := range subs.Subprojects {
		if local.Find(sp.Path) == nil {
			result.Findings = append(result.Findings, DriftFinding{
				Kind:    DriftDeclaredNotLocal,
				Path:    sp.Path,
				Message: fmt.Sprintf("declared subproject %s has no local satellite", sp.Path),
			})
		}
	}

	// 5. Local but not declared.
	for _, e := range keep {
		if !subs.IsDeclared(e.Path) {
			result.Findings = append(result.Findings, DriftFinding{
				Kind:    DriftLocalNotDeclared,
				Path:    e.Path,
				Message: fmt.Sprintf("local satellite %s is not declared in subprojects.json", e.Path),
			})
		}
	}

	if !opts.DryRun {
		local.Satellites = keep
		if err := SaveSatellitesLocal(opts.HeroDir, local); err != nil {
			return result, err
		}
	}

	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].Path != result.Findings[j].Path {
			return result.Findings[i].Path < result.Findings[j].Path
		}
		return result.Findings[i].Kind < result.Findings[j].Kind
	})
	return result, nil
}

// FormatFindings renders a repair result as a human-readable list.
func (r *RepairResult) FormatFindings() string {
	if r == nil || len(r.Findings) == 0 {
		return "no satellite drift detected"
	}
	var sb strings.Builder
	for _, f := range r.Findings {
		sb.WriteString(fmt.Sprintf("  [%s] %s\n", f.Kind, f.Message))
	}
	return sb.String()
}

// scopeForPath looks up the canonical scope from subprojects.json,
// falling back to the path itself.
func scopeForPath(m *SubprojectsManifest, relPath string) string {
	if m != nil {
		for _, sp := range m.Subprojects {
			if normalizeRelPath(sp.Path) == normalizeRelPath(relPath) {
				if sp.Scope != "" {
					return sp.Scope
				}
				return sp.Path
			}
		}
	}
	return normalizeRelPath(relPath)
}

func stringsToTargets(s []string) []Target {
	out := make([]Target, len(s))
	for i, t := range s {
		out[i] = Target(t)
	}
	return out
}

