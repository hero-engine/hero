//go:build unix

package cli

import (
	"io"

	"github.com/hero-engine/hero/internal/cli/prompt"
)

// promptContractPins makes the prompt package's exported signatures part of
// the compile.
//
// The field types are written out rather than inferred on purpose. Two changes
// that would otherwise look like harmless cleanups stop compiling here:
//
//   - collapsing IsInputTTY and IsOutputTTY into one predicate, which would
//     tie "may I prompt?" to what the OUTPUT stream happens to be;
//   - giving Secret an io.Reader parameter, which would let a caller — or a
//     test — hand it a pipe instead of the terminal, turning an unbypassable
//     refusal back into a conventional one.
var promptContractPins = struct {
	inputTTY  func(io.Reader) bool
	outputTTY func(io.Writer) bool
	prompt    func(io.Reader, io.Writer, string) (string, error)
	confirm   func(io.Reader, io.Writer, string, bool) (bool, error)
	choice    func(io.Reader, io.Writer, string, []string) (string, error)
	secret    func(string) (string, error)
}{
	inputTTY:  prompt.IsInputTTY,
	outputTTY: prompt.IsOutputTTY,
	prompt:    prompt.Prompt,
	confirm:   prompt.Confirm,
	choice:    prompt.Choice,
	secret:    prompt.Secret,
}
