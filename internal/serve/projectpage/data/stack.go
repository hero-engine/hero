package data

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// StackInputs is the per-request input bundle for the Stack section.
type StackInputs struct {
	ProjectRoot string
	HeroDir     string
}

// Stack is what the partial renders.
type Stack struct {
	// Languages is the detected language list (e.g. ["Go", "TypeScript"]).
	Languages []string

	// ActiveConventions counts conventions in `.hero/knowledge/
	// conventions/` (the count drives the section header).
	ActiveConventions int

	// RecentConventions is the three most-recently-modified conventions.
	// Each links to its source spec via .hero/-relative path.
	RecentConventions []ConventionLink

	// Detected is true when at least one language was identified.
	Detected bool
}

// ConventionLink is one row in the "recent conventions" mini-list.
type ConventionLink struct {
	Slug       string
	Title      string
	Path       string // absolute path on disk
	ModifiedAt time.Time
}

// LoadStack reads cheap stack markers from disk (go.mod, package.json,
// Cargo.toml, pyproject.toml, …) and discovers conventions under the
// hero dir. It does NOT walk the full source tree — for Phase 1 we
// trade thoroughness for sub-millisecond render time.
//
// If a future phase wires in a persisted output from the `scan` skill,
// this loader can read that file first and fall back to marker probing.
func LoadStack(in StackInputs) Stack {
	out := Stack{}
	if in.ProjectRoot != "" {
		out.Languages = detectLanguageMarkers(in.ProjectRoot)
		out.Detected = len(out.Languages) > 0
	}
	if in.HeroDir != "" {
		specs, err := spec.Discover(in.HeroDir)
		if err == nil {
			conventions := make([]ConventionLink, 0)
			for _, s := range specs {
				if s == nil || s.Type != spec.TypeConvention {
					continue
				}
				conventions = append(conventions, ConventionLink{
					Slug:       s.Slug,
					Title:      s.Title,
					Path:       s.Path,
					ModifiedAt: s.ModifiedAt,
				})
			}
			out.ActiveConventions = len(conventions)
			sort.SliceStable(conventions, func(i, j int) bool {
				return conventions[i].ModifiedAt.After(conventions[j].ModifiedAt)
			})
			if len(conventions) > 3 {
				conventions = conventions[:3]
			}
			out.RecentConventions = conventions
		}
	}
	return out
}

// detectLanguageMarkers checks for canonical project-root marker files
// and returns the matching language list. Order: most-likely-primary
// first when ties exist (Go before Node, etc.).
func detectLanguageMarkers(projectRoot string) []string {
	type marker struct {
		File     string
		Language string
	}
	markers := []marker{
		{"go.mod", "Go"},
		{"package.json", "JavaScript / TypeScript"},
		{"Cargo.toml", "Rust"},
		{"pyproject.toml", "Python"},
		{"setup.py", "Python"},
		{"requirements.txt", "Python"},
		{"pom.xml", "Java"},
		{"build.gradle", "Java / Groovy"},
		{"build.gradle.kts", "Kotlin"},
		{"Gemfile", "Ruby"},
		{"composer.json", "PHP"},
		{"mix.exs", "Elixir"},
	}
	seen := map[string]bool{}
	var langs []string
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(projectRoot, m.File)); err == nil {
			if !seen[m.Language] {
				seen[m.Language] = true
				langs = append(langs, m.Language)
			}
		}
	}
	return langs
}
