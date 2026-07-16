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
	if err := os.WriteFile(nextPath, []byte("# Session\n\nSome content.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsurePointer(nextPath); err != nil {
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
}

// TestWritePointerOnly_ReplacesEntireManagedRegion characterizes the
// sharp edge that made AGENTS.md collapse to a 7-line stub twice in
// 2026. A single-section writer renders the managed region from its own
// section list alone and splices it in wholesale — it does not merge
// with sections already in the region, because sections are not
// identified in the rendered output and cannot be recovered from it.
//
// That behavior is intended for a file whose region is only the pointer
// (NEXT.md). It is catastrophic for a file install manages. This test
// exists so the destructive half stays visible: if someone makes the
// writer merge instead, this test should be rewritten deliberately, not
// silently — and if someone points EnsurePointer at an install-managed
// file again, TestEnsurePointer_DoesNotWriteInstallManagedFiles fails.
func TestWritePointerOnly_ReplacesEntireManagedRegion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "NEXT.md")
	seeded := "# NEXT.md\n\n" +
		"<!-- hero:managed-start v=dev -->\n" +
		"## Some Other Section\n\nOTHER_SECTION_BODY\n" +
		"<!-- hero:managed-end -->\n"
	if err := os.WriteFile(path, []byte(seeded), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writePointerOnly(path, ".hero/SNAPSHOT.md"); err != nil {
		t.Fatalf("writePointerOnly: %v", err)
	}

	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "OTHER_SECTION_BODY") {
		t.Fatal("writePointerOnly merged into the region — semantics changed; " +
			"re-evaluate the EnsurePointer restriction, it may no longer be needed")
	}
	if !strings.Contains(string(got), PointerLine) {
		t.Errorf("pointer missing after write; got:\n%s", got)
	}
}

// TestEnsurePointer_DoesNotWriteInstallManagedFiles pins the fix for the
// regression this package caused: the snapshot pointer writer must never
// touch AGENTS.md or CLAUDE.md. Install owns their managed region and
// already composes the pointer as one of its sections (see
// internal/install defaultSections), so a second writer here is both
// redundant and destructive.
func TestEnsurePointer_DoesNotWriteInstallManagedFiles(t *testing.T) {
	dir := t.TempDir()
	nextPath := filepath.Join(dir, "NEXT.md")
	if err := os.WriteFile(nextPath, []byte("# Session\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A realistic install-managed file: doctrine sections plus the
	// pointer that install itself contributed.
	installed := "# AGENTS.md\n\n" +
		"<!-- hero:managed-start v=dev -->\n" +
		"## Hero — Spec-Driven AI Engineering\n\nINSTALL_DOCTRINE_BODY\n\n" +
		"## Project snapshot\n\n" + PointerLine + "\n" +
		"<!-- hero:managed-end -->\n"
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(installed), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := EnsurePointer(nextPath); err != nil {
		t.Fatalf("EnsurePointer: %v", err)
	}

	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != installed {
			t.Errorf("%s was modified by the snapshot pointer writer.\n"+
				"install owns this file's managed region; writing it here "+
				"deletes every install section.\n--- got ---\n%s", name, got)
		}
	}
}

func TestEnsurePointer_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	nextPath := filepath.Join(dir, "NEXT.md")
	if err := os.WriteFile(nextPath, []byte("# Session\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		if err := EnsurePointer(nextPath); err != nil {
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
	nextPath := filepath.Join(dir, "NEXT.md")

	legacy := "# NEXT.md\n\n" +
		"<!-- hero:managed-start v=v0.10.0 -->\n" +
		"## Hero — Spec-Driven AI Engineering\n\nOld install body.\n" +
		"<!-- hero:managed-end -->\n\n" +
		"User wrote this between the two managed blocks.\n\n" +
		"<!-- >>> hero snapshot pointer (managed) >>> -->\n" +
		PointerLine + "\n" +
		"<!-- <<< hero snapshot pointer (managed) <<< -->\n\n" +
		"User tail content after both blocks.\n"
	if err := os.WriteFile(nextPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsurePointer(nextPath); err != nil {
		t.Fatalf("EnsurePointer: %v", err)
	}

	got, _ := os.ReadFile(nextPath)
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
	nextPath := filepath.Join(dir, "NEXT.md")

	legacy := "# NEXT.md\n\n" +
		"<!-- hero:managed-start v=v0.10.0 -->\n" +
		"## Hero — Spec-Driven AI Engineering\n\nOld body.\n" +
		"<!-- hero:managed-end -->\n\n" +
		"<!-- >>> hero snapshot pointer (managed) >>> -->\n" +
		PointerLine + "\n" +
		"<!-- <<< hero snapshot pointer (managed) <<< -->\n"
	if err := os.WriteFile(nextPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsurePointer(nextPath); err != nil {
		t.Fatalf("first EnsurePointer: %v", err)
	}
	afterFirst, _ := os.ReadFile(nextPath)

	if err := EnsurePointer(nextPath); err != nil {
		t.Fatalf("second EnsurePointer: %v", err)
	}
	afterSecond, _ := os.ReadFile(nextPath)

	if string(afterFirst) != string(afterSecond) {
		t.Errorf("not idempotent after migration:\n--- first ---\n%s\n--- second ---\n%s", afterFirst, afterSecond)
	}
}
