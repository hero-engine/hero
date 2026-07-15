package spec

import "strings"

// fenceTracker tracks whether a line-by-line markdown scan is currently
// inside a fenced code block, so that structural markers a parser looks for
// (`## ` headings, `### ` sub-headings, `AC-N:` labels) are ignored when they
// are being quoted as example content rather than used as real structure.
//
// Specs in this repo document Hero's own spec format, so a fenced block that
// contains a literal `## Approach` is expected content, not a section break.
//
// Fence rules follow CommonMark closely enough for prose: a fence opens on a
// run of three or more backticks or tildes, optionally followed by an info
// string (```markdown); it closes on a run of the same character that is at
// least as long and carries no info string. Tracking the opening run's length
// is what lets a ````-delimited block quote a ``` block without closing early.
//
// Indentation is ignored entirely rather than capped at CommonMark's three
// spaces, because the parsers this feeds already detect their markers after
// trimming leading space. Matching that leniency keeps an indented example
// block from opening a fence the heading scan would then miss.
type fenceTracker struct {
	open   bool
	char   byte
	length int
}

// mark reports whether line is a fence delimiter, updating the tracker's
// state. A true result means the line is a fence marker itself and is never
// structural content.
func (f *fenceTracker) mark(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return false
	}

	c := trimmed[0]
	if c != '`' && c != '~' {
		return false
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == c {
		n++
	}
	if n < 3 {
		return false
	}
	info := strings.TrimSpace(trimmed[n:])

	if !f.open {
		// A backtick fence's info string may not itself contain a backtick;
		// that rule is what keeps inline code like ``a ` b`` from opening one.
		if c == '`' && strings.Contains(info, "`") {
			return false
		}
		f.open, f.char, f.length = true, c, n
		return true
	}

	// Only a matching, long-enough, bare fence closes the block. Anything
	// else is quoted content inside it.
	if c == f.char && n >= f.length && info == "" {
		f.open, f.char, f.length = false, 0, 0
		return true
	}
	return false
}

// inCode reports whether the scan is currently inside a fenced code block.
func (f *fenceTracker) inCode() bool { return f.open }

// skip reports whether line should be ignored by a structural scan: it is
// either a fence delimiter or sits inside a fenced block. Callers that still
// need to accumulate the line as body text should use mark/inCode directly.
func (f *fenceTracker) skip(line string) bool {
	return f.mark(line) || f.inCode()
}
