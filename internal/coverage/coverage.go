// Package coverage maps spec acceptance criteria to test files,
// reporting which criteria have test coverage and which are gaps.
package coverage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hero-engine/hero/internal/drift"
	"github.com/hero-engine/hero/internal/spec"
)

// testFilePatterns matches common test file naming conventions.
var testFileSuffixes = []string{
	"_test.go",
	".test.ts", ".test.js", ".test.tsx", ".test.jsx",
	".spec.ts", ".spec.js", ".spec.tsx", ".spec.jsx",
	"_test.py",
}

var testFilePrefixes = []string{
	"test_",
}

// skipDirs are directories excluded from test file discovery.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".git":         true,
	".hero":        true,
	"dist":         true,
	"build":        true,
	"target":       true,
}

// Match strength labels.
const (
	StrengthStrong = "strong"
	StrengthWeak   = "weak"
)

// rustHeadBytes is the maximum prefix scanned to detect inline test markers.
const rustHeadBytes = 64 * 1024

var rustTestMarkerRe = regexp.MustCompile(`#\[(?:test\b|cfg\(test\)|tokio::test\b|async_std::test\b|rstest\b)`)

// CriterionCoverage tracks test coverage for a single acceptance criterion.
type CriterionCoverage struct {
	Index         int      `json:"index"`
	Raw           string   `json:"raw"`
	Kind          string   `json:"kind"`
	Covered       bool     `json:"covered"`
	MatchStrength string   `json:"match_strength,omitempty"`
	MatchTests    []string `json:"match_tests,omitempty"`
	MatchFiles    []string `json:"match_files,omitempty"`
	Keywords      []string `json:"keywords"`
	Detail        string   `json:"detail,omitempty"`
}

// CoverageReport is the coverage analysis for a single spec.
type CoverageReport struct {
	Slug         string              `json:"slug"`
	Title        string              `json:"title"`
	Total        int                 `json:"total"`
	Covered      int                 `json:"covered"`
	StrongCount  int                 `json:"strong"`
	WeakCount    int                 `json:"weak"`
	Gaps         int                 `json:"gaps"`
	Criteria     []CriterionCoverage `json:"criteria"`
	ExitCode     int                 `json:"exit_code"`
}

// Analyze produces a coverage report for a single spec.
func Analyze(projectRoot, heroDir, slug, testDir string) (*CoverageReport, error) {
	allSpecs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	var target *spec.Spec
	for _, s := range allSpecs {
		if s.Slug == slug {
			target = s
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("spec %q not found", slug)
	}

	return analyzeSpec(projectRoot, target, testDir)
}

// AnalyzeAll produces coverage reports for all specs with acceptance criteria.
func AnalyzeAll(projectRoot, heroDir, testDir string) ([]*CoverageReport, error) {
	allSpecs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	var reports []*CoverageReport
	for _, s := range allSpecs {
		if len(s.AcceptanceCriteria()) == 0 {
			continue
		}
		r, err := analyzeSpec(projectRoot, s, testDir)
		if err != nil {
			continue
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// testFile holds the discovered file plus its lazily-loaded content and
// extracted test-name tokens.
type testFile struct {
	path       string
	relPath    string
	content    string // lowercased
	testNames  []nameTokens
}

// nameTokens is one extracted test identifier or label, plus the normalized
// keyword tokens derived from it for matching.
type nameTokens struct {
	display string   // human-readable name as it appears in source
	tokens  []string // lowercased keyword tokens
}

func analyzeSpec(projectRoot string, s *spec.Spec, testDir string) (*CoverageReport, error) {
	criteria := s.AcceptanceCriteria()
	if len(criteria) == 0 {
		return &CoverageReport{
			Slug:  s.Slug,
			Title: s.Title,
		}, nil
	}

	root := projectRoot
	if testDir != "" {
		root = filepath.Join(projectRoot, testDir)
	}
	paths, err := discoverTestFiles(root)
	if err != nil {
		return nil, fmt.Errorf("discovering test files: %w", err)
	}

	files := make([]*testFile, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		raw := string(data)
		rel, _ := filepath.Rel(projectRoot, p)
		if rel == "" {
			rel = p
		}
		files = append(files, &testFile{
			path:      p,
			relPath:   rel,
			content:   strings.ToLower(raw),
			testNames: extractTestNames(p, raw),
		})
	}

	report := &CoverageReport{
		Slug:  s.Slug,
		Title: s.Title,
		Total: len(criteria),
	}

	for i, criterion := range criteria {
		var searchText string
		if criterion.Trigger != "" {
			searchText = criterion.Trigger + " " + criterion.Behavior
		} else {
			searchText = criterion.Raw
		}
		keywords := drift.ExtractKeywords(searchText)
		lowerKeywords := lowerAll(keywords)

		cc := CriterionCoverage{
			Index:    i + 1,
			Raw:      criterion.Raw,
			Kind:     criterion.Kind.String(),
			Keywords: keywords,
		}

		strongTests, strongFiles := findStrongMatches(files, lowerKeywords)
		if len(strongTests) > 0 {
			cc.Covered = true
			cc.MatchStrength = StrengthStrong
			cc.MatchTests = strongTests
			cc.MatchFiles = strongFiles
		} else {
			weakFiles := findWeakMatches(files, lowerKeywords)
			if len(weakFiles) > 0 {
				cc.Covered = true
				cc.MatchStrength = StrengthWeak
				cc.MatchFiles = weakFiles
			}
		}

		if !cc.Covered {
			cc.Detail = fmt.Sprintf("no test mentions %s", formatKeywords(keywords))
		}

		report.Criteria = append(report.Criteria, cc)
		switch cc.MatchStrength {
		case StrengthStrong:
			report.Covered++
			report.StrongCount++
		case StrengthWeak:
			report.Covered++
			report.WeakCount++
		}
	}

	report.Gaps = report.Total - report.Covered
	if report.Gaps > 0 {
		report.ExitCode = 1
	}
	return report, nil
}

// findStrongMatches returns the matching test display names and the files they
// came from. A test name matches when it shares enough keyword overlap with
// the criterion: ≥2 overlaps normally, ≥1 when the criterion has only one
// keyword to begin with.
func findStrongMatches(files []*testFile, keywords []string) (tests, paths []string) {
	threshold := 2
	if len(keywords) <= 1 {
		threshold = 1
	}
	if len(keywords) == 0 {
		return nil, nil
	}

	seenTest := make(map[string]bool)
	seenPath := make(map[string]bool)
	for _, f := range files {
		matchedThisFile := false
		for _, tn := range f.testNames {
			if countOverlap(tn.tokens, keywords) >= threshold {
				if !seenTest[tn.display] {
					seenTest[tn.display] = true
					tests = append(tests, tn.display)
				}
				matchedThisFile = true
			}
		}
		if matchedThisFile && !seenPath[f.relPath] {
			seenPath[f.relPath] = true
			paths = append(paths, f.relPath)
		}
	}
	return tests, paths
}

// findWeakMatches falls back to the legacy whole-file ≥1-keyword substring
// check. Only invoked when no strong match was found.
func findWeakMatches(files []*testFile, keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}
	var paths []string
	seen := make(map[string]bool)
	for _, f := range files {
		if anyKeywordInContent(keywords, f.content) {
			if !seen[f.relPath] {
				seen[f.relPath] = true
				paths = append(paths, f.relPath)
			}
		}
	}
	return paths
}

func anyKeywordInContent(keywords []string, content string) bool {
	for _, kw := range keywords {
		if strings.Contains(content, kw) {
			return true
		}
		if alt := strings.ReplaceAll(kw, "-", "_"); alt != kw && strings.Contains(content, alt) {
			return true
		}
	}
	return false
}

func countOverlap(testTokens, criterionKeywords []string) int {
	if len(testTokens) == 0 {
		return 0
	}
	tokenSet := make(map[string]bool, len(testTokens))
	for _, t := range testTokens {
		tokenSet[t] = true
	}
	overlap := 0
	seen := make(map[string]bool, len(criterionKeywords))
	for _, kw := range criterionKeywords {
		if seen[kw] {
			continue
		}
		seen[kw] = true
		if tokenSet[kw] {
			overlap++
			continue
		}
		// Allow hyphen/underscore equivalence.
		if alt := strings.ReplaceAll(kw, "-", "_"); alt != kw && tokenSet[alt] {
			overlap++
		}
	}
	return overlap
}

func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.ToLower(s)
	}
	return out
}

// keywordsMatch is retained for backwards compatibility with tests that
// exercise the legacy whole-file substring behavior.
func keywordsMatch(keywords []string, content string) bool {
	for _, kw := range keywords {
		lower := strings.ToLower(kw)
		underscore := strings.ReplaceAll(lower, "-", "_")
		if strings.Contains(content, lower) || strings.Contains(content, underscore) {
			return true
		}
	}
	return false
}

func formatKeywords(keywords []string) string {
	if len(keywords) == 0 {
		return "(no keywords extracted)"
	}
	quoted := make([]string, len(keywords))
	for i, kw := range keywords {
		quoted[i] = fmt.Sprintf("%q", kw)
	}
	if len(quoted) > 5 {
		quoted = quoted[:5]
		quoted = append(quoted, "...")
	}
	return strings.Join(quoted, ", ")
}

func discoverTestFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if isTestFile(name) {
			files = append(files, path)
			return nil
		}
		if strings.HasSuffix(name, ".rs") && isRustTestFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isTestFile(name string) bool {
	for _, suffix := range testFileSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	for _, prefix := range testFilePrefixes {
		if strings.HasPrefix(name, prefix) && !strings.HasSuffix(name, ".pyc") {
			return true
		}
	}
	return false
}

// isRustTestFile recognizes a .rs file as a test file when it lives under a
// `tests/` directory (Cargo integration test convention) or contains an
// inline test attribute in its first 64KiB.
func isRustTestFile(path string) bool {
	if pathHasSegment(path, "tests") {
		return true
	}
	return rustHasTestMarker(path)
}

func pathHasSegment(path, segment string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range parts[:len(parts)-1] { // exclude filename itself
		if p == segment {
			return true
		}
	}
	return false
}

func rustHasTestMarker(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, rustHeadBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return false
	}
	return rustTestMarkerRe.Match(buf[:n])
}

// Test-name extraction.

var (
	goTestRe       = regexp.MustCompile(`func\s+((?:Test|Benchmark|Example|Fuzz)\w+)\s*\(`)
	rustTestRe     = regexp.MustCompile(`#\[(?:test|tokio::test|async_std::test|rstest)[^\]]*\][\s\S]{0,200}?fn\s+(\w+)`)
	pyTestRe       = regexp.MustCompile(`def\s+(test_\w+)\s*\(`)
	jsLabelRe      = regexp.MustCompile(`(?:^|[^\w$])(?:it|test|describe)\s*\(\s*["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `]`)
	jsFnRe         = regexp.MustCompile(`function\s+(test\w+|describe\w+)\s*\(`)
)

// extractTestNames returns the display names and normalized keyword tokens for
// each test definition or label found in content. Dispatches on file extension.
func extractTestNames(path, content string) []nameTokens {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return extractMatches(content, goTestRe)
	case ".rs":
		return extractMatches(content, rustTestRe)
	case ".py":
		return extractMatches(content, pyTestRe)
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		out := extractMatches(content, jsLabelRe)
		out = append(out, extractMatches(content, jsFnRe)...)
		return out
	}
	return nil
}

func extractMatches(content string, re *regexp.Regexp) []nameTokens {
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]nameTokens, 0, len(matches))
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		display := m[1]
		if seen[display] {
			continue
		}
		seen[display] = true
		out = append(out, nameTokens{
			display: display,
			tokens:  tokenizeTestName(display),
		})
	}
	return out
}

// tokenizeTestName converts a test identifier or label into a set of
// lowercased keyword tokens, splitting on snake_case, kebab-case, dots,
// spaces, and CamelCase. The same `drift.ExtractKeywords` filter is then
// applied so test-name and criterion-keyword spaces match symmetrically.
func tokenizeTestName(name string) []string {
	// Split CamelCase by inserting spaces before uppercase letters preceded
	// by a lowercase letter or digit.
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(name[i-1])
			if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') {
				b.WriteByte(' ')
			}
		}
		b.WriteRune(r)
	}
	separated := b.String()
	for _, sep := range []string{"_", "-", ".", "/"} {
		separated = strings.ReplaceAll(separated, sep, " ")
	}
	keywords := drift.ExtractKeywords(separated)

	// Drop generic test framework prefixes that survive the stop-word filter
	// because they are >2 chars: test, tests, benchmark, example, fuzz.
	noise := map[string]bool{
		"test": true, "tests": true, "benchmark": true,
		"example": true, "fuzz": true, "spec": true, "specs": true,
		"it": true, "describe": true,
	}
	out := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		lower := strings.ToLower(kw)
		if noise[lower] {
			continue
		}
		out = append(out, lower)
	}
	return out
}

// FormatText produces human-readable output for a coverage report.
func FormatText(r *CoverageReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n", r.Slug, r.Title)
	fmt.Fprintf(&b, "  %d/%d criteria have test coverage", r.Covered, r.Total)
	if r.Covered > 0 && (r.StrongCount > 0 || r.WeakCount > 0) {
		fmt.Fprintf(&b, " (%d strong, %d weak)", r.StrongCount, r.WeakCount)
	}
	fmt.Fprintln(&b)

	if r.Gaps > 0 {
		fmt.Fprintf(&b, "  Gaps:\n")
		for _, c := range r.Criteria {
			if !c.Covered {
				fmt.Fprintf(&b, "    criterion %d: %s\n", c.Index, truncate(c.Raw, 80))
				fmt.Fprintf(&b, "      -> %s\n", c.Detail)
			}
		}
	}

	if r.WeakCount > 0 {
		fmt.Fprintf(&b, "  Weak matches (filename only, test names didn't match):\n")
		for _, c := range r.Criteria {
			if c.MatchStrength != StrengthWeak {
				continue
			}
			file := ""
			if len(c.MatchFiles) > 0 {
				file = c.MatchFiles[0]
			}
			fmt.Fprintf(&b, "    criterion %d: matched %s on keyword %s\n",
				c.Index, file, formatKeywords(c.Keywords))
		}
	}
	return b.String()
}

// FormatJSON produces JSON output for a coverage report.
func FormatJSON(r *CoverageReport) (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
