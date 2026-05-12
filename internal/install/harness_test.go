package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// harness_test.go — end-to-end install test harness.
//
// installHarness gives every Hero install test a uniform way to:
//   - stand up a fresh source content tree (agents/, commands/, skills/)
//   - stand up a fresh target project root
//   - run install for any target with any options overrides
//   - inspect the resulting filesystem with helpers that read intent
//     ("must be a symlink to X", "must contain managed region with body Y",
//     "must be byte-identical to canonical") instead of raw os.Stat calls
//
// Each phase of the single-source-install initiative will add assertion
// helpers as new behaviors land. The harness stays target-agnostic.

// installHarness is the test fixture for an end-to-end install run.
type installHarness struct {
	t         *testing.T
	SourceDir string // populated with agents/, commands/, skills/, opencode.json
	TargetDir string // empty project root the install operates against
}

// newInstallHarness creates source and target temp directories and seeds the
// source with a minimal but representative content set covering each
// content kind plus the opencode.json source. Returns the harness for the
// caller to configure further.
func newInstallHarness(t *testing.T) *installHarness {
	t.Helper()

	h := &installHarness{
		t:         t,
		SourceDir: t.TempDir(),
		TargetDir: t.TempDir(),
	}
	h.seedSource()
	return h
}

// seedSource writes the minimal agents/, commands/, skills/, and opencode.json
// content the installer needs. Adjustable per-test by appending more files
// to h.SourceDir after construction.
func (h *installHarness) seedSource() {
	h.t.Helper()

	files := map[string]string{
		"agents/engineer.md":         "# Engineer agent\nGeneric engineer.",
		"agents/reviewer.md":         "# Reviewer agent\nReviews PRs.",
		"commands/design.md":         "# /design command\nProduces a spec.",
		"commands/deliver.md":        "# /deliver command\nImplements a spec.",
		"skills/spec-format.md":      "# spec-format skill\nDefines spec structure.",
		"skills/test-strategy.md":    "# test-strategy skill\nTest pyramid guidance.",
		"opencode.json":              `{"$schema":"https://opencode.ai/config.json"}`,
	}

	for relPath, body := range files {
		full := filepath.Join(h.SourceDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			h.t.Fatalf("seed source dir %s: %v", relPath, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			h.t.Fatalf("seed source file %s: %v", relPath, err)
		}
	}
}

// Run executes the install with the given target and optional overrides.
// Mode defaults to ModeProject; TargetDir is wired automatically; Force is
// true by default so tests can re-run cleanly. Returns the result for
// assertion on .Copied/.Merged/.Skipped if needed.
func (h *installHarness) Run(target Target, override func(*Options)) *Result {
	h.t.Helper()

	opts := Options{
		SourceDir: h.SourceDir,
		Target:    target,
		Mode:      ModeProject,
		TargetDir: h.TargetDir,
		Force:     true,
	}
	if override != nil {
		override(&opts)
	}

	res, err := Run(opts)
	if err != nil {
		h.t.Fatalf("install (%s) failed: %v", target, err)
	}
	return res
}

// Assertion helpers — call from tests with a path under TargetDir.

// mustExist asserts a path exists. Returns the FileInfo on success.
func (h *installHarness) mustExist(rel string) os.FileInfo {
	h.t.Helper()
	full := filepath.Join(h.TargetDir, rel)
	info, err := os.Stat(full)
	if err != nil {
		h.t.Fatalf("expected %s to exist: %v", rel, err)
	}
	return info
}

// mustNotExist asserts a path does not exist.
func (h *installHarness) mustNotExist(rel string) {
	h.t.Helper()
	full := filepath.Join(h.TargetDir, rel)
	if _, err := os.Stat(full); err == nil {
		h.t.Fatalf("expected %s to not exist", rel)
	}
}

// mustBeRegularFile asserts a path is a regular file (not a directory,
// symlink, or other). Returns the FileInfo.
func (h *installHarness) mustBeRegularFile(rel string) os.FileInfo {
	h.t.Helper()
	full := filepath.Join(h.TargetDir, rel)
	info, err := os.Lstat(full)
	if err != nil {
		h.t.Fatalf("expected %s to exist: %v", rel, err)
	}
	if !info.Mode().IsRegular() {
		h.t.Fatalf("expected %s to be a regular file, got mode %s", rel, info.Mode())
	}
	return info
}

// mustBeSymlink asserts a path is a symlink. Returns the resolved target
// path (raw — not absolute-resolved).
func (h *installHarness) mustBeSymlink(rel string) string {
	h.t.Helper()
	full := filepath.Join(h.TargetDir, rel)
	info, err := os.Lstat(full)
	if err != nil {
		h.t.Fatalf("expected %s to exist: %v", rel, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		h.t.Fatalf("expected %s to be a symlink, got mode %s", rel, info.Mode())
	}
	target, err := os.Readlink(full)
	if err != nil {
		h.t.Fatalf("readlink(%s): %v", rel, err)
	}
	return target
}

// mustBeDirectory asserts a path is a directory.
func (h *installHarness) mustBeDirectory(rel string) {
	h.t.Helper()
	info := h.mustExist(rel)
	if !info.IsDir() {
		h.t.Fatalf("expected %s to be a directory, got mode %s", rel, info.Mode())
	}
}

// mustContain asserts a file contains a substring.
func (h *installHarness) mustContain(rel, substr string) {
	h.t.Helper()
	full := filepath.Join(h.TargetDir, rel)
	data, err := os.ReadFile(full)
	if err != nil {
		h.t.Fatalf("read %s: %v", rel, err)
	}
	if !strings.Contains(string(data), substr) {
		h.t.Fatalf("expected %s to contain %q\n--- actual ---\n%s", rel, substr, string(data))
	}
}

// mustNotContain asserts a file does NOT contain a substring.
func (h *installHarness) mustNotContain(rel, substr string) {
	h.t.Helper()
	full := filepath.Join(h.TargetDir, rel)
	data, err := os.ReadFile(full)
	if err != nil {
		h.t.Fatalf("read %s: %v", rel, err)
	}
	if strings.Contains(string(data), substr) {
		h.t.Fatalf("expected %s to NOT contain %q\n--- actual ---\n%s", rel, substr, string(data))
	}
}

// mustHaveSameContent asserts two paths (relative to TargetDir) have
// byte-identical content. Useful for testing rendered-copy mode or that
// symlinked content matches canonical.
func (h *installHarness) mustHaveSameContent(relA, relB string) {
	h.t.Helper()
	dataA, errA := os.ReadFile(filepath.Join(h.TargetDir, relA))
	if errA != nil {
		h.t.Fatalf("read %s: %v", relA, errA)
	}
	dataB, errB := os.ReadFile(filepath.Join(h.TargetDir, relB))
	if errB != nil {
		h.t.Fatalf("read %s: %v", relB, errB)
	}
	if string(dataA) != string(dataB) {
		h.t.Fatalf("%s and %s differ\n--- %s ---\n%s\n--- %s ---\n%s",
			relA, relB, relA, string(dataA), relB, string(dataB))
	}
}

// runTwiceMustBeNoop runs install twice with the same options and asserts
// the second run produces no Copied entries — the idempotency contract. (It
// may still produce Merged entries for files like AGENTS.md that get
// rewritten, but no new files should be copied.)
func (h *installHarness) runTwiceMustBeNoop(target Target, override func(*Options)) {
	h.t.Helper()
	_ = h.Run(target, override)
	second := h.Run(target, override)
	if len(second.Copied) != 0 {
		h.t.Fatalf("expected second install to be no-op (zero Copied), got %d: %v",
			len(second.Copied), second.Copied)
	}
}
