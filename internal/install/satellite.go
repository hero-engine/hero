package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/workspace"
)

// SymlinkedDirs are the per-target subdirectories materialized as
// symlinks in a satellite. settings.json and settings.local.json are
// deliberately not in this list — they may diverge per subproject.
var SymlinkedDirs = []string{"agents", "commands", "skills"}

// TargetLayout describes how a single harness target lays out its files
// inside a project. It is the satellite-side mirror of the per-target
// resolver functions in install.go.
type TargetLayout struct {
	// Target is the harness identifier.
	Target Target
	// SubDir is the directory under the project root that holds this
	// target's content (e.g. ".claude", ".codex"). Empty if the target
	// stores content elsewhere — the satellite layer skips those.
	SubDir string
	// MarkerFile is the per-target CLAUDE.md-style marker to write in a
	// satellite folder, telling the model where the workspace lives.
	// Empty if the target has no top-level marker.
	MarkerFile string
}

// targetLayouts is the registry of per-target satellite layouts. It
// covers exactly the targets that store content under the project tree
// in a way amenable to subdirectory symlinks. (Globals like
// ~/.opencode/, ~/.cursor/rules/ are out of scope for satellites.)
//
// Codex, OpenCode, and Generic all converge on AGENTS.md at the
// project root for harness instructions — so they all use AGENTS.md
// as the per-folder satellite marker too. When multiple of those
// targets are installed at root, Materialize writes AGENTS.md exactly
// once and renders all sharing targets in the marker body.
//
// Copilot's MarkerFile is intentionally empty: Copilot's instruction
// file is .github/copilot-instructions.md at the repo root, and
// Copilot's discovery is org/repo-scoped rather than cwd-relative —
// so a per-folder marker has no read path. Symlinks under
// .github/copilot/ alone are enough.
//
// Cursor's MarkerFile is also empty: Cursor reads .cursor/rules/
// walking up from cwd, and there's no convention for a per-folder
// instruction file the model picks up automatically. Symlinks alone
// suffice for cwd-aware rule discovery.
var targetLayouts = []TargetLayout{
	{Target: TargetClaude, SubDir: ".claude", MarkerFile: "CLAUDE.md"},
	{Target: TargetCodex, SubDir: ".codex", MarkerFile: "AGENTS.md"},
	{Target: TargetOpenCode, SubDir: ".opencode", MarkerFile: "AGENTS.md"},
	{Target: TargetCursor, SubDir: filepath.Join(".cursor", "rules"), MarkerFile: ""},
	{Target: TargetCopilot, SubDir: filepath.Join(".github", "copilot"), MarkerFile: ""},
	{Target: TargetGeneric, SubDir: ".ai", MarkerFile: "AGENTS.md"},
}

// LayoutFor returns the TargetLayout for the given target, or nil if the
// target is not satellite-able.
func LayoutFor(t Target) *TargetLayout {
	for i := range targetLayouts {
		if targetLayouts[i].Target == t {
			return &targetLayouts[i]
		}
	}
	return nil
}

// DetectInstalledTargets walks the workspace root and returns the list
// of harness targets that have content directories present. This is how
// satellites know which targets to mirror.
func DetectInstalledTargets(rootDir string) []Target {
	var found []Target
	for _, layout := range targetLayouts {
		dir := filepath.Join(rootDir, layout.SubDir)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			// Require at least one of the symlinked subdirs to exist —
			// an empty .claude/ alone is not a real install.
			for _, sub := range SymlinkedDirs {
				subPath := filepath.Join(dir, sub)
				if subInfo, subErr := os.Stat(subPath); subErr == nil && subInfo.IsDir() {
					found = append(found, layout.Target)
					break
				}
			}
		}
	}
	return found
}

// SatelliteOptions controls a satellite materialization or repair.
type SatelliteOptions struct {
	// RootDir is the absolute path of the workspace root.
	RootDir string
	// SatelliteDir is the absolute path of the satellite folder.
	SatelliteDir string
	// Scope is the canonical scope identifier for this satellite.
	Scope string
	// Targets is the harness targets to materialize. If empty, defaults
	// to all targets currently installed at root (DetectInstalledTargets).
	Targets []Target
	// Version is the hero binary version for marker stamping.
	Version string
	// Force replaces existing symlinks/markers without prompting.
	Force bool
	// DryRun reports what would be done without writing anything.
	DryRun bool
}

// SatelliteResult describes the outcome of a materialization.
type SatelliteResult struct {
	Created  []string // paths that were created/updated
	Skipped  []string // paths that were left alone
	Degraded bool     // true if symlink fallback engaged (marker-only)
	Targets  []Target // targets actually materialized
}

// Materialize creates a satellite folder: subdirectory symlinks per
// target, a per-target marker file, and the .hero-satellite JSON marker.
//
// It does NOT create .hero/ in the satellite — there is exactly one
// workspace per repo. It does NOT create .mcp.json — harnesses walk up
// from cwd to find the root's.
func Materialize(opts SatelliteOptions) (*SatelliteResult, error) {
	if opts.RootDir == "" || opts.SatelliteDir == "" {
		return nil, fmt.Errorf("RootDir and SatelliteDir are required")
	}
	rootAbs, err := filepath.Abs(opts.RootDir)
	if err != nil {
		return nil, err
	}
	satAbs, err := filepath.Abs(opts.SatelliteDir)
	if err != nil {
		return nil, err
	}

	// Refuse to materialize a satellite at the workspace root itself.
	if rootAbs == satAbs {
		return nil, fmt.Errorf("satellite path is the workspace root: %s", rootAbs)
	}
	// Ensure the satellite is inside the root tree.
	rel, err := filepath.Rel(rootAbs, satAbs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("satellite path %s is not under workspace root %s", satAbs, rootAbs)
	}

	targets := opts.Targets
	if len(targets) == 0 {
		targets = DetectInstalledTargets(rootAbs)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no harness targets installed at root %s; install one first", rootAbs)
	}

	result := &SatelliteResult{}
	symlinkSupported := SymlinksSupported(satAbs)

	// First pass: materialize symlinks per target; group targets by their
	// shared marker file so we can write each marker exactly once below.
	markerGroups := map[string][]Target{} // markerFile -> targets sharing it (in iteration order)
	for _, t := range targets {
		layout := LayoutFor(t)
		if layout == nil || layout.SubDir == "" {
			result.Skipped = append(result.Skipped, fmt.Sprintf("target %s: no satellite layout", t))
			continue
		}

		targetSatBase := filepath.Join(satAbs, layout.SubDir)
		targetRootBase := filepath.Join(rootAbs, layout.SubDir)

		// If the root doesn't have this target installed, skip.
		if info, err := os.Stat(targetRootBase); err != nil || !info.IsDir() {
			result.Skipped = append(result.Skipped, fmt.Sprintf("target %s: not installed at root", t))
			continue
		}

		if !opts.DryRun {
			if err := os.MkdirAll(targetSatBase, 0o755); err != nil {
				return result, fmt.Errorf("mkdir %s: %w", targetSatBase, err)
			}
		}

		if symlinkSupported {
			for _, sub := range SymlinkedDirs {
				rootSub := filepath.Join(targetRootBase, sub)
				if info, err := os.Stat(rootSub); err != nil || !info.IsDir() {
					continue
				}
				satSub := filepath.Join(targetSatBase, sub)
				rel, err := filepath.Rel(filepath.Dir(satSub), rootSub)
				if err != nil {
					return result, fmt.Errorf("compute relative symlink target: %w", err)
				}
				if opts.DryRun {
					result.Created = append(result.Created, fmt.Sprintf("symlink %s -> %s", satSub, rel))
					continue
				}
				if err := writeSymlink(satSub, rel, opts.Force); err != nil {
					return result, fmt.Errorf("symlink %s: %w", satSub, err)
				}
				result.Created = append(result.Created, satSub)
			}
		} else {
			// Marker-only fallback: do not create symlinks, do not copy.
			result.Degraded = true
		}

		if layout.MarkerFile != "" {
			markerGroups[layout.MarkerFile] = append(markerGroups[layout.MarkerFile], layout.Target)
		}
		result.Targets = append(result.Targets, t)
	}

	// Second pass: write each marker file exactly once, listing all
	// targets that share it. Iteration order over markerGroups is not
	// stable — sort marker filenames so the output is deterministic.
	markerFiles := make([]string, 0, len(markerGroups))
	for mf := range markerGroups {
		markerFiles = append(markerFiles, mf)
	}
	sort.Strings(markerFiles)
	for _, markerFile := range markerFiles {
		sharingTargets := markerGroups[markerFile]
		markerPath := filepath.Join(satAbs, markerFile)
		content := perTargetMarker(rootAbs, satAbs, opts.Scope, sharingTargets, symlinkSupported)
		if opts.DryRun {
			result.Created = append(result.Created, fmt.Sprintf("marker %s", markerPath))
			continue
		}
		if err := writeMarkerFile(markerPath, content, opts.Force); err != nil {
			return result, fmt.Errorf("write %s: %w", markerPath, err)
		}
		result.Created = append(result.Created, markerPath)
	}

	// Always write the .hero-satellite JSON marker — it is the file
	// hero CLI reads to know it is in a satellite.
	if !opts.DryRun {
		if err := workspace.WriteMarker(satAbs, rootAbs, opts.Scope, opts.Version); err != nil {
			return result, fmt.Errorf("write satellite marker: %w", err)
		}
	}
	result.Created = append(result.Created, filepath.Join(satAbs, workspace.SatelliteMarker))

	// Sort outputs for stable comparison.
	sort.Strings(result.Created)
	sort.Strings(result.Skipped)
	return result, nil
}

// RemoveSatellite tears down a satellite folder: symlinks, markers, and
// the .hero-satellite file. Per-target marker files (CLAUDE.md etc.)
// are removed only if they are still hero-managed (i.e. they look like
// a small generated marker, not a hand-edited file).
func RemoveSatellite(satAbs string, targets []Target) error {
	if len(targets) == 0 {
		// Best-effort: try every known layout.
		for _, layout := range targetLayouts {
			targets = append(targets, layout.Target)
		}
	}
	for _, t := range targets {
		layout := LayoutFor(t)
		if layout == nil || layout.SubDir == "" {
			continue
		}
		base := filepath.Join(satAbs, layout.SubDir)
		// Remove only the symlinked subdirs and per-target marker.
		for _, sub := range SymlinkedDirs {
			path := filepath.Join(base, sub)
			if isSymlink(path) {
				_ = os.Remove(path)
			}
		}
		if layout.MarkerFile != "" {
			markerPath := filepath.Join(satAbs, layout.MarkerFile)
			if isHeroSatelliteMarkerFile(markerPath) {
				_ = os.Remove(markerPath)
			}
		}
		// If the target subdir is now empty, clean it up.
		_ = os.Remove(base)
		// Same for parent if the layout uses nested dirs (e.g. .github/copilot).
		if dir := filepath.Dir(base); dir != satAbs && dir != "." {
			_ = os.Remove(dir)
		}
	}
	return workspace.RemoveMarker(satAbs)
}

// writeSymlink creates a symlink at path pointing to target. If a
// symlink already exists at path with the same target, this is a no-op.
// If a different file/symlink exists, force=true replaces it.
func writeSymlink(path, target string, force bool) error {
	if existing, err := os.Readlink(path); err == nil {
		if existing == target {
			return nil
		}
		if !force {
			return fmt.Errorf("symlink exists with different target (%s); use --force", existing)
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	} else if _, err := os.Lstat(path); err == nil {
		// Non-symlink exists.
		if !force {
			return fmt.Errorf("non-symlink file/dir exists at %s; use --force", path)
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return os.Symlink(target, path)
}

// isSymlink reports whether the path is a symlink (broken or not).
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// SymlinksSupported reports whether the OS / filesystem at the given
// directory supports relative symlinks. It probes by attempting to
// create-and-remove a tiny symlink. The probe is not persistent.
func SymlinksSupported(dir string) bool {
	// Cheap path: POSIX systems virtually always support symlinks.
	if runtime.GOOS != "windows" {
		return true
	}
	tmpDir, err := os.MkdirTemp(dir, ".hero-symtest-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tmpDir)
	link := filepath.Join(tmpDir, "lnk")
	if err := os.Symlink("target", link); err != nil {
		return false
	}
	return true
}

// perTargetMarker returns the contents of a per-target marker file
// (e.g. CLAUDE.md or AGENTS.md inside a satellite). It tells the model
// that the real workspace lives at the resolved root and what scope is
// active.
//
// targets is the list of harness targets that share this marker file.
// When Codex and OpenCode and Generic all install at root, they share
// AGENTS.md; the marker body lists all three so the model isn't told
// it's a "codex satellite" when opencode is also reading the same file.
func perTargetMarker(rootAbs, satAbs, scope string, targets []Target, symlinks bool) string {
	rel, err := filepath.Rel(satAbs, rootAbs)
	if err != nil {
		rel = rootAbs
	}
	rel = filepath.ToSlash(rel)
	if scope == "" {
		scope = "(root)"
	}
	mode := "full satellite (symlinked agents/commands/skills)"
	if !symlinks {
		mode = "degraded satellite (marker only — open the workspace root for full Hero support)"
	}
	label := "Target"
	if len(targets) > 1 {
		label = "Targets"
	}
	names := make([]string, len(targets))
	for i, t := range targets {
		names[i] = string(t)
	}
	return fmt.Sprintf(`<!-- hero:satellite -->
# Hero satellite

This folder is a satellite of the Hero workspace at `+"`%s`"+`.

- **Scope:** `+"`%s`"+`
- **%s:** %s
- **Mode:** %s

Specs, knowledge, and events created from this folder land in the root
workspace, scoped to the identifier above. To work directly at the
workspace root, open `+"`%s`"+` in your editor or chat tool.
`, rel, scope, label, strings.Join(names, ", "), mode, rel)
}

// writeMarkerFile writes a small, generated per-target marker. If the
// file exists and is not hero-managed, refuses without --force.
func writeMarkerFile(path, content string, force bool) error {
	if existing, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(existing), "<!-- hero:satellite -->") {
			// Hero-managed — replace.
			return os.WriteFile(path, []byte(content), 0o644)
		}
		if !force {
			return fmt.Errorf("file exists and is not hero-managed: %s", path)
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// isHeroSatelliteMarkerFile reports whether the file looks like a
// satellite marker we wrote (so we can safely remove it on uninstall).
func isHeroSatelliteMarkerFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "<!-- hero:satellite -->")
}

// RecordSatellite updates satellites.local.json after a successful
// materialization (or repair). It is the bridge between Materialize
// and the local manifest.
func RecordSatellite(heroDir string, e SatelliteEntry) error {
	s, err := LoadSatellitesLocal(heroDir)
	if err != nil {
		return err
	}
	if e.InstalledAt.IsZero() {
		e.InstalledAt = time.Now().UTC()
	}
	s.Upsert(e)
	return SaveSatellitesLocal(heroDir, s)
}

// targetSliceToStrings converts a Target slice to []string for storage.
func targetSliceToStrings(ts []Target) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = string(t)
	}
	sort.Strings(out)
	return out
}
