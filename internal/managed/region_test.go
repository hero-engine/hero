package managed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSection is a SectionContributor used to drive Writer in tests.
type fakeSection struct {
	id    string
	title string
	body  string
	err   error
}

func (s fakeSection) SectionID() string                       { return s.id }
func (s fakeSection) SectionTitle() string                    { return s.title }
func (s fakeSection) Render(_ Context) (string, error)        { return s.body, s.err }

func TestWriter_RendersSectionsInOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	w := Writer{
		File: path,
		Sections: []SectionContributor{
			fakeSection{id: "a", title: "Alpha", body: "alpha body."},
			fakeSection{id: "b", title: "Bravo", body: "bravo body."},
		},
		DefaultH1: "# AGENTS.md",
	}
	if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, _ := os.ReadFile(path)
	out := string(got)

	// Order: alpha precedes bravo.
	aIdx := strings.Index(out, "alpha body.")
	bIdx := strings.Index(out, "bravo body.")
	if aIdx < 0 || bIdx < 0 || aIdx >= bIdx {
		t.Errorf("expected alpha before bravo; got order alpha=%d bravo=%d in:\n%s", aIdx, bIdx, out)
	}
	// Both H2s are present.
	if !strings.Contains(out, "## Alpha") {
		t.Errorf("missing Alpha H2:\n%s", out)
	}
	if !strings.Contains(out, "## Bravo") {
		t.Errorf("missing Bravo H2:\n%s", out)
	}
	// Exactly one marker pair.
	if c := strings.Count(out, "<!-- hero:managed-start"); c != 1 {
		t.Errorf("want 1 managed-start, got %d", c)
	}
}

func TestWriter_EmptySectionsAreSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	w := Writer{
		File: path,
		Sections: []SectionContributor{
			fakeSection{id: "a", title: "Alpha", body: "alpha body."},
			fakeSection{id: "empty", title: "Empty Section", body: ""},
			fakeSection{id: "c", title: "Charlie", body: "charlie body."},
		},
	}
	if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, _ := os.ReadFile(path)
	out := string(got)
	if strings.Contains(out, "## Empty Section") {
		t.Errorf("empty section heading should not appear:\n%s", out)
	}
	if !strings.Contains(out, "alpha body.") || !strings.Contains(out, "charlie body.") {
		t.Errorf("non-empty sections missing:\n%s", out)
	}
}

func TestWriter_EmptyTitleSkipsHeading(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	w := Writer{
		File:     path,
		Sections: []SectionContributor{
			fakeSection{id: "no-heading", title: "", body: "## My Own Heading\n\nbody"},
		},
	}
	if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, _ := os.ReadFile(path)
	out := string(got)
	if strings.Contains(out, "## \n") {
		t.Errorf("empty title leaked as bare H2 marker:\n%s", out)
	}
	if !strings.Contains(out, "## My Own Heading") {
		t.Errorf("contributor's own heading lost:\n%s", out)
	}
}

func TestWriter_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	w := Writer{
		File:     path,
		Sections: []SectionContributor{fakeSection{id: "a", title: "Alpha", body: "stable body."}},
	}
	ctx := Context{HeroVersion: "test"}

	if _, err := w.Write(ctx); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	first, _ := os.ReadFile(path)
	firstStat, _ := os.Stat(path)

	changed, err := w.Write(ctx)
	if err != nil {
		t.Fatalf("second Write: %v", err)
	}
	if changed {
		t.Errorf("expected second Write to be a no-op, got changed=true")
	}
	second, _ := os.ReadFile(path)
	secondStat, _ := os.Stat(path)

	if string(first) != string(second) {
		t.Errorf("not idempotent at byte level:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	// Filesystem mtime should be unchanged since the second write was skipped.
	if !firstStat.ModTime().Equal(secondStat.ModTime()) {
		t.Errorf("file was rewritten on the idempotent path (mtime changed: %v -> %v)", firstStat.ModTime(), secondStat.ModTime())
	}
}

func TestWriter_MigratesLegacyTwoBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	legacy := "# AGENTS.md\n\n" +
		"<!-- hero:managed-start v=v0.10.0 -->\n" +
		"## Hero — Spec-Driven AI Engineering\n\nOld install body.\n" +
		"<!-- hero:managed-end -->\n\n" +
		"User content between the two managed blocks.\n\n" +
		"<!-- >>> hero snapshot pointer (managed) >>> -->\n" +
		"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).\n" +
		"<!-- <<< hero snapshot pointer (managed) <<< -->\n\n" +
		"User tail after both blocks.\n"
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	w := Writer{
		File: path,
		Sections: []SectionContributor{
			fakeSection{id: "install", title: "Hero — Spec-Driven AI Engineering", body: "New install body."},
			fakeSection{id: "snapshot", title: "Project snapshot", body: "Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."},
		},
	}
	if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, _ := os.ReadFile(path)
	out := string(got)

	// Legacy snapshot markers must be gone.
	if strings.Contains(out, "<!-- >>> hero snapshot pointer (managed) >>> -->") {
		t.Errorf("legacy snapshot start marker survived:\n%s", out)
	}
	if strings.Contains(out, "<!-- <<< hero snapshot pointer (managed) <<< -->") {
		t.Errorf("legacy snapshot end marker survived:\n%s", out)
	}
	// Exactly one consolidated marker pair.
	if c := strings.Count(out, "<!-- hero:managed-start"); c != 1 {
		t.Errorf("want 1 managed-start marker, got %d:\n%s", c, out)
	}
	if c := strings.Count(out, "<!-- hero:managed-end -->"); c != 1 {
		t.Errorf("want 1 managed-end marker, got %d:\n%s", c, out)
	}
	// User content outside both old pairs survives.
	if !strings.Contains(out, "User content between the two managed blocks.") {
		t.Errorf("between-blocks user content lost:\n%s", out)
	}
	if !strings.Contains(out, "User tail after both blocks.") {
		t.Errorf("trailing user content lost:\n%s", out)
	}
	// Re-running is idempotent.
	first := out
	if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
		t.Fatalf("second Write after migration: %v", err)
	}
	got2, _ := os.ReadFile(path)
	if string(got2) != first {
		t.Errorf("migration not idempotent on second run:\n--- first ---\n%s\n--- second ---\n%s", first, got2)
	}
}

func TestWriter_NoOpWhenAlreadyConsolidated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	w := Writer{
		File: path,
		Sections: []SectionContributor{
			fakeSection{id: "install", title: "Install", body: "install body."},
			fakeSection{id: "snapshot", title: "Snapshot", body: "snapshot body."},
		},
	}
	if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	changed, err := w.Write(Context{HeroVersion: "test"})
	if err != nil {
		t.Fatalf("second Write: %v", err)
	}
	if changed {
		t.Errorf("second Write should be no-op")
	}
}

func TestWriter_OldLayoutWithOnlySnapshotBlock(t *testing.T) {
	// Edge: file has only the legacy snapshot block (the install block
	// was somehow removed). The snapshot block should be stripped and
	// the consolidated layout written.
	dir := t.TempDir()
	path := filepath.Join(dir, "NEXT.md")

	input := "# NEXT.md\n\nUser preamble.\n\n" +
		"<!-- >>> hero snapshot pointer (managed) >>> -->\n" +
		"Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).\n" +
		"<!-- <<< hero snapshot pointer (managed) <<< -->\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	w := Writer{
		File: path,
		Sections: []SectionContributor{
			fakeSection{id: "snapshot", title: "Project snapshot", body: "Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md)."},
		},
	}
	if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, _ := os.ReadFile(path)
	out := string(got)
	if strings.Contains(out, "<!-- >>> hero snapshot pointer (managed) >>> -->") {
		t.Errorf("legacy snapshot marker survived:\n%s", out)
	}
	if !strings.Contains(out, "User preamble.") {
		t.Errorf("user preamble lost:\n%s", out)
	}
	if !strings.Contains(out, "<!-- hero:managed-start") {
		t.Errorf("consolidated marker missing:\n%s", out)
	}
}

func TestWriter_FreshFileWithoutAnyMarkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	preexisting := "# Pre-existing User File\n\nUser wrote a lot here.\nKeep it all.\n"
	if err := os.WriteFile(path, []byte(preexisting), 0o644); err != nil {
		t.Fatal(err)
	}

	w := Writer{
		File: path,
		Sections: []SectionContributor{
			fakeSection{id: "a", title: "Alpha", body: "alpha body."},
		},
	}
	if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, _ := os.ReadFile(path)
	out := string(got)
	if !strings.HasPrefix(out, "# Pre-existing User File\n") {
		t.Errorf("user H1 should be preserved at top:\n%s", out)
	}
	if !strings.Contains(out, "User wrote a lot here.") {
		t.Errorf("user content lost:\n%s", out)
	}
	if !strings.Contains(out, "## Alpha") {
		t.Errorf("section heading missing:\n%s", out)
	}
}

func TestWriter_OneMarkerPairInvariant(t *testing.T) {
	// Walk repo-rendered fixtures and assert each carries exactly one
	// managed-start / managed-end pair. Driven inline here via a small
	// matrix; the install/snapshot tests have analogous coverage for
	// AGENTS.md / CLAUDE.md / NEXT.md.
	dir := t.TempDir()
	cases := []struct {
		name     string
		sections []SectionContributor
	}{
		{
			name: "two sections",
			sections: []SectionContributor{
				fakeSection{id: "a", title: "Alpha", body: "alpha"},
				fakeSection{id: "b", title: "Bravo", body: "bravo"},
			},
		},
		{
			name: "single section",
			sections: []SectionContributor{
				fakeSection{id: "solo", title: "Solo", body: "solo body"},
			},
		},
		{
			name: "section with empty title (owns its body)",
			sections: []SectionContributor{
				fakeSection{id: "raw", title: "", body: "## I am H2\n\nbody"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(dir, fmt.Sprintf("fixture-%s.md", c.name))
			w := Writer{File: path, Sections: c.sections}
			if _, err := w.Write(Context{HeroVersion: "test"}); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, _ := os.ReadFile(path)
			out := string(got)
			if c := strings.Count(out, "<!-- hero:managed-start"); c != 1 {
				t.Errorf("expected exactly 1 managed-start marker, got %d:\n%s", c, out)
			}
			if c := strings.Count(out, "<!-- hero:managed-end -->"); c != 1 {
				t.Errorf("expected exactly 1 managed-end marker, got %d:\n%s", c, out)
			}
		})
	}
}

func TestWriter_RendersErrorFromSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	w := Writer{
		File: path,
		Sections: []SectionContributor{
			fakeSection{id: "broken", title: "Broken", err: fmt.Errorf("boom")},
		},
	}
	_, err := w.Write(Context{HeroVersion: "test"})
	if err == nil {
		t.Fatal("expected error from broken section")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Errorf("error message should name the section id; got: %v", err)
	}
}
