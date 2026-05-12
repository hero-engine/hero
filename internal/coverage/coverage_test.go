package coverage

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"foo_test.go", true},
		{"foo.test.ts", true},
		{"foo.spec.js", true},
		{"foo.test.tsx", true},
		{"test_foo.py", true},
		{"foo.go", false},
		{"foo.ts", false},
		{"readme.md", false},
		// Plain .rs is not a test file by name; discovery uses content sniff.
		{"foo.rs", false},
	}

	for _, tt := range tests {
		if got := isTestFile(tt.name); got != tt.want {
			t.Errorf("isTestFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestKeywordsMatch(t *testing.T) {
	content := "func testcsvexport(t *testing.t) { assert streaming works }"

	tests := []struct {
		keywords []string
		want     bool
	}{
		{[]string{"csv", "export"}, true},
		{[]string{"streaming"}, true},
		{[]string{"nonexistent_keyword"}, false},
	}

	for _, tt := range tests {
		if got := keywordsMatch(tt.keywords, content); got != tt.want {
			t.Errorf("keywordsMatch(%v, ...) = %v, want %v", tt.keywords, got, tt.want)
		}
	}
}

func TestDiscoverTestFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"main.go":            "package main",
		"main_test.go":       "package main",
		"src/app.ts":         "export {}",
		"src/app.test.ts":    "describe('app')",
		"src/app.spec.js":    "test('works')",
		"tests/test_auth.py": "def test_auth():",
		"node_modules/x.js":  "skip me",
		"vendor/y_test.go":   "skip me",
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}

	found, err := discoverTestFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(found) != 4 {
		t.Errorf("expected 4 test files, got %d: %v", len(found), found)
	}
}

func TestDiscoverTestFilesRust(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		// Cargo integration test convention — included by path.
		"tests/api.rs": "fn smoke() {}",
		// Source file with inline #[test] — included by content sniff.
		"src/parser.rs": `pub fn parse() {}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn parses_input() {}
}
`,
		// Source file with no test markers — excluded.
		"src/io.rs": "pub fn read() {}\n",
		// Random .rs deep in source tree with #[tokio::test] — included.
		"src/handlers/mod.rs": `#[tokio::test]
async fn handles_request() {}
`,
		// target/ build dir is skipped entirely.
		"target/debug/build/foo.rs": "#[test] fn whatever() {}",
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}

	found, err := discoverTestFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]bool)
	for _, p := range found {
		rel, _ := filepath.Rel(dir, p)
		got[filepath.ToSlash(rel)] = true
	}

	wantIncluded := []string{
		"tests/api.rs",
		"src/parser.rs",
		"src/handlers/mod.rs",
	}
	wantExcluded := []string{
		"src/io.rs",
		"target/debug/build/foo.rs",
	}

	for _, p := range wantIncluded {
		if !got[p] {
			t.Errorf("expected %q to be discovered, got: %v", p, found)
		}
	}
	for _, p := range wantExcluded {
		if got[p] {
			t.Errorf("expected %q to be excluded, got: %v", p, found)
		}
	}
}

func TestPathHasSegment(t *testing.T) {
	cases := []struct {
		path string
		seg  string
		want bool
	}{
		{"/repo/tests/api.rs", "tests", true},
		{"/repo/src/integration_tests/api.rs", "tests", false},
		{"/repo/src/foo.rs", "tests", false},
		{"tests/api.rs", "tests", true},
		// The filename itself shouldn't count as a segment.
		{"/repo/tests", "tests", false},
	}
	for _, c := range cases {
		if got := pathHasSegment(c.path, c.seg); got != c.want {
			t.Errorf("pathHasSegment(%q, %q) = %v, want %v", c.path, c.seg, got, c.want)
		}
	}
}

func TestExtractTestNames(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		content string
		want    []string
	}{
		{
			name: "go",
			path: "foo_test.go",
			content: `package foo

func TestParsesCSV(t *testing.T) {}
func BenchmarkExport(b *testing.B) {}
func helper() {}
`,
			want: []string{"TestParsesCSV", "BenchmarkExport"},
		},
		{
			name: "rust inline",
			path: "src/parser.rs",
			content: `#[cfg(test)]
mod tests {
    #[test]
    fn parses_input() {}

    #[tokio::test]
    async fn handles_async() {}
}
`,
			want: []string{"parses_input", "handles_async"},
		},
		{
			name:    "python",
			path:    "test_auth.py",
			content: "def test_login_succeeds():\n    pass\n\ndef helper():\n    pass\n",
			want:    []string{"test_login_succeeds"},
		},
		{
			name: "typescript label",
			path: "app.test.ts",
			content: `describe("auth", () => {
    it("logs in", () => {});
    test("logs out", () => {});
});
`,
			want: []string{"auth", "logs in", "logs out"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractTestNames(c.path, c.content)
			gotNames := make([]string, len(got))
			for i, g := range got {
				gotNames[i] = g.display
			}
			sort.Strings(gotNames)
			sort.Strings(c.want)
			if !reflect.DeepEqual(gotNames, c.want) {
				t.Errorf("extractTestNames(%s): got %v, want %v", c.name, gotNames, c.want)
			}
		})
	}
}

func TestTokenizeTestName(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"TestParsesCSV", []string{"parses", "csv"}},
		{"test_warns_on_outdated_index", []string{"warns", "outdated", "index"}},
		{"handles_async", []string{"handles", "async"}},
		// Generic test prefixes are stripped.
		{"TestFoo", []string{"foo"}},
		{"test_foo", []string{"foo"}},
	}
	for _, c := range cases {
		got := tokenizeTestName(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("tokenizeTestName(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCountOverlap(t *testing.T) {
	cases := []struct {
		testTokens []string
		keywords   []string
		want       int
	}{
		{[]string{"warns", "outdated", "index"}, []string{"warns", "outdated"}, 2},
		{[]string{"warns"}, []string{"warns", "outdated"}, 1},
		{[]string{"foo"}, []string{"bar"}, 0},
		// Hyphen/underscore equivalence.
		{[]string{"rate_limit"}, []string{"rate-limit"}, 1},
		// Duplicate keywords don't double-count.
		{[]string{"foo"}, []string{"foo", "foo"}, 1},
	}
	for _, c := range cases {
		got := countOverlap(c.testTokens, c.keywords)
		if got != c.want {
			t.Errorf("countOverlap(%v, %v) = %d, want %d", c.testTokens, c.keywords, got, c.want)
		}
	}
}

func TestFindStrongMatches(t *testing.T) {
	files := []*testFile{
		{
			path:    "/x/foo_test.go",
			relPath: "foo_test.go",
			content: "func testparsescsv(t *testing.t)",
			testNames: []nameTokens{
				{display: "TestParsesCSV", tokens: []string{"parses", "csv"}},
				{display: "TestUnrelated", tokens: []string{"unrelated"}},
			},
		},
		{
			path:    "/x/bar_test.go",
			relPath: "bar_test.go",
			testNames: []nameTokens{
				{display: "TestStreamsLargeExport", tokens: []string{"streams", "large", "export"}},
			},
		},
	}

	// ≥2-keyword overlap → strong match on TestParsesCSV.
	tests, paths := findStrongMatches(files, []string{"parses", "csv"})
	if !reflect.DeepEqual(tests, []string{"TestParsesCSV"}) {
		t.Errorf("expected TestParsesCSV match, got %v", tests)
	}
	if !reflect.DeepEqual(paths, []string{"foo_test.go"}) {
		t.Errorf("expected foo_test.go path, got %v", paths)
	}

	// Single-keyword criterion: threshold drops to 1.
	tests, _ = findStrongMatches(files, []string{"export"})
	if !reflect.DeepEqual(tests, []string{"TestStreamsLargeExport"}) {
		t.Errorf("expected single-keyword match, got %v", tests)
	}

	// Only one of two keywords overlaps → no strong match.
	tests, _ = findStrongMatches(files, []string{"parses", "json"})
	if len(tests) != 0 {
		t.Errorf("expected no strong match, got %v", tests)
	}
}

func TestAnalyzeSpec_StrongMatchAndWeakFallback(t *testing.T) {
	dir := t.TempDir()

	// A test that strongly matches one criterion via test name.
	writeFile(t, dir, "parser_test.go", `package foo
import "testing"
func TestParsesCsvHeader(t *testing.T) {}
`)
	// A test whose name matches nothing relevant, but whose body mentions
	// the criterion's keywords — qualifies only as a weak match.
	writeFile(t, dir, "io_test.go", `package foo
import "testing"
func TestUnrelated(t *testing.T) {
    // touches streaming export logic
    _ = "streaming export buffer"
}
`)

	files, err := discoverTestFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	tfs := loadTestFiles(t, dir, files)

	// Strong: criterion mentions "parses" + "csv" + "header".
	strong, _ := findStrongMatches(tfs, []string{"parses", "csv", "header"})
	if len(strong) == 0 {
		t.Error("expected strong match for parses/csv/header")
	}

	// No strong match for streaming/buffer (test names don't share tokens),
	// but weak fallback should hit io_test.go.
	strong, _ = findStrongMatches(tfs, []string{"streaming", "buffer"})
	if len(strong) != 0 {
		t.Errorf("expected no strong match, got %v", strong)
	}
	weak := findWeakMatches(tfs, []string{"streaming", "buffer"})
	if len(weak) == 0 || !strings.Contains(weak[0], "io_test.go") {
		t.Errorf("expected weak match on io_test.go, got %v", weak)
	}
}

func TestRustHasTestMarker(t *testing.T) {
	dir := t.TempDir()

	withMarker := writeFile(t, dir, "with.rs", "#[test]\nfn t() {}\n")
	withoutMarker := writeFile(t, dir, "without.rs", "fn main() {}\n")
	withTokio := writeFile(t, dir, "tokio.rs", "#[tokio::test]\nasync fn t() {}\n")
	withCfg := writeFile(t, dir, "cfg.rs", "#[cfg(test)] mod x {}\n")

	if !rustHasTestMarker(withMarker) {
		t.Error("expected #[test] to be detected")
	}
	if rustHasTestMarker(withoutMarker) {
		t.Error("expected file without markers to be rejected")
	}
	if !rustHasTestMarker(withTokio) {
		t.Error("expected #[tokio::test] to be detected")
	}
	if !rustHasTestMarker(withCfg) {
		t.Error("expected #[cfg(test)] to be detected")
	}
}

func TestFormatText(t *testing.T) {
	r := &CoverageReport{
		Slug:        "csv-export",
		Title:       "CSV Export",
		Total:       3,
		Covered:     2,
		StrongCount: 1,
		WeakCount:   1,
		Gaps:        1,
		Criteria: []CriterionCoverage{
			{Index: 1, Raw: "criterion 1", Covered: true, MatchStrength: StrengthStrong, MatchTests: []string{"TestOne"}, MatchFiles: []string{"a_test.go"}},
			{Index: 2, Raw: "criterion 2 about export", Covered: true, MatchStrength: StrengthWeak, MatchFiles: []string{"b_test.go"}, Keywords: []string{"export"}},
			{Index: 3, Raw: "criterion 3 about streaming", Covered: false, Detail: `no test mentions "streaming"`, Keywords: []string{"streaming"}},
		},
	}

	text := FormatText(r)
	if text == "" {
		t.Error("expected non-empty text output")
	}
	if !strings.Contains(text, "2/3") {
		t.Error("expected coverage ratio in output")
	}
	if !strings.Contains(text, "1 strong, 1 weak") {
		t.Error("expected strong/weak summary in output")
	}
	if !strings.Contains(text, "Gaps") {
		t.Error("expected gaps section")
	}
	if !strings.Contains(text, "Weak matches") {
		t.Error("expected weak-matches section")
	}
}

// helpers

func writeFile(t *testing.T, dir, relPath, content string) string {
	t.Helper()
	p := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func loadTestFiles(t *testing.T, root string, paths []string) []*testFile {
	t.Helper()
	out := make([]*testFile, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		raw := string(data)
		rel, _ := filepath.Rel(root, p)
		out = append(out, &testFile{
			path:      p,
			relPath:   rel,
			content:   strings.ToLower(raw),
			testNames: extractTestNames(p, raw),
		})
	}
	return out
}
