package snapshot

import (
	"github.com/hero-engine/hero/internal/managed"
)

// PointerLine is the canonical one-liner inserted into NEXT.md so a
// fresh session knows the snapshot artifact exists.
// The text is fixed so legacy hand-authored insertions can be detected
// via exact-line match and not duplicated.
const PointerLine = "Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."

// PointerSectionID is the canonical SectionContributor identifier for
// the snapshot pointer. Stable string used for ordering/debugging.
const PointerSectionID = "snapshot:pointer"

// EnsurePointer makes sure .hero/NEXT.md (or its team-mode equivalent)
// carries the snapshot pointer line.
//
// NEXT.md is the only file written here. AGENTS.md and CLAUDE.md get
// the pointer from install, which composes it as one section of the
// consolidated managed region (internal/install defaultSections).
//
// Do not extend this to a file that install manages. writePointerOnly
// renders the managed region from its own section list alone, so it
// replaces — not merges — whatever the region held. Pointing it at an
// install-managed file deletes every install section. That is not
// hypothetical: it silently reduced AGENTS.md to a 7-line stub twice
// (May 31 and Jun 9 2026) after the two-block layout was consolidated
// into one region and this writer kept overwriting the whole block.
//
// Idempotent: on second and subsequent calls the orchestrator detects
// no change and skips the write.
func EnsurePointer(nextPath string) error {
	if nextPath == "" {
		return nil
	}
	return writePointerOnly(nextPath, ".hero/SNAPSHOT.md")
}

// writePointerOnly drives the orchestrator for a file whose only
// section is the snapshot pointer. It also short-circuits when
// a user has hand-authored the pointer line outside any marker pair,
// preserving the legacy "respect hand-authored insertions" behavior.
func writePointerOnly(path, snapshotRelativePath string) error {
	contrib := NewPointerSection(path, snapshotRelativePath)
	writer := managed.Writer{
		File:     path,
		Sections: []managed.SectionContributor{contrib},
	}
	_, err := writer.Write(managed.Context{File: path})
	return err
}

// pointerSection adapts the snapshot pointer line to
// managed.SectionContributor. The section emits a single markdown
// paragraph; the orchestrator wraps it with its H2 heading inside the
// managed region.
type pointerSection struct {
	filePath             string
	snapshotRelativePath string
}

// NewPointerSection returns a SectionContributor that renders the
// snapshot pointer line inside the consolidated managed region of
// filePath. snapshotRelativePath is the path SNAPSHOT.md resolves to
// from the directory containing filePath (".hero/SNAPSHOT.md" for
// root-level AGENTS.md / CLAUDE.md).
func NewPointerSection(filePath, snapshotRelativePath string) managed.SectionContributor {
	if snapshotRelativePath == "" {
		snapshotRelativePath = ".hero/SNAPSHOT.md"
	}
	return pointerSection{
		filePath:             filePath,
		snapshotRelativePath: snapshotRelativePath,
	}
}

func (s pointerSection) SectionID() string    { return PointerSectionID }
func (s pointerSection) SectionTitle() string { return "Project snapshot" }

func (s pointerSection) Render(_ managed.Context) (string, error) {
	return buildPointerLine(s.snapshotRelativePath), nil
}

func buildPointerLine(snapshotRelativePath string) string {
	if snapshotRelativePath == ".hero/SNAPSHOT.md" || snapshotRelativePath == "" {
		return PointerLine
	}
	return "Project shape: see [SNAPSHOT.md](" + snapshotRelativePath + ")."
}
