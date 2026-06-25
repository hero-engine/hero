package knowledge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeKnowledgeTestFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func readKnowledgeTestFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func TestExportPreservesKnowledgeTreeShape(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "conventions/api.md", "# API\n")
	writeKnowledgeTestFile(t, src, "notes/demo/spec.md", "# Demo\n")
	writeKnowledgeTestFile(t, src, "raw/input.txt", "raw bytes\n")
	writeKnowledgeTestFile(t, src, "code/.checksums.json", `{"ok":true}`)

	summary, err := Export(src, dst, Options{})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if summary.Copied != 4 {
		t.Fatalf("copied = %d, want 4", summary.Copied)
	}
	for _, rel := range []string{"conventions/api.md", "notes/demo/spec.md", "raw/input.txt", "code/.checksums.json"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Fatalf("expected %s in destination: %v", rel, err)
		}
	}
	if got := readKnowledgeTestFile(t, src, "conventions/api.md"); got != "# API\n" {
		t.Fatalf("source knowledge file mutated: %q", got)
	}
}

func TestExportCreatesDestinationDirectoryAndNeededChildDirectories(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "missing", "knowledge")
	writeKnowledgeTestFile(t, src, "context/nested/a.md", "alpha\n")

	if _, err := Export(src, dst, Options{}); err != nil {
		t.Fatalf("export: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "context", "nested", "a.md")); err != nil {
		t.Fatalf("expected destination child directories and file: %v", err)
	}
}

func TestExportNeverMutatesSourceKnowledgeTree(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "context/a.md", "source\n")
	writeKnowledgeTestFile(t, dst, "context/a.md", "dest\n")

	if _, err := Export(src, dst, Options{Strategy: ConflictOverwrite}); err != nil {
		t.Fatalf("export overwrite: %v", err)
	}
	if got := readKnowledgeTestFile(t, src, "context/a.md"); got != "source\n" {
		t.Fatalf("source knowledge tree mutated: %q", got)
	}
}

func TestExportFailReportsConflictsAndWritesNothing(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "context/a.md", "source a\n")
	writeKnowledgeTestFile(t, src, "context/b.md", "source b\n")
	writeKnowledgeTestFile(t, dst, "context/a.md", "dest a\n")

	summary, err := Export(src, dst, Options{Strategy: ConflictFail})
	if err == nil {
		t.Fatal("expected conflict")
	}
	var conflictErr *ConflictError
	if !errors.As(err, &conflictErr) || len(conflictErr.Conflicts) != 1 || conflictErr.Conflicts[0].RelPath != "context/a.md" {
		t.Fatalf("unexpected conflict error: %#v", err)
	}
	if summary == nil || summary.Conflicts != 1 {
		t.Fatalf("summary = %#v, want one conflict", summary)
	}
	if _, err := os.Stat(filepath.Join(dst, "context/b.md")); !os.IsNotExist(err) {
		t.Fatalf("fail strategy wrote non-conflicting file; stat err=%v", err)
	}
}

func TestExportSkipLeavesExistingAndCopiesMissing(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "context/a.md", "source a\n")
	writeKnowledgeTestFile(t, src, "context/b.md", "source b\n")
	writeKnowledgeTestFile(t, dst, "context/a.md", "dest a\n")

	summary, err := Export(src, dst, Options{Strategy: ConflictSkip})
	if err != nil {
		t.Fatalf("export skip: %v", err)
	}
	if summary.Skipped != 1 || summary.Copied != 1 {
		t.Fatalf("summary = %#v, want skipped=1 copied=1", summary)
	}
	if got := readKnowledgeTestFile(t, dst, "context/a.md"); got != "dest a\n" {
		t.Fatalf("existing file changed: %q", got)
	}
	if got := readKnowledgeTestFile(t, dst, "context/b.md"); got != "source b\n" {
		t.Fatalf("missing file not copied: %q", got)
	}
}

func TestExportOverwriteReplacesFilesAndKeepsExtras(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "context/a.md", "source a\n")
	writeKnowledgeTestFile(t, dst, "context/a.md", "dest a\n")
	writeKnowledgeTestFile(t, dst, "context/extra.md", "extra\n")

	summary, err := Export(src, dst, Options{Strategy: ConflictOverwrite})
	if err != nil {
		t.Fatalf("export overwrite: %v", err)
	}
	if summary.Overwritten != 1 {
		t.Fatalf("overwritten = %d, want 1", summary.Overwritten)
	}
	if got := readKnowledgeTestFile(t, dst, "context/a.md"); got != "source a\n" {
		t.Fatalf("file not overwritten: %q", got)
	}
	if got := readKnowledgeTestFile(t, dst, "context/extra.md"); got != "extra\n" {
		t.Fatalf("extra file changed: %q", got)
	}
}

func TestExportMergeMarkdownKnowledgeEntries(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "context/foo/spec.md", `---
title: Foo
type: context
tags: [source, shared]
created: "2026-06-24"
---

# Foo

## Source Only

source body
`)
	writeKnowledgeTestFile(t, dst, "context/foo/spec.md", `---
title: Foo
type: context
tags: [dest, shared]
status: active
---

# Foo

## Dest Only

dest body
`)

	summary, err := Export(src, dst, Options{Strategy: ConflictMerge})
	if err != nil {
		t.Fatalf("export merge: %v", err)
	}
	if summary.Merged != 1 {
		t.Fatalf("merged = %d, want 1", summary.Merged)
	}
	got := readKnowledgeTestFile(t, dst, "context/foo/spec.md")
	for _, want := range []string{"tags: [dest, shared, source]", "status: active", "created: 2026-06-24", "## Dest Only", "## Source Only"} {
		if !strings.Contains(got, want) {
			t.Fatalf("merged output missing %q:\n%s", want, got)
		}
	}
}

func TestExportMergeFailsOnAmbiguousConflictsWithoutWriting(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "context/good.md", "---\ntitle: Good\ntags: [source]\n---\n\n## Source\n")
	writeKnowledgeTestFile(t, dst, "context/good.md", "---\ntitle: Good\ntags: [dest]\n---\n\n## Dest\n")
	writeKnowledgeTestFile(t, src, "raw/blob.bin", "source")
	writeKnowledgeTestFile(t, dst, "raw/blob.bin", "dest")

	_, err := Export(src, dst, Options{Strategy: ConflictMerge})
	if err == nil {
		t.Fatal("expected merge conflict")
	}
	if got := readKnowledgeTestFile(t, dst, "context/good.md"); strings.Contains(got, "## Source") {
		t.Fatalf("merge wrote before later conflict:\n%s", got)
	}
}

func TestExportRejectsUnsafePathsAndTypes(t *testing.T) {
	t.Run("destination inside source", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "knowledge")
		writeKnowledgeTestFile(t, src, "context/a.md", "a")
		_, err := Export(src, filepath.Join(src, "nested"), Options{})
		if err == nil || !strings.Contains(err.Error(), "inside source") {
			t.Fatalf("expected inside-source error, got %v", err)
		}
	})

	t.Run("source symlink", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "knowledge")
		if err := os.MkdirAll(filepath.Join(src, "context"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("target", filepath.Join(src, "context", "link.md")); err != nil {
			t.Fatal(err)
		}
		_, err := Export(src, filepath.Join(t.TempDir(), "exported"), Options{})
		if err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("expected symlink error, got %v", err)
		}
	})

	t.Run("file directory mismatch", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "knowledge")
		dst := filepath.Join(t.TempDir(), "exported")
		writeKnowledgeTestFile(t, src, "context/a/spec.md", "a")
		writeKnowledgeTestFile(t, dst, "context/a", "file blocks directory")
		_, err := Export(src, dst, Options{Strategy: ConflictMerge})
		if err == nil || !strings.Contains(err.Error(), "destination is a file") {
			t.Fatalf("expected directory mismatch, got %v", err)
		}
	})

	t.Run("scalar frontmatter conflict", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "knowledge")
		dst := filepath.Join(t.TempDir(), "exported")
		writeKnowledgeTestFile(t, src, "context/a.md", "---\ntitle: Source\n---\n\n## Source\n")
		writeKnowledgeTestFile(t, dst, "context/a.md", "---\ntitle: Dest\n---\n\n## Dest\n")
		_, err := Export(src, dst, Options{Strategy: ConflictMerge})
		if err == nil || !strings.Contains(err.Error(), "frontmatter field") {
			t.Fatalf("expected scalar conflict, got %v", err)
		}
	})
}

func TestExportInteractiveAppliesPerConflictChoices(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "context/skip.md", "source skip\n")
	writeKnowledgeTestFile(t, dst, "context/skip.md", "dest skip\n")
	writeKnowledgeTestFile(t, src, "context/overwrite.md", "source overwrite\n")
	writeKnowledgeTestFile(t, dst, "context/overwrite.md", "dest overwrite\n")
	writeKnowledgeTestFile(t, src, "context/merge.md", "---\ntitle: Merge\ntags: [source]\n---\n\n## Source\n")
	writeKnowledgeTestFile(t, dst, "context/merge.md", "---\ntitle: Merge\ntags: [dest]\n---\n\n## Dest\n")

	choices := map[string]ConflictStrategy{
		"context/skip.md":      ConflictSkip,
		"context/overwrite.md": ConflictOverwrite,
		"context/merge.md":     ConflictMerge,
	}
	summary, err := Export(src, dst, Options{
		Strategy: ConflictInteractive,
		Prompt: func(c Conflict) (ConflictStrategy, error) {
			return choices[c.RelPath], nil
		},
	})
	if err != nil {
		t.Fatalf("interactive export: %v", err)
	}
	if summary.Skipped != 1 || summary.Overwritten != 1 || summary.Merged != 1 {
		t.Fatalf("summary = %#v, want one skip/overwrite/merge", summary)
	}
	if got := readKnowledgeTestFile(t, dst, "context/skip.md"); got != "dest skip\n" {
		t.Fatalf("skip choice changed file: %q", got)
	}
	if got := readKnowledgeTestFile(t, dst, "context/overwrite.md"); got != "source overwrite\n" {
		t.Fatalf("overwrite choice not applied: %q", got)
	}
	if got := readKnowledgeTestFile(t, dst, "context/merge.md"); !strings.Contains(got, "## Source") || !strings.Contains(got, "## Dest") {
		t.Fatalf("merge choice not applied:\n%s", got)
	}
}

func TestExportInteractiveStopsWhenPromptSelectsFail(t *testing.T) {
	src := filepath.Join(t.TempDir(), "knowledge")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "context/a.md", "source a\n")
	writeKnowledgeTestFile(t, dst, "context/a.md", "dest a\n")

	summary, err := Export(src, dst, Options{
		Strategy: ConflictInteractive,
		Prompt: func(c Conflict) (ConflictStrategy, error) {
			return ConflictFail, nil
		},
	})
	if err == nil {
		t.Fatal("expected conflict")
	}
	if summary == nil || summary.Conflicts != 1 {
		t.Fatalf("summary = %#v, want one conflict", summary)
	}
	if got := readKnowledgeTestFile(t, dst, "context/a.md"); got != "dest a\n" {
		t.Fatalf("fail choice changed file: %q", got)
	}
}

func TestExportMocksPreservesTreeShape(t *testing.T) {
	src := filepath.Join(t.TempDir(), "mocks")
	dst := filepath.Join(t.TempDir(), "exported")
	writeKnowledgeTestFile(t, src, "landing/index.html", "<main>mock</main>\n")
	writeKnowledgeTestFile(t, src, "landing/assets/app.css", "body{}\n")
	writeKnowledgeTestFile(t, src, "native/screenshot.png", "PNG\x00bytes")

	summary, err := ExportMocks(src, dst, Options{})
	if err != nil {
		t.Fatalf("export mocks: %v", err)
	}
	if summary.Copied != 3 {
		t.Fatalf("copied = %d, want 3", summary.Copied)
	}
	for _, rel := range []string{"landing/index.html", "landing/assets/app.css", "native/screenshot.png"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Fatalf("expected exported mock file %s: %v", rel, err)
		}
	}
}

func TestExportMocksConflictStrategies(t *testing.T) {
	t.Run("fail reports all conflicts and writes nothing", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "mocks")
		dst := filepath.Join(t.TempDir(), "exported")
		writeKnowledgeTestFile(t, src, "landing/index.html", "source\n")
		writeKnowledgeTestFile(t, src, "landing/assets/app.css", "source css\n")
		writeKnowledgeTestFile(t, dst, "landing/index.html", "dest\n")

		summary, err := ExportMocks(src, dst, Options{Strategy: ConflictFail})
		if err == nil || !strings.Contains(err.Error(), "mock export conflicts") || !strings.Contains(err.Error(), "landing/index.html") {
			t.Fatalf("expected mock conflict, got summary=%#v err=%v", summary, err)
		}
		if summary == nil || summary.Conflicts != 1 {
			t.Fatalf("summary = %#v, want one conflict", summary)
		}
		if _, err := os.Stat(filepath.Join(dst, "landing/assets/app.css")); !os.IsNotExist(err) {
			t.Fatalf("fail strategy wrote non-conflicting mock file; stat err=%v", err)
		}
	})

	t.Run("skip leaves existing and copies missing", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "mocks")
		dst := filepath.Join(t.TempDir(), "exported")
		writeKnowledgeTestFile(t, src, "landing/index.html", "source\n")
		writeKnowledgeTestFile(t, src, "landing/assets/app.css", "source css\n")
		writeKnowledgeTestFile(t, dst, "landing/index.html", "dest\n")

		summary, err := ExportMocks(src, dst, Options{Strategy: ConflictSkip})
		if err != nil {
			t.Fatalf("export mocks skip: %v", err)
		}
		if summary.Skipped != 1 || summary.Copied != 1 {
			t.Fatalf("summary = %#v, want skipped=1 copied=1", summary)
		}
		if got := readKnowledgeTestFile(t, dst, "landing/index.html"); got != "dest\n" {
			t.Fatalf("skip changed existing mock: %q", got)
		}
	})

	t.Run("overwrite replaces conflicting files and keeps extras", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "mocks")
		dst := filepath.Join(t.TempDir(), "exported")
		writeKnowledgeTestFile(t, src, "landing/index.html", "source\n")
		writeKnowledgeTestFile(t, dst, "landing/index.html", "dest\n")
		writeKnowledgeTestFile(t, dst, "landing/extra.txt", "extra\n")

		summary, err := ExportMocks(src, dst, Options{Strategy: ConflictOverwrite})
		if err != nil {
			t.Fatalf("export mocks overwrite: %v", err)
		}
		if summary.Overwritten != 1 {
			t.Fatalf("overwritten = %d, want 1", summary.Overwritten)
		}
		if got := readKnowledgeTestFile(t, dst, "landing/index.html"); got != "source\n" {
			t.Fatalf("overwrite did not replace mock: %q", got)
		}
		if got := readKnowledgeTestFile(t, dst, "landing/extra.txt"); got != "extra\n" {
			t.Fatalf("overwrite pruned extra mock file: %q", got)
		}
	})
}

func TestExportMocksMergeBehavior(t *testing.T) {
	t.Run("identical files count as identical", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "mocks")
		dst := filepath.Join(t.TempDir(), "exported")
		writeKnowledgeTestFile(t, src, "landing/index.html", "same\n")
		writeKnowledgeTestFile(t, dst, "landing/index.html", "same\n")

		summary, err := ExportMocks(src, dst, Options{Strategy: ConflictMerge})
		if err != nil {
			t.Fatalf("export mocks merge identical: %v", err)
		}
		if summary.Identical != 1 || summary.Merged != 0 || summary.Conflicts != 0 {
			t.Fatalf("summary = %#v, want identical=1 merged=0 conflicts=0", summary)
		}
	})

	t.Run("differing artifacts fail clearly", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "mocks")
		dst := filepath.Join(t.TempDir(), "exported")
		writeKnowledgeTestFile(t, src, "landing/index.html", "source\n")
		writeKnowledgeTestFile(t, dst, "landing/index.html", "dest\n")

		summary, err := ExportMocks(src, dst, Options{Strategy: ConflictMerge})
		if err == nil || !strings.Contains(err.Error(), "merge is not supported for mock artifacts") || !strings.Contains(err.Error(), "landing/index.html") {
			t.Fatalf("expected unsupported mock merge, got summary=%#v err=%v", summary, err)
		}
		if summary == nil || summary.Conflicts != 1 {
			t.Fatalf("summary = %#v, want one conflict", summary)
		}
	})

	t.Run("interactive merge choice returns unsupported merge", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "mocks")
		dst := filepath.Join(t.TempDir(), "exported")
		writeKnowledgeTestFile(t, src, "landing/index.html", "source\n")
		writeKnowledgeTestFile(t, dst, "landing/index.html", "dest\n")

		summary, err := ExportMocks(src, dst, Options{
			Strategy: ConflictInteractive,
			Prompt: func(c Conflict) (ConflictStrategy, error) {
				return ConflictMerge, nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "merge is not supported for mock artifacts") {
			t.Fatalf("expected unsupported mock merge, got summary=%#v err=%v", summary, err)
		}
	})
}

func TestExportMocksRejectsUnsafePathsAndTypes(t *testing.T) {
	t.Run("destination inside source", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "mocks")
		writeKnowledgeTestFile(t, src, "landing/index.html", "mock\n")
		_, err := ExportMocks(src, filepath.Join(src, "nested"), Options{})
		if err == nil || !strings.Contains(err.Error(), "inside source") {
			t.Fatalf("expected inside-source error, got %v", err)
		}
	})

	t.Run("source symlink", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "mocks")
		if err := os.MkdirAll(filepath.Join(src, "landing"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("target", filepath.Join(src, "landing", "link.html")); err != nil {
			t.Fatal(err)
		}
		_, err := ExportMocks(src, filepath.Join(t.TempDir(), "exported"), Options{})
		if err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("expected symlink error, got %v", err)
		}
	})
}
