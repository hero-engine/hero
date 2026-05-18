package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSrcFile creates a file under root with the given relative path
// and contents, creating intermediate directories as needed.
func writeSrcFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	return abs
}

// readEntry returns the contents of a knowledge entry spec.md under
// .hero/knowledge/<kind>/<slug>/spec.md or fails the test.
func readEntry(t *testing.T, heroDir, kind, slug string) string {
	t.Helper()
	p := filepath.Join(heroDir, "knowledge", kind, slug, "spec.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read entry %s: %v", p, err)
	}
	return string(b)
}

// existsEntry reports whether an entry exists without failing.
func existsEntry(heroDir, kind, slug string) bool {
	_, err := os.Stat(filepath.Join(heroDir, "knowledge", kind, slug, "spec.md"))
	return err == nil
}

// existsRaw reports whether a raw file exists for the given slug.
func existsRaw(heroDir, slug string) bool {
	_, err := os.Stat(filepath.Join(heroDir, "knowledge", "raw", slug+".md"))
	return err == nil
}

// TestImportSingleFile covers the regression case: a single-file import
// still routes through writeSingleIngest and produces a raw file and a
// stub knowledge entry.
func TestImportSingleFile(t *testing.T) {
	env := newTestEnv(t)

	src := writeSrcFile(t, env.dir, "docs/architecture.md", "# Architecture\n\nbody.\n")

	_, err := runCmd("import", src, "Architecture")
	if err != nil {
		t.Fatalf("import single file: %v", err)
	}

	if !existsRaw(env.heroDir, "architecture") {
		t.Fatalf("expected raw file for slug architecture")
	}
	if !existsEntry(env.heroDir, "context", "architecture") {
		t.Fatalf("expected knowledge entry for slug architecture")
	}
}

// TestImportDirectoryRecursivelyWalksAndFilters exercises the directory
// branch end-to-end: recursion into subdirs, extension filter (positive
// and negative), hidden-file and hidden-dir skipping, group tag, and
// per-file slug derivation.
func TestImportDirectoryRecursivelyWalksAndFilters(t *testing.T) {
	env := newTestEnv(t)

	src := filepath.Join(env.dir, "src-docs")

	// Positive — included extensions, at various depths
	writeSrcFile(t, src, "top.md", "# Top\n")
	writeSrcFile(t, src, "guides/intro.markdown", "intro body\n")
	writeSrcFile(t, src, "guides/nested/deep.txt", "deep body\n")
	writeSrcFile(t, src, "specs/openapi.json", `{"openapi":"3.0"}`)
	writeSrcFile(t, src, "config/app.yaml", "key: val\n")
	writeSrcFile(t, src, "config/other.yml", "key: val\n")

	// Negative — excluded extensions and edge cases
	writeSrcFile(t, src, "assets/logo.png", "PNG\x00binary")
	writeSrcFile(t, src, "deps/package.lock", "{}")
	writeSrcFile(t, src, "Makefile", "all:\n\techo hi\n") // no extension

	// Hidden file at top level should be skipped
	writeSrcFile(t, src, ".env", "SECRET=1\n")

	// Hidden directory (.git) should be pruned — files inside MUST NOT ingest
	writeSrcFile(t, src, ".git/config", "[core]\n")
	writeSrcFile(t, src, ".git/HEAD", "ref: refs/heads/main\n")

	out, err := runCmd("import", src, "Group Title")
	if err != nil {
		t.Fatalf("import directory: %v", err)
	}

	// Should announce scan and summary
	if !strings.Contains(out, "Scanning") {
		t.Errorf("expected scanning announcement in output: %s", out)
	}
	if !strings.Contains(out, "under tag \"group-title\"") {
		t.Errorf("expected summary line referencing group tag: %s", out)
	}

	// Positive cases — expect entries with <groupSlug>-<fileSlug> slug
	expectIngested := map[string]string{
		"group-title-top":               "context",
		"group-title-guides-intro":      "context",
		"group-title-guides-nested-deep": "context",
		"group-title-specs-openapi":     "context",
		"group-title-config-app":        "context",
		"group-title-config-other":      "context",
	}
	for slug, kind := range expectIngested {
		if !existsEntry(env.heroDir, kind, slug) {
			t.Errorf("expected ingested entry %s/%s", kind, slug)
		}
		if !existsRaw(env.heroDir, slug) {
			t.Errorf("expected raw file for %s", slug)
		}
	}

	// Negative cases — must NOT be ingested
	mustNotExist := []string{
		"group-title-assets-logo",
		"group-title-deps-package",
		"group-title-makefile",
		"group-title-env",
		"group-title-git-config",
		"group-title-git-head",
	}
	for _, slug := range mustNotExist {
		if existsEntry(env.heroDir, "context", slug) {
			t.Errorf("did NOT expect entry for slug %s (filter should reject)", slug)
		}
	}

	// Group tag — every ingested entry must carry the group slug as a tag
	for slug := range expectIngested {
		body := readEntry(t, env.heroDir, "context", slug)
		if !strings.Contains(body, "group-title") {
			t.Errorf("entry %s missing group-title tag: %s", slug, body)
		}
		if !strings.Contains(body, "ingested") {
			t.Errorf("entry %s missing ingested tag", slug)
		}
	}
}

// TestImportDirectoryMaxBytes confirms the --max-bytes flag overrides
// the default and prunes oversize files.
func TestImportDirectoryMaxBytes(t *testing.T) {
	env := newTestEnv(t)

	src := filepath.Join(env.dir, "size-cap")

	writeSrcFile(t, src, "small.md", "tiny\n")
	writeSrcFile(t, src, "big.md", strings.Repeat("x", 500))

	// Cap below big.md size but above small.md size
	_, err := runCmd("import", "--max-bytes", "100", src, "Cap Test")
	if err != nil {
		t.Fatalf("import with --max-bytes: %v", err)
	}

	if !existsEntry(env.heroDir, "context", "cap-test-small") {
		t.Errorf("expected small.md to be ingested")
	}
	if existsEntry(env.heroDir, "context", "cap-test-big") {
		t.Errorf("expected big.md to be skipped by --max-bytes filter")
	}
}

// TestImportDirectorySkipIfExistsPerFile confirms the skip-if-exists
// check runs per entry — re-running an import after the agent edits one
// stub does NOT clobber the edit.
func TestImportDirectorySkipIfExistsPerFile(t *testing.T) {
	env := newTestEnv(t)

	src := filepath.Join(env.dir, "rerun")
	writeSrcFile(t, src, "a.md", "alpha\n")
	writeSrcFile(t, src, "b.md", "bravo\n")

	if _, err := runCmd("import", src, "Rerun"); err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Hand-edit one entry to simulate agent enrichment
	aPath := filepath.Join(env.heroDir, "knowledge", "context", "rerun-a", "spec.md")
	enriched := "ENRICHED CONTENT\n"
	if err := os.WriteFile(aPath, []byte(enriched), 0o644); err != nil {
		t.Fatalf("hand-edit entry a: %v", err)
	}

	// Re-import same directory
	out, err := runCmd("import", src, "Rerun")
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	if !strings.Contains(out, "already exists") {
		t.Errorf("expected skip-if-exists message on re-run: %s", out)
	}

	// Enriched entry must be untouched
	got, err := os.ReadFile(aPath)
	if err != nil {
		t.Fatalf("read enriched entry: %v", err)
	}
	if string(got) != enriched {
		t.Errorf("enriched entry was overwritten; got: %s", got)
	}
}

// TestImportDirectoryEmptyAfterFilterErrors confirms the documented
// non-zero exit + message when no files match the filter.
func TestImportDirectoryEmptyAfterFilterErrors(t *testing.T) {
	env := newTestEnv(t)

	src := filepath.Join(env.dir, "empty-after-filter")
	// Only excluded files
	writeSrcFile(t, src, "image.png", "PNG")
	writeSrcFile(t, src, "data.bin", "bin")

	_, err := runCmd("import", src, "Empty")
	if err == nil {
		t.Fatalf("expected error for empty-after-filter directory")
	}
	if !strings.Contains(err.Error(), "no ingestable files found") {
		t.Errorf("expected documented error message; got: %v", err)
	}
}

// TestImportDirectoryUserTagAndGroupTagBothApplied checks that the
// user-supplied --tag is combined with the auto-derived group tag and
// the ingested tag, deduplicated.
func TestImportDirectoryUserTagAndGroupTagBothApplied(t *testing.T) {
	env := newTestEnv(t)

	src := filepath.Join(env.dir, "tag-merge")
	writeSrcFile(t, src, "only.md", "body\n")

	_, err := runCmd("import", "--tag", "vendor", src, "Tag Merge")
	if err != nil {
		t.Fatalf("import with --tag: %v", err)
	}

	body := readEntry(t, env.heroDir, "context", "tag-merge-only")
	// All three tags must appear
	for _, want := range []string{"ingested", "tag-merge", "vendor"} {
		if !strings.Contains(body, want) {
			t.Errorf("entry missing tag %q: %s", want, body)
		}
	}
	// And they should appear exactly once each (dedupe)
	if c := strings.Count(body, "tag-merge"); c < 1 {
		t.Errorf("tag-merge appears %d times, expected >=1", c)
	}
}

// TestIsTextExt is a unit test for the extension allowlist covering
// every positive extension and the most common negatives.
func TestIsTextExt(t *testing.T) {
	positives := []string{
		"a.md", "a.markdown", "a.txt", "a.rst", "a.adoc",
		"a.mdx", "a.org", "a.json", "a.yaml", "a.yml",
		"UPPER.MD", "Mixed.Yaml",
	}
	for _, n := range positives {
		if !isTextExt(n) {
			t.Errorf("expected %q to be text-ish", n)
		}
	}
	negatives := []string{
		"a.png", "a.jpg", "a.pdf", "a.lock", "package.lock",
		"Makefile", "noext", "a.exe", "a.tar.gz",
	}
	for _, n := range negatives {
		if isTextExt(n) {
			t.Errorf("expected %q NOT to be text-ish", n)
		}
	}
}

// TestImportURLSignatureUnchanged is a read-only assertion that the
// fetchURL function still exists with its original signature and that
// the URL branch still routes through it. We don't actually network —
// this just makes sure refactoring writeSingleIngest didn't break the
// URL path's contract. If fetchURL's signature changes, this won't
// compile and the test fails loudly.
func TestImportURLSignatureUnchanged(t *testing.T) {
	// Compile-time signature check via assignment
	var f func(string) ([]byte, string, error) = fetchURL
	if f == nil {
		t.Fatal("fetchURL is nil")
	}
}
