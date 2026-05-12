package install

import (
	"fmt"
	"io"
	"os"
)

// log.go — small abstraction over per-file progress output so JSON-mode
// installs (and any other quiet consumer) can suppress it cleanly.

// progressOut returns the writer Hero's install logic prints progress
// to. When opts.Quiet is set, prints are discarded.
func progressOut(opts Options) io.Writer {
	if opts.Quiet {
		return io.Discard
	}
	return os.Stdout
}

// progressf is a printf wrapper that honors opts.Quiet. Use it instead
// of fmt.Printf inside the install package when reporting per-file
// progress lines.
func progressf(opts Options, format string, args ...interface{}) {
	if opts.Quiet {
		return
	}
	fmt.Fprintf(progressOut(opts), format, args...)
}
