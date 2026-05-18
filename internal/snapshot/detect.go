package snapshot

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CandidateSurface is the output of pure-function inference. It
// carries enough rationale that `hero snapshot detect --explain` can
// show the user exactly which signals fired.
type CandidateSurface struct {
	ID         string
	Name       string
	Paths      []string
	Signals    []string
	Confidence float64
}

// RepoSnapshot is the minimal repo description detect.go consumes.
// Built by ScanRepo or hand-constructed in tests.
type RepoSnapshot struct {
	Root     string   // absolute repo root
	Dirs     []string // top-level entries that are directories, relative to root
	Manifests []string // paths to package manifests found (relative to root)
	HeroPaths []string // paths declared in hero.json (relative to root); optional
}

// ScanRepo walks the repo root non-recursively at depth 1, plus one
// level into a small allowlist of monorepo containers (web/, apps/,
// packages/, crates/, domains/) so multi-surface layouts are
// discoverable without a full walk.
func ScanRepo(root string) (RepoSnapshot, error) {
	rs := RepoSnapshot{Root: root}
	entries, err := os.ReadDir(root)
	if err != nil {
		return rs, err
	}
	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			rs.Dirs = append(rs.Dirs, name)
		}
	}
	sort.Strings(rs.Dirs)

	// Look for package manifests at root.
	for _, mf := range []string{"go.mod", "Cargo.toml", "package.json", "pyproject.toml"} {
		if _, err := os.Stat(filepath.Join(root, mf)); err == nil {
			rs.Manifests = append(rs.Manifests, mf)
		}
	}

	// Walk one level into monorepo containers for nested manifests.
	for _, container := range []string{"web", "apps", "packages", "crates", "domains"} {
		dir := filepath.Join(root, container)
		nested, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range nested {
			if !e.IsDir() {
				continue
			}
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			sub := filepath.Join(container, e.Name())
			rs.Dirs = append(rs.Dirs, sub)
			// Detect manifests in the nested dir.
			for _, mf := range []string{"go.mod", "Cargo.toml", "package.json", "mkdocs.yml", "wrangler.toml", "pyproject.toml"} {
				if _, err := os.Stat(filepath.Join(dir, e.Name(), mf)); err == nil {
					rs.Manifests = append(rs.Manifests, filepath.Join(sub, mf))
				}
			}
		}
	}

	sort.Strings(rs.Dirs)
	sort.Strings(rs.Manifests)
	return rs, nil
}

// Detect runs the inference rules and returns candidate surfaces.
// Pure-function: given the same RepoSnapshot, the output is
// deterministic. Confidence is heuristic — 1.0 when an explicit
// manifest anchors the surface, 0.7 when the dir naming hint fires
// alone, 0.5 for inferred-from-paths-only.
func Detect(rs RepoSnapshot) []CandidateSurface {
	var candidates []CandidateSurface
	dirs := stringSet(rs.Dirs)

	// core: a top-level Go module (go.mod present) with internal/ or
	// cmd/ adjacent is the canonical "main CLI + engine" surface.
	if hasManifest(rs.Manifests, "go.mod") && (dirs["internal"] || dirs["cmd"]) {
		paths := []string{}
		signals := []string{"go.mod at repo root"}
		if dirs["cmd"] {
			paths = append(paths, "cmd/")
			signals = append(signals, "cmd/ directory")
		}
		if dirs["internal"] {
			paths = append(paths, "internal/")
			signals = append(signals, "internal/ directory")
		}
		candidates = append(candidates, CandidateSurface{
			ID:         "core",
			Name:       "Core (CLI + engine)",
			Paths:      paths,
			Signals:    signals,
			Confidence: 1.0,
		})
	}

	// serve: an internal/serve/ subdir is the daemon + web companion.
	if hasDir(rs, "internal/serve") || dirs["serve"] {
		path := "internal/serve/"
		if !hasDir(rs, "internal/serve") {
			path = "serve/"
		}
		candidates = append(candidates, CandidateSurface{
			ID:         "serve",
			Name:       "Hero Serve",
			Paths:      []string{path},
			Signals:    []string{"internal/serve naming hint"},
			Confidence: 0.9,
		})
	}

	// mcp: internal/serve/mcp* — bundled with serve in hero today,
	// but the MCP server is a coherent unit of its own. Detect when
	// there's an mcp*.go set under serve, OR a dedicated mcp/ dir.
	if hasDir(rs, "internal/serve") && hasMCPNaming(rs.Root) || dirs["mcp"] {
		path := "internal/serve/mcp*.go"
		if dirs["mcp"] {
			path = "mcp/"
		}
		candidates = append(candidates, CandidateSurface{
			ID:         "mcp",
			Name:       "MCP server",
			Paths:      []string{path},
			Signals:    []string{"mcp naming under serve"},
			Confidence: 0.8,
		})
	}

	// web/<surface>: every direct child of web/ with a manifest is its
	// own surface. mkdocs.yml -> "docs"; wrangler.toml -> the dir name;
	// package.json -> the dir name.
	for _, mf := range rs.Manifests {
		if !strings.HasPrefix(mf, "web/") {
			continue
		}
		parts := strings.SplitN(mf, "/", 3)
		if len(parts) < 3 {
			continue
		}
		sub := parts[1]
		manifestBase := parts[2]
		id := sub
		name := titleCase(sub)
		signals := []string{"web/" + sub + "/ + " + manifestBase}
		conf := 0.95
		if manifestBase == "mkdocs.yml" {
			signals = append(signals, "mkdocs site")
		} else if manifestBase == "wrangler.toml" {
			signals = append(signals, "Cloudflare Worker")
		}
		candidates = append(candidates, CandidateSurface{
			ID:         id,
			Name:       name,
			Paths:      []string{"web/" + sub + "/"},
			Signals:    signals,
			Confidence: conf,
		})
	}

	// apps/<name>, packages/<name>, crates/<name>, domains/<pack>: each
	// becomes a surface keyed on the container plus name. For domains
	// we use "domains/<pack>" verbatim; for apps/packages/crates we
	// strip the container so "apps/api" -> "api".
	for _, container := range []string{"apps", "packages", "crates", "domains"} {
		for _, d := range rs.Dirs {
			if !strings.HasPrefix(d, container+"/") {
				continue
			}
			sub := strings.TrimPrefix(d, container+"/")
			if sub == "" || strings.Contains(sub, "/") {
				continue
			}
			id := container + "/" + sub
			if container != "domains" {
				id = sub
			}
			signals := []string{container + "/" + sub + " directory"}
			conf := 0.85
			candidates = append(candidates, CandidateSurface{
				ID:         id,
				Name:       titleCase(sub),
				Paths:      []string{d + "/"},
				Signals:    signals,
				Confidence: conf,
			})
		}
	}

	// docs: a top-level docs/ directory (not under web/) is a candidate
	// surface only when web/docs is absent. Avoid double-listing.
	if dirs["docs"] && !containsCandidate(candidates, "docs") {
		candidates = append(candidates, CandidateSurface{
			ID:         "docs",
			Name:       "Documentation",
			Paths:      []string{"docs/"},
			Signals:    []string{"docs/ directory"},
			Confidence: 0.7,
		})
	}

	// hero.json declared paths win over inference where present.
	// Recorded as a high-confidence signal additive to whatever Detect
	// already inferred. (Full hero.json parsing happens in surfaces.go.)

	// Deduplicate by id, preferring the higher-confidence candidate.
	candidates = dedupeByID(candidates)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ID < candidates[j].ID
	})
	return candidates
}

func hasDir(rs RepoSnapshot, p string) bool {
	for _, d := range rs.Dirs {
		if d == p {
			return true
		}
	}
	// Top-level dirs may be listed by their leaf; check filesystem too.
	if _, err := os.Stat(filepath.Join(rs.Root, p)); err == nil {
		return true
	}
	return false
}

func hasManifest(manifests []string, name string) bool {
	for _, m := range manifests {
		if m == name {
			return true
		}
	}
	return false
}

func hasMCPNaming(root string) bool {
	entries, err := os.ReadDir(filepath.Join(root, "internal", "serve"))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "mcp") {
			return true
		}
	}
	return false
}

func stringSet(in []string) map[string]bool {
	out := make(map[string]bool, len(in))
	for _, s := range in {
		out[s] = true
	}
	return out
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func containsCandidate(cs []CandidateSurface, id string) bool {
	for _, c := range cs {
		if c.ID == id {
			return true
		}
	}
	return false
}

func dedupeByID(cs []CandidateSurface) []CandidateSurface {
	best := map[string]CandidateSurface{}
	for _, c := range cs {
		existing, ok := best[c.ID]
		if !ok || c.Confidence > existing.Confidence {
			best[c.ID] = c
		}
	}
	out := make([]CandidateSurface, 0, len(best))
	for _, c := range best {
		out = append(out, c)
	}
	return out
}
