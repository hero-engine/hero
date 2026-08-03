// Package prompt is the single source of terminal detection and interactive
// input for Hero's CLI.
//
// Before this package existed the CLI answered "is this interactive?" four
// different ways across four files, with two mutually contradictory policies
// for reading secrets. This package replaces those forks.
//
// # Two predicates, deliberately
//
// [IsInputTTY] and [IsOutputTTY] are separate on purpose. They answer
// different questions: whether the stream a command reads from is a terminal,
// and whether the stream it writes to is one. Rendering decisions ("should I
// emit ANSI colour?") are an output question; prompting decisions ("may I ask
// the user something?") are an input question. Collapsing them into a single
// predicate is a correctness regression, not a cleanup — it would make
// `hero brief | less` render as though it were writing to a terminal, or make
// a command prompt when only its output happened to be attached to one.
//
// # Prompting rules
//
// Prompting is strictly additive. A prompt may fire only when a required
// value is missing AND the input stream is a terminal. Every pre-existing
// flag-driven or scripted invocation must behave identically, so callers gate
// on [IsInputTTY] and fail fast with their existing error text otherwise —
// they must never block waiting for input that cannot arrive.
//
// Two classes of invocation must never prompt at all:
//
//   - the agent/MCP-facing surface (focus, mail, graph node/edge, nlhook,
//     hook, brokers, run, jobs, next-emit). These carry --revision and
//     --idempotency-key semantics and are driven programmatically; a prompt
//     either hangs the caller or invites a human-invented idempotency key
//     that breaks retry semantics.
//   - anything running under --json, whose stdout is a machine-readable
//     contract.
//
// Callers read from cmd.InOrStdin() rather than os.Stdin or a package-level
// variable, which is what makes every site drivable through cobra's stream
// plumbing in a test.
package prompt

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrNoTTY is returned by [Secret] when no controlling terminal is available.
//
// Callers wrap it with the non-interactive alternative appropriate to their
// command, because that alternative differs per site (--token-stdin for
// connect, and so on). Use errors.Is to detect it.
var ErrNoTTY = errors.New("no terminal available for protected input")

// IsInputTTY reports whether r is an interactive terminal.
//
// The implementation is lifted from the only correct pre-existing fork,
// export.go's exportIsTerminal: it takes the stream as a parameter instead of
// hardcoding os.Stdin, and it asks term.IsTerminal(fd) rather than inspecting
// os.ModeCharDevice.
//
// The distinction matters. The Stat()/ModeCharDevice approach the other forks
// used reports *true* for /dev/null, because /dev/null is a character device.
// So a command run as `hero install project x < /dev/null` believed it was
// attached to a terminal, printed a prompt into a void, read EOF, and carried
// on with a default. term.IsTerminal answers the question that was actually
// meant.
//
// A non-*os.File reader (a pipe from cobra's SetIn, a strings.Reader in a
// test) is never a terminal.
func IsInputTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// IsOutputTTY reports whether w is an interactive terminal.
//
// This is the predicate for rendering decisions — width, colour, progress
// display. It is deliberately not the same function as [IsInputTTY]; see the
// package documentation.
func IsOutputTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// Prompt writes label to out and reads one line from in, returning it
// trimmed of surrounding whitespace.
//
// It does not check whether in is a terminal. Callers gate on [IsInputTTY]
// first, because the correct non-TTY behaviour is the caller's existing error
// text, not a generic one.
//
// If the stream ends before any byte is read, the returned error wraps
// io.EOF. Callers that historically treated "no input" as an empty answer
// should keep doing so explicitly.
func Prompt(in io.Reader, out io.Writer, label string) (string, error) {
	if _, err := fmt.Fprint(out, label); err != nil {
		return "", err
	}
	line, err := readLine(in)
	return strings.TrimSpace(line), err
}

// Confirm writes label to out and reads a yes/no answer, returning def when
// the answer is empty or unrecognized.
//
// Returning def for an unrecognized answer is what every call site this
// replaced already did, in both polarities: the [y/N] sites treated anything
// other than y/yes as "no", and the [Y/n] site treated anything other than
// n/no as "yes". Both reduce to "fall back to the default".
//
// End-of-stream yields def with no error, matching the pre-existing sites
// that read EOF into an empty string and proceeded with their default.
func Confirm(in io.Reader, out io.Writer, label string, def bool) (bool, error) {
	if _, err := fmt.Fprint(out, label); err != nil {
		return false, err
	}
	line, err := readLine(in)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return def, nil
	}
}

// Choice writes label and the available options to out, reads one line, and
// returns the selected option.
//
// An empty answer returns ("", nil): the caller owns its own default, because
// the default is a property of the command, not of the picker. Any other
// answer that is not exactly one of options is rejected with an error naming
// the valid values, so a typo can never be silently constructed into a bogus
// value.
func Choice(in io.Reader, out io.Writer, label string, options []string) (string, error) {
	if _, err := fmt.Fprintf(out, "%s [%s]: ", label, strings.Join(options, "|")); err != nil {
		return "", err
	}
	line, err := readLine(in)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	answer := strings.TrimSpace(line)
	if answer == "" {
		return "", nil
	}
	for _, opt := range options {
		if answer == opt {
			return answer, nil
		}
	}
	return "", fmt.Errorf("invalid choice %q — must be one of: %s",
		answer, strings.Join(options, ", "))
}

// Secret reads a value without echoing it, and refuses to read one at all
// when there is no terminal.
//
// It deliberately takes no io.Reader. The value must come from the
// controlling terminal or not at all, so there is no stream for a caller — or
// a test — to substitute. Opening /dev/tty directly is strictly stronger than
// consulting [IsInputTTY] on some passed-in stream.
//
// When a protected terminal cannot be opened, Secret returns [ErrNoTTY]. It
// never falls back to an echoed read: a fallback would accept a password or
// token from a pipe, a log-captured stream, or a shell history file.
// Automation supplies secrets through a dedicated non-interactive flag
// instead, and callers name that flag when they wrap the error.
func Secret(label string) (string, error) {
	tty, err := openSecretTerminal()
	if err != nil {
		return "", ErrNoTTY
	}
	defer tty.Close()

	if _, err := fmt.Fprint(tty, label); err != nil {
		return "", ErrNoTTY
	}
	b, err := tty.readPassword()
	fmt.Fprintln(tty)
	if err != nil {
		return "", ErrNoTTY
	}
	return strings.TrimSpace(string(b)), nil
}

// readLine reads a single line from r one byte at a time.
//
// Byte-at-a-time rather than bufio: a bufio.Reader reads ahead past the
// newline, so constructing one per call would swallow input belonging to the
// next prompt. That is exactly why connect.go had to hoist a single shared
// bufio.Reader into a mutable package-level variable. Reading unbuffered
// keeps each call self-contained, which is what lets the primitives be called
// repeatedly against the same stream — and lets the package-level variable go
// away. Prompts are human-paced, so the syscall count is irrelevant.
func readLine(r io.Reader) (string, error) {
	var b strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return strings.TrimRight(b.String(), "\r"), nil
			}
			b.WriteByte(buf[0])
		}
		if err != nil {
			// Trailing data with no final newline is still an answer.
			if errors.Is(err, io.EOF) && b.Len() > 0 {
				return strings.TrimRight(b.String(), "\r"), nil
			}
			return b.String(), err
		}
	}
}
