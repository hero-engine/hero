// Package ptytest allocates real pseudo-terminals for tests.
//
// Hero's prompt layer gates on term.IsTerminal, so any test that wants to
// exercise a "there IS a terminal" branch needs a stream the real predicate
// accepts. The alternative — teaching the production predicate to recognize a
// fake terminal type — would put a bypass in the one place that must not have
// one, since the same predicate guards whether a command may prompt at all.
//
// A real pty avoids that entirely: production code stays unaware that tests
// exist, and the branch under test is reached through the genuine check.
package ptytest

import (
	"fmt"
	"os"
)

// Open allocates a pseudo-terminal and returns its master and slave ends.
//
// The slave is a genuine terminal: term.IsTerminal reports true for it. Write
// to the master to feed input that the slave will read.
//
// Callers are responsible for closing both files. On platforms without pty
// support, or in sandboxes where /dev/ptmx is unavailable, Open returns an
// error and the caller should skip.
func Open() (master, slave *os.File, err error) {
	m, s, err := open()
	if err != nil {
		return nil, nil, fmt.Errorf("allocate pty: %w", err)
	}
	return m, s, nil
}
