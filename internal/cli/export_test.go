package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCLIKnowledgeFile(t *testing.T, env *testEnv, rel, content string) string {
	t.Helper()
	return writeSrcFile(t, filepath.Join(env.heroDir, "knowledge"), rel, content)
}

func writeCLIMockFile(t *testing.T, env *testEnv, rel, content string) string {
	t.Helper()
	return writeSrcFile(t, filepath.Join(env.heroDir, "mocks"), rel, content)
}

func TestExportKnowledgeCommandCopiesTreeAndPrintsSummary(t *testing.T) {
	env := newTestEnv(t)
	writeCLIKnowledgeFile(t, env, "context/a.md", "alpha\n")
	writeCLIKnowledgeFile(t, env, "notes/demo/spec.md", "demo\n")
	dst := filepath.Join(env.dir, "out-knowledge")

	out, err := runCmd("export", "knowledge", dst)
	if err != nil {
		t.Fatalf("export knowledge: %v", err)
	}
	if !strings.Contains(out, "Knowledge export destination:") || !strings.Contains(out, "copied=2") {
		t.Fatalf("unexpected output: %s", out)
	}
	for _, rel := range []string{"context/a.md", "notes/demo/spec.md"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Fatalf("expected exported file %s: %v", rel, err)
		}
	}
}

func TestExportKnowledgeCommandReportsCopiedSkippedOverwrittenMergedIdenticalAndConflictedCounts(t *testing.T) {
	env := newTestEnv(t)
	writeCLIKnowledgeFile(t, env, "context/a.md", "source\n")
	dst := filepath.Join(env.dir, "out")
	writeSrcFile(t, dst, "context/a.md", "dest\n")

	out, err := runCmd("export", "knowledge", dst)
	if err == nil {
		t.Fatal("expected conflict")
	}
	for _, want := range []string{"copied=0", "skipped=0", "overwritten=0", "merged=0", "identical=0", "conflicts=1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary output missing %q: %s", want, out)
		}
	}
}

func TestExportKnowledgeCommandConflictStrategies(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIKnowledgeFile(t, env, "context/a.md", "source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "context/a.md", "dest\n")

		out, err := runCmd("export", "knowledge", "--conflict", "skip", dst)
		if err != nil {
			t.Fatalf("export skip: %v", err)
		}
		if !strings.Contains(out, "skipped=1") {
			t.Fatalf("expected skipped summary: %s", out)
		}
		got, _ := os.ReadFile(filepath.Join(dst, "context/a.md"))
		if string(got) != "dest\n" {
			t.Fatalf("skip changed destination: %q", got)
		}
	})

	t.Run("overwrite", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIKnowledgeFile(t, env, "context/a.md", "source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "context/a.md", "dest\n")

		out, err := runCmd("export", "knowledge", "--conflict", "overwrite", dst)
		if err != nil {
			t.Fatalf("export overwrite: %v", err)
		}
		if !strings.Contains(out, "overwritten=1") {
			t.Fatalf("expected overwritten summary: %s", out)
		}
		got, _ := os.ReadFile(filepath.Join(dst, "context/a.md"))
		if string(got) != "source\n" {
			t.Fatalf("overwrite did not replace destination: %q", got)
		}
	})

	t.Run("merge", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIKnowledgeFile(t, env, "context/a.md", "---\ntitle: A\ntags: [source]\n---\n\n## Source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "context/a.md", "---\ntitle: A\ntags: [dest]\n---\n\n## Dest\n")

		out, err := runCmd("export", "knowledge", "--conflict", "merge", dst)
		if err != nil {
			t.Fatalf("export merge: %v", err)
		}
		if !strings.Contains(out, "merged=1") {
			t.Fatalf("expected merged summary: %s", out)
		}
		got, _ := os.ReadFile(filepath.Join(dst, "context/a.md"))
		if !strings.Contains(string(got), "## Source") || !strings.Contains(string(got), "## Dest") {
			t.Fatalf("merge did not combine sections:\n%s", got)
		}
	})
}

func TestExportKnowledgeCommandRejectsInvalidAndInteractiveInputs(t *testing.T) {
	t.Run("default fail prints conflict count", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIKnowledgeFile(t, env, "context/a.md", "source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "context/a.md", "dest\n")

		out, err := runCmd("export", "knowledge", dst)
		if err == nil || !strings.Contains(err.Error(), "context/a.md") {
			t.Fatalf("expected conflict error, got %v", err)
		}
		if !strings.Contains(out, "conflicts=1") {
			t.Fatalf("expected conflict count in output: %s", out)
		}
	})

	t.Run("invalid conflict", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIKnowledgeFile(t, env, "context/a.md", "source\n")
		_, err := runCmd("export", "knowledge", "--conflict", "wat", filepath.Join(env.dir, "out"))
		if err == nil || !strings.Contains(err.Error(), "invalid conflict strategy") {
			t.Fatalf("expected invalid strategy error, got %v", err)
		}
	})

	t.Run("interactive requires terminal", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIKnowledgeFile(t, env, "context/a.md", "source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "context/a.md", "dest\n")
		_, err := runCmd("export", "knowledge", "--conflict", "interactive", dst)
		if err == nil || !strings.Contains(err.Error(), "requires an attached terminal") {
			t.Fatalf("expected non-interactive error, got %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(dst, "context/a.md"))
		if string(got) != "dest\n" {
			t.Fatalf("interactive non-terminal changed destination: %q", got)
		}
	})
}

func TestExportKnowledgeCommandReportsWorkspaceErrors(t *testing.T) {
	env := newTestEnv(t)
	if err := os.RemoveAll(filepath.Join(env.heroDir, "knowledge")); err != nil {
		t.Fatal(err)
	}
	_, err := runCmd("export", "knowledge", filepath.Join(env.dir, "out"))
	if err == nil || !strings.Contains(err.Error(), "source knowledge dir") {
		t.Fatalf("expected missing source error, got %v", err)
	}
}

func TestExportMocksCommandCopiesTreeAndPrintsSummary(t *testing.T) {
	env := newTestEnv(t)
	writeCLIMockFile(t, env, "landing/index.html", "<main>mock</main>\n")
	writeCLIMockFile(t, env, "landing/assets/app.css", "body{}\n")
	writeCLIMockFile(t, env, "native/screenshot.png", "PNG\x00bytes")
	dst := filepath.Join(env.dir, "out-mocks")

	out, err := runCmd("export", "mocks", dst)
	if err != nil {
		t.Fatalf("export mocks: %v", err)
	}
	if !strings.Contains(out, "Mocks export destination:") || !strings.Contains(out, "copied=3") {
		t.Fatalf("unexpected output: %s", out)
	}
	for _, rel := range []string{"landing/index.html", "landing/assets/app.css", "native/screenshot.png"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Fatalf("expected exported mock file %s: %v", rel, err)
		}
	}
}

func TestExportMocksCommandConflictStrategies(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIMockFile(t, env, "landing/index.html", "source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "landing/index.html", "dest\n")

		out, err := runCmd("export", "mocks", "--conflict", "skip", dst)
		if err != nil {
			t.Fatalf("export mocks skip: %v", err)
		}
		if !strings.Contains(out, "skipped=1") {
			t.Fatalf("expected skipped summary: %s", out)
		}
		got, _ := os.ReadFile(filepath.Join(dst, "landing/index.html"))
		if string(got) != "dest\n" {
			t.Fatalf("skip changed destination: %q", got)
		}
	})

	t.Run("overwrite", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIMockFile(t, env, "landing/index.html", "source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "landing/index.html", "dest\n")

		out, err := runCmd("export", "mocks", "--conflict", "overwrite", dst)
		if err != nil {
			t.Fatalf("export mocks overwrite: %v", err)
		}
		if !strings.Contains(out, "overwritten=1") {
			t.Fatalf("expected overwritten summary: %s", out)
		}
		got, _ := os.ReadFile(filepath.Join(dst, "landing/index.html"))
		if string(got) != "source\n" {
			t.Fatalf("overwrite did not replace destination: %q", got)
		}
	})

	t.Run("merge unsupported", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIMockFile(t, env, "landing/index.html", "source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "landing/index.html", "dest\n")

		out, err := runCmd("export", "mocks", "--conflict", "merge", dst)
		if err == nil || !strings.Contains(err.Error(), "merge is not supported for mock artifacts") {
			t.Fatalf("expected unsupported merge, got out=%s err=%v", out, err)
		}
		if !strings.Contains(out, "conflicts=1") {
			t.Fatalf("expected conflict summary: %s", out)
		}
	})
}

func TestExportMocksCommandRejectsInvalidAndInteractiveInputs(t *testing.T) {
	t.Run("invalid conflict", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIMockFile(t, env, "landing/index.html", "source\n")
		_, err := runCmd("export", "mocks", "--conflict", "wat", filepath.Join(env.dir, "out"))
		if err == nil || !strings.Contains(err.Error(), "invalid conflict strategy") {
			t.Fatalf("expected invalid strategy error, got %v", err)
		}
	})

	t.Run("interactive requires terminal", func(t *testing.T) {
		env := newTestEnv(t)
		writeCLIMockFile(t, env, "landing/index.html", "source\n")
		dst := filepath.Join(env.dir, "out")
		writeSrcFile(t, dst, "landing/index.html", "dest\n")
		_, err := runCmd("export", "mocks", "--conflict", "interactive", dst)
		if err == nil || !strings.Contains(err.Error(), "requires an attached terminal") {
			t.Fatalf("expected non-interactive error, got %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(dst, "landing/index.html"))
		if string(got) != "dest\n" {
			t.Fatalf("interactive non-terminal changed destination: %q", got)
		}
	})
}

func TestExportMocksCommandReportsWorkspaceErrors(t *testing.T) {
	env := newTestEnv(t)
	if err := os.RemoveAll(filepath.Join(env.heroDir, "mocks")); err != nil {
		t.Fatal(err)
	}
	_, err := runCmd("export", "mocks", filepath.Join(env.dir, "out"))
	if err == nil || !strings.Contains(err.Error(), "source mocks dir") {
		t.Fatalf("expected missing mocks source error, got %v", err)
	}
}
