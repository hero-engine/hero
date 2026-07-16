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

	// Codex MCP wiring writes the machine-local User layer
	// (~/.codex/config.toml) even on project installs — isolate HOME so
	// no harness-driven install can touch the real user config.
	t.Setenv("HOME", t.TempDir())

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

	// Skills use the canonical nested layout `skills/<name>/SKILL.md` —
	// the shape the embedded content actually ships. installSkillsNested
	// consumes it directly; cursor's installSkillsFlat flattens it to
	// `<name>.md`. (An earlier flat seed masked a cursor bug where
	// installFlat skipped skill directories entirely.)
	files := map[string]string{
		"agents/engineer.md":            "---\nname: engineer\ndescription: Generic engineer.\n---\n# Engineer agent\nGeneric engineer.",
		"agents/reviewer.md":            "---\nname: reviewer\ndescription: Reviews PRs.\n---\n# Reviewer agent\nReviews PRs.",
		"commands/design.md":            "---\ndescription: Produces a spec.\n---\n# /design command\n",
		"commands/deliver.md":           "---\ndescription: Implements a spec.\n---\n# /deliver command\n",
		"skills/spec-format/SKILL.md":   "---\ndescription: Defines spec structure.\n---\n# spec-format skill\n",
		"skills/test-strategy/SKILL.md": "---\ndescription: Test pyramid guidance.\n---\n# test-strategy skill\n",
		"opencode.json":                 `{"$schema":"https://opencode.ai/config.json"}`,
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

// mustSatisfyContract walks the installed destination directory for
// the given (target, kind) and asserts every file matches the declared
// HarnessContract from internal/install/contracts.go. Reads bytes off
// the destination so it works for both symlinked and rendered-copy
// installs.
//
// Fails the test if no contract is declared for the cell — every
// (target, kind) the target installs MUST have a contract.
func (h *installHarness) mustSatisfyContract(target Target, kind ContentKind) {
	h.t.Helper()
	contract, ok := ContractsFor(target, kind)
	if !ok {
		h.t.Fatalf("no HarnessContract declared for (%s, %s) — add one to internal/install/contracts.go", target, kind)
	}
	dir := h.harnessDirFor(target, kind)
	entries, err := os.ReadDir(dir)
	if err != nil {
		h.t.Fatalf("(%s, %s): read installed dir %s: %v", target, kind, dir, err)
	}
	found := 0
	for _, e := range entries {
		var filePath string
		if kind == KindSkills {
			// Nested layout: <name>/SKILL.md
			if !e.IsDir() {
				continue
			}
			filePath = filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(filePath); err != nil {
				continue
			}
		} else if e.IsDir() {
			// Some kinds nest one more level (e.g. Copilot prompts under
			// .github/prompts/agents/<name>.prompt.md). Recurse one level
			// for files matching the contract suffix.
			nested, _ := os.ReadDir(filepath.Join(dir, e.Name()))
			for _, ne := range nested {
				if ne.IsDir() {
					continue
				}
				if !contractFilenameMatches(contract, ne.Name()) {
					continue
				}
				h.assertContract(filepath.Join(dir, e.Name(), ne.Name()), contract, target, kind)
				found++
			}
			continue
		} else {
			if !contractFilenameMatches(contract, e.Name()) {
				continue
			}
			filePath = filepath.Join(dir, e.Name())
		}
		h.assertContract(filePath, contract, target, kind)
		found++
	}
	if found == 0 {
		h.t.Fatalf("(%s, %s): expected at least one file at %s, found none", target, kind, dir)
	}
}

// contractFilenameMatches returns true if name is a candidate file for
// the contract — matches the FilenameSuffix if set, or ends in .md /
// .toml when no suffix is declared.
func contractFilenameMatches(c HarnessContract, name string) bool {
	if c.FilenameSuffix != "" {
		return strings.HasSuffix(name, c.FilenameSuffix)
	}
	if c.FilenameRequired != "" {
		return name == c.FilenameRequired
	}
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".toml")
}

// assertContract validates a single file against a HarnessContract.
func (h *installHarness) assertContract(absPath string, c HarnessContract, target Target, kind ContentKind) {
	h.t.Helper()
	data, err := os.ReadFile(absPath)
	if err != nil {
		h.t.Fatalf("read %s: %v", absPath, err)
	}
	if c.FilenameRequired != "" && filepath.Base(absPath) != c.FilenameRequired {
		h.t.Fatalf("(%s, %s) %s: filename %q does not match required %q", target, kind, absPath, filepath.Base(absPath), c.FilenameRequired)
	}
	if c.FilenameSuffix != "" && !strings.HasSuffix(filepath.Base(absPath), c.FilenameSuffix) {
		h.t.Fatalf("(%s, %s) %s: filename %q does not have required suffix %q", target, kind, absPath, filepath.Base(absPath), c.FilenameSuffix)
	}
	if len(c.RequiredFields) > 0 {
		switch c.Format {
		case FormatTOML:
			h.assertTomlRequiredFields(absPath, target, kind, c, data)
		default:
			h.assertYAMLRequiredFields(absPath, target, kind, c, data)
		}
	}
	if c.ContentValidator != nil {
		if err := c.ContentValidator(data); err != nil {
			h.t.Fatalf("(%s, %s) %s: content validator failed: %v", target, kind, absPath, err)
		}
	}
}

func (h *installHarness) assertYAMLRequiredFields(absPath string, target Target, kind ContentKind, c HarnessContract, data []byte) {
	h.t.Helper()
	fm, ok := harnessExtractFrontmatter(data)
	if !ok {
		h.t.Fatalf("(%s, %s) %s: missing or malformed YAML frontmatter (required keys: %v)", target, kind, absPath, c.RequiredFields)
	}
	for _, key := range c.RequiredFields {
		val, present := harnessFrontmatterValue(fm, key)
		if !present {
			h.t.Fatalf("(%s, %s) %s: missing required frontmatter key `%s:`", target, kind, absPath, key)
		}
		if strings.TrimSpace(val) == "" {
			h.t.Fatalf("(%s, %s) %s: frontmatter key `%s:` present but empty", target, kind, absPath, key)
		}
	}
}

// assertTomlRequiredFields scans top-level TOML keys for the required
// names. Uses a tiny line-based probe — sufficient for the simple TOML
// shape Hero emits (key = "value" or key = """ ... """).
func (h *installHarness) assertTomlRequiredFields(absPath string, target Target, kind ContentKind, c HarnessContract, data []byte) {
	h.t.Helper()
	for _, key := range c.RequiredFields {
		if !tomlHasTopLevelKey(data, key) {
			h.t.Fatalf("(%s, %s) %s: missing required TOML key `%s`", target, kind, absPath, key)
		}
	}
}

// tomlHasTopLevelKey returns true if data has a top-level `key = ...`
// or `key=...` line outside any [section]. Triple-quoted multi-line
// strings are skipped over.
func tomlHasTopLevelKey(data []byte, key string) bool {
	lines := strings.Split(string(data), "\n")
	inSection := false
	inMultiline := false
	prefix := key
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inMultiline {
			if strings.Contains(line, `"""`) {
				inMultiline = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inSection = !strings.HasPrefix(trimmed, "[[")
			// Treat any [section] header as exiting the top-level scope.
			inSection = strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "[[")
			continue
		}
		if inSection {
			continue
		}
		// Strip leading whitespace; check `key =` or `key=`.
		if strings.HasPrefix(trimmed, prefix) {
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			if strings.HasPrefix(rest, "=") {
				// Detect entering a multi-line string.
				value := strings.TrimSpace(strings.TrimPrefix(rest, "="))
				if strings.HasPrefix(value, `"""`) && !strings.HasSuffix(value[3:], `"""`) {
					inMultiline = true
				}
				return true
			}
		}
	}
	return false
}

// harnessDirFor returns the absolute path of the installed destination
// dir for the given (target, kind) under h.TargetDir. Mirrors the
// destBase logic in each internal/install/target_*.go.
func (h *installHarness) harnessDirFor(target Target, kind ContentKind) string {
	h.t.Helper()
	var destBase string
	switch target {
	case TargetClaude:
		destBase = filepath.Join(h.TargetDir, ".claude")
	case TargetOpenCode:
		destBase = filepath.Join(h.TargetDir, ".opencode")
	case TargetCursor:
		destBase = filepath.Join(h.TargetDir, ".cursor", "rules")
	case TargetCodex:
		destBase = filepath.Join(h.TargetDir, ".codex")
	case TargetCopilot:
		destBase = filepath.Join(h.TargetDir, ".github", "copilot")
	case TargetGeneric:
		destBase = filepath.Join(h.TargetDir, ".ai")
	default:
		h.t.Fatalf("harnessDirFor: unknown target %q", target)
	}
	return filepath.Join(destBase, string(kind))
}

// harnessExtractFrontmatter pulls the YAML block out of a
// `---\n...\n---\n` header and returns (frontmatter, ok). Returns
// ok=false if the file does not start with a frontmatter marker or
// the block is unterminated.
func harnessExtractFrontmatter(data []byte) ([]byte, bool) {
	const marker = "---\n"
	s := string(data)
	if !strings.HasPrefix(s, marker) {
		return nil, false
	}
	rest := s[len(marker):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, false
	}
	return []byte(rest[:end]), true
}

// harnessFrontmatterValue scans frontmatter for `key:` at the start of
// a line and returns (value-after-colon, present). Match is line-prefix
// only — does not parse nested YAML structures.
func harnessFrontmatterValue(fm []byte, key string) (string, bool) {
	prefix := key + ":"
	for _, line := range strings.Split(string(fm), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix), true
		}
	}
	return "", false
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
