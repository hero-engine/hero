package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests cover note.go's stdin handling after hasPipedInput() was replaced
// by an inversion of prompt.IsInputTTY.
//
// The replacement is not a straight polarity flip, and that is the whole point.
// hasPipedInput asked "is stdin NOT a character device", which answered "no
// piped input" for /dev/null as well as for a terminal. term.IsTerminal answers
// "not a terminal" for /dev/null, so a bare inversion starts READING /dev/null
// where the old code fell through to the inline text — producing an empty note
// and silently discarding text the user typed on the command line.
//
// Nothing else in the suite catches that. note.go has no golden baseline
// fixture, and the pre-existing note tests do not pin the input stream, so
// they inherit whatever fd 0 the test runner provides. They also assert
// that the inline text appears anywhere in the file, and it always does — it
// becomes the title and the H1 heading regardless of what happens to the body.
// A cold audit confirmed the gap: replacing the empty-body fallback with a bare
// inversion left all 103 packages green.
//
// So these tests do two things the existing ones do not: they pin the command's
// input stream to a real *os.File on /dev/null rather than inheriting it, and
// they assert on the note's BODY specifically rather than on the file as a
// whole.

// noteBody returns the content below the note's H1 heading — the part that
// actually varies with stdin handling.
//
// Asserting on the whole file would be useless here: `hero note <slug> <text>`
// puts the inline text in the frontmatter title AND the H1 AND the body, so a
// substring check over the file passes even when the body is empty.
func noteBody(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)
	idx := strings.Index(content, "\n# ")
	if idx < 0 {
		t.Fatalf("note has no H1 heading:\n%s", content)
	}
	rest := content[idx+1:]
	nl := strings.Index(rest, "\n")
	if nl < 0 {
		t.Fatalf("note heading has no body:\n%s", content)
	}
	return strings.TrimSpace(rest[nl+1:])
}

// setCommandStdin points the command tree's input at r, through cobra rather
// than through package state. note.go reads cmd.InOrStdin(), which resolves up
// to the root command, so this is the same plumbing a real invocation uses.
func setCommandStdin(t *testing.T, r io.Reader) {
	t.Helper()
	rootCmd.SetIn(r)
	t.Cleanup(func() { rootCmd.SetIn(os.Stdin) })
}

// useDevNullStdin points the command's input at a real /dev/null file handle.
//
// A *os.File is required, not a strings.Reader: prompt.IsInputTTY type-asserts
// to *os.File before consulting term.IsTerminal, so only a real file handle
// exercises the code path this test exists to protect. /dev/null specifically,
// because it is a character device — the exact stream the old and new
// predicates disagree about.
func useDevNullStdin(t *testing.T) {
	t.Helper()
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	t.Cleanup(func() { f.Close() })
	setCommandStdin(t, f)
}

// TestNoteInlineTextSurvivesDevNullStdin is the regression test for the
// inversion hazard, and the spec's Risk 4 ("getting the inversion backwards is
// the likeliest silent bug in this step") made concrete.
//
// Falsified: replacing note.go's `if body == "" { body = inlineText }` fallback
// with a bare inversion makes this fail with the body reduced to the
// empty-note placeholder.
func TestNoteInlineTextSurvivesDevNullStdin(t *testing.T) {
	env := newTestEnv(t)
	useDevNullStdin(t)

	const inline = "redis beats memcached for our access pattern"
	if _, err := runCmd("note", "cache-choice", inline); err != nil {
		t.Fatalf("note returned error: %v", err)
	}

	path := filepath.Join(env.heroDir, "knowledge", "notes",
		"cache-choice-redis-beats-memcached-for-our-access-pattern", "spec.md")
	body := noteBody(t, path)

	if strings.Contains(body, "<!-- Brainstorm") {
		t.Errorf("note body is the empty-note placeholder — the inline text was discarded because "+
			"stdin (/dev/null) was read instead of falling through to it.\nbody: %q", body)
	}
	if !strings.Contains(body, inline) {
		t.Errorf("note body lost the inline text.\nwant substring: %q\ngot: %q", inline, body)
	}
}

// TestNotePipedBodyStillBeatsInlineText guards the opposite direction.
//
// Without it, the fallback above could be "fixed" into an unconditional
// preference for the inline text, which would silently drop piped content —
// trading one silent data loss for another. Piped input winning is the
// pre-existing precedence and must not change.
func TestNotePipedBodyStillBeatsInlineText(t *testing.T) {
	env := newTestEnv(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	const piped = "actually the bottleneck is the N+1 query"
	go func() {
		_, _ = w.WriteString(piped + "\n")
		w.Close()
	}()
	t.Cleanup(func() { r.Close() })
	setCommandStdin(t, r)

	if _, err := runCmd("note", "bottleneck", "inline text that must lose"); err != nil {
		t.Fatalf("note returned error: %v", err)
	}

	path := filepath.Join(env.heroDir, "knowledge", "notes",
		"bottleneck-inline-text-that-must-lose", "spec.md")
	body := noteBody(t, path)

	if !strings.Contains(body, piped) {
		t.Errorf("piped body lost.\nwant substring: %q\ngot: %q", piped, body)
	}
	if strings.Contains(body, "inline text that must lose") {
		t.Errorf("inline text overrode the piped body — precedence reversed.\nbody: %q", body)
	}
}

// TestNoteWithoutInlineTextStaysEmptyOnDevNullStdin pins the case the fallback
// must NOT change: no inline text and nothing on stdin still yields an empty
// note, not something invented.
func TestNoteWithoutInlineTextStaysEmptyOnDevNullStdin(t *testing.T) {
	env := newTestEnv(t)
	useDevNullStdin(t)

	if _, err := runCmd("note", "empty-on-purpose"); err != nil {
		t.Fatalf("note returned error: %v", err)
	}

	path := filepath.Join(env.heroDir, "knowledge", "notes", "empty-on-purpose", "spec.md")
	if body := noteBody(t, path); !strings.Contains(body, "<!-- Brainstorm") {
		t.Errorf("note with no input should carry the empty-note placeholder, got: %q", body)
	}
}
