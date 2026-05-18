package cli

import (
	"fmt"
	"strings"
)

// cliHints is the registry of hero-invocation strings that user-facing
// code emits (help text addenda, generated CLAUDE.md, suggestion lines).
// Each entry is the args slice as the user would type it past `hero`.
// A test traverses every entry against rootCmd to assert the command
// and flag set are still valid — preventing the class of drift where
// a command is refactored but its printed advice rots in place.
//
// Each entry has a stable ID so multiple call-sites can share the same
// invocation without collapsing into one (distinct hints surface as
// distinct failures when the test catches drift).
var cliHints = []cliHint{
	{
		ID:   "context-imports-files",
		Args: []string{"context", "imports", "--files"},
		Hint: "context-imports suggestion in relevant.go and generated CLAUDE.md in init.go",
	},
}

type cliHint struct {
	ID   string
	Args []string
	Hint string
}

// Format renders the hint as the user would type it, e.g. `hero context imports --files`.
func (h cliHint) Format() string {
	return "hero " + strings.Join(h.Args, " ")
}

// cliHintByID returns the formatted invocation string for the given hint ID.
// Panics if the ID is unknown — call sites are compiled-in and should never
// reference a missing hint; a panic surfaces the bug at startup of a binary
// path rather than letting bad output slip through silently.
func cliHintByID(id string) string {
	for _, h := range cliHints {
		if h.ID == id {
			return h.Format()
		}
	}
	panic(fmt.Sprintf("cliHintByID: no hint registered with id %q", id))
}
