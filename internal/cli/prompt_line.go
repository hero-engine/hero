package cli

import (
	"errors"
	"io"

	"github.com/hero-engine/hero/internal/cli/prompt"
)

// End-of-stream parity for the call sites that PROPAGATE a read error.
//
// prompt.readLine treats trailing data with no final newline as a complete
// answer:
//
//	printf 'abc' | cmd   =>  ("abc", nil)
//
// bufio.Reader.ReadString('\n'), which every pre-package site used, returned
// ("abc", io.EOF) instead. Most of those sites discarded the error — the
// pre-migration reads in connect.go, install.go, install_satellites.go,
// note.go and users.go all did — so for them the two are indistinguishable and
// the migration was genuinely byte-identical.
//
// Four reads are not like that. skill.go's three (the two `skill save` fields
// and promptParam) and export.go's conflict read all act on the error, and for
// them the difference is a real behaviour change:
//
//	printf 'myname\nMy Title' | hero skill save
//	  before:  Error: reading title: EOF   exit 1, no file written
//	  without: exit 0, .hero/skills/myname.md written
//
// A failing invocation would have become a succeeding, disk-writing one. This
// child has no sanctioned behaviour changes, so those four reads go through
// here.
//
// This lives at the call sites rather than in internal/cli/prompt on purpose.
// Changing the package's semantics would silently alter the sites that discard
// their error — connect.go:775 returns "" on error and nothing covers it — and
// that is a decision for the spec that owns the package, not for this one.

// promptLine reads one line through the shared prompt package and also reports
// whether the stream ended without a terminating newline, which is the
// condition bufio.ReadString('\n') signalled by returning io.EOF.
//
// Callers that simply fail on a short read want promptLineStrict instead.
func promptLine(in io.Reader, out io.Writer, label string) (line string, unterminated bool, err error) {
	tracked := &newlineTrackingReader{r: in}
	line, err = prompt.Prompt(tracked, out, label)
	return line, tracked.endedWithoutNewline(), err
}

// promptLineStrict reads one line and treats a stream that ends without a
// newline as io.EOF, returning an empty string alongside it.
//
// That is exactly what `val, err := reader.ReadString('\n'); if err != nil {
// return "", err }` did.
func promptLineStrict(in io.Reader, out io.Writer, label string) (string, error) {
	line, unterminated, err := promptLine(in, out, label)
	if err != nil {
		return "", err
	}
	if unterminated {
		return "", io.EOF
	}
	return line, nil
}

// newlineTrackingReader records the last byte it delivered and whether the
// underlying stream reported io.EOF.
type newlineTrackingReader struct {
	r       io.Reader
	last    byte
	sawByte bool
	sawEOF  bool
}

func (t *newlineTrackingReader) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if n > 0 {
		t.last = p[n-1]
		t.sawByte = true
	}
	if errors.Is(err, io.EOF) {
		t.sawEOF = true
	}
	return n, err
}

// endedWithoutNewline reports whether the stream ran out before delivering a
// line terminator.
//
// The last-byte check is load-bearing for readers that hand back their final
// bytes and io.EOF in the same call. bufio returned a nil error for those,
// because it had already found the newline; keying only on "did we see EOF"
// would turn a perfectly terminated line into a failure.
func (t *newlineTrackingReader) endedWithoutNewline() bool {
	if !t.sawEOF {
		return false
	}
	return !t.sawByte || t.last != '\n'
}
