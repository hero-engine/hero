package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/install"
)

// TestMarkdownInvocationsResolveAgainstRootCmd is the drift gate. It
// walks every markdown surface this repo ships or renders, extracts
// `hero <command>` invocations, and asserts each one resolves against
// rootCmd. If a command is renamed, removed, or never registered, the
// stale reference fails here rather than rotting in production output.
//
// Surfaces scanned (all project-relative):
//
//   - commands/*.md                  (slash command definitions)
//   - skills/**/*.md                 (skill content)
//   - agents/*.md                    (agent role definitions)
//   - web/docs/src/**/*.md           (public docs site sources)
//   - README.md, AGENTS.md,
//     GETTING-STARTED.md             (top-level docs)
//   - rendered AGENTS.md template    (in-memory; the bytes installed
//     into downstream projects via internal/install)
//
// .hero/specs/ and .hero/planning/ are excluded by default — they
// intentionally describe broken/phantom invocations as part of bug
// reports. Use <!-- drift-test:ignore --> on individual lines where
// a doc must show a deliberately-broken invocation.
func TestMarkdownInvocationsResolveAgainstRootCmd(t *testing.T) {
	root := findProjectRoot()

	type surface struct {
		label string
		// At least one of dirs / files / rendered must populate. Each
		// dir is walked recursively (the test ignores non-.md files
		// itself; no need for caller-side filtering).
		dirs     []string
		files    []string
		rendered map[string][]byte
		// requireAny means the surface must produce at least one
		// invocation; an empty result indicates a broken glob and is
		// itself a failure.
		requireAny bool
	}

	surfaces := []surface{
		{
			label:      "commands",
			dirs:       []string{"commands"},
			requireAny: true,
		},
		{
			label:      "skills",
			dirs:       []string{"skills"},
			requireAny: true,
		},
		{
			label:      "agents",
			dirs:       []string{"agents"},
			requireAny: false, // agents/ may not contain CLI invocations
		},
		{
			label:      "web-docs",
			dirs:       []string{"web/docs/src"},
			requireAny: false,
		},
		{
			label:      "top-level",
			files:      []string{"README.md", "AGENTS.md", "GETTING-STARTED.md"},
			requireAny: true,
		},
		{
			label: "rendered-agents-md",
			rendered: map[string][]byte{
				"<rendered:internal/install/agents_md.go>": install.RenderAgentsMdBodyForDriftTest(),
			},
			requireAny: true,
		},
	}

	totalChecked := 0
	totalFailed := 0
	for _, s := range surfaces {
		var invs []Invocation
		for _, d := range s.dirs {
			abs := filepath.Join(root, d)
			invs = append(invs, collectMarkdownInvocations(t, root, abs)...)
		}
		for _, f := range s.files {
			abs := filepath.Join(root, f)
			data, err := os.ReadFile(abs)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				t.Errorf("[%s] read %s: %v", s.label, f, err)
				continue
			}
			invs = append(invs, ExtractInvocations(f, data)...)
		}
		for path, content := range s.rendered {
			invs = append(invs, ExtractInvocations(path, content)...)
		}

		if s.requireAny && len(invs) == 0 {
			t.Errorf("[%s] surface produced 0 invocations — globs may be broken or the surface is empty", s.label)
			continue
		}

		for _, inv := range invs {
			totalChecked++
			if err := ValidateInvocation(rootCmd, inv); err != nil {
				totalFailed++
				t.Errorf("[%s] %s:%d  `%s`  →  %v", s.label, inv.File, inv.Line, inv.Raw, err)
			}
		}
	}

	t.Logf("checked %d markdown invocations across all surfaces, %d failed", totalChecked, totalFailed)
}

// collectMarkdownInvocations walks dir recursively, reads every .md file,
// skips excluded paths (.hero/specs, .hero/planning), and returns all
// extracted invocations. File paths in results are project-relative.
func collectMarkdownInvocations(t *testing.T, projectRoot, dir string) []Invocation {
	t.Helper()
	var out []Invocation
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(projectRoot, path)
		if relErr != nil {
			rel = path
		}
		relSlash := filepath.ToSlash(rel)
		if isExcludedDir(relSlash) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Errorf("read %s: %v", relSlash, readErr)
			return nil
		}
		out = append(out, ExtractInvocations(relSlash, data)...)
		return nil
	})
	if err != nil {
		t.Errorf("walk %s: %v", dir, err)
	}
	return out
}
