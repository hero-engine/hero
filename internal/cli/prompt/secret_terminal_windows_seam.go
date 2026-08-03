package prompt

import "os"

const (
	windowsConsoleInput  = "CONIN$"
	windowsConsoleOutput = "CONOUT$"
)

// openWindowsConsoleFiles owns the platform-independent part of protected
// Windows console acquisition: the exact handle names, read/write mode, and
// cleanup when output acquisition fails. The Windows adapter uses this seam
// directly; keeping it portable lets its contract run on every CI host.
func openWindowsConsoleFiles(openFile func(string, int, os.FileMode) (*os.File, error)) (in, out *os.File, err error) {
	in, err = openFile(windowsConsoleInput, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	out, err = openFile(windowsConsoleOutput, os.O_RDWR, 0)
	if err != nil {
		in.Close()
		return nil, nil, err
	}
	return in, out, nil
}
