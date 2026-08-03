package prompt

import (
	"errors"
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
