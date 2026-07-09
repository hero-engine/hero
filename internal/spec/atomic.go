package spec

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to path durably and atomically: it writes
// to a temp file in the same directory, fsyncs the file, renames it over
// the destination, then fsyncs the directory so the rename is durable.
//
// This is the Go-side mirror of hero-code's writer protocol (temp +
// fsync + rename + dir-fsync) used by `hero spec set-owner` so an owner
// flip never leaves a half-written spec.md behind on crash.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".hero-atomic-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail before the rename.
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("fsync temp file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	// fsync the directory so the rename survives a crash. A failure here
	// is non-fatal on platforms that don't support directory fsync.
	if d, derr := os.Open(dir); derr == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
