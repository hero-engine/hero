package methodology

import (
	"io/fs"

	hero "github.com/hero-engine/hero"
)

// CoreFS returns a read-only filesystem rooted at the bundled
// core/methodologies/ directory. Provided for callers (registry export,
// CLI plumbing) that want the embedded preset files without reaching
// up to the top-level hero package directly. Equivalent to
// hero.CoreMethodologiesFS().
//
// NOTE: //go:embed in this package cannot reach the repo-root
// core/methodologies/ directory (embed paths are relative to the
// containing file's directory). The embed lives in content.go at the
// repo root for that reason; this function is a thin re-export.
func CoreFS() fs.FS {
	return hero.CoreMethodologiesFS()
}
