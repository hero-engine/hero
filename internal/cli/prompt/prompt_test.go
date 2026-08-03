package prompt

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/ptytest"
)

// openPTY allocates a real pseudo-terminal, skipping the test where the
// platform or sandbox cannot provide one.
func openPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	m, s, err := ptytest.Open()
	if err != nil {
		t.Skipf("%v", err)
	}
	t.Cleanup(func() {
		s.Close()
		m.Close()
	})
	return m, s
}

// --------------------------------------------------------------------------
// predicates
// --------------------------------------------------------------------------

// TestIsInputTTYOnRealTerminal is the load-bearing predicate test. Every other
// assertion in this package is satisfied by a predicate that always returns
// false, so without a real tty in the mix the whole TTY gate could be
// vacuously "correct".
func TestIsInputTTYOnRealTerminal(t *testing.T) {
	_, slave := openPTY(t)
	if !IsInputTTY(slave) {
		t.Error("IsInputTTY(pty slave) = false, want true — the predicate does not recognize a real terminal")
	}
}

func TestIsOutputTTYOnRealTerminal(t *testing.T) {
	_, slave := openPTY(t)
	if !IsOutputTTY(slave) {
		t.Error("IsOutputTTY(pty slave) = false, want true")
	}
}

func TestIsInputTTYRejectsNonTerminals(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	tests := []struct {
		name string
		in   io.Reader
	}{
		// /dev/null is the case that matters most. It IS a character device,
		// so the ModeCharDevice predicates this package replaces reported it
		// as a terminal. term.IsTerminal must not.
		{"/dev/null", devNull},
		{"os.Pipe read end", r},
		{"strings.Reader", strings.NewReader("data\n")},
		{"regular file", mustTempFile(t)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if IsInputTTY(tc.in) {
				t.Errorf("IsInputTTY(%s) = true, want false", tc.name)
			}
		})
	}
}

func TestIsOutputTTYRejectsNonTerminals(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if IsOutputTTY(w) {
		t.Error("IsOutputTTY(pipe write end) = true, want false")
	}
	if IsOutputTTY(&strings.Builder{}) {
		t.Error("IsOutputTTY(strings.Builder) = true, want false")
	}
}

// TestPredicatesAreDistinct pins the two-predicate design. A pty slave is both
// a terminal to read from and one to write to, but the predicates must remain
// separate functions over separate stream directions: a reader that is a
// terminal says nothing about the writer, and vice versa.
func TestPredicatesAreDistinct(t *testing.T) {
	_, slave := openPTY(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Terminal input, piped output — e.g. `hero cmd | less` at a keyboard.
	if !IsInputTTY(slave) {
		t.Error("input predicate should see the terminal")
	}
	if IsOutputTTY(w) {
		t.Error("output predicate must not be fooled by terminal input")
	}

	// Piped input, terminal output — e.g. `cat x | hero cmd` at a keyboard.
	if IsInputTTY(r) {
		t.Error("input predicate must not be fooled by terminal output")
	}
	if !IsOutputTTY(slave) {
		t.Error("output predicate should see the terminal")
	}
}

// --------------------------------------------------------------------------
// Prompt
// --------------------------------------------------------------------------

func TestPrompt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain line", "hello\n", "hello"},
		{"trims surrounding space", "  hello  \n", "hello"},
		{"strips CR from CRLF", "hello\r\n", "hello"},
		{"empty line", "\n", ""},
		{"no trailing newline", "hello", "hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			got, err := Prompt(strings.NewReader(tc.input), &out, "Label: ")
			if err != nil {
				t.Fatalf("Prompt: %v", err)
			}
			if got != tc.want {
				t.Errorf("Prompt = %q, want %q", got, tc.want)
			}
			if out.String() != "Label: " {
				t.Errorf("label written = %q, want %q", out.String(), "Label: ")
			}
		})
	}
}

func TestPromptEOFReturnsError(t *testing.T) {
	var out strings.Builder
	got, err := Prompt(strings.NewReader(""), &out, "Label: ")
	if !errors.Is(err, io.EOF) {
		t.Errorf("Prompt on closed stream error = %v, want io.EOF", err)
	}
	if got != "" {
		t.Errorf("Prompt on closed stream = %q, want empty", got)
	}
}

// TestPromptDoesNotSwallowFollowingInput is the reason readLine is unbuffered.
// A bufio.Reader constructed per call reads ahead past the newline, so the
// second prompt against the same stream would see nothing. That behaviour is
// what forced connect.go to hoist a shared reader into a mutable package-level
// variable; this test is what lets that variable stay deleted.
func TestPromptDoesNotSwallowFollowingInput(t *testing.T) {
	in := strings.NewReader("first\nsecond\nthird\n")
	var out strings.Builder

	for _, want := range []string{"first", "second", "third"} {
		got, err := Prompt(in, &out, "> ")
		if err != nil {
			t.Fatalf("Prompt: %v", err)
		}
		if got != want {
			t.Fatalf("Prompt = %q, want %q — the reader buffered past its line", got, want)
		}
	}
}

// --------------------------------------------------------------------------
// Confirm
// --------------------------------------------------------------------------

func TestConfirm(t *testing.T) {
	tests := []struct {
		name  string
		input string
		def   bool
		want  bool
	}{
		{"y", "y\n", false, true},
		{"yes", "yes\n", false, true},
		{"uppercase Y", "Y\n", false, true},
		{"YES mixed case", "Yes\n", false, true},
		{"n", "n\n", true, false},
		{"no", "no\n", true, false},
		{"uppercase N", "N\n", true, false},

		// Empty and unrecognized answers fall back to the default. Both
		// polarities of the sites this replaced behaved exactly this way.
		{"empty defaults to N", "\n", false, false},
		{"empty defaults to Y", "\n", true, true},
		{"garbage defaults to N", "maybe\n", false, false},
		{"garbage defaults to Y", "maybe\n", true, true},

		// EOF is not an error for a confirm: the pre-existing sites read EOF
		// into an empty string and proceeded with their default.
		{"EOF defaults to N", "", false, false},
		{"EOF defaults to Y", "", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			got, err := Confirm(strings.NewReader(tc.input), &out, "Proceed? ", tc.def)
			if err != nil {
				t.Fatalf("Confirm: %v", err)
			}
			if got != tc.want {
				t.Errorf("Confirm(input=%q, def=%v) = %v, want %v", tc.input, tc.def, got, tc.want)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Choice
// --------------------------------------------------------------------------

func TestChoiceAcceptsAValidOption(t *testing.T) {
	var out strings.Builder
	got, err := Choice(strings.NewReader("cursor\n"), &out, "Install target",
		[]string{"opencode", "cursor", "claude"})
	if err != nil {
		t.Fatalf("Choice: %v", err)
	}
	if got != "cursor" {
		t.Errorf("Choice = %q, want %q", got, "cursor")
	}
	if !strings.Contains(out.String(), "opencode|cursor|claude") {
		t.Errorf("Choice did not list its options: %q", out.String())
	}
}

func TestChoiceEmptyLeavesDefaultToCaller(t *testing.T) {
	var out strings.Builder
	got, err := Choice(strings.NewReader("\n"), &out, "Install target",
		[]string{"opencode", "cursor"})
	if err != nil {
		t.Fatalf("Choice on empty input: %v", err)
	}
	if got != "" {
		t.Errorf("Choice on empty input = %q, want empty so the caller applies its own default", got)
	}
}

// TestChoiceRejectsUnknownValue is the assertion behind AC-7: a typo must never
// be constructed into a value.
func TestChoiceRejectsUnknownValue(t *testing.T) {
	var out strings.Builder
	got, err := Choice(strings.NewReader("clade\n"), &out, "Install target",
		[]string{"opencode", "cursor", "claude"})
	if err == nil {
		t.Fatalf("Choice(%q) returned %q with no error — a typo was accepted", "clade", got)
	}
	if got != "" {
		t.Errorf("Choice returned %q alongside an error, want empty", got)
	}
	for _, want := range []string{"clade", "opencode", "cursor", "claude"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// --------------------------------------------------------------------------
// Secret
// --------------------------------------------------------------------------

// TestSecretRefusesWithoutTTY is the assertion behind AC-5. The test binary
// normally has no controlling terminal under `go test`, so /dev/tty cannot be
// opened and Secret must refuse rather than read from anywhere else.
func TestSecretRefusesWithoutTTY(t *testing.T) {
	if _, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		t.Skip("this test process has a controlling terminal; cannot exercise the refusal path here")
	}
	got, err := Secret("Password: ")
	if !errors.Is(err, ErrNoTTY) {
		t.Errorf("Secret without a tty error = %v, want ErrNoTTY", err)
	}
	if got != "" {
		t.Errorf("Secret without a tty returned %q, want empty", got)
	}
}

// TestSecretTakesNoReader documents the structural guarantee by construction:
// Secret's signature accepts no stream, so no caller and no test can
// substitute one for the terminal. This is what makes the refusal
// unbypassable rather than merely conventional.
func TestSecretTakesNoReader(t *testing.T) {
	var f func(string) (string, error) = Secret
	_ = f
}

func mustTempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "regular")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}
