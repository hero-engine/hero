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
	// Consolidated layout: pointer lives inside the hero:managed region.
	if !strings.Contains(string(next), "<!-- hero:managed-start") {
		t.Errorf("NEXT.md missing managed-region start marker")
	}
	if !strings.Contains(string(next), "<!-- hero:managed-end -->") {
		t.Errorf("NEXT.md missing managed-region end marker")
	}
	if !strings.Contains(string(next), "Some content.") {
		t.Errorf("NEXT.md lost original content; got: %q", string(next))
	}

	agents, _ := os.ReadFile(agentsPath)
	if !strings.Contains(string(agents), PointerLine) {
		t.Errorf("AGENTS.md missing pointer; got: %q", string(agents))
	}
	if !strings.Contains(string(agents), "<!-- hero:managed-start") {
		t.Errorf("AGENTS.md missing managed-region start marker")
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
	if c := strings.Count(string(next), "<!-- hero:managed-start"); c != 1 {
		t.Errorf("managed-region start marker appears %d times, want 1", c)
	}
	if c := strings.Count(string(next), "<!-- hero:managed-end -->"); c != 1 {
		t.Errorf("managed-region end marker appears %d times, want 1", c)
	}
}

// TestEnsurePointer_MigratesLegacyTwoBlockLayout simulates a file
// upgraded from a pre-consolidation Hero (install marker pair plus a
// separate legacy snapshot-pointer marker pair below). After
// EnsurePointer runs, the legacy snapshot pair is gone and the pointer
// lives inside the single consolidated managed region.
func TestEnsurePointer_MigratesLegacyTwoBlockLayout(t *testing.T) {
	dir := t.TempDir()
	agentsPath := filepath.Join(dir, "AGENTS.md")

	legacy := "# AGENTS.md\n\n" +
		"<!-- hero:managed-start v=v0.10.0 -->\n" +
		"## Hero — Spec-Driven AI Engineering\n\nOld install body.\n" +
		"<!-- hero:managed-end -->\n\n" +
		"User wrote this between the two managed blocks.\n\n" +
		"<!-- >>> hero snapshot pointer (managed) >>> -->\n" +
		PointerLine + "\n" +
		"<!-- <<< hero snapshot pointer (managed) <<< -->\n\n" +
		"User tail content after both blocks.\n"
	if err := os.WriteFile(agentsPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsurePointer("", agentsPath); err != nil {
		t.Fatalf("EnsurePointer: %v", err)
	}

	got, _ := os.ReadFile(agentsPath)
	out := string(got)

	// Legacy snapshot-pointer markers must be gone.
	if strings.Contains(out, "<!-- >>> hero snapshot pointer (managed) >>> -->") {
		t.Errorf("legacy snapshot start marker survived migration:\n%s", out)
	}
	if strings.Contains(out, "<!-- <<< hero snapshot pointer (managed) <<< -->") {
		t.Errorf("legacy snapshot end marker survived migration:\n%s", out)
	}
	// Exactly one consolidated marker pair.
	if c := strings.Count(out, "<!-- hero:managed-start"); c != 1 {
		t.Errorf("expected 1 managed-start marker, got %d:\n%s", c, out)
	}
	if c := strings.Count(out, "<!-- hero:managed-end -->"); c != 1 {
		t.Errorf("expected 1 managed-end marker, got %d:\n%s", c, out)
	}
	// Pointer survives, exactly once.
	if c := strings.Count(out, PointerLine); c != 1 {
		t.Errorf("pointer appears %d times, want 1:\n%s", c, out)
	}
	// User content outside both marker pairs survives.
	if !strings.Contains(out, "User wrote this between the two managed blocks.") {
		t.Errorf("user content between old blocks lost:\n%s", out)
	}
	if !strings.Contains(out, "User tail content after both blocks.") {
		t.Errorf("user tail content lost:\n%s", out)
	}
}

// TestEnsurePointer_IdempotentOnConsolidatedLayout confirms a second
// run against an already-migrated file is a no-op.
func TestEnsurePointer_IdempotentOnConsolidatedLayout(t *testing.T) {
	dir := t.TempDir()
	agentsPath := filepath.Join(dir, "AGENTS.md")

	legacy := "# AGENTS.md\n\n" +
		"<!-- hero:managed-start v=v0.10.0 -->\n" +
		"## Hero — Spec-Driven AI Engineering\n\nOld body.\n" +
		"<!-- hero:managed-end -->\n\n" +
		"<!-- >>> hero snapshot pointer (managed) >>> -->\n" +
		PointerLine + "\n" +
		"<!-- <<< hero snapshot pointer (managed) <<< -->\n"
	if err := os.WriteFile(agentsPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsurePointer("", agentsPath); err != nil {
		t.Fatalf("first EnsurePointer: %v", err)
	}
	afterFirst, _ := os.ReadFile(agentsPath)

	if err := EnsurePointer("", agentsPath); err != nil {
		t.Fatalf("second EnsurePointer: %v", err)
	}
	afterSecond, _ := os.ReadFile(agentsPath)

	if string(afterFirst) != string(afterSecond) {
		t.Errorf("not idempotent after migration:\n--- first ---\n%s\n--- second ---\n%s", afterFirst, afterSecond)
	}
}
