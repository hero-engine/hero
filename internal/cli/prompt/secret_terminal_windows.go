//go:build windows

package prompt

import (
	"os"

	"golang.org/x/term"
)

// openWindowsConsoleFile is a narrow platform seam. It exists so Windows CI
// can verify the console-handle names without substituting secret input.
var openWindowsConsoleFile = os.OpenFile

type windowsSecretTerminal struct {
	in  *os.File
	out *os.File
}

func openPlatformSecretTerminal() (secretTerminal, error) {
	in, out, err := openWindowsConsoleFiles(openWindowsConsoleFile)
	if err != nil {
		return nil, err
	}
	return &windowsSecretTerminal{in: in, out: out}, nil
}

func (t *windowsSecretTerminal) Write(p []byte) (int, error) {
	return t.out.Write(p)
}

func (t *windowsSecretTerminal) readPassword() ([]byte, error) {
	return term.ReadPassword(int(t.in.Fd()))
}

func (t *windowsSecretTerminal) Close() error {
	err := t.in.Close()
	if closeErr := t.out.Close(); err == nil {
		err = closeErr
	}
	return err
}
