package serve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// withHomeDir points HOME at a temp directory for the duration of the
// test so PID-file paths under ~/.hero don't escape the sandbox. The
// previous HOME value is restored on cleanup.
func withHomeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, hadOld := os.LookupEnv("HOME")
	t.Setenv("HOME", dir)
	t.Cleanup(func() {
		if hadOld {
			os.Setenv("HOME", old)
		} else {
			os.Unsetenv("HOME")
		}
	})
	return dir
}

func TestPIDFilePath_DefaultPort(t *testing.T) {
	home := withHomeDir(t)
	p, err := PIDFilePath(DefaultPort)
	if err != nil {
		t.Fatalf("PIDFilePath: %v", err)
	}
	want := filepath.Join(home, ".hero", "serve.pid")
	if p != want {
		t.Errorf("PIDFilePath(default) = %q, want %q", p, want)
	}

	// Zero treated as default.
	p2, err := PIDFilePath(0)
	if err != nil {
		t.Fatalf("PIDFilePath(0): %v", err)
	}
	if p2 != want {
		t.Errorf("PIDFilePath(0) = %q, want %q", p2, want)
	}
}

func TestPIDFilePath_NonDefaultPort(t *testing.T) {
	home := withHomeDir(t)
	p, err := PIDFilePath(8123)
	if err != nil {
		t.Fatalf("PIDFilePath: %v", err)
	}
	want := filepath.Join(home, ".hero", "serve-8123.pid")
	if p != want {
		t.Errorf("PIDFilePath(8123) = %q, want %q", p, want)
	}
}

func TestWriteReadPIDFile(t *testing.T) {
	withHomeDir(t)
	path, err := WritePIDFile(DefaultPort, "test-version")
	if err != nil {
		t.Fatalf("WritePIDFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written pid file: %v", err)
	}
	var info PIDInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("parse pid file: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("pid = %d, want %d", info.PID, os.Getpid())
	}
	if info.Port != DefaultPort {
		t.Errorf("port = %d, want %d", info.Port, DefaultPort)
	}
	if info.Version != "test-version" {
		t.Errorf("version = %q, want test-version", info.Version)
	}
	if info.StartedAt.IsZero() {
		t.Errorf("started_at is zero")
	}

	// ReadPIDFile round-trip.
	read, _, err := ReadPIDFile(DefaultPort)
	if err != nil {
		t.Fatalf("ReadPIDFile: %v", err)
	}
	if read == nil {
		t.Fatal("ReadPIDFile returned nil for an existing file")
	}
	if read.PID != info.PID || read.Port != info.Port || read.Version != info.Version {
		t.Errorf("round-trip mismatch: got %+v want %+v", read, info)
	}
}

func TestReadPIDFile_Missing(t *testing.T) {
	withHomeDir(t)
	info, path, err := ReadPIDFile(DefaultPort)
	if err != nil {
		t.Fatalf("ReadPIDFile (missing): %v", err)
	}
	if info != nil {
		t.Errorf("info = %+v, want nil for missing file", info)
	}
	if path == "" {
		t.Errorf("path should be returned even for missing files")
	}
}

func TestRemovePIDFile(t *testing.T) {
	withHomeDir(t)
	if _, err := WritePIDFile(DefaultPort, "v"); err != nil {
		t.Fatalf("WritePIDFile: %v", err)
	}
	if err := RemovePIDFile(DefaultPort); err != nil {
		t.Fatalf("RemovePIDFile: %v", err)
	}
	// Idempotent.
	if err := RemovePIDFile(DefaultPort); err != nil {
		t.Fatalf("RemovePIDFile (already gone): %v", err)
	}
}

func TestIsProcessAlive(t *testing.T) {
	if !IsProcessAlive(os.Getpid()) {
		t.Errorf("IsProcessAlive(self) = false, want true")
	}
	// PID 0 is "this process group" on most Unixes — reject it.
	if IsProcessAlive(0) {
		t.Errorf("IsProcessAlive(0) = true, want false")
	}
	// PID 1 should exist on every Unix host the tests run on.
	// Negative PIDs are invalid.
	if IsProcessAlive(-1) {
		t.Errorf("IsProcessAlive(-1) = true, want false")
	}
}
