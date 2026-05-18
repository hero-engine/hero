package data

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// entryFile is one knowledge markdown file located on disk, after the
// shared walk classifies it. Slug is the entry slug used to look up the
// entry from the URL; Kind is the top-level knowledge subdir (e.g.
// "context", "notes") regardless of whether the file is flat or sits in
// a slug-named subdirectory.
type entryFile struct {
	Path    string
	Kind    string
	Slug    string
	ModTime time.Time
}

// collectKnowledgeFiles walks `<root>` (typically `<heroDir>/knowledge`)
// and returns every entry-bearing markdown file in any of three shapes:
//
//	<kind>/<slug>.md                 (flat — legacy)
//	<kind>/<slug>/spec.md            (dir-style — slug = intermediate dir)
//	<kind>/<nested>/<slug>.md        (one level deeper)
//
// Depth is limited to two levels below the top-level kind directory.
// Directories whose names start with `.` or `_` are skipped at every
// level so cache/working folders don't accidentally surface.
//
// Dedup rule: if both `<kind>/<slug>.md` and `<kind>/<slug>/spec.md`
// exist, the flat file wins (kept first) — that's the legacy shape and
// remains the canonical resolution for the slug. Other collisions (e.g.
// same slug under different kinds) are NOT deduped here; callers decide.
func collectKnowledgeFiles(root string) []entryFile {
	subdirs, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var out []entryFile
	// Track (kind, slug) pairs we've already emitted from the flat pass
	// so the dir-style + nested passes don't double-list them.
	seen := map[string]bool{}
	key := func(kind, slug string) string { return kind + "\x00" + slug }

	// Pass 1: flat `<kind>/<slug>.md` (legacy shape, wins on collisions).
	for _, sd := range subdirs {
		if !sd.IsDir() || skippableDir(sd.Name()) {
			continue
		}
		kindDir := filepath.Join(root, sd.Name())
		files, _ := os.ReadDir(kindDir)
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			slug := strings.TrimSuffix(f.Name(), ".md")
			full := filepath.Join(kindDir, f.Name())
			st, err := os.Stat(full)
			if err != nil {
				continue
			}
			out = append(out, entryFile{
				Path: full, Kind: sd.Name(), Slug: slug, ModTime: st.ModTime(),
			})
			seen[key(sd.Name(), slug)] = true
		}
	}

	// Pass 2: dir-style `<kind>/<slug>/spec.md` AND
	// nested `<kind>/<nested>/<slug>.md` (one level deeper).
	for _, sd := range subdirs {
		if !sd.IsDir() || skippableDir(sd.Name()) {
			continue
		}
		kindDir := filepath.Join(root, sd.Name())
		inner, _ := os.ReadDir(kindDir)
		for _, child := range inner {
			if !child.IsDir() || skippableDir(child.Name()) {
				continue
			}
			childDir := filepath.Join(kindDir, child.Name())

			// Dir-style: `<kind>/<child>/spec.md` — slug is the dir name.
			specPath := filepath.Join(childDir, "spec.md")
			if st, err := os.Stat(specPath); err == nil && !st.IsDir() {
				slug := child.Name()
				if !seen[key(sd.Name(), slug)] {
					out = append(out, entryFile{
						Path: specPath, Kind: sd.Name(), Slug: slug, ModTime: st.ModTime(),
					})
					seen[key(sd.Name(), slug)] = true
				}
			}

			// Nested: `<kind>/<child>/<slug>.md` — slug is the filename.
			// (Depth-2 limit: don't recurse further.)
			deeper, _ := os.ReadDir(childDir)
			for _, f := range deeper {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
					continue
				}
				if f.Name() == "spec.md" {
					// Already handled as dir-style above.
					continue
				}
				slug := strings.TrimSuffix(f.Name(), ".md")
				full := filepath.Join(childDir, f.Name())
				st, err := os.Stat(full)
				if err != nil {
					continue
				}
				if seen[key(sd.Name(), slug)] {
					continue
				}
				out = append(out, entryFile{
					Path: full, Kind: sd.Name(), Slug: slug, ModTime: st.ModTime(),
				})
				seen[key(sd.Name(), slug)] = true
			}
		}
	}

	return out
}

// skippableDir reports whether a directory name should be skipped during
// the knowledge walk (dotfiles, underscore prefix).
func skippableDir(name string) bool {
	if name == "" {
		return true
	}
	return name[0] == '.' || name[0] == '_'
}
