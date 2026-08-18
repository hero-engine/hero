package install

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/workspace"
)

// CandidateReason describes why a folder was nominated as a subproject
// candidate. Multiple reasons may apply.
type CandidateReason string

const (
	ReasonHeroDir   CandidateReason = "existing-hero-workspace"
	ReasonGoMod     CandidateReason = "go.mod"
	ReasonNodePkg   CandidateReason = "package.json"
	ReasonCargoToml CandidateReason = "Cargo.toml"
	ReasonPyproject CandidateReason = "pyproject.toml"
	ReasonGradle    CandidateReason = "build.gradle"
	ReasonMaven     CandidateReason = "pom.xml"
)

// Candidate is a folder the detector thinks may be a subproject.
type Candidate struct {
	// Path is forward-slash, relative to workspace root.
	Path string
	// AbsPath is the absolute path.
	AbsPath string
	// Reasons is the set of signals that nominated this folder.
	Reasons []CandidateReason
	// HasNestedHero is true when this folder contains a legacy .hero/
	// directory — strong signal it was a standalone hero workspace.
	HasNestedHero bool
}

// DetectCandidates walks the workspace rooted at rootDir and returns
// candidate subproject folders. It excludes any folder already in the
// excluded list, any folder already declared, and a fixed set of common
// noise directories (node_modules, vendor, etc.).
//
// The walk depth is capped — there is no business in flagging a deeply
// nested package.json as a top-level subproject. Default cap is 4.
func DetectCandidates(rootDir string, manifest *SubprojectsManifest, depthCap int) ([]Candidate, error) {
	if depthCap <= 0 {
		depthCap = 4
	}

	noise := map[string]bool{
		"node_modules":  true,
		"vendor":        true,
		".git":          true,
		".hero":         true,
		"dist":          true,
		"build":         true,
		"out":           true,
		"output":        true,
		".output":       true,
		"target":        true,
		".cache":        true,
		"third_party":   true,
		"__generated__": true,
		"__pycache__":   true,
		"coverage":      true,
		".next":         true,
		".nuxt":         true,
		".svelte-kit":   true,
		".angular":      true,
		"bin":           true,
		"obj":           true,
		".idea":         true,
		".vscode":       true,
		"tmp":           true,
		"temp":          true,
		"logs":          true,
		"htmlcov":       true,
		".venv":         true,
		"venv":          true,
		"env":           true,
		"jspm_packages": true,
		".pnpm":         true,
		".yarn":         true,
		".parcel-cache": true,
		".nyc_output":   true,
		".agents":       true,
		".claude":       true,
		".codex":        true,
		".grok":         true,
		".opencode":     true,
		".cursor":       true,
		".github":       true,
	}

	candidates := make(map[string]*Candidate)

	addReason := func(rel, abs string, reason CandidateReason, hasHero bool) {
		c := candidates[rel]
		if c == nil {
			c = &Candidate{Path: rel, AbsPath: abs, HasNestedHero: hasHero}
			candidates[rel] = c
		}
		if hasHero {
			c.HasNestedHero = true
		}
		for _, r := range c.Reasons {
			if r == reason {
				return
			}
		}
		c.Reasons = append(c.Reasons, reason)
	}

	signals := []struct {
		filename string
		reason   CandidateReason
	}{
		{"go.mod", ReasonGoMod},
		{"package.json", ReasonNodePkg},
		{"Cargo.toml", ReasonCargoToml},
		{"pyproject.toml", ReasonPyproject},
		{"build.gradle", ReasonGradle},
		{"build.gradle.kts", ReasonGradle},
		{"pom.xml", ReasonMaven},
	}

	walkErr := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if !d.IsDir() {
			return nil
		}
		// Always skip the root itself.
		if path == rootDir {
			return nil
		}
		base := filepath.Base(path)
		if noise[base] || isVendorShaped(base) {
			return filepath.SkipDir
		}
		// Hidden dirs aren't candidates.
		if strings.HasPrefix(base, ".") {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		// Depth cap.
		if strings.Count(rel, "/")+1 > depthCap {
			return filepath.SkipDir
		}

		// Skip declared and excluded folders entirely.
		if manifest.IsDeclared(rel) || manifest.IsExcluded(rel) {
			return filepath.SkipDir
		}

		hasHero := false
		if info, err := os.Stat(filepath.Join(path, workspace.HeroDir)); err == nil && info.IsDir() {
			hasHero = true
			addReason(rel, path, ReasonHeroDir, true)
		}
		for _, sig := range signals {
			if _, err := os.Stat(filepath.Join(path, sig.filename)); err == nil {
				addReason(rel, path, sig.reason, hasHero)
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	out := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		// Folders with a nested .hero/ float to the top — strongest signal.
		if out[i].HasNestedHero != out[j].HasNestedHero {
			return out[i].HasNestedHero
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// FindNestedHeroDirs returns relative paths under rootDir that contain
// their own .hero/ directory. Used by the migration prompt to flag
// "you have leftover standalone workspaces" before any other prompts.
func FindNestedHeroDirs(rootDir string) []string {
	var nested []string
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path == rootDir {
			return nil
		}
		base := filepath.Base(path)
		if base == workspace.HeroDir {
			parent := filepath.Dir(path)
			rel, _ := filepath.Rel(rootDir, parent)
			rel = filepath.ToSlash(rel)
			if rel != "" && rel != "." {
				nested = append(nested, rel)
			}
			return filepath.SkipDir
		}
		if base == "node_modules" || base == "vendor" || strings.HasPrefix(base, ".") {
			return filepath.SkipDir
		}
		return nil
	})
	sort.Strings(nested)
	return nested
}

// ReasonStrings renders a candidate's reasons as a comma-separated list.
func (c Candidate) ReasonStrings() string {
	parts := make([]string, len(c.Reasons))
	for i, r := range c.Reasons {
		parts[i] = string(r)
	}
	return strings.Join(parts, ", ")
}

// isVendorShaped reports whether a directory's base name follows a
// common "this is a vendor tree" naming pattern. Used by DetectCandidates
// to skip vendored subprojects that the existing exact-match noise list
// (which only catches `vendor` itself) misses. Conservative on purpose:
// matches `*-vendor`, `vendor-*`, and `vendored` only. Anything more
// adventurous (`external`, `deps`, `third-party-*`) risks excluding
// legitimate first-party folders, so the user handles those via the
// walkthrough's `X` (exclude parent) shortcut.
func isVendorShaped(base string) bool {
	if base == "vendored" {
		return true
	}
	if strings.HasSuffix(base, "-vendor") {
		return true
	}
	if strings.HasPrefix(base, "vendor-") {
		return true
	}
	return false
}
