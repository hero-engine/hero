package filelock

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAcquireBlocksUntilRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	first, err := Acquire(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		lock *Lock
		err  error
	}
	acquired := make(chan result, 1)
	go func() {
		lock, err := Acquire(path, 0o600)
		acquired <- result{lock: lock, err: err}
	}()

	select {
	case got := <-acquired:
		if got.lock != nil {
			_ = got.lock.Close()
		}
		t.Fatalf("contended Acquire returned before release: %v", got.err)
	case <-time.After(100 * time.Millisecond):
	}

	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-acquired:
		if got.err != nil {
			t.Fatal(got.err)
		}
		if err := got.lock.Close(); err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("contended Acquire did not proceed after release")
	}
}

func TestTryAcquireReportsCrossProcessContentionAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	lock, err := Acquire(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	if got := runTryAcquireHelper(t, path); got != "busy" {
		t.Fatalf("contended helper = %q, want busy", got)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	if got := runTryAcquireHelper(t, path); got != "acquired" {
		t.Fatalf("released helper = %q, want acquired", got)
	}
}

func TestTryAcquireHelper(t *testing.T) {
	if os.Getenv("HERO_FILELOCK_HELPER") != "1" {
		return
	}
	path := os.Args[len(os.Args)-1]
	lock, busy, err := TryAcquire(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if busy {
		_, _ = os.Stdout.WriteString("busy")
		return
	}
	defer lock.Close()
	_, _ = os.Stdout.WriteString("acquired")
}

func TestAcquirePreservesOpenFailure(t *testing.T) {
	_, err := Acquire(filepath.Join(t.TempDir(), "missing", "state.lock"), 0o600)
	if err == nil {
		t.Fatal("Acquire succeeded with a missing parent directory")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Acquire error = %v, want os.ErrNotExist", err)
	}
}

func runTryAcquireHelper(t *testing.T, path string) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestTryAcquireHelper$", "--", path)
	cmd.Env = append(os.Environ(), "HERO_FILELOCK_HELPER=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("file-lock helper failed: %v\n%s", err, output)
	}
	switch result := string(output); {
	case strings.Contains(result, "busy"):
		return "busy"
	case strings.Contains(result, "acquired"):
		return "acquired"
	default:
		return strings.TrimSpace(result)
	}
}
