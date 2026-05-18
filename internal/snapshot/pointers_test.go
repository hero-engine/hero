package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsurePointer_FirstWrite(t *testing.T) {
	dir := t.TempDir()
	nextPath := filepath.Join(dir, "NEXT.md")
	agentsPath := filepath.Join(dir, "AGENTS.md")
	// Seed NEXT.md with existing content; AGENTS.md missing.
	if err := os.WriteFile(nextPath, []byte("# Session\n\nSome content.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsurePointer(nextPath, agentsPath); err != nil {
		t.Fatalf("EnsurePointer: %v", err)
	}

	next, _ := os.ReadFile(nextPath)
	if !strings.Contains(string(next), PointerLine) {
		t.Errorf("NEXT.md missing pointer; got: %q", string(next))
	}
	if !strings.Contains(string(next), pointerMarkerStart) {
		t.Errorf("NEXT.md missing marker block")
	}
	if !strings.Contains(string(next), "Some content.") {
		t.Errorf("NEXT.md lost original content; got: %q", string(next))
	}

	agents, _ := os.ReadFile(agentsPath)
	if !strings.Contains(string(agents), PointerLine) {
		t.Errorf("AGENTS.md missing pointer; got: %q", string(agents))
	}
}

func TestEnsurePointer_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	nextPath := filepath.Join(dir, "NEXT.md")
	if err := os.WriteFile(nextPath, []byte("# Session\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		if err := EnsurePointer(nextPath, ""); err != nil {
			t.Fatalf("EnsurePointer iter %d: %v", i, err)
		}
	}

	next, _ := os.ReadFile(nextPath)
	if c := strings.Count(string(next), PointerLine); c != 1 {
		t.Errorf("pointer line appears %d times, want 1; got:\n%s", c, string(next))
	}
	if c := strings.Count(string(next), pointerMarkerStart); c != 1 {
		t.Errorf("marker appears %d times, want 1", c)
	}
}

func TestEnsurePointer_RespectsHandAuthoredLink(t *testing.T) {
	dir := t.TempDir()
	nextPath := filepath.Join(dir, "NEXT.md")
	// User wrote the pointer themselves; we should not add a duplicate.
	custom := "# Session\n\n" + PointerLine + " I added this manually.\n"
	if err := os.WriteFile(nextPath, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsurePointer(nextPath, ""); err != nil {
		t.Fatalf("EnsurePointer: %v", err)
	}

	next, _ := os.ReadFile(nextPath)
	if c := strings.Count(string(next), PointerLine); c != 1 {
		t.Errorf("pointer line appears %d times, want 1; got:\n%s", c, string(next))
	}
}
