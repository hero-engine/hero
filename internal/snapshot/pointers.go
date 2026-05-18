package snapshot

import (
	"strings"

	"github.com/hero-engine/hero/internal/managed"
)

// PointerLine is the canonical one-liner inserted into NEXT.md and
// AGENTS.md so a fresh session knows the snapshot artifact exists.
// The text is fixed so legacy hand-authored insertions can be detected
// via exact-line match and not duplicated.
const PointerLine = "Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."

// PointerSectionID is the canonical SectionContributor identifier for
// the snapshot pointer. Stable string used for ordering/debugging.
const PointerSectionID = "snapshot:pointer"

// EnsurePointer makes sure both .hero/NEXT.md (or its team-mode
// equivalent) and AGENTS.md carry the snapshot pointer line. The
// pointer relative path is rewritten per file location so it
// resolves from whichever directory each anchor file lives in.
//
// Under the consolidated managed-region layout, the pointer is one
// section inside the single Hero-managed block. EnsurePointer composes
// a managed.Writer with just the pointer contributor (when the target
// file isn't managed by install — i.e. NEXT.md) and writes through
// it. For AGENTS.md, the install flow already wires the pointer as a
// section in the consolidated region; EnsurePointer's AGENTS.md write
// here is a fallback for callers that haven't run install (it still
// produces the same consolidated layout).
//
// Idempotent: on second and subsequent calls the orchestrator detects
// no change and skips the write.
//
// Errors writing one file do not block the other — the caller gets a
// combined error containing both failures when both occur.
func EnsurePointer(nextPath, agentsPath string) error {
	var errs []string
	if nextPath != "" {
		if err := writePointerOnly(nextPath, ".hero/SNAPSHOT.md"); err != nil {
			errs = append(errs, "NEXT.md: "+err.Error())
		}
	}
	if agentsPath != "" {
		// AGENTS.md is at the project root, so the relative pointer is
		// the same path as for NEXT.md (which actually lives in .hero/
		// — its relative path resolves the same way against the project
		// root because the pointer always names .hero/SNAPSHOT.md).
		if err := writePointerOnly(agentsPath, ".hero/SNAPSHOT.md"); err != nil {
			errs = append(errs, "AGENTS.md: "+err.Error())
		}
	}
	if len(errs) > 0 {
		return &pointerError{messages: errs}
	}
	return nil
}

type pointerError struct {
	messages []string
}

func (e *pointerError) Error() string {
	return strings.Join(e.messages, "; ")
}

// writePointerOnly drives the orchestrator for a file whose only
// section is the snapshot pointer (the NEXT.md case, or the fallback
// AGENTS.md case when install hasn't run). It also short-circuits when
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
