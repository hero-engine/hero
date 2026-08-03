//go:build !windows

package prompt

import (
	"os"

	"golang.org/x/term"
)

type unixSecretTerminal struct {
	*os.File
}

func openPlatformSecretTerminal() (secretTerminal, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return unixSecretTerminal{File: tty}, nil
}

func (t unixSecretTerminal) readPassword() ([]byte, error) {
	return term.ReadPassword(int(t.Fd()))
}
