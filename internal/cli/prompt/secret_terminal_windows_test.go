//go:build windows

package prompt

import (
	"os"
	"testing"
)

func TestOpenPlatformSecretTerminalUsesWindowsConsoleHandles(t *testing.T) {
	previous := openWindowsConsoleFile
	var names []string
	openWindowsConsoleFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		names = append(names, name)
		return os.OpenFile(os.DevNull, flag, perm)
	}
	t.Cleanup(func() { openWindowsConsoleFile = previous })

	terminal, err := openPlatformSecretTerminal()
	if err != nil {
		t.Fatalf("openPlatformSecretTerminal: %v", err)
	}
	defer terminal.Close()
	if len(names) != 2 || names[0] != "CONIN$" || names[1] != "CONOUT$" {
		t.Errorf("opened %q, want [CONIN$ CONOUT$]", names)
	}
}
