package prompt

import (
	"errors"
	"os"
	"strings"
	"testing"
)

type fakeSecretTerminal struct {
	strings.Builder
	secret []byte
	err    error
}

func (t *fakeSecretTerminal) readPassword() ([]byte, error) { return t.secret, t.err }
func (t *fakeSecretTerminal) Close() error                  { return nil }

func TestSecretReturnsErrNoTTYWhenProtectedTerminalCannotOpen(t *testing.T) {
	previous := openSecretTerminal
	openSecretTerminal = func() (secretTerminal, error) { return nil, errors.New("unavailable") }
	t.Cleanup(func() { openSecretTerminal = previous })

	if _, err := Secret("Token: "); !errors.Is(err, ErrNoTTY) {
		t.Fatalf("Secret error = %v, want ErrNoTTY", err)
	}
}

func TestSecretUsesOnlyTheProtectedTerminal(t *testing.T) {
	previous := openSecretTerminal
	terminal := &fakeSecretTerminal{secret: []byte("  protected-value  \n")}
	openSecretTerminal = func() (secretTerminal, error) { return terminal, nil }
	t.Cleanup(func() { openSecretTerminal = previous })

	got, err := Secret("Token: ")
	if err != nil {
		t.Fatalf("Secret: %v", err)
	}
	if got != "protected-value" {
		t.Errorf("Secret = %q, want protected value", got)
	}
	if terminal.String() != "Token: \n" {
		t.Errorf("terminal output = %q, want label and newline", terminal.String())
	}
}

func TestOpenWindowsConsoleFilesUsesTheProtectedConsoleHandles(t *testing.T) {
	var names []string
	openFile := func(name string, flag int, perm os.FileMode) (*os.File, error) {
		names = append(names, name)
		return os.OpenFile(os.DevNull, flag, perm)
	}

	in, out, err := openWindowsConsoleFiles(openFile)
	if err != nil {
		t.Fatalf("openWindowsConsoleFiles: %v", err)
	}
	t.Cleanup(func() {
		in.Close()
		out.Close()
	})
	if got, want := strings.Join(names, ","), windowsConsoleInput+","+windowsConsoleOutput; got != want {
		t.Errorf("opened %q, want %q", got, want)
	}
}

func TestOpenWindowsConsoleFilesClosesInputWhenOutputCannotOpen(t *testing.T) {
	input, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open input: %v", err)
	}
	openFile := func(name string, flag int, perm os.FileMode) (*os.File, error) {
		if name == windowsConsoleInput {
			return input, nil
		}
		return nil, errors.New("output unavailable")
	}

	if _, _, err := openWindowsConsoleFiles(openFile); err == nil {
		t.Fatal("openWindowsConsoleFiles succeeded with no output handle")
	}
	if _, err := input.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Errorf("input handle was not closed after output failure: %v", err)
	}
}
