package snapshot

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// PointerLine is the canonical one-liner inserted into NEXT.md and
// AGENTS.md so a fresh session knows the snapshot artifact exists.
// The text is fixed so EnsurePointer can detect prior insertions via
// exact-line match and stay idempotent.
const PointerLine = "Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."

// pointerMarkerStart / pointerMarkerEnd bracket the managed block
// the pointer writer owns. Anything outside the markers is
// preserved verbatim. Mirrors the "hero next merge driver" managed
// region convention in .gitattributes.
const (
	pointerMarkerStart = "<!-- >>> hero snapshot pointer (managed) >>> -->"
	pointerMarkerEnd   = "<!-- <<< hero snapshot pointer (managed) <<< -->"
)

// EnsurePointer makes sure both .hero/NEXT.md (or its team-mode
// equivalent) and AGENTS.md carry the snapshot pointer line. The
// pointer relative path is rewritten per file location so it
// resolves from whichever directory each anchor file lives in.
//
// Idempotent: on second and subsequent calls, the function detects
// the marker block (or the pointer line itself) and is a no-op.
//
// Errors writing one file do not block the other — the caller gets
// a combined error containing both failures when both occur.
func EnsurePointer(nextPath, agentsPath string) error {
	var errs []string
	if nextPath != "" {
		if err := ensurePointerInFile(nextPath, ".hero/SNAPSHOT.md"); err != nil {
			errs = append(errs, "NEXT.md: "+err.Error())
		}
	}
	if agentsPath != "" {
		// AGENTS.md is at the project root, so the relative pointer
		// is the same path as for NEXT.md.
		if err := ensurePointerInFile(agentsPath, ".hero/SNAPSHOT.md"); err != nil {
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

// ensurePointerInFile inserts the marker block (containing the
// pointer line) into one file if it isn't already present.
//
// Detection rules (in order):
//   1. Marker block already present → no-op.
//   2. PointerLine appears anywhere in the body → no-op (legacy
//      hand-authored insertions are honored without duplication).
//   3. Otherwise append the marker block at the end of the file.
func ensurePointerInFile(path, snapshotRelativePath string) error {
	existing, _ := os.ReadFile(path)
	src := string(existing)

	body := buildPointerBlock(snapshotRelativePath)

	// Already managed?
	if strings.Contains(src, pointerMarkerStart) {
		return nil
	}
	// Already hand-authored?
	pointer := buildPointerLine(snapshotRelativePath)
	if strings.Contains(src, pointer) {
		return nil
	}

	// Append the marker block. Preserve trailing newline rules.
	out := strings.TrimRight(src, "\n")
	if out != "" {
		out += "\n\n"
	}
	out += body + "\n"

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(out), 0o644)
}

func buildPointerLine(snapshotRelativePath string) string {
	// PointerLine is the canonical phrasing; we only swap the path
	// when callers want a non-default location.
	if snapshotRelativePath == ".hero/SNAPSHOT.md" || snapshotRelativePath == "" {
		return PointerLine
	}
	return "Project shape: see [SNAPSHOT.md](" + snapshotRelativePath + ")."
}

func buildPointerBlock(snapshotRelativePath string) string {
	var b bytes.Buffer
	b.WriteString(pointerMarkerStart)
	b.WriteString("\n")
	b.WriteString(buildPointerLine(snapshotRelativePath))
	b.WriteString("\n")
	b.WriteString(pointerMarkerEnd)
	return b.String()
}
